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
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

// mv command flags.
var (
	mvFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "move recursively",
		},
		cli.StringFlag{
			Name:  "older-than",
			Usage: "move objects older than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "newer-than",
			Usage: "move objects newer than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "storage-class, sc",
			Usage: "set storage class for new object(s) on target",
		},
		cli.StringFlag{
			Name:  "encrypt",
			Usage: "encrypt/decrypt objects (using server-side encryption with server managed keys)",
		},
		cli.StringFlag{
			Name:  "attr",
			Usage: "add custom metadata for the object",
		},
		cli.BoolFlag{
			Name:  "continue, c",
			Usage: "create or resume move session",
		},
		cli.BoolFlag{
			Name:  "preserve, a",
			Usage: "preserve filesystem attributes (mode, ownership, timestamps)",
		},
		cli.BoolFlag{
			Name:  "disable-multipart",
			Usage: "disable multipart upload feature",
		},
	}
)

// Move command.
var mvCmd = cli.Command{
	Name:         "mv",
	Usage:        "move objects",
	Action:       mainMove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(mvFlags, ioFlags...), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE [SOURCE...] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
ENVIRONMENT VARIABLES:
  MC_ENCRYPT:      list of comma delimited prefixes
  MC_ENCRYPT_KEY:  list of comma delimited prefix=secret values

EXAMPLES:
  01. Move a list of objects from local file system to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} Music/*.ogg s3/jukebox/

  02. Move a folder recursively from MinIO cloud storage to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive play/mybucket/burningman2011/ s3/mybucket/

  03. Move multiple local folders recursively to MinIO cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive backup/2014/ backup/2015/ play/archive/

  04. Move a bucket recursively from aliased Amazon S3 cloud storage to local filesystem on Windows.
      {{.Prompt}} {{.HelpName}} --recursive s3\documents\2014\ C:\Backups\2014

  05. Move files older than 7 days and 10 hours from MinIO cloud storage to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --older-than 7d10h play/mybucket/burningman2011/ s3/mybucket/

  06. Move files newer than 7 days and 10 hours from MinIO cloud storage to a local path.
      {{.Prompt}} {{.HelpName}} --newer-than 7d10h play/mybucket/burningman2011/ ~/latest/

  07. Move an object with name containing unicode characters to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} 本語 s3/andoria/

  08. Move a local folder with space separated characters to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive 'workdir/documents/May 2014/' s3/miniocloud

  09. Move a folder with encrypted objects recursively from Amazon S3 to MinIO cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive --encrypt-key "s3/documents/=32byteslongsecretkeymustbegiven1,myminio/documents/=32byteslongsecretkeymustbegiven2" s3/documents/ myminio/documents/

  10. Move a folder with encrypted objects recursively from Amazon S3 to MinIO cloud storage. In case the encryption key contains non-printable character like tab, pass the
      base64 encoded string as key.
      {{.Prompt}} {{.HelpName}} --recursive --encrypt-key "s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=,myminio/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=" s3/documents/ myminio/documents/

  11. Move a list of objects from local file system to MinIO cloud storage with specified metadata, separated by ";"
      {{.Prompt}} {{.HelpName}} --attr "key1=value1;key2=value2" Music/*.mp4 play/mybucket/

  12. Move a folder recursively from MinIO cloud storage to Amazon S3 cloud storage with Cache-Control and custom metadata, separated by ";".
      {{.Prompt}} {{.HelpName}} --attr "Cache-Control=max-age=90000,min-fresh=9000;key1=value1;key2=value2" --recursive play/mybucket/burningman2011/ s3/mybucket/

  13. Move a text file to an object storage and assign REDUCED_REDUNDANCY storage-class to the uploaded object.
      {{.Prompt}} {{.HelpName}} --storage-class REDUCED_REDUNDANCY myobject.txt play/mybucket

  14. Move a text file to an object storage and create or resume copy session.
      {{.Prompt}} {{.HelpName}} --recursive --continue dir/ play/mybucket

  15. Move a text file to an object storage and preserve the file system attribute as metadata.
      {{.Prompt}} {{.HelpName}} -a myobject.txt play/mybucket

  16. Move a text file to an object storage and disable multipart upload feature.
      {{.Prompt}} {{.HelpName}} --disable-multipart myobject.txt play/mybucket
`,
}

type removeClientInfo struct {
	client    Client
	contentCh chan *ClientContent
	errorCh   <-chan *probe.Error
}

type removeManager struct {
	removeMap      map[string]*removeClientInfo
	removeMapMutex sync.RWMutex
	wg             sync.WaitGroup
}

func (rm *removeManager) readErrors(errorCh <-chan *probe.Error, targetURL string) {
	rm.wg.Add(1)
	go func() {
		defer rm.wg.Done()
		for pErr := range errorCh {
			errorIf(pErr.Trace(targetURL), "Failed to remove in`"+targetURL+"`.")
		}
	}()
}

// This function should be parallel-safe because it is executed by ParallelManager
// If targetAlias is empty, it means we will target local FS contents
func (rm *removeManager) add(ctx context.Context, targetAlias, targetURL string) {
	rm.removeMapMutex.Lock()
	clientInfo := rm.removeMap[targetAlias]
	if clientInfo == nil {
		client, pErr := newClientFromAlias(targetAlias, targetURL)
		if pErr != nil {
			errorIf(pErr.Trace(targetURL), "Invalid argument `"+targetURL+"`.")
			return
		}

		contentCh := make(chan *ClientContent, 10000)
		errorCh := client.Remove(ctx, false, false, false, contentCh)
		rm.readErrors(errorCh, targetURL)

		clientInfo = &removeClientInfo{
			client:    client,
			contentCh: contentCh,
			errorCh:   errorCh,
		}

		rm.removeMap[targetAlias] = clientInfo
	}
	rm.removeMapMutex.Unlock()

	clientInfo.contentCh <- &ClientContent{URL: *newClientURL(targetURL)}
}

func (rm *removeManager) close() {
	for _, clientInfo := range rm.removeMap {
		close(clientInfo.contentCh)
	}

	// Wait until all on-going client.Remove() operations to finish
	rm.wg.Wait()
}

var rmManager = &removeManager{
	removeMap: make(map[string]*removeClientInfo),
}

// mainMove is the entry point for mv command.
func mainMove(cliCtx *cli.Context) error {
	ctx, cancelMove := context.WithCancel(globalContext)
	defer cancelMove()

	// Parse encryption keys per command.
	encKeyDB, err := getEncKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	// Parse metadata.
	userMetaMap := make(map[string]string)
	if cliCtx.String("attr") != "" {
		userMetaMap, err = getMetaDataEntry(cliCtx.String("attr"))
		fatalIf(err, "Unable to parse attribute %v", cliCtx.String("attr"))
	}

	// check 'copy' cli arguments.
	checkCopySyntax(ctx, cliCtx, encKeyDB, true)

	if cliCtx.NArg() == 2 {
		args := cliCtx.Args()
		srcURL := args.Get(0)
		dstURL := args.Get(1)
		if srcURL == dstURL {
			fatalIf(errDummy().Trace(), fmt.Sprintf("Source and destination urls cannot be the same: %v.", srcURL))
		}
	}

	// Check if source URLs does not have object locking enabled
	// since we cannot move them (remove them from the source)
	for _, urlStr := range cliCtx.Args()[:cliCtx.NArg()-1] {
		client, err := newClient(urlStr)
		if err != nil {
			fatalIf(err.Trace(), "Unable to parse the provided url.")
		}
		if _, ok := client.(*S3Client); ok {
			enabled, err := isBucketLockEnabled(ctx, urlStr)
			if err != nil {
				fatalIf(err.Trace(), "Unable to get bucket lock configuration of `%s`", urlStr)
			}
			if enabled {
				fatalIf(errDummy().Trace(), fmt.Sprintf("Object lock configuration is enabled on the specified bucket in alias %v.", urlStr))
			}
		}
	}

	// Additional command speific theme customization.
	console.SetColor("Copy", color.New(color.FgGreen, color.Bold))

	recursive := cliCtx.Bool("recursive")
	olderThan := cliCtx.String("older-than")
	newerThan := cliCtx.String("newer-than")
	storageClass := cliCtx.String("storage-class")
	sseKeys := os.Getenv("MC_ENCRYPT_KEY")
	if key := cliCtx.String("encrypt-key"); key != "" {
		sseKeys = key
	}

	if sseKeys != "" {
		sseKeys, err = getDecodedKey(sseKeys)
		fatalIf(err, "Unable to parse encryption keys.")
	}
	sse := cliCtx.String("encrypt")

	var session *sessionV8

	if cliCtx.Bool("continue") {
		sessionID := getHash("mv", cliCtx.Args())
		if isSessionExists(sessionID) {
			session, err = loadSessionV8(sessionID)
			fatalIf(err.Trace(sessionID), "Unable to load session.")
		} else {
			session = newSessionV8(sessionID)
			session.Header.CommandType = "mv"
			session.Header.CommandBoolFlags["recursive"] = recursive
			session.Header.CommandStringFlags["older-than"] = olderThan
			session.Header.CommandStringFlags["newer-than"] = newerThan
			session.Header.CommandStringFlags["storage-class"] = storageClass
			session.Header.CommandStringFlags["encrypt-key"] = sseKeys
			session.Header.CommandStringFlags["encrypt"] = sse
			session.Header.CommandBoolFlags["session"] = cliCtx.Bool("continue")

			if cliCtx.Bool("preserve") {
				session.Header.CommandBoolFlags["preserve"] = cliCtx.Bool("preserve")
			}
			session.Header.UserMetaData = userMetaMap
			session.Header.CommandBoolFlags["disable-multipart"] = cliCtx.Bool("disable-multipart")

			var e error
			if session.Header.RootPath, e = os.Getwd(); e != nil {
				session.Delete()
				fatalIf(probe.NewError(e), "Unable to get current working folder.")
			}

			// extract URLs.
			session.Header.CommandArgs = cliCtx.Args()
		}
	}

	e := doCopySession(ctx, cancelMove, cliCtx, session, encKeyDB, true)
	if session != nil {
		session.Delete()
	}

	console.Colorize("Copy", "Waiting for move operations to complete")
	rmManager.close()

	return e
}
