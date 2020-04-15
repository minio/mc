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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmCmd = cli.Command{
	Name:   "ilm",
	Usage:  "configure bucket lifecycle",
	Action: mainILM,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		ilmListCmd,
		ilmAddCmd,
		ilmRemoveCmd,
		ilmExportCmd,
		ilmImportCmd,
	},
}

const (
	ilmMainHeader         string = "Main-Heading"
	ilmThemeHeader        string = "Row-Header"
	ilmThemeRow           string = "Row-Normal"
	ilmThemeTick          string = "Row-Tick"
	ilmThemeExpiry        string = "Row-Expiry"
	ilmThemeResultSuccess string = "SuccessOp"
	ilmThemeResultFailure string = "FailureOp"
)

func mainILM(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, "")
	return nil
}

// setBucketILMConfiguration sets the lifecycle configuration given by ilmConfig to the bucket given by the url (urlStr)
func setBucketILMConfiguration(urlStr string, ilmConfig string) *probe.Error {
	client, pErr := newClient(urlStr)
	fatalIf(pErr, "Failed to set lifecycle configuration to "+urlStr)
	s3c, ok := client.(*S3Client)
	if !ok {
		fatalIf(errDummy().Trace(urlStr), "For "+urlStr+" unable to obtain client reference.")
	}
	if pErr = s3c.SetBucketLifecycle(ilmConfig); pErr != nil {
		return pErr
	}
	return nil
}

// getBucketILMConfiguration gets the lifecycle configuration for the bucket given by the url (urlStr)
func getBucketILMConfiguration(urlStr string) (string, *probe.Error) {
	var bktConfig string
	client, pErr := newClient(urlStr)
	fatalIf(pErr, "Failed to get lifecycle configuration to "+urlStr)
	s3c, ok := client.(*S3Client)
	if !ok {
		fatalIf(errDummy().Trace(urlStr), "For "+urlStr+" unable to obtain client reference.")
	}
	if bktConfig, pErr = s3c.GetBucketLifecycle(); pErr != nil {
		return "", pErr
	}
	return bktConfig, nil
}

// Color scheme for the table
func setILMDisplayColorScheme() {
	console.SetColor(ilmMainHeader, color.New(color.Bold, color.FgHiRed))
	console.SetColor(ilmThemeRow, color.New(color.FgHiWhite))
	console.SetColor(ilmThemeHeader, color.New(color.Bold, color.FgHiGreen))
	console.SetColor(ilmThemeTick, color.New(color.FgGreen))
	console.SetColor(ilmThemeExpiry, color.New(color.BlinkRapid, color.FgGreen))
	console.SetColor(ilmThemeResultSuccess, color.New(color.FgGreen, color.Bold))
	console.SetColor(ilmThemeResultFailure, color.New(color.FgHiYellow, color.Bold))
}
