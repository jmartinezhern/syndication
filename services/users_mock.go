// Code generated by MockGen. DO NOT EDIT.
// Source: users.go

// Package services is a generated GoMock package.
package services

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/jmartinezhern/syndication/models"
)

// MockUsers is a mock of Users interface.
type MockUsers struct {
	ctrl     *gomock.Controller
	recorder *MockUsersMockRecorder
}

// MockUsersMockRecorder is the mock recorder for MockUsers.
type MockUsersMockRecorder struct {
	mock *MockUsers
}

// NewMockUsers creates a new mock instance.
func NewMockUsers(ctrl *gomock.Controller) *MockUsers {
	mock := &MockUsers{ctrl: ctrl}
	mock.recorder = &MockUsersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUsers) EXPECT() *MockUsersMockRecorder {
	return m.recorder
}

// DeleteUser mocks base method.
func (m *MockUsers) DeleteUser(id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockUsersMockRecorder) DeleteUser(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockUsers)(nil).DeleteUser), id)
}

// NewUser mocks base method.
func (m *MockUsers) NewUser(username, password string) (models.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewUser", username, password)
	ret0, _ := ret[0].(models.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewUser indicates an expected call of NewUser.
func (mr *MockUsersMockRecorder) NewUser(username, password interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewUser", reflect.TypeOf((*MockUsers)(nil).NewUser), username, password)
}

// User mocks base method.
func (m *MockUsers) User(id string) (models.User, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "User", id)
	ret0, _ := ret[0].(models.User)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// User indicates an expected call of User.
func (mr *MockUsersMockRecorder) User(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "User", reflect.TypeOf((*MockUsers)(nil).User), id)
}

// Users mocks base method.
func (m *MockUsers) Users(page models.Page) ([]models.User, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Users", page)
	ret0, _ := ret[0].([]models.User)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// Users indicates an expected call of Users.
func (mr *MockUsersMockRecorder) Users(page interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Users", reflect.TypeOf((*MockUsers)(nil).Users), page)
}
