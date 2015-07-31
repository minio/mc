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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var mirrorCmd = cli.Command{
	Name:   "mirror",
	Usage:  "Copy files and folders from a single source to many destinations",
	Action: runCastCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Cast a bucket recursively from Minio cloud storage to multiple buckets on Amazon S3 cloud storage.
      $ mc {{.Name}} https://play.minio.io:9000/photos/2014... https://s3.amazonaws.com/backup-photos https://s3-west-1.amazonaws.com/local-photos

   2. Cast a local folder recursively to Minio cloud storage and Amazon S3 cloud storage.
      $ mc {{.Name}} backup/... https://play.minio.io:9000/archive https://s3.amazonaws.com/archive

   3. Cast a bucket from aliased Amazon S3 cloud storage to multiple folders on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014 C:\shared\volume\backup\2014

   4. Cast a local folder of non english character recursively to Amazon s3 cloud storage and Minio cloud storage.
      $ mc {{.Name}} 本語/... s3:mylocaldocuments play:backup

`,
}

// doCast - Cast an object to multiple destination. mirrorURLs status contains a copy of sURLs and error if any.
func doCast(sURLs mirrorURLs, bar *barSend, mirrorQueueCh <-chan bool, wg *sync.WaitGroup, statusCh chan<- mirrorURLs) {
	defer wg.Done() // Notify that this copy routine is done.
	defer func() {
		<-mirrorQueueCh
	}()

	if sURLs.Error != nil { // Errorneous sURLs passed.
		sURLs.Error = iodine.New(sURLs.Error, nil)
		statusCh <- sURLs
		return
	}

	if !globalQuietFlag || !globalJSONFlag {
		bar.SetCaption(sURLs.SourceContent.Name + ": ")
	}

	reader, length, err := getSource(sURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag || !globalJSONFlag {
			bar.ErrorGet(int64(length))
		}
		sURLs.Error = iodine.New(err, nil)
		statusCh <- sURLs
		return
	}

	var targetURLs []string
	for _, targetContent := range sURLs.TargetContents {
		targetURLs = append(targetURLs, targetContent.Name)
	}

	var newReader io.ReadCloser
	if globalQuietFlag || globalJSONFlag {
		console.PrintC(CastMessage{
			Source:  sURLs.SourceContent.Name,
			Targets: targetURLs,
		})
		newReader = reader
	} else {
		// set up progress
		newReader = bar.NewProxyReader(reader)
	}
	defer newReader.Close()

	err = putTargets(targetURLs, length, newReader)
	if err != nil {
		if !globalQuietFlag || !globalJSONFlag {
			bar.ErrorPut(int64(length))
		}
		sURLs.Error = iodine.New(err, nil)
		statusCh <- sURLs
		return
	}

	sURLs.Error = nil // just for safety
	statusCh <- sURLs
}

// doCastFake - Perform a fake mirror to update the progress bar appropriately.
func doCastFake(sURLs mirrorURLs, bar *barSend) {
	if !globalDebugFlag || !globalJSONFlag {
		bar.Progress(sURLs.SourceContent.Size)
	}
}

// doPrepareCastURLs scans the source URL and prepares a list of objects for mirroring.
func doPrepareCastURLs(session *sessionV2, trapCh <-chan bool) {
	sourceURL := session.Header.CommandArgs[0] // first one is source.
	targetURLs := session.Header.CommandArgs[1:]
	var totalBytes int64
	var totalObjects int

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	scanBar := scanBarFactory("")
	URLsCh := prepareCastURLs(sourceURL, targetURLs)
	done := false
	for done == false {
		select {
		case sURLs, ok := <-URLsCh:
			if !ok { // Done with URL prepration
				done = true
				break
			}
			if sURLs.Error != nil {
				console.Errorln(sURLs.Error)
				break
			}
			jsonData, err := json.Marshal(sURLs)
			if err != nil {
				session.Close()
				console.Fatalf("Unable to marshal URLs to JSON. %s\n", err)
			}
			fmt.Fprintln(dataFP, string(jsonData))
			scanBar(sURLs.SourceContent.Name)

			totalBytes += sURLs.SourceContent.Size
			totalObjects++
		case <-trapCh:
			session.Close() // If we are interrupted during the URL scanning, we drop the session.
			session.Delete()
			os.Exit(0)
		}
	}
	session.Header.TotalBytes = totalBytes
	session.Header.TotalObjects = totalObjects
	session.Save()
}

func doCastCmdSession(session *sessionV2) {
	trapCh := signalTrap(os.Interrupt, os.Kill)

	if !session.HasData() {
		doPrepareCastURLs(session, trapCh)
	}

	// Set up progress bar.
	var bar barSend
	if !globalQuietFlag || !globalJSONFlag {
		bar = newCpBar()
		bar.Extend(session.Header.TotalBytes)
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

	// Go routine to monitor doCast status and signal traps.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case sURLs, ok := <-statusCh: // Receive status.
				if !ok { // We are done here. Top level function has returned.
					bar.Finish()
					return
				}
				if sURLs.Error == nil {
					session.Header.LastCopied = sURLs.SourceContent.Name
				} else {
					console.Errorf("Failed to mirror ‘%s’, %s\n", sURLs.SourceContent.Name, NewIodine(sURLs.Error))
				}
			case <-trapCh: // Receive interrupt notification.
				session.Close()
				session.Info()
				os.Exit(0)
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
			if isCopied(sURLs.SourceContent.Name) {
				doCastFake(sURLs, &bar)
			} else {
				// Wait for other mirror routines to
				// complete. We only have limited CPU
				// and network resources.
				mirrorQueue <- true
				// Account for each mirror routines we start.
				mirrorWg.Add(1)
				// Do mirroring in background concurrently.
				go doCast(sURLs, &bar, mirrorQueue, mirrorWg, statusCh)
			}
		}
		mirrorWg.Wait()
	}()

	wg.Wait()
}

func runCastCmd(ctx *cli.Context) {
	checkCastSyntax(ctx)

	session := newSessionV2()

	var err error
	session.Header.CommandType = "mirror"
	session.Header.RootPath, err = os.Getwd()
	if err != nil {
		session.Close()
		session.Delete()
		console.Fatalf("Unable to get current working folder. %s\n", err)
	}

	// extract URLs.
	session.Header.CommandArgs, err = args2URLs(ctx.Args())
	if err != nil {
		session.Close()
		session.Delete()
		console.Fatalf("One or more unknown URL types found in %s. %s\n", ctx.Args(), err)
	}

	doCastCmdSession(session)
	session.Close()
	session.Delete()
}
