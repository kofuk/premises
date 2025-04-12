// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/kofuk/premises/runner/internal/commands/mclauncher/core (interfaces: SettingsRepository)
//
// Generated by this command:
//
//	mockgen -destination settings_mock.go -package core . SettingsRepository
//

// Package core is a generated GoMock package.
package core

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockSettingsRepository is a mock of SettingsRepository interface.
type MockSettingsRepository struct {
	ctrl     *gomock.Controller
	recorder *MockSettingsRepositoryMockRecorder
	isgomock struct{}
}

// MockSettingsRepositoryMockRecorder is the mock recorder for MockSettingsRepository.
type MockSettingsRepositoryMockRecorder struct {
	mock *MockSettingsRepository
}

// NewMockSettingsRepository creates a new mock instance.
func NewMockSettingsRepository(ctrl *gomock.Controller) *MockSettingsRepository {
	mock := &MockSettingsRepository{ctrl: ctrl}
	mock.recorder = &MockSettingsRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSettingsRepository) EXPECT() *MockSettingsRepositoryMockRecorder {
	return m.recorder
}

// AutoVersionEnabled mocks base method.
func (m *MockSettingsRepository) AutoVersionEnabled() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AutoVersionEnabled")
	ret0, _ := ret[0].(bool)
	return ret0
}

// AutoVersionEnabled indicates an expected call of AutoVersionEnabled.
func (mr *MockSettingsRepositoryMockRecorder) AutoVersionEnabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AutoVersionEnabled", reflect.TypeOf((*MockSettingsRepository)(nil).AutoVersionEnabled))
}

// GetAllowedMemSize mocks base method.
func (m *MockSettingsRepository) GetAllowedMemSize() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllowedMemSize")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetAllowedMemSize indicates an expected call of GetAllowedMemSize.
func (mr *MockSettingsRepositoryMockRecorder) GetAllowedMemSize() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllowedMemSize", reflect.TypeOf((*MockSettingsRepository)(nil).GetAllowedMemSize))
}

// GetMinecraftVersion mocks base method.
func (m *MockSettingsRepository) GetMinecraftVersion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMinecraftVersion")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetMinecraftVersion indicates an expected call of GetMinecraftVersion.
func (mr *MockSettingsRepositoryMockRecorder) GetMinecraftVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMinecraftVersion", reflect.TypeOf((*MockSettingsRepository)(nil).GetMinecraftVersion))
}

// GetServerPath mocks base method.
func (m *MockSettingsRepository) GetServerPath() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServerPath")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetServerPath indicates an expected call of GetServerPath.
func (mr *MockSettingsRepositoryMockRecorder) GetServerPath() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServerPath", reflect.TypeOf((*MockSettingsRepository)(nil).GetServerPath))
}

// GetWorldName mocks base method.
func (m *MockSettingsRepository) GetWorldName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorldName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetWorldName indicates an expected call of GetWorldName.
func (mr *MockSettingsRepositoryMockRecorder) GetWorldName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorldName", reflect.TypeOf((*MockSettingsRepository)(nil).GetWorldName))
}

// GetWorldResourceID mocks base method.
func (m *MockSettingsRepository) GetWorldResourceID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorldResourceID")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetWorldResourceID indicates an expected call of GetWorldResourceID.
func (mr *MockSettingsRepositoryMockRecorder) GetWorldResourceID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorldResourceID", reflect.TypeOf((*MockSettingsRepository)(nil).GetWorldResourceID))
}

// IsNewWorld mocks base method.
func (m *MockSettingsRepository) IsNewWorld() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsNewWorld")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsNewWorld indicates an expected call of IsNewWorld.
func (mr *MockSettingsRepositoryMockRecorder) IsNewWorld() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsNewWorld", reflect.TypeOf((*MockSettingsRepository)(nil).IsNewWorld))
}

// SetMinecraftVersion mocks base method.
func (m *MockSettingsRepository) SetMinecraftVersion(version string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetMinecraftVersion", version)
}

// SetMinecraftVersion indicates an expected call of SetMinecraftVersion.
func (mr *MockSettingsRepositoryMockRecorder) SetMinecraftVersion(version any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetMinecraftVersion", reflect.TypeOf((*MockSettingsRepository)(nil).SetMinecraftVersion), version)
}

// SetServerPath mocks base method.
func (m *MockSettingsRepository) SetServerPath(path string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetServerPath", path)
}

// SetServerPath indicates an expected call of SetServerPath.
func (mr *MockSettingsRepositoryMockRecorder) SetServerPath(path any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetServerPath", reflect.TypeOf((*MockSettingsRepository)(nil).SetServerPath), path)
}

// SetWorldResourceID mocks base method.
func (m *MockSettingsRepository) SetWorldResourceID(resourceID string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetWorldResourceID", resourceID)
}

// SetWorldResourceID indicates an expected call of SetWorldResourceID.
func (mr *MockSettingsRepositoryMockRecorder) SetWorldResourceID(resourceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetWorldResourceID", reflect.TypeOf((*MockSettingsRepository)(nil).SetWorldResourceID), resourceID)
}
