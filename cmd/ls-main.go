/*
 * MinIO Client (C) 2014-2019 MinIO, Inc.
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

// ls specific flags.
var (
	lsFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "list recursively",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "list incomplete uploads",
		},
	}
)

// list files and folders.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "list buckets and objects",
	Action: mainList,
	Before: initBeforeRunningCmd,
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
     {{.Prompt}} {{.HelpName}} s3

  2. List buckets and all its contents from Amazon S3 cloud storage recursively.
     {{.Prompt}} {{.HelpName}} --recursive s3

  3. List all contents of mybucket on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} s3/mybucket/

  4. List all contents of mybucket on Amazon S3 cloud storage on Microsoft Windows.
     {{.Prompt}} {{.HelpName}} s3\mybucket\

  5. List files recursively on a local filesystem on Microsoft Windows.
     {{.Prompt}} {{.HelpName}} --recursive C:\Users\Worf\

  6. List incomplete (previously failed) uploads of objects on Amazon S3.
     {{.Prompt}} {{.HelpName}} --incomplete s3/mybucket
`,
}

// checkListSyntax - validate all the passed arguments
func checkListSyntax(ctx context.Context, cliCtx *cli.Context) {
	args := cliCtx.Args()
	if !cliCtx.Args().Present() {
		args = []string{"."}
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}
	// extract URLs.
	URLs := cliCtx.Args()
	isIncomplete := cliCtx.Bool("incomplete")

	for _, url := range URLs {
		_, _, err := url2Stat(ctx, url, false, nil)
		if err != nil && !isURLPrefixExists(url, isIncomplete) {
			// Bucket name empty is a valid error for 'ls myminio',
			// treat it as such.
			_, buckNameEmpty := err.ToGoError().(BucketNameEmpty)
			_, noPath := err.ToGoError().(PathNotFound)
			if buckNameEmpty || noPath {
				continue
			}
			fatalIf(err.Trace(url), "Unable to stat `"+url+"`.")
		}
	}
}

// mainList - is a handler for mc ls command
func mainList(cliCtx *cli.Context) error {
	ctx, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	// Additional command specific theme customization.
	console.SetColor("File", color.New(color.Bold))
	console.SetColor("Dir", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("Time", color.New(color.FgGreen))

	// check 'ls' cliCtx arguments.
	checkListSyntax(ctx, cliCtx)

	// Set command flags from context.
	isRecursive := cliCtx.Bool("recursive")
	isIncomplete := cliCtx.Bool("incomplete")

	args := cliCtx.Args()
	// mimic operating system tool behavior.
	if !cliCtx.Args().Present() {
		args = []string{"."}
	}

	var cErr error
	for _, targetURL := range args {
		clnt, err := newClient(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
		if !strings.HasSuffix(targetURL, string(clnt.GetURL().Separator)) {
			var st *ClientContent
			st, err = clnt.Stat(ctx, isIncomplete, false, nil)
			if st != nil && err == nil && st.Type.IsDir() {
				targetURL = targetURL + string(clnt.GetURL().Separator)
				clnt, err = newClient(targetURL)
				fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
			}
		}

		if e := doList(ctx, clnt, isRecursive, isIncomplete); e != nil {
			cErr = e
		}
	}
	return cErr
}
