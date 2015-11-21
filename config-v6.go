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
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
)

// configV6 config version '6'.
type configV6 struct {
	Version string                `json:"version"`
	Aliases map[string]string     `json:"alias"`
	Hosts   map[string]hostConfig `json:"hosts"`
}

// newConfigV6 - new config version '6'.
func newConfigV6() *configV6 {
	conf := new(configV6)
	conf.Version = globalMCConfigVersion
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors.
	conf.Hosts = make(map[string]hostConfig)
	conf.Aliases = make(map[string]string)

	localHostConfig := hostConfig{
		AccessKeyID:     "",
		SecretAccessKey: "",
		API:             "S3v4",
	}

	s3HostConf := hostConfig{
		AccessKeyID:     globalAccessKeyID,
		SecretAccessKey: globalSecretAccessKey,
		API:             "S3v4",
	}

	googlHostConf := hostConfig{
		AccessKeyID:     globalAccessKeyID,
		SecretAccessKey: globalSecretAccessKey,
		API:             "S3v2",
	}

	// Your example host config.
	exampleHostConf := hostConfig{
		AccessKeyID:     globalAccessKeyID,
		SecretAccessKey: globalSecretAccessKey,
		API:             "S3v4",
	}

	playHostConfig := hostConfig{
		AccessKeyID:     "",
		SecretAccessKey: "",
		API:             "S3v4",
	}

	dlHostConfig := hostConfig{
		AccessKeyID:     "",
		SecretAccessKey: "",
		API:             "S3v4",
	}

	conf.Hosts[globalExampleHostURL] = exampleHostConf
	conf.Hosts["http://localhost:9000"] = localHostConfig
	conf.Hosts["https://dl.minio.io:9000"] = dlHostConfig
	conf.Hosts["https://s3.amazonaws.com"] = s3HostConf
	conf.Hosts["https://play.minio.io:9000"] = playHostConfig
	conf.Hosts["https://storage.googleapis.com"] = googlHostConf

	aliases := make(map[string]string)
	aliases["s3"] = "https://s3.amazonaws.com"
	aliases["dl"] = "https://dl.minio.io:9000"
	aliases["gcs"] = "https://storage.googleapis.com"
	aliases["play"] = "https://play.minio.io:9000"
	aliases["local"] = "http://localhost:9000"

	conf.Aliases = aliases
	return conf
}

// saveConfigV6 - saves an updated config.
func saveConfigV6(config *configV6) *probe.Error {
	qs, err := quick.New(config)
	if err != nil {
		return err.Trace()
	}
	return qs.Save(mustGetMcConfigPath()).Trace(mustGetMcConfigPath())
}

// loadConfigV6 - loads a new config.
func loadConfigV6() (*configV6, *probe.Error) {
	if !isMcConfigExists() {
		return nil, errInvalidArgument().Trace()
	}

	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, err.Trace()
	}

	// Cached in private global variable.
	if v := cache.Get(); v != nil { // Use previously cached config.
		return v.(quick.Config).Data().(*configV6), nil
	}

	conf := new(configV6)
	conf.Version = globalMCConfigVersion
	qconf, err := quick.New(conf)
	if err != nil {
		return nil, err.Trace()
	}

	err = qconf.Load(configFile)
	if err != nil {
		return nil, err.Trace()
	}
	cache.Put(qconf)

	return qconf.Data().(*configV6), nil
}
