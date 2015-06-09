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
	"errors"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/client/fs"
	"github.com/minio/mc/pkg/client/s3"
	"github.com/minio/minio/pkg/iodine"
)

// getSource gets a reader from URL<
func getSource(sourceURL string) (reader io.ReadCloser, length int64, err error) {
	sourceClnt, err := source2Client(sourceURL)
	if err != nil {
		return nil, 0, iodine.New(err, map[string]string{"failedURL": sourceURL})
	}
	return sourceClnt.GetObject(0, 0)
}

// putTarget writes to URL from reader.
func putTarget(targetURL string, length int64, reader io.Reader) error {
	targetClnt, err := target2Client(targetURL)
	if err != nil {
		return iodine.New(err, nil)
	}
	err = targetClnt.PutObject(length, reader)
	if err != nil {
		return iodine.New(err, map[string]string{"failedURL": targetURL})
	}
	return nil
}

// putTargets writes to URL from reader.
func putTargets(targetURLs []string, length int64, reader io.Reader) <-chan error {
	tgtReaders := make([]io.Reader, len(targetURLs))
	tgtWriters := make([]io.Writer, len(targetURLs))
	tgtClients := make([]client.Client, len(targetURLs))
	var errorCh = make(chan error)
	for i, targetURL := range targetURLs {
		tgtReaders[i], tgtWriters[i] = io.Pipe()
		tgtClient, err := target2Client(targetURL)
		if err != nil {
			errorCh <- iodine.New(err, nil)
			continue
		}
		tgtClients[i] = tgtClient
	}
	var wg sync.WaitGroup

	multiTgtWriter := io.MultiWriter(tgtWriters...)
	go io.CopyN(multiTgtWriter, reader, length)

	for i := range tgtClients {
		wg.Add(1)
		go func(targetClient client.Client, reader io.Reader, errorCh chan error) {
			defer wg.Done()
			err := targetClient.PutObject(length, reader)
			if err != nil {
				errorCh <- iodine.New(err, map[string]string{"failedURL": targetClient.URL().String()})
			}
		}(tgtClients[i], tgtReaders[i], errorCh)
	}
	wg.Wait()
	close(errorCh)
	return errorCh
}

// getNewClient gives a new client interface
func getNewClient(urlStr string, auth *hostConfig) (clnt client.Client, err error) {
	url, err := client.Parse(urlStr)
	if err != nil {
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
		s3Config.Debug = globalDebugFlag
		return s3.New(s3Config)
	case client.Filesystem:
		return fs.New(urlStr)
	}
	return nil, iodine.New(errInvalidURL{url: urlStr}, nil)
}

// url2Stat - Returns client, config and its stat Content from the URL
func url2Stat(urlStr string) (client client.Client, content *client.Content, err error) {
	config, err := getHostConfig(urlStr)
	if err != nil {
		return nil, nil, iodine.New(err, map[string]string{"URL": urlStr})
	}

	client, err = getNewClient(urlStr, config)
	if err != nil {
		return nil, nil, iodine.New(err, map[string]string{"URL": urlStr})
	}

	content, err = client.Stat()
	if err != nil {
		return nil, nil, iodine.New(err, map[string]string{"URL": urlStr})
	}

	return client, content, nil
}

func url2Client(url string) (client.Client, error) {
	// Empty source arg?
	urlParse, err := client.Parse(url)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	if urlParse.Path == "" {
		return nil, iodine.New(errors.New("invalid path"), nil)
	}

	urlonfig, err := getHostConfig(url)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	client, err := getNewClient(url, urlonfig)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return client, nil
}

// source2Client returns client and hostconfig objects from the source URL.
func source2Client(sourceURL string) (client.Client, error) {
	sourceClient, err := url2Client(sourceURL)
	if err != nil {
		return nil, iodine.New(errInvalidSource{URL: sourceURL}, nil)
	}
	return sourceClient, nil
}

// target2Client returns client and hostconfig objects from the target URL.
func target2Client(targetURL string) (client.Client, error) {
	targetClient, err := url2Client(targetURL)
	if err != nil {
		return nil, iodine.New(errInvalidTarget{URL: targetURL}, nil)
	}
	return targetClient, nil
}
