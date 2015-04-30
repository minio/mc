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
	"io"
	"path/filepath"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/client/fs"
	"github.com/minio-io/mc/pkg/client/s3"
	"github.com/minio-io/minio/pkg/iodine"
)

/*
type sourceReader struct {
	reader io.ReadCloser
	length int64
	md5hex string
}
*/

// getSourceReader -
func getSourceReader(sourceURL string, sourceConfig *hostConfig) (reader io.ReadCloser, length int64, md5hex string, err error) {
	sourceClnt, err := getNewClient(sourceURL, sourceConfig, globalDebugFlag)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"failedURL": sourceURL})
	}
	if _, err := sourceClnt.Stat(); err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"failedURL": sourceURL})
	}
	return sourceClnt.Get()
}

// getTargetWriter -
func getTargetWriter(targetURL string, targetConfig *hostConfig, md5hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return targetClnt.Put(md5hex, length)
}

// getNewClient gives a new client interface
func getNewClient(urlStr string, auth *hostConfig, debug bool) (clnt client.Client, err error) {
	t := client.GetType(urlStr)
	switch t {
	case client.Object: // Minio and S3 compatible object storage
		if auth == nil {
			return nil, iodine.New(errInvalidArgument{}, nil)
		}
		s3Config := new(s3.Config)
		s3Config.AccessKeyID = auth.AccessKeyID
		s3Config.SecretAccessKey = auth.SecretAccessKey
		s3Config.UserAgent = mcUserAgent
		s3Config.HostURL = urlStr
		s3Config.Debug = debug
		clnt = s3.New(s3Config)
		return clnt, nil
	case client.Filesystem:
		clnt = fs.New(filepath.Clean(urlStr))
		return clnt, nil
	default:
		return nil, iodine.New(errUnsupportedScheme{
			scheme: client.GetTypeToString(t),
			url:    urlStr,
		}, nil)
	}
}
