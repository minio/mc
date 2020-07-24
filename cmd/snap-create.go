/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var (
	snapCreateFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "rewind",
			Usage: "Rewind to the state of the data in the specified time",
		},
		cli.BoolFlag{
			Name:  "overwrite",
			Usage: "Allow overwriting snapshots",
		},
	}
)

var snapCreate = cli.Command{
	Name:   "create",
	Usage:  "Create a new snapshot from an S3 deployment",
	Action: mainSnapCreate,
	Before: setGlobalsFromContext,
	Flags:  append(snapCreateFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} SNAPSHOT-NAME S3-PATH

EXAMPLES:
  1. Create a new snapshot from an S3 server
      {{.Prompt}} {{.HelpName}} my-snapshot-name s3/

  2. Create a new snapshot of a particular bucket with the state of 24 hours earlier
      {{.Prompt}} {{.HelpName}} my-snapshot-name s3/mybucket/ --rewind 24h
`,
}

// createSnapMsg container for snap creation message structure
type createSnapMsg struct {
	Status       string `json:"success"`
	SnapshotName string `json:"snapshot"`
}

func (r createSnapMsg) String() string {
	return console.Colorize("SnapCreation", "The snapshot `"+r.SnapshotName+"` is created.")
}

func (r createSnapMsg) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// validate command-line args.
func checkSnapCreateSyntax(cliCtx *cli.Context) (snapName string, url string, refTime time.Time) {
	var perr *probe.Error

	args := cliCtx.Args()
	if len(args) != 2 {
		cli.ShowCommandHelpAndExit(cliCtx, "create", globalErrorExitStatus)
	}

	snapName = args.Get(0)
	targetURL := args.Get(1)
	_, perr = newClient(targetURL)
	fatalIf(perr.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")

	refTime = parseRewindFlag(cliCtx.String("rewind"))
	if refTime.IsZero() {
		refTime = time.Now().UTC()
	}

	return snapName, targetURL, refTime
}

func createSnapshotFile(snapName string, overwrite bool) (*os.File, *probe.Error) {
	snapsDir, perr := getSnapsDir()
	if perr != nil {
		return nil, perr
	}

	err := os.MkdirAll(snapsDir, 0700)
	if err != nil {
		return nil, probe.NewError(err)
	}
	snapFile := filepath.Join(snapsDir, snapName)
	if !strings.HasSuffix(snapFile, snapshotSuffix) {
		snapFile += ".snap"
	}

	if !overwrite {
		if _, err := os.Stat(snapFile); err == nil {
			return nil, probe.NewError(fmt.Errorf("%q already exists. Use --overwrite to allow overwriting", snapName))
		}
	}

	f, e := os.OpenFile(snapFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	return f, probe.NewError(e)
}

func openSnapshotFile(snapName string) (*os.File, *probe.Error) {
	snapsDir, perr := getSnapsDir()
	if perr != nil {
		return nil, perr
	}

	snapFile := filepath.Join(snapsDir, snapName)
	if !strings.HasSuffix(snapFile, snapshotSuffix) {
		snapFile += ".snap"
	}
	if _, err := os.Stat(snapFile); err != nil {
		return nil, probe.NewError(err)
	}
	f, e := os.Open(snapFile)
	return f, probe.NewError(e)
}

func createSnapshot(snapName string, s3Path string, at time.Time, overwrite bool) (perr *probe.Error) {
	alias, urlStr, hostCfg, err := expandAlias(s3Path)
	if err != nil {
		return err
	}

	s3Client, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return err
	}
	var out io.Writer
	if snapName == "-" {
		out = os.Stdout
	} else {
		f, err := createSnapshotFile(snapName, overwrite)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	ser, err := newSnapshotSerializer(out)
	if err != nil {
		return err
	}
	err = ser.AddTarget(S3Target(*hostCfg))
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			perr = probe.NewError(fmt.Errorf("panic during encode: %v", r))
		}
		if perr == nil {
			perr = ser.FinishTarget()
		}
	}()

	var entries chan<- SnapshotEntry
	var currentBucket string
	for s := range s3Client.List(context.Background(), ListOptions{timeRef: at, isRecursive: true}) {
		if s.Err != nil {
			return s.Err
		}
		if ser.HasError() {
			close(entries)
			return probe.NewError(ser.GetAsyncErr())
		}
		bucket, key := s.URL.BucketAndPrefix()
		if currentBucket != bucket || entries == nil {
			if entries != nil {
				close(entries)
			}
			// Switch to new.
			entries, err = ser.StartBucket(SnapshotBucket{Name: bucket})
			if err != nil {
				return err
			}
			currentBucket = bucket
		}

		entries <- SnapshotEntry{
			Key:            key,
			VersionID:      s.VersionID,
			Size:           s.Size,
			ModTime:        s.Time,
			ETag:           s.ETag,
			StorageClass:   s.StorageClass,
			IsDeleteMarker: s.IsDeleteMarker,
			IsLatest:       s.IsLatest,
		}
	}
	if entries != nil {
		close(entries)
	}
	return probe.NewError(ser.GetAsyncErr())
}

// main entry point for snapshot create.
func mainSnapCreate(ctx *cli.Context) error {
	console.SetColor("SnapCreation", color.New(color.FgGreen))

	// Validate command-line args.
	snapName, s3Path, at := checkSnapCreateSyntax(ctx)

	// Create a snapshot.
	fatalIf(createSnapshot(snapName, s3Path, at, ctx.Bool("overwrite")).Trace(), "Unable to create a new snapshot.")
	printMsg(createSnapMsg{
		Status:       "success",
		SnapshotName: snapName,
	})
	return nil
}
