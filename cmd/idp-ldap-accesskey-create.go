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
	"bytes"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/pkg/v2/console"
	"github.com/minio/pkg/v2/policy"
	"golang.org/x/term"
)

var idpLdapAccesskeyCreateFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "access-key",
		Usage: "set an access key for the account",
	},
	cli.StringFlag{
		Name:  "secret-key",
		Usage: "set a secret key for the  account",
	},
	cli.StringFlag{
		Name:  "policy",
		Usage: "path to a JSON policy file",
	},
	cli.StringFlag{
		Name:  "name",
		Usage: "friendly name for the account",
	},
	cli.StringFlag{
		Name:  "description",
		Usage: "description for the account",
	},
	cli.StringFlag{
		Name:  "expiry-duration",
		Usage: "duration before the access key expires",
	},
	cli.StringFlag{
		Name:  "expiry",
		Usage: "expiry date for the access key",
	},
	cli.BoolFlag{
		Name:  "login",
		Usage: "log in using ldap credentials to generate access key par for future use",
	},
}

var idpLdapAccesskeyCreateCmd = cli.Command{
	Name:         "create",
	Usage:        "create access key pairs for LDAP",
	Action:       mainIDPLdapAccesskeyCreate,
	Before:       setGlobalsFromContext,
	Flags:        append(idpLdapAccesskeyCreateFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] [TARGET]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Create a new access key pair with the same policy as the authenticated user
     {{.Prompt}} {{.HelpName}} local/
  2. Create a new access key pair with custom access key and secret key
	 {{.Prompt}} {{.HelpName}} local/ --access-key myaccesskey --secret-key mysecretkey
  4. Create a new access key pair for user with username "james" that expires in 1 day
	 {{.Prompt}} {{.HelpName}} james --expiry-duration 24h
  5. Create a new access key pair for authenticated user that expires on 2021-01-01
	 {{.Prompt}} {{.HelpName}} --expiry 2021-01-01
  6. Create a new access key pair for minio.example.com by logging in with LDAP credentials
	 {{.Prompt}} {{.HelpName}} --login minio.example.com
	`,
}

func mainIDPLdapAccesskeyCreate(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)
	targetUser := args.Get(1)

	login := ctx.Bool("login")
	accessVal := ctx.String("access-key")
	secretVal := ctx.String("secret-key")
	name := ctx.String("name")
	description := ctx.String("description")
	policyPath := ctx.String("policy")

	expDurVal := ctx.Duration("expiry-duration")
	expVal := ctx.String("expiry")
	if expVal != "" && expDurVal != 0 {
		e := fmt.Errorf("Only one of --expiry or --expiry-duration can be specified")
		fatalIf(probe.NewError(e), "Invalid flags.")
	}

	var exp time.Time
	if expVal != "" {
		location, e := time.LoadLocation("Local")
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to parse the expiry argument.")
		}

		patternMatched := false
		for _, format := range supportedTimeFormats {
			t, e := time.ParseInLocation(format, expVal, location)
			if e == nil {
				patternMatched = true
				exp = t
				break
			}
		}

		if !patternMatched {
			e := fmt.Errorf("invalid expiry date format '%s'", expVal)
			fatalIf(probe.NewError(e), "unable to parse the expiry argument.")
		}
	} else if expDurVal != 0 {
		exp = time.Now().Add(expDurVal)
	} else {
		exp = time.Unix(0, 0)
	}

	var policyBytes []byte
	if policyPath != "" {
		// Validate the policy document and ensure it has at least when statement
		var e error
		policyBytes, e = os.ReadFile(policyPath)
		fatalIf(probe.NewError(e), "Unable to open the policy document.")
		p, e := policy.ParseConfig(bytes.NewReader(policyBytes))
		fatalIf(probe.NewError(e), "Unable to parse the policy document.")
		if p.IsEmpty() {
			fatalIf(errInvalidArgument(), "Empty policy documents are not allowed.")
		}
	}

	var client *madmin.AdminClient

	// If login flag is set, use LDAP credentials to generate access key pair
	if login {
		if targetUser != "" {
			fatalIf(errInvalidArgument().Trace(targetUser), "login flag cannot be used with a target user")
		}
		isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
		if !isTerminal {
			e := fmt.Errorf("login flag cannot be used with non-interactive terminal")
			fatalIf(probe.NewError(e), "Invalid flags.")
		}

		// For login, aliasedURL is not aliased, but the actual server URL
		client = loginLDAPAccesskey(aliasedURL)
	} else {
		var err *probe.Error
		// If login flag is not set, continue normally
		client, err = newAdminClient(aliasedURL)
		fatalIf(err, "Unable to initialize admin connection.")
	}

	accessKey, secretKey, e := generateCredentials()
	fatalIf(probe.NewError(e), "Unable to generate credentials.")

	// If access key and secret key are provided, use them instead
	if accessVal != "" {
		accessKey = accessVal
	}
	if secretVal != "" {
		secretKey = secretVal
	}

	res, e := client.AddServiceAccountLDAP(globalContext,
		madmin.AddServiceAccountReq{
			Policy:      policyBytes,
			TargetUser:  targetUser,
			AccessKey:   accessKey,
			SecretKey:   secretKey,
			Name:        name,
			Description: description,
			Expiration:  &exp,
		})
	fatalIf(probe.NewError(e), "Unable to add service account.")

	m := ldapAccesskeyMessage{
		op:         "create",
		Status:     "success",
		AccessKey:  res.AccessKey,
		SecretKey:  res.SecretKey,
		Expiration: &res.Expiration,
	}
	printMsg(m)

	return nil
}

func loginLDAPAccesskey(URL string) *madmin.AdminClient {
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s", console.Colorize(cred, "Enter LDAP Username: "))
	value, _, e := reader.ReadLine()
	fatalIf(probe.NewError(e), "Unable to read username")
	username := string(value)

	fmt.Printf("%s", console.Colorize(cred, "Enter Password: "))
	bytePassword, e := term.ReadPassword(int(os.Stdin.Fd()))
	fatalIf(probe.NewError(e), "Unable to read password")
	fmt.Printf("\n")
	password := string(bytePassword)

	ldapID, e := credentials.NewLDAPIdentity(URL, username, password)
	fatalIf(probe.NewError(e), "Unable to initialize LDAP identity.")

	u, e := url.Parse(URL)
	fatalIf(probe.NewError(e), "Unable to parse server URL.")

	client, e := madmin.NewWithOptions(u.Host, &madmin.Options{
		Creds:  ldapID,
		Secure: u.Scheme == "https",
	})
	fatalIf(probe.NewError(e), "Unable to initialize admin connection.")

	return client
}
