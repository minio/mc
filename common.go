/*
 * Mini Object Storage, (C) 2014,2015 Minio, Inc.
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
	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
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
func getNewClient(c *cli.Context) (client *s3.Client, err error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, err
	}

	switch c.GlobalBool("debug") {
	case true:
		trace := s3.Trace{
			BodyTraceFlag:        false,
			RequestTransportFlag: true,
			Writer:               nil,
		}
		traceTransport := s3.GetNewTraceTransport(trace, http.DefaultTransport)
		client = s3.GetNewClient(&config.S3.Auth, traceTransport)
	case false:
		client = s3.GetNewClient(&config.S3.Auth, http.DefaultTransport)
	}

	return client, nil
}

// Parse subcommand options
func parseArgs(c *cli.Context) (args *cmdArgs, err error) {
	args = new(cmdArgs)
	args.quiet = c.GlobalBool("quiet")

	switch len(c.Args()) {
	case 1: // only one URL
		urlString, err := aliasExpand(c.Args().Get(0))
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(urlString, "http") {
			urlParsed, err := url.Parse(urlString)
			if err != nil {
				return nil, err
			}
			args.source.scheme = urlParsed.Scheme
			if urlParsed.Scheme != "" {
				if urlParsed.Host == "" {
					return nil, errHostname
				}
			}
			args.source.host = urlParsed.Host
			urlSplits := strings.Split(urlParsed.Path, "/")
			if len(urlSplits) > 1 {
				args.source.bucket = urlSplits[1]
				args.source.key = path.Join(urlSplits[2:]...)
			}
		} else {
			return nil, errInvalidScheme
		}
	case 2: // one URL and one path||URL
		switch true {
		case c.Args().Get(0) != "":
			urlString, err := aliasExpand(c.Args().Get(0))
			if err != nil {
				return nil, err
			}
			urlParsed, err := url.Parse(urlString)
			if err != nil {
				return nil, err
			}
			switch true {
			case urlParsed.Scheme == "http" || urlParsed.Scheme == "https":
				if urlParsed.Host == "" {
					if urlParsed.Path == "" {
						return nil, errInvalidScheme
					}
					return nil, errInvalidScheme
				}
				args.source.scheme = urlParsed.Scheme
				args.source.host = urlParsed.Host
				urlSplits := strings.Split(urlParsed.Path, "/")
				if len(urlSplits) > 1 {
					args.source.bucket = urlSplits[1]
					args.source.key = path.Join(urlSplits[2:]...)
				}
			case urlParsed.Scheme == "":
				if urlParsed.Host != "" {
					return nil, errInvalidScheme
				}
				if urlParsed.Path != c.Args().Get(0) {
					return nil, errInvalidScheme
				}
				if urlParsed.Path == "." {
					return nil, errFskey
				}
				args.source.key = strings.TrimPrefix(urlParsed.Path, "/")
			case urlParsed.Scheme != "http" && urlParsed.Scheme != "https":
				return nil, errInvalidScheme
			}
			fallthrough
		case c.Args().Get(1) != "":
			urlString, err := aliasExpand(c.Args().Get(1))
			if err != nil {
				return nil, err
			}

			urlParsed, err := url.Parse(urlString)
			if err != nil {
				return nil, err
			}

			switch true {
			case urlParsed.Scheme == "http" || urlParsed.Scheme == "https":
				if urlParsed.Host == "" {
					if urlParsed.Path == "" {
						return nil, errInvalidScheme
					}
					return nil, errInvalidScheme
				}
				args.destination.host = urlParsed.Host
				args.destination.scheme = urlParsed.Scheme
				urlSplits := strings.Split(urlParsed.Path, "/")
				if len(urlSplits) > 1 {
					args.destination.bucket = urlSplits[1]
					args.destination.key = path.Join(urlSplits[2:]...)
				}
			case urlParsed.Scheme == "":
				if urlParsed.Host != "" {
					return nil, errInvalidScheme
				}
				if urlParsed.Path == "." {
					args.destination.key = args.source.key
				} else {
					args.destination.key = strings.TrimPrefix(urlParsed.Path, "/")
				}
				args.destination.bucket = urlParsed.Host
			case urlParsed.Scheme != "http" && urlParsed.Scheme != "https":
				return nil, errInvalidScheme
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
