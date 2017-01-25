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
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio/pkg/probe"
)

var (
	adminHealFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "List recursively.",
		},
		cli.BoolFlag{
			Name:  "fake, k",
			Usage: "Issue a fake heal operation.",
		},
	}
)

var adminHealCmd = cli.Command{
	Name:   "heal",
	Usage:  "Manage heal tasks.",
	Before: setGlobalsFromContext,
	Action: mainAdminHeal,
	Flags:  append(adminHealFlags, globalFlags...),
	Subcommands: []cli.Command{
		adminHealListCmd,
	},
	CustomHelpTemplate: `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} [FLAGS] COMMAND

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}

COMMANDS:
   {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
   {{end}}

EXAMPLES:
    1. Heal 'testbucket' in a Minio server represented by its alias 'play'.
       $ mc admin {{.Name}} play/testbucket/
    2. Heal all objects under 'dir' prefix 
       $ mc admin {{.Name}} --recursive play/testbucket/dir/
    3. Issue a fake heal operation to see what the server could report
       $ mc admin {{.Name}} --fake play/testbucket/dir/

`,
}

// healObjectMessage container to hold repair information.
type healObjectMessage struct {
	Status string            `json:"status"`
	Bucket string            `json:"bucket"`
	Object madmin.ObjectInfo `json:"object"`
}

// String colorized service status message.
func (u healObjectMessage) String() string {
	msg := fmt.Sprintf("Object %s/%s is healed.", u.Bucket, u.Object.Key)
	return console.Colorize("Heal", msg)
}

// JSON jsonified service status Message message.
func (u healObjectMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminHealSyntax - validate all the passed arguments
func checkAdminHealSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowSubcommandHelp(ctx)
		os.Exit(1)
	}
}

// mainAdminHeal - the entry function of heal command
func mainAdminHeal(ctx *cli.Context) error {

	// Check for command syntax
	checkAdminHealSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	isRecursive := ctx.Bool("recursive")
	isFake := ctx.Bool("fake")

	console.SetColor("Heal", color.New(color.FgGreen, color.Bold))

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		return err.ToGoError()
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, object := splits[1], splits[2]

	var e error

	// Heal format if bucket is not specified and quit immediately
	if bucket == "" {
		e = client.HealFormat(isFake)
		fatalIf(probe.NewError(e), "Cannot heal the specified storage format.")
		return nil
	}

	// Heal the specified bucket
	e = client.HealBucket(bucket, isFake)
	fatalIf(probe.NewError(e), "Cannot repair bucket.")

	// Search for objects that need healing
	doneCh := make(chan struct{})
	healObjectsCh, e := client.ListObjectsHeal(bucket, object, isRecursive, doneCh)
	fatalIf(probe.NewError(e), "Cannot list objects that need to be healed.")

	// Iterate over objects that need healing
	for obj := range healObjectsCh {
		// Return for any error
		fatalIf(probe.NewError(obj.Err), "Cannot list objects that need to be healed.")

		// Check the heal status, and call heal object API only when an object can be healed
		switch healInfo := *obj.HealObjectInfo; healInfo.Status {
		case madmin.CanHeal:
			// Heal Object
			e = client.HealObject(bucket, obj.Key, isFake)
			if e != nil {
				errorIf(probe.NewError(e), "Cannot repair object: `"+obj.Key+"`")
				continue
			}

			// Print successful message
			printMsg(healObjectMessage{Bucket: bucket, Object: obj})

		case madmin.QuorumUnavailable:
			errorIf(errDummy().Trace(), obj.Key+" cannot be healed until quorum is available.")
		case madmin.Corrupted:
			errorIf(errDummy().Trace(), obj.Key+" cannot be healed, not enough information.")
		}
	}

	return nil
}
