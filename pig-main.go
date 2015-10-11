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
	"os"
	"syscall"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/probe"
)

// Display contents of a file.
var pigCmd = cli.Command{
	Name:   "pig",
	Usage:  "Write contents of stdin to files. Pig is the opposite of cat command.",
	Action: mainPig,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET...]

EXAMPLES:
   1. Write contents of stdin to an object on Amazon S3 cloud storage.
      $ mc {{.Name}} https://s3.amazonaws.com/personalbuck/meeting-notes.txt

   2. Concatinate part files to an object on Amazon S3 cloud storage.
      $ cat part1.img part2.img | mc {{.Name}} https://s3.amazonaws.com/ferenginar/gnuos.iso

   3. Stream MySQL database dump to Amazon S3 directly.
      $ mysqldump -u root -p ******* accountsdb | mc {{.Name}} https://s3.amazonaws.com/ferenginar/backups/accountsdb-oct-9-2015.sql

   4. Contatinate a zip file to two object storage servers simultaneously.
      $ cat ~/myphotos.zip | mc {{.Name}} https://s3.amazonaws.com/mybucket/photos.zip  https://minio.mystartup.io:9000/backup/photos.zip 
`,
}

// checkPigSyntax performs command-line input validation for pig command.
func checkPigSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "pig", 1) // last argument is exit code
	}
}

// pig writes contents of stdin a collection of URLs.
func pig(targetURLs []string) *probe.Error {
	URLs := []string{}
	config := mustGetMcConfig()
	for _, URL := range targetURLs {
		URLs = append(URLs, getAliasURL(URL, config.Aliases))
	}

	//Stream from stdin to multiple objects until EOF.
	// Ignore size, since os.Stat() would not return proper size all the time for local filesystem for example /proc files.
	err := putTargets(URLs, 0, os.Stdin)
	// TODO: See if this check is necessary.
	switch e := err.ToGoError().(type) {
	case *os.PathError:
		if e.Err == syscall.EPIPE {
			// stdin closed by the user. Gracefully exit.
			return nil
		}
	}
	return err.Trace()
}

// mainPig is the main entry point for pig command.
func mainPig(ctx *cli.Context) {
	checkPigSyntax(ctx)
	fatalIf(pig(ctx.Args()).Trace(ctx.Args()...), "Unable to write to one or more targets.")
}
