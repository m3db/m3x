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

package linux

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/m3db/m3x/instrument"
	xlog "github.com/m3db/m3x/log"

	"github.com/uber-go/tally"
)

type fileDescriptorsReporterState int

const (
	fileDescriptorsReporterNotStarted fileDescriptorsReporterState = iota
	fileDescriptorsReporterStarted
	fileDescriptorsReporterStopped
)

var (
	errFileDescriptorReporterAlreadyStartedOrStopped = errors.New(
		"file descriptors reporter already started or stopped")
	errFileDescriptorReporterNotRunning = errors.New(
		"file descriptors reporter not running")
)

type fileDescriptorsReporter struct {
	sync.Mutex
	state    fileDescriptorsReporterState
	logger   xlog.Logger
	fds      tally.Gauge
	interval time.Duration
	closeCh  chan struct{}
}

// NewFileDescriptorsReporter returns a new instrument.Reporter that
// reports linux file descriptor counts for the current process
func NewFileDescriptorsReporter(
	iopts instrument.Options,
) instrument.Reporter {
	scope := iopts.MetricsScope().SubScope("process").Tagged(map[string]string{
		"platform": Platform,
	})

	return &fileDescriptorsReporter{
		logger:   iopts.Logger(),
		fds:      scope.Gauge("num-fds"),
		interval: iopts.ReportInterval(),
		closeCh:  make(chan struct{}, 1),
	}
}

func (r *fileDescriptorsReporter) Start() error {
	r.Lock()
	defer r.Unlock()

	if r.state != fileDescriptorsReporterNotStarted {
		return errFileDescriptorReporterAlreadyStartedOrStopped
	}

	pid := os.Getpid()
	statPath := fmt.Sprintf("/proc/%d/fd", pid)

	if _, err := os.Open(statPath); err != nil {
		return fmt.Errorf("unable to stat process fd dir, check running on linux: %v", err)
	}

	if every <= 0 {
		return fmt.Errorf("invalid report interval: %s", reportInterval.String())
	}

	r.state = fileDescriptorsReporterStarted

	go r.reportLoop()
}

func (r *fileDescriptorsReporter) Stop() error {
	r.Lock()
	defer r.Unlock()

	if r.state != fileDescriptorsReporterStarted {
		return errFileDescriptorReporterNotRunning
	}

	r.state = fileDescriptorsReporterStopped
	close(r.closeCh)
}

func (r *fileDescriptorsReporter) reportLoop(every time.Duration) {
	tick := time.NewTicker(r.interval)
	defer tick.Stop()

	for {
		select {
		case <-r.closeCh:
			return
		case <-tick.C:
			d, err := os.Open(statPath)
			if err != nil {
				r.logger.Errorf("unable to read process fd dir: %v", err)
				continue
			}

			fnames, err := d.Readdirnames(-1)
			closeErr := d.Close()

			if err != nil {
				r.logger.Errorf("unable to read process fd dir entries: %v", err)
				continue
			}

			r.fs.Update(float64(len(fnames)))

			if closeErr != nil {
				r.logger.Errorf("unable to close process fd dir: %v", closeErr)
			}
		}
	}
}
