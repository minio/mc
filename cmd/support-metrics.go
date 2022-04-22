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

	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	dto "github.com/prometheus/client_model/go"
	prom2json "github.com/prometheus/prom2json"
)

var supportMetricsCmd = cli.Command{
	Name:         "metrics",
	Usage:        "list of prometheus metrics reported cluster wide",
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

func listPrometheusMetrics(ctx *cli.Context) error {
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

	jwt := jwtgo.NewWithClaims(jwtgo.SigningMethodHS512, jwtgo.RegisteredClaims{
		ExpiresAt: jwtgo.NewNumericDate(UTCNow().Add(defaultPrometheusJWTExpiry)),
		Subject:   hostConfig.AccessKey,
		Issuer:    "prometheus",
	})
	token, e := jwt.SignedString([]byte(hostConfig.SecretKey))
	if e != nil {
		return e
	}

	req, e := http.NewRequest(http.MethodGet, hostConfig.URL+metricsEndPoint, nil)
	if e != nil {
		return e
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, e := httpDo(req)
	if e != nil {
		return e
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if globalJSON {
			mfChan := make(chan *dto.MetricFamily)
			go func() {
				if err := prom2json.ParseReader(io.LimitReader(resp.Body, metricsRespBodyLimit), mfChan); err != nil {
					fatalIf(probe.NewError(err), "error reading metrics:")
				}
			}()
			result := []*prom2json.Family{}
			for mf := range mfChan {
				result = append(result, prom2json.NewFamily(mf))
			}
			printMsg(PrometheusMetrics{Metrics: result})
		} else {
			respBytes, e := ioutil.ReadAll(io.LimitReader(resp.Body, metricsRespBodyLimit))
			if e != nil {
				return e
			}
			console.Println(string(respBytes))
		}
	}
	return nil
}

// JSON returns jsonified message
func (pm PrometheusMetrics) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(pm, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

// String implemented to make interface compatible
func (pm PrometheusMetrics) String() string {
	return ""
}

// PrometheusMetrics mirrors the MetricFamily proto message.
type PrometheusMetrics struct {
	Metrics []*prom2json.Family `json:"family,omitempty"`
}

func mainSupportMetrics(ctx *cli.Context) error {
	checkSupportMetricsSyntax(ctx)
	if err := listPrometheusMetrics(ctx); err != nil {
		fatalIf(probe.NewError(err), "Error in listing prometheus metrics")
	}
	return nil
}
