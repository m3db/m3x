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

package xwatch

import (
	"errors"
	"sync"

	"github.com/m3db/m3x/close"
	"github.com/m3db/m3x/log"
)

// ErrSourceClosed could be thrown from SourceInput to indicate that the Source should be closed
var ErrSourceClosed = errors.New("source closed")

// SourceInput provides data for Source
type SourceInput interface {
	// Poll will be called by Source for data, any backoff/jitter logic should be handled here
	Poll() (interface{}, error)
}

// Source polls data by calling SourcePollFn and notifies its watches on updates
type Source interface {
	xclose.SimpleCloser

	// Initialized returns a channel that blocks until the Source was initialized
	Initialized() <-chan struct{}
	// Watch returns the value and an Watch
	Watch() (interface{}, Watch, error)
}

// NewSource returns a Source
func NewSource(input SourceInput, logger xlog.Logger) Source {
	s := &source{
		input:  input,
		w:      NewWatchable(),
		logger: logger,
		ch:     make(chan struct{}),
	}

	go s.run()
	return s
}

type source struct {
	sync.RWMutex

	input       SourceInput
	w           Watchable
	closed      bool
	initialized bool
	ch          chan struct{}
	logger      xlog.Logger
}

func (s *source) run() {
	for !s.isClosed() {
		data, err := s.input.Poll()
		if err == ErrSourceClosed {
			s.logger.Errorf("Upstream source is closed")
			s.Close()
			return
		}
		if err != nil {
			s.logger.Errorf("Error polling input source: %v", err)
			continue
		}

		err = s.w.Update(data)

		if err == nil && !s.initialized {
			close(s.ch)
			s.initialized = true
		}
	}
}

func (s *source) isClosed() bool {
	s.RLock()
	defer s.RUnlock()
	return s.closed
}

func (s *source) Initialized() <-chan struct{} {
	return s.ch
}

func (s *source) Close() {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.w.Close()
}

func (s *source) Watch() (interface{}, Watch, error) {
	return s.w.Watch()
}
