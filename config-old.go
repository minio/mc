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

package main

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

/////////////////// Config V7 ///////////////////
// RESERVED FOR FUTURE
