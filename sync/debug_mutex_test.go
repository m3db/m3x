package xsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugMutex(t *testing.T) {
	EnableMutexDebugging()
	defer DisableMutexDebugging()

	m := &DebugMutex{}

	assert.Empty(t, DumpOwned())

	m.Lock()
	assert.NotEmpty(t, DumpOwned())

	m.Unlock()
	assert.Empty(t, DumpOwned())

	m.RLock()
	assert.Empty(t, DumpOwned())

	m.RUnlock()
	assert.Empty(t, DumpOwned())
}
