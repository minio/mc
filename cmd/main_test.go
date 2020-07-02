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

// This function checks that only leaf commands have Before
// function configured to avoid calling it multiple times.
func TestBeforeCommand(t *testing.T) {
	var checkCmd func(string, cli.Command)

	checkCmd = func(cmdPath string, c cli.Command) {
		if len(c.Subcommands) != 0 {
			if c.Before != nil {
				t.Fatalf("Command %s does not expect to have a Before func since it is not the leaf command", cmdPath)
			}
		} else {
			if c.Before == nil {
				t.Fatalf("Command %s expects to have a Before func since it is a leaf command", cmdPath)
			}
		}

		// Test subcommands
		for _, subCmd := range c.Subcommands {
			checkCmd(cmdPath+"/"+subCmd.Name, subCmd)
		}
	}

	for _, cmd := range appCmds {
		checkCmd("", cmd)
	}
}
