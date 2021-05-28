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
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

const (
	treeEntry     = "├─ "
	treeLastEntry = "└─ "
	treeNext      = "│"
	treeLevel     = "  "
)

// Structured message depending on the type of console.
type treeMessage struct {
	Entry        string
	IsDir        bool
	BranchString string
}

// Colorized message for console printing.
func (t treeMessage) String() string {
	entryType := "File"
	if t.IsDir {
		entryType = "Dir"
	}
	return fmt.Sprintf("%s%s", t.BranchString, console.Colorize(entryType, t.Entry))
}

// JSON'ified message for scripting.
// Does No-op. JSON requests are redirected to `ls -r --json`
func (t treeMessage) JSON() string {
	fatalIf(probe.NewError(errors.New("JSON() should never be called here")), "Unable to list in tree format. Please report this issue at https://github.com/minio/mc/issues")
	return ""
}

var treeFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "files, f",
		Usage: "includes files in tree",
	},
	cli.IntFlag{
		Name:  "depth, d",
		Usage: "sets the depth threshold",
		Value: -1,
	},
	cli.StringFlag{
		Name:  "rewind",
		Usage: "display tree no later than specified date",
	},
}

// trees files and folders.
var treeCmd = cli.Command{
	Name:         "tree",
	Usage:        "list buckets and objects in a tree format",
	Action:       mainTree,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(treeFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET ...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. List all buckets and directories on MinIO object storage server in tree format.
      {{.Prompt}} {{.HelpName}} myminio

   2. List all directories in "mybucket" on MinIO object storage server in tree format.
      {{.Prompt}} {{.HelpName}} myminio/mybucket/

   3. List all directories in "mybucket" on MinIO object storage server hosted on Microsoft Windows in tree format.
      {{.Prompt}} {{.HelpName}} myminio\mybucket\

   4. List all directories and objects in "mybucket" on MinIO object storage server in tree format.
      {{.Prompt}} {{.HelpName}} --files myminio/mybucket/

   5. List all directories upto depth level '2' in tree format.
      {{.Prompt}} {{.HelpName}} --depth 2 myminio/mybucket/
`,
}

// parseTreeSyntax - validate all the passed arguments
func parseTreeSyntax(ctx context.Context, cliCtx *cli.Context) (args []string, depth int, files bool, timeRef time.Time) {
	args = cliCtx.Args()
	depth = cliCtx.Int("depth")
	files = cliCtx.Bool("files")

	rewind := cliCtx.String("rewind")
	timeRef = parseRewindFlag(rewind)

	if depth < -1 || cliCtx.Int("depth") == 0 {
		fatalIf(errInvalidArgument().Trace(args...),
			"please set a proper depth, for example: '--depth 1' to limit the tree output, default (-1) output displays everything")
	}

	if len(args) == 0 {
		return
	}

	for _, url := range args {
		if _, _, err := url2Stat(ctx, url, "", false, nil, timeRef); err != nil && !isURLPrefixExists(url, false) {
			fatalIf(err.Trace(url), "Unable to tree `"+url+"`.")
		}
	}
	return
}

// doTree - list all entities inside a folder in a tree format.
func doTree(ctx context.Context, url string, timeRef time.Time, level int, leaf bool, branchString string, depth int, includeFiles bool) error {

	targetAlias, targetURL, _ := mustExpandAlias(url)
	if !strings.HasSuffix(targetURL, "/") {
		targetURL += "/"
	}

	clnt, err := newClientFromAlias(targetAlias, targetURL)
	fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")

	prefixPath := clnt.GetURL().Path
	separator := string(clnt.GetURL().Separator)
	if !strings.HasSuffix(prefixPath, separator) {
		prefixPath = filepath.Dir(prefixPath) + "/"
	}

	bucketNameShowed := false
	var prev *ClientContent
	show := func(end bool) error {
		currbranchString := branchString
		if level == 1 && !bucketNameShowed {
			bucketNameShowed = true
			printMsg(treeMessage{
				Entry:        url,
				IsDir:        true,
				BranchString: branchString,
			})
		}

		isLevelClosed := strings.HasSuffix(currbranchString, treeLastEntry)
		if isLevelClosed {
			currbranchString = strings.TrimSuffix(currbranchString, treeLastEntry)
		} else {
			currbranchString = strings.TrimSuffix(currbranchString, treeEntry)
		}

		if level != 1 {
			if isLevelClosed {
				currbranchString += " " + treeLevel
			} else {
				currbranchString += treeNext + treeLevel
			}
		}

		if end {
			currbranchString += treeLastEntry
		} else {
			currbranchString += treeEntry
		}

		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(prev.URL.Path)
		prefixPath = filepath.ToSlash(prefixPath)

		// Trim prefix of current working dir
		prefixPath = strings.TrimPrefix(prefixPath, "."+separator)

		if prev.Type.IsDir() {
			printMsg(treeMessage{
				Entry:        strings.TrimSuffix(strings.TrimPrefix(contentURL, prefixPath), "/"),
				IsDir:        true,
				BranchString: currbranchString,
			})
		} else {
			printMsg(treeMessage{
				Entry:        strings.TrimPrefix(contentURL, prefixPath),
				IsDir:        false,
				BranchString: currbranchString,
			})
		}

		if prev.Type.IsDir() {
			url := ""
			if targetAlias != "" {
				url = targetAlias + "/" + contentURL
			} else {
				url = contentURL
			}

			if depth == -1 || level <= depth {
				if err := doTree(ctx, url, timeRef, level+1, end, currbranchString, depth, includeFiles); err != nil {
					return err
				}
			}
		}

		return nil
	}

	for content := range clnt.List(ctx, ListOptions{Recursive: false, TimeRef: timeRef, ShowDir: DirFirst}) {
		if content.Err != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to tree.")
			continue
		}

		if !includeFiles && !content.Type.IsDir() {
			continue
		}

		if prev != nil {
			if err := show(false); err != nil {
				return err
			}
		}

		prev = content
	}

	if prev != nil {
		if err := show(true); err != nil {
			return err
		}
	}

	return nil
}

// mainTree - is a handler for mc tree command
func mainTree(cliCtx *cli.Context) error {
	ctx, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	console.SetColor("File", color.New(color.Bold))
	console.SetColor("Dir", color.New(color.FgCyan, color.Bold))

	// parse 'tree' cliCtx arguments.
	args, depth, includeFiles, timeRef := parseTreeSyntax(ctx, cliCtx)

	// mimic operating system tool behavior.
	if len(args) == 0 {
		args = []string{"."}
	}

	var cErr error
	for _, targetURL := range args {
		if !globalJSON {
			if e := doTree(ctx, targetURL, timeRef, 1, false, "", depth, includeFiles); e != nil {
				cErr = e
			}
		} else {
			targetAlias, targetURL, _ := mustExpandAlias(targetURL)
			if !strings.HasSuffix(targetURL, "/") {
				targetURL += "/"
			}
			clnt, err := newClientFromAlias(targetAlias, targetURL)
			fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")
			if e := doList(ctx, clnt, true, false, false, timeRef, false); e != nil {
				cErr = e
			}
		}
	}
	return cErr
}
