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
	"io"
	"strconv"

	"os"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
)

// TODO: The usage and examples will change as the command implementation evolves after feedback.
var ilmImportCmd = cli.Command{
	Name:   "import",
	Usage:  "import lifecycle configuration in JSON format",
	Action: mainLifecycleImport,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
	{{.HelpName}} TARGET

DESCRIPTION:
	Lifecycle configuration is imported. Input is required in JSON format.

TARGET:
	This argument needs to be in the format of 'alias/bucket/prefix' or 'alias/bucket'

EXAMPLES:
1. Set lifecycle configuration for the testbucket on alias s3 to the rules imported from lifecycle.json
	{{.Prompt}} {{.HelpName}} s3/testbucket < /Users/miniouser/Documents/lifecycle.json
2. Set lifecycle configuration for the testbucket on alias s3. User is expected to enter the JSON contents on STDIN
	{{.Prompt}} {{.HelpName}} s3/testbucket

`,
}

// checkIlmSetSyntax - validate arguments passed by a user
func checkIlmImportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "import")
		os.Exit(globalErrorExitStatus)
	}
}

func mainLifecycleImport(ctx *cli.Context) error {
	checkIlmImportSyntax(ctx)
	setColorScheme()

	args := ctx.Args()
	objectURL := args.Get(0)
	var err error

	if len(args) == 1 {
		var successStr string
		// User is expected to enter the lifecycleConfiguration instance contents in JSON format
		var ilmInfo lifecycleConfiguration
		// Consume json from STDIN
		dec := json.NewDecoder(os.Stdin)
		for {
			err = dec.Decode(&ilmInfo)
			if err == io.EOF {
				break
			}
			if err != nil {
				console.Errorln("JSON import error:" + err.Error())
				return err
			}
		}
		if len(ilmInfo.Rules) <= 0 {
			successStr = "Lifecycle configuration imported. But no rules were added."
		} else if err = setILM(objectURL, ilmInfo); err != nil {
			successStr = "Failure, lifecycle configuration could not be imported. " + err.Error()
		} else {
			successStr = "Success. Lifecycle configuration imported. Number of rules " + strconv.Itoa(len(ilmInfo.Rules))
		}
		console.Println(console.Colorize(fieldThemeResultSuccess, successStr))
	}

	return nil
}
