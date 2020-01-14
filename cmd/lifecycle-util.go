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
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/credentials"
	"github.com/minio/minio/pkg/console"
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

type showDetails struct {
	allAvailable bool
	expiry       bool
	transition   bool
	initial      bool
	json         bool
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

// Get ilm info from alias & bucket
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

func showInfoFieldFirst(label string, field string) {
	displayField := fmt.Sprintf("\n%-16s: %s ", label, field)
	console.Println(console.Colorize(fieldThemeHeader, displayField))
}

func showInfoField(label string, field string) {
	displayField := fmt.Sprintf("%-16s: %s ", label, field)
	console.Println(console.Colorize(fieldThemeRow, displayField))
}

func showInfoFieldMultiple(label string, values []string) {
	console.Print(console.Colorize(fieldThemeRow, fmt.Sprintf("%-16s: ", label)))
	if len(values) == 0 {
		console.Println(console.Colorize(fieldThemeRow, blankCell))
		return
	}
	for idx := 0; idx < len(values); idx++ {
		if idx == 0 {
			console.Println(console.Colorize(fieldThemeRow, values[idx]))
		} else {
			displayField := fmt.Sprintf("%-16s  %s ", blankCell, values[idx])
			console.Println(console.Colorize(fieldThemeRow, displayField))
		}
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
