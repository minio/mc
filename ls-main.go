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
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
)

// ls specific flags.
var (
	lsFlagIncomplete = cli.BoolFlag{
		Name:  "incomplete, I",
		Usage: "remove incomplete uploads.",
	}
)

// list files and folders.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "List files and folders.",
	Action: mainList,
	Flags:  []cli.Flag{lsFlagIncomplete},
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [OPTIONS] TARGET [TARGET ...]

OPTIONS:
   --incomplete - List incomplete upload of objects.

EXAMPLES:
   1. List buckets on Amazon S3 cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/
      [2015-01-20 15:42:00 PST]     0B rom/
      [2015-01-15 00:05:40 PST]     0B zek/

   2. List buckets and all its contents from Amazon S3 cloud storage recursively.
      $ mc {{.Name}} https://s3.amazonaws.com/...
      2015-01-15 00:05:40 PST     0B zek/
      2015-03-31 14:46:33 PDT  55MiB backup.tar.gz
      2015-04-15 20:11:22 PST     0B miniocloud

   3. List files recursively on a local filesystem on Microsoft Windows.
      $ mc {{.Name}} C:\Users\Worf\...
      [2015-03-31 14:46:33 PDT] 15.00MiB Gowron\Khitomer Conference Details.pdf
      ...

   4. List files with non-English characters on Amazon S3 cloud storage.
      $ mc {{.Name}} s3/andoria/本...
      [2015-05-19 17:28:22 PDT]    41B 本語.md

   5. List folders with space separated names on Amazon S3 cloud storage. 
      $ mc {{.Name}} 's3/miniocloud/Community Files/'
      [2015-05-19 17:28:22 PDT]    41B 本語.md
    
   6. Behave like operating system ‘ls’ tool. Useful for aliasing.
      $ alias ls='mc --mimic ls'
      $ {{.Name}}
      [2015-05-19 17:28:22 PDT]    41B 本語.md

   7. List incomplete (previously failed) uploads of objects on Amazon S3. 
      $ mc {{.Name}} --incomplete s3/mybucket
      [2015-10-19 22:28:02 PDT]     41MiB backup/server112-11Nov15.tgz
`,
}

// checkListSyntax - validate all the passed arguments
func checkListSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !ctx.Args().Present() {
		if globalMimicFlag {
			args = []string{"."}
		} else {
			cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
		}
	}
	if ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
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

	// check 'ls' cli arguments
	checkListSyntax(ctx)

	args := ctx.Args()
	isIncomplete := ctx.Bool("incomplete")

	// mimic operating system tool behavior
	if globalMimicFlag && !ctx.Args().Present() {
		args = []string{"."}
	}

	targetURLs, err := args2URLs(args.Head())
	fatalIf(err.Trace(args...), "One or more unknown URL types passed.")
	for _, targetURL := range targetURLs {
		// if recursive strip off the "..."
		var clnt client.Client
		clnt, err = url2Client(stripRecursiveURL(targetURL))
		fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")

		err = doList(clnt, isURLRecursive(targetURL), isIncomplete)
		fatalIf(err.Trace(clnt.GetURL().String()), "Unable to list target ‘"+clnt.GetURL().String()+"’.")
	}
}
