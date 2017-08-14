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
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
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

var adminHealCmd = cli.Command{
	Name:            "heal",
	Usage:           "Manage heal tasks",
	Before:          adminHealBefore,
	Action:          mainAdminHeal,
	Flags:           append(adminHealFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. To format newly replaced disks in a Minio server with alias 'play'
       $ {{.HelpName}} play

    2. Heal 'testbucket' in a Minio server with alias 'play'
       $ {{.HelpName}} play/testbucket/

    3. Heal all objects under 'dir' prefix
       $ {{.HelpName}} --recursive play/testbucket/dir/

    4. Heal all uploads under 'dir' prefix
       $ {{.HelpName}} --incomplete --recursive play/testbucket/dir/

    5. Issue a fake heal operation to list all objects to be healed
       $ {{.HelpName}} --fake play

    6. Issue a fake heal operation to list all uploads to be healed
       $ {{.HelpName}} --fake --incomplete play

    7. Issue a fake heal operation to list all objects to be healed under 'dir' prefix
       $ {{.HelpName}} --recursive --fake play/testbucket/dir/

    8. Issue a fake heal operation to list all uploads to be healed under 'dir' prefix
       $ {{.HelpName}} --incomplete --recursive --fake play/testbucket/dir/

`,
}

// healMessage container to hold repair information.
type healMessage struct {
	Status string             `json:"status"`
	Result madmin.HealResult  `json:"result"`
	Bucket string             `json:"bucket"`
	Object *madmin.ObjectInfo `json:"object"`
	Upload *madmin.UploadInfo `json:"upload"`
}

// adminHealBefore used to provide users with temporary warning message
func adminHealBefore(ctx *cli.Context) error {
	color.Yellow("\t *** mc admin heal is EXPERIMENTAL ***")
	return setGlobalsFromContext(ctx)
}

// String colorized service status message.
func (u healMessage) String() string {
	allOfflineTmpl := "is not healed since many disks were offline"
	allHealedTmpl := "is healed on all disks"
	someHealedTmpl := "is healed on some disks while other disks were offline"

	msg := ""
	if u.Object != nil {
		msg = fmt.Sprintf("Object %s/%s ", u.Bucket, u.Object.Key)
	} else {
		msg = fmt.Sprintf("Upload %s/%s/%s ", u.Bucket, u.Upload.Key, u.Upload.UploadID)
	}

	switch u.Result.State {
	case madmin.HealNone:
		msg += allOfflineTmpl
	case madmin.HealPartial:
		msg += someHealedTmpl
	case madmin.HealOK:
		msg += allHealedTmpl
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

// healListMessage container to hold heal information.
type healListMessage struct {
	Status string             `json:"status"`
	Bucket string             `json:"bucket"`
	Object *madmin.ObjectInfo `json:"object"`
	Upload *madmin.UploadInfo `json:"upload"`
}

// String colorized service status message.
func (u healListMessage) String() string {
	msg := ""
	var healStatus madmin.HealStatus

	// Check if we have object heal information
	if u.Object != nil {
		msg += fmt.Sprintf("Object: %s/%s, ", u.Bucket, u.Object.Key)
		healStatus = u.Object.HealObjectInfo.Status
	} else {
		msg += fmt.Sprintf("Upload: %s/%s, ", u.Bucket, u.Upload.Key)
		healStatus = u.Upload.HealUploadInfo.Status
	}

	// Print heal status
	switch healStatus {
	case madmin.CanHeal:
		msg += "can be healed."
	case madmin.CanPartiallyHeal:
		msg += "can be partially healed."
	case madmin.Corrupted:
		msg += "cannot be healed."
	case madmin.QuorumUnavailable:
		msg += "quorum not available for healing."
	}
	return console.Colorize("Heal", msg)
}

// JSON jsonified service status Message message.
func (u healListMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.Marshal(u)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// healBucketListMessage container to hold heal information.
type healBucketListMessage struct {
	Status string            `json:"status"`
	Bucket madmin.BucketInfo `json:"bucket"`
}

// String colorized service status message.
func (u healBucketListMessage) String() string {
	msg := fmt.Sprintf("Bucket: `%s`, ", u.Bucket.Name)
	switch u.Bucket.HealBucketInfo.Status {
	case madmin.CanHeal:
		msg += "can be healed"
	case madmin.Corrupted:
		msg += "cannot be healed"
	case madmin.QuorumUnavailable:
		msg += "quorum not available for healing"
	}
	msg += ".\n"
	return console.Colorize("Heal", msg)
}

// JSON jsonified service status Message message.
func (u healBucketListMessage) JSON() string {
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
		case madmin.CanHeal, madmin.CanPartiallyHeal:
			// Heal Object
			var healResult madmin.HealResult
			if healResult, e = client.HealObject(bucket, obj.Key, isFake); e != nil {
				errorIf(probe.NewError(e), "Cannot repair object: `"+obj.Key+"`")
				continue
			}

			// Print successful message
			if isFake {
				printMsg(healListMessage{Bucket: bucket, Object: &obj})
			} else {
				printMsg(healMessage{Bucket: bucket, Object: &obj, Result: healResult})
			}
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
		case madmin.CanHeal, madmin.CanPartiallyHeal:
			// Heal Upload
			var healResult madmin.HealResult
			if healResult, e = client.HealUpload(bucket, upload.Key, upload.UploadID, isFake); e != nil {
				errorIf(probe.NewError(e), "Cannot repair upload: `"+upload.Key+"`")
				continue
			}
			// Print successful message
			if isFake {
				printMsg(healListMessage{Bucket: bucket, Upload: &upload})
			} else {
				printMsg(healMessage{Bucket: bucket, Upload: &upload, Result: healResult})
			}
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
		cli.ShowCommandHelpAndExit(ctx, "heal", 1) // last argument is exit code
	}
}

// listAllToBeHealed - list objects/uploads to be healed in the object
// store with alias `aliasedURL`
// - isIncomplete - if true we list uploads to be healed otherwise
// list objects to be healed
func listAllToBeHealed(client *madmin.AdminClient, aliasedURL string, isIncomplete bool) *probe.Error {
	s3Client, err := newClient(aliasedURL)
	if err != nil {
		return err
	}

	recursive := false
	incomplete := false
	listCh := s3Client.List(recursive, incomplete, DirFirst)

	var buckets []string
	for content := range listCh {
		// Trim the leading slash and add to the list of buckets
		buckets = append(buckets,
			strings.TrimPrefix(content.URL.Path, string(content.URL.Separator)))
	}

	// isRecursive is always true since we have empty object names
	// when `mc admin heal --fake [-r] s1/` is invoked.
	isRecursive := true

	// Iterate over all computed buckets
	for _, currBucket := range buckets {
		// Search for objects that need to be healed in the current bucket
		doneCh := make(chan struct{})

		if isIncomplete {
			listCh, e := client.ListUploadsHeal(currBucket, "", isRecursive, doneCh)
			fatalIf(probe.NewError(e), "Cannot list heal uploads.")
			// Iterate over uploads and print them when not errors
			for upload := range listCh {
				if upload.Err != nil {
					errorIf(probe.NewError(upload.Err), "Cannot heal upload `"+upload.Key+"`.")
					continue
				}
				// Skip for non-recursive use case.
				if upload.HealUploadInfo == nil {
					continue
				}
				printMsg(healListMessage{Bucket: currBucket, Upload: &upload})
			}

		} else {
			listCh, e := client.ListObjectsHeal(currBucket, "", isRecursive, doneCh)
			fatalIf(probe.NewError(e), "Cannot list heal objects.")
			// Iterate over objects and print them when not errors
			for obj := range listCh {
				if obj.Err != nil {
					errorIf(probe.NewError(obj.Err), "Cannot heal object `"+obj.Key+"`.")
					continue
				}
				// Skip for non-recursive use case.
				if obj.HealObjectInfo == nil {
					continue
				}
				printMsg(healListMessage{Bucket: currBucket, Object: &obj})
			}
		}
	}
	return nil
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
		// If --fake was given, print all objects/uploads that
		// need to be healed.
		if isFake {
			lErr := listAllToBeHealed(client, aliasedURL, isIncomplete)
			fatalIf(lErr, "Unable to list all objects to be healed")
			return nil
		}
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
