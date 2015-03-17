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
	config := getMcConfig()

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
		URL, err := aliasExpand(c.Args().Get(0))
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(URL, "http") || strings.HasPrefix(URL, "https") {
			url, err := url.Parse(URL)
			if err != nil {
				return nil, err
			}
			if !strings.HasPrefix(url.Scheme, "http") && !strings.HasPrefix(url.Scheme, "https") {
				return nil, errInvalidScheme
			}
			args.source.scheme = url.Scheme
			if url.Scheme != "" {
				if url.Host == "" {
					return nil, errHostname
				}
			}
			args.source.host = url.Host
			URLSplits := strings.Split(url.Path, "/")
			if len(URLSplits) > 1 {
				args.source.bucket = URLSplits[1]
				args.source.key = path.Join(URLSplits[2:]...)
			}
		} else {
			return nil, errInvalidScheme
		}
	case 2: // one URL and one path||URL
		switch true {
		case c.Args().Get(0) != "":
			url, err := url.Parse(c.Args().Get(0))
			if err != nil {
				return nil, err
			}
			switch true {
			case url.Scheme == "http" || url.Scheme == "https":
				if url.Host == "" {
					if url.Path == "" {
						return nil, errInvalidScheme
					}
					return nil, errInvalidScheme
				}
				args.source.scheme = url.Scheme
				args.source.host = url.Host
				URLSplits := strings.Split(url.Path, "/")
				if len(URLSplits) > 1 {
					args.source.bucket = URLSplits[1]
					args.source.key = path.Join(URLSplits[2:]...)
				}
			case url.Scheme == "":
				if url.Host != "" {
					return nil, errInvalidScheme
				}
				if url.Path != c.Args().Get(0) {
					return nil, errInvalidScheme
				}
				if url.Path == "." {
					return nil, errFskey
				}
				args.source.key = strings.TrimPrefix(url.Path, "/")
			case url.Scheme != "http" && url.Scheme != "https":
				return nil, errInvalidScheme
			}
			fallthrough
		case c.Args().Get(1) != "":
			url, err := url.Parse(c.Args().Get(1))
			if err != nil {
				return nil, err
			}
			switch true {
			case url.Scheme == "http" || url.Scheme == "https":
				if url.Host == "" {
					if url.Path == "" {
						return nil, errInvalidScheme
					}
					return nil, errInvalidScheme
				}
				args.destination.host = url.Host
				args.destination.scheme = url.Scheme
				URLSplits := strings.Split(url.Path, "/")
				if len(URLSplits) > 1 {
					args.destination.bucket = URLSplits[1]
					args.destination.key = path.Join(URLSplits[2:]...)
				}
			case url.Scheme == "":
				if url.Host != "" {
					return nil, errInvalidScheme
				}
				if url.Path == "." {
					args.destination.key = args.source.key
				} else {
					args.destination.key = strings.TrimPrefix(url.Path, "/")
				}
				args.destination.bucket = url.Host
			case url.Scheme != "http" && url.Scheme != "https":
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
