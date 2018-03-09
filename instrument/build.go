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
	"sync"
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

	// BuildDateUnixNanos is the UnixNanos since epoch representing the date this build was created.
	BuildDateUnixNanos int64 = 0

	// LogBuildInfoAtStartup controls whether we log build information at startup. If its
	// set to a non-empty string, we log the build information at process startup.
	LogBuildInfoAtStartup string

	// goVersion is the current runtime version.
	goVersion = runtime.Version()

	// buildInfoMetricName is the emitted build information metric's name.
	buildInfoMetricName = "build-information"

	// buildAgeMetricName is the emitted build age metric's name.
	buildAgeMetricName = "build-age"
)

var (
	errAlreadyStarted = errors.New("reporter already started")
	errNotStarted     = errors.New("reporter not started")
)

// LogBuildInfo logs the build information to the provided logger.
func LogBuildInfo() {
	log.Printf("Go Runtime version:  %s\n", goVersion)
	log.Printf("Build Revision:      %s\n", Revision)
	log.Printf("Build Branch:        %s\n", Branch)
	log.Printf("Build Date:          %s\n", BuildDate)
	log.Printf("Build DateUnixNanos: %s\n", BuildDateUnixNanos)
}

func init() {
	if LogBuildInfoAtStartup != "" {
		LogBuildInfo()
	}
}

type buildReporter struct {
	sync.Mutex

	opts    Options
	active  bool
	closeCh chan struct{}
	doneCh  chan struct{}
}

// NewBuildReporter returns a new build version reporter.
func NewBuildReporter(
	opts Options,
) Reporter {
	return &buildReporter{
		opts: opts,
	}
}

func (b *buildReporter) Start() error {
	b.Lock()
	defer b.Unlock()
	if b.active {
		return errAlreadyStarted
	}
	b.active = true
	b.closeCh = make(chan struct{})
	b.doneCh = make(chan struct{})
	go b.report()
	return nil
}

func (b *buildReporter) report() {
	buildTime := time.Unix(0, BuildDateUnixNanos)
	scope := b.opts.MetricsScope().Tagged(map[string]string{
		"revision":   Revision,
		"branch":     Branch,
		"build-date": BuildDate,
		"go-version": goVersion,
	})
	buildInfoGauge := scope.Gauge(buildInfoMetricName)
	buildAgeGauge := scope.Gauge(buildAgeMetricName)
	buildInfoGauge.Update(1.0)
	buildAgeGauge.Update(float64(time.Since(buildTime)))

	ticker := time.NewTicker(b.opts.ReportInterval())
	defer func() {
		close(b.doneCh)
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			buildInfoGauge.Update(1.0)
			buildAgeGauge.Update(float64(time.Since(buildTime)))

		case <-b.closeCh:
			return
		}
	}
}

func (b *buildReporter) Stop() error {
	b.Lock()
	defer b.Unlock()
	if !b.active {
		return errNotStarted
	}
	close(b.closeCh)
	<-b.doneCh
	b.active = false
	return nil
}
