package cmd

import (
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var aliasExportCmd = cli.Command{
	Name:            "export",
	ShortName:       "e",
	Usage:           "export configuration info to stdout",
	Action:          mainAliasExport,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

  Credentials to be exported will be in the following JSON format:
  
  {
    "url": "http://localhost:9000",
    "accessKey": "YJ0RI0F4R5HWY38MD873",
    "secretKey": "OHz5CT7xdMHiXnKZP0BmZ5P4G5UvWvVaxR8gljLG",
    "api": "s3v4",
    "path": "auto"
  }

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Export the provided 'alias' to credentials.json file:
     {{ .Prompt }} {{ .HelpName }} myminio/ > credentials.json

  2. Export the credentials to standard output and pipe it to import command
     {{ .Prompt }} {{ .HelpName }} alias1/  | mc alias import alias2/
`,
}

// checkAliasExportSyntax - verifies input arguments to 'alias export'.
func checkAliasExportSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if ctx.NArg() == 0 {
		showCommandHelpAndExit(ctx, 1)
	}
	if ctx.NArg() > 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for alias export command.")
	}

	alias := cleanAlias(args.Get(0))
	if !isValidAlias(alias) {
		fatalIf(errInvalidAlias(alias), "Unable to validate alias")
	}
}

// exportAlias - get an alias config
func exportAlias(alias string) {
	mcCfgV10, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config `"+mustGetMcConfigPath()+"`.")

	cfg, ok := mcCfgV10.Aliases[alias]
	if !ok {
		fatalIf(errInvalidArgument().Trace(alias), "Unable to export credentials")
	}

	buf, e := json.Marshal(cfg)
	fatalIf(probe.NewError(e).Trace(alias), "Unable to export credentials")

	console.Println(string(buf))
}

func mainAliasExport(cli *cli.Context) error {
	args := cli.Args()

	checkAliasExportSyntax(cli)

	exportAlias(cleanAlias(args.Get(0)))

	return nil
}
