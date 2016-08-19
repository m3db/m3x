package observe

import (
	"sync"

	"github.com/m3db/m3x/close"
	"github.com/m3db/m3x/log"
)

// SourceInput is a source that can be polled for data
type SourceInput interface {
	Poll() (interface{}, error)
}

// Source is a source that can be observed
// it polls on the source input and notifies observers on updates
type Source interface {
	xclose.SimpleCloser

	// Observe returns the value and an Observer that will be notified on updates
	Observe() (interface{}, Observer, error)
}

// NewSource returns a Source
func NewSource(input SourceInput, logger xlog.Logger) Source {
	s := &source{
		input:  input,
		o:      NewObservable(),
		logger: logger,
	}

	go s.run()
	return s
}

type source struct {
	sync.RWMutex

	input  SourceInput
	o      Observable
	logger xlog.Logger
	closed bool
}

func (s *source) run() {
	for !s.isClosed() {
		data, err := s.input.Poll()
		if err != nil {
			s.logger.Errorf("error polling input source: %v", err)
			continue
		}
		s.o.Update(data)
	}
}

func (s *source) isClosed() bool {
	s.RLock()
	defer s.RUnlock()
	return s.closed
}

func (s *source) Close() {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.o.Close()
}

func (s *source) Observe() (interface{}, Observer, error) {
	return s.o.Observe()
}
