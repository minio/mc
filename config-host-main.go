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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

var (
	configHostFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of config host",
	}
)

var configHostCmd = cli.Command{
	Name:   "host",
	Usage:  "List, modify and remove hosts in configuration file.",
	Flags:  []cli.Flag{configHostFlagHelp},
	Action: mainConfigHost,
	CustomHelpTemplate: `NAME:
   mc config {{.Name}} - {{.Usage}}

USAGE:
   mc config {{.Name}} OPERATION [ARGS...]

OPERATION:
   remove   Remove a host.
   list     list all hosts.
   add      Add new host.

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
   1. Add host configuration for a URL, using default signature V4. For security reasons turn off bash history
      $ set +o history
      $ mc config {{.Name}} add https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
      $ set -o history

   2. Add host configuration for a URL, using signature V2. For security reasons turn off bash history
      $ set +o history
      $ mc config {{.Name}} add https://storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v2
      $ set -o history

   3. List all hosts.
      $ mc config {{.Name}} list

   4. Remove host config.
      $ mc config {{.Name}} remove https://s3.amazonaws.com

`,
}

// hostMessage container for content message structure
type hostMessage struct {
	op              string
	Status          string `json:"status"`
	Host            string `json:"host"`
	AccessKeyID     string `json:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	API             string `json:"api,omitempty"`
}

// String colorized host message
func (a hostMessage) String() string {
	if a.op == "list" {
		message := console.Colorize("Host", fmt.Sprintf("[%s] ", a.Host))
		if a.AccessKeyID != "" || a.SecretAccessKey != "" {
			message += console.Colorize("AccessKeyID", fmt.Sprintf("<- %s,", a.AccessKeyID))
			message += console.Colorize("SecretAccessKey", fmt.Sprintf(" %s,", a.SecretAccessKey))
			message += console.Colorize("API", fmt.Sprintf(" %s", a.API))
		}
		return message
	}
	if a.op == "remove" {
		return console.Colorize("HostMessage", "Removed host ‘"+a.Host+"’ successfully.")
	}
	if a.op == "add" {
		return console.Colorize("HostMessage", "Added host ‘"+a.Host+"’ successfully.")
	}
	// should never reach here
	return ""
}

// JSON jsonified host message
func (a hostMessage) JSON() string {
	a.Status = "success"
	jsonMessageBytes, e := json.Marshal(a)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// checkConfigHostImportSyntax - check config host import syntax.
func checkConfigHostImportSyntax(ctx *cli.Context) {
	tailArgs := ctx.Args().Tail()
	if len(ctx.Args().Tail()) < 2 || len(ctx.Args().Tail()) > 3 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for import host command.")
	}

	newHostURL := tailArgs.Get(0)
	credentialsFile := tailArgs.Get(1)
	api := tailArgs.Get(2)

	if !isValidHostURL(newHostURL) {
		fatalIf(errDummy().Trace(newHostURL),
			"Invalid host ‘"+newHostURL+"’. Valid options are [http://example.test.io, https://bucket.s3.amazonaws.com].")
	}
	if strings.TrimSpace(credentialsFile) == "" {
		fatalIf(errDummy().Trace(), "Credentials file cannot be empty.")
	}
	if strings.TrimSpace(api) == "" {
		api = "S3v4"
	}
	if strings.TrimSpace(api) != "S3v2" && strings.TrimSpace(api) != "S3v4" {
		fatalIf(errInvalidArgument().Trace(api),
			"Unrecognized api version. Valid options are ‘[ S3v4, S3v2 ]’.")
	}
}

// checkConfigHostAddSyntax - check config host add syntax.
func checkConfigHostAddSyntax(ctx *cli.Context) {
	tailArgs := ctx.Args().Tail()
	if len(ctx.Args().Tail()) < 3 || len(ctx.Args().Tail()) > 4 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for host add command.")
	}
	newHostURL := tailArgs.Get(0)
	accessKeyID := tailArgs.Get(1)
	secretAccessKey := tailArgs.Get(2)
	api := tailArgs.Get(3)
	if !isValidHostURL(newHostURL) {
		fatalIf(errDummy().Trace(newHostURL),
			"Invalid host URL: ‘"+newHostURL+"’ provided. Valid options are [http://example.test.io, https://bucket.s3.amazonaws.com].")
	}
	if len(accessKeyID) != 0 {
		if !isValidAccessKey(accessKeyID) {
			fatalIf(errInvalidArgument().Trace(accessKeyID), "Invalid access key id provided.")
		}
	}
	if len(secretAccessKey) != 0 {
		if !isValidSecretKey(secretAccessKey) {
			fatalIf(errInvalidArgument().Trace(secretAccessKey), "Invalid secret access key provided.")
		}
	}
	if strings.TrimSpace(api) == "" {
		api = "S3v4"
	}
	if strings.TrimSpace(api) != "S3v2" && strings.TrimSpace(api) != "S3v4" {
		fatalIf(errInvalidArgument().Trace(api),
			"Unrecognized api version. Valid options are ‘[ S3v4, S3v2 ]’.")
	}
}

// checkConfigHostRemoveSyntax - check config host remove syntax.
func checkConfigHostRemoveSyntax(ctx *cli.Context) {
	tailArgs := ctx.Args().Tail()
	if len(ctx.Args().Tail()) != 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for remove host command.")
	}
	if !isValidHostURL(tailArgs.Get(0)) {
		fatalIf(errDummy().Trace(tailArgs.Get(0)),
			"Invalid host ‘"+tailArgs.Get(0)+"’. Valid options are [http://example.test.io, https://bucket.s3.amazonaws.com].")
	}
	if strings.TrimSpace(tailArgs.Get(0)) == "https://dl.minio.io:9000" {
		fatalIf(errDummy().Trace(tailArgs.Get(0)),
			"‘"+tailArgs.Get(0)+"’ is reserved hostname and cannot be removed.")
	}
}

func checkConfigHostSyntax(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "host", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "host", 1) // last argument is exit code
	}
	if len(ctx.Args().Tail()) > 4 {
		fatalIf(errDummy().Trace(ctx.Args().Tail()...), "Incorrect number of arguments to host command")
	}
	switch strings.TrimSpace(ctx.Args().First()) {
	case "add":
		checkConfigHostAddSyntax(ctx)
	case "import":
		checkConfigHostImportSyntax(ctx)
	case "remove":
		checkConfigHostRemoveSyntax(ctx)
	case "list":
	default:
		cli.ShowCommandHelpAndExit(ctx, "host", 1) // last argument is exit code
	}
}

func mainConfigHost(ctx *cli.Context) {
	checkConfigHostSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Host", color.New(color.FgCyan, color.Bold))
	console.SetColor("API", color.New(color.FgYellow, color.Bold))
	console.SetColor("HostMessage", color.New(color.FgGreen, color.Bold))
	console.SetColor("AccessKeyID", color.New(color.FgBlue, color.Bold))
	console.SetColor("SecretAccessKey", color.New(color.FgRed, color.Bold))

	arg := ctx.Args().First()
	tailArgs := ctx.Args().Tail()

	// Switch case through commands.
	switch strings.TrimSpace(arg) {
	// Add a host with specified credentials.
	case "add":
		hostURL := tailArgs.Get(0)
		accessKeyID := tailArgs.Get(1)
		secretAccessKey := tailArgs.Get(2)
		api := tailArgs.Get(3)
		if strings.TrimSpace(api) == "" {
			api = "S3v4"
		}
		hostCfg := hostConfig{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			API:             api,
		}
		addHost(hostURL, hostCfg)
	// Import credentials through a CSV file for a host.
	case "import":
		hostURL := tailArgs.Get(0)
		credentials := tailArgs.Get(1)
		api := tailArgs.Get(2)
		if strings.TrimSpace(api) == "" {
			api = "S3v4"
		}
		creds := getCredentials(credentials)
		var hostCfg hostConfig
		hostCfg = hostConfig{
			AccessKeyID:     creds[0].AccessKeyID,
			SecretAccessKey: creds[0].SecretAccessKey,
			API:             api,
		}
		addHost(hostURL, hostCfg)
	// Remove a host.
	case "remove":
		hostURL := tailArgs.Get(0)
		removeHost(hostURL)
	// List all configured hosts.
	case "list":
		listHosts()
	}
}

// addHost - add a host config.
func addHost(hostURL string, hostCfg hostConfig) {
	conf, err := loadConfigV6()
	fatalIf(err.Trace(), "Unable to load config version ‘6’.")

	// Add new host.
	conf.Hosts[hostURL] = hostCfg

	err = saveConfigV6(conf)
	fatalIf(err.Trace(hostURL), "Unable to update hosts in config version ‘6’.")

	printMsg(hostMessage{
		op:              "add",
		Host:            hostURL,
		AccessKeyID:     hostCfg.AccessKeyID,
		SecretAccessKey: hostCfg.SecretAccessKey,
		API:             hostCfg.API,
	})
}

// removeHost - removes a host.
func removeHost(hostURL string) {
	conf, err := loadConfigV6()
	fatalIf(err.Trace(), "Unable to load config version ‘6’.")

	// Remove host.
	delete(conf.Hosts, hostURL)

	err = saveConfigV6(conf)
	fatalIf(err.Trace(hostURL), "Unable to save deleted hosts in config version ‘6’.")

	printMsg(hostMessage{op: "remove", Host: hostURL})
}

// listHosts - list all host URLs.
func listHosts() {
	conf, err := loadConfigV6()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config version ‘6’")

	for k, v := range conf.Hosts {
		printMsg(hostMessage{
			op:              "list",
			Host:            k,
			AccessKeyID:     v.AccessKeyID,
			SecretAccessKey: v.SecretAccessKey,
			API:             v.API,
		})
	}
}

// credential - struct to read from a CSV file.
type credential struct {
	Name            string
	AccessKeyID     string
	SecretAccessKey string
}

// getCredentials - get credentials file.
func getCredentials(credentialsFile string) []credential {
	reader, err := os.Open(credentialsFile)
	if err != nil {
		fatalIf(probe.NewError(err), "Unable to open credentials file "+credentialsFile+".")
	}
	creds := []credential{}
	// skip the first line of the csv file, usually csv header.
	skipLine := 1
	if err := newCSVReader(reader).Unmarshal(&creds, skipLine); err != nil {
		fatalIf(err.Trace(credentialsFile), "Unable to read credentials file "+credentialsFile)
	}
	if len(creds) == 0 {
		fatalIf(errDummy().Trace(credentialsFile),
			"Invalid credentials file provided, need to have alteast single field.")
	}
	return creds
}
