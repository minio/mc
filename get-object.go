package main

import (
	"errors"
	"io"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func parseGetObjectInput(c *cli.Context) (bucket, key string, err error) {
	bucket = c.String("bucket")
	key = c.String("key")
	if bucket == "" {
		return "", "", errors.New("bucket name is mandatory")
	}
	if key == "" {
		return "", "", errors.New("object name is mandatory")
	}

	return bucket, key, nil
}

func doGetObject(c *cli.Context) {
	var bucket, key string
	var err error
	var objectReader io.ReadCloser
	var objectSize int64

	var accessKey, secretKey string
	accessKey, secretKey, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket, key, err = parseGetObjectInput(c)
	if err != nil {
		log.Fatal(err)
	}

	s3c := s3.NewS3Client(accessKey, secretKey)
	objectReader, objectSize, err = s3c.Get(bucket, key)
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.CopyN(os.Stdout, objectReader, objectSize)
	if err != nil {
		log.Fatal(err)
	}
}
