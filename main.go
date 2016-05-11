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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/pb"
	"github.com/pkg/profile"
)

var (
	// global flags for mc.
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
  {{.Name}} {{if .Flags}}[FLAGS] {{end}}COMMAND{{if .Flags}} [COMMAND FLAGS | -h]{{end}} [ARGUMENTS...]

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
	msg := fmt.Sprintf("‘%s’ is not a mc command. See ‘mc --help’.", command)
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
	// Refresh the config once.
	loadMcConfig = loadMcConfigFactory()
	// Ensures config file is sane.
	config, err := loadMcConfig()
	// Verify if the path is accesible before validating the config
	fatalIf(err.Trace(mustGetMcConfigPath()), "Unable to access configuration file.")

	// Validate and print error messges
	ok, errMsgs := validateConfigFile(config)
	if !ok {
		var errorMsg bytes.Buffer
		for index, errMsg := range errMsgs {
			// Print atmost 10 errors
			if index > 10 {
				break
			}
			errorMsg.WriteString(errMsg + "\n")
		}
		console.Fatalln(errorMsg.String())
	}
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
		pb.Format(int64(memstats.Alloc)).To(pb.U_BYTES),
		pb.Format(int64(memstats.TotalAlloc)).To(pb.U_BYTES),
		pb.Format(int64(memstats.HeapAlloc)).To(pb.U_BYTES),
		pb.Format(int64(memstats.HeapSys)).To(pb.U_BYTES))
	platform := fmt.Sprintf("Host: %s | OS: %s | Arch: %s", host, runtime.GOOS, runtime.GOARCH)
	goruntime := fmt.Sprintf("Version: %s | CPUs: %s", runtime.Version(), strconv.Itoa(runtime.NumCPU()))
	return map[string]string{
		"PLATFORM": platform,
		"RUNTIME":  goruntime,
		"MEM":      mem,
	}
}

// initMC - initialize 'mc'.
func initMC() {
	// Check if mc config exists.
	if !isMcConfigExists() {
		err := saveMcConfig(newMcConfig())
		fatalIf(err.Trace(), "Unable to save new mc config.")

		console.Infoln("Configuration written to ‘" + mustGetMcConfigPath() + "’. Please update your access credentials.")
	}

	// Check if mc session folder exists.
	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session config folder.")
	}

	// Check if mc share folder exists.
	if !isShareDirExists() {
		initShareConfig()
	}
}

func registerBefore(ctx *cli.Context) error {
	// Check if mc was compiled using a supported version of Golang.
	checkGoVersion()

	// Set the config folder.
	setMcConfigDir(ctx.GlobalString("config-folder"))

	// Migrate any old version of config / state files to newer format.
	migrate()

	// Initialize default config files.
	initMC()

	// Set global flags.
	setGlobalsFromContext(ctx)

	// Check if config can be read.
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
	// Suggest other close commands - allow missed, wrongly added and even transposed characters
	for _, value := range commandsTree.walk(commandsTree.root) {
		if sort.SearchStrings(closestCommands, value.(string)) < len(closestCommands) {
			continue
		}
		// 2 is arbitrary and represents the max allowed number of typed errors
		if DamerauLevenshteinDistance(command, value.(string)) < 2 {
			closestCommands = append(closestCommands, value.(string))
		}
	}
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

// mustGetProfilePath must get location that the profile will be written to.
func mustGetProfileDir() string {
	return filepath.Join(mustGetMcConfigDir(), globalProfileDir)
}

func main() {
	// Enable profiling supported modes are [cpu, mem, block].
	// ``MC_PROFILER`` supported options are [cpu, mem, block].
	switch os.Getenv("MC_PROFILER") {
	case "cpu":
		defer profile.Start(profile.CPUProfile, profile.ProfilePath(mustGetProfileDir())).Stop()
	case "mem":
		defer profile.Start(profile.MemProfile, profile.ProfilePath(mustGetProfileDir())).Stop()
	case "block":
		defer profile.Start(profile.BlockProfile, profile.ProfilePath(mustGetProfileDir())).Stop()
	}

	probe.Init() // Set project's root source path.
	probe.SetAppInfo("Release-Tag", mcReleaseTag)
	probe.SetAppInfo("Commit", mcShortCommitID)

	app := registerApp()
	app.Before = registerBefore

	app.ExtraInfo = func() map[string]string {
		if _, e := pb.GetTerminalWidth(); e != nil {
			globalQuiet = true
		}
		if globalDebug {
			return getSystemData()
		}
		return make(map[string]string)
	}

	app.RunAndExitOnError()
}
