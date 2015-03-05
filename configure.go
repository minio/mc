package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func parseConfigureInput(c *cli.Context) (auth *s3.Auth, err error) {
	accessKey := c.String("accesskey")
	secretKey := c.String("secretkey")
	endpoint := c.String("endpoint")
	pathstyle := c.Bool("pathstyle")

	if accessKey == "" {
		return nil, errAccess
	}
	if secretKey == "" {
		return nil, errSecret
	}
	if endpoint == "" {
		return nil, errEndpoint
	}

	auth = &s3.Auth{
		AccessKey:        accessKey,
		SecretAccessKey:  secretKey,
		Endpoint:         endpoint,
		S3ForcePathStyle: pathstyle,
	}

	return auth, nil
}

func doConfigure(c *cli.Context) {
	var err error
	var jAuth []byte
	var auth *s3.Auth
	auth, err = parseConfigureInput(c)
	if err != nil {
		log.Fatal(err)
	}

	jAuth, err = json.MarshalIndent(auth, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	var s3File *os.File
	home := os.Getenv("HOME")
	s3File, err = os.OpenFile(path.Join(home, Auth), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	defer s3File.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = s3File.Write(jAuth)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("")
	fmt.Println("Configuration written to", path.Join(home, Auth))
}
