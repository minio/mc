/*
 * Minio Client (C) 2017 Minio, Inc.
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
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// List of all flags supported by find command.
var (
	findFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "exec",
			Usage: "Spawn an external process for each matching object (see FORMAT)",
		},
		cli.StringFlag{
			Name:  "ignore",
			Usage: "Exclude objects matching the wildcard pattern",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "Find object names matching wildcard pattern",
		},
		cli.StringFlag{
			Name:  "newer",
			Usage: "Match all objects newer than specified time in units (see UNITS)",
		},
		cli.StringFlag{
			Name:  "older",
			Usage: "Match all objects older than specified time in units (see UNITS)",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "Match directory names matching wildcard pattern",
		},
		cli.StringFlag{
			Name:  "print",
			Usage: "Print in custom format to STDOUT (see FORMAT)",
		},
		cli.StringFlag{
			Name:  "regex",
			Usage: "Match directory and object name with PCRE regex pattern",
		},
		cli.StringFlag{
			Name:  "larger",
			Usage: "Match all objects larger than specified size in units (see UNITS)",
		},
		cli.StringFlag{
			Name:  "smaller",
			Usage: "Match all objects smaller than specified size in units (see UNITS)",
		},
		cli.UintFlag{
			Name:  "maxdepth",
			Usage: "Limit directory navigation to specified depth",
		},
		cli.BoolFlag{
			Name:  "watch",
			Usage: "Monitor a specified path for newly created files and objects",
		},
	}
)

var findCmd = cli.Command{
	Name:   "find",
	Usage:  "Search for files and objects.",
	Action: mainFind,
	Before: setGlobalsFromContext,
	Flags:  append(findFlags, globalFlags...),
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

   --older, --newer flags accept the suffixes "d", "w", "m" and "y" to refer
   to units of days, weeks, months and years respectively. With the standard
   rate of conversion being 7 days in 1 week, 30 days in 1 month, and 365
   days in one year.

FORMAT
   Support string substitutions with special interpretations for following keywords.
   Keywords supported if target is filesystem or object storage:

      {}     --> Substitutes to full path.
      {base} --> Substitutes to basename of path.
      {dir}  --> Substitutes to dirname of the path.
      {size} --> Substitutes to file size of the path.
      {time} --> Substitutes to file modified time of the path.

   Keywords supported if target is object storage:

      {url} --> Substitutes to a shareable URL of the path.

EXAMPLES:
   01. Find all "foo.jpg" in all buckets under "s3" account.
       $ {{.HelpName}} s3 --name "foo.jpg"

   02. Find all objects with ".txt" extension under "s3/mybucket".
       $ {{.HelpName}} s3/mybucket --name "*.txt"

   03. Find only the object names without the directory component under "s3/mybucket".
       $ {{.HelpName}} s3/mybucket --name "*" -print {base}

   04. Find all images with ".jpg" extension under "s3/photos", prefixed with "album".
       $ {{.HelpName}} s3/photos --name "*.jpg" --path "*/album*/*"

   05. Find all images with ".jpg", ".png", and ".gif" extensions, using regex under "s3/photos".
       $ {{.HelpName}} s3/photos --regex "(?i)\.(jpg|png|gif)$"

   06. Find all images with ".jpg" extension under "s3/bucket" and copy to "play/bucket" *continuously*.
       $ {{.HelpName}} s3/bucket --name "*.jpg" --watch --exec "mc cp {} play/bucket"

   07. Find and generate public URLs valid for 7 days, for all objects between 64 MB, and 1 GB in size under "s3" account.
       $ {{.HelpName}} s3 --larger 64MB --smaller 1GB --print {url}

   08. Find all objects created in the last week under "s3/bucket".
       $ {{.HelpName}} s3/bucket --newer 1w

   09. Find all objects which were created more than 6 months ago, and exclude the ones with ".jpg"
       extension under "s3".
       $ {{.HelpName}} s3 --older 6m --ignore "*.jpg"

   10. List all objects up to 3 levels sub-directory deep under "s3/bucket".
       $ {{.HelpName}} s3/bucket --maxdepth 3

`,
}

// checkFindSyntax - validate the passed arguments
func checkFindSyntax(ctx *cli.Context, encKeyDB map[string][]prefixSSEPair) {
	args := ctx.Args()
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
		_, _, err := url2Stat(url, false, encKeyDB)
		if err != nil && !isURLPrefixExists(url, false) {
			// Bucket name empty is a valid error for 'find myminio' unless we are using watch, treat it as such.
			if _, ok := err.ToGoError().(BucketNameEmpty); ok && !ctx.Bool("watch") {
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
	olderThan     time.Time
	newerThan     time.Time
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
func mainFind(ctx *cli.Context) error {
	// Additional command specific theme customization.
	console.SetColor("Find", color.New(color.FgGreen, color.Bold))
	console.SetColor("FindExecErr", color.New(color.FgRed, color.Italic, color.Bold))

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	checkFindSyntax(ctx, encKeyDB)

	args := ctx.Args()
	if !args.Present() {
		args = []string{"./"} // Not args present default to present directory.
	} else if args.Get(0) == "." {
		args[0] = "./" // If the arg is '.' treat it as './'.
	}

	clnt, err := newClient(args[0])
	fatalIf(err.Trace(args...), "Unable to initialize `"+args[0]+"`.")

	var olderThan, newerThan time.Time

	if ctx.String("older") != "" {
		olderThan, err = parseTime(ctx.String("older"))
		fatalIf(err.Trace(ctx.String("older")), "Unable to parse input time.")
	}
	if ctx.String("newer") != "" {
		newerThan, err = parseTime(ctx.String("newer"))
		fatalIf(err.Trace(ctx.String("newer")), "Unable to parse input time.")
	}

	// Use 'e' to indicate Go error, this is a convention followed in `mc`. For probe.Error we call it
	// 'err' and regular Go error is called as 'e'.
	var e error
	var largerSize, smallerSize uint64

	if ctx.String("larger") != "" {
		largerSize, e = humanize.ParseBytes(ctx.String("larger"))
		fatalIf(probe.NewError(e).Trace(ctx.String("larger")), "Unable to parse input bytes.")
	}

	if ctx.String("smaller") != "" {
		smallerSize, e = humanize.ParseBytes(ctx.String("smaller"))
		fatalIf(probe.NewError(e).Trace(ctx.String("smaller")), "Unable to parse input bytes.")
	}

	targetAlias, _, hostCfg, err := expandAlias(args[0])
	fatalIf(err.Trace(args[0]), "Unable to expand alias.")

	var targetFullURL string
	if hostCfg != nil {
		targetFullURL = hostCfg.URL
	}

	return doFind(&findContext{
		Context:       ctx,
		maxDepth:      ctx.Uint("maxdepth"),
		execCmd:       ctx.String("exec"),
		printFmt:      ctx.String("print"),
		namePattern:   ctx.String("name"),
		pathPattern:   ctx.String("path"),
		regexPattern:  ctx.String("regex"),
		ignorePattern: ctx.String("ignore"),
		olderThan:     olderThan,
		newerThan:     newerThan,
		largerSize:    largerSize,
		smallerSize:   smallerSize,
		watch:         ctx.Bool("watch"),
		targetAlias:   targetAlias,
		targetURL:     args[0],
		targetFullURL: targetFullURL,
		clnt:          clnt,
	})
}
