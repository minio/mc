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
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/pb"
)

// Check for the environment early on and gracefully report.
func checkConfig() {
	_, e := user.Current()
	fatalIf(probe.NewError(e), "Unable to determine current user.")

	// Ensures config file is sane
	_, err := getMcConfig()
	fatalIf(err.Trace(), "Unable to access configuration file.")
}

func migrate() {
	// Migrate config files if any.
	migrateConfig()
	// Fix broken config files if any.
	fixConfig()
	// Migrate session files if any.
	migrateSession()
}

// Get os/arch/platform specific information.
// Returns a map of current os/arch/platform/memstats
func getSystemData() map[string]string {
	host, e := os.Hostname()
	fatalIf(probe.NewError(e), "Unable to determine the hostname.")

	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)
	mem := fmt.Sprintf("Used: %s | Allocated: %s | UsedHeap: %s | AllocatedHeap: %s",
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
	}
}

func registerBefore(ctx *cli.Context) error {
	setMcConfigDir(ctx.GlobalString("config-folder"))
	globalQuietFlag = ctx.GlobalBool("quiet")
	globalMimicFlag = ctx.GlobalBool("mimic")
	globalDebugFlag = ctx.GlobalBool("debug")
	globalJSONFlag = ctx.GlobalBool("json")
	if globalDebugFlag {
		console.NoDebugPrint = false
	}
	if ctx.GlobalBool("nocolor") {
		console.SetNoColor()
	}

	verifyMCRuntime()

	// Migrate any old version of config / state files to newer format.
	migrate()

	checkConfig()
	return nil
}

// getFormattedVersion -
func getFormattedVersion() string {
	t, _ := time.Parse(time.RFC3339Nano, Version)
	if t.IsZero() {
		return ""
	}
	return t.Format(http.TimeFormat)
}

func registerApp() *cli.App {
	// Register all the commands
	registerCmd(lsCmd)      // List contents of a bucket.
	registerCmd(mbCmd)      // Make a bucket.
	registerCmd(catCmd)     // Display contents of a file.
	registerCmd(cpCmd)      // Copy objects and files from multiple sources to single destination.
	registerCmd(mirrorCmd)  // Mirror objects and files from single source to multiple destinations.
	registerCmd(sessionCmd) // Manage sessions for copy and mirror.
	registerCmd(shareCmd)   // Share documents via URL.
	registerCmd(diffCmd)    // Computer differences between two files or folders.
	registerCmd(accessCmd)  // Set access permissions.
	registerCmd(configCmd)  // Configure minio client.
	registerCmd(updateCmd)  // Check for new software updates.
	registerCmd(versionCmd) // Print version.

	// register all the flags
	registerFlag(configFlag)  // Path to configuration folder.
	registerFlag(quietFlag)   // Suppress chatty console output.
	registerFlag(mimicFlag)   // Behave like operating system tools. Use with shell aliases.
	registerFlag(jsonFlag)    // Enable json formatted output.
	registerFlag(debugFlag)   // Enable debugging output.
	registerFlag(noColorFlag) // Disable console coloring.

	app := cli.NewApp()
	app.Usage = "Minio Client for cloud storage and filesystems."
	// hide --version flag, version is a command
	app.HideVersion = true
	app.Commands = commands
	app.Flags = flags
	app.Author = "Minio.io"
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
  ` + getFormattedVersion() +
		`{{ "\n"}} {{range $key, $value := ExtraInfo}}
{{$key}}:
  {{$value}}
{{end}}`
	app.CommandNotFound = func(ctx *cli.Context, command string) {
		fatalIf(errDummy().Trace(), fmt.Sprintf("Command ‘%s’ not found.", command))
	}
	return app
}

func main() {
	app := registerApp()
	app.Before = registerBefore

	app.ExtraInfo = func() map[string]string {
		if globalDebugFlag {
			return getSystemData()
		}
		return make(map[string]string)
	}

	app.RunAndExitOnError()
}
