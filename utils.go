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
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/minio/minio-xl/pkg/probe"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// newRandomID generates a random id of regular lower case and uppercase english characters.
func newRandomID(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	sid := make([]rune, n)
	for i := range sid {
		sid[i] = letters[rand.Intn(len(letters))]
	}
	return string(sid)
}

// isBucketVirtualStyle is host virtual bucket style?.
func isBucketVirtualStyle(host string) bool {
	s3Virtual, _ := filepath.Match("*.s3*.amazonaws.com", host)
	googleVirtual, _ := filepath.Match("*.storage.googleapis.com", host)
	return s3Virtual || googleVirtual
}

// Workaround for docker images with fully static binary and 32bit linux operating systems.
// For static binaries NSS library will not be a part of the static binary hence user.Current() fails.
// For 32bit linux CGO is not enabled so it will not provide linux specific codebase.
// This code is taken from the minio project:
// (https://github.com/minio/minio/blob/8046b24b975a26d659bda12797a30631bb427e8f/utils.go#L60-L85)
func userCurrent() (*user.User, *probe.Error) {
	if os.Getenv("DOCKERIMAGE") == "1" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, probe.NewError(err)
		}
		return &user.User{Uid: "0", Gid: "0", Username: "root", Name: "root", HomeDir: wd}, nil
	}
	if runtime.GOARCH == "386" && runtime.GOOS == "linux" {
		return &user.User{
			Uid:      strconv.Itoa(os.Getuid()),
			Gid:      strconv.Itoa(os.Getgid()),
			Username: os.Getenv("USER"),
			Name:     os.Getenv("USER"),
			HomeDir:  os.Getenv("HOME"),
		}, nil
	}
	user, err := user.Current()
	if err != nil {
		return nil, probe.NewError(err)
	}
	return user, nil
}
