package observe

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestObservable(t *testing.T) {
	p := NewObservable()
	assert.Nil(t, p.Get())
	assert.Equal(t, 0, p.ObserverLen())
	assert.NoError(t, p.SetAndNotify(nil))
	get := 100
	p = NewObservable()
	p.SetAndNotify(get)
	assert.Equal(t, get, p.Get())
	v, s, err := p.GetAndSubscribe()
	assert.NotNil(t, s)
	assert.Equal(t, get, v)
	assert.NoError(t, err)
	assert.NoError(t, p.SetAndNotify(get))
	assert.Equal(t, 1, p.ObserverLen())

	p.Close()
	assert.Equal(t, 0, p.ObserverLen())
	assert.Equal(t, get, p.Get())
	_, s, err = p.GetAndSubscribe()
	assert.Nil(t, s)
	assert.Equal(t, errClosed, err)
	assert.Equal(t, errClosed, p.SetAndNotify(get))
	assert.NotPanics(t, p.Close)
}

func TestObserver(t *testing.T) {
	p := NewObservable()
	_, s, err := p.GetAndSubscribe()
	assert.NoError(t, err)

	err = p.SetAndNotify(nil)
	assert.NoError(t, err)

	_, ok := <-s.C()
	assert.True(t, ok)
	assert.Nil(t, s.Get())

	assert.Equal(t, 1, p.ObserverLen())
	s.Close()
	_, ok = <-s.C()
	assert.False(t, ok)
	assert.Equal(t, 0, p.ObserverLen())
	assert.NotPanics(t, s.Close)

	get := 100
	p = NewObservable()
	_, s, err = p.GetAndSubscribe()
	assert.NoError(t, err)

	err = p.SetAndNotify(get)
	assert.Equal(t, get, p.Get())
	assert.NoError(t, err)
	_, ok = <-s.C()
	assert.True(t, ok)
	assert.Equal(t, get, s.Get())

	// sub.Close() after p.Close()
	assert.Equal(t, 1, p.ObserverLen())
	p.Close()
	assert.Equal(t, 0, p.ObserverLen())
	s.Close()
	_, ok = <-s.C()
	assert.False(t, ok)
	assert.Equal(t, 0, p.ObserverLen())
}

func TestMultiObserver(t *testing.T) {
	p := NewObservable()
	subLen := 20
	subMap := make(map[int]Observer, subLen)
	valueMap := make(map[int]int, subLen)
	for i := 0; i < subLen; i++ {
		_, s, err := p.GetAndSubscribe()
		assert.NoError(t, err)
		subMap[i] = s
		valueMap[i] = -1
	}

	for i := 0; i < subLen; i++ {
		i := i
		testObserveAndClose(t, p, subMap, valueMap, i)
	}

	assert.Equal(t, 0, p.ObserverLen())
	p.Close()
}

func testObserveAndClose(t *testing.T, p Observable, subMap map[int]Observer, valueMap map[int]int, value interface{}) {
	err := p.SetAndNotify(value)
	assert.NoError(t, err)

	for i, s := range subMap {
		_, ok := <-s.C()
		assert.True(t, ok)
		v := s.Get().(int)
		assert.True(t, v > valueMap[i], fmt.Sprintf("Get() value should be > than before: %v, %v", v, valueMap[i]))
		valueMap[i] = v
	}

	l := p.ObserverLen()
	assert.Equal(t, len(subMap), l)

	// randomly close 1 subscriber
	for i, s := range subMap {
		s.Close()
		_, ok := <-s.C()
		assert.False(t, ok)
		p.Get()
		delete(subMap, i)
		delete(valueMap, i)
		break
	}
	assert.Equal(t, l-1, p.ObserverLen())
}

func TestAsyncObserver(t *testing.T) {
	p := NewObservable()

	subLen := 10
	var wg sync.WaitGroup

	for i := 0; i < subLen; i++ {
		_, s, err := p.GetAndSubscribe()
		assert.NoError(t, err)

		wg.Add(1)
		go func() {
			for _ = range s.C() {
				r := rand.Int63n(100)
				time.Sleep(time.Millisecond * time.Duration(r))
			}
			_, ok := <-s.C()
			// chan got closed
			assert.False(t, ok)
			// got the latest value
			assert.Equal(t, subLen-1, s.Get())
			wg.Done()
		}()
	}

	for i := 0; i < subLen; i++ {
		err := p.SetAndNotify(i)
		assert.NoError(t, err)
	}
	p.Close()
	assert.Equal(t, 0, p.ObserverLen())
	wg.Wait()
}
