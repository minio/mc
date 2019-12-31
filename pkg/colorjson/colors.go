/*
 * Copyright 2011 The Go Authors. All rights reserved
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

package colorjson

import (
	"github.com/fatih/color"
	"github.com/minio/minio/pkg/console"
)

const (
	// FgDarkGray is the shell color code for dark gray. Needs to be followed by
	// FgBlack to render dark gray
	FgDarkGray = 1
	jsonString = "jsonGreen"
	jsonBool   = "jsonRed"
	jsonNum    = "jsonRed"
	jsonKey    = "jsonBoldBlue"
	jsonNull   = "jsonBoldDarkGray"
)

func init() {
	console.SetColor(jsonString, color.New(color.FgGreen))
	console.SetColor(jsonBool, color.New(color.FgRed))
	console.SetColor(jsonNum, color.New(color.FgRed))
	console.SetColor(jsonKey, color.New(color.FgBlue, color.Bold))
	console.SetColor(jsonNull, color.New(FgDarkGray, color.FgBlack, color.Bold))
}
