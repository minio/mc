/*
 * Mini Copy (C) 2015 Minio, Inc.
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
	"net/url"
	"path/filepath"
	"strings"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// getHostURL -
func getHostURL(u *url.URL) string {
	return u.Scheme + "://" + u.Host
}

func getHostConfigs(requestURLs []string) (hostConfigs map[string]*hostConfig, err error) {
	hostConfigs = make(map[string]*hostConfig)
	for _, requestURL := range requestURLs {
		hostConfigs[requestURL], err = getHostConfig(requestURL)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
	}
	return hostConfigs, nil
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(requestURL string) (*hostConfig, error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, iodine.New(errInvalidURL{url: requestURL}, nil)
	}
	// No host matching or keys needed for filesystem requests
	if client.GetType(requestURL) == client.Filesystem {
		hostCfg := &hostConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
		}
		return hostCfg, nil
	}

	// No host matching or keys needed for localhost and 127.0.0.1 URL's skip them
	if strings.Contains(getHostURL(u), "localhost") || strings.Contains(getHostURL(u), "127.0.0.1") {
		hostCfg := &hostConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
		}
		return hostCfg, nil
	}
	for globURL, hostCfg := range config.Hosts {
		match, err := filepath.Match(globURL, getHostURL(u))
		if err != nil {
			return nil, iodine.New(errInvalidGlobURL{glob: globURL, request: requestURL}, nil)
		}
		if match {
			if hostCfg == nil {
				return nil, iodine.New(errInvalidAuth{}, nil)
			}
			// verify Auth key validity for all hosts
			if hostCfg.AccessKeyID != globalAccessKeyID && hostCfg.SecretAccessKey != globalSecretAccessKey {
				if !client.IsValidAccessKey(hostCfg.AccessKeyID) || !client.IsValidSecretKey(hostCfg.SecretAccessKey) {
					return nil, iodine.New(errInvalidAuthKeys{}, nil)
				}
			}
			return hostCfg, nil
		}
	}
	return nil, iodine.New(errNoMatchingHost{}, nil)
}
