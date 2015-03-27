/*
 * Minimalist Object Storage, (C) 2014, 2015 Minio, Inc.
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
	"path"
	"strings"
	"time"

	"net/http"
	"net/url"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/client/s3"
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
		infoCallback(s)
	}
	// Feels like wget
	bar.Format("[=> ]")
	return bar
}

// NewClient - get new client
func getNewClient(debug bool, url string) (cl client.Client, err error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, err
	}

	hostCfg, err := getHostConfig(config.DefaultHost)
	if err != nil {
		return nil, err
	}

	var auth client.Auth
	auth.AccessKeyID = hostCfg.Auth.AccessKeyID
	auth.SecretAccessKey = hostCfg.Auth.SecretAccessKey

	if debug {
		trace := s3.Trace{
			BodyTraceFlag:        false,
			RequestTransportFlag: true,
			Writer:               nil,
		}
		traceTransport := s3.GetNewTraceTransport(trace, http.DefaultTransport)
		cl = s3.GetNewClient(&auth, url, traceTransport)
	} else {
		cl = s3.GetNewClient(&auth, url, http.DefaultTransport)
	}

	return cl, nil
}

func parseDestinationArgs(urlParsed *url.URL, destination, source object) (object, error) {
	switch true {
	case urlParsed.Scheme == "http" || urlParsed.Scheme == "https":
		if urlParsed.Host == "" {
			if urlParsed.Path == "" {
				return object{}, errInvalidScheme
			}
			return object{}, errInvalidScheme
		}
		destination.host = urlParsed.Host
		destination.scheme = urlParsed.Scheme
		destination.url = urlParsed
		urlSplits := strings.Split(urlParsed.Path, "/")
		if len(urlSplits) > 1 {
			destination.bucket = urlSplits[1]
			destination.key = path.Join(urlSplits[2:]...)
		}
	case urlParsed.Scheme == "":
		if urlParsed.Host != "" {
			return object{}, errInvalidScheme
		}
		if urlParsed.Path == "." {
			destination.key = source.key
		} else {
			destination.key = strings.TrimPrefix(urlParsed.Path, "/")
		}
		destination.bucket = urlParsed.Host
	case urlParsed.Scheme != "http" && urlParsed.Scheme != "https":
		return object{}, errInvalidScheme
	}
	return destination, nil
}

func parseSourceArgs(urlParsed *url.URL, firstArg string, source object) (object, error) {
	switch true {
	case urlParsed.Scheme == "http" || urlParsed.Scheme == "https":
		if urlParsed.Host == "" {
			if urlParsed.Path == "" {
				return object{}, errInvalidScheme
			}
			return object{}, errInvalidScheme
		}
		source.scheme = urlParsed.Scheme
		source.host = urlParsed.Host
		source.url = urlParsed
		urlSplits := strings.Split(urlParsed.Path, "/")
		if len(urlSplits) > 1 {
			source.bucket = urlSplits[1]
			source.key = path.Join(urlSplits[2:]...)
		}
	case urlParsed.Scheme == "":
		if urlParsed.Host != "" {
			return object{}, errInvalidScheme
		}
		if urlParsed.Path != firstArg {
			return object{}, errInvalidScheme
		}
		if urlParsed.Path == "." {
			return object{}, errFskey
		}
		source.key = strings.TrimPrefix(urlParsed.Path, "/")
	case urlParsed.Scheme != "http" && urlParsed.Scheme != "https":
		return object{}, errInvalidScheme
	}
	return source, nil
}

func parseSingleArg(urlParsed *url.URL, source object) (object, error) {
	source.scheme = urlParsed.Scheme
	source.url = urlParsed
	if urlParsed.Scheme != "" {
		if urlParsed.Host == "" {
			return object{}, errHostname
		}
	}
	source.host = urlParsed.Host
	urlSplits := strings.Split(urlParsed.Path, "/")
	if len(urlSplits) > 1 {
		source.bucket = urlSplits[1]
		source.key = path.Join(urlSplits[2:]...)
	}
	return source, nil
}

func urlAliasExpander(arg string) (*url.URL, error) {
	urlString, err := aliasExpand(arg)
	if err != nil {
		return nil, err
	}
	urlParsed, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	return urlParsed, nil
}

// Parse subcommand options
func parseArgs(c *cli.Context) (args *cmdArgs, err error) {
	args = new(cmdArgs)
	args.quiet = c.GlobalBool("quiet")
	switch len(c.Args()) {
	case 1: // only one URL
		urlString, err := aliasExpand(c.Args().First())
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(urlString, "http") {
			urlParsed, err := url.Parse(urlString)
			if err != nil {
				return nil, err
			}
			args.source, err = parseSingleArg(urlParsed, args.source)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errInvalidScheme
		}
	case 2: // one URL and one path||URL
		switch true {
		case c.Args().First() != "":
			urlParsed, err := urlAliasExpander(c.Args().First())
			if err != nil {
				return nil, err
			}
			args.source, err = parseSourceArgs(urlParsed, c.Args().First(), args.source)
			if err != nil {
				return nil, err
			}
			fallthrough
		case c.Args().Get(1) != "":
			urlParsed, err := urlAliasExpander(c.Args().Get(1))
			if err != nil {
				return nil, err
			}
			args.destination, err = parseDestinationArgs(urlParsed, args.destination, args.source)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, errInvalidScheme
	}
	return
}

func getMcBashCompletionFilename() string {
	return path.Join(getMcConfigDir(), "mc.bash_completion")
}
