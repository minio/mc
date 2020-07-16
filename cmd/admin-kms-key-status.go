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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
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

	client, err := newAdminClient(ctx.Args().Get(0))
	fatalIf(err, "Cannot get a configured admin connection.")

	var keyID string
	if len(ctx.Args()) == 2 {
		keyID = ctx.Args().Get(1)
	}
	status, e := client.GetKeyStatus(globalContext, keyID)
	fatalIf(probe.NewError(e), "Failed to get status information")

	printMsg(kmsKeyStatusMsg(*status))
	return nil
}

type kmsKeyStatusMsg madmin.KMSKeyStatus

func (s kmsKeyStatusMsg) String() string {
	msg := fmt.Sprintf("Key: %s\n", s.KeyID)
	if s.EncryptionErr == "" {
		msg = fmt.Sprintf("%s %s", msg, "\t • Encryption ✔\n")
	} else {
		return fmt.Sprintf("%s \t • Encryption failed: %s\n", msg, s.EncryptionErr)
	}

	if s.DecryptionErr == "" {
		msg = fmt.Sprintf("%s %s", msg, "\t • Decryption ✔\n")
	} else {
		return fmt.Sprintf("%s \t • Decryption failed: %s\n", msg, s.DecryptionErr)
	}
	return msg
}

func (s kmsKeyStatusMsg) JSON() string {
	const fmtStr = `{"key-id":"%s","encryption-error":"%s","decryption-error":"%s"}`
	return fmt.Sprintf(fmtStr, s.KeyID, s.EncryptionErr, s.DecryptionErr)
}
