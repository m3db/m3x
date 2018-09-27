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

package pool

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	gounsafe "unsafe"
)

// NB: heavily inspired by https://golang.org/src/sync/pool.go
type shardedObjectPool struct {
	local []*poolLocal // local fixed-size per-P pool

	localPoolSize       int
	refillLowWatermark  int
	refillHighWatermark int

	alloc       Allocator
	initialized int32
	metrics     objectPoolMetrics
	opts        ObjectPoolOptions
}

// Local per-P Pool appendix.
type poolLocalInternal struct {
	sync.Mutex               // Protects shared.
	shared     []interface{} // Can be used by any P.

	private interface{} // Can be used only by the respective P.

	filling int32
}

type poolLocal struct {
	poolLocalInternal

	// Prevents false sharing on widespread platforms with
	// 128 mod (cache line size) = 0 .
	pad [128 - gounsafe.Sizeof(poolLocalInternal{})%128]byte
}

func NewShardedObjectPool(opts ObjectPoolOptions) ObjectPool {
	if opts == nil {
		opts = NewObjectPoolOptions()
	}

	return &shardedObjectPool{
		opts:    opts,
		metrics: newObjectPoolMetrics(opts.InstrumentOptions().MetricsScope()),
	}
}

func (p *shardedObjectPool) Init(alloc Allocator) {
	if !atomic.CompareAndSwapInt32(&p.initialized, 0, 1) {
		fn := p.opts.OnPoolAccessErrorFn()
		fn(errPoolAlreadyInitialized)
		return
	}

	p.alloc = alloc

	numProcs := runtime.GOMAXPROCS(0)
	localPoolSize := int(math.Ceil(float64(p.opts.Size()) / float64(numProcs)))
	if localPoolSize <= 0 {
		fn := p.opts.OnPoolAccessErrorFn()
		fn(fmt.Errorf(
			"unable to compute localPoolSize [%d], GOMAXPROCS: %d, Size: %d",
			localPoolSize, numProcs, p.opts.Size()))
		return
	}
	lowWatermark := int(math.Ceil(p.opts.RefillLowWatermark() * float64(localPoolSize)))
	highWatermark := int(math.Ceil(p.opts.RefillHighWatermark() * float64(localPoolSize)))
	p.localPoolSize, p.refillLowWatermark, p.refillHighWatermark = localPoolSize, lowWatermark, highWatermark

	local := make([]*poolLocal, numProcs)
	for i := 0; i < len(local); i++ {
		local[i] = &poolLocal{}
		local[i].private = p.alloc()
		shared := make([]interface{}, 0, 2*localPoolSize)
		for j := 0; j < localPoolSize; j++ {
			shared = append(shared, p.alloc())
		}
		local[i].shared = shared
	}
	p.local = local

	p.setGauges()
}

// Put adds x to the pool.
func (p *shardedObjectPool) Put(x interface{}) {
	if x == nil {
		return
	}
	l := p.pin()
	if l.private == nil {
		l.private = x
		x = nil
	}
	ProcUnpin()
	if x != nil {
		l.Lock()
		l.shared = append(l.shared, x)
		l.Unlock()
	}
}

// Get selects an arbitrary item from the Pool, removes it from the
// Pool, and returns it to the caller.
// Get may choose to ignore the pool and treat it as empty.
// Callers should not assume any relation between values passed to Put and
// the values returned by Get.
//
// If Get would otherwise return nil and p.New is non-nil, Get returns
// the result of calling p.New.
func (p *shardedObjectPool) Get() interface{} {
	l := p.pin()
	x := l.private
	l.private = nil
	ProcUnpin()
	if x == nil {
		l.Lock()
		last := len(l.shared) - 1
		if last >= 0 {
			x = l.shared[last]
			l.shared = l.shared[:last]
		}
		l.Unlock()
		if x == nil {
			x = p.getSlow()
		}
	}
	if x == nil {
		x = p.alloc()
	}
	return x
}

func (p *shardedObjectPool) getSlow() (x interface{}) {
	local := p.local
	// Try to steal one element from other procs.
	pid := ProcPin()
	ProcUnpin()
	for i := 0; i < len(local); i++ {
		l := indexLocal(local, pid+i+1)
		l.Lock()
		last := len(l.shared) - 1
		if last >= 0 {
			x = l.shared[last]
			l.shared = l.shared[:last]
			l.Unlock()
			break
		}
		l.Unlock()
	}
	return x
}

// pin pins the current goroutine to P, disables preemption and returns poolLocal pool for the P.
// Caller must call ProcUnpin() when done with the pool.
func (p *shardedObjectPool) pin() *poolLocal {
	pid := ProcPin()
	return indexLocal(p.local, pid)
}

func indexLocal(l []*poolLocal, i int) *poolLocal {
	idx := i % len(l)
	return l[idx]
}

func (p *shardedObjectPool) setGauges() {
	/*
		p.metrics.free.Update(float64(len(p.values)))
		p.metrics.total.Update(float64(p.size))
	*/
}
