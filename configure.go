package main

import (
	"encoding/json"
	"log"
	"os"
	"path"

	"github.com/codegangsta/cli"
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

func doConfigure(c *cli.Context) {
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
	s3File, err = os.OpenFile(path.Join(home, AUTH), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer s3File.Close()
	if err != nil {
		log.Fatal(err)
	}

	_, err = s3File.Write(jAuth)
	if err != nil {
		log.Fatal(err)
	}
}
