// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/kostiamol/centerms/api (interfaces: CfgProvider,DataProvider)

// Package mock_api is a generated GoMock package.
package mock_api

import (
	gomock "github.com/golang/mock/gomock"
	svc "github.com/kostiamol/centerms/svc"
	reflect "reflect"
)

// MockCfgProvider is a mock of CfgProvider interface
type MockCfgProvider struct {
	ctrl     *gomock.Controller
	recorder *MockCfgProviderMockRecorder
}

// MockCfgProviderMockRecorder is the mock recorder for MockCfgProvider
type MockCfgProviderMockRecorder struct {
	mock *MockCfgProvider
}

// NewMockCfgProvider creates a new mock instance
func NewMockCfgProvider(ctrl *gomock.Controller) *MockCfgProvider {
	mock := &MockCfgProvider{ctrl: ctrl}
	mock.recorder = &MockCfgProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCfgProvider) EXPECT() *MockCfgProviderMockRecorder {
	return m.recorder
}

// GetDevCfg mocks base method
func (m *MockCfgProvider) GetDevCfg(arg0 string) (*svc.DevCfg, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDevCfg", arg0)
	ret0, _ := ret[0].(*svc.DevCfg)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDevCfg indicates an expected call of GetDevCfg
func (mr *MockCfgProviderMockRecorder) GetDevCfg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDevCfg", reflect.TypeOf((*MockCfgProvider)(nil).GetDevCfg), arg0)
}

// SetDevCfg mocks base method
func (m *MockCfgProvider) SetDevCfg(arg0 string, arg1 *svc.DevCfg) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetDevCfg", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetDevCfg indicates an expected call of SetDevCfg
func (mr *MockCfgProviderMockRecorder) SetDevCfg(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDevCfg", reflect.TypeOf((*MockCfgProvider)(nil).SetDevCfg), arg0, arg1)
}

// SetDevInitCfg mocks base method
func (m *MockCfgProvider) SetDevInitCfg(arg0 *svc.DevMeta) (*svc.DevCfg, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetDevInitCfg", arg0)
	ret0, _ := ret[0].(*svc.DevCfg)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SetDevInitCfg indicates an expected call of SetDevInitCfg
func (mr *MockCfgProviderMockRecorder) SetDevInitCfg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDevInitCfg", reflect.TypeOf((*MockCfgProvider)(nil).SetDevInitCfg), arg0)
}

// MockDataProvider is a mock of DataProvider interface
type MockDataProvider struct {
	ctrl     *gomock.Controller
	recorder *MockDataProviderMockRecorder
}

// MockDataProviderMockRecorder is the mock recorder for MockDataProvider
type MockDataProviderMockRecorder struct {
	mock *MockDataProvider
}

// NewMockDataProvider creates a new mock instance
func NewMockDataProvider(ctrl *gomock.Controller) *MockDataProvider {
	mock := &MockDataProvider{ctrl: ctrl}
	mock.recorder = &MockDataProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDataProvider) EXPECT() *MockDataProviderMockRecorder {
	return m.recorder
}

// GetDevData mocks base method
func (m *MockDataProvider) GetDevData(arg0 string) (*svc.DevData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDevData", arg0)
	ret0, _ := ret[0].(*svc.DevData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDevData indicates an expected call of GetDevData
func (mr *MockDataProviderMockRecorder) GetDevData(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDevData", reflect.TypeOf((*MockDataProvider)(nil).GetDevData), arg0)
}

// GetDevsData mocks base method
func (m *MockDataProvider) GetDevsData() ([]svc.DevData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDevsData")
	ret0, _ := ret[0].([]svc.DevData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDevsData indicates an expected call of GetDevsData
func (mr *MockDataProviderMockRecorder) GetDevsData() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDevsData", reflect.TypeOf((*MockDataProvider)(nil).GetDevsData))
}

// SaveDevData mocks base method
func (m *MockDataProvider) SaveDevData(arg0 *svc.DevData) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveDevData", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveDevData indicates an expected call of SaveDevData
func (mr *MockDataProviderMockRecorder) SaveDevData(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveDevData", reflect.TypeOf((*MockDataProvider)(nil).SaveDevData), arg0)
}
