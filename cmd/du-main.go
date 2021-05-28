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
	"fmt"
	"net/url"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

// du specific flags.
var (
	duFlags = []cli.Flag{
		cli.IntFlag{
			Name:  "depth, d",
			Usage: "print the total for a folder prefix only if it is N or fewer levels below the command line argument",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "recursively print the total for a folder prefix",
		},
		cli.StringFlag{
			Name:  "rewind",
			Usage: "include all object versions no later than specified date",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "include all object versions",
		},
	}
)

// Summarize disk usage.
var duCmd = cli.Command{
	Name:         "du",
	Usage:        "summarize disk usage recursively",
	Action:       mainDu,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(duFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
  MC_ENCRYPT_KEY: list of comma delimited prefix=secret values

EXAMPLES:
  1. Summarize disk usage of 'jazz-songs' bucket recursively.
     {{.Prompt}} {{.HelpName}} s3/jazz-songs

  2. Summarize disk usage of 'louis' prefix in 'jazz-songs' bucket upto two levels.
     {{.Prompt}} {{.HelpName}} --depth=2 s3/jazz-songs/louis/

  3. Summarize disk usage of 'jazz-songs' bucket at a fixed date/time
     {{.Prompt}} {{.HelpName}} --rewind "2020.01.01" s3/jazz-songs/

  4. Summarize disk usage of 'jazz-songs' bucket with all objects versions
     {{.Prompt}} {{.HelpName}} --versions s3/jazz-songs/
`,
}

// Structured message depending on the type of console.
type duMessage struct {
	Prefix string `json:"prefix"`
	Size   int64  `json:"size"`
	Status string `json:"status"`
}

// Colorized message for console printing.
func (r duMessage) String() string {
	humanSize := strings.Join(strings.Fields(humanize.IBytes(uint64(r.Size))), "")

	return fmt.Sprintf("%s\t%s", console.Colorize("Size", humanSize),
		console.Colorize("Prefix", r.Prefix))
}

// JSON'ified message for scripting.
func (r duMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func du(urlStr string, timeRef time.Time, withVersions bool, depth int, encKeyDB map[string][]prefixSSEPair) (int64, error) {
	targetAlias, targetURL, _ := mustExpandAlias(urlStr)
	if !strings.HasSuffix(targetURL, "/") {
		targetURL += "/"
	}

	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(urlStr), "Failed to summarize disk usage `"+urlStr+"`.")
		return 0, exitStatus(globalErrorExitStatus) // End of journey.
	}

	contentCh := clnt.List(globalContext, ListOptions{
		TimeRef:           timeRef,
		WithOlderVersions: withVersions,
		Recursive:         false,
		ShowDir:           DirFirst,
	})
	size := int64(0)
	for content := range contentCh {
		if content.Err != nil {
			switch content.Err.ToGoError().(type) {
			// handle this specifically for filesystem related errors.
			case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, ObjectOnGlacier:
				continue
			case PathInsufficientPermission:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			}
			errorIf(content.Err.Trace(urlStr), "Failed to find disk usage of `"+urlStr+"` recursively.")
			return 0, exitStatus(globalErrorExitStatus)
		}
		if content.URL.String() == targetURL {
			continue
		}

		if content.Type.IsDir() {
			depth := depth
			if depth > 0 {
				depth--
			}

			subDirAlias := content.URL.Path
			if targetAlias != "" {
				subDirAlias = targetAlias + "/" + content.URL.Path
			}
			used, err := du(subDirAlias, timeRef, withVersions, depth, encKeyDB)
			if err != nil {
				return 0, err
			}
			size += used
		} else {
			size += content.Size
		}
	}

	if depth != 0 {
		u, err := url.Parse(targetURL)
		if err != nil {
			panic(err)
		}

		printMsg(duMessage{
			Prefix: strings.Trim(u.Path, "/"),
			Size:   size,
			Status: "success",
		})
	}

	return size, nil
}

// main for du command.
func mainDu(ctx *cli.Context) error {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "du", 1)
	}

	// Set colors.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))
	console.SetColor("Prefix", color.New(color.FgCyan, color.Bold))
	console.SetColor("Size", color.New(color.FgYellow))

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// du specific flags.
	depth := ctx.Int("depth")
	if depth == 0 {
		if ctx.Bool("recursive") {
			if !ctx.IsSet("depth") {
				depth = -1
			}
		} else {
			depth = 1
		}
	}

	withVersions := ctx.Bool("versions")
	timeRef := parseRewindFlag(ctx.String("rewind"))

	var duErr error
	for _, urlStr := range ctx.Args() {
		if _, err := du(urlStr, timeRef, withVersions, depth, encKeyDB); duErr == nil {
			duErr = err
		}
	}

	return duErr
}
