// Copyright (c) 2015-2021 MinIO, Inc.
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

var (
	catFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "rewind",
			Usage: "display an earlier object version",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "display a specific version of an object",
		},
	}
)

// Display contents of a file.
var catCmd = cli.Command{
	Name:         "cat",
	Usage:        "display object contents",
	Action:       mainCat,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(catFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE [SOURCE...]
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
ENVIRONMENT VARIABLES:
  MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

EXAMPLES:
  1. Stream an object from Amazon S3 cloud storage to mplayer standard input.
     {{.Prompt}} {{.HelpName}} s3/mysql-backups/kubecon-mysql-operator.mpv | mplayer -

  2. Concatenate contents of file1.txt and stdin to standard output.
     {{.Prompt}} {{.HelpName}} file1.txt - > file.txt

  3. Concatenate multiple files to one.
     {{.Prompt}} {{.HelpName}} part.* > complete.img

  4. Save an encrypted object from Amazon S3 cloud storage to a local file.
     {{.Prompt}} {{.HelpName}} --encrypt-key 's3/mysql-backups=32byteslongsecretkeymustbegiven1' s3/mysql-backups/backups-201810.gz > /mnt/data/recent.gz

  5. Display the content of encrypted object. In case the encryption key contains non-printable character like tab, pass the
     base64 encoded string as key.
     {{.Prompt}} {{.HelpName}} --encrypt-key "play/my-bucket/=MzJieXRlc2xvbmdzZWNyZXRrZQltdXN0YmVnaXZlbjE="  play/my-bucket/my-object

  6. Display the content of an object 10 days earlier
     {{.Prompt}} {{.HelpName}} --rewind 10d play/my-bucket/my-object

  7. Display the content of a particular object version
     {{.Prompt}} {{.HelpName}} --vid "3ddac055-89a7-40fa-8cd3-530a5581b6b8" play/my-bucket/my-object
`,
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
	n, err := s.buffer.WriteTo(s.writer)
	if err != nil {
		return 0, err
	}

	if int(n) != bufLen {
		return 0, errors.New("error when writing to stdout")
	}

	return inputLen, nil
}

// parseCatSyntax performs command-line input validation for cat command.
func parseCatSyntax(ctx *cli.Context) (args []string, versionID string, timeRef time.Time) {
	args = ctx.Args()

	versionID = ctx.String("version-id")
	rewind := ctx.String("rewind")

	if versionID != "" && rewind != "" {
		fatalIf(errInvalidArgument().Trace(), "You cannot specify --version-id and --rewind at the same time")
	}

	if versionID != "" && len(args) != 1 {
		fatalIf(errInvalidArgument().Trace(), "You need to pass at least one argument if --version-id is specified")
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Unknown flag `%s` passed.", arg))
		}
	}

	timeRef = parseRewindFlag(rewind)
	return
}

// catURL displays contents of a URL to stdout.
func catURL(ctx context.Context, sourceURL, sourceVersion string, timeRef time.Time, encKeyDB map[string][]prefixSSEPair) *probe.Error {
	var reader io.ReadCloser
	size := int64(-1)
	switch sourceURL {
	case "-":
		reader = os.Stdin
	default:
		var versionID = sourceVersion
		var err *probe.Error
		// Try to stat the object, the purpose is to:
		// 1. extract the size of S3 object so we can check if the size of the
		// downloaded object is equal to the original one. FS files
		// are ignored since some of them have zero size though they
		// have contents like files under /proc.
		// 2. extract the version ID if rewind flag is passed
		if client, content, err := url2Stat(ctx, sourceURL, sourceVersion, false, encKeyDB, timeRef); err == nil {
			if sourceVersion == "" {
				versionID = content.VersionID
			}
			if client.GetURL().Type == objectStorage {
				size = content.Size
			}
		} else {
			return err.Trace(sourceURL)
		}
		if reader, err = getSourceStreamFromURL(ctx, sourceURL, versionID, encKeyDB); err != nil {
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

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	// check 'cat' cli arguments.
	args, versionID, rewind := parseCatSyntax(cliCtx)

	// Set command flags from context.
	stdinMode := false
	if len(args) == 0 {
		stdinMode = true
	}

	// handle std input data.
	if stdinMode {
		fatalIf(catOut(os.Stdin, -1).Trace(), "Unable to read from standard input.")
		return nil
	}

	// if Args contain `-`, we need to preserve its order specially.
	if len(args) > 0 && args[0] == "-" {
		for i, arg := range os.Args {
			if arg == "cat" {
				// Overwrite cliCtx.Args with os.Args.
				args = os.Args[i+1:]
				break
			}
		}
	}

	// Convert arguments to URLs: expand alias, fix format.
	for _, url := range args {
		fatalIf(catURL(ctx, url, versionID, rewind, encKeyDB).Trace(url), "Unable to read from `"+url+"`.")
	}

	return nil
}
