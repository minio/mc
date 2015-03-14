/*
 * Mini Object Storage, (C) 2014,2015 Minio, Inc.
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
	"github.com/codegangsta/cli"
)

var cp = cli.Command{
	Name:        "cp",
	Usage:       "copy objects",
	Description: `Copies a local file or dir or object or bucket to another location locally or in S3.`,
	Action:      doFsCopy,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "recursively crawls given directory uploads to given bucket",
		},
	},
}

var ls = cli.Command{
	Name:        "ls",
	Usage:       "get list of objects",
	Description: `List Objects and common prefixes under a prefix or all Buckets`,
	Action:      doFsList,
}

var mb = cli.Command{
	Name:        "mb",
	Usage:       "makes a bucket",
	Description: "Creates an S3 bucket",
	Action:      doFsMb,
}

var configure = cli.Command{
	Name:  "config",
	Usage: "Generate configuration \"" + getMcConfigFilename() + "\" file.",
	Description: `Configure minio client configuration data. If your config
   file does not exist (the default location is ~/.auth), it will be
   automatically created for you. Note that the configure command only writes
   values to the config file. It does not use any configuration values from
   the environment variables.`,
	Action: doConfigure,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "accesskey, a",
			Value: "",
			Usage: "AWS access key id",
		},
		cli.StringFlag{
			Name:  "secretkey, s",
			Value: "",
			Usage: "AWS secret key id",
		},
	},
}

type object struct {
	bucket string
	key    string
	host   string
}

type cmdArgs struct {
	quiet       bool
	source      object
	destination object
}

var options = []cli.Command{
	cp,
	ls,
	mb,
	configure,
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
	cli.BoolFlag{
		Name:  "get-bash-completion",
		Usage: "Generate bash completion \"" + getMcBashCompletionFilename() + "\" file.",
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
