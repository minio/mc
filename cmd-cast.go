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
)

// Help message.
var castCmd = cli.Command{
	Name:   "cast",
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
   1. Cast an object from local filesystem to Amazon S3 object storage.
      $ mc {{.Name}} star-trek-episode-10-season4.ogg https://s3.amazonaws.com/trekarchive

   2. Cast a bucket recursively from Minio object storage to multiple buckets on Amazon S3 object storage.
      $ mc {{.Name}} https://play.minio.io:9000/photos/2014... https://s3.amazonaws.com/backup-photos https://s3-west-1.amazonaws.com/local-photos

   3. Cast a local folder recursively to Minio object storage and Amazon S3 object storage.
      $ mc {{.Name}} backup/... https://play.minio.io:9000/archive https://s3.amazonaws.com/archive

   4. Cast a bucket from aliased Amazon S3 object storage to multiple folders on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014 C:\shared\volume\backup\2014

   5. Cast a local directory of non english character recursively to Amazon s3 object storage and Minio object storage.
      $ mc {{.Name}} 本語/... s3:mylocaldocuments play:backup

`,
}

// doCast - Cast an object to multiple destination
func doCast(sURLs castURLs, bar *barSend, castQueue chan bool, wg *sync.WaitGroup) (err error) {
	defer wg.Done() // Notify that this copy routine is done.
	defer func() {
		<-castQueue
	}()

	if !globalQuietFlag || !globalJSONFlag {
		bar.SetCaption(sURLs.SourceContent.Name + ": ")
	}

	reader, length, err := getSource(sURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag || !globalJSONFlag {
			bar.ErrorGet(int64(length))
		}
		console.Errorf("Unable to read from source %s. %s\n", sURLs.SourceContent.Name, err)
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

	for err := range putTargets(targetURLs, length, newReader) {
		if err != nil {
			if !globalQuietFlag || !globalJSONFlag {
				bar.ErrorPut(int64(length))
			}
			console.Errorf("Unable to write to one or more destinations. %s\n", err)
		}
	}
	return nil
}

// doCastFake - Perform a fake cast to update the progress bar appropriately.
func doCastFake(sURLs castURLs, bar *barSend) (err error) {
	if !globalDebugFlag || !globalJSONFlag {
		bar.Progress(sURLs.SourceContent.Size)
	}
	return nil
}

// doPrepareCastURLs scans the source URL and prepares a list of objects for casting.
func doPrepareCastURLs(session *sessionV2, trapCh <-chan bool) {
	sourceURL := session.Header.CommandArgs[0] // first one is source.
	targetURLs := session.Header.CommandArgs[1:]
	var totalBytes int64
	var totalObjects int

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	scanBar := scanBarFactory(sourceURL)
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
			if !globalForceFlag {
				if len(sURLs.TargetContents) > 0 {
					if sURLs.TargetContents[0].Size != 0 || !sURLs.TargetContents[0].Time.IsZero() {
						console.Fatalf("Destination already exists, cannot overwrite ‘%s’ with ‘%s’. "+
							"Use ‘--force’ flag to override.\n", sURLs.TargetContents[0].Name, sURLs.SourceContent.Name)
						break
					}
				}
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

	wg := new(sync.WaitGroup)
	castQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))

	scanner := bufio.NewScanner(session.NewDataReader())
	isCopied := isCopiedFactory(session.Header.LastCopied)

	var bar barSend
	if !globalQuietFlag || !globalJSONFlag { // set up progress bar
		bar = newCpBar()
		defer bar.Finish()
		bar.Extend(session.Header.TotalBytes)
	}

	for scanner.Scan() {
		var sURLs castURLs
		json.Unmarshal([]byte(scanner.Text()), &sURLs)
		if isCopied(sURLs.SourceContent.Name) {
			doCastFake(sURLs, &bar)
		} else {
			select {
			case castQueue <- true:
				wg.Add(1)
				go doCast(sURLs, &bar, castQueue, wg)
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

func runCastCmd(ctx *cli.Context) {
	checkCastSyntax(ctx)

	session := newSessionV2()
	defer session.Close()

	var err error
	session.Header.CommandType = "cast"
	session.Header.RootPath, err = os.Getwd()
	if err != nil {
		session.Close()
		console.Fatalf("Unable to get current working directory. %s\n", err)
	}

	// extract URLs.
	session.Header.CommandArgs, err = args2URLs(ctx.Args())
	if err != nil {
		session.Close()
		console.Fatalf("One or more unknown URL types found in %s. %s\n", ctx.Args(), err)
	}

	doCastCmdSession(session)
}
