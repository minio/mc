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
	"errors"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// Check for the environment early on and gracefully report.
func checkConfig() {
	_, err := user.Current()
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: "Unable to determine current user",
			Error:   err,
		})
	}

	// If config doesn't exist, do not attempt to read it
	if !isMcConfigExist() {
		return
	}

	// Ensures config file is sane
	_, err = getMcConfig()
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unable to read config file: %s", mustGetMcConfigPath()),
			Error:   err,
		})
	}
}

// Build date
var BuildDate string

// getBuildDate -
func getBuildDate() string {
	t, _ := time.Parse(time.RFC3339Nano, BuildDate)
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC1123)
}

// Get os/arch/platform specific information.
// Returns a map of current os/arch/platform/memstats
func getSystemData() map[string]string {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)
	mem := fmt.Sprintf("Used: %s | Allocated: %s | Used-Heap: %s | Allocated-Heap: %s",
		pb.FormatBytes(int64(memstats.Alloc)),
		pb.FormatBytes(int64(memstats.TotalAlloc)),
		pb.FormatBytes(int64(memstats.HeapAlloc)),
		pb.FormatBytes(int64(memstats.HeapSys)))
	platform := fmt.Sprintf("Host: %s | OS: %s | Arch: %s",
		host,
		runtime.GOOS,
		runtime.GOARCH)
	goruntime := fmt.Sprintf("Version: %s | CPUs: %s", runtime.Version(), strconv.Itoa(runtime.NumCPU()))
	return map[string]string{
		"PLATFORM": platform,
		"RUNTIME":  goruntime,
		"MEM":      mem,
	}
}

// Version is based on MD5SUM of its binary
var Version = mustHashBinarySelf()

func main() {
	app := cli.NewApp()
	app.Usage = "Minio Client for object storage and filesystems"
	app.Version = Version
	app.Commands = commands
	app.Compiled = getBuildDate()
	app.Flags = flags
	app.Author = "Minio.io"
	app.Before = func(ctx *cli.Context) error {
		globalQuietFlag = ctx.GlobalBool("quiet")
		globalDebugFlag = ctx.GlobalBool("debug")
		globalMaxRetryFlag = ctx.GlobalInt("retry")
		globalJSONFlag = ctx.GlobalBool("json")
		if globalDebugFlag {
			app.ExtraInfo = getSystemData()
			console.NoDebugPrint = false
		}
		if globalJSONFlag {
			console.NoJSONPrint = false
		}
		themeName := ctx.GlobalString("theme")
		switch {
		case console.IsValidTheme(themeName) != true:
			console.Fatals(ErrorMessage{
				Message: fmt.Sprintf("Please choose from this list: %s.", console.GetThemeNames()),
				Error:   fmt.Errorf("Theme ‘%s’ is not supported.", themeName),
			})
		default:
			err := console.SetTheme(themeName)
			if err != nil {
				console.Fatals(ErrorMessage{
					Message: fmt.Sprintf("Failed to set theme ‘%s’.", themeName),
					Error:   err,
				})
			}
		}
		checkConfig()
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		if !isMcConfigExist() {
			console.Fatals(ErrorMessage{
				Message: "Please run \"mc config generate\"",
				Error:   iodine.New(errors.New("\"mc\" is not configured"), nil),
			})
		}
		return nil
	}
	app.CustomAppHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

USAGE:
  {{.Name}} {{if .Flags}}[global flags] {{end}}command{{if .Flags}} [command flags]{{end}} [arguments...]

COMMANDS:
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .Flags}}
GLOBAL FLAGS:
  {{range .Flags}}{{.}}
  {{end}}{{end}}
VERSION:
  {{.Version}}
  {{if .Compiled}}
BUILD:
  {{.Compiled}}{{end}}
  {{range $key, $value := .ExtraInfo}}
{{$key}}:
  {{$value}}
{{end}}
`
	app.RunAndExitOnError()
}
