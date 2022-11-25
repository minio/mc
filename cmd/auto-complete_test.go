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
	"testing"

	"github.com/minio/cli"
)

func TestAutoCompletionCompletness(t *testing.T) {
	var checkCompletion func(cmd cli.Command, cmdPath string) error

	checkCompletion = func(cmd cli.Command, cmdPath string) error {
		if cmd.Subcommands != nil {
			for _, subCmd := range cmd.Subcommands {
				if cmd.Hidden {
					continue
				}
				err := checkCompletion(subCmd, cmdPath+"/"+subCmd.Name)
				if err != nil {
					return err
				}
			}
			return nil
		}
		_, ok := completeCmds[cmdPath]
		if !ok && !cmd.Hidden {
			return fmt.Errorf("Completion for `%s` not found", cmdPath)
		}
		return nil
	}

	for _, cmd := range appCmds {
		if cmd.Hidden {
			continue
		}
		err := checkCompletion(cmd, "/"+cmd.Name)
		if err != nil {
			t.Fatalf("Missing completion function: %v", err)
		}

	}
}
