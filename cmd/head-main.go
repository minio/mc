/*
 * MinIO Client, (C) 2018 MinIO, Inc.
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
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	headFlags = []cli.Flag{
		cli.Int64Flag{
			Name:  "n,lines",
			Usage: "print the first 'n' lines",
			Value: 10,
		},
	}
)

// Display contents of a file.
var headCmd = cli.Command{
	Name:   "head",
	Usage:  "display first 'n' lines of an object",
	Action: mainHead,
	Before: setGlobalsFromContext,
	Flags:  append(append(headFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE [SOURCE...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
   MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

NOTE:
   '{{.HelpName}}' automatically decompresses 'gzip', 'bzip2' compressed objects.

EXAMPLES:
   1. Display only first line from a 'gzip' compressed object on Amazon S3.
      $ {{.HelpName}} -n 1 s3/csv-data/population.csv.gz

   2. Display only first line from server encrypted object on Amazon S3.
      $ {{.HelpName}} -n 1 --encrypt-key 's3/csv-data=32byteslongsecretkeymustbegiven1' s3/csv-data/population.csv
`,
}

// headURL displays contents of a URL to stdout.
func headURL(sourceURL string, encKeyDB map[string][]prefixSSEPair, nlines int64) *probe.Error {
	var reader io.ReadCloser
	switch sourceURL {
	case "-":
		reader = os.Stdin
	default:
		var err *probe.Error
		var metadata map[string]string
		if reader, metadata, err = getSourceStreamMetadataFromURL(sourceURL, encKeyDB); err != nil {
			return err.Trace(sourceURL)
		}
		ctype := metadata["Content-Type"]
		if strings.Contains(ctype, "gzip") {
			var e error
			reader, e = gzip.NewReader(reader)
			if e != nil {
				return probe.NewError(e)
			}
			defer reader.Close()
		} else if strings.Contains(ctype, "bzip") {
			defer reader.Close()
			reader = ioutil.NopCloser(bzip2.NewReader(reader))
		} else {
			defer reader.Close()
		}
	}
	return headOut(reader, nlines).Trace(sourceURL)
}

// headOut reads from reader stream and writes to stdout. Also check the length of the
// read bytes against size parameter (if not -1) and return the appropriate error
func headOut(r io.Reader, nlines int64) *probe.Error {
	var stdout io.Writer

	// In case of a user showing the object content in a terminal,
	// avoid printing control and other bad characters to avoid
	// terminal session corruption
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		stdout = newPrettyStdout(os.Stdout)
	} else {
		stdout = os.Stdout
	}

	// Initialize a new scanner.
	scn := bufio.NewScanner(r)

	// Negative number of lines means default number of lines.
	if nlines < 0 {
		nlines = 10
	}

	for scn.Scan() && nlines > 0 {
		if _, e := stdout.Write(scn.Bytes()); e != nil {
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
		stdout.Write([]byte("\n"))
		nlines--
	}
	if e := scn.Err(); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// mainHead is the main entry point for head command.
func mainHead(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// Set command flags from context.
	stdinMode := false
	if !ctx.Args().Present() {
		stdinMode = true
	}

	// handle std input data.
	if stdinMode {
		fatalIf(headOut(os.Stdin, ctx.Int64("lines")).Trace(), "Unable to read from standard input.")
		return nil
	}

	// Convert arguments to URLs: expand alias, fix format.
	for _, url := range ctx.Args() {
		fatalIf(headURL(url, encKeyDB, ctx.Int64("lines")).Trace(url), "Unable to read from `"+url+"`.")
	}

	return nil
}
