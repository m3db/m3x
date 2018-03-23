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
	"unsafe"

	"github.com/golang/mock/gomock"

	"github.com/m3db/m3x/pool"
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
		require.Equal(t, strMap[entry.KeyString()], entry.Value().(string))
	}
	require.Equal(t, len(strMap), iterated)
}

func TestMapCollision(t *testing.T) {
	m := NewMap(MapOptions{})
	// Always collide
	m.hash = func(_ MapKey) MapHash { return 0 }

	// Insert foo, ensure set at fake hash
	m.Set(StringID("foo"), "a")

	v, ok := m.lookup[0]
	assert.True(t, ok)
	assert.Equal(t, "foo", v.KeyString())
	assert.Equal(t, "a", v.value.(string))

	// Insert bar, ensure collides and both next to each other
	m.Set(StringID("bar"), "b")

	v, ok = m.lookup[0]
	assert.True(t, ok)
	assert.Equal(t, "foo", v.KeyString())
	assert.Equal(t, "a", v.value.(string))

	v, ok = m.lookup[1]
	assert.True(t, ok)
	assert.Equal(t, "bar", v.KeyString())
	assert.Equal(t, "b", v.value.(string))

	// Ensure set for the colliding key works
	m.Set(StringID("bar"), "c")

	v, ok = m.lookup[0]
	assert.True(t, ok)
	assert.Equal(t, "foo", v.KeyString())
	assert.Equal(t, "a", v.value.(string))

	v, ok = m.lookup[1]
	assert.True(t, ok)
	assert.Equal(t, "bar", v.KeyString())
	assert.Equal(t, "c", v.value.(string))
}

func TestMapWithSize(t *testing.T) {
	m := NewMap(MapOptions{Size: 42})
	m.Set(StringID("foo"), "bar")
	v, ok := m.Get(StringID("foo"))
	require.True(t, ok)
	require.Equal(t, "bar", v.(string))
}

func TestMapWithPooling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	key := BytesMapKey([]byte("foo"))
	value := ValueType("a")

	pool := pool.NewMockBytesPool(ctrl)

	m := NewMap(MapOptions{Pool: pool})

	mockPooledSlice := make([]byte, 0, 3)
	pool.EXPECT().Get(len(key.Bytes())).Return(mockPooledSlice)
	m.Set(key, value)
	require.Equal(t, 1, m.Len())

	// Now ensure that the key is from the pool and not our original key
	for _, entry := range m.Iter() {
		type slice struct {
			array unsafe.Pointer
			len   int
			cap   int
		}

		keyBytes := entry.Key().Bytes()

		rawPooledSlice := (*slice)(unsafe.Pointer(&mockPooledSlice))
		rawKeySlice := (*slice)(unsafe.Pointer(&keyBytes))

		require.True(t, rawPooledSlice.array == rawKeySlice.array)
	}

	// Now delete the key to simulate returning to pool
	pool.EXPECT().Put(mockPooledSlice[:3])
	m.Delete(key)
	require.Equal(t, 0, m.Len())
}
