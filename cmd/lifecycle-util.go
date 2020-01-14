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
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"strings"

	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/credentials"
	"github.com/minio/minio/pkg/console"
)

const (
	idWidth             int = 13
	prefixWidth         int = 9
	statusWidth         int = 9
	expiryWidth         int = 9
	expiryDatesWidth    int = 13
	tagWidth            int = 20
	transitionWidth     int = 13
	transitionDateWidth int = 13
	storageClassWidth   int = 16
)

const (
	leftAlign   int = 1
	centerAlign int = 2
	rightAlign  int = 3
)

const (
	idLabel             string = "ID"
	prefixLabel         string = "Prefix"
	statusLabel         string = "Enabled"
	statusDisabledLabel string = "Disabled"
	expiryLabel         string = "Expiry"
	expiryDatesLabel    string = "Date/Days"
	singleTagLabel      string = "Tag"
	tagLabel            string = "Tags"
	transitionLabel     string = "Transition"
	transitionDateLabel string = "Date/Days"
	storageClassLabel   string = "Storage-Class"
)

const (
	expiryDatesLabelKey     string = "Expiry-Dates"
	transitionDatesLabelKey string = "Transition-Date"
	transitionDaysLabelKey  string = "Transition"
)

const (
	expiryDatesLabelFlag string = "Expiry-Date"
	expiryDaysLabelFlag  string = "Expiry"
)

const (
	fieldMainHeader         string = "Main-Heading"
	fieldThemeHeader        string = "Row-Header"
	fieldThemeRow           string = "Row-Normal"
	fieldThemeTick          string = "Row-Tick"
	fieldThemeExpiry        string = "Row-Expiry"
	fieldThemeResultSuccess string = "SucessOp"
)

const (
	tickCell      string = "\u2713"
	crossTickCell string = "\u2717"
	blankCell     string = " "
)

const defaultDateFormat string = "2006-01-02"

const (
	tagSeperator    string = ","
	keyValSeperator string = ":"
	tableSeperator  string = "|"
	// tableSeperator string = ""
)

type tableCellInfo struct {
	label       string
	multLabels  []string
	labelKey    string
	fieldTheme  string
	columnWidth int
	align       int
}

type showDetails struct {
	allAvailable bool
	expiry       bool
	transition   bool
	json         bool
	minimum      bool
}

func getCentered(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	length := (float64(maxLen - lblLth)) / float64(2)
	rptLth := (int)(math.Floor(length / float64(len(toPadWith))))
	if rptLth <= 0 {
		rptLth = 1
	}
	output := strings.Repeat(toPadWith, rptLth) + label + strings.Repeat(toPadWith, rptLth)
	return output
}

func getLeftAlgined(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	length := maxLen - lblLth
	if length <= 0 {
		return label
	}
	output := label + strings.Repeat(toPadWith, length)
	return output
}

func getRightAligned(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	length := maxLen - lblLth
	if length <= 0 {
		return label
	}
	output := strings.Repeat(toPadWith, length) + label
	return output
}

func getBucketNameFromURL(urlStr string) string {
	clientURL := newClientURL(urlStr)
	bucketName := splitStr(clientURL.String(), "/", 3)[1]
	return bucketName
}

func getHostCfgFromURL(urlStr string) *hostConfigV9 {
	clientURL := newClientURL(urlStr)
	alias := splitStr(clientURL.String(), "/", 3)[0]
	return mustGetHostConfig(alias)
}

func getParsedHostURL(urlStr string) (*url.URL, *probe.Error) {
	hostCfg := getHostCfgFromURL(urlStr)
	if hostCfg == nil {
		return nil, errInvalidURL(urlStr)
	}
	hostURLParse, err := url.Parse(hostCfg.URL)
	if err != nil {
		return nil, probe.NewError(err)
	}
	return hostURLParse, nil
}

func getMinioClient(urlStr string) (*minio.Client, *probe.Error) {
	var api *minio.Client
	hostCfg := getHostCfgFromURL(urlStr)
	if hostCfg == nil {
		return nil, errInvalidTarget(urlStr)
	}
	creds := credentials.NewStaticV4(hostCfg.AccessKey, hostCfg.SecretKey, "")
	options := minio.Options{
		Creds:        creds,
		Secure:       false,
		Region:       "",
		BucketLookup: minio.BucketLookupPath,
	}
	parsedHostURL, err := getParsedHostURL(urlStr)
	if err != nil {
		return nil, err
	}
	api, apierr := minio.NewWithOptions(parsedHostURL.Host, &options) // Used as hostname or host:port
	if apierr != nil {
		// fatalIf(probe.NewError(err), "Unable to connect `"+hostURLParse.Host+"` with obtained credentials.")]
		return nil, probe.NewError(apierr)
	}
	return api, nil
}

// Get lifecycle info from alias & bucket
func getIlmInfo(urlStr string) (string, *probe.Error) {
	bucketName := getBucketNameFromURL(urlStr)
	if bucketName == "" {
		bkterrstr := "Could not find bucket name."
		console.Colorize(fieldMainHeader, bkterrstr)
		return "", errInvalidTarget(urlStr)
	}
	api, err := getMinioClient(urlStr)
	if err != nil {
		return "", err
	}
	if api == nil {
		console.Errorln("Cannot call Get Bucket Lifecycle API. Couldn't obtain reference to API caller.")
		return "", errInvalidURL(urlStr)
	}

	lifecycleInfo, glcerr := api.GetBucketLifecycle(bucketName)

	if glcerr != nil {
		glcerrStr := "Could not get LifeCycle configuration for:" + urlStr + ". Error: " + glcerr.Error()
		console.Println(console.Colorize(fieldMainHeader, glcerrStr))
		return "", errInvalidURL(urlStr)
	}

	return lifecycleInfo, nil
}

func printIlmJSON(info ilmResult) {
	ilmJSON, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		console.Println("Unable to get JSON representation of lifecycle management structure.\n Error: " + err.Error())
	} else {
		console.Println(string(ilmJSON))
	}
}

func readFileToString(file string) string {
	cbfr, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}
	return string(cbfr)
}

func checkFileCompatibility(jsonContents string) bool {
	var ilmInfo ilmResult
	bfr := []byte(jsonContents)
	if err := json.Unmarshal(bfr, &ilmInfo); err != nil {
		return false
	}
	return true
}

func checkFileNamePathExists(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}
	return nil
}
