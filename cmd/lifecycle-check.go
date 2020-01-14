/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"encoding/xml"
	"os"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmCheckCmd = cli.Command{
	Name:   "check",
	Usage:  "Check if json file has valid Information bucket/object lifecycle management setting",
	Action: mainLifecycleCheck,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
	{{.HelpName}} FILE

DESCRIPTION:
	Check the lifecycle configuration in the JSON file provided for validity.

FILE:
	This argument needs to be the correct path to the .json file with Lifecycle configuration.

EXAMPLES:
1. Check lifecycle management rules provided by s3_34bkt_lifecycle.json in JSON format.
	{{.Prompt}} {{.HelpName}} /Users/miniouser/Documents/s3_34bkt_lifecycle.json

`,
}

func checkIlmCheckSyntax(ctx *cli.Context) {
	// fmt.Println(len(ctx.Args()))
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "check")
		os.Exit(globalErrorExitStatus)
	}
}

func mainLifecycleCheck(ctx *cli.Context) error {
	checkIlmCheckSyntax(ctx)
	setColorScheme()
	fileNamePath := ctx.Args()[0]
	if err := checkFileNamePathExists(fileNamePath); err != nil {
		fatalIf(probe.NewError(err), "File error:"+fileNamePath)
	}

	fileContents := readFileToString(fileNamePath)
	if fileContents == "" || !checkFileCompatibility(fileContents) {
		console.Println("Found compatibility issues with file contents from: " + fileNamePath + ". May not be able to set bucket lifecycle.")
	}

	var ilm ilmResult
	if err := json.Unmarshal([]byte(fileContents), &ilm); err != nil {
		errorIf(probe.NewError(err), "Unable to get lifecycle configuration from file: "+fileNamePath)
		return err
	}
	_, err := xml.Marshal(ilm)

	if err != nil {
		errorIf(probe.NewError(err), "Unable to set lifecycle from contents of file: "+fileNamePath)
		return err
	}

	if len(ilm.Rules) == 0 {
		console.Println(console.Colorize(fieldMainHeader, "The rule list is empty."))
		return nil
	}

	successStr := "Success. Lifecycle configuration in file:" + fileNamePath + " is valid."
	console.Println(console.Colorize(fieldThemeResultSuccess, successStr))

	return nil
}
