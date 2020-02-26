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
	stdlibjson "encoding/json"
	"encoding/xml"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio/pkg/console"
)

const (
	ilmErrorExitStatus = 1
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
	ilmJSONbytes, err = stdlibjson.MarshalIndent(l, "", "    ")
	if err != nil {
		console.Errorln("Unable to get lifecycle configuration in JSON format.")
		os.Exit(ilmErrorExitStatus)
	}
	return string(ilmJSONbytes)
}

type ilmRuleMessage struct {
	ID                  string
	Prefix              string
	Status              string
	ExpirationOnOrAfter string
	TransitionOnOrAfter string
	StorageClass        string
	TagFilters          string
}

type ilmMessage struct {
	NumRules int
	Rules    []ilmRuleMessage
}

func (m ilmMessage) JSON() string {
	var ilmMsgJSONbytes []byte
	var err error
	ilmMsgJSONbytes, err = stdlibjson.MarshalIndent(m, "", "    ")
	if err != nil {
		console.Errorln("Unable to get lifecycle configuration in JSON format.")
		os.Exit(ilmErrorExitStatus)
	}
	return string(ilmMsgJSONbytes)
}

func (m ilmMessage) String() (ilmOut string) {
	ilmOut += "Num of Rules: " + strconv.Itoa(m.NumRules) + "\n"
	for _, ruleMsg := range m.Rules {
		ilmOut += "Rule ID: " + ruleMsg.ID + "\n"
		ilmOut += "    Prefix                       : " + ruleMsg.Prefix + "\n"
		ilmOut += "    Status                       : " + ruleMsg.Status + "\n"
		ilmOut += "    Expiry (On or After)         : " + ruleMsg.ExpirationOnOrAfter + "\n"
		ilmOut += "    Transition (On or After)     : " + ruleMsg.TransitionOnOrAfter + "\n"
		ilmOut += "    Storage Class                : " + ruleMsg.StorageClass + "\n"
		ilmOut += "    Tags for Filter              : " + ruleMsg.TagFilters + "\n"
		ilmOut += "\n"
	}
	return ilmOut
}

func parseILMRuleMessage(ilm LifecycleConfiguration) ilmMessage {
	var ilmMsg ilmMessage
	for _, rule := range ilm.Rules {
		var ilmRuleMsg ilmRuleMessage
		var tagArr []string
		ilmRuleMsg.ID = rule.ID
		ilmRuleMsg.Status = rule.Status
		ilmRuleMsg.Prefix = getPrefixVal(rule)
		tagArr = getTagArr(rule)
		ilmRuleMsg.TagFilters = strings.Join(tagArr, ",")
		ilmRuleMsg.ExpirationOnOrAfter = strings.TrimSpace(getExpiryDateVal(rule))
		ilmRuleMsg.TransitionOnOrAfter = strings.TrimSpace(getTransitionDate(rule))
		ilmRuleMsg.StorageClass = strings.TrimSpace(getStorageClassName(rule))
		ilmMsg.Rules = append(ilmMsg.Rules, ilmRuleMsg)
	}
	ilmMsg.NumRules = len(ilm.Rules)
	return ilmMsg
}

// DisplayILMJSON display in JSON format. Hence separate function.
func DisplayILMJSON(ilmXML string) {
	var err error
	var ilmMsg ilmMessage
	var ilmInfo LifecycleConfiguration
	if ilmXML == "" {
		return
	}
	if err = xml.Unmarshal([]byte(ilmXML), &ilmInfo); err != nil {
		console.Errorln("Error assigning existing lifecycle configuration in XML: " + err.Error())
		return
	}

	// Assign Message
	ilmMsg = parseILMRuleMessage(ilmInfo)
	// Show ILM
	console.Println(ilmMsg.JSON())
}
