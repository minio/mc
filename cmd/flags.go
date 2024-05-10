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
		Name:   "disable-pager, dp",
		Usage:  "disable mc internal pager and print to raw stdout",
		EnvVar: envPrefix + globalDisablePagerEnv,
		Hidden: false,
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

// bundled encryption flags
var encFlags = []cli.Flag{
	encCFlag,
	encKSMFlag,
	encS3Flag,
}

var encCFlag = cli.StringSliceFlag{
	Name:  "enc-c",
	Usage: "encrypt/decrypt objects using client provided keys. (multiple keys can be provided) Format: Raw base64 encoding.",
}

var encKSMFlag = cli.StringSliceFlag{
	Name:   "enc-kms",
	Usage:  "encrypt/decrypt objects using specific server-side encryption keys. (multiple keys can be provided)",
	EnvVar: envPrefix + "ENC_KMS",
}

var encS3Flag = cli.StringSliceFlag{
	Name:   "enc-s3",
	Usage:  "encrypt/decrypt objects using server-side default keys and configurations. (multiple keys can be provided).",
	EnvVar: envPrefix + "ENC_S3",
}
