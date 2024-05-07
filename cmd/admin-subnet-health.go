// Copyright (c) 2015-2022 MinIO, Inc.
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
	"fmt"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/minio-go/v7/pkg/set"
)

var adminSubnetHealthCmd = cli.Command{
	Name:               "health",
	Usage:              "generate MinIO health report for SUBNET",
	OnUsageError:       onUsageError,
	Action:             mainSubnetHealth,
	Before:             setGlobalsFromContext,
	Hidden:             true,
	Flags:              supportDiagFlags, // No need to append globalFlags as top level command would add them
	CustomHelpTemplate: "This command is deprecated and will be removed in a future release. Use 'mc support diag' instead.\n",
}

func mainSubnetHealth(ctx *cli.Context) error {
	boolValSet := set.CreateStringSet("true", "false")
	newCmd := []string{"mc support diag"}
	newCmd = append(newCmd, ctx.Args()...)
	for _, flg := range ctx.Command.Flags {
		flgName := flg.GetName()
		if !ctx.IsSet(flgName) {
			continue
		}

		// replace the deprecated --offline with --airgap
		if flgName == "offline" {
			flgName = "airgap"
		}

		flgStr := "--" + flgName
		flgVal := ctx.String(flgName)
		if !boolValSet.Contains(flgVal) {
			flgStr = fmt.Sprintf("%s \"%s\"", flgStr, flgVal)
		}
		newCmd = append(newCmd, flgStr)
	}

	deprecatedError(strings.Join(newCmd, " "))
	return nil
}
