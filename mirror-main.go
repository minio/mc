/*
 * Minio Client, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// mirror specific flags.
var (
	mirrorFlagForce = cli.BoolFlag{
		Name:  "force",
		Usage: "Force overwrite of an existing target(s).",
	}
	mirrorFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of mirror.",
	}
)

//  Mirror folders recursively from a single source to many destinations
var mirrorCmd = cli.Command{
	Name:   "mirror",
	Usage:  "Mirror folders recursively from a single source to many destinations.",
	Action: mainMirror,
	Flags:  []cli.Flag{mirrorFlagForce, mirrorFlagHelp},
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] SOURCE TARGET [TARGET...]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Mirror a bucket recursively from Minio cloud storage to a bucket on Amazon S3 cloud storage.
      $ mc {{.Name}} https://play.minio.io:9000/photos/2014 https://s3.amazonaws.com/backup-photos

   2. Mirror a local folder recursively to Minio cloud storage, Amazon S3 cloud storage and Google Cloud Storage.
      $ mc {{.Name}} backup/ https://play.minio.io:9000/archive https://s3.amazonaws.com/archive https://storage.googleapis.com/miniocloud

   3. Mirror a bucket from aliased Amazon S3 cloud storage to multiple folders on Windows.
      $ mc {{.Name}} s3/documents/2014/ C:\backup\2014 C:\shared\volume\backup\2014
`,
}

var (
	mirrorIsForce = false // mirror specific force flag set via command line
)

// mirrorMessage container for file mirror messages
type mirrorMessage struct {
	Source  string   `json:"source"`
	Targets []string `json:"targets"`
}

// String colorized mirror message
func (m mirrorMessage) String() string {
	return console.Colorize("Mirror", fmt.Sprintf("‘%s’ -> ‘%s’", m.Source, m.Targets))
}

// JSON jsonified mirror message
func (m mirrorMessage) JSON() string {
	mirrorMessageBytes, e := json.Marshal(m)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(mirrorMessageBytes)
}

// doMirror - Mirror an object to multiple destination. mirrorURLs status contains a copy of sURLs and error if any.
func doMirror(sURLs mirrorURLs, progressReader *barSend, mirrorQueueCh <-chan bool, wg *sync.WaitGroup, statusCh chan<- mirrorURLs) {
	defer wg.Done() // Notify that this copy routine is done.
	defer func() {
		<-mirrorQueueCh
	}()

	if sURLs.Error != nil { // Errorneous sURLs passed.
		sURLs.Error = sURLs.Error.Trace()
		statusCh <- sURLs
		return
	}

	if !globalQuietFlag && !globalJSONFlag {
		progressReader.SetCaption(sURLs.SourceContent.URL.String() + ": ")
	}

	reader, length, err := getSource(sURLs.SourceContent.URL.String())
	if err != nil {
		if !globalQuietFlag && !globalJSONFlag {
			progressReader.ErrorGet(int64(length))
		}
		sURLs.Error = err.Trace(sURLs.SourceContent.URL.String())
		statusCh <- sURLs
		return
	}

	var targetURLs []string
	for _, targetContent := range sURLs.TargetContents {
		targetURLs = append(targetURLs, targetContent.URL.String())
	}

	if globalQuietFlag || globalJSONFlag {
		printMsg(mirrorMessage{
			Source:  sURLs.SourceContent.URL.String(),
			Targets: targetURLs,
		})
	}
	// set up progress
	newReader := progressReader.NewProxyReader(reader)
	defer newReader.Close()

	err = putTargets(targetURLs, length, newReader)
	if err != nil {
		if !globalQuietFlag && !globalJSONFlag {
			progressReader.ErrorPut(int64(length))
		}
		sURLs.Error = err.Trace(targetURLs...)
		statusCh <- sURLs
		return
	}

	sURLs.Error = nil // just for safety
	statusCh <- sURLs
}

// doMirrorFake - Perform a fake mirror to update the progress bar appropriately.
func doMirrorFake(sURLs mirrorURLs, progressReader *barSend) {
	progressReader.Progress(sURLs.SourceContent.Size)
}

// doPrepareMirrorURLs scans the source URL and prepares a list of objects for mirroring.
func doPrepareMirrorURLs(session *sessionV3, trapCh <-chan bool) {
	sourceURL := session.Header.CommandArgs[0] // first one is source.
	targetURLs := session.Header.CommandArgs[1:]
	var totalBytes int64
	var totalObjects int

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	var scanBar scanBarFunc
	if !globalQuietFlag && !globalJSONFlag { // set up progress bar
		scanBar = scanBarFactory()
	}

	// will be true if '--force' is provided on the command line.
	mirrorIsForce = session.Header.CommandBoolFlag.Value

	URLsCh := prepareMirrorURLs(sourceURL, targetURLs)
	done := false
	for done == false {
		select {
		case sURLs, ok := <-URLsCh:
			if !ok { // Done with URL prepration
				done = true
				break
			}
			if sURLs.Error != nil {
				// Print in new line and adjust to top so that we don't print over the ongoing scan bar
				if !globalQuietFlag && !globalJSONFlag {
					console.Eraseline()
				}
				errorIf(sURLs.Error.Trace(), "Unable to prepare URLs for mirroring.")
				break
			}
			if sURLs.isEmpty() {
				break
			}
			jsonData, err := json.Marshal(sURLs)
			if err != nil {
				session.Delete()
				fatalIf(probe.NewError(err), "Unable to marshal URLs into JSON.")
			}
			fmt.Fprintln(dataFP, string(jsonData))
			if !globalQuietFlag && !globalJSONFlag {
				scanBar(sURLs.SourceContent.URL.String())
			}

			totalBytes += sURLs.SourceContent.Size
			totalObjects++
		case <-trapCh:
			// Print in new line and adjust to top so that we don't print over the ongoing scan bar
			if !globalQuietFlag && !globalJSONFlag {
				console.Eraseline()
			}
			session.Delete() // If we are interrupted during the URL scanning, we drop the session.
			os.Exit(0)
		}
	}
	session.Header.TotalBytes = totalBytes
	session.Header.TotalObjects = totalObjects
	session.Save()
}

// Session'fied mirror command.
func doMirrorSession(session *sessionV3) {
	trapCh := signalTrap(os.Interrupt, os.Kill)

	if !session.HasData() {
		doPrepareMirrorURLs(session, trapCh)
	}

	// Set up progress bar.
	progressReader := newProgressBar(session.Header.TotalBytes, globalJSONFlag || globalQuietFlag)

	// Prepare URL scanner from session data file.
	scanner := bufio.NewScanner(session.NewDataReader())
	// isCopied returns true if an object has been already copied
	// or not. This is useful when we resume from a session.
	isCopied := isCopiedFactory(session.Header.LastCopied)

	wg := new(sync.WaitGroup)
	// Limit numner of mirror routines based on available CPU resources.
	mirrorQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
	defer close(mirrorQueue)
	// Status channel for receiveing mirror return status.
	statusCh := make(chan mirrorURLs)

	// Go routine to monitor doMirror status and signal traps.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case sURLs, ok := <-statusCh: // Receive status.
				if !ok { // We are done here. Top level function has returned.
					if globalQuietFlag {
						console.Println(console.Colorize("Mirror", progressReader.Stats().String()))
						return
					}
					if !globalJSONFlag {
						progressReader.Finish()
						return
					}
					return
				}
				if sURLs.Error == nil {
					session.Header.LastCopied = sURLs.SourceContent.URL.String()
					session.Save()
				} else {
					// Print in new line and adjust to top so that we don't print over the ongoing progress bar
					if !globalQuietFlag && !globalJSONFlag {
						console.Eraseline()
					}
					errorIf(sURLs.Error.Trace(), fmt.Sprintf("Failed to mirror ‘%s’.", sURLs.SourceContent.URL.String()))
					// all the cases which are handled where session should be saved are contained in the following
					// switch case, we shouldn't be saving sessions for all errors since some errors might need to be
					// reported to user properly.
					//
					// All other critical cases should be handled properly gracefully
					// handle more errors and save the session.
					switch sURLs.Error.ToGoError().(type) {
					case *net.OpError:
						session.CloseAndDie()
					case net.Error:
						session.CloseAndDie()
					}
				}
			case <-trapCh: // Receive interrupt notification.
				// Print in new line and adjust to top so that we don't print over the ongoing progress bar
				if !globalQuietFlag && !globalJSONFlag {
					console.Eraseline()
				}
				session.CloseAndDie()
			}
		}
	}()

	// Go routine to perform concurrently mirroring.
	wg.Add(1)
	go func() {
		defer wg.Done()
		mirrorWg := new(sync.WaitGroup)
		defer close(statusCh)

		for scanner.Scan() {
			var sURLs mirrorURLs
			json.Unmarshal([]byte(scanner.Text()), &sURLs)
			if isCopied(sURLs.SourceContent.URL.String()) {
				doMirrorFake(sURLs, progressReader)
			} else {
				// Wait for other mirror routines to
				// complete. We only have limited CPU
				// and network resources.
				mirrorQueue <- true
				// Account for each mirror routines we start.
				mirrorWg.Add(1)
				// Do mirroring in background concurrently.
				go doMirror(sURLs, progressReader, mirrorQueue, mirrorWg, statusCh)
			}
		}
		mirrorWg.Wait()
	}()

	wg.Wait()
}

// Main entry point for mirror command.
func mainMirror(ctx *cli.Context) {
	checkMirrorSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Mirror", color.New(color.FgGreen, color.Bold))

	var e error
	session := newSessionV3()
	session.Header.CommandType = "mirror"
	session.Header.RootPath, e = os.Getwd()
	if e != nil {
		session.Delete()
		fatalIf(probe.NewError(e), "Unable to get current working folder.")
	}

	// If force flag is set save it with in session
	session.Header.CommandBoolFlag.Key = "force"
	session.Header.CommandBoolFlag.Value = ctx.Bool("force")

	// extract URLs.
	var err *probe.Error
	session.Header.CommandArgs, err = args2URLs(ctx.Args())
	if err != nil {
		session.Delete()
		fatalIf(err.Trace(ctx.Args()...), fmt.Sprintf("One or more unknown argument types found in ‘%s’.", ctx.Args()))
	}

	doMirrorSession(session)
	session.Delete()
}
