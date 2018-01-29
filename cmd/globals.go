/*
 * Minio Client (C) 2015 Minio, Inc.
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

// Package cmd contains all the global variables and constants. ONLY TO BE ACCESSED VIA GET/SET FUNCTIONS.
package cmd

import (
	"crypto/x509"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

// mc configuration related constants.
const (
	minGoVersion = ">= 1.7.5" // mc requires at least Go v1.7.5
)

const (
	globalMCConfigVersion = "9"

	globalMCConfigDir        = ".mc/"
	globalMCConfigWindowsDir = "mc\\"
	globalMCConfigFile       = "config.json"
	globalMCCertsDir         = "certs"
	globalMCCAsDir           = "CAs"

	// session config and shared urls related constants
	globalSessionDir           = "session"
	globalSharedURLsDataDir    = "share"
	globalSessionConfigVersion = "8"

	// Profile directory for dumping profiler outputs.
	globalProfileDir = "profile"

	// Global error exit status.
	globalErrorExitStatus = 1
)

var (
	globalQuiet    = false // Quiet flag set via command line
	globalJSON     = false // Json flag set via command line
	globalDebug    = false // Debug flag set via command line
	globalNoColor  = false // No Color flag set via command line
	globalInsecure = false // Insecure flag set via command line

	// WHEN YOU ADD NEXT GLOBAL FLAG, MAKE SURE TO ALSO UPDATE SESSION CODE AND CODE BELOW.
)

var (
	// Terminal width
	globalTermWidth int

	// CA root certificates, a nil value means system certs pool will be used
	globalRootCAs *x509.CertPool
)

// Set global states. NOTE: It is deliberately kept monolithic to ensure we dont miss out any flags.
func setGlobals(quiet, debug, json, noColor, insecure bool) {
	globalQuiet = globalQuiet || quiet
	globalDebug = globalDebug || debug
	globalJSON = globalJSON || json
	globalNoColor = globalNoColor || noColor
	globalInsecure = globalInsecure || insecure

	// Enable debug messages if requested.
	if globalDebug {
		console.DebugPrint = true
	}

	// Disable colorified messages if requested.
	if globalNoColor || globalQuiet {
		console.SetColorOff()
	}
}

// Set global states. NOTE: It is deliberately kept monolithic to ensure we dont miss out any flags.
func setGlobalsFromContext(ctx *cli.Context) error {
	quiet := ctx.IsSet("quiet")
	debug := ctx.IsSet("debug")
	json := ctx.IsSet("json")
	noColor := ctx.IsSet("no-color")
	insecure := ctx.IsSet("insecure")
	setGlobals(quiet, debug, json, noColor, insecure)
	return nil
}
