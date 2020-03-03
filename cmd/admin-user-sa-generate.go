/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"io/ioutil"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminUserSAGenerateCmd = cli.Command{
	Name:   "generate",
	Usage:  "generate a new service account",
	Action: mainAdminUserSAGenerate,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET PARENT-USER [POLICY_FILE]

PARENT-USER:
  The parent user.

POLICY_FILE:
  The path of the policy to apply for the new service account.
  When unspecified, the policy of the parent user will be evaluated
  instead for all type of service accounts requests.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Add a new service account under the name of 'foobar' to MinIO server.
     {{.Prompt}} {{.HelpName}} myminio foobar /tmp/policy.json
`,
}

func checkAdminUserSAGenerateSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || len(ctx.Args()) > 3 {
		cli.ShowCommandHelpAndExit(ctx, "generate", 1) // last argument is exit code
	}
}

// saMessage container for content message structure
type saMessage struct {
	AccessKey    string `json:"accessKey,omitempty"`
	SecretKey    string `json:"secretKey,omitempty"`
	SessionToken string `json:"sessionToken,omitempty"`
}

func (u saMessage) String() string {
	dot := console.Colorize("SA", "‣ ")
	msg := dot + console.Colorize("SA", "Access Key: ") + console.Colorize("AccessKey", u.AccessKey) + "\n"
	msg += dot + console.Colorize("SA", "Secret Key: ") + console.Colorize("SecretKey", u.SecretKey) + "\n"
	msg += dot + console.Colorize("SA", "Session Token: ") + console.Colorize("SessionToken", u.SessionToken)

	return msg
}

func (u saMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func mainAdminUserSAGenerate(ctx *cli.Context) error {
	setSACommandColors()
	checkAdminUserSAGenerateSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	parentUser := args.Get(1)
	policyDocPath := args.Get(2)

	var policyDoc []byte
	var e error

	if len(policyDocPath) > 0 {
		policyDoc, e = ioutil.ReadFile(policyDocPath)
		fatalIf(probe.NewError(e).Trace(args...), "Cannot load the policy document")
	}

	creds, e := client.AddServiceAccount(globalContext, parentUser, string(policyDoc))
	fatalIf(probe.NewError(e).Trace(args...), "Cannot add new service account")

	printMsg(saMessage{
		AccessKey:    creds.AccessKey,
		SecretKey:    creds.SecretKey,
		SessionToken: creds.SessionToken,
	})

	return nil
}
