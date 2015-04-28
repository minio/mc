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
	"os"
	"os/user"
	"path"
	"runtime"

	"github.com/minio-io/mc/pkg/quick"
	"github.com/minio-io/minio/pkg/iodine"
)

type configV1 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]*hostConfig
}

// getMcConfigDir - construct minio client config folder
func getMcConfigDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	// For windows the path is slightly different
	switch runtime.GOOS {
	case "windows":
		return path.Join(u.HomeDir, mcConfigWindowsDir), nil
	default:
		return path.Join(u.HomeDir, mcConfigDir), nil
	}
}

// createMcConfigDir - create minio client config folder
func createMcConfigDir() error {
	p, err := getMcConfigDir()
	if err != nil {
		return iodine.New(err, nil)
	}
	err = os.MkdirAll(p, 0700)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// getMcConfigPath - construct minio client configuration path
func getMcConfigPath() (string, error) {
	dir, err := getMcConfigDir()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return path.Join(dir, mcConfigFile), nil
}

// mustGetMcConfigPath - similar to getMcConfigPath, ignores errors
func mustGetMcConfigPath() string {
	p, _ := getMcConfigPath()
	return p
}

// getMcConfig - reads configuration file and returns config
func getMcConfig() (config *configV1, err error) {
	if !isMcConfigExist() {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	conf := newConfigV1()
	if config := quick.New(conf); config != nil {
		if err := config.Load(configFile); err != nil {
			return nil, iodine.New(err, nil)
		}
		return config.Data().(*configV1), nil
	}
	return nil, iodine.New(errInvalidArgument{}, nil)
}

// isMcConfigExist returns true/false if config exists
func isMcConfigExist() bool {
	configFile, err := getMcConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configFile)
	if err != nil {
		return false
	}
	return true
}

// writeConfig - write configuration file
func writeConfig(config quick.Config) error {
	err := createMcConfigDir()
	if err != nil {
		return iodine.New(err, nil)
	}
	configPath, err := getMcConfigPath()
	if err != nil {
		return iodine.New(err, nil)
	}
	if err := config.Save(configPath); err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// newConfigV1() - get new config version 1.0
func newConfigV1() *configV1 {
	conf := new(configV1)
	conf.Version = mcCurrentConfigVersion
	// make sure to allocate map's otherwise Golang
	// exists silently without providing any errors
	conf.Hosts = make(map[string]*hostConfig)
	conf.Aliases = make(map[string]string)
	return conf
}

// newConfig - get new config interface
func newConfig() (config quick.Config) {
	conf := newConfigV1()
	s3HostConf := new(hostConfig)
	s3HostConf.AccessKeyID = globalAccessKeyID
	s3HostConf.SecretAccessKey = globalSecretAccessKey

	// Your example host config
	exampleHostConf := new(hostConfig)
	exampleHostConf.AccessKeyID = globalAccessKeyID
	exampleHostConf.SecretAccessKey = globalSecretAccessKey

	conf.Hosts[exampleHostURL] = exampleHostConf
	conf.Hosts["http*://s3*.amazonaws.com"] = s3HostConf

	aliases := make(map[string]string)
	aliases["s3"] = "https://s3.amazonaws.com"
	aliases["localhost"] = "http://localhost:9000"
	conf.Aliases = aliases
	config = quick.New(conf)

	return config
}
