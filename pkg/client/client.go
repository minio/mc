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
	"io"
	"time"

	"encoding/xml"
)

// Client - client interface
type Client interface {
	Get(bucket, object string) (body io.ReadCloser, size int64, err error)
	GetPartial(bucket, key string, offset, length int64) (body io.ReadCloser, size int64, err error)
	Put(bucket, object string, size int64, body io.Reader) error
	Stat(bucket, object string) (size int64, date time.Time, err error)
	PutBucket(bucket string) error
	ListBuckets() ([]*Bucket, error)
	ListObjects(bucket string, startAt, prefix, delimiter string, maxKeys int) (items []*Item, prefixes []*Prefix, err error)
}

// Bucket - carries s3 bucket reply header
type Bucket struct {
	Name         string
	CreationDate XMLTime // 2006-02-03T16:45:09.000Z
}

// Item - object item list
type Item struct {
	Key          string
	LastModified XMLTime
	Size         int64
}

// Prefix - common prefix
type Prefix struct {
	Prefix string
}

// Maxkeys listed per ListObjects()
const (
	Maxkeys = 1000
)

// Date format
const (
	iso8601Format = "2006-01-02T15:04:05.000Z"
)

// XMLTime - time wrapper
type XMLTime struct {
	Time time.Time
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
