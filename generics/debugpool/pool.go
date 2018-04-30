// Copyright (c) 2018 Uber Technologies, Inc.
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

package debugpool

import (
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/cheekybits/genny/generic"
)

// elemType is the generic type for use with the debug pool.
type elemType generic.Type

// elemTypeEqualsFn allows users to override equality checks
// for `elemType` instances.
type elemTypeEqualsFn func(a, b elemType) bool

// elemTypePool is the backing pool wrapped by the debug pool.
type elemTypePool interface {
	generic.Type

	Init()
	Get() elemType
	Put(elemType)
}

// debugElemTypePoolOpts allows users to override default behaviour.
type debugElemTypePoolOpts struct {
	disallowUntrackedPuts bool
	equalsFn              elemTypeEqualsFn
}

// newDebugElemTypePool returns a new debugElemTypePool
func newDebugElemTypePool(opts debugElemTypePoolOpts, backingPool elemTypePool) *debugElemTypePool {
	if opts.equalsFn == nil {
		// NB(prateek): fall-back to == in the worst case
		opts.equalsFn = func(a, b elemType) bool {
			return a == b
		}
	}
	return &debugElemTypePool{opts: opts, elemTypePool: backingPool}
}

// debugElemTypePool wraps the underlying elemTypePool to allow
// tests to track Gets/Puts.
type debugElemTypePool struct {
	sync.Mutex
	elemTypePool
	numGets      int
	numPuts      int
	pendingItems []debugElemType
	allGetItems  []debugElemType

	opts debugElemTypePoolOpts
}

type debugElemType struct {
	value    elemType
	getStack []byte // getStack is the Stacktrace for the retrieval of this item
}

func (p *debugElemTypePool) Init() {
	p.Lock()
	defer p.Unlock()
	p.elemTypePool.Init()
}

func (p *debugElemTypePool) Get() elemType {
	p.Lock()
	defer p.Unlock()

	e := p.elemTypePool.Get()

	p.numGets++
	item := debugElemType{
		value:    e,
		getStack: debug.Stack(),
	}
	p.pendingItems = append(p.pendingItems, item)
	p.allGetItems = append(p.allGetItems, item)

	return e
}

func (p *debugElemTypePool) Put(value elemType) {
	p.Lock()
	defer p.Unlock()

	idx := -1
	for i, item := range p.pendingItems {
		if p.opts.equalsFn(item.value, value) {
			idx = i
			break
		}
	}

	if idx == -1 && p.opts.disallowUntrackedPuts {
		panic(fmt.Errorf("untracked object (%v) returned to pool", value))
	}

	if idx != -1 {
		// update slice
		p.pendingItems = append(p.pendingItems[:idx], p.pendingItems[idx+1:]...)
	}
	p.numPuts++

	p.elemTypePool.Put(value)
}
