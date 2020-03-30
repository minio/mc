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
	"errors"
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/cmd/ilm"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var ilmRemoveFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "id",
		Usage: "id of the lifecycle rule",
	},
	cli.BoolFlag{
		Name:  "force",
		Usage: "force flag is to be used when deleting all lifecycle configuration rules for the bucket",
	},
	cli.BoolFlag{
		Name:  "all",
		Usage: "delete all lifecycle configuration rules of the bucket, force flag enforced",
	},
}

var ilmRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove (if any) existing lifecycle configuration rule with the id",
	Action: mainILMRemove,
	Before: setGlobalsFromContext,
	Flags:  append(ilmRemoveFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

DESCRIPTION:
  Remove the lifecycle configuration rule for the bucket with the ID or all configurations only if specified (--all --force).


EXAMPLES:
  1. Remove the lifecycle management configuration rule given by ID "Documents" for testbucket on alias s3. ID is case sensitive.
     {{.Prompt}} {{.HelpName}} --id "Documents" s3/testbucket
  2. Remove ALL the lifecycle management configuration rules for testbucket on alias s3. Because the result is complete removal, the use of --force flag is enforced.
     {{.Prompt}} {{.HelpName}} --all --force s3/testbucket


`,
}

type ilmRmMessage struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Target string `json:"target"`
	All    bool   `json:"all"`
}

// tagSetMessage console colorized output.
func (i ilmRmMessage) String() string {
	msg := "Rule ID `" + i.ID + "` from target " + i.Target + " removed."
	if i.All {
		msg = "Rules for `" + i.Target + "` removed."
	}
	return console.Colorize(ilmThemeResultSuccess, msg)
}

// JSON tagSetMessage.
func (i ilmRmMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func checkILMRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "remove")
		os.Exit(globalErrorExitStatus)
	}
	ilmAll := ctx.Bool("all")
	ilmForce := ctx.Bool("force")
	forceChk := (ilmAll && ilmForce) || (!ilmAll && !ilmForce)
	if !forceChk {
		fatalIf(probe.NewError(errors.New("Flag missing or wrong")),
			"Mandatory to enter --all and --force flag together for mc "+ctx.Command.FullName()+".")
	}
	if ilmAll && ilmForce {
		return
	}
	ilmID := ctx.String("id")
	if ilmID == "" {
		fatalIf(probe.NewError(errors.New("ID of lifecycle rule missing")), "Please provide id.")
	}
}

func ilmAllRemove(urlStr string) error {
	if err := setBucketILMConfiguration(urlStr, ""); err != nil {
		return err
	}
	return nil
}

func ilmIDRemove(ilmID string, urlStr string) error {
	var lfcInfoXML string
	var err error
	if lfcInfoXML, err = getBucketILMConfiguration(urlStr); err != nil {
		return err
	}
	if lfcInfoXML == "" {
		return errors.New("Lifecycle configuration for `" + urlStr + "` not set")
	}
	if lfcInfoXML, err = ilm.RemoveILMRule(lfcInfoXML, ilmID); err != nil {
		return err
	}
	if err = setBucketILMConfiguration(urlStr, lfcInfoXML); err != nil {
		return err
	}

	return nil
}

func mainILMRemove(ctx *cli.Context) error {
	checkILMRemoveSyntax(ctx)
	setILMDisplayColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	var err error
	var ilmAll, ilmForce bool
	var ilmID string
	ilmAll = ctx.Bool("all")
	ilmForce = ctx.Bool("force")
	if ilmAll && ilmForce {
		err = ilmAllRemove(objectURL)
		fatalIf(probe.NewError(err), "Failed to remove all rules for `"+objectURL+"`.")
		printMsg(ilmRmMessage{
			Status: "success",
			ID:     ilmID,
			All:    true,
			Target: objectURL,
		})
		return nil
	}
	if ilmID = ctx.String("id"); ilmID == "" {
		fatalIf(probe.NewError(errors.New("ID not provided")),
			"Failed to remove lifecycle rule")
	}
	err = ilmIDRemove(ilmID, objectURL)
	fatalIf(probe.NewError(err), "Failed to remove rule. ID `"+ilmID+"` Target "+objectURL+". ")
	printMsg(ilmRmMessage{
		Status: "success",
		ID:     ilmID,
		Target: objectURL,
		All:    false,
	})
	return nil
}
