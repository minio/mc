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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var catFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "rewind",
		Usage: "display an earlier object version",
	},
	cli.StringFlag{
		Name:  "version-id, vid",
		Usage: "display a specific version of an object",
	},
	cli.BoolFlag{
		Name:  "zip",
		Usage: "extract from remote zip file (MinIO server source only)",
	},
	cli.Int64Flag{
		Name:  "offset",
		Usage: "start offset",
	},
	cli.Int64Flag{
		Name:  "tail",
		Usage: "tail number of bytes at ending of file",
	},
	cli.IntFlag{
		Name:  "part-number",
		Usage: "download only a specific part number",
	},
}

// Display contents of a file.
var catCmd = cli.Command{
	Name:         "cat",
	Usage:        "display object contents",
	Action:       mainCat,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(catFlags, encCFlag), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Stream an object from Amazon S3 cloud storage to mplayer standard input.
     {{.Prompt}} {{.HelpName}} s3/mysql-backups/kubecon-mysql-operator.mpv | mplayer -

  2. Concatenate contents of file1.txt and stdin to standard output.
     {{.Prompt}} {{.HelpName}} file1.txt - > file.txt

  3. Concatenate multiple files to one.
     {{.Prompt}} {{.HelpName}} part.* > complete.img

  4. Save an encrypted object from Amazon S3 cloud storage to a local file.
     {{.Prompt}} {{.HelpName}} --enc-c "play/my-bucket/=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA" s3/mysql-backups/backups-201810.gz > /mnt/data/recent.gz

  5. Display the content of encrypted object. In case the encryption key contains non-printable character like tab, pass the
     base64 encoded string as key.
     {{.Prompt}} {{.HelpName}} --enc-c "play/my-bucket/=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA" play/my-bucket/my-object

  6. Display the content of an object 10 days earlier
     {{.Prompt}} {{.HelpName}} --rewind 10d play/my-bucket/my-object

  7. Display the content of a particular object version
     {{.Prompt}} {{.HelpName}} --vid "3ddac055-89a7-40fa-8cd3-530a5581b6b8" play/my-bucket/my-object
`,
}

// checkCatSyntax - validate all the passed arguments
func checkCatSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// prettyStdout replaces some non printable characters
// with <hex> format to be better viewable by the user
type prettyStdout struct {
	// Internal data to pretty-print
	writer io.Writer
	// Internal buffer which contains pretty printed
	// form of binary (no printable) characters
	buffer *bytes.Buffer
}

// newPrettyStdout returns an initialized prettyStdout struct
func newPrettyStdout(w io.Writer) *prettyStdout {
	return &prettyStdout{
		writer: w,
		buffer: bytes.NewBuffer([]byte{}),
	}
}

// Read() returns pretty printed binary characters
func (s prettyStdout) Write(input []byte) (int, error) {
	inputLen := len(input)

	// Convert no printable characters to '^?'
	// and fill into s.buffer
	for len(input) > 0 {
		r, size := utf8.DecodeRune(input)
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			s.buffer.WriteRune(r)
		} else {
			s.buffer.WriteString("^?")
		}
		input = input[size:]
	}

	bufLen := s.buffer.Len()

	// Copy all buffer content to the writer (stdout)
	n, e := s.buffer.WriteTo(s.writer)
	if e != nil {
		return 0, e
	}

	if int(n) != bufLen {
		return 0, errors.New("error when writing to stdout")
	}

	return inputLen, nil
}

type catOpts struct {
	args      []string
	versionID string
	timeRef   time.Time
	startO    int64
	tailO     int64
	partN     int
	isZip     bool
	stdinMode bool
}

// parseCatSyntax performs command-line input validation for cat command.
func parseCatSyntax(ctx *cli.Context) catOpts {
	// Validate command-line arguments.
	checkCatSyntax(ctx)

	var o catOpts
	o.args = ctx.Args()

	o.versionID = ctx.String("version-id")
	rewind := ctx.String("rewind")

	if o.versionID != "" && rewind != "" {
		fatalIf(errInvalidArgument().Trace(), "You cannot specify --version-id and --rewind at the same time")
	}

	if o.versionID != "" && len(o.args) != 1 {
		fatalIf(errInvalidArgument().Trace(), "You need to pass at least one argument if --version-id is specified")
	}

	for _, arg := range o.args {
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Unknown flag `%s` passed.", arg))
		}
	}

	o.stdinMode = len(o.args) == 0

	o.timeRef = parseRewindFlag(rewind)
	o.isZip = ctx.Bool("zip")
	o.startO = ctx.Int64("offset")
	o.tailO = ctx.Int64("tail")
	o.partN = ctx.Int("part-number")
	if o.tailO != 0 && o.startO != 0 {
		fatalIf(errInvalidArgument().Trace(), "You cannot specify both --tail and --offset")
	}
	if o.tailO < 0 || o.startO < 0 {
		fatalIf(errInvalidArgument().Trace(), "You cannot specify negative --tail or --offset")
	}
	if o.isZip && (o.tailO != 0 || o.startO != 0) {
		fatalIf(errInvalidArgument().Trace(), "You cannot combine --zip with --tail or --offset")
	}
	if o.stdinMode && (o.isZip || o.startO != 0 || o.tailO != 0) {
		fatalIf(errInvalidArgument().Trace(), "You cannot use --zip --tail or --offset with stdin")
	}
	if (o.tailO != 0 || o.startO != 0) && o.partN > 0 {
		fatalIf(errInvalidArgument().Trace(), "You cannot use --part-number with --tail or --offset")
	}

	return o
}

// catURL displays contents of a URL to stdout.
func catURL(ctx context.Context, sourceURL string, encKeyDB map[string][]prefixSSEPair, o catOpts) *probe.Error {
	var reader io.ReadCloser
	size := int64(-1)
	switch sourceURL {
	case "-":
		reader = os.Stdin
	default:
		versionID := o.versionID
		var err *probe.Error
		// Try to stat the object, the purpose is to:
		// 1. extract the size of S3 object so we can check if the size of the
		// downloaded object is equal to the original one. FS files
		// are ignored since some of them have zero size though they
		// have contents like files under /proc.
		// 2. extract the version ID if rewind flag is passed
		if client, content, err := url2Stat(ctx, url2StatOptions{
			urlStr:                  sourceURL,
			versionID:               o.versionID,
			fileAttr:                false,
			encKeyDB:                encKeyDB,
			timeRef:                 o.timeRef,
			isZip:                   o.isZip,
			ignoreBucketExistsCheck: false,
		}); err == nil {
			if o.versionID == "" {
				versionID = content.VersionID
			}
			if o.tailO > 0 && content.Size > 0 {
				o.startO = max(content.Size-o.tailO,
					// Return all.
					0)
			}
			if client.GetURL().Type == objectStorage {
				size = content.Size - o.startO
				if size < 0 {
					err := probe.NewError(fmt.Errorf("specified offset (%d) bigger than file (%d)", o.startO, content.Size))
					return err.Trace(sourceURL)
				}
			}
			if o.partN != 0 {
				size = int64(-1)
			}
		} else {
			return err.Trace(sourceURL)
		}
		gopts := GetOptions{VersionID: versionID, Zip: o.isZip, RangeStart: o.startO, PartNumber: o.partN}
		if reader, err = getSourceStreamFromURL(ctx, sourceURL, encKeyDB, getSourceOpts{
			GetOptions: gopts,
			preserve:   false,
		}); err != nil {
			return err.Trace(sourceURL)
		}
		defer reader.Close()
	}
	return catOut(reader, size).Trace(sourceURL)
}

// catOut reads from reader stream and writes to stdout. Also check the length of the
// read bytes against size parameter (if not -1) and return the appropriate error
func catOut(r io.Reader, size int64) *probe.Error {
	var n int64
	var e error

	var stdout io.Writer

	// In case of a user showing the object content in a terminal,
	// avoid printing control and other bad characters to avoid
	// terminal session corruption
	if isTerminal() {
		stdout = newPrettyStdout(os.Stdout)
	} else {
		stdout = os.Stdout
	}

	// Read till EOF.
	if n, e = io.Copy(stdout, r); e != nil {
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
func mainCat(cliCtx *cli.Context) error {
	ctx, cancelCat := context.WithCancel(globalContext)
	defer cancelCat()

	encKeyDB, err := validateAndCreateEncryptionKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check 'cat' cli arguments.
	o := parseCatSyntax(cliCtx)

	// handle std input data.
	if o.stdinMode {
		fatalIf(catOut(os.Stdin, -1).Trace(), "Unable to read from standard input.")
		return nil
	}

	// if Args contain `-`, we need to preserve its order specially.
	if len(o.args) > 0 && o.args[0] == "-" {
		for i, arg := range os.Args {
			if arg == "cat" {
				// Overwrite cliCtx.Args with os.Args.
				o.args = os.Args[i+1:]
				break
			}
		}
	}

	// Convert arguments to URLs: expand alias, fix format.
	for _, url := range o.args {
		fatalIf(catURL(ctx, url, encKeyDB, o).Trace(url), "Unable to read from `"+url+"`.")
	}

	return nil
}
