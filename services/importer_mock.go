// Code generated by MockGen. DO NOT EDIT.
// Source: importer.go

// Package services is a generated GoMock package.
package services

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockImporter is a mock of Importer interface.
type MockImporter struct {
	ctrl     *gomock.Controller
	recorder *MockImporterMockRecorder
}

// MockImporterMockRecorder is the mock recorder for MockImporter.
type MockImporterMockRecorder struct {
	mock *MockImporter
}

// NewMockImporter creates a new mock instance.
func NewMockImporter(ctrl *gomock.Controller) *MockImporter {
	mock := &MockImporter{ctrl: ctrl}
	mock.recorder = &MockImporterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockImporter) EXPECT() *MockImporterMockRecorder {
	return m.recorder
}

// Import mocks base method.
func (m *MockImporter) Import(arg0 []byte, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Import", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Import indicates an expected call of Import.
func (mr *MockImporterMockRecorder) Import(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Import", reflect.TypeOf((*MockImporter)(nil).Import), arg0, arg1)
}
