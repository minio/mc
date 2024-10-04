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
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

const (
	healthCheckInterval = 5 * time.Second
)

var readyFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "cluster-read",
		Usage: "check if the cluster has enough read quorum",
	},
	cli.BoolFlag{
		Name:  "maintenance",
		Usage: "check if the cluster is taken down for maintenance",
	},
}

// Checks if the cluster is ready or not
var readyCmd = cli.Command{
	Name:         "ready",
	Usage:        "checks if the cluster is ready or not",
	Action:       mainReady,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(readyFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
EXAMPLES:
  1. Check if the cluster is ready or not
     {{.Prompt}} {{.HelpName}} myminio

  2. Check if the cluster has enough read quorum
     {{.Prompt}} {{.HelpName}} myminio --cluster-read

  3. Check if the cluster is taken down for maintenance
     {{.Prompt}} {{.HelpName}} myminio --maintenance
`,
}

type readyMessage struct {
	Status          string `json:"status"`
	Alias           string `json:"alias"`
	Healthy         bool   `json:"healthy"`
	MaintenanceMode bool   `json:"maintenanceMode"`
	WriteQuorum     int    `json:"writeQuorum"`
	HealingDrives   int    `json:"healingDrives"`

	Err error `json:"error"`
}

func (r readyMessage) String() string {
	switch {
	case r.Healthy:
		return color.GreenString(fmt.Sprintf("The cluster '%s' is ready", r.Alias))
	case r.Err != nil:
		return color.RedString(fmt.Sprintf("The cluster '%s' is unreachable: %s", r.Alias, r.Err.Error()))
	default:
		return color.RedString(fmt.Sprintf("The cluster '%s' is not ready", r.Alias))
	}
}

// JSON jsonified ready result
func (r readyMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

// mainReady - main handler for mc ready command.
func mainReady(cliCtx *cli.Context) error {
	if !cliCtx.Args().Present() {
		exitCode := 1
		showCommandHelpAndExit(cliCtx, exitCode)
	}

	// Set command flags from context.
	clusterRead := cliCtx.Bool("cluster-read")
	maintenance := cliCtx.Bool("maintenance")

	ctx, cancelClusterReady := context.WithCancel(globalContext)
	defer cancelClusterReady()
	aliasedURL := cliCtx.Args().Get(0)

	anonClient, err := newAnonymousClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Couldn't construct anonymous client for `"+aliasedURL+"`.")

	healthOpts := madmin.HealthOpts{
		ClusterRead: clusterRead,
		Maintenance: maintenance,
	}

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			healthResult, hErr := anonClient.Healthy(ctx, healthOpts)
			printMsg(readyMessage{
				Alias:           aliasedURL,
				Status:          "success",
				Healthy:         healthResult.Healthy,
				MaintenanceMode: healthResult.MaintenanceMode,
				WriteQuorum:     healthResult.WriteQuorum,
				HealingDrives:   healthResult.HealingDrives,
				Err:             hErr,
			})
			if healthResult.Healthy {
				return nil
			}
			timer.Reset(healthCheckInterval)
		}
	}
}
