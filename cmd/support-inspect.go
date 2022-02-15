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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/secure-io/sio-go"
)

var supportInspectFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "encrypt",
		Usage: "encrypt content with one time key for confidential data",
	},
	cli.StringFlag{
		Name:  "export",
		Value: "json",
		Usage: "exports inspect data as JSON or data JSON from 'xl.meta', supported values are 'json' or 'djson'",
	},
}

var supportInspectCmd = cli.Command{
	Name:            "inspect",
	Usage:           "upload raw object contents for analysis",
	Action:          mainSupportInspect,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportInspectFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Download 'xl.meta' for a specific object from all the drives in a zip file.
     {{.Prompt}} {{.HelpName}} myminio/bucket/test*/xl.meta

  2. Download all constituent parts for a specific object, and optionally encrypt the downloaded zip.
     {{.Prompt}} {{.HelpName}} --encrypt myminio/bucket/test*/*/part.*

  3. Download recursively all objects at a prefix. NOTE: This can be an expensive operation use it with caution.
     {{.Prompt}} {{.HelpName}} myminio/bucket/test/**
`,
}

func checkSupportInspectSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "inspect", 1) // last argument is exit code
	}

	if ctx.IsSet("export") && globalJSON {
		fatalIf(errInvalidArgument(), "--export=type cannot be specified with --json flag")
	}
}

// mainSupportInspect - the entry function of inspect command
func mainSupportInspect(ctx *cli.Context) error {
	// Check for command syntax
	checkSupportInspectSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	encrypt := ctx.Bool("encrypt")

	console.SetColor("File", color.New(color.FgWhite, color.Bold))
	console.SetColor("Key", color.New(color.FgHiRed, color.Bold))

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, prefix := splits[1], splits[2]

	shellName, _ := getShellName()
	if runtime.GOOS != "windows" && shellName != "bash" && strings.Contains(prefix, "*") {
		console.Infoln("Your shell is auto determined as '" + shellName + "', wildcard patterns are only supported with 'bash' SHELL.")
	}

	key, r, ierr := client.Inspect(context.Background(), madmin.InspectOptions{
		Volume: bucket,
		File:   prefix,
	})
	fatalIf(probe.NewError(ierr).Trace(aliasedURL), "Unable to inspect file.")

	// Create profile zip file
	tmpFile, e := ioutil.TempFile("", "mc-inspect-")
	fatalIf(probe.NewError(e), "Unable to download file data.")

	ext := "enc"
	if !encrypt || ctx.IsSet("export") {
		ext = "zip"
		r = decryptInspect(key, r)
	}

	// Copy zip content to target download file
	_, e = io.Copy(tmpFile, r)
	fatalIf(probe.NewError(e), "Unable to download file data.")

	// Close everything
	r.Close()
	tmpFile.Close()

	// Create an id that is also crc.
	var id [4]byte
	binary.LittleEndian.PutUint32(id[:], crc32.ChecksumIEEE(key[:]))

	// We use 4 bytes of the 32 bytes to identify they file.
	downloadPath := fmt.Sprintf("inspect.%s.%s", hex.EncodeToString(id[:]), ext)
	fi, e := os.Stat(downloadPath)
	if e == nil && !fi.IsDir() {
		e = moveFile(downloadPath, downloadPath+"."+time.Now().Format(dateTimeFormatFilename))
		fatalIf(probe.NewError(e), "Unable to create a backup of "+downloadPath)
	} else {
		if !os.IsNotExist(e) {
			fatal(probe.NewError(e), "Unable to download file data")
		}
	}

	fatalIf(probe.NewError(moveFile(tmpFile.Name(), downloadPath)), "Unable to rename downloaded data, file exists at %s", tmpFile.Name())
	if ctx.IsSet("export") {
		switch v := ctx.String("export"); v {
		case "json":
			inspectToExportType(downloadPath, false)
		case "djson":
			inspectToExportType(downloadPath, true)
		default:
			os.Remove(downloadPath)
			fatalIf(errInvalidArgument().Trace("export="+v), "Unable to export inspect data")
		}
		os.Remove(downloadPath)
		return nil
	}

	hexKey := hex.EncodeToString(id[:]) + hex.EncodeToString(key[:])
	if !globalJSON {
		if !encrypt {
			console.Infof("File data successfully downloaded as %s\n", console.Colorize("File", downloadPath))
			return nil
		}
		console.Infof("Encrypted file data successfully downloaded as %s\n", console.Colorize("File", downloadPath))
		console.Infof("Decryption key: %s\n\n", console.Colorize("Key", hexKey))

		console.Info("The decryption key will ONLY be shown here. It cannot be recovered.\n")
		console.Info("The encrypted file can safely be shared without the decryption key.\n")
		console.Info("Even with the decryption key, data stored with encryption cannot be accessed.\n")
		return nil
	}

	v := struct {
		File string `json:"file"`
		Key  string `json:"key,omitempty"`
	}{
		File: downloadPath,
		Key:  hexKey,
	}
	if !encrypt {
		v.Key = ""
	}
	b, e := json.Marshal(v)
	fatalIf(probe.NewError(e), "Unable to serialize data")
	console.Println(string(b))
	return nil
}

func decryptInspect(key [32]byte, r io.Reader) io.ReadCloser {
	stream, err := sio.AES_256_GCM.Stream(key[:])
	fatalIf(probe.NewError(err), "Unable to initiate decryption")

	// Zero nonce, we only use each key once, and 32 bytes is plenty.
	nonce := make([]byte, stream.NonceSize())
	return ioutil.NopCloser(stream.DecryptReader(r, nonce, nil))
}
