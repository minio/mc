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
