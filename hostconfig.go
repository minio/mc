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
	"path/filepath"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/probe"
)

type hostConfig struct {
	AccessKeyID     string
	SecretAccessKey string
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(URL string) (hostConfig, *probe.Error) {
	config, err := getMcConfig()
	if err != nil {
		return hostConfig{}, err.Trace()
	}
	{
		url, err := client.Parse(URL)
		if err != nil {
			return hostConfig{}, probe.NewError(err)
		}
		// No host matching or keys needed for filesystem requests
		if url.Type == client.Filesystem {
			hostCfg := hostConfig{
				AccessKeyID:     "",
				SecretAccessKey: "",
			}
			return hostCfg, nil
		}
		for globURL, hostCfg := range config.Hosts {
			match, err := filepath.Match(globURL, url.Host)
			if err != nil {
				return hostConfig{}, probe.NewError(eInvalidGlobURL{glob: globURL, request: URL})
			}
			if match {
				return hostCfg, nil
			}
		}
	}
	return hostConfig{}, probe.NewError(eNoMatchingHost{URL: URL})
}
