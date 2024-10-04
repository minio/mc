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
	"github.com/minio/cli"
	"github.com/minio/pkg/v3/console"
)

var policyFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "list recursively",
	},
}

// Manage anonymous access to buckets and objects.
var policyCmd = cli.Command{
	Name:         "policy",
	Usage:        "manage anonymous access to buckets and objects",
	Action:       mainPolicy,
	Hidden:       true,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(policyFlags, globalFlags...),
	CustomHelpTemplate: `Please use 'mc anonymous'
`,
}

func mainPolicy(_ *cli.Context) error {
	console.Infoln("Please use 'mc anonymous'")
	return nil
}
