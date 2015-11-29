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
	"io"
	"math"
	"os"
	"runtime"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/pb"
)

// mirror specific flags.
var (
	mirrorFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of mirror.",
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "Force overwrite of an existing target(s).",
		},
	}
)

//  Mirror folders recursively from a single source to many destinations
var mirrorCmd = cli.Command{
	Name:   "mirror",
	Usage:  "Mirror folders recursively from a single source to single destination.",
	Action: mainMirror,
	Flags:  append(mirrorFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] SOURCE TARGET

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Mirror a bucket recursively from Minio cloud storage to a bucket on Amazon S3 cloud storage.
      $ mc {{.Name}} play.minio.io:9000/photos/2014 s3.amazonaws.com/backup-photos

   2. Mirror a local folder recursively to Amazon S3 cloud storage.
      $ mc {{.Name}} backup/ s3.amazonaws.com/archive

   3. Mirror a bucket from aliased Amazon S3 cloud storage to a folder on Windows.
      $ mc {{.Name}} s3/documents/2014/ C:\backup\2014
`,
}

// mirrorMessage container for file mirror messages
type mirrorMessage struct {
	Status string `json:"status"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// String colorized mirror message
func (m mirrorMessage) String() string {
	return console.Colorize("Mirror", fmt.Sprintf("‘%s’ -> ‘%s’", m.Source, m.Target))
}

// JSON jsonified mirror message
func (m mirrorMessage) JSON() string {
	m.Status = "success"
	mirrorMessageBytes, e := json.Marshal(m)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(mirrorMessageBytes)
}

// mirrorStatMessage container for mirror accounting message
type mirrorStatMessage struct {
	Total       int64
	Transferred int64
	Speed       float64
}

// mirrorStatMessage mirror accounting message
func (c mirrorStatMessage) String() string {
	speedBox := pb.FormatBytes(int64(c.Speed))
	if speedBox == "" {
		speedBox = "0 MB"
	} else {
		speedBox = speedBox + "/s"
	}
	message := fmt.Sprintf("Total: %s, Transferred: %s, Speed: %s", pb.FormatBytes(c.Total),
		pb.FormatBytes(c.Transferred), speedBox)
	return message
}

// doMirror - Mirror an object to multiple destination. mirrorURLs status contains a copy of sURLs and error if any.
func doMirror(sURLs mirrorURLs, progressReader *barSend, accountingReader *accounter, mirrorQueueCh <-chan bool, wg *sync.WaitGroup, statusCh chan<- mirrorURLs) {
	defer wg.Done() // Notify that this copy routine is done.
	defer func() {
		<-mirrorQueueCh
	}()

	if sURLs.Error != nil { // Errorneous sURLs passed.
		sURLs.Error = sURLs.Error.Trace()
		statusCh <- sURLs
		return
	}

	targetURL := sURLs.TargetContent.URL.String()
	sourceURL := sURLs.SourceContent.URL.String()
	length := sURLs.SourceContent.Size

	if !globalQuiet && !globalJSON {
		progressReader.SetCaption(sourceURL + ": ")
	}

	reader, err := getSource(sourceURL)
	if err != nil {
		if !globalQuiet && !globalJSON {
			progressReader.ErrorGet(length)
		}
		sURLs.Error = err.Trace(sourceURL)
		statusCh <- sURLs
		return
	}

	var newReader io.ReadSeeker
	if globalQuiet || globalJSON {
		printMsg(mirrorMessage{
			Source: sourceURL,
			Target: targetURL,
		})
		if globalJSON {
			newReader = reader
		}
		if globalQuiet {
			newReader = accountingReader.NewProxyReader(reader)
		}
	} else {
		// set up progress
		newReader = progressReader.NewProxyReader(reader)
	}
	err = putTarget(targetURL, newReader, length)
	if err != nil {
		if !globalQuiet && !globalJSON {
			progressReader.ErrorPut(length)
		}
		sURLs.Error = err.Trace(targetURL)
		statusCh <- sURLs
		return
	}

	sURLs.Error = nil // just for safety
	statusCh <- sURLs
}

// doMirrorFake - Perform a fake mirror to update the progress bar appropriately.
func doMirrorFake(sURLs mirrorURLs, progressReader *barSend) {
	if !globalDebug && !globalJSON {
		progressReader.Progress(sURLs.SourceContent.Size)
	}
}

// doPrepareMirrorURLs scans the source URL and prepares a list of objects for mirroring.
func doPrepareMirrorURLs(session *sessionV5, isForce bool, trapCh <-chan bool) {
	sourceURL := session.Header.CommandArgs[0] // first one is source.
	targetURL := session.Header.CommandArgs[1]
	var totalBytes int64
	var totalObjects int

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	var scanBar scanBarFunc
	if !globalQuiet && !globalJSON { // set up progress bar
		scanBar = scanBarFactory()
	}

	URLsCh := prepareMirrorURLs(sourceURL, targetURL, isForce)
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
				if !globalQuiet && !globalJSON {
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
			if !globalQuiet && !globalJSON {
				scanBar(sURLs.SourceContent.URL.String())
			}

			totalBytes += sURLs.SourceContent.Size
			totalObjects++
		case <-trapCh:
			// Print in new line and adjust to top so that we don't print over the ongoing scan bar
			if !globalQuiet && !globalJSON {
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
func doMirrorSession(session *sessionV5) {
	isForce := session.Header.CommandBoolFlags["force"]
	trapCh := signalTrap(os.Interrupt, syscall.SIGTERM)

	if !session.HasData() {
		doPrepareMirrorURLs(session, isForce, trapCh)
	}

	// Enable accounting reader by default.
	accntReader := newAccounter(session.Header.TotalBytes)

	// Set up progress bar.
	var progressReader *barSend
	if !globalQuiet && !globalJSON {
		progressReader = newProgressBar(session.Header.TotalBytes)
	}

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
					if !globalQuiet && !globalJSON {
						progressReader.Finish()
					} else {
						accntStat := accntReader.Stat()
						mrStatMessage := mirrorStatMessage{
							Total:       accntStat.Total,
							Transferred: accntStat.Transferred,
							Speed:       accntStat.Speed,
						}
						console.Println(console.Colorize("Mirror", mrStatMessage.String()))
					}
					return
				}
				if sURLs.Error == nil {
					session.Header.LastCopied = sURLs.SourceContent.URL.String()
					session.Save()
				} else {
					// Print in new line and adjust to top so that we don't print over the ongoing progress bar
					if !globalQuiet && !globalJSON {
						console.Eraseline()
					}
					errorIf(sURLs.Error.Trace(), fmt.Sprintf("Failed to mirror ‘%s’.", sURLs.SourceContent.URL.String()))
					// for all non critical errors we can continue for the remaining files
					switch sURLs.Error.ToGoError().(type) {
					// handle this specifically for filesystem related errors.
					case client.BrokenSymlink:
						continue
					case client.TooManyLevelsSymlink:
						continue
					case client.PathNotFound:
						continue
					case client.PathInsufficientPermission:
						continue
					}
					// for critical errors we should exit. Session can be resumed after the user figures out the problem
					session.CloseAndDie()
				}
			case <-trapCh: // Receive interrupt notification.
				// Print in new line and adjust to top so that we don't print over the ongoing progress bar
				if !globalQuiet && !globalJSON {
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
				go doMirror(sURLs, progressReader, accntReader, mirrorQueue, mirrorWg, statusCh)
			}
		}
		mirrorWg.Wait()
	}()

	wg.Wait()
}

// Main entry point for mirror command.
func mainMirror(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'mirror' cli arguments.
	checkMirrorSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Mirror", color.New(color.FgGreen, color.Bold))

	var e error
	session := newSessionV5()
	session.Header.CommandType = "mirror"
	session.Header.RootPath, e = os.Getwd()
	if e != nil {
		session.Delete()
		fatalIf(probe.NewError(e), "Unable to get current working folder.")
	}

	// Set command flags from context.
	isForce := ctx.Bool("force")
	session.Header.CommandBoolFlags["force"] = isForce

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
