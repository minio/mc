/*
 * Minio Client, (C) 2015 Minio, Inc.
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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
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

   1. Set bucket to "private" on Amazon S3 cloud storage.
      $ mc {{.Name}} private https://s3.amazonaws.com/burningman2011

   2. Set bucket to "public" on Amazon S3 cloud storage.
      $ mc {{.Name}} public https://s3.amazonaws.com/shared

   3. Set bucket to "authenticated" on Amazon S3 cloud storage to provide read access to IAM Authenticated Users group.
      $ mc {{.Name}} authenticated https://s3.amazonaws.com/shared-authenticated

   4. Set folder to world readwrite (chmod 777) on local filesystem.
      $ mc {{.Name}} public /shared/Music

`,
}

func runAccessCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
	}
	config := mustGetMcConfig()
	acl := bucketACL(ctx.Args().First())
	if !acl.isValidBucketACL() {
		console.Fatalf("Valid types are [private, public, readonly]. %s\n", errInvalidACL{acl: acl.String()})
	}
	for _, arg := range ctx.Args().Tail() {
		targetURL, err := getExpandedURL(arg, config.Aliases)
		if err != nil {
			switch e := iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				console.Fatalf("Unknown type of URL %s. %s\n", e.url, err)
			default:
				console.Fatalf("Unable to parse argument %s. %s\n", arg, err)
			}
		}
		msg, err := doUpdateAccessCmd(targetURL, acl)
		if err != nil {
			console.Fatalln(msg)
		}
		console.Infoln(msg)
	}
}

func doUpdateAccessCmd(targetURL string, targetACL bucketACL) (string, error) {
	var err error
	var clnt client.Client
	clnt, err = target2Client(targetURL)
	if err != nil {
		msg := fmt.Sprintf("Unable to initialize client for ‘%s’", targetURL)
		return msg, NewIodine(iodine.New(err, nil))
	}
	return doUpdateAccess(clnt, targetACL)
}

func doUpdateAccess(clnt client.Client, targetACL bucketACL) (string, error) {
	err := clnt.SetBucketACL(targetACL.String())
	if err != nil {
		msg := fmt.Sprintf("Failed to add bucket access policy for URL ‘%s’", clnt.URL().String())
		return msg, NewIodine(iodine.New(err, nil))
	}
	return "Bucket access policy updated successfully : " + clnt.URL().String(), nil
}
