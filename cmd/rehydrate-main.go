// Copyright (c) 2022 MinIO, Inc.
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

import "github.com/minio/cli"

var rehydrateCmd = cli.Command{
	Name:         "rehydrate",
	Usage:        "Temporarily restore tiered objects",
	Action:       mainILMRestore,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(ilmRestoreFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Restore a copy of one or more objects from its remote tier. This copy automatically expires
  after the specified number of days (Default 1 day).

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Restore one specific object
     {{.Prompt}} {{.HelpName}} myminio/mybucket/path/to/object

  2. Restore a specific object version
     {{.Prompt}} {{.HelpName}} --vid "CL3sWgdSN2pNntSf6UnZAuh2kcu8E8si" myminio/mybucket/path/to/object

  3. Restore all objects under a specific prefix
     {{.Prompt}} {{.HelpName}} --recursive myminio/mybucket/dir/

  4. Restore all objects with all versions under a specific prefix
     {{.Prompt}} {{.HelpName}} --recursive --versions myminio/mybucket/dir/
`,
}
