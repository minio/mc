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
	"path"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

var options = []cli.Command{
	Cp,
	Ls,
	Mb,
	Sync,
	Configure,
}

func getAuthFilePath() string {
	home := os.Getenv("HOME")
	return path.Join(home, AUTH)
}

func getAWSEnvironment() (auth *s3.Auth, err error) {
	var s3Auth *os.File
	var accessKey, secretKey string
	s3Auth, err = os.OpenFile(getAuthFilePath(), os.O_RDWR, 0666)
	defer s3Auth.Close()
	if err != nil {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		if accessKey == "" && secretKey == "" {
			return nil, missingAccessSecretErr
		}
		if accessKey == "" {
			return nil, missingAccessErr
		}

		if secretKey == "" {
			return nil, missingSecretErr
		}
		auth = &s3.Auth{
			AccessKey: accessKey,
			SecretKey: secretKey,
			Hostname:  "s3.amazonaws.com",
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
