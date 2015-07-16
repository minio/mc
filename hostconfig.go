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
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

type hostConfig struct {
	AccessKeyID     string
	SecretAccessKey string
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(URL string) (*hostConfig, error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}
	url, err := client.Parse(URL)
	if err != nil {
		return nil, NewIodine(iodine.New(errInvalidURL{URL: URL}, nil))
	}
	// No host matching or keys needed for filesystem requests
	if url.Type == client.Filesystem {
		hostCfg := &hostConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
		}
		return hostCfg, nil
	}
	// No host matching or keys needed for 127.0.0.1 URL's skip them
	if strings.Contains(url.Host, "127.0.0.1") {
		hostCfg := &hostConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
		}
		return hostCfg, nil
	}
	for globURL, hostCfg := range config.Hosts {
		match, err := filepath.Match(globURL, url.Host)
		if err != nil {
			return nil, NewIodine(iodine.New(errInvalidGlobURL{glob: globURL, request: URL}, nil))
		}
		if match {
			if hostCfg == nil {
				return nil, NewIodine(iodine.New(errInvalidAuth{}, nil))
			}
			return hostCfg, nil
		}
	}
	return nil, NewIodine(iodine.New(errNoMatchingHost{}, nil))
}

// mustGetHostConfig retrieves host specific configuration such as access keys, exits upon error
func mustGetHostConfig(URL string) *hostConfig {
	hostCfg, err := getHostConfig(URL)
	if err != nil {
		console.Fatalf("Unable to retrieve host configuration. %s\n", NewIodine(iodine.New(err, nil)))
	}
	return hostCfg
}
