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

package cmd

import (
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/pkg/console"
)

var ilmSubcommands = []cli.Command{
	ilmAddCmd,
	ilmEditCmd,
	ilmLsCmd,
	ilmRmCmd,
	ilmExportCmd,
	ilmImportCmd,
}

var ilmCmd = cli.Command{
	Name:            "ilm",
	Usage:           "manage bucket lifecycle",
	Action:          mainILM,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	Subcommands:     ilmSubcommands,
}

const (
	ilmMainHeader         string = "Main-Heading"
	ilmThemeHeader        string = "Row-Header"
	ilmThemeRow           string = "Row-Normal"
	ilmThemeTick          string = "Row-Tick"
	ilmThemeExpiry        string = "Row-Expiry"
	ilmThemeResultSuccess string = "SuccessOp"
	ilmThemeResultFailure string = "FailureOp"
)

func mainILM(ctx *cli.Context) error {
	commandNotFound(ctx, ilmSubcommands)
	return nil
}

// Color scheme for the table
func setILMDisplayColorScheme() {
	console.SetColor(ilmMainHeader, color.New(color.Bold, color.FgHiRed))
	console.SetColor(ilmThemeRow, color.New(color.FgHiWhite))
	console.SetColor(ilmThemeHeader, color.New(color.Bold, color.FgHiGreen))
	console.SetColor(ilmThemeTick, color.New(color.FgGreen))
	console.SetColor(ilmThemeExpiry, color.New(color.BlinkRapid, color.FgGreen))
	console.SetColor(ilmThemeResultSuccess, color.New(color.FgGreen, color.Bold))
	console.SetColor(ilmThemeResultFailure, color.New(color.FgHiYellow, color.Bold))
}
