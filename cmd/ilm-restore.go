// Copyright (c) 2015-2022 MinIO, Inc.
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
	"time"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
)

// ilm restore specific flags.
var (
	ilmRestoreFlags = []cli.Flag{
		cli.IntFlag{
			Name:  "days",
			Value: 1,
			Usage: "keep the restored copy for N days",
		},
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "apply recursively",
		},
		cli.BoolFlag{
			Name:  "versions",
			Usage: "apply on versions",
		},
		cli.StringFlag{
			Name:  "version-id, vid",
			Usage: "select a specific version id",
		},
	}
)

var ilmRestoreCmd = cli.Command{
	Name:         "restore",
	Usage:        "restore archived objects",
	Action:       mainILMRestore,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(append(ilmRestoreFlags, encCFlag), globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Restore a copy of one or more objects from its remote tier. This copy automatically expires
  after the specified number of days (Default 1 day).

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Restore one specific object
     {{.Prompt}} {{.HelpName}} myminio/mybucket/path/to/object

  2. Restore a specific object version
     {{.Prompt}} {{.HelpName}} --vid "CL3sWgdSN2pNntSf6UnZAuh2kcu8E8si" myminio/mybucket/path/to/object

  3. Restore all objects under a specific prefix
     {{.Prompt}} {{.HelpName}} --recursive myminio/mybucket/dir/

  4. Restore all objects with all versions under a specific prefix
     {{.Prompt}} {{.HelpName}} --recursive --versions myminio/mybucket/dir/

  5. Restore an SSE-C encrypted object.
     {{.Prompt}} {{.HelpName}} --enc-c "myminio/mybucket/=MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA" myminio/mybucket/myobject.txt
`,
}

// checkILMRestoreSyntax - validate arguments passed by user
func checkILMRestoreSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, globalErrorExitStatus)
	}

	if ctx.Int("days") <= 0 {
		fatalIf(errDummy().Trace(), "--days should be equal or greater than 1")
	}

	if ctx.Bool("version-id") && (ctx.Bool("recursive") || ctx.Bool("versions")) {
		fatalIf(errDummy().Trace(), "You cannot combine --version-id with --recursive or --versions flags.")
	}
}

// Send Restore S3 API
func restoreObject(ctx context.Context, targetAlias, targetURL, versionID string, days int) *probe.Error {
	clnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		return err
	}

	return clnt.Restore(ctx, versionID, days)
}

// Send restore S3 API request to one or more objects depending on the arguments
func sendRestoreRequests(ctx context.Context, targetAlias, targetURL, targetVersionID string, recursive, applyOnVersions bool, days int, restoreSentReq chan *probe.Error) {
	defer close(restoreSentReq)

	client, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		restoreSentReq <- err
		return
	}

	if !recursive {
		err := restoreObject(ctx, targetAlias, targetURL, targetVersionID, days)
		restoreSentReq <- err
		return
	}

	prev := ""
	for content := range client.List(ctx, ListOptions{
		Recursive:         true,
		WithOlderVersions: applyOnVersions,
		ShowDir:           DirNone,
	}) {
		if content.Err != nil {
			errorIf(content.Err.Trace(client.GetURL().String()), "Unable to list folder.")
			continue
		}
		err := restoreObject(ctx, targetAlias, content.URL.String(), content.VersionID, days)
		if err != nil {
			restoreSentReq <- err
			continue
		}
		// Avoid sending the status of each separate version
		// of the same object name.
		if prev != content.URL.String() {
			prev = content.URL.String()
			restoreSentReq <- nil
		}
	}
}

// Wait until an object which receives restore request is completely restored in the fast tier
func waitRestoreObject(ctx context.Context, targetAlias, targetURL, versionID string, encKeyDB map[string][]prefixSSEPair) *probe.Error {
	clnt, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		return err
	}

	for {
		opts := StatOptions{
			versionID: versionID,
			sse:       getSSE(targetAlias+clnt.GetURL().Path, encKeyDB[targetAlias]),
		}
		st, err := clnt.Stat(ctx, opts)
		if err != nil {
			return err
		}
		if st.Restore == nil {
			return probe.NewError(fmt.Errorf("`%s` did not receive restore request", targetURL))
		}
		if st.Restore != nil && !st.Restore.OngoingRestore {
			return nil
		}
		// Restore still going on, wait for 5 seconds before checking again
		time.Sleep(5 * time.Second)
	}
}

// Check and wait the restore status of one or more objects one by one.
func checkRestoreStatus(ctx context.Context, targetAlias, targetURL, targetVersionID string, recursive, applyOnVersions bool, encKeyDB map[string][]prefixSSEPair, restoreStatus chan *probe.Error) {
	defer close(restoreStatus)

	client, err := newClientFromAlias(targetAlias, targetURL)
	if err != nil {
		restoreStatus <- err
		return
	}

	if !recursive {
		restoreStatus <- waitRestoreObject(ctx, targetAlias, targetURL, targetVersionID, encKeyDB)
		return
	}

	prev := ""
	for content := range client.List(ctx, ListOptions{
		Recursive:         true,
		WithOlderVersions: applyOnVersions,
		ShowDir:           DirNone,
	}) {
		if content.Err != nil {
			restoreStatus <- content.Err
			continue
		}

		err := waitRestoreObject(ctx, targetAlias, content.URL.String(), content.VersionID, encKeyDB)
		if err != nil {
			restoreStatus <- err
			continue
		}

		if prev != content.URL.String() {
			prev = content.URL.String()
			restoreStatus <- nil
		}
	}
}

var dotCycle = 0

// Clear and print text in the same line
func printStatus(msg string, args ...any) {
	if globalJSON {
		return
	}

	dotCycle++
	dots := bytes.Repeat([]byte{'.'}, dotCycle%3+1)
	fmt.Print("\n\033[1A\033[K")
	fmt.Printf(msg+string(dots), args...)
}

// Receive restore request & restore finished status and print in the console
func showRestoreStatus(restoreReqStatus, restoreFinishedStatus chan *probe.Error, doneCh chan struct{}) {
	var sent, finished int
	var done bool

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for !done {
		select {
		case err, ok := <-restoreReqStatus:
			if !ok {
				done = true
				break
			}
			errorIf(err.Trace(), "Unable to send restore request.")
			if err == nil {
				sent++
			}
		case <-ticker.C:
		}

		printStatus("Sent restore requests to %d object(s)", sent)
	}

	if !globalJSON {
		fmt.Println("")
	}

	done = false

	for !done {
		select {
		case err, ok := <-restoreFinishedStatus:
			if !ok {
				done = true
				break
			}
			errorIf(err.Trace(), "Unable to check for restore status")
			if err == nil {
				finished++
			}
		case <-ticker.C:
		}
		printStatus("%d/%d object(s) successfully restored", finished, sent)
	}

	if !globalJSON {
		fmt.Println("")
	} else {
		type ilmRestore struct {
			Status   string `json:"status"`
			Restored int    `json:"restored"`
		}

		msgBytes, _ := json.Marshal(ilmRestore{Status: "success", Restored: sent})
		fmt.Println(string(msgBytes))
	}

	close(doneCh)
}

func mainILMRestore(cliCtx *cli.Context) (cErr error) {
	ctx, cancelILMRestore := context.WithCancel(globalContext)
	defer cancelILMRestore()

	checkILMRestoreSyntax(cliCtx)

	args := cliCtx.Args()
	aliasedURL := args.Get(0)

	versionID := cliCtx.String("version-id")
	recursive := cliCtx.Bool("recursive")
	includeVersions := cliCtx.Bool("versions")
	days := cliCtx.Int("days")

	encKeyDB, err := validateAndCreateEncryptionKeys(cliCtx)
	fatalIf(err, "Unable to parse encryption keys.")

	targetAlias, targetURL, _ := mustExpandAlias(aliasedURL)
	if targetAlias == "" {
		fatalIf(errDummy().Trace(), "Unable to restore the given URL")
	}

	restoreReqStatus := make(chan *probe.Error)
	restoreStatus := make(chan *probe.Error)

	done := make(chan struct{})

	go func() {
		showRestoreStatus(restoreReqStatus, restoreStatus, done)
	}()

	sendRestoreRequests(ctx, targetAlias, targetURL, versionID, recursive, includeVersions, days, restoreReqStatus)
	checkRestoreStatus(ctx, targetAlias, targetURL, versionID, recursive, includeVersions, encKeyDB, restoreStatus)

	// Wait until the UI printed all the status
	<-done

	return nil
}
