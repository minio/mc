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
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/client/fs"
	"github.com/minio/mc/pkg/client/s3"
	"github.com/minio/minio/pkg/iodine"
)

// Check if the target URL represents folder. It may or may not exist yet.
func isTargetURLDir(targetURL string) bool {
	targetURLParse, err := client.Parse(targetURL)
	if err != nil {
		return false
	}
	_, targetContent, err := url2Stat(targetURL)
	if err != nil {
		if targetURLParse.Path == string(targetURLParse.Separator) && targetURLParse.Scheme != "" {
			return false
		}
		if strings.HasSuffix(targetURLParse.Path, string(targetURLParse.Separator)) {
			return true
		}
		return false
	}
	if !targetContent.Type.IsDir() { // Target is a dir.
		return false
	}
	return true
}

// getSource gets a reader from URL<
func getSource(sourceURL string) (reader io.ReadCloser, length int64, err error) {
	sourceClnt, err := source2Client(sourceURL)
	if err != nil {
		return nil, 0, NewIodine(iodine.New(err, map[string]string{"failedURL": sourceURL}))
	}
	return sourceClnt.GetObject(0, 0)
}

// putTarget writes to URL from reader.
func putTarget(targetURL string, length int64, reader io.Reader) error {
	targetClnt, err := target2Client(targetURL)
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}
	err = targetClnt.PutObject(length, reader)
	if err != nil {
		return NewIodine(iodine.New(err, map[string]string{"failedURL": targetURL}))
	}
	return nil
}

// putTargets writes to URL from reader.
func putTargets(targetURLs []string, length int64, reader io.Reader) error {
	var tgtReaders []io.ReadCloser
	var tgtWriters []io.WriteCloser
	var tgtClients []client.Client

	for _, targetURL := range targetURLs {
		tgtClient, err := target2Client(targetURL)
		if err != nil {
			return iodine.New(err, nil)
		}
		tgtClients = append(tgtClients, tgtClient)
		tgtReader, tgtWriter := io.Pipe()
		tgtReaders = append(tgtReaders, tgtReader)
		tgtWriters = append(tgtWriters, tgtWriter)
	}

	go func() {
		var writers []io.Writer
		for _, tgtWriter := range tgtWriters {
			defer tgtWriter.Close()
			writers = append(writers, io.Writer(tgtWriter))
		}
		multiTgtWriter := io.MultiWriter(writers...)
		io.CopyN(multiTgtWriter, reader, length)
	}()

	var wg sync.WaitGroup
	errorCh := make(chan error, len(tgtClients))

	func() { // Parallel putObject
		defer close(errorCh) // Each routine gets to return one err status.

		for i := range tgtClients {
			wg.Add(1)
			// make local copy for go routine
			tgtClient := tgtClients[i]
			tgtReader := tgtReaders[i]

			go func(targetClient client.Client, reader io.ReadCloser, errorCh chan<- error) {
				defer wg.Done()
				err := targetClient.PutObject(length, reader)
				if err != nil {
					errorCh <- iodine.New(err, map[string]string{"failedURL": targetClient.URL().String()})
					reader.Close()
					return
				}
				errorCh <- err // return nil error = success.
			}(tgtClient, tgtReader, errorCh)
		}
		wg.Wait()
	}()

	for err := range errorCh {
		if err != nil { // Return on first error encounter.
			return err
		}
	}
	return nil // success.
}

// getNewClient gives a new client interface
func getNewClient(urlStr string, auth *hostConfig) (clnt client.Client, err error) {
	url, err := client.Parse(urlStr)
	if err != nil {
		return nil, NewIodine(iodine.New(errInvalidURL{URL: urlStr}, map[string]string{"URL": urlStr}))
	}
	switch url.Type {
	case client.Object: // Minio and S3 compatible cloud storage
		if auth == nil {
			return nil, NewIodine(iodine.New(errInvalidArgument{}, nil))
		}
		s3Config := new(s3.Config)
		s3Config.AccessKeyID = func() string {
			if auth.AccessKeyID == globalAccessKeyID {
				return ""
			}
			return auth.AccessKeyID
		}()
		s3Config.SecretAccessKey = func() string {
			if auth.SecretAccessKey == globalSecretAccessKey {
				return ""
			}
			return auth.SecretAccessKey
		}()
		s3Config.AppName = "Minio"
		s3Config.AppVersion = getVersion()
		s3Config.AppComments = []string{os.Args[0], runtime.GOOS, runtime.GOARCH}
		s3Config.HostURL = urlStr
		s3Config.Debug = globalDebugFlag
		return s3.New(s3Config)
	case client.Filesystem:
		return fs.New(urlStr)
	}
	return nil, NewIodine(iodine.New(errInvalidURL{URL: urlStr}, nil))
}

// url2Stat - Returns client, config and its stat Content from the URL
func url2Stat(urlStr string) (client client.Client, content *client.Content, err error) {
	client, err = url2Client(urlStr)
	if err != nil {
		return nil, nil, NewIodine(iodine.New(err, map[string]string{"URL": urlStr}))
	}

	content, err = client.Stat()
	if err != nil {
		return nil, nil, NewIodine(iodine.New(err, map[string]string{"URL": urlStr}))
	}

	return client, content, nil
}

func url2Client(url string) (client.Client, error) {
	// Empty source arg?
	urlParse, err := client.Parse(url)
	if err != nil {
		return nil, NewIodine(iodine.New(err, map[string]string{"URL": url}))
	}

	if urlParse.Path == "" {
		return nil, NewIodine(iodine.New(errInvalidURL{URL: url}, map[string]string{"URL": url}))
	}

	urlconfig, err := getHostConfig(url)
	if err != nil {
		return nil, NewIodine(iodine.New(err, map[string]string{"URL": url}))
	}

	client, err := getNewClient(url, urlconfig)
	if err != nil {
		return nil, NewIodine(iodine.New(err, map[string]string{"URL": url}))
	}

	return client, nil
}

// source2Client returns client and hostconfig objects from the source URL.
func source2Client(sourceURL string) (client.Client, error) {
	sourceClient, err := url2Client(sourceURL)
	if err != nil {
		return nil, NewIodine(iodine.New(errInvalidSource{URL: sourceURL}, map[string]string{"URL": sourceURL}))
	}
	return sourceClient, nil
}

// target2Client returns client and hostconfig objects from the target URL.
func target2Client(targetURL string) (client.Client, error) {
	targetClient, err := url2Client(targetURL)
	if err != nil {
		return nil, NewIodine(iodine.New(errInvalidTarget{URL: targetURL}, map[string]string{"URL": targetURL}))
	}
	return targetClient, nil
}
