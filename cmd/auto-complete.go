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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/minio/cli"
	"github.com/posener/complete"
)

// fsComplete knows how to complete file/dir names by the given path
type fsComplete struct{}

// predictPathWithTilde completes an FS path which starts with a `~/`
func (fs fsComplete) predictPathWithTilde(a complete.Args) []string {
	homeDir, e := os.UserHomeDir()
	if e != nil || homeDir == "" {
		return nil
	}
	// Clean the home directory path
	homeDir = strings.TrimRight(homeDir, "/")

	// Replace the first occurrence of ~ with the real path and complete
	a.Last = strings.Replace(a.Last, "~", homeDir, 1)
	predictions := complete.PredictFiles("*").Predict(a)

	// Restore ~ to avoid disturbing the completion user experience
	for i := range predictions {
		predictions[i] = strings.Replace(predictions[i], homeDir, "~", 1)
	}

	return predictions
}

func (fs fsComplete) Predict(a complete.Args) []string {
	if strings.HasPrefix(a.Last, "~/") {
		return fs.predictPathWithTilde(a)
	}
	return complete.PredictFiles("*").Predict(a)
}

func completeAdminConfigKeys(aliasPath, keyPrefix string) (prediction []string) {
	// Convert alias/bucket/incompl to alias/bucket/ to list its contents
	parentDirPath := filepath.Dir(aliasPath) + "/"
	clnt, err := newAdminClient(parentDirPath)
	if err != nil {
		return nil
	}

	h, e := clnt.HelpConfigKV(globalContext, "", "", false)
	if e != nil {
		return nil
	}

	for _, hkv := range h.KeysHelp {
		if strings.HasPrefix(hkv.Key, keyPrefix) {
			prediction = append(prediction, hkv.Key)
		}
	}

	return prediction
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
	for content := range clnt.List(globalContext, ListOptions{Recursive: false, ShowDir: DirFirst}) {
		cmplS3Path := alias + getKey(content)
		if content.Type.IsDir() {
			if !strings.HasSuffix(cmplS3Path, "/") {
				cmplS3Path += "/"
			}
		}
		if strings.HasPrefix(cmplS3Path, s3Path) {
			prediction = append(prediction, cmplS3Path)
		}
	}

	// If completion found only one directory, recursively scan it.
	if len(prediction) == 1 && strings.HasSuffix(prediction[0], "/") {
		prediction = append(prediction, completeS3Path(prediction[0])...)
	}

	return
}

type adminConfigComplete struct{}

func (adm adminConfigComplete) Predict(a complete.Args) (prediction []string) {
	defer func() {
		sort.Strings(prediction)
	}()

	loadMcConfig = loadMcConfigFactory()
	conf, err := loadMcConfig()
	if err != nil {
		return
	}

	// We have already predicted the keys, we are done.
	if len(a.Completed) == 3 {
		return
	}

	arg := a.Last
	lastArg := a.LastCompleted
	if _, ok := conf.Aliases[filepath.Clean(a.LastCompleted)]; !ok {
		if strings.IndexByte(arg, '/') == -1 {
			// Only predict alias since '/' is not found
			for alias := range conf.Aliases {
				if strings.HasPrefix(alias, arg) {
					prediction = append(prediction, alias+"/")
				}
			}
		} else {
			prediction = completeAdminConfigKeys(arg, "")
		}
	} else {
		prediction = completeAdminConfigKeys(lastArg, arg)
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
		for alias := range conf.Aliases {
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
	for alias := range conf.Aliases {
		if strings.HasPrefix(alias, arg) {
			prediction = append(prediction, alias+"/")
		}
	}

	return
}

var (
	adminConfigCompleter = adminConfigComplete{}
	s3Completer          = s3Complete{}
	aliasCompleter       = aliasComplete{}
	fsCompleter          = fsComplete{}
)

// The list of all commands supported by mc with their mapping
// with their bash completer function
var completeCmds = map[string]complete.Predictor{
	// S3 API level commands
	"/ls":        complete.PredictOr(s3Completer, fsCompleter),
	"/cp":        complete.PredictOr(s3Completer, fsCompleter),
	"/mv":        complete.PredictOr(s3Completer, fsCompleter),
	"/rm":        complete.PredictOr(s3Completer, fsCompleter),
	"/rb":        complete.PredictOr(s3Complete{deepLevel: 2}, fsCompleter),
	"/cat":       complete.PredictOr(s3Completer, fsCompleter),
	"/head":      complete.PredictOr(s3Completer, fsCompleter),
	"/diff":      complete.PredictOr(s3Completer, fsCompleter),
	"/find":      complete.PredictOr(s3Completer, fsCompleter),
	"/mirror":    complete.PredictOr(s3Completer, fsCompleter),
	"/pipe":      complete.PredictOr(s3Completer, fsCompleter),
	"/stat":      complete.PredictOr(s3Completer, fsCompleter),
	"/watch":     complete.PredictOr(s3Completer, fsCompleter),
	"/anonymous": complete.PredictOr(s3Completer, fsCompleter),
	"/tree":      complete.PredictOr(s3Complete{deepLevel: 2}, fsCompleter),
	"/du":        complete.PredictOr(s3Complete{deepLevel: 2}, fsCompleter),

	"/retention/set":   s3Completer,
	"/retention/clear": s3Completer,
	"/retention/info":  s3Completer,

	"/legalhold/set":   s3Completer,
	"/legalhold/clear": s3Completer,
	"/legalhold/info":  s3Completer,

	"/sql": s3Completer,
	"/mb":  aliasCompleter,

	"/event/add":    s3Complete{deepLevel: 2},
	"/event/list":   s3Complete{deepLevel: 2},
	"/event/remove": s3Complete{deepLevel: 2},

	"/encrypt/set":   s3Complete{deepLevel: 2},
	"/encrypt/info":  s3Complete{deepLevel: 2},
	"/encrypt/clear": s3Complete{deepLevel: 2},

	"/replicate/add":     s3Complete{deepLevel: 2},
	"/replicate/edit":    s3Complete{deepLevel: 2},
	"/replicate/update":  s3Complete{deepLevel: 2},
	"/replicate/list":    s3Complete{deepLevel: 2},
	"/replicate/remove":  s3Complete{deepLevel: 2},
	"/replicate/backlog": s3Complete{deepLevel: 2},

	"/replicate/export":        s3Complete{deepLevel: 2},
	"/replicate/import":        s3Complete{deepLevel: 2},
	"/replicate/status":        s3Complete{deepLevel: 2},
	"/replicate/resync/start":  s3Complete{deepLevel: 3},
	"/replicate/resync/status": s3Complete{deepLevel: 3},

	"/tag/list":   s3Completer,
	"/tag/remove": s3Completer,
	"/tag/set":    s3Completer,

	"/version/info":    s3Complete{deepLevel: 2},
	"/version/enable":  s3Complete{deepLevel: 2},
	"/version/suspend": s3Complete{deepLevel: 2},

	"/lock/compliance": s3Completer,
	"/lock/governance": s3Completer,
	"/lock/clear":      s3Completer,
	"/lock/info":       s3Completer,

	"/share/download": s3Completer,
	"/share/list":     nil,
	"/share/upload":   s3Completer,

	"/ilm/list":    s3Complete{deepLevel: 2},
	"/ilm/add":     s3Complete{deepLevel: 2},
	"/ilm/edit":    s3Complete{deepLevel: 2},
	"/ilm/remove":  s3Complete{deepLevel: 2},
	"/ilm/export":  s3Complete{deepLevel: 2},
	"/ilm/import":  s3Complete{deepLevel: 2},
	"/ilm/restore": s3Completer,

	"/ilm/rule/list":    s3Complete{deepLevel: 2},
	"/ilm/rule/add":     s3Complete{deepLevel: 2},
	"/ilm/rule/edit":    s3Complete{deepLevel: 2},
	"/ilm/rule/remove":  s3Complete{deepLevel: 2},
	"/ilm/rule/export":  s3Complete{deepLevel: 2},
	"/ilm/rule/import":  s3Complete{deepLevel: 2},
	"/ilm/rule/restore": s3Completer,

	"/undo": s3Completer,

	// Admin API commands MinIO only.
	"/admin/heal": s3Completer,

	"/admin/info": aliasCompleter,
	"/admin/logs": aliasCompleter,

	"/admin/config/get":     adminConfigCompleter,
	"/admin/config/set":     adminConfigCompleter,
	"/admin/config/reset":   adminConfigCompleter,
	"/admin/config/import":  aliasCompleter,
	"/admin/config/export":  aliasCompleter,
	"/admin/config/history": aliasCompleter,
	"/admin/config/restore": aliasCompleter,

	"/admin/decom/start":         aliasCompleter,
	"/admin/decom/status":        aliasCompleter,
	"/admin/decom/cancel":        aliasCompleter,
	"/admin/decommission/start":  aliasCompleter,
	"/admin/decommission/status": aliasCompleter,
	"/admin/decommission/cancel": aliasCompleter,

	"/admin/rebalance/start":  aliasCompleter,
	"/admin/rebalance/status": aliasCompleter,
	"/admin/rebalance/stop":   aliasCompleter,

	"/admin/trace":     aliasCompleter,
	"/admin/speedtest": aliasCompleter,
	"/admin/console":   aliasCompleter,
	"/admin/update":    aliasCompleter,
	"/admin/inspect":   s3Completer,
	"/admin/top/locks": aliasCompleter,
	"/admin/top/api":   aliasCompleter,

	"/admin/scanner/status": aliasCompleter,
	"/admin/scanner/trace":  aliasCompleter,

	"/admin/service/stop":     aliasCompleter,
	"/admin/service/restart":  aliasCompleter,
	"/admin/service/freeze":   aliasCompleter,
	"/admin/service/unfreeze": aliasCompleter,

	"/admin/prometheus/generate": aliasCompleter,
	"/admin/prometheus/metrics":  aliasCompleter,

	"/admin/profile/start": aliasCompleter,
	"/admin/profile/stop":  aliasCompleter,

	"/idp/openid/add":     aliasCompleter,
	"/idp/openid/update":  aliasCompleter,
	"/idp/openid/remove":  aliasCompleter,
	"/idp/openid/list":    aliasCompleter,
	"/idp/openid/info":    aliasCompleter,
	"/idp/openid/enable":  aliasCompleter,
	"/idp/openid/disable": aliasCompleter,

	"/idp/openid/accesskey/list":    aliasCompleter,
	"/idp/openid/accesskey/ls":      aliasCompleter,
	"/idp/openid/accesskey/info":    aliasCompleter,
	"/idp/openid/accesskey/remove":  aliasCompleter,
	"/idp/openid/accesskey/rm":      aliasCompleter,
	"/idp/openid/accesskey/edit":    aliasCompleter,
	"/idp/openid/accesskey/enable":  aliasCompleter,
	"/idp/openid/accesskey/disable": aliasCompleter,

	"/idp/ldap/add":     aliasCompleter,
	"/idp/ldap/update":  aliasCompleter,
	"/idp/ldap/remove":  aliasCompleter,
	"/idp/ldap/list":    aliasCompleter,
	"/idp/ldap/info":    aliasCompleter,
	"/idp/ldap/enable":  aliasCompleter,
	"/idp/ldap/disable": aliasCompleter,

	"/idp/ldap/policy/entities": aliasCompleter,
	"/idp/ldap/policy/attach":   aliasCompleter,
	"/idp/ldap/policy/detach":   aliasCompleter,

	"/idp/ldap/accesskey/create":            aliasCompleter,
	"/idp/ldap/accesskey/create-with-login": aliasCompleter,
	"/idp/ldap/accesskey/list":              aliasCompleter,
	"/idp/ldap/accesskey/ls":                aliasCompleter,
	"/idp/ldap/accesskey/remove":            aliasCompleter,
	"/idp/ldap/accesskey/rm":                aliasCompleter,
	"/idp/ldap/accesskey/info":              aliasCompleter,
	"/idp/ldap/accesskey/edit":              aliasCompleter,
	"/idp/ldap/accesskey/enable":            aliasCompleter,
	"/idp/ldap/accesskey/disable":           aliasCompleter,
	"/idp/ldap/accesskey/sts-revoke":        aliasCompleter,

	"/admin/accesskey/create":     aliasCompleter,
	"/admin/accesskey/list":       aliasCompleter,
	"/admin/accesskey/ls":         aliasCompleter,
	"/admin/accesskey/remove":     aliasCompleter,
	"/admin/accesskey/rm":         aliasCompleter,
	"/admin/accesskey/info":       aliasCompleter,
	"/admin/accesskey/edit":       aliasCompleter,
	"/admin/accesskey/enable":     aliasCompleter,
	"/admin/accesskey/disable":    aliasCompleter,
	"/admin/accesskey/sts-revoke": aliasCompleter,

	"/admin/policy/info":     aliasCompleter,
	"/admin/policy/update":   aliasCompleter,
	"/admin/policy/add":      aliasCompleter,
	"/admin/policy/remove":   aliasCompleter,
	"/admin/policy/create":   aliasCompleter,
	"/admin/policy/list":     aliasCompleter,
	"/admin/policy/attach":   aliasCompleter,
	"/admin/policy/detach":   aliasCompleter,
	"/admin/policy/entities": aliasCompleter,

	"/admin/user/add":     aliasCompleter,
	"/admin/user/disable": aliasCompleter,
	"/admin/user/enable":  aliasCompleter,
	"/admin/user/list":    aliasCompleter,
	"/admin/user/remove":  aliasCompleter,
	"/admin/user/info":    aliasCompleter,
	"/admin/user/policy":  aliasCompleter,

	"/admin/user/svcacct/add":     aliasCompleter,
	"/admin/user/svcacct/list":    aliasCompleter,
	"/admin/user/svcacct/remove":  aliasCompleter,
	"/admin/user/svcacct/info":    aliasCompleter,
	"/admin/user/svcacct/edit":    aliasCompleter,
	"/admin/user/svcacct/set":     aliasCompleter,
	"/admin/user/svcacct/enable":  aliasCompleter,
	"/admin/user/svcacct/disable": aliasCompleter,

	"/admin/user/sts/info": aliasCompleter,

	"/admin/group/add":     aliasCompleter,
	"/admin/group/disable": aliasCompleter,
	"/admin/group/enable":  aliasCompleter,
	"/admin/group/list":    aliasCompleter,
	"/admin/group/remove":  aliasCompleter,
	"/admin/group/info":    aliasCompleter,

	"/admin/bucket/remote/add":    aliasCompleter,
	"/admin/bucket/remote/edit":   aliasCompleter,
	"/admin/bucket/remote/remove": aliasCompleter,
	"/admin/bucket/quota":         aliasCompleter,
	"/admin/bucket/info":          s3Complete{deepLevel: 2},

	"/admin/kms/key/create": aliasCompleter,
	"/admin/kms/key/status": aliasCompleter,
	"/admin/kms/key/list":   aliasCompleter,

	"/admin/subnet/health":   aliasCompleter,
	"/admin/subnet/register": aliasCompleter,

	"/admin/tier/add":    nil,
	"/admin/tier/edit":   nil,
	"/admin/tier/list":   nil,
	"/admin/tier/info":   nil,
	"/admin/tier/remove": nil,
	"/admin/tier/verify": nil,

	"/ilm/tier/info":   nil,
	"/ilm/tier/list":   nil,
	"/ilm/tier/add":    nil,
	"/ilm/tier/update": nil,
	"/ilm/tier/check":  nil,
	"/ilm/tier/remove": nil,

	"/admin/replicate/add":           aliasCompleter,
	"/admin/replicate/update":        aliasCompleter,
	"/admin/replicate/edit":          aliasCompleter,
	"/admin/replicate/info":          aliasCompleter,
	"/admin/replicate/status":        aliasCompleter,
	"/admin/replicate/remove":        aliasCompleter,
	"/admin/replicate/resync/start":  aliasCompleter,
	"/admin/replicate/resync/cancel": aliasCompleter,
	"/admin/replicate/resync/status": aliasCompleter,

	"/admin/cluster/bucket/export": aliasCompleter,
	"/admin/cluster/bucket/import": aliasCompleter,
	"/admin/cluster/iam/export":    aliasCompleter,
	"/admin/cluster/iam/import":    aliasCompleter,

	"/alias/set":    nil,
	"/alias/list":   aliasCompleter,
	"/alias/remove": aliasCompleter,
	"/alias/import": nil,
	"/alias/export": aliasCompleter,

	"/support/callhome":     aliasCompleter,
	"/support/register":     aliasCompleter,
	"/support/diag":         aliasCompleter,
	"/support/profile":      aliasCompleter,
	"/support/proxy/set":    aliasCompleter,
	"/support/proxy/show":   aliasCompleter,
	"/support/proxy/remove": aliasCompleter,
	"/support/inspect":      aliasCompleter,
	"/support/perf":         aliasCompleter,
	"/support/metrics":      aliasCompleter,
	"/support/status":       aliasCompleter,
	"/support/top/locks":    aliasCompleter,
	"/support/top/api":      aliasCompleter,
	"/support/top/drive":    aliasCompleter,
	"/support/top/disk":     aliasCompleter,
	"/support/top/net":      aliasCompleter,
	"/support/top/rpc":      aliasCompleter,
	"/support/upload":       aliasCompleter,

	"/license/register": aliasCompleter,
	"/license/info":     aliasCompleter,
	"/license/update":   aliasCompleter,

	"/update":         nil,
	"/ready":          aliasCompleter,
	"/ping":           aliasCompleter,
	"/od":             nil,
	"/batch/generate": aliasCompleter,
	"/batch/start":    aliasCompleter,
	"/batch/list":     aliasCompleter,
	"/batch/status":   aliasCompleter,
	"/batch/describe": aliasCompleter,
	"/batch/cancel":   aliasCompleter,

	"/quota/set":   aliasCompleter,
	"/quota/info":  aliasCompleter,
	"/quota/clear": aliasCompleter,
	"/put":         complete.PredictOr(s3Completer, fsCompleter),
	"/get":         complete.PredictOr(s3Completer, fsCompleter),

	"/cors/set":    s3Complete{deepLevel: 2},
	"/cors/get":    s3Complete{deepLevel: 2},
	"/cors/remove": s3Complete{deepLevel: 2},
}

// flagsToCompleteFlags transforms a cli.Flag to complete.Flags
// understood by posener/complete library.
func flagsToCompleteFlags(flags []cli.Flag) complete.Flags {
	complFlags := make(complete.Flags)
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
		if subCmd.Hidden {
			continue
		}
		complCmd.Sub[subCmd.Name] = cmdToCompleteCmd(subCmd, parentPath+"/"+cmd.Name)
		for _, alias := range subCmd.Aliases {
			complCmd.Sub[alias] = cmdToCompleteCmd(subCmd, parentPath+"/"+cmd.Name)
		}
	}

	complCmd.Flags = flagsToCompleteFlags(cmd.Flags)
	complCmd.Args = completeCmds[parentPath+"/"+cmd.Name]
	return complCmd
}

// Main function to answer to bash completion calls
func mainComplete() error {
	// Recursively register all commands and subcommands
	// along with global and local flags
	complCmds := make(complete.Commands)
	for _, cmd := range appCmds {
		if cmd.Hidden {
			continue
		}
		complCmds[cmd.Name] = cmdToCompleteCmd(cmd, "")
		for _, alias := range cmd.Aliases {
			complCmds[alias] = cmdToCompleteCmd(cmd, "")
		}
	}
	complFlags := flagsToCompleteFlags(globalFlags)
	mcComplete := complete.Command{
		Sub:         complCmds,
		GlobalFlags: complFlags,
	}
	// Answer to bash completion call
	complete.New(filepath.Base(os.Args[0]), mcComplete).Run()
	return nil
}
