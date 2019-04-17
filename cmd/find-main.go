/*
 * MinIO Client (C) 2017 MinIO, Inc.
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
	Name:   "find",
	Usage:  "search for objects",
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

   09. Find all objects which were created are older than 2 days, 5 hours and 10 minutes and exclude the ones with ".jpg"
       extension under "s3".
       $ {{.HelpName}} s3 --older-than 2d5h10m --ignore "*.jpg"

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

	var olderThan, newerThan string

	if ctx.String("older-than") != "" {
		olderThan = ctx.String("older-than")
	}
	if ctx.String("newer-than") != "" {
		newerThan = ctx.String("newer-than")
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
