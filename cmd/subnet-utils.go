// Copyright (c) 2015-2024 MinIO, Inc.
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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/licverifier"
	"github.com/minio/pkg/v3/subnet"
	"github.com/tidwall/gjson"
	"golang.org/x/term"
)

const (
	subnetRespBodyLimit     = 1 << 20 // 1 MiB
	minioSubscriptionURL    = "https://min.io/subscription"
	subnetPublicKeyPath     = "/downloads/license-pubkey.pem"
	minioDeploymentIDHeader = "x-minio-deployment-id"
)

var subnetCommonFlags = append(supportGlobalFlags, cli.StringFlag{
	Name:   "api-key",
	Usage:  "API Key of the account on SUBNET",
	EnvVar: "_MC_SUBNET_API_KEY",
})

// SubnetBaseURL - returns the base URL of SUBNET
func SubnetBaseURL() string {
	return subnet.BaseURL(GlobalDevMode)
}

func subnetIssueURL(issueNum int) string {
	return fmt.Sprintf("%s/issues/%d", SubnetBaseURL(), issueNum)
}

// SubnetUploadURL - returns the upload URL for the given upload type
func SubnetUploadURL(uploadType string) string {
	return fmt.Sprintf("%s/api/%s/upload", SubnetBaseURL(), uploadType)
}

// SubnetRegisterURL - returns the cluster registration URL
func SubnetRegisterURL() string {
	return SubnetBaseURL() + "/api/cluster/register"
}

func subnetUnregisterURL(depID string) string {
	return SubnetBaseURL() + "/api/cluster/unregister?deploymentId=" + depID
}

func subnetLicenseRenewURL() string {
	return SubnetBaseURL() + "/api/cluster/renew-license"
}

func subnetOfflineRegisterURL(regToken string) string {
	return SubnetBaseURL() + "/cluster/register?token=" + regToken
}

func subnetLoginURL() string {
	return SubnetBaseURL() + "/api/auth/login"
}

func subnetAPIKeyURL() string {
	return SubnetBaseURL() + "/api/auth/api-key"
}

func subnetMFAURL() string {
	return SubnetBaseURL() + "/api/auth/mfa-login"
}

func checkURLReachable(url string) *probe.Error {
	_, e := subnetHeadReq(url, nil)
	if e != nil {
		return probe.NewError(e).Trace(url)
	}
	return nil
}

func subnetURLWithAuth(reqURL, apiKey string) (string, map[string]string, error) {
	if len(apiKey) == 0 {
		// API key not available in minio/mc config.
		// Ask the user to log in to get auth token
		token, e := subnetLogin()
		if e != nil {
			return "", nil, e
		}
		apiKey, e = getSubnetAPIKeyUsingAuthToken(token)
		if e != nil {
			return "", nil, e
		}
	}
	return reqURL, SubnetAPIKeyAuthHeaders(apiKey), nil
}

// SubnetHeaders - type for SUBNET request headers
type SubnetHeaders map[string]string

func (h SubnetHeaders) addDeploymentIDHeader(alias string) {
	h[minioDeploymentIDHeader] = getAdminInfo(alias).DeploymentID
}

func subnetTokenAuthHeaders(authToken string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + authToken}
}

// SubnetLicenseAuthHeaders - returns the headers for SUBNET license authentication
func SubnetLicenseAuthHeaders(lic string) map[string]string {
	return map[string]string{"x-subnet-license": lic}
}

// SubnetAPIKeyAuthHeaders - returns the headers for SUBNET API key authentication
func SubnetAPIKeyAuthHeaders(apiKey string) SubnetHeaders {
	return map[string]string{"x-subnet-api-key": apiKey}
}

func getSubnetClient() *http.Client {
	client := httpClient(0)
	if GlobalSubnetProxyURL != nil {
		client.Transport.(*http.Transport).Proxy = http.ProxyURL(GlobalSubnetProxyURL)
	}
	return client
}

func subnetHTTPDo(req *http.Request) (resp *http.Response, err error) {
	resp, err = getSubnetClient().Do(req)
	if err == nil && globalDebug {
		dumpHTTPReq(req, resp)
	}
	return
}

// dumpHTTP - dump HTTP request and response.
func dumpHTTPReq(req *http.Request, resp *http.Response) error {
	// Starts http dump.
	_, err := fmt.Fprintln(os.Stderr, "---------START-HTTP---------")
	if err != nil {
		return err
	}

	hdrs := req.Header
	for _, hdr := range []string{"Authorization", "x-subnet-license", "x-subnet-api-key"} {
		if val := hdrs.Get(hdr); val != "" {
			req.Header.Set(hdr, strings.Repeat("*", len(val)))
		}
	}

	query := req.URL.Query()
	for _, q := range []string{"api-key", "api_key"} {
		if val := query.Get(q); val != "" {
			query.Add(q, strings.Repeat("*", len(val)))
		}
	}
	req.URL.RawQuery = query.Encode()

	// Only display request header.
	reqTrace, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}

	// Write request to trace output.
	_, err = fmt.Fprint(os.Stderr, string(reqTrace))
	if err != nil {
		return err
	}

	respTrace, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}

	// Write response to trace output.
	_, err = fmt.Fprint(os.Stderr, strings.TrimSuffix(string(respTrace), "\r\n"))
	if err != nil {
		return err
	}

	// Ends the http dump.
	_, err = fmt.Fprintln(os.Stderr, "---------END-HTTP---------")
	return err
}

func subnetReqDo(r *http.Request, headers map[string]string) (string, error) {
	for k, v := range headers {
		r.Header.Add(k, v)
	}

	ct := r.Header.Get("Content-Type")
	if len(ct) == 0 {
		r.Header.Add("Content-Type", "application/json")
	}

	resp, e := subnetHTTPDo(r)
	if e != nil {
		return "", e
	}

	defer resp.Body.Close()
	respBytes, e := io.ReadAll(io.LimitReader(resp.Body, subnetRespBodyLimit))
	if e != nil {
		return "", e
	}
	respStr := string(respBytes)

	if resp.StatusCode == http.StatusOK {
		return respStr, nil
	}
	return respStr, fmt.Errorf("Request failed with code %d with error: %s", resp.StatusCode, respStr)
}

func subnetHeadReq(reqURL string, headers map[string]string) (string, error) {
	r, e := http.NewRequest(http.MethodHead, reqURL, nil)
	if e != nil {
		return "", e
	}
	return subnetReqDo(r, headers)
}

func subnetGetReq(reqURL string, headers map[string]string) (string, error) {
	r, e := http.NewRequest(http.MethodGet, reqURL, nil)
	if e != nil {
		return "", e
	}
	return subnetReqDo(r, headers)
}

// SubnetPostReq - makes a POST request to SUBNET
func SubnetPostReq(reqURL string, payload any, headers map[string]string) (string, error) {
	body, e := json.Marshal(payload)
	if e != nil {
		return "", e
	}
	r, e := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if e != nil {
		return "", e
	}
	return subnetReqDo(r, headers)
}

func getMinIOSubSysConfig(client *madmin.AdminClient, subSys string) ([]madmin.SubsysConfig, error) {
	buf, e := client.GetConfigKV(globalContext, subSys)
	if e != nil {
		return nil, e
	}

	return madmin.ParseServerConfigOutput(string(buf))
}

func getMinIOSubnetConfig(alias string) []madmin.SubsysConfig {
	if globalSubnetConfig != nil {
		return globalSubnetConfig
	}

	client, err := newAdminClient(alias)
	fatalIf(err, "Unable to initialize admin connection.")

	var e error
	globalSubnetConfig, e = getMinIOSubSysConfig(client, madmin.SubnetSubSys)
	if e != nil && e.Error() != "unknown sub-system subnet" {
		fatal(probe.NewError(e), "Unable to get server config for subnet")
	}

	return globalSubnetConfig
}

func getKeyFromSubnetConfig(alias, key string) (string, bool) {
	scfg := getMinIOSubnetConfig(alias)

	// This function only works for fetch config from single target sub-systems
	// in the server config and is enough for now.
	if len(scfg) == 0 {
		return "", false
	}

	return scfg[0].Lookup(key)
}

func getSubnetAPIKeyFromConfig(alias string) string {
	// get the subnet api_key config from MinIO if available
	apiKey, supported := getKeyFromSubnetConfig(alias, "api_key")
	if supported {
		return apiKey
	}

	// otherwise get it from mc config
	return mcConfig().Aliases[alias].APIKey
}

func setGlobalSubnetProxyFromConfig(alias string) error {
	if GlobalSubnetProxyURL != nil {
		// proxy already set
		return nil
	}

	var (
		proxy     string
		supported bool
	)

	if env, ok := os.LookupEnv("_MC_SUBNET_PROXY_URL"); ok {
		proxy = env
		supported = env != ""
	} else {
		proxy, supported = getKeyFromSubnetConfig(alias, "proxy")
	}

	// get the subnet proxy config from MinIO if available
	if supported && len(proxy) > 0 {
		proxyURL, e := url.Parse(proxy)
		if e != nil {
			return e
		}
		GlobalSubnetProxyURL = proxyURL
	}
	return nil
}

func getSubnetLicenseFromConfig(alias string) string {
	// get the subnet license config from MinIO if available
	lic, supported := getKeyFromSubnetConfig(alias, "license")
	if supported {
		return lic
	}

	// otherwise get it from mc config
	return mcConfig().Aliases[alias].License
}

func mcConfig() *configV10 {
	loadMcConfig = loadMcConfigFactory()
	config, err := loadMcConfig()
	fatalIf(err.Trace(mustGetMcConfigPath()), "Unable to access configuration file.")
	return config
}

func minioConfigSupportsSubSys(client *madmin.AdminClient, subSys string) bool {
	help, e := client.HelpConfigKV(globalContext, "", "", false)
	fatalIf(probe.NewError(e), "Unable to get minio config keys")

	for _, h := range help.KeysHelp {
		if h.Key == subSys {
			return true
		}
	}

	return false
}

func setSubnetAPIKeyInMcConfig(alias, apiKey string) {
	aliasCfg := mcConfig().Aliases[alias]
	if len(apiKey) > 0 {
		aliasCfg.APIKey = apiKey
	}

	setAlias(alias, aliasCfg)
}

func setSubnetLicenseInMcConfig(alias, lic string) {
	aliasCfg := mcConfig().Aliases[alias]
	if len(lic) > 0 {
		aliasCfg.License = lic
	}
	setAlias(alias, aliasCfg)
}

func setSubnetConfig(alias, subKey, cfgVal string) {
	client, err := newAdminClient(alias)
	fatalIf(err, "Unable to initialize admin connection.")

	cfgKey := "subnet " + subKey
	_, e := client.SetConfigKV(globalContext, cfgKey+"="+cfgVal)
	fatalIf(probe.NewError(e), "Unable to set "+cfgKey+" config on MinIO")
}

func setSubnetAPIKey(alias, apiKey string) {
	if len(apiKey) == 0 {
		fatal(errDummy().Trace(), "API Key must not be empty.")
	}

	_, apiKeySupported := getKeyFromSubnetConfig(alias, "api_key")
	if !apiKeySupported {
		setSubnetAPIKeyInMcConfig(alias, apiKey)
		return
	}

	setSubnetConfig(alias, "api_key", apiKey)
}

func setSubnetLicense(alias, lic string) {
	if len(lic) == 0 {
		fatal(errDummy().Trace(), "License must not be empty.")
	}

	_, licSupported := getKeyFromSubnetConfig(alias, "license")
	if !licSupported {
		setSubnetLicenseInMcConfig(alias, lic)
		return
	}

	setSubnetConfig(alias, "license", lic)
}

// GetClusterRegInfo - returns the cluster registration info
func GetClusterRegInfo(admInfo madmin.InfoMessage, clusterName string) ClusterRegistrationInfo {
	noOfPools := 1
	noOfDrives := 0
	for _, srvr := range admInfo.Servers {
		for _, poolNumber := range srvr.PoolNumbers {
			if poolNumber > noOfPools {
				noOfPools = poolNumber
			}
		}
		if len(srvr.PoolNumbers) == 0 {
			if srvr.PoolNumber != math.MaxInt && srvr.PoolNumber > noOfPools {
				noOfPools = srvr.PoolNumber
			}
		}
		noOfDrives += len(srvr.Disks)
	}

	totalSpace, usedSpace := getDriveSpaceInfo(admInfo)

	return ClusterRegistrationInfo{
		DeploymentID: admInfo.DeploymentID,
		ClusterName:  clusterName,
		UsedCapacity: admInfo.Usage.Size,
		Info: ClusterInfo{
			MinioVersion:    admInfo.Servers[0].Version,
			NoOfServerPools: noOfPools,
			NoOfServers:     len(admInfo.Servers),
			NoOfDrives:      noOfDrives,
			TotalDriveSpace: totalSpace,
			UsedDriveSpace:  usedSpace,
			NoOfBuckets:     admInfo.Buckets.Count,
			NoOfObjects:     admInfo.Objects.Count,
		},
	}
}

func getDriveSpaceInfo(admInfo madmin.InfoMessage) (uint64, uint64) {
	total := uint64(0)
	used := uint64(0)
	for _, srvr := range admInfo.Servers {
		for _, d := range srvr.Disks {
			total += d.TotalSpace
			used += d.UsedSpace
		}
	}
	return total, used
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
	fmt.Print("SUBNET username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	if len(username) == 0 {
		return "", errors.New("Username cannot be empty. If you don't have one, please create one from here: " + minioSubscriptionURL)
	}

	fmt.Print("Password: ")
	bytepw, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()

	loginReq := map[string]string{
		"username": username,
		"password": string(bytepw),
	}
	respStr, e := SubnetPostReq(subnetLoginURL(), loginReq, nil)
	if e != nil {
		return "", e
	}

	mfaRequired := gjson.Get(respStr, "mfa_required").Bool()
	if mfaRequired {
		mfaToken := gjson.Get(respStr, "mfa_token").String()
		fmt.Print("OTP received in email: ")
		byteotp, _ := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()

		mfaLoginReq := SubnetMFAReq{Username: username, OTP: string(byteotp), Token: mfaToken}
		respStr, e = SubnetPostReq(subnetMFAURL(), mfaLoginReq, nil)
		if e != nil {
			return "", e
		}
	}

	token := gjson.Get(respStr, "token_info.access_token")
	if token.Exists() {
		return token.String(), nil
	}
	return "", fmt.Errorf("access token not found in response")
}

// getSubnetCreds - returns the API key and license.
// If only one of them is available, and if `--airgap` is not
// passed, it will attempt to fetch the other from SUBNET
// and save to config
func getSubnetCreds(alias string) (string, string, error) {
	apiKey := getSubnetAPIKeyFromConfig(alias)
	lic := getSubnetLicenseFromConfig(alias)

	if (len(apiKey) > 0 && len(lic) > 0) ||
		(len(apiKey) == 0 && len(lic) == 0) ||
		globalAirgapped {
		return apiKey, lic, nil
	}

	var e error
	// Not airgapped, and only one of api-key or license is available
	// Try to fetch and save the other.
	if len(apiKey) > 0 {
		lic, e = getSubnetLicenseUsingAPIKey(alias, apiKey)
	} else {
		apiKey, e = getSubnetAPIKeyUsingLicense(lic)
		if e == nil {
			setSubnetAPIKey(alias, apiKey)
		}
	}

	if e != nil {
		return "", "", e
	}

	return apiKey, lic, nil
}

// getSubnetAPIKey - returns the SUBNET API key.
// Returns error if the cluster is not registered with SUBNET.
func getSubnetAPIKey(alias string) (string, error) {
	apiKey, lic, e := getSubnetCreds(alias)
	if e != nil {
		return "", e
	}
	if len(apiKey) == 0 && len(lic) == 0 {
		e = fmt.Errorf("Please register the cluster first by running 'mc license register %s'", alias)
		return "", e
	}
	return apiKey, nil
}

func getSubnetAPIKeyUsingLicense(lic string) (string, error) {
	return getSubnetAPIKeyUsingAuthHeaders(SubnetLicenseAuthHeaders(lic))
}

func getSubnetAPIKeyUsingAuthToken(authToken string) (string, error) {
	return getSubnetAPIKeyUsingAuthHeaders(subnetTokenAuthHeaders(authToken))
}

func getSubnetAPIKeyUsingAuthHeaders(authHeaders map[string]string) (string, error) {
	resp, e := subnetGetReq(subnetAPIKeyURL(), authHeaders)
	if e != nil {
		return "", e
	}
	return extractSubnetCred("api_key", gjson.Parse(resp))
}

func getSubnetLicenseUsingAPIKey(alias, apiKey string) (string, error) {
	regInfo := GetClusterRegInfo(getAdminInfo(alias), alias)
	_, lic, e := registerClusterOnSubnet(regInfo, alias, apiKey)
	return lic, e
}

// registerClusterOnSubnet - Registers the given cluster on SUBNET using given API key for auth
// If the API key is empty, user will be asked to log in using SUBNET credentials.
func registerClusterOnSubnet(clusterRegInfo ClusterRegistrationInfo, alias, apiKey string) (string, string, error) {
	regURL, headers, e := subnetURLWithAuth(SubnetRegisterURL(), apiKey)
	if e != nil {
		return "", "", e
	}

	regToken, e := generateRegToken(clusterRegInfo)
	if e != nil {
		return "", "", e
	}

	reqPayload := ClusterRegistrationReq{Token: regToken}
	resp, e := SubnetPostReq(regURL, reqPayload, headers)
	if e != nil {
		return "", "", e
	}

	return extractAndSaveSubnetCreds(alias, resp)
}

func removeSubnetAuthConfig(alias string) {
	setSubnetConfig(alias, "api_key", "")
	setSubnetConfig(alias, "license", "")
}

// unregisterClusterFromSubnet - Unregisters the given cluster from SUBNET using given API key for auth
func unregisterClusterFromSubnet(depID, apiKey string) error {
	regURL, headers, e := subnetURLWithAuth(subnetUnregisterURL(depID), apiKey)
	if e != nil {
		return e
	}

	_, e = SubnetPostReq(regURL, nil, headers)
	return e
}

// validateAndSaveLic - validates the given license in minio config
// If the license contains api key and the saveApiKey arg is true,
// api key is also saved in the minio config
func validateAndSaveLic(lic, alias string, saveAPIKey bool) string {
	li, e := parseLicense(lic)
	fatalIf(probe.NewError(e), "Error parsing license")

	if li.ExpiresAt.Before(time.Now()) {
		fatalIf(errDummy().Trace(), fmt.Sprintf("License has expired on %s", li.ExpiresAt))
	}

	if len(li.DeploymentID) > 0 && li.DeploymentID != uuid.Nil.String() && li.DeploymentID != getAdminInfo(alias).DeploymentID {
		fatalIf(errDummy().Trace(), fmt.Sprintf("License is invalid for the deployment %s", alias))
	}

	setSubnetLicense(alias, lic)
	if len(li.APIKey) > 0 && saveAPIKey {
		setSubnetAPIKey(alias, li.APIKey)
	}

	return li.APIKey
}

// extractAndSaveSubnetCreds - extract license from response and set it in minio config
func extractAndSaveSubnetCreds(alias, resp string) (string, string, error) {
	parsedResp := gjson.Parse(resp)

	lic, e := extractSubnetCred("license_v2", parsedResp)
	if e != nil {
		return "", "", e
	}
	if len(lic) > 0 {
		apiKey := validateAndSaveLic(lic, alias, true)
		if len(apiKey) > 0 {
			return apiKey, lic, nil
		}
	}

	apiKey, e := extractSubnetCred("api_key", parsedResp)
	if e != nil {
		return "", "", e
	}
	if len(apiKey) > 0 {
		setSubnetAPIKey(alias, apiKey)
	}

	return apiKey, lic, nil
}

func extractSubnetCred(key string, resp gjson.Result) (string, error) {
	result := resp.Get(key)
	if result.Index == 0 {
		return "", fmt.Errorf("Couldn't extract %s from SUBNET response: %s", key, resp)
	}
	return result.String(), nil
}

// parseLicense parses the license with the bundle public key and return it's information
func parseLicense(license string) (*licverifier.LicenseInfo, error) {
	client := getSubnetClient()
	lv := subnet.LicenseValidator{
		Client:            *client,
		ExpiryGracePeriod: 0,
	}
	lv.Init(GlobalDevMode)
	return lv.ParseLicense(license)
}

func prepareSubnetUploadURL(uploadURL, alias, apiKey string) (string, map[string]string) {
	var e error
	if len(apiKey) == 0 {
		// api key not passed as flag. check if it's available in the config
		apiKey, e = getSubnetAPIKey(alias)
		fatalIf(probe.NewError(e), "Unable to retrieve SUBNET API key")
	}

	reqURL, headers, e := subnetURLWithAuth(uploadURL, apiKey)
	fatalIf(probe.NewError(e).Trace(uploadURL), "Unable to fetch SUBNET authentication")

	return reqURL, headers
}

func getAPIKeyFlag(ctx *cli.Context) (string, error) {
	apiKey := ctx.String("api-key")

	if len(apiKey) == 0 {
		return "", nil
	}

	_, e := uuid.Parse(apiKey)
	if e != nil {
		return "", e
	}

	return apiKey, nil
}

func initSubnetConnectivity(ctx *cli.Context, aliasedURL string, failOnConnErr bool) (string, string) {
	if ctx.Bool("airgap") && len(ctx.String("api-key")) > 0 {
		fatal(errDummy().Trace(), "--api-key is not applicable in airgap mode")
	}

	alias, _ := url2Alias(aliasedURL)

	apiKey, e := getAPIKeyFlag(ctx)
	fatalIf(probe.NewError(e), "Error in reading --api-key flag:")

	// if `--airgap` is provided no need to test SUBNET connectivity.
	if !globalAirgapped {
		e = setGlobalSubnetProxyFromConfig(alias)
		fatalIf(probe.NewError(e), "Error in setting SUBNET proxy:")

		sbu := SubnetBaseURL()
		err := checkURLReachable(sbu)
		if err != nil && failOnConnErr {
			fatal(err.Trace(aliasedURL), "Unable to reach %s, please use --airgap if there is no connectivity to SUBNET", sbu)
		}
	}

	return alias, apiKey
}
