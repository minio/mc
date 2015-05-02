/*
 * Minio Client, (C) 2015 Minio, Inc.
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
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

func runCatCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cat", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	config, err := getMcConfig()
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("loading config file failed with following reason: [%s]\n", iodine.ToError(err))
	}

	// Convert arguments to URLs: expand alias, fix format...
	urls, err := getExpandedURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("reading URL [%s] failed with following reason: [%s]\n", e.url, e)
		default:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("reading URLs failed with following reason: [%s]\n", e)
		}
	}

	sourceURLs := urls
	sourceURLConfigMap, err := getHostConfigs(sourceURLs)
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("reading host config for URL [%s] failed with following reason: [%s]\n", sourceURLs, iodine.ToError(err))
	}
	humanReadable, err := doCatCmd(sourceURLConfigMap, globalDebugFlag)
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalln(humanReadable)
	}
}

func doCatCmd(sourceURLConfigMap map[string]*hostConfig, debug bool) (string, error) {
	for url, config := range sourceURLConfigMap {
		sourceClnt, err := getNewClient(url, config, debug)
		if err != nil {
			return "Unable to create client: " + url, iodine.New(err, nil)
		}
		reader, size, sourceMd5, err := sourceClnt.Get()
		if err != nil {
			return "Unable to retrieve file: " + url, iodine.New(err, nil)
		}
		defer reader.Close()
		hasher := md5.New()
		mw := io.MultiWriter(os.Stdout, hasher)
		_, err = io.CopyN(mw, reader, size)
		if err != nil {
			switch e := iodine.ToError(err).(type) {
			case *os.PathError:
				return "Reading data to stdout failed, you system be having problems.. please report this error", iodine.New(e, nil)
			default:
				return "Reading data from source failed: " + url, iodine.New(errors.New("Copy data from source failed"), nil)
			}
		}
		actualMd5 := hex.EncodeToString(hasher.Sum(nil))
		if sourceMd5 != actualMd5 {
			return "Md5sum mismatch, must be error in transmit what you are looking at might be corrupted", iodine.New(errors.New("corrupted data"), nil)
		}
	}
	return "", nil
}
