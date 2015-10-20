/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"os"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio-xl/pkg/probe"
)

// remove a file or folder.
var rmCmd = cli.Command{
	Name:   "rm",
	Usage:  "Remove file or bucket.",
	Action: mainRm,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} TARGET [incomplete] [force]

   incomplete - remove incomplete uploads
   force      - force recursive remove

EXAMPLES:
   1. Remove a file on Cloud storage
     $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/louis/file01.mp3

   2. Remove a folder recursively on Cloud storage
     $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/louis/... force

   3. Remove a bucket on Minio cloud storage
     $ mc {{.Name}} https://play.minio.io:9000/mongodb-backup

   4. Remove a bucket on Cloud storage recursively
     $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/... force

   5. Remove a file on local filesystem:
      $ mc {{.Name}} march/expenses.doc

   6. Remove a file named "force" on local filesystem:
      $ mc {{.Name}} force force

   7. Remove incomplete upload of a file on Cloud storage:
      $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/louis/file01.mp3 incomplete

   2. Remove incomplete uploads of folder recursively on Cloud storage
      $ mc {{.Name}} force https://s3.amazonaws.com/jazz-songs/louis/... incomplete force

`,
}

func rmList(url string) (<-chan string, *probe.Error) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to get client object for "+url)
		return nil, err.Trace()
	}
	in := clnt.List(true, false)
	out := make(chan string)

	var depthFirst func(currentDir string) (*client.Content, bool)

	depthFirst = func(currentDir string) (*client.Content, bool) {
		entry, ok := <-in
		for {
			if !ok || !strings.HasPrefix(entry.Content.Name, currentDir) {
				return entry.Content, ok
			}
			if entry.Content.Type.IsRegular() {
				out <- entry.Content.Name
			}
			if entry.Content.Type.IsDir() {
				var content *client.Content
				content, ok = depthFirst(entry.Content.Name)
				out <- entry.Content.Name
				entry = client.ContentOnChannel{Content: content}
				continue
			}
			entry, ok = <-in
		}
	}

	go func() {
		depthFirst("")
		close(out)
	}()
	return out, nil
}

func rm(url string) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to get client object for "+url)
		return
	}
	err = clnt.Remove()
	errorIf(err.Trace(), "Unable to remove "+url)
}

func rmAll(url string) {
	urlPartial1 := url2Dir(url)
	out, err := rmList(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to List "+url)
		return
	}
	for urlPartial2 := range out {
		urlFull := urlPartial1 + urlPartial2
		newclnt, e := url2Client(urlFull)
		if e != nil {
			errorIf(e, "Unable to create client object : "+urlFull)
			continue
		}
		err = newclnt.Remove()
		errorIf(err, "Unable to remove : "+urlFull)
	}
}

func rmIncompleteUpload(url string) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to get client object for "+url)
		return
	}
	err = clnt.RemoveIncompleteUpload()
	errorIf(err.Trace(), "Unable to remove "+url)
}

func rmAllIncompleteUploads(url string) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to get client object for "+url)
		return
	}
	urlPartial1 := url2Dir(url)
	ch := clnt.List(true, true)
	for entry := range ch {
		urlFull := urlPartial1 + entry.Content.Name
		newclnt, e := url2Client(urlFull)
		if e != nil {
			errorIf(e, "Unable to create client object : "+urlFull)
			continue
		}
		err = newclnt.RemoveIncompleteUpload()
		errorIf(err, "Unable to remove : "+urlFull)
	}
}

func checkRmSyntax(ctx *cli.Context) {
	args, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(), "args2URL failed")
	var force bool
	length := len(args)
	if length == 0 {
		cli.ShowCommandHelpAndExit(ctx, "rm", 1) // last argument is exit code.
	}
	if len(args) == 1 && args[0] == "force" {
		cli.ShowCommandHelpAndExit(ctx, "rm", 1)
	}
	if args[length-1] == "force" {
		force = true
		args = args[:length-1]
		length--
	}
	if args[length-1] == "incomplete" {
		args = args[:length-1]
	}
	// If input validation fails then provide context sensitive help without displaying generic help message.
	// The context sensitive help is shown per argument instead of all arguments to keep the help display
	// as well as the code simple. Also most of the times there will be just one arg
	for _, arg := range args {
		url := client.NewURL(arg)
		if strings.HasSuffix(arg, string(url.Separator)) {
			helpStr := "Usage : mc rm " + arg + recursiveSeparator + " force"
			fatalIf(errDummy().Trace(), helpStr)
		}
		if isURLRecursive(arg) && !force {
			helpStr := "Usage : mc rm " + arg + " force"
			fatalIf(errDummy().Trace(), helpStr)
		}
		if url.Type == client.Filesystem {
			// For local file system we don't support "mc rm fileprefix..." just like the behavior of "mc ls fileprefix..."
			// So recursive delete has to be of the form "mc rm dir1/dir2/..."
			isRecursive := isURLRecursive(arg)
			path := stripRecursiveURL(arg)
			if isRecursive && (strings.HasSuffix(path, string(url.Separator)) == false) {
				helpStr := "Usage : mc rm " + path + string(url.Separator) + recursiveSeparator + " force"
				fatalIf(errDummy().Trace(), helpStr)
			}
			_, content, err := url2Stat(path)
			if err != nil {
				fatalIf(err.Trace(), "url2stat error on "+arg)
			}
			if content.Type&os.ModeDir != 0 && !isRecursive {
				helpStr := "Usage : mc rm " + arg + string(url.Separator) + recursiveSeparator + " force"
				fatalIf(errDummy().Trace(), helpStr)
			}
			continue
		}
	}
}

func mainRm(ctx *cli.Context) {
	checkRmSyntax(ctx)
	args, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(), "args2URL failed")
	var incomplete bool
	length := len(args)

	if args[length-1] == "force" {
		args = args[:length-1]
		length--
	}
	if args[length-1] == "incomplete" {
		args = args[:length-1]
		incomplete = true
	}
	if incomplete {
		for _, arg := range args {
			if isURLRecursive(arg) {
				url := stripRecursiveURL(arg)
				rmAllIncompleteUploads(url)
			} else {
				rmIncompleteUpload(arg)
			}
		}
	} else {
		for _, arg := range args {
			if isURLRecursive(arg) {
				url := stripRecursiveURL(arg)
				rmAll(url)
			} else {
				rm(arg)
			}
		}
	}
}
