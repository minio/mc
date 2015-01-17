package main

import (
	"errors"
	"log"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func parseListObjectsInput(c *cli.Context) (bucket string, err error) {
	bucket = c.String("bucket")
	if bucket == "" {
		return "", errors.New("bucket name is mandatory")
	}
	return bucket, nil
}

func doListObjects(c *cli.Context) {
	var err error
	var accessKey, secretKey, bucket string
	accessKey, secretKey, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket, err = parseListObjectsInput(c)
	if err != nil {
		log.Fatal(err)
	}

	var items []*s3.Item
	s3c := s3.NewS3Client(accessKey, secretKey, "s3.amazonaws.com")
	// Gets 1000 maxkeys supported with GET Bucket API
	items, err = s3c.GetBucket(bucket, "", s3.MAX_OBJECT_LIST)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		log.Println(item)
	}
}
