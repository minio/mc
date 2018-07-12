/*
 * Minio Client, (C) 2015, 2016, 2017 Minio, Inc.
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
	"github.com/minio/mc/pkg/probe"
)

var (
	pipeFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "encrypt-key",
			Usage: "Encrypt object (using server-side encryption)",
		},
	}
)

// Display contents of a file.
var pipeCmd = cli.Command{
	Name:   "pipe",
	Usage:  "Redirect STDIN to an object or file or STDOUT.",
	Action: mainPipe,
	Before: setGlobalsFromContext,
	Flags:  append(pipeFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [TARGET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

ENVIRONMENT VARIABLES:
   MC_ENCRYPT_KEY: List of comma delimited prefix=secret values

EXAMPLES:
   1. Write contents of stdin to a file on local filesystem.
      $ {{.HelpName}} /tmp/hello-world.go

   2. Write contents of stdin to an object on Amazon S3 cloud storage.
      $ {{.HelpName}} s3/personalbuck/meeting-notes.txt

   3. Copy an ISO image to an object on Amazon S3 cloud storage.
      $ cat debian-8.2.iso | {{.HelpName}} s3/ferenginar/gnuos.iso

   4. Stream MySQL database dump to Amazon S3 directly.
      $ mysqldump -u root -p ******* accountsdb | {{.HelpName}} s3/ferenginar/backups/accountsdb-oct-9-2015.sql

   5. Stream an object to Amazon S3 cloud storage and encrypt on server.
      $ {{.HelpName}} --encrypt-key "s3/ferenginar/=32byteslongsecretkeymustbegiven1" s3/ferenginar/klingon_opera_aktuh_maylotah.ogg

`,
}

func pipe(targetURL string, encKeyDB map[string][]prefixSSEPair) *probe.Error {
	if targetURL == "" {
		// When no target is specified, pipe cat's stdin to stdout.
		return catOut(os.Stdin, -1).Trace()
	}
	alias, _ := url2Alias(targetURL)
	sseKey := getSSEKey(targetURL, encKeyDB[alias])

	// Stream from stdin to multiple objects until EOF.
	// Ignore size, since os.Stat() would not return proper size all the time
	// for local filesystem for example /proc files.
	_, err := putTargetStreamWithURL(targetURL, os.Stdin, -1, sseKey)
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
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// validate pipe input arguments.
	checkPipeSyntax(ctx)

	if len(ctx.Args()) == 0 {
		err = pipe("", nil)
		fatalIf(err.Trace("stdout"), "Unable to write to one or more targets.")
	} else {
		// extract URLs.
		URLs := ctx.Args()
		err = pipe(URLs[0], encKeyDB)
		fatalIf(err.Trace(URLs[0]), "Unable to write to one or more targets.")
	}

	// Done.
	return nil
}
