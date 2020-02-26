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
	ilm "github.com/minio/mc/cmd/ilm"
	"github.com/minio/minio/pkg/console"
)

var ilmExportCmd = cli.Command{
	Name:   "export",
	Usage:  "export lifecycle configuration in JSON format",
	Action: mainILMExport,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Lifecycle configuration of the target bucket exported in JSON format.

EXAMPLES:
  1. Redirect output of lifecycle configuration rules of the testbucket on alias s3 to the file lifecycle.json
     {{.Prompt}} {{.HelpName}} s3/testbucket >> /Users/miniouser/Documents/lifecycle.json
  2. Show lifecycle configuration rules of the testbucket on alias s3 on STDOUT
     {{.Prompt}} {{.HelpName}} s3/testbucket

`,
}

// checkILMExportSyntax - validate arguments passed by user
func checkILMExportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "export")
		os.Exit(globalErrorExitStatus)
	}
}

func mainILMExport(ctx *cli.Context) error {
	checkILMExportSyntax(ctx)
	setILMDisplayColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	var err error
	var ilmInfoXML string

	if ilmInfoXML, err = getILMXML(objectURL); err != nil {
		console.Errorln(err.Error() + ". Error getting lifecycle configuration.")
		return err
	}
	ilm.PrintILMJSON(ilmInfoXML)
	return nil
}
