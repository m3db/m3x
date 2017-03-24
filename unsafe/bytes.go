// Copyright (c) 2016 Uber Technologies, Inc.
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

package xunsafe

import (
	"reflect"
	"unsafe"
)

// ToString converts a byte slice to a string with zero heap memory allocations.
// The returned string shares the same memory space as the byte slice passed in.
// It is the caller's responsibility to make sure the byte slice passed in is
// not modified in any way until end of life.
func ToString(b ImmutableBytes) string {
	// NB(xichen): we need to declare a real string so internally the compiler knows
	// to use an unsafe pointer to keep track of the underlying memory so that once
	// the string's internal pointer is updated with the byte slice's pointer, the
	// compiler won't prematurely GC the memory when the byte slice is out of scope.
	var str string
	if len(b) == 0 {
		return str
	}

	// NB(xichen): this makes sure that even if GC relocates the byte slice's underlying
	// memory after this assignment, the corresponding unsafe.Pointer in the internal
	// string struct will be updated accordingly to reflect the memory relocation.
	strHeader := (*reflect.StringHeader)(unsafe.Pointer(&str))
	strHeader.Data = (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data

	// NB(xichen): it is important that we access b after assigning the data pointer
	// of the slice header to the data pointer of the string header to make sure the
	// bye slice doesn't get GC'ed before the assignment happens.
	l := len(b)
	strHeader.Len = l

	// NB(xichen): it is not an issue if the byte slice's length and capacity disagree
	// because GC does not use the header information to determine how much memory
	// has been allocated. Instead, each address is mapped to a span internally by
	// truncating the address to the page boundary and reverse looking up the span
	// it belongs to, and each span has a fixed object size.
	return str
}
