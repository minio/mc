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
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

var configHostListCmd = cli.Command{
	Name:            "list",
	ShortName:       "ls",
	Usage:           "Lists hosts in configuration file.",
	Action:          mainConfigHostList,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [ALIAS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all hosts.
     $ {{.HelpName}}

  2. List a specific host.
     $ {{.HelpName}} s3
`,
}

// Input argument validator..
func checkConfigHostListSyntax(ctx *cli.Context) {
	args := ctx.Args()

	if len(ctx.Args()) > 1 {
		fatalIf(errInvalidArgument().Trace(args...),
			"Incorrect number of arguments to list hosts.")
	}

	if args.Get(0) != "" {
		if !isValidAlias(args.Get(0)) {
			fatalIf(errDummy().Trace(args.Get(0)),
				"Invalid alias `"+args.Get(0)+"`.")
		}
	}
}

func mainConfigHostList(ctx *cli.Context) error {
	checkConfigHostListSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("HostMessage", color.New(color.FgGreen))
	console.SetColor("Alias", color.New(color.FgCyan, color.Bold))
	console.SetColor("URL", color.New(color.FgCyan))
	console.SetColor("AccessKey", color.New(color.FgBlue))
	console.SetColor("SecretKey", color.New(color.FgBlue))
	console.SetColor("API", color.New(color.FgYellow))
	console.SetColor("Lookup", color.New(color.FgCyan))

	args := ctx.Args()
	listHosts(args.Get(0)) // List all configured hosts.
	return nil
}

// Prints all the hosts.
func printHosts(hosts ...hostMessage) {
	var maxAlias = 0
	for _, host := range hosts {
		if len(host.Alias) > maxAlias {
			maxAlias = len(host.Alias)
		}
	}
	for _, host := range hosts {
		if !globalJSON {
			// Format properly for alignment based on alias length only in non json mode.
			host.Alias = fmt.Sprintf("%-*.*s:", maxAlias, maxAlias, host.Alias)
		}
		if host.AccessKey == "" || host.SecretKey == "" {
			host.AccessKey = ""
			host.SecretKey = ""
			host.API = ""
		}
		printMsg(host)
	}
}

// byAlias is a collection satisfying sort.Interface
type byAlias []hostMessage

func (d byAlias) Len() int           { return len(d) }
func (d byAlias) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d byAlias) Less(i, j int) bool { return d[i].Alias < d[j].Alias }

// listHosts - list all host URLs or a requested host.
func listHosts(alias string) {
	conf, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version `"+globalMCConfigVersion+"`.")

	// If specific alias is requested, look for it and print.
	if alias != "" {
		if v, ok := conf.Hosts[alias]; ok {
			printHosts(hostMessage{
				op:          "list",
				prettyPrint: false,
				Alias:       alias,
				URL:         v.URL,
				AccessKey:   v.AccessKey,
				SecretKey:   v.SecretKey,
				API:         v.API,
				Lookup:      v.Lookup,
			})
			return
		}
		fatalIf(errInvalidAliasedURL(alias), "No such alias `"+alias+"` found.")
	}

	var hosts []hostMessage
	for k, v := range conf.Hosts {
		hosts = append(hosts, hostMessage{
			op:          "list",
			prettyPrint: true,
			Alias:       k,
			URL:         v.URL,
			AccessKey:   v.AccessKey,
			SecretKey:   v.SecretKey,
			API:         v.API,
			Lookup:      v.Lookup,
		})
	}

	// Sort hosts by alias names lexically.
	sort.Sort(byAlias(hosts))

	// Display all the hosts.
	printHosts(hosts...)
}
