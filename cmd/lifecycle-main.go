/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"fmt"
	"time"

	"github.com/minio/cli"
)

// TODO: The usage and examples will change as the command implementation evolves after feedback.
var ilmCmd = cli.Command{
	Name:   "ilm",
	Usage:  "Bucket lifecycle management",
	Action: mainLifecycle,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		ilmShowCmd,
		ilmRemoveCmd,
		ilmSetCmd,
		ilmGenerateCmd,
		ilmCheckCmd,
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
	Tag     *tagFilter            `xml:"Tag,omitempty" json:"Tags,omitempty"`
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
	TagFilters                     []tagFilter                     `xml:"Tag,omitempty" json:"Tag,omitempty"`
}

type ilmResult struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration,omitempty" json:"-"`
	Rules   []lifecycleRule `xml:"Rule"`
}

func mainLifecycle(ctx *cli.Context) error {
	fmt.Println("lc-main")
	cli.ShowCommandHelp(ctx, "")
	return nil
}
