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
	"syscall"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/pb"
)

// cp command flags.
var (
	cpFlagRecursive = cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "Copy recursively.",
	}
	cpFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of cp.",
	}
)

// Copy command.
var cpCmd = cli.Command{
	Name:   "cp",
	Usage:  "Copy one or more objects to a target.",
	Action: mainCopy,
	Flags:  append(globalFlags, cpFlagRecursive, cpFlagHelp),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] SOURCE [SOURCE...] TARGET

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Copy a list of objects from local file system to Amazon S3 cloud storage.
      $ mc {{.Name}} Music/*.ogg s3.amazonaws.com/jukebox/

   2. Copy a folder recursively from Minio cloud storage to Amazon S3 cloud storage.
      $ mc {{.Name}} --recursive play.minio.io:9000/mybucket/burningman2011/ s3.amazonaws.com/mybucket/

   3. Copy multiple local folders recursively to Minio cloud storage.
      $ mc {{.Name}} --recursive backup/2014/ backup/2015/ play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 cloud storage to local filesystem on Windows.
      $ mc {{.Name}} --recursive s3/documents/2014/ C:\Backups\2014

   5. Copy an object with name containing unicode characters to Amazon S3 cloud storage.
      $ mc {{.Name}} 本語 s3/andoria/

   6. Copy a local folder with space separated characters to Amazon S3 cloud storage.
      $ mc {{.Name}} --recursive 'workdir/documents/May 2014/' s3/miniocloud
`,
}

// copyMessage container for file copy messages
type copyMessage struct {
	Status string `json:"status"`
	Source string `json:"source"`
	Target string `json:"target"`
	Length int64  `json:"length"`
}

// String colorized copy message
func (c copyMessage) String() string {
	return console.Colorize("Copy", fmt.Sprintf("‘%s’ -> ‘%s’", c.Source, c.Target))
}

// JSON jsonified copy message
func (c copyMessage) JSON() string {
	c.Status = "success"
	copyMessageBytes, err := json.Marshal(c)
	fatalIf(probe.NewError(err), "Failed to marshal copy message.")

	return string(copyMessageBytes)
}

// copyStatMessage container for copy accounting message
type copyStatMessage struct {
	Total       int64
	Transferred int64
	Speed       float64
}

// copyStatMessage copy accounting message
func (c copyStatMessage) String() string {
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

// doCopy - Copy a singe file from source to destination
func doCopy(cpURLs copyURLs, progressReader *barSend, accountingReader *accounter, cpQueue <-chan bool, wg *sync.WaitGroup, statusCh chan<- copyURLs) {
	defer wg.Done() // Notify that this copy routine is done.
	defer func() {
		<-cpQueue
	}()

	if cpURLs.Error != nil {
		cpURLs.Error.Trace()
		statusCh <- cpURLs
		return
	}

	if !globalQuiet && !globalJSON {
		progressReader.SetCaption(cpURLs.SourceContent.URL.String() + ": ")
	}

	reader, length, err := getSource(cpURLs.SourceContent.URL.String())
	if err != nil {
		if !globalQuiet && !globalJSON {
			progressReader.ErrorGet(length)
		}
		cpURLs.Error = err.Trace(cpURLs.SourceContent.URL.String())
		statusCh <- cpURLs
		return
	}

	var newReader io.ReadCloser
	if globalQuiet || globalJSON {
		printMsg(copyMessage{
			Source: cpURLs.SourceContent.URL.String(),
			Target: cpURLs.TargetContent.URL.String(),
			Length: cpURLs.SourceContent.Size,
		})
		// No accounting necessary for JSON output.
		if globalJSON {
			newReader = reader
		}
		// Proxy reader to accounting reader only during quiet mode.
		if globalQuiet {
			newReader = accountingReader.NewProxyReader(reader)
		}
	} else {
		// set up progress
		newReader = progressReader.NewProxyReader(reader)
	}
	defer newReader.Close()

	if err := putTarget(cpURLs.TargetContent.URL.String(), length, newReader); err != nil {
		if !globalQuiet && !globalJSON {
			progressReader.ErrorPut(length)
		}
		cpURLs.Error = err.Trace(cpURLs.TargetContent.URL.String())
		statusCh <- cpURLs
		return
	}

	cpURLs.Error = nil // just for safety
	statusCh <- cpURLs
}

// doCopyFake - Perform a fake copy to update the progress bar appropriately.
func doCopyFake(cURLs copyURLs, progressReader *barSend) {
	if !globalQuiet && !globalJSON {
		progressReader.Progress(cURLs.SourceContent.Size)
	}
}

// doPrepareCopyURLs scans the source URL and prepares a list of objects for copying.
func doPrepareCopyURLs(session *sessionV5, trapCh <-chan bool) {
	// Separate source and target. 'cp' can take only one target,
	// but any number of sources.
	sourceURLs := session.Header.CommandArgs[:len(session.Header.CommandArgs)-1]
	targetURL := session.Header.CommandArgs[len(session.Header.CommandArgs)-1] // Last one is target

	var totalBytes int64
	var totalObjects int

	// Access recursive flag inside the session header.
	isRecursive := session.Header.CommandBoolFlags["recursive"]

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	var scanBar scanBarFunc
	if !globalQuiet && !globalJSON { // set up progress bar
		scanBar = scanBarFactory()
	}

	URLsCh := prepareCopyURLs(sourceURLs, targetURL, isRecursive)
	done := false

	for done == false {
		select {
		case cpURLs, ok := <-URLsCh:
			if !ok { // Done with URL prepration
				done = true
				break
			}
			if cpURLs.Error != nil {
				// Print in new line and adjust to top so that we don't print over the ongoing scan bar
				if !globalQuiet && !globalJSON {
					console.Eraseline()
				}
				if strings.Contains(cpURLs.Error.ToGoError().Error(), " is a folder.") {
					errorIf(cpURLs.Error.Trace(), "Folder cannot be copied. Please use ‘...’ suffix.")
				} else {
					errorIf(cpURLs.Error.Trace(), "Unable to prepare URL for copying.")
				}
				break
			}

			jsonData, err := json.Marshal(cpURLs)
			if err != nil {
				session.Delete()
				fatalIf(probe.NewError(err), "Unable to prepare URL for copying. Error in JSON marshaling.")
			}
			fmt.Fprintln(dataFP, string(jsonData))
			if !globalQuiet && !globalJSON {
				scanBar(cpURLs.SourceContent.URL.String())
			}

			totalBytes += cpURLs.SourceContent.Size
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

func doCopySession(session *sessionV5) {
	trapCh := signalTrap(os.Interrupt, syscall.SIGTERM)

	if !session.HasData() {
		doPrepareCopyURLs(session, trapCh)
	}

	// Enable accounting reader by default.
	accntReader := newAccounter(session.Header.TotalBytes)

	// Enable progress bar reader only during default mode.
	var progressReader *barSend
	if !globalQuiet && !globalJSON { // set up progress bar
		progressReader = newProgressBar(session.Header.TotalBytes)
	}

	// Prepare URL scanner from session data file.
	scanner := bufio.NewScanner(session.NewDataReader())
	// isCopied returns true if an object has been already copied
	// or not. This is useful when we resume from a session.
	isCopied := isCopiedFactory(session.Header.LastCopied)

	wg := new(sync.WaitGroup)
	// Limit number of copy routines based on available CPU resources.
	cpQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
	defer close(cpQueue)

	// Status channel for receiveing copy return status.
	statusCh := make(chan copyURLs)

	// Go routine to monitor doCopy status and signal traps.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case cpURLs, ok := <-statusCh: // Receive status.
				if !ok { // We are done here. Top level function has returned.
					if !globalQuiet && !globalJSON {
						progressReader.Finish()
					}
					if globalQuiet {
						accntStat := accntReader.Stat()
						cpStatMessage := copyStatMessage{
							Total:       accntStat.Total,
							Transferred: accntStat.Transferred,
							Speed:       accntStat.Speed,
						}
						console.Println(console.Colorize("Copy", cpStatMessage.String()))
					}
					return
				}
				if cpURLs.Error == nil {
					session.Header.LastCopied = cpURLs.SourceContent.URL.String()
					session.Save()
				} else {
					// Print in new line and adjust to top so that we don't print over the ongoing progress bar
					if !globalQuiet && !globalJSON {
						console.Eraseline()
					}
					errorIf(cpURLs.Error.Trace(cpURLs.SourceContent.URL.String()),
						fmt.Sprintf("Failed to copy ‘%s’.", cpURLs.SourceContent.URL.String()))
					// for all non critical errors we can continue for the remaining files
					switch cpURLs.Error.ToGoError().(type) {
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
				if !globalQuiet && !globalJSON {
					console.Eraseline()
				}
				session.CloseAndDie()
			}
		}
	}()

	// Go routine to perform concurrently copying.
	wg.Add(1)
	go func() {
		defer wg.Done()
		copyWg := new(sync.WaitGroup)
		defer close(statusCh)

		for scanner.Scan() {
			var cpURLs copyURLs
			json.Unmarshal([]byte(scanner.Text()), &cpURLs)
			if isCopied(cpURLs.SourceContent.URL.String()) {
				doCopyFake(cpURLs, progressReader)
			} else {
				// Wait for other copy routines to
				// complete. We only have limited CPU
				// and network resources.
				cpQueue <- true
				// Account for each copy routines we start.
				copyWg.Add(1)
				// Do copying in background concurrently.
				go doCopy(cpURLs, progressReader, accntReader, cpQueue, copyWg, statusCh)
			}
		}
		copyWg.Wait()
	}()
	wg.Wait()
}

// mainCopy is the entry point for cp command.
func mainCopy(ctx *cli.Context) {
	checkCopySyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Copy", color.New(color.FgGreen, color.Bold))

	// Set global flags from context.
	setGlobalsFromContext(ctx)

	session := newSessionV5()
	session.Header.CommandType = "cp"
	session.Header.CommandBoolFlags["recursive"] = ctx.Bool("recursive")

	var e error
	if session.Header.RootPath, e = os.Getwd(); e != nil {
		session.Delete()
		fatalIf(probe.NewError(e), "Unable to get current working folder.")
	}

	// extract URLs.
	var err *probe.Error
	if session.Header.CommandArgs, err = args2URLs(ctx.Args()); err != nil {
		session.Delete()
		fatalIf(err.Trace(ctx.Args()...), "One or more unknown URL types passed.")
	}

	doCopySession(session)
	session.Delete()
}
