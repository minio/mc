package main

import (
	"errors"
	"io"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func parseGetObject(c *cli.Context) (bucket, key string, err error) {
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

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if accessKey == "" && secretKey == "" {
		errstr := `You can configure your credentials by running "mc configure"`
		log.Fatal(errstr)
	}
	if accessKey == "" {
		errstr := `Partial credentials found in the env, missing : AWS_ACCESS_KEY_ID`
		log.Fatal(errstr)
	}

	if secretKey == "" {
		errstr := `Partial credentials found in the env, missing : AWS_SECRET_ACCESS_KEY`
		log.Fatal(errstr)
	}

	bucket, key, err = parseGetObject(c)
	if err != nil {
		log.Fatal(err)
	}
	s3c := s3.NewS3Client(accessKey, secretKey)
	objectReader, objectSize, err = s3c.Get(bucket, key)
	if err != nil {
		log.Fatal(err)
	}
	io.CopyN(os.Stdout, objectReader, objectSize)
}
