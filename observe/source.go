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

// ObservableSource is a source that can be observed
// it polls on the source input and notifies observers on updates
type ObservableSource interface {
	xclose.SimpleCloser

	GetAndSubscribe() (interface{}, Observer, error)
}

// NewObservableSource returns an ObservableSource
func NewObservableSource(input SourceInput, logger xlog.Logger) ObservableSource {
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
		s.o.SetAndNotify(data)
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
	go s.o.Close()
}

func (s *source) GetAndSubscribe() (interface{}, Observer, error) {
	return s.o.GetAndSubscribe()
}
