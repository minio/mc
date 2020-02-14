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
	"encoding/xml"
	"errors"
	"os"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove (if any) existing lifecycle configuration rule with the id",
	Action: mainLifecycleRemove,
	Before: setGlobalsFromContext,
	Flags:  append(ilmRemoveFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
	{{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
	{{range .VisibleFlags}}{{.}}
	{{end}}
	
DESCRIPTION:
	Remove the lifecycle configuration rule for the bucket denoted by the ID or all configurations only if specified (--all --force).


EXAMPLES:
1. Remove the lifecycle management configuration rule denoted by ID with value "Documents" for the testbucket on alias s3. ID is case sensitive.
	{{.Prompt}} {{.HelpName}} --id "Documents" s3/testbucket
2. Remove ALL the lifecycle management configuration rules for the testbucket on alias s3. Because the result is complete removal, the use of --force flag is enforced.
	{{.Prompt}} {{.HelpName}} --all --force s3/testbucket


`,
}

var ilmRemoveFlags = []cli.Flag{
	cli.StringFlag{
		Name:  strings.ToLower(idLabel),
		Usage: "id for the rule, unique value & case-sensitive",
	},
	cli.BoolFlag{
		Name:  forceLabel,
		Usage: "force flag to be used when deleting all lifecycle configuration rules of the bucket",
	},
	cli.BoolFlag{
		Name:  allLabel,
		Usage: "delete all lifecycle configuration rules of the bucket, force flag enforced",
	},
}

func checkIlmRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) >= 2 {
		cli.ShowCommandHelp(ctx, "remove")
		os.Exit(globalErrorExitStatus)
	}
	ilmAll := ctx.Bool(allLabel)
	ilmForce := ctx.Bool(forceLabel)
	forceChk := (ilmAll && ilmForce) || (!ilmAll && !ilmForce)
	if !forceChk {
		fatalIf(errInvalidArgument(), "Mandatory to enter --all & --force flag together. Force flag enforced for all deletion (only).")
	}

	args := ctx.Args()
	objectURL := args.Get(0)
	//Empty or whatever
	_, err := getIlmInfo(objectURL)
	if err != nil {
		console.Errorln(console.Colorize(fieldMainHeader, "Possible error in the arguments or access. "+err.String()))
		os.Exit(globalErrorExitStatus)
	}
}

func ilmAllRemove(urlStr string) error {
	bkt := getBucketNameFromURL(urlStr)
	api, apierr := getMinioClient(urlStr)
	if apierr != nil {
		console.Errorln("Error getting API. " + apierr.String() + ". Error:" + apierr.ToGoError().Error())
		return apierr.ToGoError()
	}
	if api == nil {
		apierrstr := "unable to initialize the API to remove bucket lifecycle configuration"
		console.Errorln(apierrstr)
		return errors.New(apierrstr)
	}
	if ilmErr := api.SetBucketLifecycle(bkt, ""); ilmErr != nil {
		failureStr := "Failure. Lifecycle configuration not removed for bucket " + bkt
		console.Errorln(failureStr)
		return ilmErr
	}
	successStr := "Success. Lifecycle configuration removed for bucket " + bkt
	console.Println(console.Colorize(fieldThemeResultSuccess, successStr))

	return nil
}

func ilmIDRemove(ilmID string, urlStr string) (bool, error) {
	var err error
	var pErr *probe.Error
	var lfcInfoXML string
	var lfcInfo lifecycleConfiguration

	api, pErr := getMinioClient(urlStr)
	if pErr != nil {
		return false, pErr.ToGoError()
	}
	if api == nil {
		errstr := "unable to call the API to remove lifecycle rule, target: " + urlStr
		return false, errors.New(errstr)
	}
	if lfcInfoXML, pErr = getIlmInfo(urlStr); pErr != nil {
		return false, pErr.ToGoError()
	}
	if lfcInfoXML != "" {
		if err = xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo); err != nil {
			return false, err
		}
		idx := 0
		ruleFound := false
		foundIdx := -1
		for range lfcInfo.Rules {
			rule := lfcInfo.Rules[idx]
			if rule.ID == ilmID {
				ruleFound = true
				foundIdx = idx
			}
			idx++
		}
		if ruleFound && foundIdx != -1 && len(lfcInfo.Rules) > 1 {
			lfcInfo.Rules = append(lfcInfo.Rules[:foundIdx], lfcInfo.Rules[foundIdx+1:]...)
		} else if ruleFound && foundIdx != -1 && len(lfcInfo.Rules) <= 1 && ilmAllRemove(urlStr) == nil { // Only rule. Remove all.
			return true, nil
		}
		if ruleFound && setILM(urlStr, lfcInfo) == nil {
			return true, nil
		}
	}
	console.Println(console.Colorize(fieldThemeResultFailure, "Rule with ID `"+ilmID+"` not found or could not be deleted."))

	return false, nil
}

func mainLifecycleRemove(ctx *cli.Context) error {
	setColorScheme()
	checkIlmRemoveSyntax(ctx)
	args := ctx.Args()
	objectURL := args.Get(0)
	var err error
	var ilmAll, ilmForce, res bool
	var ilmID string
	ilmAll = ctx.Bool(strings.ToLower(allLabel))
	ilmForce = ctx.Bool(strings.ToLower(forceLabel))
	failStr := "Failure. Lifecycle configuration could not be removed."
	if ilmAll && ilmForce {
		if err = ilmAllRemove(objectURL); err != nil {
			failStr += " Error: " + err.Error()
			console.Println(console.Colorize(fieldThemeResultFailure, failStr))
			return err
		}
		return nil
	}
	ilmID = ctx.String(strings.ToLower(idLabel))
	if ilmID != "" {
		if res, err = ilmIDRemove(ilmID, objectURL); err != nil {
			failStr += "ID: " + ilmID + "Target: " + objectURL + " Error: " + err.Error()
			console.Println(console.Colorize(fieldThemeResultFailure, failStr))
		}
		if !res {
			cli.ShowCommandHelp(ctx, "remove")
		} else {
			successStr := "Success. Lifecycle configuration rule with ID `" + ilmID + "` removed."
			console.Println(console.Colorize(fieldThemeResultSuccess, successStr))
		}
	} else {
		console.Println("ID for lifecycle configuration rule not specified.")
	}
	return err
}
