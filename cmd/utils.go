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

package cmd

import (
	"crypto/tls"
	"errors"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

func isErrIgnored(err *probe.Error) (ignored bool) {
	// For all non critical errors we can continue for the remaining files.
	switch err.ToGoError().(type) {
	// Handle these specifically for filesystem related errors.
	case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, PathInsufficientPermission:
		ignored = true
	// Handle these specifically for object storage related errors.
	case BucketNameEmpty, ObjectMissing, ObjectAlreadyExists:
		ignored = true
	case ObjectAlreadyExistsAsDirectory, BucketDoesNotExist, BucketInvalid, ObjectOnGlacier:
		ignored = true
	default:
		ignored = false
	}
	return ignored
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// newRandomID generates a random id of regular lower case and uppercase english characters.
func newRandomID(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	sid := make([]rune, n)
	for i := range sid {
		sid[i] = letters[rand.Intn(len(letters))]
	}
	return string(sid)
}

// dumpTlsCertificates prints some fields of the certificates received from the server.
// Fields will be inspected by the user, so they must be conscise and useful
func dumpTLSCertificates(t *tls.ConnectionState) {
	for _, cert := range t.PeerCertificates {
		console.Debugln("TLS Certificate found: ")
		if len(cert.Issuer.Country) > 0 {
			console.Debugln(" >> Country: " + cert.Issuer.Country[0])
		}
		if len(cert.Issuer.Organization) > 0 {
			console.Debugln(" >> Organization: " + cert.Issuer.Organization[0])
		}
		console.Debugln(" >> Expires: " + cert.NotAfter.String())
	}
}

// isStdIO checks if the input parameter is one of the standard input/output streams
func isStdIO(reader io.Reader) bool {
	return reader == os.Stdin || reader == os.Stdout || reader == os.Stderr
}

// splitStr splits a string into n parts, empty strings are added
// if we are not able to reach n elements
func splitStr(path, sep string, n int) []string {
	splits := strings.SplitN(path, sep, n)
	// Add empty strings if we found elements less than nr
	for i := n - len(splits); i > 0; i-- {
		splits = append(splits, "")
	}
	return splits
}

// buildS3Config fetches config related to the specified alias
// to create a new config structure
func buildS3Config(alias, urlStr string) (*Config, *probe.Error) {
	hostCfg := mustGetHostConfig(alias)
	if hostCfg == nil {
		return nil, probe.NewError(errors.New("The specified alias cannot be found"))
	}

	// We have a valid alias and hostConfig. We populate the
	// credentials from the match found in the config file.
	s3Config := new(Config)

	// Fetch keys from the environnement, otherwise, get them from the config file
	keys := splitStr(os.Getenv("MC_SECRET_"+alias), ":", 2)
	if isValidAccessKey(keys[0]) && isValidSecretKey(keys[1]) {
		s3Config.AccessKey = keys[0]
		s3Config.SecretKey = keys[1]
	} else {
		if keys[0] != "" {
			console.Errorln("Access/Secret keys associated to `" + alias + "' " +
				"are found in your environment but not suitable for use. " +
				"Falling back to the standard config.")
		}
	}

	if s3Config.AccessKey == "" {
		s3Config.AccessKey = hostCfg.AccessKey
		s3Config.SecretKey = hostCfg.SecretKey
	}

	s3Config.Signature = hostCfg.API
	s3Config.AppName = "mc"
	s3Config.AppVersion = Version
	s3Config.AppComments = []string{os.Args[0], runtime.GOOS, runtime.GOARCH}
	s3Config.HostURL = urlStr
	s3Config.Debug = globalDebug
	s3Config.Insecure = globalInsecure

	return s3Config, nil
}
