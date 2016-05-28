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

package main

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
			Name:  "help, h",
			Usage: "Help of ls.",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "List recursively.",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "Remove incomplete uploads.",
		},
	}
)

// list files and folders.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "List files and folders.",
	Action: mainList,
	Flags:  append(lsFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. List buckets on Amazon S3 cloud storage.
      $ mc {{.Name}} s3

   2. List buckets and all its contents from Amazon S3 cloud storage recursively.
      $ mc {{.Name}} --recursive s3

   3. List all contents of mybucket on Amazon S3 cloud storage.
      $ mc {{.Name}} s3/mybucket/

   4. List all contents of mybucket on Amazon S3 cloud storage on Microsoft Windows.
      $ mc {{.Name}} s3\mybucket\

   5. List files recursively on a local filesystem on Microsoft Windows.
      $ mc {{.Name}} --recursive C:\Users\Worf\

   6. List incomplete (previously failed) uploads of objects on Amazon S3. 
      $ mc {{.Name}} --incomplete s3/mybucket
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

	for _, url := range URLs {
		_, _, err := url2Stat(url)
		if err != nil && !isURLPrefixExists(url, isIncomplete) {
			// Bucket name empty is a valid error for 'ls myminio',
			// treat it as such.
			if _, ok := err.ToGoError().(BucketNameEmpty); ok {
				continue
			}
			fatalIf(err.Trace(url), "Unable to stat ‘"+url+"’.")
		}
	}
}

// mainList - is a handler for mc ls command
func mainList(ctx *cli.Context) {
	// Additional command speific theme customization.
	console.SetColor("File", color.New(color.FgWhite))
	console.SetColor("Dir", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))
	console.SetColor("Time", color.New(color.FgGreen))

	// Set global flags from context.
	setGlobalsFromContext(ctx)

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

	for _, targetURL := range args {
		var clnt Client
		clnt, err := newClient(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")

		var st *clientContent
		if st, err = clnt.Stat(); err != nil {
			switch err.ToGoError().(type) {
			case BucketNameEmpty:
			// For aliases like ``mc ls s3`` it's acceptable to receive BucketNameEmpty error.
			// Nothing to do.
			default:
				fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")
			}
		} else if st.Type.IsDir() {
			if !strings.HasSuffix(targetURL, string(clnt.GetURL().Separator)) {
				targetURL = targetURL + string(clnt.GetURL().Separator)
			}
			clnt, err = newClient(targetURL)
			fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")
		}

		err = doList(clnt, isRecursive, isIncomplete)
		if err != nil {
			errorIf(err.Trace(clnt.GetURL().String()), "Unable to list target ‘"+clnt.GetURL().String()+"’.")
			continue
		}
	}
}
