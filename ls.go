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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

/// ls - related internal functions

// iso8601 date
const (
	printDate = "2006-01-02 15:04:05 MST"
)

// printJSON rather than colored output
func printJSON(content *client.Content) {
	type jsonContent struct {
		Filetype string `json:"content-type"`
		Date     string `json:"last-modified"`
		Size     string `json:"size"`
		Name     string `json:"name"`
	}
	contentJSON := new(jsonContent)
	contentJSON.Date = content.Time.Local().Format(printDate)
	contentJSON.Size = humanize.IBytes(uint64(content.Size))
	contentJSON.Filetype = func() string {
		if content.Type.IsDir() {
			return "inode/directory"
		}
		if content.Type.IsRegular() {
			return "application/octet-stream"
		}
		return "application/octet-stream"
	}()
	contentJSON.Name = content.Name
	contentBytes, _ := json.MarshalIndent(contentJSON, "", "\t")
	fmt.Println(string(contentBytes))
}

// printContent prints content meta-data
func printContent(date time.Time, size int64, name string, fileType os.FileMode) {
	fmt.Printf(console.Time("[%s] ", date.Local().Format(printDate)))
	fmt.Printf(console.Size("%6s ", humanize.IBytes(uint64(size))))

	// just making it explicit
	switch {
	case fileType.IsDir() == true:
		// if one finds a prior suffix no need to append a new one
		switch {
		case strings.HasSuffix(name, "/") == true:
			fmt.Println(console.Dir("%s", name))
		default:
			fmt.Println(console.Dir("%s/", name))
		}
	default:
		fmt.Println(console.File("%s", name))
	}
}

// doList - list all entities inside a folder
func doList(clnt client.Client, targetURL string, recursive bool) error {
	var err error
	for contentCh := range clnt.List(recursive) {
		if contentCh.Err != nil {
			err = contentCh.Err
			break
		}
		contentName := contentCh.Content.Name
		if recursive {
			// this special handling is necessary since we are sending back absolute paths with in ListRecursive()
			// a user would not wish to see a URL just for recursive and not for regular List()
			//
			// To be consistent we have to filter them out
			contentName = strings.TrimPrefix(contentName, strings.TrimSuffix(targetURL, "/")+"/")
		}
		switch {
		case globalJSONFlag == true:
			printJSON(contentCh.Content)
		default:
			printContent(contentCh.Content.Time, contentCh.Content.Size, contentName, contentCh.Content.Type)
		}
	}
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	return nil
}
