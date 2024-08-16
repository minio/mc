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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/minio-go/v7/pkg/set"
)

var (
	metricsV3Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Usage: "bucket name to list metrics for. only applicable with api version v3 for metric type 'api, replication'",
		},
	}

	metricsV3SubSystems = set.CreateStringSet("api", "system", "debug", "cluster",
		"ilm", "audit", "logger", "replication", "notification", "scanner")

	bucketMetricsSubSystems = set.CreateStringSet("api", "replication")
)

const metricsV3EndPointRoot = "/minio/metrics/v3"

func getMetricsV3Path(subsys string, bucket string) string {
	params := url.Values{}

	metricsPath := metricsV3EndPointRoot
	if len(bucket) > 0 {
		metricsPath += "/bucket"
	}

	if len(subsys) > 0 {
		metricsPath += "/" + subsys
	}

	if len(bucket) > 0 {
		// bucket specific metrics endpoints have '/bucket' prefix and bucket name as suffix.
		// e.g. bucket api metrics: /bucket/api/mybucket
		metricsPath += "/" + bucket
	}

	qparams := params.Encode()
	if len(qparams) > 0 {
		metricsPath += "?" + qparams
	}
	return metricsPath
}

func validateV3Args(subsys string, bucket string) {
	if subsys != "" && !metricsV3SubSystems.Contains(subsys) {
		fatalIf(errInvalidArgument().Trace(),
			"invalid metric type `"+subsys+"`. valid values are `"+
				strings.Join(metricsV3SubSystems.ToSlice(), ", ")+"`")
	}

	if len(bucket) > 0 {
		bms := strings.Join(bucketMetricsSubSystems.ToSlice(), ", ")
		if len(subsys) == 0 {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("metric type must be passed with --bucket. valid values are `%s`", bms))
		}

		if !bucketMetricsSubSystems.Contains(subsys) {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("--bucket is applicable only for metric types `%s`", bms))
		}
	}
}

func printPrometheusMetricsV3(ctx *cli.Context, req prometheusMetricsReq) error {
	bucket := ctx.String("bucket")
	validateV3Args(req.subsystem, bucket)

	metricsURL := req.aliasURL + getMetricsV3Path(req.subsystem, bucket)

	resp, e := fetchMetrics(metricsURL, req.token)
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
