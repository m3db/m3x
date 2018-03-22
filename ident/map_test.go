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

package ident

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapGet(t *testing.T) {
	m := NewMap(MapOptions{})

	_, ok := m.Get(StringID("foo"))
	require.False(t, ok)

	m.Set(StringID("foo"), "bar")

	v, ok := m.Get(StringID("foo"))
	require.True(t, ok)
	require.Equal(t, "bar", v.(string))
}

func TestMapSetOverwrite(t *testing.T) {
	m := NewMap(MapOptions{})

	_, ok := m.Get(StringID("foo"))
	require.False(t, ok)

	m.Set(StringID("foo"), "bar")
	m.Set(StringID("foo"), "baz")
	m.Set(StringID("foo"), "qux")

	v, ok := m.Get(StringID("foo"))
	require.True(t, ok)
	require.Equal(t, "qux", v.(string))
}

func TestMapHas(t *testing.T) {
	m := NewMap(MapOptions{})

	exists := m.Has(StringID("foo"))
	require.False(t, exists)

	m.Set(StringID("foo"), "bar")

	exists = m.Has(StringID("foo"))
	require.True(t, exists)
}

func TestMapDelete(t *testing.T) {
	m := NewMap(MapOptions{})

	m.Set(StringID("foo"), "a")
	m.Set(StringID("bar"), "b")
	m.Set(StringID("baz"), "c")
	require.Equal(t, 3, m.Len())

	m.Delete(StringID("foo"))
	require.Equal(t, 2, m.Len())

	m.Delete(StringID("keyThatDoesNotExist"))
	require.Equal(t, 2, m.Len())
}

func TestMapIter(t *testing.T) {
	m := NewMap(MapOptions{})

	strMap := make(map[string]string)
	set := func(k, v string) {
		m.Set(StringID(k), v)
		strMap[k] = v
	}

	set("foo", "a")
	set("bar", "b")
	set("baz", "c")

	iterated := 0
	for _, entry := range m.Iter() {
		iterated++
		require.Equal(t, strMap[entry.Key().String()], entry.Value().(string))
	}
	require.Equal(t, len(strMap), iterated)
}

func TestMapCollision(t *testing.T) {
	m := NewMap(MapOptions{})
	// Always collide
	m.hash = func(_ ID) MapHash { return 0 }

	// Insert foo, ensure set at fake hash
	m.Set(StringID("foo"), "a")

	v, ok := m.lookup[0]
	assert.True(t, ok)
	assert.Equal(t, "foo", v.key.id.String())
	assert.Equal(t, "a", v.value.(string))

	// Insert bar, ensure collides and both next to each other
	m.Set(StringID("bar"), "b")

	v, ok = m.lookup[0]
	assert.True(t, ok)
	assert.Equal(t, "foo", v.key.id.String())
	assert.Equal(t, "a", v.value.(string))

	v, ok = m.lookup[1]
	assert.True(t, ok)
	assert.Equal(t, "bar", v.key.id.String())
	assert.Equal(t, "b", v.value.(string))

	// Ensure set for the colliding key works
	m.Set(StringID("bar"), "c")

	v, ok = m.lookup[0]
	assert.True(t, ok)
	assert.Equal(t, "foo", v.key.id.String())
	assert.Equal(t, "a", v.value.(string))

	v, ok = m.lookup[1]
	assert.True(t, ok)
	assert.Equal(t, "bar", v.key.id.String())
	assert.Equal(t, "c", v.value.(string))
}
