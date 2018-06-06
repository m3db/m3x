// Copyright (c) 2017 Uber Technologies, Inc.
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

package sync

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

const multiTestWorkerPoolSize = 2

func TestGetRoutine(t *testing.T) {
	var count uint32

	multi := NewMultiWorkerPool(multiTestWorkerPoolSize, testWorkerPoolSize)
	multi.Init()

	var multiWg sync.WaitGroup
	for pool := 0; pool < multiTestWorkerPoolSize*2; pool++ {
		go func() {
			multiWg.Add(1)
			p := multi.GetPool()
			var wg sync.WaitGroup
			for i := 0; i < testWorkerPoolSize*2; i++ {
				wg.Add(1)
				p.Go(func() {
					atomic.AddUint32(&count, 1)
					wg.Done()
				})
			}
			wg.Wait()
			multi.PutPool(p)
			multiWg.Done()
		}()
	}
	multiWg.Wait()

	require.Equal(t, uint32(testWorkerPoolSize*2*multiTestWorkerPoolSize*2), count)
}
