package observe

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/m3db/m3x/log"
	"github.com/stretchr/testify/assert"
)

func TestSource(t *testing.T) {
	testSource(t, 30, 25, 20)
	testSource(t, 22, 18, 20)
	testSource(t, 15, 10, 20)
	testSource(t, 28, 30, 20)
	testSource(t, 19, 21, 20)
	testSource(t, 13, 15, 20)
}

func testSource(t *testing.T, inputErrAfter int, closeAfter int, observerNum int) {
	input := testSourceInput(inputErrAfter)
	s := NewObservableSource(input, xlog.SimpleLogger)

	var wg sync.WaitGroup

	// create a few observers
	for i := 0; i < observerNum; i++ {
		wg.Add(1)
		_, o, err := s.GetAndSubscribe()
		assert.NoError(t, err)

		i := i
		go func() {
			var v interface{}
			count := 0
			for _ = range o.C() {
				if v != nil {
					assert.True(t, o.Get().(int64) >= v.(int64))
				}
				v = o.Get()
				if count > i {
					o.Close()
				}
				count++
			}
			wg.Done()
		}()
	}

	// schedule a thread to close Source
	wg.Add(1)
	go func() {
		for input.(*fakeInput).called < closeAfter {
			time.Sleep(1 * time.Millisecond)
		}
		s.Close()
		assert.True(t, s.(*source).isClosed())
		// test Close again
		s.Close()
		assert.True(t, s.(*source).isClosed())
		wg.Done()
	}()

	wg.Wait()
}

func testSourceInput(errAfter int) SourceInput {
	return &fakeInput{errAfter: errAfter}
}

type fakeInput struct {
	errAfter int
	called   int
}

func (i *fakeInput) Poll() (interface{}, error) {
	i.called++
	time.Sleep(time.Millisecond)
	if i.errAfter > 0 {
		i.errAfter--
		return time.Now().Unix(), nil
	}
	return nil, errors.New("mock error")
}
