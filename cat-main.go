/*
 * Minio Client, (C) 2015 Minio, Inc.
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
	"io"
	"os"
	"syscall"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/probe"
)

// Help message.
var catCmd = cli.Command{
	Name:   "cat",
	Usage:  "Display contents of a file",
	Action: runCatCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE [SOURCE...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Concantenate an object from Amazon S3 cloud storage to mplayer standard input.
      $ mc {{.Name}} https://s3.amazonaws.com/ferenginar/klingon_opera_aktuh_maylotah.ogg | mplayer -

   2. Concantenate a file from local filesystem to standard output.
      $ mc {{.Name}} khitomer-accords.txt

   3. Concantenate multiple files from local filesystem to standard output.
      $ mc {{.Name}} *.txt > newfile.txt

   4. Concatenate a non english file name from Amazon S3 cloud storage.
      $ mc {{.Name}} s3:andoria/本語 > /tmp/本語

`,
}

func runCatCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cat", 1) // last argument is exit code
	}
	config := mustGetMcConfig()
	// Convert arguments to URLs: expand alias, fix format...
	for _, arg := range ctx.Args() {
		sourceURL, err := getExpandedURL(arg, config.Aliases)
		Fatal(err)
		Fatal(doCatCmd(sourceURL))
	}
}

func doCatCmd(sourceURL string) *probe.Error {
	sourceClnt, err := source2Client(sourceURL)
	if err != nil {
		return err.Trace()
	}
	// ignore size, since os.Stat() would not return proper size all the time for local filesystem
	// for example /proc files.
	reader, _, err := sourceClnt.GetObject(0, 0)
	if err != nil {
		return err.Trace()
	}
	defer reader.Close()
	// read till EOF
	if _, err := io.Copy(os.Stdout, reader); err != nil {
		switch e := err.(type) {
		case *os.PathError:
			if e.Err == syscall.EPIPE {
				// stdout closed by the user. Gracefully exit.
				return nil
			}
			return probe.New(err)
		default:
			return probe.New(err)
		}
	}
	return nil
}
