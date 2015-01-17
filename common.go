/*
 * Mini Object Storage, (C) 2014 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
