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
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var (
	policyFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "List recursively.",
		},
	}
)

// Set public policy
var policyCmd = cli.Command{
	Name:   "policy",
	Usage:  "Manage anonymous access to objects.",
	Action: mainPolicy,
	Before: setGlobalsFromContext,
	Flags:  append(policyFlags, globalFlags...),
	CustomHelpTemplate: `Name:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] PERMISSION TARGET
  {{.HelpName}} [FLAGS] TARGET
  {{.HelpName}} list [FLAGS] TARGET

PERMISSION:
  Allowed policies are: [none, download, upload, public].
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
EXAMPLES:
   1. Set bucket to "download" on Amazon S3 cloud storage.
      $ {{.HelpName}} download s3/burningman2011

   2. Set bucket to "public" on Amazon S3 cloud storage.
      $ {{.HelpName}} public s3/shared

   3. Set bucket to "upload" on Amazon S3 cloud storage.
      $ {{.HelpName}} upload s3/incoming

   4. Set a prefix to "public" on Amazon S3 cloud storage.
      $ {{.HelpName}} public s3/public-commons/images

   5. Get bucket permissions.
      $ {{.HelpName}} s3/shared

   6. List policies set to a specified bucket.
      $ {{.HelpName}} list s3/shared

   7. List public object URLs recursively.
      $ {{.HelpName}} --recursive links s3/shared/

`,
}

// policyRules contains policy rule
type policyRules struct {
	Resource string `json:"resource"`
	Allow    string `json:"allow"`
}

// String colorized access message.
func (s policyRules) String() string {
	return console.Colorize("Policy", s.Resource+" => "+s.Allow+"")
}

// JSON jsonified policy message.
func (s policyRules) JSON() string {
	policyJSONBytes, e := json.Marshal(s)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(policyJSONBytes)
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
			"Access permission for `"+s.Bucket+"` is set to `"+string(s.Perms)+"`")
	}
	if s.Operation == "get" {
		return console.Colorize("Policy",
			"Access permission for `"+s.Bucket+"`"+" is `"+string(s.Perms)+"`")
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

// policyLinksMessage is container for policy links command
type policyLinksMessage struct {
	Status string `json:"status"`
	URL    string `json:"url"`
}

// String colorized access message.
func (s policyLinksMessage) String() string {
	return console.Colorize("Policy", string(s.URL))
}

// JSON jsonified policy message.
func (s policyLinksMessage) JSON() string {
	policyJSONBytes, e := json.Marshal(s)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(policyJSONBytes)
}

// checkPolicySyntax check for incoming syntax.
func checkPolicySyntax(ctx *cli.Context) {
	argsLength := len(ctx.Args())
	// Always print a help message when we have extra arguments
	if argsLength > 2 {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1) // last argument is exit code.
	}
	// Always print a help message when no arguments specified
	if argsLength < 1 {
		cli.ShowCommandHelpAndExit(ctx, "policy", 1)
	}

	firstArg := ctx.Args().Get(0)

	// More syntax checking
	switch accessPerms(firstArg) {
	case accessNone, accessDownload, accessUpload, accessPublic:
		// Always expect two arguments when a policy permission is provided
		if argsLength != 2 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}
	case "list":
		// Always expect an argument after list cmd
		if argsLength != 2 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}
	case "links":
		// Always expect an argument after links cmd
		if argsLength != 2 {
			cli.ShowCommandHelpAndExit(ctx, "policy", 1)
		}

	default:
		if argsLength == 2 {
			fatalIf(errDummy().Trace(),
				"Unrecognized permission `"+string(firstArg)+"`. Allowed values are [none, download, upload, public].")
		}
	}
}

// Convert an accessPerms to a string recognizable by minio-go
func accessPermToString(perm accessPerms) string {
	policy := ""
	switch perm {
	case accessNone:
		policy = "none"
	case accessDownload:
		policy = "readonly"
	case accessUpload:
		policy = "writeonly"
	case accessPublic:
		policy = "readwrite"
	}
	return policy
}

// doSetAccess do set access.
func doSetAccess(targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	policy := accessPermToString(targetPERMS)
	if err = clnt.SetAccess(policy); err != nil {
		return err.Trace(targetURL, string(targetPERMS))
	}
	return nil
}

// Convert a minio-go permission to accessPerms type
func stringToAccessPerm(perm string) accessPerms {
	var policy accessPerms
	switch perm {
	case "none":
		policy = accessNone
	case "readonly":
		policy = accessDownload
	case "writeonly":
		policy = accessUpload
	case "readwrite":
		policy = accessPublic
	}
	return policy
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
	return stringToAccessPerm(perm), nil
}

// doGetAccessRules do get access rules.
func doGetAccessRules(targetURL string) (r map[string]string, err *probe.Error) {
	clnt, err := newClient(targetURL)
	if err != nil {
		return map[string]string{}, err.Trace(targetURL)
	}
	return clnt.GetAccessRules()
}

// Run policy list command
func runPolicyListCmd(ctx *cli.Context) {
	targetURL := ctx.Args().Last()
	policies, err := doGetAccessRules(targetURL)
	if err != nil {
		switch err.ToGoError().(type) {
		case APINotImplemented:
			fatalIf(err.Trace(), "Unable to list policies of a non S3 url `"+targetURL+"`.")
		default:
			fatalIf(err.Trace(targetURL), "Unable to list policies of target `"+targetURL+"`.")
		}
	}
	for k, v := range policies {
		printMsg(policyRules{Resource: k, Allow: v})
	}
}

// Run policy links command
func runPolicyLinksCmd(ctx *cli.Context) {
	// Get alias/bucket/prefix argument
	targetURL := ctx.Args().Last()

	// Fetch all policies associated to the passed url
	policies, err := doGetAccessRules(targetURL)
	if err != nil {
		switch err.ToGoError().(type) {
		case APINotImplemented:
			fatalIf(err.Trace(), "Unable to list policies of a non S3 url `"+targetURL+"`.")
		default:
			fatalIf(err.Trace(targetURL), "Unable to list policies of target `"+targetURL+"`.")
		}
	}

	// Extract alias from the passed argument, we'll need it to
	// construct new pathes to list public objects
	alias, path := url2Alias(targetURL)

	isRecursive := ctx.Bool("recursive")
	isIncomplete := false

	// Iterate over policy rules to fetch public urls, then search
	// for objects under those urls
	for k, v := range policies {
		// Trim the asterisk in policy rules
		policyPath := strings.TrimSuffix(k, "*")
		// Check if current policy prefix is related to the url passed by the user
		if !strings.HasPrefix(policyPath, path) {
			continue
		}
		// Check if the found policy has read permission
		perm := stringToAccessPerm(v)
		if perm != accessDownload && perm != accessPublic {
			continue
		}
		// Construct the new path to search for public objects
		newURL := alias + "/" + policyPath
		clnt, err := newClient(newURL)
		fatalIf(err.Trace(newURL), "Unable to initialize target `"+targetURL+"`.")
		// Search for public objects
		for content := range clnt.List(isRecursive, isIncomplete, DirFirst) {
			if content.Err != nil {
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			}

			if content.Type.IsDir() && isRecursive {
				continue
			}

			// Encode public URL
			u, e := url.Parse(content.URL.String())
			errorIf(probe.NewError(e), "Unable to parse url `"+content.URL.String()+"`.")
			publicURL := u.String()

			// Construct the message to be displayed to the user
			msg := policyLinksMessage{
				Status: "success",
				URL:    publicURL,
			}
			// Print the found object
			printMsg(msg)
		}
	}
}

// Run policy cmd to fetch set permission
func runPolicyCmd(ctx *cli.Context) {
	perms := accessPerms(ctx.Args().First())
	if perms.isValidAccessPERM() {
		targetURL := ctx.Args().Last()
		err := doSetAccess(targetURL, perms)
		// Upon error exit.
		if err != nil {
			switch err.ToGoError().(type) {
			case APINotImplemented:
				fatalIf(err.Trace(), "Unable to set policy of a non S3 url `"+targetURL+"`.")
			default:
				fatalIf(err.Trace(targetURL, string(perms)),
					"Unable to set policy `"+string(perms)+"` for `"+targetURL+"`.")

			}
		}

		printMsg(policyMessage{
			Status:    "success",
			Operation: "set",
			Bucket:    targetURL,
			Perms:     perms,
		})
	} else {
		targetURL := ctx.Args().First()
		perms, err := doGetAccess(targetURL)
		if err != nil {
			switch err.ToGoError().(type) {
			case APINotImplemented:
				fatalIf(err.Trace(), "Unable to get policy of a non S3 url `"+targetURL+"`.")
			default:
				fatalIf(err.Trace(targetURL), "Unable to get policy for `"+targetURL+"`.")
			}
		}

		printMsg(policyMessage{
			Status:    "success",
			Operation: "get",
			Bucket:    targetURL,
			Perms:     perms,
		})
	}
}

func mainPolicy(ctx *cli.Context) error {

	// check 'policy' cli arguments.
	checkPolicySyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Policy", color.New(color.FgGreen, color.Bold))

	switch ctx.Args().First() {
	case "list":
		// policy list alias/bucket/prefix
		runPolicyListCmd(ctx)
	case "links":
		// policy links alias/bucket/prefix
		runPolicyLinksCmd(ctx)
	default:
		// policy [download|upload|public|] alias/bucket/prefix
		runPolicyCmd(ctx)
	}
	return nil
}
