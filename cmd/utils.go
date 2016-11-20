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
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

func isErrIgnored(err *probe.Error) bool {
	// For all non critical errors we can continue for the remaining files.
	switch err.ToGoError().(type) {
	case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, PathInsufficientPermission:
		return true
	// Handle these specifically for object storage related errors.
	case BucketNameEmpty, ObjectMissing, ObjectAlreadyExists:
		return true
	case ObjectAlreadyExistsAsDirectory, BucketDoesNotExist, BucketInvalid, ObjectOnGlacier:
		return true
	}
	return false
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

// isBucketVirtualStyle is host virtual bucket style?.
func isBucketVirtualStyle(host string) bool {
	s3Virtual, _ := filepath.Match("*.s3*.amazonaws.com", host)
	googleVirtual, _ := filepath.Match("*.storage.googleapis.com", host)
	return s3Virtual || googleVirtual
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
