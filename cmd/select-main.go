/*
 * Minio Client, (C) 2017 Minio, Inc.
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
	"github.com/minio/mc/pkg/probe"
)

var (
	selectFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "expression",
			Usage: "SQL query expression.",
		},
	}
)

// Display contents of a file.
var selectCmd = cli.Command{
	Name:   "select",
	Usage:  "Run select queries on objects.",
	Action: mainSelect,
	Before: setGlobalsFromContext,
	Flags:  append(selectFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Run a query on a set of objects recursively on s3 account.
      $ {{.HelpName}} --expression "select * from S3Object" s3/personalbucket/my-large-csvs/

   2. Run a query on an object on minio account.
      $ {{.HelpName}} --expression "select count(s.power) from S3Object" myminio/iot-devices/power-ratio.csv
`,
}

func selectQl(targetURL, expression string) *probe.Error {
	targetClnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	outputer, err := targetClnt.Select(expression)
	if err != nil {
		return err.Trace(targetURL, expression)
	}
	for event := range outputer {
		os.Stdout.Write(event)
	}

	return nil
}

// check select input arguments.
func checkSelectSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "select", 1) // last argument is exit code.
	}
}

// mainSelect is the main entry point for select command.
func mainSelect(ctx *cli.Context) error {

	// validate select input arguments.
	checkSelectSyntax(ctx)

	// extract URLs.
	URLs := ctx.Args()
	for _, url := range URLs {
		if !isAliasURLDir(url) {
			errorIf(selectQl(url, ctx.String("expression")).Trace(url), "Unable to run select")
			continue
		}
		targetAlias, targetURL, _ := mustExpandAlias(url)
		clnt, err := newClientFromAlias(targetAlias, targetURL)
		if err != nil {
			errorIf(err.Trace(url), "Unable to initialize target `"+url+"`.")
			continue
		}
		for content := range clnt.List(true, false, DirLast) {
			if content.Err != nil {
				errorIf(content.Err.Trace(url), "Unable to list on target `"+url+"`.")
			}
			errorIf(selectQl(targetAlias+content.URL.Path, ctx.String("expression")).Trace(content.URL.String()), "Unable to run select")
		}
	}

	// Done.
	return nil
}
