// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/stackrox/rox/central/scrape/factory (interfaces: ScrapeFactory)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	framework "github.com/stackrox/rox/central/compliance/framework"
	compliance "github.com/stackrox/rox/generated/internalapi/compliance"
	concurrency "github.com/stackrox/rox/pkg/concurrency"
	reflect "reflect"
)

// MockScrapeFactory is a mock of ScrapeFactory interface
type MockScrapeFactory struct {
	ctrl     *gomock.Controller
	recorder *MockScrapeFactoryMockRecorder
}

// MockScrapeFactoryMockRecorder is the mock recorder for MockScrapeFactory
type MockScrapeFactoryMockRecorder struct {
	mock *MockScrapeFactory
}

// NewMockScrapeFactory creates a new mock instance
func NewMockScrapeFactory(ctrl *gomock.Controller) *MockScrapeFactory {
	mock := &MockScrapeFactory{ctrl: ctrl}
	mock.recorder = &MockScrapeFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockScrapeFactory) EXPECT() *MockScrapeFactoryMockRecorder {
	return m.recorder
}

// RunScrape mocks base method
func (m *MockScrapeFactory) RunScrape(arg0 framework.ComplianceDomain, arg1 concurrency.Waitable) (map[string]*compliance.ComplianceReturn, error) {
	ret := m.ctrl.Call(m, "RunScrape", arg0, arg1)
	ret0, _ := ret[0].(map[string]*compliance.ComplianceReturn)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RunScrape indicates an expected call of RunScrape
func (mr *MockScrapeFactoryMockRecorder) RunScrape(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunScrape", reflect.TypeOf((*MockScrapeFactory)(nil).RunScrape), arg0, arg1)
}
