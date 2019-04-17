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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var (
	rbFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force",
			Usage: "allow a recursive remove operation",
		},
		cli.BoolFlag{
			Name:  "dangerous",
			Usage: "allow site-wide removal of objects",
		},
	}
)

// remove a bucket.
var rbCmd = cli.Command{
	Name:   "rb",
	Usage:  "remove a bucket",
	Action: mainRemoveBucket,
	Before: setGlobalsFromContext,
	Flags:  append(rbFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
EXAMPLES:
   1. Remove an empty bucket on Amazon S3 cloud storage
      $ {{.HelpName}} s3/mybucket
	 
   2. Remove a directory hierarchy.
      $ {{.HelpName}} /tmp/this/new/dir1
	 
   3. Remove bucket 'jazz-songs' and all its contents
      $ {{.HelpName}} --force s3/jazz-songs

   4. Remove all buckets and objects recursively from S3 host
      $ {{.HelpName}} --force --dangerous s3
`,
}

// removeBucketMessage is container for delete bucket success and failure messages.
type removeBucketMessage struct {
	Status string `json:"status"`
	Bucket string `json:"bucket"`
}

// String colorized delete bucket message.
func (s removeBucketMessage) String() string {
	return console.Colorize("RemoveBucket", fmt.Sprintf("Removed `%s` successfully.", s.Bucket))
}

// JSON jsonified remove bucket message.
func (s removeBucketMessage) JSON() string {
	removeBucketJSONBytes, e := json.Marshal(s)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(removeBucketJSONBytes)
}

// Validate command line arguments.
func checkRbSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() {
		exitCode := 1
		cli.ShowCommandHelpAndExit(ctx, "rb", exitCode)
	}
	// Set command flags from context.
	isForce := ctx.Bool("force")
	isDangerous := ctx.Bool("dangerous")

	for _, url := range ctx.Args() {
		if isNamespaceRemoval(url) {
			if isForce && isDangerous {
				continue
			}
			fatalIf(errDummy().Trace(),
				"This operation results in **site-wide** removal of buckets. If you are really sure, retry this command with ‘--force’ and ‘--dangerous’ flags.")
		}
	}
}

// deletes a bucket and all its contents
func deleteBucket(url string) *probe.Error {
	targetAlias, targetURL, _ := mustExpandAlias(url)
	clnt, pErr := newClientFromAlias(targetAlias, targetURL)
	if pErr != nil {
		return pErr
	}
	var isIncomplete bool
	isRemoveBucket := true
	contentCh := make(chan *clientContent)
	errorCh := clnt.Remove(isIncomplete, isRemoveBucket, contentCh)

	for content := range clnt.List(true, false, DirLast) {
		if content.Err != nil {
			switch content.Err.ToGoError().(type) {
			case PathInsufficientPermission:
				errorIf(content.Err.Trace(url), "Failed to remove `"+url+"`.")
				// Ignore Permission error.
				continue
			}
			close(contentCh)
			return content.Err
		}
		urlString := content.URL.Path

		sent := false
		for !sent {
			select {
			case contentCh <- content:
				sent = true
			case pErr := <-errorCh:
				switch pErr.ToGoError().(type) {
				case PathInsufficientPermission:
					errorIf(pErr.Trace(urlString), "Failed to remove `"+urlString+"`.")
					// Ignore Permission error.
					continue
				}
				close(contentCh)
				return pErr
			}
		}
		// list internally mimics recursive directory listing of object prefixes for s3 similar to FS.
		// The rmMessage needs to be printed only for actual buckets being deleted and not objects.
		tgt := strings.TrimPrefix(urlString, string(filepath.Separator))
		if !strings.Contains(tgt, string(filepath.Separator)) && tgt != targetAlias {
			printMsg(removeBucketMessage{
				Bucket: targetAlias + urlString, Status: "success",
			})
		}
	}

	// Remove the given url since the user will always want to remove it.
	alias, _ := url2Alias(targetURL)
	if alias != "" {
		contentCh <- &clientContent{URL: *newClientURL(targetURL)}
	}

	// Finish removing and print all the remaining errors
	close(contentCh)
	for pErr := range errorCh {
		switch pErr.ToGoError().(type) {
		case PathInsufficientPermission:
			errorIf(pErr.Trace(url), "Failed to remove `"+url+"`.")
			// Ignore Permission error.
			continue
		}
		return pErr
	}
	return nil
}

// isNamespaceRemoval returns true if alias
// is not qualified by bucket
func isNamespaceRemoval(url string) bool {
	// clean path for aliases like s3/.
	//Note: UNC path using / works properly in go 1.9.2 even though it breaks the UNC specification.
	url = filepath.ToSlash(filepath.Clean(url))
	// namespace removal applies only for non FS. So filter out if passed url represents a directory
	if !isAliasURLDir(url, nil) {
		_, path := url2Alias(url)
		return (path == "")
	}
	return false
}

// mainRemoveBucket is entry point for rb command.
func mainRemoveBucket(ctx *cli.Context) error {
	// check 'rb' cli arguments.
	checkRbSyntax(ctx)
	isForce := ctx.Bool("force")

	// Additional command specific theme customization.
	console.SetColor("RemoveBucket", color.New(color.FgGreen, color.Bold))

	var cErr error
	for _, targetURL := range ctx.Args() {
		// Instantiate client for URL.
		clnt, err := newClient(targetURL)
		if err != nil {
			errorIf(err.Trace(targetURL), "Invalid target `"+targetURL+"`.")
			cErr = exitStatus(globalErrorExitStatus)
			continue
		}
		_, err = clnt.Stat(false, false, nil)
		if err != nil {
			switch err.ToGoError().(type) {
			case BucketNameEmpty:
			default:
				errorIf(err.Trace(targetURL), "Unable to validate target `"+targetURL+"`.")
				cErr = exitStatus(globalErrorExitStatus)
				continue

			}
		}
		isEmpty := true
		for range clnt.List(true, false, DirNone) {
			isEmpty = false
			break
		}
		// For all recursive operations make sure to check for 'force' flag.
		if !isForce && !isEmpty {
			fatalIf(errDummy().Trace(), "`"+targetURL+"` is not empty. Retry this command with ‘--force’ flag if you want to remove `"+targetURL+"` and all its contents")
		}
		e := deleteBucket(targetURL)
		fatalIf(e.Trace(targetURL), "Failed to remove `"+targetURL+"`.")
	}
	return cErr
}
