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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/minio/cli"
	"github.com/minio/minio-xl/pkg/probe"
)

var (
	catFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of cat",
		},
	}
)

// Display contents of a file.
var catCmd = cli.Command{
	Name:   "cat",
	Usage:  "Display contents of a file.",
	Action: mainCat,
	Flags:  append(catFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] SOURCE [SOURCE...]

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Stream an object from Amazon S3 cloud storage to mplayer standard input.
      $ mc {{.Name}} s3.amazonaws.com/ferenginar/klingon_opera_aktuh_maylotah.ogg | mplayer -

   2. Concantenate contents of file1.txt and stdin to standard output.
      $ mc {{.Name}} file1.txt - > file.txt

   3. Concantenate multiple files to one.
      $ mc {{.Name}} part.* > complete.img

`,
}

// checkCatSyntax performs command-line input validation for cat command.
func checkCatSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if !args.Present() {
		args = []string{"-"}
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Unknown flag ‘%s’ passed.", arg))
		}
	}
}

// catURL displays contents of a URL to stdout.
func catURL(sourceURL string) *probe.Error {
	var reader io.ReadSeeker
	switch sourceURL {
	case "-":
		reader = os.Stdin
	default:
		sourceClnt, err := url2Client(sourceURL)
		if err != nil {
			return err.Trace(sourceURL)
		}
		// Ignore size, since os.Stat() would not return proper size all the
		// time for local filesystem for example /proc files.
		reader, err = sourceClnt.Get(0, 0)
		if err != nil {
			return err.Trace(sourceURL)
		}
	}
	return catOut(reader).Trace(sourceURL)
}

// catOut reads from reader stream and writes to stdout.
func catOut(r io.Reader) *probe.Error {
	// Read till EOF.
	if _, err := io.Copy(os.Stdout, r); err != nil {
		switch e := err.(type) {
		case *os.PathError:
			if e.Err == syscall.EPIPE {
				// stdout closed by the user. Gracefully exit.
				return nil
			}
			return probe.NewError(err)
		default:
			return probe.NewError(err)
		}
	}
	return nil
}

// mainCat is the main entry point for cat command.
func mainCat(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'cat' cli arguments.
	checkCatSyntax(ctx)

	// Set command flags from context.
	stdinMode := false
	if !ctx.Args().Present() {
		stdinMode = true
	}

	// handle std input data.
	if stdinMode {
		fatalIf(catOut(os.Stdin).Trace(), "Unable to read from standard input.")
		return
	}

	// if Args contain ‘-’, we need to preserve its order specially.
	args := []string(ctx.Args())
	if ctx.Args().First() == "-" {
		for i, arg := range os.Args {
			if arg == "cat" {
				// Overwrite ctx.Args with os.Args.
				args = os.Args[i+1:]
				break
			}
		}
	}

	// Convert arguments to URLs: expand alias, fix format.
	URLs, err := args2URLs(args)
	fatalIf(err.Trace(args...), "Unable to parse arguments.")

	for _, url := range URLs {
		fatalIf(catURL(url).Trace(url), "Unable to read from ‘"+url+"’.")
	}
}
