// Copyright (c) 2015-2021 MinIO, Inc.
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
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminReplicateStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "display site replication status",
	Action:       mainAdminReplicationStatus,
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
  1. Display Site Replication status:
     {{.Prompt}} {{.HelpName}} minio1
`,
}

type srStatus madmin.SRStatusInfo

func (i srStatus) JSON() string {
	bs, e := json.MarshalIndent(madmin.SRStatusInfo(i), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (i srStatus) String() string {
	var messages []string

	// Color palette initialization
	console.SetColor("Summary", color.New(color.FgWhite, color.Bold))
	console.SetColor("SummaryHdr", color.New(color.FgCyan, color.Bold))
	console.SetColor("SummaryDtl", color.New(color.FgGreen, color.Bold))
	coloredDot := console.Colorize("Status", dot)

	info := madmin.SRStatusInfo(i)
	if !info.Enabled {
		messages = []string{"SiteReplication: off"}
		return console.Colorize("UserMessage", strings.Join(messages, "\n"))
	}
	messages = append(messages, console.Colorize("SummaryHdr", "Sites:"))
	for _, peer := range info.Sites {
		messages = append(messages, console.Colorize("SummaryDtl", fmt.Sprintf("Name: %s, Endpoint: %s, DeploymentID: %s", peer.Name, peer.Endpoint, peer.DeploymentID)))
	}
	messages = append(messages, console.Colorize("Summary", fmt.Sprintf("Unique entries across sites: %d Buckets, %d Policies, %d Users, %d Groups", info.MaxBuckets, info.MaxPolicies, info.MaxUsers, info.MaxGroups))+"\n")

	nameIDMap := make(map[string]string)
	var siteNames []string
	for dID := range info.StatsSummary {
		sname := strings.ToTitle(info.Sites[dID].Name)
		siteNames = append(siteNames, sname)
		nameIDMap[sname] = dID
	}
	sort.Strings(siteNames)
	rowLegend := []string{"Buckets", "Policies", "Users", "Groups"}
	legendHdr := []string{"Site"}
	legendFields := []Field{{"Entity", 15}}
	detailFields := make([][]Field, len(rowLegend))
	for _, sname := range siteNames {
		legendHdr = append(legendHdr, sname)
		legendFields = append(legendFields, Field{"sname", 15})
	}
	messages = append(messages,
		console.Colorize("SummaryHdr", fmt.Sprintf("%s  %s\n", coloredDot, console.Colorize("Summary", "Site replication status"))))

	siteHdr := console.Colorize("SummaryHdr", newPrettyTable(" | ",
		legendFields...,
	).buildRow(legendHdr...))
	messages = append(messages, siteHdr)
	var buckets, policies, users, groups []string
	for i, row := range rowLegend {
		detailFields[i] = make([]Field, len(siteNames)+1)
		detailFields[i][0] = Field{"Entity", 15}
		switch i {
		case 0:
			buckets = append(buckets, row)
		case 1:
			policies = append(policies, row)
		case 2:
			users = append(users, row)
		default:
			groups = append(groups, row)
		}
	}

	rows := make([]string, 4)
	fieldLen := 15
	for j, sname := range siteNames {
		dID := nameIDMap[sname]
		ss := info.StatsSummary[dID]
		var theme, msgStr string
		for i := range rowLegend {
			switch i {
			case 0:
				theme, msgStr = colorizedStatus(ss.ReplicatedBuckets, ss.TotalBucketsCount)
				buckets = append(buckets, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}
			case 1:
				theme, msgStr = colorizedStatus(ss.ReplicatedIAMPolicies, ss.TotalIAMPoliciesCount)
				policies = append(policies, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}
			case 2:
				theme, msgStr = colorizedStatus(ss.ReplicatedUsers, ss.TotalUsersCount)
				users = append(users, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}

			case 3:
				theme, msgStr = colorizedStatus(ss.ReplicatedGroups, ss.TotalGroupsCount)
				groups = append(groups, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}
			}
		}
	}
	for i := range rowLegend {
		switch i {
		case 0:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(buckets...)
		case 1:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(policies...)
		case 2:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(users...)
		case 3:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(groups...)
		}
	}
	messages = append(messages, rows...)
	messages = append(messages, "\n")
	messages = append(messages,
		console.Colorize("SummaryHdr", fmt.Sprintf("%s  %s\n", coloredDot, console.Colorize("Summary", "Bucket metadata replication Summary"))))
	messages = append(messages, siteHdr)

	rowLegend = []string{"Retention", "Encryption", "Tags", "Policy"}
	detailFields = make([][]Field, len(rowLegend))

	var retention, encryption, tags, bpolicies []string
	for i, row := range rowLegend {
		detailFields[i] = make([]Field, len(siteNames)+1)
		detailFields[i][0] = Field{"Entity", 15}
		switch i {
		case 0:
			retention = append(retention, row)
		case 1:
			encryption = append(encryption, row)
		case 2:
			tags = append(tags, row)
		default:
			bpolicies = append(bpolicies, row)
		}
	}
	rows = make([]string, len(rowLegend))
	for j, sname := range siteNames {
		dID := nameIDMap[sname]
		ss := info.StatsSummary[dID]
		var theme, msgStr string
		for i := range rowLegend {
			switch i {
			case 0:
				theme, msgStr = colorizedStatus(ss.ReplicatedLockConfig, ss.TotalLockConfigCount)
				retention = append(retention, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}
			case 1:
				theme, msgStr = colorizedStatus(ss.ReplicatedSSEConfig, ss.TotalSSEConfigCount)
				encryption = append(encryption, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}

			case 2:
				theme, msgStr = colorizedStatus(ss.ReplicatedTags, ss.TotalTagsCount)
				tags = append(tags, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}

			case 3:
				theme, msgStr = colorizedStatus(ss.ReplicatedBucketPolicies, ss.TotalBucketPoliciesCount)
				bpolicies = append(bpolicies, msgStr)
				detailFields[i][j+1] = Field{theme, fieldLen}
			}
		}
	}
	for i := range rowLegend {
		switch i {
		case 0:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(retention...)
		case 1:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(encryption...)
		case 2:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(tags...)
		case 3:
			rows[i] = newPrettyTable(" | ",
				detailFields[i]...).buildRow(bpolicies...)
		}
	}
	messages = append(messages, rows...)
	return console.Colorize("UserMessage", strings.Join(messages, "\n"))
}

func mainAdminReplicationStatus(ctx *cli.Context) error {
	{
		// Check argument count
		argsNr := len(ctx.Args())
		if argsNr != 1 {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
				"Need exactly one alias argument.")
		}
	}

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	info, e := client.SRStatusInfo(globalContext, madmin.SRStatusOptions{})
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get cluster replication information")

	printMsg(srStatus(info))

	return nil
}

// returns theme and colorized status text
func colorizedStatus(cnt, total int) (string, string) {
	if cnt == total && cnt == 0 {
		return "Entity", ""
	}
	status := fmt.Sprintf("%d/%d OK", cnt, total)
	return "Entity", fmt.Sprintf("%-15s", status)
}
