/*
 * Modern Copy, (C) 2015 Minio, Inc.
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
	"errors"
	"fmt"
	"io"
	"time"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	mcUpdateURL = "http://minio.io/updates/mc.json"
)

type updateResults struct {
	Version     uint // this is config version
	LatestBuild string
	Signature   string
}

func getReq(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		msg := fmt.Sprintf("s3 client; invalid URL: %v", err)
		return nil, errors.New(msg)
	}
	req.Header.Set("User-Agent", "Minio Client")
	return req, nil
}

func doUpdateCmd(ctx *cli.Context) {
	req, err := getReq(mcUpdateURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(err)
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(err)
	}
	if res.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("received invalid http status: %v", res.StatusCode)
		log.Debug.Println(iodine.New(errors.New(msg), nil))
		console.Fatalln(msg)
	}
	ures := updateResults{}
	err = json.NewDecoder(io.TeeReader(res.Body, ioutil.Discard)).Decode(&ures)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(err)
	}
	latest, err := time.Parse(time.RFC3339Nano, ures.LatestBuild)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(err)
	}
	if latest.After(ctx.App.Compiled) {
		// FIXME : find some proper versioning scheme here
		printUpdateNotify("new", "old")
	}
}
