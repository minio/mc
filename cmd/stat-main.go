/*
 * MinIO Client (C) 2017 MinIO, Inc.
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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
)

// stat specific flags.
var (
	statFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "stat all objects recursively",
		},
	}
)

// show object metadata
var statCmd = cli.Command{
	Name:   "stat",
	Usage:  "show object metadata",
	Action: mainStat,
	Before: initBeforeRunningCmd,
	Flags:  append(append(statFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
  MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

EXAMPLES:
  1. Stat all contents of mybucket on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} s3/mybucket/

  2. Stat all contents of mybucket on Amazon S3 cloud storage on Microsoft Windows.
     {{.Prompt}} {{.HelpName}} s3\mybucket\

  3. Stat files recursively on a local filesystem on Microsoft Windows.
     {{.Prompt}} {{.HelpName}} --recursive C:\Users\Worf\

  4. Stat encrypted files on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} --encrypt-key "s3/personal-docs/=32byteslongsecretkeymustbegiven1" s3/personal-docs/2018-account_report.docx

  5. Stat encrypted files on Amazon S3 cloud storage. In case the encryption key contains non-printable character like tab, pass the
     base64 encoded string as key.
     {{.Prompt}} {{.HelpName}} --encrypt-key "s3/personal-document/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=" s3/personal-document/2019-account_report.docx
`,
}

// checkStatSyntax - validate all the passed arguments
func checkStatSyntax(ctx context.Context, cliCtx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	if !cliCtx.Args().Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "stat", 1) // last argument is exit code
	}

	args := cliCtx.Args()
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}
	// extract URLs.
	URLs := cliCtx.Args()
	isIncomplete := false

	for _, url := range URLs {
		_, _, err := url2Stat(ctx, url, false, encKeyDB)
		if err != nil && !isURLPrefixExists(url, isIncomplete) {
			fatalIf(err.Trace(url), "Unable to stat `"+url+"`.")
		}
	}
}

// mainStat - is a handler for mc stat command
func mainStat(cliCtx *cli.Context) error {
	ctx, cancelStat := context.WithCancel(globalContext)
	defer cancelStat()

	// Additional command specific theme customization.
	console.SetColor("Name", color.New(color.Bold, color.FgCyan))
	console.SetColor("Date", color.New(color.FgWhite))
	console.SetColor("Size", color.New(color.FgWhite))
	console.SetColor("ETag", color.New(color.FgWhite))
	console.SetColor("Metadata", color.New(color.FgWhite))

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check 'stat' cli arguments.
	checkStatSyntax(ctx, cliCtx, encKeyDB)

	// Set command flags from context.
	isRecursive := cliCtx.Bool("recursive")

	args := cliCtx.Args()
	// mimic operating system tool behavior.
	if !cliCtx.Args().Present() {
		args = []string{"."}
	}

	var cErr error
	for _, targetURL := range args {
		stats, err := statURL(ctx, targetURL, false, isRecursive, encKeyDB)
		if err != nil {
			fatalIf(err, "Unable to stat `"+targetURL+"`.")
		}
		for _, stat := range stats {
			st := parseStat(stat)
			if !globalJSON {
				printStat(st)
			} else {
				console.Println(st.JSON())
			}
		}
	}
	return cErr

}
