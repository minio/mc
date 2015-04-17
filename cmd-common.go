/*
 * Mini Copy, (C) 2014, 2015 Minio, Inc.
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
	"net/url"
	"path"
	"time"

	"io"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/client/fs"
	"github.com/minio-io/mc/pkg/client/s3"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

// StartBar -- instantiate a progressbar
func startBar(size int64) *pb.ProgressBar {
	bar := pb.New(int(size))
	bar.SetUnits(pb.U_BYTES)
	bar.SetRefreshRate(time.Millisecond * 10)
	bar.NotPrint = true
	bar.ShowSpeed = true
	bar.Callback = func(s string) {
		// Colorize
		console.Info("\r" + s)
	}
	// Feels like wget
	bar.Format("[=> ]")
	return bar
}

func getMcBashCompletionFilename() (string, error) {
	configDir, err := getMcConfigDir()
	if err != nil {
		return "", err
	}
	return path.Join(configDir, "mc.bash_completion"), nil
}

func mustGetMcBashCompletionFilename() string {
	p, _ := getMcBashCompletionFilename()
	return p
}

type clientManager interface {
	getSourceReader(urlStr string) (reader io.ReadCloser, length int64, md5hex string, err error)
	getTargetWriter(urlStr string, md5Hex string, length int64) (io.WriteCloser, error)
	getNewClient(urlStr string, debug bool) (clnt client.Client, err error)
}

type mcClientManager struct{}

func (manager mcClientManager) getSourceReader(urlStr string) (reader io.ReadCloser, length int64, md5hex string, err error) {
	sourceClnt, err := manager.getNewClient(urlStr, globalDebugFlag)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": urlStr})
	}
	// Get a reader for the source object
	bucket, object, err := url2Object(urlStr)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": urlStr})
	}

	// check if the bucket is valid
	if err := sourceClnt.StatBucket(bucket); err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": urlStr})
	}
	return sourceClnt.Get(bucket, object)
}

func (manager mcClientManager) getTargetWriter(urlStr string, md5Hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := manager.getNewClient(urlStr, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	bucket, object, err := url2Object(urlStr)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	// check if bucket is valid
	if err := targetClnt.StatBucket(bucket); err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": urlStr})
	}
	return targetClnt.Put(bucket, object, md5Hex, length)
}

// getNewClient gets a new client
// TODO refactor this to be more testable
func (manager mcClientManager) getNewClient(urlStr string, debug bool) (clnt client.Client, err error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, iodine.New(errInvalidURL{url: urlStr}, nil)
	}
	switch getURLType(urlStr) {
	case urlS3: // Minio and S3 compatible object storage
		hostCfg, err := getHostConfig(u.String())
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		if hostCfg == nil {
			return nil, iodine.New(errInvalidAuth{}, nil)
		}
		auth := new(s3.Auth)
		if _, ok := hostCfg["Auth.AccessKeyID"]; ok {
			auth.AccessKeyID = hostCfg["Auth.AccessKeyID"]
		}
		if _, ok := hostCfg["Auth.SecretAccessKey"]; ok {
			auth.SecretAccessKey = hostCfg["Auth.SecretAccessKey"]
		}
		clnt = s3.GetNewClient(urlStr, auth, mcUserAgent, debug)
		return clnt, nil
	case urlFS:
		clnt = fs.GetNewClient(urlStr)
		return clnt, nil
	default:
		return nil, iodine.New(errUnsupportedScheme{scheme: getURLType(urlStr)}, nil)
	}
}

func getTargetWriters(manager clientManager, urls []string, md5Hex string, length int64) ([]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for _, u := range urls {
		writer, err := manager.getTargetWriter(u, md5Hex, length)
		if err != nil {
			// close all writers
			for _, targetWriter := range targetWriters {
				targetWriter.Close()
			}
			return nil, iodine.New(errInvalidURL{url: u}, nil)
		}
		targetWriters = append(targetWriters, writer)
	}
	return targetWriters, nil
}
