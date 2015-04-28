/*
 * Minio Client (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this fs except in compliance with the License.
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

package mocks

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

// httpTracer is a mock object
type httpTracer struct {
	mock.Mock
}

// Request is a mock method
func (m *httpTracer) Request(req *http.Request) error {
	ret := m.Called(req)

	r0 := ret.Error(0)

	return r0
}

// Response is a mock method
func (m *httpTracer) Response(res *http.Response) error {
	ret := m.Called(res)

	r0 := ret.Error(0)

	return r0
}
