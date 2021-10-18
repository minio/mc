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

var adminHealthFlags = append([]cli.Flag{
	HealthDataTypeFlag{
		Name:   "test",
		Usage:  "choose health tests to run [" + fullOptions.String() + "]",
		Value:  nil,
		EnvVar: "MC_HEALTH_TEST,MC_OBD_TEST",
		Hidden: true,
	},
	cli.DurationFlag{
		Name:   "deadline",
		Usage:  "maximum duration that health tests should be allowed to run",
		Value:  1 * time.Hour,
		EnvVar: "MC_HEALTH_DEADLINE,MC_OBD_DEADLINE",
	},
	cli.StringFlag{
		Name:  "license",
		Usage: "SUBNET license key",
	},
	cli.IntFlag{
		Name:  "schedule",
		Usage: "Schedule automatic generation and upload of health reports to SUBNET based on the specified number of days (e.g. --schedule 2 is every 2 days)",
		Value: 0,
	},
	cli.BoolFlag{
		Name:   "full",
		Usage:  "Include long running tests (takes longer to generate the report)",
		Hidden: false,
	},
}, subnetCommonFlags...)

var adminSubnetHealthCmd = cli.Command{
	Name:         "health",
	Usage:        "run health check for SUBNET",
	OnUsageError: onUsageError,
	Action:       mainAdminHealth,
	Before:       setGlobalsFromContext,
	Flags:        append(adminHealthFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Upload health diagnostics for the MinIO cluster with alias 'play' (https://play.min.io by default) to SUBNET
     {{.Prompt}} {{.HelpName}} play
  2. Upload health diagnostics for the MinIO cluster with alias 'play' (https://play.min.io by default) to SUBNET using the proxy https://192.168.1.3:3128
     {{.Prompt}} {{.HelpName}} play --subnet-proxy https://192.168.1.3:3128
  3. Periodically upload health diagnostics for the MinIO cluster with alias 'play' (https://play.min.io by default) to SUBNET every 2 days
     {{.Prompt}} {{.HelpName}} play --schedule 2
  4. Generate health diagnostics report for the MinIO cluster with alias 'play' (https://play.min.io by default) to and output them to the current working directory
     {{.Prompt}} {{.HelpName}} play --airgap
`,
}

// checkAdminHealthSyntax - validate arguments passed by a user
func checkAdminHealthSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "health", 1) // last argument is exit code
	}
}

//compress and tar health report output
func tarGZ(healthInfo interface{}, version string, filename string, showMessages bool) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
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
		console.Infoln("Health data saved at", filename)
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

func mainAdminHealth(ctx *cli.Context) error {
	checkAdminHealthSyntax(ctx)

	// Get the alias parameter from cli
	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)

	license, schedule, name, offline := fetchSubnetUploadFlags(ctx)

	uploadToSubnet := !offline
	if uploadToSubnet && !subnetReachable() {
		console.Fatalln(subnetNotReachableMsg())
	}

	uploadPeriodically := schedule != 0

	e := validateFlags(uploadToSubnet, uploadPeriodically, name)
	fatalIf(probe.NewError(e), "Invalid flags.")

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	if len(name) == 0 {
		name = alias
	}
	// Main execution
	execAdminHealth(ctx, client, alias, license, name, uploadToSubnet)

	if uploadToSubnet && uploadPeriodically {
		// Periodic upload to subnet
		for {
			sleepDuration := time.Hour * 24 * time.Duration(schedule)
			console.Infoln("Waiting for", sleepDuration, "before running health diagnostics again.")
			time.Sleep(sleepDuration)
			execAdminHealth(ctx, client, alias, license, name, uploadToSubnet)
		}
	}
	return nil
}

func fetchSubnetUploadFlags(ctx *cli.Context) (string, int, string, bool) {
	// license flag is passed when the health data
	// is to be uploadeD to SUBNET
	license := ctx.String("license")

	// non-zero schedule means that health diagnostics
	// are to be run periodically and uploaded to subnet
	schedule := ctx.Int("schedule")

	// If set (along with --license), this will be passed to
	// subnet as the name of the cluster
	name := ctx.String("name")

	// If set, the health report file will not be uploaded
	// to subnet and will only be saved locally.
	offline := ctx.Bool("airgap") || ctx.Bool("offline")

	return license, schedule, name, offline
}

func validateFlags(uploadToSubnet bool, uploadPeriodically bool, name string) error {
	if uploadToSubnet {
		if globalJSON {
			return errors.New("--json and --license should not be passed together")
		}
		return nil
	}

	if globalDevMode {
		return errors.New("--dev is not applicable in airgap mode")
	}

	if uploadPeriodically {
		return errors.New("--schedule is applicable only when --license is also passed")
	}

	if len(name) > 0 {
		return errors.New("--name is applicable only when --license is also passed")
	}

	return nil
}

func execAdminHealth(ctx *cli.Context, client *madmin.AdminClient, alias string, license string, clusterName string, uploadToSubnet bool) {
	var reqURL string
	var headers map[string]string

	filename := fmt.Sprintf("%s-health_%s.json.gz", filepath.Clean(alias), UTCNow().Format("20060102150405"))
	if uploadToSubnet {
		// Retrieve subnet credentials (login/license) beforehand as
		// it can take a long time to fetch the health information
		reqURL, headers = prepareHealthUploadURL(alias, clusterName, filename, license)
	}

	healthInfo, version, e := fetchServerHealthInfo(ctx, client)
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
	fatalIf(probe.NewError(e), "Unable to create health report file")

	if uploadToSubnet {

		e = uploadHealthReport(alias, filename, reqURL, headers)
		if e == nil {
			// Delete the report after successful upload
			deleteFile(filename)
		}
		fatalIf(probe.NewError(e), "Unable to upload health report to SUBNET portal")
	}
}

func prepareHealthUploadURL(alias string, clusterName string, filename string, license string) (string, map[string]string) {
	if len(clusterName) == 0 {
		clusterName = alias
	}

	uploadURL := subnetHealthUploadURL()

	if len(license) == 0 {
		license = getSubnetLicenseFromConfig(alias)
	}

	reqURL, headers, e := subnetURLWithAuth(uploadURL, license)
	fatalIf(probe.NewError(e), "Unable to get SUBNET auth")

	reqURL = fmt.Sprintf("%s&clustername=%s&filename=%s", reqURL, clusterName, filename)
	return reqURL, headers
}

func uploadHealthReport(alias string, filename string, reqURL string, headers map[string]string) error {
	req, e := subnetUploadReq(reqURL, filename)
	if e != nil {
		return e
	}

	resp, herr := subnetExecReq(req, headers)
	if herr != nil {
		return herr
	}

	// extract license from response and set it in minio config
	subnetLic := gjson.Parse(resp).Get("license").String()
	if len(subnetLic) > 0 {
		setSubnetLicenseConfig(alias, subnetLic)
	}

	msg := "MinIO Health data was successfully uploaded to SUBNET."
	clusterURL, _ := url.PathUnescape(gjson.Get(resp, "cluster_url").String())
	if len(clusterURL) > 0 {
		msg += fmt.Sprintf(" Can be viewed at: %s", clusterURL)
	}
	console.Infoln(msg)
	return nil
}

func subnetUploadReq(url string, filename string) (*http.Request, error) {
	file, _ := os.Open(filename)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}
	writer.Close()

	r, _ := http.NewRequest("POST", url, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	return r, nil
}

func fetchServerHealthInfo(ctx *cli.Context, client *madmin.AdminClient) (interface{}, string, error) {
	opts := GetHealthDataTypeSlice(ctx, "test")
	if len(*opts) == 0 {
		full := ctx.Bool("full")
		if full {
			opts = &fullOptions
		} else {
			opts = &liteOptions
		}
	}

	optsMap := make(map[madmin.HealthDataType]struct{})
	for _, opt := range *opts {
		optsMap[opt] = struct{}{}
	}

	spinners := []string{"/", "|", "\\", "--", "|"}

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
	syserr := spinner("System Errors", madmin.HealthDataTypeSysErrors)
	syssrv := spinner("System Services", madmin.HealthDataTypeSysServices)
	sysconfig := spinner("System Config", madmin.HealthDataTypeSysConfig)

	progressV0 := func(info madmin.HealthInfoV0) {
		noOfServers := len(info.Sys.CPUInfo)
		_ = admin(len(info.Minio.Info.Servers) > 0) &&
			cpu(len(info.Sys.CPUInfo) > 0) &&
			diskHw(len(info.Sys.DiskHwInfo) > 0) &&
			osInfo(len(info.Sys.OsInfo) > 0) &&
			mem(len(info.Sys.MemInfo) > 0) &&
			process(len(info.Sys.ProcInfo) > 0) &&
			config(info.Minio.Config != nil) &&
			drive(len(info.Perf.DriveInfo) > 0) &&
			net(noOfServers == 1 || (len(info.Perf.Net) > 1 && len(info.Perf.NetParallel.Addr) > 0))
	}

	progress := func(info madmin.HealthInfo) {
		noOfServers := len(info.Sys.CPUInfo)
		_ = cpu(len(info.Sys.CPUInfo) > 0) &&
			diskHw(len(info.Sys.Partitions) > 0) &&
			osInfo(len(info.Sys.OSInfo) > 0) &&
			mem(len(info.Sys.MemInfo) > 0) &&
			process(len(info.Sys.ProcInfo) > 0) &&
			config(info.Minio.Config.Config != nil) &&
			drive(len(info.Perf.Drives) > 0) &&
			net(noOfServers == 1 || (len(info.Perf.Net) > 1 && len(info.Perf.NetParallel.Addr) > 0)) &&
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
			return fmt.Errorf("valid options include %s", fullOptions.String())
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
	return fmt.Sprintf("--%s                       %s", f.Name, f.Usage)
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

var liteOptions = HealthDataTypeSlice(madmin.HealthDataTypesLite)
var fullOptions = HealthDataTypeSlice(madmin.HealthDataTypesList)
