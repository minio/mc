// Copyright (c) 2015-2021 MinIO, Inc.
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

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/tidwall/gjson"
)

var adminSubnetRegisterCmd = cli.Command{
	Name:         "register",
	Usage:        "register cluster with subnet",
	OnUsageError: onUsageError,
	Action:       mainAdminRegister,
	Before:       setGlobalsFromContext,
	Flags:        append(subnetCommonFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Register the cluster at 'play' MinIO server.
     {{.Prompt}} {{.HelpName}} play
  2. Register the cluster at 'play' MinIO server, naming it as 'play-cluster' on subnet
     {{.Prompt}} {{.HelpName}} play --name play-cluster
  3. Register the cluster at 'play' MinIO server, using the proxy 192.168.1.3:3128
     {{.Prompt}} {{.HelpName}} play --subnet-proxy 192.168.1.3:3128
`,
}

// checkAdminRegisterSyntax - validate arguments passed by a user
func checkAdminRegisterSyntax(ctx *cli.Context) {
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
}

// SubnetLoginReq - JSON payload of the subnet login api
type SubnetLoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SubnetMFAReq - JSON payload of the subnet mfa api
type SubnetMFAReq struct {
	Username string `json:"username"`
	OTP      string `json:"otp"`
	Token    string `json:"token"`
}

func mainAdminRegister(ctx *cli.Context) error {
	checkAdminRegisterSyntax(ctx)

	offlineMode := ctx.Bool(("offline"))
	if !offlineMode && !subnetReachable() {
		console.Fatalln("Subnet not reachable.")
	}

	// Get the alias parameter from cli
	aliasedURL := ctx.Args().Get(0)

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Fetch info of all servers (cluster or single server)
	admInfo, e := client.ServerInfo(globalContext)
	fatalIf(probe.NewError(e), "Unable to fetch cluster info")

	clusterName := ctx.String("name")
	if len(clusterName) == 0 {
		clusterName = aliasedURL
	}

	regInfo := getClusterRegInfo(admInfo, clusterName)

	if offlineMode {
		registerOffline(regInfo, aliasedURL)
	} else {
		registerOnline(regInfo, aliasedURL, clusterName)
	}

	msg := fmt.Sprintln("Cluster", aliasedURL, "has been successfully registered.")
	console.Infoln(msg)

	return nil
}

func registerOffline(clusterRegInfo ClusterRegistrationInfo, alias string) {
	regToken, e := generateRegToken(clusterRegInfo)
	fatalIf(probe.NewError(e), "Unable to generate registration token")

	subnetRegisterPageURL := "https://subnet.min.io/cluster/register"

	fmt.Print("The registration token for the cluster " + clusterRegInfo.ClusterName + ` is:

` + regToken + `

Please follow these steps to complete the registration:

1) Copy the registration token shown above
2) Open ` + subnetRegisterPageURL + `
3) Paste the registration token there and submit the form
4) Copy the license string generated
5) Paste it here: `)

	reader := bufio.NewReader(os.Stdin)
	lic, e := reader.ReadString('\n')
	fatalIf(probe.NewError(e), "Error in reading license string")
	lic = strings.Trim(lic, "\n")

	if len(lic) > 0 {
		e := verifySubnetLicense(lic)
		fatalIf(probe.NewError(e), "License could not be verified")
		setSubnetLicenseConfig(alias, lic)
	} else {
		console.Fatalln("Registration failed as license was not provided. Please run the command again to complete registration.")
	}
}

func registerOnline(clusterRegInfo ClusterRegistrationInfo, aliasedURL string, clusterName string) {
	resp, e := registerClusterOnSubnet(aliasedURL, clusterRegInfo)
	fatalIf(probe.NewError(e), "Cluster registration on subnet failed")

	// extract license from response and set it in minio config
	subnetLic := gjson.Parse(resp).Get("license").String()
	if len(subnetLic) > 0 {
		setSubnetLicenseConfig(aliasedURL, subnetLic)
	}
}
