// Copyright (c) 2015-2021 MinIO, Inc.
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
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

// verify specific flags.
var (
	verifyFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "versions",
			Usage: "list all versions",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "list recursively",
		},
	}
)

// Verify files and versions
var verifyCmd = cli.Command{
	Name:         "verify",
	Usage:        "Check the etag of objects",
	Action:       mainVerify,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(verifyFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Verify the ETag of all objects in 'myminio'
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkVerifySyntax - validate all the passed arguments
func checkVerifySyntax(ctx context.Context, cliCtx *cli.Context) (string, bool, bool) {
	args := cliCtx.Args()
	if !cliCtx.Args().Present() {
		args = []string{"."}
	}

	if len(args) != 1 {
		fatalIf(errInvalidArgument().Trace(args...), "Unable to process multiple arguments.")
	}

	arg := args[0]

	if strings.TrimSpace(arg) == "" {
		fatalIf(errInvalidArgument().Trace(args...), "Unable to validate empty argument.")
	}

	isRecursive := cliCtx.Bool("recursive")
	withOlderVersions := cliCtx.Bool("versions")

	return arg, isRecursive, withOlderVersions
}

// getMD5Sum returns MD5 sum of given data.
func getMD5Sum(data []byte) []byte {
	hash := md5.New()
	hash.Write(data)
	return hash.Sum(nil)
}

// getMD5Hash returns MD5 hash in hex encoding of given data.
func getMD5Hash(data []byte) string {
	return hex.EncodeToString(getMD5Sum(data))
}

func downloadAndCheckETag(ctx context.Context, alias, url, versionID string) (bool, *probe.Error) {
	clnt, err := newClientFromAlias(alias, url)
	if err != nil {
		return false, err
	}

	st, err := clnt.Stat(ctx, StatOptions{versionID: versionID})
	if err != nil {
		return false, err
	}

	parts := 1
	s := strings.Split(st.ETag, "-")
	if len(s) > 1 {
		if p, err := strconv.Atoi(s[1]); err == nil {
			parts = p
		} else {
			return false, probe.NewError(errors.New("unrecognized ETag format"))
		}
	}

	var partsMD5Sum [][]byte

	for p := 1; p <= parts; p++ {
		rd, err := clnt.Get(context.Background(), GetOptions{VersionID: versionID, PartNumber: p})
		if err != nil {
			return false, err
		}
		h := md5.New()
		if _, err := io.Copy(h, rd); err != nil {
			return false, probe.NewError(err)
		}
		partsMD5Sum = append(partsMD5Sum, h.Sum(nil))
	}

	corrupted := false

	switch len(partsMD5Sum) {
	case 1:
		md5sum := fmt.Sprintf("%x", partsMD5Sum[0])
		if md5sum != st.ETag {
			corrupted = true
		}
	default:
		var totalMD5SumBytes []byte
		for _, sum := range partsMD5Sum {
			totalMD5SumBytes = append(totalMD5SumBytes, sum...)
		}
		s3MD5 := fmt.Sprintf("%s-%d", getMD5Hash(totalMD5SumBytes), len(partsMD5Sum))
		if s3MD5 != st.ETag {
			corrupted = true
		}
	}

	return corrupted, nil
}

// mainVerify - is a handler for mc verify command
func mainVerify(cliCtx *cli.Context) error {
	ctx, cancelVerify := context.WithCancel(globalContext)
	defer cancelVerify()

	// Additional command specific theme customization.
	console.SetColor("ETagMismatch", color.New(color.Bold))

	// check 'verify' cliCtx arguments.
	targetURL, isRecursive, olderVersions := checkVerifySyntax(ctx, cliCtx)

	clnt, err := newClient(targetURL)
	fatalIf(err.Trace(targetURL), "Unable to initialize target `"+targetURL+"`.")

	alias, _, _, _ := expandAlias(targetURL)

	if clnt.GetURL().Type != objectStorage {
		fatalIf(errInvalidArgument().Trace(targetURL), "The path `"+targetURL+"` does not support verifying objects hashes.")
	}

	// Capture last error
	var cErr error

	for content := range clnt.List(ctx, ListOptions{
		Recursive:         isRecursive,
		WithOlderVersions: olderVersions,
		ShowDir:           DirNone,
	}) {
		if content.Err != nil {
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to verify object/folder.")
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}

		if content.StorageClass == s3StorageClassGlacier {
			continue
		}

		corrupted, err := downloadAndCheckETag(ctx, alias, content.URL.String(), content.VersionID)
		fatalIf(err, "Unable to check")

		if corrupted {
			errorIf(errDummy(), "Corrupted object/version detected", content.URL.String())
		} else {
			fmt.Println(content.URL.String(), "is fine")
		}
	}

	return cErr
}
