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

// This package contains all the global variables and constants. ONLY TO BE ACCESSED VIA GET/SET FUNCTIONS.
package main

var (
	globalQuietFlag = false // Quiet flag set via command line
	globalMimicFlag = false // Unix flag set via command line
	globalJSONFlag  = false // Json flag set via command line
	globalDebugFlag = false // Debug flag set via command line
)

// mc configuration related constants.
const (
	globalMCPreviousConfigVersion = "1.0.0"
	globalMCCurrentConfigVersion  = "1.0.1"
	globalMCVersion               = Version

	globalMCConfigDir        = ".mc/"
	globalMCConfigWindowsDir = "mc\\"
	globalMCConfigFile       = "config.json"

	// session config related constants
	globalSessionDir = "session"

	// default access and secret key
	// do not pass accesskeyid and secretaccesskey through cli
	// users should manually edit them, add a stub entry
	globalAccessKeyID     = "YOUR-ACCESS-KEY-ID-HERE"
	globalSecretAccessKey = "YOUR-SECRET-ACCESS-KEY-HERE"

	// default host
	globalExampleHostURL = "YOUR-EXAMPLE.COM"
)
