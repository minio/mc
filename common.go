/*
 * Mini Object Storage, (C) 2014,2015 Minio, Inc.
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
	"encoding/json"
	"os"
	"os/user"
	"path"

	"github.com/minio-io/mc/pkg/s3"
)

func getAuthFilePath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return path.Join(u.HomeDir, Auth), nil
}

func getAWSEnvironment() (auth *s3.Auth, err error) {
	var s3Auth *os.File
	var accessKey, secretKey, endpoint string

	authFile, err := getAuthFilePath()
	if err != nil {
		return nil, err
	}
	s3Auth, err = os.Open(authFile)
	defer s3Auth.Close()

	if err != nil {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		endpoint = os.Getenv("S3_ENDPOINT")
		if endpoint == "" {
			endpoint = "s3.amazonaws.com"
		}
		auth = &s3.Auth{
			AccessKey:       accessKey,
			SecretAccessKey: secretKey,
			Endpoint:        endpoint,
		}
	} else {
		var n int
		s3Authbytes := make([]byte, 256)
		n, err = s3Auth.Read(s3Authbytes)
		err = json.Unmarshal(s3Authbytes[:n], &auth)
		if err != nil {
			return nil, err
		}
	}
	return auth, nil
}
