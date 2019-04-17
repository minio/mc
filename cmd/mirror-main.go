/*
 * MinIO Client, (C) 2015, 2016 MinIO, Inc.
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

package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// mirror specific flags.
var (
	mirrorFlags = []cli.Flag{
		cli.BoolFlag{
			Name:   "force",
			Usage:  "force allows forced overwrite or removal of object(s) on target",
			Hidden: true, // Hidden since this option is deprecated.
		},
		cli.BoolFlag{
			Name:  "overwrite",
			Usage: "overwrite object(s) on target",
		},
		cli.BoolFlag{
			Name:  "fake",
			Usage: "perform a fake mirror operation",
		},
		cli.BoolFlag{
			Name:  "watch, w",
			Usage: "watch and synchronize changes",
		},
		cli.BoolFlag{
			Name:  "remove",
			Usage: "remove extraneous object(s) on target",
		},
		cli.StringFlag{
			Name:  "region",
			Usage: "specify region when creating new bucket(s) on target",
			Value: "us-east-1",
		},
		cli.BoolFlag{
			Name:  "a",
			Usage: "preserve bucket policy rules on target bucket(s)",
		},
		cli.StringSliceFlag{
			Name:  "exclude",
			Usage: "exclude object(s) that match specified object name pattern",
		},
		cli.StringFlag{
			Name:  "older-than",
			Usage: "filter object(s) older than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "newer-than",
			Usage: "filter object(s) newer than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "storage-class, sc",
			Usage: "specify storage class for new object(s) on target",
		},
		cli.StringFlag{
			Name:  "encrypt",
			Usage: "encrypt/decrypt objects (using server-side encryption with server managed keys)",
		},
	}
)

//  Mirror folders recursively from a single source to many destinations
var mirrorCmd = cli.Command{
	Name:   "mirror",
	Usage:  "synchronize object(s) to a remote site",
	Action: mainMirror,
	Before: setGlobalsFromContext,
	Flags:  append(append(mirrorFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
   MC_ENCRYPT:      list of comma delimited prefixes
   MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

EXAMPLES:
   1. Mirror a bucket recursively from MinIO cloud storage to a bucket on Amazon S3 cloud storage.
      $ {{.HelpName}} play/photos/2014 s3/backup-photos

   2. Mirror a local folder recursively to Amazon S3 cloud storage.
      $ {{.HelpName}} backup/ s3/archive

   3. Only mirror files that are newer than 7 days, 10 hours and 30 minutes to Amazon S3 cloud storage.
      $ {{.HelpName}} --newer-than "7d10h30m" backup/ s3/archive

   4. Mirror a bucket from aliased Amazon S3 cloud storage to a folder on Windows.
      $ {{.HelpName}} s3\documents\2014\ C:\backup\2014

   5. Mirror a bucket from aliased Amazon S3 cloud storage to a local folder use '--overwrite' to overwrite destination.
      $ {{.HelpName}} --overwrite s3/miniocloud miniocloud-backup

   6. Mirror a bucket from MinIO cloud storage to a bucket on Amazon S3 cloud storage and remove any extraneous
      files on Amazon S3 cloud storage.
      $ {{.HelpName}} --remove play/photos/2014 s3/backup-photos/2014

   7. Continuously mirror a local folder recursively to MinIO cloud storage. '--watch' continuously watches for
      new objects, uploads and removes extraneous files on Amazon S3 cloud storage.
      $ {{.HelpName}} --remove --watch /var/lib/backups play/backups

   8. Mirror a bucket from aliased Amazon S3 cloud storage to a local folder.
      Exclude all .* files and *.temp files when mirroring.
      $ {{.HelpName}} --exclude ".*" --exclude "*.temp" s3/test ~/test

   9. Mirror objects newer than 10 days from bucket test to a local folder.
      $ {{.HelpName}} --newer-than 10d s3/test ~/localfolder

  10. Mirror objects older than 30 days from Amazon S3 bucket test to a local folder.
      $ {{.HelpName}} --older-than 30d s3/test ~/test

  11. Mirror server encrypted objects from MinIO cloud storage to a bucket on Amazon S3 cloud storage
      $ {{.HelpName}} --encrypt-key "minio/photos=32byteslongsecretkeymustbegiven1,s3/archive=32byteslongsecretkeymustbegiven2" minio/photos/ s3/archive/
`,
}

type mirrorJob struct {

	// the channel to trap SIGKILL signals
	trapCh <-chan bool

	// mutex for shutdown, this prevents the shutdown
	// to be initiated multiple times
	m *sync.Mutex

	// channel for errors
	errorCh chan *probe.Error

	// the global watcher object, which receives notifications of created
	// and deleted files
	watcher *Watcher

	// Hold operation status information
	status Status

	queueCh  chan func() URLs
	parallel *ParallelManager

	// channel for status messages
	statusCh chan URLs

	TotalObjects int64
	TotalBytes   int64

	sourceURL string
	targetURL string

	isFake, isRemove, isOverwrite, isWatch bool
	olderThan, newerThan                   string
	storageClass                           string

	excludeOptions []string
	encKeyDB       map[string][]prefixSSEPair
}

// mirrorMessage container for file mirror messages
type mirrorMessage struct {
	Status     string `json:"status"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Size       int64  `json:"size"`
	TotalCount int64  `json:"totalCount"`
	TotalSize  int64  `json:"totalSize"`
}

// String colorized mirror message
func (m mirrorMessage) String() string {
	return console.Colorize("Mirror", fmt.Sprintf("`%s` -> `%s`", m.Source, m.Target))
}

// JSON jsonified mirror message
func (m mirrorMessage) JSON() string {
	m.Status = "success"
	mirrorMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(mirrorMessageBytes)
}

// doRemove - removes files on target.
func (mj *mirrorJob) doRemove(sURLs URLs) URLs {
	if mj.isFake {
		return sURLs.WithError(nil)
	}

	// Construct proper path with alias.
	targetWithAlias := filepath.Join(sURLs.TargetAlias, sURLs.TargetContent.URL.Path)
	clnt, pErr := newClient(targetWithAlias)
	if pErr != nil {
		return sURLs.WithError(pErr)
	}

	contentCh := make(chan *clientContent, 1)
	contentCh <- &clientContent{URL: *newClientURL(sURLs.TargetContent.URL.Path)}
	close(contentCh)
	isRemoveBucket := false
	errorCh := clnt.Remove(false, isRemoveBucket, contentCh)
	for pErr := range errorCh {
		if pErr != nil {
			switch pErr.ToGoError().(type) {
			case PathInsufficientPermission:
				// Ignore Permission error.
				continue
			}
			return sURLs.WithError(pErr)
		}
	}

	return sURLs.WithError(nil)
}

// doMirror - Mirror an object to multiple destination. URLs status contains a copy of sURLs and error if any.
func (mj *mirrorJob) doMirror(ctx context.Context, cancelMirror context.CancelFunc, sURLs URLs) URLs {

	if sURLs.Error != nil { // Erroneous sURLs passed.
		return sURLs.WithError(sURLs.Error.Trace())
	}

	//s For a fake mirror make sure we update respective progress bars
	// and accounting readers under relevant conditions.
	if mj.isFake {
		mj.status.Add(sURLs.SourceContent.Size)
		return sURLs.WithError(nil)
	}

	sourceAlias := sURLs.SourceAlias
	sourceURL := sURLs.SourceContent.URL
	targetAlias := sURLs.TargetAlias
	targetURL := sURLs.TargetContent.URL
	length := sURLs.SourceContent.Size

	mj.status.SetCaption(sourceURL.String() + ": ")

	if mj.storageClass != "" {
		if sURLs.TargetContent.Metadata == nil {
			sURLs.TargetContent.Metadata = make(map[string]string)
		}
		sURLs.TargetContent.Metadata["X-Amz-Storage-Class"] = mj.storageClass
	}

	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
	targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))
	mj.status.PrintMsg(mirrorMessage{
		Source:     sourcePath,
		Target:     targetPath,
		Size:       length,
		TotalCount: sURLs.TotalCount,
		TotalSize:  sURLs.TotalSize,
	})
	return uploadSourceToTargetURL(ctx, sURLs, mj.status, mj.encKeyDB)
}

// Update progress status
func (mj *mirrorJob) monitorMirrorStatus() (errDuringMirror bool) {
	// now we want to start the progress bar
	mj.status.Start()
	defer mj.status.Finish()

	for sURLs := range mj.statusCh {
		if sURLs.Error != nil {
			switch {
			case sURLs.SourceContent != nil:
				if !isErrIgnored(sURLs.Error) {
					errorIf(sURLs.Error.Trace(sURLs.SourceContent.URL.String()),
						fmt.Sprintf("Failed to copy `%s`.", sURLs.SourceContent.URL.String()))
					errDuringMirror = true
				}
			case sURLs.TargetContent != nil:
				// When sURLs.SourceContent is nil, we know that we have an error related to removing
				errorIf(sURLs.Error.Trace(sURLs.TargetContent.URL.String()),
					fmt.Sprintf("Failed to remove `%s`.", sURLs.TargetContent.URL.String()))
				errDuringMirror = true
			default:
				errorIf(sURLs.Error.Trace(), "Failed to perform mirroring action.")
				errDuringMirror = true
			}
		}

		if sURLs.SourceContent != nil {
		} else if sURLs.TargetContent != nil {
			// Construct user facing message and path.
			targetPath := filepath.ToSlash(filepath.Join(sURLs.TargetAlias, sURLs.TargetContent.URL.Path))
			size := sURLs.TargetContent.Size
			mj.status.PrintMsg(rmMessage{Key: targetPath, Size: size})
		}
	}

	return
}

// this goroutine will watch for notifications, and add modified objects to the queue
func (mj *mirrorJob) watchMirror(ctx context.Context, cancelMirror context.CancelFunc) {
	for {
		select {
		case event, ok := <-mj.watcher.Events():
			if !ok {
				return
			}

			// It will change the expanded alias back to the alias
			// again, by replacing the sourceUrlFull with the sourceAlias.
			// This url will be used to mirror.
			sourceAlias, sourceURLFull, _ := mustExpandAlias(mj.sourceURL)

			// If the passed source URL points to fs, fetch the absolute src path
			// to correctly calculate targetPath
			if sourceAlias == "" {
				tmpSrcURL, err := filepath.Abs(sourceURLFull)
				if err == nil {
					sourceURLFull = tmpSrcURL
				}
			}
			eventPath := event.Path

			if runtime.GOOS == "darwin" {
				// Strip the prefixes in the event path. Happens in darwin OS only
				eventPath = eventPath[strings.Index(eventPath, sourceURLFull):len(eventPath)]
			}

			sourceURL := newClientURL(eventPath)
			// trim trailing slash from source url
			sourceURLStr := strings.TrimSuffix(sourceURLFull, string(sourceURL.Separator))
			aliasedPath := strings.Replace(eventPath, sourceURLStr, mj.sourceURL, -1)

			// build target path, it is the relative of the eventPath with the sourceUrl
			// joined to the targetURL.
			sourceSuffix := strings.TrimPrefix(eventPath, sourceURLFull)
			//Skip the object, if it matches the Exclude options provided
			if matchExcludeOptions(mj.excludeOptions, sourceSuffix) {
				continue
			}

			targetPath := urlJoinPath(mj.targetURL, sourceSuffix)

			// newClient needs the unexpanded  path, newCLientURL needs the expanded path
			targetAlias, expandedTargetPath, _ := mustExpandAlias(targetPath)
			targetURL := newClientURL(expandedTargetPath)
			sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))
			srcSSE := getSSE(sourcePath, mj.encKeyDB[sourceAlias])
			tgtSSE := getSSE(targetPath, mj.encKeyDB[targetAlias])

			if event.Type == EventCreate {
				// we are checking if a destination file exists now, and if we only
				// overwrite it when force is enabled.
				mirrorURL := URLs{
					SourceAlias:   sourceAlias,
					SourceContent: &clientContent{URL: *sourceURL},
					TargetAlias:   targetAlias,
					TargetContent: &clientContent{URL: *targetURL},
					encKeyDB:      mj.encKeyDB,
				}
				if event.Size == 0 {
					sourceClient, err := newClient(aliasedPath)
					if err != nil {
						// cannot create sourceclient
						mj.statusCh <- mirrorURL.WithError(err)
						continue
					}
					sourceContent, err := sourceClient.Stat(false, false, srcSSE)
					if err != nil {
						// source doesn't exist anymore
						mj.statusCh <- mirrorURL.WithError(err)
						continue
					}
					targetClient, err := newClient(targetPath)
					if err != nil {
						// cannot create targetclient
						mj.statusCh <- mirrorURL.WithError(err)
						return
					}
					shouldQueue := false
					if !mj.isOverwrite {
						_, err = targetClient.Stat(false, false, tgtSSE)
						if err == nil {
							continue
						} // doesn't exist
						shouldQueue = true
					}
					if shouldQueue || mj.isOverwrite {
						mirrorURL.TotalCount = mj.TotalObjects
						mirrorURL.TotalSize = mj.TotalBytes
						// adjust total, because we want to show progress of the item still queued to be copied.
						mj.status.SetTotal(mj.status.Total() + sourceContent.Size).Update()
						mj.statusCh <- mj.doMirror(ctx, cancelMirror, mirrorURL)
					}
					continue
				}
				shouldQueue := false
				if !mj.isOverwrite {
					targetClient, err := newClient(targetPath)
					if err != nil {
						// cannot create targetclient
						mj.statusCh <- mirrorURL.WithError(err)
						return
					}
					_, err = targetClient.Stat(false, false, tgtSSE)
					if err == nil {
						continue
					} // doesn't exist
					shouldQueue = true
				}
				if shouldQueue || mj.isOverwrite {
					mirrorURL.SourceContent.Size = event.Size
					mirrorURL.TotalCount = mj.TotalObjects
					mirrorURL.TotalSize = mj.TotalBytes
					// adjust total, because we want to show progress of the itemj stiil queued to be copied.
					mj.status.SetTotal(mj.status.Total() + event.Size).Update()
					mj.statusCh <- mj.doMirror(ctx, cancelMirror, mirrorURL)
				}
			} else if event.Type == EventRemove {
				mirrorURL := URLs{
					SourceAlias:   sourceAlias,
					SourceContent: nil,
					TargetAlias:   targetAlias,
					TargetContent: &clientContent{URL: *targetURL},
					encKeyDB:      mj.encKeyDB,
				}
				mirrorURL.TotalCount = mj.TotalObjects
				mirrorURL.TotalSize = mj.TotalBytes
				if mirrorURL.TargetContent != nil && mj.isRemove {
					mj.statusCh <- mj.doRemove(mirrorURL)
				}
			}

		case err := <-mj.watcher.Errors():
			switch err.ToGoError().(type) {
			case APINotImplemented:
				errorIf(err.Trace(), "Unable to Watch on source, ignoring.")
				return
			}
			mj.statusCh <- URLs{Error: err}
			return
		case <-mj.trapCh:
			return
		}
	}
}

func (mj *mirrorJob) watchSourceURL(recursive bool) *probe.Error {
	sourceClient, err := newClient(mj.sourceURL)
	if err == nil {
		return mj.watcher.Join(sourceClient, recursive)
	} // Failed to initialize client.
	return err
}

func (mj *mirrorJob) watchURL(sourceClient Client) *probe.Error {
	return mj.watcher.Join(sourceClient, true)
}

// Fetch urls that need to be mirrored
func (mj *mirrorJob) startMirror(ctx context.Context, cancelMirror context.CancelFunc) {
	var totalBytes int64
	var totalObjects int64

	stopParallel := func() {
		close(mj.queueCh)
		mj.parallel.wait()
	}

	URLsCh := prepareMirrorURLs(mj.sourceURL, mj.targetURL, mj.isFake, mj.isOverwrite, mj.isRemove, mj.excludeOptions, mj.encKeyDB)

	for {
		select {
		case sURLs, ok := <-URLsCh:
			if !ok {
				stopParallel()
				return
			}
			if sURLs.Error != nil {
				stopParallel()
				mj.statusCh <- sURLs
				return
			}

			if sURLs.SourceContent != nil {
				if mj.olderThan != "" && isOlder(sURLs.SourceContent.Time, mj.olderThan) {
					continue
				}
				if mj.newerThan != "" && isNewer(sURLs.SourceContent.Time, mj.newerThan) {
					continue
				}
				// copy
				totalBytes += sURLs.SourceContent.Size
			}

			totalObjects++
			mj.TotalBytes = totalBytes
			mj.TotalObjects = totalObjects
			mj.status.SetTotal(totalBytes)

			// Save total count.
			sURLs.TotalCount = mj.TotalObjects
			// Save totalSize.
			sURLs.TotalSize = mj.TotalBytes

			if sURLs.SourceContent != nil {
				mj.queueCh <- func() URLs {
					return mj.doMirror(ctx, cancelMirror, sURLs)
				}
			} else if sURLs.TargetContent != nil && mj.isRemove {
				mj.queueCh <- func() URLs {
					return mj.doRemove(sURLs)
				}
			}
		case <-mj.trapCh:
			stopParallel()
			cancelMirror()
			return
		}
	}
}

// when using a struct for copying, we could save a lot of passing of variables
func (mj *mirrorJob) mirror(ctx context.Context, cancelMirror context.CancelFunc) bool {

	var wg sync.WaitGroup

	// Starts watcher loop for watching for new events.
	if mj.isWatch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mj.watchMirror(ctx, cancelMirror)
		}()
	}

	// Start mirroring.
	wg.Add(1)
	go func() {
		defer wg.Done()
		mj.startMirror(ctx, cancelMirror)
	}()

	// Close statusCh when both watch & mirror quits
	go func() {
		wg.Wait()
		close(mj.statusCh)
	}()

	return mj.monitorMirrorStatus()
}

func newMirrorJob(srcURL, dstURL string, isFake, isRemove, isOverwrite, isWatch bool, excludeOptions []string, olderThan, newerThan string, storageClass string, encKeyDB map[string][]prefixSSEPair) *mirrorJob {
	mj := mirrorJob{
		trapCh: signalTrap(os.Interrupt, syscall.SIGTERM, syscall.SIGKILL),
		m:      new(sync.Mutex),

		sourceURL: srcURL,
		targetURL: dstURL,

		isFake:         isFake,
		isRemove:       isRemove,
		isOverwrite:    isOverwrite,
		isWatch:        isWatch,
		excludeOptions: excludeOptions,
		olderThan:      olderThan,
		newerThan:      newerThan,
		storageClass:   storageClass,
		encKeyDB:       encKeyDB,
		statusCh:       make(chan URLs),
		watcher:        NewWatcher(UTCNow()),
	}

	mj.parallel, mj.queueCh = newParallelManager(mj.statusCh)

	// we'll define the status to use here,
	// do we want the quiet status? or the progressbar
	var status = NewProgressStatus(mj.parallel)
	if globalQuiet {
		status = NewQuietStatus(mj.parallel)
	} else if globalJSON {
		status = NewDummyStatus(mj.parallel)
	}
	mj.status = status

	return &mj
}

// copyBucketPolicies - copy policies from source to dest
func copyBucketPolicies(srcClt, dstClt Client, isOverwrite bool) *probe.Error {
	rules, err := srcClt.GetAccessRules()
	if err != nil {
		return err
	}
	// Set found rules to target bucket if permitted
	for _, r := range rules {
		originalRule, _, err := dstClt.GetAccess()
		if err != nil {
			return err
		}
		// Set rule only if it doesn't exist in the target bucket
		// or force flag is activated
		if originalRule == "none" || isOverwrite {
			err = dstClt.SetAccess(r, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// runMirror - mirrors all buckets to another S3 server
func runMirror(srcURL, dstURL string, ctx *cli.Context, encKeyDB map[string][]prefixSSEPair) bool {
	// This is kept for backward compatibility, `--force` means
	// --overwrite.
	isOverwrite := ctx.Bool("force")
	if !isOverwrite {
		isOverwrite = ctx.Bool("overwrite")
	}

	// Create a new mirror job and execute it
	mj := newMirrorJob(srcURL, dstURL,
		ctx.Bool("fake"),
		ctx.Bool("remove"),
		isOverwrite,
		ctx.Bool("watch"),
		ctx.StringSlice("exclude"),
		ctx.String("older-than"),
		ctx.String("newer-than"),
		ctx.String("storage-class"),
		encKeyDB)

	srcClt, err := newClient(srcURL)
	fatalIf(err, "Unable to initialize `"+srcURL+"`.")

	dstClt, err := newClient(dstURL)
	fatalIf(err, "Unable to initialize `"+srcURL+"`.")

	if ctx.Bool("a") && (srcClt.GetURL().Type != objectStorage || dstClt.GetURL().Type != objectStorage) {
		fatalIf(errDummy(), "Synchronizing bucket policies is only possible when both source & target point to S3 servers.")
	}

	mirrorAllBuckets := (srcClt.GetURL().Type == objectStorage && srcClt.GetURL().Path == "/") ||
		(dstClt.GetURL().Type == objectStorage && dstClt.GetURL().Path == "/")

	if mirrorAllBuckets {
		// Synchronize buckets using dirDifference function
		for d := range dirDifference(srcClt, dstClt, srcURL, dstURL) {
			if d.Error != nil {
				mj.status.fatalIf(d.Error, fmt.Sprintf("Failed to start monitoring."))
			}
			if d.Diff == differInSecond {
				// Ignore buckets that only exist in target instance
				continue
			}

			sourceSuffix := strings.TrimPrefix(d.FirstURL, srcClt.GetURL().String())
			newSrcURL := path.Join(srcURL, sourceSuffix)
			newTgtURL := path.Join(dstURL, sourceSuffix)

			newSrcClt, _ := newClient(newSrcURL)
			newDstClt, _ := newClient(newTgtURL)

			if d.Diff == differInFirst {
				// Bucket only exists in the source, create the same bucket in the destination
				if err := newDstClt.MakeBucket(ctx.String("region"), false); err != nil {
					errorIf(err, "Cannot created bucket in `"+newTgtURL+"`.")
					continue
				}
				// Copy policy rules from source to dest if flag is activated
				if ctx.Bool("a") {
					if err := copyBucketPolicies(srcClt, dstClt, isOverwrite); err != nil {
						errorIf(err, "Cannot copy bucket policies to `"+newDstClt.GetURL().String()+"`.")
					}
				}
			}

			if mj.isWatch {
				// monitor mode will watch the source folders for changes,
				// and queue them for copying.
				if err := mj.watchURL(newSrcClt); err != nil {
					mj.status.fatalIf(err, fmt.Sprintf("Failed to start monitoring."))
				}
			}
		}
	}

	if !mirrorAllBuckets && mj.isWatch {
		// monitor mode will watch the source folders for changes,
		// and queue them for copying.
		if err := mj.watchURL(srcClt); err != nil {
			mj.status.fatalIf(err, fmt.Sprintf("Failed to start monitoring."))
		}
	}

	ctxt, cancelMirror := context.WithCancel(context.Background())
	defer cancelMirror()

	// Start mirroring job
	return mj.mirror(ctxt, cancelMirror)
}

// Main entry point for mirror command.
func mainMirror(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check 'mirror' cli arguments.
	checkMirrorSyntax(ctx, encKeyDB)

	// Additional command specific theme customization.
	console.SetColor("Mirror", color.New(color.FgGreen, color.Bold))

	args := ctx.Args()

	srcURL := args[0]
	tgtURL := args[1]

	if errorDetected := runMirror(srcURL, tgtURL, ctx, encKeyDB); errorDetected {
		return exitStatus(globalErrorExitStatus)
	}

	return nil
}
