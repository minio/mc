// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var headFlags = []cli.Flag{
	cli.Int64Flag{
		Name:  "n,lines",
		Usage: "print the first 'n' lines",
		Value: 10,
	},
	cli.StringFlag{
		Name:  "rewind",
		Usage: "select an object version at specified time",
	},
	cli.StringFlag{
		Name:  "version-id, vid",
		Usage: "select an object version to display",
	},
	cli.BoolFlag{
		Name:  "zip",
		Usage: "extract from remote zip file (MinIO server source only)",
	},
}

// Display contents of a file.
var headCmd = cli.Command{
	Name:         "head",
	Usage:        "display first 'n' lines of an object",
	Action:       mainHead,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(headFlags, encCFlag), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

NOTE:
  '{{.HelpName}}' automatically decompresses 'gzip', 'bzip2' compressed objects.

EXAMPLES:
  1. Display only first line from a 'gzip' compressed object on Amazon S3.
     {{.Prompt}} {{.HelpName}} -n 1 s3/csv-data/population.csv.gz

  2. Display only first line from server encrypted object on Amazon S3.
     {{.Prompt}} {{.HelpName}} -n 1 --enc-c 's3/csv-data=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA' s3/csv-data/population.csv

  3. Display only first line from server encrypted object on Amazon S3.
     {{.Prompt}} {{.HelpName}} --enc-c "s3/json-data=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"  s3/json-data/population.json

  4. Display the first lines of a specific object version.
     {{.Prompt}} {{.HelpName}} --version-id "3ddac055-89a7-40fa-8cd3-530a5581b6b8" s3/json-data/population.json
`,
}

// headURL displays contents of a URL to stdout.
func headURL(sourceURL, sourceVersion string, timeRef time.Time, encKeyDB map[string][]prefixSSEPair, nlines int64, zip bool) *probe.Error {
	var reader io.ReadCloser
	switch sourceURL {
	case "-":
		reader = os.Stdin
	default:
		var err *probe.Error
		var content *ClientContent
		if reader, content, err = getSourceStreamMetadataFromURL(context.Background(), sourceURL, sourceVersion, timeRef, encKeyDB, zip); err != nil {
			return err.Trace(sourceURL)
		}

		ctype := content.Metadata["Content-Type"]
		if strings.Contains(ctype, "gzip") {
			var e error
			reader, e = gzip.NewReader(reader)
			if e != nil {
				return probe.NewError(e)
			}
			defer reader.Close()
		} else if strings.Contains(ctype, "bzip") {
			defer reader.Close()
			reader = io.NopCloser(bzip2.NewReader(reader))
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
	if isTerminal() {
		stdout = newPrettyStdout(os.Stdout)
	} else {
		stdout = os.Stdout
	}

	// Initialize a new scanner.
	br := bufio.NewReader(r)

	// Negative number of lines means default number of lines.
	if nlines < 0 {
		nlines = 10
	}

	for nlines > 0 {
		line, _, err := br.ReadLine()
		if err != nil && !errors.Is(err, io.EOF) {
			return probe.NewError(err)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if _, e := stdout.Write(line); e != nil {
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
	return nil
}

// parseHeadSyntax performs command-line input validation for head command.
func parseHeadSyntax(ctx *cli.Context) (args []string, versionID string, timeRef time.Time) {
	args = ctx.Args()

	versionID = ctx.String("version-id")
	rewind := ctx.String("rewind")

	if versionID != "" && rewind != "" {
		fatalIf(errInvalidArgument().Trace(), "You cannot specify --version-id and --rewind at the same time")
	}

	if versionID != "" && len(args) != 1 {
		fatalIf(errInvalidArgument().Trace(), "You need to pass at least one argument if --version-id is specified")
	}

	timeRef = parseRewindFlag(rewind)
	return
}

// mainHead is the main entry point for head command.
func mainHead(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encryptionKeys, err := validateAndCreateEncryptionKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	args, versionID, timeRef := parseHeadSyntax(ctx)

	stdinMode := len(args) == 0

	// handle std input data.
	if stdinMode {
		fatalIf(headOut(os.Stdin, ctx.Int64("lines")).Trace(), "Unable to read from standard input.")
		return nil
	}

	// Convert arguments to URLs: expand alias, fix format.
	for _, url := range ctx.Args() {
		err = headURL(
			url,
			versionID,
			timeRef,
			encryptionKeys,
			ctx.Int64("lines"),
			ctx.Bool("zip"),
		)
		fatalIf(err.Trace(url), "Unable to read from `"+url+"`.")
	}

	return nil
}
