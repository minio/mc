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
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/v3/console"
)

const (
	licRegisterMsgTag  = "licenseRegisterMessage"
	licRegisterLinkTag = "licenseRegisterLink"
)

var licenseRegisterFlags = append([]cli.Flag{
	cli.StringFlag{
		Name:  "name",
		Usage: "Specify the name to associate to this MinIO cluster in SUBNET",
	},
	cli.StringFlag{
		Name:  "license",
		Usage: "license of the account on SUBNET",
	},
}, subnetCommonFlags...)

var licenseRegisterCmd = cli.Command{
	Name:         "register",
	Usage:        "register with MinIO Subscription Network",
	OnUsageError: onUsageError,
	Action:       mainLicenseRegister,
	Before:       setGlobalsFromContext,
	Flags:        licenseRegisterFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Register MinIO cluster at alias 'play' on SUBNET, using api key for auth
     {{.Prompt}} {{.HelpName}} play --api-key 08efc836-4289-dbd4-ad82-b5e8b6d25577

  2. Register MinIO cluster at alias 'play' on SUBNET, using license file ./minio.license
     {{.Prompt}} {{.HelpName}} play --license ./minio.license

  3. Register MinIO cluster at alias 'play' on SUBNET, using api key for auth,
     and "play-cluster" as the preferred name for the cluster on SUBNET.
     {{.Prompt}} {{.HelpName}} play --api-key 08efc836-4289-dbd4-ad82-b5e8b6d25577 --name play-cluster

  4. Register MinIO cluster at alias 'play' on SUBNET in an airgapped environment
     {{.Prompt}} {{.HelpName}} play --airgap

  5. Register MinIO cluster at alias 'play' on SUBNET, using alias as the cluster name.
     This asks for SUBNET credentials if the cluster is not already registered.
     {{.Prompt}} {{.HelpName}} play
`,
}

type licRegisterMessage struct {
	Status string `json:"status"`
	Alias  string `json:"-"`
	Action string `json:"action,omitempty"`
	Type   string `json:"type"`
	URL    string `json:"url,omitempty"`
}

// String colorized license register message
func (li licRegisterMessage) String() string {
	var msg string
	switch li.Type {
	case "online":
		msg = console.Colorize(licRegisterMsgTag, fmt.Sprintf("%s %s successfully.", li.Alias, li.Action))
	case "offline":
		msg = fmt.Sprintln("Open the following URL in the browser to register", li.Alias, "on SUBNET:")
		msg = console.Colorize(licRegisterMsgTag, msg) + console.Colorize(licRegisterLinkTag, li.URL)
	}
	return msg
}

// JSON jsonified license register message
func (li licRegisterMessage) JSON() string {
	jsonBytes, e := json.MarshalIndent(li, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

// checkLicenseRegisterSyntax - validate arguments passed by a user
func checkLicenseRegisterSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// ClusterRegistrationReq - JSON payload of the subnet api for cluster registration
// Contains a registration token created by base64 encoding  of the registration info
type ClusterRegistrationReq struct {
	Token string `json:"token"`
}

// ClusterRegistrationInfo - Information stored in the cluster registration token
type ClusterRegistrationInfo struct {
	DeploymentID string      `json:"deployment_id"`
	ClusterName  string      `json:"cluster_name"`
	UsedCapacity uint64      `json:"used_capacity"`
	Info         ClusterInfo `json:"info"`
}

// ClusterInfo - The "info" sub-node of the cluster registration information struct
// Intended to be extensible i.e. more fields will be added as and when required
type ClusterInfo struct {
	MinioVersion    string `json:"minio_version"`
	NoOfServerPools int    `json:"no_of_server_pools"`
	NoOfServers     int    `json:"no_of_servers"`
	NoOfDrives      int    `json:"no_of_drives"`
	NoOfBuckets     uint64 `json:"no_of_buckets"`
	NoOfObjects     uint64 `json:"no_of_objects"`
	TotalDriveSpace uint64 `json:"total_drive_space"`
	UsedDriveSpace  uint64 `json:"used_drive_space"`
}

// SubnetLoginReq - JSON payload of the SUBNET login api
type SubnetLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SubnetMFAReq - JSON payload of the SUBNET mfa api
type SubnetMFAReq struct {
	Username string `json:"username"`
	OTP      string `json:"otp"`
	Token    string `json:"token"`
}

func isPlay(endpoint url.URL) (bool, error) {
	playEndpoint := "https://play.min.io"
	if globalAirgapped {
		return endpoint.String() == playEndpoint, nil
	}

	aliasIPs, e := net.LookupHost(endpoint.Hostname())
	if e != nil {
		return false, e
	}
	aliasIPSet := set.CreateStringSet(aliasIPs...)

	playURL, e := url.Parse(playEndpoint)
	if e != nil {
		return false, e
	}

	playIPs, e := net.LookupHost(playURL.Hostname())
	if e != nil {
		return false, e
	}

	playIPSet := set.CreateStringSet(playIPs...)
	return !aliasIPSet.Intersection(playIPSet).IsEmpty(), nil
}

func validateNotPlay(aliasedURL string) {
	client := getClient(aliasedURL)
	endpoint := client.GetEndpointURL()
	if endpoint == nil {
		fatal(errDummy().Trace(), "invalid endpoint on alias "+aliasedURL)
		return
	}

	isplay, e := isPlay(*endpoint)
	fatalIf(probe.NewError(e), "error checking if endpoint is play:")

	if isplay {
		fatal(errDummy().Trace(), "play is a public demo cluster; cannot be registered")
	}
}

func mainLicenseRegister(ctx *cli.Context) error {
	console.SetColor(licRegisterMsgTag, color.New(color.FgGreen, color.Bold))
	console.SetColor(licRegisterLinkTag, color.New(color.FgWhite, color.Bold))
	checkLicenseRegisterSyntax(ctx)

	// Get the alias parameter from cli
	aliasedURL := ctx.Args().Get(0)
	validateNotPlay(aliasedURL)

	licFile := ctx.String("license")

	var alias, accAPIKey string
	if len(licFile) > 0 {
		licBytes, e := os.ReadFile(licFile)
		fatalIf(probe.NewError(e), fmt.Sprintf("Unable to read license file %s", licFile))
		alias, _ = url2Alias(aliasedURL)
		accAPIKey = validateAndSaveLic(string(licBytes), alias, true)
	} else {
		alias, accAPIKey = initSubnetConnectivity(ctx, aliasedURL, false)
	}

	clusterName := ctx.String("name")
	if len(clusterName) == 0 {
		clusterName = alias
	} else {
		if globalAirgapped {
			fatalIf(errInvalidArgument(), "'--name' is not allowed in airgapped mode")
		}
	}

	regInfo := GetClusterRegInfo(getAdminInfo(aliasedURL), clusterName)

	lrm := licRegisterMessage{Status: "success", Alias: alias}
	if !globalAirgapped {
		alreadyRegistered := false
		if len(accAPIKey) == 0 {
			apiKey, _, e := getSubnetCreds(alias)
			fatalIf(probe.NewError(e), "Error in fetching subnet API Key")
			if len(apiKey) > 0 {
				alreadyRegistered = true
				accAPIKey = apiKey
			}
		} else {
			apiKey := getSubnetAPIKeyFromConfig(alias)
			if len(apiKey) > 0 {
				alreadyRegistered = true
			}
		}

		lrm.Type = "online"
		_, _, e := registerClusterOnSubnet(regInfo, alias, accAPIKey)
		if e == nil {
			lrm.Action = "registered"
			if alreadyRegistered {
				lrm.Action = "updated"
			}
			printMsg(lrm)
			return nil
		}

		console.Println("Could not register cluster with SUBNET: ", e.Error())
	}

	// Airgapped mode OR online mode with registration failure
	lrm.Type = "offline"

	regToken, e := generateRegToken(regInfo)
	fatalIf(probe.NewError(e), "Unable to generate registration token")

	lrm.URL = subnetOfflineRegisterURL(regToken)
	printMsg(lrm)
	return nil
}

func getAdminInfo(aliasedURL string) madmin.InfoMessage {
	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Fetch info of all servers (cluster or single server)
	admInfo, e := client.ServerInfo(globalContext)
	fatalIf(probe.NewError(e), "Unable to fetch cluster info")

	return admInfo
}
