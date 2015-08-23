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
	"errors"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/probe"
)

// Help message.
var accessCmd = cli.Command{
	Name:   "access",
	Usage:  "Set access permissions",
	Action: mainAccess,
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

func mainAccess(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
	}
	config := mustGetMcConfig()
	acl := bucketACL(ctx.Args().First())
	if !acl.isValidBucketACL() {
		fatalIf(probe.NewError(errors.New("")),
			"Unrecognized permission ‘"+acl.String()+"’. Allowed values are [private, public, readonly].")
	}
	for _, arg := range ctx.Args().Tail() {
		targetURL, err := getCanonicalizedURL(arg, config.Aliases)
		fatalIf(err.Trace(arg), "Unable to parse URL argument ‘"+arg+"’.")
		fatalIf(doUpdateAccessCmd(targetURL, acl).Trace(targetURL), "Unable to set access permission ‘"+acl.String()+"’ for URL ‘"+targetURL+"’.")
	}
}

func doUpdateAccessCmd(targetURL string, targetACL bucketACL) *probe.Error {
	var clnt client.Client
	clnt, err := target2Client(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	if err = clnt.SetBucketACL(targetACL.String()); err != nil {
		return err.Trace(targetURL, targetACL.String())
	}
	return nil
}
