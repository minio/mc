package main

import (
	"errors"
	"hash"
	"io"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

type fsMeta struct {
	bucket string
	body   string
	key    string
	get    bool
	put    bool
}

func parseCpOptions(c *cli.Context) (fsmeta fsMeta, err error) {
	var localPath, s3Path string
	var get, put bool
	switch len(c.Args()) {
	case 1:
		return fsMeta{}, errors.New("Missing <S3Path> or <LocalPath>")
	case 2:
		if strings.HasPrefix(c.Args().Get(0), "s3://") {
			s3Path = c.Args().Get(0)
			localPath = c.Args().Get(1)
			get = true
			put = false
		} else if strings.HasPrefix(c.Args().Get(1), "s3://") {
			s3Path = c.Args().Get(1)
			localPath = c.Args().Get(0)
			get = false
			put = true
		}
	default:
		return fsMeta{}, errors.New("Arguments missing <S3Path> or <LocalPath>")
	}
	fsmeta.bucket = strings.Split(s3Path, "s3://")[1]
	fsmeta.body = localPath
	fsmeta.key = localPath
	fsmeta.get = get
	fsmeta.put = put

	return fsmeta, nil
}

func doFsCopy(c *cli.Context) {
	var auth *s3.Auth
	var err error
	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	s3c := s3.NewS3Client(auth)

	var fsmeta fsMeta
	fsmeta, err = parseCpOptions(c)
	if err != nil {
		log.Fatal(err)
	}
	var bodyFile *os.File
	bodyFile, err = os.Open(fsmeta.body)
	defer bodyFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	if fsmeta.put {
		var bodyBuffer io.Reader
		var size int64
		var md5hash hash.Hash
		md5hash, bodyBuffer, size, err = getPutMetadata(bodyFile)
		if err != nil {
			log.Fatal(err)
		}

		err = s3c.Put(fsmeta.bucket, fsmeta.key, md5hash, size, bodyBuffer)
		if err != nil {
			log.Fatal(err)
		}
	} else if fsmeta.get {
		var objectReader io.ReadCloser
		var objectSize int64
		objectReader, objectSize, err = s3c.Get(fsmeta.bucket, fsmeta.key)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.CopyN(bodyFile, objectReader, objectSize)
		if err != nil {
			log.Fatal(err)
		}
	}
}
