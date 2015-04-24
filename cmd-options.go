/*
 * Mini Copy (C) 2014, 2015 Minio, Inc.
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

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
)

// List of commands
var (
	accessCmd = cli.Command{
		Name:   "access",
		Usage:  "Set permissions [public, private, readonly] for buckets and folders.",
		Action: runAccessCmd,
		CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} PERMISSION TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:

   1. Set bucket to "private" on Amazon S3 object storage
      $ mc {{.Name}} private https://s3.amazonaws.com/burningman2011

   2. Set bucket to "public" on Amazon S3 object storage
      $ mc {{.Name}} public https://s3.amazonaws.com/shared

   3. Set folder to world readwrite (chmod 777) on local filesystem
      $ mc {{.Name}} public /shared/Music

`,
	}
	catCmd = cli.Command{
		Name:   "cat",
		Usage:  "Concantenate an object to standard output",
		Action: runCatCmd,
		CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Concantenate an object from Amazon S3 object storage to mplayer standard input
      $ mc {{.Name}} https://s3.amazonaws.com/jukebox/klingon_opera_aktuh_maylotah.ogg | mplayer -

   2. Concantenate a file from local filesystem to standard output.
      $ mc {{.Name}} khitomer-accords.txt

`,
	}
	cpCmd = cli.Command{
		Name:   "cp",
		Usage:  "Copy objects and files",
		Action: runCopyCmd,
		CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Copy an object from Amazon S3 object storage to local fileystem.
      $ mc {{.Name}} https://s3.amazonaws.com/jukebox/klingon_opera_aktuh_maylotah.ogg wakeup.ogg

   2. Copy a bucket recursively from Minio object storage to Amazon S3 object storage
      $ mc {{.Name}} http://localhost:9000/photos/burningman2011... https://s3.amazonaws.com/burningman/

   3. Copy a local folder recursively to Minio object storage and Amazon S3 object storage
      $ mc {{.Name}} backup/... http://localhost:9000/archive/ https://s3.amazonaws.com/archive/

   4. Copy an object from Amazon S3 object storage to local filesystem on Windows.
      $ mc {{.Name}} s3:documents/2014/... Documents\backup\2014

`,
	}

	lsCmd = cli.Command{
		Name:   "ls",
		Usage:  "List files and objects",
		Action: runListCmd,
		CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. List objects on Minio object storage
      $ mc {{.Name}} http://localhost:9000/backup/
      2015-03-28 12:47:50 PDT      51.00 MB 2006-Jan-1/backup.tar.gz
      2015-03-31 14:46:33 PDT      55.00 MB 2006-Mar-1/backup.tar.gz

   2. List buckets on Amazon S3 object storage
      $ mc {{.Name}} https://s3.amazonaws.com/
      2015-01-20 15:42:00 PST               rom
      2015-01-15 00:05:40 PST               zek

   3. List buckets and objects from Minio object storage and Amazon S3 object storage
      $ mc {{.Name}} https://s3.amazonaws.com/ http://localhost:9000/backup/
      2015-01-20 15:42:00 PST               rom
      2015-01-15 00:05:40 PST               zek
      2015-03-28 12:47:50 PDT      51.00 MB 2006-Jan-1/backup.tar.gz
      2015-03-31 14:46:33 PDT      55.00 MB 2006-Mar-1/backup.tar.gz

   4. List objects on local filesystem on Windows
      $ mc {{.Name}} C:\Users\Worf
      2015-03-28 12:47:50 PDT      11.00 MB Martok\Klingon Council Ministers.pdf
      2015-03-31 14:46:33 PDT      15.00 MB Gowron\Khitomer Conference Details.pdf

`,
	}

	mbCmd = cli.Command{
		Name:   "mb",
		Usage:  "Make a bucket",
		Action: runMakeBucketCmd,
		CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Create a bucket on Amazon S3 object storage
      $ mc {{.Name}} https://s3.amazonaws.com/public-document-store

   2. Create a bucket on Minio object storage
      $ mc {{.Name}} http://localhost:9000/mongodb-backup

   3. Create multiple buckets on Amazon S3 object storage and Minio object storage
      $ mc {{.Name}} https://s3.amazonaws.com/public-photo-store https://s3.amazonaws.com/public-store http://localhost:9000/mongodb-backup

`,
	}
	//   Configure minio client configuration.
	//
	//   NOTE: that the configure command only writes values to the config file.
	//   It does not use any configuration values from the environment variables.`,
	configCmd = cli.Command{
		Name:   "config",
		Usage:  "Generate configuration \"" + mustGetMcConfigPath() + "\" file.",
		Action: doConfigCmd,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "alias",
				Usage: "Add URL aliases into config",
			},
		},
		CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} generate {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}
EXAMPLES:
   1. Generate mc config
      $ mc config generate

   2. Add alias URLs
      $ mc config --alias "zek https://s3.amazonaws.com/"

`,
	}
	updateCmd = cli.Command{
		Name:        "update",
		Usage:       "Check for new software updates",
		Description: "",
		Action:      doUpdateCmd,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "yes, y",
				Usage: "Download and update local binary",
			},
		},
	}
)

var options = []cli.Command{
	accessCmd,
	lsCmd,
	catCmd,
	mbCmd,
	cpCmd,
	configCmd,
	updateCmd,
}

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
			Name:  "debug",
			Usage: "Enable HTTP tracing",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "Supress chatty console output",
		},
		cli.IntFlag{
			Name:  "retry",
			Usage: "Number of retry count",
			Value: 5,
		},
	}
)
