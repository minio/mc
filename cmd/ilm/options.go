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
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/rs/xid"
)

const defaultILMDateFormat string = "2006-01-02"

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
	ID string

	Status *bool

	Prefix                *string
	Tags                  *string
	ObjectSizeLessThan    *int64
	ObjectSizeGreaterThan *int64
	ExpiryDate            *string
	ExpiryDays            *string
	TransitionDate        *string
	TransitionDays        *string
	StorageClass          *string

	ExpiredObjectDeleteMarker               *bool
	NoncurrentVersionExpirationDays         *int
	NewerNoncurrentExpirationVersions       *int
	NoncurrentVersionTransitionDays         *int
	NewerNoncurrentTransitionVersions       *int
	NoncurrentVersionTransitionStorageClass *string
	ExpiredObjectAllversions                *bool
}

// Filter returns lifecycle.Filter appropriate for opts
func (opts LifecycleOptions) Filter() lifecycle.Filter {
	var f lifecycle.Filter
	var tags []lifecycle.Tag
	var predCount int
	if opts.Tags != nil {
		tags = extractILMTags(*opts.Tags)
		predCount += len(tags)
	}
	var prefix string
	if opts.Prefix != nil {
		prefix = *opts.Prefix
		predCount++
	}

	var szLt, szGt int64
	if opts.ObjectSizeLessThan != nil {
		szLt = *opts.ObjectSizeLessThan
		predCount++
	}

	if opts.ObjectSizeGreaterThan != nil {
		szGt = *opts.ObjectSizeGreaterThan
		predCount++
	}

	if predCount >= 2 {
		f.And = lifecycle.And{
			Tags:                  tags,
			Prefix:                prefix,
			ObjectSizeLessThan:    szLt,
			ObjectSizeGreaterThan: szGt,
		}
	} else {
		// In a valid lifecycle rule filter at most one of the
		// following will only be set.
		f.Prefix = prefix
		f.ObjectSizeGreaterThan = szGt
		f.ObjectSizeLessThan = szLt
		if len(tags) >= 1 {
			f.Tag = tags[0]
		}
	}

	return f
}

// ToILMRule creates lifecycle.Configuration based on LifecycleOptions
func (opts LifecycleOptions) ToILMRule() (lifecycle.Rule, *probe.Error) {
	var (
		id, status string

		nonCurrentVersionExpirationDays         lifecycle.ExpirationDays
		newerNonCurrentExpirationVersions       int
		nonCurrentVersionTransitionDays         lifecycle.ExpirationDays
		newerNonCurrentTransitionVersions       int
		nonCurrentVersionTransitionStorageClass string
	)

	id = opts.ID
	status = func() string {
		if opts.Status != nil && !*opts.Status {
			return "Disabled"
		}
		// Generating a new ILM rule without explicit status is enabled
		return "Enabled"
	}()

	expiry, err := parseExpiry(opts.ExpiryDate, opts.ExpiryDays, opts.ExpiredObjectDeleteMarker, opts.ExpiredObjectAllversions)
	if err != nil {
		return lifecycle.Rule{}, err
	}

	transition, err := parseTransition(opts.StorageClass, opts.TransitionDate, opts.TransitionDays)
	if err != nil {
		return lifecycle.Rule{}, err
	}

	if opts.NoncurrentVersionExpirationDays != nil {
		nonCurrentVersionExpirationDays = lifecycle.ExpirationDays(*opts.NoncurrentVersionExpirationDays)
	}
	if opts.NewerNoncurrentExpirationVersions != nil {
		newerNonCurrentExpirationVersions = *opts.NewerNoncurrentExpirationVersions
	}
	if opts.NoncurrentVersionTransitionDays != nil {
		nonCurrentVersionTransitionDays = lifecycle.ExpirationDays(*opts.NoncurrentVersionTransitionDays)
	}
	if opts.NewerNoncurrentTransitionVersions != nil {
		newerNonCurrentTransitionVersions = *opts.NewerNoncurrentTransitionVersions
	}
	if opts.NoncurrentVersionTransitionStorageClass != nil {
		nonCurrentVersionTransitionStorageClass = *opts.NoncurrentVersionTransitionStorageClass
	}

	newRule := lifecycle.Rule{
		ID:         id,
		RuleFilter: opts.Filter(),
		Status:     status,
		Expiration: expiry,
		Transition: transition,
		NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{
			NoncurrentDays:          nonCurrentVersionExpirationDays,
			NewerNoncurrentVersions: newerNonCurrentExpirationVersions,
		},
		NoncurrentVersionTransition: lifecycle.NoncurrentVersionTransition{
			NoncurrentDays:          nonCurrentVersionTransitionDays,
			NewerNoncurrentVersions: newerNonCurrentTransitionVersions,
			StorageClass:            nonCurrentVersionTransitionStorageClass,
		},
	}

	if err := validateILMRule(newRule); err != nil {
		return lifecycle.Rule{}, err
	}

	return newRule, nil
}

func strPtr(s string) *string {
	ptr := s
	return &ptr
}

func intPtr(i int) *int {
	ptr := i
	return &ptr
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	ptr := b
	return &ptr
}

// GetLifecycleOptions create LifeCycleOptions based on cli inputs
func GetLifecycleOptions(ctx *cli.Context) (LifecycleOptions, *probe.Error) {
	var (
		id string

		status *bool

		prefix         *string
		tags           *string
		sizeLt         *int64
		sizeGt         *int64
		expiryDate     *string
		expiryDays     *string
		transitionDate *string
		transitionDays *string
		tier           *string

		expiredObjectDeleteMarker         *bool
		noncurrentVersionExpirationDays   *int
		newerNoncurrentExpirationVersions *int
		noncurrentVersionTransitionDays   *int
		newerNoncurrentTransitionVersions *int
		noncurrentTier                    *string
		expiredObjectAllversions          *bool
	)

	id = ctx.String("id")
	if id == "" {
		id = xid.New().String()
	}

	switch {
	case ctx.IsSet("disable"):
		status = boolPtr(!ctx.Bool("disable"))
	case ctx.IsSet("enable"):
		status = boolPtr(ctx.Bool("enable"))
	}

	if ctx.IsSet("prefix") {
		prefix = strPtr(ctx.String("prefix"))
	} else {
		// Calculating the prefix for the aliased URL is deprecated in Aug 2022
		// split the first arg i.e. path into alias, bucket and prefix
		result := strings.SplitN(ctx.Args().First(), "/", 3)
		// get the prefix from path
		if len(result) > 2 {
			p := result[len(result)-1]
			if len(p) > 0 {
				prefix = &p
			}
		}
	}

	expiryRulesCount := 0

	if ctx.IsSet("size-lt") {
		szStr := ctx.String("size-lt")
		szLt, err := humanize.ParseBytes(szStr)
		if err != nil || szLt > math.MaxInt64 {
			return LifecycleOptions{}, probe.NewError(fmt.Errorf("size-lt value %s is invalid", szStr))
		}

		sizeLt = int64Ptr(int64(szLt))
	}
	if ctx.IsSet("size-gt") {
		szStr := ctx.String("size-gt")
		szGt, err := humanize.ParseBytes(szStr)
		if err != nil || szGt > math.MaxInt64 {
			return LifecycleOptions{}, probe.NewError(fmt.Errorf("size-gt value %s is invalid", szStr))
		}
		sizeGt = int64Ptr(int64(szGt))
	}

	// For backward-compatibility
	if ctx.IsSet("storage-class") {
		tier = strPtr(strings.ToUpper(ctx.String("storage-class")))
	}
	if ctx.IsSet("noncurrentversion-transition-storage-class") {
		noncurrentTier = strPtr(strings.ToUpper(ctx.String("noncurrentversion-transition-storage-class")))
	}
	if ctx.IsSet("tier") {
		tier = strPtr(strings.ToUpper(ctx.String("tier")))
	}
	if f := "transition-tier"; ctx.IsSet(f) {
		tier = strPtr(strings.ToUpper(ctx.String(f)))
	}
	if ctx.IsSet("noncurrentversion-tier") {
		noncurrentTier = strPtr(strings.ToUpper(ctx.String("noncurrentversion-tier")))
	}
	if f := "noncurrent-transition-tier"; ctx.IsSet(f) {
		noncurrentTier = strPtr(strings.ToUpper(ctx.String(f)))
	}
	if tier != nil && !ctx.IsSet("transition-days") && !ctx.IsSet("transition-date") {
		return LifecycleOptions{}, probe.NewError(errors.New("transition-date or transition-days must be set"))
	}
	if noncurrentTier != nil && !ctx.IsSet("noncurrentversion-transition-days") && !ctx.IsSet("noncurrent-transition-days") {
		return LifecycleOptions{}, probe.NewError(errors.New("noncurrentversion-transition-days must be set"))
	}
	// for MinIO transition storage-class is same as label defined on
	// `mc admin bucket remote add --service ilm --label` command
	if ctx.IsSet("tags") {
		tags = strPtr(ctx.String("tags"))
	}
	if ctx.IsSet("expiry-date") {
		expiryRulesCount++
		expiryDate = strPtr(ctx.String("expiry-date"))
	}
	if ctx.IsSet("expiry-days") {
		expiryRulesCount++
		expiryDays = strPtr(ctx.String("expiry-days"))
	}
	if f := "expire-days"; ctx.IsSet(f) {
		expiryDays = strPtr(ctx.String(f))
	}
	if ctx.IsSet("transition-date") {
		transitionDate = strPtr(ctx.String("transition-date"))
	}
	if ctx.IsSet("transition-days") {
		transitionDays = strPtr(ctx.String("transition-days"))
	}
	if ctx.IsSet("expired-object-delete-marker") {
		expiredObjectDeleteMarker = boolPtr(ctx.Bool("expired-object-delete-marker"))
	}
	if f := "expire-delete-marker"; ctx.IsSet(f) {
		expiryRulesCount++
		expiredObjectDeleteMarker = boolPtr(ctx.Bool(f))
	}
	if ctx.IsSet("noncurrentversion-expiration-days") {
		noncurrentVersionExpirationDays = intPtr(ctx.Int("noncurrentversion-expiration-days"))
	}
	if f := "noncurrent-expire-days"; ctx.IsSet(f) {
		ndaysStr := ctx.String(f)
		ndays, err := strconv.Atoi(ndaysStr)
		if err != nil {
			return LifecycleOptions{}, probe.NewError(fmt.Errorf("failed to parse %s: %v", f, err))
		}
		noncurrentVersionExpirationDays = &ndays
	}
	if ctx.IsSet("newer-noncurrentversions-expiration") {
		newerNoncurrentExpirationVersions = intPtr(ctx.Int("newer-noncurrentversions-expiration"))
	}
	if f := "noncurrent-expire-newer"; ctx.IsSet(f) {
		newerNoncurrentExpirationVersions = intPtr(ctx.Int(f))
	}
	if ctx.IsSet("noncurrentversion-transition-days") {
		noncurrentVersionTransitionDays = intPtr(ctx.Int("noncurrentversion-transition-days"))
	}
	if f := "noncurrent-transition-days"; ctx.IsSet(f) {
		noncurrentVersionTransitionDays = intPtr(ctx.Int(f))
	}
	if ctx.IsSet("newer-noncurrentversions-transition") {
		newerNoncurrentTransitionVersions = intPtr(ctx.Int("newer-noncurrentversions-transition"))
	}
	if f := "noncurrent-transition-newer"; ctx.IsSet(f) {
		newerNoncurrentTransitionVersions = intPtr(ctx.Int(f))
	}
	if ctx.IsSet("expire-all-object-versions") {
		expiredObjectAllversions = boolPtr(ctx.Bool("expire-all-object-versions"))
	}

	if expiryRulesCount > 1 {
		return LifecycleOptions{}, probe.NewError(errors.New("only one of expiry-date, expiry-days and expire-delete-marker can be used in a single rule. Try adding multiple rules to achieve the desired effect"))
	}

	return LifecycleOptions{
		ID:                                      id,
		Status:                                  status,
		Prefix:                                  prefix,
		Tags:                                    tags,
		ObjectSizeLessThan:                      sizeLt,
		ObjectSizeGreaterThan:                   sizeGt,
		ExpiryDate:                              expiryDate,
		ExpiryDays:                              expiryDays,
		TransitionDate:                          transitionDate,
		TransitionDays:                          transitionDays,
		StorageClass:                            tier,
		ExpiredObjectDeleteMarker:               expiredObjectDeleteMarker,
		NoncurrentVersionExpirationDays:         noncurrentVersionExpirationDays,
		NewerNoncurrentExpirationVersions:       newerNoncurrentExpirationVersions,
		NoncurrentVersionTransitionDays:         noncurrentVersionTransitionDays,
		NewerNoncurrentTransitionVersions:       newerNoncurrentTransitionVersions,
		NoncurrentVersionTransitionStorageClass: noncurrentTier,
		ExpiredObjectAllversions:                expiredObjectAllversions,
	}, nil
}

// ApplyRuleFields applies non nil fields of LifcycleOptions to the existing lifecycle rule
func ApplyRuleFields(dest *lifecycle.Rule, opts LifecycleOptions) *probe.Error {
	// If src has tags, it should override the destination
	if opts.Tags != nil {
		dest.RuleFilter.And.Tags = extractILMTags(*opts.Tags)
	}

	// since prefix is a part of command args, it is always present in the src rule and
	// it should be always set to the destination.
	if opts.Prefix != nil {
		if dest.RuleFilter.And.Tags != nil {
			dest.RuleFilter.And.Prefix = *opts.Prefix
		} else {
			dest.RuleFilter.Prefix = *opts.Prefix
		}
	}

	// only one of expiration day, date or transition day, date is expected
	if opts.ExpiryDate != nil {
		date, err := parseExpiryDate(*opts.ExpiryDate)
		if err != nil {
			return err
		}
		dest.Expiration.Date = date
		// reset everything else
		dest.Expiration.Days = 0
		dest.Expiration.DeleteMarker = false
	} else if opts.ExpiryDays != nil {
		days, err := parseExpiryDays(*opts.ExpiryDays)
		if err != nil {
			return err
		}
		dest.Expiration.Days = days
		// reset everything else
		dest.Expiration.Date = lifecycle.ExpirationDate{}
	} else if opts.ExpiredObjectDeleteMarker != nil {
		dest.Expiration.DeleteMarker = lifecycle.ExpireDeleteMarker(*opts.ExpiredObjectDeleteMarker)
		dest.Expiration.Days = 0
		dest.Expiration.Date = lifecycle.ExpirationDate{}
	}
	if opts.ExpiredObjectAllversions != nil {
		dest.Expiration.DeleteAll = lifecycle.ExpirationBoolean(*opts.ExpiredObjectAllversions)
	}

	if opts.TransitionDate != nil {
		date, err := parseTransitionDate(*opts.TransitionDate)
		if err != nil {
			return err
		}
		dest.Transition.Date = date
		// reset everything else
		dest.Transition.Days = 0
	} else if opts.TransitionDays != nil {
		days, err := parseTransitionDays(*opts.TransitionDays)
		if err != nil {
			return err
		}
		dest.Transition.Days = days
		// reset everything else
		dest.Transition.Date = lifecycle.ExpirationDate{}
	}

	if opts.NoncurrentVersionExpirationDays != nil {
		dest.NoncurrentVersionExpiration.NoncurrentDays = lifecycle.ExpirationDays(*opts.NoncurrentVersionExpirationDays)
	}

	if opts.NewerNoncurrentExpirationVersions != nil {
		dest.NoncurrentVersionExpiration.NewerNoncurrentVersions = *opts.NewerNoncurrentExpirationVersions
	}

	if opts.NoncurrentVersionTransitionDays != nil {
		dest.NoncurrentVersionTransition.NoncurrentDays = lifecycle.ExpirationDays(*opts.NoncurrentVersionTransitionDays)
	}

	if opts.NewerNoncurrentTransitionVersions != nil {
		dest.NoncurrentVersionTransition.NewerNoncurrentVersions = *opts.NewerNoncurrentTransitionVersions
	}

	if opts.NoncurrentVersionTransitionStorageClass != nil {
		dest.NoncurrentVersionTransition.StorageClass = *opts.NoncurrentVersionTransitionStorageClass
	}

	if opts.StorageClass != nil {
		dest.Transition.StorageClass = *opts.StorageClass
	}

	// Updated the status
	if opts.Status != nil {
		dest.Status = func() string {
			if *opts.Status {
				return "Enabled"
			}
			return "Disabled"
		}()
	}

	return nil
}
