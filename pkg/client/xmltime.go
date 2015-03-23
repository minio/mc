/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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

package client

import (
	"time"

	"encoding/xml"
)

// Date format
const (
	iso8601Format = "2006-01-02T15:04:05.000Z"
)

// XMLTime - time wrapper
type XMLTime struct {
	time.Time
}

// UnmarshalXML - unmarshal incoming xml
func (c *XMLTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	d.DecodeElement(&v, &start)
	parse, _ := time.Parse(iso8601Format, v)
	*c = XMLTime{parse}
	return nil
}

// UnmarshalXMLAttr - unmarshal specific attr
func (c *XMLTime) UnmarshalXMLAttr(attr xml.Attr) error {
	t, _ := time.Parse(iso8601Format, attr.Value)
	*c = XMLTime{t}
	return nil
}

// String - xml to string
func (c *XMLTime) String() string {
	return c.Time.Format(iso8601Format)
}
