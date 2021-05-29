// Copyright (c) 2015-2021 MinIO, Inc.
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
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
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

var aliasSubcommands = []cli.Command{
	aliasSetCmd,
	aliasListCmd,
	aliasRemoveCmd,
}

var aliasCmd = cli.Command{
	Name:            "alias",
	Usage:           "set, remove and list aliases in configuration file",
	Action:          mainAlias,
	Before:          setGlobalsFromContext,
	HideHelpCommand: true,
	Flags:           append(aliasFlags, globalFlags...),
	Subcommands:     aliasSubcommands,
}

// mainAlias is the handle for "mc alias" command. provides sub-commands which write configuration data in json format to config file.
func mainAlias(ctx *cli.Context) error {
	commandNotFound(ctx, aliasSubcommands)
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
