// Copyright (c) 2015-2021 MinIO, Inc.
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
	"fmt"
	"net/url"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"

	jwtgo "github.com/dgrijalva/jwt-go"
	json "github.com/minio/colorjson"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultJobName     = "minio-job"
	defaultMetricsPath = "/minio/v2/metrics/cluster"
)

var prometheusFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "public",
		Usage: "disable bearer token generation for scrape_configs",
	},
}

var adminPrometheusGenerateCmd = cli.Command{
	Name:            "generate",
	Usage:           "generates prometheus config",
	Action:          mainAdminPrometheusGenerate,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(prometheusFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Generate a default prometheus config.
     {{.Prompt}} {{.HelpName}} myminio

`,
}

// PrometheusConfig - container to hold the top level scrape config.
type PrometheusConfig struct {
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs,omitempty"`
}

// String colorized prometheus config yaml.
func (c PrometheusConfig) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("error creating config string: %s", err)
	}
	return console.Colorize("yaml", string(b))
}

// JSON jsonified prometheus config.
func (c PrometheusConfig) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(c.ScrapeConfigs[0], "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

// StatConfig - container to hold the targets config.
type StatConfig struct {
	Targets []string `yaml:",flow" json:"targets"`
}

// String colorized stat config yaml.
func (t StatConfig) String() string {
	b, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("error creating config string: %s", err)
	}
	return console.Colorize("yaml", string(b))
}

// JSON jsonified stat config.
func (t StatConfig) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(t.Targets, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

// ScrapeConfig configures a scraping unit for Prometheus.
type ScrapeConfig struct {
	JobName       string       `yaml:"job_name" json:"jobName"`
	BearerToken   string       `yaml:"bearer_token,omitempty" json:"bearerToken,omitempty"`
	MetricsPath   string       `yaml:"metrics_path,omitempty" json:"metricsPath"`
	Scheme        string       `yaml:"scheme,omitempty" json:"scheme"`
	StaticConfigs []StatConfig `yaml:"static_configs,omitempty" json:"staticConfigs"`
}

const (
	defaultPrometheusJWTExpiry = 100 * 365 * 24 * time.Hour
)

var defaultConfig = PrometheusConfig{
	ScrapeConfigs: []ScrapeConfig{
		{
			JobName:     defaultJobName,
			MetricsPath: defaultMetricsPath,
			StaticConfigs: []StatConfig{
				{
					Targets: []string{""},
				},
			},
		},
	},
}

// checkAdminPrometheusSyntax - validate all the passed arguments
func checkAdminPrometheusSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "generate", 1) // last argument is exit code
	}
}

func generatePrometheusConfig(ctx *cli.Context) error {
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

	u, e := url.Parse(hostConfig.URL)
	if e != nil {
		return e
	}

	if !ctx.Bool("public") {
		jwt := jwtgo.NewWithClaims(jwtgo.SigningMethodHS512, jwtgo.StandardClaims{
			ExpiresAt: UTCNow().Add(defaultPrometheusJWTExpiry).Unix(),
			Subject:   hostConfig.AccessKey,
			Issuer:    "prometheus",
		})

		token, e := jwt.SignedString([]byte(hostConfig.SecretKey))
		if e != nil {
			return e
		}

		// Setting the values
		defaultConfig.ScrapeConfigs[0].BearerToken = token
	}
	defaultConfig.ScrapeConfigs[0].Scheme = u.Scheme
	defaultConfig.ScrapeConfigs[0].StaticConfigs[0].Targets[0] = u.Host

	printMsg(defaultConfig)

	return nil
}

// mainAdminPrometheus is the handle for "mc admin prometheus generate" sub-command.
func mainAdminPrometheusGenerate(ctx *cli.Context) error {

	console.SetColor("yaml", color.New(color.FgGreen))

	checkAdminPrometheusSyntax(ctx)

	if err := generatePrometheusConfig(ctx); err != nil {
		return nil
	}

	return nil
}
