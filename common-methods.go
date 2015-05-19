/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"runtime"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/client/fs"
	"github.com/minio/mc/pkg/client/s3"
	"github.com/minio/minio/pkg/iodine"
)

// getSource -
func getSource(sourceURL string, sourceConfig *hostConfig) (reader io.Reader, length uint64, md5hex string, err error) {
	sourceClnt, err := getNewClient(sourceURL, sourceConfig, globalDebugFlag)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"failedURL": sourceURL})
	}
	return sourceClnt.GetObject(0, 0)
}

// putTarget -
func putTarget(targetURL string, targetConfig *hostConfig, md5hex string, length uint64, data io.Reader) error {
	targetClnt, err := getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return iodine.New(err, nil)
	}
	md5bytes, err := hex.DecodeString(md5hex)
	if err != nil {
		return iodine.New(err, nil)
	}
	md5Base64 := base64.StdEncoding.EncodeToString(md5bytes)
	err = targetClnt.CreateObject(md5Base64, length, data)
	if err != nil {
		return iodine.New(err, map[string]string{"failedURL": targetURL})
	}
	return nil
}

// getNewClient gives a new client interface
func getNewClient(urlStr string, auth *hostConfig, debug bool) (clnt client.Client, err error) {
	url := client.Parse(urlStr)
	if url == nil {
		return nil, iodine.New(errInvalidURL{url: urlStr}, nil)
	}
	switch url.Type {
	case client.Object: // Minio and S3 compatible object storage
		if auth == nil {
			return nil, iodine.New(errInvalidArgument{}, nil)
		}
		s3Config := new(s3.Config)
		s3Config.AccessKeyID = auth.AccessKeyID
		s3Config.SecretAccessKey = auth.SecretAccessKey
		s3Config.AppName = "Minio"
		s3Config.AppVersion = Version
		s3Config.AppComments = []string{os.Args[0], runtime.GOOS, runtime.GOARCH}
		s3Config.HostURL = urlStr
		s3Config.Debug = debug
		return s3.New(s3Config)
	case client.Filesystem:
		return fs.New(urlStr)
	}
	return nil, iodine.New(errInvalidURL{url: urlStr}, nil)
}
