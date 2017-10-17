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

// Package sync implements synchronization facililites such as worker pools.
package sync

import (
	"sync/atomic"
	"time"

	"github.com/m3db/m3x/instrument"

	"github.com/uber-go/tally"
)

type instrumentedWorkerPool struct {
	pool    WorkerPool
	closeCh chan struct{}
	size    int
	metrics workerPoolMetrics
	opts    instrument.Options
}

// NewInstrumentedWorkerPool creates a new instrumented worker pool.
func NewInstrumentedWorkerPool(size int, opts instrument.Options) WorkerPool {
	return &instrumentedWorkerPool{
		pool:    NewWorkerPool(size),
		closeCh: make(chan struct{}, 1),
		size:    size,
		metrics: newWorkerPoolMetrics(size, opts),
		opts:    opts,
	}
}

func (p *instrumentedWorkerPool) Init() {
	p.pool.Init()
	p.metrics.incIdle(int64(p.size))
	go p.metricLoop()
}

func (p *instrumentedWorkerPool) Close() error {
	if err := p.pool.Close(); err != nil {
		return err
	}
	close(p.closeCh)
	return nil
}

func (p *instrumentedWorkerPool) metricLoop() {
	ticker := time.NewTicker(p.opts.ReportInterval())
	defer ticker.Stop()
	for {
		select {
		case <-p.closeCh:
			return
		case <-ticker.C:
			p.metrics.update()
		}
	}
}

func (p *instrumentedWorkerPool) Go(work Work) {
	p.metrics.incWaiting(1)
	p.pool.Go(p.instrumentedWork(work))
}

func (p *instrumentedWorkerPool) GoIfAvailable(work Work) bool {
	p.metrics.incWaiting(1)
	if p.pool.GoIfAvailable(p.instrumentedWork(work)) {
		return true
	}
	p.metrics.unavailable.Inc(1)
	p.metrics.incWaiting(-1)
	return false
}

func (p *instrumentedWorkerPool) GoWithTimeout(work Work, timeout time.Duration) bool {
	p.metrics.incWaiting(1)
	if p.pool.GoWithTimeout(p.instrumentedWork(work), timeout) {
		return true
	}
	p.metrics.timeouts.Inc(1)
	p.metrics.incWaiting(-1)
	return false
}

func (p *instrumentedWorkerPool) instrumentedWork(w Work) Work {
	return func() {
		p.metrics.incWaiting(-1)
		p.metrics.incIdle(-1)
		p.metrics.incBusy(1)
		w()
		p.metrics.incBusy(-1)
		p.metrics.incIdle(1)
		p.metrics.run.Inc(1)
	}
}

type workerPoolMetrics struct {
	timeouts    tally.Counter
	unavailable tally.Counter
	run         tally.Counter
	idle        tally.Gauge
	busy        tally.Gauge
	waiting     tally.Gauge
	total       tally.Gauge

	numIdle    int64
	numBusy    int64
	numWaiting int64
	numTotal   int64
}

func newWorkerPoolMetrics(size int, opts instrument.Options) workerPoolMetrics {
	scope := opts.MetricsScope().SubScope("workers")
	return workerPoolMetrics{
		timeouts:    scope.Counter("timeouts"),
		unavailable: scope.Counter("unavailable"),
		run:         scope.Counter("run"),
		idle:        scope.Gauge("idle"),
		busy:        scope.Gauge("busy"),
		waiting:     scope.Gauge("waiting"),
		total:       scope.Gauge("total"),
		numTotal:    int64(size),
	}
}

func (m *workerPoolMetrics) update() {
	m.idle.Update(float64(atomic.LoadInt64(&m.numIdle)))
	m.busy.Update(float64(atomic.LoadInt64(&m.numBusy)))
	m.waiting.Update(float64(atomic.LoadInt64(&m.numWaiting)))
	m.total.Update(float64(atomic.LoadInt64(&m.numTotal)))
}

func (m *workerPoolMetrics) incIdle(n int64) {
	atomic.AddInt64(&m.numIdle, n)
}

func (m *workerPoolMetrics) incBusy(n int64) {
	atomic.AddInt64(&m.numBusy, n)
}

func (m *workerPoolMetrics) incWaiting(n int64) {
	atomic.AddInt64(&m.numWaiting, n)
}
