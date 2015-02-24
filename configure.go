package main

import (
	"encoding/json"
	"log"
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func parseConfigureInput(c *cli.Context) (accessKey, secretKey, endpoint string, err error) {
	accessKey = c.String("accesskey")
	secretKey = c.String("secretkey")
	endpoint = c.String("endpoint")
	if accessKey == "" {
		return "", "", "", configAccessErr
	}
	if secretKey == "" {
		return "", "", "", configSecretErr
	}
	if endpoint == "" {
		return "", "", "", configEndpointErr
	}
	return accessKey, secretKey, endpoint, nil
}

func doConfigure(c *cli.Context) {
	var err error
	var jAuth []byte
	var accessKey, secretKey, endpoint string
	accessKey, secretKey, endpoint, err = parseConfigureInput(c)
	if err != nil {
		log.Fatal(err)
	}

	auth := s3.NewAuth(accessKey, secretKey, endpoint)
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
