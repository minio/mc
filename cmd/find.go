// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/shlex"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
	"golang.org/x/text/unicode/norm"

	// golang does not support flat keys for path matching, find does
	"github.com/minio/pkg/v3/wildcard"
)

// findMessage holds JSON and string values for printing find command output.
type findMessage struct {
	contentMessage
}

// String calls tells the console what to print and how to print it.
func (f findMessage) String() string {
	var msg string
	msg += f.Key
	if f.VersionID != "" {
		msg += " (" + f.VersionID + ")"
	}
	return console.Colorize("Find", msg)
}

// JSON formats output to be JSON output.
func (f findMessage) JSON() string {
	return f.contentMessage.JSON()
}

// nameMatch is similar to filepath.Match but only matches the
// base path of the input, if we couldn't find a match we
// also proceed to look for similar strings alone and print it.
//
// pattern:
//
//	{ term }
//
// term:
//
//	'*'         matches any sequence of non-Separator characters
//	'?'         matches any single non-Separator character
//	'[' [ '^' ] { character-range } ']'
//	            character class (must be non-empty)
//	c           matches character c (c != '*', '?', '\\', '[')
//	'\\' c      matches character c
//
// character-range:
//
//	c           matches character c (c != '\\', '-', ']')
//	'\\' c      matches character c
//	lo '-' hi   matches character c for lo <= c <= hi
func nameMatch(pattern, path string) bool {
	matched, e := filepath.Match(pattern, filepath.Base(path))
	errorIf(probe.NewError(e).Trace(pattern, path), "Unable to match with input pattern.")
	if !matched {
		for _, pathComponent := range strings.Split(path, "/") {
			matched = pathComponent == pattern
			if matched {
				break
			}
		}
	}
	return matched
}

func patternMatch(pattern, match string) bool {
	pattern = strings.ToLower(pattern)
	match = strings.ToLower(match)
	return wildcard.Match(pattern, match)
}

// pathMatch reports whether path matches the wildcard pattern.
// supports  '*' and '?' wildcards in the pattern string.
// unlike path.Match(), considers a path as a flat name space
// while matching the pattern. The difference is illustrated in
// the example here https://play.golang.org/p/Ega9qgD4Qz .
func pathMatch(pattern, path string) bool {
	return wildcard.Match(pattern, path)
}

func getExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if pe, ok := err.(*exec.ExitError); ok {
		if es, ok := pe.Sys().(syscall.WaitStatus); ok {
			return es.ExitStatus()
		}
	}
	return 1
}

// execFind executes the input command line, additionally formats input
// for the command line in accordance with subsititution arguments.
func execFind(ctx context.Context, args string, fileContent contentMessage) {
	split, err := shlex.Split(args)
	if err != nil {
		console.Println(console.Colorize("FindExecErr", "Unable to parse --exec: "+err.Error()))
		os.Exit(getExitStatus(err))
	}
	if len(split) == 0 {
		return
	}
	for i, arg := range split {
		split[i] = stringsReplace(ctx, arg, fileContent)
	}
	cmd := exec.Command(split[0], split[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			console.Println(console.Colorize("FindExecErr", strings.TrimSpace(stderr.String())))
		}
		console.Println(console.Colorize("FindExecErr", err.Error()))
		// Return exit status of the command run
		os.Exit(getExitStatus(err))
	}
	console.PrintC(out.String())
}

// watchFind - enables listening on the input path, listens for all file/object
// created actions. Asynchronously executes the input command line, also allows
// formatting for the command line in accordance with subsititution arguments.
func watchFind(ctxCtx context.Context, ctx *findContext) {
	// Watch is not enabled, return quickly.
	if !ctx.watch {
		return
	}
	options := WatchOptions{
		Recursive: true,
		Events:    []string{"put"},
	}
	watchObj, err := ctx.clnt.Watch(ctxCtx, options)
	fatalIf(err.Trace(ctx.targetAlias), "Unable to watch with given options.")

	// Loop until user CTRL-C the command line.
	for {
		select {
		case <-globalContext.Done():
			console.Println()
			close(watchObj.DoneChan)
			return
		case events, ok := <-watchObj.Events():
			if !ok {
				return
			}

			for _, event := range events {
				time, e := time.Parse(time.RFC3339, event.Time)
				if e != nil {
					errorIf(probe.NewError(e).Trace(event.Time), "Unable to parse event time.")
					continue
				}

				find(ctxCtx, ctx, contentMessage{
					Key:  getAliasedPath(ctx, event.Path),
					Time: time,
					Size: event.Size,
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
}

// Descend at most (a non-negative integer) levels of files
// below the starting-prefix and trims the suffix. This function
// returns path as is without manipulation if the maxDepth is 0
// i.e (not set).
func trimSuffixAtMaxDepth(startPrefix, path, separator string, maxDepth uint) string {
	if maxDepth == 0 {
		return path
	}
	// Remove the requested prefix from consideration, maxDepth is
	// only considered for all other levels excluding the starting prefix.
	path = strings.TrimPrefix(path, startPrefix)
	pathComponents := strings.SplitAfter(path, separator)
	if len(pathComponents) >= int(maxDepth) {
		pathComponents = pathComponents[:maxDepth]
	}
	pathComponents = append([]string{startPrefix}, pathComponents...)
	return strings.Join(pathComponents, "")
}

// Get aliased path used finally in printing, trim paths to ensure
// that we have removed the fully qualified paths and original
// start prefix (targetAlias) is retained. This function also honors
// maxDepth if set then the resultant path will be trimmed at requested
// maxDepth.
func getAliasedPath(ctx *findContext, path string) string {
	separator := string(ctx.clnt.GetURL().Separator)
	prefixPath := ctx.clnt.GetURL().String()
	var aliasedPath string
	if ctx.targetAlias != "" {
		aliasedPath = ctx.targetAlias + strings.TrimPrefix(path, strings.TrimSuffix(ctx.targetFullURL, separator))
	} else {
		aliasedPath = path
		// look for prefix path, if found filter at that, Watch calls
		// for example always provide absolute path. So for relative
		// prefixes we need to employ this kind of code.
		if i := strings.Index(path, prefixPath); i > 0 {
			aliasedPath = path[i:]
		}
	}
	return trimSuffixAtMaxDepth(ctx.targetURL, aliasedPath, separator, ctx.maxDepth)
}

func find(ctxCtx context.Context, ctx *findContext, fileContent contentMessage) {
	// Match the incoming content, didn't match return.
	if !matchFind(ctx, fileContent) {
		return
	} // For all matching content

	// proceed to either exec, format the output string.
	if ctx.execCmd != "" {
		execFind(ctxCtx, ctx.execCmd, fileContent)
		return
	}
	if ctx.printFmt != "" {
		fileContent.Key = stringsReplace(ctxCtx, ctx.printFmt, fileContent)
	}
	printMsg(findMessage{fileContent})
}

// doFind - find is main function body which interprets and executes
// all the input parameters.
func doFind(ctxCtx context.Context, ctx *findContext) error {
	// If watch is enabled we will wait on the prefix perpetually
	// for all I/O events until canceled by user, if watch is not enabled
	// following defer is a no-op.
	defer watchFind(ctxCtx, ctx)

	lstOptions := ListOptions{
		WithOlderVersions: ctx.withVersions,
		WithDeleteMarkers: ctx.withVersions,
		Recursive:         true,
		ShowDir:           DirFirst,
		WithMetadata:      len(ctx.matchMeta) > 0 || len(ctx.matchTags) > 0,
	}

	// iterate over all content which is within the given directory
	for content := range ctx.clnt.List(globalContext, lstOptions) {
		if content.Err != nil {
			switch content.Err.ToGoError().(type) {
			// handle this specifically for filesystem related errors.
			case BrokenSymlink:
				errorIf(content.Err.Trace(ctx.clnt.GetURL().String()), "Unable to list broken link.")
				continue
			case TooManyLevelsSymlink:
				errorIf(content.Err.Trace(ctx.clnt.GetURL().String()), "Unable to list too many levels link.")
				continue
			case PathNotFound:
				errorIf(content.Err.Trace(ctx.clnt.GetURL().String()), "Unable to list folder.")
				continue
			case PathInsufficientPermission:
				errorIf(content.Err.Trace(ctx.clnt.GetURL().String()), "Unable to list folder.")
				continue
			}
			fatalIf(content.Err.Trace(ctx.clnt.GetURL().String()), "Unable to list folder.")
			continue
		}
		if content.StorageClass == s3StorageClassGlacier {
			continue
		}

		fileKeyName := getAliasedPath(ctx, content.URL.String())
		fileContent := contentMessage{
			Key:       fileKeyName,
			VersionID: content.VersionID,
			Time:      content.Time.Local(),
			Size:      content.Size,
			Metadata:  content.UserMetadata,
			Tags:      content.Tags,
		}

		// Match the incoming content, didn't match return.
		if !matchFind(ctx, fileContent) {
			continue
		} // For all matching content

		// proceed to either exec, format the output string.
		if ctx.execCmd != "" {
			execFind(ctxCtx, ctx.execCmd, fileContent)
			continue
		}
		if ctx.printFmt != "" {
			fileContent.Key = stringsReplace(ctxCtx, ctx.printFmt, fileContent)
		}

		printMsg(findMessage{fileContent})
	}

	// Success, notice watch will execute in defer only if enabled and this call
	// will return after watch is canceled.
	return nil
}

// stringsReplace - formats the string to remove {} and replace each
// with the appropriate argument
func stringsReplace(ctx context.Context, args string, fileContent contentMessage) string {
	// replace all instances of {}
	str := args

	str = strings.ReplaceAll(str, "{}", fileContent.Key)

	// replace all instances of {""}
	str = strings.ReplaceAll(str, `{""}`, strconv.Quote(fileContent.Key))

	// replace all instances of {base}
	str = strings.ReplaceAll(str, "{base}", filepath.Base(fileContent.Key))

	// replace all instances of {"base"}
	str = strings.ReplaceAll(str, `{"base"}`, strconv.Quote(filepath.Base(fileContent.Key)))

	// replace all instances of {dir}
	str = strings.ReplaceAll(str, "{dir}", filepath.Dir(fileContent.Key))

	// replace all instances of {"dir"}
	str = strings.ReplaceAll(str, `{"dir"}`, strconv.Quote(filepath.Dir(fileContent.Key)))

	// replace all instances of {size}
	str = strings.ReplaceAll(str, "{size}", humanize.IBytes(uint64(fileContent.Size)))

	// replace all instances of {"size"}
	str = strings.ReplaceAll(str, `{"size"}`, strconv.Quote(humanize.IBytes(uint64(fileContent.Size))))

	// replace all instances of {time}
	str = strings.ReplaceAll(str, "{time}", fileContent.Time.Format(printDate))

	// replace all instances of {"time"}
	str = strings.ReplaceAll(str, `{"time"}`, strconv.Quote(fileContent.Time.Format(printDate)))

	// replace all instances of {url}
	if strings.Contains(str, "{url}") {
		str = strings.ReplaceAll(str, "{url}", getShareURL(ctx, fileContent.Key))
	}

	// replace all instances of {"url"}
	if strings.Contains(str, `{"url"}`) {
		str = strings.ReplaceAll(str, `{"url"}`, strconv.Quote(getShareURL(ctx, fileContent.Key)))
	}

	// replace all instances of {version}
	str = strings.ReplaceAll(str, `{version}`, fileContent.VersionID)

	// replace all instances of {"version"}
	str = strings.ReplaceAll(str, `{"version"}`, strconv.Quote(fileContent.VersionID))

	return str
}

// matchFind matches whether fileContent matches appropriately with standard
// "pattern matching" flags requested by the user, such as "name", "path", "regex" ..etc.
func matchFind(ctx *findContext, fileContent contentMessage) (match bool) {
	match = true
	prefixPath := ctx.targetURL
	// Add separator only if targetURL doesn't already have separator.
	if !strings.HasPrefix(prefixPath, string(ctx.clnt.GetURL().Separator)) {
		prefixPath = ctx.targetURL + string(ctx.clnt.GetURL().Separator)
	}
	// Trim the prefix such that we will apply file path matching techniques
	// on path excluding the starting prefix.
	path := strings.TrimPrefix(fileContent.Key, prefixPath)
	if match && ctx.ignorePattern != "" {
		match = !pathMatch(ctx.ignorePattern, path)
	}
	if match && ctx.namePattern != "" {
		match = nameMatch(ctx.namePattern, path)
	}
	if match && ctx.pathPattern != "" {
		match = pathMatch(ctx.pathPattern, path)
	}
	if match && ctx.regexPattern != nil {
		match = ctx.regexPattern.MatchString(path)
	}
	if match && ctx.olderThan != "" {
		match = !isOlder(fileContent.Time, ctx.olderThan)
	}
	if match && ctx.newerThan != "" {
		match = !isNewer(fileContent.Time, ctx.newerThan)
	}
	if match && ctx.largerSize > 0 {
		match = int64(ctx.largerSize) < fileContent.Size
	}
	if match && ctx.smallerSize > 0 {
		match = int64(ctx.smallerSize) > fileContent.Size
	}
	if match && len(ctx.matchMeta) > 0 {
		match = matchMetadataRegexMaps(ctx.matchMeta, fileContent.Metadata)
	}
	if match && len(ctx.matchTags) > 0 {
		match = matchRegexMaps(ctx.matchTags, fileContent.Tags)
	}
	return match
}

// 7 days in seconds.
var defaultSevenDays = time.Duration(604800) * time.Second

// getShareURL is used in conjunction with the {url} substitution
// argument to generate and return presigned URLs, returns error if any.
func getShareURL(ctx context.Context, path string) string {
	targetAlias, targetURLFull, _, err := expandAlias(path)
	fatalIf(err.Trace(path), "Unable to expand alias.")

	clnt, err := newClientFromAlias(targetAlias, targetURLFull)
	fatalIf(err.Trace(targetAlias, targetURLFull), "Unable to initialize client instance from alias.")

	content, err := clnt.Stat(ctx, StatOptions{})
	fatalIf(err.Trace(targetURLFull, targetAlias), "Unable to lookup file/object.")

	// Skip if it is a directory.
	if content.Type.IsDir() {
		return ""
	}

	objectURL := content.URL.String()
	newClnt, err := newClientFromAlias(targetAlias, objectURL)
	fatalIf(err.Trace(targetAlias, objectURL), "Unable to initialize new client from alias.")

	// Set default expiry for each url (point of no longer valid), to be 7 days
	shareURL, err := newClnt.ShareDownload(ctx, "", defaultSevenDays)
	fatalIf(err.Trace(targetAlias, objectURL), "Unable to generate share url.")

	return shareURL
}

// getRegexMap returns a map from the StringSlice key.
// Each entry must be key=regex.
// Will exit with error if an un-parsable entry is found.
func getRegexMap(cliCtx *cli.Context, key string) map[string]*regexp.Regexp {
	sl := cliCtx.StringSlice(key)
	if len(sl) == 0 {
		return nil
	}
	reMap := make(map[string]*regexp.Regexp, len(sl))
	for _, v := range sl {
		split := strings.SplitN(v, "=", 2)
		if len(split) < 2 {
			err := probe.NewError(fmt.Errorf("want one = separator, got none"))
			fatalIf(err.Trace(v), "Unable to split key+value. Must be key=regex")
		}
		// No value means it should not exist or be empty.
		if len(split[1]) == 0 {
			reMap[split[0]] = nil
			continue
		}
		// Normalize character encoding.
		norm := norm.NFC.String(split[1])
		var err error
		reMap[split[0]], err = regexp.Compile(norm)
		if err != nil {
			fatalIf(probe.NewError(err), fmt.Sprintf("Unable to compile metadata regex for %s=%s", split[0], split[1]))
		}
	}
	return reMap
}

// matchRegexMaps will check if all regexes in 'm' match values in 'v' with the same key.
// If a regex is nil, it must either not exist in v or have a 0 length value.
func matchRegexMaps(m map[string]*regexp.Regexp, v map[string]string) bool {
	for k, reg := range m {
		if reg == nil {
			if v[k] != "" {
				return false
			}
			// Does not exist or empty, that is fine.
			continue
		}
		val, ok := v[k]
		if !ok || !reg.MatchString(norm.NFC.String(val)) {
			return false
		}
	}
	return true
}

// matchMetadataRegexMaps will check if all regexes in 'm' match values in 'v' with the same key.
// If a regex is nil, it must either not exist in v or have a 0 length value.
func matchMetadataRegexMaps(m map[string]*regexp.Regexp, v map[string]string) bool {
	for k, reg := range m {
		if reg == nil {
			if v[k] != "" {
				return false
			}
			// Does not exist or empty, that is fine.
			continue
		}
		val, ok := v[k]
		if !ok {
			val, ok = v[http.CanonicalHeaderKey(fmt.Sprintf("X-Amz-Meta-%s", k))]
		}
		if !ok || !reg.MatchString(norm.NFC.String(val)) {
			return false
		}
	}
	return true
}
