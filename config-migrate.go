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

package main

import (
	"fmt"
	"strings"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/quick"
)

// migrate config files from the any older version to the latest.
func migrateConfig() {
	// Migrate config V1 to V101
	migrateConfigV1ToV101()
	// Migrate config V101 to V2
	migrateConfigV101ToV2()
	// Migrate config V2 to V3
	migrateConfigV2ToV3()
	// Migrate config V3 to V4
	migrateConfigV3ToV4()
	// Migrate config V4 to V5
	migrateConfigV4ToV5()
	// Migrate config V5 to V6
	migrateConfigV5ToV6()
	// Migrate config V6 to V7
	migrateConfigV6ToV7()
	// Migrate config V7 to V8
	migrateConfigV7ToV8()
}

// Migrate from config version 1.0 to 1.0.1. Populate example entries and save it back.
func migrateConfigV1ToV101() {
	if !isMcConfigExists() {
		return
	}
	mcCfgV1, err := quick.Load(mustGetMcConfigPath(), newConfigV1())
	fatalIf(err.Trace(), "Unable to load config version ‘1’.")

	// If loaded config version does not match 1.0.0, we do nothing.
	if mcCfgV1.Version() != "1.0.0" {
		return
	}

	// 1.0.1 is compatible to 1.0.0. We are just adding new entries.
	cfgV101 := newConfigV101()

	// Copy aliases.
	for k, v := range mcCfgV1.Data().(*configV1).Aliases {
		cfgV101.Aliases[k] = v
	}

	// Copy hosts.
	for k, hostCfgV1 := range mcCfgV1.Data().(*configV1).Hosts {
		cfgV101.Hosts[k] = hostConfigV101{
			AccessKeyID:     hostCfgV1.AccessKeyID,
			SecretAccessKey: hostCfgV1.SecretAccessKey,
		}
	}

	// Example localhost entry.
	if _, ok := cfgV101.Hosts["localhost:*"]; !ok {
		cfgV101.Hosts["localhost:*"] = hostConfigV101{}
	}

	// Example loopback IP entry.
	if _, ok := cfgV101.Hosts["127.0.0.1:*"]; !ok {
		cfgV101.Hosts["127.0.0.1:*"] = hostConfigV101{}
	}

	// Example AWS entry.
	// Look for glob string (not glob match). We used to support glob based key matching earlier.
	if _, ok := cfgV101.Hosts["*.s3*.amazonaws.com"]; !ok {
		cfgV101.Hosts["*.s3*.amazonaws.com"] = hostConfigV101{
			AccessKeyID:     "YOUR-ACCESS-KEY-ID-HERE",
			SecretAccessKey: "YOUR-SECRET-ACCESS-KEY-HERE",
		}
	}

	// Save the new config back to the disk.
	mcCfgV101, err := quick.New(cfgV101)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘1.0.1’.")
	err = mcCfgV101.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘1.0.1’.")

	console.Infof("Successfully migrated %s from version ‘1.0.0’ to version ‘1.0.1’.\n", mustGetMcConfigPath())
}

// Migrate from config ‘1.0.1’ to ‘2’. Drop semantic versioning and move to integer versioning. No other changes.
func migrateConfigV101ToV2() {
	if !isMcConfigExists() {
		return
	}
	mcCfgV101, err := quick.Load(mustGetMcConfigPath(), newConfigV101())
	fatalIf(err.Trace(), "Unable to load config version ‘1.0.1’.")

	// update to newer version
	if mcCfgV101.Version() != "1.0.1" {
		return
	}

	cfgV2 := newConfigV2()

	// Copy aliases.
	for k, v := range mcCfgV101.Data().(*configV101).Aliases {
		cfgV2.Aliases[k] = v
	}

	// Copy hosts.
	for k, hostCfgV101 := range mcCfgV101.Data().(*configV101).Hosts {
		cfgV2.Hosts[k] = hostConfigV2{
			AccessKeyID:     hostCfgV101.AccessKeyID,
			SecretAccessKey: hostCfgV101.SecretAccessKey,
		}
	}

	mcCfgV2, err := quick.New(cfgV2)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘2’.")

	err = mcCfgV2.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘2’.")

	console.Infof("Successfully migrated %s from version ‘1.0.1’ to version ‘2’.\n", mustGetMcConfigPath())
}

// Migrate from config ‘2’ to ‘3’. Use ‘-’ separated names for
// hostConfig using struct json tags.
func migrateConfigV2ToV3() {
	if !isMcConfigExists() {
		return
	}

	mcCfgV2, err := quick.Load(mustGetMcConfigPath(), newConfigV2())
	fatalIf(err.Trace(), "Unable to load mc config V2.")

	// update to newer version
	if mcCfgV2.Version() != "2" {
		return
	}

	cfgV3 := newConfigV3()

	// Copy aliases.
	for k, v := range mcCfgV2.Data().(*configV2).Aliases {
		cfgV3.Aliases[k] = v
	}

	// Copy hosts.
	for k, hostCfgV2 := range mcCfgV2.Data().(*configV2).Hosts {
		// New hostConfV3 uses struct json tags.
		cfgV3.Hosts[k] = hostConfigV3{
			AccessKeyID:     hostCfgV2.AccessKeyID,
			SecretAccessKey: hostCfgV2.SecretAccessKey,
		}
	}

	mcNewCfgV3, err := quick.New(cfgV3)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘3’.")

	err = mcNewCfgV3.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘3’.")

	console.Infof("Successfully migrated %s from version ‘2’ to version ‘3’.\n", mustGetMcConfigPath())
}

// Migrate from config version ‘3’ to ‘4’. Introduce API Signature
// field in host config. Also Use JavaScript notation for field names.
func migrateConfigV3ToV4() {
	if !isMcConfigExists() {
		return
	}
	mcCfgV3, err := quick.Load(mustGetMcConfigPath(), newConfigV3())
	fatalIf(err.Trace(), "Unable to load mc config V2.")

	// update to newer version
	if mcCfgV3.Version() != "3" {
		return
	}

	cfgV4 := newConfigV4()
	for k, v := range mcCfgV3.Data().(*configV3).Aliases {
		cfgV4.Aliases[k] = v
	}
	// New hostConfig has API signature. All older entries were V4
	// only. So it is safe to assume V4 as default for all older
	// entries.
	// HostConfigV4 als uses JavaScript naming notation for struct JSON tags.
	for host, hostCfgV3 := range mcCfgV3.Data().(*configV3).Hosts {
		cfgV4.Hosts[host] = hostConfigV4{
			AccessKeyID:     hostCfgV3.AccessKeyID,
			SecretAccessKey: hostCfgV3.SecretAccessKey,
			Signature:       "v4",
		}
	}

	mcNewCfgV4, err := quick.New(cfgV4)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘4’.")

	err = mcNewCfgV4.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘4’.")

	console.Infof("Successfully migrated %s from version ‘3’ to version ‘4’.\n", mustGetMcConfigPath())

}

// Migrate config version ‘4’ to ‘5’. Rename hostConfigV4.Signature  -> hostConfigV5.API.
func migrateConfigV4ToV5() {
	if !isMcConfigExists() {
		return
	}
	mcCfgV4, err := quick.Load(mustGetMcConfigPath(), newConfigV4())
	fatalIf(err.Trace(), "Unable to load mc config V4.")

	// update to newer version
	if mcCfgV4.Version() != "4" {
		return
	}

	cfgV5 := newConfigV5()
	for k, v := range mcCfgV4.Data().(*configV4).Aliases {
		cfgV5.Aliases[k] = v
	}
	for host, hostCfgV4 := range mcCfgV4.Data().(*configV4).Hosts {
		cfgV5.Hosts[host] = hostConfigV5{
			AccessKeyID:     hostCfgV4.AccessKeyID,
			SecretAccessKey: hostCfgV4.SecretAccessKey,
			API:             "v4", // Rename from .Signature to .API
		}
	}

	mcNewCfgV5, err := quick.New(cfgV5)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘5’.")

	err = mcNewCfgV5.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘5’.")

	console.Infof("Successfully migrated %s from version ‘4’ to version ‘5’.\n", mustGetMcConfigPath())
}

// Migrate config version ‘5’ to ‘6’. Add google cloud storage servers
// to host config. Also remove "." from s3 aws glob rule.
func migrateConfigV5ToV6() {
	if !isMcConfigExists() {
		return
	}
	mcCfgV5, err := quick.Load(mustGetMcConfigPath(), newConfigV5())
	fatalIf(err.Trace(), "Unable to load mc config V5.")

	// update to newer version
	if mcCfgV5.Version() != "5" {
		return
	}

	cfgV6 := newConfigV6()

	// Add new Google Cloud Storage alias.
	cfgV6.Aliases["gcs"] = "https://storage.googleapis.com"

	for k, v := range mcCfgV5.Data().(*configV5).Aliases {
		cfgV6.Aliases[k] = v
	}

	// Add defaults.
	cfgV6.Hosts["*s3*amazonaws.com"] = hostConfigV6{
		AccessKeyID:     "YOUR-ACCESS-KEY-ID-HERE",
		SecretAccessKey: "YOUR-SECRET-ACCESS-KEY-HERE",
		API:             "S3v4",
	}
	cfgV6.Hosts["*storage.googleapis.com"] = hostConfigV6{
		AccessKeyID:     "YOUR-ACCESS-KEY-ID-HERE",
		SecretAccessKey: "YOUR-SECRET-ACCESS-KEY-HERE",
		API:             "S3v2",
	}

	for host, hostCfgV5 := range mcCfgV5.Data().(*configV5).Hosts {
		// Find any matching s3 entry and copy keys from it to newer generalized glob entry.
		if strings.Contains(host, "s3") {
			if (hostCfgV5.AccessKeyID == "YOUR-ACCESS-KEY-ID-HERE") ||
				(hostCfgV5.SecretAccessKey == "YOUR-SECRET-ACCESS-KEY-HERE") ||
				hostCfgV5.AccessKeyID == "" ||
				hostCfgV5.SecretAccessKey == "" {
				continue // Skip defaults.
			}
			// Now we have real keys set by the user. Copy
			// them over to newer glob rule.
			// Original host entry has "." in the glob rule.
			host = "*s3*amazonaws.com" // Use this glob entry.
		}

		cfgV6.Hosts[host] = hostConfigV6{
			AccessKeyID:     hostCfgV5.AccessKeyID,
			SecretAccessKey: hostCfgV5.SecretAccessKey,
			API:             hostCfgV5.API,
		}
	}

	mcNewCfgV6, err := quick.New(cfgV6)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘6’.")

	err = mcNewCfgV6.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘6’.")

	console.Infof("Successfully migrated %s from version ‘5’ to version ‘6’.\n", mustGetMcConfigPath())
}

// Migrate config version ‘6’ to ‘7'. Remove alias map and introduce
// named Host config. Also no more glob match for host config entries.
func migrateConfigV6ToV7() {
	if !isMcConfigExists() {
		return
	}

	mcCfgV6, err := quick.Load(mustGetMcConfigPath(), newConfigV6())
	fatalIf(err.Trace(), "Unable to load mc config V6.")

	if mcCfgV6.Version() != "6" {
		return
	}

	cfgV7 := newConfigV7()
	aliasIndex := 0

	// old Aliases.
	oldAliases := mcCfgV6.Data().(*configV6).Aliases

	// We dropped alias support in v7. We only need to migrate host configs.
	for host, hostCfgV6 := range mcCfgV6.Data().(*configV6).Hosts {
		// Look through old aliases, if found any matching save those entries.
		for aliasName, aliasedHost := range oldAliases {
			if aliasedHost == host {
				cfgV7.Hosts[aliasName] = hostConfigV7{
					URL:       host,
					AccessKey: hostCfgV6.AccessKeyID,
					SecretKey: hostCfgV6.SecretAccessKey,
					API:       hostCfgV6.API,
				}
				continue
			}
		}
		if hostCfgV6.AccessKeyID == "YOUR-ACCESS-KEY-ID-HERE" ||
			hostCfgV6.SecretAccessKey == "YOUR-SECRET-ACCESS-KEY-HERE" ||
			hostCfgV6.AccessKeyID == "" ||
			hostCfgV6.SecretAccessKey == "" {
			// Ignore default entries. configV7.loadDefaults() will re-insert them back.
		} else if host == "https://s3.amazonaws.com" {
			// Only one entry can exist for "s3" domain.
			cfgV7.Hosts["s3"] = hostConfigV7{
				URL:       host,
				AccessKey: hostCfgV6.AccessKeyID,
				SecretKey: hostCfgV6.SecretAccessKey,
				API:       hostCfgV6.API,
			}
		} else if host == "https://storage.googleapis.com" {
			// Only one entry can exist for "gcs" domain.
			cfgV7.Hosts["gcs"] = hostConfigV7{
				URL:       host,
				AccessKey: hostCfgV6.AccessKeyID,
				SecretKey: hostCfgV6.SecretAccessKey,
				API:       hostCfgV6.API,
			}
		} else {
			// Assign a generic "cloud1", cloud2..." key
			// for all other entries that has valid keys set.
			alias := fmt.Sprintf("cloud%d", aliasIndex)
			aliasIndex++
			cfgV7.Hosts[alias] = hostConfigV7{
				URL:       host,
				AccessKey: hostCfgV6.AccessKeyID,
				SecretKey: hostCfgV6.SecretAccessKey,
				API:       hostCfgV6.API,
			}
		}
	}
	// Load default settings.
	cfgV7.loadDefaults()
	mcNewCfgV7, err := quick.New(cfgV7)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘7’.")

	err = mcNewCfgV7.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘7’.")

	console.Infof("Successfully migrated %s from version ‘6’ to version ‘7’.\n", mustGetMcConfigPath())
}

// Migrate config version ‘7’ to ‘8'. Remove hosts
// 'play.minio.io:9002' and 'dl.minio.io:9000'.
func migrateConfigV7ToV8() {
	if !isMcConfigExists() {
		return
	}

	mcCfgV7, err := quick.Load(mustGetMcConfigPath(), newConfigV7())
	fatalIf(err.Trace(), "Unable to load mc config V7.")

	if mcCfgV7.Version() != "7" {
		return
	}

	cfgV8 := newConfigV8()
	// We dropped alias support in v7. We only need to migrate host configs.
	for host, hostCfgV7 := range mcCfgV7.Data().(*configV7).Hosts {
		// Ignore 'player', 'play' and 'dl' aliases.
		if host == "player" || host == "dl" || host == "play" {
			continue
		}
		hostCfgV8 := hostConfigV8{}
		hostCfgV8.URL = hostCfgV7.URL
		hostCfgV8.AccessKey = hostCfgV7.AccessKey
		hostCfgV8.SecretKey = hostCfgV7.SecretKey
		hostCfgV8.API = hostCfgV7.API
		cfgV8.Hosts[host] = hostCfgV8
	}
	// Load default settings.
	cfgV8.loadDefaults()
	mcNewCfgV8, err := quick.New(cfgV8)
	fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘8’.")

	err = mcNewCfgV8.Save(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to save config version ‘8’.")

	console.Infof("Successfully migrated %s from version ‘7’ to version ‘8’.\n", mustGetMcConfigPath())
}
