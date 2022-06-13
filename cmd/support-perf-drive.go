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
	"context"
	"fmt"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

func mainAdminSpeedtestDrive(ctx *cli.Context, aliasedURL string) error {
	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	blocksize, e := humanize.ParseBytes(ctx.String("blocksize"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse blocksize")
		return nil
	}
	if blocksize < 0 {
		fatalIf(errInvalidArgument(), "blocksize cannot be <= 0")
		return nil
	}

	filesize, e := humanize.ParseBytes(ctx.String("filesize"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse filesize")
		return nil
	}
	if filesize < 0 {
		fatalIf(errInvalidArgument(), "filesize cannot be <= 0")
		return nil
	}

	serial := ctx.Bool("serial")

	resultCh, err := client.DriveSpeedtest(ctxt, madmin.DriveSpeedTestOpts{
		Serial:    serial,
		BlockSize: uint64(blocksize),
		FileSize:  uint64(filesize),
	})
	fatalIf(probe.NewError(err), "Failed to execute drive speedtest")

	for result := range resultCh {
		if result.Version != "" {
			if globalJSON {
				jsonBytes, e := json.MarshalIndent(result, "", " ")
				fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
				fmt.Println(string(jsonBytes))
				continue
			}
			tbl := console.NewTable(func() []*color.Color {
				colors := []*color.Color{
					color.New(color.FgWhite, color.BgBlack, color.Bold),
				}
				for range result.DrivePerf {
					colors = append(colors, color.New(color.FgGreen))
				}
				return colors
			}(), []bool{false, false, false, false}, 0)
			cellText := make([][]string, len(result.DrivePerf)+1)
			cellText[0] = []string{
				"Node",
				"Path",
				"Read",
				"Write",
			}
			trailerIfGreaterThan := func(in string, max int) string {
				if len(in) < max {
					return ""
				}
				return "..."
			}
			for i, item := range result.DrivePerf {
				cellText[i+1] = []string{
					fmt.Sprintf("%.64s%s", result.Endpoint,
						trailerIfGreaterThan(result.Endpoint, 64)),
					fmt.Sprintf("%.64s%s", item.Path,
						trailerIfGreaterThan(result.Endpoint, 64)),
					humanize.IBytes(item.ReadThroughput) + "/s",
					humanize.IBytes(item.WriteThroughput) + "/s",
				}
			}
			if len(result.DrivePerf) > 0 {
				tbl.DisplayTable(cellText)
			}
			if result.Error != "" {
				fmt.Println(color.New(color.FgRed, color.Bold).Sprintf("ERROR"), result.Error)
			}
		}
	}

	return nil
}
