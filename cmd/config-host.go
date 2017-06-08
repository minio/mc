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
	"encoding/json"
	"fmt"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

var configHostCmd = cli.Command{
	Name:   "host",
	Usage:  "List, modify and remove hosts in configuration file.",
	Action: mainConfigHost,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		configHostAddCmd,
		configHostRemoveCmd,
		configHostListCmd,
	},
	HideHelpCommand: true,
}

// mainConfigHost is the handle for "mc config host" command.
func mainConfigHost(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "remove", "list" have their own main.
}

// hostMessage container for content message structure
type hostMessage struct {
	op         string
	Status     string `json:"status"`
	Alias      string `json:"alias"`
	URL        string `json:"URL"`
	AccessKey  string `json:"accessKey,omitempty"`
	SecretKey  string `json:"secretKey,omitempty"`
	API        string `json:"api,omitempty"`
	Encryption string `json:"encryption,omitempty"`
}

// String colorized host message
func (h hostMessage) String() string {
	switch h.op {
	case "list":
		message := console.Colorize("Alias", fmt.Sprintf("%s: ", h.Alias))
		message += console.Colorize("URL", fmt.Sprintf("%-30.30s", h.URL))
		if h.AccessKey != "" || h.SecretKey != "" {
			message += console.Colorize("AccessKey", fmt.Sprintf("  %-20.20s", h.AccessKey))
			message += console.Colorize("SecretKey", fmt.Sprintf("  %-40.40s", h.SecretKey))
			message += console.Colorize("API", fmt.Sprintf("  %.20s", h.API))
		}
		if h.Encryption != "" {
			message += console.Colorize("Encryption", fmt.Sprintf("  %.20s", h.Encryption))
		}
		return message
	case "remove":
		return console.Colorize("HostMessage", "Removed `"+h.Alias+"` successfully.")
	case "add":
		return console.Colorize("HostMessage", "Added `"+h.Alias+"` successfully.")
	default:
		return ""
	}
}

// JSON jsonified host message
func (h hostMessage) JSON() string {
	h.Status = "success"
	jsonMessageBytes, e := json.Marshal(h)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}
