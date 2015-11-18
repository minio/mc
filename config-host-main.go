package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
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
      $ mc config {{.Name}} add s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
      $ set -o history

   2. Add host configuration for a URL, using s3 api v2. For security reasons turn off bash history
      $ set +o history
      $ mc config {{.Name}} add storage.googleapis.com BKIKJAA5BMMU2RHO6IBB V7f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 S3v2
      $ set -o history

   3. List all hosts.
      $ mc config {{.Name}} list

   4. Remove host config.
      $ mc config {{.Name}} remove s3.amazonaws.com

`,
}

// hostMessage container for content message structure
type hostMessage struct {
	op              string
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
	jsonMessageBytes, e := json.Marshal(a)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
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
		fatalIf(errDummy().Trace(), "Incorrect number of arguments to host command")
	}
	switch strings.TrimSpace(ctx.Args().First()) {
	case "add":
		if len(ctx.Args().Tail()) < 3 || len(ctx.Args().Tail()) > 4 {
			fatalIf(errInvalidArgument().Trace(), "Incorrect number of arguments for add host command.")
		}
	case "remove":
		if len(ctx.Args().Tail()) != 1 {
			fatalIf(errInvalidArgument().Trace(), "Incorrect number of arguments for remove host command.")
		}
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

	switch strings.TrimSpace(arg) {
	case "add":
		addHost(tailArgs.Get(0), tailArgs.Get(1), tailArgs.Get(2), tailArgs.Get(3))
	case "remove":
		removeHost(tailArgs.Get(0))
	case "list":
		listHosts()
	}
}

func listHosts() {
	conf := new(configV6)
	conf.Version = globalMCConfigVersion
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]hostConfig)
	conf.Aliases = make(map[string]string)
	config, err := quick.New(conf)
	fatalIf(err.Trace(), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV6)
	for k, v := range newConf.Hosts {
		printMsg(hostMessage{op: "list", Host: k, AccessKeyID: v.AccessKeyID, SecretAccessKey: v.SecretAccessKey, API: v.API})
	}
}

func removeHost(hostGlob string) {
	if strings.TrimSpace(hostGlob) == "" {
		fatalIf(errDummy().Trace(), "Alias or URL cannot be empty.")
	}
	if strings.TrimSpace(hostGlob) == "dl.minio.io:9000" {
		fatalIf(errDummy().Trace(), "‘"+hostGlob+"’ is reserved hostname and cannot be removed.")
	}
	config, err := newConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV6)
	if _, ok := newConf.Hosts[hostGlob]; !ok {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Host glob ‘%s’ does not exist.", hostGlob))
	}
	delete(newConf.Hosts, hostGlob)

	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")
	err = writeConfig(newConfig)
	fatalIf(err.Trace(hostGlob), "Unable to save host glob ‘"+hostGlob+"’.")

	printMsg(hostMessage{op: "remove", Host: hostGlob})
}

// isValidSecretKey - validate secret key
func isValidSecretKey(secretAccessKey string) bool {
	if secretAccessKey == "" {
		return true
	}
	regex := regexp.MustCompile("^.{40}$")
	return regex.MatchString(secretAccessKey)
}

// isValidAccessKey - validate access key
func isValidAccessKey(accessKeyID string) bool {
	if accessKeyID == "" {
		return true
	}
	regex := regexp.MustCompile("^[A-Z0-9\\-\\.\\_\\~]{20}$")
	return regex.MatchString(accessKeyID)
}

// addHost - add new host
func addHost(newHost, accessKeyID, secretAccessKey, api string) {
	if strings.TrimSpace(newHost) == "" {
		fatalIf(errDummy().Trace(), "Unable to proceed, empty arguments provided.")
	}
	if len(accessKeyID) != 0 {
		if !isValidAccessKey(accessKeyID) {
			fatalIf(errInvalidArgument().Trace(), "Invalid access key id provided.")
		}
	}
	if len(secretAccessKey) != 0 {
		if !isValidSecretKey(secretAccessKey) {
			fatalIf(errInvalidArgument().Trace(), "Invalid secret access key provided.")
		}
	}
	if strings.TrimSpace(api) == "" {
		api = "S3v4"
	}
	if strings.TrimSpace(api) != "S3v2" && strings.TrimSpace(api) != "S3v4" {
		fatalIf(errInvalidArgument().Trace(), "Unrecognized version name provided, supported inputs are ‘S3v4’, ‘S3v2’")
	}
	config, err := newConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")

	newConf := config.Data().(*configV6)
	for savedHost := range newConf.Hosts {
		if savedHost == newHost {
			newConf.Hosts[savedHost] = hostConfig{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				API:             api,
			}
		} else {
			newConf.Hosts[newHost] = hostConfig{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				API:             api,
			}
		}
	}

	newConfig, err := quick.New(newConf)
	fatalIf(err.Trace(globalMCConfigVersion), "Failed to initialize ‘quick’ configuration data structure.")

	err = writeConfig(newConfig)
	fatalIf(err.Trace(newHost), "Unable to save new host ‘"+newHost+"’.")

	printMsg(hostMessage{
		op:              "add",
		Host:            newHost,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		API:             api,
	})
}
