/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

// find specific flags
var (
	findFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "name",
			Usage: "Find files of a matching name",
		},
		cli.BoolFlag{
			Name:  "path",
			Usage: "Find global matches on the entire path",
		},
		cli.BoolFlag{
			Name:  "regex",
			Usage: "Matches all files with PCRE regular expression",
		},
		cli.BoolFlag{
			Name:  "watch",
			Usage: "Watches a path and preforms actions as and when file events are received",
		},
	}
)

var findCmd = cli.Command{
	Name:   "find",
	Usage:  "Finds files which match the given set of parameters.",
	Action: mainFind,
	Flags:  append(findFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} PATH FLAG EXPRESSION

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Find file 
`,
}

//checkFindSyntax - validate the passed arguments
func checkFindSyntax(ctx *cli.Context) {
	args := ctx.Args()

	//help message on [mc][find]
	if !args.Present() {
		cli.ShowCommandHelpAndExit(ctx, "find", 1)
	}

	mcConfig, _ := loadMcConfig()
	if _, ok := mcConfig.Hosts[args[1]]; !ok {
		fatalIf(errInvalidArgument().Trace(args...), args[1]+" Is not a valid directory or host")
	}
	//verify that there are no empty arguments
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}

}

//mainFind - handler for mc find commands
func mainFind(ctx *cli.Context) error {
	// Additional command specific theme customization.
	console.SetColor("File", color.New(color.Bold))
	console.SetColor("Dir", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("Time", color.New(color.FgGreen))

	var cErr error

	checkFindSyntax(ctx)

	args := ctx.Args()

	if !ctx.Args().Present() {
		args = []string{"."}
	}

	for _, targetURL := range args {
		var clnt Client
		clnt, err := newClient(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to initialize `"+targetURL+"`")

		switch {
		case ctx.Bool("name"):
			cErr = doFind(clnt, 0, args[0])
		case ctx.Bool("path"):
			cErr = doFind(clnt, 1, args[0])
		case ctx.Bool("regex"):
			cErr = doFind(clnt, 2, args[0])
		case ctx.Bool("watch"):
			cErr = doFind(clnt, 3, args[0])
		default:
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}
	return cErr
}
