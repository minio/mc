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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	snapCreateFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "rewind",
			Usage: "Rewind to the state of the data in the specified time",
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

// validate command-line args.
func checkSnapCreateSyntax(cliCtx *cli.Context) (snapName string, url string, refTime time.Time) {
	var perr *probe.Error
	var err error

	args := cliCtx.Args()
	if len(args) != 2 {
		cli.ShowCommandHelpAndExit(cliCtx, "create", globalErrorExitStatus)
	}

	snapName = args.Get(0)
	targetURL := args.Get(1)
	_, perr = newClient(targetURL)
	fatalIf(perr.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")

	rewindStr := cliCtx.String("rewind")
	if rewindStr != "" {
		refTime, err = time.Parse(time.RFC3339, rewindStr)
		if err != nil {
			d, err := time.ParseDuration(rewindStr)
			fatalIf(probe.NewError(err), "Unable to parse at argument.")
			refTime = time.Now().Add(-d)
		}
	} else {
		refTime = time.Now().UTC()
	}

	return snapName, targetURL, refTime
}

func initSnapshotDir(snapName string) *probe.Error {
	snapsDir, err := getSnapsDir()
	if err != nil {
		return err
	}

	snapDir := filepath.Join(snapsDir, snapName)

	if _, e := os.Stat(snapDir); e == nil {
		return probe.NewError(errors.New("snapshot already exist"))
	} else {
		if !os.IsNotExist(e) {
			return probe.NewError(e)
		}
	}

	e := os.MkdirAll(filepath.Join(snapDir, "buckets"), 0700)
	if e != nil {
		return probe.NewError(e)
	}

	return nil
}

func createSnapshotFile(snapName, filename string) (*os.File, *probe.Error) {
	snapsDir, err := getSnapsDir()
	if err != nil {
		return nil, err
	}

	snapDir := filepath.Join(snapsDir, snapName)
	snapFile := filepath.Join(snapDir, filename)

	f, e := os.OpenFile(snapFile, os.O_WRONLY|os.O_CREATE, 0600)
	if e != nil {
		return nil, probe.NewError(e)
	}
	return f, nil
}

func createSnapshot(snapName string, s3Path string, at time.Time) *probe.Error {
	alias, urlStr, hostCfg, err := expandAlias(s3Path)
	if err != nil {
		return err
	}

	s3Client, err := newClientFromAlias(alias, urlStr)
	if err != nil {
		return err
	}

	err = initSnapshotDir(snapName)
	if err != nil {
		return err
	}

	metadataFile, err := createSnapshotFile(snapName, "metadata.json")
	if err != nil {
		return err
	}
	defer metadataFile.Close()
	metadataBytes, e := json.Marshal(S3Target(*hostCfg))
	if e != nil {
		return probe.NewError(e)
	}
	if _, e := metadataFile.Write(metadataBytes); e != nil {
		return probe.NewError(e)
	}

	var snapshotMarker snapshotSerializer
	defer snapshotMarker.CleanUp()

	var entry SnapshotEntry
	for s := range s3Client.Snapshot(context.Background(), at) {
		if s.Err != nil {
			return s.Err
		}
		bucket, key := s.URL.BucketAndPrefix()

		if snapshotMarker.bucket != bucket {
			// Close previous if any.
			err := snapshotMarker.Close()
			if err != nil {
				return err
			}
			// Switch to new.
			err = snapshotMarker.ResetFile(snapName, bucket)
			if err != nil {
				return err
			}
		}

		entry = SnapshotEntry{
			Key:            key,
			VersionID:      s.VersionID,
			Size:           s.Size,
			ModTime:        s.Time,
			ETag:           s.ETag,
			StorageClass:   s.StorageClass,
			IsDeleteMarker: s.IsDeleteMarker,
			IsLatest:       s.IsLatest,
		}

		// Write object type, currently only 0 is used and 255 indicates EOF.
		e := snapshotMarker.WriteInt8(0)
		if e != nil {
			return probe.NewError(e)
		}
		e = entry.EncodeMsg(snapshotMarker.Writer)
		if e != nil {
			return probe.NewError(e)
		}
	}

	return snapshotMarker.Close()

}

// main entry point for snapshot create.
func mainSnapCreate(ctx *cli.Context) error {
	// Validate command-line args.
	snapName, s3Path, at := checkSnapCreateSyntax(ctx)

	// Create a snapshot.
	fatalIf(createSnapshot(snapName, s3Path, at).Trace(), "Unable to create a new snapshot.")
	return nil
}
