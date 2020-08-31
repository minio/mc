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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var encryptSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set encryption config",
	Action: mainEncryptSet,
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
  1. Enable SSE-S3 auto encryption on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} sse-s3 myminio/mybucket

  2. Enable SSE-KMS auto encryption with kms key on bucket "mybucket" for alias "s3".
     {{.Prompt}} {{.HelpName}} sse-kms arn:aws:kms:us-east-1:xxx:key/xxx s3/mybucket  
`,
}

// checkEncryptSetSyntax - validate all the passed arguments
func checkEncryptSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || len(ctx.Args()) > 3 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
}

type encryptSetMessage struct {
	Op         string `json:"op"`
	Status     string `json:"status"`
	URL        string `json:"url"`
	Encryption struct {
		Algorithm string `json:"algorithm,omitempty"`
		KeyID     string `json:"keyId,omitempty"`
	} `json:"encryption,omitempty"`
}

func (v encryptSetMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v encryptSetMessage) String() string {
	return console.Colorize("encryptSetMessage", fmt.Sprintf("Auto encryption configuration has been set successfully for %s", v.URL))
}

func mainEncryptSet(cliCtx *cli.Context) error {
	ctx, cancelencryptSet := context.WithCancel(globalContext)
	defer cancelencryptSet()

	console.SetColor("encryptSetMessage", color.New(color.FgGreen))

	checkEncryptSetSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(len(args) - 1)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	var algorithm, keyID string
	switch len(args) {
	case 3:
		algorithm = strings.ToLower(args[0])
		keyID = args[1]
	case 2:
		algorithm = strings.ToLower(args[0])
	}
	if algorithm != "sse-s3" && algorithm != "sse-kms" {
		fatalIf(probe.NewError(fmt.Errorf("Unknown argument `%s` passed", algorithm)), "Invalid encryption algorithm")
	}
	fatalIf(client.SetEncryption(ctx, algorithm, keyID), "Unable to enable auto encryption")
	msg := encryptSetMessage{
		Op:     "set",
		Status: "success",
		URL:    aliasedURL,
	}
	msg.Encryption.Algorithm = algorithm
	printMsg(msg)
	return nil
}
