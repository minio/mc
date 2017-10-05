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

package cmd

import (
	"bytes"
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

// findMessage holds JSON and string values for printing find command output.
type findMessage struct {
	contentMessage
}

// String calls tells the console what to print and how to print it.
func (f findMessage) String() string {
	return console.Colorize("Find", f.contentMessage.Key)
}

// JSON formats output to be JSON output.
func (f findMessage) JSON() string {
	return f.contentMessage.JSON()
}

// nameMatch is similar to filepath.Match but only matches the
// base path of the input.
//
// pattern:
// 	{ term }
// term:
// 	'*'         matches any sequence of non-Separator characters
// 	'?'         matches any single non-Separator character
// 	'[' [ '^' ] { character-range } ']'
// 	            character class (must be non-empty)
// 	c           matches character c (c != '*', '?', '\\', '[')
// 	'\\' c      matches character c
// character-range:
// 	c           matches character c (c != '\\', '-', ']')
// 	'\\' c      matches character c
// 	lo '-' hi   matches character c for lo <= c <= hi
//
func nameMatch(pattern, path string) bool {
	matched, e := filepath.Match(pattern, filepath.Base(path))
	errorIf(probe.NewError(e).Trace(pattern, path), "Unable to match with input pattern")
	return matched
}

// pathMatch reports whether path matches the wildcard pattern.
// supports  '*' and '?' wildcards in the pattern string.
// unlike path.Match(), considers a path as a flat name space
// while matching the pattern. The difference is illustrated in
// the example here https://play.golang.org/p/Ega9qgD4Qz .
func pathMatch(pattern, path string) bool {
	return wildcard.Match(pattern, path)
}

// regexMatch reports whether path matches the regex pattern.
func regexMatch(pattern, path string) bool {
	matched, e := regexp.MatchString(pattern, path)
	errorIf(probe.NewError(e).Trace(pattern), "Unable to regex match with input pattern")
	return matched
}

// olderThanMatch matches whether if the createTime is older than the allowed threshold.
func olderThanMatch(threshold string, createTime time.Time) bool {
	t, err := parseTime(threshold)
	fatalIf(err.Trace(threshold), "Unable to parse input threshold value into time.Time")
	return createTime.Before(t)
}

// newerThanMatch matches whether if the createTime is newer than the allowed threshold.
func newerThanMatch(threshold string, createTime time.Time) bool {
	t, err := parseTime(threshold)
	fatalIf(err.Trace(threshold), "Unable to parse input threshold value into time.Time")
	return createTime.After(t) || createTime.Equal(t)
}

// largerSizeMatch matches whether if the input size bytes is larger than the allowed threshold.
func largerSizeMatch(bytesFmt string, size int64) bool {
	i, e := humanize.ParseBytes(bytesFmt)
	fatalIf(probe.NewError(e).Trace(bytesFmt), "Unable to parse input threshold value into size bytes")

	return int64(i) < size
}

// smallerSizeMatch matches whether if the input size bytes is smaller than the allowed threshold.
func smallerSizeMatch(bytesFmt string, size int64) bool {
	i, e := humanize.ParseBytes(bytesFmt)
	fatalIf(probe.NewError(e).Trace(bytesFmt), "Unable to parse input threshold value into size bytes")

	return int64(i) > size
}

// doFindPrint prints the output in accordance with the supplied substitution arguments
func doFindPrint(ctx *cli.Context, fileContent contentMessage) {
	fileContent.Key = stringsReplace(ctx.String("print"), fileContent)
	printMsg(findMessage{fileContent})
}

// doFindExec executes the input command line, additionally formats input
// for the command line in accordance with subsititution arguments.
func doFindExec(ctx *cli.Context, fileContent contentMessage) {
	commandString := stringsReplace(ctx.String("exec"), fileContent)
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

// doFindWatch - enables listening on the input path, listens for all file/object
// created actions. Asynchronously executes the input command line, also allows
// formatting for the command line in accordance with subsititution arguments.
func doFindWatch(ctx *cli.Context, path string) {
	params := watchParams{
		recursive: true,
		accountID: fmt.Sprintf("%d", time.Now().Unix()),
		events:    []string{"put"},
	}

	// Extract the hostname alias from the path name if present
	targetAlias, _, _, err := expandAlias(path)
	fatalIf(err.Trace(path), "Unable to expand alias.")

	clnt, content, err := url2Stat(path)
	fatalIf(err.Trace(path), "Unable to lookup.")

	fileContent := parseContent(content)
	watchEventsFind(ctx, clnt, params, targetAlias, fileContent)
}

// doFind - find is main function body which interprets and executes
// all the input parameters.
func doFind(targetURL string, clnt Client, ctx *cli.Context) error {
	targetAlias, _, _, err := expandAlias(targetURL)
	fatalIf(err.Trace(targetURL), "Unable to expand alias")

	separator := string(clnt.GetURL().Separator)
	// iterate over all content which is within the given directory
	for content := range clnt.List(true, false, DirNone) {
		if content.Err != nil {
			switch content.Err.ToGoError().(type) {
			// handle this specifically for filesystem related errors.
			case BrokenSymlink:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list broken link.")
				continue
			case TooManyLevelsSymlink:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list too many levels link.")
				continue
			case PathNotFound:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			case PathInsufficientPermission:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			case ObjectOnGlacier:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "")
				continue
			}
			fatalIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			continue
		}

		fileContent := parseContent(content)
		if targetAlias != "" {
			fileContent.Key = path.Join(targetAlias, fileContent.Key)
		}

		if ctx.String("maxdepth") != "" {
			maxDepth, e := strconv.Atoi(ctx.String("maxdepth"))
			fatalIf(probe.NewError(e), "Error parsing string passed to flag maxdepth")

			newPath := ""

			// We are going to convert path into newPath based on the
			// maxdepth value, split the strings at separator properly.
			pathParts := strings.SplitAfter(fileContent.Key, separator)

			// Check if max-depth is 2, but if the given object only
			// has a maximum depth of 1 use that instead.
			if (len(pathParts)-1) < maxDepth || maxDepth < 0 {
				maxDepth = len(pathParts) - 1
			}

			// Construct new path based on the requested maxDepth.
			for j := 0; j <= maxDepth; j++ {
				newPath += pathParts[j]
			}

			fileContent.Key = newPath
		}

		// Maxdepth can modify the filepath to end as a directory prefix
		// to be consistent with the find behavior, we wont list directories
		// so any paths which end with a separator are ignored.
		if strings.HasSuffix(fileContent.Key, separator) {
			continue
		}

		match := doFindMatch(ctx, fileContent)
		if !ctx.Bool("watch") && match && ctx.String("exec") != "" {
			doFindExec(ctx, fileContent)
		} else if match && ctx.String("print") != "" {
			doFindPrint(ctx, fileContent)
		} else if match {
			printMsg(findMessage{fileContent})
		}
		if ctx.Bool("watch") {
			doFindWatch(ctx, ctx.Args().Get(0))
		}
	}
	return nil
}

// stringsReplace - formats the string to remove {} and replace each
// with the appropriate argument
func stringsReplace(args string, fileContent contentMessage) string {
	// replace all instances of {}
	str := args
	if strings.Contains(str, "{}") {
		str = strings.Replace(str, "{}", fileContent.Key, -1)
	}

	// replace all instances of {""}
	if strings.Contains(str, `{""}`) {
		str = strings.Replace(str, `{""}`, strconv.Quote(fileContent.Key), -1)
	}

	// replace all instances of {base}
	if strings.Contains(str, "{base}") {
		str = strings.Replace(str, "{base}", filepath.Base(fileContent.Key), -1)
	}

	// replace all instances of {"base"}
	if strings.Contains(str, `{"base"}`) {
		str = strings.Replace(str, `{"base"}`, strconv.Quote(filepath.Base(fileContent.Key)), -1)
	}

	// replace all instances of {dir}
	if strings.Contains(str, "{dir}") {
		str = strings.Replace(str, "{dir}", filepath.Dir(fileContent.Key), -1)
	}

	// replace all instances of {"dir"}
	if strings.Contains(str, `{"dir"}`) {
		str = strings.Replace(str, `{"dir"}`, strconv.Quote(filepath.Dir(fileContent.Key)), -1)
	}

	// replace all instances of {size}
	if strings.Contains(str, "{size}") {
		str = strings.Replace(str, "{size}", humanize.IBytes(uint64(fileContent.Size)), -1)
	}

	// replace all instances of {"size"}
	if strings.Contains(str, `{"size"}`) {
		str = strings.Replace(str, `{"size"}`, strconv.Quote(humanize.IBytes(uint64(fileContent.Size))), -1)
	}

	// replace all instances of {time}
	if strings.Contains(str, "{time}") {
		str = strings.Replace(str, "{time}", fileContent.Time.Format(printDate), -1)
	}

	// replace all instances of {"time"}
	if strings.Contains(str, `{"time"}`) {
		str = strings.Replace(str, `{"time"}`, strconv.Quote(fileContent.Time.Format(printDate)), -1)
	}

	// replace all instances of {url}
	if strings.Contains(str, "{url}") {
		str = strings.Replace(str, "{url}", getShareURL(fileContent.Key), -1)
	}

	// replace all instances of {"url"}
	if strings.Contains(str, `{"url"}`) {
		str = strings.Replace(str, `{"url"}`, strconv.Quote(getShareURL(fileContent.Key)), -1)
	}

	return str
}

// watchEventsFind used in conjunction with doFindWatch method to detect
// and preform desired action when an object is created
func watchEventsFind(ctx *cli.Context, clnt Client, params watchParams, alias string, fileContent contentMessage) {
	watchObj, err := clnt.Watch(params)
	fatalIf(err.Trace(alias), "Cannot watch with given params")

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
				fileContent := contentMessage{
					Key:  alias + strings.TrimPrefix(event.Path, clnt.GetURL().String()),
					Time: time,
					Size: event.Size,
				}

				// check to see if the newly create object matches the given params
				match := doFindMatch(ctx, fileContent)
				if match && ctx.String("exec") != "" {
					doFindExec(ctx, fileContent)
				} else if match && ctx.String("print") != "" {
					doFindPrint(ctx, fileContent)
				} else if match {
					printMsg(findMessage{fileContent})
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

// doFindMatch matches whether fileContent matches appropriately with standard
// "pattern matching" flags requested by the user, such as "name", "path", "regex" ..etc.
func doFindMatch(ctx *cli.Context, fileContent contentMessage) (match bool) {
	match = true
	if !ctx.Bool("or") {
		if ctx.String("ignore") != "" {
			match = !pathMatch(ctx.String("ignore"), fileContent.Key)
		}
		return match
	}
	if ctx.String("name") != "" {
		match = nameMatch(ctx.String("name"), fileContent.Key)
	} else if ctx.String("path") != "" {
		match = pathMatch(ctx.String("path"), fileContent.Key)
	} else if ctx.String("regex") != "" {
		match = regexMatch(ctx.String("regex"), fileContent.Key)
	} else if ctx.String("older") != "" {
		match = olderThanMatch(ctx.String("older"), fileContent.Time)
	} else if ctx.String("newer") != "" {
		match = newerThanMatch(ctx.String("newer"), fileContent.Time)
	} else if ctx.String("larger") != "" {
		match = largerSizeMatch(ctx.String("larger"), fileContent.Size)
	} else if ctx.String("smaller") != "" {
		match = smallerSizeMatch(ctx.String("smaller"), fileContent.Size)
	}
	return match
}

// parseTime - parses input value into a corresponding time value in
// time.Time by adding the input time duration to local UTC time.Now().
func parseTime(duration string) (time.Time, *probe.Error) {
	if duration == "" {
		return time.Time{}, errInvalidArgument().Trace(duration)
	}

	conversion := map[string]int{
		"d": 1,
		"w": 7,
		"m": 30,
		"y": 365,
	}

	// Parse the incoming pattern if its exact number.
	i, e := strconv.Atoi(duration)
	if e != nil {
		// If cant parse as regular string look for
		// a conversion multiplier, either d,w,m,y.
		p := duration[len(duration)-1:]
		i, e = strconv.Atoi(duration[:len(duration)-1])
		if e != nil {
			// if we still cant parse, user input is invalid, return error.
			return time.Time{}, probe.NewError(e)
		}
		i = i * conversion[strings.ToLower(p)]
	}

	now := UTCNow()

	// Find all time in which the time in which the object was just created is after the current time
	t := time.Date(now.Year(), now.Month(), now.Day()-i, now.Hour(), now.Minute(), 0, 0, time.UTC)

	// if we reach this line, user has passed a valid alphanumeric string
	return t, nil
}

// 7 days in seconds.
var defaultSevenDays = time.Duration(604800) * time.Second

// getShareURL is used in conjunction with the {url} substitution
// argument to generate and return presigned URLs, returns error if any.
func getShareURL(path string) string {
	targetAlias, targetURLFull, _, err := expandAlias(path)
	fatalIf(err.Trace(path), "Unable to expand alias")

	clnt, err := newClientFromAlias(targetAlias, targetURLFull)
	fatalIf(err.Trace(targetAlias, targetURLFull), "Unable to newClientFromAlias")

	content, err := clnt.Stat(false)
	fatalIf(err.Trace(targetURLFull, targetAlias), "Unable to lookup file/object.")

	// Skip if its a directory.
	if content.Type.IsDir() {
		return ""
	}

	objectURL := content.URL.String()
	newClnt, err := newClientFromAlias(targetAlias, objectURL)
	fatalIf(err.Trace(targetAlias, objectURL), "Unable to initialize new client from alias.")

	// Set default expiry for each url (point of no longer valid), to be 7 days
	shareURL, err := newClnt.ShareDownload(defaultSevenDays)
	fatalIf(err.Trace(targetAlias, objectURL), "Unable to generate share url.")

	return shareURL
}
