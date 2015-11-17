/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

package main

import (
	"encoding/json"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

var (
	configVersionFlagHelp = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Help of config version",
	}
)

// Print config version.
var configVersionCmd = cli.Command{
	Name:   "version",
	Usage:  "Print config version.",
	Action: mainConfigVersion,
	Flags:  []cli.Flag{configVersionFlagHelp},
	CustomHelpTemplate: `NAME:
   mc config {{.Name}} - {{.Usage}}

USAGE:
   mc config {{.Name}}

`,
}

func mainConfigVersion(ctx *cli.Context) {
	if ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "version", 1) // last argument is exit code
	}

	config, err := newConfig()
	fatalIf(err.Trace(), "Failed to initialize ‘quick’ configuration data structure.")

	configPath := mustGetMcConfigPath()
	err = config.Load(configPath)
	fatalIf(err.Trace(configPath), "Unable to load config path")

	// convert interface{} back to its original struct
	newConf := config.Data().(*configV6)
	type Version string
	if globalJSONFlag {
		tB, e := json.Marshal(
			struct {
				Version Version `json:"version"`
			}{Version: Version(newConf.Version)},
		)
		fatalIf(probe.NewError(e), "Unable to construct version string.")
		console.Println(string(tB))
		return
	}
	console.Println(newConf.Version)
}
