/*
 * Minio Go Library for Amazon S3 Legacy v2 Signature Compatible Cloud Storage (C) 2015 Minio, Inc.
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

package minio

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

// decoder provides a unified decoding method interface
type decoder interface {
	Decode(v interface{}) error
}

// acceptTypeDecoder provide decoded value in given acceptType
func acceptTypeDecoder(body io.Reader, acceptType string, v interface{}) error {
	var d decoder
	switch {
	case acceptType == "application/xml":
		d = xml.NewDecoder(body)
	case acceptType == "application/json":
		d = json.NewDecoder(body)
	default:
		d = xml.NewDecoder(body)
	}
	return d.Decode(v)
}
