/*
 * Minio Client (C) 2015 Minio, Inc.
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

package console

import "github.com/fatih/color"

// MiniTheme - Minio's default color theme
var MiniTheme = Theme{
	Debug: (color.New(color.FgWhite, color.Faint, color.Italic)),
	Fatal: (color.New(color.FgRed, color.Italic, color.Bold)),
	Error: (color.New(color.FgYellow, color.Italic)),
	Info:  (color.New(color.FgGreen, color.Bold)),
	File:  (color.New(color.FgWhite)),
	Dir:   (color.New(color.FgCyan, color.Bold)),
	Size:  (color.New(color.FgYellow)),
	Time:  (color.New(color.FgGreen)),
	Retry: (color.New(color.FgMagenta, color.Bold)),
	JSON:  (color.New(color.FgWhite, color.Italic)),
	Print: (color.New()),
}

// WhiteTheme - All white color theme
var WhiteTheme = Theme{
	Debug: (color.New(color.FgWhite, color.Faint, color.Italic)),
	Fatal: (color.New(color.FgWhite, color.Bold, color.Italic)),
	Error: (color.New(color.FgWhite, color.Bold, color.Italic)),
	Info:  (color.New(color.FgWhite, color.Bold)),
	File:  (color.New(color.FgWhite, color.Bold)),
	Dir:   (color.New(color.FgWhite, color.Bold)),
	Size:  (color.New(color.FgWhite, color.Bold)),
	Time:  (color.New(color.FgWhite, color.Bold)),
	Retry: (color.New(color.FgWhite, color.Bold)),
	JSON:  (color.New(color.FgWhite, color.Bold, color.Italic)),
	Print: (color.New()),
}

// NoColorTheme - Disables color theme
var NoColorTheme = Theme{
	Debug: (color.New()),
	Fatal: (color.New()),
	Error: (color.New()),
	Info:  (color.New()),
	File:  (color.New()),
	Dir:   (color.New()),
	Size:  (color.New()),
	Time:  (color.New()),
	Retry: (color.New()),
	JSON:  (color.New()),
	Print: (color.New()),
}
