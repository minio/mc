/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"fmt"
	"strings"

	neturl "net/url"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// du specific flags.
var (
	duFlags = []cli.Flag{
		cli.IntFlag{
			Name:  "max-depth, d",
			Usage: "print the total for a folder prefix only if it is N or fewer levels below the command line argument",
		},
		cli.BoolFlag{
			Name:  "human-readable, H",
			Usage: "print sizes in human readable format (e.g., 1K 234M 2G)",
		},
	}
)

// Summarize disk usage.
var duCmd = cli.Command{
	Name:   "du",
	Usage:  "Summarize disk usage folder prefixes recursively.",
	Action: mainDu,
	Before: setGlobalsFromContext,
	Flags:  append(append(duFlags, ioFlags...), globalFlags...),
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
      $ {{.HelpName}} s3/jazz-songs

   2. Summarize disk usage of 'louis' prefix in 'jazz-songs' bucket recursively.
      $ {{.HelpName}} -H s3/jazz-songs/louis/
`,
}

// Structured message depending on the type of console.
type duMessage struct {
	Prefix string `json:"prefix"`
	Size   string `json:"size"`
	Status string `json:"status"`
}

// Colorized message for console printing.
func (r duMessage) String() string {
	return console.Colorize("Du", fmt.Sprintf("%s\t%s", r.Size, r.Prefix))
}

// JSON'ified message for scripting.
func (r duMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func du(url string, isHumanReadable bool, maxDepth int, encKeyDB map[string][]prefixSSEPair) (int64, error) {
	targetAlias, targetURL, _ := mustExpandAlias(url)
	if !strings.HasSuffix(targetURL, "/") {
		targetURL += "/"
	}

	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		errorIf(pErr.Trace(url), "Failed to summarize disk usage `"+url+"`.")
		return 0, exitStatus(globalErrorExitStatus) // End of journey.
	}

	isRecursive := false
	isIncomplete := false
	contentCh := clnt.List(isRecursive, isIncomplete, DirFirst)
	size := int64(0)
	for content := range contentCh {
		if content.Err != nil {
			errorIf(content.Err.Trace(url), "Failed to find disk usage of `"+url+"` recursively.")
			return 0, exitStatus(globalErrorExitStatus)
		}

		if content.URL.String() == targetURL {
			continue
		}

		if content.Type.IsDir() {
			depth := maxDepth
			if depth > 0 {
				depth--
			}

			subDirAlias := content.URL.Path
			if targetAlias != "" {
				subDirAlias = targetAlias + "/" + content.URL.Path
			}
			used, err := du(subDirAlias, isHumanReadable, depth, encKeyDB)
			if err != nil {
				return 0, err
			}
			size += used
		} else {
			size += content.Size
		}
	}

	if maxDepth != 0 {
		var sizeStr string
		if isHumanReadable {
			sizeStr = humanize.Bytes(uint64(size))
		} else {
			sizeStr = fmt.Sprintf("%d", size)
		}

		u, err := neturl.Parse(targetURL)
		if err != nil {
			panic(err)
		}

		printMsg(duMessage{
			Prefix: strings.Trim(u.Path, "/"),
			Size:   sizeStr,
			Status: "success",
		})
	}

	return size, nil
}

// main for du command.
func mainDu(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// du specific flags.
	isHumanReadable := ctx.Bool("human-readable")
	maxDepth := ctx.Int("max-depth")
	if maxDepth == 0 {
		maxDepth = -1
	}

	// Set color.
	console.SetColor("Remove", color.New(color.FgGreen, color.Bold))

	var duErr error
	for _, url := range ctx.Args() {
		if _, err := du(url, isHumanReadable, maxDepth, encKeyDB); duErr == nil {
			duErr = err
		}
	}

	return duErr
}
