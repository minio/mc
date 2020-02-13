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
	"flag"
	"fmt"
	"strings"
	"syscall"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminOBDFlags = []cli.Flag{
	OBDDataTypeFlag{
		Name:   "data",
		Usage:  "diagnostics type, possible values are " + options.String() + " (default $all)",
		Value:  nil,
		EnvVar: "MINIO_OBD_DATA",
	},
}

var adminOBDCmd = cli.Command{
	Name:   "obd",
	Usage:  "run on-board diagnostics",
	Action: mainAdminOBD,
	Before: setGlobalsFromContext,
	Flags:  append(adminOBDFlags, globalFlags...),
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
	data, err := json.Marshal(u)
	if err != nil {
		fatalIf(probe.NewError(err), "unable to marshal into JSON.")
	}
	return string(data)
}

// JSON jsonifies service status message.
func (u clusterOBDStruct) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(u, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminInfoSyntax - validate arguments passed by a user
func checkAdminOBDSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "obd", 1) // last argument is exit code
	}
}

func mainAdminOBD(ctx *cli.Context) error {
	checkAdminOBDSyntax(ctx)

	types := GetOBDDataTypeSlice(ctx, "data")

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if len(*types) == 0 {
		types = &options
	}

	var clusterOBDInfo clusterOBDStruct
	// Fetch info of all servers (cluster or single server)
	adminOBDInfo, e := client.ServerOBDInfo(*types)
	if e != nil {
		clusterOBDInfo.Status = "error"
		clusterOBDInfo.Error = e.Error()
	} else {
		clusterOBDInfo.Status = "success"
		clusterOBDInfo.Error = ""
	}
	clusterOBDInfo.Info = adminOBDInfo
	
	printMsg(clusterOBDStruct(clusterOBDInfo))

	return nil
}

type OBDDataTypeSlice []madmin.OBDDataType

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

func (d *OBDDataTypeSlice) Value() []madmin.OBDDataType {
	return *d
}

func (d *OBDDataTypeSlice) Get() interface{} {
	return *d
}

type OBDDataTypeFlag struct {
	Name   string
	Usage  string
	EnvVar string
	Hidden bool
	Value  *OBDDataTypeSlice
}

func (f OBDDataTypeFlag) String() string {
	return fmt.Sprintf("--%s                        %s", f.Name, f.Usage)
}

func (f OBDDataTypeFlag) GetName() string {
	return f.Name
}

func GetOBDDataTypeSlice(c *cli.Context, name string) *OBDDataTypeSlice {
	generic := c.Generic(name)
	if generic == nil {
		return nil
	}
	if obdData, ok := generic.(*OBDDataTypeSlice); ok {
		return obdData
	}
	return nil
}

func GetGlobalOBDDataTypeSlice(c *cli.Context, name string) *OBDDataTypeSlice {
	generic := c.GlobalGeneric(name)
	if generic == nil {
		return nil
	}
	if obdData, ok := generic.(*OBDDataTypeSlice); ok {
		return obdData
	}
	return nil
}

func (f OBDDataTypeFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

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
