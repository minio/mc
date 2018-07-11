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

// ls specific flags.
var (
	lsFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "List recursively.",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "List incomplete uploads.",
		},
		cli.StringFlag{
			Name:  "encrypt-key",
			Usage: "Encrypt/Decrypt (using server-side encryption)",
		},
	}
)

// list files and folders.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "List files and folders.",
	Action: mainList,
	Before: setGlobalsFromContext,
	Flags:  append(lsFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. List buckets on Amazon S3 cloud storage.
      $ {{.HelpName}} s3

   2. List buckets and all its contents from Amazon S3 cloud storage recursively.
      $ {{.HelpName}} --recursive s3

   3. List all contents of mybucket on Amazon S3 cloud storage.
      $ {{.HelpName}} s3/mybucket/

   4. List all contents of mybucket on Amazon S3 cloud storage on Microsoft Windows.
      $ {{.HelpName}} s3\mybucket\

   5. List files recursively on a local filesystem on Microsoft Windows.
      $ {{.HelpName}} --recursive C:\Users\Worf\

   6. List incomplete (previously failed) uploads of objects on Amazon S3. 
      $ {{.HelpName}} --incomplete s3/mybucket

`,
}

// checkListSyntax - validate all the passed arguments
func checkListSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !ctx.Args().Present() {
		args = []string{"."}
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}
	// extract URLs.
	URLs := ctx.Args()
	isIncomplete := ctx.Bool("incomplete")

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	for _, url := range URLs {
		_, _, err := url2Stat(url, false, encKeyDB)
		if err != nil && !isURLPrefixExists(url, isIncomplete) {
			// Bucket name empty is a valid error for 'ls myminio',
			// treat it as such.
			if _, ok := err.ToGoError().(BucketNameEmpty); ok {
				continue
			}
			fatalIf(err.Trace(url), "Unable to stat `"+url+"`.")
		}
	}
}

// mainList - is a handler for mc ls command
func mainList(ctx *cli.Context) error {
	// Additional command specific theme customization.
	console.SetColor("File", color.New(color.Bold))
	console.SetColor("Dir", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("Time", color.New(color.FgGreen))

	// check 'ls' cli arguments.
	checkListSyntax(ctx)

	// Set command flags from context.
	isRecursive := ctx.Bool("recursive")
	isIncomplete := ctx.Bool("incomplete")

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

		var st *clientContent
		if st, err = clnt.Stat(isIncomplete, false, ""); err != nil {
			switch err.ToGoError().(type) {
			case BucketNameEmpty:
			// For aliases like ``mc ls s3`` it's acceptable to receive BucketNameEmpty error.
			// Nothing to do.
			default:
				fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
			}
		} else if st.Type.IsDir() {
			if !strings.HasSuffix(targetURL, string(clnt.GetURL().Separator)) {
				targetURL = targetURL + string(clnt.GetURL().Separator)
			}
			clnt, err = newClient(targetURL)
			fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
		}
		if e := doList(clnt, isRecursive, isIncomplete); e != nil {
			cErr = e
		}
	}
	return cErr
}
