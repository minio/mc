// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package cmd contains all the global variables and constants. ONLY TO BE ACCESSED VIA GET/SET FUNCTIONS.
package cmd

import (
	"context"
	"crypto/x509"

	"github.com/minio/cli"
	"github.com/minio/pkg/console"
)

const (
	globalMCConfigVersion = "10"

	globalMCConfigFile = "config.json"
	globalMCCertsDir   = "certs"
	globalMCCAsDir     = "CAs"

	// session config and shared urls related constants
	globalSessionDir           = "session"
	globalSharedURLsDataDir    = "share"
	globalSessionConfigVersion = "8"

	// Profile directory for dumping profiler outputs.
	globalProfileDir = "profile"

	// Global error exit status.
	globalErrorExitStatus = 1

	// Global CTRL-C (SIGINT, #2) exit status.
	globalCancelExitStatus = 130

	// Global SIGKILL (#9) exit status.
	globalKillExitStatus = 137

	// Global SIGTERM (#15) exit status
	globalTerminatExitStatus = 143
)

var (
	globalQuiet    = false // Quiet flag set via command line
	globalJSON     = false // Json flag set via command line
	globalJSONLine = false // Print json as single line.
	globalDebug    = false // Debug flag set via command line
	globalNoColor  = false // No Color flag set via command line
	globalInsecure = false // Insecure flag set via command line

	globalContext, globalCancel = context.WithCancel(context.Background())
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
	globalJSONLine = !isTerminal() && json
	globalJSON = globalJSON || json
	globalNoColor = globalNoColor || noColor || globalJSONLine
	globalInsecure = globalInsecure || insecure

	// Disable colorified messages if requested.
	if globalNoColor || globalQuiet {
		console.SetColorOff()
	}
}

// Set global states. NOTE: It is deliberately kept monolithic to ensure we dont miss out any flags.
func setGlobalsFromContext(ctx *cli.Context) error {
	quiet := ctx.IsSet("quiet") || ctx.GlobalIsSet("quiet")
	debug := ctx.IsSet("debug") || ctx.GlobalIsSet("debug")
	json := ctx.IsSet("json") || ctx.GlobalIsSet("json")
	noColor := ctx.IsSet("no-color") || ctx.GlobalIsSet("no-color")
	insecure := ctx.IsSet("insecure") || ctx.GlobalIsSet("insecure")
	setGlobals(quiet, debug, json, noColor, insecure)
	return nil
}
