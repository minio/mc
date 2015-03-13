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
	"bytes"
	"os"
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

// getBashCompletion -
func getBashCompletion() {
	var b bytes.Buffer
	b.WriteString(mcBashCompletion)
	f := getMcBashCompletionFilename()
	fl, err := os.OpenFile(f, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer fl.Close()
	_, err = fl.Write(b.Bytes())
	if err != nil {
		fatal(err.Error())
	}
	msg := "\nConfiguration written to " + f
	msg = msg + "\n\n$ source ${HOME}/.minio/mc/mc.bash_completion\n"
	msg = msg + "$ echo 'source ${HOME}/.minio/mc/mc.bash_completion' >> ${HOME}/.bashrc\n"
	info(msg)
}

// NewClient - get new client
func getNewClient(c *cli.Context) (*s3.Client, error) {
	var client *s3.Client

	config, err := getMcConfig()
	if err != nil {
		return nil, err
	}

	switch c.GlobalBool("debug") {
	case true:
		trace := s3Trace{BodyTraceFlag: false, RequestTransportFlag: true, Writer: nil}
		traceTransport := s3.GetNewTraceTransport(trace, http.DefaultTransport)
		client = s3.GetNewClient(&config.S3.Auth, traceTransport)
	case false:
		client = s3.GetNewClient(&config.S3.Auth, http.DefaultTransport)
	}

	return client, nil
}

// Parse global options
func parseGlobalOptions(c *cli.Context) {
	switch true {
	case c.Bool("get-bash-completion") == true:
		getBashCompletion()
	}
}

// Parse subcommand options
func parseOptions(c *cli.Context) (cmdoptions *cmdOptions, err error) {
	cmdoptions = new(cmdOptions)
	switch len(c.Args()) {
	case 1:
		if strings.HasPrefix(c.Args().Get(0), "s3://") {
			uri, err := url.Parse(c.Args().Get(0))
			if err != nil {
				return nil, err
			}
			if uri.Scheme != "s3" {
				return nil, errInvalidScheme
			}
			cmdoptions.bucket = uri.Host
			cmdoptions.key = strings.TrimPrefix(uri.Path, "/")
		} else {
			return nil, errInvalidScheme
		}
	case 2:
		if strings.HasPrefix(c.Args().Get(0), "s3://") {
			uri, err := url.Parse(c.Args().Get(0))
			if err != nil {
				return nil, err
			}
			cmdoptions.bucket = uri.Host
			if uri.Path == "" {
				return nil, errFskey
			}
			cmdoptions.key = strings.TrimPrefix(uri.Path, "/")
			if c.Args().Get(1) == "." {
				cmdoptions.body = path.Base(cmdoptions.key)
			} else {
				cmdoptions.body = c.Args().Get(1)
			}
			cmdoptions.isget = true
			cmdoptions.isput = false
		} else if strings.HasPrefix(c.Args().Get(1), "s3://") {
			uri, err := url.Parse(c.Args().Get(1))
			if err != nil {
				return nil, err
			}
			cmdoptions.bucket = uri.Host
			if uri.Path == "" {
				cmdoptions.key = c.Args().Get(0)
			} else {
				cmdoptions.key = strings.TrimPrefix(uri.Path, "/")
			}
			cmdoptions.body = c.Args().Get(0)
			cmdoptions.isget = false
			cmdoptions.isput = true
		}
	default:
		return nil, errInvalidScheme
	}
	return
}

func getMcBashCompletionFilename() string {
	return path.Join(getMcConfigDir(), "mc.bash_completion")
}
