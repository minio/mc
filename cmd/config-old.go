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

/////////////////// Config V1 ///////////////////
type hostConfigV1 struct {
	AccessKeyID     string
	SecretAccessKey string
}

type configV1 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]hostConfigV1
}

// newConfigV1() - get new config version 1.0.0
func newConfigV1() *configV1 {
	conf := new(configV1)
	conf.Version = "1.0.0"
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]hostConfigV1)
	return conf
}

/////////////////// Config V101 ///////////////////
type hostConfigV101 hostConfigV1

type configV101 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]hostConfigV101
}

// newConfigV101() - get new config version 1.0.1
func newConfigV101() *configV101 {
	conf := new(configV101)
	conf.Version = "1.0.1"
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]hostConfigV101)
	return conf
}

/////////////////// Config V2 ///////////////////
type hostConfigV2 hostConfigV1

type configV2 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]hostConfigV2
}

// newConfigV2() - get new config version 2
func newConfigV2() *configV2 {
	conf := new(configV2)
	conf.Version = "2"
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]hostConfigV2)
	return conf
}

/////////////////// Config V3 ///////////////////
type hostConfigV3 struct {
	AccessKeyID     string `json:"access-key-id"`
	SecretAccessKey string `json:"secret-access-key"`
}

type configV3 struct {
	Version string                  `json:"version"`
	Aliases map[string]string       `json:"alias"`
	Hosts   map[string]hostConfigV3 `json:"hosts"`
}

// newConfigV3 - get new config version 3.
func newConfigV3() *configV3 {
	conf := new(configV3)
	conf.Version = "3"
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]hostConfigV3)
	return conf
}

/////////////////// Config V4 ///////////////////
type hostConfigV4 struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Signature       string `json:"signature"`
}

type configV4 struct {
	Version string                  `json:"version"`
	Aliases map[string]string       `json:"alias"`
	Hosts   map[string]hostConfigV4 `json:"hosts"`
}

func newConfigV4() *configV4 {
	conf := new(configV4)
	conf.Version = "4"
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]hostConfigV4)
	return conf
}

/////////////////// Config V5 ///////////////////
type hostConfigV5 struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	API             string `json:"api"`
}

type configV5 struct {
	Version string                  `json:"version"`
	Aliases map[string]string       `json:"alias"`
	Hosts   map[string]hostConfigV5 `json:"hosts"`
}

func newConfigV5() *configV5 {
	conf := new(configV5)
	conf.Version = "5"
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]hostConfigV5)
	return conf
}

/////////////////// Config V6 ///////////////////
type hostConfigV6 struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	API             string `json:"api"`
}

type configV6 struct {
	Version string                  `json:"version"`
	Aliases map[string]string       `json:"alias"`
	Hosts   map[string]hostConfigV6 `json:"hosts"`
}

// newConfigV6 - new config version '6'.
func newConfigV6() *configV6 {
	conf := new(configV6)
	conf.Version = "6"
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]hostConfigV6)
	return conf
}

/////////////////// Config V6 ///////////////////
// hostConfig configuration of a host - version '7'.
type hostConfigV7 struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`
}

// configV7 config version.
type configV7 struct {
	Version string                  `json:"version"`
	Hosts   map[string]hostConfigV7 `json:"hosts"`
}

// newConfigV7 - new config version '7'.
func newConfigV7() *configV7 {
	cfg := new(configV7)
	cfg.Version = "7"
	cfg.Hosts = make(map[string]hostConfigV7)
	return cfg
}

func (c *configV7) loadDefaults() {
	// MinIO server running locally.
	c.setHost("local", hostConfigV7{
		URL:       "http://localhost:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})

	// Amazon S3 cloud storage service.
	c.setHost("s3", hostConfigV7{
		URL:       "https://s3.amazonaws.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v4",
	})

	// Google cloud storage service.
	c.setHost("gcs", hostConfigV7{
		URL:       "https://storage.googleapis.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v2",
	})

	// MinIO anonymous server for demo.
	c.setHost("play", hostConfigV7{
		URL:       "https://play.min.io",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})

	// MinIO demo server with public secret and access keys.
	c.setHost("player", hostConfigV7{
		URL:       "https://play.min.io:9002",
		AccessKey: "Q3AM3UQ867SPQQA43P2F",
		SecretKey: "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		API:       "S3v4",
	})

	// MinIO public download service.
	c.setHost("dl", hostConfigV7{
		URL:       "https://dl.min.io:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})
}

// SetHost sets host config if not empty.
func (c *configV7) setHost(alias string, cfg hostConfigV7) {
	if _, ok := c.Hosts[alias]; !ok {
		c.Hosts[alias] = cfg
	}
}

/////////////////// Config V8 ///////////////////
// configV8 config version.
// hostConfig configuration of a host.
type hostConfigV8 struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`
}
type configV8 struct {
	Version string                  `json:"version"`
	Hosts   map[string]hostConfigV8 `json:"hosts"`
}

// newConfigV8 - new config version.
func newConfigV8() *configV8 {
	cfg := new(configV8)
	cfg.Version = globalMCConfigVersion
	cfg.Hosts = make(map[string]hostConfigV8)
	return cfg
}

// SetHost sets host config if not empty.
func (c *configV8) setHost(alias string, cfg hostConfigV8) {
	if _, ok := c.Hosts[alias]; !ok {
		c.Hosts[alias] = cfg
	}
}

// load default values for missing entries.
func (c *configV8) loadDefaults() {
	// MinIO server running locally.
	c.setHost("local", hostConfigV8{
		URL:       "http://localhost:9000",
		AccessKey: "",
		SecretKey: "",
		API:       "S3v4",
	})

	// Amazon S3 cloud storage service.
	c.setHost("s3", hostConfigV8{
		URL:       "https://s3.amazonaws.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v4",
	})

	// Google cloud storage service.
	c.setHost("gcs", hostConfigV8{
		URL:       "https://storage.googleapis.com",
		AccessKey: defaultAccessKey,
		SecretKey: defaultSecretKey,
		API:       "S3v2",
	})

	// MinIO anonymous server for demo.
	c.setHost("play", hostConfigV8{
		URL:       "https://play.min.io",
		AccessKey: "Q3AM3UQ867SPQQA43P2F",
		SecretKey: "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		API:       "S3v4",
	})
}

/////////////////// Config V9 ///////////////////

// hostConfig configuration of a host.
type hostConfigV9 struct {
	URL          string `json:"url"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	SessionToken string `json:"sessionToken,omitempty"`
	API          string `json:"api"`
	Lookup       string `json:"lookup"`
}

// configV8 config version.
type configV9 struct {
	Version string                  `json:"version"`
	Hosts   map[string]hostConfigV9 `json:"hosts"`
}

func newConfigV9() *configV9 {
	cfg := new(configV9)
	cfg.Version = "9"
	cfg.Hosts = make(map[string]hostConfigV9)
	return cfg
}

/////////////////// Config V10 ///////////////////
// RESERVED FOR FUTURE
