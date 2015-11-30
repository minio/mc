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

If you do not have a working Golang environment, please follow [Install Golang](./INSTALLGO.md).

```sh
$ go get github.com/minio/minio-go
```

## Example

### ListBuckets()

This example shows how to List your buckets.

```go
package main

import (
	"log"

	"github.com/minio/minio-go"
)

func main() {
	config := minio.Config{
		Endpoint:        "https://s3.amazonaws.com",
		AccessKeyID:     "YOUR-ACCESS-KEY-HERE",
		SecretAccessKey: "YOUR-PASSWORD-HERE",
	}

	// Default is Signature Version 4. To enable Signature Version 2 do the following.
	// config.Signature = minio.SignatureV2

	s3Client, err := minio.New(config)
	if err != nil {
	    log.Fatalln(err)
	}
	for bucket := range s3Client.ListBuckets() {
		if bucket.Err != nil {
			log.Fatalln(bucket.Err)
		}
		log.Println(bucket)
	}
}
```

## Documentation

### Bucket Operations.
* [MakeBucket(bucketName, BucketACL) error](examples/s3/makebucket.go)
* [BucketExists(bucketName) error](examples/s3/bucketexists.go)
* [RemoveBucket(bucketName) error](examples/s3/removebucket.go)
* [GetBucketACL(bucketName) (BucketACL, error)](examples/s3/getbucketacl.go)
* [SetBucketACL(bucketName, BucketACL) error)](examples/s3/setbucketacl.go)
* [ListBuckets() <-chan BucketStat](examples/s3/listbuckets.go)
* [ListObjects(bucketName, prefix, recursive) <-chan ObjectStat](examples/s3/listobjects.go)
* [ListIncompleteUploads(bucketName, prefix, recursive) <-chan ObjectMultipartStat](examples/s3/listincompleteuploads.go)

### Object Operations.
* [PutObject(bucketName, objectName, contentType, io.ReadSeeker) error](examples/s3/putobject.go)
* [GetObject(bucketName, objectName) (io.ReadSeeker, error)](examples/s3/getobject.go)
* [GetPartialObject(bucketName, objectName, offset, length) (io.ReadSeeker, error)](examples/s3/getpartialobject.go)
* [StatObject(bucketName, objectName) (ObjectStat, error)](examples/s3/statobject.go)
* [RemoveObject(bucketName, objectName) error](examples/s3/removeobject.go)
* [RemoveIncompleteUpload(bucketName, objectName) <-chan error](examples/s3/removeincompleteupload.go)

### Presigned Operations.
* [PresignedGetObject(bucketName, objectName, time.Duration) (string, error)](examples/s3/presignedgetobject.go)
* [PresignedPutObject(bucketName, objectName, time.Duration) (string, error)](examples/s3/presignedputobject.go)
* [PresignedPostPolicy(NewPostPolicy()) (map[string]string, error)](examples/s3/presignedpostpolicy.go)

### API Reference

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/minio/minio-go)

## Contribute

[Contributors Guide](./CONTRIBUTING.md)

[![Build Status](https://travis-ci.org/minio/minio-go.svg)](https://travis-ci.org/minio/minio-go) [![Build status](https://ci.appveyor.com/api/projects/status/1ep7n2resn6fk1w6?svg=true)](https://ci.appveyor.com/project/harshavardhana/minio-go)
