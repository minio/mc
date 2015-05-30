/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

package main

import (
	"fmt"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

//// List of commands

var lsCmd = cli.Command{
	Name:   "ls",
	Usage:  "List files and folders",
	Action: runListCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. List objects recursively on Minio object storage
      $ mc {{.Name}} http://play.minio.io:9000/backup/...
      [2015-03-28 12:47:50 PDT]  34MiB 2006-Jan-1/backup.tar.gz
      [2015-03-31 14:46:33 PDT]  55MiB 2006-Mar-1/backup.tar.gz

   2. List buckets on Amazon S3 object storage
      $ mc {{.Name}} https://s3.amazonaws.com/
      [2015-01-20 15:42:00 PST]     0B rom/
      [2015-01-15 00:05:40 PST]     0B zek/

   3. List buckets from Amazon S3 object storage and recursively list objects from Minio object storage
      $ mc {{.Name}} https://s3.amazonaws.com/ http://play.minio.io:9000/backup/...
      2015-01-15 00:05:40 PST     0B zek/
      2015-03-31 14:46:33 PDT  55MiB 2006-Mar-1/backup.tar.gz

   4. List files recursively on local filesystem on Windows
      $ mc {{.Name}} C:\Users\Worf\...
      [2015-03-28 12:47:50 PDT] 11.00MiB Martok\Klingon Council Ministers.pdf
      [2015-03-31 14:46:33 PDT] 15.00MiB Gowron\Khitomer Conference Details.pdf

   5. List files with non english characters on Amazon S3 object storage
      $ mc ls s3:andoria/本...
      [2015-05-19 17:21:49 PDT]    41B 本語.pdf
      [2015-05-19 17:24:19 PDT]    41B 本語.txt
      [2015-05-19 17:28:22 PDT]    41B 本語.md

`,
}

var cpCmd = cli.Command{
	Name:   "cp",
	Usage:  "Copy files and folders from many sources to a single destination",
	Action: runCopyCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE [SOURCE...] TARGET {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Copy list of objects from local file system to Amazon S3 object storage
      $ mc {{.Name}} Music/*.ogg https://s3.amazonaws.com/jukebox/

   2. Copy a bucket recursively from Minio object storage to Amazon S3 object storage
      $ mc {{.Name}} http://play.minio.io:9000/photos/burningman2011... https://s3.amazonaws.com/private-photos/burningman/

   3. Copy multiple local folders recursively to Minio object storage
      $ mc {{.Name}} backup/2014/... backup/2015/... http://play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 object storage to local filesystem on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014

   5. Copy an object of non english characters to Amazon S3 object storage
      $ mc {{.Name}} 本語 s3:andoria/本語

`,
}

var syncCmd = cli.Command{
	Name:   "sync",
	Usage:  "Copy files and folders from a single source to many destinations",
	Action: runSyncCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Sync an object from local filesystem to Amazon S3 object storage
      $ mc {{.Name}} star-trek-episode-10-season4.ogg https://s3.amazonaws.com/trekarchive

   2. Sync a bucket recursively from Minio object storage to multiple buckets on Amazon S3 object storage
      $ mc {{.Name}} http://play.minio.io:9000/photos/2014... https://s3.amazonaws.com/backup-photos https://s3.amazonaws.com/my-photos

   3. Sync a local folder recursively to Minio object storage and Amazon S3 object storage
      $ mc {{.Name}} backup/... http://play.minio.io:9000/archive https://s3.amazonaws.com/archive

   4. Sync a bucket from aliased Amazon S3 object storage to multiple folders on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014 C:\shared\volume\backup\2014

   5. Sync a local file of non english character to Amazon s3 object storage
      $ mc {{.Name}} 本語/... s3:mylocaldocuments C:\backup\2014 play:backup

`,
}

var diffCmd = cli.Command{
	Name:        "diff",
	Usage:       "Compute differences between two files or folders",
	Description: "NOTE: This command *DOES NOT* check for content similarity, which means objects with same size, but different content will not be spotted.",
	Action:      runDiffCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} FIRST SECOND {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Compare foo.ogg on a local filesystem with bar.ogg on Amazon AWS cloud storage.
      $ mc {{.Name}} foo.ogg  https://s3.amazonaws.com/jukebox/bar.ogg

   2. Compare two different directories on a local filesystem.
      $ mc {{.Name}} ~/Photos /Media/Backup/Photos

`,
}

var catCmd = cli.Command{
	Name:   "cat",
	Usage:  "Display contents of a file",
	Action: runCatCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE [SOURCE...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Concantenate an object from Amazon S3 object storage to mplayer standard input
      $ mc {{.Name}} https://s3.amazonaws.com/ferenginar/klingon_opera_aktuh_maylotah.ogg | mplayer -

   2. Concantenate a file from local filesystem to standard output.
      $ mc {{.Name}} khitomer-accords.txt

   3. Concantenate multiple files from local filesystem to standard output.
      $ mc {{.Name}} *.txt > newfile.txt

   4. Concatenate a non english file name from Amazon S3 object storage
      $ mc {{.Name}} s3:andoria/本語 > /tmp/本語

`,
}

var mbCmd = cli.Command{
	Name:   "mb",
	Usage:  "Make a bucket or folder",
	Action: runMakeBucketCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Create a bucket on Amazon S3 object storage
      $ mc {{.Name}} https://s3.amazonaws.com/public-document-store

   2. Create a bucket on Minio object storage
      $ mc {{.Name}} http://play.minio.io:9000/mongodb-backup

   3. Create multiple buckets on Amazon S3 object storage and Minio object storage
      $ mc {{.Name}} https://s3.amazonaws.com/public-photo-store http://play.minio.io:9000/mongodb-backup

`,
}

var accessCmd = cli.Command{
	Name:   "access",
	Usage:  "Set access permissions",
	Action: runAccessCmd,
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} PERMISSION TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:

   1. Set bucket to "private" on Amazon S3 object storage
      $ mc {{.Name}} private https://s3.amazonaws.com/burningman2011

   2. Set bucket to "public" on Amazon S3 object storage
      $ mc {{.Name}} public https://s3.amazonaws.com/shared

   3. Set bucket to "authenticated" on Amazon S3 object storage to provide read access to IAM Authenticated Users group
      $ mc {{.Name}} authenticated https://s3.amazonaws.com/shared-authenticated

   4. Set folder to world readwrite (chmod 777) on local filesystem
      $ mc {{.Name}} public /shared/Music

`,
}

//   Configure minio client
//
//   ----
//   NOTE: that the configure command only writes values to the config file.
//   It does not use any configuration values from the environment variables.
//
//   One needs to edit configuration file manually, this is purposefully done
//   so to avoid taking credentials over cli arguments. It is a security precaution
//   ----
//
var configCmd = cli.Command{
	Name:   "config",
	Usage:  "Generate default configuration file [~/.mc/config.json]",
	Action: runConfigCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} generate
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} alias NAME HOSTURL

EXAMPLES:
   1. Generate mc config
      $ mc config generate

   2. Add alias URLs
      $ mc config alias zek https://s3.amazonaws.com/

`,
}

var updateCmd = cli.Command{
	Name:   "update",
	Usage:  "Check for new software updates",
	Action: runUpdateCmd,
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}}

EXAMPLES:
   1. Check for new updates
      $ mc update

`,
}

// Collection of mc commands currently supported are
//
//  ls     List files and objects
//  cp     Copy objects and files from multiple sources to single destination
//  sync   Copy objects and files from single source to multiple destionations
//  mb     Make a bucket
//  access Set permissions [public, private, readonly, authenticated] for buckets and folders.
//  cat    Concantenate an object to standard output
//  config Generate configuration "/home/harsha/.mc/config.json" file.
//  update Check for new software updates
//
var commands = []cli.Command{
	lsCmd,
	mbCmd,
	catCmd,
	cpCmd,
	syncCmd,
	diffCmd,
	accessCmd,
	configCmd,
	updateCmd,
	// Add your new commands starting from here
}

// Collection of mc flags currently supported
//
//  --theme       "minimal" Choose a console theme from this list [*minimal*, nocolor, white]
//  --json        Enable json formatted output
//  --debug       Enable HTTP tracing
//  --quiet, -q   Supress chatty console output
//  --version, -v print the version
//
var (
	flags = []cli.Flag{
		cli.StringFlag{
			Name:  "theme",
			Value: console.GetDefaultThemeName(),
			Usage: fmt.Sprintf("Choose a console theme from this list [%s]", func() string {
				keys := []string{}
				for _, themeName := range console.GetThemeNames() {
					if console.GetThemeName() == themeName {
						themeName = "*" + themeName + "*"
					}
					keys = append(keys, themeName)
				}
				return strings.Join(keys, ", ")
			}()),
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "Enable json formatted output",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable HTTP tracing",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "Supress chatty console output",
		},
		// Add your new flags starting here
	}
)
