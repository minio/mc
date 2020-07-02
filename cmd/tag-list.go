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
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var tagListCmd = cli.Command{
	Name:   "list",
	Usage:  "list tags of a bucket or an object",
	Action: mainListTag,
	Before: initBeforeRunningCmd,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
   List tags assigned to a bucket or an object

EXAMPLES:
  1. List the tags assigned to an object.
     {{.Prompt}} {{.HelpName}} myminio/testbucket/testobject

  2. List the tags assigned to an object in JSON format.
     {{.Prompt}} {{.HelpName}} --json myminio/testbucket/testobject

  3. List the tags assigned to a bucket.
     {{.Prompt}} {{.HelpName}} myminio/testbucket

  4. List the tags assigned to a bucket in JSON format.
     {{.Prompt}} {{.HelpName}} --json s3/testbucket
`,
}

// tagListMessage structure for displaying tag
type tagListMessage struct {
	Tags   map[string]string `json:"tagset,omitempty"`
	Status string            `json:"status"`
	URL    string            `json:"url"`
}

func (t tagListMessage) JSON() string {
	tagJSONbytes, err := json.MarshalIndent(t, "", "  ")
	fatalIf(probe.NewError(err), "Unable to marshal into JSON for "+t.URL)
	return string(tagJSONbytes)
}

func (t tagListMessage) String() string {
	keys := []string{}
	maxKeyLen := 4 // len("Name")
	for key := range t.Tags {
		keys = append(keys, key)
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}
	sort.Strings(keys)

	maxKeyLen += 2 // add len(" :")
	strs := []string{
		fmt.Sprintf("%v%*v %v", console.Colorize("Name", "Name"), maxKeyLen-4, ":", console.Colorize("Name", t.URL)),
	}
	for _, key := range keys {
		strs = append(
			strs,
			fmt.Sprintf("%v%*v %v", console.Colorize("Key", key), maxKeyLen-len(key), ":", console.Colorize("Value", t.Tags[key])),
		)
	}

	return strings.Join(strs, "\n")
}

func mainListTag(cliCtx *cli.Context) error {
	ctx, cancelListTag := context.WithCancel(globalContext)
	defer cancelListTag()

	if len(cliCtx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(cliCtx, "list", globalErrorExitStatus)
	}

	targetURL := cliCtx.Args().Get(0)
	clnt, err := newClient(targetURL)
	fatalIf(err, "Unable to initialize target "+targetURL)

	tags, err := clnt.GetTags(ctx)
	fatalIf(err, "Unable to fetch tags for "+targetURL)

	tagMap := tags.ToMap()
	if len(tagMap) == 0 {
		fatalIf(probe.NewError(errors.New("check 'mc tag set --help' on how to set tags")), "No tags found  for "+targetURL)
	}

	console.SetColor("Name", color.New(color.Bold, color.FgCyan))
	console.SetColor("Key", color.New(color.FgGreen))
	console.SetColor("Value", color.New(color.FgYellow))

	printMsg(tagListMessage{
		Tags:   tagMap,
		Status: "success",
		URL:    targetURL,
	})
	return nil
}
