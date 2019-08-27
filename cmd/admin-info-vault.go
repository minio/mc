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
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

const (
	vault         = "BoldGreen"
	vaultDegraded = "BoldYellow"
	vaultFail     = "BoldRed"
)

var adminInfoVault = cli.Command{
	Name:   "vault",
	Usage:  "display MinIO server cpu information",
	Action: mainAdminVaultInfo,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}
 
 USAGE:
   {{.HelpName}} TARGET
 
 FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
 EXAMPLES:
   1. Get server CPU information of the 'play' MinIO server.
		$ {{.HelpName}} play/
 
 `,
}

// serverMonitorMessage holds service status info along with
// cpu, mem, net and disk monitoristics
type serverVaultMessage struct {
	Monitor string            `json:"status"`
	Service string            `json:"service"`
	Addr    string            `json:"address"`
	Err     string            `json:"error"`
	Vault   *madmin.VaultInfo `json:"vault,omitempty"`
}

func (s serverVaultMessage) JSON() string {
	s.Monitor = "success"
	if s.Err != "" {
		s.Monitor = "fail"
	}
	statusJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON")

	return string(statusJSONBytes)
}

func (s serverVaultMessage) String() string {
	msg := ""
	dot := "●"
	msg += fmt.Sprintf("%s %s        \n", console.Colorize(vault, dot), console.Colorize(vault, " Vault Info"))
	msg += fmt.Sprintf("   EndPoint          : %s\n", s.Vault.Endpoint)
	msg += fmt.Sprintf("   Name              : %s\n", s.Vault.Name)
	msg += fmt.Sprintf("   Type              : %s\n", s.Vault.Type)

	if s.Err != "" || s.Service == "off" {
		msg += fmt.Sprintf("   Vault Status      : %s\n", console.Colorize(vaultFail, "offline"))
		msg += fmt.Sprintf("   Error             : %s\n", console.Colorize(vaultFail, s.Err))
		return msg
	}
	msg += fmt.Sprintf("   Vault Status      : %s\n", console.Colorize(vault, "online"))
	if s.Vault.Perm.EncryptionErr != "" {

		msg += fmt.Sprintf("   Encryption Error  : %s\n", console.Colorize(vaultFail, s.Vault.Perm.EncryptionErr))
	} else {
		msg += fmt.Sprintf("   Encryption        : %s\n", console.Colorize(vault, "OK"))
	}

	if s.Vault.Perm.DecryptionErr != "" {

		msg += fmt.Sprintf("   Decryption Error  : %s\n", console.Colorize(vaultFail, s.Vault.Perm.DecryptionErr))
	} else {
		msg += fmt.Sprintf("   Decryption        : %s\n", console.Colorize(vault, "OK"))
	}

	return msg
}
func mainAdminVaultInfo(ctx *cli.Context) error {
	checkAdminVaultInfoSyntax(ctx)

	// set the console colors
	console.SetColor(vault, color.New(color.FgGreen, color.Bold))
	console.SetColor(vaultDegraded, color.New(color.FgYellow, color.Bold))
	console.SetColor(vaultFail, color.New(color.FgRed, color.Bold))

	// Get the alias
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO admin client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	printOfflineErrorMessage := func(err error) {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		printMsg(serverVaultMessage{
			Addr:    aliasedURL,
			Service: "off",
			Err:     errMsg,
		})
	}

	processErr := func(e error) error {
		switch e.(type) {
		case *json.SyntaxError:
			printOfflineErrorMessage(e)
			return e
		case *url.Error:
			printOfflineErrorMessage(e)
			return e
		default:
			// If the error is not nil and unrecognized, just print it and exit
			fatalIf(probe.NewError(e), "Cannot get service status.")
		}
		return nil
	}

	vaultInfo, e := client.ServerVaultInfo()
	if err := processErr(e); err != nil {
		// exit immediately if error encountered
		return nil
	}

	if vaultInfo.Error != "" {
		printMsg(serverVaultMessage{
			Service: "off",
			Addr:    aliasedURL,
			Err:     vaultInfo.Error,
			Vault:   &vaultInfo,
		})
	} else {
		printMsg(serverVaultMessage{
			Service: "on",
			Addr:    aliasedURL,
			Vault:   &vaultInfo,
		})
	}
	return nil
}

// checkAdminMonitorSyntax - validate all the passed arguments
func checkAdminVaultInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {

		exit := globalErrorExitStatus
		cli.ShowCommandHelpAndExit(ctx, "vault", exit)
	}
}
