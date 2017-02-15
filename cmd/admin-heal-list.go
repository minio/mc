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
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio/pkg/probe"
)

var (
	adminHealListFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "List recursively.",
		},
	}
)

var adminHealListCmd = cli.Command{
	Name:   "list",
	Usage:  "Get the list of buckets or objects that need to be healed.",
	Before: setGlobalsFromContext,
	Action: mainAdminHealList,
	Flags:  append(adminHealListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} ALIAS/BUCKET/PREFIX

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. List objects than need to be healed related to 'testbucket' in a Minio server represented by its alias 'play'.
       $ {{.HelpName}} play/testbucket/

    2. Recursively list objects than need to be healed.
       $ {{.HelpName}} --recursive play/

`,
}

// healListMessage container to hold heal information.
type healListMessage struct {
	Status string            `json:"status"`
	Bucket string            `json:"bucket"`
	Object madmin.ObjectInfo `json:"object"`
}

// String colorized service status message.
func (u healListMessage) String() string {
	msg := fmt.Sprintf("Object: %s/%s, ", u.Bucket, u.Object.Key)
	switch u.Object.HealObjectInfo.Status {
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

// checkAdminHealListSyntax - validate all the passed arguments
func checkAdminHealListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "list", 1) // last argument is exit code
	}
}

func mainAdminHealList(ctx *cli.Context) error {

	// Check for heal list syntax
	checkAdminHealListSyntax(ctx)
	// Set console theme for heal command
	console.SetColor("Heal", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	isRecursive := ctx.Bool("recursive")

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		return err.ToGoError()
	}

	// Transform windows backslash to the regular slash in the aliased url
	aliasedURL = filepath.ToSlash(aliasedURL)

	// Compute bucket and object from aliased url
	splits := splitStr(aliasedURL, "/", 3)
	bucket, object := splits[1], splits[2]

	// If bucket is not specified, list buckets that need to be healed
	if bucket == "" {
		bucketsToHeal, e := client.ListBucketsHeal()
		fatalIf(probe.NewError(e), "Cannot list buckets that need healing.")
		// Iterate over buckets that need healing and print them
		for _, bucketHeal := range bucketsToHeal {
			printMsg(healBucketListMessage{Bucket: bucketHeal})
		}
	}

	// When bucket is not specified and recursive flag is not activated,
	// it means we will not scan inside buckets
	if bucket == "" && !isRecursive {
		return nil
	}

	// Now, it is time to examine for objects that need to be healed
	var buckets []string
	if bucket == "" {
		// Bucket is not specified, so we target all buckets so we need to list them.
		s3Client, err := newClient(aliasedURL)
		if err != nil {
			fatalIf(err, "Cannot resolve the provided path.")
		}
		recursive := false
		incomplete := false
		listCh := s3Client.List(recursive, incomplete, DirFirst)
		for content := range listCh {
			// Trim the leading slash and add to the list of buckets
			buckets = append(buckets,
				strings.TrimPrefix(content.URL.Path, string(content.URL.Separator)))
		}
	} else {
		// Bucket is passed in argument, so we will only search for objects in the specified bucket
		buckets = append(buckets, bucket)
	}

	// Iterate over all computed buckets
	for _, currBucket := range buckets {
		// Search for objects that need to be healed in the current bucket
		doneCh := make(chan struct{})
		listCh, e := client.ListObjectsHeal(currBucket, object, isRecursive, doneCh)
		fatalIf(probe.NewError(e), "Cannot list heal objects.")
		// Iterate over objects and print them when not errors
		for obj := range listCh {
			if obj.Err != nil {
				errorIf(probe.NewError(obj.Err), "Cannot heal object `"+obj.Key+"`.")
				continue
			}
			printMsg(healListMessage{Bucket: currBucket, Object: obj})
		}
	}

	return nil
}
