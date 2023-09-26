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
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v2/console"
)

var idpLdapAccesskeyCreateFlags = []cli.Flag{
	cli.DurationFlag{
		Name:   "expiration, e",
		Usage:  "expiration for temporary access keys",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "accesskey",
		Usage:  "access key to create",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "secretkey",
		Usage:  "secret key to give the new access key",
		Hidden: true,
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
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  TODO: add examples	
	`,
}

type ldapCredentialsMessage struct {
	Status       string    `json:"status,omitempty"`
	AccessKey    string    `json:"accessKey,omitempty"`
	ParentUser   string    `json:"parentUser,omitempty"`
	SecretKey    string    `json:"secretKey,omitempty"`
	SessionToken string    `json:"sessionToken,omitempty"`
	Expiration   time.Time `json:"expiration,omitempty"`
}

func (m ldapCredentialsMessage) String() string {
	accessKey := m.AccessKey
	secretKey := m.SecretKey
	sessionToken := m.SessionToken
	expiration := m.Expiration
	expirationS := "NONE"
	if expiration == (time.Unix(0, 0)) {
		expirationS = expiration.Format(time.RFC3339)
	}

	return fmt.Sprintf("TODO: clean this\nAccess Key: %s\nSecret Key: %s\nSession Token: %s\nExpiration: %s\n", accessKey, secretKey, sessionToken, expirationS)
}

func (m ldapCredentialsMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func mainIDPLdapAccesskeyCreate(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	console.SetColor("Title", color.New(color.FgGreen))
	console.SetColor("AccessKey", color.New(color.FgCyan))
	console.SetColor("SecretKey", color.New(color.FgCyan))
	console.SetColor("Expiration", color.New(color.FgYellow))

	args := ctx.Args()
	aliasedURL := args.Get(0)

	expVal := ctx.Duration("expiration")
	exp := time.Now().Add(expVal)
	accessVal := ctx.String("accesskey")
	secretVal := ctx.String("secretkey")

	if expVal == 0 {
		exp = time.Unix(0, 0)
	}

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	accessKey, secretKey, e := generateCredentials()
	fatalIf(probe.NewError(e), "Unable to generate credentials.")

	// If access key and secret key are provided, use them instead
	if accessVal != "" {
		accessKey = accessVal
	}
	if secretVal != "" {
		secretKey = secretVal
	}

	parentUser, e := client.AccountInfo(globalContext, madmin.AccountOpts{})
	fatalIf(probe.NewError(e), "Unable to get account info.")

	res, e := client.AddServiceAccount(globalContext,
		madmin.AddServiceAccountReq{
			AccessKey:  accessKey,
			SecretKey:  secretKey,
			Expiration: &exp,
		})
	fatalIf(probe.NewError(e), "Unable to add service account.")

	m := ldapAccesskeyMessage{
		op:         "create",
		Status:     "success",
		AccessKey:  res.AccessKey,
		SecretKey:  res.SecretKey,
		Expiration: &res.Expiration,
		ParentUser: parentUser.AccountName,
	}

	printMsg(m)

	return nil
}
