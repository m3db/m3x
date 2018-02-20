// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/m3db/m3x/pool (interfaces: CheckedBytesPool)

package pool

import (
	"github.com/m3db/m3x/checked"

	"github.com/golang/mock/gomock"
)

// Mock of CheckedBytesPool interface
type MockCheckedBytesPool struct {
	ctrl     *gomock.Controller
	recorder *_MockCheckedBytesPoolRecorder
}

// Recorder for MockCheckedBytesPool (not exported)
type _MockCheckedBytesPoolRecorder struct {
	mock *MockCheckedBytesPool
}

func NewMockCheckedBytesPool(ctrl *gomock.Controller) *MockCheckedBytesPool {
	mock := &MockCheckedBytesPool{ctrl: ctrl}
	mock.recorder = &_MockCheckedBytesPoolRecorder{mock}
	return mock
}

func (_m *MockCheckedBytesPool) EXPECT() *_MockCheckedBytesPoolRecorder {
	return _m.recorder
}

func (_m *MockCheckedBytesPool) Get(_param0 int) checked.Bytes {
	ret := _m.ctrl.Call(_m, "Get", _param0)
	ret0, _ := ret[0].(checked.Bytes)
	return ret0
}

func (_mr *_MockCheckedBytesPoolRecorder) Get(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Get", arg0)
}

func (_m *MockCheckedBytesPool) Init() {
	_m.ctrl.Call(_m, "Init")
}

func (_mr *_MockCheckedBytesPoolRecorder) Init() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Init")
}
