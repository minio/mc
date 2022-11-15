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
	"testing"

	"github.com/minio/cli"
)

func TestCLIOnUsageError(t *testing.T) {
	var checkOnUsageError func(cli.Command, string)
	checkOnUsageError = func(cmd cli.Command, parentCmd string) {
		if cmd.Subcommands != nil {
			for _, subCmd := range cmd.Subcommands {
				if cmd.Hidden {
					continue
				}
				checkOnUsageError(subCmd, parentCmd+" "+cmd.Name)
			}
			return
		}
		if !cmd.Hidden && cmd.OnUsageError == nil {
			t.Errorf("On usage error for `%s` not found", parentCmd+" "+cmd.Name)
		}
	}

	for _, cmd := range appCmds {
		if cmd.Hidden {
			continue
		}
		checkOnUsageError(cmd, "")
	}
}
