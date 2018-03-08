/*
 * Minio Client (C) 2015 Minio, Inc.
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

import "github.com/minio/mc/pkg/probe"

// URLs contains source and target urls
type URLs struct {
	SourceAlias   string
	SourceContent *clientContent
	TargetAlias   string
	TargetContent *clientContent
	TotalCount    int64
	TotalSize     int64
	encKeyDB      map[string][]prefixSSEPair
	SrcSSEKey     string
	TgtSSEKey     string
	Error         *probe.Error `json:"-"`
}

// WithError sets the error and returns object
func (m URLs) WithError(err *probe.Error) URLs {
	m.Error = err
	return m
}

// Equal tests if both urls are equal
func (m URLs) Equal(n URLs) bool {
	if m.SourceContent == nil && n.SourceContent == nil {
	} else if m.SourceContent != nil && n.SourceContent == nil {
		return false
	} else if m.SourceContent == nil && n.SourceContent != nil {
		return false
	} else if m.SourceContent.URL != n.SourceContent.URL {
		return false
	}

	if m.TargetContent == nil && n.TargetContent == nil {
	} else if m.TargetContent != nil && n.TargetContent == nil {
		return false
	} else if m.TargetContent == nil && n.TargetContent != nil {
		return false
	} else if m.TargetContent.URL != n.TargetContent.URL {
		return false
	}

	return true
}

func (m URLs) isEmpty() bool {
	if m.SourceContent == nil && m.TargetContent == nil && m.Error == nil {
		return true
	}
	// If remove flag is set then sourceContent is usually nil.
	if m.SourceContent != nil {
		if m.SourceContent.Size == 0 && m.TargetContent == nil && m.Error == nil {
			return true
		}
	}
	return false
}
