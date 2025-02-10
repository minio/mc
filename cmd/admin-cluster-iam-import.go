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
	"github.com/minio/pkg/v3/console"
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
	messages = append(messages, processIAMEntities(info.Skipped, "Skipped")...)
	messages = append(messages, processIAMEntities(info.Removed, "Removed")...)
	messages = append(messages, processIAMEntities(info.Added, "Added")...)
	messages = append(messages, processErrIAMEntities(info.Failed)...)
	return strings.Join(messages, "\n")
}

func processIAMEntities(entities madmin.IAMEntities, action string) []string {
	var messages []string
	if len(entities.Policies) > 0 {
		messages = append(messages, fmt.Sprintf("%s policies: %v", action, strings.Join(entities.Policies, ", ")))
	}
	if len(entities.Users) > 0 {
		messages = append(messages, fmt.Sprintf("%s users: %v", action, strings.Join(entities.Users, ", ")))
	}
	if len(entities.Groups) > 0 {
		messages = append(messages, fmt.Sprintf("%s groups: %v", action, strings.Join(entities.Groups, ", ")))
	}
	if len(entities.ServiceAccounts) > 0 {
		messages = append(messages, fmt.Sprintf("%s service accounts: %v", action, strings.Join(entities.ServiceAccounts, ", ")))
	}
	var users []string
	for _, pol := range entities.UserPolicies {
		for name := range pol {
			users = append(users, name)
		}
	}
	if len(users) > 0 {
		messages = append(messages, fmt.Sprintf("%s policies for users: %v", action, strings.Join(users, ", ")))
	}
	var groups []string
	for _, pol := range entities.GroupPolicies {
		for name := range pol {
			groups = append(groups, name)
		}
	}
	if len(groups) > 0 {
		messages = append(messages, fmt.Sprintf("%s policies for groups: %v", action, strings.Join(groups, ", ")))
	}
	var stsarr []string
	for _, pol := range entities.STSPolicies {
		for name := range pol {
			stsarr = append(stsarr, name)
		}
	}
	if len(stsarr) > 0 {
		messages = append(messages, fmt.Sprintf("%s policies for sts: %v", action, strings.Join(stsarr, ", ")))
	}
	return messages
}

func processErrIAMEntities(entities madmin.IAMErrEntities) []string {
	var messages []string

	var policies []string
	for _, entry := range entities.Policies {
		policies = append(policies, entry.Name)
	}
	if len(policies) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add policies: %v", strings.Join(policies, ", ")))
	}
	var users []string
	for _, entry := range entities.Users {
		users = append(users, entry.Name)
	}
	if len(users) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add users: %v", strings.Join(users, ", ")))
	}
	var groups []string
	for _, entry := range entities.Groups {
		groups = append(groups, entry.Name)
	}
	if len(groups) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add groups: %v", strings.Join(groups, ", ")))
	}
	var sas []string
	for _, entry := range entities.ServiceAccounts {
		sas = append(sas, entry.Name)
	}
	if len(sas) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add service accounts: %v", strings.Join(sas, ", ")))
	}
	var polusers []string
	for _, pol := range entities.UserPolicies {
		polusers = append(polusers, pol.Name)
	}
	if len(polusers) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add policies for users: %v", strings.Join(polusers, ", ")))
	}
	var polgroups []string
	for _, pol := range entities.GroupPolicies {
		polgroups = append(polgroups, pol.Name)
	}
	if len(polgroups) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add policies for groups: %v", strings.Join(polgroups, ", ")))
	}
	var polsts []string
	for _, pol := range entities.STSPolicies {
		polsts = append(polsts, pol.Name)
	}
	if len(polsts) > 0 {
		messages = append(messages, fmt.Sprintf("Failed to add policies for sts: %v", strings.Join(polsts, ", ")))
	}
	return messages
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
	defer f.Close()

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	iamr, e := client.ImportIAMV2(context.Background(), f)
	if e != nil {
		f.Seek(0, 0)
		e = client.ImportIAM(context.Background(), f)
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to import IAM info.")
		if !globalJSON {
			console.Infof("IAM info imported to %s from %s\n", aliasedURL, args.Get(1))
		}
	} else {
		printMsg(iamImportInfo(iamr))
	}

	return nil
}
