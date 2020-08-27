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
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

// Extracts the tags provided by user. The tagfilter array will be put in lifecycleRule structure.
func extractILMTags(tagLabelVal string) []lifecycle.Tag {
	var ilmTagKVList []lifecycle.Tag
	for _, tag := range strings.Split(tagLabelVal, tagSeperator) {
		if tag == "" {
			// split returns empty for empty tagLabelVal, skip it.
			continue
		}
		lfcTag := lifecycle.Tag{}
		kvs := strings.SplitN(tag, keyValSeperator, 2)
		if len(kvs) == 2 {
			lfcTag.Key = kvs[0]
			lfcTag.Value = kvs[1]
		} else {
			lfcTag.Key = kvs[0]
		}
		ilmTagKVList = append(ilmTagKVList, lfcTag)
	}
	return ilmTagKVList
}

// Some of these rules are enforced by Amazon S3 standards.
// For example: Transition has to happen before Expiry.
// Storage class must be specified if transition date/days is provided.
func validateTranExpDate(rule lifecycle.Rule) error {
	expiryDateSet := !rule.Expiration.IsDateNull()
	expiryDaySet := !rule.Expiration.IsDaysNull()

	transitionSet := !rule.Transition.IsNull()
	transitionDateSet := transitionSet && !rule.Transition.IsDateNull()
	transitionDaySet := transitionSet && !rule.Transition.IsDaysNull()
	errMsg := "Error in Transition/Expiration Date/days compatibility. Transition should happen before Expiration"
	if transitionDateSet && expiryDateSet {
		if rule.Expiration.Date.Before(rule.Transition.Date.Time) {
			return errors.New(errMsg)
		}
	}
	if transitionDaySet && expiryDaySet {
		if rule.Transition.Days >= rule.Expiration.Days {
			return errors.New(errMsg)
		}
	}
	if transitionDateSet && rule.Transition.StorageClass == "" {
		return errors.New("if storage class is set a valid transitionDate or transitionDay must be set")
	}
	if transitionDaySet && rule.Transition.StorageClass == "" {
		return errors.New("if storage class is set a valid transitionDate or transitionDay must be set")
	}
	return nil
}

func validateTranDays(rule lifecycle.Rule) error {
	transitionSet := !rule.Transition.IsNull()
	transitionDaySet := transitionSet && !rule.Transition.IsDaysNull()
	if transitionDaySet && rule.Transition.Days < 30 && strings.ToLower(rule.Transition.StorageClass) == "standard_ia" {
		return errors.New("Transition Date/Days are less than or equal to 30 when Storage class is STANDARD_IA")
	}
	return nil
}

// Amazon S3 requires a minimum of one action for a rule to be added.
func validateRuleAction(rule lifecycle.Rule) error {
	expirySet := !rule.Expiration.IsNull()
	transitionSet := !rule.Transition.IsNull()
	if !expirySet && !transitionSet {
		errMsg := "At least one action (Expiry or Transition) needs to be specified in a rule."
		return errors.New(errMsg)
	}
	return nil
}

// Check if any date is before than cur date
func validateTranExpCurdate(rule lifecycle.Rule) error {
	var e error
	expirySet := !rule.Expiration.IsNull()
	transitionSet := !rule.Transition.IsNull()
	transitionDateSet := transitionSet && !rule.Transition.IsDateNull()
	expiryDateSet := expirySet && !rule.Expiration.IsDateNull()
	currentTime := time.Now()
	curTimeStr := currentTime.Format(defaultILMDateFormat)
	currentTime, e = time.Parse(defaultILMDateFormat, curTimeStr)
	if e != nil {
		return e
	}
	if expirySet && expiryDateSet && rule.Expiration.Date.Before(currentTime) {
		e = errors.New("Expiry date falls before or on today's date")
	} else if transitionSet && transitionDateSet && rule.Transition.Date.Before(currentTime) {
		e = errors.New("Transition date falls before or on today's date")
	}
	return e
}

// Check S3 compatibility for the new rule and some other basic checks.
func validateILMRule(rule lifecycle.Rule) *probe.Error {
	if e := validateRuleAction(rule); e != nil {
		return probe.NewError(e)
	}
	if e := validateTranExpCurdate(rule); e != nil {
		return probe.NewError(e)
	}
	if e := validateTranExpDate(rule); e != nil {
		return probe.NewError(e)
	}
	if e := validateTranDays(rule); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// Returns valid lifecycleTransition to be included in lifecycleRule
func parseTransition(storageClass, transitionDateStr, transitionDayStr string) (transition lifecycle.Transition, err *probe.Error) {
	storageClass = strings.ToUpper(storageClass) // Just-in-case the user has entered lower case characters.
	if transitionDateStr != "" {
		transitionDate, e := time.Parse(defaultILMDateFormat, transitionDateStr)
		if e != nil {
			return lifecycle.Transition{}, probe.NewError(e)
		}
		transition.Date = lifecycle.ExpirationDate{Time: transitionDate}
	} else if transitionDayStr != "" {
		transitionDay, e := strconv.Atoi(transitionDayStr)
		if e != nil {
			return lifecycle.Transition{}, probe.NewError(e)
		}
		transition.Days = lifecycle.ExpirationDays(transitionDay)
	}
	transition.StorageClass = storageClass
	return transition, nil
}

// Returns lifecycleExpiration to be included in lifecycleRule
func parseExpiry(expiryDateStr, expiryDayStr string) (lfcExp lifecycle.Expiration, err *probe.Error) {
	if expiryDateStr != "" {
		date, e := time.Parse(defaultILMDateFormat, expiryDateStr)
		if e != nil {
			return lifecycle.Expiration{}, probe.NewError(e)
		}
		if date.IsZero() {
			return lifecycle.Expiration{}, probe.NewError(errors.New("expiration date cannot be set to zero"))
		}
		lfcExp.Date = lifecycle.ExpirationDate{Time: date}
	}

	if expiryDayStr != "" {
		days, e := strconv.Atoi(expiryDayStr)
		if e != nil {
			return lfcExp, probe.NewError(e)
		}
		if days == 0 {
			return lifecycle.Expiration{}, probe.NewError(errors.New("expiration days cannot be set to zero"))
		}
		lfcExp.Days = lifecycle.ExpirationDays(days)
	}

	return lfcExp, nil
}
