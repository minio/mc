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
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// hashBinary computes MD5SUM of a binary file on disk
func hashBinary(progName string) (string, error) {
	path, err := exec.LookPath(progName)
	if err != nil {
		return "", err
	}

	m := md5.New()

	file, err := os.Open(path) // For read access.
	if err != nil {
		return "", err
	}

	io.Copy(m, file)
	return fmt.Sprintf("%x", m.Sum(nil)), nil
}

// mustHashBinarySelf masks any error returned by hashBinary
func mustHashBinarySelf() string {
	hash, _ := hashBinary(os.Args[0])
	return hash
}
