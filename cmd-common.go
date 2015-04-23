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
	"net"
	"path/filepath"
	"time"

	"io"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/client/fs"
	"github.com/minio-io/mc/pkg/client/s3"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

// isValidRetry - check if we should retry for the given error sequence
func isValidRetry(err error) bool {
	err = iodine.New(err, nil)
	if err == nil {
		return false
	}
	// DNSError, Network Operation error
	switch e := iodine.ToError(err).(type) {
	case *net.DNSError:
		return true
	case *net.OpError:
		switch e.Op {
		case "read", "write", "dial":
			return true
		}
	}
	return false
}

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

// clientManager interface for mock tests
type clientManager interface {
	getSourceReader(sourceURL string, sourceConfig *hostConfig) (reader io.ReadCloser, length int64, md5hex string, err error)
	getTargetWriter(targetURL string, targetConfig *hostConfig, md5Hex string, length int64) (io.WriteCloser, error)
	getNewClient(urlStr string, config *hostConfig, debug bool) (clnt client.Client, err error)
}

type mcClientManager struct{}

// getSourceReader -
func (manager mcClientManager) getSourceReader(sourceURL string, sourceConfig *hostConfig) (
	reader io.ReadCloser, length int64, md5hex string, err error) {

	sourceClnt, err := manager.getNewClient(sourceURL, sourceConfig, globalDebugFlag)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"failedURL": sourceURL})
	}
	// check if the bucket is valid
	// For object storage URL's do a StatBucket(), not necessary for fs client
	if client.GetType(sourceURL) != client.Filesystem {
		if err := sourceClnt.Stat(); err != nil {
			return nil, 0, "", iodine.New(err, map[string]string{"failedURL": sourceURL})
		}
	}
	return sourceClnt.Get()
}

// getTargetWriter -
func (manager mcClientManager) getTargetWriter(targetURL string, targetConfig *hostConfig, md5Hex string, length int64) (
	io.WriteCloser, error) {
	targetClnt, err := manager.getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	// check if bucket is valid, if not create it on target
	// For object storage URL's do a StatBucket() and PutBucket(), not necessary for fs client
	if client.GetType(targetURL) != client.Filesystem {
		if err := targetClnt.Stat(); err != nil {
			switch iodine.ToError(err).(type) {
			case s3.BucketNotFound:
				err := targetClnt.PutBucket("")
				if err != nil {
					return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
				}
			default:
				return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
			}
		}
	}
	return targetClnt.Put(md5Hex, length)
}

// getNewClient gives a new client interface
func (manager mcClientManager) getNewClient(urlStr string, auth *hostConfig, debug bool) (clnt client.Client, err error) {
	switch client.GetType(urlStr) {
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
			scheme: client.GetType(urlStr),
			url:    urlStr,
		}, nil)
	}
}

// getTargetWriters -
func getTargetWriters(manager clientManager, targetURLConfigMap map[string]*hostConfig, md5Hex string, length int64) (
	[]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for targetURL, targetConfig := range targetURLConfigMap {
		writer, err := manager.getTargetWriter(targetURL, targetConfig, md5Hex, length)
		if err != nil {
			// close all writers
			for _, targetWriter := range targetWriters {
				targetWriter.Close()
			}
			return nil, iodine.New(err, nil)
		}
		targetWriters = append(targetWriters, writer)
	}
	return targetWriters, nil
}
