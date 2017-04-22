package xsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugMutex(t *testing.T) {
	EnableMutexDebugging()
	defer DisableMutexDebugging()

	m := &DebugRWMutex{}

	assert.Empty(t, DumpLocks(0))

	m.Lock()
	assert.NotEmpty(t, DumpLocks(0))

	m.Unlock()
	assert.Empty(t, DumpLocks(0))

	m.RLock()
	assert.Empty(t, DumpLocks(0))

	m.RUnlock()
	assert.Empty(t, DumpLocks(0))
}

func TestDebugMutexContention(t *testing.T) {
	SetMutexContentionTrigger(2)

	EnableMutexDebugging()
	defer DisableMutexDebugging()

	m := &DebugRWMutex{}

	m.RLock()
	m.RLock()

	assert.Panics(t, func() {
		m.Lock()
	})
}
