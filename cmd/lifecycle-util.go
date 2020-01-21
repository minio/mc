/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"errors"
	"math"
	"net/url"
	"os"
	"strings"

	stdlibjson "encoding/json"
	"encoding/xml"

	"github.com/mattn/go-isatty"
	json "github.com/minio/mc/pkg/colorjson"

	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/credentials"
	"github.com/minio/minio/pkg/console"
)

const (
	idWidth             int = 12
	prefixWidth         int = 10
	statusWidth         int = 12
	expiryWidth         int = 8
	expiryDatesWidth    int = 14
	tagWidth            int = 22
	transitionWidth     int = 14
	transitionDateWidth int = 18
	storageClassWidth   int = 18
)

const (
	leftAlign   int = 1
	centerAlign int = 2
	rightAlign  int = 3
)

const (
	idLabel             string = "ID"
	prefixLabel         string = "Prefix"
	statusLabel         string = "Enabled "
	statusDisabledLabel string = "Disabled"
	expiryLabel         string = "Expiry"
	expiryDatesLabel    string = "Date/Days "
	singleTagLabel      string = "Tag"
	tagLabel            string = "Tags"
	transitionLabel     string = "Transition"
	transitionDateLabel string = "Date/Days "
	storageClassLabel   string = "Storage-Class "
	forceLabel          string = "force"
	allLabel            string = "all"
)

const (
	statusLabelKey          string = "Enabled"
	storageClassLabelKey    string = "Storage-Class"
	expiryDatesLabelKey     string = "Expiry-Dates"
	transitionDatesLabelKey string = "Transition-Date"
	transitionDaysLabelKey  string = "Transition-Days"
)

const (
	expiryDatesLabelFlag string = "Expiry-Date"
	expiryDaysLabelFlag  string = "Expiry-Days"
)

const (
	fieldMainHeader         string = "Main-Heading"
	fieldThemeHeader        string = "Row-Header"
	fieldThemeRow           string = "Row-Normal"
	fieldThemeTick          string = "Row-Tick"
	fieldThemeExpiry        string = "Row-Expiry"
	fieldThemeResultSuccess string = "SucessOp"
	fieldThemeResultFailure string = "FailureOp"
)

const (
	tickCell      string = "\u2713 "
	crossTickCell string = "\u2717 "
	blankCell     string = "  "
)

const defaultILMDateFormat string = "2006-01-02"

const (
	tagSeperator    string = ","
	keyValSeperator string = ":"
	tableSeperator  string = "|"
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
	if lblLth > 1 && lblLth%2 != 0 {
		lblLth++
	} else if lblLth == 1 {
		lblLth = 2
	}
	length := (float64(maxLen - lblLth)) / float64(2)
	rptLth := (int)(math.Floor(length / float64(len(toPadWith))))
	leftRptLth := rptLth
	rightRptLth := rptLth
	if rptLth <= 0 {
		leftRptLth = 1
		rightRptLth = 0
	}
	output := strings.Repeat(toPadWith, leftRptLth) + label + strings.Repeat(toPadWith, rightRptLth)
	return output
}

func getLeftAlgined(label string, maxLen int) string {
	const toPadWith string = " "
	lblLth := len(label)
	length := maxLen - lblLth
	if length <= 0 {
		return label
	}
	output := strings.Repeat(toPadWith, 1) + label + strings.Repeat(toPadWith, length-1)
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
		return nil, probe.NewError(apierr)
	}
	return api, nil
}

func getIlmConfig(urlStr string) (lfcInfo lifecycleConfiguration, err error) {
	var lfcInfoXML string
	var pErr *probe.Error
	if lfcInfoXML, pErr = getIlmInfo(urlStr); pErr != nil {
		return lfcInfo, pErr.ToGoError()
	}
	if lfcInfoXML == "" {
		console.Infoln("The lifecycle configuration for " + urlStr + " has not been set.")
		return lfcInfo, nil
	}

	if err = xml.Unmarshal([]byte(lfcInfoXML), &lfcInfo); err != nil {
		console.Errorln("Unable to extract lifecycle configuration from:" + urlStr)
		return lfcInfo, err
	}
	return lfcInfo, nil
}

// Get lifecycle info (XML) from alias & bucket
func getIlmInfo(urlStr string) (string, *probe.Error) {
	bucketName := getBucketNameFromURL(urlStr)
	if bucketName == "" {
		err := errors.New("bucket not found")
		fatalIf(probe.NewError(err), "Bucket Not found.")
	}
	api, err := getMinioClient(urlStr)
	if err != nil || api == nil {
		fatalIf(err, "Unable to call lifecycle methods on "+urlStr)
	}

	lifecycleInfo, glcerr := api.GetBucketLifecycle(bucketName)
	var pErr *probe.Error
	if glcerr != nil {
		pErr = probe.NewError(glcerr)
		if _, glcerr = api.GetBucketLocation(bucketName); glcerr != nil {
			fatalIf(probe.NewError(errors.New("bucket location/information access error")), "Unable to access bucket location or bucket lifecycle configuration.")
		} else {
			pErr = nil
		}
		return "", pErr
	}

	return lifecycleInfo, nil
}

func printIlmJSON(info lifecycleConfiguration) {
	var ilmJSON []byte
	var err error
	// mc ilm generate output is also intended to be used to redirect to a file and used with mc ilm set
	if isatty.IsTerminal(os.Stdout.Fd()) {
		ilmJSON, err = json.MarshalIndent(info, "", "    ")
	} else {
		ilmJSON, err = stdlibjson.MarshalIndent(info, "", "    ")
	}
	if err != nil {
		console.Println("Unable to get JSON representation of lifecycle management structure.\n Error: " + err.Error())
	} else {
		console.Println(string(ilmJSON) + "\n")
	}
}
