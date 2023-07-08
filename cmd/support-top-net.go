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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var supportTopNetFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "count, c",
		Usage: "show up to N Nets",
		Value: 10,
	},
}

var supportTopNetCmd = cli.Command{
	Name:            "net",
	Aliases:         []string{"network"},
	HiddenAliases:   true,
	Usage:           "show real-time net metrics",
	Action:          mainSupportTopNet,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportTopNetFlags, supportGlobalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display net metrics
      {{.Prompt}} {{.HelpName}} myminio/
`,
}

// checkSupportTopNetSyntax - validate all the passed arguments
func checkSupportTopNetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainSupportTopNet(ctx *cli.Context) error {
	checkSupportTopNetSyntax(ctx)

	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)
	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	info, e := client.ServerInfo(ctxt)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to initialize admin client.")

	var endpoint []string
	for _, srv := range info.Servers {
		endpoint = append(endpoint, srv.Endpoint)
	}

	// MetricsOptions are options provided to Metrics call.
	opts := madmin.MetricsOptions{
		Type:     madmin.MetricNet,
		Interval: time.Second,
		ByDisk:   true,
		N:        ctx.Int("count"),
	}

	p := tea.NewProgram(initTopNetUI(endpoint, ctx.Int("count")))
	go func() {
		out := func(m madmin.RealtimeMetrics) {
			for _, metric := range m.ByNet {
				p.Send(topNetResult{
					endPoint: metric.EndPoint,
					stats:    metric,
				})
			}
		}

		e := client.Metrics(ctxt, opts, out)
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to fetch top net events")
		}
		p.Quit()
	}()

	if _, e := p.Run(); e != nil {
		cancel()
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch top net events")
	}

	return nil
}
