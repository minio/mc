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

// checkILMImportSyntax - validate arguments passed by user
func checkILMImportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "import")
		os.Exit(globalErrorExitStatus)
	}
}

var ilmImportCmd = cli.Command{
	Name:   "import",
	Usage:  "import lifecycle configuration in JSON format",
	Action: mainILMImport,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Lifecycle configuration is imported. Input is required in JSON format.

EXAMPLES:
  1. Set lifecycle configuration for the testbucket on alias s3 to the rules imported from lifecycle.json
     {{.Prompt}} {{.HelpName}} s3/testbucket < /Users/miniouser/Documents/lifecycle.json
  2. Set lifecycle configuration for the testbucket on alias s3. User is expected to enter the JSON contents on STDIN
     {{.Prompt}} {{.HelpName}} s3/testbucket

`,
}

func mainILMImport(ctx *cli.Context) error {
	checkILMImportSyntax(ctx)
	setILMDisplayColorScheme()

	args := ctx.Args()
	objectURL := args.Get(0)
	var err error
	var ilmXML string
	if ilmXML, err = ilm.ReadILMConfigJSON(objectURL); err != nil {
		console.Println(console.Colorize(fieldThemeResultFailure, err.Error()+" Reading lifecycle configuration failed."))
		return err
	}
	if err = setBucketILMConfiguration(objectURL, ilmXML); err != nil {
		console.Println(console.Colorize(fieldThemeResultFailure, err.Error()+" Setting Lifecycle configuration failed."))
		return err
	}

	return nil
}
