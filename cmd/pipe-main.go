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

package cmd

import (
	"os"
	"syscall"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/probe"
)

var (
	pipeFlags = []cli.Flag{}
)

// Display contents of a file.
var pipeCmd = cli.Command{
	Name:   "pipe",
	Usage:  "Redirect STDIN to an object or file or STDOUT.",
	Action: mainPipe,
	Before: setGlobalsFromContext,
	Flags:  append(pipeFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] [TARGET]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Write contents of stdin to a file on local filesystem.
      $ mc {{.Name}} /tmp/hello-world.go

   2. Write contents of stdin to an object on Amazon S3 cloud storage.
      $ mc {{.Name}} s3/personalbuck/meeting-notes.txt

   3. Copy an ISO image to an object on Amazon S3 cloud storage.
      $ cat debian-8.2.iso | mc {{.Name}} s3/ferenginar/gnuos.iso

   4. Stream MySQL database dump to Amazon S3 directly.
      $ mysqldump -u root -p ******* accountsdb | mc {{.Name}} s3/ferenginar/backups/accountsdb-oct-9-2015.sql
`,
}

func pipe(targetURL string) *probe.Error {
	if targetURL == "" {
		// When no target is specified, pipe cat's stdin to stdout.
		return catOut(os.Stdin, -1).Trace()
	}

	// Stream from stdin to multiple objects until EOF.
	// Ignore size, since os.Stat() would not return proper size all the time
	// for local filesystem for example /proc files.
	_, err := putTargetStream(targetURL, os.Stdin, -1)
	// TODO: See if this check is necessary.
	switch e := err.ToGoError().(type) {
	case *os.PathError:
		if e.Err == syscall.EPIPE {
			// stdin closed by the user. Gracefully exit.
			return nil
		}
	}
	return err.Trace(targetURL)
}

// check pipe input arguments.
func checkPipeSyntax(ctx *cli.Context) {
	if len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "pipe", 1) // last argument is exit code.
	}
}

// mainPipe is the main entry point for pipe command.
func mainPipe(ctx *cli.Context) error {

	// validate pipe input arguments.
	checkPipeSyntax(ctx)

	if len(ctx.Args()) == 0 {
		err := pipe("")
		fatalIf(err.Trace("stdout"), "Unable to write to one or more targets.")
	} else {
		// extract URLs.
		URLs := ctx.Args()
		err := pipe(URLs[0])
		fatalIf(err.Trace(URLs[0]), "Unable to write to one or more targets.")
	}

	// Done.
	return nil
}
