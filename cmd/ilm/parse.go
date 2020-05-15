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
)

// Extracts the tags provided by user. The tagfilter array will be put in lifecycleRule structure.
func extractILMTags(tagLabelVal string) []LifecycleTag {
	var ilmTagKVList []LifecycleTag
	for _, tag := range strings.Split(tagLabelVal, tagSeperator) {
		if tag == "" {
			// split returns empty for empty tagLabelVal, skip it.
			continue
		}
		lfcTag := LifecycleTag{}
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

func validateTranDays(rule LifecycleRule) error {
	transitionSet := (rule.Transition != nil)
	transitionDaySet := transitionSet && (rule.Transition.TransitionInDays > 0)
	if transitionDaySet && rule.Transition.TransitionInDays < 30 && strings.ToLower(rule.Transition.StorageClass) == "standard_ia" {
		return errors.New("Transition Date/Days are less than or equal to 30 and Storage class is STANDARD_IA")
	}
	return nil
}

// Amazon S3 requires a minimum of one action for a rule to be added.
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

// Check S3 compatibility for the new rule and some other basic checks.
func validateILMRule(rule LifecycleRule) *probe.Error {
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
func parseTransition(storageClass, transitionDateStr, transitionDayStr string) (transition LifecycleTransition, err *probe.Error) {
	storageClass = strings.ToUpper(storageClass) // Just-in-case the user has entered lower case characters.
	if storageClass != "" {
		if transitionDateStr != "" {
			transitionDate, e := time.Parse(defaultILMDateFormat, transitionDateStr)
			if e != nil {
				return LifecycleTransition{}, probe.NewError(e)
			}
			transition.TransitionDate = &transitionDate
		} else if transitionDayStr != "" {
			transitionDay, e := strconv.Atoi(transitionDayStr)
			if e != nil {
				return LifecycleTransition{}, probe.NewError(e)
			}
			transition.TransitionInDays = transitionDay
		} else {
			return LifecycleTransition{}, probe.NewError(errors.New("if storage class is set a valid transitionDate or transitionDay must be set"))
		}
	}

	return transition, nil
}

// Returns lifecycleExpiration to be included in lifecycleRule
func parseExpiry(expiryDateStr, expiryDayStr string) (lfcExp LifecycleExpiration, err *probe.Error) {
	if expiryDateStr != "" {
		date, e := time.Parse(defaultILMDateFormat, expiryDateStr)
		if e != nil {
			return LifecycleExpiration{}, probe.NewError(e)
		}
		if date.IsZero() {
			return LifecycleExpiration{}, probe.NewError(errors.New("expiration date cannot be set to zero"))
		}
		lfcExp.ExpirationDate = &date
	}

	if expiryDayStr != "" {
		days, e := strconv.Atoi(expiryDayStr)
		if e != nil {
			return lfcExp, probe.NewError(e)
		}
		if days == 0 {
			return LifecycleExpiration{}, probe.NewError(errors.New("expiration days cannot be set to zero"))
		}
		lfcExp.ExpirationInDays = days
	}

	return lfcExp, nil
}
