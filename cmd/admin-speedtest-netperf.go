// Copyright (c) 2022 MinIO, Inc.
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
	"strings"
	"time"

	"github.com/briandowns/spinner"
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

type netperfResult madmin.NetperfResult

func (m netperfResult) String() (msg string) {
	for _, r := range m.NodeResults {
		msg += fmt.Sprintf("%s TX: %s/s RX: %s/s", r.Endpoint, humanize.IBytes(uint64(r.TX)), humanize.IBytes(uint64(r.RX)))
		if r.Error != "" {
			msg += " Error: " + r.Error
		}
		msg += "\n"
	}
	msg = strings.TrimSuffix(msg, "\n")
	return msg
}

func (m netperfResult) JSON() string {
	JSONBytes, e := json.MarshalIndent(m, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(JSONBytes)
}

func mainAdminSpeedtestNetperf(ctx *cli.Context, aliasedURL string) error {
	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	duration, e := time.ParseDuration(ctx.String("duration"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse duration")
		return nil
	}
	if duration <= 0 {
		fatalIf(errInvalidArgument(), "duration cannot be 0 or negative")
		return nil
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	if !globalJSON {
		s.Start()
	}

	result, err := client.Netperf(ctxt, duration)
	s.Stop()
	fatalIf(probe.NewError(err), "Failed to execute netperf")
	printMsg(netperfResult(result))
	return nil
}
