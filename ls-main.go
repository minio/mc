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
	lsFlagRecursive = cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "List recursively.",
	}
	lsFlagIncomplete = cli.BoolFlag{
		Name:  "incomplete, I",
		Usage: "Remove incomplete uploads.",
	}
	lsFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of ls.",
	}
)

// list files and folders.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "List files and folders.",
	Action: mainList,
	Flags:  []cli.Flag{lsFlagRecursive, lsFlagIncomplete, lsFlagHelp},
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. List buckets on Amazon S3 cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/

   2. List buckets and all its contents from Amazon S3 cloud storage recursively.
      $ mc {{.Name}} --recursive https://s3.amazonaws.com/

   3. List files recursively on a local filesystem on Microsoft Windows.
      $ mc {{.Name}} --recursive C:\Users\Worf\

   4. List files with non-English characters on Amazon S3 cloud storage.
      $ mc {{.Name}} s3/andoria/本

   5. List folders with space separated names on Amazon S3 cloud storage. 
      $ mc {{.Name}} 's3/miniocloud/Community Files/'
    
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
	isRecursive := ctx.Bool("recursive")
	isIncomplete := ctx.Bool("incomplete")

	// mimic operating system tool behavior
	if !ctx.Args().Present() {
		args = []string{"."}
	}

	targetURLs, err := args2URLs(args.Head())
	fatalIf(err.Trace(args...), "One or more unknown URL types passed.")
	for _, targetURL := range targetURLs {
		var clnt client.Client
		clnt, err = url2Client(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")

		err = doList(clnt, isRecursive, isIncomplete)
		if err != nil {
			errorIf(err.Trace(clnt.GetURL().String()), "Unable to list target ‘"+clnt.GetURL().String()+"’.")
			continue
		}
	}
}
