package observe

import (
	"errors"
	"sync"

	"github.com/m3db/m3x/close"
)

var errClosed = errors.New("closed")

type closer func()

// Observer observes an Observable instance, can get notification when the Observable updates
type Observer interface {
	xclose.SimpleCloser

	// C returns the notification channel
	C() <-chan struct{}
	// Get returns the latest value of the Observable instance
	Get() interface{}
}

// Observable can be observed by Observers
type Observable interface {
	xclose.SimpleCloser

	// Get returns the latest value
	Get() interface{}
	// Observe returns the value and an Observer that will be notified on updates
	Observe() (interface{}, Observer, error)
	// Update sets the value and notify observers
	Update(interface{}) error
	// ObserverLen returns the number of observers
	ObserverLen() int
}

// NewObservable returns an Observable
func NewObservable() Observable {
	return &observable{}
}

type observable struct {
	sync.RWMutex

	value  interface{}
	active []chan struct{}
	closed bool
}

func (o *observable) Get() interface{} {
	o.RLock()
	v := o.value
	o.RUnlock()
	return v
}

func (o *observable) Observe() (interface{}, Observer, error) {
	o.Lock()

	if o.closed {
		o.Unlock()
		return nil, nil, errClosed
	}

	c := make(chan struct{}, 1)
	o.active = append(o.active, c)
	closeFn := o.closeFunc(c)
	o.Unlock()

	observer := &observer{o: o, c: c, closeFn: closeFn}
	return o.Get(), observer, nil
}

func (o *observable) Update(v interface{}) error {
	o.Lock()
	defer o.Unlock()

	if o.closed {
		return errClosed
	}

	o.value = v

	for _, s := range o.active {
		select {
		case s <- struct{}{}:
		default:
		}
	}

	return nil
}

func (o *observable) ObserverLen() int {
	o.RLock()
	l := len(o.active)
	o.RUnlock()
	return l
}

func (o *observable) Close() {
	o.Lock()
	defer o.Unlock()

	if o.closed {
		return
	}

	o.closed = true

	for _, ch := range o.active {
		close(ch)
	}
	o.active = nil
}

func (o *observable) closeFunc(c chan struct{}) closer {
	return func() {
		o.Lock()
		defer o.Unlock()

		if o.closed {
			return
		}

		close(c)

		for i, s := range o.active {
			if s == c {
				o.active = append(o.active[:i], o.active[i+1:]...)
			}
		}
	}
}

type observer struct {
	sync.Mutex

	o       Observable
	c       <-chan struct{}
	closed  bool
	closeFn closer
}

func (o *observer) C() <-chan struct{} {
	return o.c
}

func (o *observer) Get() interface{} {
	return o.o.Get()
}

func (o *observer) Close() {
	o.Lock()
	defer o.Unlock()

	if o.closed {
		return
	}

	o.closed = true

	if o.closeFn != nil {
		o.closeFn()
	}
}
