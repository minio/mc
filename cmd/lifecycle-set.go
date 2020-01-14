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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

// TODO: The usage and examples will change as the command implementation evolves after feedback.
var ilmSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set Information bucket/object lifecycle management information",
	Action: mainLifecycleSet,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
}

// checkIlmSetSyntax - validate arguments passed by a user
func checkIlmSetSyntax(ctx *cli.Context) {
	// fmt.Println(len(ctx.Args()))
	if len(ctx.Args()) == 0 || len(ctx.Args()) != 2 {
		cli.ShowCommandHelp(ctx, "")
		os.Exit(globalErrorExitStatus)
	}
}

func setIlmFromFile(urlstr string, file string) {
	api, perr := getMinioClient(urlstr)
	fatalIf(perr, "Unable to get client to set lifecycle from url: "+urlstr)
	bkt := getBucketNameFromURL(urlstr)
	if bkt == "" || len(bkt) == 0 {
		bkterrstr := fmt.Sprintf("%s", "Error bucket name "+urlstr)
		console.Println(console.Colorize(fieldMainHeader, bkterrstr))
		return
	}
	fileContents := readFileToString(file)
	if fileContents == "" || !checkFileCompatibility(fileContents) {
		console.Println("Found compatibility issues with file contents from: " + file + ". May not be able to set bucket lifecycle.")
	}
	var ilm ilmResult
	if err := json.Unmarshal([]byte(fileContents), &ilm); err != nil {
		errorIf(probe.NewError(err), "Unable to set lifecycle for bucket: "+bkt+" from file: "+file)
	}
	// console.Println(ilm)
	cbfr, err := xml.Marshal(ilm)
	if err != nil {
		errorIf(probe.NewError(err), "Unable to set lifecycle from contents of file: "+file)
	}
	ilmContents := string(cbfr)
	// console.Println(ilmContents)
	if err = api.SetBucketLifecycle(bkt, ilmContents); err != nil {
		fatalIf(probe.NewError(err), "Unable to set lifecycle for bucket: "+bkt+". URL: "+urlstr+". File: "+file)
	}
}

func mainLifecycleSet(ctx *cli.Context) error {
	checkIlmSetSyntax(ctx)
	setColorScheme()
	fileNamePath := ctx.Args()[1]
	if err := checkFileNamePathExists(fileNamePath); err != nil {
		fatalIf(probe.NewError(err), "File error:"+fileNamePath)
	}
	args := ctx.Args()
	objectURL := args.Get(0)

	setIlmFromFile(objectURL, fileNamePath)
	// console.Println("Success.")
	successStr := fmt.Sprintf("%s", "Success. Lifecycle configuration set from file:"+fileNamePath)
	console.Println(console.Colorize(fieldThemeResultSuccess, successStr))
	return nil
}
