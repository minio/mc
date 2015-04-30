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
	"errors"
	"fmt"
	"runtime"
	"time"

	"encoding/json"
	"net/http"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	mcUpdateURL = "http://dl.minio.io:9000/binaries/"
)

type updateResults struct {
	Version     uint // this is config version
	LatestBuild string
	Signature   string
}

// getRequest -
func getRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		msg := fmt.Sprintf("s3 client; invalid URL: %v", err)
		return nil, errors.New(msg)
	}
	req.Header.Set("User-Agent", mcUserAgent)
	return req, nil
}

// runUpdateCmd -
func runUpdateCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	mcUpdateBinaryURL := mcUpdateURL + runtime.GOOS + "/mc"
	switch ctx.Args().First() {
	case "check":
		req, err := getRequest(mcUpdateBinaryURL)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("Unable to update:", mcUpdateBinaryURL)
		}
		res, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("Unable to retrieve:", mcUpdateBinaryURL)
		}
		if res.StatusCode != http.StatusOK {
			msg := fmt.Sprint("Received invalid HTTP status: ", res.StatusCode)
			log.Debug.Println(iodine.New(errors.New(msg), nil))
			console.Fatalln(msg)
		}
		results := updateResults{}
		err = json.NewDecoder(res.Body).Decode(&results)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("Unable to parse JSON:", mcUpdateURL)
		}
		latest, err := time.Parse(time.RFC3339Nano, results.LatestBuild)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("Unable to parse update time:", results.LatestBuild)
		}
		current, _ := time.Parse(time.RFC3339Nano, BuildDate)
		if current.IsZero() {
			console.Fatalln("BuildDate is zero, must be a wrong build exiting..")
		}
		if latest.After(current) {
			printUpdateNotify("new", "old")
		}
	case "install":
		console.Fatalln("Functionality not implemented yet")
	}
}
