// Copyright (c) 2015-2023 MinIO, Inc.
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
	"net/http"
	"net/url"
	"os"
	"slices"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/pkg/v3/console"
	"golang.org/x/term"
)

var idpLdapAccesskeyCreateWithLoginFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "ldap-username",
		Usage: "username to login as (prompt if empty)",
	},
	cli.StringFlag{
		Name:  "ldap-password",
		Usage: "password for ldap-user (prompt if empty)",
	},
}

var idpLdapAccesskeyCreateWithLoginCmd = cli.Command{
	Name:         "create-with-login",
	Usage:        "login using LDAP credentials to generate access key pair",
	Action:       mainIDPLdapAccesskeyCreateWithLogin,
	Before:       setGlobalsFromContext,
	Flags:        slices.Concat(idpLdapAccesskeyCreateWithLoginFlags, idpLdapAccesskeyCreateFlags, globalFlags),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] URL

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a new access key pair for https://minio.example.com by logging in with LDAP credentials
     {{.Prompt}} {{.HelpName}} https://minio.example.com

  2. Create a new access key pair for http://localhost:9000 via login with custom access key and secret key 
     {{.Prompt}} {{.HelpName}} http://localhost:9000 --access-key myaccesskey --secret-key mysecretkey
`,
}

func mainIDPLdapAccesskeyCreateWithLogin(ctx *cli.Context) error {
	if !ctx.Args().Present() {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
	if !isTerminal {
		e := fmt.Errorf("login flag cannot be used with a non-interactive terminal")
		fatalIf(probe.NewError(e), "unable to read from STDIN")
	}

	client, opts := loginLDAPAccesskey(ctx)

	res, e := client.AddServiceAccountLDAP(globalContext, opts)
	fatalIf(probe.NewError(e), "unable to add service account")

	m := accesskeyMessage{
		op:          "create",
		Status:      "success",
		AccessKey:   res.AccessKey,
		SecretKey:   res.SecretKey,
		Expiration:  &res.Expiration,
		Name:        opts.Name,
		Description: opts.Description,
	}
	printMsg(m)

	return nil
}

func loginLDAPAccesskey(ctx *cli.Context) (*madmin.AdminClient, madmin.AddServiceAccountReq) {
	urlStr := ctx.Args().First()

	u, e := url.Parse(urlStr)
	fatalIf(probe.NewError(e), "unable to parse server URL")

	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	reader := bufio.NewReader(os.Stdin)

	username := ctx.String("ldap-username")
	if username == "" {
		fmt.Printf("%s", console.Colorize(cred, "Enter LDAP Username: "))
		value, _, e := reader.ReadLine()
		fatalIf(probe.NewError(e), "unable to read username")
		username = string(value)
	}

	password := ctx.String("ldap-password")
	if password == "" {
		fmt.Printf("%s", console.Colorize(cred, "Enter LDAP Password: "))
		bytePassword, e := term.ReadPassword(int(os.Stdin.Fd()))
		fatalIf(probe.NewError(e), "unable to read password")
		fmt.Printf("\n")
		password = string(bytePassword)
	}

	stsCreds, e := credentials.NewLDAPIdentity(urlStr, username, password)
	fatalIf(probe.NewError(e), "unable to initialize LDAP identity")

	tempCreds, e := stsCreds.GetWithContext(&credentials.CredContext{
		Client: http.DefaultClient,
	})
	fatalIf(probe.NewError(e), "unable to create a temporary account from LDAP identity")

	client, e := madmin.NewWithOptions(u.Host, &madmin.Options{
		Creds:  stsCreds,
		Secure: u.Scheme == "https",
	})
	fatalIf(probe.NewError(e), "unable to initialize admin connection")

	return client, accessKeyCreateOpts(ctx, tempCreds.AccessKeyID)
}
