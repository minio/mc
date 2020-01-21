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
	"os"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
)

var ilmExportCmd = cli.Command{
	Name:   "export",
	Usage:  "export lifecycle configuration in JSON format",
	Action: mainLifecycleExport,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
	{{.HelpName}} TARGET

DESCRIPTION:
	Lifecycle configuration of the target bucket exported in JSON format.

TARGET:
	This argument needs to be in the format of 'alias/bucket/prefix' or 'alias/bucket'

EXAMPLES:
1. Redirect output of lifecycle configuration rules of the test34bucket on alias s3 to the file s3_34bkt_lifecycle.json
	{{.Prompt}} {{.HelpName}} s3/test34bucket >> /Users/miniouser/Documents/s3_34bkt_lifecycle.json
2. Show lifecycle configuration rules of the test34bucket on alias s3 on STDOUT
	{{.Prompt}} {{.HelpName}} s3/test34bucket

`,
}

// checkIlmExportSyntax - validate arguments passed by a user
func checkIlmExportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "export")
		os.Exit(globalErrorExitStatus)
	}
}

func mainLifecycleExport(ctx *cli.Context) error {
	checkIlmExportSyntax(ctx)
	setColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	var err error
	var ilmInfo lifecycleConfiguration

	if len(args) == 1 {
		if ilmInfo, err = getIlmConfig(objectURL); err != nil {
			console.Errorln("Error getting lifecycle configuration: " + err.Error())
			return err
		}
		if len(ilmInfo.Rules) > 0 {
			printIlmJSON(ilmInfo)
		}
	}
	return nil
}
