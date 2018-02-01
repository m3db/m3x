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

// NewIDsIterator returns a new IDsIterator over the given IDs.
func NewIDsIterator(ids ...ID) IDsIterator {
	return NewIDSliceIterator(ids)
}

// NewIDSliceIterator returns a new IDsIterator over a slice.
func NewIDSliceIterator(ids []ID) IDsIterator {
	iter := &idSliceIter{
		backingSlice: ids,
		currentIdx:   -1,
	}
	return iter
}

type idSliceIter struct {
	backingSlice []ID
	currentIdx   int
	currentID    ID
}

func (i *idSliceIter) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.backingSlice) {
		i.currentID = i.backingSlice[i.currentIdx]
		return true
	}
	i.currentID = nil
	return false
}

func (i *idSliceIter) Current() ID {
	return i.currentID
}

func (i *idSliceIter) Err() error {
	return nil
}

func (i *idSliceIter) Remaining() int {
	if r := len(i.backingSlice) - 1 - i.currentIdx; r >= 0 {
		return r
	}
	return 0
}

// NewStringIDsIterator returns a new IDsIterator over the given IDs.
func NewStringIDsIterator(ids ...string) IDsIterator {
	return NewStringIDsSliceIterator(ids)
}

// NewStringIDsSliceIterator returns a new IDsIterator over a slice of strings.
func NewStringIDsSliceIterator(ids []string) IDsIterator {
	iter := &stringSliceIter{
		backingSlice: ids,
		currentIdx:   -1,
	}
	return iter
}

type stringSliceIter struct {
	backingSlice []string
	currentIdx   int
	currentID    ID
}

func (i *stringSliceIter) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.backingSlice) {
		i.currentID = StringID(i.backingSlice[i.currentIdx])
		return true
	}
	i.currentID = nil
	return false
}

func (i *stringSliceIter) Current() ID {
	return i.currentID
}

func (i *stringSliceIter) Err() error {
	return nil
}

func (i *stringSliceIter) Remaining() int {
	if r := len(i.backingSlice) - 1 - i.currentIdx; r >= 0 {
		return r
	}
	return 0
}
