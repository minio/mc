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
	"fmt"
	"math"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/rs/xid"
)

const defaultILMDateFormat string = "2006-01-02"

// Align text in label to center, pad with spaces on either sides.
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

// Align text in label to left, pad with spaces.
func getLeftAligned(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	length := maxLen - lblLth
	if length <= 0 {
		return label
	}
	output := strings.Repeat(toPadWith, 1) + label + strings.Repeat(toPadWith, length-1)
	return output
}

// Align text in label to right, pad with spaces.
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
func RemoveILMRule(lfcCfg *lifecycle.Configuration, ilmID string) (*lifecycle.Configuration, *probe.Error) {
	if lfcCfg == nil {
		return lfcCfg, probe.NewError(fmt.Errorf("lifecycle configuration not set"))
	}
	if len(lfcCfg.Rules) == 0 {
		return lfcCfg, probe.NewError(fmt.Errorf("lifecycle configuration not set"))
	}
	n := 0
	for _, rule := range lfcCfg.Rules {
		if rule.ID != ilmID {
			lfcCfg.Rules[n] = rule
			n++
		}
	}
	if n == len(lfcCfg.Rules) && len(lfcCfg.Rules) > 0 {
		// if there was no filtering then rules will be of same length, means we didn't find
		// our ilm id return an error here.
		return lfcCfg, probe.NewError(fmt.Errorf("lifecycle rule for id '%s' not found", ilmID))
	}
	lfcCfg.Rules = lfcCfg.Rules[:n]
	return lfcCfg, nil
}

// LifecycleOptions is structure to encapsulate
type LifecycleOptions struct {
	ID                string
	Prefix            string
	Status            bool
	IsTagsSet         bool
	IsStorageClassSet bool
	Tags              string
	ExpiryDate        string
	ExpiryDays        string
	TransitionDate    string
	TransitionDays    string
	StorageClass      string

	ExpiredObjectDeleteMarker               bool
	NoncurrentVersionExpirationDays         int
	NoncurrentVersionTransitionDays         int
	NoncurrentVersionTransitionStorageClass string
}

// ToConfig create lifecycle.Configuration based on LifecycleOptions
func (opts LifecycleOptions) ToConfig(config *lifecycle.Configuration) (*lifecycle.Configuration, *probe.Error) {
	expiry, err := parseExpiry(opts.ExpiryDate, opts.ExpiryDays, opts.ExpiredObjectDeleteMarker)
	if err != nil {
		return nil, err.Trace(opts.ExpiryDate, opts.ExpiryDays)
	}

	transition, err := parseTransition(opts.StorageClass, opts.TransitionDate, opts.TransitionDays)
	if err != nil {
		return nil, err.Trace(opts.StorageClass, opts.TransitionDate, opts.TransitionDays)
	}
	andVal := lifecycle.And{
		Tags: extractILMTags(opts.Tags),
	}

	filter := lifecycle.Filter{Prefix: opts.Prefix}
	if len(andVal.Tags) > 0 {
		filter.And = andVal
		filter.And.Prefix = opts.Prefix
		filter.Prefix = ""
	}

	newRule := lifecycle.Rule{
		ID:         opts.ID,
		RuleFilter: filter,
		Status: func() string {
			if opts.Status {
				return "Enabled"
			}
			return "Disabled"
		}(),
		Expiration: expiry,
		Transition: transition,
		NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{
			NoncurrentDays: lifecycle.ExpirationDays(opts.NoncurrentVersionExpirationDays),
		},
		NoncurrentVersionTransition: lifecycle.NoncurrentVersionTransition{
			NoncurrentDays: lifecycle.ExpirationDays(opts.NoncurrentVersionTransitionDays),
			StorageClass:   opts.NoncurrentVersionTransitionStorageClass,
		},
	}

	ruleFound := false
	for i, rule := range config.Rules {
		if rule.ID != newRule.ID {
			continue
		}
		config.Rules[i] = applyRuleFields(newRule, config.Rules[i], opts)
		if err := validateILMRule(config.Rules[i]); err != nil {
			return nil, err.Trace(opts.ID)
		}
		ruleFound = true
		break
	}

	if !ruleFound {
		if err := validateILMRule(newRule); err != nil {
			return nil, err.Trace(opts.ID)
		}
		config.Rules = append(config.Rules, newRule)
	}
	return config, nil
}

// GetLifecycleOptions create LifeCycleOptions based on cli inputs
func GetLifecycleOptions(ctx *cli.Context) LifecycleOptions {
	var id string = ctx.String("id")
	if id == "" {
		id = xid.New().String()
	}
	// split the first arg i.e. path into alias, bucket and prefix
	result := strings.SplitN(ctx.Args().First(), "/", 3)
	// get the prefix from path
	var prefix string
	if len(result) > 2 {
		prefix = result[len(result)-1]
	}
	scSet := ctx.IsSet("storage-class")
	sc := strings.ToUpper(ctx.String("storage-class"))
	noncurrentSC := strings.ToUpper(ctx.String("noncurrentversion-transition-storage-class"))
	// for MinIO transition storage-class is same as label defined on
	// `mc admin bucket remote add --service ilm --label` command
	return LifecycleOptions{
		ID:                                      id,
		Prefix:                                  prefix,
		Status:                                  !ctx.Bool("disable"),
		IsTagsSet:                               ctx.IsSet("tags"),
		IsStorageClassSet:                       scSet,
		Tags:                                    ctx.String("tags"),
		ExpiryDate:                              ctx.String("expiry-date"),
		ExpiryDays:                              ctx.String("expiry-days"),
		TransitionDate:                          ctx.String("transition-date"),
		TransitionDays:                          ctx.String("transition-days"),
		StorageClass:                            sc,
		ExpiredObjectDeleteMarker:               ctx.Bool("expired-object-delete-marker"),
		NoncurrentVersionExpirationDays:         ctx.Int("noncurrentversion-expiration-days"),
		NoncurrentVersionTransitionDays:         ctx.Int("noncurrentversion-transition-days"),
		NoncurrentVersionTransitionStorageClass: noncurrentSC,
	}
}

// Applies non empty fields from src to dest Rule and return the dest Rule
func applyRuleFields(src lifecycle.Rule, dest lifecycle.Rule, opts LifecycleOptions) lifecycle.Rule {
	// since prefix is a part of command args, it is always present in the src rule and
	// it should be always set to the destination.
	dest.RuleFilter.Prefix = src.RuleFilter.Prefix

	// If src has tags, it should override the destination
	if len(src.RuleFilter.And.Tags) > 0 {
		dest.RuleFilter.And.Tags = src.RuleFilter.And.Tags
		dest.RuleFilter.And.Prefix = src.RuleFilter.And.Prefix
		dest.RuleFilter.Prefix = ""
	}

	if src.RuleFilter.And.Tags == nil {
		if opts.IsTagsSet {
			// If src tags is empty and isTagFlagSet then user provided the --tag flag with "" value
			// dest tags should be deleted
			dest.RuleFilter.And.Tags = []lifecycle.Tag{}
			dest.RuleFilter.And.Prefix = ""
			dest.RuleFilter.Prefix = src.RuleFilter.Prefix
		} else {
			if dest.RuleFilter.And.Tags != nil {
				// Update prefixes only
				dest.RuleFilter.And.Prefix = src.RuleFilter.Prefix
				dest.RuleFilter.Prefix = ""
			} else {
				dest.RuleFilter.Prefix = src.RuleFilter.Prefix
				dest.RuleFilter.And.Prefix = ""
			}
		}
	}

	// only one of expiration day, date or transition day, date is expected
	if !src.Expiration.IsDateNull() {
		dest.Expiration.Date = src.Expiration.Date
		// reset everything else
		dest.Expiration.Days = 0
		dest.Expiration.DeleteMarker = false
	} else if !src.Expiration.IsDaysNull() {
		dest.Expiration.Days = src.Expiration.Days
		// reset everything else
		dest.Expiration.Date = lifecycle.ExpirationDate{}
	} else if src.Expiration.IsDeleteMarkerExpirationEnabled() {
		dest.Expiration.DeleteMarker = true
		dest.Expiration.Days = 0
		dest.Expiration.Date = lifecycle.ExpirationDate{}
	}

	if !src.Transition.IsDateNull() {
		dest.Transition.Date = src.Transition.Date
		// reset everything else
		dest.Transition.Days = 0
	} else if !src.Transition.IsDaysNull() {
		dest.Transition.Days = src.Transition.Days
		// reset everything else
		dest.Transition.Date = lifecycle.ExpirationDate{}
	}

	if !src.NoncurrentVersionExpiration.IsDaysNull() {
		dest.NoncurrentVersionExpiration.NoncurrentDays = src.NoncurrentVersionExpiration.NoncurrentDays
	}

	if !src.NoncurrentVersionTransition.IsDaysNull() {
		dest.NoncurrentVersionTransition.NoncurrentDays = src.NoncurrentVersionTransition.NoncurrentDays
	}

	if src.NoncurrentVersionTransition.StorageClass != "" {
		dest.NoncurrentVersionTransition.StorageClass = src.NoncurrentVersionTransition.StorageClass
	}

	if opts.IsStorageClassSet {
		dest.Transition.StorageClass = src.Transition.StorageClass
	}

	// Updated the status
	if src.Status != "" {
		dest.Status = src.Status
	}
	return dest
}
