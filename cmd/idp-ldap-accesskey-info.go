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
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
)

var idpLdapAccesskeyInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "info about given access key pairs for LDAP",
	Action:       mainIDPLdapAccesskeyInfo,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET ACCESSKEY [ACCESSKEY...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get info for the access key "testkey"
	 {{.Prompt}} {{.HelpName}} local/ testkey
  2. Get info for the access keys "testkey" and "testkey2"
	 {{.Prompt}} {{.HelpName}} local/ testkey testkey2
	`,
}

type ldapAccesskeyMessage struct {
	op            string
	Status        string          `json:"status"`
	AccessKey     string          `json:"accessKey"`
	SecretKey     string          `json:"secretKey,omitempty"`
	ParentUser    string          `json:"parentUser,omitempty"`
	AccountStatus string          `json:"accountStatus,omitempty"`
	ImpliedPolicy bool            `json:"impliedPolicy,omitempty"`
	Policy        json.RawMessage `json:"policy,omitempty"`
	Name          string          `json:"name,omitempty"`
	Description   string          `json:"description,omitempty"`
	Expiration    *time.Time      `json:"expiration,omitempty"`
}

func (m ldapAccesskeyMessage) String() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")) // green
	o := strings.Builder{}
	switch m.op {
	case "info":
		expirationStr := "NONE"
		if m.Expiration != nil && !m.Expiration.IsZero() && !m.Expiration.Equal(timeSentinel) {
			expirationStr = humanize.Time(*m.Expiration)
		}
		policyStr := "embedded"
		if m.ImpliedPolicy {
			policyStr = "implied"
		}
		statusStr := "enabled"
		if m.AccountStatus == "off" {
			statusStr = "disabled"
		}
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Access Key:"), m.AccessKey))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Parent User:"), m.ParentUser))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Status:"), statusStr))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Policy:"), policyStr))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Name:"), m.Name))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Description:"), m.Description))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Expiration:"), expirationStr))
	case "create":
		expirationStr := "NONE"
		if m.Expiration != nil && !m.Expiration.IsZero() && !m.Expiration.Equal(timeSentinel) {
			expirationStr = m.Expiration.String()
		}
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Access Key:"), m.AccessKey))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Secret Key:"), m.SecretKey))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Expiration:"), expirationStr))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Name:"), m.Name))
		o.WriteString(iFmt(0, "%s %s\n", labelStyle.Render("Description:"), m.Description))
	case "remove":
		o.WriteString(labelStyle.Render(iFmt(0, "Successfully removed access key `%s`.", m.AccessKey)))
	case "edit":
		o.WriteString(labelStyle.Render(iFmt(0, "Successfully edited access key `%s`.", m.AccessKey)))
	case "enable":
		o.WriteString(labelStyle.Render(iFmt(0, "Successfully enabled access key `%s`.", m.AccessKey)))
	case "disable":
		o.WriteString(labelStyle.Render(iFmt(0, "Successfully disabled access key `%s`.", m.AccessKey)))
	}
	return o.String()
}

func (m ldapAccesskeyMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

func mainIDPLdapAccesskeyInfo(ctx *cli.Context) error {
	return commonAccesskeyInfo(ctx)
}

// currently no difference between ldap and builtin accesskey info
func commonAccesskeyInfo(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)
	accessKeys := args.Tail()

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	for _, accessKey := range accessKeys {
		// Assume service account by default
		res, e := client.InfoServiceAccount(globalContext, accessKey)
		if e != nil {
			// If not a service account must be sts
			tempRes, e := client.TemporaryAccountInfo(globalContext, accessKey)
			if e != nil {
				errorIf(probe.NewError(e), "Unable to retrieve access key %s info.", accessKey)
			} else {
				m := ldapAccesskeyMessage{
					op:            "info",
					AccessKey:     accessKey,
					Status:        "success",
					ParentUser:    tempRes.ParentUser,
					AccountStatus: tempRes.AccountStatus,
					ImpliedPolicy: tempRes.ImpliedPolicy,
					Policy:        json.RawMessage(tempRes.Policy),
					Name:          tempRes.Name,
					Description:   tempRes.Description,
					Expiration:    nilExpiry(tempRes.Expiration),
				}
				printMsg(m)
			}
		} else {
			m := ldapAccesskeyMessage{
				op:            "info",
				AccessKey:     accessKey,
				Status:        "success",
				ParentUser:    res.ParentUser,
				AccountStatus: res.AccountStatus,
				ImpliedPolicy: res.ImpliedPolicy,
				Policy:        json.RawMessage(res.Policy),
				Name:          res.Name,
				Description:   res.Description,
				Expiration:    nilExpiry(res.Expiration),
			}
			printMsg(m)
		}
	}

	return nil
}

func nilExpiry(expiry *time.Time) *time.Time {
	if expiry != nil && expiry.Equal(timeSentinel) {
		return nil
	}
	return expiry
}
