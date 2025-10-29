// Copyright (c) 2015-2022 MinIO, Inc.
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
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/mc/pkg/probe"
)

func marshalPublicKey(pub any) (publicKeyBytes []byte, e error) {
	// pkcs1PublicKey reflects the ASN.1 structure of a PKCS #1 public key.
	type pkcs1PublicKey struct {
		N *big.Int
		E int
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		publicKeyBytes, e = asn1.Marshal(pkcs1PublicKey{
			N: pub.N,
			E: pub.E,
		})
		if e != nil {
			return nil, e
		}
	case *ecdsa.PublicKey:
		pubKey, e := pub.ECDH()
		if e != nil {
			return nil, e
		}
		publicKeyBytes = pubKey.Bytes()
	case ed25519.PublicKey:
		publicKeyBytes = pub
	default:
		return nil, fmt.Errorf("x509: unsupported public key type: %T", pub)
	}

	return publicKeyBytes, nil
}

// promptTrustSelfSignedCert connects to the given endpoint and
// checks whether the peer certificate can be verified.
// If not, it computes a fingerprint of the peer certificate
// public key, asks the user to confirm the fingerprint and
// adds the peer certificate to the local trust store in the
// CAs directory.
func promptTrustSelfSignedCert(ctx context.Context, endpoint, alias string) (*x509.Certificate, *probe.Error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if e != nil {
		return nil, probe.NewError(e)
	}

	// no need to probe certs for http endpoints.
	if req.URL.Scheme == "http" {
		return nil, nil
	}

	client := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialTLSContext: newCustomDialTLSContext(&tls.Config{
				RootCAs: globalRootCAs, // make sure to use loaded certs before probing
			}),
		},
	}

	_, te := client.Do(req)
	if te == nil {
		// certs are already trusted system wide, nothing to do.
		return nil, nil
	}

	if !strings.Contains(te.Error(), "certificate signed by unknown authority") &&
		!strings.Contains(te.Error(), "certificate is not trusted") /* darwin specific error message */ {
		return nil, probe.NewError(te)
	}

	// Now, we fetch the peer certificate, compute the SHA-256 of
	// public key and let the user confirm the fingerprint.
	// If the user confirms, we store the peer certificate in the CAs
	// directory and retry.
	peerCert, e := fetchPeerCertificate(ctx, endpoint)
	if e != nil {
		return nil, probe.NewError(e)
	}

	if peerCert.IsCA && len(peerCert.AuthorityKeyId) == 0 {
		// If peerCert is its own CA then AuthorityKeyId will be empty
		// which means the SubjeyKeyId is the sha1.Sum(publicKeyBytes)
		// Refer - SubjectKeyId generated using method 1 in RFC 5280, Section 4.2.1.2:
		publicKeyBytes, e := marshalPublicKey(peerCert.PublicKey)
		if e != nil {
			return nil, probe.NewError(e)
		}
		h := sha1.Sum(publicKeyBytes)
		if !bytes.Equal(h[:], peerCert.SubjectKeyId) {
			return nil, probe.NewError(te)
		}
	} else {
		// Check that the subject key id is equal to the authority key id.
		// If true, the certificate is its own issuer, and therefore, a
		// self-signed certificate. Otherwise, the certificate has been
		// issued by some other certificate that is just not trusted.
		if !bytes.Equal(peerCert.SubjectKeyId, peerCert.AuthorityKeyId) {
			return nil, probe.NewError(te)
		}
	}

	fingerprint := sha256.Sum256(peerCert.RawSubjectPublicKeyInfo)
	fmt.Printf("Fingerprint of %s public key: %s\nConfirm public key y/N: ", color.GreenString(alias), color.YellowString(hex.EncodeToString(fingerprint[:])))
	answer, e := bufio.NewReader(os.Stdin).ReadString('\n')
	if e != nil {
		return nil, probe.NewError(e)
	}
	if answer = strings.ToLower(answer); answer != "y\n" && answer != "yes\n" {
		return nil, probe.NewError(te)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: peerCert.Raw})
	if e = os.WriteFile(filepath.Join(mustGetCAsDir(), alias+".crt"), certPEM, 0o644); e != nil {
		return nil, probe.NewError(e)
	}
	return peerCert, nil
}

// fetchPeerCertificate uses the given transport to fetch the peer
// certificate from the given endpoint.
func fetchPeerCertificate(ctx context.Context, endpoint string) (*x509.Certificate, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if e != nil {
		return nil, e
	}
	client := http.Client{
		Transport: &http.Transport{
			DialTLSContext: newCustomDialTLSContext(&tls.Config{
				InsecureSkipVerify: true,
			}),
		},
	}
	resp, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("Unable to read remote TLS certificate")
	}
	return resp.TLS.PeerCertificates[0], nil
}
