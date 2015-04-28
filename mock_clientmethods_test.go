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

import (
	"io"

	"github.com/minio-io/mc/pkg/client"
	"github.com/stretchr/testify/mock"
)

// MockclientMethods is a mock for testing, please ignore.
type MockclientMethods struct {
	mock.Mock
}

func (m *MockclientMethods) getSourceReader(urlStr string, auth *hostConfig) (io.ReadCloser, int64, string, error) {
	ret := m.Called(urlStr, auth)

	var r0 io.ReadCloser
	untypedR0 := ret.Get(0)
	if untypedR0 != nil {
		r0 = untypedR0.(io.ReadCloser)
	} else {
		r0 = nil
	}
	r1 := ret.Get(1).(int64)
	r2 := ret.Get(2).(string)
	r3 := ret.Error(3)

	return r0, r1, r2, r3
}
func (m *MockclientMethods) getTargetWriter(urlStr string, auth *hostConfig, md5Hex string, length int64) (io.WriteCloser, error) {
	ret := m.Called(urlStr, auth, md5Hex, length)

	var r0 io.WriteCloser
	untypedR0 := ret.Get(0)
	if untypedR0 != nil {
		r0 = untypedR0.(io.WriteCloser)
	} else {
		r0 = nil
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *MockclientMethods) getNewClient(urlStr string, auth *hostConfig, debug bool) (client.Client, error) {
	ret := m.Called(urlStr, auth, debug)

	var r0 client.Client
	untypedR0 := ret.Get(0)
	if untypedR0 != nil {
		r0 = untypedR0.(client.Client)
	} else {
		r0 = nil
	}
	r1 := ret.Error(1)

	return r0, r1
}
