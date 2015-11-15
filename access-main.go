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
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// Set access permissions.
var accessCmd = cli.Command{
	Name:   "access",
	Usage:  "Set or get access permissions.",
	Action: mainAccess,
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} set PERMISSION TARGET [TARGET ...]
   mc {{.Name}} get TARGET [TARGET ...]

   PERMISSION = private | readonly | public | authorized {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}

EXAMPLES:
   1. Set bucket to "private" on Amazon S3 cloud storage.
      $ mc {{.Name}} set private https://s3.amazonaws.com/burningman2011

   2. Set bucket to "public" on Amazon S3 cloud storage.
      $ mc {{.Name}} set public https://s3.amazonaws.com/shared

   3. Set bucket to "authenticated" on Amazon S3 cloud storage to provide read access to IAM Authenticated Users group.
      $ mc {{.Name}} set authorized https://s3.amazonaws.com/shared-authenticated

   4. Get bucket permissions.
      $ mc {{.Name}} get https://s3.amazonaws.com/shared

   5. Get bucket permissions.
      $ mc {{.Name}} get https://storage.googleapis.com/miniocloud
`,
}

// accessMessage is container for access command on bucket success and failure messages
type accessMessage struct {
	Operation string      `json:"operation"`
	Status    string      `json:"status"`
	Bucket    string      `json:"bucket"`
	Perms     accessPerms `json:"permission"`
}

// String colorized access message
func (s accessMessage) String() string {
	if s.Operation == "set" {
		return console.Colorize("Access", "Set access permission ‘"+string(s.Perms)+"’ updated successfully for ‘"+s.Bucket+"’")
	}
	if s.Operation == "get" {
		return console.Colorize("Access", "Access permission for ‘"+s.Bucket+"’"+" is ‘"+string(s.Perms)+"’")
	}
	// nothing to print
	return ""
}

// JSON jsonified access message
func (s accessMessage) JSON() string {
	accessJSONBytes, err := json.Marshal(s)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(accessJSONBytes)
}

func checkAccessSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
	}
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
	}
	switch ctx.Args().First() {
	case "set":
		if len(ctx.Args().Tail()) < 2 {
			cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
		}
		perms := accessPerms(ctx.Args().Tail().Get(0))
		if !perms.isValidAccessPERM() {
			fatalIf(errDummy().Trace(),
				"Unrecognized permission ‘"+perms.String()+"’. Allowed values are [private, public, readonly].")
		}
		for _, arg := range ctx.Args().Tail().Tail() {
			if strings.TrimSpace(arg) == "" {
				fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
			}
		}
	case "get":
		if len(ctx.Args().Tail()) < 1 {
			cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
		}
	}
}

func doSetAccess(targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := url2Client(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	if err = clnt.SetBucketAccess(targetPERMS.String()); err != nil {
		return err.Trace(targetURL, targetPERMS.String())
	}
	return nil
}

func doGetAccess(targetURL string) (perms accessPerms, err *probe.Error) {
	clnt, err := url2Client(targetURL)
	if err != nil {
		return "", err.Trace(targetURL)
	}
	acl, err := clnt.GetBucketAccess()
	if err != nil {
		return "", err.Trace(targetURL)
	}
	return aclToPerms(acl), nil
}

func mainAccess(ctx *cli.Context) {
	checkAccessSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Access", color.New(color.FgGreen, color.Bold))
	config := mustGetMcConfig()

	switch ctx.Args().Get(0) {
	case "set":
		perms := accessPerms(ctx.Args().Tail().Get(0))
		for _, arg := range ctx.Args().Tail().Tail() {
			targetURL := getAliasURL(arg, config.Aliases)

			fatalIf(doSetAccess(targetURL, perms).Trace(targetURL, string(perms)), "Unable to set access permission ‘"+string(perms)+"’ for ‘"+targetURL+"’.")

			printMsg(accessMessage{
				Operation: "set",
				Status:    "success",
				Bucket:    targetURL,
				Perms:     perms,
			})
		}
	case "get":
		for _, arg := range ctx.Args().Tail() {
			targetURL := getAliasURL(arg, config.Aliases)
			perms, err := doGetAccess(targetURL)
			fatalIf(err.Trace(targetURL), "Unable to get access permission for ‘"+targetURL+"’.")

			printMsg(accessMessage{
				Operation: "get",
				Status:    "success",
				Bucket:    targetURL,
				Perms:     perms,
			})
		}
	}
}
