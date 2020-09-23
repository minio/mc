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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var adminKMSKeyStatusCmd = cli.Command{
	Name:   "status",
	Usage:  "request status information for a KMS master key",
	Action: mainAdminKMSKeyStatus,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [KEY_NAME]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get default master key and its status from a MinIO server/cluster.
     $ {{.HelpName}} play
  2. Get the status of one particular master key from a MinIO server/cluster.
     $ {{.HelpName}} play my-master-key
`,
}

// adminKMSKeyCmd is the handle for the "mc admin kms key" command.
func mainAdminKMSKeyStatus(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "status", 1) // last argument is exit code
	}

	console.SetColor("StatusSuccess", color.New(color.FgGreen, color.Bold))
	console.SetColor("StatusError", color.New(color.FgRed, color.Bold))

	client, err := newAdminClient(ctx.Args().Get(0))
	fatalIf(err, "Unable to get a configured admin connection.")

	var keyID string
	if len(ctx.Args()) == 2 {
		keyID = ctx.Args().Get(1)
	}
	status, e := client.GetKeyStatus(globalContext, keyID)
	fatalIf(probe.NewError(e), "Failed to get status information")

	printMsg(kmsKeyStatusMsg{
		KeyID:         status.KeyID,
		Encryption:    status.EncryptionErr == "",
		Decryption:    status.DecryptionErr == "",
		EncryptionErr: status.EncryptionErr,
		DecryptionErr: status.DecryptionErr,
	})
	return nil
}

type kmsKeyStatusMsg struct {
	KeyID         string `json:"keyId"`
	Encryption    bool   `json:"encryption"`
	Decryption    bool   `json:"decryption"`
	EncryptionErr string `json:"encryptionError,omitempty"`
	DecryptionErr string `json:"decryptionError,omitempty"`
	Status        string `json:"status"`
}

func (s kmsKeyStatusMsg) JSON() string {
	s.Status = "success"
	if !s.Encryption && !s.Decryption {
		s.Status = "error"
	}
	kmsBytes, e := json.MarshalIndent(s, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(kmsBytes)
}

func (s kmsKeyStatusMsg) String() string {
	msg := fmt.Sprintf("Key: %s\n", s.KeyID)
	if s.Encryption {
		msg += "   - Encryption " + console.Colorize("StatusSuccess", "✔") + "\n"
	} else {
		msg += fmt.Sprintf("   - Encryption %s (%s)\n", console.Colorize("StatusError", "✗"), s.EncryptionErr)
	}

	if s.Decryption {
		msg += "   - Decryption " + console.Colorize("StatusSuccess", "✔") + "\n"
	} else {
		msg += fmt.Sprintf("   - Decryption %s (%s)\n", console.Colorize("StatusError", "✗"), s.DecryptionErr)
	}
	return msg
}
