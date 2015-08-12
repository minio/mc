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
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/probe"
)

// Help message.
var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "List files and folders",
	Action: mainList,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. List objects recursively on Minio cloud storage.
      $ mc {{.Name}} https://play.minio.io:9000/backup/...
      [2015-03-28 12:47:50 PDT]  34MiB 2006-Jan-1/backup.tar.gz
      [2015-03-31 14:46:33 PDT]  55MiB 2006-Mar-1/backup.tar.gz

   2. List buckets on Amazon S3 cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/
      [2015-01-20 15:42:00 PST]     0B rom/
      [2015-01-15 00:05:40 PST]     0B zek/

   3. List buckets from Amazon S3 cloud storage and recursively list objects from Minio cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/ https://play.minio.io:9000/backup/...
      2015-01-15 00:05:40 PST     0B zek/
      2015-03-31 14:46:33 PDT  55MiB 2006-Mar-1/backup.tar.gz

   4. List files recursively on local filesystem on Windows.
      $ mc {{.Name}} C:\Users\Worf\...
      [2015-03-28 12:47:50 PDT] 11.00MiB Martok\Klingon Council Ministers.pdf
      [2015-03-31 14:46:33 PDT] 15.00MiB Gowron\Khitomer Conference Details.pdf

   5. List files with non english characters on Amazon S3 cloud storage.
      $ mc ls s3:andoria/本...
      [2015-05-19 17:21:49 PDT]    41B 本語.pdf
      [2015-05-19 17:24:19 PDT]    41B 本語.txt
      [2015-05-19 17:28:22 PDT]    41B 本語.md

`,
}

// mainList - is a handler for mc ls command
func mainList(ctx *cli.Context) {
	args := ctx.Args()
	if globalAliasFlag {
		if !ctx.Args().Present() {
			args = []string{"."}
		}
	} else if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}

	config := mustGetMcConfig()
	for _, arg := range args {
		targetURL, err := getExpandedURL(arg, config.Aliases)
		fatalIf(err)
		// if recursive strip off the "..."
		newTargetURL := stripRecursiveURL(targetURL)
		fatalIf(doListCmd(newTargetURL, isURLRecursive(targetURL)))
	}
}

// doListCmd list files on target
func doListCmd(targetURL string, recursive bool) *probe.Error {
	clnt, err := target2Client(targetURL)
	if err != nil {
		return err.Trace()
	}
	return doList(clnt, recursive).Trace()
}
