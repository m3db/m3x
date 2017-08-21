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
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	// Revision is the VCS revision associated with this build. Overridden using ldflags
	// at compile time. Example:
	// $ go build -ldflags "-X github.com/m3db/m3x/instrument.Revision=abcdef" ...
	// Adapted from: https://www.atatus.com/blog/golang-auto-build-versioning/
	Revision = "unknown"

	// Branch is the VCS branch associated with this build.
	Branch = "unknown"

	// BuildDate is the date this build was created.
	BuildDate = "unknown"

	// LogBuildInfoAtStartup controls whether we log build information at startup.
	LogBuildInfoAtStartup = false

	// goVersion is the current runtime version.
	goVersion = runtime.Version()

	// metricName is the emitted metric's name.
	metricName = "build-information"
)

var (
	errAlreadyStarted = errors.New("reporter already started")
	errNotStarted     = errors.New("reporter not started")
)

// LogBuildInformation logs the build information to the provided logger.
func LogBuildInformation() {
	log.Printf("Go Runtime version: %s\n", goVersion)
	log.Printf("Build Revision:     %s\n", Revision)
	log.Printf("Build Branch:       %s\n", Branch)
	log.Printf("Build Date:         %s\n", BuildDate)
}

func init() {
	if LogBuildInfoAtStartup {
		LogBuildInformation()
	}
}

type buildReporter struct {
	opts    Options
	active  int64
	closeCh chan struct{}
	doneCh  chan struct{}
}

// NewBuildReporter returns a new version reporter.
func NewBuildReporter(
	opts Options,
) BuildReporter {
	return &buildReporter{
		opts: opts,
	}
}

func (v *buildReporter) Start() error {
	if swapped := atomic.CompareAndSwapInt64(&v.active, 0, 1); !swapped {
		return errAlreadyStarted
	}
	v.closeCh = make(chan struct{}, 1)
	v.doneCh = make(chan struct{}, 1)
	go v.report()
	return nil
}

func (v *buildReporter) report() {
	scope := v.opts.MetricsScope().Tagged(map[string]string{
		"revision":   Revision,
		"branch":     Branch,
		"build-date": BuildDate,
		"go-version": goVersion,
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

func (v *buildReporter) Close() error {
	if active := atomic.LoadInt64(&v.active); active == 0 {
		return errNotStarted
	}
	close(v.closeCh)
	<-v.doneCh
	atomic.StoreInt64(&v.active, 0)
	return nil
}
