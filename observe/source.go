package observe

import (
	"sync"

	"github.com/m3db/m3x/close"
	"github.com/m3db/m3x/log"
)

type SourceInput interface {
	Poll() (interface{}, error)
}

type ObservableSource interface {
	xclose.SimpleCloser

	GetAndSubscribe() (interface{}, Observer, error)
}

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
	o, err := s.o.Subscribe()
	val := s.o.Get()
	return val, o, err
}
