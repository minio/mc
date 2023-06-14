// Copyright (c) 2015-2022 MinIO, Inc.
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
	"math"
	"sort"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/console"
)

var replicateStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "show server side replication status",
	Action:       mainReplicateStatus,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} TARGET

FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
EXAMPLES:
  1. Get server side replication metrics for bucket "mybucket" for alias "myminio".
       {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkReplicateStatusSyntax - validate all the passed arguments
func checkReplicateStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

type replicateStatusMessage struct {
	Op                string                `json:"op"`
	URL               string                `json:"url"`
	Status            string                `json:"status"`
	ReplicationStatus replication.Metrics   `json:"replicationStatus"`
	Targets           []madmin.BucketTarget `json:"remoteTargets"`
	cfg               replication.Config    `json:"-"`
}

func (s replicateStatusMessage) JSON() string {
	s.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (s replicateStatusMessage) String() string {
	if s.ReplicationStatus.FailedSize == 0 &&
		s.ReplicationStatus.ReplicaSize == 0 &&
		s.ReplicationStatus.ReplicatedSize == 0 {
		return "Replication status not available."
	}

	coloredDot := console.Colorize("Headers", dot)

	maxLen := 15
	var contents [][]string
	var (
		failCount = s.ReplicationStatus.FailedCount
		failedSz  = s.ReplicationStatus.FailedSize
		replSz    = s.ReplicationStatus.ReplicatedSize
		replicaSz = s.ReplicationStatus.ReplicaSize
	)
	for arn, st := range s.ReplicationStatus.Stats { // Remove stale ARNs from stats
		staleARN := true
		for _, r := range s.cfg.Rules {
			if r.Destination.Bucket == arn {
				staleARN = false
				break
			}
		}
		if staleARN {
			failCount -= st.FailedCount
			failedSz -= st.FailedSize
			replicaSz -= st.ReplicaSize
			replSz -= st.ReplicatedSize
		}
	}
	// normalize stats, avoid negative values
	failCount = uint64(math.Max(float64(failCount), 0))
	failedSz = uint64(math.Max(float64(failedSz), 0))
	replicaSz = uint64(math.Max(float64(replicaSz), 0))
	replSz = uint64(math.Max(float64(replSz), 0))

	var rows string
	arntheme := []string{"Headers"}
	theme := []string{"Failed", "Replicated", "Replica"}
	contents = append(contents, []string{"Failed", humanize.IBytes(failedSz), humanize.Comma(int64(failCount))})
	contents = append(contents, []string{"Replicated", humanize.IBytes(replSz), ""})
	contents = append(contents, []string{"Replica", humanize.IBytes(replicaSz), ""})
	var th string

	r := console.Colorize("THeaderBold", newPrettyTable(" | ",
		Field{"Summary", 95},
	).buildRow("Summary: "))
	rows += r
	rows += "\n"
	hIdx := 0
	for i, row := range contents {
		if i%3 == 0 {
			if hIdx > 0 {
				rows += "\n"
			}
			hIdx++
			rows += console.Colorize("TgtHeaders", newPrettyTable(" | ",
				Field{"Status", 21},
				Field{"Size", maxLen},
				Field{"Count", maxLen},
			).buildRow("Replication Status   ", "Size (Bytes)", "Count"))
			rows += "\n"
		}

		idx := i % 3
		th = theme[idx]
		r := console.Colorize(th, newPrettyTable(" | ",
			Field{"Status", 21},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow("   "+row[0], row[1], row[2])+"\n")
		rows += r
	}

	tgtDetails := make(map[string][][]string)
	var arns []string
	for arn, st := range s.ReplicationStatus.Stats {
		var tgtDetail [][]string
		tgtDetail = append(tgtDetail, []string{"Failed", humanize.IBytes(st.FailedSize), humanize.Comma(int64(st.FailedCount))})
		tgtDetail = append(tgtDetail, []string{"Replicated", humanize.IBytes(st.ReplicatedSize), ""})
		tgtDetails[arn] = tgtDetail
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	if len(arns) > 0 {
		rows += "\n"
		r := console.Colorize("THeaderBold", newPrettyTable(" | ",
			Field{"Target statuses", 120},
		).buildRow("Remote Target Statuses: "))
		rows += r
		rows += "\n"
	}
	for i, arn := range arns {
		if i > 0 {
			rows += "\n"
		}
		staleARN := true
		for _, r := range s.cfg.Rules {
			if r.Destination.Bucket == arn {
				staleARN = false
				break
			}
		}
		if staleARN {
			continue // skip historic metrics for deleted targets
		}
		var ep string
		for _, t := range s.Targets {
			if t.Arn == arn {
				ep = t.Endpoint
				break
			}
		}
		th = arntheme[0]
		var hdrStr, hdrDet string
		hdrStr = ep
		if hdrStr != "" {
			hdrDet = console.Colorize("Values", arn)
		} else {
			hdrStr = arn
		}
		r := console.Colorize(th, newPrettyTable(" | ",
			Field{"Ep", 100},
		).buildRow(fmt.Sprintf("%s %s", coloredDot, hdrStr)))
		rows += r
		rows += "\n"
		if hdrDet != "" {
			r = console.Colorize("THeader", newPrettyTable(" | ",
				Field{"Arn", 100},
			).buildRow("  "+"ARN: "+hdrDet))
			rows += r
			rows += "\n"
			bwStat, ok := s.ReplicationStatus.Stats[arn]
			if ok && bwStat.BandWidthLimitInBytesPerSecond > 0 {
				limit := humanize.Bytes(uint64(bwStat.BandWidthLimitInBytesPerSecond))
				current := humanize.Bytes(uint64(bwStat.CurrentBandwidthInBytesPerSecond))
				if bwStat.BandWidthLimitInBytesPerSecond == 0 {
					limit = "N/A" // N/A means cluster bandwidth is not configured
				}

				r = console.Colorize("THeaderBold", newPrettyTable("",
					Field{"B/w limit Hdr", 80},
				).buildRow("  Configured Max Bandwidth (Bps): "+console.Colorize("Values", limit)))
				rows += r
				rows += "\n"
				r = console.Colorize("THeaderBold", newPrettyTable("",
					Field{"B/w limit Hdr", 80},
				).buildRow("  Current Bandwidth (Bps): "+console.Colorize("Values", current)))
				rows += r
			}
			rows += "\n"
		}
		rows += console.Colorize("TgtHeaders", newPrettyTable(" | ",
			Field{"Status", 21},
			Field{"Size", maxLen},
			Field{"Count", maxLen},
		).buildRow("Replication Status   ", "Size (Bytes)", "Count"))
		rows += "\n"

		tgtDetail, ok := tgtDetails[arn]
		if ok {
			for i, row := range tgtDetail {
				idx := i % 2
				th = theme[idx]
				r := console.Colorize(th, newPrettyTable(" | ",
					Field{"Status", 21},
					Field{"Size", maxLen},
					Field{"Count", maxLen},
				).buildRow("   "+row[0], row[1], row[2])+"\n")
				rows += r
			}
		}
	}
	return console.Colorize("replicateStatusMessage", rows)
}

func mainReplicateStatus(cliCtx *cli.Context) error {
	ctx, cancelReplicateStatus := context.WithCancel(globalContext)
	defer cancelReplicateStatus()

	console.SetColor("THeader", color.New(color.FgWhite))

	console.SetColor("Headers", color.New(color.Bold, color.FgGreen))
	console.SetColor("Values", color.New(color.FgGreen))
	console.SetColor("THeaderBold", color.New(color.FgWhite))

	console.SetColor("TgtHeaders", color.New(color.Bold, color.FgCyan))

	console.SetColor("Replica", color.New(color.FgCyan))
	console.SetColor("Failed", color.New(color.Bold, color.FgRed))

	checkReplicateStatusSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	replicateStatus, err := client.GetReplicationMetrics(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication status")
	// Create a new MinIO Admin Client
	admClient, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")
	_, sourceBucket := url2Alias(args[0])
	targets, e := admClient.ListRemoteTargets(globalContext, sourceBucket, "")
	fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch remote target.")
	cfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to fetch replication configuration.")

	printMsg(replicateStatusMessage{
		Op:                cliCtx.Command.Name,
		URL:               aliasedURL,
		ReplicationStatus: replicateStatus,
		Targets:           targets,
		cfg:               cfg,
	})

	return nil
}
