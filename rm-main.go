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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// remove a file or folder.
var rmCmd = cli.Command{
	Name:   "rm",
	Usage:  "Remove file or bucket [WARNING: Use with care].",
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

   4. Remove incomplete upload of a file on Cloud storage:
      $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/louis/file01.mp3 incomplete

   5. Remove incomplete uploads of folder recursively on Cloud storage
      $ mc {{.Name}} https://s3.amazonaws.com/jazz-songs/louis/... incomplete force

`,
}

type rmListOnChannel struct {
	keyName string
	err     *probe.Error
}

type rmMessage struct {
	Name string `json:"name"`
}

func (msg rmMessage) String() string {
	return console.Colorize("Remove", fmt.Sprintf("removed ‘%s’", msg.Name))
}

func (msg rmMessage) JSON() string {
	msgBytes, err := json.Marshal(msg)
	fatalIf(probe.NewError(err), "Failed to marshal remove message.")
	return string(msgBytes)
}

func rmList(url string) <-chan rmListOnChannel {
	rmListCh := make(chan rmListOnChannel)
	clnt, err := url2Client(url)
	if err != nil {
		rmListCh <- rmListOnChannel{
			keyName: "",
			err:     err.Trace(url),
		}
		return rmListCh
	}
	in := clnt.List(true, false)
	var depthFirst func(currentDir string) (*client.Content, bool)
	depthFirst = func(currentDir string) (*client.Content, bool) {
		entry, ok := <-in
		for {
			if entry.Err != nil {
				rmListCh <- rmListOnChannel{
					keyName: "",
					err:     entry.Err,
				}
				return nil, false
			}
			if !ok || !strings.HasPrefix(entry.Content.URL.Path, currentDir) {
				return entry.Content, ok
			}
			if entry.Content.Type.IsRegular() {
				rmListCh <- rmListOnChannel{
					keyName: entry.Content.URL.String(),
					err:     nil,
				}
			}
			if entry.Content.Type.IsDir() {
				var content *client.Content
				content, ok = depthFirst(entry.Content.URL.String())
				rmListCh <- rmListOnChannel{
					keyName: entry.Content.URL.String(),
					err:     nil,
				}
				entry = client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
				continue
			}
			entry, ok = <-in
		}
	}
	go func() {
		depthFirst("")
		close(rmListCh)
	}()
	return rmListCh
}

func rmSingle(url string, rmPrint rmPrinterFunc) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(url), "Unable to get client object for "+url+".")
		return
	}
	err = clnt.Remove(false)
	if err == nil {
		rmPrint(rmMessage{url})
	}
	errorIf(err.Trace(url), "Unable to remove "+url+".")
}

func rmAll(url string, rmPrint rmPrinterFunc) {
	for rmListCh := range rmList(url) {
		if rmListCh.err != nil {
			// if rmList throws an error die here.
			fatalIf(rmListCh.err.Trace(), "Unable to list : "+url+" .")
		}
		newClnt, err := url2Client(rmListCh.keyName)
		if err != nil {
			errorIf(err.Trace(rmListCh.keyName), "Unable to create client object : "+rmListCh.keyName+" .")
			continue
		}
		err = newClnt.Remove(false)
		if err == nil {
			rmPrint(rmMessage{rmListCh.keyName})
		}
		errorIf(err.Trace(rmListCh.keyName), "Unable to remove : "+rmListCh.keyName+" .")
	}

}

func rmIncompleteUpload(url string, rmPrint rmPrinterFunc) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(), "Unable to get client object for "+url+" .")
		return
	}
	err = clnt.Remove(true)
	if err == nil {
		rmPrint(rmMessage{url})
	}
	errorIf(err.Trace(), "Unable to remove "+url+" .")
}

func rmAllIncompleteUploads(url string, rmPrint rmPrinterFunc) {
	clnt, err := url2Client(url)
	if err != nil {
		errorIf(err.Trace(url), "Unable to get client object for "+url+" .")
		return
	}
	for entry := range clnt.List(true, true) {
		newURL := entry.Content.URL
		newClnt, err := url2Client(newURL.String())
		if err != nil {
			errorIf(err.Trace(newURL.String()), "Unable to create client object : "+newURL.String()+" .")
			continue
		}
		err = newClnt.Remove(true)
		if err == nil {
			rmPrint(rmMessage{newURL.String()})
		}
		errorIf(err.Trace(newURL.String()), "Unable to remove : "+newURL.String()+" .")
	}
}

func setRmPalette(style string) {
	console.SetCustomPalette(map[string]*color.Color{
		"Remove": color.New(color.FgGreen, color.Bold),
	})
	if style == "light" {
		console.SetCustomPalette(map[string]*color.Color{
			"Remove": color.New(color.FgWhite, color.Bold),
		})
		return
	}
	/// Add more styles here
	if style == "nocolor" {
		// All coloring options exhausted, setting nocolor safely
		console.SetNoColor()
	}
}

func checkRmSyntax(ctx *cli.Context) {
	args := ctx.Args()

	var force bool
	var incomplete bool
	if !args.Present() || args.First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "rm", 1) // last argument is exit code.
	}
	if len(args) == 1 && args.Get(0) == "force" {
		return
	}
	if len(args) == 2 && args.Get(0) == "force" && args.Get(1) == "incomplete" ||
		len(args) == 2 && args.Get(1) == "force" && args.Get(0) == "incomplete" {
		return
	}
	if args.Last() == "force" {
		force = true
		args = args[:len(args)-1]
	}
	if args.Last() == "incomplete" {
		incomplete = true
		args = args[:len(args)-1]
	}

	// By this time we have sanitized the input args and now we have only the URLs parse them properly
	// and validate.
	URLs, err := args2URLs(args)
	fatalIf(err.Trace(ctx.Args()...), "Unable to parse arguments.")

	// If input validation fails then provide context sensitive help without displaying generic help message.
	// The context sensitive help is shown per argument instead of all arguments to keep the help display
	// as well as the code simple. Also most of the times there will be just one arg
	for _, url := range URLs {
		u := client.NewURL(url)
		var helpStr string
		if strings.HasSuffix(url, string(u.Separator)) {
			if incomplete {
				helpStr = "Usage : mc rm " + url + recursiveSeparator + " incomplete force"
			} else {
				helpStr = "Usage : mc rm " + url + recursiveSeparator + " force"
			}
			fatalIf(errDummy().Trace(), helpStr)
		}
		if isURLRecursive(url) && !force {
			if incomplete {
				helpStr = "Usage : mc rm " + url + " incomplete force"
			} else {
				helpStr = "Usage : mc rm " + url + " force"
			}
			fatalIf(errDummy().Trace(), helpStr)
		}
	}
}

type rmPrinterFunc func(rmMessage)

func rmPrinterFuncGenerate() rmPrinterFunc {
	var scanBar scanBarFunc
	if !globalJSONFlag && !globalQuietFlag {
		scanBar = scanBarFactory()
	}
	return func(msg rmMessage) {
		if globalJSONFlag || globalQuietFlag {
			printMsg(msg)
			return
		}
		scanBar(msg.Name)
	}
}

func mainRm(ctx *cli.Context) {
	checkRmSyntax(ctx)
	var incomplete bool
	var force bool

	setRmPalette(ctx.GlobalString("colors"))

	args := ctx.Args()
	if len(args) != 1 {
		if len(args) == 2 && args.Get(0) == "force" && args.Get(1) == "incomplete" ||
			len(args) == 2 && args.Get(0) == "incomplete" && args.Get(1) == "force" {
			args = args[:]
		} else {
			if args.Last() == "force" {
				force = true
				args = args[:len(args)-1]
			}
			if args.Last() == "incomplete" {
				incomplete = true
				args = args[:len(args)-1]
			}
		}
	}

	URLs, err := args2URLs(args)
	fatalIf(err.Trace(ctx.Args()...), "Unable to parse arguments.")

	rmPrint := rmPrinterFuncGenerate()

	// execute for incomplete
	if incomplete {
		for _, url := range URLs {
			if isURLRecursive(url) && force {
				rmAllIncompleteUploads(stripRecursiveURL(url), rmPrint)
			} else {
				rmIncompleteUpload(url, rmPrint)
			}
		}
		return
	}
	for _, url := range URLs {
		if isURLRecursive(url) && force {
			rmAll(stripRecursiveURL(url), rmPrint)
		} else {
			rmSingle(url, rmPrint)
		}
	}
	if !globalJSONFlag && !globalQuietFlag {
		console.Eraseline()
	}
}
