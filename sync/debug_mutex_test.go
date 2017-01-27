package xsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugMutex(t *testing.T) {
	EnableMutexDebugging()
	defer DisableMutexDebugging()

	m := &DebugRWMutex{}

	assert.Empty(t, DumpLocks())

	m.Lock()
	assert.NotEmpty(t, DumpLocks())

	m.Unlock()
	assert.Empty(t, DumpLocks())

	m.RLock()
	assert.Empty(t, DumpLocks())

	m.RUnlock()
	assert.Empty(t, DumpLocks())
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
