/*
 * MinIO Client (C) 2014-2020 MinIO, Inc.
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
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

//   Configure an alias in MinIO Client
//
//   ----
//   NOTE: that the alias command only writes values to the config file.
//   It does not use any configuration values from the environment variables.
//
//   One needs to edit configuration file manually, this is purposefully done
//   so to avoid taking credentials over cli arguments. It is a security precaution
//   ----
//

var (
	aliasFlags = []cli.Flag{}
)

var aliasCmd = cli.Command{
	Name:            "alias",
	Usage:           "set, remove and list aliases in configuration file",
	Action:          mainAlias,
	Before:          setGlobalsFromContext,
	HideHelpCommand: true,
	Flags:           append(aliasFlags, globalFlags...),
	Subcommands: []cli.Command{
		aliasSetCmd,
		aliasListCmd,
		aliasRemoveCmd,
	},
}

// mainAlias is the handle for "mc alias" command. provides sub-commands which write configuration data in json format to config file.
func mainAlias(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like add, list and remove have their own main.
}

// aliasMessage container for content message structure
type aliasMessage struct {
	op          string
	prettyPrint bool
	Status      string `json:"status"`
	Alias       string `json:"alias"`
	URL         string `json:"URL"`
	AccessKey   string `json:"accessKey,omitempty"`
	SecretKey   string `json:"secretKey,omitempty"`
	API         string `json:"api,omitempty"`
	Path        string `json:"path,omitempty"`
	// Deprecated field, replaced by Path
	Lookup string `json:"lookup,omitempty"`
}

// Print the config information of one alias, when prettyPrint flag
// is activated, fields contents are cut and '...' will be added to
// show a pretty table of all aliases configurations
func (h aliasMessage) String() string {
	switch h.op {
	case "list":
		// Create a new pretty table with cols configuration
		t := newPrettyRecord(2,
			Row{"Alias", "Alias"},
			Row{"URL", "URL"},
			Row{"AccessKey", "AccessKey"},
			Row{"SecretKey", "SecretKey"},
			Row{"API", "API"},
			Row{"Path", "Path"},
		)
		// Handle deprecated lookup
		path := h.Path
		if path == "" {
			path = h.Lookup
		}
		return t.buildRecord(h.Alias, h.URL, h.AccessKey, h.SecretKey, h.API, path)
	case "remove":
		return console.Colorize("AliasMessage", "Removed `"+h.Alias+"` successfully.")
	case "add": // add is deprecated
		fallthrough
	case "set":
		return console.Colorize("AliasMessage", "Added `"+h.Alias+"` successfully.")
	default:
		return ""
	}
}

// JSON jsonified host message
func (h aliasMessage) JSON() string {
	h.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(h, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}
