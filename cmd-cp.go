/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"strings"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var cpCmd = cli.Command{
	Name:   "cp",
	Usage:  "Copy files and folders from many sources to a single destination",
	Action: runCopyCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE [SOURCE...] TARGET {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Copy list of objects from local file system to Amazon S3 object storage.
      $ mc {{.Name}} Music/*.ogg https://s3.amazonaws.com/jukebox/

   2. Copy a bucket recursively from Minio object storage to Amazon S3 object storage.
      $ mc {{.Name}} https://play.minio.io:9000/photos/burningman2011... https://s3.amazonaws.com/private-photos/burningman/

   3. Copy multiple local folders recursively to Minio object storage.
      $ mc {{.Name}} backup/2014/... backup/2015/... https://play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 object storage to local filesystem on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014

   5. Copy an object of non english characters to Amazon S3 object storage.
      $ mc {{.Name}} 本語 s3:andoria/本語

`,
}

// doCopy - Copy a singe file from source to destination
func doCopy(cpURLs copyURLs, bar *barSend, cpQueue chan bool, wg *sync.WaitGroup) error {
	defer wg.Done() // Notify that this copy routine is done.
	defer func() {
		<-cpQueue
	}()

	if !globalQuietFlag || !globalJSONFlag {
		bar.SetCaption(cpURLs.SourceContent.Name + ": ")
	}

	reader, length, err := getSource(cpURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag || !globalJSONFlag {
			bar.ErrorGet(length)
		}
		return NewIodine(iodine.New(err, map[string]string{"URL": cpURLs.SourceContent.Name}))
	}

	var newReader io.ReadCloser
	if globalQuietFlag || globalJSONFlag {
		console.Infoln(CopyMessage{
			Source: cpURLs.SourceContent.Name,
			Target: cpURLs.TargetContent.Name,
			Length: cpURLs.SourceContent.Size,
		})
		newReader = reader
	} else {
		// set up progress
		newReader = bar.NewProxyReader(reader)
	}
	defer newReader.Close()

	err = putTarget(cpURLs.TargetContent.Name, length, newReader)
	if err != nil {
		if !globalQuietFlag || !globalJSONFlag {
			bar.ErrorPut(length)
		}
		console.Errorln(NewIodine(err))
	}
	return nil
}

// doCopyFake - Perform a fake copy to update the progress bar appropriately.
func doCopyFake(sURLs copyURLs, bar *barSend) (err error) {
	if !globalQuietFlag || !globalJSONFlag {
		bar.Progress(sURLs.SourceContent.Size)
	}
	return nil
}

// doPrepareCopyURLs scans the source URL and prepares a list of objects for copying.
func doPrepareCopyURLs(session *sessionV2, trapCh <-chan bool) {
	// Separate source and target. 'cp' can take only one target,
	// but any number of sources, even the recursive URLs mixed in-between.
	sourceURLs := session.Header.CommandArgs[:len(session.Header.CommandArgs)-1]
	targetURL := session.Header.CommandArgs[len(session.Header.CommandArgs)-1] // Last one is target

	var totalBytes int64
	var totalObjects int

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()
	scanBar := scanBarFactory(strings.Join(sourceURLs, " "))
	URLsCh := prepareCopyURLs(sourceURLs, targetURL)
	done := false

	for done == false {
		select {
		case cpURLs, ok := <-URLsCh:
			if !ok { // Done with URL prepration
				done = true
				break
			}
			if cpURLs.Error != nil {
				console.Errorln(cpURLs.Error)
				break
			}

			jsonData, err := json.Marshal(cpURLs)
			if err != nil {
				session.Close()
				console.Fatalf("Unable to marshal URLs to JSON. %s\n", err)
			}
			fmt.Fprintln(dataFP, string(jsonData))
			scanBar(cpURLs.SourceContent.Name)

			totalBytes += cpURLs.SourceContent.Size
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

func doCopyCmdSession(session *sessionV2) {
	trapCh := signalTrap(os.Interrupt, os.Kill)

	if !session.HasData() {
		doPrepareCopyURLs(session, trapCh)
	}

	wg := new(sync.WaitGroup)
	cpQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
	defer close(cpQueue)

	scanner := bufio.NewScanner(session.NewDataReader())
	isCopied := isCopiedFactory(session.Header.LastCopied)

	var bar barSend
	if !globalQuietFlag || !globalJSONFlag { // set up progress bar
		bar = newCpBar()
		defer bar.Finish()
		bar.Extend(session.Header.TotalBytes)
	}

	for scanner.Scan() {
		var cpURLs copyURLs
		json.Unmarshal([]byte(scanner.Text()), &cpURLs)
		if isCopied(cpURLs.SourceContent.Name) {
			doCopyFake(cpURLs, &bar)
		} else {
			select {
			case cpQueue <- true:
				wg.Add(1)
				go doCopy(cpURLs, &bar, cpQueue, wg)
				session.Header.LastCopied = cpURLs.SourceContent.Name
			case <-trapCh:
				session.Save()
				session.Info()
				os.Exit(0)
			}
		}
	}
	wg.Wait()
}

// runCopyCmd is bound to sub-command
func runCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}

	session := newSessionV2()
	defer session.Close()

	var err error
	session.Header.CommandType = "cp"
	session.Header.RootPath, err = os.Getwd()
	if err != nil {
		session.Close()
		console.Fatalf("Unable to get current working directory. %s\n", err)
	}

	// extract URLs.
	session.Header.CommandArgs, err = args2URLs(ctx.Args())
	if err != nil {
		session.Close()
		console.Fatalf("One or more unknown URL types found %s. %s\n", ctx.Args(), err)
	}

	doCopyCmdSession(session)
}
