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
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

// List of all flags supported by find command.
var (
	findFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "exec",
			Usage: "spawn an external process for each matching object (see FORMAT)",
		},
		cli.StringFlag{
			Name:  "ignore",
			Usage: "exclude objects matching the wildcard pattern",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "find object names matching wildcard pattern",
		},
		cli.StringFlag{
			Name:  "newer-than",
			Usage: "match all objects newer than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "older-than",
			Usage: "match all objects older than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "match directory names matching wildcard pattern",
		},
		cli.StringFlag{
			Name:  "print",
			Usage: "print in custom format to STDOUT (see FORMAT)",
		},
		cli.StringFlag{
			Name:  "regex",
			Usage: "match directory and object name with PCRE regex pattern",
		},
		cli.StringFlag{
			Name:  "larger",
			Usage: "match all objects larger than specified size in units (see UNITS)",
		},
		cli.StringFlag{
			Name:  "smaller",
			Usage: "match all objects smaller than specified size in units (see UNITS)",
		},
		cli.UintFlag{
			Name:  "maxdepth",
			Usage: "limit directory navigation to specified depth",
		},
		cli.BoolFlag{
			Name:  "watch",
			Usage: "monitor a specified path for newly created object(s)",
		},
	}
)

var findCmd = cli.Command{
	Name:         "find",
	Usage:        "search for objects",
	Action:       mainFind,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(findFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} PATH [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
UNITS
  --smaller, --larger flags accept human-readable case-insensitive number
  suffixes such as "k", "m", "g" and "t" referring to the metric units KB,
  MB, GB and TB respectively. Adding an "i" to these prefixes, uses the IEC
  units, so that "gi" refers to "gibibyte" or "GiB". A "b" at the end is
  also accepted. Without suffixes the unit is bytes.

  --older-than, --newer-than flags accept the string for days, hours and minutes 
  i.e. 1d2h30m states 1 day, 2 hours and 30 minutes.

FORMAT
  Support string substitutions with special interpretations for following keywords.
  Keywords supported if target is filesystem or object storage:

     {}     --> Substitutes to full path.
     {base} --> Substitutes to basename of path.
     {dir}  --> Substitutes to dirname of the path.
     {size} --> Substitutes to object size of the path.
     {time} --> Substitutes to object modified time of the path.

  Keywords supported if target is object storage:

     {url} --> Substitutes to a shareable URL of the path.

EXAMPLES:
  01. Find all "foo.jpg" in all buckets under "s3" account.
      {{.Prompt}} {{.HelpName}} s3 --name "foo.jpg"

  02. Find all objects with ".txt" extension under "s3/mybucket".
      {{.Prompt}} {{.HelpName}} s3/mybucket --name "*.txt"

  03. Find only the object names without the directory component under "s3/mybucket".
      {{.Prompt}} {{.HelpName}} s3/mybucket --name "*" -print {base}

  04. Find all images with ".jpg" extension under "s3/photos", prefixed with "album".
      {{.Prompt}} {{.HelpName}} s3/photos --name "*.jpg" --path "*/album*/*"

  05. Find all images with ".jpg", ".png", and ".gif" extensions, using regex under "s3/photos".
      {{.Prompt}} {{.HelpName}} s3/photos --regex "(?i)\.(jpg|png|gif)$"

  06. Find all images with ".jpg" extension under "s3/bucket" and copy to "play/bucket" *continuously*.
      {{.Prompt}} {{.HelpName}} s3/bucket --name "*.jpg" --watch --exec "mc cp {} play/bucket"

  07. Find and generate public URLs valid for 7 days, for all objects between 64 MB, and 1 GB in size under "s3" account.
      {{.Prompt}} {{.HelpName}} s3 --larger 64MB --smaller 1GB --print {url}

  08. Find all objects created in the last week under "s3/bucket".
      {{.Prompt}} {{.HelpName}} s3/bucket --newer-than 7d

  09. Find all objects which were created are older than 2 days, 5 hours and 10 minutes and exclude the ones with ".jpg"
      extension under "s3".
      {{.Prompt}} {{.HelpName}} s3 --older-than 2d5h10m --ignore "*.jpg"

  10. List all objects up to 3 levels sub-directory deep under "s3/bucket".
      {{.Prompt}} {{.HelpName}} s3/bucket --maxdepth 3
`,
}

// checkFindSyntax - validate the passed arguments
func checkFindSyntax(ctx context.Context, cliCtx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	args := cliCtx.Args()
	if !args.Present() {
		args = []string{"./"} // No args just default to present directory.
	} else if args.Get(0) == "." {
		args[0] = "./" // If the arg is '.' treat it as './'.
	}

	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}

	// Extract input URLs and validate.
	for _, url := range args {
		_, _, err := url2Stat(ctx, url, "", false, encKeyDB, time.Time{})
		if err != nil && !isURLPrefixExists(url, false) {
			// Bucket name empty is a valid error for 'find myminio' unless we are using watch, treat it as such.
			if _, ok := err.ToGoError().(BucketNameEmpty); ok && !cliCtx.Bool("watch") {
				continue
			}
			fatalIf(err.Trace(url), "Unable to stat `"+url+"`.")
		}
	}
}

// Find context is container to hold all parsed input arguments,
// each parsed input is stored in its native typed form for
// ease of repurposing.
type findContext struct {
	*cli.Context
	execCmd       string
	ignorePattern string
	namePattern   string
	pathPattern   string
	regexPattern  string
	maxDepth      uint
	printFmt      string
	olderThan     string
	newerThan     string
	largerSize    uint64
	smallerSize   uint64
	watch         bool

	// Internal values
	targetAlias   string
	targetURL     string
	targetFullURL string
	clnt          Client
}

// mainFind - handler for mc find commands
func mainFind(cliCtx *cli.Context) error {
	ctx, cancelFind := context.WithCancel(globalContext)
	defer cancelFind()

	// Additional command specific theme customization.
	console.SetColor("Find", color.New(color.FgGreen, color.Bold))
	console.SetColor("FindExecErr", color.New(color.FgRed, color.Italic, color.Bold))

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	checkFindSyntax(ctx, cliCtx, encKeyDB)

	args := cliCtx.Args()
	if !args.Present() {
		args = []string{"./"} // Not args present default to present directory.
	} else if args.Get(0) == "." {
		args[0] = "./" // If the arg is '.' treat it as './'.
	}

	clnt, err := newClient(args[0])
	fatalIf(err.Trace(args...), "Unable to initialize `"+args[0]+"`.")

	var olderThan, newerThan string

	if cliCtx.String("older-than") != "" {
		olderThan = cliCtx.String("older-than")
	}
	if cliCtx.String("newer-than") != "" {
		newerThan = cliCtx.String("newer-than")
	}

	// Use 'e' to indicate Go error, this is a convention followed in `mc`. For probe.Error we call it
	// 'err' and regular Go error is called as 'e'.
	var e error
	var largerSize, smallerSize uint64

	if cliCtx.String("larger") != "" {
		largerSize, e = humanize.ParseBytes(cliCtx.String("larger"))
		fatalIf(probe.NewError(e).Trace(cliCtx.String("larger")), "Unable to parse input bytes.")
	}

	if cliCtx.String("smaller") != "" {
		smallerSize, e = humanize.ParseBytes(cliCtx.String("smaller"))
		fatalIf(probe.NewError(e).Trace(cliCtx.String("smaller")), "Unable to parse input bytes.")
	}

	targetAlias, _, hostCfg, err := expandAlias(args[0])
	fatalIf(err.Trace(args[0]), "Unable to expand alias.")

	var targetFullURL string
	if hostCfg != nil {
		targetFullURL = hostCfg.URL
	}

	return doFind(ctx, &findContext{
		Context:       cliCtx,
		maxDepth:      cliCtx.Uint("maxdepth"),
		execCmd:       cliCtx.String("exec"),
		printFmt:      cliCtx.String("print"),
		namePattern:   cliCtx.String("name"),
		pathPattern:   cliCtx.String("path"),
		regexPattern:  cliCtx.String("regex"),
		ignorePattern: cliCtx.String("ignore"),
		olderThan:     olderThan,
		newerThan:     newerThan,
		largerSize:    largerSize,
		smallerSize:   smallerSize,
		watch:         cliCtx.Bool("watch"),
		targetAlias:   targetAlias,
		targetURL:     args[0],
		targetFullURL: targetFullURL,
		clnt:          clnt,
	})
}
