package mocks

import "github.com/stretchr/testify/mock"

import "net/http"

// HTTPTracer is a mock object
type HTTPTracer struct {
	mock.Mock
}

// Request is a mock method
func (m *HTTPTracer) Request(req *http.Request) error {
	ret := m.Called(req)

	r0 := ret.Error(0)

	return r0
}

// Response is a mock method
func (m *HTTPTracer) Response(res *http.Response) error {
	ret := m.Called(res)

	r0 := ret.Error(0)

	return r0
}
