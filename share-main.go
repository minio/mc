/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// Share documents via URL.
var shareCmd = cli.Command{
	Name:   "share",
	Usage:  "Download and upload documents.",
	Action: mainShare,
	Subcommands: []cli.Command{
		shareDownload,
		shareUpload,
		shareList,
	},
	CustomHelpTemplate: `NAME:
  {{.Name}} - {{.Usage}}

USAGE:
  {{.Name}} command [arguments...]

COMMANDS:
  {{range .Commands}}{{ .Name }}{{ "\t" }}{{.Usage}}
  {{end}}
`,
}

// structured share command messages version '1'
type shareMessageV1 struct {
	Expiry time.Duration `json:"expiry"`
	URL    string        `json:"url"`
	Key    string        `json:"keyName"`
}

// structured share command messages version '2'
type shareMessageV2 shareMessageV1

// structured share command messages version '3'
type shareMessageV3 struct {
	Expiry      time.Duration     `json:"expiry"`
	DownloadURL string            `json:"downloadUrl,omitempty"`
	UploadInfo  map[string]string `json:"uploadInfo,omitempty"`
	Key         string            `json:"keyName"`
}

// shareMessage this points to latest share command message structure.
type shareMessage shareMessageV3

// String - regular colorized message
func (s shareMessage) String() string {
	if len(s.DownloadURL) > 0 {
		return console.Colorize("Share", fmt.Sprintf("%s", s.DownloadURL))
	}
	var key string
	URL := client.NewURL(s.Key)
	postURL := URL.Scheme + URL.SchemeSeparator + URL.Host + string(URL.Separator) + s.UploadInfo["bucket"] + " "
	curlCommand := "curl " + postURL
	for k, v := range s.UploadInfo {
		if k == "key" {
			key = v
			continue
		}
		curlCommand = curlCommand + fmt.Sprintf("-F %s=%s ", k, v)
	}
	curlCommand = curlCommand + fmt.Sprintf("-F key=%s ", key) + "-F file=@<FILE> "
	emphasize := console.Colorize("File", "<FILE>")
	curlCommand = strings.Replace(curlCommand, "<FILE>", emphasize, -1)
	return console.Colorize("Share", fmt.Sprintf("%s", curlCommand))
}

// JSON json message for share command
func (s shareMessage) JSON() string {
	var shareMessageBytes []byte
	var err error
	if len(s.DownloadURL) > 0 {
		shareMessageBytes, err = json.Marshal(struct {
			Expiry      humanizedTime `json:"expiry"`
			DownloadURL string        `json:"downloadUrl"`
			Key         string        `json:"keyName"`
		}{
			Expiry:      timeDurationToHumanizedTime(s.Expiry),
			DownloadURL: s.DownloadURL,
			Key:         s.Key,
		})
	} else {
		var key string
		URL := client.NewURL(s.Key)
		postURL := URL.Scheme + URL.SchemeSeparator + URL.Host + string(URL.Separator) + s.UploadInfo["bucket"] + " "
		curlCommand := "curl " + postURL
		for k, v := range s.UploadInfo {
			if k == "key" {
				key = v
				continue
			}
			curlCommand = curlCommand + fmt.Sprintf("-F %s=%s ", k, v)
		}
		curlCommand = curlCommand + fmt.Sprintf("-F key=%s ", key) + "-F file=@<FILE> "

		shareMessageBytes, err = json.Marshal(struct {
			Expiry        humanizedTime `json:"expiry"`
			UploadCommand string        `json:"uploadCommand"`
			Key           string        `json:"keyName"`
		}{
			Expiry:        timeDurationToHumanizedTime(s.Expiry),
			UploadCommand: curlCommand,
			Key:           s.Key,
		})
	}

	fatalIf(probe.NewError(err), "Failed to marshal into JSON.")

	// json encoding escapes ampersand into its unicode character which is not usable directly for share
	// and fails with cloud storage. convert them back so that they are usable
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u003c"), []byte("<"), -1)
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u003e"), []byte(">"), -1)
	return string(shareMessageBytes)
}

// mainShare - main handler for mc share command
func mainShare(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowAppHelp(ctx)
	}
}
