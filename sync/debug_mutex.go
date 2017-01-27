package xsync

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// DebugRWMutex is a RWMutex that tracks its ownership.
type DebugRWMutex struct {
	m sync.RWMutex
	r int64 // The number of readers.
}

// Lock locks DebugRWMutex for writing.
func (m *DebugRWMutex) Lock() {
	if mutexDebuggingFlag &&
		atomic.LoadInt64(&m.r) >= int64(mutexContentionTrigger) {
		panic("debug: contention @ " + traceback(callstack(0)))
	}

	m.m.Lock()
	track(m, lock)
}

// Unlock unlocks DebugRWMutex for writing.
func (m *DebugRWMutex) Unlock() {
	track(m, unlock)
	m.m.Unlock()
}

// RLock locks DebugRWMutex for reading.
func (m *DebugRWMutex) RLock() {
	m.m.RLock()
	track(m, rlock)
}

// RUnlock undoes a single RLock call.
func (m *DebugRWMutex) RUnlock() {
	track(m, runlock)
	m.m.RUnlock()
}

// RLocker returns a Locker interface implemented via calls to RLock
// and RUnlock.
func (m *DebugRWMutex) RLocker() sync.Locker {
	return (*rlocker)(m)
}

type rlocker DebugRWMutex

func (r *rlocker) Lock()   { (*DebugRWMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*DebugRWMutex)(r).RUnlock() }

const _StackDepth = 16

var (
	mutexDebuggingFlag     bool
	mutexContentionTrigger = 10
)

// DisableMutexDebugging turns mutex debugging off.
func DisableMutexDebugging() {
	mutexDebuggingFlag = false
}

// EnableMutexDebugging turns mutex debugging on.
func EnableMutexDebugging() {
	mutexDebuggingFlag = true
}

// SetMutexContentionTrigger sets the minimum number of concurrent
// readers for a write lock attempt to trigger a panic.
func SetMutexContentionTrigger(n int) {
	mutexContentionTrigger = n
}

type mutexOp int

const (
	lock mutexOp = iota
	unlock
	rlock
	runlock
)

type lockInfo struct {
	ts time.Time
	cs []uintptr
}

var locks struct {
	sync.Mutex
	m map[*DebugRWMutex]lockInfo
}

func track(m *DebugRWMutex, op mutexOp) {
	if !mutexDebuggingFlag {
		return
	}

	switch op {
	case lock:
		locks.Lock()
		locks.m[m] = lockInfo{time.Now(), callstack(1)}
		locks.Unlock()
	case unlock:
		locks.Lock()
		delete(locks.m, m)
		locks.Unlock()
	case rlock:
		atomic.AddInt64(&m.r, +1)
	case runlock:
		atomic.AddInt64(&m.r, -1)
	}
}

func callstack(skip int) []uintptr {
	r := make([]uintptr, _StackDepth)
	n := runtime.Callers(skip+2, r) // Skips the caller & itself.

	return r[:n]
}

func traceback(l []uintptr) string {
	var (
		b    = new(bytes.Buffer)
		n    runtime.Frame
		more = len(l) != 0
	)

	for f := runtime.CallersFrames(l); more; {
		n, more = f.Next()
		fmt.Fprintf(b, "%s:%d\n\t%s\n", n.File, n.Line, n.Function)
	}

	return b.String()
}

// DumpLocks returns all currently locked mutexes.
func DumpLocks() []string {
	var r []string

	locks.Lock()

	for m, l := range locks.m {
		r = append(r, fmt.Sprintf(
			"%p @ %s\n%s", m, time.Since(l.ts), traceback(l.cs)))
	}

	locks.Unlock()

	return r
}

func init() {
	locks.m = make(map[*DebugRWMutex]lockInfo)
}
