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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

/// LS - related internal functions

// iso8601 date
const (
	printDate = "2006-01-02 15:04:05 MST"
)

// printItem prints item meta-data
func printItem(date time.Time, v int64, name string, fileType os.FileMode) {
	fmt.Printf(console.Time("[%s] ", date.Local().Format(printDate)))
	fmt.Printf(console.Size("%6s ", humanize.IBytes(uint64(v))))

	// just making it explicit
	if fileType.IsDir() {
		// if one finds a prior suffix no need to append a new one
		if strings.HasSuffix(name, "/") {
			fmt.Println(console.Dir("%s", name))
		} else {
			fmt.Println(console.Dir("%s/", name))
		}
	}
	if fileType.IsRegular() {
		fmt.Println(console.File("%s", name))
	}
}

// doList - list all entities inside a directory
func doList(clnt client.Client, targetURL string) error {
	var err error
	for itemCh := range clnt.List() {
		if itemCh.Err != nil {
			err = itemCh.Err
			break
		}
		printItem(itemCh.Item.Time, itemCh.Item.Size, itemCh.Item.Name, itemCh.Item.FileType)
	}
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	return nil
}

// doListRecursive - list all entities inside directories and sub-directories recursively
func doListRecursive(clnt client.Client, targetURL string) error {
	var err error
	for itemCh := range clnt.ListRecursive() {
		if itemCh.Err != nil {
			err = itemCh.Err
			break
		}
		printItem(itemCh.Item.Time, itemCh.Item.Size, itemCh.Item.Name, itemCh.Item.FileType)
	}
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	return nil
}
