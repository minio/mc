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
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var encryptInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "show bucket encryption status",
	Action: mainEncryptInfo,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display bucket encryption status for bucket "mybucket".
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkversionInfoSyntax - validate all the passed arguments
func checkEncryptInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
}

type encryptInfoMessage struct {
	Op         string `json:"op"`
	Status     string `json:"status"`
	URL        string `json:"url"`
	Encryption struct {
		Algorithm string `json:"algorithm,omitempty"`
		KeyID     string `json:"keyId,omitempty"`
	} `json:"encryption,omitempty"`
}

func (v encryptInfoMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v encryptInfoMessage) String() string {
	msg := ""
	switch v.Encryption.Algorithm {
	case "":
		msg = fmt.Sprintf("Auto encryption is not enabled for %s ", v.URL)
	default:
		msg = "Auto encryption 'sse-s3' is enabled"
	}
	if v.Encryption.KeyID != "" {
		msg = fmt.Sprintf("Auto encrytion 'sse-kms' is enabled with KeyID: %s", v.Encryption.KeyID)
	}
	return console.Colorize("encryptInfoMessage", msg)
}

func mainEncryptInfo(cliCtx *cli.Context) error {
	ctx, cancelEncryptInfo := context.WithCancel(globalContext)
	defer cancelEncryptInfo()

	console.SetColor("encryptInfoMessage", color.New(color.FgGreen))

	checkEncryptInfoSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	algorithm, keyID, e := client.GetEncryption(ctx)
	fatalIf(e, "Unable to get encryption info")
	msg := encryptInfoMessage{
		Op:     cliCtx.Command.Name,
		Status: "success",
		URL:    aliasedURL,
	}
	msg.Encryption.Algorithm = algorithm
	msg.Encryption.KeyID = keyID
	printMsg(msg)
	return nil
}
