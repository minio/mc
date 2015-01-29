package minio

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
)

type Auth struct {
	AccessKey       string
	SecretAccessKey string
	Hostname        string
	CACert          string
	CertPEM         string
	KeyPEM          string
}

type TlsConfig struct {
	CABlock      []byte
	CertPEMBlock []byte
	KeyPEMBlock  []byte
}

func (a *Auth) loadKeys(cert string, key string) (*TlsConfig, error) {
	caBlock, err := ioutil.ReadFile(cert)
	if err != nil {
		return nil, err
	}
	keyBlock, err := ioutil.ReadFile(key)
	if err != nil {
		return nil, err
	}
	t := &TlsConfig{}
	t.CABlock = caBlock
	t.CertPEMBlock = caBlock
	t.KeyPEMBlock = keyBlock
	return t, nil
}

func (a *Auth) getTlsTransport() (*http.Transport, error) {
	tlsconfig, err := a.loadKeys(a.CACert, a.KeyPEM)
	if err != nil {
		return nil, err
	}
	var cert tls.Certificate
	cert, err = tls.X509KeyPair(tlsconfig.CertPEMBlock, tlsconfig.KeyPEMBlock)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(tlsconfig.CABlock)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return transport, nil
}
