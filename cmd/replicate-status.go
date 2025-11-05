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
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/v3/console"
	"github.com/olekukonko/tablewriter"
)

var replicateStatusFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "backlog,b",
		Usage: "show most recent failures for one or more nodes. Valid values are 'all', or node name",
		Value: "all",
	},
	cli.BoolFlag{
		Name:  "nodes,n",
		Usage: "show replication speed for all nodes",
	},
}

var replicateStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "show server side replication status",
	Action:       mainReplicateStatus,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateStatusFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET/BUCKET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server side replication metrics for bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. Get replication speed across nodes for bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} --nodes  myminio/mybucket
`,
}

// checkReplicateStatusSyntax - validate all the passed arguments
func checkReplicateStatusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

type replicateStatusMessage struct {
	Op      string                `json:"op"`
	URL     string                `json:"url"`
	Status  string                `json:"status"`
	Metrics replication.MetricsV2 `json:"replicationstats"`
	Targets []madmin.BucketTarget `json:"remoteTargets"`
	cfg     replication.Config    `json:"-"`
}

func (s replicateStatusMessage) JSON() string {
	s.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (s replicateStatusMessage) String() string {
	q := s.Metrics.QueueStats
	rs := s.Metrics.CurrentStats

	if s.cfg.Empty() {
		return "Replication is not configured."
	}

	var (
		replSz       = rs.ReplicatedSize
		replCount    = rs.ReplicatedCount
		replicaCount = rs.ReplicaCount
		replicaSz    = rs.ReplicaSize
		failed       = rs.Errors
		qs           = q.QStats()
	)
	for arn, st := range rs.Stats { // Remove stale ARNs from stats
		staleARN := true
		for _, r := range s.cfg.Rules {
			if r.Destination.Bucket == arn || s.cfg.Role == arn {
				staleARN = false
				break
			}
		}
		if staleARN {
			replSz -= st.ReplicatedSize
			replCount -= int64(st.ReplicatedCount)
		}
	}
	// normalize stats, avoid negative values
	replSz = uint64(math.Max(float64(replSz), 0))
	if replCount < 0 {
		replCount = 0
	}
	// for queue stats
	qtots := rs.QStats
	coloredDot := console.Colorize("qStatusOK", dot)
	if qtots.Curr.Count > qtots.Avg.Count {
		coloredDot = console.Colorize("qStatusWarn", dot)
	}
	var sb strings.Builder

	// Set table header
	table := tablewriter.NewWriter(&sb)
	table.SetAutoWrapText(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs

	uiFn := func(theme string) func(string) string {
		return func(s string) string {
			return console.Colorize(theme, s)
		}
	}
	titleui := uiFn("title")
	valueui := uiFn("value")
	hdrui := uiFn("THeaderBold")
	keyui := uiFn("key")
	maxui := uiFn("Peak")
	avgui := uiFn("Avg")

	addRowF := func(format string, vals ...any) {
		s := fmt.Sprintf(format, vals...)
		table.Append([]string{s})
	}
	var arns []string
	for arn := range rs.Stats {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	addRowF(titleui("Replication status since %s"), humanize.RelTime(time.Now(), time.Now().Add(time.Duration(s.Metrics.Uptime)*time.Second), "", "ago"))
	singleTgt := len(arns) == 1
	staleARN := false
	for i, arn := range arns {
		if i > 0 && !staleARN {
			addRowF("\n")
		}
		staleARN = true
		for _, r := range s.cfg.Rules {
			if r.Destination.Bucket == arn || s.cfg.Role == arn {
				staleARN = false
				break
			}
		}
		if staleARN {
			continue // skip historic metrics for deleted targets
		}
		var ep string
		var tgt madmin.BucketTarget
		for _, t := range s.Targets {
			if t.Arn == arn {
				ep = t.Endpoint
				tgt = t
				break
			}
		}
		nodeName := ep
		if nodeName == "" {
			nodeName = arn
		}
		nodeui := uiFn(getNodeTheme(nodeName))
		currDowntime := time.Duration(0)
		if !tgt.Online && !tgt.LastOnline.IsZero() {
			currDowntime = UTCNow().Sub(tgt.LastOnline)
		}
		// normalize because total downtime is calculated at server side at heartbeat interval, may be slightly behind
		totalDowntime := max(currDowntime, tgt.TotalDowntime)
		nodeStr := nodeui(nodeName)
		addRowF("%s", nodeui(nodeStr))
		stat, ok := rs.Stats[arn]
		if ok {
			addRowF(titleui("Replicated:                   ")+humanize.Comma(int64(stat.ReplicatedCount))+keyui(" objects")+" (%s", valueui(humanize.IBytes(stat.ReplicatedSize))+")")
		}
		healthDot := console.Colorize("online", dot)
		if !tgt.Online {
			healthDot = console.Colorize("offline", dot)
		}

		var linkStatus string
		if tgt.Online {
			linkStatus = healthDot + fmt.Sprintf(" online (total downtime: %s)", valueui(timeDurationToHumanizedDuration(totalDowntime).String()))
		} else {
			linkStatus = healthDot + fmt.Sprintf(" offline %s (total downtime: %s)", valueui(timeDurationToHumanizedDuration(currDowntime).String()), valueui(timeDurationToHumanizedDuration(totalDowntime).String()))
		}
		if singleTgt { // for single target - combine summary section into the target section
			addRowF("%s", titleui("Queued:                       ")+coloredDot+" "+humanize.Comma(int64(qtots.Curr.Count))+keyui(" objects, ")+valueui(humanize.IBytes(uint64(qtots.Curr.Bytes)))+
				" ("+avgui("avg")+": "+humanize.Comma(int64(qtots.Avg.Count))+keyui(" objects, ")+valueui(humanize.IBytes(uint64(qtots.Avg.Bytes)))+
				" ; "+maxui("max:")+" "+humanize.Comma(int64(qtots.Max.Count))+keyui(" objects, ")+valueui(humanize.IBytes(uint64(qtots.Max.Bytes)))+")")
			addRowF("%s", titleui("Workers:                      ")+valueui(humanize.Comma(int64(qs.Workers.Curr)))+avgui(" (avg: ")+humanize.Comma(int64(qs.Workers.Avg))+maxui("; max: ")+humanize.Comma(int64(qs.Workers.Max))+")")
		}
		tgtXfer := qs.TgtXferStats[arn][replication.Total]
		addRowF(titleui("Transfer Rate:                ")+"%s/s ("+keyui("avg: ")+"%s/s"+keyui("; max: ")+"%s/s", valueui(humanize.Bytes(uint64(tgtXfer.CurrRate))), valueui(humanize.Bytes(uint64(tgtXfer.AvgRate))), valueui(humanize.Bytes(uint64(tgtXfer.PeakRate))))
		addRowF(titleui("Latency:                      ")+"%s ("+keyui("avg: ")+"%s"+keyui("; max: ")+"%s)", valueui(tgt.Latency.Curr.Round(time.Millisecond).String()), valueui(tgt.Latency.Avg.Round(time.Millisecond).String()), valueui(tgt.Latency.Max.Round(time.Millisecond).String()))

		addRowF(titleui("Link:                         %s"), linkStatus)
		addRowF(titleui("Errors:                       ")+"%s in last 1 minute; %s in last 1hr; %s since uptime", valueui(humanize.Comma(int64(stat.Failed.LastMinute.Count))), valueui(humanize.Comma(int64(stat.Failed.LastHour.Count))), valueui(humanize.Comma(int64(stat.Failed.Totals.Count))))

		bwStat, ok := rs.Stats[arn]
		if ok && bwStat.BandWidthLimitInBytesPerSecond > 0 {
			limit := "N/A"   // N/A means cluster bandwidth is not configured
			current := "N/A" // N/A means cluster bandwidth is not configured
			if bwStat.CurrentBandwidthInBytesPerSecond > 0 {
				current = humanize.Bytes(uint64(bwStat.CurrentBandwidthInBytesPerSecond))
				current = fmt.Sprintf("%s/s", current)
			}
			if bwStat.BandWidthLimitInBytesPerSecond > 0 {
				limit = humanize.Bytes(uint64(bwStat.BandWidthLimitInBytesPerSecond))
				limit = fmt.Sprintf("%s/s", limit)
			}
			addRowF(titleui("Configured Max Bandwidth (Bps): ")+"%s"+titleui("   Current Bandwidth (Bps): ")+"%s", valueui(limit), valueui(current))
		}

	}
	if !singleTgt {
		xfer := qs.XferStats[replication.Total]
		addRowF("%s", hdrui("\nSummary:"))
		addRowF(titleui("Replicated:                   ")+humanize.Comma(int64(replCount))+keyui(" objects")+" (%s", valueui(humanize.IBytes(replSz))+")")
		addRowF("%s", titleui("Queued:                       ")+coloredDot+" "+humanize.Comma(int64(qtots.Curr.Count))+keyui(" objects, ")+valueui(humanize.IBytes(uint64(qtots.Curr.Bytes)))+
			" ("+avgui("avg")+": "+humanize.Comma(int64(qtots.Avg.Count))+keyui(" objects, ")+valueui(humanize.IBytes(uint64(qtots.Avg.Bytes)))+
			" ; "+maxui("max:")+" "+humanize.Comma(int64(qtots.Max.Count))+keyui(" objects, ")+valueui(humanize.IBytes(uint64(qtots.Max.Bytes)))+")")
		addRowF("%s", titleui("Workers:                      ")+valueui(humanize.Comma(int64(qs.Workers.Curr)))+avgui(" (avg: ")+humanize.Comma(int64(qs.Workers.Avg))+maxui("; max: ")+humanize.Comma(int64(qs.Workers.Max))+")")
		addRowF(titleui("Received:                     ")+"%s"+keyui(" objects")+" (%s)", humanize.Comma(int64(replicaCount)), valueui(humanize.IBytes(uint64(replicaSz))))
		addRowF(titleui("Transfer Rate:                ")+"%s/s"+avgui(" (avg: ")+"%s/s"+maxui("; max: ")+"%s/s)", valueui(humanize.Bytes(uint64(xfer.CurrRate))), valueui(humanize.Bytes(uint64(xfer.AvgRate))), valueui(humanize.Bytes(uint64(xfer.PeakRate))))
		addRowF(titleui("Errors:                       ")+"%s in last 1 minute; %s in last 1hr; %s since uptime", valueui(humanize.Comma(int64(failed.LastMinute.Count))), valueui(humanize.Comma(int64(failed.LastHour.Count))), valueui(humanize.Comma(int64(failed.Totals.Count))))
	}

	table.Render()
	return sb.String()
}

func mainReplicateStatus(cliCtx *cli.Context) error {
	ctx, cancelReplicateStatus := context.WithCancel(globalContext)
	defer cancelReplicateStatus()

	console.SetColor("title", color.New(color.FgCyan))
	console.SetColor("value", color.New(color.FgWhite, color.Bold))

	console.SetColor("key", color.New(color.FgWhite))
	console.SetColor("THeaderBold", color.New(color.Bold, color.FgWhite))
	console.SetColor("Replica", color.New(color.FgCyan))
	console.SetColor("Failed", color.New(color.Bold, color.FgRed))
	for _, c := range colors {
		console.SetColor(fmt.Sprintf("Node%d", c), color.New(c))
	}
	console.SetColor("Replicated", color.New(color.FgCyan))
	console.SetColor("In-Queue", color.New(color.Bold, color.FgYellow))
	console.SetColor("Avg", color.New(color.FgCyan))
	console.SetColor("Peak", color.New(color.FgYellow))
	console.SetColor("Current", color.New(color.FgCyan))
	console.SetColor("Uptime", color.New(color.FgWhite))
	console.SetColor("qStatusWarn", color.New(color.FgYellow, color.Bold))
	console.SetColor("qStatusOK", color.New(color.FgGreen, color.Bold))
	console.SetColor("online", color.New(color.FgGreen, color.Bold))
	console.SetColor("offline", color.New(color.FgRed, color.Bold))

	for _, c := range colors {
		console.SetColor(fmt.Sprintf("Node%d", c), color.New(color.Bold, c))
	}
	checkReplicateStatusSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	// Create a new MinIO Admin Client
	admClient, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")
	_, sourceBucket := url2Alias(args[0])

	replicateStatus, err := client.GetReplicationMetrics(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication status")
	targets, e := admClient.ListRemoteTargets(globalContext, sourceBucket, "")
	fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch remote target.")
	cfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to fetch replication configuration.")

	if cliCtx.IsSet("nodes") {
		printMsg(replicateXferMessage{
			Op:             cliCtx.Command.Name,
			Status:         "success",
			ReplQueueStats: replicateStatus.QueueStats,
		})
		return nil
	}

	printMsg(replicateStatusMessage{
		Op:      cliCtx.Command.Name,
		URL:     aliasedURL,
		Metrics: replicateStatus,
		Targets: targets,
		cfg:     cfg,
	})

	return nil
}

type replicateXferMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	replication.ReplQueueStats
}

func (m replicateXferMessage) JSON() string {
	m.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (m replicateXferMessage) String() string {
	var rows []string
	maxLen := 0

	for _, rqs := range m.Nodes {
		if len(rqs.NodeName) > maxLen {
			maxLen = len(rqs.NodeName)
		}
		lrgX := rqs.XferStats[replication.Large]
		smlX := rqs.XferStats[replication.Small]
		rows = append(rows, console.Colorize("", newPrettyTable(" | ",
			Field{getNodeTheme(rqs.NodeName), len(rqs.NodeName) + 3},
			Field{"Uptime:", 15},
			Field{"Lbl", 25},
			Field{"Avg", 12},
			Field{"Peak", 12},
			Field{"Current", 12},
			Field{"Workers", 10},
		).buildRow(rqs.NodeName, humanize.RelTime(time.Now(), time.Now().Add(time.Duration(rqs.Uptime)*time.Second), "", ""), "Large Objects (>=128 MiB)", fmt.Sprintf("%s/s", humanize.Bytes(uint64(lrgX.AvgRate))), fmt.Sprintf("%s/s", humanize.Bytes(uint64(lrgX.PeakRate))), fmt.Sprintf("%s/s", humanize.Bytes(uint64(lrgX.CurrRate))), fmt.Sprintf("%d", int(rqs.Workers.Avg)))))

		rows = append(rows, console.Colorize("", newPrettyTable(" | ",
			Field{getNodeTheme(rqs.NodeName), len(rqs.NodeName) + 3},
			Field{"Uptime:", 15},
			Field{"Lbl", 25},
			Field{"Avg", 12},
			Field{"Peak", 12},
			Field{"Current", 12},
			Field{"Workers", 10},
		).buildRow(rqs.NodeName, humanize.RelTime(time.Now(), time.Now().Add(time.Duration(rqs.Uptime)*time.Second), "", ""), "Small Objects (<128 MiB)", fmt.Sprintf("%s/s", humanize.Bytes(uint64(smlX.AvgRate))), fmt.Sprintf("%s/s", humanize.Bytes(uint64(smlX.PeakRate))), fmt.Sprintf("%s/s", humanize.Bytes(uint64(smlX.CurrRate))), fmt.Sprintf("%d", int(rqs.Workers.Avg)))))
	}

	hdrSlc := []string{
		console.Colorize("THeaderBold", newPrettyTable(" | ",
			Field{"", maxLen + 3},
			Field{"Uptime:", 15},
			Field{"Lbl", 25},
			Field{"XferRate", 42},
			Field{"Workers", 12}).buildRow("Node Name", "Uptime", "Label", "         Transfer Rate      ", "Workers")),
		console.Colorize("THeaderBold", newPrettyTable(" | ",
			Field{"", maxLen + 3},
			Field{"Uptime:", 15},
			Field{"Lbl", 25},
			Field{"Avg", 12},
			Field{"Peak", 12},
			Field{"Current", 12},
			Field{"Workers", 10}).buildRow("", "", "", "Avg", "Peak", "Current", "")),
	}

	return strings.Join(append(hdrSlc, rows...), "\n")
}

// colorize node name
func getNodeTheme(nodeName string) string {
	nodeHash := fnv.New32a()
	nodeHash.Write([]byte(nodeName))
	nHashSum := nodeHash.Sum32()
	idx := nHashSum % uint32(len(colors))
	return fmt.Sprintf("Node%d", colors[idx])
}
