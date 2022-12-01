package cmd

import "github.com/minio/cli"

var ilmDepCmds = []cli.Command{
	ilmDepAddCmd,
	ilmDepEditCmd,
	ilmDepLsCmd,
	ilmDepRmCmd,
	ilmDepExportCmd,
	ilmDepImportCmd,
	ilmDepRestoreCmd,
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
	ilmDepRestoreCmd = cli.Command{
		Name:         "restore",
		Usage:        "restore archived objects",
		Action:       mainILMRestore,
		Hidden:       true,
		OnUsageError: onUsageError,
		Before:       setGlobalsFromContext,
		Flags:        append(ilmRestoreFlags, globalFlags...),
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Create a restored copy of one or more objects archived on a remote tier. The copy automatically expires
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
)
