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

// list files and folders.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "List files and folders.",
	Action: mainList,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET ...]

EXAMPLES:
   1. List buckets on Amazon S3 cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/
      [2015-01-20 15:42:00 PST]     0B rom/
      [2015-01-15 00:05:40 PST]     0B zek/

   2. List buckets from Amazon S3 cloud storage and recursively list objects from Minio cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/ https://play.minio.io:9000/backup/...
      2015-01-15 00:05:40 PST     0B zek/
      2015-03-31 14:46:33 PDT  55MiB 2006-Mar-1/backup.tar.gz

   3. List files recursively on local filesystem on Windows.
      $ mc {{.Name}} C:\Users\Worf\...
      [2015-03-28 12:47:50 PDT] 11.00MiB Martok\Klingon Council Ministers.pdf
      [2015-03-31 14:46:33 PDT] 15.00MiB Gowron\Khitomer Conference Details.pdf

   4. List files with non english characters on Amazon S3 cloud storage.
      $ mc ls s3/andoria/本...
      [2015-05-19 17:24:19 PDT]    41B 本語.txt
      [2015-05-19 17:28:22 PDT]    41B 本語.md

   5. List files with space characters on Amazon S3 cloud storage. 
      $ mc ls 's3/miniocloud/Community Files/'
      [2015-05-19 17:24:19 PDT]    41B 本語.txt
      [2015-05-19 17:28:22 PDT]    41B 本語.md
    
   6. Behave like operating system tool ‘ls’, used for shell aliases.
      $ mc --mimic ls
      [2015-05-19 17:28:22 PDT]    41B 本語.md

`,
}

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
	checkListSyntax(ctx)

	args := ctx.Args()
	// Operating system tool behavior
	if globalMimicFlag && !ctx.Args().Present() {
		args = []string{"."}
	}

	console.SetCustomTheme(map[string]*color.Color{
		"File": color.New(color.FgWhite),
		"Dir":  color.New(color.FgCyan, color.Bold),
		"Size": color.New(color.FgYellow),
		"Time": color.New(color.FgGreen),
	})

	targetURLs, err := args2URLs(args)
	fatalIf(err.Trace(args...), "One or more unknown URL types passed.")

	var lsPrefixMode = len(targetURLs) > 1
	for _, targetURL := range targetURLs {
		// if recursive strip off the "..."
		clnt, err := target2Client(stripRecursiveURL(targetURL))
		fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")

		err = doList(clnt, isURLRecursive(targetURL), lsPrefixMode)
		fatalIf(err.Trace(targetURL), "Unable to list target ‘"+targetURL+"’.")
	}
}
