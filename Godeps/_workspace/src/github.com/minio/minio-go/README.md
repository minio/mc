# Minimal object storage library in Go [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Install [![Build Status](https://travis-ci.org/minio/minio-go.svg)](https://travis-ci.org/minio/minio-go)

```sh
$ go get github.com/minio/minio-go
```
## Example

```go
package main

import (
	"log"

	"github.com/minio/minio-go"
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
* [MakeBucket(bucket, acl) error](examples/s3/makebucket.go)
* [BucketExists(bucket) error](examples/s3/bucketexists.go)
* [RemoveBucket(bucket) error](examples/s3/removebucket.go)
* [GetBucketACL(bucket) (BucketACL, error)](examples/s3/getbucketacl.go)
* [SetBucketACL(bucket, BucketACL) error)](examples/s3/setbucketacl.go)
* [ListObjects(bucket, prefix, recursive) <-chan ObjectStat](examples/s3/listobjects.go)
* [ListBuckets() <-chan BucketStat](examples/s3/listbuckets.go)
* [DropAllIncompleteUploads(bucket) <-chan error](examples/s3/dropallincompleteuploads.go)
* [DropIncompleteUploads(bucket, object) <-chan error](examples/s3/dropincompleteuploads.go)

### Object Level
* [PutObject(bucket, object, size, io.Reader) error](examples/s3/putobject.go)
* [GetObject(bucket, object) (io.Reader, ObjectStat, error)](examples/s3/getobject.go)
* [GetPartialObject(bucket, object, offset, length) (io.Reader, ObjectStat, error)](examples/s3/getpartialobject.go)
* [StatObject(bucket, object) (ObjectStat, error)](examples/s3/statobject.go)
* [RemoveObject(bucket, object) error](examples/s3/removeobject.go)

### API Reference

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/minio/minio-go)

## Contribute

[Contributors Guide](./CONTRIBUTING.md)
