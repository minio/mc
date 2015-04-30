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
	"runtime"
	"time"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	mcUpdateURL = "http://dl.minio.io:9000/updates/2015/Apr" + "/mc" + "." + runtime.GOOS + "." + runtime.GOARCH
)

func doUpdateCheck(config *hostConfig) (string, error) {
	clnt, err := getNewClient(mcUpdateURL, config, globalDebugFlag)
	if err != nil {
		return "Unable to create client: " + mcUpdateURL, iodine.New(err, map[string]string{"failedURL": mcUpdateURL})
	}
	latest, err := clnt.Stat()
	if err != nil {
		return "Unable to create client: " + mcUpdateURL, iodine.New(err, map[string]string{"failedURL": mcUpdateURL})
	}
	current, _ := time.Parse(time.RFC3339Nano, BuildDate)
	if current.IsZero() {
		return "BuildDate is zero, must be a wrong build exiting..", iodine.New(err, map[string]string{"failedURL": mcUpdateURL})
	}
	if latest.Time.After(current) {
		printUpdateNotify("new", "old")
	}
	return "", nil

}

// runUpdateCmd -
func runUpdateCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	hostConfig, err := getHostConfig(mcUpdateURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read configuration for host [%s]. Reason: [%s].\n", mcUpdateURL, iodine.ToError(err))
	}
	switch ctx.Args().First() {
	case "check":
		msg, err := doUpdateCheck(hostConfig)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf(msg)
		}
	}
}
