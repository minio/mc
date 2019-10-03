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
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v6/pkg/credentials"
)

var stsClientGrantsCmd = cli.Command{
	Name:   "client-grants",
	Usage:  "get credentials from client grants",
	Action: mainSTSClientGrants,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET client_url=<URL> client_id=<ID> client_secret=<SECRET>

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get credentials from client grants for 'play' alias of MinIO server.
     {{.Prompt}} {{.HelpName}} play/ client_url=https://myclient-grantsserver.com client_id=myclientid client_secret=myclientsecret
`,
}

// stsClientGrantsMessage is container for ServerUpdate success and failure messages.
type stsClientGrantsMessage struct {
	Status          string `json:"status"`
	Alias           string `json:"-"`
	TargetURL       string `json:"-"`
	AccessKeyID     string `json:"accessKey"`
	SecretAccessKey string `json:"secretKey"`
	SessionToken    string `json:"sessionToken"`
	SignerType      string `json:"signer"`
}

// String colorized serverUpdate message.
func (s stsClientGrantsMessage) String() string {
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
func (s stsClientGrantsMessage) JSON() string {
	serverUpdateJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serverUpdateJSONBytes)
}

func mainSTSClientGrants(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 4 {
		cli.ShowCommandHelpAndExit(ctx, "client-grants", 1)
	}

	alias, targetURL, _, perr := expandAlias(args.Get(0))
	fatalIf(perr, "Unable to get target URL for alias.")

	clientURL := args.Get(1)
	if !strings.HasPrefix(clientURL, "client_url=") {
		cli.ShowCommandHelpAndExit(ctx, "client-grants", 1)
	}
	clientURL = strings.TrimPrefix(clientURL, "client_url=")
	if clientURL == "" {
		fatalIf(probe.NewError(errors.New("empty client grants URL")), "'client_url' must be provided.")
	}

	clientID := args.Get(2)
	if !strings.HasPrefix(clientID, "clientID=") {
		cli.ShowCommandHelpAndExit(ctx, "client-grants", 1)
	}
	clientID = strings.TrimPrefix(clientID, "client_id=")

	clientSecret := args.Get(3)
	if !strings.HasPrefix(clientSecret, "client_secret=") {
		cli.ShowCommandHelpAndExit(ctx, "client-grants", 1)
	}
	clientSecret = strings.TrimPrefix(clientSecret, "client_secret=")

	getTokenExpiry := func() (*credentials.ClientGrantsToken, error) {
		req, err := http.NewRequest(http.MethodPost, clientURL, strings.NewReader("grant_type=client_credentials"))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(clientID, clientSecret)
		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed with %s", resp.Status)
		}

		var idpToken struct {
			AccessToken string `json:"access_token"`
			Expiry      int    `json:"expires_in"`
		}

		if err = json.NewDecoder(resp.Body).Decode(&idpToken); err != nil {
			return nil, err
		}

		return &credentials.ClientGrantsToken{
			Token:  idpToken.AccessToken,
			Expiry: idpToken.Expiry,
		}, nil
	}

	cred, err := credentials.NewSTSClientGrants(targetURL, getTokenExpiry)
	fatalIf(probe.NewError(err), "Unable to create new ClientGrants identity.")

	value, err := cred.Get()
	fatalIf(probe.NewError(err), "Unable to get credentials value from ClientGrants identity.")

	printMsg(stsClientGrantsMessage{
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
