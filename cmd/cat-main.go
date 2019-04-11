/*
 * MinIO Client, (C) 2015, 2016, 2017 MinIO, Inc.
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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	// This is kept dummy for future purposes
	// and also to add ioFlags and globalFlags
	// in CLI registration.
	catFlags = []cli.Flag{}
)

// Display contents of a file.
var catCmd = cli.Command{
	Name:   "cat",
	Usage:  "display object contents",
	Action: mainCat,
	Before: setGlobalsFromContext,
	Flags:  append(append(catFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE [SOURCE...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
   MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

EXAMPLES:
   1. Stream an object from Amazon S3 cloud storage to mplayer standard input.
      $ {{.HelpName}} s3/mysql-backups/kubecon-mysql-operator.mpv | mplayer -

   2. Concatenate contents of file1.txt and stdin to standard output.
      $ {{.HelpName}} file1.txt - > file.txt

   3. Concatenate multiple files to one.
      $ {{.HelpName}} part.* > complete.img

   4. Save an encrypted object from Amazon S3 cloud storage to a local file.
      $ {{.HelpName}} --encrypt-key 's3/mysql-backups=32byteslongsecretkeymustbegiven1' s3/mysql-backups/backups-201810.gz > /mnt/data/recent.gz
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
func catURL(sourceURL string, encKeyDB map[string][]prefixSSEPair) *probe.Error {
	var reader io.ReadCloser
	size := int64(-1)
	switch sourceURL {
	case "-":
		reader = os.Stdin
	default:
		var err *probe.Error
		// Try to stat the object, the purpose is to extract the
		// size of S3 object so we can check if the size of the
		// downloaded object is equal to the original one. FS files
		// are ignored since some of them have zero size though they
		// have contents like files under /proc.
		client, content, err := url2Stat(sourceURL, false, encKeyDB)
		if err == nil && client.GetURL().Type == objectStorage {
			size = content.Size
		}
		if reader, err = getSourceStreamFromURL(sourceURL, encKeyDB); err != nil {
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
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
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
func mainCat(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

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
		fatalIf(catURL(url, encKeyDB).Trace(url), "Unable to read from `"+url+"`.")
	}

	return nil
}
