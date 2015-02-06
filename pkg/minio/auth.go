package minio

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type Auth struct {
	AccessKey string
	SecretKey string
	Hostname  string
	CertPEM   string
	KeyPEM    string
}

type TlsConfig struct {
	CertPEMBlock []byte
	KeyPEMBlock  []byte
}

func (a *Auth) loadKeys(cert string, key string) (*TlsConfig, error) {
	certBlock, err := ioutil.ReadFile(cert)
	if err != nil {
		return nil, err
	}
	keyBlock, err := ioutil.ReadFile(key)
	if err != nil {
		return nil, err
	}
	t := &TlsConfig{}
	t.CertPEMBlock = certBlock
	t.KeyPEMBlock = keyBlock
	return t, nil
}

func (a *Auth) getTlsTransport() (*http.Transport, error) {
	if a.CertPEM == "" || a.KeyPEM == "" {
		return &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
		}, nil
	}

	tlsconfig, err := a.loadKeys(a.CertPEM, a.KeyPEM)
	if err != nil {
		return nil, err
	}
	var cert tls.Certificate
	cert, err = tls.X509KeyPair(tlsconfig.CertPEMBlock, tlsconfig.KeyPEMBlock)
	if err != nil {
		return nil, err
	}

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return transport, nil
}
