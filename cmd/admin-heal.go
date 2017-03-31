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
			Usage: "Heal recursively",
		},
		cli.BoolFlag{
			Name:  "fake, k",
			Usage: "Issue a fake heal operation",
		},
		cli.BoolFlag{
			Name:  "incomplete, I",
			Usage: "Heal uploads recursively",
		},
	}
)

var adminHealHelpTemplate = `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [COMMAND]

COMMANDS:
  {{range .VisibleCommands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Heal 'testbucket' in a Minio server represented by its alias 'play'.
       $ {{.HelpName}} play/testbucket/

    2. Heal all objects under 'dir' prefix
       $ {{.HelpName}} --recursive play/testbucket/dir/

    3. Heal all uploads under 'dir' prefix
       $ {{.HelpName}} --incomplete --recursive play/testbucket/dir/

    4. Issue a fake heal operation to see what the server could report
       $ {{.HelpName}} --fake play/testbucket/dir/

`
var adminHealCmd = cli.Command{
	Name:   "heal",
	Usage:  "Manage heal tasks",
	Before: setGlobalsFromContext,
	Action: mainAdminHeal,
	Flags:  append(adminHealFlags, globalFlags...),
	Subcommands: []cli.Command{
		adminHealListCmd,
	},
	HideHelpCommand: true,
}

// healMessage container to hold repair information.
type healMessage struct {
	Status string             `json:"status"`
	Bucket string             `json:"bucket"`
	Object *madmin.ObjectInfo `json:"object"`
	Upload *madmin.UploadInfo `json:"upload"`
}

// String colorized service status message.
func (u healMessage) String() string {
	msg := ""
	if u.Object != nil {
		msg += fmt.Sprintf("Object %s/%s is healed.", u.Bucket, u.Object.Key)
	} else {
		msg += fmt.Sprintf("Upload %s/%s/%s is healed.", u.Bucket, u.Upload.Key, u.Upload.UploadID)
	}
	return console.Colorize("Heal", msg)
}

// JSON jsonified service status Message message.
func (u healMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

func healObjects(client *madmin.AdminClient, bucket, object string, isRecursive, isFake bool) {
	// Search for objects that need healing
	doneCh := make(chan struct{})
	healObjectsCh, e := client.ListObjectsHeal(bucket, object, isRecursive, doneCh)
	fatalIf(probe.NewError(e), "Cannot list objects to be healed.")

	// Iterate over objects that need healing
	for obj := range healObjectsCh {
		// Continue to next object upon any error.
		if obj.Err != nil {
			errorIf(probe.NewError(obj.Err), "Cannot list objects to be healed.")
			continue
		}
		// Heal object info is nil skip it, must be a directory.
		if obj.HealObjectInfo == nil {
			continue
		}

		// Check the heal status, and call heal object API only when an object can be healed
		switch healInfo := *obj.HealObjectInfo; healInfo.Status {
		case madmin.CanHeal:
			// Heal Object
			if e = client.HealObject(bucket, obj.Key, isFake); e != nil {
				errorIf(probe.NewError(e), "Cannot repair object: `"+obj.Key+"`")
				continue
			}

			// Print successful message
			printMsg(healMessage{Bucket: bucket, Object: &obj})
		case madmin.QuorumUnavailable:
			errorIf(errDummy().Trace(), obj.Key+" cannot be healed until quorum is available.")
		case madmin.Corrupted:
			errorIf(errDummy().Trace(), obj.Key+" cannot be healed, not enough information.")
		}
	}
}

func healUploads(client *madmin.AdminClient, bucket, object string, isRecursive, isFake bool) {
	// Search for uploads that need healing
	doneCh := make(chan struct{})
	healUploadsCh, e := client.ListUploadsHeal(bucket, object, isRecursive, doneCh)
	fatalIf(probe.NewError(e), "Cannot list uploads to be healed.")

	// Iterate over uploads that need healing
	for upload := range healUploadsCh {
		// Continue to next upload upon any error.
		if upload.Err != nil {
			errorIf(probe.NewError(upload.Err), "Cannot list uploads to be healed.")
			continue
		}
		// Heal upload info is nil skip it, must be a directory.
		if upload.HealUploadInfo == nil {
			continue
		}

		// Check the heal status, and call heal upload API only when an upload can be healed
		switch healInfo := *upload.HealUploadInfo; healInfo.Status {
		case madmin.CanHeal:
			// Heal Upload
			if e = client.HealUpload(bucket, upload.Key, upload.UploadID, isFake); e != nil {
				errorIf(probe.NewError(e), "Cannot repair upload: `"+upload.Key+"`")
				continue
			}
			// Print successful message
			printMsg(healMessage{Bucket: bucket, Upload: &upload})
		case madmin.QuorumUnavailable:
			errorIf(errDummy().Trace(), upload.Key+" cannot be healed until quorum is available.")
		case madmin.Corrupted:
			errorIf(errDummy().Trace(), upload.Key+" cannot be healed, not enough information.")
		}
	}
}

// checkAdminHealSyntax - validate all the passed arguments
func checkAdminHealSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.HelpPrinter(ctx.App.Writer, adminHealHelpTemplate, ctx.App)
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
	isIncomplete := ctx.Bool("incomplete")
	isFake := ctx.Bool("fake")

	console.SetColor("Heal", color.New(color.FgGreen, color.Bold))

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Cannot initialize admin client.")
		return nil
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, object := splits[1], splits[2]

	var e error

	// Heal format if bucket is not specified and quit immediately
	if bucket == "" {
		e = client.HealFormat(isFake)
		fatalIf(probe.NewError(e), "Cannot heal the disks.")
		return nil
	}

	// Heal the specified bucket
	e = client.HealBucket(bucket, isFake)
	fatalIf(probe.NewError(e), "Cannot heal bucket.")

	if !isIncomplete {
		healObjects(client, bucket, object, isRecursive, isFake)
	} else {
		healUploads(client, bucket, object, isRecursive, isFake)
	}

	return nil
}
