/*
 * Minio Client (C) 2017 Minio, Inc.
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

// stat specific flags.
var (
	statFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "Stat recursively.",
		},
		cli.StringFlag{
			Name:  "encrypt-key",
			Usage: "Encrypt/Decrypt (using server-side encryption)",
		},
	}
)

// stat files and folders.
var statCmd = cli.Command{
	Name:   "stat",
	Usage:  "Stat contents of objects and folders.",
	Action: mainStat,
	Before: setGlobalsFromContext,
	Flags:  append(statFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

ENVIRONMENT VARIABLES:
   MC_ENCRYPT_KEY: List of comma delimited prefix=secret values

EXAMPLES:
   1. Stat all contents of mybucket on Amazon S3 cloud storage.
      $ {{.HelpName}} s3/mybucket/

   2. Stat all contents of mybucket on Amazon S3 cloud storage on Microsoft Windows.
      $ {{.HelpName}} s3\mybucket\

   3. Stat files recursively on a local filesystem on Microsoft Windows.
      $ {{.HelpName}} --recursive C:\Users\Worf\
   
   4. Stat files which are encrypted on the server side
      $ {{.HelpName}} --encrypt-key "s3/ferenginar=32byteslongsecretkeymustbegiven1" s3/ferenginar/klingon_opera_aktuh_maylotah.ogg
`,
}

// checkStatSyntax - validate all the passed arguments
func checkStatSyntax(ctx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "stat", 1) // last argument is exit code
	}

	args := ctx.Args()
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}
	// extract URLs.
	URLs := ctx.Args()
	isIncomplete := false

	for _, url := range URLs {
		_, _, err := url2Stat(url, false, encKeyDB)
		if err != nil && !isURLPrefixExists(url, isIncomplete) {
			fatalIf(err.Trace(url), "Unable to stat `"+url+"`.")
		}
	}
}

// mainStat - is a handler for mc stat command
func mainStat(ctx *cli.Context) error {
	// Additional command specific theme customization.
	console.SetColor("Name", color.New(color.Bold, color.FgCyan))
	console.SetColor("Date", color.New(color.FgWhite))
	console.SetColor("Size", color.New(color.FgWhite))
	console.SetColor("ETag", color.New(color.FgWhite))

	console.SetColor("EncryptionHeaders", color.New(color.FgWhite))
	console.SetColor("Metadata", color.New(color.FgWhite))

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check 'stat' cli arguments.
	checkStatSyntax(ctx, encKeyDB)

	// Set command flags from context.
	isRecursive := ctx.Bool("recursive")

	args := ctx.Args()
	// mimic operating system tool behavior.
	if !ctx.Args().Present() {
		args = []string{"."}
	}

	var cErr error
	for _, targetURL := range args {
		var clnt Client
		clnt, err := newClient(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")

		targetAlias, _, _ := mustExpandAlias(targetURL)
		return doStat(clnt, isRecursive, targetAlias, targetURL, encKeyDB)
	}
	return cErr

}
