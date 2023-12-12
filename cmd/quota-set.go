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
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v2/console"
)

var quotaSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "size",
		Usage: "set a hard quota, disallowing writes after quota is reached",
	},
	cli.StringFlag{
		Name:  "concurrent-requests-count",
		Usage: "set the concurrent requests count for bucket",
	},
	cli.StringFlag{
		Name:  "apis",
		Usage: "comma separated names of S3 APIs (e.g. PutObject, ListObjects)",
	},
	cli.StringFlag{
		Name:  "throttle-rules-file",
		Usage: "JSON file containing throttle rules",
	},
}

var quotaSetCmd = cli.Command{
	Name:         "set",
	Usage:        "set bucket quota",
	Action:       mainQuotaSet,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(quotaSetFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [--size QUOTA] [--concurrent-requests-count COUNT --apis API-NAMES] [--throttle-rules-file JSON-FILE]

QUOTA
  quota accepts human-readable case-insensitive number
  suffixes such as "k", "m", "g" and "t" referring to the metric units KB,
  MB, GB and TB respectively. Adding an "i" to these prefixes, uses the IEC
  units, so that "gi" refers to "gibibyte" or "GiB". A "b" at the end is
  also accepted. Without suffixes the unit is bytes.

COUNT
  throttle accepts any non-negative integer value for concurrent-requests-count.
  The requets get evenly distributed among the cluster MinIO nodes.

API-NAMES
  a comma separated list of S3 APIs. The actual names could be values like "PutObject"
  or patterns like "Get*"

JSON-FILE
  a JSON file containing throttle rules defined in below format '[{"concurrentRequestsCount": 100,"apis":["PutObject", "ListObjects"]},{"concurrentRequestsCount": 100,"apis": ["Get*"]}]}'

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set hard quota of 1gb for a bucket "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --size 1GB

  2. Set bucket throttle for specific APIs with concurrent no of requets
     {{.Prompt}} {{.HelpName}} myminio/mybucket --concurrent-requests-count 100 --apis "PutObject,ListObjects"

  3. Set bucket throttle using JSON file payload
     {{.Prompt}} {{.HelpName}} myminio/mybucket --throttle-rules-file JSON-FILE
`,
}

// quotaMessage container for content message structure
type quotaMessage struct {
	op            string
	Status        string                      `json:"status"`
	Bucket        string                      `json:"bucket"`
	Quota         uint64                      `json:"quota,omitempty"`
	QuotaType     string                      `json:"type,omitempty"`
	ThrottleRules []madmin.BucketThrottleRule `json:"throttleRules"`
}

func (q quotaMessage) String() string {
	switch q.op {
	case "set":
		msg := "Successfully set "
		if q.Quota > 0 {
			msg += fmt.Sprintf("quota of %s on `%s`", humanize.IBytes(q.Quota), q.Bucket)
		}
		// if throttle rules as well set
		if len(q.ThrottleRules) > 0 {
			if q.Quota > 0 {
				msg += "\nThrottle configuration:"
			} else {
				msg += "throttle configuration:"
			}
			for _, rule := range q.ThrottleRules {
				msg += fmt.Sprintf("\n- Concurrent Requests Count: %d, APIs: %s", rule.ConcurrentRequestsCount, strings.Join(rule.APIs[:], ","))
			}
		}
		return console.Colorize("QuotaMessage", msg)
	case "clear":
		return console.Colorize("QuotaMessage",
			fmt.Sprintf("Successfully cleared bucket quota configured on `%s`", q.Bucket))
	default:
		msg := fmt.Sprintf("Bucket `%s` has %s quota of %s", q.Bucket, q.QuotaType, humanize.IBytes(q.Quota))
		if len(q.ThrottleRules) > 0 {
			msg += "\nThrottle configuration:"
			for _, rule := range q.ThrottleRules {
				msg += fmt.Sprintf("\n- Concurrent Requests Count: %d, APIs: %s", rule.ConcurrentRequestsCount, strings.Join(rule.APIs[:], ","))
			}
		}
		return console.Colorize("QuotaInfo", msg)
	}
}

func (q quotaMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(q, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// checkQuotaSetSyntax - validate all the passed arguments
func checkQuotaSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainQuotaSet is the handler for "mc quota set" command.
func mainQuotaSet(ctx *cli.Context) error {
	checkQuotaSetSyntax(ctx)

	console.SetColor("QuotaMessage", color.New(color.FgGreen))
	console.SetColor("QuotaInfo", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	_, targetURL := url2Alias(args[0])
	if !ctx.IsSet("size") && !ctx.IsSet("concurrent-requests-count") && !ctx.IsSet("throttle-rules-file") {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"--size or --concurrent-requests-count with --apis or --throttle-rules-file flag(s) needs to be set.")
	}
	if ctx.IsSet("concurrent-requests-count") && !ctx.IsSet("apis") {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"--apis needs to be set with --concurrent-requests-count")
	}
	if ctx.IsSet("concurrent-requests-count") && ctx.IsSet("throttle-rules-file") {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"--concurrent-requests-count cannot be set with --throttle-rules-file")
	}

	// Get existing bucket quota details
	qCfg, e := client.GetBucketQuota(globalContext, targetURL)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get bucket quota")
	if e != nil {
		qCfg = madmin.BucketQuota{}
	}

	qMsg := quotaMessage{
		op:     ctx.Command.Name,
		Bucket: targetURL,
		Status: "success",
	}
	if ctx.IsSet("size") {
		qType := madmin.HardQuota
		quotaStr := ctx.String("size")
		quota, e := humanize.ParseBytes(quotaStr)
		fatalIf(probe.NewError(e).Trace(quotaStr), "Unable to parse quota")
		qCfg.Type = qType
		qCfg.Quota = quota
		qMsg.Quota = quota
		qMsg.QuotaType = string(qType)
	}

	if ctx.IsSet("throttle-rules-file") {
		ruleFile := ctx.String("throttle-rules-file")
		file, err := os.Open(ruleFile)
		if err != nil {
			return fmt.Errorf("failed reading file: %s: %v", ruleFile, err)
		}
		defer file.Close()
		var rules []madmin.BucketThrottleRule
		if json.NewDecoder(file).Decode(&rules) != nil {
			return fmt.Errorf("failed to parse throttle rules file: %s: %v", ruleFile, err)
		}
		for _, rule := range rules {
			sort.Slice(rule.APIs, func(i, j int) bool {
				return rule.APIs[i] < rule.APIs[j]
			})
			ruleExists := false
			for idx, eRule := range qCfg.ThrottleRules {
				sort.Slice(eRule.APIs, func(i, j int) bool {
					return eRule.APIs[i] < eRule.APIs[j]
				})
				if slices.Equal(rule.APIs, eRule.APIs) {
					qCfg.ThrottleRules[idx].ConcurrentRequestsCount = rule.ConcurrentRequestsCount
					ruleExists = true
					break
				}
			}
			if !ruleExists {
				qCfg.ThrottleRules = append(qCfg.ThrottleRules, rule)
			}
		}
		qMsg.ThrottleRules = rules
	}
	if ctx.IsSet("concurrent-requests-count") && ctx.IsSet("apis") {
		countStr := ctx.String("concurrent-requests-count")
		nCount, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("failed to parse concurrent-requests-count: %v", err)
		}
		concurrentReqCount := nCount

		apis := strings.Split(ctx.String("apis"), ",")
		sort.Slice(apis, func(i, j int) bool {
			return apis[i] < apis[j]
		})
		ruleExists := false
		for idx, eRule := range qCfg.ThrottleRules {
			sort.Slice(eRule.APIs, func(i, j int) bool {
				return eRule.APIs[i] < eRule.APIs[j]
			})
			if slices.Equal(apis, eRule.APIs) {
				qCfg.ThrottleRules[idx].ConcurrentRequestsCount = uint64(concurrentReqCount)
				ruleExists = true
				break
			}
		}
		rule := madmin.BucketThrottleRule{ConcurrentRequestsCount: uint64(concurrentReqCount), APIs: apis}
		if !ruleExists {
			qCfg.ThrottleRules = append(qCfg.ThrottleRules, rule)
		}
		qMsg.ThrottleRules = []madmin.BucketThrottleRule{rule}
	}

	fatalIf(probe.NewError(client.SetBucketQuota(globalContext, targetURL, &qCfg)).Trace(args...), "Unable to set bucket quota")

	printMsg(qMsg)

	return nil
}
