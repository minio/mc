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
	"encoding/json"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Share documents via URL.
var shareList = cli.Command{
	Name:   "list",
	Usage:  "List the shared URLs",
	Action: mainShareList,
	CustomHelpTemplate: `NAME:
   mc share {{.Name}} - {{.Usage}}

USAGE:
   mc share {{.Name}}

EXAMPLES:
   $ mc share {{.Name}}

`,
}

func mainShareList(ctx *cli.Context) {
	shareDataDirSetup()
	setSharePalette(ctx.GlobalString("colors"))
	err := doShareList()
	fatalIf(err.Trace(), "Unable to list shared URLs.")
}

// doShareList list shared url's
func doShareList() *probe.Error {
	sURLs, err := loadSharedURLsV3()
	if err != nil {
		return err.Trace()
	}
	saveList := newSharedURLsV3()
	for _, data := range sURLs.URLs {
		if time.Since(data.Date) > data.Message.Expiry {
			continue
		}
		saveList.URLs = append(saveList.URLs, data)
		expiry := data.Message.Expiry - time.Since(data.Date)
		if !globalJSONFlag {
			var kind string
			if len(data.Message.DownloadURL) > 0 {
				kind = "Download"
			} else {
				kind = "Upload"
			}
			msg := console.Colorize("Share", "Name: ")
			msg += console.Colorize("URL", data.Message.Key+" ("+kind+")\n")
			msg += console.Colorize("Share", "Expiry: ")
			msg += console.Colorize("Expires", timeDurationToHumanizedTime(expiry))
			msg += "\n"
			console.Println(msg)
			continue
		}
		var shareMessageBytes []byte
		var err error
		s := data.Message
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
			shareMessageBytes, err = json.Marshal(struct {
				Expiry     humanizedTime     `json:"expiry"`
				UploadInfo map[string]string `json:"uploadInfo"`
				Key        string            `json:"keyName"`
			}{
				Expiry:     timeDurationToHumanizedTime(s.Expiry),
				UploadInfo: s.UploadInfo,
				Key:        s.Key,
			})
		}
		if err != nil {
			return probe.NewError(err)
		}
		console.Println(string(shareMessageBytes))
	}
	if err := saveSharedURLsV3(saveList); err != nil {
		return err.Trace()
	}
	return nil
}
