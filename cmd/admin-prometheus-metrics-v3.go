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
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/v3/console"
)

var (
	metricsV3Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Usage: "bucket name to list metrics for. only applicable with api version v3 for metric type 'bucket'",
		},
		cli.BoolFlag{
			Name:  "list",
			Usage: "list the available metrics. only applicable with api version v3",
		},
	}

	metricsV3SubSystems = set.CreateStringSet("api", "system", "debug", "cluster",
		"ilm", "audit", "logger", "replication", "notification", "scanner")

	bucketMetricsSubSystems = set.CreateStringSet("api", "replication")
)

const metricsV3EndPointRoot = "/minio/metrics/v3"

func printPrometheusMetricsV3(ctx *cli.Context, req prometheusMetricsReq) error {
	subsys := req.subsystem
	if subsys != "" && !metricsV3SubSystems.Contains(subsys) {
		fatalIf(errInvalidArgument().Trace(),
			"invalid metric type `"+subsys+"`. valid values are `"+
				strings.Join(metricsV3SubSystems.ToSlice(), ", ")+"`")
	}

	list := ctx.Bool("list")
	bucket := ctx.String("bucket")
	metricsURL := req.aliasURL + metricsV3EndPointRoot
	params := url.Values{}

	if len(bucket) > 0 {
		bms := strings.Join(bucketMetricsSubSystems.ToSlice(), ", ")
		if len(subsys) == 0 {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("metric type must be passed with --bucket. valid values are `"+bms+"`"))
		}

		if !bucketMetricsSubSystems.Contains(subsys) {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("--bucket is applicable only for metric types `"+bms+"`"))
		}

		metricsURL += "/bucket"
	}

	if len(subsys) > 0 {
		metricsURL += "/" + subsys
	}

	if len(bucket) > 0 {
		// bucket specific metrics endpoints have '/bucket' prefix and bucket name as suffix.
		// e.g. bucket api metrics: /bucket/api/mybucket
		metricsURL += "/" + bucket
	}

	if list {
		params.Add("list", "true")
	}

	qparams := params.Encode()
	if len(qparams) > 0 {
		metricsURL += "?" + qparams
	}

	resp, e := fetchMetrics(metricsURL, req.token)
	if e != nil {
		return e
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		printMsg(prometheusMetricsV3Reader{
			prometheusMetricsReader: prometheusMetricsReader{Reader: resp.Body},
			isList:                  list,
		})
		return nil
	}

	return errors.New(resp.Status)
}

// JSON returns jsonified message
func (pm prometheusMetricsV3Reader) JSON() string {
	if !pm.isList {
		return pm.prometheusMetricsReader.JSON()
	}

	metricInfos := []prometheusMetricInfo{}
	rows := pm.readMetricsV3List()
	for idx, row := range rows {
		if idx == 0 || len(row) != 4 {
			// first row is the header, skip it.
			continue
		}
		metricInfos = append(metricInfos, prometheusMetricInfo{
			Name:   strings.TrimSpace(row[0]),
			Type:   strings.TrimSpace(row[1]),
			Help:   strings.TrimSpace(row[2]),
			Labels: strings.TrimSpace(row[3]),
		})
	}

	jsonMessageBytes, e := json.MarshalIndent(metricInfos, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (pm prometheusMetricsV3Reader) readMetricsV3List() [][]string {
	scanner := bufio.NewScanner(pm.Reader)
	lineNum := 0
	rows := [][]string{}
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if lineNum == 2 {
			// second line is just a separator, skip it.
			continue
		}
		// remove leading and trailing '|', remove '`' and finally split by '|'
		// example row: | `minio_bucket_api_total` | `counter` | Total number of requests for a bucket | `bucket,name,type,server` |
		row := strings.Split(strings.ReplaceAll(strings.Trim(line, "|"), "`", ""), "|")
		rows = append(rows, row)
	}
	return rows
}

func (pm prometheusMetricsV3Reader) printMetricsV3List() error {
	rows := pm.readMetricsV3List()
	dspOrder := []col{colGreen}
	for i := 0; i < len(rows)-1; i++ {
		dspOrder = append(dspOrder, colGrey)
	}

	var printColors []*color.Color
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}

	// four columns - name, type, help, labels
	tbl := console.NewTable(printColors, []bool{false, false, false, false}, 0)
	return tbl.DisplayTable(rows)
}

// String - returns the string representation of the prometheus metrics
func (pm prometheusMetricsV3Reader) String() string {
	if !pm.isList {
		return pm.prometheusMetricsReader.String()
	}

	e := pm.printMetricsV3List()
	fatalIf(probe.NewError(e), "Unable to render table view")

	return ""
}

// prometheusMetricsV3Reader is used for printing
// the prometheus metrics returned by the v3 api.
type prometheusMetricsV3Reader struct {
	prometheusMetricsReader
	isList bool
}

// prometheusMetricInfo contains information about a prometheus metric.
type prometheusMetricInfo struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Help   string `json:"help"`
	Labels string `json:"labels"`
}
