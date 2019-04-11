/*
 * MinIO Client (C) 2015 MinIO, Inc.
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
	"sync"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

const (
	defaultAccessKey = "YOUR-ACCESS-KEY-HERE"
	defaultSecretKey = "YOUR-SECRET-KEY-HERE"
)

var (
	// set once during first load.
	cacheCfgV9 *configV9
	// All access to mc config file should be synchronized.
	cfgMutex = &sync.RWMutex{}
)

// hostConfig configuration of a host.
type hostConfigV9 struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`
	Lookup    string `json:"lookup"`
}

// configV8 config version.
type configV9 struct {
	Version string                  `json:"version"`
	Hosts   map[string]hostConfigV9 `json:"hosts"`
}

// newConfigV9 - new config version.
func newConfigV9() *configV9 {
	cfg := new(configV9)
	cfg.Version = globalMCConfigVersion
	cfg.Hosts = make(map[string]hostConfigV9)
	return cfg
}

// SetHost sets host config if not empty.
func (c *configV9) setHost(alias string, cfg hostConfigV9) {
	if _, ok := c.Hosts[alias]; !ok {
		c.Hosts[alias] = cfg
	}
}

// load default values for missing entries.
func (c *configV9) loadDefaults() {
	// MinIO server running locally.
	c.setHost("local", hostConfigV9{
		URL:       "http://localhost:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
		Lookup:    "auto",
	})

	// Amazon S3 cloud storage service.
	c.setHost("s3", hostConfigV9{
		URL:       "https://s3.amazonaws.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v4",
		Lookup:    "dns",
	})

	// Google cloud storage service.
	c.setHost("gcs", hostConfigV9{
		URL:       "https://storage.googleapis.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v2",
		Lookup:    "dns",
	})

	// MinIO anonymous server for demo.
	c.setHost("play", hostConfigV9{
		URL:       "https://play.min.io:9000",
		AccessKey: "Q3AM3UQ867SPQQA43P2F",
		SecretKey: "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		API:       "S3v4",
		Lookup:    "auto",
	})
}

// loadConfigV9 - loads a new config.
func loadConfigV9() (*configV9, *probe.Error) {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	// If already cached, return the cached value.
	if cacheCfgV9 != nil {
		return cacheCfgV9, nil
	}

	if !isMcConfigExists() {
		return nil, errInvalidArgument().Trace()
	}

	// Initialize a new config loader.
	qc, e := quick.NewConfig(newConfigV9(), nil)
	if e != nil {
		return nil, probe.NewError(e)
	}

	// Load config at configPath, fails if config is not
	// accessible, malformed or version missing.
	if e = qc.Load(mustGetMcConfigPath()); e != nil {
		return nil, probe.NewError(e)
	}

	cfgV9 := qc.Data().(*configV9)

	// Cache config.
	cacheCfgV9 = cfgV9

	// Success.
	return cfgV9, nil
}

// saveConfigV8 - saves an updated config.
func saveConfigV9(cfgV9 *configV9) *probe.Error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	qs, e := quick.NewConfig(cfgV9, nil)
	if e != nil {
		return probe.NewError(e)
	}

	// update the cache.
	cacheCfgV9 = cfgV9

	e = qs.Save(mustGetMcConfigPath())
	if e != nil {
		return probe.NewError(e).Trace(mustGetMcConfigPath())
	}
	return nil
}
