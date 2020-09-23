/*
 * MinIO Client (C) 2019 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"
	"net/url"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"

	jwtgo "github.com/dgrijalva/jwt-go"
	json "github.com/minio/mc/pkg/colorjson"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultJobName     = "minio-job"
	defaultMetricsPath = "/minio/prometheus/metrics"
)

var adminPrometheusGenerateCmd = cli.Command{
	Name:            "generate",
	Usage:           "generates prometheus config",
	Action:          mainAdminPrometheusGenerate,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
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
	BearerToken   string       `yaml:"bearer_token" json:"bearerToken"`
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

	u, err := url.Parse(hostConfig.URL)
	if err != nil {
		return err
	}

	jwt := jwtgo.NewWithClaims(jwtgo.SigningMethodHS512, jwtgo.StandardClaims{
		ExpiresAt: UTCNow().Add(defaultPrometheusJWTExpiry).Unix(),
		Subject:   hostConfig.AccessKey,
		Issuer:    "prometheus",
	})

	token, err := jwt.SignedString([]byte(hostConfig.SecretKey))
	if err != nil {
		return err
	}

	// Setting the values
	defaultConfig.ScrapeConfigs[0].BearerToken = token
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
