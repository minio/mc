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
	"os"

	"github.com/minio/cli"
	ilm "github.com/minio/mc/cmd/ilm"
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
  ILM add command adds one rule to the existing (if any) set of lifecycle configuration rules. If a rule with the ID already exists it will be replaced.

EXAMPLES:
  1. Add rule for the testbucket on s3. Both expiry & transition are given as dates.
     {{.Prompt}} {{.HelpName}} --id "Devices" --prefix "dev/" --expiry-date "2020-09-17" --transition-date "2020-05-01" --storage-class "GLACIER" s3/testbucket

  2. Add rule for the testbucket on s3. Both expiry and transition are given as number of days.
     {{.Prompt}} {{.HelpName}} --id "Docs" --prefix "doc/" --expiry-days "200" --transition-days "300 days" --storage-class "GLACIER" s3/testbucket

  3. Add rule for the testbucket on s3. Only expiry is given as number of days.
     {{.Prompt}} {{.HelpName}} --id "Docs" --prefix "doc/" --expiry-days "200" --tags "docformat:docx" --tags "plaintextformat:txt" --tags "PDFFormat:pdf" s3/testbucket

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
	cli.StringSliceFlag{
		Name:  "tags",
		Usage: "format '<key>:<value>'; multiple values allowed for multiple key/value pairs",
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

// Validate user given arguments
func checkILMAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelp(ctx, "add")
		os.Exit(globalErrorExitStatus)
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
	if lfcInfoXML, err = getILMXML(objectURL); err != nil {
		console.Errorln(err.Error() + " Error generating lifecycle contents in XML. Target " + objectURL)
		return err
	}

	if lfcInfoXML, err = ilm.GetILMRuleToSet(ctx, lfcInfoXML); err != nil {
		console.Errorln(err.Error() + " Error getting lifecycle rule to set from the user input.")
		return err
	}

	if err = setBucketILMConfiguration(objectURL, lfcInfoXML); err != nil {
		console.Errorln(err.Error() + " Error setting lifecycle rule with id `" + id + "` to set from the user input.")
		return err
	}

	console.Println(console.Colorize(fieldThemeResultSuccess, "Success. Lifecycle configuration rule added with ID `"+id+"`."))
	return nil
}
