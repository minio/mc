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
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/fatih/color"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

// HelpTmpl template used by all sub-systems
const HelpTmpl = `{{if ne .SubSys ""}}{{colorBlueBold "KEY:"}}
{{if .MultipleTargets}}{{colorYellowBold .SubSys}}[:name]{{"\t"}}{{else}}{{colorYellowBold .SubSys}}{{"\t"}}{{end}}{{.Description}}

{{colorBlueBold "ARGS:"}}{{range .KeysHelp}}
{{if .Optional}}{{colorYellowBold .Key}}{{else}}{{colorRedBold .Key}}*{{end}}{{"\t"}}({{.Type}}){{"\t"}}{{.Description}}{{end}}{{else}}{{colorBlueBold "KEYS:"}}{{range .KeysHelp}}
{{colorGreenBold .Key}}{{"\t"}}{{.Description}}{{end}}{{end}}`

var funcMap = template.FuncMap{
	"colorBlueBold":   color.New(color.FgBlue, color.Bold).SprintfFunc(),
	"colorYellowBold": color.New(color.FgYellow, color.Bold).SprintfFunc(),
	"colorCyanBold":   color.New(color.FgCyan, color.Bold).SprintFunc(),
	"colorRedBold":    color.New(color.FgRed, color.Bold).SprintfFunc(),
	"colorGreenBold":  color.New(color.FgGreen, color.Bold).SprintfFunc(),
}

// HelpTemplate - captures config help template
var HelpTemplate = template.Must(template.New("config-help").Funcs(funcMap).Parse(HelpTmpl))

// HelpEnvTemplate - captures config help template
var HelpEnvTemplate = template.Must(template.New("config-help-env").Funcs(funcMap).Parse(HelpTmpl))

// configHelpMessage container to hold locks information.
type configHelpMessage struct {
	Status  string      `json:"status"`
	Value   madmin.Help `json:"help"`
	envOnly bool
}

// String colorized service status message.
func (u configHelpMessage) String() string {
	var s strings.Builder
	w := tabwriter.NewWriter(&s, 1, 8, 2, ' ', 0)
	var e error
	if !u.envOnly {
		e = HelpTemplate.Execute(w, u.Value)
	} else {
		e = HelpEnvTemplate.Execute(w, u.Value)
	}
	fatalIf(probe.NewError(e), "Unable to initialize template writer")

	w.Flush()

	return s.String()
}

// JSON jsonified service status Message message.
func (u configHelpMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}
