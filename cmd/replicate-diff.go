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
	"fmt"
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
		cli.ShowCommandHelpAndExit(ctx, "diff", 1) // last argument is exit code
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

func (r replicateDiffMessage) String() string {
	message := console.Colorize("Time", fmt.Sprintf("[%s]", r.ReplicationTimestamp.Format(printDate))) + " "
	message += console.Colorize("MTime", fmt.Sprintf("[%s]", r.LastModified.Format(printDate))) + " "
	if r.arn == "" { // report overall replication status
		if r.DeleteReplicationStatus != "" {
			message += r.colorizeReplStatus(r.DeleteReplicationStatus) + " "
		} else {
			message += r.colorizeReplStatus(r.ReplicationStatus) + " "
		}
	} else { // report target replication diff
		for arn, t := range r.Targets {
			if arn != r.arn {
				continue
			}
			if t.DeleteReplicationStatus != "" {
				message += r.colorizeReplStatus(t.DeleteReplicationStatus) + " "
			} else {
				message += r.colorizeReplStatus(t.ReplicationStatus) + " "
			}
		}
		if len(r.Targets) == 0 {
			message += r.colorizeReplStatus("") + " "
		}
	}

	fileDesc := ""

	if r.VersionID != "" {
		fileDesc += console.Colorize("VersionID", " "+r.VersionID)
		if r.IsDeleteMarker {
			fileDesc += console.Colorize("DEL", " DEL")
		} else {
			fileDesc += console.Colorize("PUT", " PUT")
		}
		message += fileDesc + " "
	}

	message += console.Colorize("Obj", r.Object)
	return message
}

func (r replicateDiffMessage) colorizeReplStatus(st string) string {
	maxLen := 7
	if r.verbose {
		maxLen = 9
	}
	switch st {
	case "PENDING":
		return console.Colorize("PStatus", fmt.Sprintf("%-*.*s", maxLen, maxLen, st))
	case "FAILED":
		return console.Colorize("FStatus", fmt.Sprintf("%-*.*s", maxLen, maxLen, st))
	case "COMPLETED", "COMPLETE":
		return console.Colorize("CStatus", fmt.Sprintf("%-*.*s", maxLen, maxLen, st))
	default:
		return fmt.Sprintf("%-*.*s", maxLen, maxLen, st)
	}
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
		printDiff(oi, arn, verbose)
	}
	return nil
}

func printDiff(di madmin.DiffInfo, arn string, verbose bool) {
	printMsg(replicateDiffMessage{
		DiffInfo: di,
		arn:      arn,
		verbose:  verbose,
	})
}
