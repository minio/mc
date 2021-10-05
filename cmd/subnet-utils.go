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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"encoding/base64"
	"encoding/json"

	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/licverifier"
	"github.com/tidwall/gjson"
	"golang.org/x/term"
)

const minioSubscriptionURL = "https://min.io/subscription"

var subnetCommonFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "name",
		Usage: "Specify the name to associate to this MinIO cluster in SUBNET",
	},
	cli.StringFlag{
		Name:  "subnet-proxy",
		Usage: "Specify the HTTP(S) proxy URL to use for connecting to SUBNET",
	},
	cli.BoolFlag{
		Name:   "offline",
		Usage:  "Use in environments without network access to SUBNET (e.g. airgapped, firewalled, etc.)",
		Hidden: false,
	},
	cli.BoolFlag{
		Name:   "dev",
		Usage:  "Development mode - talks to local subnet",
		Hidden: true,
	},
}

func subnetBaseURL() string {
	if globalDevMode {
		return "http://localhost:9000"
	}

	return "https://subnet.min.io"
}

func subnetHealthUploadURL(clusterName string, filename string) string {
	return subnetBaseURL() + "/api/health/upload"
}

func subnetRegisterURL() string {
	return subnetBaseURL() + "/api/cluster/register"
}

func subnetLoginURL() string {
	return subnetBaseURL() + "/api/auth/login"
}

func subnetOrgsURL() string {
	return subnetBaseURL() + "/api/auth/organizations"
}

func subnetMFAURL() string {
	return subnetBaseURL() + "/api/auth/mfa-login"
}

func urlReachable(url string) bool {
	r, e := http.Head(url)
	return e == nil && r.StatusCode == http.StatusOK
}

func subnetReachable() bool {
	return urlReachable(subnetBaseURL())
}

func subnetURLWithAuth(reqURL string, license string) (string, map[string]string, error) {
	headers := map[string]string{}
	if len(license) > 0 {
		// Add license in url for authentication
		reqURL = reqURL + "?license=" + license
	} else {
		// License not available in minio/mc config.
		// Ask the user to log in to get auth token
		token, e := subnetLogin()
		if e != nil {
			return "", nil, e
		}
		headers = subnetAuthHeaders(token)

		accID, err := getSubnetAccID(headers)
		if err != nil {
			return "", headers, e
		}

		reqURL = reqURL + "?aid=" + accID
	}
	return reqURL, headers, nil
}

func subnetAuthHeaders(authToken string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + authToken}
}

func execReq(req *http.Request) (*http.Response, error) {
	client := httpClient(10 * time.Second)
	if globalSubnetProxyURL != nil {
		client.Transport.(*http.Transport).Proxy = http.ProxyURL(globalSubnetProxyURL)
	}
	return client.Do(req)
}

func subnetExecReq(r *http.Request, headers map[string]string) (string, error) {
	for k, v := range headers {
		r.Header.Add(k, v)
	}

	ct := r.Header.Get("Content-Type")
	if len(ct) == 0 {
		r.Header.Add("Content-Type", "application/json")
	}

	resp, e := execReq(r)
	if e != nil {
		return "", e
	}

	defer resp.Body.Close()
	respBytes, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		return "", e
	}
	respStr := string(respBytes)

	if resp.StatusCode == http.StatusOK {
		return respStr, nil
	}
	return respStr, fmt.Errorf("Request failed with code %d and error: %s", resp.StatusCode, respStr)
}

func subnetGetReq(reqURL string, headers map[string]string) (string, error) {
	r, _ := http.NewRequest("GET", reqURL, nil)
	return subnetExecReq(r, headers)
}

func subnetPostReq(reqURL string, payload interface{}, headers map[string]string) (string, error) {
	body, _ := json.Marshal(payload)
	r, _ := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	return subnetExecReq(r, headers)
}

func getSubnetLicenseFromConfig(aliasedURL string) string {
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if minioConfigSupportsLicense(client) {
		sh, pe := client.HelpConfigKV(globalContext, "subnet", "license", false)
		fatalIf(probe.NewError(pe), "Unable to get config keys for subnet")

		buf, e := client.GetConfigKV(globalContext, "subnet")
		fatalIf(probe.NewError(e), "Unable to get server subnet config")

		tgt, e := madmin.ParseSubSysTarget(buf, sh)
		fatalIf(probe.NewError(e), "Unable to parse sub-system target subnet")

		lic := tgt.KVS.Get("license")
		if len(lic) > 0 {
			return lic
		}
	}

	return mcConfig().Aliases[aliasedURL].License
}

func mcConfig() *configV10 {
	loadMcConfig = loadMcConfigFactory()
	config, err := loadMcConfig()
	fatalIf(err.Trace(mustGetMcConfigPath()), "Unable to access configuration file.")
	return config
}

func minioConfigSupportsLicense(client *madmin.AdminClient) bool {
	help, e := client.HelpConfigKV(globalContext, "", "", false)
	fatalIf(probe.NewError(e), "Unable to get minio config keys")

	for _, h := range help.KeysHelp {
		if h.Key == "subnet" {
			return true
		}
	}

	return false
}

func setSubnetLicenseConfig(aliasedURL string, lic string) {
	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if minioConfigSupportsLicense(client) {
		configStr := "subnet license=" + lic
		_, e := client.SetConfigKV(globalContext, configStr)
		fatalIf(probe.NewError(e), "Unable to set subnet license config on minio")
		return
	}
	mcCfg := mcConfig()
	aliasCfg := mcCfg.Aliases[aliasedURL]
	aliasCfg.License = lic
	setAlias(aliasedURL, aliasCfg)
}

func getClusterRegInfo(admInfo madmin.InfoMessage, clusterName string) ClusterRegistrationInfo {
	noOfPools := 1
	noOfDrives := 0
	for _, srvr := range admInfo.Servers {
		if srvr.PoolNumber > noOfPools {
			noOfPools = srvr.PoolNumber
		}
		noOfDrives += len(srvr.Disks)
	}

	return ClusterRegistrationInfo{
		DeploymentID: admInfo.DeploymentID,
		ClusterName:  clusterName,
		UsedCapacity: admInfo.Usage.Size,
		Info: ClusterInfo{
			MinioVersion:    admInfo.Servers[0].Version,
			NoOfServerPools: noOfPools,
			NoOfServers:     len(admInfo.Servers),
			NoOfDrives:      noOfDrives,
			NoOfBuckets:     admInfo.Buckets.Count,
			NoOfObjects:     admInfo.Objects.Count,
		},
	}
}

func generateRegToken(clusterRegInfo ClusterRegistrationInfo) (string, error) {
	token, e := json.Marshal(clusterRegInfo)
	if e != nil {
		return "", e
	}

	return base64.StdEncoding.EncodeToString(token), nil
}

func subnetLogin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Subnet username: ")
	username, _ := reader.ReadString('\n')
	username = strings.Trim(username, "\n")

	if len(username) == 0 {
		return "", errors.New("Username cannot be empty. If you don't have one, please create one from here: " + minioSubscriptionURL)
	}

	fmt.Print("Password: ")
	bytepw, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()

	loginReq := map[string]string{
		"username": username,
		"password": string(bytepw),
	}
	respStr, e := subnetPostReq(subnetLoginURL(), loginReq, nil)
	if e != nil {
		return "", e
	}

	mfaRequired := gjson.Get(respStr, "mfa_required").Bool()
	if mfaRequired {
		mfaToken := gjson.Get(respStr, "mfa_token").String()
		fmt.Print("OTP received in email: ")
		byteotp, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()

		mfaLoginReq := SubnetMFAReq{Username: username, OTP: string(byteotp), Token: mfaToken}
		respStr, e = subnetPostReq(subnetMFAURL(), mfaLoginReq, nil)
	}
	if e != nil {
		return "", e
	}

	token := gjson.Get(respStr, "token_info.access_token")
	if token.Exists() {
		return token.String(), nil
	}
	return "", fmt.Errorf("access token not found in response")
}

func getSubnetAccID(headers map[string]string) (string, error) {
	respStr, e := subnetGetReq(subnetOrgsURL(), headers)
	if e != nil {
		return "", e
	}
	data := gjson.Parse(respStr)
	orgs := data.Array()
	idx := 1
	if len(orgs) > 1 {
		fmt.Println("You are part of multiple organizations on Subnet:")
		for idx, org := range orgs {
			fmt.Println("  ", idx+1, ":", org.Get("company"))
		}
		fmt.Print("Please choose the organization for this cluster: ")
		reader := bufio.NewReader(os.Stdin)
		accIdx, _ := reader.ReadString('\n')
		accIdx = strings.Trim(accIdx, "\n")
		idx, e = strconv.Atoi(accIdx)
		if e != nil {
			return "", e
		}
		if idx > len(orgs) {
			msg := "Invalid choice for organization. Please run the command again."
			return "", fmt.Errorf(msg)
		}
	}
	return orgs[idx-1].Get("accountId").String(), nil
}

// registerClusterOnSubnet - Registers the given cluster on subnet
func registerClusterOnSubnet(aliasedURL string, clusterRegInfo ClusterRegistrationInfo) (string, error) {
	lic := getSubnetLicenseFromConfig(aliasedURL)

	regURL, headers, e := subnetURLWithAuth(subnetRegisterURL(), lic)
	if e != nil {
		return "", e
	}

	regToken, e := generateRegToken(clusterRegInfo)
	if e != nil {
		return "", e
	}

	reqPayload := ClusterRegistrationReq{Token: regToken}
	return subnetPostReq(regURL, reqPayload, headers)
}

func verifySubnetLicense(lic string) error {
	var pemBytes []byte
	if globalDevMode {
		pemBytes = []byte(`-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEbo+e1wpBY4tBq9AONKww3Kq7m6QP/TBQ
mr/cKCUyBL7rcAvg0zNq1vcSrUSGlAmY3SEDCu3GOKnjG/U4E7+p957ocWSV+mQU
9NKlTdQFGF3+aO6jbQ4hX/S5qPyF+a3z
-----END PUBLIC KEY-----`)
	} else {
		pemBytes = []byte(`-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEaK31xujr6/rZ7ZfXZh3SlwovjC+X8wGq
qkltaKyTLRENd4w3IRktYYCRgzpDLPn/nrf7snV/ERO5qcI7fkEES34IVEr+2Uff
JkO2PfyyAYEO/5dBlPh1Undu9WQl6J7B
-----END PUBLIC KEY-----`)
	}
	lv, e := licverifier.NewLicenseVerifier(pemBytes)
	if e != nil {
		return e
	}
	_, e = lv.Verify(lic)
	return e
}
