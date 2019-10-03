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
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v6/pkg/credentials"
	"golang.org/x/oauth2"
)

var stsWebIdentityCmd = cli.Command{
	Name:   "web-identity",
	Usage:  "get credentials from web identity",
	Action: mainSTSWebIdentity,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET web_identity_url=<URL>

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get credentials from web identity server for 'play' alias of MinIO server.
     {{.Prompt}} {{.HelpName}} play/ web_identity_url=https://myweb-identity-server.com client_id=myclientid client_secret=myclientsecret port=65432
`,
}

// stsWebIdentityMessage is container for ServerUpdate success and failure messages.
type stsWebIdentityMessage struct {
	Status          string `json:"status"`
	Alias           string `json:"-"`
	TargetURL       string `json:"-"`
	AccessKeyID     string `json:"accessKey"`
	SecretAccessKey string `json:"secretKey"`
	SessionToken    string `json:"sessionToken"`
	SignerType      string `json:"signer"`
}

// String colorized serverUpdate message.
func (s stsWebIdentityMessage) String() string {
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
func (s stsWebIdentityMessage) JSON() string {
	serverUpdateJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serverUpdateJSONBytes)
}

// DiscoveryDoc - parses the output from openid-configuration
// for example https://accounts.google.com/.well-known/openid-configuration
type DiscoveryDoc struct {
	Issuer                           string   `json:"issuer,omitempty"`
	AuthEndpoint                     string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint                    string   `json:"token_endpoint,omitempty"`
	UserInfoEndpoint                 string   `json:"userinfo_endpoint,omitempty"`
	RevocationEndpoint               string   `json:"revocation_endpoint,omitempty"`
	JwksURI                          string   `json:"jwks_uri,omitempty"`
	ResponseTypesSupported           []string `json:"response_types_supported,omitempty"`
	SubjectTypesSupported            []string `json:"subject_types_supported,omitempty"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported,omitempty"`
	ScopesSupported                  []string `json:"scopes_supported,omitempty"`
	TokenEndpointAuthMethods         []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	ClaimsSupported                  []string `json:"claims_supported,omitempty"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported,omitempty"`
}

func parseDiscoveryDoc(webIdentityURL string) (*DiscoveryDoc, error) {
	req, err := http.NewRequest(http.MethodGet, webIdentityURL, nil)
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Transport: http.DefaultTransport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with %s", resp.Status)
	}
	doc := DiscoveryDoc{}
	if err = json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func mainSTSWebIdentity(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 5 {
		cli.ShowCommandHelpAndExit(ctx, "web-identity", 1)
	}

	alias, targetURL, _, perr := expandAlias(args.Get(0))
	fatalIf(perr, "Unable to get target URL for alias.")

	webIdentityURL := args.Get(1)
	if !strings.HasPrefix(webIdentityURL, "web_identity_url=") {
		cli.ShowCommandHelpAndExit(ctx, "web-identity", 1)
	}
	webIdentityURL = strings.TrimPrefix(webIdentityURL, "web_identity_url=")
	if webIdentityURL == "" {
		fatalIf(probe.NewError(errors.New("empty web identity URL")), "'web-identity_url' must be provided.")
	}

	clientID := args.Get(2)
	if !strings.HasPrefix(clientID, "clientID=") {
		cli.ShowCommandHelpAndExit(ctx, "web-identity", 1)
	}
	clientID = strings.TrimPrefix(clientID, "client_id=")

	clientSecret := args.Get(3)
	if !strings.HasPrefix(clientSecret, "client_secret=") {
		cli.ShowCommandHelpAndExit(ctx, "web-identity", 1)
	}
	clientSecret = strings.TrimPrefix(clientSecret, "client_secret=")

	port := args.Get(4)
	if !strings.HasPrefix(port, "port=") {
		cli.ShowCommandHelpAndExit(ctx, "web-identity", 1)
	}
	port = strings.TrimPrefix(port, "port=")

	doc, err := parseDiscoveryDoc(webIdentityURL)
	fatalIf(probe.NewError(err), "unable to get discovery document")

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  doc.AuthEndpoint,
			TokenURL: doc.TokenEndpoint,
		},
		RedirectURL: "http://localhost:" + port + "/oauth2/callback",
		Scopes:      []string{"openid"},
	}

	state := func() string {
		b := make([]byte, 32)
		rand.Read(b)
		return base64.RawURLEncoding.EncodeToString(b)
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	http.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		getWebTokenExpiry := func() (*credentials.WebIdentityToken, error) {
			oauth2Token, err := config.Exchange(context.Background(), r.URL.Query().Get("code"))
			if err != nil {
				return nil, err
			}
			if !oauth2Token.Valid() {
				return nil, errors.New("invalid token")
			}

			return &credentials.WebIdentityToken{
				Token:  oauth2Token.Extra("id_token").(string),
				Expiry: int(oauth2Token.Expiry.Sub(time.Now().UTC()).Seconds()),
			}, nil
		}

		cred, err := credentials.NewSTSWebIdentity(targetURL, getWebTokenExpiry)
		fatalIf(probe.NewError(err), "Unable to create new web identity.")

		value, err := cred.Get()
		fatalIf(probe.NewError(err), "Unable to get credentials value from web identity.")

		printMsg(stsWebIdentityMessage{
			Status:          "success",
			Alias:           alias,
			TargetURL:       targetURL,
			AccessKeyID:     value.AccessKeyID,
			SecretAccessKey: value.SecretAccessKey,
			SessionToken:    value.SessionToken,
			SignerType:      value.SignerType.String(),
		})
	})

	fmt.Println("Open URL http://localhost:" + port + " in your web browser to get web identity credentials here")
	fmt.Println("Once web identity credentials are printed, press Ctrl-C to terminate")
	http.ListenAndServe("localhost:"+port, nil)

	return nil
}
