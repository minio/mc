/*
 * Minio Client (C) 2019 MinIO, Inc.
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
	"bufio"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/minio/cli"
	"github.com/minio/gokrb5/v7/config"
	cr "github.com/minio/minio-go/v6/pkg/credentials"
)

var (
	krbFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "krb-config",
			Value: "/etc/krb5.conf",
			Usage: "path to Kerberos client configuration",
		},
		cli.StringFlag{
			Name:  "principal",
			Usage: "Minio service principal (by default \"minio/URL_HOST@REALM\")",
		},
		cli.StringFlag{
			Name:  "realm",
			Usage: "Realm to use if not the default from Kerberos client configuration",
		},
	}

	configStsKrbAddCmd = cli.Command{
		Name:            "add",
		Usage:           "Add Kerberos STS based Minio instance access",
		Action:          configStsKrbAdd,
		Before:          setGlobalsFromContext,
		Flags:           append(krbFlags, globalFlags...),
		HideHelpCommand: true,
		CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS URL USERNAME [PASSWORD]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add access using Kerberos.
     $ {{.HelpName}} myminio http://minio.intranet.com:9001 mojojojo secretpw

`,
	}
)

func checkStsKerberosSyntax(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 3 && len(args) != 4 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for sts kerberos command.")
	}

	alias := args.Get(0)
	url := args.Get(1)
	if !isValidAlias(alias) {
		fatalIf(errInvalidAlias(alias), "Invalid alias.")
	}

	if !isValidHostURL(url) {
		fatalIf(errInvalidURL(url), "Invalid URL.")
	}
}

func configStsKrbAdd(ctx *cli.Context) error {
	checkStsKerberosSyntax(ctx)

	var (
		args     = ctx.Args()
		alias    = args.Get(0)
		endpoint = trimTrailingSeparator(args.Get(1))
		username = args.Get(2)
		password = args.Get(3)
	)

	epURL := newClientURL(endpoint)
	if epURL.Type != objectStorage {
		log.Fatalf("Endpoint must be a network resource")
	}

	krbConfigFile := "/etc/krb5.conf"
	krbConfOpt := ctx.String("krb-config")
	if krbConfOpt != "" {
		krbConfigFile = krbConfOpt
	}
	krbConfig := loadKrbConfig(krbConfigFile)

	realm := ctx.String("realm")
	if realm == "" {
		realm = krbConfig.LibDefaults.DefaultRealm
	}

	userPrincipal := username

	servicePrincipal := ctx.String("principal")
	if servicePrincipal == "" {
		servicePrincipal = fmt.Sprintf("minio/%s", epURL.Host)
	}

	if password == "" {
		password = promptForPassword()
	}

	ki, err := cr.NewKerberosIdentity(endpoint, krbConfig, userPrincipal, password, realm, servicePrincipal)
	if err != nil {
		log.Fatalf("Kerberos Login error: %v", err)
	}

	v, err := ki.Get()
	if err != nil {
		log.Fatalf("Error getting credentials after login: %v", err)
	}

	mcCfg, err2 := loadMcConfig()
	fatalIf(err2.Trace(globalMCConfigVersion), "Unable to load config `"+mustGetMcConfigPath()+"`.")

	mcCfg.Hosts[alias] = mcHostConfig{
		URL:          endpoint,
		AccessKey:    v.AccessKeyID,
		SecretKey:    v.SecretAccessKey,
		API:          v.SignerType.String(),
		Lookup:       "auto",
		SessionToken: v.SessionToken,
	}
	err2 = saveMcConfig(mcCfg)
	fatalIf(err2.Trace(alias), "Unable to update hosts in config version `"+mustGetMcConfigPath()+"`.")

	return nil
}

func loadKrbConfig(krbConfigFile string) *config.Config {
	cfg, err := config.Load(krbConfigFile)
	if err != nil {
		log.Fatalf("Error loading Kerberos client configuration file (%s): %v", krbConfigFile, err)
	}
	return cfg
}

func promptForPassword() string {
	f := bufio.NewWriter(os.Stdout)
	f.Write([]byte("Enter Password: "))
	f.Flush()
	passwordBytes, err := terminal.ReadPassword(0)
	if err != nil {
		log.Fatalf("Error reading password!")
	}
	return string(passwordBytes)
}
