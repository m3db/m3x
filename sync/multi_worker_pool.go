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

// Package sync implements synchronization facililites such as worker pools.
package sync

// MultiWorkerPool provides a pool for WorkerPools. This can be useful if multiple
// independent processes need to share a set amount of pools, but with bounding conditions
// as to how many workers a single process can consume
type MultiWorkerPool interface {
	// Init initializes the pool.
	Init()

	// GetPool provides a WorkerPool
	GetPool() WorkerPool

	// PutPool returns a WorkerPool to the pool
	PutPool(pool WorkerPool)
}

type multiWorkerPool struct {
	poolCh   chan WorkerPool
	poolSize int
}

// NewMultiWorkerPool creates a new multi worker pool.
func NewMultiWorkerPool(size int, poolSize int) MultiWorkerPool {
	return &multiWorkerPool{
		poolCh:   make(chan WorkerPool, size),
		poolSize: poolSize,
	}
}

func (p *multiWorkerPool) Init() {
	for i := 0; i < cap(p.poolCh); i++ {
		innerPool := NewWorkerPool(p.poolSize)
		innerPool.Init()
		p.poolCh <- innerPool
	}
}

func (p *multiWorkerPool) GetPool() WorkerPool {
	return <-p.poolCh
}

func (p *multiWorkerPool) PutPool(pool WorkerPool) {
	p.poolCh <- pool
}
