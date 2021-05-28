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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"golang.org/x/crypto/ssh/terminal"
)

var adminUserAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a new user",
	Action:       mainAdminUserAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ACCESSKEY SECRETKEY

ACCESSKEY:
  Also called as username.

SECRETKEY:
  Also called as password.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add a new user 'foobar' to MinIO server.
     {{.DisableHistory}}
     {{.Prompt}} {{.HelpName}} myminio foobar foo12345
     {{.EnableHistory}}
  2. Add a new user 'foobar' to MinIO server, prompting for keys.
     {{.Prompt}} {{.HelpName}} myminio
     Enter Access Key: foobar
     Enter Secret Key: foobar12345
  3. Add a new user 'foobar' to MinIO server using piped keys.
     {{.DisableHistory}}
     {{.Prompt}} echo -e "foobar\nfoobar12345" | {{.HelpName}} myminio
     {{.EnableHistory}}
`,
}

// checkAdminUserAddSyntax - validate all the passed arguments
func checkAdminUserAddSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr > 3 || argsNr < 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for user add command.")
	}
}

// userMessage container for content message structure
type userMessage struct {
	op         string
	Status     string   `json:"status"` // TODO: remove this?
	AccessKey  string   `json:"accessKey,omitempty"`
	SecretKey  string   `json:"secretKey,omitempty"`
	PolicyName string   `json:"policyName,omitempty"`
	UserStatus string   `json:"userStatus,omitempty"`
	MemberOf   []string `json:"memberOf,omitempty"`
}

func (u userMessage) String() string {
	switch u.op {
	case "list":
		userFieldMaxLen := 9
		accessFieldMaxLen := 20
		policyFieldMaxLen := 20

		// Create a new pretty table with cols configuration
		return newPrettyTable("  ",
			Field{"UserStatus", userFieldMaxLen},
			Field{"AccessKey", accessFieldMaxLen},
			Field{"PolicyName", policyFieldMaxLen},
		).buildRow(u.UserStatus, u.AccessKey, u.PolicyName)
	case "info":
		return console.Colorize("UserMessage", strings.Join(
			[]string{
				fmt.Sprintf("AccessKey: %s", u.AccessKey),
				fmt.Sprintf("Status: %s", u.UserStatus),
				fmt.Sprintf("PolicyName: %s", u.PolicyName),
				fmt.Sprintf("MemberOf: %s", strings.Join(u.MemberOf, ",")),
			}, "\n"))
	case "remove":
		return console.Colorize("UserMessage", "Removed user `"+u.AccessKey+"` successfully.")
	case "disable":
		return console.Colorize("UserMessage", "Disabled user `"+u.AccessKey+"` successfully.")
	case "enable":
		return console.Colorize("UserMessage", "Enabled user `"+u.AccessKey+"` successfully.")
	case "add":
		return console.Colorize("UserMessage", "Added user `"+u.AccessKey+"` successfully.")
	}
	return ""
}

func (u userMessage) JSON() string {
	u.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// fetchUserKeys - returns the access and secret key
func fetchUserKeys(args cli.Args) (string, string) {
	accessKey := ""
	secretKey := ""
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	isTerminal := terminal.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	argCount := len(args)

	if argCount == 1 {
		if isTerminal {
			fmt.Printf("%s", console.Colorize(cred, "Enter Access Key: "))
		}
		value, _, _ := reader.ReadLine()
		accessKey = string(value)
	} else {
		accessKey = args.Get(1)
	}

	if argCount == 1 || argCount == 2 {
		if isTerminal {
			fmt.Printf("%s", console.Colorize(cred, "Enter Secret Key: "))
			bytePassword, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Printf("\n")
			secretKey = string(bytePassword)
		} else {
			value, _, _ := reader.ReadLine()
			secretKey = string(value)
		}
	} else {
		secretKey = args.Get(2)
	}

	return accessKey, secretKey
}

// mainAdminUserAdd is the handle for "mc admin user add" command.
func mainAdminUserAdd(ctx *cli.Context) error {
	checkAdminUserAddSyntax(ctx)

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	accessKey, secretKey := fetchUserKeys(args)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	fatalIf(probe.NewError(client.AddUser(globalContext, accessKey, secretKey)).Trace(args...), "Unable to add new user")

	printMsg(userMessage{
		op:         "add",
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		UserStatus: "enabled",
	})

	return nil
}
