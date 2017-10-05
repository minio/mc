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

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

// find specific flags
var (
	findFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "Find object names matching wildcard pattern",
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "Match directory names matching wildcard pattern",
		},
		cli.StringFlag{
			Name:  "regex",
			Usage: "Match directory and object name with PCRE regex pattern",
		},
		cli.StringFlag{
			Name:  "print",
			Usage: "Print in custom format to STDOUT (see FORMAT)",
		},
		cli.StringFlag{
			Name:  "exec",
			Usage: "Spawn an external process for each matching object (see FORMAT)",
		},
		cli.StringFlag{
			Name:  "ignore",
			Usage: "Exclude objects matching the wildcard pattern",
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
			Name:  "larger",
			Usage: "Match all objects larger than specified size in units (see UNITS)",
		},
		cli.StringFlag{
			Name:  "smaller",
			Usage: "Match all objects smaller than specified size in units (see UNITS)",
		},
		cli.StringFlag{
			Name:  "maxdepth",
			Usage: "Limit directory navigation to specified depth",
		},
		cli.BoolFlag{
			Name:  "watch",
			Usage: "Monitor specified location for newly created objects",
		},
		cli.BoolFlag{
			Name:  "or",
			Usage: "Changes the matching criteria from an \"and\" to an \"or\"",
		},
	}
)

var findCmd = cli.Command{
	Name:   "find",
	Usage:  "Finds files which match the given set of parameters.",
	Action: mainFind,
	Before: setGlobalsFromContext,
	Flags:  append(findFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} PATH [FLAG...]

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
   01. Find all files named foo from all buckets.
       $ {{.HelpName}} s3 --name "file"

   02. Find all text files from mybucket.
       $ {{.HelpName}} s3/mybucket --name "*.txt"

   03. Print only the object names without the directory component under this bucket.
       $ {{.HelpName}} s3/bucket --name "*" -print {base}

   04. Copy all jpg files from AWS S3 photos bucket to minio play test bucket.
       $ {{.HelpName}} s3/photos --name "*.jpg" --exec "mc cp {} play/test"

   05. Find all jpg images from any folder prefixed with album.
       $ {{.HelpName}} s3/photos --name "*.jpg" --path "*/album*/*"

   06. Find all jpgs, pngs, and gifs using regex
       $ {{.HelpName}} s3/photos --regex "(?i)\.(jpg|png|gif)$"

   07. Mirror all photos from s3 bucket *coninuously* from the s3 bucket to minio play test bucket.
       $ {{.HelpName}} s3/buck --name "*foo" --watch --exec "mc cp {} play/test"

   08. Generate self expiring urls (7 days), for all objects between 64 MB, and 1 GB in size.
       $ {{.HelpName}} s3 --larger 64MB --smaller 1GB --print {url}

   09. Find all files under the s3 bucket which were created within a week.
       $ {{.HelpName}} s3/bucket --newer 1w

   10. Find all files which were created more than 6 months ago ignoring files ending in jpg.
       $ {{.HelpName}} s3 --older 6m --ignore "*.jpg"

   11. List all objects up to 3 levels subdirectory deep.
       $ {{.HelpName}} s3/bucket --maxdepth 3

`,
}

// checkFindSyntax - validate the passed arguments
func checkFindSyntax(ctx *cli.Context) {
	args := ctx.Args()

	// help message on [mc][find]
	if !args.Present() {
		cli.ShowCommandHelpAndExit(ctx, "find", 1)
	}

	if ctx.Bool("watch") && !strings.Contains(args[0], "/") {
		console.Println("Users must specify a bucket name for watch")
		console.Fatalln()
	}

	// verify that there are no empty arguments
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
		}
	}

}

// mainFind - handler for mc find commands
func mainFind(ctx *cli.Context) error {
	// Additional command specific theme customization.
	console.SetColor("Find", color.New(color.FgGreen, color.Bold))

	checkFindSyntax(ctx)

	args := ctx.Args()

	if !ctx.Args().Present() {
		args = []string{"."}
	}

	clnt, err := newClient(args[0])
	fatalIf(err.Trace(args...), "Unable to initialize `"+args[0]+"`")

	return doFind(args[0], clnt, ctx)
}
