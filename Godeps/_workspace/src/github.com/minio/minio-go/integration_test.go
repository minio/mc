// +build ignore

/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

package integration_test

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/minio/minio-go"
)

// SET THESE!
var bucket = ""
var accessKeyID = ""
var secretAccessKey = ""

var config = minio.Config{
	AccessKeyID:     accessKeyID,
	SecretAccessKey: secretAccessKey,
	Endpoint:        "https://s3-us-west-2.amazonaws.com",
}
var client, _ = minio.New(config)

var standardConfig = minio.Config{
	AccessKeyID:     accessKeyID,
	SecretAccessKey: secretAccessKey,
	Endpoint:        "https://s3.amazonaws.com",
}
var standardClient, _ = minio.New(standardConfig)

func TestMakeBucket(t *testing.T) {
	err := client.MakeBucket(bucket, "private")
	if err == nil {
		t.Fail()
	}
}

func TestMakeBucketAuthenticatedRead(t *testing.T) {
	client.RemoveBucket(bucket + "-auth")
	err := client.MakeBucket(bucket+"-auth", "authenticated-read")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	bucketAcl, err := client.GetBucketACL(bucket + "-auth")
	if err != nil {
		t.Error(err)
	}
	if bucketAcl != "authenticated-read" {
		t.Error("not authenticated-read")
	}
	client.RemoveBucket(bucket + "-auth")

	bucketAcl, err = client.GetBucketACL(bucket + "-auth")
	if err == nil {
		t.Error(err)
	}
}

func TestRemoveBucketDifferentRegion(t *testing.T) {
	err := client.RemoveBucket("goroutine-standard-region")
	if err == nil {
		t.Fail()
	}
}

func TestRemoveBucketNonExistantBucket(t *testing.T) {
	err := client.RemoveBucket("goroutine-not-a-bucket")
	log.Println(err)
	if err == nil {
		t.Fail()
	}
}

func TestRemoveBucketNotOwned(t *testing.T) {
	err := standardClient.RemoveBucket("bucket")
	if err == nil {
		t.Fail()
	}
	if len(err.Error()) == 0 {
		t.Error("Error message should be populated")
	}
}

func TestRemoveBucketNotOwnedAnotherRegion(t *testing.T) {
	err := standardClient.RemoveBucket("bucket")
	if err == nil {
		t.Fail()
	}
	if len(err.Error()) == 0 {
		t.Error("Error message should be populated")
	}
}

func TestMakeBucketStandardRegion(t *testing.T) {
	err := client.MakeBucket(bucket+"-standard", "private")
	if err == nil {
		t.Fail()
	}
}

func TestBucketAcls(t *testing.T) {
	err := client.SetBucketACL(bucket, "public-read-write")
	if err != nil {
		t.Error("Set prw failed")
	}
	acl, err := client.GetBucketACL(bucket)
	if err != nil {
		t.Error("get prw failed")
	}
	if acl.String() != "public-read-write" {
		t.Error("prw wasn't prw")
	}

	err = client.SetBucketACL(bucket, "public-read")
	if err != nil {
		t.Error("set pr failed")
	}
	acl, err = client.GetBucketACL(bucket)
	if err != nil {
		t.Error("get pr failed")
	}
	if acl.String() != "public-read" {
		t.Error("pr wasn't pr")
	}

	err = client.SetBucketACL(bucket, "authenticated-read")
	if err != nil {
		t.Error("set ar failed")
	}
	acl, err = client.GetBucketACL(bucket)
	if err != nil {
		t.Error("get ar failed")
	}
	if acl.String() != "authenticated-read" {
		t.Error("ar wasn't ar")
	}

	err = client.SetBucketACL(bucket, "private")
	if err != nil {
		t.Error("set p failed")
	}
	acl, err = client.GetBucketACL(bucket)
	if err != nil {
		t.Error("get p failed")
	}
	if acl.String() != "private" {
		t.Error("p wasn't p")
	}
}

func TestListBucket(t *testing.T) {
	buckets := client.ListBuckets()
	for b := range buckets {
		if b.Err != nil {
			t.Error(b.Err)
		}
		log.Println(b)
	}
}

func TestBucketExists(t *testing.T) {
	if err := client.BucketExists(bucket); err != nil {
		if len(err.Error()) == 0 {
			t.Error("emnpty error")
		}
		t.Error(err)
	}
}

func TestBucketExistsOwnedByOtherUser(t *testing.T) {
	err := standardClient.BucketExists("bucket")
	if err == nil {
		t.Error("Should return an error")
	}
	if len(err.Error()) == 0 {
		t.Error("emnpty error")
	}
}

func TestBucketOnAnotherRegion(t *testing.T) {
	err := standardClient.BucketExists(bucket)
	log.Println(bucket)
	log.Println("err:", err)
	if err != nil {
		if len(err.Error()) == 0 {
			t.Error("emnpty error")
		}
	} else {
		t.Error("Expecting error")
	}
}

func TestBucketOnAnotherRegionOwnedByAnotherUser(t *testing.T) {
	err := standardClient.BucketExists("bucket")
	log.Println(bucket)
	log.Println("err:", err)
	if err != nil {
		if len(err.Error()) == 0 {
			t.Error("emnpty error")
		}
	} else {
		t.Error("Expecting error")
	}
}

func TestPutSmallObject(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/obj"
	if err := client.PutObject(bucket, key, "", 11, input); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallObjectTooSmall(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/toosmall"
	if err := client.PutObject(bucket, key, "", 10, input); err == nil {
		t.Error("Should fail when length is too small")
	}
	_, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
}

func TestPutSmallObjectTooLarge(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/toolarge"
	if err := client.PutObject(bucket, key, "", 12, input); err == nil {
		t.Error("Should fail when length is too large")
	}
	_, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
}

func TestPutSmallTextFile(t *testing.T) {
	key := "small/text_file"
	fileName := "/usr/bin/ldd"
	file, err := os.Open(fileName)
	defer file.Close()
	if err != nil {
		t.Error(err)
	}
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Error(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallBinaryFile(t *testing.T) {
	key := "small/binary_file"
	fileName := "/bin/ls"
	file, err := os.Open(fileName)
	defer file.Close()
	if err != nil {
		t.Error(err)
	}
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Error(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallObjectContentType(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/text_plain"
	if err := client.PutObject(bucket, key, "text/plain", 11, input); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallObjectWithQuestionMark(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/obj?ect"
	if err := client.PutObject(bucket, key, "", 11, input); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallObjectWithHash(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/obj#ect"
	if err := client.PutObject(bucket, key, "", 11, input); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallObjectWithUnicode1(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/世界"
	if err := client.PutObject(bucket, key, "", 11, input); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallObjectWithUnicode2(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/世界世"
	if err := client.PutObject(bucket, key, "", 11, input); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutSmallObjectWithUnicode3(t *testing.T) {
	input := bytes.NewBufferString("hello world")
	key := "small/世界世界"
	if err := client.PutObject(bucket, key, "", 11, input); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutLargeTextFile(t *testing.T) {
	key := "large/text_file"
	fileName := "/tmp/11m_text"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	log.Println(stat)
}

func TestPutLargeBinaryFile(t *testing.T) {
	key := "large/binary_file"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	if stat.Key != key {
		t.Error("invalid stat")
	}
}

func TestPutLargeSizeTooSmall(t *testing.T) {
	key := "large/toosmall"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err = client.PutObject(bucket, key, "", fileStat.Size()-1, file); err == nil {
		t.Error("Should fail when length is too large")
	}
}

func TestPutLargeSizeTooLarge(t *testing.T) {
	key := "large/toolarge"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err = client.PutObject(bucket, key, "", fileStat.Size()+1, file); err == nil {
		t.Error("Should fail when length is too large")
	}
}

func TestPutLargeBinaryFileContentType(t *testing.T) {
	key := "large/text_plain"
	fileName := "/tmp/11m_text"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "plain/text", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	if stat.Key != key {
		t.Error("invalid stat")
	}
}

func TestPutLargeBinaryFileWithQuestionMark(t *testing.T) {
	key := "large/obj?ect"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	if stat.Key != key {
		t.Error("invalid stat")
	}
}

func TestPutLargeBinaryFileWithHashMark(t *testing.T) {
	key := "large/obj#ect"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	if stat.Key != key {
		t.Error("invalid stat")
	}
}

func TestPutLargeBinaryFileWithUnicode1(t *testing.T) {
	key := "large/世界"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	if stat.Key != key {
		t.Error("invalid stat")
	}
}

func TestPutLargeBinaryFileWithUnicode2(t *testing.T) {
	key := "large/世界世"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error("Should fail when length is too large")
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	if stat.Key != key {
		t.Error("invalid stat")
	}
}

func TestPutLargeBinaryFileWithUnicode3(t *testing.T) {
	errCh := client.DropAllIncompleteUploads(bucket)
	for e := range errCh {
		t.Error(e)
	}
	key := "large/世界世界"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size(), file); err != nil {
		t.Error(err)
	}
	stat, err := client.StatObject(bucket, key)
	if err != nil {
		t.Error(err)
	}
	if stat.Key != key {
		t.Error("invalid stat")
	}
}

func TestPutLargeBinaryFileWithUnicode4(t *testing.T) {
	key := "large/4世界世界世"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size()-1, file); err == nil {
		t.Error("Should fail when length is too large")
	}
}

func TestPutLargeBinaryFileWithUnicode5(t *testing.T) {
	key := "large/5世界世界世界"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size()-1, file); err == nil {
		t.Error("Should fail when length is too large")
	}
}

func TestPutLargeBinaryFileWithUnicode6(t *testing.T) {
	key := "large/6世界世界世界世"
	fileName := "/tmp/11m_binary"
	file, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStat, err := os.Stat(fileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PutObject(bucket, key, "", fileStat.Size()-1, file); err == nil {
		t.Error("Should fail when length is too large")
	}
}

func TestDropAllIncomplete(t *testing.T) {
	errCh := client.DropAllIncompleteUploads(bucket)
	for err := range errCh {
		t.Error(err)
	}
}
