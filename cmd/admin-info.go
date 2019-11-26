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
	"fmt"
	"strconv"
	"time"

	// "net/url"

	humanize "github.com/dustin/go-humanize"
	"github.com/dustin/go-humanize/english"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "display MinIO server information",
	Action: mainAdminInfo,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server information of the 'play' MinIO server.
     {{.Prompt}} {{.HelpName}} play/
`,
}

// InfoMessage container to hold server admin related information.
type infoMessage madmin.InfoMessage

// String colorized service status message.
func (u infoMessage) String() (msg string) {
	msg += "\n"
	dot := "●"

	// Iterate through servers and collect a short info for each.
	var totalOnlineDisksCluster int
	var totalOfflineDisksCluster int
	for _, srv := range u.Servers {
		// When MinIO server is offline ("Mode" field)
		if u.Mode == "offline" {
			msg += fmt.Sprintf("%s  %s\n", console.Colorize("InfoFail", dot), console.Colorize("PrintB", srv.Addr))
			msg += fmt.Sprintf("   Uptime: %s\n", console.Colorize("InfoFail", "offline"))
			return
		}

		// Print server title
		msg += fmt.Sprintf("%s  %s\n", console.Colorize("Info", dot), console.Colorize("PrintB", srv.Addr))

		// Uptime (this is per server)
		msg += fmt.Sprintf("   Uptime: %s\n", console.Colorize("Info",
			humanize.RelTime(time.Now(), time.Now().Add(srv.Uptime), "", "")))

		// Version (this is per server)
		version := srv.Version
		if srv.Version == "DEVELOPMENT.GOGET" {
			version = "<development>"
		}
		msg += fmt.Sprintf("   Version: %s\n", version)

		// Network info.
		// How do we make sure network information is good/ok to move on.
		// if srv.Network != "" {
		// 	msg += fmt.Sprintf("   Network: %s %s\n", srv.Network, console.Colorize("Info", "OK "))
		// }

		// Info on Drives for each server.
		// The goal is to print something like;
		// "Drives: 1/2 OK"
		var totOffline int
		var totalOnline int
		var dispNoOfDisks string
		for _, disk := range srv.Disk {
			if disk.State == "ok" {
				totalOnline++
			} else {
				totOffline++
			}
		}
		// How do we decide if Disks/Drives are "OK"?
		totalDisksPerServer := totalOnline + totOffline
		totalOnlineDisksCluster += totalOnline
		totalOfflineDisksCluster += totOffline

		dispNoOfDisks = strconv.Itoa(totalOnline) + "/" + strconv.Itoa(totalDisksPerServer)
		msg += fmt.Sprintf("   Drives: %s %s\n", dispNoOfDisks, console.Colorize("Info", "OK "))
		// if sqsARNs != "" {
		// 	msg += fmt.Sprintf("SQS ARNs: %s\n", sqsARNs)
		// }
		//
		// // Incoming/outgoing
		// if v, ok := srv.StorageInfo.(xlBackend); ok {
		// 	upBackends := 0
		// 	downBackends := 0
		// 	for _, set := range v.Sets {
		// 		for i, s := range set {
		// 			if len(s.Endpoint) > 0 && (strings.Contains(s.Endpoint, srv.Addr) || s.Endpoint[i] == '/' || s.Endpoint[i] == '.') {
		// 				if s.State == "ok" {
		// 					upBackends++
		// 				} else {
		// 					downBackends++
		// 				}
		// 			}
		// 		}
		// 	}
		// 	upBackendsString := fmt.Sprintf("%d", upBackends)
		// 	if downBackends != 0 {
		// 		upBackendsString = console.Colorize("InfoFail", fmt.Sprintf("%d", upBackends))
		// 	}
		// 	msg += fmt.Sprintf("Drives: %s/%d %s\n", upBackendsString,
		// 		upBackends+downBackends, console.Colorize("Info", "OK"))
		// }
		msg += "\n"
	}

	// Summary line on Used space, total number of buckets and objects
	usedTotal := humanize.IBytes(uint64(u.Usage.Size))

	msg += fmt.Sprintf("%s Used, %s, %s\n", usedTotal,
		english.Plural(u.Buckets.Count, "Bucket", ""),
		english.Plural(u.Objects.Count, "Object", ""))
	msg += fmt.Sprintf("%s online, %s offline\n",
		english.Plural(totalOnlineDisksCluster, "drive", ""),
		english.Plural(totalOfflineDisksCluster, "drive", ""))

	return
}

// JSON jsonified service status message.
func (u infoMessage) JSON() string {
	// srv.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminInfoSyntax - validate all the passed arguments
func checkAdminServerInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}
}

func mainAdminInfo(ctx *cli.Context) error {
	checkAdminServerInfoSyntax(ctx)

	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoDegraded", color.New(color.FgYellow, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	// args := ctx.Args()
	// aliasedURL := args.Get(0)
	//
	// Create a new MinIO Admin Client
	// client, err := newAdminClient(aliasedURL)
	// fatalIf(err, "Unable to initialize admin connection.")
	//
	// printOfflineErrorMessage := func(err error) {
	// 	errMsg := ""
	// 	if err != nil {
	// 		errMsg = err.Error()
	// 	}
	// 	printMsg(infoMessage{
	// 		Addr:    aliasedURL,
	// 		Service: "off",
	// 		Err:     errMsg,
	// 	})
	// }
	//
	// processErr := func(e error) error {
	// 	switch e.(type) {
	// 	case *json.SyntaxError:
	// 		println("Error:", e)
	// 		return e
	// 	case *url.Error:
	// 		println("Error:", e)
	// 		return e
	// 	default:
	// 		// If the error is not nil and unrecognized, just print it and exit
	// 		fatalIf(probe.NewError(e), "Cannot get service status.")
	// 	}
	// 	return nil
	// }
	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Fetch info of all servers (cluster or single server)
	admInfo, _ := client.ServerAdminInfo()

	// admInfo := madmin.InfoMessage{
	// 	Mode:         "online",
	// 	Domain:       []string{"minio"},
	// 	Region:       "us-east-1",
	// 	SQSARN:       []string{},
	// 	DeploymentID: "6faeded5-5cf3-4133-8a37-07c5d500207c",
	// 	Buckets: struct {
	// 		Count int "json:\"count\""
	// 	}{
	// 		Count: 20,
	// 	},
	// 	Objects: struct {
	// 		Count int "json:\"count\""
	// 	}{
	// 		Count: 123450,
	// 	},
	// 	Usage: struct {
	// 		Size uint64 "json:\"size\""
	// 	}{
	// 		Size: 257893451230,
	// 	},
	// 	Services: struct {
	// 		Vault  madmin.Vault         "json:\"vault\""
	// 		LDAP   madmin.LDAP          "json:\"ldap\""
	// 		Logger []madmin.Logger      "json:\"logger\""
	// 		Audit  []madmin.Audit       "json:\"audit\""
	// 		Notif  madmin.Notifications "json:\"notifications\""
	// 	}{
	// 		Vault: madmin.Vault{
	// 			Status:  "online",
	// 			Encrypt: "",
	// 			Decrypt: "",
	// 			Update:  "",
	// 		},
	// 		LDAP: madmin.LDAP{
	// 			Status: "offline",
	// 		},
	// 		Logger: []madmin.Logger{},
	// 		// {
	// 		// 	"logger1: struct{ Status string }{
	// 		// 		Status: "online",
	// 		// 	},
	// 		// },
	// 		// {
	// 		// 	"logger2: struct{ Status string }{
	// 		// 		Status: "offline",
	// 		// 	},
	// 		// }},
	// 		Audit: []madmin.Audit{},
	// 		// {
	// 		// 	"1: struct{ Status string }{
	// 		// 		Status: "online",
	// 		// 	},
	// 		// },
	// 		// {
	// 		// 	"2: struct{ Status string }{
	// 		// 		Status: "offline",
	// 		// 	},
	// 		// }},
	// 		Notif: madmin.Notifications{},
	// 		// {
	// 		// 	"amqp: {
	// 		// 		{
	// 		// 			"amqp1: {
	// 		// 				"status: "online",
	// 		// 			},
	// 		// 		},
	// 		// 		{
	// 		// 			"amqp2: {
	// 		//				"status: "online",
	// 		// 			},
	// 		// 		}},
	// 		// },
	// 		// {
	// 		// 	"mqtt: {
	// 		// 		{
	// 		// 			"1: {
	// 		// 				"status: "online",
	// 		// 			},
	// 		// 		},
	// 		// 		{
	// 		// 			"2: {
	// 		// 				"status: "online",
	// 		// 			},
	// 		// 		}},
	// 		// }},
	// 	},
	// 	Backend: madmin.BackendInfo{
	// 		Type:             2,
	// 		OnlineDisks:      madmin.BackendDisks{},
	// 		OfflineDisks:     madmin.BackendDisks{},
	// 		StandardSCData:   2,
	// 		StandardSCParity: 2,
	// 		RRSCData:         2,
	// 		RRSCParity:       2,
	// 	},
	// 	Servers: []madmin.ServerProp{
	// 		madmin.ServerProp{
	// 			// State:    "ok",
	// 			Addr:     "127.0.0.1:9000",
	// 			Uptime:   7568764866791,
	// 			Version:  "2019-09-17T18:03:33Z",
	// 			CommitID: "368fb3d8f6ee78650d474408564fa82dcf405941",
	// 			Network:  "4/4",
	// 			Disk: []madmin.Disk{
	// 				{
	// 					DrivePath:       "/tmp/data1",
	// 					State:           "online",
	// 					Model:           "seagate ...",
	// 					TotalSpace:      10000000000,
	// 					UsedSpace:       50000000000,
	// 					UUID:            "37a38d0d-21bc-4902-8119-6dac9eb1b772",
	// 					ReadThroughput:  400,
	// 					WriteThroughPut: 500,
	// 					ReadLatency:     0.15,
	// 					WriteLatency:    0.16,
	// 					Utilization:     12.34,
	// 				},
	// 				{
	// 					DrivePath: "/tmp/data2",
	// 					State:     "offline",
	// 					UUID:      "47a38d0d-21bc-4902-8119-6dac9eb1b772",
	// 				}},
	// 		},
	// 		{
	// 			// State:    "ok",
	// 			Addr:     "127.0.0.1:9001",
	// 			Uptime:   7568764866791,
	// 			Version:  "2019-09-17T18:03:33Z",
	// 			CommitID: "368fb3d8f6ee78650d474408564fa82dcf405941",
	// 			Network:  "4/4",
	// 			Disk: []madmin.Disk{
	// 				{
	// 					DrivePath:       "/tmp/data1",
	// 					State:           "online",
	// 					Model:           "seagate ...",
	// 					TotalSpace:      10000000000,
	// 					UsedSpace:       50000000000,
	// 					UUID:            "37a38d0d-21bc-4902-8119-6dac9eb1b772",
	// 					ReadThroughput:  400,
	// 					WriteThroughPut: 500,
	// 					ReadLatency:     0.15,
	// 					WriteLatency:    0.16,
	// 					Utilization:     23.45,
	// 				},
	// 				{
	// 					DrivePath: "/tmp/data2",
	// 					State:     "offline",
	// 					UUID:      "47a38d0d-21bc-4902-8119-6dac9eb1b772",
	// 				},
	// 				{
	// 					DrivePath:       "/tmp/data3",
	// 					State:           "ok",
	// 					Model:           "seagate ...",
	// 					TotalSpace:      10000000000,
	// 					UsedSpace:       50000000000,
	// 					UUID:            "37a38d0d-21bc-4902-8119-6dac9eb1b772",
	// 					ReadThroughput:  444,
	// 					WriteThroughPut: 555,
	// 					ReadLatency:     0.20,
	// 					WriteLatency:    0.21,
	// 					Utilization:     45.6,
	// 				},
	// 			},
	// 		}},
	// printMsg(infoMsg)

	// var infoMsg madmin.InfoMessage
	infoMsg := infoMessage(admInfo)
	printMsg(infoMsg)
	// printMsg(admInfo)

	return nil
}
