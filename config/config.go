/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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

package config

import "github.com/minio-io/minio/pkg/iodine"

// Config
type Config struct {
	Version uint            // Config version
	Hosts   map[string]Auth // URL and their respective Auth map
	Aliases Aliases         // aliasName and URL map
}

// AddAlias - add a alias into existing alias list
func (c *Config) AddAlias(aliasName string, aliasURL string) error {
	c.Aliases = make(Aliases)
	if c.Aliases.IsExists(aliasName) {
		return iodine.New(AliasExists{Name: aliasName}, nil)
	}
	c.Aliases.Set(aliasName, aliasURL)
	return nil
}

func (c *Config) HostExists(hostURL string) bool {
	for host := range c.Hosts {
		if host == hostURL {
			return true
		}
	}
	return false
}

func (c *Config) AddAuth(hostURL, accessKeyID, secretAccessKey string) error {
	if c.HostExists(hostURL) {
		return iodine.New(HostExists{Name: hostURL}, nil)
	}
	auth := Auth{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}
	if !auth.IsValidAccessKey() || !auth.IsValidSecretKey() {
		return iodine.New(InvalidAuthKeys{}, nil)
	}
	c.Hosts = make(map[string]Auth)
	c.Hosts[hostURL] = auth
	return nil
}
