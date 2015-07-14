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
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/pb"
)

// Check for the environment early on and gracefully report.
func checkConfig() {
	_, err := user.Current()
	if err != nil {
		console.Fatalf("Unable to determine current user. %s\n", err)
	}

	if !isSessionDirExists() {
		if err := createSessionDir(); err != nil {
			console.Fatalf("Unable to create session directory. %s\n", err)
		}
	}

	// If config doesn't exist, do not attempt to read it
	if !isMcConfigExists() {
		return
	}

	// Ensures config file is sane
	_, err = getMcConfig()
	if err != nil {
		console.Fatalf("Unable to read config file. %s\n", err)
	}

}

// Get os/arch/platform specific information.
// Returns a map of current os/arch/platform/memstats
func getSystemData() map[string]string {
	host, err := os.Hostname()
	if err != nil {
		console.Fatalf("Unable to determine hostname. %s\n", err)
	}
	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)
	mem := fmt.Sprintf("Used: %s | Allocated: %s | Used-Heap: %s | Allocated-Heap: %s",
		pb.FormatBytes(int64(memstats.Alloc)),
		pb.FormatBytes(int64(memstats.TotalAlloc)),
		pb.FormatBytes(int64(memstats.HeapAlloc)),
		pb.FormatBytes(int64(memstats.HeapSys)))
	platform := fmt.Sprintf("Host: %s | OS: %s | Arch: %s", host, runtime.GOOS, runtime.GOARCH)
	goruntime := fmt.Sprintf("Version: %s | CPUs: %s", runtime.Version(), strconv.Itoa(runtime.NumCPU()))
	return map[string]string{
		"PLATFORM": platform,
		"RUNTIME":  goruntime,
		"MEM":      mem,
		"TAG":      Tag,
	}
}

func migrate() {
	// Migrate session files if any.
	migrateSession()
}

func main() {
	// Migrate any old version of config / state files to newer format.
	migrate()

	// Enable GOMAXPROCS to default to number of CPUs.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Register all the commands
	registerCmd(lsCmd)      // List contents of a bucket
	registerCmd(mbCmd)      // make a bucket
	registerCmd(catCmd)     // concantenate an object to standard output
	registerCmd(cpCmd)      // copy objects and files from multiple sources to single destination
	registerCmd(syncCmd)    // sync objects and files from single source to multiple destinations
	registerCmd(sessionCmd) // session handling for resuming copy and sync operations
	registerCmd(diffCmd)    // compare two objects
	registerCmd(accessCmd)  // set permissions [public, private, readonly, authenticated] for buckets and folders.
	registerCmd(configCmd)  // generate configuration "/home/harsha/.mc/config.json" file.
	registerCmd(updateCmd)  // update Check for new software updates

	// register all the flags
	registerFlag(configFlag) // path to config folder
	registerFlag(quietFlag)  // suppress console output
	registerFlag(aliasFlag)  // OS toolchain mimic
	registerFlag(themeFlag)  // console theme flag
	registerFlag(jsonFlag)   // json formatted output
	registerFlag(debugFlag)  // enable debugging output

	app := cli.NewApp()
	app.Usage = "Minio Client for object storage and filesystems"
	app.Version = Version
	app.Commands = commands
	app.Compiled = getVersion()
	app.Flags = flags
	app.Author = "Minio.io"
	app.Before = func(ctx *cli.Context) error {
		if ctx.GlobalString("config") != "" {
			setMcConfigDir(ctx.GlobalString("config"))
		}

		globalQuietFlag = ctx.GlobalBool("quiet")
		globalAliasFlag = ctx.GlobalBool("alias")
		globalDebugFlag = ctx.GlobalBool("debug")
		globalJSONFlag = ctx.GlobalBool("json")
		if globalDebugFlag {
			app.ExtraInfo = getSystemData()
			console.NoDebugPrint = false
		}
		themeName := ctx.GlobalString("theme")
		switch {
		case console.IsValidTheme(themeName) != true:
			console.Errorf("Invalid theme, please choose from the following list: %s.\n", console.GetThemeNames())
			return errInvalidTheme{Theme: themeName}
		default:
			err := console.SetTheme(themeName)
			if err != nil {
				console.Errorf("Failed to set theme ‘%s’.", themeName)
				return err
			}
		}
		checkConfig()
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		if !isMcConfigExists() {
			console.Fatalf("Please run \"mc config generate\". %s\n", errNotConfigured{})
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
  {{if .Compiled}}
  {{.Compiled}}{{end}}
  {{range $key, $value := .ExtraInfo}}
{{$key}}:
  {{$value}}
{{end}}
`
	app.RunAndExitOnError()
}
