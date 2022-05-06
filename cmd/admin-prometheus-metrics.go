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
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	dto "github.com/prometheus/client_model/go"
	prom2json "github.com/prometheus/prom2json"
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
  {{.HelpName}} TARGET
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List of metrics reported cluster wide.
     {{.Prompt}} {{.HelpName}} play
`,
}

const (
	metricsRespBodyLimit = 10 << 20 // 10 MiB
	metricsEndPoint      = "/minio/v2/metrics/cluster"
)

// checkSupportMetricsSyntax - validate arguments passed by a user
func checkSupportMetricsSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "metrics", 1) // last argument is exit code
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

	req, e := http.NewRequest(http.MethodGet, hostConfig.URL+metricsEndPoint, nil)
	if e != nil {
		return e
	}
	req.Header.Add("Authorization", "Bearer "+token)
	client := httpClient(10 * time.Second)
	resp, e := client.Do(req)
	if e != nil {
		return e
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		printMsg(prometheusMetricsReader{Reader: io.LimitReader(resp.Body, metricsRespBodyLimit)})
	}
	return nil
}

// JSON returns jsonified message
func (pm prometheusMetricsReader) JSON() string {
	mfChan := make(chan *dto.MetricFamily)
	go func() {
		if err := prom2json.ParseReader(pm.Reader, mfChan); err != nil {
			fatalIf(probe.NewError(err), "error reading metrics:")
		}
	}()
	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}
	jsonMessageBytes, e := json.MarshalIndent(result, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

// String - returns the string representation of the prometheus metrics
func (pm prometheusMetricsReader) String() string {
	respBytes, e := ioutil.ReadAll(pm.Reader)
	if e != nil {
		fatalIf(probe.NewError(e), "error reading metrics:")
	}
	return string(respBytes)
}

// prometheusMetricsReader mirrors the MetricFamily proto message.
type prometheusMetricsReader struct {
	Reader io.Reader
}

func mainSupportMetrics(ctx *cli.Context) error {
	checkSupportMetricsSyntax(ctx)
	if err := printPrometheusMetrics(ctx); err != nil {
		fatalIf(probe.NewError(err), "Error in listing prometheus metrics")
	}
	return nil
}
