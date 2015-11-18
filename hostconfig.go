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
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio-xl/pkg/probe"
)

// hostConfig configuration of a host.
type hostConfig struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	API             string `json:"api"`
}

// getHostConfig retrieves host specific configuration such as access keys, signature type.
func getHostConfig(URL string) (hostConfig, *probe.Error) {
	config, err := getMcConfig()
	if err != nil {
		return hostConfig{}, err.Trace()
	}
	url := client.NewURL(URL)
	// No host matching or keys needed for filesystem requests.
	if url.Type == client.Filesystem {
		hostCfg := hostConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
			API:             "fs",
		}
		return hostCfg, nil
	}
	// if host is exact return quickly.
	if _, ok := config.Hosts[url.Host]; ok {
		return config.Hosts[url.Host], nil
	}
	// if host matches a suffix return.
	for savedURL, hostCfg := range config.Hosts {
		if strings.HasSuffix(url.Host, savedURL) {
			return hostCfg, nil
		}
	}
	// return error if cannot be matched.
	return hostConfig{}, errNoMatchingHost(URL).Trace()
}
