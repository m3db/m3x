package pool

import "time"

// ResourcePool is a fixed-sized pool for long-lived expensive objects,
// typically things like connections.  It differs from sync.Pool which is
// tailored for short-lived pooling.  Pooled objects are pre-allocated
// when the pool is created, retrieved via Get(), and returned via Release().
// If a pooled object breaks, it should be returned via Destroy() which will
// cause a new object to be allocated.
type ResourcePool interface {
	// Get retrieves an object from the pool, blocking until the object is available
	Get() interface{}

	// GetWithDeadline retrieves an object from the pool, blocking
	// until the object is available or the deadline passes
	GetWithDeadline(deadline time.Time) interface{}

	// GetOrAlloc retrieves an object from the pool or creates
	// a new object if the pool is empty. If the number of
	// objects created exceeds the size of the pool, extra
	// objects will be eventually reclaimed by the GC.
	GetOrAlloc() (interface{}, error)

	// Release returns an object to the pool and returns immediately.
	// If the pool is full because additional objects were created with
	// GetOrAlloc then extra objects will be eventually reclaimed by the GC.
	Release(interface{})

	// Destroy marks an object as being broken
	Destroy(interface{})
}

// AllocFunc is a function used for allocating objects for the pool
type AllocFunc func() (interface{}, error)

// ValidateFunc is a function used for validate objects in the pool
type ValidateFunc func(interface{}) bool

// StandardResourcePoolOptions are options that control the behavior of a standard object pool
type StandardResourcePoolOptions struct {
	// TestOnRelease validates an object asynchronously when released.  Objects that
	// fail validation are not returned to the pool, and are replaced with new allocations
	TestOnRelease ValidateFunc

	// TestOnGet validates an object synchronously when retrieved.  Objects that fail
	// validation are not returned to the caller, and are replaced with new allocations
	TestOnGet ValidateFunc

	// ReallocRetryWait is the amount of time to wait before retrying if a reallocation fails
	ReallocRetryWait time.Duration
}

// NewStandardResourcePool creates a new object pool of the given size
func NewStandardResourcePool(size int, alloc AllocFunc, opts *StandardResourcePoolOptions) (ResourcePool, error) {
	objects := make(chan interface{}, size)
	for i := 0; i < size; i++ {
		o, err := alloc()
		if err != nil {
			return nil, err
		}

		objects <- o
	}

	if opts == nil {
		opts = &StandardResourcePoolOptions{}
	}

	reallocRetryWait := opts.ReallocRetryWait
	if reallocRetryWait == time.Duration(0) {
		reallocRetryWait = time.Millisecond * 500
	}

	return &standardPool{
		objects:          objects,
		alloc:            alloc,
		testOnGet:        opts.TestOnGet,
		testOnRelease:    opts.TestOnRelease,
		reallocRetryWait: reallocRetryWait,
	}, nil
}

type standardPool struct {
	objects          chan interface{}
	alloc            AllocFunc
	testOnGet        ValidateFunc
	testOnRelease    ValidateFunc
	reallocRetryWait time.Duration
}

func (p *standardPool) GetWithDeadline(deadline time.Time) interface{} {
	for {
		select {
		case <-time.After(deadline.Sub(time.Now())):
			return nil
		case o := <-p.objects:
			if p.confirmValidOnGet(o) {
				return o
			}
		}
	}
}

func (p *standardPool) Get() interface{} {
	for {
		o := <-p.objects
		if p.confirmValidOnGet(o) {
			return o
		}
	}
}

func (p *standardPool) GetOrAlloc() (interface{}, error) {
	for {
		select {
		case o := <-p.objects:
			if p.confirmValidOnGet(o) {
				return o, nil
			}
		default:
			o, err := p.alloc()
			if err != nil {
				return nil, err
			}
			if p.confirmValidOnGet(o) {
				return o, nil
			}
		}
	}
}

func (p *standardPool) confirmValidOnGet(o interface{}) bool {
	if p.testOnGet == nil {
		return true
	}

	if p.testOnGet(o) {
		return true
	}

	go p.realloc()
	return false
}

func (p *standardPool) Release(o interface{}) {
	if p.testOnRelease == nil {
		select {
		case p.objects <- o:
			return
		default:
			return
		}
	}

	go func() {
		if p.testOnRelease(o) {
			select {
			case p.objects <- o:
				return
			default:
				return
			}
		}

		p.realloc()
	}()
}

func (p *standardPool) Destroy(o interface{}) {
	go p.realloc()
}

func (p *standardPool) realloc() {
	for {
		o, err := p.alloc()
		if err == nil {
			p.objects <- o
			return
		}

		time.Sleep(p.reallocRetryWait)
	}
}
