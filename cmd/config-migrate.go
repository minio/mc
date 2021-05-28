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
	"fmt"
	"strings"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/minio/pkg/quick"
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
	// Migrate config V8 to V9
	migrateConfigV8ToV9()
	// Migrate config V9 to V10
	migrateConfigV9ToV10()
}

// Migrate from config version 1.0 to 1.0.1. Populate example entries and save it back.
func migrateConfigV1ToV101() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version `1`.")
	if anyCfg.Version() != "1.0.0" {
		return
	}

	mcCfgV1, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV1())
	fatalIf(probe.NewError(e), "Unable to load config version `1`.")

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
	mcCfgV101, e := quick.NewConfig(cfgV101, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `1.0.1`.")
	e = mcCfgV101.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `1.0.1`.")

	console.Infof("Successfully migrated %s from version `1.0.0` to version `1.0.1`.\n", mustGetMcConfigPath())
}

// Migrate from config `1.0.1` to `2`. Drop semantic versioning and move to integer versioning. No other changes.
func migrateConfigV101ToV2() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version `1`.")
	if anyCfg.Version() != "1.0.1" {
		return
	}

	mcCfgV101, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV101())
	fatalIf(probe.NewError(e), "Unable to load config version `1.0.1`.")

	// update to newer version

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

	mcCfgV2, e := quick.NewConfig(cfgV2, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `2`.")

	e = mcCfgV2.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `2`.")

	console.Infof("Successfully migrated %s from version `1.0.1` to version `2`.\n", mustGetMcConfigPath())
}

// Migrate from config `2` to `3`. Use `-` separated names for
// hostConfig using struct json tags.
func migrateConfigV2ToV3() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "2" {
		return
	}

	mcCfgV2, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV2())
	fatalIf(probe.NewError(e), "Unable to load mc config V2.")

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

	mcNewCfgV3, e := quick.NewConfig(cfgV3, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `3`.")

	e = mcNewCfgV3.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `3`.")

	console.Infof("Successfully migrated %s from version `2` to version `3`.\n", mustGetMcConfigPath())
}

// Migrate from config version `3` to `4`. Introduce API Signature
// field in host config. Also Use JavaScript notation for field names.
func migrateConfigV3ToV4() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "3" {
		return
	}

	mcCfgV3, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV3())
	fatalIf(probe.NewError(e), "Unable to load mc config V3.")

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

	mcNewCfgV4, e := quick.NewConfig(cfgV4, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `4`.")

	e = mcNewCfgV4.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `4`.")

	console.Infof("Successfully migrated %s from version `3` to version `4`.\n", mustGetMcConfigPath())

}

// Migrate config version `4` to `5`. Rename hostConfigV4.Signature  -> hostConfigV5.API.
func migrateConfigV4ToV5() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "4" {
		return
	}

	mcCfgV4, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV4())
	fatalIf(probe.NewError(e), "Unable to load mc config V4.")

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

	mcNewCfgV5, e := quick.NewConfig(cfgV5, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `5`.")

	e = mcNewCfgV5.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `5`.")

	console.Infof("Successfully migrated %s from version `4` to version `5`.\n", mustGetMcConfigPath())
}

// Migrate config version `5` to `6`. Add google cloud storage servers
// to host config. Also remove "." from s3 aws glob rule.
func migrateConfigV5ToV6() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "5" {
		return
	}

	mcCfgV5, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV5())
	fatalIf(probe.NewError(e), "Unable to load mc config V5.")

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

	mcNewCfgV6, e := quick.NewConfig(cfgV6, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `6`.")

	e = mcNewCfgV6.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `6`.")

	console.Infof("Successfully migrated %s from version `5` to version `6`.\n", mustGetMcConfigPath())
}

// Migrate config version `6` to `7'. Remove alias map and introduce
// named Host config. Also no more glob match for host config entries.
func migrateConfigV6ToV7() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "6" {
		return
	}

	mcCfgV6, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV6())
	fatalIf(probe.NewError(e), "Unable to load mc config V6.")

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
	mcNewCfgV7, e := quick.NewConfig(cfgV7, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `7`.")

	e = mcNewCfgV7.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `7`.")

	console.Infof("Successfully migrated %s from version `6` to version `7`.\n", mustGetMcConfigPath())
}

// Migrate config version `7` to `8'. Remove hosts
// 'play.min.io:9002' and 'dl.min.io:9000'.
func migrateConfigV7ToV8() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "7" {
		return
	}

	mcCfgV7, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV7())
	fatalIf(probe.NewError(e), "Unable to load mc config V7.")

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
	mcNewCfgV8, e := quick.NewConfig(cfgV8, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `8`.")

	e = mcNewCfgV8.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `8`.")

	console.Infof("Successfully migrated %s from version `7` to version `8`.\n", mustGetMcConfigPath())
}

// Migrate config version `8` to `9'. Add optional field virtual
func migrateConfigV8ToV9() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "8" {
		return
	}

	mcCfgV8, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV8())
	fatalIf(probe.NewError(e), "Unable to load mc config V8.")

	cfgV9 := newConfigV9()
	// We dropped alias support in v8. We only need to migrate host configs.
	for host, hostCfgV8 := range mcCfgV8.Data().(*configV8).Hosts {
		// Ignore 'player', 'play' and 'dl' aliases.
		if host == "player" || host == "dl" || host == "play" {
			continue
		}
		hostCfgV9 := hostConfigV9{}
		hostCfgV9.URL = hostCfgV8.URL
		hostCfgV9.AccessKey = hostCfgV8.AccessKey
		hostCfgV9.SecretKey = hostCfgV8.SecretKey
		hostCfgV9.API = hostCfgV8.API
		hostCfgV9.Lookup = "auto"
		cfgV9.Hosts[host] = hostCfgV9
	}

	mcNewCfgV9, e := quick.NewConfig(cfgV9, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `9`.")

	e = mcNewCfgV9.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `9`.")

	console.Infof("Successfully migrated %s from version `8` to version `9`.\n", mustGetMcConfigPath())
}

// Migrate config version `9` to `10'. Rename 'hosts' to 'aliases' and 'lookup' to 'path'
func migrateConfigV9ToV10() {
	if !isMcConfigExists() {
		return
	}

	// Check the config version and quit early if the actual version is out of this function scope
	anyCfg, e := quick.LoadConfig(mustGetMcConfigPath(), nil, &ConfigAnyVersion{})
	fatalIf(probe.NewError(e), "Unable to load config version.")
	if anyCfg.Version() != "9" {
		return
	}

	mcCfgV9, e := quick.LoadConfig(mustGetMcConfigPath(), nil, newConfigV9())
	fatalIf(probe.NewError(e), "Unable to load mc config V8.")

	cfgV10 := newConfigV10()
	isEmpty := true
	// We dropped alias support in v8. We only need to migrate host configs.
	for host, hostCfgV9 := range mcCfgV9.Data().(*configV9).Hosts {
		isEmpty = false
		hostCfgV10 := aliasConfigV10{}
		hostCfgV10.URL = hostCfgV9.URL
		hostCfgV10.AccessKey = hostCfgV9.AccessKey
		hostCfgV10.SecretKey = hostCfgV9.SecretKey
		hostCfgV10.API = hostCfgV9.API
		switch hostCfgV9.Lookup {
		case "dns":
			hostCfgV10.Path = "off"
		case "path":
			hostCfgV10.Path = "on"
		default:
			hostCfgV10.Path = "auto"
		}

		cfgV10.Aliases[host] = hostCfgV10
	}

	if isEmpty {
		// Load default settings.
		cfgV10.loadDefaults()
	}

	mcNewCfgV10, e := quick.NewConfig(cfgV10, nil)
	fatalIf(probe.NewError(e), "Unable to initialize quick config for config version `10`.")

	e = mcNewCfgV10.Save(mustGetMcConfigPath())
	fatalIf(probe.NewError(e), "Unable to save config version `10`.")

	console.Infof("Successfully migrated %s from version `9` to version `10`.\n", mustGetMcConfigPath())
}
