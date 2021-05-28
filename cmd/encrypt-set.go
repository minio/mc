// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var encryptSetCmd = cli.Command{
	Name:         "set",
	Usage:        "set encryption config",
	Action:       mainEncryptSet,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
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
