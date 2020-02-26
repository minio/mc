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
	"errors"

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
		ilmShowCmd,
		ilmAddCmd,
		ilmRemoveCmd,
		ilmExportCmd,
		ilmImportCmd,
	},
}

const (
	fieldMainHeader         string = "Main-Heading"
	fieldThemeHeader        string = "Row-Header"
	fieldThemeRow           string = "Row-Normal"
	fieldThemeTick          string = "Row-Tick"
	fieldThemeExpiry        string = "Row-Expiry"
	fieldThemeResultSuccess string = "SucessOp"
	fieldThemeResultFailure string = "FailureOp"
)

func mainILM(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, "")
	return nil
}

// Get lifecycle info (XML) from alias & bucket
func getILMXML(urlStr string) (string, error) {
	lifecycleInfo, err := getBucketILMConfiguration(urlStr)
	if err != nil {
		return "", err
	}
	ok := checkILMBucketAccess(urlStr)
	if !ok {
		fatalIf(probe.NewError(errors.New("access failed "+urlStr)), "Unable to access target or lifecycle configuration.")
	}
	return lifecycleInfo, nil
}

func checkILMBucketAccess(urlStr string) bool {
	alias, _ := url2Alias(urlStr)
	if alias == "" {
		fatalIf(errInvalidAliasedURL(urlStr), "Unable to set tags to target "+urlStr)
	}
	clnt, pErr := newClient(urlStr)
	if pErr != nil {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "Cannot parse the provided url "+urlStr)
	}
	s3c, ok := clnt.(*s3Client)
	if !ok {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "For "+urlStr+" unable to obtain client reference.")
	}
	ok, _ = s3c.api.BucketExists(getILMBucketNameFromURL(urlStr))
	return ok
}

// setBucketILMConfiguration sets the lifecycle configuration given by ilmConfig to the bucket given by the url (urlStr)
func setBucketILMConfiguration(urlStr string, ilmConfig string) error {
	var err error
	alias, _ := url2Alias(urlStr)
	if alias == "" {
		fatalIf(errInvalidAliasedURL(urlStr), "Unable to set tags to target "+urlStr)
	}
	clnt, pErr := newClient(urlStr)
	if pErr != nil {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "Cannot parse the provided url "+urlStr)
	}
	s3c, ok := clnt.(*s3Client)
	if !ok {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "For "+urlStr+" unable to obtain client reference.")
	}
	bucket, _ := s3c.url2BucketAndObject()
	if err = s3c.api.SetBucketLifecycle(bucket, ilmConfig); err != nil {
		return err
	}
	return nil
}

// getBucketILMConfiguration gets the lifecycle configuration for the bucket given by the url (urlStr)
func getBucketILMConfiguration(urlStr string) (string, error) {
	var bktConfig string
	var err error

	alias, _ := url2Alias(urlStr)
	if alias == "" {
		fatalIf(errInvalidAliasedURL(urlStr), "Unable to set tags to target "+urlStr)
	}
	clnt, pErr := newClient(urlStr)
	if pErr != nil {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "Cannot parse the provided url "+urlStr)
	}
	s3c, ok := clnt.(*s3Client)
	if !ok {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "For "+urlStr+" unable to obtain client reference.")
	}
	bucket, _ := s3c.url2BucketAndObject()
	if bktConfig, err = s3c.api.GetBucketLifecycle(bucket); err != nil {
		return "", err
	}
	return bktConfig, nil
}

// getBucketNameFromURL - return bucket name from the 'alias/bucket'
func getILMBucketNameFromURL(urlStr string) string {
	alias, _ := url2Alias(urlStr)
	if alias == "" {
		fatalIf(errInvalidAliasedURL(urlStr), "Unable to set tags to target "+urlStr)
	}
	clnt, pErr := newClient(urlStr)
	if pErr != nil {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "Cannot parse the provided url "+urlStr)
	}
	s3c, ok := clnt.(*s3Client)
	if !ok {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "For "+urlStr+" unable to obtain client reference.")
	}
	bucket, _ := s3c.url2BucketAndObject()
	return bucket
}

// Color scheme for the table
func setILMDisplayColorScheme() {
	console.SetColor(fieldMainHeader, color.New(color.Bold, color.FgHiRed))
	console.SetColor(fieldThemeRow, color.New(color.FgHiWhite))
	console.SetColor(fieldThemeHeader, color.New(color.Bold, color.FgHiGreen))
	console.SetColor(fieldThemeTick, color.New(color.FgGreen))
	console.SetColor(fieldThemeExpiry, color.New(color.BlinkRapid, color.FgGreen))
	console.SetColor(fieldThemeResultSuccess, color.New(color.FgGreen, color.Bold))
	console.SetColor(fieldThemeResultFailure, color.New(color.FgHiYellow, color.Bold))
}
