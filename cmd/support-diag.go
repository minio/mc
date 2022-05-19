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
	"bytes"
	"context"
	gojson "encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/klauspost/compress/gzip"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/tidwall/gjson"
)

var supportDiagFlags = append([]cli.Flag{
	HealthDataTypeFlag{
		Name:   "test",
		Usage:  "choose specific diagnostics to run [" + options.String() + "]",
		Value:  nil,
		Hidden: true,
	},
	cli.DurationFlag{
		Name:   "deadline",
		Usage:  "maximum duration diagnostics should be allowed to run",
		Value:  1 * time.Hour,
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "license",
		Usage:  "SUBNET license key",
		Hidden: true, // deprecated dec 2021
	},
	cli.StringFlag{
		Name:   "name",
		Usage:  "Specify the name to associate to this MinIO cluster in SUBNET",
		Hidden: true, // deprecated may 2022
	},
}, subnetCommonFlags...)

var supportDiagCmd = cli.Command{
	Name:         "diag",
	Aliases:      []string{"diagnostics"},
	Usage:        "upload health data for diagnostics",
	OnUsageError: onUsageError,
	Action:       mainSupportDiag,
	Before:       setGlobalsFromContext,
	Flags:        append(supportDiagFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Upload MinIO diagnostics report for 'play' (https://play.min.io by default) to SUBNET
     {{.Prompt}} {{.HelpName}} play

  2. Generate MinIO diagnostics report for alias 'play' (https://play.min.io by default) save and upload to SUBNET manually
     {{.Prompt}} {{.HelpName}} play --airgap
`,
}

// checkSupportDiagSyntax - validate arguments passed by a user
func checkSupportDiagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "diag", 1) // last argument is exit code
	}
}

// compress and tar MinIO diagnostics output
func tarGZ(healthInfo interface{}, version string, filename string, showMessages bool) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return err
	}
	defer f.Close()

	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	enc := gojson.NewEncoder(gzWriter)

	header := struct {
		Version string `json:"version"`
	}{Version: version}

	if err := enc.Encode(header); err != nil {
		return err
	}

	if err := enc.Encode(healthInfo); err != nil {
		return err
	}

	if showMessages {
		warningMsgBoundary := "*********************************************************************************"
		warning := warnText("                                   WARNING!!")
		warningContents := infoText(`     ** THIS FILE MAY CONTAIN SENSITIVE INFORMATION ABOUT YOUR ENVIRONMENT **
     ** PLEASE INSPECT CONTENTS BEFORE SHARING IT ON ANY PUBLIC FORUM **`)

		warningMsgHeader := infoText(warningMsgBoundary)
		warningMsgTrailer := infoText(warningMsgBoundary)
		console.Printf("%s\n%s\n%s\n%s\n", warningMsgHeader, warning, warningContents, warningMsgTrailer)
		console.Infoln("MinIO diagnostics report saved at", filename)
	}

	return nil
}

func infoText(s string) string {
	console.SetColor("INFO", color.New(color.FgGreen, color.Bold))
	return console.Colorize("INFO", s)
}

func greenText(s string) string {
	console.SetColor("GREEN", color.New(color.FgGreen))
	return console.Colorize("GREEN", s)
}

func warnText(s string) string {
	console.SetColor("WARN", color.New(color.FgRed, color.Bold))
	return console.Colorize("WARN", s)
}

func mainSupportDiag(ctx *cli.Context) error {
	checkSupportDiagSyntax(ctx)

	// Get the alias parameter from cli
	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)

	license, offline := fetchSubnetUploadFlags(ctx)

	// license should be provided for us to reach subnet
	// if `--airgap` is provided do not need to reach out.
	uploadToSubnet := !offline
	if uploadToSubnet {
		fatalIf(checkURLReachable(subnetBaseURL()).Trace(aliasedURL), "Unable to reach %s to upload MinIO diagnostics report, please use --airgap to upload manually", subnetBaseURL())
	}

	e := validateFlags(uploadToSubnet)
	fatalIf(probe.NewError(e), "unable to parse input values")

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Main execution
	execSupportDiag(ctx, client, alias, license, uploadToSubnet)

	return nil
}

func fetchSubnetUploadFlags(ctx *cli.Context) (string, bool) {
	// license info to upload to subnet.
	license := ctx.String("license")

	// If set, the MinIO diagnostics will not be uploaded
	// to subnet and will only be saved locally.
	offline := ctx.Bool("airgap") || ctx.Bool("offline")

	return license, offline
}

func validateFlags(uploadToSubnet bool) error {
	if uploadToSubnet {
		if globalJSON {
			return errors.New("--json is applicable only when --airgap is also passed")
		}
		return nil
	}

	if globalDevMode {
		return errors.New("--dev is not applicable in airgap mode")
	}

	return nil
}

func execSupportDiag(ctx *cli.Context, client *madmin.AdminClient, alias string, license string, uploadToSubnet bool) {
	var reqURL string
	var headers map[string]string

	filename := fmt.Sprintf("%s-health_%s.json.gz", filepath.Clean(alias), UTCNow().Format("20060102150405"))
	if uploadToSubnet {
		// Retrieve subnet credentials (login/license) beforehand as
		// it can take a long time to fetch the health information
		reqURL, headers = prepareDiagUploadURL(alias, filename, license)
	}

	healthInfo, version, e := fetchServerDiagInfo(ctx, client)
	fatalIf(probe.NewError(e), "Unable to fetch health information.")

	if globalJSON {
		switch version {
		case madmin.HealthInfoVersion0:
			printMsg(healthInfo.(madmin.HealthInfoV0))
		case madmin.HealthInfoVersion:
			printMsg(healthInfo.(madmin.HealthInfo))
		}
		return
	}

	e = tarGZ(healthInfo, version, filename, !uploadToSubnet)
	fatalIf(probe.NewError(e), "Unable to save MinIO diagnostics report")

	if uploadToSubnet {
		e = uploadDiagReport(alias, filename, reqURL, headers)
		fatalIf(probe.NewError(e), "Unable to upload MinIO diagnostics report to SUBNET portal")
	}
}

func prepareDiagUploadURL(alias string, filename string, license string) (string, map[string]string) {
	apiKey := ""
	if len(license) == 0 {
		apiKey = getSubnetAPIKeyFromConfig(alias)

		if len(apiKey) == 0 {
			license = getSubnetLicenseFromConfig(alias)
			if len(license) == 0 {
				// Both api key and license not available. Ask user to register the cluster first
				e := fmt.Errorf("Please register the cluster first by running 'mc support register %s', or use --airgap flag", alias)
				fatalIf(probe.NewError(e), "Cluster not registered.")
			}
		}
	}

	uploadURL := subnetHealthUploadURL()

	reqURL, headers, e := subnetURLWithAuth(uploadURL, apiKey, license)
	fatalIf(probe.NewError(e).Trace(uploadURL), "Unable to fetch SUBNET authentication")

	reqURL = fmt.Sprintf("%s&filename=%s", reqURL, filename)
	return reqURL, headers
}

func uploadDiagReport(alias string, filename string, reqURL string, headers map[string]string) error {
	e := setSubnetProxyFromConfig(alias)
	if e != nil {
		return e
	}

	req, e := subnetUploadReq(reqURL, filename)
	if e != nil {
		return e
	}

	resp, e := subnetReqDo(req, headers)
	if e != nil {
		return e
	}

	extractAndSaveAPIKey(alias, resp)

	// Delete the report after successful upload
	os.Remove(filename)

	msg := "MinIO diagnostics report was successfully uploaded to SUBNET."
	clusterURL, _ := url.PathUnescape(gjson.Get(resp, "cluster_url").String())
	if len(clusterURL) > 0 {
		msg += fmt.Sprintf(" Please click here to view our analysis: %s", clusterURL)
	}
	console.Infoln(msg)
	return nil
}

func subnetUploadReq(url string, filename string) (*http.Request, error) {
	file, e := os.Open(filename)
	if e != nil {
		return nil, e
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, e := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if e != nil {
		return nil, e
	}
	if _, e = io.Copy(part, file); e != nil {
		return nil, e
	}
	writer.Close()

	r, e := http.NewRequest(http.MethodPost, url, &body)
	if e != nil {
		return nil, e
	}
	r.Header.Add("Content-Type", writer.FormDataContentType())

	return r, nil
}

func fetchServerDiagInfo(ctx *cli.Context, client *madmin.AdminClient) (interface{}, string, error) {
	opts := GetHealthDataTypeSlice(ctx, "test")
	if len(*opts) == 0 {
		opts = &options
	}

	optsMap := make(map[madmin.HealthDataType]struct{})
	for _, opt := range *opts {
		optsMap[opt] = struct{}{}
	}

	spinners := []string{"∙∙∙", "●∙∙", "∙●∙", "∙∙●"}
	cont, cancel := context.WithCancel(globalContext)
	defer cancel()

	startSpinner := func(s string) func() {
		ctx, cancel := context.WithCancel(cont)
		printText := func(t string, sp string, rewind int) {
			console.RewindLines(rewind)

			dot := infoText(dot)
			t = fmt.Sprintf("%s ...", t)
			t = greenText(t)
			sp = infoText(sp)
			toPrint := fmt.Sprintf("%s %s %s ", dot, t, sp)
			console.Printf("%s\n", toPrint)
		}
		i := 0
		sp := func() string {
			i = i + 1
			i = i % len(spinners)
			return spinners[i]
		}

		done := make(chan bool)
		doneToggle := false
		go func() {
			printText(s, sp(), 0)
			for {
				time.Sleep(500 * time.Millisecond) // 2 fps
				if ctx.Err() != nil {
					printText(s, check, 1)
					done <- true
					return
				}
				printText(s, sp(), 1)
			}
		}()
		return func() {
			cancel()
			if !doneToggle {
				<-done
				os.Stdout.Sync()
				doneToggle = true
			}
		}
	}

	spinner := func(resource string, opt madmin.HealthDataType) func(bool) bool {
		var spinStopper func()
		done := false

		_, ok := optsMap[opt] // check if option is enabled
		if globalJSON || !ok {
			return func(bool) bool {
				return true
			}
		}

		return func(cond bool) bool {
			if done {
				return done
			}
			if spinStopper == nil {
				spinStopper = startSpinner(resource)
			}
			if cond {
				done = true
				spinStopper()
			}
			return done
		}
	}

	admin := spinner("Admin Info", madmin.HealthDataTypeMinioInfo)
	cpu := spinner("CPU Info", madmin.HealthDataTypeSysCPU)
	diskHw := spinner("Disk Info", madmin.HealthDataTypeSysDriveHw)
	osInfo := spinner("OS Info", madmin.HealthDataTypeSysOsInfo)
	mem := spinner("Mem Info", madmin.HealthDataTypeSysMem)
	process := spinner("Process Info", madmin.HealthDataTypeSysLoad)
	config := spinner("Server Config", madmin.HealthDataTypeMinioConfig)
	drive := spinner("Drive Test", madmin.HealthDataTypePerfDrive)
	net := spinner("Network Test", madmin.HealthDataTypePerfNet)
	obj := spinner("Objects Test", madmin.HealthDataTypePerfObj)
	syserr := spinner("System Errors", madmin.HealthDataTypeSysErrors)
	syssrv := spinner("System Services", madmin.HealthDataTypeSysServices)
	sysconfig := spinner("System Config", madmin.HealthDataTypeSysConfig)

	progressV0 := func(info madmin.HealthInfoV0) {
		_ = admin(len(info.Minio.Info.Servers) > 0) &&
			cpu(len(info.Sys.CPUInfo) > 0) &&
			diskHw(len(info.Sys.DiskHwInfo) > 0) &&
			osInfo(len(info.Sys.OsInfo) > 0) &&
			mem(len(info.Sys.MemInfo) > 0) &&
			process(len(info.Sys.ProcInfo) > 0) &&
			config(info.Minio.Config != nil)
	}

	progressV2 := func(info madmin.HealthInfoV2) {
		_ = cpu(len(info.Sys.CPUInfo) > 0) &&
			diskHw(len(info.Sys.Partitions) > 0) &&
			osInfo(len(info.Sys.OSInfo) > 0) &&
			mem(len(info.Sys.MemInfo) > 0) &&
			process(len(info.Sys.ProcInfo) > 0) &&
			config(info.Minio.Config.Config != nil) &&
			syserr(len(info.Sys.SysErrs) > 0) &&
			syssrv(len(info.Sys.SysServices) > 0) &&
			sysconfig(len(info.Sys.SysConfig) > 0) &&
			admin(len(info.Minio.Info.Servers) > 0)
	}

	progress := func(info madmin.HealthInfo) {
		noOfServers := len(info.Sys.CPUInfo)
		_ = cpu(len(info.Sys.CPUInfo) > 0) &&
			diskHw(len(info.Sys.Partitions) > 0) &&
			osInfo(len(info.Sys.OSInfo) > 0) &&
			mem(len(info.Sys.MemInfo) > 0) &&
			process(len(info.Sys.ProcInfo) > 0) &&
			config(info.Minio.Config.Config != nil) &&
			drive(len(info.Perf.DrivePerf) > 0) &&
			obj(len(info.Perf.ObjPerf) > 0) &&
			net(noOfServers == 1 || len(info.Perf.NetPerf) > 0) &&
			syserr(len(info.Sys.SysErrs) > 0) &&
			syssrv(len(info.Sys.SysServices) > 0) &&
			sysconfig(len(info.Sys.SysConfig) > 0) &&
			admin(len(info.Minio.Info.Servers) > 0)
	}

	var err error
	// Fetch info of all servers (cluster or single server)
	resp, version, err := client.ServerHealthInfo(cont, *opts, ctx.Duration("deadline"))
	if err != nil {
		cancel()
		return nil, "", err
	}

	var healthInfo interface{}

	decoder := json.NewDecoder(resp.Body)
	switch version {
	case madmin.HealthInfoVersion0:
		info := madmin.HealthInfoV0{}
		for {
			if err = decoder.Decode(&info); err != nil {
				if errors.Is(err, io.EOF) {
					err = nil
				}

				break
			}

			progressV0(info)
		}

		// Old minio versions don't return the MinIO info in
		// response of the healthinfo api. So fetch it separately
		minioInfo, err := client.ServerInfo(globalContext)
		if err != nil {
			info.Minio.Error = err.Error()
		} else {
			info.Minio.Info = minioInfo
		}

		healthInfo = MapHealthInfoToV1(info, nil)
		version = madmin.HealthInfoVersion1
	case madmin.HealthInfoVersion2:
		info := madmin.HealthInfoV2{}
		for {
			if err = decoder.Decode(&info); err != nil {
				if errors.Is(err, io.EOF) {
					err = nil
				}

				break
			}

			progressV2(info)
		}
		healthInfo = info
	case madmin.HealthInfoVersion:
		info := madmin.HealthInfo{}
		for {
			if err = decoder.Decode(&info); err != nil {
				if errors.Is(err, io.EOF) {
					err = nil
				}

				break
			}

			progress(info)
		}
		healthInfo = info
	}

	// cancel the context if obdChan has returned.
	cancel()
	return healthInfo, version, err
}

// HealthDataTypeSlice is a typed list of health tests
type HealthDataTypeSlice []madmin.HealthDataType

// Set - sets the flag to the given value
func (d *HealthDataTypeSlice) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		if obdData, ok := madmin.HealthDataTypesMap[strings.Trim(v, " ")]; ok {
			*d = append(*d, obdData)
		} else {
			return fmt.Errorf("valid options include %s", options.String())
		}
	}
	return nil
}

// String - returns the string representation of the health datatypes
func (d *HealthDataTypeSlice) String() string {
	val := ""
	for _, obdData := range *d {
		formatStr := "%s"
		if val != "" {
			formatStr = fmt.Sprintf("%s,%%s", formatStr)
		} else {
			formatStr = fmt.Sprintf("%s%%s", formatStr)
		}
		val = fmt.Sprintf(formatStr, val, string(obdData))
	}
	return val
}

// Value - returns the value
func (d *HealthDataTypeSlice) Value() []madmin.HealthDataType {
	return *d
}

// Get - returns the value
func (d *HealthDataTypeSlice) Get() interface{} {
	return *d
}

// HealthDataTypeFlag is a typed flag to represent health datatypes
type HealthDataTypeFlag struct {
	Name   string
	Usage  string
	EnvVar string
	Hidden bool
	Value  *HealthDataTypeSlice
}

// String - returns the string to be shown in the help message
func (f HealthDataTypeFlag) String() string {
	return cli.FlagStringer(f)
}

// GetName - returns the name of the flag
func (f HealthDataTypeFlag) GetName() string {
	return f.Name
}

// GetHealthDataTypeSlice - returns the list of set health tests
func GetHealthDataTypeSlice(c *cli.Context, name string) *HealthDataTypeSlice {
	generic := c.Generic(name)
	if generic == nil {
		return nil
	}
	return generic.(*HealthDataTypeSlice)
}

// GetGlobalHealthDataTypeSlice - returns the list of set health tests set globally
func GetGlobalHealthDataTypeSlice(c *cli.Context, name string) *HealthDataTypeSlice {
	generic := c.GlobalGeneric(name)
	if generic == nil {
		return nil
	}
	return generic.(*HealthDataTypeSlice)
}

// Apply - applies the flag
func (f HealthDataTypeFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError - applies with error
func (f HealthDataTypeFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				newVal := &HealthDataTypeSlice{}
				for _, s := range strings.Split(envVal, ",") {
					s = strings.TrimSpace(s)
					if err := newVal.Set(s); err != nil {
						return fmt.Errorf("could not parse %s as health datatype value for flag %s: %s", envVal, f.Name, err)
					}
				}
				f.Value = newVal
				break
			}
		}
	}

	for _, name := range strings.Split(f.Name, ",") {
		name = strings.Trim(name, " ")
		if f.Value == nil {
			f.Value = &HealthDataTypeSlice{}
		}
		set.Var(f.Value, name, f.Usage)
	}
	return nil
}

var options = HealthDataTypeSlice(madmin.HealthDataTypesList)
