package main

import (
	"io"

	"github.com/minio-io/mc/pkg/client"
	"github.com/stretchr/testify/mock"
)

// MockclientManager is a mock for testing, please ignore.
type MockclientManager struct {
	mock.Mock
}

func (m *MockclientManager) getSourceReader(urlStr string) (io.ReadCloser, int64, string, error) {
	ret := m.Called(urlStr)

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
func (m *MockclientManager) getTargetWriter(urlStr string, md5Hex string, length int64) (io.WriteCloser, error) {
	ret := m.Called(urlStr, md5Hex, length)

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
func (m *MockclientManager) getNewClient(urlStr string, debug bool) (client.Client, error) {
	ret := m.Called(urlStr, debug)

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
