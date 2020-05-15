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

// Package ilm contains all the ilm related functionality. TO BE ACCESSED VIA GET FUNCTIONS or Operation-related Global functions.
package ilm

import (
	"encoding/json"
	"encoding/xml"
	"time"

	"github.com/minio/minio/pkg/console"
)

// AbortIncompleteMultipartUpload structure
type AbortIncompleteMultipartUpload struct {
	XMLName             xml.Name `xml:"AbortIncompleteMultipartUpload,omitempty"  json:"-"`
	DaysAfterInitiation int64    `xml:"DaysAfterInitiation,omitempty" json:"DaysAfterInitiation,omitempty"`
}

// NoncurrentVersionTransition structure
type NoncurrentVersionTransition struct {
	XMLName          xml.Name `xml:"NoncurrentVersionTransition,omitempty"  json:"-"`
	StorageClass     string   `xml:"StorageClass,omitempty" json:"StorageClass,omitempty"`
	TransitionInDays int      `xml:"TransitionInDays,omitempty" json:"TransitionInDays,omitempty"`
}

// LifecycleTag structure key/value pair representing an object tag to apply lifecycle configuration
type LifecycleTag struct {
	XMLName xml.Name `xml:"Tag,omitempty" json:"-"`
	Key     string   `xml:"Key,omitempty" json:"Key,omitempty"`
	Value   string   `xml:"Value,omitempty" json:"Value,omitempty"`
}

// LifecycleTransition structure - transition details of lifeycle configuration
type LifecycleTransition struct {
	XMLName          xml.Name   `xml:"Transition" json:"-"`
	TransitionDate   *time.Time `xml:"Date,omitempty" json:"Date,omitempty"`
	StorageClass     string     `xml:"StorageClass,omitempty" json:"StorageClass,omitempty"`
	TransitionInDays int        `xml:"Days,omitempty" json:"Days,omitempty"`
}

// LifecycleAndOperator And Rule for LifecycleTag, to be used in LifecycleRuleFilter
type LifecycleAndOperator struct {
	XMLName xml.Name       `xml:"And,omitempty" json:"-"`
	Prefix  string         `xml:"Prefix,omitempty" json:"Prefix,omitempty"`
	Tags    []LifecycleTag `xml:"Tag,omitempty" json:"Tags,omitempty"`
}

// LifecycleRuleFilter will be used in selecting rule(s) for lifecycle configuration
type LifecycleRuleFilter struct {
	XMLName xml.Name              `xml:"Filter" json:"-"`
	And     *LifecycleAndOperator `xml:"And,omitempty" json:"And,omitempty"`
	Prefix  string                `xml:"Prefix,omitempty" json:"Prefix,omitempty"`
	Tag     *LifecycleTag         `xml:"Tag,omitempty" json:"-"`
}

// LifecycleExpiration structure - expiration details of lifeycle configuration
type LifecycleExpiration struct {
	XMLName          xml.Name   `xml:"Expiration,omitempty" json:"-"`
	ExpirationDate   *time.Time `xml:"Date,omitempty" json:"Date,omitempty"`
	ExpirationInDays int        `xml:"Days,omitempty" json:"Days,omitempty"`
}

// LifecycleRule represents a single rule in lifecycle configuration
type LifecycleRule struct {
	XMLName                        xml.Name                        `xml:"Rule,omitempty" json:"-"`
	AbortIncompleteMultipartUpload *AbortIncompleteMultipartUpload `xml:"AbortIncompleteMultipartUpload,omitempty" json:"AbortIncompleteMultipartUpload,omitempty"`
	Expiration                     *LifecycleExpiration            `xml:"Expiration,omitempty" json:"Expiration,omitempty"`
	ID                             string                          `xml:"ID" json:"ID"`
	RuleFilter                     *LifecycleRuleFilter            `xml:"Filter,omitempty" json:"Filter,omitempty"`
	NoncurrentVersionTransition    *NoncurrentVersionTransition    `xml:"NoncurrentVersionTransition,omitempty" json:"NoncurrentVersionTransition,omitempty"`
	NoncurrentVersionTransitions   []NoncurrentVersionTransition   `xml:"NoncurrentVersionTransitions,omitempty" json:"NoncurrentVersionTransitions,omitempty"`
	Prefix                         string                          `xml:"Prefix,omitempty" json:"Prefix,omitempty"`
	Status                         string                          `xml:"Status" json:"Status"`
	Transition                     *LifecycleTransition            `xml:"Transition,omitempty" json:"Transition,omitempty"`
	Transitions                    []LifecycleTransition           `xml:"Transitions,omitempty" json:"Transitions,omitempty"`
	TagFilters                     []LifecycleTag                  `xml:"Tag,omitempty" json:"Tags,omitempty"`
}

// LifecycleConfiguration is a collection of LifecycleRule objects.
type LifecycleConfiguration struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration,omitempty" json:"-"`
	Rules   []LifecycleRule `xml:"Rule"`
}

// JSON jsonified lifecycle configuration.
func (l LifecycleConfiguration) JSON() string {
	var ilmJSONbytes []byte
	var err error
	ilmJSONbytes, err = json.MarshalIndent(l, "", "    ")
	if err != nil {
		console.Fatal("Failed to get lifecycle configuration in JSON format.")
	}
	return string(ilmJSONbytes)
}
