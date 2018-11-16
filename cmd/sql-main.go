/*
 * Minio Client, (C) 2018 Minio, Inc.
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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/mimedb"
)

var (
	sqlFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "query, e",
			Usage: "sql query expression",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "sql query recursively",
		},
		cli.StringFlag{
			Name:  "encrypt-key",
			Usage: "encrypt/decrypt objects (using server-side encryption)",
		},
	}
)

// Display contents of a file.
var sqlCmd = cli.Command{
	Name:   "sql",
	Usage:  "run sql queries on objects",
	Action: mainSQL,
	Before: setGlobalsFromContext,
	Flags:  append(sqlFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Run a query on a set of objects recursively on s3 account.
      $ {{.HelpName}} --recursive --query "select * from S3Object" s3/personalbucket/my-large-csvs/

   2. Run a query on an object on minio account.
      $ {{.HelpName}} --query "select count(s.power) from S3Object" myminio/iot-devices/power-ratio.csv

   3. Run a query on an encrypted object with client provided keys.
      $ {{.HelpName}} --encrypt-key "myminio/iot-devices=32byteslongsecretkeymustbegiven1" \
            --query "select count(s.power) from S3Object" myminio/iot-devices/power-ratio-encrypted.csv
`,
}

func sqlSelect(targetURL, expression string, encKeyDB map[string][]prefixSSEPair) *probe.Error {
	alias, _, _, err := expandAlias(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}

	targetClnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}

	sseKey := getSSEKey(targetURL, encKeyDB[alias])
	outputer, err := targetClnt.Select(expression, sseKey)
	if err != nil {
		return err.Trace(targetURL, expression)
	}
	defer outputer.Close()

	_, e := io.Copy(os.Stdout, outputer)
	return probe.NewError(e)
}

// check sql input arguments.
func checkSQLSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		cli.ShowCommandHelpAndExit(ctx, "sql", 1) // last argument is exit code.
	}
}

// mainSQL is the main entry point for sql command.
func mainSQL(ctx *cli.Context) error {
	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(ctx)
	fatalIf(err, "Unable to parse encryption keys.")

	// validate sql input arguments.
	checkSQLSyntax(ctx)

	// extract URLs.
	URLs := ctx.Args()
	query := ctx.String("query")
	for _, url := range URLs {
		if !isAliasURLDir(url, encKeyDB) {
			errorIf(sqlSelect(url, query, encKeyDB).Trace(url), "Unable to run sql")
			continue
		}
		targetAlias, targetURL, _ := mustExpandAlias(url)
		clnt, err := newClientFromAlias(targetAlias, targetURL)
		if err != nil {
			errorIf(err.Trace(url), "Unable to initialize target `"+url+"`.")
			continue
		}

		for content := range clnt.List(ctx.Bool("recursive"), false, DirNone) {
			if content.Err != nil {
				errorIf(content.Err.Trace(url), "Unable to list on target `"+url+"`.")
				continue
			}
			contentType := mimedb.TypeByExtension(filepath.Ext(content.URL.Path))
			for _, cTypeSuffix := range supportedContentTypes {
				if strings.Contains(contentType, cTypeSuffix) {
					errorIf(sqlSelect(targetAlias+content.URL.Path, query,
						encKeyDB).Trace(content.URL.String()), "Unable to run sql")
				}
			}
		}
	}

	// Done.
	return nil
}
