/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"path/filepath"
	"sort"
	"strings"

	"github.com/minio/cli"
	"github.com/posener/complete"
)

// fsComplete knows how to complete file/dir names by the given path
type fsComplete struct{}

func (fs fsComplete) Predict(a complete.Args) (prediction []string) {
	return complete.PredictFiles("*").Predict(a)
}

// Complete S3 path. If the prediction result is only one directory,
// then recursively scans it. This is needed to satisfy posener/complete
// (look at posener/complete.PredictFiles)
func completeS3Path(s3Path string) (prediction []string) {

	// Convert alias/bucket/incompl to alias/bucket/ to list its contents
	parentDirPath := filepath.Dir(s3Path) + "/"
	clnt, err := newClient(parentDirPath)
	if err != nil {
		return nil
	}

	// Calculate alias from the path
	alias := splitStr(s3Path, "/", 3)[0]

	// List dirPath content and only pick elements that corresponds
	// to the path that we want to complete
	for content := range clnt.List(false, false, DirFirst) {
		completeS3Path := alias + getKey(content)
		if content.Type.IsDir() {
			if !strings.HasSuffix(completeS3Path, "/") {
				completeS3Path += "/"
			}
		}
		if strings.HasPrefix(completeS3Path, s3Path) {
			prediction = append(prediction, completeS3Path)
		}
	}

	// If completion found only one directory, recursively scan it.
	if len(prediction) == 1 && strings.HasSuffix(prediction[0], "/") {
		prediction = append(prediction, completeS3Path(prediction[0])...)
	}

	return
}

// s3Complete knows how to complete an mc s3 path
type s3Complete struct {
	deepLevel int
}

func (s3 s3Complete) Predict(a complete.Args) (prediction []string) {
	defer func() {
		sort.Strings(prediction)
	}()

	loadMcConfig = loadMcConfigFactory()
	conf, err := loadMcConfig()
	if err != nil {
		return nil
	}

	arg := a.Last

	if strings.IndexByte(arg, '/') == -1 {
		// Only predict alias since '/' is not found
		for alias := range conf.Hosts {
			if strings.HasPrefix(alias, arg) {
				prediction = append(prediction, alias+"/")
			}
		}
		if len(prediction) == 1 && strings.HasSuffix(prediction[0], "/") {
			prediction = append(prediction, completeS3Path(prediction[0])...)
		}
	} else {
		// Complete S3 path until the specified path deep level
		if s3.deepLevel > 0 {
			if strings.Count(arg, "/") >= s3.deepLevel {
				return []string{arg}
			}
		}
		// Predict S3 path
		prediction = completeS3Path(arg)
	}

	return
}

// aliasComplete only completes aliases
type aliasComplete struct{}

func (al aliasComplete) Predict(a complete.Args) (prediction []string) {
	defer func() {
		sort.Strings(prediction)
	}()

	loadMcConfig = loadMcConfigFactory()
	conf, err := loadMcConfig()
	if err != nil {
		return nil
	}

	arg := a.Last
	for alias := range conf.Hosts {
		if strings.HasPrefix(alias, arg) {
			prediction = append(prediction, alias+"/")
		}
	}

	return
}

var s3Completer = s3Complete{}
var aliasCompleter = aliasComplete{}
var fsCompleter = fsComplete{}

// The list of all commands supported by mc with their mapping
// with their bash completer function
var completeCmds = map[string]complete.Predictor{
	"/ls":     complete.PredictOr(s3Completer, fsCompleter),
	"/cp":     complete.PredictOr(s3Completer, fsCompleter),
	"/rm":     complete.PredictOr(s3Completer, fsCompleter),
	"/rb":     complete.PredictOr(s3Complete{deepLevel: 2}, fsCompleter),
	"/cat":    complete.PredictOr(s3Completer, fsCompleter),
	"/head":   complete.PredictOr(s3Completer, fsCompleter),
	"/diff":   complete.PredictOr(s3Completer, fsCompleter),
	"/find":   complete.PredictOr(s3Completer, fsCompleter),
	"/mirror": complete.PredictOr(s3Completer, fsCompleter),
	"/pipe":   complete.PredictOr(s3Completer, fsCompleter),
	"/stat":   complete.PredictOr(s3Completer, fsCompleter),
	"/watch":  complete.PredictOr(s3Completer, fsCompleter),
	"/policy": complete.PredictOr(s3Completer, fsCompleter),

	"/mb":  aliasCompleter,
	"/sql": s3Completer,

	"/admin/info":       aliasCompleter,
	"/admin/heal":       s3Completer,
	"/admin/credential": aliasCompleter,

	"/admin/config/get": aliasCompleter,
	"/admin/config/set": aliasCompleter,

	"/admin/service/status":  aliasCompleter,
	"/admin/service/restart": aliasCompleter,
	"/admin/service/stop":    aliasCompleter,

	"/admin/profile/start": aliasCompleter,
	"/admin/profile/stop":  aliasCompleter,

	"/admin/policy/add":    aliasCompleter,
	"/admin/policy/list":   aliasCompleter,
	"/admin/policy/remove": aliasCompleter,

	"/admin/user/add":     aliasCompleter,
	"/admin/user/policy:": aliasCompleter,
	"/admin/user/disable": aliasCompleter,
	"/admin/user/enable":  aliasCompleter,
	"/admin/user/list":    aliasCompleter,
	"/admin/user/remove":  aliasCompleter,

	"/event/add":    aliasCompleter,
	"/event/list":   aliasCompleter,
	"/event/remove": aliasCompleter,

	"/session/clear":  nil,
	"/session/list":   nil,
	"/session/resume": nil,

	"/share/download": nil,
	"/share/list":     nil,
	"/share/upload":   nil,

	"/config/host/add":    nil,
	"/config/host/list":   aliasCompleter,
	"/config/host/remove": aliasCompleter,

	"/update":  nil,
	"/version": nil,
}

// flagsToCompleteFlags transforms a cli.Flag to complete.Flags
// understood by posener/complete library.
func flagsToCompleteFlags(flags []cli.Flag) complete.Flags {
	var complFlags = make(complete.Flags)
	for _, f := range flags {
		for _, s := range strings.Split(f.GetName(), ",") {
			var flagName string
			s = strings.TrimSpace(s)
			if len(s) == 1 {
				flagName = "-" + s
			} else {
				flagName = "--" + s
			}
			complFlags[flagName] = complete.PredictNothing
		}
	}
	return complFlags
}

// This function recursively transforms cli.Command to complete.Command
// understood by posener/complete library.
func cmdToCompleteCmd(cmd cli.Command, parentPath string) complete.Command {
	var complCmd complete.Command
	complCmd.Sub = make(complete.Commands)

	for _, subCmd := range cmd.Subcommands {
		complCmd.Sub[subCmd.Name] = cmdToCompleteCmd(subCmd, parentPath+"/"+cmd.Name)
	}

	complCmd.Flags = flagsToCompleteFlags(cmd.Flags)
	complCmd.Args = completeCmds[parentPath+"/"+cmd.Name]
	return complCmd
}

// Main function to answer to bash completion calls
func mainComplete() error {
	// Recursively register all commands and subcommands
	// along with global and local flags
	var complCmds = make(complete.Commands)
	for _, cmd := range appCmds {
		complCmds[cmd.Name] = cmdToCompleteCmd(cmd, "")
	}
	complFlags := flagsToCompleteFlags(globalFlags)
	mcComplete := complete.Command{
		Sub:         complCmds,
		GlobalFlags: complFlags,
	}
	// Answer to bash completion call
	complete.New("mc", mcComplete).Run()
	return nil
}
