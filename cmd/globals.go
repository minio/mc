// Copyright (c) 2015-2022 MinIO, Inc.
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
	"net/url"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
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
	globalQuiet          = false               // Quiet flag set via command line
	globalJSON           = false               // Json flag set via command line
	globalJSONLine       = false               // Print json as single line.
	globalDebug          = false               // Debug flag set via command line
	globalNoColor        = false               // No Color flag set via command line
	globalInsecure       = false               // Insecure flag set via command line
	globalDevMode        = false               // dev flag set via command line
	globalAirgapped      = false               // Airgapped flag set via command line
	globalSubnetProxyURL *url.URL              // Proxy to be used for communication with subnet
	globalSubnetConfig   []madmin.SubsysConfig // Subnet config

	globalConnReadDeadline  time.Duration
	globalConnWriteDeadline time.Duration

	globalLimitUpload   uint64
	globalLimitDownload uint64

	globalContext, globalCancel = context.WithCancel(context.Background())
)

var (
	// Terminal width
	globalTermWidth int

	globalHelpPager *termPager

	// CA root certificates, a nil value means system certs pool will be used
	globalRootCAs *x509.CertPool
)

// Set global states. NOTE: It is deliberately kept monolithic to ensure we dont miss out any flags.
func setGlobalsFromContext(ctx *cli.Context) error {
	quiet := ctx.IsSet("quiet") || ctx.GlobalIsSet("quiet")
	debug := ctx.IsSet("debug") || ctx.GlobalIsSet("debug")
	json := ctx.IsSet("json") || ctx.GlobalIsSet("json")
	noColor := ctx.IsSet("no-color") || ctx.GlobalIsSet("no-color")
	insecure := ctx.IsSet("insecure") || ctx.GlobalIsSet("insecure")
	devMode := ctx.IsSet("dev") || ctx.GlobalIsSet("dev")
	airgapped := ctx.IsSet("airgap") || ctx.GlobalIsSet("airgap")

	globalQuiet = globalQuiet || quiet
	globalDebug = globalDebug || debug
	globalJSONLine = !isTerminal() && json
	globalJSON = globalJSON || json
	globalNoColor = globalNoColor || noColor || globalJSONLine
	globalInsecure = globalInsecure || insecure
	globalDevMode = globalDevMode || devMode
	globalAirgapped = globalAirgapped || airgapped

	// Disable colorified messages if requested.
	if globalNoColor || globalQuiet {
		console.SetColorOff()
	}

	globalConnReadDeadline = ctx.Duration("conn-read-deadline")
	if globalConnReadDeadline <= 0 {
		globalConnReadDeadline = ctx.GlobalDuration("conn-read-deadline")
	}

	globalConnWriteDeadline = ctx.Duration("conn-write-deadline")
	if globalConnWriteDeadline <= 0 {
		globalConnWriteDeadline = ctx.GlobalDuration("conn-write-deadline")
	}

	limitUploadStr := ctx.String("limit-upload")
	if limitUploadStr == "" {
		limitUploadStr = ctx.GlobalString("limit-upload")
	}
	if limitUploadStr != "" {
		var e error
		globalLimitUpload, e = humanize.ParseBytes(limitUploadStr)
		if e != nil {
			return e
		}
	}

	limitDownloadStr := ctx.String("limit-download")
	if limitDownloadStr == "" {
		limitDownloadStr = ctx.GlobalString("limit-download")
	}

	if limitDownloadStr != "" {
		var e error
		globalLimitDownload, e = humanize.ParseBytes(limitDownloadStr)
		if e != nil {
			return e
		}
	}

	return nil
}
