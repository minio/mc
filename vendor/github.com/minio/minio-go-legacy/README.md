# Minio Go Library for Amazon S3 Legacy v2 Signature Compatible Cloud Storage [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Install [![Build Status](https://travis-ci.org/minio/minio-go-legacy.svg)](https://travis-ci.org/minio/minio-go-legacy)

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
* [ListObjects(bucket, prefix, recursive) <-chan ObjectStat](examples/listobjects.go)
* [ListBuckets() <-chan BucketStat](examples/listbuckets.go)
* [DropAllIncompleteUploads(bucket) <-chan error](examples/dropallincompleteuploads.go)

### Object Level
* [PutObject(bucket, object, size, io.Reader) error](examples/putobject.go)
* [GetObject(bucket, object) (io.Reader, ObjectStat, error)](examples/getobject.go)
* [GetPartialObject(bucket, object, offset, length) (io.Reader, ObjectStat, error)](examples/getpartialobject.go)
* [StatObject(bucket, object) (ObjectStat, error)](examples/statobject.go)
* [RemoveObject(bucket, object) error](examples/removeobject.go)
* [DropIncompleteUpload(bucket, object) <-chan error](examples/dropincompleteuploads.go)
* [PresignedGetObject(bucket, object, expires) (string, error)](examples/presignedgetobject.go)

### API Reference

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/minio/minio-go-legacy)

## Contribute

[Contributors Guide](./CONTRIBUTING.md)
