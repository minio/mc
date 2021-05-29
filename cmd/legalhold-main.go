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
	"errors"
	"fmt"
	"net/http"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/pkg/console"
)

var legalHoldSubcommands = []cli.Command{
	legalHoldSetCmd,
	legalHoldClearCmd,
	legalHoldInfoCmd,
}

var legalHoldCmd = cli.Command{
	Name:        "legalhold",
	Usage:       "manage legal hold for object(s)",
	Action:      mainLegalHold,
	Before:      setGlobalsFromContext,
	Flags:       globalFlags,
	Subcommands: legalHoldSubcommands,
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

var errBucketLockConfigNotFound = errors.New("bucket lock config not found")

func isBucketLockEnabled(ctx context.Context, aliasedURL string) (bool, *probe.Error) {
	st, err := getBucketLockStatus(ctx, aliasedURL)
	if err == nil {
		return st == "Enabled", nil
	}
	if err.ToGoError() == errBucketLockConfigNotFound {
		return false, nil
	}
	return false, err
}

// Check if the bucket corresponding to the target url has object locking enabled
func getBucketLockStatus(ctx context.Context, aliasedURL string) (status string, err *probe.Error) {
	clnt, err := newClient(aliasedURL)
	if err != nil {
		return "", err
	}

	status, _, _, _, err = clnt.GetObjectLockConfig(ctx)
	if err != nil {
		errResp := minio.ToErrorResponse(err.ToGoError())
		if errResp.StatusCode == http.StatusNotFound {
			return "", probe.NewError(errBucketLockConfigNotFound)
		}
		return "", err
	}

	return status, nil
}

// main for retention command.
func mainLegalHold(ctx *cli.Context) error {
	commandNotFound(ctx, legalHoldSubcommands)
	return nil
}
