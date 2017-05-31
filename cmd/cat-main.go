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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/probe"
)

var (
	catFlags = []cli.Flag{}
)

// Display contents of a file.
var catCmd = cli.Command{
	Name:   "cat",
	Usage:  "Display file and object contents.",
	Action: mainCat,
	Before: setGlobalsFromContext,
	Flags:  append(catFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE [SOURCE...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Stream an object from Amazon S3 cloud storage to mplayer standard input.
      $ {{.HelpName}} s3/ferenginar/klingon_opera_aktuh_maylotah.ogg | mplayer -

   2. Concantenate contents of file1.txt and stdin to standard output.
      $ {{.HelpName}} file1.txt - > file.txt

   3. Concantenate multiple files to one.
      $ {{.HelpName}} part.* > complete.img

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
			fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Unknown flag `%s` passed.", arg))
		}
	}
}

// catURL displays contents of a URL to stdout.
func catURL(sourceURL string) *probe.Error {
	var reader io.Reader
	size := int64(-1)
	switch sourceURL {
	case "-":
		reader = os.Stdin
	default:
		var err *probe.Error
		client, content, err := url2Stat(sourceURL)
		if err != nil {
			return err.Trace(sourceURL)
		}
		// Ignore size for filesystem objects since os.Stat() would not
		// return proper size all the time, for example with /proc files.
		if client.GetURL().Type == objectStorage {
			size = content.Size
		}
		if reader, err = getSourceStreamFromURL(sourceURL); err != nil {
			return err.Trace(sourceURL)
		}
	}
	return catOut(reader, size).Trace(sourceURL)
}

// catOut reads from reader stream and writes to stdout. Also check the length of the
// read bytes against size parameter (if not -1) and return the appropriate error
func catOut(r io.Reader, size int64) *probe.Error {
	var n int64
	var e error

	// Read till EOF.
	if n, e = io.Copy(os.Stdout, r); e != nil {
		switch e := e.(type) {
		case *os.PathError:
			if e.Err == syscall.EPIPE {
				// stdout closed by the user. Gracefully exit.
				return nil
			}
			return probe.NewError(e)
		default:
			return probe.NewError(e)
		}
	}
	if size != -1 && n < size {
		return probe.NewError(UnexpectedEOF{
			TotalSize:    size,
			TotalWritten: n,
		})
	}
	if size != -1 && n > size {
		return probe.NewError(UnexpectedEOF{
			TotalSize:    size,
			TotalWritten: n,
		})
	}
	return nil
}

// mainCat is the main entry point for cat command.
func mainCat(ctx *cli.Context) error {

	// check 'cat' cli arguments.
	checkCatSyntax(ctx)

	// Set command flags from context.
	stdinMode := false
	if !ctx.Args().Present() {
		stdinMode = true
	}

	// handle std input data.
	if stdinMode {
		fatalIf(catOut(os.Stdin, -1).Trace(), "Unable to read from standard input.")
		return nil
	}

	// if Args contain `-`, we need to preserve its order specially.
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
	for _, url := range args {
		fatalIf(catURL(url).Trace(url), "Unable to read from `"+url+"`.")
	}
	return nil
}
