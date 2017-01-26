package xsync

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
)

const _StackDepth = 16

// DebugMutex is a RWMutex that tracks its ownership.
type DebugMutex struct {
	m sync.RWMutex
}

// RLock locks DebugMutex for reading.
func (m *DebugMutex) RLock() { m.m.RLock() }

// RUnlock undoes a single RLock call.
func (m *DebugMutex) RUnlock() { m.m.RUnlock() }

// Lock locks DebugMutex for writing.
func (m *DebugMutex) Lock() {
	m.m.Lock()
	insert(m)
}

// Unlock unlocks DebugMutex for writing.
func (m *DebugMutex) Unlock() {
	remove(m)
	m.m.Unlock()
}

// RLocker returns a Locker interface implemented via calls to RLock
// and RUnlock.
func (m *DebugMutex) RLocker() sync.Locker {
	return (*rlocker)(m)
}

type rlocker DebugMutex

func (r *rlocker) Lock()   { r.m.RLock() }
func (r *rlocker) Unlock() { r.m.RUnlock() }

var mutexDebuggingFlag bool

// DisableMutexDebugging turns mutex debugging off.
func DisableMutexDebugging() {
	mutexDebuggingFlag = false
}

// EnableMutexDebugging turns mutex debugging on.
func EnableMutexDebugging() {
	mutexDebuggingFlag = true
}

var activeLocks struct {
	sync.Mutex
	m map[*DebugMutex][]uintptr
}

func insert(m *DebugMutex) {
	if !mutexDebuggingFlag {
		return
	}

	r := make([]uintptr, _StackDepth)
	n := runtime.Callers(3, r)

	activeLocks.Lock()
	activeLocks.m[m] = r[:n]
	activeLocks.Unlock()
}

func remove(m *DebugMutex) {
	if !mutexDebuggingFlag {
		return
	}

	activeLocks.Lock()
	delete(activeLocks.m, m)
	activeLocks.Unlock()
}

func owner(l []uintptr) string {
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

// DumpOwned returns all currently locked mutexes.
func DumpOwned() []string {
	var r []string

	activeLocks.Lock()

	for m, l := range activeLocks.m {
		r = append(r, fmt.Sprintf("mutex @ %p\n%s", m, owner(l)))
	}

	activeLocks.Unlock()

	return r
}

func init() {
	activeLocks.m = make(map[*DebugMutex][]uintptr)
}
