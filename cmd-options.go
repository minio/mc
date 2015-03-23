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

	"github.com/codegangsta/cli"
)

var cpCmd = cli.Command{
	Name:        "cp",
	Usage:       "copy objects",
	Description: `Copies a local file or dir or object or bucket to another location locally or in S3.`,
	Action:      doCopyCmd,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "recursively crawls given directory uploads to given bucket",
		},
	},
}

var lsCmd = cli.Command{
	Name:        "ls",
	Usage:       "get list of objects",
	Description: `List Objects and common prefixes under a prefix or all Buckets`,
	Action:      doListCmd,
}

var mbCmd = cli.Command{
	Name:        "mb",
	Usage:       "makes a bucket",
	Description: "Creates an S3 bucket",
	Action:      doMakeBucketCmd,
}

var configCmd = cli.Command{
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

var donutCmd = cli.Command{
	Name:        "donut",
	Usage:       "donut admin",
	Description: "",
	Subcommands: donutOptions,
}

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
	donutCmd,
	configCmd,
}

var flags = []cli.Flag{
	cli.BoolFlag{
		Name:  "debug",
		Usage: "enable HTTP tracing",
	},
	cli.BoolFlag{
		Name:  "quiet, q",
		Usage: "disable chatty output, such as the progress bar",
	},
}

var mcBashCompletion = `#!/bin/bash

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
