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
	"errors"
	"fmt"
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v6/pkg/credentials"
)

var stsLDAPCmd = cli.Command{
	Name:   "ldap",
	Usage:  "get credentials from LDAP",
	Action: mainSTSLDAP,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ldap_url=<URL> ldapuser=<USER> ldappassword=<PASSWORD>

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get credentials from LDAP server for 'play' alias of MinIO server.
     {{.Prompt}} {{.HelpName}} play/ ldap_url=ldap://myldapserver.com ldapuser=myldapuser ldappassword=myldappassword
`,
}

// stsLDAPMessage is container for ServerUpdate success and failure messages.
type stsLDAPMessage struct {
	Status          string `json:"status"`
	Alias           string `json:"-"`
	TargetURL       string `json:"-"`
	AccessKeyID     string `json:"accessKey"`
	SecretAccessKey string `json:"secretKey"`
	SessionToken    string `json:"sessionToken"`
	SignerType      string `json:"signer"`
}

// String colorized serverUpdate message.
func (s stsLDAPMessage) String() string {
	var m string

	m += "Environment variables for AWS SDKs\n"
	m += fmt.Sprintf("AWS_ACCESS_KEY_ID=%v\n", s.AccessKeyID)
	m += fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%v\n", s.SecretAccessKey)
	m += fmt.Sprintf("AWS_SESSION_TOKEN=%v\n\n", s.SessionToken)

	m += "Environment variables for mc\n"
	m += fmt.Sprintf("export MC_HOST_%v=http://%v:%v@%v/\n", s.Alias, s.AccessKeyID, s.SecretAccessKey, s.TargetURL)
	m += fmt.Sprintf("export MC_HOST_SESSION_TOKEN_%v=%v", s.Alias, s.SessionToken)

	return m
}

// JSON jsonified server update message.
func (s stsLDAPMessage) JSON() string {
	serverUpdateJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serverUpdateJSONBytes)
}

func mainSTSLDAP(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 4 {
		cli.ShowCommandHelpAndExit(ctx, "ldap", 1)
	}

	alias, targetURL, _, perr := expandAlias(args.Get(0))
	fatalIf(perr, "Unable to get target URL for alias.")

	ldapURL := args.Get(1)
	if !strings.HasPrefix(ldapURL, "ldap_url=") {
		cli.ShowCommandHelpAndExit(ctx, "ldap", 1)
	}
	ldapURL = strings.TrimPrefix(ldapURL, "ldap_url=")
	if ldapURL == "" {
		fatalIf(probe.NewError(errors.New("empty LDAP URL")), "'ldap_url' must be provided.")
	}

	ldapUser := args.Get(2)
	if !strings.HasPrefix(ldapUser, "ldapuser=") {
		cli.ShowCommandHelpAndExit(ctx, "ldap", 1)
	}
	ldapUser = strings.TrimPrefix(ldapUser, "ldapUser=")

	ldapPassword := args.Get(3)
	if !strings.HasPrefix(ldapPassword, "ldappassword=") {
		cli.ShowCommandHelpAndExit(ctx, "ldap", 1)
	}
	ldapPassword = strings.TrimPrefix(ldapPassword, "ldapPassword=")

	cred, err := credentials.NewLDAPIdentity(ldapURL, ldapUser, ldapPassword)
	fatalIf(probe.NewError(err), "Unable to create new LDAP identity.")

	value, err := cred.Get()
	fatalIf(probe.NewError(err), "Unable to get credentials value from LDAP identity.")

	printMsg(stsLDAPMessage{
		Status:          "success",
		Alias:           alias,
		TargetURL:       targetURL,
		AccessKeyID:     value.AccessKeyID,
		SecretAccessKey: value.SecretAccessKey,
		SessionToken:    value.SessionToken,
		SignerType:      value.SignerType.String(),
	})

	return nil
}
