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

	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/client/fs"
	"github.com/minio/mc/pkg/client/s3"
)

// Check if the target URL represents folder. It may or may not exist yet.
func isTargetURLDir(targetURL string) bool {
	targetURLParse, err := client.Parse(targetURL)
	if err != nil {
		return false
	}

	_, targetContent, perr := url2Stat(targetURL)
	if perr != nil {
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
func getSource(sourceURL string) (reader io.ReadCloser, length int64, err *probe.Error) {
	sourceClnt, err := source2Client(sourceURL)
	if err != nil {
		return nil, 0, err.Trace()
	}
	return sourceClnt.GetObject(0, 0)
}

// putTarget writes to URL from reader.
func putTarget(targetURL string, length int64, reader io.Reader) *probe.Error {
	targetClnt, err := target2Client(targetURL)
	if err != nil {
		return err.Trace()
	}
	err = targetClnt.PutObject(length, reader)
	if err != nil {
		return err.Trace()
	}
	return nil
}

// putTargets writes to URL from reader.
func putTargets(targetURLs []string, length int64, reader io.Reader) *probe.Error {
	var tgtReaders []io.ReadCloser
	var tgtWriters []io.WriteCloser
	var tgtClients []client.Client

	for _, targetURL := range targetURLs {
		tgtClient, err := target2Client(targetURL)
		if err != nil {
			return err.Trace()
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
	errorCh := make(chan *probe.Error, len(tgtClients))

	func() { // Parallel putObject
		defer close(errorCh) // Each routine gets to return one err status.

		for i := range tgtClients {
			wg.Add(1)
			// make local copy for go routine
			tgtClient := tgtClients[i]
			tgtReader := tgtReaders[i]

			go func(targetClient client.Client, reader io.ReadCloser, errorCh chan<- *probe.Error) {
				defer wg.Done()
				err := targetClient.PutObject(length, reader)
				if err != nil {
					errorCh <- err.Trace()
					reader.Close()
					return
				}
				errorCh <- nil
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
func getNewClient(urlStr string, auth hostConfig) (client.Client, *probe.Error) {
	url, err := client.Parse(urlStr)
	if err != nil {
		return nil, probe.NewError(err)
	}
	switch url.Type {
	case client.Object: // Minio and S3 compatible cloud storage
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
	return nil, probe.NewError(errInitClient{url: urlStr})
}

// url2Stat - Returns client, config and its stat Content from the URL
func url2Stat(urlStr string) (client client.Client, content *client.Content, err *probe.Error) {
	client, err = url2Client(urlStr)
	if err != nil {
		return nil, nil, err.Trace()
	}

	content, err = client.Stat()
	if err != nil {
		return nil, nil, err.Trace()
	}

	return client, content, nil
}

func isValidURL(url string) bool {
	// Empty source arg?
	urlParse, err := client.Parse(url)
	if err != nil {
		return false
	}

	if urlParse.Path == "" {
		return false
	}

	return true
}

func url2Client(url string) (client.Client, *probe.Error) {
	if !isValidURL(url) {
		return nil, probe.NewError(errInvalidURL{URL: url})
	}
	urlconfig, err := getHostConfig(url)
	if err != nil {
		return nil, err.Trace()
	}

	client, err := getNewClient(url, urlconfig)
	if err != nil {
		return nil, err.Trace()
	}

	return client, nil
}

// source2Client returns client and hostconfig objects from the source URL.
func source2Client(sourceURL string) (client.Client, *probe.Error) {
	sourceClient, err := url2Client(sourceURL)
	if err != nil {
		return nil, err.Trace()
	}
	return sourceClient, nil
}

// target2Client returns client and hostconfig objects from the target URL.
func target2Client(targetURL string) (client.Client, *probe.Error) {
	targetClient, err := url2Client(targetURL)
	if err != nil {
		return nil, err.Trace()
	}
	return targetClient, nil
}
