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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Help message.
var shareCmd = cli.Command{
	Name:   "share",
	Usage:  "Share presigned URLs from cloud storage",
	Action: runShareCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Generate presigned url for an object with expiration of 10minutes
      $ mc {{.Name}} https://s3.amazonaws.com/backup/2006-Mar-1/backup.tar.gz expire 10m

   2. Generate presigned url for all objects at given path
      $ mc {{.Name}} https://s3.amazonaws.com/backup... expire 20m

`,
}

// runShareCmd - is a handler for mc share command
func runShareCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "share", 1) // last argument is exit code
	}
	args := ctx.Args()
	config := mustGetMcConfig()
	for _, arg := range args {
		targetURL, err := getExpandedURL(arg, config.Aliases)
		Fatal(err)
		// if recursive strip off the "..."
		newTargetURL := stripRecursiveURL(targetURL)
		Fatal(doShareCmd(newTargetURL, isURLRecursive(targetURL)))
	}
}

// doShareCmd share files from target
func doShareCmd(targetURL string, recursive bool) *probe.Error {
	clnt, err := target2Client(targetURL)
	if err != nil {
		return err.Trace()
	}
	err = doShare(clnt, recursive)
	if err != nil {
		return err.Trace()
	}
	return nil
}

func path2Bucket(u *client.URL) (bucketName string) {
	pathSplits := strings.SplitN(u.Path, "?", 2)
	splits := strings.SplitN(pathSplits[0], string(u.Separator), 3)
	switch len(splits) {
	case 0, 1:
		bucketName = ""
	case 2:
		bucketName = splits[1]
	case 3:
		bucketName = splits[1]
	}
	return bucketName
}

func doShare(clnt client.Client, recursive bool) *probe.Error {
	var err *probe.Error
	for contentCh := range clnt.List(recursive) {
		if contentCh.Err != nil {
			switch contentCh.Err.ToError().(type) {
			// handle this specifically for filesystem
			case client.ISBrokenSymlink:
				Error(contentCh.Err)
				continue
			}
			if os.IsNotExist(contentCh.Err.ToError()) || os.IsPermission(contentCh.Err.ToError()) {
				Error(contentCh.Err)
				continue
			}
			err = contentCh.Err
			break
		}
		if err != nil {
			return err.Trace()
		}
		targetParser := clnt.URL()
		targetParser.Path = path2Bucket(targetParser) + string(targetParser.Separator) + contentCh.Content.Name
		newClnt, err := url2Client(targetParser.String())
		if err != nil {
			return err.Trace()
		}
		// TODO enable expiry
		expire := time.Duration(1000) * time.Second
		presignedURL, err := newClnt.PresignedGetObject(time.Duration(1000)*time.Second, 0, 0)
		if err != nil {
			return err.Trace()
		}
		console.PrintC(fmt.Sprintf("Succesfully generated shared URL with expiry %s, please copy: %s\n", expire, presignedURL))
	}
	return nil
}
