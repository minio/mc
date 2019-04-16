/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

var (
	// set once during first load.
	cacheCfgV10 *configV10
)

type hostConfigV10 struct {
	URL          string `json:"url"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	API          string `json:"api"`
	Lookup       string `json:"lookup"`
	SessionToken string `json:"sessionToken"`
}

type configV10 struct {
	Version string                   `json:"version"`
	Hosts   map[string]hostConfigV10 `json:"hosts"`
}

// newConfigV10 - new config version.
func newConfigV10() *configV10 {
	cfg := new(configV10)
	cfg.Version = globalMCConfigVersion
	cfg.Hosts = make(map[string]hostConfigV10)
	return cfg
}

// SetHost sets host config if not empty.
func (c *configV10) setHost(alias string, cfg hostConfigV10) {
	if _, ok := c.Hosts[alias]; !ok {
		c.Hosts[alias] = cfg
	}
}

// load default values for missing entries.
func (c *configV10) loadDefaults() {
	// MinIO server running locally.
	c.setHost("local", hostConfigV10{
		URL:       "http://localhost:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
		Lookup:    "auto",
	})

	// Amazon S3 cloud storage service.
	c.setHost("s3", hostConfigV10{
		URL:       "https://s3.amazonaws.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v4",
		Lookup:    "dns",
	})

	// Google cloud storage service.
	c.setHost("gcs", hostConfigV10{
		URL:       "https://storage.googleapis.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v2",
		Lookup:    "dns",
	})

	// MinIO anonymous server for demo.
	c.setHost("play", hostConfigV10{
		URL:       "https://play.min.io:9000",
		AccessKey: "Q3AM3UQ867SPQQA43P2F",
		SecretKey: "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		API:       "S3v4",
		Lookup:    "auto",
	})
}

// loadConfigV10 - loads a new config.
func loadConfigV10() (*configV10, *probe.Error) {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	// If already cached, return the cached value.
	if cacheCfgV10 != nil {
		return cacheCfgV10, nil
	}

	if !isMcConfigExists() {
		return nil, errInvalidArgument().Trace()
	}

	// Initialize a new config loader.
	qc, e := quick.NewConfig(newConfigV10(), nil)
	if e != nil {
		return nil, probe.NewError(e)
	}

	// Load config at configPath, fails if config is not
	// accessible, malformed or version missing.
	if e = qc.Load(mustGetMcConfigPath()); e != nil {
		return nil, probe.NewError(e)
	}

	cfgV10 := qc.Data().(*configV10)

	// Cache config.
	cacheCfgV10 = cfgV10

	// Success.
	return cfgV10, nil
}

func saveConfigV10(cfgV10 *configV10) *probe.Error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	qs, e := quick.NewConfig(cfgV10, nil)
	if e != nil {
		return probe.NewError(e)
	}

	// update the cache.
	cacheCfgV10 = cfgV10

	e = qs.Save(mustGetMcConfigPath())
	if e != nil {
		return probe.NewError(e).Trace(mustGetMcConfigPath())
	}
	return nil
}
