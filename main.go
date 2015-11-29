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
	"runtime"
	"sort"
	"strconv"

	"github.com/minio/cli"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/pb"
	"github.com/olekukonko/ts"
)

var (
	mcFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Show help.",
		},
	}
)

// Help template for mc
var mcHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

USAGE:
  {{.Name}} {{if .Flags}}[FLAGS] {{end}}COMMAND{{if .Flags}} [COMMAND FLAGS]{{end}} [ARGUMENTS...]

COMMANDS:
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .Flags}}
GLOBAL FLAGS:
  {{range .Flags}}{{.}}
  {{end}}{{end}}
VERSION:
  ` + mcVersion +
	`{{ "\n"}}{{range $key, $value := ExtraInfo}}
{{$key}}:
  {{$value}}
{{end}}`

// Function invoked when invalid command is passed.
func commandNotFound(ctx *cli.Context, command string) {
	msg := fmt.Sprintf("‘%s’ is not a mc command. See ‘mc help’.", command)
	closestCommands := findClosestCommands(command)
	if len(closestCommands) > 0 {
		msg += fmt.Sprintf("\n\nDid you mean one of these?\n")
		if len(closestCommands) == 1 {
			cmd := closestCommands[0]
			msg += fmt.Sprintf("        ‘%s’", cmd)
		} else {
			for _, cmd := range closestCommands {
				msg += fmt.Sprintf("        ‘%s’\n", cmd)
			}
		}
	}
	fatalIf(errDummy().Trace(), msg)
}

// Check for sane config environment early on and gracefully report.
func checkConfig() {
	// Ensures config file is sane.
	_, err := loadMcConfig()
	fatalIf(err.Trace(mustGetMcConfigPath()), "Unable to access configuration file.")
}

func migrate() {
	// Fix broken config files if any.
	fixConfig()
	// Migrate config files if any.
	migrateConfig()
	// Migrate session files if any.
	migrateSession()
	// Migrate shared urls if any.
	migrateShare()
}

// Get os/arch/platform specific information.
// Returns a map of current os/arch/platform/memstats.
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

	// Verify golang runtime.
	verifyMCRuntime()

	// Migrate any old version of config / state files to newer format.
	migrate()

	// Set global flags.
	setGlobalsFromContext(ctx)

	// Checkconfig if it can be read.
	checkConfig()

	return nil
}

// findClosestCommands to match a given string with commands trie tree.
func findClosestCommands(command string) []string {
	var closestCommands []string
	for _, value := range commandsTree.PrefixMatch(command) {
		closestCommands = append(closestCommands, value.(string))
	}
	sort.Strings(closestCommands)
	return closestCommands
}

func registerApp() *cli.App {
	// Register all the commands (refer flags.go)
	registerCmd(lsCmd)      // List contents of a bucket.
	registerCmd(mbCmd)      // Make a bucket.
	registerCmd(catCmd)     // Display contents of a file.
	registerCmd(pipeCmd)    // Write contents of stdin to a file.
	registerCmd(shareCmd)   // Share documents via URL.
	registerCmd(cpCmd)      // Copy objects and files from multiple sources to single destination.
	registerCmd(mirrorCmd)  // Mirror objects and files from single source to multiple destinations.
	registerCmd(diffCmd)    // Computer differences between two files or folders.
	registerCmd(rmCmd)      // Remove a file or bucket
	registerCmd(accessCmd)  // Set access permissions.
	registerCmd(sessionCmd) // Manage sessions for copy and mirror.
	registerCmd(configCmd)  // Configure minio client.
	registerCmd(updateCmd)  // Check for new software updates.
	registerCmd(versionCmd) // Print version.

	app := cli.NewApp()
	app.Usage = "Minio Client for cloud storage and filesystems."
	app.Commands = commands
	app.Author = "Minio.io"
	app.Flags = append(mcFlags, globalFlags...)
	app.CustomAppHelpTemplate = mcHelpTemplate
	app.CommandNotFound = commandNotFound // handler function declared above.
	return app
}

func main() {
	probe.Init() // Set project's root source path.
	probe.SetAppInfo("Release-Tag", mcReleaseTag)
	probe.SetAppInfo("Commit", mcShortCommitID)

	app := registerApp()
	app.Before = registerBefore

	app.ExtraInfo = func() map[string]string {
		if _, e := ts.GetSize(); e != nil {
			globalQuiet = true
		}
		if globalDebug {
			return getSystemData()
		}
		return make(map[string]string)
	}

	app.RunAndExitOnError()
}
