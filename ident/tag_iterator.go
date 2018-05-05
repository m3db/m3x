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

package ident

import "errors"

var (
	errInvalidNumberInputsToIteratorMatcher = errors.New("inputs must be specified in name-value pairs (i.e. divisible by 2)")
)

// NewTagIterator returns a new TagSliceIterator over the given Tags.
func NewTagIterator(tags ...Tag) TagSliceIterator {
	return NewTagSliceIterator(tags)
}

// MustNewTagStringsIterator returns a TagIterator over a slice of strings,
// panic'ing if it encounters an error.
func MustNewTagStringsIterator(inputs ...string) TagIterator {
	iter, err := NewTagStringsIterator(inputs...)
	if err != nil {
		panic(err.Error())
	}
	return iter
}

// NewTagStringsIterator returns a TagIterator over a slice of strings.
func NewTagStringsIterator(inputs ...string) (TagIterator, error) {
	if len(inputs)%2 != 0 {
		return nil, errInvalidNumberInputsToIteratorMatcher
	}
	tags := make(Tags, 0, len(inputs)/2)
	for i := 0; i < len(inputs); i += 2 {
		tags = append(tags, StringTag(inputs[i], inputs[i+1]))
	}
	return NewTagSliceIterator(tags), nil
}

// NewTagSliceIterator returns a TagSliceIterator over a slice.
func NewTagSliceIterator(tags Tags) TagSliceIterator {
	iter := &tagSliceIter{}
	iter.Reset(tags)
	return iter
}

type tagSliceIter struct {
	backingSlice Tags
	currentIdx   int
	currentTag   Tag
}

func (i *tagSliceIter) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.backingSlice) {
		i.currentTag = i.backingSlice[i.currentIdx]
		return true
	}
	i.currentTag = Tag{}
	return false
}

func (i *tagSliceIter) Current() Tag {
	return i.currentTag
}

func (i *tagSliceIter) Err() error {
	return nil
}

func (i *tagSliceIter) Close() {
	i.backingSlice = nil
	i.currentIdx = 0
	i.currentTag = Tag{}
}

func (i *tagSliceIter) Remaining() int {
	if r := len(i.backingSlice) - 1 - i.currentIdx; r >= 0 {
		return r
	}
	return 0
}

func (i *tagSliceIter) Duplicate() TagIterator {
	return &tagSliceIter{
		backingSlice: i.backingSlice,
		currentIdx:   i.currentIdx,
		currentTag:   i.currentTag,
	}
}

func (i *tagSliceIter) Reset(tags Tags) {
	i.backingSlice = tags
	i.currentIdx = -1
	if len(tags) != 0 {
		// NB(r): To support calls to Current() before the first Next() is called.
		i.currentTag = tags[0]
	} else {
		i.currentTag = Tag{}
	}
}

// EmptyTagIterator returns an iterator over no tags.
var EmptyTagIterator TagIterator = &emptyTagIterator{}

type emptyTagIterator struct{}

func (e *emptyTagIterator) Next() bool             { return false }
func (e *emptyTagIterator) Current() Tag           { return Tag{} }
func (e *emptyTagIterator) Err() error             { return nil }
func (e *emptyTagIterator) Close()                 {}
func (e *emptyTagIterator) Remaining() int         { return 0 }
func (e *emptyTagIterator) Duplicate() TagIterator { return e }
