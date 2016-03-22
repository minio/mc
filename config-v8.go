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
	"sync"

	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

const (
	defaultAccessKey = "YOUR-ACCESS-KEY-HERE"
	defaultSecretKey = "YOUR-SECRET-KEY-HERE"
)

var (
	// set once during first load.
	cacheCfgV8 *configV8
	// All access to mc config file should be synchronized.
	cfgMutex = &sync.RWMutex{}
)

// hostConfig configuration of a host.
type hostConfigV8 struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`
}

// configV8 config version.
type configV8 struct {
	Version string                  `json:"version"`
	Hosts   map[string]hostConfigV8 `json:"hosts"`
}

// newConfigV8 - new config version.
func newConfigV8() *configV8 {
	cfg := new(configV8)
	cfg.Version = globalMCConfigVersion
	cfg.Hosts = make(map[string]hostConfigV8)
	return cfg
}

// SetHost sets host config if not empty.
func (c *configV8) setHost(alias string, cfg hostConfigV8) {
	if _, ok := c.Hosts[alias]; !ok {
		c.Hosts[alias] = cfg
	}
}

// load default values for missing entries.
func (c *configV8) loadDefaults() {
	// Minio server running locally.
	c.setHost("local", hostConfigV8{
		URL:       "http://localhost:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})

	// Amazon S3 cloud storage service.
	c.setHost("s3", hostConfigV8{
		URL:       "https://s3.amazonaws.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v4",
	})

	// Google cloud storage service.
	c.setHost("gcs", hostConfigV8{
		URL:       "https://storage.googleapis.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v2",
	})

	// Minio anonymous server for demo.
	c.setHost("play", hostConfigV8{
		URL:       "https://play.minio.io:9000",
		AccessKey: "Q3AM3UQ867SPQQA43P2F",
		SecretKey: "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		API:       "S3v4",
	})
}

// loadConfigV8 - loads a new config.
func loadConfigV8() (*configV8, *probe.Error) {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	// Cached in private global variable.
	if cacheCfgV8 != nil {
		return cacheCfgV8, nil
	}

	if !isMcConfigExists() {
		return nil, errInvalidArgument().Trace()
	}

	mcCfgV8, err := quick.Load(mustGetMcConfigPath(), newConfigV8())
	fatalIf(err.Trace(), "Unable to load mc config file ‘"+mustGetMcConfigPath()+"’.")

	cfgV8 := mcCfgV8.Data().(*configV8)

	// cache it.
	cacheCfgV8 = cfgV8

	return cfgV8, nil
}

// saveConfigV8 - saves an updated config.
func saveConfigV8(cfgV8 *configV8) *probe.Error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	qs, err := quick.New(cfgV8)
	if err != nil {
		return err.Trace()
	}

	// update the cache.
	cacheCfgV8 = cfgV8

	return qs.Save(mustGetMcConfigPath()).Trace(mustGetMcConfigPath())
}
