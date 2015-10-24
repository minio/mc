# Minio Go Library for Amazon S3 Legacy Signature v2 Compatible Cloud Storage [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Install

```sh
$ go get github.com/minio/minio-go-legacy
```
## Example

```go
package main

import (
	"log"

	"github.com/minio/minio-go-legacy"
)

func main() {
	config := minio.Config{
		AccessKeyID:     "YOUR-ACCESS-KEY-HERE",
		SecretAccessKey: "YOUR-PASSWORD-HERE",
		Endpoint:        "https://s3.amazonaws.com",
	}
	s3Client, err := minio.New(config)
	if err != nil {
	    log.Fatalln(err)
	}
	for bucket := range s3Client.ListBuckets() {
		if bucket.Err != nil {
			log.Fatalln(bucket.Err)
		}
		log.Println(bucket.Stat)
	}
}
```

## Documentation

### Bucket Level
* [MakeBucket(bucket, acl) error](examples/makebucket.go)
* [BucketExists(bucket) error](examples/bucketexists.go)
* [RemoveBucket(bucket) error](examples/removebucket.go)
* [GetBucketACL(bucket) (BucketACL, error)](examples/getbucketacl.go)
* [SetBucketACL(bucket, BucketACL) error)](examples/setbucketacl.go)
* [ListBuckets() <-chan BucketStat](examples/listbuckets.go)
* [ListObjects(bucket, prefix, recursive) <-chan ObjectStat](examples/listobjects.go)
* [ListIncompleteUploads(bucket, prefix, recursive) <-chan ObjectMultipartStat](examples/listincompleteuploads.go)

### Object Level
* [PutObject(bucket, object, size, io.Reader) error](examples/putobject.go)
* [GetObject(bucket, object) (io.Reader, ObjectStat, error)](examples/getobject.go)
* [GetPartialObject(bucket, object, offset, length) (io.Reader, ObjectStat, error)](examples/getpartialobject.go)
* [StatObject(bucket, object) (ObjectStat, error)](examples/statobject.go)
* [RemoveObject(bucket, object) error](examples/removeobject.go)
* [RemoveIncompleteUpload(bucket, object) <-chan error](examples/removeincompleteupload.go)

### Presigned Bucket/Object Level
* [PresignedGetObject(bucket, object, time.Duration) (string, error)](examples/s3/presignedgetobject.go)
* [PresignedPutObject(bucket, object, time.Duration) (string, error)](examples/s3/presignedputobject.go)
* [PresignedPostPolicy(NewPostPolicy()) (map[string]string, error)](examples/s3/presignedpostpolicy.go)

### API Reference

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/minio/minio-go-legacy)

## Contribute

[Contributors Guide](./CONTRIBUTING.md)

[![Build Status](https://travis-ci.org/minio/minio-go-legacy.svg)](https://travis-ci.org/minio/minio-go-legacy) [![Build status](https://ci.appveyor.com/api/projects/status/1ep7n2resn6fk1w6?svg=true)](https://ci.appveyor.com/project/harshavardhana/minio-go)
