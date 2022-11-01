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
	"context"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var replicateDiffFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "unique role ARN",
	},
	cli.BoolFlag{
		Name:  "verbose,v",
		Usage: "include replicated versions",
	},
}

var replicateDiffCmd = cli.Command{
	Name:         "diff",
	Usage:        "show unreplicated object versions",
	Action:       mainReplicateDiff,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateDiffFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} TARGET

FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
EXAMPLES:
  1. Show unreplicated objects on "myminio" alias for objects in prefix "path/to/prefix" of "mybucket" for a specific remote target
     {{.Prompt}} {{.HelpName}} myminio/mybucket/path/to/prefix --arn <remote-arn>

  2. Show unreplicated objects on "myminio" alias for objects in prefix "path/to/prefix" of "mybucket" for all targets.
     {{.Prompt}} {{.HelpName}} myminio/mybucket/path/to/prefix
`,
}

// checkReplicateDiffSyntax - validate all the passed arguments
func checkReplicateDiffSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

type replicateDiffMessage struct {
	Op string `json:"op"`
	madmin.DiffInfo
	OpStatus string `json:"opStatus"`
	arn      string `json:"-"`
	verbose  bool   `json:"-"`
}

func (r replicateDiffMessage) JSON() string {
	r.OpStatus = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

const (
	repAttemptFieldMaxLen = 23
	mtimeFieldMaxLen      = 23
	statusFieldMaxLen     = 9
	vIDFieldMaxLen        = 36
	opFieldMaxLen         = 3
	objectFieldMaxLen     = -1
)

func printReplicateDiffHeader() {
	if globalJSON {
		return
	}
	console.Println(console.Colorize("Headers", newPrettyTable(" | ",
		Field{"LastReplicated", repAttemptFieldMaxLen},
		Field{"Created", mtimeFieldMaxLen},
		Field{"Status", statusFieldMaxLen},
		Field{"VID", vIDFieldMaxLen},
		Field{"Op", opFieldMaxLen},
		Field{"Object", objectFieldMaxLen},
	).buildRow("Attempted At", "Created", "Status", "VersionID", "Op", "Object")))
}

func (r replicateDiffMessage) String() string {
	op := ""
	if r.VersionID != "" {
		switch r.IsDeleteMarker {
		case true:
			op = "DEL"
		default:
			op = "PUT"
		}
	}
	st := r.replStatus()
	replTimeStamp := r.ReplicationTimestamp.Format(printDate)
	switch {
	case st == "PENDING":
		replTimeStamp = ""
	case op == "DEL":
		replTimeStamp = ""
	}

	return console.Colorize("diffMsg", newPrettyTable(" | ",
		Field{"Time", repAttemptFieldMaxLen},
		Field{"MTime", mtimeFieldMaxLen},
		Field{statusTheme(st), statusFieldMaxLen},
		Field{"VersionID", vIDFieldMaxLen},
		Field{op, opFieldMaxLen},
		Field{"Obj", objectFieldMaxLen},
	).buildRow(replTimeStamp, r.LastModified.Format(printDate), st, r.VersionID, op, r.Object))
}

func statusTheme(st string) string {
	switch st {
	case "PENDING":
		return "PStatus"
	case "FAILED":
		return "FStatus"
	case "COMPLETED", "COMPLETE":
		return "CStatus"
	default:
		return "Status"
	}
}

func (r replicateDiffMessage) replStatus() string {
	var st string
	if r.arn == "" { // report overall replication status
		if r.DeleteReplicationStatus != "" {
			st = r.DeleteReplicationStatus
		} else {
			st = r.ReplicationStatus
		}
	} else { // report target replication diff
		for arn, t := range r.Targets {
			if arn != r.arn {
				continue
			}
			if t.DeleteReplicationStatus != "" {
				st = t.DeleteReplicationStatus
			} else {
				st = t.ReplicationStatus
			}
		}
		if len(r.Targets) == 0 {
			st = ""
		}
	}
	return st
}

func mainReplicateDiff(cliCtx *cli.Context) error {
	ctx, cancelReplicateDiff := context.WithCancel(globalContext)
	defer cancelReplicateDiff()

	console.SetColor("Obj", color.New(color.Bold))
	console.SetColor("DEL", color.New(color.FgRed))
	console.SetColor("PUT", color.New(color.FgGreen))
	console.SetColor("VersionID", color.New(color.FgHiBlue))
	console.SetColor("Time", color.New(color.FgYellow))
	console.SetColor("MTime", color.New(color.FgWhite))
	console.SetColor("PStatus", color.New(color.Bold, color.FgHiYellow))
	console.SetColor("FStatus", color.New(color.Bold, color.FgHiRed))
	console.SetColor("CStatus", color.New(color.Bold, color.FgHiGreen))
	console.SetColor("Headers", color.New(color.Bold, color.FgHiGreen))

	checkReplicateDiffSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, prefix := splits[1], splits[2]
	if bucket == "" {
		fatalIf(errInvalidArgument(), "bucket not specified in `"+aliasedURL+"`.")
	}

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")
	verbose := cliCtx.Bool("verbose")
	arn := cliCtx.String("arn")
	showHdr := true
	// Start listening to replication diff.
	diffCh := client.BucketReplicationDiff(ctx, bucket, madmin.ReplDiffOpts{
		Verbose: verbose,
		ARN:     arn,
		Prefix:  prefix,
	})
	for oi := range diffCh {
		if oi.Err != nil {
			fatalIf(probe.NewError(oi.Err), "Unable to fetch replicate diff")
		}
		printDiff(oi, arn, verbose, showHdr)
		showHdr = false
	}
	return nil
}

func printDiff(di madmin.DiffInfo, arn string, verbose, showHdr bool) {
	if showHdr {
		printReplicateDiffHeader()
	}
	printMsg(replicateDiffMessage{
		DiffInfo: di,
		arn:      arn,
		verbose:  verbose,
	})
}
