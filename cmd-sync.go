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
	"github.com/minio/mc/pkg/yielder"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var syncCmd = cli.Command{
	Name:   "sync",
	Usage:  "Copy files and folders from a single source to many destinations",
	Action: runSyncCmd,
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
   1. Sync an object from local filesystem to Amazon S3 object storage.
      $ mc {{.Name}} star-trek-episode-10-season4.ogg https://s3.amazonaws.com/trekarchive

   2. Sync a bucket recursively from Minio object storage to multiple buckets on Amazon S3 object storage.
      $ mc {{.Name}} https://play.minio.io:9000/photos/2014... https://s3.amazonaws.com/backup-photos https://s3-west-1.amazonaws.com/local-photos

   3. Sync a local folder recursively to Minio object storage and Amazon S3 object storage.
      $ mc {{.Name}} backup/... https://play.minio.io:9000/archive https://s3.amazonaws.com/archive

   4. Sync a bucket from aliased Amazon S3 object storage to multiple folders on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014 C:\shared\volume\backup\2014

   5. Sync a local directory of non english character recursively to Amazon s3 object storage and Minio object storage.
      $ mc {{.Name}} 本語/... s3:mylocaldocuments play:backup

`,
}

// doSync - Sync an object to multiple destination
func doSync(sURLs syncURLs, bar *barSend, syncQueue chan bool, wg *sync.WaitGroup) (err error) {
	defer wg.Done() // Notify that this copy routine is done.

	if !globalQuietFlag || !globalJSONFlag {
		bar.SetCaption(sURLs.SourceContent.Name + ": ")
	}

	reader, length, err := getSource(sURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag || !globalJSONFlag {
			bar.ErrorGet(int64(length))
			console.Errors(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}
	}
	defer reader.Close()

	var targetURLs []string
	for _, targetContent := range sURLs.TargetContents {
		targetURLs = append(targetURLs, targetContent.Name)
	}

	var newReader io.Reader
	if globalQuietFlag || globalJSONFlag {
		console.Infos(SyncMessage{
			Source:  sURLs.SourceContent.Name,
			Targets: targetURLs,
		})
		newReader = yielder.NewReader(reader)
	} else {
		// set up progress
		newReader = bar.NewProxyReader(yielder.NewReader(reader))
	}

	for err := range putTargets(targetURLs, length, newReader) {
		if err != nil {
			if !globalQuietFlag || !globalJSONFlag {
				bar.ErrorPut(int64(length))
			}
		}
	}

	<-syncQueue // Notify the copy queue that it is free to pickup next routine.

	return nil
}

// doSyncFake - Perform a fake sync to update the progress bar appropriately.
func doSyncFake(sURLs syncURLs, bar *barSend) (err error) {
	bar.Progress(sURLs.SourceContent.Size)
	return nil
}

// doPrepareSyncURLs scans the source URL and prepares a list of objects for syncing.
func doPrepareSyncURLs(session *sessionV2, trapCh <-chan bool) {
	sourceURL := session.Header.CommandArgs[0] // first one is source.
	targetURLs := session.Header.CommandArgs[1:]
	var totalBytes int64
	var totalObjects int

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	scanBar := scanBarFactory(sourceURL)
	URLsCh := prepareSyncURLs(sourceURL, targetURLs)
	done := false
	for done == false {
		select {
		case sURLs, ok := <-URLsCh:
			if !ok { // Done with URL prepration
				done = true
				break
			}
			if sURLs.Error != nil {
				console.Errors(ErrorMessage{
					Message: "Failed with",
					Error:   iodine.New(sURLs.Error, nil),
				})
				break
			}

			jsonData, err := json.Marshal(sURLs)
			if err != nil {
				session.Close()
				console.Fatals(ErrorMessage{
					Message: fmt.Sprintf("Unable to marshal URLs to JSON for ‘%s’", sURLs.SourceContent.Name),
					Error:   iodine.New(err, nil),
				})
			}
			fmt.Fprintln(dataFP, string(jsonData))
			scanBar(sURLs.SourceContent.Name)

			totalBytes += sURLs.SourceContent.Size
			totalObjects++
		case <-trapCh:
			session.Close() // If we are interrupted during the URL scanning, we drop the session.
			os.Exit(0)
		}
	}
	session.Header.TotalBytes = totalBytes
	session.Header.TotalObjects = totalObjects
	session.Save()
}

func doSyncCmdSession(session *sessionV2) {
	trapCh := signalTrap(os.Interrupt, os.Kill)

	if !session.HasData() {
		doPrepareSyncURLs(session, trapCh)
	}

	wg := new(sync.WaitGroup)
	syncQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
	defer close(syncQueue)

	scanner := bufio.NewScanner(session.NewDataReader())
	isCopied := isCopiedFactory(session.Header.LastCopied)

	var bar barSend
	if !globalQuietFlag || !globalJSONFlag { // set up progress bar
		bar = newCpBar()
		defer bar.Finish()
		bar.Extend(session.Header.TotalBytes)
	}

	for scanner.Scan() {
		var sURLs syncURLs
		json.Unmarshal([]byte(scanner.Text()), &sURLs)
		if isCopied(sURLs.SourceContent.Name) {
			doSyncFake(sURLs, &bar)
		} else {
			select {
			case syncQueue <- true:
				wg.Add(1)
				go doSync(sURLs, &bar, syncQueue, wg)
				session.Header.LastCopied = sURLs.SourceContent.Name
			case <-trapCh:
				session.Save()
				session.Info()
				os.Exit(0)
			}
		}
	}
	wg.Wait()
}

func runSyncCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "sync", 1) // last argument is exit code
	}

	session := newSessionV2()
	defer session.Close()

	session.Header.CommandType = "sync"
	session.Header.RootPath, _ = os.Getwd()

	// extract URLs.
	var err error
	session.Header.CommandArgs, err = args2URLs(ctx.Args())
	if err != nil {
		session.Close()
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unknown URL types found: ‘%s’", ctx.Args()),
			Error:   iodine.New(err, nil),
		})
	}

	doSyncCmdSession(session)
}
