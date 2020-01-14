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
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
)

var ilmGenerateCmd = cli.Command{
	Name:   "generate",
	Usage:  "Generate Information bucket/object lifecycle management information in JSON",
	Action: mainLifecycleGenerate,
	Before: setGlobalsFromContext,
	Flags:  append(ilmGenerateFlags, globalFlags...),
}

var ilmGenerateFlags = []cli.Flag{
	cli.StringFlag{
		Name: strings.ToLower(idLabel),
	},
	cli.StringFlag{
		Name: strings.ToLower(prefixLabel),
	},
	cli.StringSliceFlag{
		Name: strings.ToLower(tagLabel),
	},
	cli.StringFlag{
		Name: strings.ToLower(expiryDatesLabelFlag),
	},
	cli.StringFlag{
		Name: strings.ToLower(expiryDaysLabelFlag),
	},
	cli.StringFlag{
		Name: strings.ToLower(transitionDatesLabelKey),
	},
	cli.StringFlag{
		Name: strings.ToLower(transitionDaysLabelKey),
	},
	cli.StringFlag{
		Name: strings.ToLower(storageClassLabel),
	},
	cli.BoolFlag{
		Name: strings.ToLower(statusDisabledLabel),
	},
}

func checkIlmTranExpDateErr(rule lifecycleRule) bool {
	expirySet := (rule.Expiration != nil)
	expiryDateSet := expirySet && rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()
	expiryDaySet := expirySet && rule.Expiration.ExpirationInDays > 0

	transitionSet := (rule.Transition != nil)
	transitionDateSet := transitionSet && (rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero())
	transitionDaySet := transitionSet && (rule.Transition.TransitionInDays > 0)

	if transitionDateSet && expiryDateSet {
		return rule.Expiration.ExpirationDate.Before(*(rule.Transition.TransitionDate))
	}
	if transitionDaySet && expiryDaySet {
		return rule.Transition.TransitionInDays >= rule.Expiration.ExpirationInDays
	}
	return true
}

func checkIlmObject(rule lifecycleRule) bool {
	if checkIlmTranExpDateErr(rule) {
		console.Errorln("Error in Transition/Expiry date compatibility.")
		return false
	}

	return true
}

// Validate user given arguments
func checkIlmGenerateSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		cli.ShowCommandHelp(ctx, "")
		os.Exit(globalErrorExitStatus)
	}
	args := ctx.Args()
	objectURL := args.Get(0)
	//Empty or whatever
	_, err := getIlmInfo(objectURL)
	if err != nil {
		console.Println(console.Colorize(fieldMainHeader, "Possible error in the arguments or access. "+err.String()))
		os.Exit(globalErrorExitStatus)
	}

}

func extractILMTags(tagLabelVal []string) []tagFilter {
	var ilmTagKVList []tagFilter
	for tagIdx := 0; tagIdx < len(tagLabelVal); tagIdx++ {
		key := splitStr(tagLabelVal[tagIdx], keyValSeperator, 2)[0]
		val := splitStr(tagLabelVal[tagIdx], keyValSeperator, 2)[1]
		if key != "" && val != "" {
			ilmTagKVList = append(ilmTagKVList, tagFilter{Key: key, Value: val})
		}
	}
	return ilmTagKVList
}

func extractExpiry(expirationArg string) lifecycleExpiration {
	var rgxPtrn = regexp.MustCompile(`^[0-9]+ days`)
	var expiry lifecycleExpiration
	//	expiry.ExpirationInDays = 0

	expiryDate, dateparseerr := time.Parse(defaultDateFormat, expirationArg)
	if rgxPtrn.MatchString(expirationArg) {
		expirationDaysStr := splitStr(expirationArg, " ", 2)[0]
		if expirationDaysStr != "" {
			expiry.ExpirationInDays, _ = strconv.Atoi(expirationDaysStr)
		}
	} else if !expiryDate.IsZero() {
		expiry.ExpirationDate = &expiryDate
	} else if dateparseerr != nil {
		console.Println("Expiry date argument extraction resulted in an error. Generated JSON may have an error or Expiry information missing. Error: " + dateparseerr.Error())
	} else {
		dateerror := fmt.Sprintf("%s", "Expiry date argument extraction resulted in an error. Generated JSON may have an error or Expiry information missing.")
		console.Println(console.Colorize(fieldMainHeader, dateerror))
		console.Errorln(dateparseerr.Error())

	}
	return expiry
}

func extractTransition(transitionDateArg string, storageClassArg string) lifecycleTransition {
	var tran lifecycleTransition
	errExtract := false
	var parseDateErr error
	var rgxPtrn = regexp.MustCompile(`^[0-9]+ days`)
	if rgxPtrn.MatchString(transitionDateArg) {
		transitionDaysStr := splitStr(transitionDateArg, " ", 2)[0]
		if transitionDaysStr != "" {
			tran.TransitionInDays, parseDateErr = strconv.Atoi(transitionDaysStr)
		}
		if transitionDaysStr == "" || parseDateErr != nil {
			errExtract = true
		}
	} else {
		tran.TransitionDate = &time.Time{}
		*(tran.TransitionDate), parseDateErr = time.Parse(defaultDateFormat, transitionDateArg)
		if parseDateErr != nil {
			errExtract = true
		}
	}
	if errExtract {
		console.Errorln("Error in Transition Date Args: " + transitionDateArg + " Storage-Class: " + storageClassArg + ". Error:" + parseDateErr.Error())
	} else {
		tran.StorageClass = storageClassArg
	}
	return tran
}

func getExpiry(ctx *cli.Context) lifecycleExpiration {
	var expiry lifecycleExpiration
	expiryArg := ""
	switch {
	case ctx.String(strings.ToLower(expiryDaysLabelFlag)) != "":
		expiryArg = ctx.String(strings.ToLower(expiryDaysLabelFlag))
	case ctx.String(strings.ToLower(expiryDatesLabelFlag)) != "":
		expiryArg = ctx.String(strings.ToLower(expiryDatesLabelFlag))
	}
	if expiryArg != "" {
		expiry = extractExpiry(expiryArg)
	}
	return expiry
}

func getTransition(ctx *cli.Context) lifecycleTransition {
	var transition lifecycleTransition
	transitionDateArg := ""
	storageClassArg := ctx.String(strings.ToLower(storageClassLabel))
	switch {
	case ctx.String(strings.ToLower(transitionDatesLabelKey)) != "":
		transitionDateArg = ctx.String(strings.ToLower(transitionDatesLabelKey))
	case ctx.String(strings.ToLower(transitionDaysLabelKey)) != "":
		transitionDateArg = ctx.String(strings.ToLower(transitionDaysLabelKey))
	}
	transitionCheck := (transitionDateArg != "" && storageClassArg != "") ||
		(transitionDateArg == "" && storageClassArg == "")
	if !transitionCheck {
		console.Errorln("Error in Transition argument specification. Please check.")
		os.Exit(globalErrorExitStatus)
	} else if transitionDateArg != "" && storageClassArg != "" {
		transition = extractTransition(transitionDateArg, storageClassArg)
	}
	return transition
}

func getILMInfoFromUserValues(ctx *cli.Context, lfcInfoP *ilmResult) lifecycleRule {
	if lfcInfoP == nil {
		return lifecycleRule{}
	}
	var expiry lifecycleExpiration
	var transition lifecycleTransition

	ilmID := ctx.String(strings.ToLower(idLabel))
	ilmPrefix := ctx.String(strings.ToLower(prefixLabel))
	if ilmID == "" && ilmPrefix == "" {
		console.Errorln("Cannot have a rule without ID , Prefix. Please re-enter.")
		os.Exit(globalErrorExitStatus)
	}
	ilmStatus := statusLabel
	if ilmDisabled := ctx.Bool(strings.ToLower(statusDisabledLabel)); ilmDisabled {
		ilmStatus = statusDisabledLabel
	}
	ilmTagKVList := extractILMTags(ctx.StringSlice(strings.ToLower(tagLabel)))

	expiry = getExpiry(ctx)
	transition = getTransition(ctx)

	var andVal lifecycleAndOperator
	andVal.Tags = ilmTagKVList
	filter := lifecycleRuleFilter{Prefix: ilmPrefix}
	if len(andVal.Tags) > 0 {
		filter.And = &andVal
		filter.And.Prefix = filter.Prefix
		filter.Prefix = ""
	}
	var expP *lifecycleExpiration
	var transP *lifecycleTransition
	if (transition.TransitionDate != nil &&
		!transition.TransitionDate.IsZero()) || transition.TransitionInDays > 0 {
		transP = &transition
	}
	if (expiry.ExpirationDate != nil &&
		!expiry.ExpirationDate.IsZero()) || expiry.ExpirationInDays > 0 {
		expP = &expiry
	}

	newRule := lifecycleRule{
		ID:         ilmID,
		RuleFilter: &filter,
		Status:     ilmStatus,
		Expiration: expP,
		Transition: transP,
	}
	(*lfcInfoP).Rules = append((*lfcInfoP).Rules, newRule)

	return newRule
}

func mainLifecycleGenerate(ctx *cli.Context) error {
	console.SetColor(fieldThemeHeader, color.New(color.Bold, color.FgHiBlue))
	checkIlmGenerateSyntax(ctx)
	lfcInfo := ilmResult{}
	args := ctx.Args()
	objectURL := args.Get(0)
	//Empty or whatever
	lfcInfoXML, err := getIlmInfo(objectURL)
	if err != nil {
		console.Println("Error generating lifecycle contents in XML: " + err.String())
		return err.ToGoError()
	}
	if lfcInfoXML != "" {
		unerr := xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo)
		if unerr != nil {
			console.Println("Error converting generated lifecycle contents in XML to object: " + unerr.Error())
			return unerr
		}
	}
	rule := getILMInfoFromUserValues(ctx, &lfcInfo)
	if !checkIlmObject(rule) {
		console.Errorln("Error found in generated/combined rule.")
	} else {
		printIlmJSON(lfcInfo)
	}
	return nil
}
