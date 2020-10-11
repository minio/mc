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
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminOBDFlags = []cli.Flag{
	OBDDataTypeFlag{
		Name:   "test",
		Usage:  "choose OBD tests to run [" + options.String() + "]",
		Value:  nil,
		EnvVar: "MC_HEALTH_TEST,MC_OBD_TEST",
		Hidden: true,
	},
	cli.DurationFlag{
		Name:   "deadline",
		Usage:  "maximum duration that OBD tests should be allowed to run",
		Value:  3600 * time.Second,
		EnvVar: "MC_HEALTH_DEADLINE,MC_OBD_DEADLINE",
	},
}

var adminOBDCmd = cli.Command{
	Name:    "health",
	Aliases: []string{"obd"},
	Usage:   "run on-board diagnostics",
	Action:  mainAdminOBD,
	Before:  setGlobalsFromContext,
	Flags:   append(adminOBDFlags, globalFlags...),
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

type clusterOBDStruct struct {
	Status string         `json:"status"`
	Error  string         `json:"error,omitempty"`
	Info   madmin.OBDInfo `json:"obdInfo,omitempty"`
}

func (u clusterOBDStruct) String() string {
	return u.JSON()
}

// JSON jsonifies service status message.
func (u clusterOBDStruct) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(u, " ", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminInfoSyntax - validate arguments passed by a user
func checkAdminOBDSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "health", 1) // last argument is exit code
	}
}

//compress and tar obd output
func tarGZ(c clusterOBDStruct, alias string) error {
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

func mainAdminOBD(ctx *cli.Context) error {
	checkAdminOBDSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	opts := GetOBDDataTypeSlice(ctx, "test")
	if len(*opts) == 0 {
		opts = &options
	}

	optsMap := make(map[madmin.OBDDataType]struct{})
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

	spinner := func(resource string, opt madmin.OBDDataType) func(bool) bool {
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

	clusterOBDInfo := clusterOBDStruct{}

	admin := spinner("Admin Info", madmin.OBDDataTypeMinioInfo)
	cpu := spinner("CPU Info", madmin.OBDDataTypeSysCPU)
	diskHw := spinner("Disk Info", madmin.OBDDataTypeSysDiskHw)
	osInfo := spinner("OS Info", madmin.OBDDataTypeSysOsInfo)
	mem := spinner("Mem Info", madmin.OBDDataTypeSysMem)
	process := spinner("Process Info", madmin.OBDDataTypeSysLoad)
	config := spinner("Server Config", madmin.OBDDataTypeMinioConfig)
	log := spinner("Server Log", madmin.OBDDataTypeLog)
	drive := spinner("Drive Test", madmin.OBDDataTypePerfDrive)
	net := spinner("Network Test", madmin.OBDDataTypePerfNet)

	progress := func(info madmin.OBDInfo) {
		_ = log(len(info.Logging.ServersLog) > 0) &&
			admin(len(info.Minio.Info.Servers) > 0) &&
			cpu(len(info.Sys.CPUInfo) > 0) &&
			diskHw(len(info.Sys.DiskHwInfo) > 0) &&
			osInfo(len(info.Sys.OsInfo) > 0) &&
			mem(len(info.Sys.MemInfo) > 0) &&
			process(len(info.Sys.ProcInfo) > 0) &&
			config(info.Minio.Config != nil) &&
			drive(len(info.Perf.DriveInfo) > 0) &&
			net(len(info.Perf.Net) > 1 && len(info.Perf.NetParallel.Addr) > 0)
	}

	// Fetch info of all servers (cluster or single server)
	obdChan := client.ServerOBDInfo(cont, *opts, ctx.Duration("deadline"))
	for adminOBDInfo := range obdChan {
		if adminOBDInfo.Error != "" {
			clusterOBDInfo.Status = "Error"
			clusterOBDInfo.Error = adminOBDInfo.Error
			clusterOBDInfo.Info.Error = ""
			clusterOBDInfo.Info.Minio.Info = madmin.InfoMessage{}
			break
		}

		clusterOBDInfo.Status = "Success"
		clusterOBDInfo.Info = adminOBDInfo
		progress(adminOBDInfo)
	}

	// cancel the context if obdChan has returned.
	cancel()

	if globalJSON {
		printMsg(clusterOBDInfo)
		return nil
	}

	if clusterOBDInfo.Error != "" {
		console.Println(warnText("unable to obtain health information:"), clusterOBDInfo.Error)
		return nil
	}

	return tarGZ(clusterOBDInfo, aliasedURL)
}

// OBDDataTypeSlice is a typed list of OBD tests
type OBDDataTypeSlice []madmin.OBDDataType

// Set - sets the flag to the given value
func (d *OBDDataTypeSlice) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		if obdData, ok := madmin.OBDDataTypesMap[strings.Trim(v, " ")]; ok {
			*d = append(*d, obdData)
		} else {
			return fmt.Errorf("valid options include %s", options.String())
		}
	}
	return nil
}

// String - returns the string representation of the OBD datatypes
func (d *OBDDataTypeSlice) String() string {
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
func (d *OBDDataTypeSlice) Value() []madmin.OBDDataType {
	return *d
}

// Get - returns the value
func (d *OBDDataTypeSlice) Get() interface{} {
	return *d
}

// OBDDataTypeFlag is a typed flag to represent OBD datatypes
type OBDDataTypeFlag struct {
	Name   string
	Usage  string
	EnvVar string
	Hidden bool
	Value  *OBDDataTypeSlice
}

// String - returns the string to be shown in the help message
func (f OBDDataTypeFlag) String() string {
	return fmt.Sprintf("--%s                       %s", f.Name, f.Usage)
}

// GetName - returns the name of the flag
func (f OBDDataTypeFlag) GetName() string {
	return f.Name
}

// GetOBDDataTypeSlice - returns the list of set OBD tests
func GetOBDDataTypeSlice(c *cli.Context, name string) *OBDDataTypeSlice {
	generic := c.Generic(name)
	if generic == nil {
		return nil
	}
	return generic.(*OBDDataTypeSlice)
}

// GetGlobalOBDDataTypeSlice - returns the list of set OBD tests set globally
func GetGlobalOBDDataTypeSlice(c *cli.Context, name string) *OBDDataTypeSlice {
	generic := c.GlobalGeneric(name)
	if generic == nil {
		return nil
	}
	return generic.(*OBDDataTypeSlice)
}

// Apply - applies the flag
func (f OBDDataTypeFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

// ApplyWithError - applies with error
func (f OBDDataTypeFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				newVal := &OBDDataTypeSlice{}
				for _, s := range strings.Split(envVal, ",") {
					s = strings.TrimSpace(s)
					if err := newVal.Set(s); err != nil {
						return fmt.Errorf("could not parse %s as OBD datatype value for flag %s: %s", envVal, f.Name, err)
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
			f.Value = &OBDDataTypeSlice{}
		}
		set.Var(f.Value, name, f.Usage)
	}
	return nil
}

var options = OBDDataTypeSlice(madmin.OBDDataTypesList)
