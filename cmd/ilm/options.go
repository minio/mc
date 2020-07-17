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
	"fmt"
	"math"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
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

type lifecycleOptions struct {
	ID             string
	Prefix         string
	Status         bool
	Tags           string
	ExpiryDate     string
	ExpiryDays     string
	TransitionDate string
	TransitionDays string
	StorageClass   string
}

func (opts lifecycleOptions) ToConfig(config *lifecycle.Configuration) (*lifecycle.Configuration, *probe.Error) {
	expiry, err := parseExpiry(opts.ExpiryDate, opts.ExpiryDays)
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
	}

	if err := validateILMRule(newRule); err != nil {
		return nil, err.Trace(opts.ID)
	}

	ruleFound := false
	for i, rule := range config.Rules {
		if rule.ID != newRule.ID {
			continue
		}
		config.Rules[i] = newRule
		ruleFound = true
		break
	}

	if !ruleFound {
		config.Rules = append(config.Rules, newRule)
	}

	return config, nil
}

func getLifecycleOptions(ctx *cli.Context) lifecycleOptions {
	return lifecycleOptions{
		ID:             ctx.String("id"),
		Prefix:         ctx.String("prefix"),
		Status:         !ctx.Bool("disable"),
		Tags:           ctx.String("tags"),
		ExpiryDate:     ctx.String("expiry-date"),
		ExpiryDays:     ctx.String("expiry-days"),
		TransitionDate: ctx.String("transition-date"),
		TransitionDays: ctx.String("transition-days"),
		StorageClass:   ctx.String("storage-class"),
	}
}

// ApplyNewILMConfig apply new lifecycle rules to existing lifecycle configuration, this
// function returns modified/overwritten rules if any.
func ApplyNewILMConfig(ctx *cli.Context, config *lifecycle.Configuration) (*lifecycle.Configuration, *probe.Error) {
	opts := getLifecycleOptions(ctx)
	return opts.ToConfig(config)
}
