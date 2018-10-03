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
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkerPoolPolicyConvertsToOptions(t *testing.T) {
	// Default policy
	wpp := WorkerPoolPolicy{}
	opts, size := wpp.ToOptions()
	assert.False(t, opts.GrowOnDemand())
	assert.NotZero(t, opts.NumShards())
	assert.NotZero(t, opts.KillWorkerProbability())
	assert.Equal(t, defaultWorkerPoolStaticSize, size)

	// Default policy with a growing pool
	wpp = WorkerPoolPolicy{GrowOnDemand: true}
	opts, size = wpp.ToOptions()
	assert.True(t, opts.GrowOnDemand())
	assert.NotZero(t, opts.NumShards())
	assert.NotZero(t, opts.KillWorkerProbability())
	assert.Equal(t, defaultWorkerPoolDynamicSize, size)

	// Full policy
	wpp = WorkerPoolPolicy{
		GrowOnDemand:          true,
		Size:                  100,
		NumShards:             200,
		KillWorkerProbability: 0.5,
	}
	opts, size = wpp.ToOptions()
	assert.True(t, opts.GrowOnDemand())
	assert.Equal(t, int64(200), opts.NumShards())
	assert.Equal(t, 0.5, opts.KillWorkerProbability())
	assert.Equal(t, 100, size)

}
