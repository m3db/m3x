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

package context

import (
	stdctx "context"
	"sync"

	xopentracing "github.com/m3db/m3x/opentracing"
	"github.com/m3db/m3x/resource"

	"github.com/opentracing/opentracing-go"
)

// NB(r): using golang.org/x/net/context is too GC expensive.
// Instead, we just embed one.
type ctx struct {
	sync.RWMutex

	goCtx         stdctx.Context
	pool          contextPool
	done          bool
	wg            sync.WaitGroup
	finalizeables []finalizeable
	parent        Context
}

type finalizeable struct {
	finalizer resource.Finalizer
	closer    resource.Closer
}

// NewContext creates a new context.
func NewContext() Context {
	return newPooledContext(nil)
}

// NewPooledContext returns a new context that is returned to a pool when closed.
func newPooledContext(pool contextPool) Context {
	return &ctx{pool: pool}
}

func (c *ctx) GoContext() (stdctx.Context, bool) {
	if c.goCtx == nil {
		return nil, false
	}

	return c.goCtx, true
}

func (c *ctx) SetGoContext(v stdctx.Context) {
	c.goCtx = v
}

func (c *ctx) IsClosed() bool {
	parent := c.ParentCtx()
	if parent == nil {
		return parent.IsClosed()
	}

	c.RLock()
	done := c.done
	c.RUnlock()

	return done
}

func (c *ctx) RegisterFinalizer(f resource.Finalizer) {
	parent := c.ParentCtx()
	if parent != nil {
		parent.RegisterFinalizer(f)
		return
	}

	c.registerFinalizeable(finalizeable{finalizer: f})
}

func (c *ctx) RegisterCloser(f resource.Closer) {
	parent := c.ParentCtx()
	if parent != nil {
		parent.RegisterCloser(f)
		return
	}

	c.registerFinalizeable(finalizeable{closer: f})
}

func (c *ctx) registerFinalizeable(f finalizeable) {
	if c.Lock(); c.done {
		c.Unlock()
		return
	}

	if c.finalizeables != nil {
		c.finalizeables = append(c.finalizeables, f)
		c.Unlock()
		return
	}

	if c.pool != nil {
		c.finalizeables = append(c.pool.getFinalizeables(), f)
	} else {
		c.finalizeables = append(allocateFinalizeables(), f)
	}

	c.Unlock()
}

func allocateFinalizeables() []finalizeable {
	return make([]finalizeable, 0, defaultInitFinalizersCap)
}

func (c *ctx) DependsOn(blocker Context) {
	parent := c.ParentCtx()
	if parent != nil {
		parent.DependsOn(blocker)
		return
	}

	c.Lock()

	if !c.done {
		c.wg.Add(1)
		blocker.RegisterFinalizer(c)
	}

	c.Unlock()
}

// Finalize handles a call from another context that was depended upon closing.
func (c *ctx) Finalize() {
	c.wg.Done()
}

type closeMode int

const (
	closeAsync closeMode = iota
	closeBlock
)

func (c *ctx) Close() {
	parent := c.ParentCtx()
	if parent != nil {
		parent.Close()
		return
	}

	c.close(closeAsync)
}

func (c *ctx) BlockingClose() {
	parent := c.ParentCtx()
	if parent != nil {
		parent.BlockingClose()
		return
	}

	c.close(closeBlock)
}

func (c *ctx) close(mode closeMode) {
	if c.Lock(); c.done {
		c.Unlock()
		return
	}

	c.done = true
	c.Unlock()

	if c.finalizeables == nil {
		c.returnToPool()
		return
	}

	// Capture finalizeables to avoid concurrent r/w if Reset
	// is used after a caller waits for the finalizers to finish
	f := c.finalizeables
	c.finalizeables = nil

	switch mode {
	case closeAsync:
		go c.finalize(f)
	case closeBlock:
		c.finalize(f)
	}
}

func (c *ctx) finalize(f []finalizeable) {
	// Wait for dependencies.
	c.wg.Wait()

	// Now call finalizers.
	for i := range f {
		if f[i].finalizer != nil {
			f[i].finalizer.Finalize()
			f[i].finalizer = nil
		}
		if f[i].closer != nil {
			f[i].closer.Close()
			f[i].closer = nil
		}
	}

	if c.pool != nil {
		c.pool.putFinalizeables(f)
	}

	c.returnToPool()
}

func (c *ctx) Reset() {
	parent := c.ParentCtx()
	if parent != nil {
		parent.Reset()
		return
	}

	c.Lock()
	c.done, c.finalizeables, c.goCtx = false, nil, nil
	c.Unlock()
}

func (c *ctx) returnToPool() {
	if c.pool == nil {
		return
	}

	c.Reset()
	c.pool.Put(c)
}

func (c *ctx) NewChildContext() Context {
	var child Context
	if c.pool == nil {
		child = NewContext()
	} else {
		child = c.pool.Get()
	}

	child.SetParentCtx(c)

	return child
}

func (c *ctx) SetParentCtx(parentCtx Context) {
	// check to see if parent exists?
	c.parent = parentCtx
}

func (c *ctx) ParentCtx() Context {
	c.RLock()
	defer c.RUnlock()
	return c.parent
}

func (c *ctx) StartSpan(name string) (Context, opentracing.Span, bool) {
	goCtx, exists := c.GoContext()
	if !exists {
		return c, nil, false
	}

	// Figure this out
	// jaegerctx, ok := goCtx.(jaeger.SpanContext) // figure out how to cast since it doesn't satisfy our ctx's interface
	// if !ok {
	// 	return c, nil, false
	// }

	// if !jaegerctx.Sampled() {
	// 	return c, nil, false
	// }

	sp, spCtx := xopentracing.StartSpanFromContext(goCtx, name)
	child := c.NewChildContext()
	child.SetGoContext(spCtx)
	return child, sp, true
}
