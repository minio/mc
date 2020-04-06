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

package ilm

import (
	"encoding/xml"
	"errors"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
)

const defaultILMDateFormat string = "2006-01-02"

func getCenterAligned(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	if lblLth > 1 && lblLth%2 != 0 {
		lblLth++
	} else if lblLth == 1 {
		lblLth = 2
	}
	length := (float64(maxLen - lblLth)) / float64(2)
	rptLth := (int)(math.Floor(length / float64(len(toPadWith))))
	leftRptLth := rptLth
	rightRptLth := rptLth
	if rptLth <= 0 {
		leftRptLth = 1
		rightRptLth = 0
	}
	output := strings.Repeat(toPadWith, leftRptLth) + label + strings.Repeat(toPadWith, rightRptLth)
	return output
}

func splitStr(path, sep string, n int) []string {
	splits := strings.SplitN(path, sep, n)
	for i := n - len(splits); i > 0; i-- {
		splits = append(splits, "")
	}
	return splits
}

func getLeftAlgined(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	length := maxLen - lblLth
	if length <= 0 {
		return label
	}
	output := strings.Repeat(toPadWith, 1) + label + strings.Repeat(toPadWith, length-1)
	return output
}

func getRightAligned(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	length := maxLen - lblLth
	if length <= 0 {
		return label
	}
	output := strings.Repeat(toPadWith, length) + label
	return output
}

// RemoveILMRule - Remove the ILM rule (with ilmID) from the configuration in XML that is provided.
func RemoveILMRule(lfcInfoXML string, ilmID string) (string, error) {
	var lfcInfo LifecycleConfiguration
	var err error
	if lfcInfoXML != "" {
		if err = xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo); err != nil {
			return "", err
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
		if ruleFound && len(lfcInfo.Rules) > 1 {
			lfcInfo.Rules = append(lfcInfo.Rules[:foundIdx], lfcInfo.Rules[foundIdx+1:]...)
		} else if ruleFound && len(lfcInfo.Rules) <= 1 {
			return "", nil // Only rule. Remove all.
		}
		if ruleFound {
			var bytes []byte
			if bytes, err = xml.Marshal(lfcInfo); err != nil {
				return "", err
			}
			return string(bytes), nil
		}
	}
	return "", errors.New("Rule not found")
}

// GetILMJSON Get ILM in JSON format
func GetILMJSON(ilmXML string) (string, error) {
	var err error
	var ilmInfo LifecycleConfiguration
	if ilmXML == "" {
		return ilmXML, errors.New("Empty lifecycle configuration")
	}
	if err = xml.Unmarshal([]byte(ilmXML), &ilmInfo); err != nil {
		return "", err
	}
	return ilmInfo.JSON(), nil
}

// GetILMConfig Get ILM configuration populated with values
func GetILMConfig(ilmXML string) (ilmInfo LifecycleConfiguration, err error) {
	if ilmXML == "" {
		return LifecycleConfiguration{}, errors.New("Empty lifecycle configuration")
	}
	if err = xml.Unmarshal([]byte(ilmXML), &ilmInfo); err != nil {
		return LifecycleConfiguration{}, err
	}
	return ilmInfo, nil
}

// ReadILMConfigJSON read from stdin and set to bucket
func ReadILMConfigJSON(urlStr string) (string, error) {
	// User is expected to enter the lifecycleConfiguration instance contents in JSON format
	var ilmInfo LifecycleConfiguration
	var bytes []byte
	var err error
	// Consume json from STDIN
	dec := json.NewDecoder(os.Stdin)
	for {
		err = dec.Decode(&ilmInfo)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	if len(ilmInfo.Rules) == 0 {
		err = errors.New("Empty configuration")
		return "", err
	}

	if bytes, err = xml.Marshal(ilmInfo); err != nil {
		return "", nil
	}
	return string(bytes), nil
}

func getBucketILMXML(ilm LifecycleConfiguration) (string, error) {
	var err error
	var cBfr []byte
	setILMErr := func(errStr string) string {
		var setErrStr string
		if err != nil {
			setErrStr = errStr + ". " + err.Error()
		}
		return setErrStr
	}
	var errStr string
	if cBfr, err = xml.Marshal(ilm); err != nil {
		errStr = setILMErr("XML Conversion error.")
		return "", errors.New(errStr)
	}
	bktLCStr := string(cBfr)

	return bktLCStr, nil
}

// Adds/Replaces a lifecycleRule in the lifecycleConfiguration structure.
// lifecycleConfiguration structure has the existing(if any) lifecycle configuration rules for the bucket.
func getILMRuleFromUserValues(ctx *cli.Context, lfcInfoP *LifecycleConfiguration) (LifecycleRule, error) {
	var ilmExpiry LifecycleExpiration
	var ilmTransition LifecycleTransition
	var err error
	var newRule LifecycleRule
	var ilmTagKVList []LifecycleTag

	if lfcInfoP == nil {
		return LifecycleRule{}, nil
	}

	ilmID := ctx.String(strings.ToLower(idLabel))
	ilmPrefix := ctx.String(strings.ToLower(prefixLabel))
	if ilmID == "" {
		return LifecycleRule{}, errors.New("lifecycle configuration rule cannot be added without ID")
	}
	ilmStatus := statusLabelKey
	if ilmDisabled := ctx.Bool(strings.ToLower(statusDisabledLabel)); ilmDisabled {
		ilmStatus = statusDisabledLabel
	}
	ilmTag := ctx.String(strings.ToLower(tagLabel))
	if ilmTagKVList, err = extractILMTags(ilmTag); err != nil {
		return LifecycleRule{}, err

	}

	if ilmExpiry, err = getExpiry(ctx); err != nil {
		return LifecycleRule{}, err
	}

	if ilmTransition, err = getTransition(ctx); err != nil {
		return LifecycleRule{}, err
	}
	var andVal LifecycleAndOperator
	andVal.Tags = ilmTagKVList
	filter := LifecycleRuleFilter{Prefix: ilmPrefix}
	if len(andVal.Tags) > 0 {
		filter.And = &andVal
		filter.And.Prefix = filter.Prefix
		filter.Prefix = ""
	}
	var expP *LifecycleExpiration
	var transP *LifecycleTransition
	if (ilmTransition.TransitionDate != nil &&
		!ilmTransition.TransitionDate.IsZero()) || ilmTransition.TransitionInDays > 0 {
		transP = &ilmTransition
	}
	if (ilmExpiry.ExpirationDate != nil &&
		!ilmExpiry.ExpirationDate.IsZero()) || ilmExpiry.ExpirationInDays > 0 {
		expP = &ilmExpiry
	}

	newRule = LifecycleRule{
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

// GetILMRuleToSet Get the rule in XML and set it to the object URL.
func GetILMRuleToSet(ctx *cli.Context, lfcInfoXML string) (string, error) {
	var err error
	var lfcInfo LifecycleConfiguration
	var rule LifecycleRule
	var bktILM string
	if lfcInfoXML != "" {
		if err = xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo); err != nil {
			errStr := "XML conversion of lifecycle configuration, " + err.Error()
			return "", errors.New(errStr)
		}
	}

	if rule, err = getILMRuleFromUserValues(ctx, &lfcInfo); err != nil {
		errStr := err.Error() + ". Error getting input values for new rule"
		return "", errors.New(errStr)
	}
	if err = checkILMRule(rule); err != nil {
		errStr := err.Error() + ". Invalid lifecycle configuration rule"
		return "", errors.New(errStr)
	}
	if bktILM, err = getBucketILMXML(lfcInfo); err != nil {
		failureStr := "`" + rule.ID + "` Error: " + err.Error()
		return "", errors.New(failureStr)
	}
	return bktILM, nil
}

// Extracts the tags provided by user. The tagfilter array will be put in lifecycleRule structure.
func extractILMTags(tagLabelVal string) ([]LifecycleTag, error) {
	if tagLabelVal == "" {
		return nil, nil
	}
	var ilmTagKVList []LifecycleTag
	tagValues := strings.Split(tagLabelVal, tagSeperator)
	for tagIdx, tag := range tagValues {
		var key, val string
		if !strings.Contains(tag, keyValSeperator) {
			key = tag
			val = ""
		} else {
			key = splitStr(tag, keyValSeperator, 2)[0]
			val = splitStr(tag, keyValSeperator, 2)[1]
		}
		if key != "" {
			ilmTagKVList = append(ilmTagKVList, LifecycleTag{Key: key, Value: val})
		} else {
			return nil, errors.New("Failed extracting tag argument number " + strconv.Itoa(tagIdx+1) + " " + tag + " in lifecycle configuration rule")
		}
	}
	return ilmTagKVList, nil
}

// Some of these rules are enforced by Amazon S3 standards.
// For example: Transition has to happen before Expiry.
// Storage class must be specified if transition date/days is provided.
func validateTranExpDate(rule LifecycleRule) error {
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
func validateTranDays(rule LifecycleRule) error {
	transitionSet := (rule.Transition != nil)
	transitionDaySet := transitionSet && (rule.Transition.TransitionInDays > 0)
	if transitionDaySet && rule.Transition.TransitionInDays < 30 && strings.ToLower(rule.Transition.StorageClass) == "standard_ia" {
		return errors.New("Transition Date/Days are less than or equal to 30 and Storage class is STANDARD_IA")
	}
	return nil
}

// Amazon S3 requires atleast one action for a rule to be added.
func validateRuleAction(rule LifecycleRule) error {
	expirySet := (rule.Expiration != nil)
	transitionSet := (rule.Transition != nil)
	if !expirySet && !transitionSet {
		errMsg := "At least one action (Expiry or Transition) needs to be specified in a rule."
		return errors.New(errMsg)
	}
	return nil
}

// Check if any date is before than cur date
func validateTranExpCurdate(rule LifecycleRule) error {
	var err error
	expirySet := (rule.Expiration != nil)
	transitionSet := (rule.Transition != nil)
	transitionDateSet := transitionSet && (rule.Transition.TransitionDate != nil && !rule.Transition.TransitionDate.IsZero())
	expiryDateSet := expirySet && rule.Expiration.ExpirationDate != nil && !rule.Expiration.ExpirationDate.IsZero()
	currentTime := time.Now()
	curTimeStr := currentTime.Format(defaultILMDateFormat)
	currentTime, err = time.Parse(defaultILMDateFormat, curTimeStr)
	if err != nil {
		return err
	}
	if expirySet && expiryDateSet && rule.Expiration.ExpirationDate.Before(currentTime) {
		err = errors.New("Expiry date falls before or on today's date")
	} else if transitionSet && transitionDateSet && rule.Transition.TransitionDate.Before(currentTime) {
		err = errors.New("Transition date falls before or on today's date")
	}
	return err
}

func checkILMRule(rule LifecycleRule) error {
	var err error

	if err = validateRuleAction(rule); err != nil {
		return err
	}
	if err = validateTranExpCurdate(rule); err != nil {
		return err
	}
	if err = validateTranExpDate(rule); err != nil {
		return err
	}
	if err = validateTranDays(rule); err != nil {
		return err
	}
	return nil
}

// Returns valid lifecycleTransition to be but in lifecycleRule
func getTransition(ctx *cli.Context) (LifecycleTransition, error) {
	var transition LifecycleTransition
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
		return LifecycleTransition{}, nil
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
		return transition, errors.New("transition/storage class argument error")
	case err != nil:
		return LifecycleTransition{}, err
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
func getExpiry(ctx *cli.Context) (expiry LifecycleExpiration, err error) {
	var expiryDate time.Time
	var expiryArg string
	switch {
	case ctx.String(strings.ToLower(expiryDatesLabelFlag)) != "":
		expiryArg = ctx.String(strings.ToLower(expiryDatesLabelFlag))
		if expiryDate, err = time.Parse(defaultILMDateFormat, expiryArg); err != nil || expiryDate.IsZero() {
			errStr := "Error in Expiration argument " + expiryArg + " date conversion."
			if err != nil {
				errStr += err.Error()
			}
			err = errors.New(errStr)
		} else {
			expiry.ExpirationDate = &expiryDate
		}
	case ctx.String(strings.ToLower(expiryDaysLabelFlag)) != "":
		expiryArg = ctx.String(strings.ToLower(expiryDaysLabelFlag))
		if expiry.ExpirationInDays, err = strconv.Atoi(expiryArg); err != nil {
			errStr := "Error in Expiration argument " + expiryArg + ". " + err.Error()
			err = errors.New(errStr)
		}
	}

	return expiry, err
}
