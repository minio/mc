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
