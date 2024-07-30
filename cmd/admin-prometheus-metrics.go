// Copyright (c) 2015-2024 MinIO, Inc.
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
	"strings"
	"time"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
)

var metricsFlags = append(metricsV3Flags,
	cli.StringFlag{
		Name:  "api-version",
		Usage: "version of metrics api to use. valid values are ['v2', 'v3']. defaults to 'v2' if not specified.",
		Value: "v2",
	})

var metricsV2SubSystems = set.CreateStringSet("node", "bucket", "cluster", "resource")

var adminPrometheusMetricsCmd = cli.Command{
	Name:         "metrics",
	Usage:        "print prometheus metrics",
	OnUsageError: onUsageError,
	Action:       mainSupportMetrics,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, metricsFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
USAGE:
  {{.HelpName}} TARGET [METRIC-TYPE]

METRIC-TYPE:
  valid values are
    api-version v2 ['cluster', 'node', 'bucket', 'resource']. defaults to 'cluster' if not specified.
    api-version v3 ["api", "system", "debug", "cluster", "ilm", "audit", "logger", "replication", "notification", "scanner"]. defaults to all if not specified.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES (v3):
  1. API metrics
     {{.Prompt}} {{.HelpName}} play api --api-version v3

  2. API metrics for the bucket 'mybucket'
     {{.Prompt}} {{.HelpName}} play api --bucket mybucket --api-version v3

  3. System metrics
     {{.Prompt}} {{.HelpName}} play system --api-version v3

  4. Debug metrics
     {{.Prompt}} {{.HelpName}} play debug --api-version v3

  5. Cluster metrics
     {{.Prompt}} {{.HelpName}} play cluster --api-version v3

  6. ILM metrics
     {{.Prompt}} {{.HelpName}} play ilm --api-version v3

  7. Audit metrics
     {{.Prompt}} {{.HelpName}} play audit --api-version v3

  8. Logger metrics
     {{.Prompt}} {{.HelpName}} play logger --api-version v3

  9. Replication metrics
     {{.Prompt}} {{.HelpName}} play replication --api-version v3

  10. Replication metrics for the bucket 'mybucket'
      {{.Prompt}} {{.HelpName}} play replication --bucket mybucket --api-version v3

  11. Notification metrics
      {{.Prompt}} {{.HelpName}} play notification --api-version v3

  12. Scanner metrics
      {{.Prompt}} {{.HelpName}} play scanner --api-version v3

EXAMPLES (v2):
  1. Metrics reported cluster wide.
     {{.Prompt}} {{.HelpName}} play

  2. Metrics reported at node level.
     {{.Prompt}} {{.HelpName}} play node

  3. Metrics reported at bucket level.
     {{.Prompt}} {{.HelpName}} play bucket

  4. Resource metrics.
     {{.Prompt}} {{.HelpName}} play resource
`,
}

const metricsEndPointRoot = "/minio/v2/metrics/"

type prometheusMetricsReq struct {
	aliasURL  string
	token     string
	subsystem string
}

// checkSupportMetricsSyntax - validate arguments passed by a user
func checkSupportMetricsSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func fetchMetrics(metricsURL string, token string) (*http.Response, error) {
	req, e := http.NewRequest(http.MethodGet, metricsURL, nil)
	if e != nil {
		return nil, e
	}
	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	client := httpClient(60 * time.Second)
	return client.Do(req)
}

func validateV2Args(ctx *cli.Context, subsys string) {
	for _, flag := range metricsV3Flags {
		flagName := flag.GetName()
		if ctx.IsSet(flagName) {
			fatalIf(errInvalidArgument().Trace(), "Flag `"+flagName+"` is not supported with v2 metrics")
		}
	}

	if !metricsV2SubSystems.Contains(subsys) {
		fatalIf(errInvalidArgument().Trace(),
			"invalid metric type `"+subsys+"`. valid values are `"+
				strings.Join(metricsV2SubSystems.ToSlice(), ", ")+"`")
	}
}

func printPrometheusMetricsV2(ctx *cli.Context, req prometheusMetricsReq) error {
	subsys := req.subsystem
	if subsys == "" {
		subsys = "cluster"
	}
	validateV2Args(ctx, subsys)

	resp, e := fetchMetrics(req.aliasURL+metricsEndPointRoot+subsys, req.token)
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
	apiVer := ctx.String("api-version")

	metricsReq := prometheusMetricsReq{
		aliasURL:  hostConfig.URL,
		token:     token,
		subsystem: metricsSubSystem,
	}

	switch apiVer {
	case "v2":
		err := printPrometheusMetricsV2(ctx, metricsReq)
		fatalIf(probe.NewError(err), "Unable to list prometheus metrics with api-version v2.")
	case "v3":
		err := printPrometheusMetricsV3(ctx, metricsReq)
		fatalIf(probe.NewError(err), "Unable to list prometheus metrics with api-version v3.")
	default:
		fatalIf(errInvalidArgument().Trace(), "Invalid api version `"+apiVer+"`")
	}

	return nil
}
