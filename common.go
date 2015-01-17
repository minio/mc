package main

import (
	"errors"
	"os"

	"github.com/codegangsta/cli"
)

type MinioClient struct {
	bucketName string
	keyName    string
	body       string
	bucketAcls string
	policy     string
	region     string
	query      string // TODO
}

var Options = []cli.Command{
	Cp,
	Ls,
	Mb,
	Mv,
	Rb,
	Rm,
	Sync,
	GetObject,
	PutObject,
	ListObjects,
	ListBuckets,
	Configure,
}

func getAWSEnvironment() (accessKey, secretKey string, err error) {
	accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	if accessKey == "" && secretKey == "" {
		errstr := `You can configure your credentials by running "mc configure"`
		return "", "", errors.New(errstr)
	}
	if accessKey == "" {
		errstr := `Partial credentials found in the env, missing : AWS_ACCESS_KEY_ID`
		return "", "", errors.New(errstr)
	}

	if secretKey == "" {
		errstr := `Partial credentials found in the env, missing : AWS_SECRET_ACCESS_KEY`
		return "", "", errors.New(errstr)
	}

	return accessKey, secretKey, nil
}
