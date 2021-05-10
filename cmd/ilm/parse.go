// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
		return errors.New("if transitionDate or transitionDay is set, a valid storage class must be set")
	}
	if transitionDaySet && rule.Transition.StorageClass == "" {
		return errors.New("if transitionDate or transitionDay is set, a valid storage class must be set")
	}
	if rule.Transition.StorageClass != "" && (!transitionDateSet && !transitionDaySet) {
		return errors.New("if storage class is set, transitionDate or transitionDay must be set")
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
	noncurrentExpirySet := !rule.NoncurrentVersionExpiration.IsDaysNull()
	noncurrentTransitionSet := !rule.NoncurrentVersionTransition.IsDaysNull()
	if !expirySet && !transitionSet && !noncurrentExpirySet && !noncurrentTransitionSet {
		errMsg := "At least one action (Expiry, Transition, NoncurrentExpiry or NoncurrentTransition) needs to be specified in a rule."
		return errors.New(errMsg)
	}
	return nil
}

func validateExpiration(rule lifecycle.Rule) error {
	var i int
	if !rule.Expiration.IsDaysNull() {
		i++
	}
	if !rule.Expiration.IsDateNull() {
		i++
	}
	if rule.Expiration.IsDeleteMarkerExpirationEnabled() {
		i++
	}
	if i > 1 {
		return errors.New("Only one parameter under Expiration can be specified")
	}
	return nil
}

func validateNoncurrentExpiration(rule lifecycle.Rule) error {
	days := rule.NoncurrentVersionExpiration.NoncurrentDays
	if days < 0 {
		return errors.New("NoncurrentVersionExpiration.NoncurrentDays is not a positive integer")
	}
	return nil
}

func validateNoncurrentTransition(rule lifecycle.Rule) error {
	days := rule.NoncurrentVersionTransition.NoncurrentDays
	storageClass := rule.NoncurrentVersionTransition.StorageClass
	if days < 0 {
		return errors.New("NoncurrentVersionTransition.NoncurrentDays is not a positive integer")
	}
	if days > 0 && storageClass == "" || days == 0 && storageClass != "" {
		return errors.New("Both NoncurrentVersionTransition NoncurrentDays and StorageClass need to be specified")
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
	if e := validateExpiration(rule); e != nil {
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
	if e := validateNoncurrentExpiration(rule); e != nil {
		return probe.NewError(e)
	}
	if e := validateNoncurrentTransition(rule); e != nil {
		return probe.NewError(e)
	}

	return nil
}

// Returns valid lifecycleTransition to be included in lifecycleRule
func parseTransition(storageClass, transitionDateStr, transitionDayStr string) (transition lifecycle.Transition, err *probe.Error) {
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
func parseExpiry(expiryDateStr, expiryDayStr string, expiredDeleteMarker bool) (lfcExp lifecycle.Expiration, err *probe.Error) {
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

	if expiredDeleteMarker {
		lfcExp.DeleteMarker = true
	}

	return lfcExp, nil
}
