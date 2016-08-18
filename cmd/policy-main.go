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

package cmd

import (
	"encoding/json"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var (
	policyFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of policy.",
		},
	}
)

// Set public policy
var policyCmd = cli.Command{
	Name:   "policy",
	Usage:  "Set public policy on bucket or prefix.",
	Action: mainPolicy,
	Flags:  append(policyFlags, globalFlags...),
	CustomHelpTemplate: `Name:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] PERMISSION TARGET
   mc {{.Name}} [FLAGS] TARGET

PERMISSION:
   Allowed policies are: [none, download, upload, both].

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Set bucket to "download" on Amazon S3 cloud storage.
      $ mc {{.Name}} download s3/burningman2011

   2. Set bucket to "both" on Amazon S3 cloud storage.
      $ mc {{.Name}} both s3/shared

   3. Set bucket to "upload" on Amazon S3 cloud storage.
      $ mc {{.Name}} upload s3/incoming

   4. Set a prefix to "both" on Amazon S3 cloud storage.
      $ mc {{.Name}} both s3/public-commons/images

   5. Get bucket permissions.
      $ mc {{.Name}} s3/shared

`,
}

// policyMessage is container for policy command on bucket success and failure messages.
type policyMessage struct {
	Operation string      `json:"operation"`
	Status    string      `json:"status"`
	Bucket    string      `json:"bucket"`
	Perms     accessPerms `json:"permission"`
}

// String colorized access message.
func (s policyMessage) String() string {
	if s.Operation == "set" {
		return console.Colorize("Policy",
			"Access permission for ‘"+s.Bucket+"’ is set to ‘"+string(s.Perms)+"’")
	}
	if s.Operation == "get" {
		return console.Colorize("Policy",
			"Access permission for ‘"+s.Bucket+"’"+" is ‘"+string(s.Perms)+"’")
	}
	// nothing to print
	return ""
}

// JSON jsonified policy message.
func (s policyMessage) JSON() string {
	policyJSONBytes, e := json.Marshal(s)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(policyJSONBytes)
}

// checkPolicySyntax check for incoming syntax.
func checkPolicySyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1) // last argument is exit code.
	}
	if len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1) // last argument is exit code.
	}
	if len(ctx.Args()) == 2 {
		perms := accessPerms(ctx.Args().Get(0))
		if !perms.isValidAccessPERM() {
			fatalIf(errDummy().Trace(),
				"Unrecognized permission ‘"+string(perms)+"’. Allowed values are [none, download, upload, both].")
		}
	}
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1) // last argument is exit code.
	}
}

// doSetAccess do set access.
func doSetAccess(targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	policy := ""
	switch targetPERMS {
	case accessNone:
		policy = "none"
	case accessDownload:
		policy = "readonly"
	case accessUpload:
		policy = "writeonly"
	case accessBoth:
		policy = "readwrite"
	}
	if err = clnt.SetAccess(policy); err != nil {
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
	var policy accessPerms
	switch perm {
	case "none":
		policy = accessNone
	case "readonly":
		policy = accessDownload
	case "writeonly":
		policy = accessUpload
	case "readwrite":
		policy = accessBoth
	}

	return policy, nil
}

func mainPolicy(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'policy' cli arguments.
	checkPolicySyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Policy", color.New(color.FgGreen, color.Bold))

	perms := accessPerms(ctx.Args().First())
	if perms.isValidAccessPERM() {
		targetURL := ctx.Args().Last()
		err := doSetAccess(targetURL, perms)
		// Upon error exit.
		fatalIf(err.Trace(targetURL, string(perms)),
			"Unable to set policy ‘"+string(perms)+"’ for ‘"+targetURL+"’.")
		printMsg(policyMessage{
			Status:    "success",
			Operation: "set",
			Bucket:    targetURL,
			Perms:     perms,
		})
	} else {
		targetURL := ctx.Args().First()
		perms, err := doGetAccess(targetURL)
		fatalIf(err.Trace(targetURL), "Unable to get policy for ‘"+targetURL+"’.")
		printMsg(policyMessage{
			Status:    "success",
			Operation: "get",
			Bucket:    targetURL,
			Perms:     perms,
		})
	}
}
