/*
 * Minio Client (C) 2017 Minio, Inc.
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

// Package cmd stores all of the mc utilities
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"

	// golang does not support flat keys for path matching, find does
	"github.com/minio/minio/pkg/wildcard"
)

// findMSG holds JSON and string values for printing
type findMSG struct {
	Path string `json:"path"`
}

// String calls tells the console what to print and how to print it
func (f findMSG) String() string {
	return console.Colorize("Find", f.Path)
}

// JSON formats output to be JSON output
func (f findMSG) JSON() string {
	f.Path = "path"
	jsonMessageBytes, e := json.Marshal(f)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// nameMatch pattern matches off of the base of the filepath
func nameMatch(path, pattern string) (bool, error) {
	base := filepath.Base(path)

	return filepath.Match(pattern, base)
}

// pathMatch pattern matches off of of the entire filepath
func pathMatch(path, pattern string) bool {
	return wildcard.Match(pattern, path)
}

// regexMatch pattern matches off of the entire filepath using regex library
func regexMatch(path, pattern string) (bool, error) {
	return regexp.MatchString(pattern, path)
}

// doFindPrint prints the output in accordance with the supplied substitution arguments
func doFindPrint(path string, ctx *cli.Context, fileContent contentMessage) {
	printString := SubArgsHelper(ctx.String("print"), path, fileContent)
	printMsg(findMSG{
		Path: printString,
	})
}

// doFindExec passes the users input along to the command line, also dealing with substitution arguments
func doFindExec(ctx *cli.Context, path string, fileContent contentMessage) {
	commandString := SubArgsHelper(ctx.String("exec"), path, fileContent)
	commandArgs := strings.Split(commandString, " ")

	cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		console.Fatalln(err)
		console.Fatalln()
	}
	console.Println(string(out.Bytes()))
}

// doFindWatch watches the server side to see if a given action is preformed, if yes then a can provided by exec can be executed
func doFindWatch(path string, ctx *cli.Context) {
	params := watchParams{
		recursive: true,
		accountID: fmt.Sprintf("%d", time.Now().Unix()),
		events:    []string{"put"},
	}

	pathnameParts := strings.SplitAfter(path, "/")
	alias := strings.TrimSuffix(pathnameParts[0], "/")

	_, e := getHostConfig(alias) // extract the hostname alias from the path name if present
	noAlias := (e == nil)

	clnt, content, err := url2Stat(path)
	fatalIf(err, "Unable to construct client")

	fileContent := parseContent(content)
	if noAlias {
		//pass hostname (alias) to to watchEvents so that it can remove from pathnames returned from watch
		watchEvents(ctx, clnt, params, alias, fileContent)
		return
	}
	watchEvents(ctx, clnt, params, "", fileContent)
	return
}

// doFindOlder checks to see if the given object was created before or after the given time at which an object was created
func doFindOlder(createTime time.Time, pattern string) bool {
	i, err := TimeHelper(pattern)
	now := time.Now()
	fatalIf(probe.NewError(err), "Error parsing string passed to flag older")

	//find all time in which the time in which the object was just created is after the current time
	t := time.Date(now.Year(), now.Month(), now.Day()-i, now.Hour(), now.Minute(), 0, 0, time.UTC)
	return createTime.Before(t)
}

// doFindNewer checks to see if the given object was created before the given threshold
func doFindNewer(createTime time.Time, pattern string) bool {
	i, err := TimeHelper(pattern)
	now := time.Now()

	fatalIf(probe.NewError(err), "Error parsing string passed to flag newer")

	t := time.Date(now.Year(), now.Month(), now.Day()-i, now.Hour(), now.Minute(), 0, 0, time.UTC)
	return createTime.After(t) || createTime.Equal(t)
}

// doFindLargerSize checks to see if the given object is larger than the given threshold
func doFindLargerSize(size int64, pattern string) bool {
	i, err := humanize.ParseBytes(pattern)
	fatalIf(probe.NewError(err), "Error parsing string passed to flag larger")

	return int64(i) < size
}

// doFindSmallerSize checks to see if the given object is smaller than the given threshold
func doFindSmallerSize(size int64, pattern string) bool {
	i, err := humanize.ParseBytes(pattern)
	fatalIf(probe.NewError(err), "Error parsing string passed to flag smaller")

	return int64(i) > size
}

// DoFind is used to handle most of the users input
func DoFind(clnt Client, ctx *cli.Context) {
	pathnameParts := strings.SplitAfter(ctx.Args().Get(0), "/")
	alias := strings.TrimSuffix(pathnameParts[0], "/")
	_, err := getHostConfig(alias)

	// iterate over all content which is within the given directory
	for content := range clnt.List(true, false, DirNone) {
		fileContent := parseContent(content)
		filePath := fileContent.Key

		// traversing in a object store not a file path
		if err == nil {
			filePath = path.Join(alias, filePath)
		}

		if ctx.String("maxdepth") != "" {
			i, e := strconv.Atoi(ctx.String("maxdepth"))
			s := ""

			fatalIf(probe.NewError(e), "Error parsing string passed to flag maxdepth")

			// we are going to be parsing the path by x amounts
			pathParts := strings.SplitAfter(filePath, "/")

			// handle invalid params
			// ex. user specifies:
			// maxdepth 2, but the given object only has a maxdepth of 1
			if (len(pathParts)-1) < i || i < 0 {

				// -1 is meant to handle each the array being 0 indexed, but the size not being 0 indexed
				i = len(pathParts) - 1
			}

			// append portions of path into a string
			for j := 0; j <= i; j++ {
				s += pathParts[j]
			}

			filePath = s
			fileContent.Key = s
		}

		// maxdepth can modify the filepath to end in a directory
		// to be consistent with find we do not want to be listing directories
		// so any parms which end in / will be ignored
		if !strings.HasSuffix(filePath, "/") {

			orBool := ctx.Bool("or")

			match := fileContentMatch(fileContent, orBool, ctx)

			if match && ctx.String("print") != "" {
				doFindPrint(filePath, ctx, fileContent)
			} else if match {
				printMsg(findMSG{
					Path: filePath,
				})
			}

			if !ctx.Bool("watch") && match && ctx.String("exec") != "" {
				doFindExec(ctx, filePath, fileContent)
			}
		}

		if ctx.Bool("watch") {
			doFindWatch(ctx.Args().Get(0), ctx)
		}
	}

}

// SubArgsHelper formats the string to remove {} and replace each with the appropriate argument
func SubArgsHelper(args, path string, fileContent contentMessage) string {

	// replace all instances of {}
	str := args
	if strings.Contains(str, "{}") {
		str = strings.Replace(str, "{}", path, -1)
	}

	// replace all instances of {base}
	if strings.Contains(str, "{base}") {
		str = strings.Replace(str, "{base}", filepath.Base(path), -1)
	}

	// replace all instances of {dir}
	if strings.Contains(str, "{dir}") {
		str = strings.Replace(str, "{dir}", filepath.Dir(path), -1)
	}

	// replace all instances of {size}
	if strings.Contains(str, "{size}") {
		s := humanize.IBytes(uint64(fileContent.Size))
		str = strings.Replace(str, "{size}", s, -1)
	}

	if strings.Contains(str, "{url}") {
		s := GetPurl(path)
		str = strings.Replace(str, "{url}", s, -1)
	}

	if strings.Contains(str, "{time}") {
		t := fileContent.Time.String()
		str = strings.Replace(str, "{time}", t, -1)
	}

	// replace all instances of {""}
	if strings.Contains(str, "{\""+"\"}") {
		str = strings.Replace(str, "{\""+"\"}", strconv.Quote(path), -1)
	}

	// replace all instances of {"base"}
	if strings.Contains(str, "{\""+"base"+"\"}") {
		str = strings.Replace(str, "{\""+"base"+"\"}", strconv.Quote(filepath.Base(path)), -1)
	}

	// replace all instances of {"dir"}
	if strings.Contains(str, "{\""+"dir"+"\"}") {
		str = strings.Replace(str, "{\""+"dir"+"\"}", strconv.Quote(filepath.Dir(path)), -1)
	}

	if strings.Contains(str, "{\""+"url"+"\"}") {
		s := GetPurl(path)
		str = strings.Replace(str, "{\""+"url"+"\"}", strconv.Quote(s), -1)
	}

	if strings.Contains(str, "{\""+"size"+"\"}") {
		s := humanize.IBytes(uint64(fileContent.Size))
		str = strings.Replace(str, "{\""+"size"+"\"}", strconv.Quote(s), -1)
	}

	if strings.Contains(str, "{\""+"time"+"\"}") {
		str = strings.Replace(str, "{\""+"time"+"\"}", strconv.Quote(fileContent.Time.String()), -1)
	}

	return str
}

// watchEvents used in conjunction with doFindWatch method to detect and preform desired action when an object is created
func watchEvents(ctx *cli.Context, clnt Client, params watchParams, alias string, fileContent contentMessage) {
	watchObj, err := clnt.Watch(params)
	fatalIf(err, "Cannot watch with given params")

	// get client url and remove
	cliTemp, _ := getHostConfig(alias)
	cliTempURL := cliTemp.URL

	// enables users to kill using the control + c
	trapCh := signalTrap(os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(1)

	// opens a channel of all content created on the server, any errors detected,
	// and scanning for the user input (Control + C)
	go func() {
		defer wg.Done()

		// loop until user kills the channel input
		for {
			select {
			case <-trapCh:
				console.Println()
				close(watchObj.doneChan)
				return
			case event, ok := <-watchObj.Events():
				if !ok {
					return
				}

				time, _ := time.Parse(time.RFC822, event.Time)
				msg := contentMessage{
					Key:  alias + strings.TrimPrefix(event.Path, cliTempURL),
					Time: time,
					Size: event.Size,
				}

				msg.Key = alias + msg.Key

				// check to see if the newly create object matches the given params
				match := fileContentMatch(msg, ctx.Bool("or"), ctx)

				if match && ctx.String("exec") != "" {
					doFindExec(ctx, msg.Key, fileContent)
				}

				if match && ctx.String("print") != "" {
					doFindExec(ctx, msg.Key, fileContent)
				} else if match {
					printMsg(findMSG{
						Path: msg.Key,
					})
				}

			case err, ok := <-watchObj.Errors():
				if !ok {
					return
				}
				errorIf(err, "Unable to watch for events.")
				return
			}
		}
	}()

	wg.Wait()
}

// fileContentMatch is used to take the params passed to find, in addition to the current
// file and call the appropriate "pattern matching methods"
func fileContentMatch(fileContent contentMessage, orOp bool, ctx *cli.Context) bool {
	match := true

	if (match && !orOp) && ctx.String("ignore") != "" {
		match = !pathMatch(fileContent.Key, ctx.String("ignore"))
	}

	// verify that the newly added object matches all of the other specified params
	if (match || orOp) && ctx.String("name") != "" {
		tmp, err := nameMatch(fileContent.Key, ctx.String("name"))
		match = tmp

		fatalIf(probe.NewError(err), "Name could not be matched")
	}

	if (match || orOp) && ctx.String("path") != "" {
		match = pathMatch(fileContent.Key, ctx.String("path"))
	}

	if (match || orOp) && ctx.String("regex") != "" {
		temp, e := regexMatch(fileContent.Key, ctx.String("regex"))
		match = temp

		fatalIf(probe.NewError(e), "Regex could not be matched")
	}

	if (match || orOp) && ctx.String("older") != "" {
		match = doFindOlder(fileContent.Time, ctx.String("older"))
	}

	if (match || orOp) && ctx.String("newer") != "" {
		match = doFindNewer(fileContent.Time, ctx.String("newer"))
	}

	if (match || orOp) && ctx.String("larger") != "" {
		match = doFindLargerSize(fileContent.Size, ctx.String("larger"))
	}

	if (match || orOp) && ctx.String("smaller") != "" {
		match = doFindSmallerSize(fileContent.Size, ctx.String("smaller"))
	}

	return match
}

// TimeHelper is used in conjunction with the Newer and Older flags to convert the input
// into an interperable integer (in days)
func TimeHelper(pattern string) (int, error) {
	var i int
	var t string
	var err error
	conversion := map[string]int{
		"d": 1,
		"w": 7,
		"m": 30,
		"y": 365,
	}
	i, err = strconv.Atoi(pattern)
	if err != nil {
		t = pattern[len(pattern)-2:]
		i, err = strconv.Atoi(pattern[:len(pattern)-2])

		if err != nil {
			return 0, err

		}
	}

	return i * conversion[strings.ToLower(t)], nil
}

// GetPurl is used in conjunction with the {url} substitution argument to return presigned URLs
func GetPurl(path string) string {
	targetAlias, targetURLFull, _, err := expandAlias(path)
	fatalIf(err, "Error with expand alias")
	clnt, err := newClientFromAlias(targetAlias, targetURLFull)
	fatalIf(err, "Error with newClientFromAlias")

	isIncomplete := false

	objectsCh := make(chan *clientContent)

	content, err := clnt.Stat(isIncomplete)

	fatalIf(err, "Error with client stat")

	// piping all content into the object channel to be processed
	go func() {
		defer close(objectsCh)
		objectsCh <- content
	}()

	// get content from channel to be converted into presigned URL
	for content := range objectsCh {
		fatalIf(content.Err, "Error with content")

		if content.Type.IsDir() {
			continue
		}

		objectURL := content.URL.String()
		newClnt, err := newClientFromAlias(targetAlias, objectURL)
		fatalIf(err, "Error with newClientFromAlias")

		// set default expiry for each url (point of no longer valid), to be 7 days
		expiry := time.Duration(604800) * time.Second
		shareURL, err := newClnt.ShareDownload(expiry)
		fatalIf(err, "Error with ShareDownloa")

		return shareURL
	}
	return ""
}
