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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zip"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var adminClusterIAMImportCmd = cli.Command{
	Name:            "import",
	Usage:           "imports IAM info from zipped file",
	Action:          mainClusterIAMImport,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET/BUCKET /path/to/myminio-iam-info.zip

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set IAM info from previously exported metadata zip file.
     {{.Prompt}} {{.HelpName}} myminio /tmp/myminio-iam-info.zip

`,
}

type iamImportInfo madmin.ImportIAMResult

func (i iamImportInfo) JSON() string {
	bs, e := json.MarshalIndent(madmin.ImportIAMResult(i), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (i iamImportInfo) String() string {
	var messages []string
	info := madmin.ImportIAMResult(i)
	if len(info.Skipped) > 0 {
		messages = append(messages, fmt.Sprintf("Skipped Entries: %v", strings.Join(info.Skipped, ", ")))
	}
	if len(info.Removed) > 0 {
		messages = append(messages, fmt.Sprintf("Removed Entries: %v", strings.Join(info.Removed, ", ")))
	}
	if len(info.Added) > 0 {
		messages = append(messages, fmt.Sprintf("Newly Added Entries: %v", strings.Join(info.Added, ", ")))
	}
	if len(info.Failed) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add Entries: %v", strings.Join(info.Failed, ", ")))
	}
	return strings.Join(messages, "\n")
}

func checkIAMImportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainClusterIAMImport - iam info import command
func mainClusterIAMImport(ctx *cli.Context) error {
	// Check for command syntax
	checkIAMImportSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := filepath.ToSlash(args.Get(0))
	aliasedURL = filepath.Clean(aliasedURL)

	var r io.Reader
	var sz int64
	f, e := os.Open(args.Get(1))
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to get IAM info")
	}
	if st, e := f.Stat(); e == nil {
		sz = st.Size()
	}
	defer f.Close()
	r = f

	_, e = zip.NewReader(r.(io.ReaderAt), sz)
	fatalIf(probe.NewError(e).Trace(args...), fmt.Sprintf("Unable to read zip file %s", args.Get(1)))

	f, e = os.Open(args.Get(1))
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get IAM info")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	iamr, e := client.ImportIAMV2(context.Background(), f)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to import IAM info.")

	printMsg(iamImportInfo(iamr))
	return nil
}
