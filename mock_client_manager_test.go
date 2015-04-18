package main

import "github.com/stretchr/testify/mock"

import (
	"github.com/minio-io/mc/pkg/client"
	"io"
)

type mockClientManager struct {
	mock.Mock
}

func (m *mockClientManager) getSourceReader(sourceURL string) (io.ReadCloser, int64, string, error) {
	ret := m.Called(sourceURL)

	r0 := ret.Get(0).(io.ReadCloser)
	r1 := ret.Get(1).(int64)
	r2 := ret.Get(2).(string)
	r3 := ret.Error(3)

	return r0, r1, r2, r3
}
func (m *mockClientManager) getTargetWriter(targetURL string, md5Hex string, length int64) (io.WriteCloser, error) {
	ret := m.Called(targetURL, md5Hex, length)

	r0 := ret.Get(0).(io.WriteCloser)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *mockClientManager) getNewClient(targetURL string, debug bool) (client.Client, error) {
	ret := m.Called(targetURL, debug)

	r0 := ret.Get(0).(client.Client)
	r1 := ret.Error(1)

	return r0, r1
}
