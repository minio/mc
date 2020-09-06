/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
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
	Name:   "mv",
	Usage:  "move objects",
	Action: mainMove,
	Before: setGlobalsFromContext,
	Flags:  append(append(mvFlags, ioFlags...), globalFlags...),
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
	doneCh         chan struct{}
	isClosed       int32
	wg             sync.WaitGroup
}

func (rm *removeManager) readErrors(errorCh <-chan *probe.Error, targetURL string) {
	rm.wg.Add(1)
	go func() {
		defer rm.wg.Done()
		var stop bool
		for !stop {
			select {
			case pErr, ok := <-errorCh:
				if ok {
					errorIf(pErr.Trace(targetURL), "Failed to remove in`"+targetURL+"`.")
				}
			case <-rm.doneCh:
				stop = true
			}
		}

		for pErr := range errorCh {
			if pErr != nil {
				errorIf(pErr.Trace(targetURL), "Failed to remove in `"+targetURL+"`.")
			}
		}
	}()
}

func (rm *removeManager) add(ctx context.Context, targetAlias, targetURL string) {
	if atomic.LoadInt32(&rm.isClosed) != 0 {
		return
	}

	rm.removeMapMutex.RLock()
	clientInfo := rm.removeMap[targetAlias]
	rm.removeMapMutex.RUnlock()

	if clientInfo == nil {
		client, pErr := newClientFromAlias(targetAlias, targetURL)
		if pErr != nil {
			errorIf(pErr.Trace(targetURL), "Invalid argument `"+targetURL+"`.")
			return
		}

		contentCh := make(chan *ClientContent)
		errorCh := client.Remove(ctx, false, false, false, contentCh)
		rm.readErrors(errorCh, targetURL)

		clientInfo = &removeClientInfo{
			client:    client,
			contentCh: contentCh,
			errorCh:   errorCh,
		}

		rm.removeMapMutex.Lock()
		rm.removeMap[targetAlias] = clientInfo
		rm.removeMapMutex.Unlock()
	}

	go func() {
		clientInfo.contentCh <- &ClientContent{URL: *newClientURL(targetURL)}
	}()
}

func (rm *removeManager) close() {
	atomic.StoreInt32(&rm.isClosed, 1)
	rm.removeMapMutex.Lock()
	defer rm.removeMapMutex.Unlock()

	for _, clientInfo := range rm.removeMap {
		close(clientInfo.contentCh)
	}

	close(rm.doneCh)

	rm.wg.Wait()
}

var rmManager = &removeManager{
	removeMap: make(map[string]*removeClientInfo),
	doneCh:    make(chan struct{}),
}

func bgRemove(ctx context.Context, url string) {
	remove := func(targetAlias, targetURL string) {
		clnt, pErr := newClientFromAlias(targetAlias, targetURL)
		if pErr != nil {
			errorIf(pErr.Trace(url), "Invalid argument `"+url+"`.")
		}

		contentCh := make(chan *ClientContent, 1)
		contentCh <- &ClientContent{URL: *newClientURL(targetURL)}
		close(contentCh)
		errorCh := clnt.Remove(ctx, false, false, false, contentCh)
		for pErr := range errorCh {
			if pErr != nil {
				errorIf(pErr.Trace(url), "Failed to remove `"+url+"`.")
				switch pErr.ToGoError().(type) {
				case PathInsufficientPermission:
					// Ignore Permission error.
					continue
				}
			}
		}
	}

	targetAlias, targetURL, _ := mustExpandAlias(url)
	if targetAlias == "" {
		// File system does not support batch deletion hence use individual deletion.
		go remove(targetAlias, targetURL)
		return
	}

	rmManager.add(ctx, targetAlias, targetURL)
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

	for _, urlStr := range cliCtx.Args() {
		client, err := newClient(urlStr)
		if err != nil {
			fatalIf(err.Trace(), "Unable to parse the provided url.")
		}

		if s3Client, ok := client.(*S3Client); ok {
			if _, _, _, _, err = s3Client.GetObjectLockConfig(ctx); err == nil {
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
