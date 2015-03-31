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
	"errors"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minio-io/donut"
)

// donutDriver - creates a new single disk drivers driver using donut
type donutDriver struct {
	donut donut.Donut
}

// Object split blockSize defaulted at 10MB
const (
	blockSize = 10 * 1024 * 1024
)

// IsValidBucketName reports whether bucket is a valid bucket name, per Amazon's naming restrictions.
// See http://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
func IsValidBucketName(bucket string) bool {
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}
	if bucket[0] == '.' || bucket[len(bucket)-1] == '.' {
		return false
	}
	if match, _ := regexp.MatchString("\\.\\.", bucket); match == true {
		return false
	}
	// We don't support buckets with '.' in them
	match, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", bucket)
	return match
}

// GetNewClient returns an initialized donut driver
func GetNewClient(donutName string, nodeDiskMap map[string][]string) (Client, error) {
	var err error

	d := new(donutDriver)
	d.donut, err = donut.NewDonut(donutName, nodeDiskMap)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// byBucketName is a type for sorting bucket metadata by bucket name
type byBucketName []*Bucket

func (b byBucketName) Len() int           { return len(b) }
func (b byBucketName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byBucketName) Less(i, j int) bool { return b[i].Name < b[j].Name }

// ListBuckets returns a list of buckets
func (d *donutDriver) ListBuckets() (results []*Bucket, err error) {
	buckets, err := d.donut.ListBuckets()
	if err != nil {
		return nil, err
	}
	for name := range buckets {
		t := XMLTime{
			Time: time.Now(),
		}
		result := &Bucket{
			Name: name,
			// TODO Add real created date
			CreationDate: t,
		}
		results = append(results, result)
	}
	sort.Sort(byBucketName(results))
	return results, nil
}

// PutBucket creates a new bucket
func (d *donutDriver) PutBucket(bucketName string) error {
	if IsValidBucketName(bucketName) && !strings.Contains(bucketName, ".") {
		return d.donut.MakeBucket(bucketName)
	}
	return errors.New("Invalid bucket")
}

// Get retrieves an object and writes it to a writer
func (d *donutDriver) Get(bucketName, objectName string) (body io.ReadCloser, size int64, err error) {
	if bucketName == "" || strings.TrimSpace(bucketName) == "" {
		return nil, 0, errors.New("invalid argument")
	}
	if objectName == "" || strings.TrimSpace(objectName) == "" {
		return nil, 0, errors.New("invalid argument")
	}
	reader, writer := io.Pipe()
	buckets, err := d.donut.ListBuckets()
	if err != nil {
		return nil, 0, err
	}
	if _, ok := buckets[bucketName]; !ok {
		return nil, 0, errors.New("bucket does not exist")
	}
	objects, err := buckets[bucketName].ListObjects()
	if _, ok := objects[objectName]; !ok {
		return nil, 0, errors.New("object does not exist")
	}
	donutObjectMetadata, err := objects[objectName].GetDonutObjectMetadata()
	if err != nil {
		return nil, 0, err
	}
	size, err = strconv.ParseInt(donutObjectMetadata["size"], 10, 64)
	if err != nil {
		return nil, 0, err
	}
	go buckets[bucketName].GetObject(objectName, writer, donutObjectMetadata)
	return reader, size, nil
}

// Put creates a new object
func (d *donutDriver) Put(bucketName, objectKey string, size int64, contents io.Reader) error {
	buckets, err := d.donut.ListBuckets()
	if err != nil {
		return err
	}
	objects, err := buckets[bucketName].ListObjects()
	if _, ok := objects[objectKey]; ok {
		return errors.New("Object exists")
	}
	err = buckets[bucketName].PutObject(objectKey, contents)
	if err != nil {
		return err
	}
	return nil
}

// GetPartial retrieves an object range and writes it to a writer
func (d *donutDriver) GetPartial(bucket, object string, start, length int64) (body io.ReadCloser, size int64, err error) {
	return nil, 0, errors.New("Not Implemented")
}

// Stat - gets metadata information about the object
func (d *donutDriver) Stat(bucket, object string) (size int64, date time.Time, err error) {
	return 0, time.Time{}, errors.New("Not Implemented")
}

// bySize implements sort.Interface for []Item based on the Size field.
type bySize []*Item

func (a bySize) Len() int           { return len(a) }
func (a bySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a bySize) Less(i, j int) bool { return a[i].Size < a[j].Size }

// ListObjects - returns list of objects
func (d *donutDriver) ListObjects(bucketName, startAt, prefix, delimiter string, maxKeys int) (items []*Item, prefixes []*Prefix, err error) {
	buckets, err := d.donut.ListBuckets()
	if err != nil {
		return nil, nil, err
	}
	objectList, err := buckets[bucketName].ListObjects()
	if err != nil {
		return nil, nil, err
	}
	var objects []string
	for key := range objectList {
		objects = append(objects, key)
	}
	sort.Strings(objects)
	if prefix != "" {
		objects = filterPrefix(objects, prefix)
		objects = removePrefix(objects, prefix)
	}
	if maxKeys <= 0 || maxKeys > 1000 {
		maxKeys = 1000
	}
	var actualObjects []string
	var commonPrefixes []string
	if strings.TrimSpace(delimiter) != "" {
		actualObjects = filterDelimited(objects, delimiter)
		commonPrefixes = filterNotDelimited(objects, delimiter)
		commonPrefixes = extractDir(commonPrefixes, delimiter)
		commonPrefixes = uniqueObjects(commonPrefixes)
	} else {
		actualObjects = objects
	}

	for _, prefix := range commonPrefixes {
		prefixes = append(prefixes, &Prefix{Prefix: prefix})
	}
	for _, object := range actualObjects {
		metadata, err := objectList[object].GetDonutObjectMetadata()
		if err != nil {
			return nil, nil, err
		}
		t1, err := time.Parse(time.RFC3339Nano, metadata["created"])
		if err != nil {
			return nil, nil, err
		}
		t := XMLTime{
			Time: t1,
		}
		size, err := strconv.ParseInt(metadata["size"], 10, 64)
		if err != nil {
			return nil, nil, err
		}
		item := &Item{
			Key:          object,
			LastModified: t,
			Size:         size,
		}
		items = append(items, item)
	}
	sort.Sort(bySize(items))
	return items, prefixes, nil
}
