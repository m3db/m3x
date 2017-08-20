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

package instrument

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	// Githash is the githash associated with this build. Overridden using ldflags
	// at compile time. Example:
	// $ go build -ldflags "-X github.com/m3db/m3x/instrument.Githash=abcdef" ...
	// Adapted from: https://www.atatus.com/blog/golang-auto-build-versioning/
	Githash = "unknown"

	// GoVersion is the current runtime version
	goVersion = runtime.Version()

	// metricName is the emitted metric's name
	metricName = "build-information"
)

var (
	errAlreadyStarted = errors.New("reporter already started")
	errNotStarted     = errors.New("reporter not started")
)

type versionReporter struct {
	opts    Options
	active  int64
	closeCh chan struct{}
	doneCh  chan struct{}
}

// NewVersionReporter returns a new version reporter
func NewVersionReporter(
	opts Options,
) VersionReporter {
	return &versionReporter{
		opts: opts,
	}
}

func (v *versionReporter) Start() error {
	if active := atomic.LoadInt64(&v.active); active != 0 {
		return errAlreadyStarted
	}
	atomic.StoreInt64(&v.active, 1)
	v.closeCh = make(chan struct{}, 1)
	v.doneCh = make(chan struct{}, 1)
	go v.report()
	return nil
}

func (v *versionReporter) report() {
	v.opts.Logger().Infof("Go runtime version: %s", goVersion)
	v.opts.Logger().Infof("Build githash: %s", Githash)

	scope := v.opts.MetricsScope().Tagged(map[string]string{
		"githash":    Githash,
		"go.version": goVersion,
	})
	gauge := scope.Gauge(metricName)
	gauge.Update(1.0)

	ticker := time.NewTicker(v.opts.ReportInterval())
	defer func() {
		close(v.doneCh)
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			gauge.Update(1.0)
		case <-v.closeCh:
			return
		}
	}
}

func (v *versionReporter) Close() error {
	if active := atomic.LoadInt64(&v.active); active == 0 {
		return errNotStarted
	}
	close(v.closeCh)
	<-v.doneCh
	atomic.StoreInt64(&v.active, 0)
	return nil
}
