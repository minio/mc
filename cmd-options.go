/*
 * Minimalist Object Storage, (C) 2014,2015 Minio, Inc.
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
	"net/url"

	"github.com/minio-io/cli"
)

// List of commands
var (
	cpCmd = cli.Command{
		Name:        "cp",
		Usage:       "copy objects and files",
		Description: "Copy files and objects recursively between Amazon S3, Donut and Filesystem",
		Action:      doCopyCmd,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "recursive, r",
				Usage: "recursively crawls a given directory or bucket",
			},
		},
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

   2. Copy a bucket recursive from Donut to Amazon S3 object storage
      $ mc {{.Name}} --recursive donut://home/photos/burningman2011 https://s3.amazonaws.com/burningman/

   3. Copy a local folder to Donut and Amazon S3 object storage
      $ mc {{.Name}} --recursive backup/ donut://archive/backup/ https://s3.amazonaws.com/backup/

`,
	}

	lsCmd = cli.Command{
		Name:        "ls",
		Usage:       "list files and objects",
		Description: `List files and objects recursively between Amazon S3, Donut and Filesystem`,
		Action:      doListCmd,
		CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. List objects on Donut storage
      $ mc {{.Name}} donut://archive/backup/
      2015-03-28 12:47:50 PDT      51.00 MB 2006-Jan-1/backup.tar.gz
      2015-03-31 14:46:33 PDT      55.00 MB 2006-Mar-1/backup.tar.gz

   2. List buckets on Amazon S3 object storage
      $ mc {{.Name}} https://s3.amazonaws.com/
      2015-01-20 15:42:00 PST               rom
      2015-01-15 00:05:40 PST               zek

`,
	}

	mbCmd = cli.Command{
		Name:        "mb",
		Usage:       "makes a bucket",
		Description: "Creates an S3 bucket",
		Action:      doMakeBucketCmd,
	}

	configCmd = cli.Command{
		Name:  "config",
		Usage: "Generate configuration \"" + getMcConfigFilename() + "\" file.",
		Description: `Configure minio client configuration data. If your config
   file does not exist (the default location is ~/.auth), it will be
   automatically created for you. Note that the configure command only writes
   values to the config file. It does not use any configuration values from
   the environment variables.`,
		Action: doConfigCmd,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "accesskeyid, a",
				Value: "",
				Usage: "AWS access key ID",
			},
			cli.StringFlag{
				Name:  "secretkey, s",
				Value: "",
				Usage: "AWS secret access key",
			},
			cli.StringFlag{
				Name:  "alias",
				Value: "",
				Usage: "Add aliases into config",
			},
			cli.BoolFlag{
				Name:  "completion",
				Usage: "Generate bash completion \"" + getMcBashCompletionFilename() + "\" file.",
			},
		},
	}
)

type object struct {
	scheme string // protocol type: possible values are http, https, donut, nil
	host   string
	// Bucket name can also be a DNS name. "." is allowed, with certain restrictions.
	// Read more at http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
	bucket string
	key    string
	url    *url.URL
}

type cmdArgs struct {
	quiet       bool
	source      object
	destination object
}

var options = []cli.Command{
	cpCmd,
	lsCmd,
	mbCmd,
	configCmd,
}

var (
	flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable HTTP tracing",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "disable chatty output, such as the progress bar",
		},
	}
)

var (
	mcBashCompletion = `#!/bin/bash

_mc_completion() {
    local cur prev opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _mc_completion mc
`
)
