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

package instrument

const invariantViolatedMetricName = "system-invariant-violated"

// EmitInvariantViolation emits a metric to indicate a system invariant has
// been violated. Users of this method are expected to monitor/alert off this
// metric to ensure they're notified when such an event occurs. Further, they
// should log further information to aid diagnostics of the system invariant
// violated at the callsite of the violation.
func EmitInvariantViolation(opts Options) {
	// NB(prateek): there's no need to cache this metric. It should be never
	// be called in production systems unless something is seriously messed
	// up. At which point, the extra map alloc should be of no concern.
	opts.MetricsScope().Counter(invariantViolatedMetricName).Inc(1)
}
