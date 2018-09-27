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

// +build !race

package pool

import (
	"runtime"
	"runtime/debug"
	"testing"
)

func newTestPool() *shardedObjectPool {
	p := NewShardedObjectPool(nil).(*shardedObjectPool)
	p.Init(func() interface{} {
		return nil
	})
	return p
}

func TestPool(t *testing.T) {
	// disable GC so we can control when it happens.
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	p := newTestPool()
	if p.Get() != nil {
		t.Fatal("expected empty")
	}

	// Make sure that the goroutine doesn't migrate to another P
	// between Put and Get calls.
	ProcPin()
	p.Put("a")
	p.Put("b")
	if g := p.Get(); g != "a" {
		t.Fatalf("got %#v; want a", g)
	}
	if g := p.Get(); g != "b" {
		t.Fatalf("got %#v; want b", g)
	}
	if g := p.Get(); g != nil {
		t.Fatalf("got %#v; want nil", g)
	}
	ProcUnpin()

	p.Put("c")
	debug.SetGCPercent(100) // to allow following GC to actually run
	runtime.GC()
	if g := p.Get(); g != nil {
		t.Fatalf("got %#v; want nil after GC", g)
	}
}

/*
func TestPoolNew(t *testing.T) {
	// disable GC so we can control when it happens.
	defer debug.SetGCPercent(debug.SetGCPercent(-1))

	i := 0
	p := Pool{
		New: func() interface{} {
			i++
			return i
		},
	}
	if v := p.Get(); v != 1 {
		t.Fatalf("got %v; want 1", v)
	}
	if v := p.Get(); v != 2 {
		t.Fatalf("got %v; want 2", v)
	}

	// Make sure that the goroutine doesn't migrate to another P
	// between Put and Get calls.
	ProcPin()
	p.Put(42)
	if v := p.Get(); v != 42 {
		t.Fatalf("got %v; want 42", v)
	}
	ProcUnpin()

	if v := p.Get(); v != 3 {
		t.Fatalf("got %v; want 3", v)
	}
}

// Test that Pool does not hold pointers to previously cached resources.
func TestPoolGC(t *testing.T) {
	testPool(t, true)
}

// Test that Pool releases resources on GC.
func TestPoolRelease(t *testing.T) {
	testPool(t, false)
}

func testPool(t *testing.T, drain bool) {
	var p Pool
	const N = 100
loop:
	for try := 0; try < 3; try++ {
		var fin, fin1 uint32
		for i := 0; i < N; i++ {
			v := new(string)
			runtime.SetFinalizer(v, func(vv *string) {
				atomic.AddUint32(&fin, 1)
			})
			p.Put(v)
		}
		if drain {
			for i := 0; i < N; i++ {
				p.Get()
			}
		}
		for i := 0; i < 5; i++ {
			runtime.GC()
			time.Sleep(time.Duration(i*100+10) * time.Millisecond)
			// 1 pointer can remain on stack or elsewhere
			if fin1 = atomic.LoadUint32(&fin); fin1 >= N-1 {
				continue loop
			}
		}
		t.Fatalf("only %v out of %v resources are finalized on try %v", fin1, N, try)
	}
}

func TestPoolStress(t *testing.T) {
	const P = 10
	N := int(1e6)
	if testing.Short() {
		N /= 100
	}
	var p Pool
	done := make(chan bool)
	for i := 0; i < P; i++ {
		go func() {
			var v interface{} = 0
			for j := 0; j < N; j++ {
				if v == nil {
					v = 0
				}
				p.Put(v)
				v = p.Get()
				if v != nil && v.(int) != 0 {
					t.Errorf("expect 0, got %v", v)
					break
				}
			}
			done <- true
		}()
	}
	for i := 0; i < P; i++ {
		<-done
	}
}
*/

func BenchmarkShardedPoolGetPut(b *testing.B) {
	opts := NewObjectPoolOptions().SetSize(1)
	pool := NewObjectPool(opts)
	pool.Init(func() interface{} {
		return 1
	})

	for n := 0; n < b.N; n++ {
		o := pool.Get()
		pool.Put(o)
	}
}

// ❯ go test -bench=Pool -cpu 1,2,4 -run '^$' ./pool
// goos: darwin
// goarch: amd64
// pkg: github.com/m3db/m3x/pool
// BenchmarkObjectPoolGetPut        	20000000	        89.7 ns/op
// BenchmarkObjectPoolGetPut-2      	20000000	        88.9 ns/op
// BenchmarkObjectPoolGetPut-4      	20000000	        88.8 ns/op
// BenchmarkShardedPoolGetPut       	20000000	        89.0 ns/op
// BenchmarkShardedPoolGetPut-2     	20000000	        89.7 ns/op
// BenchmarkShardedPoolGetPut-4     	20000000	        89.5 ns/op
// BenchmarkShardedPoolPutGet       	50000000	        37.0 ns/op
// BenchmarkShardedPoolPutGet-2     	100000000	        18.9 ns/op
// BenchmarkShardedPoolPutGet-4     	100000000	        15.8 ns/op
// BenchmarkChannelPoolPutGet       	20000000	        92.1 ns/op
// BenchmarkChannelPoolPutGet-2     	10000000	       145 ns/op
// BenchmarkChannelPoolPutGet-4     	10000000	       167 ns/op
// BenchmarkShardedPoolOverflow     	  200000	      5807 ns/op
// BenchmarkShardedPoolOverflow-2   	  500000	      2986 ns/op
// BenchmarkShardedPoolOverflow-4   	  500000	      2525 ns/op
// BenchmarkChannelPoolOverflow     	  200000	      5821 ns/op
// BenchmarkChannelPoolOverflow-2   	  500000	      3012 ns/op
// BenchmarkChannelPoolOverflow-4   	  500000	      2581 ns/op
// PASS
// ok  	github.com/m3db/m3x/pool	30.338s

func BenchmarkShardedPoolPutGet(b *testing.B) {
	p := NewShardedObjectPool(nil)
	p.Init(func() interface{} {
		return 0
	})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.Put(1)
			p.Get()
		}
	})
}
func BenchmarkChannelPoolPutGet(b *testing.B) {
	p := NewObjectPool(nil)
	p.Init(func() interface{} {
		return 0
	})
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.Put(1)
			p.Get()
		}
	})
}

func BenchmarkShardedPoolOverflow(b *testing.B) {
	p := NewShardedObjectPool(nil)
	p.Init(func() interface{} {
		return 0
	})
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for b := 0; b < 100; b++ {
				p.Put(1)
			}
			for b := 0; b < 100; b++ {
				p.Get()
			}
		}
	})
}

func BenchmarkChannelPoolOverflow(b *testing.B) {
	p := NewShardedObjectPool(nil)
	p.Init(func() interface{} {
		return 0
	})
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for b := 0; b < 100; b++ {
				p.Put(1)
			}
			for b := 0; b < 100; b++ {
				p.Get()
			}
		}
	})
}
