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
	"strconv"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/throttle"
	"github.com/minio/pkg/v2/console"
)

var throttleSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "concurrent-requests-count",
		Usage: "set the concurrent requests count for bucket",
	},
	cli.StringFlag{
		Name:  "apis",
		Usage: "comma separated names of S3 APIs (e.g. PutObject, ListObjects)",
	},
	cli.StringFlag{
		Name:  "file",
		Usage: "JSON file containing throttle rules",
	},
}

var throttleSetCmd = cli.Command{
	Name:         "set",
	Usage:        "set bucket throttle",
	Action:       mainThrottleSet,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, throttleSetFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [--concurrent-requests-count COUNT --apis API-NAMES] [--file JSON-FILE]

COUNT
  throttle accepts any non-negative ineteger value for concurrent-requests-count.
  The requets get evenly distributed among the cluster MinIO nodes.

API-NAMES
  a comma separated list of S3 APIs. The actual names could be values like "PutObject"
  or patterns like "Get*"

JSON-FILE
  a JSON file containing throttle rules defined in below format
  {
    "Rules": [
      {
        "concurrentRequestsCount": 100,
        "apis": "PutObject,ListObjects"
      },
      {
        "concurrentRequestsCount": 100,
        "apis": "Get*"
      }
    ]
  }

FLAGS:
  {{range VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set bucket throttle for specific APIs with concurrent no of requets
     {{.Prompt}} {{.HelpName}} myminio/mybucket --concurrent-requests-count 100 --apis "PutObject,ListObjects"

  2. Set bucket throttle using JSON file payload
     {{.Prompt}} {{.HelpName}} myminio/mybucket --file JSON-FILE
`,
}

// throttleMessage container for content message structure
type throttleMessage struct {
	op     string
	Status string          `json:"status"`
	Bucket string          `json:"bucket"`
	Rules  []throttle.Rule `json:"rules"`
}

func (t throttleMessage) String() string {
	switch t.op {
	case "set":
		return console.Colorize("ThrottleMessage",
			fmt.Sprintf("Successfully set bucket throttle for `%s`", t.Bucket))
	case "clear":
		return console.Colorize("ThrottleMessage",
			fmt.Sprintf("Successfully cleared bucket throttle configured on `%s`", t.Bucket))
	default:
		str := fmt.Sprintf("Bucket `%s` has below throttle values set", t.Bucket)
		for _, rule := range t.Rules {
			str += fmt.Sprintf("Concurrent Requests Count: %d, APIs: %s", rule.ConcurrentRequestsCount, rule.APIs)
		}
		return console.Colorize("ThrottleInfo", str)
	}
}

func (q throttleMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(q, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// checkThrottleSetSyntax - validate all the passed arguments
func checkThrottleSetSyntax(ctx *cli.Context) {
	fmt.Println("Args: ", ctx.Args())
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainThrottleSet is the handler for "mc throttle set" command.
func mainThrottleSet(ctx *cli.Context) error {
	checkThrottleSetSyntax(ctx)

	console.SetColor("ThrottleMessage", color.New(color.FgGreen))
	console.SetColor("ThrottleInfo", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	_, targetURL := url2Alias(args[0])
	if !ctx.IsSet("concurrent-requests-count") || !ctx.IsSet("file") {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"--concurrent-requests-count with --apis or --file flag(s) need to be set.")
	}
	if ctx.IsSet("concurrent-requests-count") && !ctx.IsSet("apis") {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"--apis needs to be set with --concurrent-requests-count")
	}

	// throttle configuration that is already set.
	cfg, err1 := client.GetBucketThrottle(globalContext, targetURL)
	if err1 != nil {
		return fmt.Errorf("Unable to fetch throttle rules for %s: %v", targetURL, err1)
	}

	if ctx.IsSet("file") {
		ruleFile := ctx.String("file")
		file, err := os.Open(ruleFile)
		if err != nil {
			return fmt.Errorf("failed reading file: %s: %v", ruleFile, err)
		}
		defer file.Close()
		var rules []throttle.Rule
		if json.NewDecoder(file).Decode(&rules) != nil {
			return fmt.Errorf("failed tp parse throttle rules file: %s: %v", ruleFile, err)
		}
		for _, rule := range rules {
			cfg.Rules = append(cfg.Rules, rule)
		}
	} else {
		countStr := ctx.String("concurrent-requests-count")
		nCount, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("failed to parse concurrent-requests-count: %v", err)
		}
		concurrentReqCount := nCount

		apis := ctx.String("apis")
		cfg.Rules = append(cfg.Rules, throttle.Rule{ConcurrentRequestsCount: uint64(concurrentReqCount), APIs: apis})
	}

	fatalIf(probe.NewError(client.SetBucketThrottle(globalContext, targetURL, cfg)).Trace(args...), "Unable to set bucket throttle")

	printMsg(throttleMessage{
		op:     ctx.Command.Name,
		Bucket: targetURL,
		Rules:  cfg.Rules,
	})

	return nil
}
