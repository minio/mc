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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminConfigGetCmd = cli.Command{
	Name:         "get",
	Usage:        "interactively retrieve a config key parameters",
	Before:       setGlobalsFromContext,
	Action:       mainAdminConfigGet,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  The output includes environment variables set on the server. These cannot be overridden from the client.

  1. Get the current region setting on MinIO server.
     {{.Prompt}} {{.HelpName}} play/ region
     region name=us-east-1

  2. Get the current notification settings for Webhook target on MinIO server
     {{.Prompt}} {{.HelpName}} myminio/ notify_webhook
     notify_webhook endpoint="http://localhost:8080" auth_token= queue_limit=10000 queue_dir="/home/events"

  3. Get the current compression settings on MinIO server
     {{.Prompt}} {{.HelpName}} myminio/ compression
     compression extensions=".txt,.csv" mime_types="text/*"
`,
}

var errInvalidEnvVarLine = errors.New("expected env var line of the form `# MINIO_...=...`")

type envOverride struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// configValue represents the value for a configuration parameter including the
// env value if present.
type configValue struct {
	Value       string       `json:"value"`
	EnvOverride *envOverride `json:"env_override,omitempty"`
}

type subsysConfig struct {
	SubSystem string                 `json:"subSystem"`
	Target    string                 `json:"target,omitempty"`
	KV        map[string]configValue `json:"kv"`
}

type configGetMessage struct {
	Status string         `json:"status"`
	Config []subsysConfig `json:"config"`
	value  []byte
}

func parseEnvVarLine(s, subSystem, target string) (envVar, configVar, value string, err error) {
	s = strings.TrimPrefix(s, madmin.KvComment+madmin.KvSpaceSeparator)
	ps := strings.SplitN(s, madmin.KvSeparator, 2)
	if len(ps) != 2 {
		err = errInvalidEnvVarLine
		return
	}

	envVar = ps[0]
	value = ps[1]

	configVar = strings.TrimPrefix(envVar, madmin.EnvPrefix+strings.ToUpper(subSystem)+madmin.EnvWordDelimiter)
	if target != madmin.Default {
		configVar = strings.TrimSuffix(configVar, madmin.EnvWordDelimiter+target)
	}
	configVar = strings.ToLower(configVar)
	return
}

// Assume validity of input to simplify the parsing.
func parseConfigLine(s string) (subSys, target string, kvs []madmin.KV) {
	ps := strings.SplitN(s, madmin.KvSpaceSeparator, 2)

	ws := strings.SplitN(ps[0], madmin.SubSystemSeparator, 2)
	subSys = ws[0]
	target = madmin.Default
	if len(ws) == 2 {
		target = ws[1]
	}

	if len(ps) == 1 {
		return
	}

	// Parse keys and values
	text := ps[1]
	for len(text) > 0 {
		ts := strings.SplitN(text, madmin.KvSeparator, 2)
		kv := madmin.KV{Key: ts[0]}
		text = ts[1]

		// Value may be double quoted.
		if strings.HasPrefix(text, madmin.KvDoubleQuote) {
			text = strings.TrimPrefix(text, madmin.KvDoubleQuote)
			ts := strings.SplitN(text, madmin.KvDoubleQuote, 2)
			kv.Value = ts[0]
			text = strings.TrimSpace(ts[1])
		} else {
			ts := strings.SplitN(text, madmin.KvSpaceSeparator, 2)
			kv.Value = ts[0]
			if len(ts) == 2 {
				text = ts[1]
			} else {
				text = ""
			}
		}
		kvs = append(kvs, kv)
	}
	return
}

func isEnvLine(s string) bool {
	return strings.HasPrefix(s, madmin.EnvLinePrefix)
}

func isCommentLine(s string) bool {
	return strings.HasPrefix(s, madmin.KvComment)
}

func getConfigLineSubSystemAndTarget(s string) (subSys, target string) {
	words := strings.SplitN(s, madmin.KvSpaceSeparator, 2)
	pieces := strings.SplitN(words[0], madmin.SubSystemSeparator, 2)
	if len(pieces) == 2 {
		return pieces[0], pieces[1]
	}
	// If no target is present, it is the default target.
	return pieces[0], madmin.Default
}

func parseServerConfigOutputToJSONObject(s string) ([]subsysConfig, error) {
	lines := strings.Split(s, "\n")

	// Clean up config lines
	var configLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			configLines = append(configLines, line)
		}
	}

	// Parse out config lines into groups corresponding to a single subsystem.
	//
	// How does it work? The server output is a list of lines, where each line
	// may be one of:
	//
	//   1. A config line for a single subsystem (and optional target). For
	//   example, "site region=us-east-1" or "identity_openid:okta k1=v1 k2=v2".
	//
	//   2. A comment line showing an environment variable set on the server.
	//   For example "# MINIO_SITE_NAME=my-cluster".
	//
	//   3. Comment lines with other content. These will not start with `#
	//   MINIO_`.
	//
	// For the structured JSON representation, only lines of type 1 and 2 are
	// required as they correspond to configuration specified by an
	// administrator.
	//
	// Additionally, after ignoring lines of type 3 above:
	//
	//   1. environment variable lines for a subsystem (and target if present)
	//   appear consecutively.
	//
	//   2. exactly one config line for a subsystem and target immediately
	//   follows the env var lines for the same subsystem and target.
	//
	// The parsing logic below classifies each line and groups them by
	// subsystem and target.
	var configGroups [][]string
	var subSystems []string
	var targets []string
	var currGroup []string
	for _, line := range configLines {
		if isEnvLine(line) {
			currGroup = append(currGroup, line)
		} else if isCommentLine(line) {
			continue
		} else {
			subSys, target := getConfigLineSubSystemAndTarget(line)
			currGroup = append(currGroup, line)
			configGroups = append(configGroups, currGroup)
			subSystems = append(subSystems, subSys)
			targets = append(targets, target)
		}
	}

	var res []subsysConfig
	for i, group := range configGroups {
		sc := subsysConfig{
			SubSystem: subSystems[i],
			KV:        make(map[string]configValue),
		}
		if targets[i] != madmin.Default {
			sc.Target = targets[i]
		}
		for _, line := range group {
			if isEnvLine(line) {
				envVar, configVar, value, err := parseEnvVarLine(line, subSystems[i], targets[i])
				if err != nil {
					return nil, err
				}
				sc.KV[configVar] = configValue{
					EnvOverride: &envOverride{
						Name:  envVar,
						Value: value,
					},
				}
				continue
			}

			_, _, kvs := parseConfigLine(line)
			for _, kv := range kvs {
				cv, ok := sc.KV[kv.Key]
				if ok {
					cv.Value = kv.Value
					sc.KV[kv.Key] = cv
				} else {
					sc.KV[kv.Key] = configValue{Value: kv.Value}
				}
			}

		}

		res = append(res, sc)
	}

	return res, nil
}

// String colorized service status message.
func (u configGetMessage) String() string {
	console.SetColor("EnvVar", color.New(color.FgYellow))
	bio := bufio.NewReader(bytes.NewReader(u.value))
	var lines []string
	for {
		s, err := bio.ReadString('\n')
		// Make lines displaying environment variables bold.
		if strings.HasPrefix(s, "# MINIO_") {
			s = strings.TrimPrefix(s, "# ")
			parts := strings.SplitN(s, "=", 2)
			s = fmt.Sprintf("# %s=%s", console.Colorize("EnvVar", parts[0]), parts[1])
			lines = append(lines, s)
		} else {
			lines = append(lines, s)
		}
		if err == io.EOF {
			break
		}
		fatalIf(probe.NewError(err), "Unable to marshal to string.")
	}
	return strings.Join(lines, "")
}

// JSON jsonified service status Message message.
func (u configGetMessage) JSON() string {
	u.Status = "success"
	var err error
	u.Config, err = parseServerConfigOutputToJSONObject(string(u.value))
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigGetSyntax - validate all the passed arguments
func checkAdminConfigGetSyntax(ctx *cli.Context) {
	if !ctx.Args().Present() || len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "get", 1) // last argument is exit code
	}
}

func mainAdminConfigGet(ctx *cli.Context) error {
	checkAdminConfigGetSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if len(ctx.Args()) == 1 {
		// Call get config API
		hr, e := client.HelpConfigKV(globalContext, "", "", false)
		fatalIf(probe.NewError(e), "Unable to get help for the sub-system")

		// Print
		printMsg(configHelpMessage{
			Value:   hr,
			envOnly: false,
		})

		return nil
	}

	subSys := strings.Join(args.Tail(), " ")

	// Call get config API
	buf, e := client.GetConfigKV(globalContext, subSys)
	fatalIf(probe.NewError(e), "Unable to get server '%s' config", args.Tail())

	if globalJSON {
		printMsg(configGetMessage{
			value: buf,
		})
	} else {
		// Print
		printMsg(configGetMessage{
			value: buf,
		})
	}

	return nil
}
