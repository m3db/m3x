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
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/m3db/m3x/instrument"

	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
)

func tallyTestScopeKey(k string) string {
	return k + "+"
}

func checkMetrics(wantG, wantC map[string]int64, s tally.TestScope) bool {
	return len(checkMetricsExtended(wantG, wantC, s)) == 0
}

func checkMetricsExtended(wantG, wantC map[string]int64, s tally.TestScope) []string {
	errors := []string{}
	snap := s.Snapshot()
	gauges := snap.Gauges()
	for key, value := range wantG {
		gauge, ok := gauges[tallyTestScopeKey(key)]
		if !ok {
			errors = append(errors, fmt.Sprintf("missing expected gauge: %s", gauge))
			continue
		}
		if gauge.Value() != float64(value) {
			errors = append(errors, fmt.Sprintf("incorrect value for gauge %s, expected: %d, observed: %f", key, value, gauge.Value()))
			continue
		}
	}
	counters := snap.Counters()
	for key, value := range wantC {
		counter, ok := counters[tallyTestScopeKey(key)]
		if !ok {
			errors = append(errors, fmt.Sprintf("missing expected counter: %s", key))
			continue
		}
		if counter.Value() != value {
			errors = append(errors, fmt.Sprintf("incorrect value for counter %s, expected: %d, observed: %d", key, value, counter.Value()))
			continue
		}
	}
	return errors
}

func assertMetrics(t *testing.T, wantG, wantC map[string]int64, s tally.TestScope) {
	start := time.Now()
	for !checkMetrics(wantG, wantC, s) {
		if time.Since(start) > time.Second {
			t.Logf("Timed out waiting for metrics to be equal")
			break
		}
		runtime.Gosched()
	}

	errors := checkMetricsExtended(wantG, wantC, s)
	if len(errors) == 0 {
		return
	}
	assert.FailNow(t, strings.Join(errors, "\n"))
}

func TestInstrumentedWorkerPoolClose(t *testing.T) {
	defer leaktest.CheckTimeout(t, time.Second)()
	scope := tally.NewTestScope("", nil)
	opts := instrument.NewOptions().
		SetMetricsScope(scope).
		SetReportInterval(100 * time.Millisecond)
	p := NewInstrumentedWorkerPool(10, opts)
	p.Init()
	assert.NoError(t, p.Close())
}
func TestInstrumentedWorkerPool(t *testing.T) {
	defer leaktest.CheckTimeout(t, time.Second)()

	scope := tally.NewTestScope("", nil)
	opts := instrument.NewOptions().
		SetMetricsScope(scope).
		SetReportInterval(100 * time.Millisecond)
	p := NewInstrumentedWorkerPool(10, opts)
	p.Init()
	defer func() {
		assert.NoError(t, p.Close())
	}()

	// Consume all available workers
	var ready sync.WaitGroup
	var complete sync.WaitGroup
	release := make(chan struct{})
	for i := 0; i < 10; i++ {
		ready.Add(1)
		complete.Add(1)
		p.Go(func() {
			ready.Done()
			<-release
			complete.Done()
		})
	}

	ready.Wait()

	// Make sure we have the right metrics
	assertMetrics(t, map[string]int64{
		"workers.idle":    0,
		"workers.waiting": 0,
		"workers.busy":    10,
		"workers.total":   10},
		map[string]int64{}, scope)

	// Add another goroutine - this one should block since
	// there are no workers available
	complete.Add(1)
	launched := make(chan struct{})
	go func() {
		close(launched)
		p.Go(func() { complete.Done() })
	}()
	<-launched

	assertMetrics(t, map[string]int64{
		"workers.idle":    0,
		"workers.waiting": 1,
		"workers.busy":    10,
		"workers.total":   10},
		map[string]int64{}, scope)

	// Add another goroutine - this should timeout
	ran := p.GoWithTimeout(func() {}, 100*time.Millisecond)
	assert.False(t, ran)
	assertMetrics(t, map[string]int64{
		"workers.idle":    0,
		"workers.waiting": 1,
		"workers.busy":    10,
		"workers.total":   10},
		map[string]int64{
			"workers.timeouts": 1,
		}, scope)

	// Add another goroutine - this should return false and not run.
	ran = p.GoIfAvailable(func() {})
	assert.False(t, ran)
	assertMetrics(t, map[string]int64{
		"workers.idle":    0,
		"workers.waiting": 1,
		"workers.busy":    10,
		"workers.total":   10},
		map[string]int64{
			"workers.timeouts":    1,
			"workers.unavailable": 1,
		}, scope)

	// Release everything and let them complete
	close(release)
	complete.Wait()

	assertMetrics(t, map[string]int64{
		"workers.idle":    10,
		"workers.waiting": 0,
		"workers.busy":    0,
		"workers.total":   10},
		map[string]int64{
			"workers.timeouts":    1,
			"workers.unavailable": 1,
			"workers.run":         11,
		}, scope)
}
