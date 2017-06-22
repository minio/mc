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

package cmd

import (
	//"fmt"
	"path/filepath"
	"regexp"
	"strings"
	//"github.com/dustin/go-humanize"
	//"github.com/minio/mc/pkg/console"
	//"github.com/minio/minio/pkg/probe"
)

func nameFlag(fileContent contentMessage, desired string) {
	object := filepath.Base(fileContent.Key)
	//filepath.Match adds wildcard support
	if ok, _ := filepath.Match(desired, object); ok {
		printMsg(fileContent)
	}
}

// .*/bucket1/.*
func pathFlag(fileContent contentMessage, desired string) {
	dir := filepath.Dir(fileContent.Key) + "/"
	if ok, _ := regexp.MatchString(desired, dir); ok {
		printMsg(fileContent)
	}
}

func regexFlag(fileContent contentMessage, desired string) {
	dir, file := filepath.Split(fileContent.Key)

	if ok, _ := filepath.Match(desired, file); ok {
		printMsg(fileContent)
	}

	if ok, _ := regexp.MatchString(desired, dir); ok {
		printMsg(fileContent)
	}
}

//return an error if we encounter an error
func doFind(clnt Client, state int, desired string) error {
	prefixPath := clnt.GetURL().Path
	separator := string(clnt.GetURL().Separator)

	if !strings.HasSuffix(prefixPath, separator) {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, separator)+1]
	}

	var cErr error

	//set recurisve to be true, set isIncomplete to be false, set Directory to be iota (DitOpt/ list directory option)
	for content := range clnt.List(true, false, DirNone) {

		//Convert any os specific delimiters to "/"
		contentURL := filepath.ToSlash(content.URL.Path)
		prefixPath = filepath.ToSlash(prefixPath)

		//Trim prefix path from the content path
		contentURL = strings.TrimPrefix(contentURL, prefixPath)
		content.URL.Path = contentURL

		//parsed content is going to be a structure of all of the different files at the base of a tree
		//basically stops whenever an object is encountered
		fileContent := parseContent(content)
		switch state {
		case 0:
			nameFlag(fileContent, desired)
		case 1:
			pathFlag(fileContent, desired)
		case 2:
			regexFlag(fileContent, desired)
		}
	}
	//we will add error checking later right now but for the time being this should just return nil
	return cErr

}
