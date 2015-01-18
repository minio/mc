package main

import (
	"log"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func dumpBuckets(v []*s3.Bucket) {
	for _, b := range v {
		log.Printf("Bucket :%#v", b)
	}
}

func doListBuckets(c *cli.Context) {
	var accessKey, secretKey string
	var err error
	accessKey, secretKey, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	var buckets []*s3.Bucket
	s3c := s3.NewS3Client(accessKey, secretKey, "s3.amazonaws.com")
	buckets, err = s3c.Buckets()
	if err != nil {
		log.Fatal(err)
	}

	dumpBuckets(buckets)
}
