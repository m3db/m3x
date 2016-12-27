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

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracebackReadAfterFree(t *testing.T) {
	SetTraceback(true)
	defer SetTraceback(false)

	elem := &struct {
		RefCount
		x int
	}{
		x: 42,
	}

	finalized := 0
	elem.SetFinalizer(FinalizerFn(func() {
		finalized++
	}))

	elem.IncRef()
	assert.Equal(t, 0, finalized)

	elem.DecRef()
	assert.Equal(t, 0, finalized)

	elem.Finalize()
	assert.Equal(t, 1, finalized)

	var err error
	SetPanicFn(func(e error) {
		err = e
	})
	defer ResetPanicFn()

	elem.IncReads()

	require.Error(t, err)

	str := err.Error()
	assert.True(t, strings.Contains(str, "read after free: reads=1, ref=0"))
	assert.True(t, strings.Contains(str, "IncReads, ref=0, unixnanos="))
	assert.True(t, strings.Contains(str, "checked.(*RefCount).IncReads"))
	assert.True(t, strings.Contains(str, "DecRef, ref=0, unixnanos="))
	assert.True(t, strings.Contains(str, "checked.(*RefCount).DecRef"))
	assert.True(t, strings.Contains(str, "IncRef, ref=1, unixnanos="))
	assert.True(t, strings.Contains(str, "checked.(*RefCount).IncRef"))
}
