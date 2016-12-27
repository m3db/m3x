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

package checked

// Finalizer finalizes a checked resource.
type Finalizer interface {
	Finalize()
}

// FinalizerFn is a function literal that is a finalizer.
type FinalizerFn func()

// Finalize will call the function literal as a finalizer.
func (fn FinalizerFn) Finalize() {
	fn()
}

// Ref is an entity that checks reference counts.
type Ref interface {
	// IncRef increments the reference count to this entity.
	IncRef()

	// DecRef decrements the reference count to this entity.
	DecRef()

	// XfrRef transfers the reference to this entity from object to another.
	XfrRef()

	// NumRef returns the reference count to this entity.
	NumRef() int

	// Finalizer returns the finalizer if any or nil otherwise.
	Finalizer() Finalizer

	// SetFinalizer sets the finalizer.
	SetFinalizer(f Finalizer)
}

// Read is an entity that checks reads.
type Read interface {
	Ref

	// IncReads increments the reads count to this entity.
	IncReads()

	// DecReads decrements the reads count to this entity.
	DecReads()

	// NumReads returns the active reads count to this entity.
	NumReads() int
}

// ReadWrite is an entity that checks reads and writes.
type ReadWrite interface {
	Read

	// IncWrites increments the writes count to this entity.
	IncWrites()

	// DecWrites decrements the writes count to this entity.
	DecWrites()

	// NumWrites returns the active writes count to this entity.
	NumWrites() int
}

// BytesFinalizer finalizes a checked byte slice.
type BytesFinalizer interface {
	FinalizeBytes(b Bytes)
}

// BytesOptions is a bytes option
type BytesOptions interface {
	// Finalizer is a bytes finalizer to call when finalized.
	Finalizer() BytesFinalizer

	// SetFinalizer sets a bytes finalizer to call when finalized.
	SetFinalizer(value BytesFinalizer) BytesOptions
}
