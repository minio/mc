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

// NewClient - get new client
// TODO refactor this to be more testable
func getNewClient(urlStr string, debug bool) (clnt client.Client, err error) {
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
		if hostCfg.Auth == nil {
			return nil, iodine.New(errInvalidAuth{}, nil)
		}
		auth := new(s3.Auth)
		if hostCfg.Auth.AccessKeyID != "" || hostCfg.Auth.SecretAccessKey != "" {
			auth.AccessKeyID = hostCfg.Auth.AccessKeyID
			auth.SecretAccessKey = hostCfg.Auth.SecretAccessKey
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
