/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/minio/mc/internal/github.com/minio/cli"
	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/pkg/console"
)

var versionCmd = cli.Command{
	Name:   "version",
	Usage:  "Print version",
	Action: mainVersion,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
`,
}

func mainVersion(ctxx *cli.Context) {
	t, _ := time.Parse(time.RFC3339Nano, Version)
	if t.IsZero() {
		console.Println("")
		return
	}
	type Version struct {
		Value  time.Time `json:"value"`
		Format string    `json:"format"`
	}
	if globalJSONFlag {
		tB, err := json.MarshalIndent(struct {
			Version Version `json:"version"`
		}{Version: Version{t, "RFC3339Nano"}}, "", "\t")
		if err != nil {
			console.Fatalln(probe.NewError(err))
		}
		console.Println(console.JSON(string(tB)))
		return
	}
	console.Println(t.Format(http.TimeFormat))
}
