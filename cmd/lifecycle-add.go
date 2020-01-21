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
	"strconv"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio/pkg/console"
)

var ilmAddCmd = cli.Command{
	Name:   "add",
	Usage:  "add a lifecycle configuration rule to existing (if any) rules of the bucket",
	Action: mainLifecycleAdd,
	Before: setGlobalsFromContext,
	Flags:  append(ilmAddFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
 {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
 {{range .VisibleFlags}}{{.}}
 {{end}}
DESCRIPTION:
	ILM add command adds one rule to the existing (if any) set of lifecycle configuration rules. If a rule with the ID already exists it will be replaced.

TARGET:
	This argument needs to be in the format of 'alias/bucket'

EXAMPLES:
1. Add rule for the test34bucket on s3. Both expiry & transition are given as dates.
	{{.Prompt}} {{.HelpName}} --id "Devices" --prefix "dev/" --expiry-date "2020-09-17" --transition-date "2020-05-01" --storage-class "GLACIER" s3/test34bucket

2. Add rule for the test34bucket on s3. Both expiry and transition are given as number of days.
	{{.Prompt}} {{.HelpName}} --id "Docs" --prefix "doc/" --expiry-days "200" --transition-days "300 days" --storage-class "GLACIER" s3/test34bucket

3. Add rule for the test34bucket on s3. Only expiry is given as number of days.
	{{.Prompt}} {{.HelpName}} --id "Docs" --prefix "doc/" --expiry-days "200" --tags "docformat:docx" --tags "plaintextformat:txt" --tags "PDFFormat:pdf" s3/test34bucket

`,
}

var ilmAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  strings.ToLower(idLabel),
		Usage: "id for the rule, should be an unique value",
	},
	cli.StringFlag{
		Name:  strings.ToLower(prefixLabel),
		Usage: "prefix to apply the lifecycle configuration rule",
	},
	cli.StringSliceFlag{
		Name:  strings.ToLower(tagLabel),
		Usage: "format '<key>:<value>'; multiple values allowed for multiple key/value pairs",
	},
	cli.StringFlag{
		Name:  strings.ToLower(expiryDatesLabelFlag),
		Usage: "format 'YYYY-mm-dd' the date of expiration",
	},
	cli.StringFlag{
		Name:  strings.ToLower(expiryDaysLabelFlag),
		Usage: "the number of days to expiration",
	},
	cli.StringFlag{
		Name:  strings.ToLower(transitionDatesLabelKey),
		Usage: "format 'YYYY-MM-DD' for the date to transition",
	},
	cli.StringFlag{
		Name:  strings.ToLower(transitionDaysLabelKey),
		Usage: "the number of days to transition",
	},
	cli.StringFlag{
		Name:  strings.ToLower(storageClassLabelKey),
		Usage: "storage class for transition (STANDARD_IA, ONEZONE_IA, GLACIER. Etc)",
	},
	cli.BoolFlag{
		Name:  strings.ToLower(statusDisabledLabel),
		Usage: "disable the rule",
	},
}

// Validate user given arguments
func checkIlmAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		cli.ShowCommandHelp(ctx, "add")
		os.Exit(globalErrorExitStatus)
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

// Extracts the tags provided by user. The tagfilter array will be put in lifecycleRule structure.
func extractILMTags(tagLabelVal []string) ([]tagFilter, error) {
	var ilmTagKVList []tagFilter
	for tagIdx := 0; tagIdx < len(tagLabelVal); tagIdx++ {
		key := splitStr(tagLabelVal[tagIdx], keyValSeperator, 2)[0]
		val := splitStr(tagLabelVal[tagIdx], keyValSeperator, 2)[1]
		if key != "" && val != "" {
			ilmTagKVList = append(ilmTagKVList, tagFilter{Key: key, Value: val})
		} else {
			return nil, errors.New("extracting tag argument lifecycle configuration rule failed")
		}
	}
	return ilmTagKVList, nil
}

// Some of these rules are enforced by Amazon S3 standards.
// For example: Transition has to happen before Expiry.
// Storage class must be specified if transition date/days is provided.
func validateTranExpDate(rule lifecycleRule) error {
	expirySet := (rule.Expiration != nil)
	expiryDateSet := expirySet && rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()
	expiryDaySet := expirySet && rule.Expiration.ExpirationInDays > 0

	transitionSet := (rule.Transition != nil)
	transitionDateSet := transitionSet && (rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero())
	transitionDaySet := transitionSet && (rule.Transition.TransitionInDays > 0)
	errMsg := "Error in Transition/Expiration Date/days compatibility. Transition should happen before Expiration"
	if transitionDateSet && expiryDateSet {
		if rule.Expiration.ExpirationDate.Before(*(rule.Transition.TransitionDate)) {
			return errors.New(errMsg)
		}
	}
	if transitionDaySet && expiryDaySet {
		if rule.Transition.TransitionInDays >= rule.Expiration.ExpirationInDays {
			return errors.New(errMsg)
		}
	}
	return nil
}

// Rule(s) enforced by Amazon S3 standards.
func validateTranDays(rule lifecycleRule) error {
	transitionSet := (rule.Transition != nil)
	transitionDaySet := transitionSet && (rule.Transition.TransitionInDays > 0)
	if transitionDaySet && rule.Transition.TransitionInDays <= 30 && strings.ToLower(rule.Transition.StorageClass) == "standard_ia" {
		return errors.New("Transition Date/Days are less than or equal to 30 and Storage class is STANDARD_IA")
	}
	return nil
}

func checkIlmObject(rule lifecycleRule) error {
	var err error
	if err = validateTranExpDate(rule); err != nil {
		return err
	}
	if err = validateTranDays(rule); err != nil {
		return err
	}
	// More rules could be added, if needed.
	return nil
}

// Returns valid lifecycleTransition to be but in lifecycleRule
func getTransition(ctx *cli.Context) (lifecycleTransition, error) {
	var transition lifecycleTransition
	var err error
	var transitionDateArg string
	var transitionDate time.Time
	var transitionDay int
	storageClassArg := ctx.String(strings.ToLower(storageClassLabelKey))
	transitionDayCheck := ctx.String(strings.ToLower(transitionDaysLabelKey)) != ""
	transitionDateCheck := ctx.String(strings.ToLower(transitionDatesLabelKey)) != ""
	transitionNoneCheck := (!transitionDayCheck && !transitionDateCheck && storageClassArg == "")
	switch {
	case transitionNoneCheck:
		return lifecycleTransition{}, nil
	case transitionDateCheck:
		transitionDateArg = ctx.String(strings.ToLower(transitionDatesLabelKey))
		transitionDate, err = time.Parse(defaultILMDateFormat, transitionDateArg)
	case transitionDayCheck:
		transitionDateArg = ctx.String(strings.ToLower(transitionDaysLabelKey))
		transitionDay, err = strconv.Atoi(transitionDateArg)
	}
	storageClassArg = strings.ToUpper(storageClassArg) // Just-in-case the user has entered lower case.

	transitionArgCheck := (transitionDateArg != "" && storageClassArg != "")
	// Different kinds of errors
	switch {
	case !transitionArgCheck:
		console.Errorln("Error in Transition/Storage class argument specification.")
		return transition, errors.New("transition/storage class argument error")
	case err != nil:
		console.Errorln("Error in transition date/day(s) argument. Error:" + err.Error())
		return lifecycleTransition{}, err
	case transitionDayCheck || transitionDateCheck:
		transition.StorageClass = storageClassArg
		if transitionDayCheck {
			transition.TransitionInDays = transitionDay
		} else if transitionDateCheck {
			transition.TransitionDate = &transitionDate
		}
	}

	return transition, err
}

// Returns lifecycleExpiration to be included in lifecycleRule struct
func getExpiry(ctx *cli.Context) (lifecycleExpiration, error) {
	var expiry lifecycleExpiration
	var err error
	var expiryDate time.Time
	var expiryArg string
	switch {
	case ctx.String(strings.ToLower(expiryDatesLabelFlag)) != "":
		expiryArg = ctx.String(strings.ToLower(expiryDatesLabelFlag))
		expiryDate, err = time.Parse(defaultILMDateFormat, expiryArg)
		if err != nil || expiryDate.IsZero() {
			errStr := "Error in Expiration argument:" + expiryArg + " date conversion."
			if err != nil {
				errStr += " " + err.Error()
			}
			console.Errorln(errStr)
		} else {
			expiry.ExpirationDate = &expiryDate
		}
	case ctx.String(strings.ToLower(expiryDaysLabelFlag)) != "":
		expiryArg = ctx.String(strings.ToLower(expiryDaysLabelFlag))
		expiry.ExpirationInDays, err = strconv.Atoi(expiryArg)
		if err != nil {
			errStr := "Error in Expiration argument:" + expiryArg + " days conversion:" + err.Error()
			console.Errorln(errStr)
		}
	}

	return expiry, err
}

// Adds/Replaces a lifecycleRule in the lifecycleConfiguration structure.
// lifecycleConfiguration structure has the existing(if any) lifecycle configuration rules for the bucket.
func getILMRuleFromUserValues(ctx *cli.Context, lfcInfoP *lifecycleConfiguration) (lifecycleRule, error) {
	var expiry lifecycleExpiration
	var transition lifecycleTransition
	var err error
	var newRule lifecycleRule
	var ilmTagKVList []tagFilter

	if lfcInfoP == nil {
		return lifecycleRule{}, nil
	}

	ilmID := ctx.String(strings.ToLower(idLabel))
	ilmPrefix := ctx.String(strings.ToLower(prefixLabel))
	if ilmID == "" {
		console.Errorln("Lifecycle configuration rule cannot added without ID")
		return lifecycleRule{}, errors.New("lifecycle configuration rule cannot added without ID")
	}
	ilmStatus := statusLabelKey
	if ilmDisabled := ctx.Bool(strings.ToLower(statusDisabledLabel)); ilmDisabled {
		ilmStatus = statusDisabledLabel
	}
	tagValue := ctx.StringSlice(strings.ToLower(tagLabel))
	if ilmTagKVList, err = extractILMTags(tagValue); err != nil {
		console.Errorln("Error in Tags argument.")
		return lifecycleRule{}, err

	}

	if expiry, err = getExpiry(ctx); err != nil {
		return lifecycleRule{}, err
	}

	if transition, err = getTransition(ctx); err != nil {
		return lifecycleRule{}, err
	}
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

	newRule = lifecycleRule{
		ID:         ilmID,
		RuleFilter: &filter,
		Status:     ilmStatus,
		Expiration: expP,
		Transition: transP,
	}
	idx := 0
	ruleFound := false
	for range lfcInfoP.Rules {
		rule := lfcInfoP.Rules[idx]
		if rule.ID == ilmID {
			ruleFound = true
			lfcInfoP.Rules[idx] = newRule
		}
		idx++
	}
	if !ruleFound {
		lfcInfoP.Rules = append(lfcInfoP.Rules, newRule)
	}
	return newRule, nil
}

// Calls SetBucketLifecycle with the XML representation of lifecycleConfiguration type.
func setILM(urlStr string, ilm lifecycleConfiguration) error {
	var err error
	var pErr *probe.Error
	var cBfr []byte
	var bkt string
	var api *minio.Client
	showILMErr := func(errStr string) {
		if pErr != nil {
			err = pErr.ToGoError()
		}
		if err != nil {
			errStr += ". " + err.Error()
			console.Errorln(errStr)
		}
	}
	var errStr string
	if api, pErr = getMinioClient(urlStr); pErr != nil {
		errStr = "Unable to get client to set lifecycle from url: " + urlStr
		showILMErr(errStr)
		return errors.New(errStr)
	}

	if bkt = getBucketNameFromURL(urlStr); bkt == "" || len(bkt) == 0 {
		errStr = "Error bucket name " + urlStr
		showILMErr(errStr)
		return errors.New(errStr)
	}

	if cBfr, err = xml.Marshal(ilm); err != nil {
		showILMErr("XML Conversion error.")
		return err
	}
	bktLCStr := string(cBfr)

	if err = api.SetBucketLifecycle(bkt, bktLCStr); err != nil {
		errStr = "Unable to set lifecycle for bucket: " + bkt + ". Target: " + urlStr
		showILMErr(errStr)
		return errors.New(errStr)
	}

	return nil
}

func mainLifecycleAdd(ctx *cli.Context) error {
	checkIlmAddSyntax(ctx)
	setColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	var err error
	var pErr *probe.Error
	var lfcInfo lifecycleConfiguration
	var lfcInfoXML string
	var rule lifecycleRule
	if lfcInfoXML, pErr = getIlmInfo(objectURL); pErr != nil {
		console.Errorln("Error generating lifecycle contents in XML: " + pErr.ToGoError().Error() + " Target:" + objectURL)
		return pErr.ToGoError()
	}
	if lfcInfoXML != "" {
		if err = xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo); err != nil {
			console.Errorln("Error assigning existing lifecycle configuration in XML: " + err.Error() + " Target:" + objectURL)
			return err
		}
	}

	if rule, err = getILMRuleFromUserValues(ctx, &lfcInfo); err != nil {
		console.Errorln("Error in new rule:" + err.Error())
		return err
	}
	if err = checkIlmObject(rule); err != nil {
		console.Errorln("Invalid lifecycle configuration rule argument. " + err.Error())
		return err
	}
	if err = setILM(objectURL, lfcInfo); err != nil {
		failureStr := "Failure. Lifecycle configuration add rule with ID `" + rule.ID + "` Error: " + err.Error()
		console.Println(console.Colorize(fieldThemeResultFailure, failureStr))
		return err
	}
	successStr := "Success. Lifecycle configuration rule added with ID `" + rule.ID + "`."
	console.Println(console.Colorize(fieldThemeResultSuccess, successStr))
	return nil
}
