# Minio Go Library for Amazon S3 Compatible Cloud Storage [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Description

Minio Go library is a simple client library for S3 compatible cloud storage servers. Supports AWS Signature Version 4 and 2. AWS Signature Version 4 is chosen as default.

List of supported cloud storage providers.

 - AWS Signature Version 4
   - Amazon S3
   - Minio

 - AWS Signature Version 2
   - Google Cloud Storage (Compatibility Mode) 
   - Openstack Swift + Swift3 middleware
   - Ceph Object Gateway
   - Riak CS

## Install

```sh
$ go get github.com/minio/minio-go
```

## Example

### ListBuckets() - AWS Signature Version 4.

This example shows how to List your buckets with default AWS Signature Version 4.

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

### ListBuckets() - AWS Signature Version 2.

This example shows how to List your buckets with AWS Signature Version 2.

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
		Signature:       minio.SignatureV2,        
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
* [MakeBucket(bucket, acl) error](examples/s3-v4/makebucket.go)
* [BucketExists(bucket) error](examples/s3-v4/bucketexists.go)
* [RemoveBucket(bucket) error](examples/s3-v4/removebucket.go)
* [GetBucketACL(bucket) (BucketACL, error)](examples/s3-v4/getbucketacl.go)
* [SetBucketACL(bucket, BucketACL) error)](examples/s3-v4/setbucketacl.go)
* [ListBuckets() <-chan BucketStat](examples/s3-v4/listbuckets.go)
* [ListObjects(bucket, prefix, recursive) <-chan ObjectStat](examples/s3-v4/listobjects.go)
* [ListIncompleteUploads(bucket, prefix, recursive) <-chan ObjectMultipartStat](examples/s3-v4/listincompleteuploads.go)

### Object Level
* [PutObject(bucket, object, size, io.Reader) error](examples/s3-v4/putobject.go)
* [GetObject(bucket, object) (io.Reader, ObjectStat, error)](examples/s3-v4/getobject.go)
* [GetPartialObject(bucket, object, offset, length) (io.Reader, ObjectStat, error)](examples/s3-v4/getpartialobject.go)
* [StatObject(bucket, object) (ObjectStat, error)](examples/s3-v4/statobject.go)
* [RemoveObject(bucket, object) error](examples/s3-v4/removeobject.go)
* [RemoveIncompleteUpload(bucket, object) <-chan error](examples/s3-v4/removeincompleteupload.go)

### Presigned Bucket/Object Level
* [PresignedGetObject(bucket, object, time.Duration) (string, error)](examples/s3-v4/presignedgetobject.go)
* [PresignedPutObject(bucket, object, time.Duration) (string, error)](examples/s3-v4/presignedputobject.go)
* [PresignedPostPolicy(NewPostPolicy()) (map[string]string, error)](examples/s3-v4/presignedpostpolicy.go)

## Additional Documentation

More examples with AWS Signature Version 2 can be found [here](examples/s3-v2)

### API Reference

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/minio/minio-go)

## Contribute

[Contributors Guide](./CONTRIBUTING.md)

[![Build Status](https://travis-ci.org/minio/minio-go.svg)](https://travis-ci.org/minio/minio-go) [![Build status](https://ci.appveyor.com/api/projects/status/1ep7n2resn6fk1w6?svg=true)](https://ci.appveyor.com/project/harshavardhana/minio-go)
