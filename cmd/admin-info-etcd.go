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
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

const (
	etcd         = "BoldGreen"
	etcdDegraded = "BoldYellow"
	etcdFail     = "BoldRed"
)

var adminInfoEtcd = cli.Command{
	Name:   "etcd",
	Usage:  "display MinIO server etcd information",
	Action: mainAdminETCDInfo,
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
	1. Get server CPU information of the 'play' MinIO server.
		 $ {{.HelpName}} play/
  
  `,
}

// serverMonitorMessage holds service status info along with
// cpu, mem, net and disk monitoristics
type serverEtcdMessage struct {
	Monitor string           `json:"status"`
	Service string           `json:"service"`
	Addr    string           `json:"address"`
	Err     string           `json:"error"`
	ETCD    *madmin.ETCDInfo `json:"etcd,omitempty"`
}

func (s serverEtcdMessage) JSON() string {
	s.Monitor = "success"
	if s.Err != "" {
		s.Monitor = "fail"
	}
	statusJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON")

	return string(statusJSONBytes)
}

func (s serverEtcdMessage) String() string {
	msg := ""
	dot := "●"
	msg += fmt.Sprintf("%s %s        \n", console.Colorize(etcd, dot), console.Colorize(etcd, " ETCD Info"))
	msg += fmt.Sprintf("   EndPoint          : %s\n", s.ETCD.ETCDEndpoint)
	msg += fmt.Sprintf("   Domain            : %s\n", s.ETCD.Domain)
	msg += fmt.Sprintf("   Public IPs        : %s\n", s.ETCD.PublicIPs)

	if s.Err != "" || s.Service == "off" {
		msg += fmt.Sprintf("   ETCD Status       : %s\n", console.Colorize(etcdFail, "offline"))
		msg += fmt.Sprintf("   Error             : %s\n", console.Colorize(etcdFail, s.Err))
		return msg
	}
	msg += fmt.Sprintf("   ETCD Status       : %s\n", console.Colorize(etcd, "online"))
	return msg
}
func mainAdminETCDInfo(ctx *cli.Context) error {
	checkAdminEtcdInfoSyntax(ctx)

	// set the console colors
	console.SetColor(etcd, color.New(color.FgGreen, color.Bold))
	console.SetColor(etcdDegraded, color.New(color.FgYellow, color.Bold))
	console.SetColor(etcdFail, color.New(color.FgRed, color.Bold))

	// Get the alias
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO admin client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	printOfflineErrorMessage := func(err error) {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		printMsg(serverEtcdMessage{
			Addr:    aliasedURL,
			Service: "off",
			Err:     errMsg,
		})
	}

	processErr := func(e error) error {
		switch e.(type) {
		case *json.SyntaxError:
			printOfflineErrorMessage(e)
			return e
		case *url.Error:
			printOfflineErrorMessage(e)
			return e
		default:
			// If the error is not nil and unrecognized, just print it and exit
			fatalIf(probe.NewError(e), "Cannot get service status.")
		}
		return nil
	}

	etcdInfo, e := client.ServerETCDInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	if etcdInfo.Error != "" {
		printMsg(serverEtcdMessage{
			Service: "off",
			Addr:    aliasedURL,
			Err:     etcdInfo.Error,
			ETCD:    &etcdInfo,
		})
	} else {
		printMsg(serverEtcdMessage{
			Service: "on",
			Addr:    aliasedURL,
			ETCD:    &etcdInfo,
		})
	}
	return nil
}

// checkAdminMonitorSyntax - validate all the passed arguments
func checkAdminEtcdInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {

		exit := globalErrorExitStatus
		cli.ShowCommandHelpAndExit(ctx, "etcd", exit)
	}
}
