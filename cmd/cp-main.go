/*
 * MinIO Client (C) 2014-2020 MinIO, Inc.
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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

// cp command flags.
var (
	cpFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "rewind",
			Usage: "roll back object(s) to current version at specified time",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "select an object version to copy",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "copy recursively",
		},
		cli.StringFlag{
			Name:  "older-than",
			Usage: "copy objects older than L days, M hours and N minutes",
		},
		cli.StringFlag{
			Name:  "newer-than",
			Usage: "copy objects newer than L days, M hours and N minutes",
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
			Usage: "create or resume copy session",
		},
		cli.BoolFlag{
			Name:  "preserve, a",
			Usage: "preserve filesystem attributes (mode, ownership, timestamps)",
		},
		cli.BoolFlag{
			Name:  "disable-multipart",
			Usage: "disable multipart upload feature",
		},
		cli.BoolFlag{
			Name:  "md5",
			Usage: "force all upload(s) to calculate md5sum checksum",
		},
		cli.StringFlag{
			Name:  rmFlag,
			Usage: "retention mode to be applied on the object (governance, compliance)",
		},
		cli.StringFlag{
			Name:  rdFlag,
			Usage: "retention duration for the object in d days or y years",
		},
		cli.StringFlag{
			Name:  lhFlag,
			Usage: "apply legal hold to the copied object (on, off)",
		},
	}
)

var rmFlag = "retention-mode"
var rdFlag = "retention-duration"
var lhFlag = "legal-hold"

// ErrInvalidMetadata reflects invalid metadata format
var ErrInvalidMetadata = errors.New("specified metadata should be of form key1=value1;key2=value2;... and so on")

// Copy command.
var cpCmd = cli.Command{
	Name:   "cp",
	Usage:  "copy objects",
	Action: mainCopy,
	Before: setGlobalsFromContext,
	Flags:  append(append(cpFlags, ioFlags...), globalFlags...),
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
  01. Copy a list of objects from local file system to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} Music/*.ogg s3/jukebox/

  02. Copy a folder recursively from MinIO cloud storage to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive play/mybucket/burningman2011/ s3/mybucket/

  03. Copy multiple local folders recursively to MinIO cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive backup/2014/ backup/2015/ play/archive/

  04. Copy a bucket recursively from aliased Amazon S3 cloud storage to local filesystem on Windows.
      {{.Prompt}} {{.HelpName}} --recursive s3\documents\2014\ C:\Backups\2014

  05. Copy files older than 7 days and 10 hours from MinIO cloud storage to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --older-than 7d10h play/mybucket/burningman2011/ s3/mybucket/

  06. Copy files newer than 7 days and 10 hours from MinIO cloud storage to a local path.
      {{.Prompt}} {{.HelpName}} --newer-than 7d10h play/mybucket/burningman2011/ ~/latest/

  07. Copy an object with name containing unicode characters to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} 本語 s3/andoria/

  08. Copy a local folder with space separated characters to Amazon S3 cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive 'workdir/documents/May 2014/' s3/miniocloud

  09. Copy a folder with encrypted objects recursively from Amazon S3 to MinIO cloud storage.
      {{.Prompt}} {{.HelpName}} --recursive --encrypt-key "s3/documents/=32byteslongsecretkeymustbegiven1,myminio/documents/=32byteslongsecretkeymustbegiven2" s3/documents/ myminio/documents/

  10. Copy a folder with encrypted objects recursively from Amazon S3 to MinIO cloud storage. In case the encryption key contains non-printable character like tab, pass the
      base64 encoded string as key.
      {{.Prompt}} {{.HelpName}} --recursive --encrypt-key "s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=,myminio/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=" s3/documents/ myminio/documents/

  11. Copy a list of objects from local file system to MinIO cloud storage with specified metadata, separated by ";"
      {{.Prompt}} {{.HelpName}} --attr "key1=value1;key2=value2" Music/*.mp4 play/mybucket/

  12. Copy a folder recursively from MinIO cloud storage to Amazon S3 cloud storage with Cache-Control and custom metadata, separated by ";".
      {{.Prompt}} {{.HelpName}} --attr "Cache-Control=max-age=90000,min-fresh=9000;key1=value1;key2=value2" --recursive play/mybucket/burningman2011/ s3/mybucket/

  13. Copy a text file to an object storage and assign REDUCED_REDUNDANCY storage-class to the uploaded object.
      {{.Prompt}} {{.HelpName}} --storage-class REDUCED_REDUNDANCY myobject.txt play/mybucket

  14. Copy a text file to an object storage and create or resume copy session.
      {{.Prompt}} {{.HelpName}} --recursive --continue dir/ play/mybucket

  15. Copy a text file to an object storage and preserve the file system attribute as metadata.
      {{.Prompt}} {{.HelpName}} -a myobject.txt play/mybucket

  16. Copy a text file to an object storage with object lock mode set to 'GOVERNANCE' with retention duration 1 day.
      {{.Prompt}} {{.HelpName}} --retention-mode governance --retention-duration 1d locked.txt play/locked-bucket/

  17. Copy a text file to an object storage with legal-hold enabled.
      {{.Prompt}} {{.HelpName}} --legal-hold on locked.txt play/locked-bucket/

  18. Copy a text file to an object storage and disable multipart upload feature.
      {{.Prompt}} {{.HelpName}} --disable-multipart myobject.txt play/mybucket

  19. Roll back 10 days in the past to copy the content of 'mybucket'
      {{.Prompt}} {{.HelpName}} --rewind 10d -r play/mybucket/ /tmp/dest/

`,
}

// copyMessage container for file copy messages
type copyMessage struct {
	Status     string `json:"status"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Size       int64  `json:"size"`
	TotalCount int64  `json:"totalCount"`
	TotalSize  int64  `json:"totalSize"`
}

// String colorized copy message
func (c copyMessage) String() string {
	return console.Colorize("Copy", fmt.Sprintf("`%s` -> `%s`", c.Source, c.Target))
}

// JSON jsonified copy message
func (c copyMessage) JSON() string {
	c.Status = "success"
	copyMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(copyMessageBytes)
}

// Progress - an interface which describes current amount
// of data written.
type Progress interface {
	Get() int64
	SetTotal(int64)
}

// ProgressReader can be used to update the progress of
// an on-going transfer progress.
type ProgressReader interface {
	io.Reader
	Progress
}

// doCopy - Copy a single file from source to destination
func doCopy(ctx context.Context, cpURLs URLs, pg ProgressReader, encKeyDB map[string][]prefixSSEPair, isMvCmd bool, preserve bool) URLs {
	if cpURLs.Error != nil {
		cpURLs.Error = cpURLs.Error.Trace()
		return cpURLs
	}

	sourceAlias := cpURLs.SourceAlias
	sourceURL := cpURLs.SourceContent.URL
	targetAlias := cpURLs.TargetAlias
	targetURL := cpURLs.TargetContent.URL
	length := cpURLs.SourceContent.Size
	sourcePath := filepath.ToSlash(filepath.Join(sourceAlias, sourceURL.Path))

	if progressReader, ok := pg.(*progressBar); ok {
		progressReader.SetCaption(cpURLs.SourceContent.URL.String() + ": ")
	} else {
		targetPath := filepath.ToSlash(filepath.Join(targetAlias, targetURL.Path))
		printMsg(copyMessage{
			Source:     sourcePath,
			Target:     targetPath,
			Size:       length,
			TotalCount: cpURLs.TotalCount,
			TotalSize:  cpURLs.TotalSize,
		})
	}

	urls := uploadSourceToTargetURL(ctx, cpURLs, pg, encKeyDB, preserve)
	if isMvCmd && urls.Error == nil {
		bgRemove(ctx, sourcePath)
	}

	return urls
}

// doCopyFake - Perform a fake copy to update the progress bar appropriately.
func doCopyFake(ctx context.Context, cpURLs URLs, pg Progress, isMvCmd bool) URLs {
	if progressReader, ok := pg.(*progressBar); ok {
		progressReader.ProgressBar.Add64(cpURLs.SourceContent.Size)
	}

	if isMvCmd {
		sourcePath := filepath.ToSlash(filepath.Join(cpURLs.SourceAlias, cpURLs.SourceContent.URL.Path))
		bgRemove(ctx, sourcePath)
	}

	return cpURLs
}

// doPrepareCopyURLs scans the source URL and prepares a list of objects for copying.
func doPrepareCopyURLs(ctx context.Context, session *sessionV8, cancelCopy context.CancelFunc) (totalBytes, totalObjects int64) {
	// Separate source and target. 'cp' can take only one target,
	// but any number of sources.
	sourceURLs := session.Header.CommandArgs[:len(session.Header.CommandArgs)-1]
	targetURL := session.Header.CommandArgs[len(session.Header.CommandArgs)-1] // Last one is target

	// Access recursive flag inside the session header.
	isRecursive := session.Header.CommandBoolFlags["recursive"]
	rewind := session.Header.CommandStringFlags["rewind"]
	versionID := session.Header.CommandStringFlags["version-id"]
	olderThan := session.Header.CommandStringFlags["older-than"]
	newerThan := session.Header.CommandStringFlags["newer-than"]
	encryptKeys := session.Header.CommandStringFlags["encrypt-key"]
	encrypt := session.Header.CommandStringFlags["encrypt"]
	encKeyDB, err := parseAndValidateEncryptionKeys(encryptKeys, encrypt)
	fatalIf(err, "Unable to parse encryption keys.")

	// Create a session data file to store the processed URLs.
	dataFP := session.NewDataWriter()

	var scanBar scanBarFunc
	if !globalQuiet && !globalJSON { // set up progress bar
		scanBar = scanBarFactory()
	}

	URLsCh := prepareCopyURLs(ctx, sourceURLs, targetURL, isRecursive, encKeyDB, olderThan, newerThan, parseRewindFlag(rewind), versionID)
	done := false
	for !done {
		select {
		case cpURLs, ok := <-URLsCh:
			if !ok { // Done with URL preparation
				done = true
				break
			}
			if cpURLs.Error != nil {
				// Print in new line and adjust to top so that we don't print over the ongoing scan bar
				if !globalQuiet && !globalJSON {
					console.Eraseline()
				}
				if strings.Contains(cpURLs.Error.ToGoError().Error(), " is a folder.") {
					errorIf(cpURLs.Error.Trace(), "Folder cannot be copied. Please use `...` suffix.")
				} else {
					errorIf(cpURLs.Error.Trace(), "Unable to prepare URL for copying.")
				}
				break
			}

			jsonData, e := json.Marshal(cpURLs)
			if e != nil {
				session.Delete()
				fatalIf(probe.NewError(e), "Unable to prepare URL for copying. Error in JSON marshaling.")
			}
			dataFP.Write(jsonData)
			dataFP.Write([]byte{'\n'})
			if !globalQuiet && !globalJSON {
				scanBar(cpURLs.SourceContent.URL.String())
			}

			totalBytes += cpURLs.SourceContent.Size
			totalObjects++
		case <-globalContext.Done():
			cancelCopy()
			// Print in new line and adjust to top so that we don't print over the ongoing scan bar
			if !globalQuiet && !globalJSON {
				console.Eraseline()
			}
			session.Delete() // If we are interrupted during the URL scanning, we drop the session.
			os.Exit(0)
		}
	}

	session.Header.TotalBytes = totalBytes
	session.Header.TotalObjects = totalObjects
	session.Save()
	return
}

func doCopySession(ctx context.Context, cancelCopy context.CancelFunc, cli *cli.Context, session *sessionV8, encKeyDB map[string][]prefixSSEPair, isMvCmd bool) error {
	var isCopied func(string) bool
	var totalObjects, totalBytes int64

	var cpURLsCh = make(chan URLs, 10000)

	// Store a progress bar or an accounter
	var pg ProgressReader

	// Enable progress bar reader only during default mode.
	if !globalQuiet && !globalJSON { // set up progress bar
		pg = newProgressBar(totalBytes)
	} else {
		pg = newAccounter(totalBytes)
	}

	sourceURLs := cli.Args()[:len(cli.Args())-1]
	targetURL := cli.Args()[len(cli.Args())-1] // Last one is target

	tgtClnt, err := newClient(targetURL)
	fatalIf(err, "Unable to initialize `"+targetURL+"`.")

	// Check if the target bucket has object locking enabled
	var withLock bool
	if _, _, _, _, err = tgtClnt.GetObjectLockConfig(ctx); err == nil {
		withLock = true
	}

	if session != nil {
		// isCopied returns true if an object has been already copied
		// or not. This is useful when we resume from a session.
		isCopied = isLastFactory(session.Header.LastCopied)

		if !session.HasData() {
			totalBytes, totalObjects = doPrepareCopyURLs(ctx, session, cancelCopy)
		} else {
			totalBytes, totalObjects = session.Header.TotalBytes, session.Header.TotalObjects
		}

		pg.SetTotal(totalBytes)

		go func() {
			// Prepare URL scanner from session data file.
			urlScanner := bufio.NewScanner(session.NewDataReader())
			for {
				if !urlScanner.Scan() || urlScanner.Err() != nil {
					close(cpURLsCh)
					break
				}

				var cpURLs URLs
				if e := json.Unmarshal([]byte(urlScanner.Text()), &cpURLs); e != nil {
					errorIf(probe.NewError(e), "Unable to unmarshal %s", urlScanner.Text())
					continue
				}

				cpURLsCh <- cpURLs
			}

		}()
	} else {
		// Access recursive flag inside the session header.
		isRecursive := cli.Bool("recursive")
		olderThan := cli.String("older-than")
		newerThan := cli.String("newer-than")
		rewind := cli.String("rewind")
		versionID := cli.String("version-id")

		go func() {
			totalBytes := int64(0)
			for cpURLs := range prepareCopyURLs(ctx, sourceURLs, targetURL, isRecursive,
				encKeyDB, olderThan, newerThan, parseRewindFlag(rewind), versionID) {
				if cpURLs.Error != nil {
					// Print in new line and adjust to top so that we
					// don't print over the ongoing scan bar
					if !globalQuiet && !globalJSON {
						console.Eraseline()
					}
					if strings.Contains(cpURLs.Error.ToGoError().Error(),
						" is a folder.") {
						errorIf(cpURLs.Error.Trace(),
							"Folder cannot be copied. Please use `...` suffix.")
					} else {
						errorIf(cpURLs.Error.Trace(),
							"Unable to start copying.")
					}
					break
				} else {
					totalBytes += cpURLs.SourceContent.Size
					pg.SetTotal(totalBytes)
					totalObjects++
				}
				cpURLsCh <- cpURLs
			}
			close(cpURLsCh)
		}()
	}

	var quitCh = make(chan struct{})
	var statusCh = make(chan URLs)

	parallel, queueCh := newParallelManager(statusCh)

	go func() {
		gracefulStop := func() {
			close(queueCh)
			parallel.wait()
			close(statusCh)
		}

		for {
			select {
			case <-quitCh:
				gracefulStop()
				return
			case cpURLs, ok := <-cpURLsCh:
				if !ok {
					gracefulStop()
					return
				}

				// Save total count.
				cpURLs.TotalCount = totalObjects

				// Save totalSize.
				cpURLs.TotalSize = totalBytes

				// Initialize target metadata.
				cpURLs.TargetContent.Metadata = make(map[string]string)

				// Initialize target user metadata.
				cpURLs.TargetContent.UserMetadata = make(map[string]string)

				// Check and handle storage class if passed in command line args
				if storageClass := cli.String("storage-class"); storageClass != "" {
					cpURLs.TargetContent.Metadata["X-Amz-Storage-Class"] = storageClass
				}

				// update Object retention related fields
				if session != nil {
					cpURLs.TargetContent.RetentionMode = session.Header.CommandStringFlags[rmFlag]
					if cpURLs.TargetContent.RetentionMode != "" {
						cpURLs.TargetContent.RetentionEnabled = true
					}
					cpURLs.TargetContent.RetentionDuration = session.Header.CommandStringFlags[rdFlag]
					cpURLs.TargetContent.LegalHold = strings.ToUpper(session.Header.CommandStringFlags[lhFlag])
					if cpURLs.TargetContent.LegalHold != "" {
						cpURLs.TargetContent.LegalHoldEnabled = true
					}
				} else {
					if rm := cli.String(rmFlag); rm != "" {
						cpURLs.TargetContent.RetentionMode = rm
						cpURLs.TargetContent.RetentionEnabled = true
					}
					if rd := cli.String(rdFlag); rd != "" {
						cpURLs.TargetContent.RetentionDuration = rd
					}
					if lh := cli.String(lhFlag); lh != "" {
						cpURLs.TargetContent.LegalHold = strings.ToUpper(lh)
						cpURLs.TargetContent.LegalHoldEnabled = true
					}
				}

				preserve := cli.Bool("preserve")
				if cli.String("attr") != "" {
					preserve = true
					userMetaMap, _ := getMetaDataEntry(cli.String("attr"))
					for metadataKey, metaDataVal := range userMetaMap {
						cpURLs.TargetContent.UserMetadata[metadataKey] = metaDataVal
					}
				}

				cpURLs.MD5 = cli.Bool("md5") || withLock
				cpURLs.DisableMultipart = cli.Bool("disable-multipart")

				// Verify if previously copied, notify progress bar.
				if isCopied != nil && isCopied(cpURLs.SourceContent.URL.String()) {
					queueCh <- func() URLs {
						return doCopyFake(ctx, cpURLs, pg, isMvCmd)
					}
				} else {
					queueCh <- func() URLs {
						return doCopy(ctx, cpURLs, pg, encKeyDB, isMvCmd, preserve)
					}
				}
			}
		}
	}()

	var retErr error
	errSeen := false
	cpAllFilesErr := true

loop:
	for {
		select {
		case <-globalContext.Done():
			close(quitCh)
			cancelCopy()
			// Receive interrupt notification.
			if !globalQuiet && !globalJSON {
				console.Eraseline()
			}
			if session != nil {
				session.CloseAndDie()
			}
			break loop
		case cpURLs, ok := <-statusCh:
			// Status channel is closed, we should return.
			if !ok {
				break loop
			}
			if cpURLs.Error == nil {
				if session != nil {
					session.Header.LastCopied = cpURLs.SourceContent.URL.String()
					session.Save()
				}
				cpAllFilesErr = false
			} else {

				// Set exit status for any copy error
				retErr = exitStatus(globalErrorExitStatus)

				// Print in new line and adjust to top so that we
				// don't print over the ongoing progress bar.
				if !globalQuiet && !globalJSON {
					console.Eraseline()
				}
				errorIf(cpURLs.Error.Trace(cpURLs.SourceContent.URL.String()),
					fmt.Sprintf("Failed to copy `%s`.", cpURLs.SourceContent.URL.String()))
				if isErrIgnored(cpURLs.Error) {
					cpAllFilesErr = false
					continue loop
				}

				errSeen = true
				if progressReader, pgok := pg.(*progressBar); pgok {
					if progressReader.ProgressBar.Get() > 0 {
						writeContSize := (int)(cpURLs.SourceContent.Size)
						totalPGSize := (int)(progressReader.ProgressBar.Total)
						written := (int)(progressReader.ProgressBar.Get())
						if totalPGSize > writeContSize && written > writeContSize {
							progressReader.ProgressBar.Set((written - writeContSize))
							progressReader.ProgressBar.Update()
						}
					}
				}

				if session != nil {
					// For critical errors we should exit. Session
					// can be resumed after the user figures out
					// the  problem.
					session.copyCloseAndDie(session.Header.CommandBoolFlags["session"])
				}
			}
		}
	}

	if progressReader, ok := pg.(*progressBar); ok {
		if (errSeen && totalObjects == 1) || (cpAllFilesErr && totalObjects > 1) {
			console.Eraseline()
		} else if progressReader.ProgressBar.Get() > 0 {
			progressReader.ProgressBar.Finish()
		}
	} else {
		if accntReader, ok := pg.(*accounter); ok {
			printMsg(accntReader.Stat())
		}
	}

	return retErr
}

// validate the passed metadataString and populate the map
func getMetaDataEntry(metadataString string) (map[string]string, *probe.Error) {
	metaDataMap := make(map[string]string)
	r := strings.NewReader(metadataString)

	type pToken int
	const (
		KEY pToken = iota
		VALUE
	)

	type pState int
	const (
		NORMAL pState = iota
		QSTRING
		DQSTRING
	)

	var key, value strings.Builder

	writeRune := func(ch rune, pt pToken) {
		if pt == KEY {
			key.WriteRune(ch)
		} else if pt == VALUE {
			value.WriteRune(ch)
		} else {
			panic("Invalid parser token type")
		}
	}

	ps := NORMAL
	pt := KEY
	p := 0

	for ; ; p++ {
		ch, _, err := r.ReadRune()
		if err != nil {
			//eof
			if ps == QSTRING || ps == DQSTRING || pt == KEY {
				return nil, probe.NewError(ErrInvalidMetadata)
			}
			metaDataMap[http.CanonicalHeaderKey(key.String())] = value.String()
			return metaDataMap, nil
		}

		if ch == '"' {
			if ps == DQSTRING {
				ps = NORMAL
			} else if ps == QSTRING {
				writeRune(ch, pt)
			} else if ps == NORMAL {
				ps = DQSTRING
			} else {
				break
			}
			continue
		}

		if ch == '\'' {
			if ps == QSTRING {
				ps = NORMAL
			} else if ps == DQSTRING {
				writeRune(ch, pt)
			} else if ps == NORMAL {
				ps = QSTRING
			} else {
				break
			}
			continue
		}

		if ch == '=' {
			if ps == QSTRING || ps == DQSTRING {
				writeRune(ch, pt)
			} else if pt == KEY {
				pt = VALUE
			} else if pt == VALUE {
				writeRune(ch, pt)
			} else {
				break
			}
			continue
		}

		if ch == ';' {
			if ps == QSTRING || ps == DQSTRING {
				writeRune(ch, pt)
			} else if pt == KEY {
				return nil, probe.NewError(ErrInvalidMetadata)
			} else if pt == VALUE {
				metaDataMap[http.CanonicalHeaderKey(key.String())] = value.String()
				key.Reset()
				value.Reset()
				pt = KEY
			} else {
				break
			}
			continue
		}

		writeRune(ch, pt)
	}

	fatalErr := fmt.Sprintf("Invalid parser state at index: %d", p)
	panic(fatalErr)
}

// mainCopy is the entry point for cp command.
func mainCopy(cliCtx *cli.Context) error {
	ctx, cancelCopy := context.WithCancel(globalContext)
	defer cancelCopy()

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
	checkCopySyntax(ctx, cliCtx, encKeyDB, false)

	// Additional command specific theme customization.
	console.SetColor("Copy", color.New(color.FgGreen, color.Bold))

	recursive := cliCtx.Bool("recursive")
	rewind := cliCtx.String("rewind")
	versionID := cliCtx.String("version-id")
	olderThan := cliCtx.String("older-than")
	newerThan := cliCtx.String("newer-than")
	storageClass := cliCtx.String("storage-class")
	retentionMode := cliCtx.String(rmFlag)
	retentionDuration := cliCtx.String(rdFlag)
	legalHold := strings.ToUpper(cliCtx.String(lhFlag))
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
		sessionID := getHash("cp", cliCtx.Args())
		if isSessionExists(sessionID) {
			session, err = loadSessionV8(sessionID)
			fatalIf(err.Trace(sessionID), "Unable to load session.")
		} else {
			session = newSessionV8(sessionID)
			session.Header.CommandType = "cp"
			session.Header.CommandBoolFlags["recursive"] = recursive
			session.Header.CommandStringFlags["rewind"] = rewind
			session.Header.CommandStringFlags["version-id"] = versionID
			session.Header.CommandStringFlags["older-than"] = olderThan
			session.Header.CommandStringFlags["newer-than"] = newerThan
			session.Header.CommandStringFlags["storage-class"] = storageClass
			session.Header.CommandStringFlags[rmFlag] = retentionMode
			session.Header.CommandStringFlags[rdFlag] = retentionDuration
			session.Header.CommandStringFlags[lhFlag] = legalHold
			session.Header.CommandStringFlags["encrypt-key"] = sseKeys
			session.Header.CommandStringFlags["encrypt"] = sse
			session.Header.CommandBoolFlags["session"] = cliCtx.Bool("continue")

			if cliCtx.Bool("preserve") {
				session.Header.CommandBoolFlags["preserve"] = cliCtx.Bool("preserve")
			}
			session.Header.UserMetaData = userMetaMap
			session.Header.CommandBoolFlags["md5"] = cliCtx.Bool("md5")
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

	e := doCopySession(ctx, cancelCopy, cliCtx, session, encKeyDB, false)
	if session != nil {
		session.Delete()
	}

	return e
}
