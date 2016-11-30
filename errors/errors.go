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

package xerrors

import "bytes"

// FirstError will return the first non nil error
func FirstError(errs ...error) error {
	for i := range errs {
		if errs[i] != nil {
			return errs[i]
		}
	}
	return nil
}

type containedError struct {
	inner error
}

func (e containedError) Error() string {
	return e.inner.Error()
}

func (e containedError) InnerError() error {
	return e.inner
}

// ContainedError is an error with a contained error
type ContainedError interface {
	InnerError() error
}

// InnerError returns the packaged inner error if this is an error that contains another
func InnerError(err error) error {
	contained, ok := err.(ContainedError)
	if !ok {
		return nil
	}
	return contained.InnerError()
}

type renamedError struct {
	containedError
	renamed error
}

// NewRenamedError returns a new error that packages an inner error with a renamed error
func NewRenamedError(inner, renamed error) error {
	return renamedError{containedError{inner}, renamed}
}

func (e renamedError) Error() string {
	return e.renamed.Error()
}

func (e renamedError) InnerError() error {
	return e.inner
}

type invalidParamsError struct {
	containedError
}

// NewInvalidParamsError creates a new invalid params error
func NewInvalidParamsError(inner error) error {
	return invalidParamsError{containedError{inner}}
}

func (e invalidParamsError) Error() string {
	return e.inner.Error()
}

func (e invalidParamsError) InnerError() error {
	return e.inner
}

// IsInvalidParams returns true if this is an invalid params error
func IsInvalidParams(err error) bool {
	return GetInnerInvalidParamsError(err) != nil
}

// GetInnerInvalidParamsError returns an inner invalid params error
// if contained by this error, nil otherwise
func GetInnerInvalidParamsError(err error) error {
	for err != nil {
		if _, ok := err.(invalidParamsError); ok {
			return InnerError(err)
		}
		err = InnerError(err)
	}
	return nil
}

type retryableError struct {
	containedError
}

// NewRetryableError creates a new retryable error
func NewRetryableError(inner error) error {
	return retryableError{containedError{inner}}
}

func (e retryableError) Error() string {
	return e.inner.Error()
}

func (e retryableError) InnerError() error {
	return e.inner
}

// IsRetryableError returns true if this is a retryable error
func IsRetryableError(err error) bool {
	return GetInnerRetryableError(err) != nil
}

// GetInnerRetryableError returns an inner retryable error
// if contained by this error, nil otherwise
func GetInnerRetryableError(err error) error {
	for err != nil {
		if _, ok := err.(retryableError); ok {
			return InnerError(err)
		}
		err = InnerError(err)
	}
	return nil
}

type nonRetryableError struct {
	containedError
}

// NewNonRetryableError creates a new non-retryable error
func NewNonRetryableError(inner error) error {
	return nonRetryableError{containedError{inner}}
}

func (e nonRetryableError) Error() string {
	return e.inner.Error()
}

func (e nonRetryableError) InnerError() error {
	return e.inner
}

// IsNonRetryableError returns true if this is a non-retryable error
func IsNonRetryableError(err error) bool {
	return GetInnerNonRetryableError(err) != nil
}

// GetInnerNonRetryableError returns an inner non-retryable error
// if contained by this error, nil otherwise
func GetInnerNonRetryableError(err error) error {
	for err != nil {
		if _, ok := err.(nonRetryableError); ok {
			return InnerError(err)
		}
		err = InnerError(err)
	}
	return nil
}

// MultiError is an immutable error that packages a list of errors.
// TODO(xichen): we may want to limit the number of errors included.
type MultiError struct {
	err    error // optimization for single error case
	errors []error
}

// NewMultiError creates a new MultiError object.
func NewMultiError() MultiError {
	return MultiError{}
}

// Empty returns true if the MultiError has no errors
func (e MultiError) Empty() bool {
	return e.err == nil
}

func (e MultiError) Error() string {
	if e.err == nil {
		return ""
	}
	if len(e.errors) == 0 {
		return e.err.Error()
	}
	var b bytes.Buffer
	for i := range e.errors {
		b.WriteString(e.errors[i].Error())
		b.WriteString("\n")
	}
	b.WriteString(e.err.Error())
	return b.String()
}

// Add adds an error returns a new MultiError object.
func (e MultiError) Add(err error) MultiError {
	if err == nil {
		return e
	}
	me := e
	if me.err == nil {
		me.err = err
		return me
	}
	me.errors = append(me.errors, me.err)
	me.err = err
	return me
}

// FinalError returns the final error if any.
func (e MultiError) FinalError() error {
	if e.err == nil {
		return nil
	}
	return e
}

func (e MultiError) NumErrors() int {
	if e.err == nil {
		return 0
	}
	return len(e.errors) + 1
}
