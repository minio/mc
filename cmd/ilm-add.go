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
	"errors"
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/cmd/ilm"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmAddCmd = cli.Command{
	Name:   "add",
	Usage:  "add a lifecycle configuration rule to existing (if any) rules of the bucket",
	Action: mainILMAdd,
	Before: setGlobalsFromContext,
	Flags:  append(ilmAddFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  ILM add command appends one rule to the existing set of lifecycle configuration rules. If a rule with the ID already exists it will be replaced.

EXAMPLES:
  1. Add rule for testbucket on s3. Both expiry & transition are entered as dates.
     {{.Prompt}} {{.HelpName}} --id "Devices" --prefix "dev/" --expiry-date "2020-09-17" --transition-date "2020-05-01" --storage-class "GLACIER" s3/testbucket

  2. Add rule for testbucket on s3. Both expiry and transition are number of days.
     {{.Prompt}} {{.HelpName}} --id "Docs" --prefix "doc/" --expiry-days "200" --transition-days "300 days" --storage-class "GLACIER" s3/testbucket

  3. Add rule for testbucket on s3. Only expiry is given as number of days and transition is not set.
     {{.Prompt}} {{.HelpName}} --id "Docs" --prefix "doc/" --expiry-days "200" --tags "docformat=docx&plaintextformat=txt&exportFormat=pdf" s3/testbucket

`,
}

var ilmAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "id",
		Usage: "id for the rule, should be an unique value",
	},
	cli.StringFlag{
		Name:  "prefix",
		Usage: "prefix to apply the lifecycle configuration rule",
	},
	cli.StringFlag{
		Name:  "tags",
		Usage: "format '<key1>=<value1>&<key2>=<value2>&<key3>=<value3>', multiple values allowed for multiple key/value pairs",
	},
	cli.StringFlag{
		Name:  "expiry-date",
		Usage: "format 'YYYY-mm-dd' the date of expiration",
	},
	cli.StringFlag{
		Name:  "expiry-days",
		Usage: "the number of days to expiration",
	},
	cli.StringFlag{
		Name:  "transition-date",
		Usage: "format 'YYYY-MM-DD' for the date to transition",
	},
	cli.StringFlag{
		Name:  "transition-days",
		Usage: "the number of days to transition",
	},
	cli.StringFlag{
		Name:  "storage-class",
		Usage: "storage class for transition (STANDARD_IA, ONEZONE_IA, GLACIER. Etc)",
	},
	cli.BoolFlag{
		Name:  "disabled",
		Usage: "disable the rule",
	},
}

type ilmAddMessage struct {
	Status string `json:"status"`
	Target string `json:"target"`
	ID     string `json:"id"`
}

func (i ilmAddMessage) String() string {
	return console.Colorize(ilmThemeResultSuccess, "Lifecycle configuration rule added with ID `"+i.ID+"` to "+i.Target+".")
}

func (i ilmAddMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate user given arguments
func checkILMAddSyntax(ctx *cli.Context) {
	id := ctx.String("id")
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "add")
		os.Exit(globalErrorExitStatus)
	}
	if id == "" {
		e := errors.New("ID for a rule cannot be empty")
		fatalIf(probe.NewError(e), "Refer mc "+ctx.Command.FullName()+" --help for more details")
	}
}

// Calls SetBucketLifecycle with the XML representation of lifecycleConfiguration type.
func mainILMAdd(ctx *cli.Context) error {
	var lfcInfoXML string
	var err error
	checkILMAddSyntax(ctx)
	setILMDisplayColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	id := ctx.String("id")
	lfcInfoXML, err = getILMXML(objectURL)
	fatalIf(probe.NewError(err), "Failed to generate lifecycle configuration on "+objectURL)
	lfcInfoXML, err = ilm.GetILMRuleToSet(ctx, lfcInfoXML)
	fatalIf(probe.NewError(err), "Failed to get lifecycle rule from the user input")
	err = setBucketILMConfiguration(objectURL, lfcInfoXML)
	fatalIf(probe.NewError(err), "Failed to set lifecycle rule with id `"+id+"`")
	printMsg(ilmAddMessage{
		Status: "success",
		Target: objectURL,
		ID:     id,
	})
	return nil
}
