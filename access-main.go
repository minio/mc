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
	"encoding/json"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Help message.
var accessCmd = cli.Command{
	Name:   "access",
	Usage:  "Set access permissions.",
	Action: mainAccess,
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} PERMISSION TARGET [TARGET ...] {{if .Description}}

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

// AccessMessage is container for access command on bucket success and failure messages
type AccessMessage struct {
	Status string      `json:"status"`
	Bucket string      `json:"bucket"`
	Perms  bucketPerms `json:"permission"`
}

func (s AccessMessage) String() string {
	if !globalJSONFlag {
		return console.Colorize("Access", "Set access permission ‘"+string(s.Perms)+"’ updated successfully for ‘"+s.Bucket+"’")
	}
	accessJSONBytes, err := json.Marshal(s)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(accessJSONBytes)
}

func checkAccessSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
	}
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
	}
	perms := bucketPerms(ctx.Args().First())
	if !perms.isValidBucketPERM() {
		fatalIf(errDummy().Trace(),
			"Unrecognized permission ‘"+perms.String()+"’. Allowed values are [private, public, readonly].")
	}
	for _, arg := range ctx.Args().Tail() {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
		}
	}
}

func mainAccess(ctx *cli.Context) {
	checkAccessSyntax(ctx)

	console.SetCustomTheme(map[string]*color.Color{
		"Access": color.New(color.FgGreen, color.Bold),
	})

	perms := bucketPerms(ctx.Args().First())

	config := mustGetMcConfig()
	for _, arg := range ctx.Args().Tail() {
		targetURL, err := getCanonicalizedURL(arg, config.Aliases)
		fatalIf(err.Trace(arg), "Unable to parse argument ‘"+arg+"’.")

		fatalIf(doUpdateAccessCmd(targetURL, perms).Trace(targetURL, string(perms)), "Unable to set access permission ‘"+string(perms)+"’ for ‘"+targetURL+"’.")

		console.Println(AccessMessage{
			Status: "success",
			Bucket: targetURL,
			Perms:  perms,
		})
	}
}

func doUpdateAccessCmd(targetURL string, targetPERMS bucketPerms) *probe.Error {
	var clnt client.Client
	clnt, err := target2Client(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	if err = clnt.SetBucketACL(targetPERMS.String()); err != nil {
		return err.Trace(targetURL, targetPERMS.String())
	}
	return nil
}
