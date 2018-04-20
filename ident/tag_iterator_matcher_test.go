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

package ident_test

import (
	"fmt"
	"testing"

	"github.com/m3db/m3x/ident"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTagIteratorMatcher(t *testing.T) {
	iter := ident.NewTagIterator(
		ident.StringTag("hello", "there"),
		ident.StringTag("foo", "bar"))
	matcher := ident.NewTagIteratorMatcher("hello", "there", "foo", "bar")
	assert.True(t, matcher.Matches(iter))
}

func TestTagIteratorMatcherNotMatching(t *testing.T) {
	matcher := ident.NewTagIteratorMatcher()
	assert.False(t, matcher.Matches(1))

	iter := ident.NewTagIterator(ident.StringTag("hello", "there"))
	assert.False(t, matcher.Matches(iter))
}

func TestTagIteratorMatcherInvalid(t *testing.T) {
	assert.Panics(t, func() {
		ident.NewTagIteratorMatcher("hello")
	})
	assert.Panics(t, func() {
		ident.NewTagIteratorMatcher("hello", "a", "b")
	})
}

func TestTagIteratorMatcherErrCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	iter := ident.NewMockTagIterator(ctrl)
	gomock.InOrder(
		iter.EXPECT().Duplicate().Return(iter),
		iter.EXPECT().Next().Return(true),
		iter.EXPECT().Current().Return(ident.StringTag("a", "b")),
		iter.EXPECT().Next().Return(false),
		iter.EXPECT().Err().Return(fmt.Errorf("random error")),
	)
	matcher := ident.NewTagIteratorMatcher("a", "b")
	assert.False(t, matcher.Matches(iter))
}
