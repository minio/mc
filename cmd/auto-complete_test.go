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
