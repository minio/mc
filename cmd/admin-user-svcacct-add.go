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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
	"github.com/minio/pkg/v3/policy"
)

var adminUserSvcAcctAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "access-key",
		Usage: "set an access key for the service account",
	},
	cli.StringFlag{
		Name:  "secret-key",
		Usage: "set a secret key for the service account",
	},
	cli.StringFlag{
		Name:  "policy",
		Usage: "path to a JSON policy file",
	},
	cli.StringFlag{
		Name:  "name",
		Usage: "friendly name for the service account",
	},
	cli.StringFlag{
		Name:  "description",
		Usage: "description for the service account",
	},
	cli.StringFlag{
		Name:   "comment",
		Hidden: true,
		Usage:  "description for the service account (DEPRECATED: use --description instead)",
	},
	cli.StringFlag{
		Name:  "expiry",
		Usage: "time of expiration for the service account",
	},
}

var adminUserSvcAcctAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a new service account",
	Action:       mainAdminUserSvcAcctAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminUserSvcAcctAddFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS ACCOUNT [FLAGS]

ACCOUNT:
  An account could be a regular MinIO user, STS or LDAP user.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add a new service account for user 'foobar' to MinIO server with a name and description.
     {{.Prompt}} {{.HelpName}} myminio foobar --name uploaderKey --description "foobar uploader scripts"

  2. Add a new service account to MinIO server with specified access key and secret key for user 'foobar'.
     {{.Prompt}} {{.HelpName}} myminio foobar --access-key "myaccesskey" --secret-key "mysecretkey"

  3. Add a new service account to MinIO server with specified access key and random secret key for user 'foobar'.
     {{.Prompt}} {{.HelpName}} myminio foobar --access-key "myaccesskey"

  4. Add a new service account to MinIO server with specified secret key and random access key for user 'foobar'.
     {{.Prompt}} {{.HelpName}} myminio foobar --secret-key "mysecretkey"

  5. Add a new service account to MinIO server with specified expiry date in the future for user 'foobar'.
     {{.Prompt}} {{.HelpName}} myminio foobar --expiry 2023-06-24T10:00:00-07:00
`,
}

// checkAdminUserSvcAcctAddSyntax - validate all the passed arguments
func checkAdminUserSvcAcctAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1)
	}
}

// acctMessage container for content message structure
type acctMessage struct {
	op            acctOp
	Status        string          `json:"status"`
	AccessKey     string          `json:"accessKey,omitempty"`
	SecretKey     string          `json:"secretKey,omitempty"`
	ParentUser    string          `json:"parentUser,omitempty"`
	ImpliedPolicy bool            `json:"impliedPolicy,omitempty"`
	Policy        json.RawMessage `json:"policy,omitempty"`
	Name          string          `json:"name,omitempty"`
	Description   string          `json:"description,omitempty"`
	AccountStatus string          `json:"accountStatus,omitempty"`
	MemberOf      []string        `json:"memberOf,omitempty"`
	Expiration    *time.Time      `json:"expiration,omitempty"`
}

const (
	accessFieldMaxLen = 20
)

type acctOp int

const (
	svcAccOpAdd = acctOp(iota)
	svcAccOpList
	svcAccOpInfo
	svcAccOpRemove
	svcAccOpDisable
	svcAccOpEnable
	svcAccOpSet

	stsAccOpInfo

	// Maximum length for MinIO access key.
	// There is no max length enforcement for access keys
	accessKeyMaxLen = 20

	// Maximum length for Expiration timestamp
	expirationMaxLen = 29

	// Alpha numeric table used for generating access keys.
	alphaNumericTable = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Total length of the alpha numeric table.
	alphaNumericTableLen = byte(len(alphaNumericTable))

	// Maximum secret key length for MinIO, this
	// is used when autogenerating new credentials.
	// There is no max length enforcement for secret keys
	secretKeyMaxLen = 40
)

var supportedTimeFormats = []string{
	"2006-01-02",
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	time.RFC3339,
}

func (u acctMessage) String() string {
	switch u.op {
	case svcAccOpList:
		// Create a new pretty table with cols configuration
		return newPrettyTable(" | ",
			Field{"AccessKey", accessFieldMaxLen},
			Field{"Expiration", expirationMaxLen},
		).buildRow(u.AccessKey, func() string {
			if u.Expiration != nil && !u.Expiration.IsZero() {
				return (*u.Expiration).String()
			}
			return "no-expiry"
		}())
	case stsAccOpInfo, svcAccOpInfo:
		policyField := ""
		if u.ImpliedPolicy {
			policyField = "implied"
		} else {
			policyField = "embedded"
		}
		return console.Colorize("AccMessage", strings.Join(
			[]string{
				fmt.Sprintf("AccessKey: %s", u.AccessKey),
				fmt.Sprintf("ParentUser: %s", u.ParentUser),
				fmt.Sprintf("Status: %s", u.AccountStatus),
				fmt.Sprintf("Name: %s", u.Name),
				fmt.Sprintf("Description: %s", u.Description),
				fmt.Sprintf("Policy: %s", policyField),
				func() string {
					if u.Expiration != nil {
						return fmt.Sprintf("Expiration: %s", humanize.Time(*u.Expiration))
					}
					return "Expiration: no-expiry"
				}(),
			}, "\n"))
	case svcAccOpRemove:
		return console.Colorize("AccMessage", "Removed service account `"+u.AccessKey+"` successfully.")
	case svcAccOpDisable:
		return console.Colorize("AccMessage", "Disabled service account `"+u.AccessKey+"` successfully.")
	case svcAccOpEnable:
		return console.Colorize("AccMessage", "Enabled service account `"+u.AccessKey+"` successfully.")
	case svcAccOpAdd:
		if u.Expiration != nil && !u.Expiration.IsZero() && !u.Expiration.Equal(timeSentinel) {
			return console.Colorize("AccMessage",
				fmt.Sprintf("Access Key: %s\nSecret Key: %s\nExpiration: %s", u.AccessKey, u.SecretKey, *u.Expiration))
		}
		return console.Colorize("AccMessage",
			fmt.Sprintf("Access Key: %s\nSecret Key: %s\nExpiration: no-expiry", u.AccessKey, u.SecretKey))
	case svcAccOpSet:
		return console.Colorize("AccMessage", "Edited service account `"+u.AccessKey+"` successfully.")
	}
	return ""
}

// generateCredentials - creates randomly generated credentials of maximum
// allowed length.
func generateCredentials() (accessKey, secretKey string, err *probe.Error) {
	readBytes := func(size int) (data []byte, e error) {
		data = make([]byte, size)
		var n int
		if n, e = rand.Read(data); e != nil {
			return nil, e
		} else if n != size {
			return nil, fmt.Errorf("Not enough data. Expected to read: %v bytes, got: %v bytes", size, n)
		}
		return data, nil
	}

	// Generate access key.
	keyBytes, e := readBytes(accessKeyMaxLen)
	if e != nil {
		return "", "", probe.NewError(e)
	}
	for i := range accessKeyMaxLen {
		keyBytes[i] = alphaNumericTable[keyBytes[i]%alphaNumericTableLen]
	}
	accessKey = string(keyBytes)

	// Generate secret key.
	keyBytes, e = readBytes(secretKeyMaxLen)
	if e != nil {
		return "", "", probe.NewError(e)
	}

	secretKey = strings.ReplaceAll(string([]byte(base64.StdEncoding.EncodeToString(keyBytes))[:secretKeyMaxLen]),
		"/", "+")

	return accessKey, secretKey, nil
}

func (u acctMessage) JSON() string {
	u.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// mainAdminUserSvcAcctAdd is the handle for "mc admin user svcacct add" command.
func mainAdminUserSvcAcctAdd(ctx *cli.Context) error {
	checkAdminUserSvcAcctAddSyntax(ctx)

	console.SetColor("AccMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	user := args.Get(1)

	accessKey := ctx.String("access-key")
	secretKey := ctx.String("secret-key")
	policyPath := ctx.String("policy")
	name := ctx.String("name")
	description := ctx.String("description")
	if description == "" {
		description = ctx.String("comment")
	}
	expiry := ctx.String("expiry")

	// generate access key and secret key
	if len(accessKey) <= 0 || len(secretKey) <= 0 {
		randomAccessKey, randomSecretKey, err := generateCredentials()
		if err != nil {
			fatalIf(err, "unable to generate randomized access credentials")
		}
		if len(accessKey) <= 0 {
			accessKey = randomAccessKey
		}
		if len(secretKey) <= 0 {
			secretKey = randomSecretKey
		}
	}

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	opts := madmin.AddServiceAccountReq{
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		Name:        name,
		Description: description,
		TargetUser:  user,
	}

	if policyPath != "" {
		// Validate the policy document and ensure it has at least when statement
		policyBytes, e := os.ReadFile(policyPath)
		fatalIf(probe.NewError(e), "unable to read the policy document")

		p, e := policy.ParseConfig(bytes.NewReader(policyBytes))
		fatalIf(probe.NewError(e), "unable to parse the policy document")

		if p.IsEmpty() {
			fatalIf(errInvalidArgument(), "empty policies are not allowed")
		}

		opts.Policy = policyBytes
	}

	if expiry != "" {
		location, e := time.LoadLocation("Local")
		fatalIf(probe.NewError(e), "unable to load local location. verify your local TZ=<val> settings")

		var found bool
		for _, format := range supportedTimeFormats {
			t, e := time.ParseInLocation(format, expiry, location)
			if e == nil {
				found = true
				opts.Expiration = &t
				break
			}
		}

		if !found {
			fatalIf(probe.NewError(fmt.Errorf("expiry argument is not matching any of the supported patterns")), "unable to parse the expiry argument")
		}
	}

	creds, e := client.AddServiceAccount(globalContext, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to add a new service account.")

	printMsg(acctMessage{
		op:            svcAccOpAdd,
		AccessKey:     creds.AccessKey,
		SecretKey:     creds.SecretKey,
		Expiration:    &creds.Expiration,
		AccountStatus: "enabled",
	})

	return nil
}
