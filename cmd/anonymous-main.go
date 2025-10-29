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
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var anonymousFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "list recursively",
	},
}

// Manage anonymous access to buckets and objects.
var anonymousCmd = cli.Command{
	Name:         "anonymous",
	Usage:        "manage anonymous access to buckets and objects",
	Action:       mainAnonymous,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(anonymousFlags, globalFlags...),
	CustomHelpTemplate: `Name:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] set PERMISSION TARGET
  {{.HelpName}} [FLAGS] set-json FILE TARGET
  {{.HelpName}} [FLAGS] get TARGET
  {{.HelpName}} [FLAGS] get-json TARGET
  {{.HelpName}} [FLAGS] list TARGET
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
PERMISSION:
  Allowed policies are: [private, public, download, upload].

FILE:
  A valid S3 anonymous JSON filepath.

EXAMPLES:
  1. Set bucket to "download" on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} set download s3/mybucket

  2. Set bucket to "public" on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} set public s3/shared

  3. Set bucket to "upload" on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} set upload s3/incoming

  4. Set anonymous to "public" for bucket with prefix on Amazon S3 cloud storage.
     {{.Prompt}} {{.HelpName}} set public s3/public-commons/images

  5. Set a custom prefix based bucket anonymous on Amazon S3 cloud storage using a JSON file.
     {{.Prompt}} {{.HelpName}} set-json /path/to/anonymous.json s3/public-commons/images 

  6. Get bucket permissions.
     {{.Prompt}} {{.HelpName}} get s3/shared

  7. Get bucket permissions in JSON format.
     {{.Prompt}} {{.HelpName}} get-json s3/shared

  8. List policies set to a specified bucket.
     {{.Prompt}} {{.HelpName}} list s3/shared

  9. List public object URLs recursively.
     {{.Prompt}} {{.HelpName}} --recursive links s3/shared/
`,
}

// anonymousRules contains anonymous rule
type anonymousRules struct {
	Resource string `json:"resource"`
	Allow    string `json:"allow"`
}

// String colorized access message.
func (s anonymousRules) String() string {
	return console.Colorize("Anonymous", s.Resource+" => "+s.Allow+"")
}

// JSON jsonified anonymous message.
func (s anonymousRules) JSON() string {
	anonymousJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(anonymousJSONBytes)
}

// anonymousMessage is container for anonymous command on bucket success and failure messages.
type anonymousMessage struct {
	Operation string         `json:"operation"`
	Status    string         `json:"status"`
	Bucket    string         `json:"bucket"`
	Perms     accessPerms    `json:"permission"`
	Anonymous map[string]any `json:"anonymous,omitempty"`
}

// String colorized access message.
func (s anonymousMessage) String() string {
	if s.Operation == "set" {
		return console.Colorize("Anonymous",
			"Access permission for `"+s.Bucket+"` is set to `"+string(s.Perms)+"`")
	}
	if s.Operation == "get" {
		return console.Colorize("Anonymous",
			"Access permission for `"+s.Bucket+"`"+" is `"+string(s.Perms)+"`")
	}
	if s.Operation == "set-json" {
		return console.Colorize("Anonymous",
			"Access permission for `"+s.Bucket+"`"+" is set from `"+string(s.Perms)+"`")
	}
	if s.Operation == "get-json" {
		anonymous, e := json.MarshalIndent(s.Anonymous, "", " ")
		fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
		return string(anonymous)
	}
	// nothing to print
	return ""
}

// JSON jsonified anonymous message.
func (s anonymousMessage) JSON() string {
	anonymousJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(anonymousJSONBytes)
}

// anonymousLinksMessage is container for anonymous links command
type anonymousLinksMessage struct {
	Status string `json:"status"`
	URL    string `json:"url"`
}

// String colorized access message.
func (s anonymousLinksMessage) String() string {
	return console.Colorize("Anonymous", s.URL)
}

// JSON jsonified anonymous message.
func (s anonymousLinksMessage) JSON() string {
	anonymousJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(anonymousJSONBytes)
}

// checkAnonymousSyntax check for incoming syntax.
func checkAnonymousSyntax(ctx *cli.Context) {
	argsLength := len(ctx.Args())
	// Always print a help message when we have extra arguments
	if argsLength > 3 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code.
	}
	// Always print a help message when no arguments specified
	if argsLength < 1 {
		showCommandHelpAndExit(ctx, 1)
	}

	firstArg := ctx.Args().Get(0)
	secondArg := ctx.Args().Get(1)

	// More syntax checking
	switch accessPerms(firstArg) {
	case "set":
		// Always expect three arguments when setting a anonymous permission.
		if argsLength != 3 {
			showCommandHelpAndExit(ctx, 1)
		}
		if accessPerms(secondArg) != accessNone &&
			accessPerms(secondArg) != accessDownload &&
			accessPerms(secondArg) != accessUpload &&
			accessPerms(secondArg) != accessPrivate &&
			accessPerms(secondArg) != accessPublic {
			fatalIf(errDummy().Trace(),
				"Unrecognized permission `"+secondArg+"`. Allowed values are [private, public, download, upload].")
		}

	case "set-json":
		// Always expect three arguments when setting a anonymous permission.
		if argsLength != 3 {
			showCommandHelpAndExit(ctx, 1)
		}
	case "get", "get-json":
		// get or get-json always expects two arguments
		if argsLength != 2 {
			showCommandHelpAndExit(ctx, 1)
		}
	case "list":
		// Always expect an argument after list cmd
		if argsLength != 2 {
			showCommandHelpAndExit(ctx, 1)
		}
	case "links":
		// Always expect an argument after links cmd
		if argsLength != 2 {
			showCommandHelpAndExit(ctx, 1)
		}
	default:
		showCommandHelpAndExit(ctx, 1)
	}
}

// Convert an accessPerms to a string recognizable by minio-go
func accessPermToString(perm accessPerms) string {
	anonymous := ""
	switch perm {
	case accessNone, accessPrivate:
		anonymous = "none"
	case accessDownload:
		anonymous = "readonly"
	case accessUpload:
		anonymous = "writeonly"
	case accessPublic:
		anonymous = "readwrite"
	case accessCustom:
		anonymous = "custom"
	}
	return anonymous
}

// doSetAccess do set access.
func doSetAccess(ctx context.Context, targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	anonymous := accessPermToString(targetPERMS)
	if err = clnt.SetAccess(ctx, anonymous, false); err != nil {
		return err.Trace(targetURL, string(targetPERMS))
	}
	return nil
}

// doSetAccessJSON do set access JSON.
func doSetAccessJSON(ctx context.Context, targetURL string, targetPERMS accessPerms) *probe.Error {
	clnt, err := newClient(targetURL)
	if err != nil {
		return err.Trace(targetURL)
	}
	fileReader, e := os.Open(string(targetPERMS))
	if e != nil {
		fatalIf(probe.NewError(e).Trace(), "Unable to set anonymous for `"+targetURL+"`.")
	}
	defer fileReader.Close()

	const maxJSONSize = 120 * 1024 // 120KiB
	configBuf := make([]byte, maxJSONSize+1)

	n, e := io.ReadFull(fileReader, configBuf)
	if e == nil {
		return probe.NewError(bytes.ErrTooLarge).Trace(targetURL)
	}
	if e != io.ErrUnexpectedEOF {
		return probe.NewError(e).Trace(targetURL)
	}

	configBytes := configBuf[:n]
	if err = clnt.SetAccess(ctx, string(configBytes), true); err != nil {
		return err.Trace(targetURL, string(targetPERMS))
	}
	return nil
}

// Convert a minio-go permission to accessPerms type
func stringToAccessPerm(perm string) accessPerms {
	var anonymous accessPerms
	switch perm {
	case "none":
		anonymous = accessPrivate
	case "readonly":
		anonymous = accessDownload
	case "writeonly":
		anonymous = accessUpload
	case "readwrite":
		anonymous = accessPublic
	case "private":
		anonymous = accessPrivate
	case "custom":
		anonymous = accessCustom
	}
	return anonymous
}

// doGetAccess do get access.
func doGetAccess(ctx context.Context, targetURL string) (perms accessPerms, anonymousStr string, err *probe.Error) {
	clnt, err := newClient(targetURL)
	if err != nil {
		return "", "", err.Trace(targetURL)
	}
	perm, anonymousJSON, err := clnt.GetAccess(ctx)
	if err != nil {
		return "", "", err.Trace(targetURL)
	}
	return stringToAccessPerm(perm), anonymousJSON, nil
}

// doGetAccessRules do get access rules.
func doGetAccessRules(ctx context.Context, targetURL string) (r map[string]string, err *probe.Error) {
	clnt, err := newClient(targetURL)
	if err != nil {
		return map[string]string{}, err.Trace(targetURL)
	}
	return clnt.GetAccessRules(ctx)
}

// Run anonymous list command
func runAnonymousListCmd(args cli.Args) {
	ctx, cancelAnonymousList := context.WithCancel(globalContext)
	defer cancelAnonymousList()

	targetURL := args.First()
	policies, err := doGetAccessRules(ctx, targetURL)
	if err != nil {
		switch err.ToGoError().(type) {
		case APINotImplemented:
			fatalIf(err.Trace(), "Unable to list policies of a non S3 url `"+targetURL+"`.")
		default:
			fatalIf(err.Trace(targetURL), "Unable to list policies of target `"+targetURL+"`.")
		}
	}
	for k, v := range policies {
		printMsg(anonymousRules{Resource: k, Allow: v})
	}
}

// Run anonymous links command
func runAnonymousLinksCmd(args cli.Args, recursive bool) {
	ctx, cancelAnonymousLinks := context.WithCancel(globalContext)
	defer cancelAnonymousLinks()

	// Get alias/bucket/prefix argument
	targetURL := args.First()

	// Fetch all policies associated to the passed url
	policies, err := doGetAccessRules(ctx, targetURL)
	if err != nil {
		switch err.ToGoError().(type) {
		case APINotImplemented:
			fatalIf(err.Trace(), "Unable to list policies of a non S3 url `"+targetURL+"`.")
		default:
			fatalIf(err.Trace(targetURL), "Unable to list policies of target `"+targetURL+"`.")
		}
	}

	// Extract alias from the passed argument, we'll need it to
	// construct new pathes to list public objects
	alias, path := url2Alias(targetURL)

	// Iterate over anonymous rules to fetch public urls, then search
	// for objects under those urls
	for k, v := range policies {
		// Trim the asterisk in anonymous rules
		anonymousPath := strings.TrimSuffix(k, "*")
		// Check if current anonymous prefix is related to the url passed by the user
		if !strings.HasPrefix(anonymousPath, path) {
			continue
		}
		// Check if the found anonymous has read permission
		perm := stringToAccessPerm(v)
		if perm != accessDownload && perm != accessPublic {
			continue
		}
		// Construct the new path to search for public objects
		newURL := alias + "/" + anonymousPath
		clnt, err := newClient(newURL)
		fatalIf(err.Trace(newURL), "Unable to initialize target `"+targetURL+"`.")
		// Search for public objects
		for content := range clnt.List(globalContext, ListOptions{Recursive: recursive, ShowDir: DirFirst}) {
			if content.Err != nil {
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			}

			if content.Type.IsDir() && recursive {
				continue
			}

			// Encode public URL
			u, e := url.Parse(content.URL.String())
			errorIf(probe.NewError(e), "Unable to parse url `%s`.", content.URL)
			publicURL := u.String()

			// Construct the message to be displayed to the user
			msg := anonymousLinksMessage{
				Status: "success",
				URL:    publicURL,
			}
			// Print the found object
			printMsg(msg)
		}
	}
}

// Run anonymous cmd to fetch set permission
func runAnonymousCmd(args cli.Args) {
	ctx, cancelAnonymous := context.WithCancel(globalContext)
	defer cancelAnonymous()

	var targetURL, anonymousStr string
	var perms accessPerms
	var probeErr *probe.Error

	operation := args.First()
	switch operation {
	case "set":
		perms = accessPerms(args.Get(1))
		if !perms.isValidAccessPERM() {
			fatalIf(errDummy().Trace(), "Invalid access permission: `"+string(perms)+"`.")
		}
		targetURL = args.Get(2)
		probeErr = doSetAccess(ctx, targetURL, perms)
		if probeErr == nil {
			perms, _, probeErr = doGetAccess(ctx, targetURL)
		}
	case "set-json":
		perms = accessPerms(args.Get(1))
		if !perms.isValidAccessFile() {
			fatalIf(errDummy().Trace(), "Invalid access file: `"+string(perms)+"`.")
		}
		targetURL = args.Get(2)
		probeErr = doSetAccessJSON(ctx, targetURL, perms)
	case "get", "get-json":
		targetURL = args.Get(1)
		perms, anonymousStr, probeErr = doGetAccess(ctx, targetURL)
	default:
		fatalIf(errDummy().Trace(), "Invalid operation: `"+operation+"`.")
	}
	// Upon error exit.
	if probeErr != nil {
		switch probeErr.ToGoError().(type) {
		case APINotImplemented:
			fatalIf(probeErr.Trace(), "Unable to "+operation+" anonymous of a non S3 url `"+targetURL+"`.")
		default:
			fatalIf(probeErr.Trace(targetURL, string(perms)),
				"Unable to "+operation+" anonymous `"+string(perms)+"` for `"+targetURL+"`.")
		}
	}
	anonymousJSON := map[string]any{}
	if anonymousStr != "" {
		e := json.Unmarshal([]byte(anonymousStr), &anonymousJSON)
		fatalIf(probe.NewError(e), "Unable to unmarshal custom anonymous file.")
	}
	printMsg(anonymousMessage{
		Status:    "success",
		Operation: operation,
		Bucket:    targetURL,
		Perms:     perms,
		Anonymous: anonymousJSON,
	})
}

func mainAnonymous(ctx *cli.Context) error {
	// check 'anonymous' cli arguments.
	checkAnonymousSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("Anonymous", color.New(color.FgGreen, color.Bold))

	switch ctx.Args().First() {
	case "set", "set-json", "get", "get-json":
		// anonymous set [private|public|download|upload] alias/bucket/prefix
		// anonymous set-json path-to-anonymous-json-file alias/bucket/prefix
		// anonymous get alias/bucket/prefix
		// anonymous get-json alias/bucket/prefix
		runAnonymousCmd(ctx.Args())
	case "list":
		// anonymous list alias/bucket/prefix
		runAnonymousListCmd(ctx.Args().Tail())
	case "links":
		// anonymous links alias/bucket/prefix
		runAnonymousLinksCmd(ctx.Args().Tail(), ctx.Bool("recursive"))
	default:
		// Shows command example and exit
		showCommandHelpAndExit(ctx, 1)
	}
	return nil
}
