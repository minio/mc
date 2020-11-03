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
	"context"
	gojson "encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/klauspost/compress/gzip"
	"github.com/minio/cli"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminHealthFlags = []cli.Flag{
	HealthDataTypeFlag{
		Name:   "test",
		Usage:  "choose health tests to run [" + options.String() + "]",
		Value:  nil,
		EnvVar: "MC_HEALTH_TEST,MC_OBD_TEST",
		Hidden: true,
	},
	cli.DurationFlag{
		Name:   "deadline",
		Usage:  "maximum duration that health tests should be allowed to run",
		Value:  3600 * time.Second,
		EnvVar: "MC_HEALTH_DEADLINE,MC_OBD_DEADLINE",
	},
}

var adminSubnetHealthCmd = cli.Command{
	Name:   "health",
	Usage:  "run health check for Subnet",
	Action: mainAdminHealth,
	Before: setGlobalsFromContext,
	Flags:  append(adminHealthFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get server information of the 'play' MinIO server.
     {{.Prompt}} {{.HelpName}} play/
`,
}

// checkAdminInfoSyntax - validate arguments passed by a user
func checkAdminHealthSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "health", 1) // last argument is exit code
	}
}

//compress and tar obd output
func tarGZ(c HealthReportInfo, alias string) error {
	filename := fmt.Sprintf("%s-health_%s.json.gz", filepath.Clean(alias), time.Now().Format("20060102150405"))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	defer func() {
		console.Infoln("Health data saved at", filename)
	}()

	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	enc := gojson.NewEncoder(gzWriter)

	header := HealthReportHeader{
		Subnet: Health{
			Health: SchemaVersion{
				Version: "v1",
			},
		},
	}

	if err := enc.Encode(header); err != nil {
		return err
	}

	if err := enc.Encode(c); err != nil {
		return err
	}

	warningMsgBoundary := "*********************************************************************************"
	warning := warnText("                                   WARNING!!")
	warningContents := infoText(`     ** THIS FILE MAY CONTAIN SENSITIVE INFORMATION ABOUT YOUR ENVIRONMENT ** 
     ** PLEASE INSPECT CONTENTS BEFORE SHARING IT ON ANY PUBLIC FORUM **`)

	warningMsgHeader := infoText(warningMsgBoundary)
	warningMsgTrailer := infoText(warningMsgBoundary)
	console.Printf("%s\n%s\n%s\n%s\n", warningMsgHeader, warning, warningContents, warningMsgTrailer)

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
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	healthInfo, e := fetchServerHealthInfo(ctx, client)
	clusterHealthInfo := mapHealthInfoToV1(healthInfo, e)

	if globalJSON {
		printMsg(clusterHealthInfo)
		return nil
	}

	if clusterHealthInfo.GetError() != "" {
		console.Println(warnText("unable to obtain health information:"), clusterHealthInfo.GetError())
		return nil
	}

	return tarGZ(clusterHealthInfo, aliasedURL)
}

func fetchServerHealthInfo(ctx *cli.Context, client *madmin.AdminClient) (madmin.HealthInfo, error) {
	opts := GetHealthDataTypeSlice(ctx, "test")
	if len(*opts) == 0 {
		opts = &options
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
	diskHw := spinner("Disk Info", madmin.HealthDataTypeSysDiskHw)
	osInfo := spinner("OS Info", madmin.HealthDataTypeSysOsInfo)
	mem := spinner("Mem Info", madmin.HealthDataTypeSysMem)
	process := spinner("Process Info", madmin.HealthDataTypeSysLoad)
	config := spinner("Server Config", madmin.HealthDataTypeMinioConfig)
	drive := spinner("Drive Test", madmin.HealthDataTypePerfDrive)
	net := spinner("Network Test", madmin.HealthDataTypePerfNet)

	progress := func(info madmin.HealthInfo) {
		_ = admin(len(info.Minio.Info.Servers) > 0) &&
			cpu(len(info.Sys.CPUInfo) > 0) &&
			diskHw(len(info.Sys.DiskHwInfo) > 0) &&
			osInfo(len(info.Sys.OsInfo) > 0) &&
			mem(len(info.Sys.MemInfo) > 0) &&
			process(len(info.Sys.ProcInfo) > 0) &&
			config(info.Minio.Config != nil) &&
			drive(len(info.Perf.DriveInfo) > 0) &&
			net(len(info.Perf.Net) > 1 && len(info.Perf.NetParallel.Addr) > 0)
	}

	var err error
	var healthInfo madmin.HealthInfo

	// Fetch info of all servers (cluster or single server)
	obdChan := client.ServerHealthInfo(cont, *opts, ctx.Duration("deadline"))
	for adminHealthInfo := range obdChan {
		if adminHealthInfo.Error != "" {
			err = errors.New(adminHealthInfo.Error)
			break
		}

		healthInfo = adminHealthInfo
		progress(adminHealthInfo)
	}

	// cancel the context if obdChan has returned.
	cancel()
	return healthInfo, err
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

var options = HealthDataTypeSlice(madmin.HealthDataTypesList)
