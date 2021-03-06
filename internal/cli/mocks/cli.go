// Code generated by MockGen. DO NOT EDIT.
// Source: ifaces/ifaces.go

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	bindown "github.com/willabides/bindown/v3"
	ifaces "github.com/willabides/bindown/v3/internal/cli/ifaces"
	reflect "reflect"
)

// MockConfigFile is a mock of ConfigFile interface
type MockConfigFile struct {
	ctrl     *gomock.Controller
	recorder *MockConfigFileMockRecorder
}

// MockConfigFileMockRecorder is the mock recorder for MockConfigFile
type MockConfigFileMockRecorder struct {
	mock *MockConfigFile
}

// NewMockConfigFile creates a new mock instance
func NewMockConfigFile(ctrl *gomock.Controller) *MockConfigFile {
	mock := &MockConfigFile{ctrl: ctrl}
	mock.recorder = &MockConfigFileMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockConfigFile) EXPECT() *MockConfigFileMockRecorder {
	return m.recorder
}

// Write mocks base method
func (m *MockConfigFile) Write(outputJSON bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", outputJSON)
	ret0, _ := ret[0].(error)
	return ret0
}

// Write indicates an expected call of Write
func (mr *MockConfigFileMockRecorder) Write(outputJSON interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockConfigFile)(nil).Write), outputJSON)
}

// AddChecksums mocks base method
func (m *MockConfigFile) AddChecksums(dependencies []string, systems []bindown.SystemInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddChecksums", dependencies, systems)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddChecksums indicates an expected call of AddChecksums
func (mr *MockConfigFileMockRecorder) AddChecksums(dependencies, systems interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddChecksums", reflect.TypeOf((*MockConfigFile)(nil).AddChecksums), dependencies, systems)
}

// Validate mocks base method
func (m *MockConfigFile) Validate(dependencies []string, systems []bindown.SystemInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Validate", dependencies, systems)
	ret0, _ := ret[0].(error)
	return ret0
}

// Validate indicates an expected call of Validate
func (mr *MockConfigFileMockRecorder) Validate(dependencies, systems interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Validate", reflect.TypeOf((*MockConfigFile)(nil).Validate), dependencies, systems)
}

// InstallDependency mocks base method
func (m *MockConfigFile) InstallDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigInstallDependencyOpts) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstallDependency", dependencyName, sysInfo, opts)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InstallDependency indicates an expected call of InstallDependency
func (mr *MockConfigFileMockRecorder) InstallDependency(dependencyName, sysInfo, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstallDependency", reflect.TypeOf((*MockConfigFile)(nil).InstallDependency), dependencyName, sysInfo, opts)
}

// DownloadDependency mocks base method
func (m *MockConfigFile) DownloadDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigDownloadDependencyOpts) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DownloadDependency", dependencyName, sysInfo, opts)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DownloadDependency indicates an expected call of DownloadDependency
func (mr *MockConfigFileMockRecorder) DownloadDependency(dependencyName, sysInfo, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DownloadDependency", reflect.TypeOf((*MockConfigFile)(nil).DownloadDependency), dependencyName, sysInfo, opts)
}

// ExtractDependency mocks base method
func (m *MockConfigFile) ExtractDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigExtractDependencyOpts) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExtractDependency", dependencyName, sysInfo, opts)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExtractDependency indicates an expected call of ExtractDependency
func (mr *MockConfigFileMockRecorder) ExtractDependency(dependencyName, sysInfo, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExtractDependency", reflect.TypeOf((*MockConfigFile)(nil).ExtractDependency), dependencyName, sysInfo, opts)
}

// AddDependencyFromTemplate mocks base method
func (m *MockConfigFile) AddDependencyFromTemplate(templateName string, opts *bindown.AddDependencyFromTemplateOpts) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddDependencyFromTemplate", templateName, opts)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddDependencyFromTemplate indicates an expected call of AddDependencyFromTemplate
func (mr *MockConfigFileMockRecorder) AddDependencyFromTemplate(templateName, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddDependencyFromTemplate", reflect.TypeOf((*MockConfigFile)(nil).AddDependencyFromTemplate), templateName, opts)
}

// MissingDependencyVars mocks base method
func (m *MockConfigFile) MissingDependencyVars(depName string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MissingDependencyVars", depName)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MissingDependencyVars indicates an expected call of MissingDependencyVars
func (mr *MockConfigFileMockRecorder) MissingDependencyVars(depName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MissingDependencyVars", reflect.TypeOf((*MockConfigFile)(nil).MissingDependencyVars), depName)
}

// SetDependencyVars mocks base method
func (m *MockConfigFile) SetDependencyVars(depName string, vars map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetDependencyVars", depName, vars)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetDependencyVars indicates an expected call of SetDependencyVars
func (mr *MockConfigFileMockRecorder) SetDependencyVars(depName, vars interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDependencyVars", reflect.TypeOf((*MockConfigFile)(nil).SetDependencyVars), depName, vars)
}

// UnsetDependencyVars mocks base method
func (m *MockConfigFile) UnsetDependencyVars(depName string, vars []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnsetDependencyVars", depName, vars)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnsetDependencyVars indicates an expected call of UnsetDependencyVars
func (mr *MockConfigFileMockRecorder) UnsetDependencyVars(depName, vars interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnsetDependencyVars", reflect.TypeOf((*MockConfigFile)(nil).UnsetDependencyVars), depName, vars)
}

// SetTemplateVars mocks base method
func (m *MockConfigFile) SetTemplateVars(tmplName string, vars map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetTemplateVars", tmplName, vars)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetTemplateVars indicates an expected call of SetTemplateVars
func (mr *MockConfigFileMockRecorder) SetTemplateVars(tmplName, vars interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTemplateVars", reflect.TypeOf((*MockConfigFile)(nil).SetTemplateVars), tmplName, vars)
}

// UnsetTemplateVars mocks base method
func (m *MockConfigFile) UnsetTemplateVars(tmplName string, vars []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnsetTemplateVars", tmplName, vars)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnsetTemplateVars indicates an expected call of UnsetTemplateVars
func (mr *MockConfigFileMockRecorder) UnsetTemplateVars(tmplName, vars interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnsetTemplateVars", reflect.TypeOf((*MockConfigFile)(nil).UnsetTemplateVars), tmplName, vars)
}

// MockConfigLoader is a mock of ConfigLoader interface
type MockConfigLoader struct {
	ctrl     *gomock.Controller
	recorder *MockConfigLoaderMockRecorder
}

// MockConfigLoaderMockRecorder is the mock recorder for MockConfigLoader
type MockConfigLoaderMockRecorder struct {
	mock *MockConfigLoader
}

// NewMockConfigLoader creates a new mock instance
func NewMockConfigLoader(ctrl *gomock.Controller) *MockConfigLoader {
	mock := &MockConfigLoader{ctrl: ctrl}
	mock.recorder = &MockConfigLoaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockConfigLoader) EXPECT() *MockConfigLoaderMockRecorder {
	return m.recorder
}

// Load mocks base method
func (m *MockConfigLoader) Load(filename string, noDefaultDirs bool) (ifaces.ConfigFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Load", filename, noDefaultDirs)
	ret0, _ := ret[0].(ifaces.ConfigFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Load indicates an expected call of Load
func (mr *MockConfigLoaderMockRecorder) Load(filename, noDefaultDirs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Load", reflect.TypeOf((*MockConfigLoader)(nil).Load), filename, noDefaultDirs)
}
