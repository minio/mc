package main

import (
	"encoding/json"
	"log"
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/minio"
	"github.com/minio-io/mc/pkg/s3"
)

func parseConfigureInput(c *cli.Context) (accessKey, secretKey string, err error) {
	accessKey = c.String("accesskey")
	secretKey = c.String("secretkey")
	if accessKey == "" {
		return "", "", configAccessErr
	}
	if secretKey == "" {
		return "", "", configSecretErr
	}
	return accessKey, secretKey, nil
}

func doMinioConfigure(c *cli.Context) {
	hostname := c.String("hostname")
	if hostname == "" {
		log.Fatal("Invalid hostname")
	}

	caFile := c.String("cacert")
	if caFile == "" {
		log.Fatal("invalid CA file")
	}

	certFile := c.String("cert")
	if certFile == "" {
		log.Fatal("invalid certificate")
	}

	keyFile := c.String("key")
	if keyFile == "" {
		log.Fatal("invalid key")
	}

	/*
		var accessKey, secretKey string
		var err error
		accessKey, secretKey, err = parseConfigureInput(c)
		if err != nil {
			log.Fatal(err)
		}
	*/
	auth := &minio.Auth{
		//		AccessKey:       accessKey,
		//		SecretAccessKey: secretKey,
		Hostname: hostname,
		CACert:   caFile,
		CertPEM:  certFile,
		KeyPEM:   keyFile,
	}

	jAuth, err := json.Marshal(auth)
	if err != nil {
		log.Fatal(err)
	}

	var minioFile *os.File
	home := os.Getenv("HOME")
	minioFile, err = os.OpenFile(path.Join(home, MINIO_AUTH), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer minioFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	_, err = minioFile.Write(jAuth)
	if err != nil {
		log.Fatal(err)
	}

}

func doS3Configure(c *cli.Context) {
	var err error
	var jAuth []byte
	var accessKey, secretKey string
	accessKey, secretKey, err = parseConfigureInput(c)
	if err != nil {
		log.Fatal(err)
	}

	auth := s3.NewAuth(accessKey, secretKey, "s3.amazonaws.com")
	jAuth, err = json.Marshal(auth)
	if err != nil {
		log.Fatal(err)
	}

	var s3File *os.File
	home := os.Getenv("HOME")
	s3File, err = os.OpenFile(path.Join(home, S3_AUTH), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer s3File.Close()
	if err != nil {
		log.Fatal(err)
	}

	_, err = s3File.Write(jAuth)
	if err != nil {
		log.Fatal(err)
	}
}
