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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var supportRegisterFlags = append([]cli.Flag{
	cli.StringFlag{
		Name:  "name",
		Usage: "Specify the name to associate to this MinIO cluster in SUBNET",
	},
}, subnetCommonFlags...)

var supportRegisterCmd = cli.Command{
	Name:         "register",
	Usage:        "register with MinIO subscription network",
	OnUsageError: onUsageError,
	Action:       mainSupportRegister,
	Before:       setGlobalsFromContext,
	Flags:        append(supportRegisterFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Register MinIO cluster at alias 'play' on SUBNET, using alias as the cluster name.
     {{.Prompt}} {{.HelpName}} play

  2. Register MinIO cluster at alias 'play' on SUBNET, using the name "play-cluster".
     {{.Prompt}} {{.HelpName}} play --name play-cluster
`,
}

// checkSupportRegisterSyntax - validate arguments passed by a user
func checkSupportRegisterSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "register", 1) // last argument is exit code
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

func mainSupportRegister(ctx *cli.Context) error {
	checkSupportRegisterSyntax(ctx)

	// Get the alias parameter from cli
	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)

	offline := ctx.Bool("airgap") || ctx.Bool("offline")
	if !offline {
		fatalIf(checkURLReachable(subnetBaseURL()).Trace(aliasedURL), "Unable to reach %s register", subnetBaseURL())
	}

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Fetch info of all servers (cluster or single server)
	admInfo, e := client.ServerInfo(globalContext)
	fatalIf(probe.NewError(e), "Unable to fetch cluster info")

	clusterName := ctx.String("name")
	if len(clusterName) == 0 {
		clusterName = alias
	} else {
		if offline {
			fatalIf(errInvalidArgument(), "'--name' is not allowed in airgapped mode")
		}
	}

	regInfo := getClusterRegInfo(admInfo, clusterName)

	if offline {
		registerOffline(regInfo, alias)
	} else {
		registerOnline(regInfo, alias, clusterName)
	}
	console.Infoln(fmt.Sprintln("Cluster", clusterName, "successfully registered on SUBNET."))
	return nil
}

func registerOffline(clusterRegInfo ClusterRegistrationInfo, alias string) {
	regToken, e := generateRegToken(clusterRegInfo)
	fatalIf(probe.NewError(e), "Unable to generate registration token")

	subnetRegisterPageURL := "https://subnet.min.io/cluster/register"

	fmt.Print(`Step 1: Use the following token to register your cluster at ` + subnetRegisterPageURL + `

` + regToken + `

Step 2: Enter the API key generated by SUBNET: `)

	reader := bufio.NewReader(os.Stdin)
	apiKey, e := reader.ReadString('\n')
	fatalIf(probe.NewError(e), "Error in reading API key")
	apiKey = strings.TrimSpace(apiKey)

	if len(apiKey) > 0 {
		_, e := uuid.Parse(apiKey)
		fatalIf(probe.NewError(e), "Invalid API key specified:")
		setSubnetAPIKeyConfig(alias, apiKey)
	} else {
		console.Fatalln("Invalid API key specified. Please run the command again with a valid SUBNET API key to complete registration.")
	}
}

func registerOnline(clusterRegInfo ClusterRegistrationInfo, alias string, clusterName string) {
	resp, e := registerClusterOnSubnet(alias, clusterRegInfo)
	fatalIf(probe.NewError(e), "Could not register cluster with SUBNET:")

	extractAndSaveAPIKey(alias, resp)
}
