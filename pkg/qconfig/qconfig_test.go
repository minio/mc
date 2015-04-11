/*
 * QConfig - Quick way to implement a configuration file
 *
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

package qconfig

import (
	"os"
	"testing"

	. "github.com/minio-io/check"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

var version = Version{1, 0, 0}

func (s *MySuite) TestVersion(c *C) {
	cfg := NewConfig(version)
	c.Assert(cfg.GetVersion(), DeepEquals, version)
}

func (s *MySuite) TestGetSet(c *C) {
	defer os.RemoveAll("test.json")
	cfg := NewConfig(version)
	/*
		GetVersion() Version
		SetInt(string, int)
		GetInt(string) int
		SetIntList(string, []int)
		GetIntList(string) []int
		SetFloat64(string, float64)
		GetFloat64(string) float64
		SetString(string, string)
		GetString(string) string
		SetStringList(string, []string)
		GetStringList(string) []string
		SetMapString(string, map[string]string)
		GetMapString(string) map[string]string
		SetMapStringList(string, map[string][]string)
		GetMapStringList(string) map[string][]string
		SaveConfig(string) error
		LoadConfig(string) error
		String() string
	*/

	cfg.SetFloat64("Pi", 3.1415)
	pi := cfg.GetFloat64("Pi")
	c.Assert(pi, Equals, 3.1415)
	cfg.SaveConfig("test.json")

	newCfg := NewConfig(version)
	newCfg.SetInt("NewInt", 99)
	newCfg.LoadConfig("test.json")
	/*

		cfg.Set("MyDonut", "/media/disk1", "/media/disk2", "/media/badDisk99", "/media/badDisk100")
		cfg.Set("MyDonut1", "/media/disk1")
		cfg.SaveConfig("test.json")

		newCfg := NewConfig(1)
		newCfg.LoadConfig("test.json")

		fmt.Printf("%v\n", newCfg.String())

		c.Assert(err, IsNil)
		c.Assert(len(buckets), Equals, 0)

		err = donut.PutBucket("foo")
		c.Assert(err, Not(IsNil))

		bucketNamesProvided := []string{"bar", "foo"}
		c.Assert(bucketNamesReceived, DeepEquals, bucketNamesProvided)

		err = donut.PutBucket("foobar")
		c.Assert(err, IsNil)
		bucketNamesProvided = append(bucketNamesProvided, "foobar")

		buckets, err = donut.ListBuckets()
		c.Assert(err, IsNil)
		bucketNamesReceived = getBucketNames(buckets)
		c.Assert(bucketNamesReceived, DeepEquals, bucketNamesProvided)

		var actualData bytes.Buffer
		_, err = io.Copy(&actualData, reader)
		c.Assert(err, IsNil)
		c.Assert(actualData.Bytes(), DeepEquals, []byte(data))
		c.Assert(int64(actualData.Len()), Equals, size)
	*/

}
