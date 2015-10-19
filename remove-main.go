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
   mc {{.Name}} TARGET

EXAMPLES:
   1. Remove a file on Cloud storage
     $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/louis/file01.mp3

   2. Remove a folder recursively on Cloud storage
     $ mc {{.Name}} force https://s3.amazonaws.com/jazz-songs/louis/...

   3. Remove a bucket on Minio cloud storage
     $ mc {{.Name}} https://play.minio.io:9000/mongodb-backup

   4. Remove a bucket on Cloud storage recursively
     $ mc {{.Name}} force https://s3.amazonaws.com/jazz-songs/...

   5. Remove a file on local filesystem:
      $ mc {{.Name}} march/expenses.doc

   6. Remove a file named "force" on local filesystem:
      $ mc {{.Name}} force force
`,
}

func rmList(url string) (<-chan string, *probe.Error) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to get client object for "+url)
		return nil, err.Trace()
	}
	in := clnt.List(true)
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
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to get client object for "+url)
		return
	}
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
	_, err = clnt.Stat()
	if err == nil {
		err = clnt.Remove()
		errorIf(err, "Unable to remove : "+clnt.URL().String())
	}
}

func checkRmSyntax(ctx *cli.Context) {
	args, err := args2URLs(ctx.Args())
	fatalIf(err.Trace(), "args2URL failed")
	var force bool
	if len(args) == 0 {
		cli.ShowCommandHelpAndExit(ctx, "rm", 1) // last argument is exit code.
	}
	if len(args) == 1 && args[0] == "force" {
		cli.ShowCommandHelpAndExit(ctx, "rm", 1)
	}
	if args[0] == "force" {
		force = true
		args = args[1:]
	}
	// If input validation fails then provide context sensitive help without displaying generic help message.
	// The context sensitive help is shown per argument instead of all arguments to keep the help display
	// as well as the code simple. Also most of the times there will be just one arg
	for _, arg := range args {
		url := client.NewURL(arg)
		if strings.HasSuffix(arg, string(url.Separator)) {
			helpStr := "Usage : mc rm force " + arg + recursiveSeparator
			fatalIf(errDummy().Trace(), helpStr)
		}
		if isURLRecursive(arg) && !force {
			helpStr := "Usage : mc rm force " + arg
			fatalIf(errDummy().Trace(), helpStr)
		}
		if url.Type == client.Filesystem {
			// For local file system we don't support "mc rm fileprefix..." just like the behavior of "mc ls fileprefix..."
			// So recursive delete has to be of the form "mc rm dir1/dir2/..."
			isRecursive := isURLRecursive(arg)
			path := stripRecursiveURL(arg)
			if isRecursive && (strings.HasSuffix(path, string(url.Separator)) == false) {
				helpStr := "Usage : mc rm force " + path + string(url.Separator) + recursiveSeparator
				fatalIf(errDummy().Trace(), helpStr)
			}
			_, content, err := url2Stat(path)
			if err != nil {
				fatalIf(err.Trace(), "url2stat error on "+arg)
			}
			if content.Type&os.ModeDir != 0 && !isRecursive {
				helpStr := "Usage : mc rm force " + arg + string(url.Separator) + recursiveSeparator
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
	if args[0] == "force" {
		args = args[1:]
	}
	for _, arg := range args {
		if isURLRecursive(arg) {
			url := stripRecursiveURL(arg)
			rmAll(url)
		} else {
			rm(arg)
		}
	}
}
