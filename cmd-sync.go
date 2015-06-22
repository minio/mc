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
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"runtime"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/countlock"
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

// doSyncSession - Sync an object to multiple destination
func doSyncSession(sURLs syncURLs, bar *barSend, syncQueue chan bool, ssCh chan syncSession, wg *sync.WaitGroup, s *sessionV1) {
	defer wg.Done()
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if !globalQuietFlag {
		bar.SetCaption(sURLs.SourceContent.Name + ": ")
	}
	_, ok := s.Files[sURLs.SourceContent.Name]
	if ok {
		if !globalQuietFlag {
			bar.ErrorGet(int64(sURLs.SourceContent.Size))
		}
		<-syncQueue // Signal that this copy routine is done.
		return
	}
	reader, length, err := getSource(sURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorGet(int64(length))
		}
		ssCh <- syncSession{
			Error: iodine.New(err, map[string]string{"URL": sURLs.SourceContent.Name}),
			Done:  false,
		}
	}
	defer reader.Close()

	var targetURLs []string
	for _, targetContent := range sURLs.TargetContents {
		targetURLs = append(targetURLs, targetContent.Name)
	}

	var newReader io.Reader
	switch globalQuietFlag {
	case true:
		newReader = yielder.NewReader(reader)
	default:
		// set up progress
		newReader = bar.NewProxyReader(yielder.NewReader(reader))
	}
	for err := range putTargets(targetURLs, length, newReader) {
		if err != nil {
			if !globalQuietFlag {
				bar.ErrorPut(int64(length))
			}
			ssCh <- syncSession{
				Error: iodine.New(err, nil),
				Done:  false,
			}
		}
	}
	<-syncQueue // Signal that this copy routine is done.
	// store files which have finished copying
	s.Files[sURLs.SourceContent.Name] = true
}

func doPrepareSyncURLs(sourceURL string, targetURLs []string, bar barSend, lock countlock.Locker) {
	for sURLs := range prepareSyncURLs(sourceURL, targetURLs) {
		if sURLs.Error != nil {
			// no need to print errors here, any error here
			// will be printed later during Sync()
			continue
		}
		if !globalQuietFlag {
			bar.Extend(sURLs.SourceContent.Size)
			lock.Up() // Let copy routine know that it has to catch up.
		}
	}
}

type syncSession struct {
	Error error
	Done  bool
}

func trapSync(ssCh chan syncSession) {
	// Go signal notification works by sending `os.Signal`
	// values on a channel.
	sigs := make(chan os.Signal, 1)

	// `signal.Notify` registers the given channel to
	// receive notifications of the specified signals.
	signal.Notify(sigs, os.Kill, os.Interrupt)

	// This executes a blocking receive for signals.
	// When it gets one it'll then notify the program
	// that it can finish.
	<-sigs
	ssCh <- syncSession{
		Error: nil,
		Done:  true,
	}
}

func doSyncCmdSession(bar barSend, s *sessionV1) <-chan syncSession {
	ssCh := make(chan syncSession)
	// Separate source and target. 'sync' can take only one source.
	// but any number of targets, even the recursive URLs mixed in-between.
	sourceURL := s.URLs[0] // first one is source
	targetURLs := s.URLs[1:]

	go func(sourceURL string, targetURLs []string, bar barSend, ssCh chan syncSession, s *sessionV1) {
		defer close(ssCh)
		go trapSync(ssCh)

		var lock countlock.Locker
		if !globalQuietFlag {
			// Keep progress-bar and copy routines in sync.
			lock = countlock.New()
			defer lock.Close()
		}

		wg := new(sync.WaitGroup)
		syncQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
		defer close(syncQueue)

		go doPrepareSyncURLs(sourceURL, targetURLs, bar, lock)

		for sURLs := range prepareSyncURLs(sourceURL, targetURLs) {
			if sURLs.Error != nil {
				ssCh <- syncSession{
					Error: iodine.New(sURLs.Error, nil),
					Done:  false,
				}
				continue
			}
			syncQueue <- true
			wg.Add(1)
			if !globalQuietFlag {
				lock.Down() // Do not jump ahead of the progress bar builder above.
			}
			go doSyncSession(sURLs, &bar, syncQueue, ssCh, wg, s)
		}
		wg.Wait()
	}(sourceURL, targetURLs, bar, ssCh, s)
	return ssCh
}

func runSyncCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "sync", 1) // last argument is exit code
	}

	if !isMcConfigExist() {
		console.Fatals(ErrorMessage{
			Message: "Please run \"mc config generate\"",
			Error:   iodine.New(errNotConfigured{}, nil),
		})
	}

	if !isSessionDirExists() {
		if err := createSessionDir(); err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}
	}

	s, err := newSession()
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: "Failed with",
			Error:   iodine.New(err, nil),
		})
	}
	s.CommandType = "sync"

	// extract URLs.
	s.URLs, err = args2URLs(ctx.Args())
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unknown URL types found: ‘%s’", ctx.Args()),
			Error:   iodine.New(err, nil),
		})
	}

	var bar barSend
	// set up progress bar
	if !globalQuietFlag {
		bar = newCpBar()
	}

	for ss := range doSyncCmdSession(bar, s) {
		if ss.Error != nil {
			console.Errors(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(ss.Error, nil),
			})
		}
		if ss.Done {
			if err := saveSession(s); err != nil {
				console.Fatals(ErrorMessage{
					Message: "Failed wtih",
					Error:   iodine.New(err, nil),
				})
			}
			console.Infos(InfoMessage{
				Message: "\nSession terminated. To resume session type ‘mc session resume " + s.SessionID + "’",
			})
			// this os.Exit is needed really to exit in-case of "os.Interrupt"
			os.Exit(0)
		}
	}

	if !globalQuietFlag {
		bar.Finish()
		// ignore any error returned here
		clearSession(s.SessionID)
	}
}
