// Copyright (c) 2015-2023 MinIO, Inc.
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
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/klauspost/compress/gzip"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

const (
	anonymizeFlag     = "anonymize"
	anonymizeStandard = "standard"
	anonymizeStrict   = "strict"
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
		Name:  anonymizeFlag,
		Usage: "Data anonymization mode (standard|strict)",
		Value: anonymizeStandard,
	},
}, subnetCommonFlags...)

var supportDiagCmd = cli.Command{
	Name:         "diag",
	Aliases:      []string{"diagnostics"},
	Usage:        "upload health data for diagnostics",
	OnUsageError: onUsageError,
	Action:       mainSupportDiag,
	Before:       setGlobalsFromContext,
	Flags:        supportDiagFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Upload MinIO diagnostics report for cluster with alias 'myminio' to SUBNET
     {{.Prompt}} {{.HelpName}} myminio

  2. Generate MinIO diagnostics report for cluster with alias 'myminio', save and upload to SUBNET manually
     {{.Prompt}} {{.HelpName}} myminio --airgap

  3. Upload MinIO diagnostics report for cluster with alias 'myminio' to SUBNET, with strict anonymization
     {{.Prompt}} {{.HelpName}} myminio --anonymize=strict
`,
}

type supportDiagMessage struct {
	Status string `json:"status"`
}

// String colorized status message
func (s supportDiagMessage) String() string {
	return console.Colorize(supportSuccessMsgTag, "MinIO diagnostics report was successfully uploaded to SUBNET.")
}

// JSON jsonified status message
func (s supportDiagMessage) JSON() string {
	s.Status = "success"
	return toJSON(s)
}

// checkSupportDiagSyntax - validate arguments passed by a user
func checkSupportDiagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	anon := ctx.String(anonymizeFlag)
	if anon != anonymizeStandard && anon != anonymizeStrict {
		fatal(errDummy().Trace(), "Invalid anonymization mode. Valid options are 'standard' or 'strict'.")
	}
}

// compress and tar MinIO diagnostics output
func tarGZ(healthInfo any, version, filename string) error {
	data, e := TarGZHealthInfo(healthInfo, version)
	if e != nil {
		return e
	}

	e = os.WriteFile(filename, data, 0o666)
	if e != nil {
		return e
	}

	if globalAirgapped {
		warningMsgBoundary := "*********************************************************************************"
		warning := warnText("                                   WARNING!!")
		warningContents := infoText(`     ** THIS FILE MAY CONTAIN SENSITIVE INFORMATION ABOUT YOUR ENVIRONMENT **
     ** PLEASE INSPECT CONTENTS BEFORE SHARING IT ON ANY PUBLIC FORUM **`)

		warningMsgHeader := infoText(warningMsgBoundary)
		warningMsgTrailer := infoText(warningMsgBoundary)
		console.Printf("%s\n%s\n%s\n%s\n", warningMsgHeader, warning, warningContents, warningMsgTrailer)
		console.Infoln("MinIO diagnostics report saved at ", filename)
	}

	return nil
}

// TarGZHealthInfo - compress and tar MinIO diagnostics output
func TarGZHealthInfo(healthInfo any, version string) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	gzWriter := gzip.NewWriter(buffer)

	enc := gojson.NewEncoder(gzWriter)

	header := struct {
		Version string `json:"version"`
	}{Version: version}

	if e := enc.Encode(header); e != nil {
		return nil, e
	}

	if e := enc.Encode(healthInfo); e != nil {
		return nil, e
	}

	if e := gzWriter.Close(); e != nil {
		return nil, e
	}

	return buffer.Bytes(), nil
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
	alias, apiKey := initSubnetConnectivity(ctx, aliasedURL, true)
	if len(apiKey) == 0 {
		// api key not passed as flag. Check that the cluster is registered.
		apiKey = validateClusterRegistered(alias, true)
	}

	// Create a new MinIO Admin Client
	client := getClient(aliasedURL)

	// Main execution
	execSupportDiag(ctx, client, alias, apiKey)

	return nil
}

func execSupportDiag(ctx *cli.Context, client *madmin.AdminClient, alias, apiKey string) {
	var reqURL string
	var headers map[string]string
	setSuccessMessageColor()

	filename := fmt.Sprintf("%s-health_%s.json.gz", filepath.Clean(alias), UTCNow().Format("20060102150405"))
	if !globalAirgapped {
		// Retrieve subnet credentials (login/license) beforehand as
		// it can take a long time to fetch the health information
		uploadURL := SubnetUploadURL("health")
		reqURL, headers = prepareSubnetUploadURL(uploadURL, alias, apiKey)
	}

	healthInfo, version, e := fetchServerDiagInfo(ctx, client)
	fatalIf(probe.NewError(e), "Unable to fetch health information.")

	if globalJSON && globalAirgapped {
		switch version {
		case madmin.HealthInfoVersion0:
			printMsg(healthInfo.(madmin.HealthInfoV0))
		case madmin.HealthInfoVersion2:
			printMsg(healthInfo.(madmin.HealthInfoV2))
		case madmin.HealthInfoVersion:
			printMsg(healthInfo.(madmin.HealthInfo))
		}
		return
	}

	e = tarGZ(healthInfo, version, filename)
	fatalIf(probe.NewError(e), "Unable to save MinIO diagnostics report")

	if !globalAirgapped {
		_, e = (&SubnetFileUploader{
			alias:             alias,
			FilePath:          filename,
			ReqURL:            reqURL,
			Headers:           headers,
			DeleteAfterUpload: true,
		}).UploadFileToSubnet()
		fatalIf(probe.NewError(e), "Unable to upload MinIO diagnostics report to SUBNET portal")

		printMsg(supportDiagMessage{})
	}
}

func fetchServerDiagInfo(ctx *cli.Context, client *madmin.AdminClient) (any, string, error) {
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
		printText := func(t, sp string, rewind int) {
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
	process := spinner("Process Info", madmin.HealthDataTypeSysProcess)
	config := spinner("Server Config", madmin.HealthDataTypeMinioConfig)
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

	// Fetch info of all servers (cluster or single server)
	resp, version, e := client.ServerHealthInfo(cont, *opts, ctx.Duration("deadline"), ctx.String(anonymizeFlag))
	if e != nil {
		cancel()
		return nil, "", e
	}

	var healthInfo any

	decoder := json.NewDecoder(resp.Body)
	switch version {
	case madmin.HealthInfoVersion0:
		info := madmin.HealthInfoV0{}
		for {
			if e = decoder.Decode(&info); e != nil {
				if errors.Is(e, io.EOF) {
					e = nil
				}

				break
			}

			progressV0(info)
		}

		// Old minio versions don't return the MinIO info in
		// response of the healthinfo api. So fetch it separately
		minioInfo, e := client.ServerInfo(globalContext)
		if e != nil {
			info.Minio.Error = e.Error()
		} else {
			info.Minio.Info = minioInfo
		}

		healthInfo = MapHealthInfoToV1(info, nil)
		version = madmin.HealthInfoVersion1
	case madmin.HealthInfoVersion2:
		info := madmin.HealthInfoV2{}
		for {
			if e = decoder.Decode(&info); e != nil {
				if errors.Is(e, io.EOF) {
					e = nil
				}

				break
			}

			progressV2(info)
		}
		healthInfo = info
	case madmin.HealthInfoVersion:
		healthInfo, e = receiveHealthInfo(decoder)
	}

	// cancel the context if supportDiagChan has returned.
	cancel()
	return healthInfo, version, e
}

// HealthDataTypeSlice is a typed list of health tests
type HealthDataTypeSlice []madmin.HealthDataType

// Set - sets the flag to the given value
func (d *HealthDataTypeSlice) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		if supportDiagData, ok := madmin.HealthDataTypesMap[strings.Trim(v, " ")]; ok {
			*d = append(*d, supportDiagData)
		} else {
			return fmt.Errorf("valid options include %s", options.String())
		}
	}
	return nil
}

// String - returns the string representation of the health datatypes
func (d *HealthDataTypeSlice) String() string {
	val := ""
	for _, supportDiagData := range *d {
		formatStr := "%s"
		if val != "" {
			formatStr = fmt.Sprintf("%s,%%s", formatStr)
		} else {
			formatStr = fmt.Sprintf("%s%%s", formatStr)
		}
		val = fmt.Sprintf(formatStr, val, string(supportDiagData))
	}
	return val
}

// Value - returns the value
func (d *HealthDataTypeSlice) Value() []madmin.HealthDataType {
	return *d
}

// Get - returns the value
func (d *HealthDataTypeSlice) Get() any {
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
					if e := newVal.Set(s); e != nil {
						return fmt.Errorf("could not parse %s as health datatype value for flag %s: %s", envVal, f.Name, e)
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
