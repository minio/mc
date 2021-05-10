// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

	"github.com/mattn/go-ieproxy"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/httptracer"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NewAdminFactory encloses New function with client cache.
func NewAdminFactory() func(config *Config) (*madmin.AdminClient, *probe.Error) {
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
			// Admin API only supports signature v4.
			creds := credentials.NewStaticV4(config.AccessKey, config.SecretKey, config.SessionToken)

			// Not found. Instantiate a new MinIO
			var e error
			api, e = madmin.NewWithOptions(hostName, &madmin.Options{
				Creds:  creds,
				Secure: useTLS,
			})
			if e != nil {
				return nil, probe.NewError(e)
			}

			// Keep TLS config.
			tlsConfig := &tls.Config{
				RootCAs: globalRootCAs,
				// Can't use SSLv3 because of POODLE and BEAST
				// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
				// Can't use TLSv1.1 because of RC4 cipher usage
				MinVersion: tls.VersionTLS12,
			}
			if config.Insecure {
				tlsConfig.InsecureSkipVerify = true
			}

			var transport http.RoundTripper = &http.Transport{
				Proxy: ieproxy.GetProxyFunc(),
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 15 * time.Second,
				}).DialContext,
				MaxIdleConnsPerHost:   256,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 10 * time.Second,
				TLSClientConfig:       tlsConfig,
				// Set this value so that the underlying transport round-tripper
				// doesn't try to auto decode the body of objects with
				// content-encoding set to `gzip`.
				//
				// Refer:
				//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
				DisableCompression: true,
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
	alias, urlStrFull, aliasCfg, err := expandAlias(aliasedURL)
	if err != nil {
		return nil, err.Trace(aliasedURL)
	}
	// Verify if the aliasedURL is a real URL, fail in those cases
	// indicating the user to add alias.
	if aliasCfg == nil && urlRgx.MatchString(aliasedURL) {
		return nil, errInvalidAliasedURL(aliasedURL).Trace(aliasedURL)
	}

	if aliasCfg == nil {
		return nil, probe.NewError(fmt.Errorf("No valid configuration found for '%s' host alias", urlStrFull))
	}

	s3Config := NewS3Config(urlStrFull, aliasCfg)

	s3Client, err := s3AdminNew(s3Config)
	if err != nil {
		return nil, err.Trace(alias, urlStrFull)
	}
	return s3Client, nil
}

// s3AdminNew returns an initialized minioAdmin structure. If debug is enabled,
// it also enables an internal trace transport.
var s3AdminNew = NewAdminFactory()
