// Copyright (c) 2015-2022 MinIO, Inc.
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

// Used in tags. Ex: --tags "key1=value1&key2=value2&key3=value3"
const (
	tagSeperator    string = "&"
	keyValSeperator string = "="
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
	transitionSet := !rule.Transition.IsNull()
	transitionDateSet := transitionSet && !rule.Transition.IsDateNull()
	if transitionDateSet && expiryDateSet {
		if rule.Expiration.Date.Before(rule.Transition.Date.Time) {
			return errors.New("transition should apply before expiration")
		}
	}
	if transitionDateSet && rule.Transition.StorageClass == "" {
		return errors.New("missing transition storage-class")
	}
	return nil
}

func validateTranDays(rule lifecycle.Rule) error {
	if rule.Transition.Days < 0 {
		return errors.New("number of days to transition can't be negative")
	}
	if rule.Transition.Days < 30 && strings.ToLower(rule.Transition.StorageClass) == "standard_ia" {
		return errors.New("number of days to transition should be >= 30 with STANDARD_IA storage-class")
	}
	return nil
}

// Amazon S3 requires a minimum of one action for a rule to be added.
func validateRuleAction(rule lifecycle.Rule) error {
	expirySet := !rule.Expiration.IsNull()
	transitionSet := !rule.Transition.IsNull()
	noncurrentExpirySet := !rule.NoncurrentVersionExpiration.IsDaysNull()
	newerNoncurrentVersionsExpiry := rule.NoncurrentVersionExpiration.NewerNoncurrentVersions > 0
	noncurrentTransitionSet := rule.NoncurrentVersionTransition.StorageClass != ""
	newerNoncurrentVersionsTransition := rule.NoncurrentVersionTransition.NewerNoncurrentVersions > 0
	if !expirySet && !transitionSet && !noncurrentExpirySet && !noncurrentTransitionSet && !newerNoncurrentVersionsExpiry && !newerNoncurrentVersionsTransition {
		return errors.New("at least one of Expiry, Transition, NoncurrentExpiry, NoncurrentVersionTransition actions should be specified in a rule")
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
		return errors.New("only one parameter under Expiration can be specified")
	}
	return nil
}

func validateTransition(rule lifecycle.Rule) error {
	if !rule.Transition.IsDaysNull() && !rule.Transition.IsDateNull() {
		return errors.New("only one parameter under Transition can be specified")
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
	if days > 0 && storageClass == "" {
		return errors.New("both NoncurrentVersionTransition NoncurrentDays and StorageClass need to be specified")
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
		e = errors.New("expiry date falls before or on today's date")
	} else if transitionSet && transitionDateSet && rule.Transition.Date.Before(currentTime) {
		e = errors.New("transition date falls before or on today's date")
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
	if e := validateTransition(rule); e != nil {
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

func parseTransitionDate(transitionDateStr string) (lifecycle.ExpirationDate, *probe.Error) {
	transitionDate, e := time.Parse(defaultILMDateFormat, transitionDateStr)
	if e != nil {
		return lifecycle.ExpirationDate{}, probe.NewError(e)
	}
	return lifecycle.ExpirationDate{Time: transitionDate}, nil
}

func parseTransitionDays(transitionDaysStr string) (lifecycle.ExpirationDays, *probe.Error) {
	transitionDays, e := strconv.Atoi(transitionDaysStr)
	if e != nil {
		return lifecycle.ExpirationDays(0), probe.NewError(e)
	}
	return lifecycle.ExpirationDays(transitionDays), nil
}

// Returns valid lifecycleTransition to be included in lifecycleRule
func parseTransition(storageClass, transitionDateStr, transitionDaysStr *string) (lifecycle.Transition, *probe.Error) {
	var transition lifecycle.Transition
	if transitionDateStr != nil {
		transitionDate, err := parseTransitionDate(*transitionDateStr)
		if err != nil {
			return lifecycle.Transition{}, err
		}
		transition.Date = transitionDate
	}
	if transitionDaysStr != nil {
		transitionDays, err := parseTransitionDays(*transitionDaysStr)
		if err != nil {
			return lifecycle.Transition{}, err
		}
		transition.Days = transitionDays
	}
	if storageClass != nil {
		transition.StorageClass = *storageClass
	}
	return transition, nil
}

func parseExpiryDate(expiryDateStr string) (lifecycle.ExpirationDate, *probe.Error) {
	date, e := time.Parse(defaultILMDateFormat, expiryDateStr)
	if e != nil {
		return lifecycle.ExpirationDate{}, probe.NewError(e)
	}
	if date.IsZero() {
		return lifecycle.ExpirationDate{}, probe.NewError(errors.New("expiration date cannot be set to zero"))
	}
	return lifecycle.ExpirationDate{Time: date}, nil
}

func parseExpiryDays(expiryDayStr string) (lifecycle.ExpirationDays, *probe.Error) {
	days, e := strconv.Atoi(expiryDayStr)
	if e != nil {
		return lifecycle.ExpirationDays(0), probe.NewError(e)
	}
	if days == 0 {
		return lifecycle.ExpirationDays(0), probe.NewError(errors.New("expiration days cannot be set to zero"))
	}
	return lifecycle.ExpirationDays(days), nil
}

// Returns lifecycleExpiration to be included in lifecycleRule
func parseExpiry(expiryDate, expiryDays *string, expiredDeleteMarker, expiredObjectAllVersions *bool) (lfcExp lifecycle.Expiration, err *probe.Error) {
	if expiryDate != nil {
		date, err := parseExpiryDate(*expiryDate)
		if err != nil {
			return lifecycle.Expiration{}, err
		}
		lfcExp.Date = date
	}

	if expiryDays != nil {
		days, err := parseExpiryDays(*expiryDays)
		if err != nil {
			return lifecycle.Expiration{}, err
		}
		lfcExp.Days = days
	}

	if expiredDeleteMarker != nil {
		lfcExp.DeleteMarker = lifecycle.ExpireDeleteMarker(*expiredDeleteMarker)
	}

	if expiredObjectAllVersions != nil {
		lfcExp.DeleteAll = lifecycle.ExpirationBoolean(*expiredObjectAllVersions)
	}

	return lfcExp, nil
}
