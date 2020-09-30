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

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio/pkg/console"
)

var legalHoldCmd = cli.Command{
	Name:   "legalhold",
	Usage:  "manage legal hold for object(s)",
	Action: mainLegalHold,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		legalHoldSetCmd,
		legalHoldClearCmd,
		legalHoldInfoCmd,
	},
}

// Structured message depending on the type of console.
type legalHoldCmdMessage struct {
	LegalHold minio.LegalHoldStatus `json:"legalhold"`
	URLPath   string                `json:"urlpath"`
	Key       string                `json:"key"`
	VersionID string                `json:"versionID"`
	Status    string                `json:"status"`
	Err       error                 `json:"error,omitempty"`
}

// Colorized message for console printing.
func (l legalHoldCmdMessage) String() string {
	if l.Err != nil {
		return console.Colorize("LegalHoldMessageFailure", "Unable to set object legal hold status `"+l.Key+"`. "+l.Err.Error())
	}
	op := "set"
	if l.LegalHold == minio.LegalHoldDisabled {
		op = "cleared"
	}

	msg := fmt.Sprintf("Object legal hold successfully %s for `%s`", op, l.Key)
	if l.VersionID != "" {
		msg += fmt.Sprintf(" (version-id=%s)", l.VersionID)
	}
	msg += "."
	return console.Colorize("LegalHoldSuccess", msg)
}

// JSON'ified message for scripting.
func (l legalHoldCmdMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Check if the bucket corresponding to the target url has
// object locking enabled, this to show a pretty error message
func checkBucketLockSupport(ctx context.Context, aliasedURL string) {
	clnt, err := newClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(), "Unable to parse the provided url.")
	}

	status, _, _, _, err := clnt.GetObjectLockConfig(ctx)
	if err != nil {
		fatalIf(err.Trace(), "Unable to get bucket object lock configuration from `%s`", aliasedURL)
	}

	if status != "Enabled" {
		fatalIf(errDummy().Trace(), "Remote bucket does not support locking `%s`", aliasedURL)
	}
}

// main for retention command.
func mainLegalHold(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
}
