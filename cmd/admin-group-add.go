/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminGroupAddCmd = cli.Command{
	Name:   "add",
	Usage:  "add users to a new or existing group",
	Action: mainAdminGroupAdd,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET GROUPNAME MEMBERS...

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add users 'fivecent' and 'tencent' to the group 'allcents':
     {{.Prompt}} {{.HelpName}} myminio allcents fivecent tencent
`,
}

// checkAdminGroupAddSyntax - validate all the passed arguments
func checkAdminGroupAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) < 3 {
		cli.ShowCommandHelpAndExit(ctx, "add", 1) // last argument is exit code
	}
}

// groupMessage container for content message structure
type groupMessage struct {
	op          string
	Status      string   `json:"status"`
	GroupName   string   `json:"groupName,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	Members     []string `json:"members,omitempty"`
	GroupStatus string   `json:"groupStatus,omitempty"`
	GroupPolicy string   `json:"groupPolicy,omitempty"`
}

func (u groupMessage) String() string {
	switch u.op {
	case "list":
		var s []string
		for _, g := range u.Groups {
			s = append(s, console.Colorize("GroupMessage", g))
		}
		return strings.Join(s, "\n")
	case "disable":
		return console.Colorize("GroupMessage", "Disabled group `"+u.GroupName+"` successfully.")
	case "enable":
		return console.Colorize("GroupMessage", "Enabled group `"+u.GroupName+"` successfully.")
	case "add":
		membersStr := fmt.Sprintf("{%s}", strings.Join(u.Members, ","))
		return console.Colorize("GroupMessage", "Added members "+membersStr+" to group "+u.GroupName+" successfully.")
	case "remove":
		if len(u.Members) > 0 {
			membersStr := fmt.Sprintf("{%s}", strings.Join(u.Members, ","))
			return console.Colorize("GroupMessage", "Removed members "+membersStr+" from group "+u.GroupName+" successfully.")
		}
		return console.Colorize("GroupMessage", "Removed group "+u.GroupName+" successfully.")
	case "info":
		return strings.Join([]string{
			console.Colorize("GroupMessage", "Group: "+u.GroupName),
			console.Colorize("GroupMessage", "Status: "+u.GroupStatus),
			console.Colorize("GroupMessage", "Policy: "+u.GroupPolicy),
			console.Colorize("GroupMessage", "Members: "+strings.Join(u.Members, ",")),
		}, "\n")

	}
	return ""
}

func (u groupMessage) JSON() string {
	u.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// mainAdminGroupAdd is the handle for "mc admin group add" command.
func mainAdminGroupAdd(ctx *cli.Context) error {
	checkAdminGroupAddSyntax(ctx)

	console.SetColor("GroupMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	members := []string{}
	for i := 2; i < ctx.NArg(); i++ {
		members = append(members, args.Get(i))
	}
	gAddRemove := madmin.GroupAddRemove{
		Group:    args.Get(1),
		Members:  members,
		IsRemove: false,
	}
	fatalIf(probe.NewError(client.UpdateGroupMembers(globalContext, gAddRemove)).Trace(args...), "Unable to add new group")

	printMsg(groupMessage{
		op:        "add",
		GroupName: args.Get(1),
		Members:   members,
	})

	return nil
}
