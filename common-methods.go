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
	"github.com/minio/minio-xl/pkg/probe"
)

// Check if the target URL represents folder. It may or may not exist yet.
func isTargetURLDir(targetURL string) bool {
	targetURLParse := client.NewURL(targetURL)
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

// getSource gets a reader from URL
func getSource(sourceURL string) (reader io.ReadCloser, length int64, err *probe.Error) {
	sourceClnt, err := url2Client(sourceURL)
	if err != nil {
		return nil, 0, err.Trace()
	}
	return sourceClnt.Get(0, 0)
}

// putTarget writes to URL from reader. If length=0, read until EOF.
func putTarget(targetURL string, length int64, reader io.Reader) *probe.Error {
	targetClnt, err := url2Client(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	err = targetClnt.Put(length, reader)
	if err != nil {
		return err.Trace(targetURL)
	}
	return nil
}

// putTargets writes to URL from reader. If length=0, read until EOF.
func putTargets(targetURLs []string, length int64, reader io.Reader) *probe.Error {
	var tgtReaders []*io.PipeReader
	var tgtWriters []*io.PipeWriter
	var tgtClients []client.Client
	errCh := make(chan *probe.Error)
	defer close(errCh)

	for _, targetURL := range targetURLs {
		tgtClient, err := url2Client(targetURL)
		if err != nil {
			return err.Trace(targetURL)
		}
		tgtClients = append(tgtClients, tgtClient)
		tgtReader, tgtWriter := io.Pipe()
		tgtReaders = append(tgtReaders, tgtReader)
		tgtWriters = append(tgtWriters, tgtWriter)
	}

	go func() {
		var writers []io.Writer
		for _, tgtWriter := range tgtWriters {
			writers = append(writers, io.Writer(tgtWriter))
		}

		multiTgtWriter := io.MultiWriter(writers...)
		var e error
		switch length {
		case 0:
			_, e = io.Copy(multiTgtWriter, reader)
		default:
			_, e = io.CopyN(multiTgtWriter, reader, length)
		}
		for _, tgtWriter := range tgtWriters {
			if e != nil {
				tgtWriter.CloseWithError(e)
			}
			tgtWriter.Close()
		}
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
				defer reader.Close()
				err := targetClient.Put(length, reader)
				if err != nil {
					errorCh <- err.Trace()
					return
				}
			}(tgtClient, tgtReader, errorCh)
		}
		wg.Wait()
	}()

	// Return on first error encounter.
	err := <-errorCh
	if err != nil {
		return err.Trace()
	}

	return nil // success.
}

// getNewClient gives a new client interface
func getNewClient(urlStr string, auth hostConfig) (client.Client, *probe.Error) {
	url := client.NewURL(urlStr)
	switch url.Type {
	case client.Object: // Minio and S3 compatible cloud storage
		s3Config := new(client.Config)
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
		s3Config.Signature = auth.API
		s3Config.AppName = "Minio"
		s3Config.AppVersion = mcVersion
		s3Config.AppComments = []string{os.Args[0], runtime.GOOS, runtime.GOARCH}
		s3Config.HostURL = urlStr
		s3Config.Debug = globalDebugFlag

		s3Client, err := s3.New(s3Config)
		if err != nil {
			return nil, err.Trace()
		}
		return s3Client, nil
	case client.Filesystem:
		fsClient, err := fs.New(urlStr)
		if err != nil {
			return nil, err.Trace()
		}
		return fsClient, nil
	}
	return nil, errInitClient(urlStr).Trace()
}

// url2Client - convenience wrapper for getNewClient
func url2Client(url string) (client.Client, *probe.Error) {
	urlconfig, err := getHostConfig(url)
	if err != nil {
		return nil, err.Trace(url)
	}
	client, err := getNewClient(url, urlconfig)
	if err != nil {
		return nil, err.Trace(url)
	}
	return client, nil
}
