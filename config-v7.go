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
	cacheCfgV7 *configV7
	// All access to mc config file should be synchronized.
	cfgMutex = &sync.RWMutex{}
)

// hostConfig configuration of a host.
type hostConfigV7 struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`
}

// configV7 config version.
type configV7 struct {
	Version string                  `json:"version"`
	Hosts   map[string]hostConfigV7 `json:"hosts"`
}

// newConfigV7 - new config version.
func newConfigV7() *configV7 {
	cfg := new(configV7)
	cfg.Version = globalMCConfigVersion
	cfg.Hosts = make(map[string]hostConfigV7)
	return cfg
}

// SetHost sets host config if not empty.
func (c *configV7) setHost(alias string, cfg hostConfigV7) {
	if _, ok := c.Hosts[alias]; !ok {
		c.Hosts[alias] = cfg
	}
}

// load default values for missing entries.
func (c *configV7) loadDefaults() {
	// Minio server running locally.
	c.setHost("local", hostConfigV7{
		URL:       "http://localhost:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})

	// Amazon S3 cloud storage service.
	c.setHost("s3", hostConfigV7{
		URL:       "https://s3.amazonaws.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v4",
	})

	// Google cloud storage service.
	c.setHost("gcs", hostConfigV7{
		URL:       "https://storage.googleapis.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v2",
	})

	// Minio anonymous server for demo.
	c.setHost("play", hostConfigV7{
		URL:       "https://play.minio.io:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})

	// Minio demo server with public secret and access keys.
	c.setHost("player", hostConfigV7{
		URL:       "https://play.minio.io:9002",
		AccessKey: "Q3AM3UQ867SPQQA43P2F",
		SecretKey: "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		API:       "S3v4",
	})

	// Minio public download service.
	c.setHost("dl", hostConfigV7{
		URL:       "https://dl.minio.io:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})
}

// loadConfigV7 - loads a new config.
func loadConfigV7() (*configV7, *probe.Error) {
	cfgMutex.RLock()
	defer cfgMutex.RUnlock()

	// Cached in private global variable.
	if cacheCfgV7 != nil {
		return cacheCfgV7, nil
	}

	if !isMcConfigExists() {
		return nil, errInvalidArgument().Trace()
	}

	mcCfgV7, err := quick.Load(mustGetMcConfigPath(), newConfigV7())
	fatalIf(err.Trace(), "Unable to load mc config file ‘"+mustGetMcConfigPath()+"’.")

	cfgV7 := mcCfgV7.Data().(*configV7)

	// cache it.
	cacheCfgV7 = cfgV7

	return cfgV7, nil
}

// saveConfigV7 - saves an updated config.
func saveConfigV7(cfgV7 *configV7) *probe.Error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	qs, err := quick.New(cfgV7)
	if err != nil {
		return err.Trace()
	}

	// update the cache.
	cacheCfgV7 = cfgV7

	return qs.Save(mustGetMcConfigPath()).Trace(mustGetMcConfigPath())
}
