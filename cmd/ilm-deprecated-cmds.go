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

var ilmDepCmds = []cli.Command{
	ilmDepAddCmd,
	ilmDepEditCmd,
	ilmDepLsCmd,
	ilmDepRmCmd,
	ilmDepExportCmd,
	ilmDepImportCmd,
}

var (
	ilmDepAddCmd = cli.Command{
		Name:         "add",
		Usage:        "add a lifecycle configuration rule for a bucket",
		Action:       mainILMAdd,
		Hidden:       true, // to avoid being listed in `mc ilm`
		OnUsageError: onUsageError,
		Before:       setGlobalsFromContext,
		Flags:        append(ilmAddFlags, globalFlags...),
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  Add a lifecycle configuration rule.

EXAMPLES:
  1. Add a lifecycle rule with an expiration action for all objects in mybucket.
     {{.Prompt}} {{.HelpName}} --expire-days "200" myminio/mybucket

  2. Add a lifecycle rule with a transition and a noncurrent version transition action for objects with prefix doc/ in mybucket.
     Tiers must exist in MinIO. Use existing tiers or add new tiers.
     {{.Prompt}} mc tier add minio myminio MINIOTIER-1 --endpoint https://warm-minio-1.com \
         --access-key ACCESSKEY --secret-key SECRETKEY --bucket bucket1 --prefix prefix1

     {{.Prompt}} mc tier add minio myminio MINIOTIER-2 --endpoint https://warm-minio-2.com \
         --access-key ACCESSKEY --secret-key SECRETKEY --bucket bucket2 --prefix prefix2

     {{.Prompt}} {{.HelpName}} --prefix "doc/" --transition-days "90" --transition-tier "MINIOTIER-1" \
          --noncurrent-transition-days "45" --noncurrent-transition-tier "MINIOTIER-2" \
          myminio/mybucket/

  3. Add a lifecycle rule with an expiration and a noncurrent version expiration action for all objects with prefix doc/ in mybucket.
     {{.Prompt}} {{.HelpName}} --prefix "doc/" --expire-days "300" --noncurrent-expire-days "100" \
          myminio/mybucket/
`,
	}
	ilmDepRmCmd = cli.Command{
		Name:         "rm",
		Usage:        "remove (if any) existing lifecycle configuration rule",
		Action:       mainILMRemove,
		Hidden:       true, // to avoid being listed in `mc ilm`
		OnUsageError: onUsageError,
		Before:       setGlobalsFromContext,
		Flags:        append(ilmRemoveFlags, globalFlags...),
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  Remove a lifecycle configuration rule for the bucket by ID, optionally you can remove
  all the lifecycle rules on a bucket with '--all --force' option.

EXAMPLES:
  1. Remove the lifecycle management configuration rule given by ID "bgrt1ghju" for mybucket on alias 'myminio'. ID is case sensitive.
     {{.Prompt}} {{.HelpName}} --id "bgrt1ghju" myminio/mybucket

  2. Remove ALL the lifecycle management configuration rules for mybucket on alias 'myminio'.
     Because the result is complete removal, the use of --force flag is enforced.
     {{.Prompt}} {{.HelpName}} --all --force myminio/mybucket
`,
	}

	ilmDepEditCmd = cli.Command{
		Name:         "edit",
		Usage:        "modify a lifecycle configuration rule with given id",
		Action:       mainILMEdit,
		Hidden:       true, // to avoid being listed in `mc ilm`
		OnUsageError: onUsageError,
		Before:       setGlobalsFromContext,
		Flags:        append(ilmEditFlags, globalFlags...),
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  Modify a lifecycle configuration rule with given id.

EXAMPLES:
  1. Modify the expiration date for an existing rule with id "rHTY.a123".
     {{.Prompt}} {{.HelpName}} --id "rHTY.a123" --expiry-date "2020-09-17" s3/mybucket

  2. Modify the expiration and transition days for an existing rule with id "hGHKijqpo123".
     {{.Prompt}} {{.HelpName}} --id "hGHKijqpo123" --expiry-days "300" \
          --transition-days "200" --storage-class "GLACIER" s3/mybucket

  3. Disable the rule with id "rHTY.a123".
     {{.Prompt}} {{.HelpName}} --id "rHTY.a123" --disable s3/mybucket

`,
	}

	ilmDepLsCmd = cli.Command{
		Name:         "ls",
		Usage:        "lists lifecycle configuration rules set on a bucket",
		Action:       mainILMList,
		Hidden:       true, // to avoid being listed in `mc ilm`
		OnUsageError: onUsageError,
		Before:       setGlobalsFromContext,
		Flags:        append(ilmListFlags, globalFlags...),
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  List lifecycle configuration rules set on a bucket.

EXAMPLES:
  1. List the lifecycle management rules (all fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. List the lifecycle management rules (expration date/days fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --expiry myminio/mybucket

  3. List the lifecycle management rules (transition date/days, storage class fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --transition myminio/mybucket

  4. List the lifecycle management rules in JSON format for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --json myminio/mybucket
`,
	}

	ilmDepExportCmd = cli.Command{
		Name:         "export",
		Usage:        "export lifecycle configuration in JSON format",
		Action:       mainILMExport,
		Hidden:       true, // to avoid being listed in `mc ilm`
		OnUsageError: onUsageError,
		Before:       setGlobalsFromContext,
		Flags:        globalFlags,
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Exports lifecycle configuration in JSON format to STDOUT.

EXAMPLES:
  1. Export lifecycle configuration for 'mybucket' to 'lifecycle.json' file.
     {{.Prompt}} {{.HelpName}} myminio/mybucket > lifecycle.json

  2. Print lifecycle configuration for 'mybucket' to STDOUT.
     {{.Prompt}} {{.HelpName}} play/mybucket
`,
	}

	ilmDepImportCmd = cli.Command{
		Name:         "import",
		Usage:        "import lifecycle configuration in JSON format",
		Action:       mainILMImport,
		Hidden:       true, // to avoid being listed in `mc ilm`
		OnUsageError: onUsageError,
		Before:       setGlobalsFromContext,
		Flags:        globalFlags,
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Import entire lifecycle configuration from STDIN, input file is expected to be in JSON format.

EXAMPLES:
  1. Set lifecycle configuration for the mybucket on alias 'myminio' to the rules imported from lifecycle.json
     {{.Prompt}} {{.HelpName}} myminio/mybucket < lifecycle.json

  2. Set lifecycle configuration for the mybucket on alias 'myminio'. User is expected to enter the JSON contents on STDIN
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
	}
)
