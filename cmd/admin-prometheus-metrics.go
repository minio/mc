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
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var adminPrometheusMetricsCmd = cli.Command{
	Name:         "metrics",
	Usage:        "print cluster wide prometheus metrics",
	OnUsageError: onUsageError,
	Action:       mainSupportMetrics,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
USAGE:
  {{.HelpName}} TARGET [METRIC-TYPE]

METRIC-TYPE:
  valid values are ['cluster', 'node', 'bucket', 'resource']. Defaults to 'cluster' if not specified.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List of metrics reported cluster wide.
     {{.Prompt}} {{.HelpName}} play

  2. List of metrics reported at node level.
     {{.Prompt}} {{.HelpName}} play node

  3. List of metrics reported at bucket level.
     {{.Prompt}} {{.HelpName}} play bucket

  4. List of resource metrics.
     {{.Prompt}} {{.HelpName}} play resource
`,
}

const metricsEndPointRoot = "/minio/v2/metrics/"

// checkSupportMetricsSyntax - validate arguments passed by a user
func checkSupportMetricsSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func printPrometheusMetrics(ctx *cli.Context) error {
	// Get the alias parameter from cli
	args := ctx.Args()
	alias := cleanAlias(args.Get(0))

	if !isValidAlias(alias) {
		fatalIf(errInvalidAlias(alias), "Invalid alias.")
	}
	hostConfig := mustGetHostConfig(alias)
	if hostConfig == nil {
		fatalIf(errInvalidAliasedURL(alias), "No such alias `"+alias+"` found.")
		return nil
	}

	token, e := getPrometheusToken(hostConfig)
	if e != nil {
		return e
	}
	metricsSubSystem := args.Get(1)
	switch metricsSubSystem {
	case "node", "bucket", "cluster", "resource":
	case "":
		metricsSubSystem = "cluster"
	default:
		fatalIf(errInvalidArgument().Trace(), "invalid metric type '%v'", metricsSubSystem)
	}

	req, e := http.NewRequest(http.MethodGet, hostConfig.URL+metricsEndPointRoot+metricsSubSystem, nil)
	if e != nil {
		return e
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	client := httpClient(60 * time.Second)
	resp, e := client.Do(req)
	if e != nil {
		return e
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		printMsg(prometheusMetricsReader{Reader: resp.Body})
		return nil
	}

	return errors.New(resp.Status)
}

// JSON returns jsonified message
func (pm prometheusMetricsReader) JSON() string {
	results, e := madmin.ParsePrometheusResults(pm.Reader)
	fatalIf(probe.NewError(e), "Unable to parse Prometheus metrics.")

	jsonMessageBytes, e := json.MarshalIndent(results, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

// String - returns the string representation of the prometheus metrics
func (pm prometheusMetricsReader) String() string {
	_, e := io.Copy(os.Stdout, pm.Reader)

	fatalIf(probe.NewError(e), "Unable to read Prometheus metrics.")

	return ""
}

// prometheusMetricsReader mirrors the MetricFamily proto message.
type prometheusMetricsReader struct {
	Reader io.Reader
}

func mainSupportMetrics(ctx *cli.Context) error {
	checkSupportMetricsSyntax(ctx)

	fatalIf(probe.NewError(printPrometheusMetrics(ctx)), "Unable to list prometheus metrics.")

	return nil
}
