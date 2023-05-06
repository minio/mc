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
	"github.com/minio/madmin-go/v2"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	iampolicy "github.com/minio/pkg/iam/policy"
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
		Name:  "comment",
		Usage: "personal note for the service account",
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
  {{.HelpName}} ALIAS ACCOUNT

ACCOUNT:
  An account could be a regular MinIO user, STS ou LDAP user.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add a new service account for user 'foobar' to MinIO server.
     {{.Prompt}} {{.HelpName}} myminio foobar
  2. Add a new service account using the specified access key and secret key for user 'foobar' to MinIO server.
     {{.Prompt}} {{.HelpName}} myminio foobar --access-key "myaccesskey" --secret-key "mysecretkey"
  3. Add a new service account to MinIO server with specified access key and random secret key for user'foobar'.
     {{.Prompt}} {{.HelpName}} myminio foobar --access-key "myaccesskey"
  4. Add a new service account to MinIO server with specified secret key and random access key for user'foobar'.
     {{.Prompt}} {{.HelpName}} myminio foobar --secret-key "mysecretkey"
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
	Comment       string          `json:"comment,omitempty"`
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

	// Alpha numeric table used for generating access keys.
	alphaNumericTable = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Total length of the alpha numeric table.
	alphaNumericTableLen = byte(len(alphaNumericTable))

	// Maximum secret key length for MinIO, this
	// is used when autogenerating new credentials.
	// There is no max length enforcement for secret keys
	secretKeyMaxLen = 40
)

func (u acctMessage) String() string {
	switch u.op {
	case svcAccOpList:
		// Create a new pretty table with cols configuration
		return newPrettyTable("  ",
			Field{"AccessKey", accessFieldMaxLen},
		).buildRow(u.AccessKey)
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
				fmt.Sprintf("Comment: %s", u.Comment),
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
		return console.Colorize("AccMessage",
			fmt.Sprintf("Access Key: %s\nSecret Key: %s", u.AccessKey, u.SecretKey))
	case svcAccOpSet:
		return console.Colorize("AccMessage", "Edited service account `"+u.AccessKey+"` successfully.")
	}
	return ""
}

// generateCredentials - creates randomly generated credentials of maximum
// allowed length.
func generateCredentials() (accessKey, secretKey string, err error) {
	readBytes := func(size int) (data []byte, err error) {
		data = make([]byte, size)
		var n int
		if n, err = rand.Read(data); err != nil {
			return nil, err
		} else if n != size {
			return nil, fmt.Errorf("Not enough data. Expected to read: %v bytes, got: %v bytes", size, n)
		}
		return data, nil
	}

	// Generate access key.
	keyBytes, err := readBytes(accessKeyMaxLen)
	if err != nil {
		return "", "", err
	}
	for i := 0; i < accessKeyMaxLen; i++ {
		keyBytes[i] = alphaNumericTable[keyBytes[i]%alphaNumericTableLen]
	}
	accessKey = string(keyBytes)

	// Generate secret key.
	keyBytes, err = readBytes(secretKeyMaxLen)
	if err != nil {
		return "", "", err
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
	comment := ctx.String("comment")

	// generate access key and secret key
	if len(accessKey) <= 0 || len(secretKey) <= 0 {
		randomAccessKey, randomSecretKey, err := generateCredentials()
		if err != nil {
			fatalIf(probe.NewError(err), "Unable to add a new service account")
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

	var policyBytes []byte
	if policyPath != "" {
		// Validate the policy document and ensure it has at least when statement
		var e error
		policyBytes, e = os.ReadFile(policyPath)
		fatalIf(probe.NewError(e), "Unable to open the policy document.")
		p, e := iampolicy.ParseConfig(bytes.NewReader(policyBytes))
		fatalIf(probe.NewError(e), "Unable to parse the policy document.")
		if p.IsEmpty() {
			fatalIf(errInvalidArgument(), "Empty policy documents are not allowed.")
		}
	}

	opts := madmin.AddServiceAccountReq{
		Policy:     policyBytes,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		Comment:    comment,
		TargetUser: user,
	}

	creds, e := client.AddServiceAccount(globalContext, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to add a new service account")

	printMsg(acctMessage{
		op:            svcAccOpAdd,
		AccessKey:     creds.AccessKey,
		SecretKey:     creds.SecretKey,
		AccountStatus: "enabled",
	})

	return nil
}
