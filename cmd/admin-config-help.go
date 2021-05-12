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
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/fatih/color"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
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
