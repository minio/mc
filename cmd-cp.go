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

// doCopySession - Copy a singe file from source to destination
func doCopySession(cURLs cpURLs, bar *barSend, s *sessionV1) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if !globalQuietFlag {
		bar.SetCaption(cURLs.SourceContent.Name + ": ")
	}

	_, ok := s.Files[cURLs.SourceContent.Name]
	if ok {
		bar.ErrorGet(int64(cURLs.SourceContent.Size))
		return nil
	}

	reader, length, err := getSource(cURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorGet(int64(length))
		}
		return iodine.New(err, map[string]string{"URL": cURLs.SourceContent.Name})
	}
	defer reader.Close()

	var newReader io.Reader
	switch globalQuietFlag || globalJSONFlag {
	case true:
		console.Infos(CopyMessage{
			Source: cURLs.SourceContent.Name,
			Target: cURLs.TargetContent.Name,
			Length: cURLs.SourceContent.Size,
		})
		newReader = yielder.NewReader(reader)
	default:
		// set up progress
		newReader = bar.NewProxyReader(yielder.NewReader(reader))
	}

	err = putTarget(cURLs.TargetContent.Name, length, newReader)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorPut(int64(length))
		}
		return iodine.New(err, map[string]string{"URL": cURLs.TargetContent.Name})
	}

	// store files which have finished copying
	s.Files[cURLs.SourceContent.Name] = true
	return nil
}

type cpSession struct {
	Error error
	Done  bool
}

func doCopyInRoutineSession(cURLs cpURLs, bar *barSend, cpQueue chan bool, cpsCh chan cpSession, wg *sync.WaitGroup, s *sessionV1) {
	defer wg.Done()
	if err := doCopySession(cURLs, bar, s); err != nil {
		cpsCh <- cpSession{
			Error: iodine.New(err, nil),
			Done:  false,
		}
	}
	// Signal that this copy routine is done.
	<-cpQueue
}

func doPrepareCopyURLs(sourceURLs []string, targetURL string, bar barSend, lock countlock.Locker) {
	for cURLs := range prepareCopyURLs(sourceURLs, targetURL) {
		if cURLs.Error != nil {
			// no need to print errors here, any error here
			// will be printed later during Copy()
			continue
		}
		if !globalQuietFlag {
			bar.Extend(cURLs.SourceContent.Size)
			lock.Up() // Let copy routine know that it has to catch up.
		}
	}
}

func trapCp(cpsCh chan cpSession) {
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
	cpsCh <- cpSession{
		Error: nil,
		Done:  true,
	}
}

func doCopyCmdSession(bar barSend, s *sessionV1) <-chan cpSession {
	// Separate source and target. 'cp' can take only one target,
	// but any number of sources, even the recursive URLs mixed in-between.
	sourceURLs := s.URLs[:len(s.URLs)-1]
	targetURL := s.URLs[len(s.URLs)-1] // Last one is target

	cpsCh := make(chan cpSession)
	go func(sourceURLs []string, targetURL string, bar barSend, cpsCh chan cpSession, s *sessionV1) {
		defer close(cpsCh)
		go trapCp(cpsCh)

		var lock countlock.Locker
		if !globalQuietFlag {
			// Keep progress-bar and copy routines in sync.
			lock = countlock.New()
			defer lock.Close()
		}

		// Wait for all copy routines to complete.
		wg := new(sync.WaitGroup)

		// Pool limited copy routines in parallel.
		cpQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
		defer close(cpQueue)

		go doPrepareCopyURLs(sourceURLs, targetURL, bar, lock)

		for cURLs := range prepareCopyURLs(sourceURLs, targetURL) {
			if cURLs.Error != nil {
				cpsCh <- cpSession{
					Error: cURLs.Error,
					Done:  false,
				}
				continue
			}
			cpQueue <- true // Wait for existing pool to drain.
			wg.Add(1)       // keep track of all the goroutines
			if !globalQuietFlag {
				lock.Down() // Do not jump ahead of the progress bar builder above.
			}
			go doCopyInRoutineSession(cURLs, &bar, cpQueue, cpsCh, wg, s)
		}
		wg.Wait() // wait for the go routines to complete
	}(sourceURLs, targetURL, bar, cpsCh, s)
	return cpsCh
}

// runCopyCmd is bound to sub-command
func runCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}

	if !isMcConfigExists() {
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
	s.CommandType = "cp"
	s.RootPath, _ = os.Getwd()

	// extract URLs.
	s.URLs, err = args2URLs(ctx.Args())
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unknown URL types: ‘%s’", ctx.Args()),
			Error:   iodine.New(err, nil),
		})
	}

	var bar barSend
	// set up progress bar
	if !globalQuietFlag {
		bar = newCpBar()
	}

	for cps := range doCopyCmdSession(bar, s) {
		if cps.Error != nil {
			console.Errors(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(cps.Error, nil),
			})
		}
		if cps.Done {
			if err := saveSession(s); err != nil {
				console.Fatals(ErrorMessage{
					Message: "Failed with",
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
