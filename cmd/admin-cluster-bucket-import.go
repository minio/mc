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

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/klauspost/compress/zip"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminClusterBucketImportCmd = cli.Command{
	Name:            "import",
	Usage:           "restore bucket metadata from a zip file",
	Action:          mainClusterBucketImport,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET/[BUCKET] /path/to/backups/bucket-metadata.zip

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Recover bucket metadata for all buckets from previously saved bucket metadata backup.
     {{.Prompt}} {{.HelpName}} myminio /backups/myminio-bucket-metadata.zip
`,
}

func checkBucketImportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainClusterBucketImport - bucket metadata import command
func mainClusterBucketImport(ctx *cli.Context) error {
	// Check for command syntax
	checkBucketImportSyntax(ctx)
	console.SetColor("Name", color.New(color.Bold, color.FgCyan))
	console.SetColor("success", color.New(color.Bold, color.FgGreen))
	console.SetColor("warning", color.New(color.Bold, color.FgYellow))
	console.SetColor("errors", color.New(color.Bold, color.FgRed))
	console.SetColor("statusMsg", color.New(color.Bold, color.FgHiWhite))
	console.SetColor("failCell", color.New(color.FgRed))
	console.SetColor("passCell", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	var r io.Reader
	var sz int64
	f, e := os.Open(args.Get(1))
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to get bucket metadata")
	}
	if st, e := f.Stat(); e == nil {
		sz = st.Size()
	}
	defer f.Close()
	r = f

	_, e = zip.NewReader(r.(io.ReaderAt), sz)
	fatalIf(probe.NewError(e).Trace(args...), fmt.Sprintf("Unable to read zip file %s", args.Get(1)))

	f, e = os.Open(args.Get(1))
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get bucket metadata")
	defer f.Close()

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	aliasedURL = filepath.Clean(aliasedURL)
	_, bucket := url2Alias(aliasedURL)

	rpt, e := client.ImportBucketMetadata(context.Background(), bucket, f)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to import bucket metadata.")

	printMsg(importMetaMsg{
		BucketMetaImportErrs: rpt,
		Status:               "success",
		URL:                  aliasedURL,
		Op:                   ctx.Command.Name,
	})

	return nil
}

type importMetaMsg struct {
	madmin.BucketMetaImportErrs
	Op     string
	URL    string `json:"url"`
	Status string `json:"status"`
}

func statusTick(s madmin.MetaStatus) string {
	switch {
	case s.Err != "":
		return console.Colorize("failCell", crossTickCell)
	case !s.IsSet:
		return blankCell
	default:
		return console.Colorize("passCell", tickCell)
	}
}

func (i importMetaMsg) String() string {
	m := i.Buckets
	totBuckets := len(m)
	totErrs := 0
	for _, st := range m {
		if st.ObjectLock.Err != "" || st.Versioning.Err != "" ||
			st.SSEConfig.Err != "" || st.Tagging.Err != "" ||
			st.Lifecycle.Err != "" || st.Quota.Err != "" ||
			st.Policy.Err != "" || st.Notification.Err != "" ||
			st.Cors.Err != "" || st.Err != "" {
			totErrs++
		}
	}
	var b strings.Builder
	numSch := "success"
	if totErrs > 0 {
		numSch = "warning"
	}
	msg := "\n" + console.Colorize(numSch, totBuckets-totErrs) +
		console.Colorize("statusMsg", "/") +
		console.Colorize("success", totBuckets) +
		console.Colorize("statusMsg", " buckets were imported successfully.")
	fmt.Fprintln(&b, msg)
	if totErrs > 0 {
		fmt.Fprintln(&b, console.Colorize("errors", "Errors: \n"))
		for bucket, st := range m {
			if st.ObjectLock.Err != "" || st.Versioning.Err != "" ||
				st.SSEConfig.Err != "" || st.Tagging.Err != "" ||
				st.Lifecycle.Err != "" || st.Quota.Err != "" ||
				st.Policy.Err != "" || st.Notification.Err != "" ||
				st.Cors.Err != "" || st.Err != "" {
				fmt.Fprintln(&b, printImportErrs(bucket, st))
			}
		}
	}
	return b.String()
}

func (i importMetaMsg) JSON() string {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", " ")
	// Disable escaping special chars to display XML tags correctly
	enc.SetEscapeHTML(false)

	fatalIf(probe.NewError(enc.Encode(i.Buckets)), "Unable to marshal into JSON.")
	return buf.String()
}

// pretty print import errors
func printImportErrs(bucket string, r madmin.BucketStatus) string {
	var b strings.Builder
	placeHolder := ""
	key := fmt.Sprintf("%-10s: %s", "Name", bucket)
	fmt.Fprintln(&b, console.Colorize("Name", key))

	if r.Err != "" {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, console.Colorize("errors", "Error: "), r.Err)
		fmt.Fprintln(&b)
	}
	if r.ObjectLock.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Object lock: ", statusTick(r.ObjectLock))
		fmt.Fprintln(&b)
	}
	if r.Versioning.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Versioning: ", statusTick(r.Versioning))
		fmt.Fprintln(&b)
	}

	if r.SSEConfig.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Encryption: ", statusTick(r.SSEConfig))
		fmt.Fprintln(&b)
	}
	if r.Lifecycle.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Lifecycle: ", statusTick(r.Lifecycle))
		fmt.Fprintln(&b)
	}
	if r.Notification.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Notification: ", statusTick(r.Notification))
		fmt.Fprintln(&b)
	}
	if r.Quota.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Quota: ", statusTick(r.Quota))
		fmt.Fprintln(&b)
	}
	if r.Policy.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Policy: ", statusTick(r.Policy))
		fmt.Fprintln(&b)
	}
	if r.Tagging.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "Tagging: ", statusTick(r.Tagging))
		fmt.Fprintln(&b)
	}
	if r.Cors.IsSet {
		fmt.Fprintf(&b, "%2s%s %s", placeHolder, "CORS: ", statusTick(r.Cors))
		fmt.Fprintln(&b)
	}
	return b.String()
}
