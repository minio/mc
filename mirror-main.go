/*
 * Minio Client, (C) 2015, 2016 Minio, Inc.
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
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
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
		cli.BoolFlag{
			Name:  "fake",
			Usage: "Perform a fake mirror operation.",
		},
		cli.BoolFlag{
			Name:  "remove",
			Usage: "Remove extraneous file(s) on target.",
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
      $ mc {{.Name}} play/photos/2014 s3/backup-photos

   2. Mirror a local folder recursively to Amazon S3 cloud storage.
      $ mc {{.Name}} backup/ s3/archive

   3. Mirror a bucket from aliased Amazon S3 cloud storage to a folder on Windows.
      $ mc {{.Name}} s3\documents\2014\ C:\backup\2014

   4. Mirror a bucket from aliased Amazon S3 cloud storage to a local folder use '--force' to overwrite destination.
      $ mc {{.Name}} --force s3/miniocloud miniocloud-backup

   5. Fake mirror a bucket from Minio cloud storage to a bucket on Amazon S3 cloud storage.
      $ mc {{.Name}} --fake play/photos/2014 s3/backup-photos/2014

   6. Mirror a bucket from Minio cloud storage to a bucket on Amazon S3 cloud storage and remove any extraneous
      files on Amazon S3 cloud storage. NOTE: '--remove' is only supported with '--force'.
      $ mc {{.Name}} --force --remove play/photos/2014 s3/backup-photos/2014
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
	speedBox := pb.Format(int64(c.Speed)).To(pb.U_BYTES).String()
	if speedBox == "" {
		speedBox = "0 MB"
	} else {
		speedBox = speedBox + "/s"
	}
	message := fmt.Sprintf("Total: %s, Transferred: %s, Speed: %s", pb.Format(c.Total).To(pb.U_BYTES),
		pb.Format(c.Transferred).To(pb.U_BYTES), speedBox)
	return message
}

// doRemove - removes files on target.
func doRemove(sURLs mirrorURLs, fakeRemove bool) mirrorURLs {
	targetAlias := sURLs.TargetAlias
	targetURL := sURLs.TargetContent.URL

	// We are not removing incomplete uploads.
	isIncomplete := false

	// Remove extraneous file on target.
	err := rm(targetAlias, targetURL.String(), isIncomplete, fakeRemove)
	if err != nil {
		sURLs.Error = err.Trace(targetAlias, targetURL.String())
		return sURLs
	}

	sURLs.Error = nil // just for safety
	return sURLs
}

// doMirror - Mirror an object to multiple destination. mirrorURLs status contains a copy of sURLs and error if any.
func doMirror(sURLs mirrorURLs, progressReader *progressBar, accountingReader *accounter, fakeMirror bool) mirrorURLs {
	if sURLs.Error != nil { // Errorneous sURLs passed.
		sURLs.Error = sURLs.Error.Trace()
		return sURLs
	}

	sourceAlias := sURLs.SourceAlias
	sourceURL := sURLs.SourceContent.URL
	targetAlias := sURLs.TargetAlias
	targetURL := sURLs.TargetContent.URL
	length := sURLs.SourceContent.Size

	if !globalQuiet && !globalJSON {
		progressReader = progressReader.SetCaption(sourceURL.String() + ": ")
	}

	var progress io.Reader
	if globalQuiet || globalJSON {
		sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
		targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))
		printMsg(mirrorMessage{
			Source: sourcePath,
			Target: targetPath,
		})
		if globalQuiet || globalJSON {
			progress = accountingReader
		}
	} else {
		// Set up progress bar.
		progress = progressReader.ProgressBar
	}

	// For a fake mirror make sure we update respective progress bars
	// and accounting readers under relevant conditions.
	if fakeMirror {
		if !globalJSON && !globalQuiet {
			progressReader.ProgressBar.Add64(sURLs.SourceContent.Size)
		} else {
			accountingReader.Add(sURLs.SourceContent.Size)
		}
		sURLs.Error = nil
		return sURLs
	}
	// If source size is <= 5GB and operation is across same server type try to use Copy.
	if length <= fiveGB && sourceURL.Type == targetURL.Type {
		// FS -> FS Copy includes alias in path.
		if sourceURL.Type == fileSystem {
			sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
			err := copySourceStreamFromAlias(targetAlias, targetURL.String(), sourcePath, length, progress)
			if err != nil {
				sURLs.Error = err.Trace(sourceURL.String())
				return sURLs
			}
		} else if sourceURL.Type == objectStorage {
			if sourceAlias == targetAlias {
				// If source/target are object storage their aliases must be the same
				// Do not include alias inside path for ObjStore -> ObjStore.
				err := copySourceStreamFromAlias(targetAlias, targetURL.String(), sourceURL.Path, length, progress)
				if err != nil {
					sURLs.Error = err.Trace(sourceURL.String())
					return sURLs
				}
			} else {
				reader, err := getSourceStreamFromAlias(sourceAlias, sourceURL.String())
				if err != nil {
					sURLs.Error = err.Trace(sourceURL.String())
					return sURLs
				}
				_, err = putTargetStreamFromAlias(targetAlias, targetURL.String(), reader, length, progress)
				if err != nil {
					sURLs.Error = err.Trace(targetURL.String())
					return sURLs
				}
			}
		}
	} else {
		// Standard GET/PUT for size > 5GB
		reader, err := getSourceStreamFromAlias(sourceAlias, sourceURL.String())
		if err != nil {
			sURLs.Error = err.Trace(sourceURL.String())
			return sURLs
		}
		_, err = putTargetStreamFromAlias(targetAlias, targetURL.String(), reader, length, progress)
		if err != nil {
			sURLs.Error = err.Trace(targetURL.String())
			return sURLs
		}
	}
	sURLs.Error = nil // just for safety
	return sURLs
}

// doPrepareMirrorURLs scans the source URL and prepares a list of objects for mirroring.
func doPrepareMirrorURLs(session *sessionV7, isForce bool, isFake bool, isRemove bool, trapCh <-chan bool) {
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

	URLsCh := prepareMirrorURLs(sourceURL, targetURL, isForce, isFake, isRemove)
	done := false
	for !done {
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
			jsonData, e := json.Marshal(sURLs)
			if e != nil {
				session.Delete()
				fatalIf(probe.NewError(e), "Unable to marshal URLs into JSON.")
			}
			fmt.Fprintln(dataFP, string(jsonData))
			if !globalQuiet && !globalJSON {
				// Source content is empty if removal is requested,
				// put targetContent on to scan bar.
				if sURLs.SourceContent != nil {
					scanBar(sURLs.SourceContent.URL.String())
				} else if sURLs.TargetContent != nil && isRemove {
					scanBar(sURLs.TargetContent.URL.String())
				}
			}
			// Remember totalBytes only for mirror not for removal,
			if sURLs.SourceContent != nil {
				totalBytes += sURLs.SourceContent.Size
			}
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
func doMirrorSession(session *sessionV7) {
	isForce := session.Header.CommandBoolFlags["force"]
	isFake := session.Header.CommandBoolFlags["fake"]
	isRemove := session.Header.CommandBoolFlags["remove"]

	// Initialize signal trap.
	trapCh := signalTrap(os.Interrupt, syscall.SIGTERM)

	if !session.HasData() {
		doPrepareMirrorURLs(session, isForce, isFake, isRemove, trapCh)
	}

	// Enable accounting reader by default.
	accntReader := newAccounter(session.Header.TotalBytes)

	// Set up progress bar.
	var progressReader *progressBar
	if !globalQuiet && !globalJSON {
		progressReader = newProgressBar(session.Header.TotalBytes)
	}

	// Prepare URL scanner from session data file.
	urlScanner := bufio.NewScanner(session.NewDataReader())

	// isCopied returns true if an object has been already copied
	// or not. This is useful when we resume from a session.
	isCopied := isLastFactory(session.Header.LastCopied)

	// isRemoved returns true if an object has been already removed or
	// not. This is useful when we resume from a session.
	isRemoved := isLastFactory(session.Header.LastRemoved)

	// Wait on status of doMirror() operation.
	var statusCh = make(chan mirrorURLs)

	// Add a wait group for the below go-routine.
	var wg = new(sync.WaitGroup)
	wg.Add(1)

	// Go routine to monitor signal traps if any.
	go func() {
		defer wg.Done()
		for {
			select {
			case <-trapCh:
				// Receive interrupt notification.
				if !globalQuiet && !globalJSON {
					console.Eraseline()
				}
				session.CloseAndDie()
			case sURLs, ok := <-statusCh:
				// Status channel is closed, we should return.
				if !ok {
					return
				}
				if sURLs.Error == nil {
					if sURLs.SourceContent != nil {
						session.Header.LastCopied = sURLs.SourceContent.URL.String()
						session.Save()
					} else if sURLs.TargetContent != nil && isRemove {
						session.Header.LastRemoved = sURLs.TargetContent.URL.String()
						session.Save()

						// Construct user facing message and path.
						targetPath := filepath.ToSlash(filepath.Join(sURLs.TargetAlias, sURLs.TargetContent.URL.Path))
						if !globalQuiet && !globalJSON {
							console.Eraseline()
						}
						printMsg(rmMessage{
							Status: "success",
							URL:    targetPath,
						})
					}
				} else {
					// Print in new line and adjust to top so that we
					// don't print over the ongoing progress bar.
					if !globalQuiet && !globalJSON {
						console.Eraseline()
					}
					if sURLs.SourceContent != nil {
						errorIf(sURLs.Error.Trace(sURLs.SourceContent.URL.String()),
							fmt.Sprintf("Failed to copy ‘%s’.", sURLs.SourceContent.URL.String()))
					} else {
						// When sURLs.SourceContent is nil, we know that we have an error related to removing
						errorIf(sURLs.Error.Trace(sURLs.TargetContent.URL.String()),
							fmt.Sprintf("Failed to remove ‘%s’.", sURLs.TargetContent.URL.String()))
					}
					// For all non critical errors we can continue for the
					// remaining files.
					switch sURLs.Error.ToGoError().(type) {
					// Handle this specifically for filesystem related errors.
					case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, PathInsufficientPermission:
						continue
					case BucketNameEmpty, ObjectMissing, ObjectAlreadyExists, BucketDoesNotExist, BucketInvalid:
						continue
					}
					// For critical errors we should exit. Session
					// can be resumed after the user figures out
					// the problem.
					session.CloseAndDie()
				}
			}
		}
	}()

	// Loop through all urls and mirror.
	for urlScanner.Scan() {
		var sURLs mirrorURLs

		// Unmarshal copyURLs from each line.
		json.Unmarshal([]byte(urlScanner.Text()), &sURLs)

		if sURLs.SourceContent != nil {
			// Verify if previously copied or if its a fake mirror, set
			// fake mirror accordingly.
			fakeMirror := isCopied(sURLs.SourceContent.URL.String()) || isFake
			// Perform mirror operation.
			statusCh <- doMirror(sURLs, progressReader, accntReader, fakeMirror)
		} else if sURLs.TargetContent != nil && isRemove {
			fakeRemove := isRemoved(sURLs.TargetContent.URL.String()) || isFake
			// Perform remove operation.
			statusCh <- doRemove(sURLs, fakeRemove)
		}
	}

	// Close the goroutine.
	close(statusCh)

	// Wait for the goroutines to finish.
	wg.Wait()

	if !globalQuiet && !globalJSON {
		if progressReader.ProgressBar.Get() > 0 {
			progressReader.ProgressBar.Finish()
		}
	} else {
		accntStat := accntReader.Stat()
		mrStatMessage := mirrorStatMessage{
			Total:       accntStat.Total,
			Transferred: accntStat.Transferred,
			Speed:       accntStat.Speed,
		}
		console.Println(console.Colorize("Mirror", mrStatMessage.String()))
	}
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
	session := newSessionV7()
	session.Header.CommandType = "mirror"
	session.Header.RootPath, e = os.Getwd()
	if e != nil {
		session.Delete()
		fatalIf(probe.NewError(e), "Unable to get current working folder.")
	}

	// Set command flags from context.
	isForce := ctx.Bool("force")
	isFake := ctx.Bool("fake")
	isRemove := ctx.Bool("remove")
	session.Header.CommandBoolFlags["force"] = isForce
	session.Header.CommandBoolFlags["fake"] = isFake
	session.Header.CommandBoolFlags["remove"] = isRemove

	// extract URLs.
	session.Header.CommandArgs = ctx.Args()
	doMirrorSession(session)
	session.Delete()
}
