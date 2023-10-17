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

package cmd

import (
	"time"

	"github.com/minio/cli"
)

const envPrefix = "MC_"

// Collection of mc flags currently supported
var globalFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "config-dir, C",
		Value:  mustGetMcConfigDir(),
		Usage:  "path to configuration folder",
		EnvVar: envPrefix + "CONFIG_DIR",
	},
	cli.BoolFlag{
		Name:   "quiet, q",
		Usage:  "disable progress bar display",
		EnvVar: envPrefix + "QUIET",
	},
	cli.BoolFlag{
		Name:   "no-color",
		Usage:  "disable color theme",
		EnvVar: envPrefix + "NO_COLOR",
	},
	cli.BoolFlag{
		Name:   "json",
		Usage:  "enable JSON lines formatted output",
		EnvVar: envPrefix + "JSON",
	},
	cli.BoolFlag{
		Name:   "debug",
		Usage:  "enable debug output",
		EnvVar: envPrefix + "DEBUG",
	},
	cli.BoolFlag{
		Name:   "insecure",
		Usage:  "disable SSL certificate verification",
		EnvVar: envPrefix + "INSECURE",
	},
	cli.StringFlag{
		Name:   "limit-upload",
		Usage:  "limits uploads to a maximum rate in KiB/s, MiB/s, GiB/s. (default: unlimited)",
		EnvVar: envPrefix + "LIMIT_UPLOAD",
	},
	cli.StringFlag{
		Name:   "limit-download",
		Usage:  "limits downloads to a maximum rate in KiB/s, MiB/s, GiB/s. (default: unlimited)",
		EnvVar: envPrefix + "LIMIT_DOWNLOAD",
	},
	cli.DurationFlag{
		Name:   "conn-read-deadline",
		Usage:  "custom connection READ deadline",
		Hidden: true,
		Value:  10 * time.Minute,
	},
	cli.DurationFlag{
		Name:   "conn-write-deadline",
		Usage:  "custom connection WRITE deadline",
		Hidden: true,
		Value:  10 * time.Minute,
	},
}

// Flags common across all I/O commands such as cp, mirror, stat, pipe etc.
var ioFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "encrypt-key",
		Usage:  "encrypt/decrypt objects (using server-side encryption with customer provided keys)",
		EnvVar: envPrefix + "ENCRYPT_KEY",
	},
	cli.StringFlag{
		Name:   "encrypt",
		Usage:  "encrypt/decrypt objects (using server-side encryption with server managed keys)",
		EnvVar: envPrefix + "ENCRYPT",
	},
}
