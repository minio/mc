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
	"os"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var ilmCmd = cli.Command{
	Name:   "ilm",
	Usage:  "configure bucket lifecycle",
	Action: mainLifecycle,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		ilmShowCmd,
		ilmAddCmd,
		ilmRemoveCmd,
		ilmExportCmd,
		ilmImportCmd,
	},
}

type abortIncompleteMultipartUpload struct {
	XMLName             xml.Name `xml:"AbortIncompleteMultipartUpload,omitempty"  json:"-"`
	DaysAfterInitiation int64    `xml:"DaysAfterInitiation,omitempty" json:"DaysAfterInitiation,omitempty"`
}

type noncurrentVersionTransition struct {
	XMLName          xml.Name `xml:"NoncurrentVersionTransition,omitempty"  json:"-"`
	StorageClass     string   `xml:"StorageClass,omitempty" json:"StorageClass,omitempty"`
	TransitionInDays int      `xml:"TransitionInDays,omitempty" json:"TransitionInDays,omitempty"`
}

type tagFilter struct {
	XMLName xml.Name `xml:"Tag,omitempty" json:"-"`
	Key     string   `xml:"Key,omitempty" json:"Key,omitempty"`
	Value   string   `xml:"Value,omitempty" json:"Value,omitempty"`
}

type lifecycleTransition struct {
	XMLName          xml.Name   `xml:"Transition" json:"-"`
	TransitionDate   *time.Time `xml:"Date,omitempty" json:"Date,omitempty"`
	StorageClass     string     `xml:"StorageClass,omitempty" json:"StorageClass,omitempty"`
	TransitionInDays int        `xml:"Days,omitempty" json:"Days,omitempty"`
}

type lifecycleAndOperator struct {
	XMLName xml.Name    `xml:"And,omitempty" json:"-"`
	Prefix  string      `xml:"Prefix,omitempty" json:"Prefix,omitempty"`
	Tags    []tagFilter `xml:"Tag,omitempty" json:"Tags,omitempty"`
}

type lifecycleRuleFilter struct {
	XMLName xml.Name              `xml:"Filter" json:"-"`
	And     *lifecycleAndOperator `xml:"And,omitempty" json:"And,omitempty"`
	Prefix  string                `xml:"Prefix,omitempty" json:"Prefix,omitempty"`
	Tag     *tagFilter            `xml:"Tag,omitempty" json:"-"`
}

type lifecycleExpiration struct {
	XMLName          xml.Name   `xml:"Expiration,omitempty" json:"-"`
	ExpirationDate   *time.Time `xml:"Date,omitempty" json:"Date,omitempty"`
	ExpirationInDays int        `xml:"Days,omitempty" json:"Days,omitempty"`
}

type lifecycleRule struct {
	XMLName                        xml.Name                        `xml:"Rule,omitempty" json:"-"`
	AbortIncompleteMultipartUpload *abortIncompleteMultipartUpload `xml:"AbortIncompleteMultipartUpload,omitempty" json:"AbortIncompleteMultipartUpload,omitempty"`
	Expiration                     *lifecycleExpiration            `xml:"Expiration,omitempty" json:"Expiration,omitempty"`
	ID                             string                          `xml:"ID" json:"ID"`
	RuleFilter                     *lifecycleRuleFilter            `xml:"Filter,omitempty" json:"Filter,omitempty"`
	NoncurrentVersionTransition    *noncurrentVersionTransition    `xml:"NoncurrentVersionTransition,omitempty" json:"NoncurrentVersionTransition,omitempty"`
	NoncurrentVersionTransitions   []noncurrentVersionTransition   `xml:"NoncurrentVersionTransitions,omitempty" json:"NoncurrentVersionTransitions,omitempty"`
	Prefix                         string                          `xml:"Prefix,omitempty" json:"Prefix,omitempty"`
	Status                         string                          `xml:"Status" json:"Status"`
	Transition                     *lifecycleTransition            `xml:"Transition,omitempty" json:"Transition,omitempty"`
	Transitions                    []lifecycleTransition           `xml:"Transitions,omitempty" json:"Transitions,omitempty"`
	TagFilters                     []tagFilter                     `xml:"Tag,omitempty" json:"Tags,omitempty"`
}

type lifecycleConfiguration struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration,omitempty" json:"-"`
	Rules   []lifecycleRule `xml:"Rule"`
}

// checkIlmMainSyntax - validate arguments passed by a user
func checkIlmMainSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelp(ctx, "")
		os.Exit(globalErrorExitStatus)
	}
}

func mainLifecycle(ctx *cli.Context) error {
	checkIlmMainSyntax(ctx)
	setColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	var pErr *probe.Error
	clnt, pErr := newClient(objectURL)
	if pErr != nil {
		cli.ShowCommandHelp(ctx, "")
		return pErr.Trace(objectURL).ToGoError()
	}

	_, pErr = clnt.GetBucketLifecycle()
	if pErr != nil {
		cli.ShowCommandHelp(ctx, "")
		return pErr.ToGoError()
	}
	cli.ShowCommandHelp(ctx, "show")

	return nil
}
