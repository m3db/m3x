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

package sync

import (
	"fmt"
	"math/rand"
	"sync"
)

type pooledWorkerPool struct {
	sync.Mutex
	workChs               []chan workerInstruction
	numShards             int
	killWorkerProbability float64
	rand                  *rand.Rand
}

type workerInstruction struct {
	work      Work
	shouldDie bool
}

// NewPooledWorkerPool creates a new worker pool.
func NewPooledWorkerPool(size int, opts PooledWorkerPoolOptions) (PooledWorkerPool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("pooled worker pool size too small: %d", size)
	}

	numShards := opts.NumShards()
	if size < numShards {
		numShards = size
	}

	workChs := make([]chan workerInstruction, numShards)
	for i := range workChs {
		workChs[i] = make(chan workerInstruction, size/numShards)
	}

	return &pooledWorkerPool{
		workChs:               workChs,
		numShards:             numShards,
		killWorkerProbability: opts.KillWorkerProbability(),
		rand: rand.New(rand.NewSource(opts.RandSeed())),
	}, nil
}

func (p *pooledWorkerPool) Init() {
	for _, workCh := range p.workChs {
		for i := 0; i < cap(workCh); i++ {
			p.spawnWorker(workCh)
		}
	}
}

func (p *pooledWorkerPool) Go(work Work) {
	instruction := workerInstruction{work: work}
	randVal := p.getRandFloat64()
	killWorker := randVal < p.killWorkerProbability
	instruction.shouldDie = killWorker

	// Pick an index based on the random value. Example:
	// 		numShards = 10
	// 		randVal = 0.14
	// 		int(0.14 * float64(10)) = 1
	// 		1 % 10 == 1
	workChIdx := p.rand.Intn(p.numShards)
	workCh := p.workChs[workChIdx]
	workCh <- instruction
}

func (p *pooledWorkerPool) spawnWorker(workCh chan workerInstruction) {
	go func() {
		for instruction := range workCh {
			instruction.work()
			if instruction.shouldDie {
				p.spawnWorker(workCh)
				return
			}
		}
	}()
}

func (p *pooledWorkerPool) getRandFloat64() float64 {
	p.Lock()
	f := p.rand.Float64()
	p.Unlock()
	return f
}