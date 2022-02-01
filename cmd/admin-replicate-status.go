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

var adminReplicateStatusFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "buckets",
		Usage: "display only buckets",
	},
	cli.BoolFlag{
		Name:  "policies",
		Usage: "display only policies",
	},
	cli.BoolFlag{
		Name:  "users",
		Usage: "display only users",
	},
	cli.BoolFlag{
		Name:  "groups",
		Usage: "display only groups",
	},
	cli.BoolFlag{
		Name:  "all",
		Usage: "display all available site replication status",
	},
	cli.StringFlag{
		Name:  "bucket",
		Usage: "display bucket sync status",
	},
	cli.StringFlag{
		Name:  "policy",
		Usage: "display policy sync status",
	},
	cli.StringFlag{
		Name:  "user",
		Usage: "display user sync status",
	},
	cli.StringFlag{
		Name:  "group",
		Usage: "display group sync status",
	},
}

// Some cell values
const (
	tickCell      string = "✔ "
	crossTickCell string = "✗ "
	blankCell     string = " "
	fieldLen             = 15
)

var adminReplicateStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "display site replication status",
	Action:       mainAdminReplicationStatus,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminReplicateStatusFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
	1. Display overall site replication status:
		 {{.Prompt}} {{.HelpName}} minio1

	2. Display site replication status of buckets across sites
		 {{.Prompt}} {{.HelpName}} minio1 --buckets

	3. Drill down and view site replication status of bucket "bucket"
		 {{.Prompt}} {{.HelpName}} minio1 --bucket bucket

	4. Drill down and view site replication status of user "foo"
		 {{.Prompt}} {{.HelpName}} minio1 --user foo
`,
}

type srStatus struct {
	madmin.SRStatusInfo
	opts madmin.SRStatusOptions
}

func (i srStatus) JSON() string {
	bs, e := json.MarshalIndent(madmin.SRStatusInfo(i.SRStatusInfo), "", " ")
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

	nameIDMap := make(map[string]string)
	var siteNames []string
	info := i.SRStatusInfo

	for dID := range info.Sites {
		sname := strings.ToTitle(info.Sites[dID].Name)
		siteNames = append(siteNames, sname)
		nameIDMap[sname] = dID
	}

	if !info.Enabled {
		messages = []string{"SiteReplication is not enabled"}
		return console.Colorize("UserMessage", strings.Join(messages, "\n"))
	}
	sort.Strings(siteNames)
	legendHdr := []string{"Site"}
	legendFields := []Field{{"Entity", 15}}
	for _, sname := range siteNames {
		legendHdr = append(legendHdr, sname)
		legendFields = append(legendFields, Field{"sname", 15})
	}

	if i.opts.Buckets {
		messages = append(messages,
			console.Colorize("SummaryHdr", "Bucket replication status:"))
		switch {
		case i.MaxBuckets == 0:
			messages = append(messages, console.Colorize("Summary", "No Buckets present\n"))
		default:
			msg := console.Colorize(i.getTheme(len(info.BucketStats) == 0), fmt.Sprintf("%d/%d Buckets in sync", info.MaxBuckets-len(info.BucketStats), info.MaxBuckets)) + "\n"
			messages = append(messages, fmt.Sprintf("%s  %s", coloredDot, msg))
			if len(i.BucketStats) > 0 {
				messages = append(messages, i.siteHeader(siteNames, "Bucket"))
			}
			var detailFields []Field
			for b, ssMap := range i.BucketStats {
				var details []string
				details = append(details, b)
				detailFields = append(detailFields, legendFields[0])
				for _, sname := range siteNames {
					detailFields = append(detailFields, legendFields[0])
					dID := nameIDMap[sname]
					ss := ssMap[dID]
					switch {
					case !ss.HasBucket:
						details = append(details, fmt.Sprintf("%s Bucket", blankCell))
					case ss.OLockConfigMismatch, ss.PolicyMismatch, ss.QuotaCfgMismatch, ss.ReplicationCfgMismatch, ss.TagMismatch:
						details = append(details, fmt.Sprintf("%s in-sync", crossTickCell))
					default:
						details = append(details, fmt.Sprintf("%s in-sync", tickCell))

					}
				}
				messages = append(messages, newPrettyTable(" | ",
					detailFields...).buildRow(details...))
				messages = append(messages, "")
			}
		}
	}
	if i.opts.Policies {
		messages = append(messages,
			console.Colorize("SummaryHdr", "Policy replication status:"))
		switch {
		case i.MaxPolicies == 0:
			messages = append(messages, console.Colorize("Summary", "No Policies present\n"))
		default:
			msg := console.Colorize(i.getTheme(len(i.PolicyStats) == 0), fmt.Sprintf("%d/%d Policies in sync", info.MaxPolicies-len(info.PolicyStats), info.MaxPolicies)) + "\n"
			messages = append(messages, fmt.Sprintf("%s  %s", coloredDot, msg))

			if len(i.PolicyStats) > 0 {
				messages = append(messages, i.siteHeader(siteNames, "Policy"))
			}
			var detailFields []Field
			for b, ssMap := range i.PolicyStats {
				var details []string
				details = append(details, b)
				detailFields = append(detailFields, legendFields[0])
				for _, sname := range siteNames {
					detailFields = append(detailFields, legendFields[0])
					dID := nameIDMap[sname]
					ss := ssMap[dID]
					switch {
					case !ss.HasPolicy:
						details = append(details, fmt.Sprintf("%s Policy", blankCell))
					case ss.PolicyMismatch:
						details = append(details, fmt.Sprintf("%s in-sync", crossTickCell))
					default:
						details = append(details, fmt.Sprintf("%s in-sync", tickCell))
					}
				}
				messages = append(messages, newPrettyTable(" | ",
					detailFields...).buildRow(details...))
			}
			if len(i.PolicyStats) > 0 {
				messages = append(messages, "")
			}
		}
	}
	if i.opts.Users {
		messages = append(messages,
			console.Colorize("SummaryHdr", "User replication status:"))
		switch {
		case i.MaxUsers == 0:
			messages = append(messages, console.Colorize("Summary", "No Users present\n"))
		default:
			msg := console.Colorize(i.getTheme(len(i.UserStats) == 0), fmt.Sprintf("%d/%d Users in sync", info.MaxUsers-len(i.UserStats), info.MaxUsers)) + "\n"
			messages = append(messages, fmt.Sprintf("%s  %s", coloredDot, msg))

			if len(i.UserStats) > 0 {
				messages = append(messages, i.siteHeader(siteNames, "User"))
			}
			var detailFields []Field
			for b, ssMap := range i.UserStats {
				var details []string
				details = append(details, b)
				detailFields = append(detailFields, legendFields[0])
				for _, sname := range siteNames {
					detailFields = append(detailFields, legendFields[0])
					dID := nameIDMap[sname]
					ss, ok := ssMap[dID]

					switch {
					case !ss.HasUser:
						details = append(details, fmt.Sprintf("%s", blankCell))
					case !ok, ss.UserInfoMismatch:
						details = append(details, fmt.Sprintf("%s in-sync", crossTickCell))
					default:
						details = append(details, fmt.Sprintf("%s in-sync", tickCell))
					}
				}
				messages = append(messages, newPrettyTable(" | ",
					detailFields...).buildRow(details...))
			}
			if len(i.UserStats) > 0 {
				messages = append(messages, "")
			}

		}
	}
	if i.opts.Groups {
		messages = append(messages,
			console.Colorize("SummaryHdr", "Group replication status:"))
		switch {
		case i.MaxGroups == 0:
			messages = append(messages, console.Colorize("Summary", "No Groups present\n"))
		default:
			msg := console.Colorize(i.getTheme(len(i.GroupStats) == 0), fmt.Sprintf("%d/%d Groups in sync", i.MaxGroups-len(i.GroupStats), i.MaxGroups)) + "\n"
			messages = append(messages, fmt.Sprintf("%s  %s", coloredDot, msg))

			if len(i.GroupStats) > 0 {
				messages = append(messages, i.siteHeader(siteNames, "Group"))
			}
			var detailFields []Field
			for b, ssMap := range i.GroupStats {
				var details []string
				details = append(details, b)
				detailFields = append(detailFields, legendFields[0])
				for _, sname := range siteNames {
					detailFields = append(detailFields, legendFields[0])
					dID := nameIDMap[sname]
					ss := ssMap[dID]
					switch {
					case !ss.HasGroup:
						details = append(details, fmt.Sprintf("%s", blankCell))
					case ss.GroupDescMismatch:
						details = append(details, fmt.Sprintf("%s in-sync", crossTickCell))
					default:
						details = append(details, fmt.Sprintf("%s in-sync", tickCell))
					}
				}
				messages = append(messages, newPrettyTable(" | ",
					detailFields...).buildRow(details...))
			}
			if len(i.GroupStats) > 0 {
				messages = append(messages, "")
			}
		}
	}

	switch i.opts.Entity {
	case madmin.SRBucketEntity:
		messages = append(messages, i.getBucketStatusSummary(siteNames, nameIDMap, "Bucket")...)
	case madmin.SRPolicyEntity:
		messages = append(messages, i.getPolicyStatusSummary(siteNames, nameIDMap, "Policy")...)
	case madmin.SRUserEntity:
		messages = append(messages, i.getUserStatusSummary(siteNames, nameIDMap, "User")...)
	case madmin.SRGroupEntity:
		messages = append(messages, i.getGroupStatusSummary(siteNames, nameIDMap, "Group")...)

	}

	return console.Colorize("UserMessage", strings.Join(messages, "\n"))
}

func (i srStatus) siteHeader(siteNames []string, legend string) string {
	legendHdr := []string{legend}
	legendFields := []Field{{"Entity", 15}}
	for _, sname := range siteNames {
		legendHdr = append(legendHdr, sname)
		legendFields = append(legendFields, Field{"sname", 15})
	}
	return console.Colorize("SummaryHdr", newPrettyTable(" | ",
		legendFields...,
	).buildRow(legendHdr...))
}

func (i srStatus) getTheme(match bool) string {
	theme := "UserMessage"
	if !match {
		theme = "WarningMessage"
	}
	return theme
}

func (i srStatus) getBucketStatusSummary(siteNames []string, nameIDMap map[string]string, legend string) []string {
	var messages []string
	coloredDot := console.Colorize("Status", dot)
	var found bool
	for _, st := range i.SRStatusInfo.BucketStats[i.opts.EntityValue] {
		if st.HasBucket {
			found = true
			break
		}
	}
	if !found {
		messages = append(messages, console.Colorize("Summary", fmt.Sprintf("Bucket %s not found\n", i.opts.EntityValue)))
		return messages
	}
	messages = append(messages,
		console.Colorize("SummaryHdr", fmt.Sprintf("%s  %s\n", coloredDot, console.Colorize("Summary", "Bucket config replication summary for: ")+console.Colorize("UserMessage", i.opts.EntityValue))))
	siteHdr := i.siteHeader(siteNames, legend)
	messages = append(messages, siteHdr)

	rowLegend := []string{"Tags", "Policy", "Quota", "Retention", "Encryption", "Replication"}
	detailFields := make([][]Field, len(rowLegend))

	var retention, encryption, tags, bpolicies, quota, replication []string
	for i, row := range rowLegend {
		detailFields[i] = make([]Field, len(siteNames)+1)
		detailFields[i][0] = Field{"Entity", 15}
		switch i {
		case 0:
			tags = append(tags, row)
		case 1:
			bpolicies = append(bpolicies, row)
		case 2:
			quota = append(quota, row)
		case 3:
			retention = append(retention, row)
		case 4:
			encryption = append(encryption, row)
		case 5:
			replication = append(replication, row)
		}
	}
	rows := make([]string, len(rowLegend))
	for j, sname := range siteNames {
		dID := nameIDMap[sname]
		ss := i.SRStatusInfo.BucketStats[i.opts.EntityValue][dID]
		var theme, msgStr string
		for r := range rowLegend {
			switch r {
			case 0:
				theme, msgStr = syncStatus(ss.TagMismatch, ss.HasTagsSet)
				tags = append(tags, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			case 1:
				theme, msgStr = syncStatus(ss.PolicyMismatch, ss.HasPolicySet)
				bpolicies = append(bpolicies, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			case 2:
				theme, msgStr = syncStatus(ss.QuotaCfgMismatch, ss.HasQuotaCfgSet)
				quota = append(quota, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			case 3:
				theme, msgStr = syncStatus(ss.OLockConfigMismatch, ss.HasOLockConfigSet)
				retention = append(retention, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			case 4:
				theme, msgStr = syncStatus(ss.SSEConfigMismatch, ss.HasSSECfgSet)
				encryption = append(encryption, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			case 5:
				theme, msgStr = syncStatus(ss.ReplicationCfgMismatch, ss.HasReplicationCfg)
				replication = append(replication, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}

			}
		}
	}
	for r := range rowLegend {
		switch r {
		case 0:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(tags...)
		case 1:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(bpolicies...)
		case 2:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(quota...)
		case 3:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(retention...)
		case 4:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(encryption...)
		case 5:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(replication...)

		}
	}
	messages = append(messages, rows...)
	return messages
}

func (i srStatus) getPolicyStatusSummary(siteNames []string, nameIDMap map[string]string, legend string) []string {
	var messages []string
	coloredDot := console.Colorize("Status", dot)
	var found bool
	for _, st := range i.SRStatusInfo.PolicyStats[i.opts.EntityValue] {
		if st.HasPolicy {
			found = true
			break
		}
	}
	if !found {
		messages = append(messages, console.Colorize("Summary", fmt.Sprintf("Policy %s not found\n", i.opts.EntityValue)))
		return messages
	}

	rowLegend := []string{"Policy"}
	detailFields := make([][]Field, len(rowLegend))

	var policies []string
	detailFields[0] = make([]Field, len(siteNames)+1)
	detailFields[0][0] = Field{"Entity", 15}
	policies = append(policies, "Policy")
	rows := make([]string, len(rowLegend))
	for j, sname := range siteNames {
		dID := nameIDMap[sname]
		ss := i.SRStatusInfo.PolicyStats[i.opts.EntityValue][dID]
		var theme, msgStr string
		for r := range rowLegend {
			switch r {
			case 0:
				theme, msgStr = syncStatus(ss.PolicyMismatch, ss.HasPolicy)
				policies = append(policies, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			}
		}
	}
	for r := range rowLegend {
		switch r {
		case 0:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(policies...)
		}
	}
	messages = append(messages,
		console.Colorize("SummaryHdr", fmt.Sprintf("%s  %s\n", coloredDot, console.Colorize("Summary", "Policy replication summary for: ")+console.Colorize("UserMessage", i.opts.EntityValue))))
	siteHdr := i.siteHeader(siteNames, legend)
	messages = append(messages, siteHdr)

	messages = append(messages, rows...)
	return messages
}

func (i srStatus) getUserStatusSummary(siteNames []string, nameIDMap map[string]string, legend string) []string {
	var messages []string
	coloredDot := console.Colorize("Status", dot)
	var found bool
	for _, st := range i.SRStatusInfo.UserStats[i.opts.EntityValue] {
		if st.HasUser {
			found = true
			break
		}
	}
	if !found {
		messages = append(messages, console.Colorize("Summary", fmt.Sprintf("User %s not found\n", i.opts.EntityValue)))
		return messages
	}

	rowLegend := []string{"Info", "Policy mapping"}
	detailFields := make([][]Field, len(rowLegend))

	var users, policyMapping []string
	for i, row := range rowLegend {
		detailFields[i] = make([]Field, len(siteNames)+1)
		detailFields[i][0] = Field{"Entity", 15}
		switch i {
		case 0:
			users = append(users, row)
		default:
			policyMapping = append(policyMapping, row)
		}
	}
	rows := make([]string, len(rowLegend))
	for j, sname := range siteNames {
		dID := nameIDMap[sname]
		ss := i.SRStatusInfo.UserStats[i.opts.EntityValue][dID]
		var theme, msgStr string
		for r := range rowLegend {
			switch r {
			case 0:
				theme, msgStr = syncStatus(ss.UserInfoMismatch, ss.HasUser)
				users = append(users, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			case 1:
				theme, msgStr = syncStatus(ss.PolicyMismatch, ss.HasPolicyMapping)
				policyMapping = append(policyMapping, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			}
		}
	}
	for r := range rowLegend {
		switch r {
		case 0:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(users...)
		case 1:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(policyMapping...)
		}
	}
	messages = append(messages,
		console.Colorize("SummaryHdr", fmt.Sprintf("%s  %s\n", coloredDot, console.Colorize("Summary", "User replication summary for: ")+console.Colorize("UserMessage", i.opts.EntityValue))))
	siteHdr := i.siteHeader(siteNames, legend)
	messages = append(messages, siteHdr)

	messages = append(messages, rows...)
	return messages
}

func (i srStatus) getGroupStatusSummary(siteNames []string, nameIDMap map[string]string, legend string) []string {
	var messages []string
	coloredDot := console.Colorize("Status", dot)
	rowLegend := []string{"Info", "Policy mapping"}
	detailFields := make([][]Field, len(rowLegend))
	var found bool
	for _, st := range i.SRStatusInfo.GroupStats[i.opts.EntityValue] {
		if st.HasGroup {
			found = true
			break
		}
	}
	if !found {
		messages = append(messages, console.Colorize("Summary", fmt.Sprintf("Group %s not found\n", i.opts.EntityValue)))
		return messages
	}

	var groups, policyMapping []string
	for i, row := range rowLegend {
		detailFields[i] = make([]Field, len(siteNames)+1)
		detailFields[i][0] = Field{"Entity", 15}
		switch i {
		case 0:
			groups = append(groups, row)
		default:
			policyMapping = append(policyMapping, row)
		}
	}
	rows := make([]string, len(rowLegend))
	// b := i.opts.EntityValue
	for j, sname := range siteNames {
		dID := nameIDMap[sname]
		ss := i.SRStatusInfo.GroupStats[i.opts.EntityValue][dID]
		// sm := i.SRStatusInfo.StatsSummary
		var theme, msgStr string
		for r := range rowLegend {
			switch r {
			case 0:
				theme, msgStr = syncStatus(ss.GroupDescMismatch, ss.HasGroup)
				groups = append(groups, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			case 1:
				theme, msgStr = syncStatus(ss.PolicyMismatch, ss.HasPolicyMapping)
				policyMapping = append(policyMapping, msgStr)
				detailFields[r][j+1] = Field{theme, fieldLen}
			}
		}
	}
	for r := range rowLegend {
		switch r {
		case 0:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(groups...)
		case 1:
			rows[r] = newPrettyTable(" | ",
				detailFields[r]...).buildRow(policyMapping...)
		}
	}
	messages = append(messages,
		console.Colorize("SummaryHdr", fmt.Sprintf("%s  %s\n", coloredDot, console.Colorize("Summary", "Group replication summary for: ")+console.Colorize("UserMessage", i.opts.EntityValue))))
	siteHdr := i.siteHeader(siteNames, legend)
	messages = append(messages, siteHdr)

	messages = append(messages, rows...)
	return messages
}

// Calculate srstatus options for command line flags
func srStatusOpts(ctx *cli.Context) (opts madmin.SRStatusOptions) {
	if !(ctx.IsSet("buckets") ||
		ctx.IsSet("users") ||
		ctx.IsSet("groups") ||
		ctx.IsSet("policies") ||
		ctx.IsSet("bucket") ||
		ctx.IsSet("user") ||
		ctx.IsSet("group") ||
		ctx.IsSet("policy") ||
		ctx.IsSet("all")) || ctx.IsSet("all") {
		opts.Buckets = true
		opts.Users = true
		opts.Groups = true
		opts.Policies = true
		return
	}
	opts.Buckets = ctx.Bool("buckets")
	opts.Policies = ctx.Bool("policies")
	opts.Users = ctx.Bool("users")
	opts.Groups = ctx.Bool("groups")
	for _, name := range []string{"bucket", "user", "group", "policy"} {
		if ctx.IsSet(name) {
			opts.Entity = madmin.GetSREntityType(name)
			opts.EntityValue = ctx.String(name)
			break
		}
	}
	return
}

func mainAdminReplicationStatus(ctx *cli.Context) error {
	{
		// Check argument count
		argsNr := len(ctx.Args())
		if argsNr != 1 {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
				"Need exactly one alias argument.")
		}
		groupStatus := ctx.IsSet("buckets") || ctx.IsSet("groups") || ctx.IsSet("users") || ctx.IsSet("policies")
		indivStatus := ctx.IsSet("bucket") || ctx.IsSet("group") || ctx.IsSet("user") || ctx.IsSet("policy")
		if groupStatus && indivStatus {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
				"Cannot specify both (bucket|group|policy|user) flag and one or more of buckets|groups|policies|users) flag(s)")
		}
		setSlc := []bool{ctx.IsSet("bucket"), ctx.IsSet("user"), ctx.IsSet("group"), ctx.IsSet("policy")}
		count := 0
		for _, s := range setSlc {
			if s {
				count++
			}
		}
		if count > 1 {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
				"Cannot specify more than one of --bucket, --policy, --user, --group flags at the same time")
		}
	}

	console.SetColor("UserMessage", color.New(color.FgGreen))
	console.SetColor("WarningMessage", color.New(color.FgYellow))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")
	opts := srStatusOpts(ctx)
	info, e := client.SRStatusInfo(globalContext, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get cluster replication status")

	printMsg(srStatus{
		SRStatusInfo: info,
		opts:         opts,
	})

	return nil
}

func syncStatus(mismatch, set bool) (string, string) {
	if !set {
		return "Entity", fmt.Sprintf("%s", blankCell)
	}
	if mismatch {
		return "Entity", fmt.Sprintf("%s", crossTickCell)
	}

	return "Entity", fmt.Sprintf("%s", tickCell)
}
