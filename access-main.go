/*
 * Minio Client, (C) 2015, 2016 Minio, Inc.
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

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	accessFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of access.",
		},
	}
)

// Set public access permissions.
var accessCmd = cli.Command{
	Name:   "access",
	Usage:  "Set public access permissions on bucket or prefix.",
	Action: mainAccess,
	Flags:  append(accessFlags, globalFlags...),
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] PERMISSION TARGET
   mc {{.Name}} [FLAGS] TARGET

PERMISSION:
   Allowed permissions are: [none, readonly, readwrite, writeonly].

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Set bucket to "readonly" on Amazon S3 cloud storage.
      $ mc {{.Name}} readonly s3/burningman2011

   2. Set bucket to "readwrite" on Amazon S3 cloud storage.
      $ mc {{.Name}} readwrite s3/shared

   3. Set bucket to "writeonly" on Amazon S3 cloud storage.
      $ mc {{.Name}} writeonly s3/incoming

   4. Set a prefix to "readwrite" on Amazon S3 cloud storage.
      $ mc {{.Name}} readwrite s3/public-commons/images

   5. Get bucket permissions.
      $ mc {{.Name}} s3/shared

`,
}

// accessMessage is container for access command on bucket success and failure messages.
type accessMessage struct {
	Operation string      `json:"operation"`
	Status    string      `json:"status"`
	Bucket    string      `json:"bucket"`
	Perms     accessPerms `json:"permission"`
}

// String colorized access message.
func (s accessMessage) String() string {
	if s.Operation == "set" {
		return console.Colorize("Access",
			"Access permission for ‘"+s.Bucket+"’ is set to ‘"+string(s.Perms)+"’")
	}
	if s.Operation == "get" {
		return console.Colorize("Access",
			"Access permission for ‘"+s.Bucket+"’"+" is ‘"+string(s.Perms)+"’")
	}
	// nothing to print
	return ""
}

// JSON jsonified access message.
func (s accessMessage) JSON() string {
	accessJSONBytes, e := json.Marshal(s)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(accessJSONBytes)
}

// checkAccessSyntax check for incoming syntax.
func checkAccessSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code.
	}
	if len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code.
	}
	if len(ctx.Args()) == 2 {
		perms := accessPerms(ctx.Args().Get(0))
		if !perms.isValidAccessPERM() {
			fatalIf(errDummy().Trace(),
				"Unrecognized permission ‘"+string(perms)+"’. Allowed values are [none, readonly, readwrite, writeonly].")
		}
	}
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code.
	}
}

// doSetAccess do set access.
func doSetAccess(targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	if err = clnt.SetAccess(string(targetPERMS)); err != nil {
		return err.Trace(targetURL, string(targetPERMS))
	}
	return nil
}

// doGetAccess do get access.
func doGetAccess(targetURL string) (perms accessPerms, err *probe.Error) {
	clnt, err := newClient(targetURL)
	if err != nil {
		return "", err.Trace(targetURL)
	}
	perm, err := clnt.GetAccess()
	if err != nil {
		return "", err.Trace(targetURL)
	}
	return accessPerms(perm), nil
}

func mainAccess(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'access' cli arguments.
	checkAccessSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Access", color.New(color.FgGreen, color.Bold))

	perms := accessPerms(ctx.Args().First())
	if perms.isValidAccessPERM() {
		targetURL := ctx.Args().Last()
		err := doSetAccess(targetURL, perms)
		// Upon error exit.
		fatalIf(err.Trace(targetURL, string(perms)),
			"Unable to set access permission ‘"+string(perms)+"’ for ‘"+targetURL+"’.")
		printMsg(accessMessage{
			Status:    "success",
			Operation: "set",
			Bucket:    targetURL,
			Perms:     perms,
		})
	} else {
		targetURL := ctx.Args().First()
		perms, err := doGetAccess(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to get access permission for ‘"+targetURL+"’.")
		printMsg(accessMessage{
			Status:    "success",
			Operation: "get",
			Bucket:    targetURL,
			Perms:     perms,
		})
	}
}
