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
	"net/url"
	"path"
	"strings"

	"github.com/codegangsta/cli"
)

func parseOptions(c *cli.Context) (fsoptions *fsOptions, err error) {
	fsoptions = new(fsOptions)
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
			fsoptions.bucket = uri.Host
			fsoptions.key = strings.TrimPrefix(uri.Path, "/")
		} else {
			return nil, errInvalidScheme
		}
	case 2:
		if strings.HasPrefix(c.Args().Get(0), "s3://") {
			uri, err := url.Parse(c.Args().Get(0))
			if err != nil {
				return nil, err
			}
			fsoptions.bucket = uri.Host
			if uri.Path == "" {
				return nil, errFskey
			}
			fsoptions.key = strings.TrimPrefix(uri.Path, "/")
			if c.Args().Get(1) == "." {
				fsoptions.body = path.Base(fsoptions.key)
			} else {
				fsoptions.body = c.Args().Get(1)
			}
			fsoptions.isget = true
			fsoptions.isput = false
		} else if strings.HasPrefix(c.Args().Get(1), "s3://") {
			uri, err := url.Parse(c.Args().Get(1))
			if err != nil {
				return nil, err
			}
			fsoptions.bucket = uri.Host
			if uri.Path == "" {
				fsoptions.key = c.Args().Get(0)
			} else {
				fsoptions.key = strings.TrimPrefix(uri.Path, "/")
			}
			fsoptions.body = c.Args().Get(0)
			fsoptions.isget = false
			fsoptions.isput = true
		}
	default:
		return nil, errInvalidScheme
	}
	return
}
