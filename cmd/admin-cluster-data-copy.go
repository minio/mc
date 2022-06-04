// Copyright (c) 2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/pkg/console"
)

var adminClusterDataCopyFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "skip, s",
		Usage: "number of entries to skip from object listing",
		Value: 0,
	},
	cli.IntFlag{
		Name:  "skip_dm",
		Usage: "number of entries to skip from delete marker listing",
		Value: 0,
	},
	cli.BoolFlag{
		Name:  "fake",
		Usage: "perform a fake copy",
	},
	cli.BoolFlag{
		Name:  "log,l",
		Usage: "enable logging",
	},
	cli.BoolFlag{
		Name:  "replica",
		Usage: "mark as replica on target",
	},
	cli.BoolFlag{
		Name:  "source",
		Usage: "mark as replication source",
	},
}

var adminClusterDataCopyCmd = cli.Command{
	Name:            "copy",
	Usage:           "copy the objects in previously generated listing from source to target cluster",
	Action:          mainClusterDataCopy,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(globalFlags, adminClusterDataCopyFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE/[BUCKET]/[PREFIX] TARGET/[BUCKET] /path/to/list-dir

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Copy the object(s) versions in previously created list from myminio1/mybucket to myminio2/bucket.
    {{.Prompt}} {{.HelpName}} myminio1/mybucket myminio2/bucket /tmp/data

  2. Copy the object(s) versions in previously created list from myminio1/mybucket to myminio2/bucket after skipping
     10000 entries from obj_listing.json and 2000 entries from dm_listing.json.
	{{.Prompt}} {{.HelpName}} myminio/mybucket myminio2/bucket /tmp/data --skip 10000 --skip_dm 2000
`,
}

const (
	failCopyFile = "copy_fails.txt"
	logCopyFile  = "copy_success.txt"
)

func checkDataCopySyntax(ctx *cli.Context) (srcClnt, tgtClnt Client, dataDir string) {
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(ctx, "copy", 1) // last argument is exit code
	}
	var err *probe.Error
	// extract URLs.
	URLs := ctx.Args()
	dataDir = URLs[2]
	srcClnt, err = newClient(URLs[0])
	if err != nil {
		fatalIf(err.Trace(URLs[0]), "Unable to initialize target `"+URLs[0]+"`.")
	}
	tgtClnt, err = newClient(URLs[1])
	if err != nil {
		fatalIf(err.Trace(URLs[1]), "Unable to initialize target `"+URLs[1]+"`.")
	}
	return
}

// mainClusterDataCopy - creates a listing of all object versions in a bucket prefix
func mainClusterDataCopy(cliCtx *cli.Context) error {
	console.SetColor("dataCpMsg", color.New(color.Bold, color.FgHiGreen))

	ctx, cancelCopy := context.WithCancel(globalContext)
	defer cancelCopy()
	cs, err := newcopyState(cliCtx)
	fatalIf(probe.NewError(err).Trace(cliCtx.Args()...), "Unable to copy data")
	cs.start(ctx)

	// queue object listing
	f, ferr := os.Open(path.Join(cs.dataDir, objListing))
	if ferr != nil {
		log.Fatalln("Could not open file path", path.Join(cs.dataDir, objListing), ferr)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		o := scanner.Text()
		if cs.skip > 0 {
			cs.skip--
			continue
		}
		slc := strings.SplitN(o, ",", 4)
		if len(slc) < 3 || len(slc) > 4 {
			cs.logDMsg(fmt.Sprintf("error processing line :%s ", o), nil)
		}
		obj := objInfo{
			bucket:       strings.TrimSpace(slc[0]),
			object:       strings.TrimSpace(slc[1]),
			versionID:    strings.TrimSpace(slc[2]),
			deleteMarker: strings.TrimSpace(slc[3]) == "true",
		}
		cs.queueUploadTask(obj)
		cs.logDMsg(fmt.Sprintf("adding %s to copy queue", o), nil)
	}
	if err := scanner.Err(); err != nil {
		cs.logDMsg(fmt.Sprintf("error processing file :%s ", objListing), err)
		os.Exit(1)
	}

	// queue dm listing
	df, derr := os.Open(path.Join(cs.dataDir, dmListing))
	if derr != nil {
		log.Fatalln("Could not open file path", path.Join(cs.dataDir, dmListing), derr)
	}
	scanner = bufio.NewScanner(df)
	for scanner.Scan() {
		o := scanner.Text()
		if cs.skipDm > 0 {
			cs.skipDm--
			continue
		}
		slc := strings.SplitN(o, ",", 4)
		if len(slc) < 3 || len(slc) > 4 {
			cs.logDMsg(fmt.Sprintf("error processing line :%s ", o), nil)
		}
		obj := objInfo{
			bucket:       strings.TrimSpace(slc[0]),
			object:       strings.TrimSpace(slc[1]),
			versionID:    strings.TrimSpace(slc[2]),
			deleteMarker: strings.TrimSpace(slc[3]) == "true",
		}
		cs.queueUploadTask(obj)
		cs.logDMsg(fmt.Sprintf("adding %s to copy queue", o), nil)
	}
	if err := scanner.Err(); err != nil {
		cs.logDMsg(fmt.Sprintf("error processing file :%s ", objListing), err)
		os.Exit(1)
	}

	cs.finish(ctx)
	return nil
}

type objInfo struct {
	bucket       string
	object       string
	versionID    string
	deleteMarker bool
}

func (i objInfo) String() string {
	return fmt.Sprintf("%s,%s,%s,%t", i.bucket, i.object, i.versionID, i.deleteMarker)
}

type copyState struct {
	objectCh chan objInfo
	failedCh chan copyErr
	logCh    chan objInfo
	wg       sync.WaitGroup
	count    uint64
	failCnt  uint64

	dataDir    string
	fake       bool
	log        bool
	skip       int
	skipDm     int
	srcClnt    Client
	tgtClnt    Client
	srcBucket  string
	tgtBucket  string
	replStatus miniogo.ReplicationStatus
	startTime  time.Time
}

type copyErr struct {
	object objInfo
	err    error
}

var copyConcurrent = 100

func newcopyState(cliCtx *cli.Context) (*copyState, error) {
	// Check for command syntax
	srcClnt, tgtClnt, dataDir := checkDataCopySyntax(cliCtx)
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	srcAliasedURL := filepath.Clean(aliasedURL)
	_, sourceBucket := url2Alias(srcAliasedURL)
	srcVersioned := checkIfBucketIsVersioned(globalContext, srcAliasedURL)

	var replStatus miniogo.ReplicationStatus
	switch {
	case cliCtx.IsSet("replica"):
		replStatus = miniogo.ReplicationStatusReplica
	case cliCtx.IsSet("source"):
		replStatus = miniogo.ReplicationStatusComplete
	}
	aliasedURL = args.Get(1)
	tgtAliasedURL := filepath.Clean(aliasedURL)
	_, tgtBucket := url2Alias(tgtAliasedURL)

	tgtVersioned := checkIfBucketIsVersioned(globalContext, tgtAliasedURL)

	if (srcVersioned || tgtVersioned) && !(srcVersioned && tgtVersioned) {
		return nil, fmt.Errorf("%s and %s should both have versioning enabled", srcAliasedURL, tgtAliasedURL)
	}
	if runtime.GOMAXPROCS(0) > copyConcurrent {
		copyConcurrent = runtime.GOMAXPROCS(0)
	}
	cs := &copyState{
		objectCh: make(chan objInfo, copyConcurrent),
		failedCh: make(chan copyErr, copyConcurrent),
		logCh:    make(chan objInfo, copyConcurrent),
		dataDir:  dataDir,
		fake:     cliCtx.IsSet("fake"),
		log:      cliCtx.IsSet("log"),
		skip:     cliCtx.Int("skip"),
		skipDm:   cliCtx.Int("skip_dm"),

		startTime:  time.Now(),
		srcBucket:  sourceBucket,
		tgtBucket:  tgtBucket,
		replStatus: replStatus,
		srcClnt:    srcClnt,
		tgtClnt:    tgtClnt,
	}
	return cs, nil
}

func (c *copyState) queueUploadTask(obj objInfo) {
	c.objectCh <- obj
}

// Increase count processed
func (c *copyState) incCount() {
	atomic.AddUint64(&c.count, 1)
}

// Get total count processed
func (c *copyState) getCount() uint64 {
	return atomic.LoadUint64(&c.count)
}

// Increase count failed
func (c *copyState) incFailCount() {
	atomic.AddUint64(&c.failCnt, 1)
}

// Get total count failed
func (c *copyState) getFailCount() uint64 {
	return atomic.LoadUint64(&c.failCnt)
}

// addWorker creates a new worker to process tasks
func (c *copyState) addWorker(ctx context.Context) {
	c.wg.Add(1)
	// Add a new worker.
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case obj, ok := <-c.objectCh:
				if !ok {
					return
				}
				c.logDMsg(fmt.Sprintf("Copying...%s", obj), nil)
				if err := c.copyObject(ctx, obj); err != nil {
					c.incFailCount()
					c.logMsg(fmt.Sprintf("error copying object %s: %s", obj, err))
					c.failedCh <- copyErr{object: obj, err: err}
					continue
				}
				c.incCount()
				c.logCh <- obj
			}
		}
	}()
}

func (c *copyState) finish(ctx context.Context) {
	close(c.objectCh)
	c.wg.Wait() // wait on workers to finish
	close(c.failedCh)
	close(c.logCh)
	if c.fake {
		c.logMsg("copy dry run complete")
	} else {
		end := time.Now()
		latency := end.Sub(c.startTime).Seconds()
		syncCnt := c.getCount() - c.getFailCount()
		dm := DataCpMessage{
			op:        "Copy",
			Status:    "success",
			StartTime: c.startTime,
			EndTime:   end,
			InpCnt:    int(c.getCount()),
			FailCnt:   int(c.getFailCount()),
			SyncCnt:   int(syncCnt),
			Latency:   int(latency),
		}
		c.logMsg(fmt.Sprintf("Copied %s / %s objects with latency %d secs", humanize.Comma(int64(syncCnt)), humanize.Comma(int64(c.getCount())), int64(latency)))
		printMsg(dm)
	}
}

func (c *copyState) start(ctx context.Context) {
	if c == nil {
		return
	}
	for i := 0; i < copyConcurrent; i++ {
		c.addWorker(ctx)
	}
	go func() {
		f, err := os.OpenFile(path.Join(c.dataDir, getFileName(failCopyFile, "")), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			c.logDMsg("could not create + copy_fails.txt", err)
			return
		}
		fwriter := bufio.NewWriter(f)
		defer fwriter.Flush()
		defer f.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case o, ok := <-c.failedCh:
				if !ok {
					return
				}
				if _, err := f.WriteString(o.object.String() + " : " + o.err.Error() + "\n"); err != nil {
					c.logMsg(fmt.Sprintf("Error writing to copy_fails.txt for "+o.object.String(), err))
					os.Exit(1)
				}

			}
		}
	}()
	go func() {
		f, err := os.OpenFile(path.Join(c.dataDir, getFileName(logCopyFile, "")), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			c.logDMsg("could not create + copy_log.txt", err)
			return
		}
		fwriter := bufio.NewWriter(f)
		defer fwriter.Flush()
		defer f.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case obj, ok := <-c.logCh:
				if !ok {
					return
				}
				if _, err := f.WriteString(obj.String() + "\n"); err != nil {
					c.logMsg(fmt.Sprintf("Error writing to copy_log.txt for "+obj.String(), err))
					os.Exit(1)
				}

			}
		}
	}()
}

func isMethodNotAllowedErr(err error) bool {
	switch err.Error() {
	case "The specified method is not allowed against this resource.":
		return true
	case "405 Method Not Allowed":
		return true
	}
	return false
}

func (c *copyState) copyObject(ctx context.Context, si objInfo) error {
	srcClient, ok := c.srcClnt.(*S3Client)
	if !ok {
		fatalIf(errDummy().Trace(), fmt.Sprintf("The provided url %s doesn't point to a MinIO server.", c.srcClnt.GetURL().String()))
	}

	tgtClient, ok := c.tgtClnt.(*S3Client)
	if !ok {
		fatalIf(errDummy().Trace(), fmt.Sprintf("The provided url %s doesn't point to a MinIO server.", c.tgtClnt.GetURL().String()))
	}
	tokens := splitStr(c.tgtClnt.GetURL().Path, string(c.tgtClnt.GetURL().Separator), 3)
	tgtBucket := tokens[1]

	obj, err := srcClient.api.GetObject(ctx, si.bucket, si.object, miniogo.GetObjectOptions{
		VersionID: si.versionID,
	})
	if err != nil {
		return err
	}

	oi, err := obj.Stat()
	if err != nil && !(isMethodNotAllowedErr(err) && si.deleteMarker) {
		return err
	}
	defer obj.Close()
	if c.fake {
		c.logMsg(fmt.Sprintf("%s (%s)", oi.Key, oi.VersionID))
		return nil
	}

	if si.deleteMarker {
		_, err = tgtClient.api.StatObject(ctx, tgtBucket, si.object, miniogo.StatObjectOptions{
			VersionID: si.versionID,
		})
		if err.Error() == "The specified key does not exist." {
			rmOpts := miniogo.AdvancedRemoveOptions{
				ReplicationDeleteMarker: si.deleteMarker,
				ReplicationMTime:        oi.LastModified,
				ReplicationRequest:      true, // always set this to distinguish between `mc mirror` replication and serverside
			}
			if c.replStatus != "" {
				rmOpts.ReplicationStatus = c.replStatus
			}
			err := tgtClient.api.RemoveObject(ctx, tgtBucket, si.object, miniogo.RemoveObjectOptions{
				VersionID: si.versionID,
				Internal:  rmOpts,
			})
			if err != nil {
				return err
			}
			_, err = tgtClient.api.StatObject(ctx, tgtBucket, si.object, miniogo.StatObjectOptions{
				VersionID: si.versionID,
			})
			return err
		}
		if isMethodNotAllowedErr(err) {
			c.logDMsg("object already exists on MinIO "+si.object+"("+si.versionID+") not copied", err)
			return nil
		}
		return err
	}
	enc, ok := oi.Metadata[ContentEncoding]
	if !ok {
		enc = oi.Metadata[strings.ToLower(ContentEncoding)]
	}
	pOpts := miniogo.AdvancedPutOptions{
		SourceMTime:     oi.LastModified,
		SourceVersionID: oi.VersionID,
		SourceETag:      oi.ETag,
	}
	if c.replStatus != "" {
		pOpts.ReplicationStatus = c.replStatus
	}
	uoi, err := tgtClient.api.PutObject(ctx, tgtBucket, oi.Key, obj, oi.Size, miniogo.PutObjectOptions{
		Internal:        pOpts,
		UserMetadata:    oi.UserMetadata,
		ContentType:     oi.ContentType,
		StorageClass:    oi.StorageClass,
		UserTags:        oi.UserTags,
		ContentEncoding: strings.Join(enc, ","),
	})
	if err != nil {
		c.logDMsg("upload to minio failed for "+oi.Key, err)
		return err
	}
	if uoi.Size != oi.Size {
		err = fmt.Errorf("expected size %d, uploaded %d", oi.Size, uoi.Size)
		c.logDMsg("upload to minio failed for "+oi.Key, err)
		return err
	}
	c.logDMsg("Uploaded "+uoi.Key+" successfully", nil)
	return nil
}

const (
	// ContentEncoding http header
	ContentEncoding = "Content-Encoding"
)

func (c *copyState) logMsg(msg string) {
	if c.log {
		fmt.Println(msg)
	}
}

func getFileName(fname, prefix string) string {
	if prefix == "" {
		return fmt.Sprintf("%s%s", fname, time.Now().Format(".01-02-2006-15-04-05"))
	}
	return fmt.Sprintf("%s_%s%s", fname, prefix, time.Now().Format(".01-02-2006-15-04-05"))
}

// log debug statements
func (c *copyState) logDMsg(msg string, err error) {
	if globalDebug {
		if err == nil {
			fmt.Println(msg)
			return
		}
		fmt.Println(msg, " :", err)
	}
}

// DataCpMessage container for content message structure
type DataCpMessage struct {
	op        string
	Status    string    `json:"status"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	InpCnt    int       `json:"input_count"`
	SyncCnt   int       `json:"synced_count"`
	FailCnt   int       `json:"fail_count"`
	Latency   int       `json:"latency"`
}

func (d DataCpMessage) String() string {
	message := console.Colorize("dataCpMsg", fmt.Sprintf("Copied %s / %s objects with latency %d secs", humanize.Comma(int64(d.SyncCnt)), humanize.Comma(int64(d.InpCnt)), int64(d.Latency)))
	return message
}

// JSON returns jsonified message
func (d DataCpMessage) JSON() string {
	d.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(d, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}
