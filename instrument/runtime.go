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

package instrument

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/uber-go/tally"
)

// StartReportingRuntimeMetrics starts reporting runtime metrics.
func StartReportingRuntimeMetrics(
	scope tally.Scope,
	reportInterval time.Duration,
) *RuntimeMetricsReporter {
	runtimeMetricsReporter := NewRuntimeMetricsReporter(scope, reportInterval)
	runtimeMetricsReporter.Start()
	return runtimeMetricsReporter
}

type runtimeMetrics struct {
	NumGoRoutines   tally.Gauge
	GoMaxProcs      tally.Gauge
	MemoryAllocated tally.Gauge
	MemoryHeap      tally.Gauge
	MemoryHeapIdle  tally.Gauge
	MemoryHeapInuse tally.Gauge
	MemoryStack     tally.Gauge
	NumGC           tally.Counter
	GcPauseMs       tally.Timer
	lastNumGC       uint32
}

// RuntimeMetricsReporter A struct containing the state of the RuntimeMetricsReporter.
type RuntimeMetricsReporter struct {
	scope          tally.Scope
	reportInterval time.Duration
	metrics        runtimeMetrics
	started        bool
	quit           chan struct{}
}

// NewRuntimeMetricsReporter Creates a new RuntimeMetricsReporter.
func NewRuntimeMetricsReporter(
	scope tally.Scope,
	reportInterval time.Duration,
) *RuntimeMetricsReporter {
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)

	memoryScope := scope.SubScope("memory")
	return &RuntimeMetricsReporter{
		scope:          scope,
		reportInterval: reportInterval,
		metrics: runtimeMetrics{
			NumGoRoutines:   scope.Gauge("num-goroutines"),
			GoMaxProcs:      scope.Gauge("gomaxprocs"),
			MemoryAllocated: memoryScope.Gauge("allocated"),
			MemoryHeap:      memoryScope.Gauge("heap"),
			MemoryHeapIdle:  memoryScope.Gauge("heapidle"),
			MemoryHeapInuse: memoryScope.Gauge("heapinuse"),
			MemoryStack:     memoryScope.Gauge("stack"),
			NumGC:           memoryScope.Counter("num-gc"),
			GcPauseMs:       memoryScope.Timer("gc-pause-ms"),
			lastNumGC:       memstats.NumGC,
		},
		started: false,
		quit:    make(chan struct{}),
	}
}

// Start Starts the reporter thread that periodically emits metrics.
func (r *RuntimeMetricsReporter) Start() {
	if r.started {
		return
	}
	go func() {
		ticker := time.NewTicker(r.reportInterval)
		for {
			select {
			case <-ticker.C:
				r.report()
			case <-r.quit:
				ticker.Stop()
				return
			}
		}
	}()
	r.started = true
}

// Stop Stops reporting of runtime metrics.
// The reporter cannot be started again after it's been stopped.
func (r *RuntimeMetricsReporter) Stop() {
	close(r.quit)
}

// report Sends runtime metrics to the local metrics collector.
func (r *RuntimeMetricsReporter) report() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	r.metrics.NumGoRoutines.Update(float64(runtime.NumGoroutine()))
	r.metrics.GoMaxProcs.Update(float64(runtime.GOMAXPROCS(0)))
	r.metrics.MemoryAllocated.Update(float64(memStats.Alloc))
	r.metrics.MemoryHeap.Update(float64(memStats.HeapAlloc))
	r.metrics.MemoryHeapIdle.Update(float64(memStats.HeapIdle))
	r.metrics.MemoryHeapInuse.Update(float64(memStats.HeapInuse))
	r.metrics.MemoryStack.Update(float64(memStats.StackInuse))

	// memStats.NumGC is a perpetually incrementing counter (unless it wraps at 2^32).
	num := memStats.NumGC
	lastNum := atomic.SwapUint32(&r.metrics.lastNumGC, num)
	if delta := num - lastNum; delta > 0 {
		r.metrics.NumGC.Inc(int64(delta))
		if delta > 255 {
			// too many GCs happened, the timestamps buffer got wrapped around. Report only the last 256.
			lastNum = num - 256
		}
		for i := lastNum; i != num; i++ {
			pause := memStats.PauseNs[i%256]
			r.metrics.GcPauseMs.Record(time.Duration(pause))
		}
	}
}
