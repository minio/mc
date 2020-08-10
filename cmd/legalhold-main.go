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
	"encoding/json"
	"fmt"

	"github.com/minio/cli"
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
	VersionID string                `json:"versionID"`
	Status    string                `json:"status"`
	Err       error                 `json:"error,omitempty"`
}

// Colorized message for console printing.
func (l legalHoldCmdMessage) String() string {
	if l.Err != nil {
		return console.Colorize("LegalHoldMessageFailure", "Cannot set object legal hold status `"+l.URLPath+"`. "+l.Err.Error())
	}
	return console.Colorize("LegalHoldSuccess", fmt.Sprintf("Object legal hold successfully set for `%s`.", l.URLPath))
}

// JSON'ified message for scripting.
func (l legalHoldCmdMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// main for retention command.
func mainLegalHold(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
}
