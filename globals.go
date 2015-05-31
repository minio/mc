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

// This package contains all the global variables and constants
package main

var (
	globalQuietFlag = false // Quiet flag set via command line
	globalAliasFlag = false // Alias flag set via command line
	globalJSONFlag  = false // Json flag set via command line
	globalDebugFlag = false // Debug flag set via command line

	mcCurrentConfigVersion = "1.0.0"
)

// mc configuration related constants
const (
	mcConfigDir        = ".mc/"
	mcConfigWindowsDir = "mc/"
	mcConfigFile       = "config.json"
)

// default access and secret key
const (
	// do not pass accesskeyid and secretaccesskey through cli
	// users should manually edit them, add a stub entry
	globalAccessKeyID     = "YOUR-ACCESS-KEY-ID-HERE"
	globalSecretAccessKey = "YOUR-SECRET-ACCESS-KEY-HERE"
)

// default host
const (
	exampleHostURL = "YOUR-EXAMPLE.COM"
)
