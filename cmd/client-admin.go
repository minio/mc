/*
 * MinIO Client (C) 2017 MinIO, Inc.
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
	"crypto/tls"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

// newAdminFactory encloses New function with client cache.
func newAdminFactory() func(config *Config) (*madmin.AdminClient, *probe.Error) {
	clientCache := make(map[uint32]*madmin.AdminClient)
	mutex := &sync.Mutex{}

	// Return New function.
	return func(config *Config) (*madmin.AdminClient, *probe.Error) {
		// Creates a parsed URL.
		targetURL, e := url.Parse(config.HostURL)
		if e != nil {
			return nil, probe.NewError(e)
		}
		// By default enable HTTPs.
		useTLS := true
		if targetURL.Scheme == "http" {
			useTLS = false
		}

		// Save if target supports virtual host style.
		hostName := targetURL.Host

		// Generate a hash out of s3Conf.
		confHash := fnv.New32a()
		confHash.Write([]byte(hostName + config.AccessKey + config.SecretKey))
		confSum := confHash.Sum32()

		// Lookup previous cache by hash.
		mutex.Lock()
		defer mutex.Unlock()
		var api *madmin.AdminClient
		var found bool
		if api, found = clientCache[confSum]; !found {
			// Not found. Instantiate a new MinIO
			var e error
			api, e = madmin.New(hostName, config.AccessKey, config.SecretKey, useTLS)
			if e != nil {
				return nil, probe.NewError(e)
			}

			// Keep TLS config.
			tlsConfig := &tls.Config{RootCAs: globalRootCAs}
			if config.Insecure {
				tlsConfig.InsecureSkipVerify = true
			}

			var transport http.RoundTripper = &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       tlsConfig,
			}

			if config.Debug {
				transport = httptracer.GetNewTraceTransport(newTraceV4(), transport)
			}

			// Set custom transport.
			api.SetCustomTransport(transport)

			// Set app info.
			api.SetAppInfo(config.AppName, config.AppVersion)

			// Cache the new MinIO Client with hash of config as key.
			clientCache[confSum] = api
		}

		// Store the new api object.
		return api, nil
	}
}

// newAdminClient gives a new client interface
func newAdminClient(aliasedURL string) (*madmin.AdminClient, *probe.Error) {
	alias, urlStrFull, hostCfg, err := expandAlias(aliasedURL)
	if err != nil {
		return nil, err.Trace(aliasedURL)
	}
	// Verify if the aliasedURL is a real URL, fail in those cases
	// indicating the user to add alias.
	if hostCfg == nil && urlRgx.MatchString(aliasedURL) {
		return nil, errInvalidAliasedURL(aliasedURL).Trace(aliasedURL)
	}

	if hostCfg == nil {
		return nil, probe.NewError(fmt.Errorf("The specified alias: %s not found", urlStrFull))
	}

	s3Config := newS3Config(urlStrFull, hostCfg)

	s3Client, err := s3AdminNew(s3Config)
	if err != nil {
		return nil, err.Trace(alias, urlStrFull)
	}
	return s3Client, nil
}

// s3AdminNew returns an initialized minioAdmin structure. If debug is enabled,
// it also enables an internal trace transport.
var s3AdminNew = newAdminFactory()
