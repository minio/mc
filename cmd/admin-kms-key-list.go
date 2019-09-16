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
	"bufio"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var (
	kmsListFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "list recursively",
		},
	}
)

var adminKMSListKeysCmd = cli.Command{
	Name:   "list",
	Usage:  "list objects and KMS master keys",
	Action: mainAdminKMSListKeys,
	Before: setGlobalsFromContext,
	Flags:  append(kmsListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [KEY-NAME-PREFIX]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all objects encrypted with the default master key at a MinIO server/cluster.
     $ {{.HelpName}} -r play
  2. List all objects encrypted with the default master key within a bucket.
     $ {{.HelpName}} -r play/bucket
  3. List all objects encrypted with the default master key within a bucket but don't
     list recursively.
     $ {{.HelpName}} play/bucket
  4. List all objects encrypted with a master key that starts with "my" within a bucket.
     $ {{.HelpName}} -r play/bucket my
  5. Count the number of all objects  encrypted with the the master key: 'my-master-key'. 
     $ {{.HelpName}} -r --json play my-master-key | jq '."key-id"' | uniq -c
  6. Count the number of objects in bucket encrypted with the default master key.
     $ {{.HelpName}} play/bucket --json | jq '."key-id"' | uniq -c
`,
}

// mainAdminKMSListKeys is the handle for the "mc admin kms key" command.
func mainAdminKMSListKeys(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}

	aliasURL := strings.SplitN(ctx.Args().Get(0), "/", 2)
	if len(aliasURL) == 1 {
		aliasURL = append(aliasURL, "")
	}
	if len(aliasURL) == 2 {
		aliasURL = append(aliasURL, "")
	}

	client, e := newAdminClient(aliasURL[0])
	fatalIf(e, "Cannot get a configured admin connection.")

	var keyID string
	if len(ctx.Args()) == 2 {
		keyID = ctx.Args().Get(1)
	}
	recursive := ctx.IsSet("recursive") || ctx.IsSet("r")
	stream, err := client.ListKeys(aliasURL[1], aliasURL[2], keyID, recursive)
	fatalIf(probe.NewError(err), "Failed to get status information")

	var listResp madmin.KMSListKeyResponse
	tokenStream := bufio.NewScanner(stream)
	for tokenStream.Scan() {
		if err = json.Unmarshal(tokenStream.Bytes(), &listResp); err != nil {
			fatalIf(probe.NewError(err), "Failed to list objects and KMS master keys")
			break
		}

		if globalJSON {
			console.Println(listResp.JSON())
		} else {
			masterKey := console.Colorize("Info", listResp.KeyID)
			size := console.Colorize("Yellow", fmt.Sprintf("%7s ", strings.Join(strings.Fields(humanize.IBytes(listResp.Size)), "")))
			path := console.Colorize("BoldWhite", path.Join(listResp.Bucket, listResp.Object))
			console.Printf("%s \t%s %s\n", masterKey, size, path)
		}
	}
	if err := tokenStream.Err(); err != nil {
		fatalIf(probe.NewError(err), "Failed to list objects and KMS master keys")
	}
	return nil
}
