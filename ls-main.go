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
	"github.com/minio/minio-xl/pkg/probe"
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
      [2015-03-31 14:46:33 PDT] 15.00MiB Gowron\Khitomer Conference Details.pdf

   4. List files with non english characters on Amazon S3 cloud storage.
      $ mc ls s3/andoria/本...
      [2015-05-19 17:28:22 PDT]    41B 本語.md

   5. List files with space characters on Amazon S3 cloud storage. 
      $ mc ls 's3/miniocloud/Community Files/'
      [2015-05-19 17:28:22 PDT]    41B 本語.md
    
   6. Behave like operating system tool ‘ls’, used for shell aliases.
      $ mc --mimic ls
      [2015-05-19 17:28:22 PDT]    41B 本語.md

   7. List incompletely uploaded files for a given bucket
      $ mc ls s3/miniocloud incomplete
      [2015-10-19 22:28:02 PDT]     0B bin/

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

func setListPalette(style string) {
	console.SetCustomPalette(map[string]*color.Color{
		"File": color.New(color.FgWhite),
		"Dir":  color.New(color.FgCyan, color.Bold),
		"Size": color.New(color.FgYellow),
		"Time": color.New(color.FgGreen),
	})
	if style == "light" {
		console.SetCustomPalette(map[string]*color.Color{
			"File": color.New(color.FgWhite, color.Bold),
			"Dir":  color.New(color.FgWhite, color.Bold),
			"Size": color.New(color.FgWhite, color.Bold),
			"Time": color.New(color.FgWhite, color.Bold),
		})
		return
	}
	/// Add more styles here
	if style == "nocolor" {
		// All coloring options exhausted, setting nocolor safely
		console.SetNoColor()
	}
}

// mainList - is a handler for mc ls command
func mainList(ctx *cli.Context) {
	setListPalette(ctx.GlobalString("colors"))
	checkListSyntax(ctx)

	args := ctx.Args()
	// Operating system tool behavior
	if globalMimicFlag && !ctx.Args().Present() {
		args = []string{"."}
	}

	var targetURLs []string
	var err *probe.Error
	if args.Last() == "incomplete" {
		targetURLs, err = args2URLs(args.Head())
		fatalIf(err.Trace(args...), "One or more unknown URL types passed.")
		for _, targetURL := range targetURLs {
			// if recursive strip off the "..."
			var clnt client.Client
			clnt, err = url2Client(stripRecursiveURL(targetURL))
			fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")

			err = doListIncomplete(clnt, isURLRecursive(targetURL), len(targetURLs) > 1)
			fatalIf(err.Trace(clnt.GetURL().String()), "Unable to list target ‘"+clnt.GetURL().String()+"’.")
		}
	} else {
		targetURLs, err = args2URLs(args)
		fatalIf(err.Trace(args...), "One or more unknown URL types passed.")
		for _, targetURL := range targetURLs {
			// if recursive strip off the "..."
			var clnt client.Client
			clnt, err = url2Client(stripRecursiveURL(targetURL))
			fatalIf(err.Trace(targetURL), "Unable to initialize target ‘"+targetURL+"’.")

			err = doList(clnt, isURLRecursive(targetURL), len(targetURLs) > 1)
			fatalIf(err.Trace(clnt.GetURL().String()), "Unable to list target ‘"+clnt.GetURL().String()+"’.")
		}
	}

}
