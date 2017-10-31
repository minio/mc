/*
 * Minio Client (C) 2014, 2015, 2016, 2017 Minio, Inc.
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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/words"
	"github.com/pkg/profile"
)

var (
	// global flags for mc.
	mcFlags = []cli.Flag{}
)

// Help template for mc
var mcHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

USAGE:
  {{.Name}} {{if .VisibleFlags}}[FLAGS] {{end}}COMMAND{{if .VisibleFlags}} [COMMAND FLAGS | -h]{{end}} [ARGUMENTS...]

COMMANDS:
  {{range .VisibleCommands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .VisibleFlags}}
GLOBAL FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
VERSION:
  ` + Version +
	`{{ "\n"}}{{range $key, $value := ExtraInfo}}
{{$key}}:
  {{$value}}
{{end}}`

// Main starts mc application
func Main() {
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
	probe.SetAppInfo("Release-Tag", ReleaseTag)
	probe.SetAppInfo("Commit", ShortCommitID)

	// Fetch terminal size, if not available, automatically
	// set globalQuiet to true.
	if w, e := pb.GetTerminalWidth(); e != nil {
		globalQuiet = true
	} else {
		globalTermWidth = w
	}

	app := registerApp()
	app.Before = registerBefore
	app.ExtraInfo = func() map[string]string {
		if globalDebug {
			return getSystemData()
		}
		return make(map[string]string)
	}
	app.RunAndExitOnError()
}

// Function invoked when invalid command is passed.
func commandNotFound(ctx *cli.Context, command string) {
	msg := fmt.Sprintf("`%s` is not a mc command. See `mc --help`.", command)
	closestCommands := findClosestCommands(command)
	if len(closestCommands) > 0 {
		msg += fmt.Sprintf("\n\nDid you mean one of these?\n")
		if len(closestCommands) == 1 {
			cmd := closestCommands[0]
			msg += fmt.Sprintf("        `%s`", cmd)
		} else {
			for _, cmd := range closestCommands {
				msg += fmt.Sprintf("        `%s`\n", cmd)
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
		console.Fatal(errorMsg.String())
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

		if !globalQuiet && !globalJSON {
			console.Infoln("Configuration written to `" + mustGetMcConfigPath() + "`. Please update your access credentials.")
		}
	}

	// Check if mc session folder exists.
	if !isSessionDirExists() {
		fatalIf(createSessionDir().Trace(), "Unable to create session config folder.")
	}

	// Check if mc share folder exists.
	if !isShareDirExists() {
		initShareConfig()
	}

	// Check if certs dir exists
	if !isCertsDirExists() {
		fatalIf(createCertsDir().Trace(), "Unable to create `CAs` folder.")
	}

	// Check if CAs dir exists
	if !isCAsDirExists() {
		fatalIf(createCAsDir().Trace(), "Unable to create `CAs` folder.")
	}

	// Load all authority certificates present in CAs dir
	loadRootCAs()
}

func registerBefore(ctx *cli.Context) error {
	// Check if mc was compiled using a supported version of Golang.
	checkGoVersion()

	// Set the config folder.
	setMcConfigDir(ctx.GlobalString("config-folder"))

	// Migrate any old version of config / state files to newer format.
	migrate()

	// Set global flags.
	setGlobalsFromContext(ctx)

	// Initialize default config files.
	initMC()

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
	for _, value := range commandsTree.Walk(commandsTree.Root()) {
		if sort.SearchStrings(closestCommands, value.(string)) < len(closestCommands) {
			continue
		}
		// 2 is arbitrary and represents the max allowed number of typed errors
		if words.DamerauLevenshteinDistance(command, value.(string)) < 2 {
			closestCommands = append(closestCommands, value.(string))
		}
	}
	return closestCommands
}

// Check for updates and print a notification message
func checkUpdate(ctx *cli.Context) {
	// Do not print update messages, if quiet flag is set.
	if ctx.Bool("quiet") || ctx.GlobalBool("quiet") {
		older, downloadURL, err := getUpdateInfo(1 * time.Second)
		if err != nil {
			// Its OK to ignore any errors during getUpdateInfo() here.
			return
		}
		if older > time.Duration(0) {
			console.Println(colorizeUpdateMessage(downloadURL, older))
		}
	}
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
	registerCmd(findCmd)    // Find specific String patterns
	registerCmd(statCmd)    // Stat contents of a bucket/object
	registerCmd(diffCmd)    // Computer differences between two files or folders.
	registerCmd(rmCmd)      // Remove a file or bucket
	registerCmd(eventsCmd)  // Add events cmd
	registerCmd(watchCmd)   // Add watch cmd
	registerCmd(policyCmd)  // Set policy permissions.
	registerCmd(adminCmd)   // Manage Minio servers
	registerCmd(sessionCmd) // Manage sessions for copy and mirror.
	registerCmd(configCmd)  // Configure minio client.
	registerCmd(updateCmd)  // Check for new software updates.
	registerCmd(versionCmd) // Print version.

	cli.HelpFlag = cli.BoolFlag{
		Name:  "help, h",
		Usage: "Show help.",
	}

	cli.BashCompletionFlag = cli.BoolFlag{
		Name:   "compgen",
		Usage:  "Enables bash-completion for all commands and subcommands.",
		Hidden: true,
	}

	app := cli.NewApp()
	app.Action = func(ctx *cli.Context) {
		if strings.HasPrefix(Version, "RELEASE.") {
			// Check for new updates from dl.minio.io.
			checkUpdate(ctx)
		}
		cli.ShowAppHelp(ctx)
	}

	app.HideVersion = true
	app.HideHelpCommand = true
	app.Usage = "Minio Client for cloud storage and filesystems."
	app.Commands = commands
	app.Author = "Minio.io"
	app.Version = Version
	app.Flags = append(mcFlags, globalFlags...)
	app.CustomAppHelpTemplate = mcHelpTemplate
	app.CommandNotFound = commandNotFound // handler function declared above.
	app.EnableBashCompletion = true

	return app
}

// mustGetProfilePath must get location that the profile will be written to.
func mustGetProfileDir() string {
	return filepath.Join(mustGetMcConfigDir(), globalProfileDir)
}
