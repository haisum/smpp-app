package queue

import (
	"gopkg.in/stretchr/testify.v1/mock"
)

type MockMQ struct{
	mock.Mock
}

func (m *MockMQ) Init(url string, ex string, pCount int) error{
	args := m.Called(url, ex, pCount)
	return args.Error(0)
}

func (m *MockMQ) Publish(key string, msg []byte, priority Priority) error {
	args := m.Called(key, msg, priority)
	return args.Error(0)
}


func (m *MockMQ) Bind(group string, keys []string, handler Handler) error{
	args := m.Called(group, keys, handler)
	return args.Error(0)
}

func (m *MockMQ) Close() error {
	args := m.Called()
	return args.Error(0)
}