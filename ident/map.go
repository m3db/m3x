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
	"bytes"

	"github.com/m3db/m3x/pool"

	"github.com/cheekybits/genny/generic"
)

const (
	offset64 = 14695981039346656037
	prime64  = 1099511628211
)

// ValueType is the generic value type for use with the specialized hash map.
type ValueType generic.Type

// Map uses the genny package to provide a generic map that can be specialized
// by running the following command from this directory:
// ```
// VALUE_TYPE=YourValueType go generate
// ```
// This will output to stdout the generated source file to use for your map.
// It uses linear probing by incrementing the number of the hash created when
// hashing the identifier if there is a collision. The hashing algorithm used is
// fnv hash.
// The map is a value type and not an interface to allow for less painful
// upgrades when adding/removing methods, it is not likely to need mocking so
// an interface would not be super useful either.
type Map struct {
	// lookup uses fnv hash of the identifier for the key and the value
	// wraps the value type and the ident (used to ensure lookup is correct
	// when dealing with collisions), we use fnv partially because lookups of
	// maps with uint64 keys has a fast path for Go.
	lookup map[MapHash]MapEntry

	hash func(MapKey) MapHash

	opts MapOptions
}

// Bytes implements MapKey, BytesMapKey is a simple implementation
// of an identifier that can be used with the identifier map.
func (k BytesMapKey) Bytes() []byte {
	return k
}

// Finalize implements MapKey, BytesMapKey is a simple implementation
// of an identifier that can be used with the identifier map.
func (k BytesMapKey) Finalize() {
}

// MapHash is the hash for a given map entry, this is public to support
// iterating over the map using a native Go for loop.
type MapHash uint64

// MapEntry is an entry in the map, this is public to support iterating
// over the map using a native Go for loop.
type MapEntry struct {
	// key is used to check equality on lookups to resolve collisions
	key mapKey
	// value type stored
	value ValueType
}

type mapKey struct {
	id       MapKey
	fromPool bool
	finalize bool
}

// Key returns the map entry key.
func (e MapEntry) Key() MapKey {
	return e.key.id
}

// KeyString returns the map entry key as a string.
func (e MapEntry) KeyString() string {
	return string(e.key.id.Bytes())
}

// Value returns the map entry value.
func (e MapEntry) Value() ValueType {
	return e.value
}

// MapOptions is a set of options used when creating an identifier map.
type MapOptions struct {
	// Size is the initial size of the map to create.
	Size int64
	// Pool is an underlying pool to use when taking copies of the ID
	// when using the default Set semantics.
	Pool pool.BytesPool
}

// NewMap returns a new identifier map.
func NewMap(opts MapOptions) *Map {
	var lookup map[MapHash]MapEntry
	if opts.Size > 0 {
		// Only use the constructor with specific size if set to let
		// go's default size be used if no specific size was set
		lookup = make(map[MapHash]MapEntry, opts.Size)
	} else {
		lookup = make(map[MapHash]MapEntry)
	}
	return &Map{
		lookup: lookup,
		hash:   hashFnv,
		opts:   opts,
	}
}

func (m *Map) newMapKey(id MapKey, opts mapKeyOptions) mapKey {
	key := mapKey{id: id, finalize: opts.finalizeKey}
	if !opts.copyKey {
		return key
	}

	bytes := id.Bytes()
	if m.opts.Pool == nil {
		key.id = BytesMapKey(append([]byte(nil), bytes...))
		return key
	}

	keyLen := len(bytes)
	result := m.opts.Pool.Get(keyLen)[:keyLen]
	copy(result, bytes)

	key.fromPool = true
	key.id = BytesMapKey(result)
	return key
}

func (m *Map) releaseMapKey(key mapKey) {
	if key.fromPool {
		m.opts.Pool.Put(key.id.Bytes())
	} else if key.finalize {
		key.id.Finalize()
	}
}

// Get returns a value in the map for an identifier if found.
func (m *Map) Get(id MapKey) (ValueType, bool) {
	hash := m.hash(id)
	for v, ok := m.lookup[hash]; ok; v, ok = m.lookup[hash] {
		if bytes.Equal(v.key.id.Bytes(), id.Bytes()) {
			return v.value, true
		}
		// Linear probe to "next" to this entry (really a rehash)
		hash++
	}
	var empty ValueType
	return empty, false
}

// Set will set the value for an identifier.
func (m *Map) Set(id MapKey, value ValueType) {
	m.set(id, value, mapKeyOptions{
		copyKey: true,
		// NB(r): Only finalize the key if we're not creating copy from
		// internal pool, otherwise we explicitly return it to the pool
		// instead of finalizing it.
		finalizeKey: m.opts.Pool == nil,
	})
}

// SetNoCopyKey will set the value for an identifier but will
// avoid copying the identifier, it will finalize the key if/when evicted.
func (m *Map) SetNoCopyKey(id MapKey, value ValueType) {
	m.set(id, value, mapKeyOptions{
		copyKey:     false,
		finalizeKey: true,
	})
}

// SetNoCopyNoFinalizeKey will set the value for an identifier but will
// avoid copying the identifier and will not finalize the key if/when evicted.
func (m *Map) SetNoCopyNoFinalizeKey(id MapKey, value ValueType) {
	m.set(id, value, mapKeyOptions{
		copyKey:     false,
		finalizeKey: false,
	})
}

type mapKeyOptions struct {
	copyKey     bool
	finalizeKey bool
}

func (m *Map) set(id MapKey, value ValueType, opts mapKeyOptions) {
	hash := m.hash(id)
	for v, ok := m.lookup[hash]; ok; v, ok = m.lookup[hash] {
		if bytes.Equal(v.key.id.Bytes(), id.Bytes()) {
			m.lookup[hash] = MapEntry{
				key:   v.key,
				value: value,
			}
			return
		}
		// Linear probe to "next" to this entry (really a rehash)
		hash++
	}

	key := m.newMapKey(id, opts)
	m.lookup[hash] = MapEntry{
		key:   key,
		value: value,
	}
}

// Iter provides the underlying map to allow for using a native Go for loop
// to iterate the map, however callers should only ever read and not write
// the map.
func (m *Map) Iter() map[MapHash]MapEntry {
	return m.lookup
}

// Len returns the number of map entries in the map.
func (m *Map) Len() int {
	return len(m.lookup)
}

// Has returns true if value exists for key, false otherwise, it is
// shorthand for a call to Get that doesn't return the value.
func (m *Map) Has(id MapKey) bool {
	_, ok := m.Get(id)
	return ok
}

// Delete will remove a value set in the map for the specified key.
func (m *Map) Delete(id MapKey) {
	hash := m.hash(id)
	for v, ok := m.lookup[hash]; ok; v, ok = m.lookup[hash] {
		if bytes.Equal(v.key.id.Bytes(), id.Bytes()) {
			delete(m.lookup, hash)
			m.releaseMapKey(v.key)
			return
		}
		// Linear probe to "next" to this entry (really a rehash)
		hash++
	}
}

// Reset will reset the map by simply deleting all keys to avoid
// allocating a new map.
func (m *Map) Reset() {
	for hash, v := range m.lookup {
		delete(m.lookup, hash)
		m.releaseMapKey(v.key)
	}
}

// ResetByReallocateMap will avoid deleting all keys and reallocate
// a new map, this is useful if you believe you have a larage map and
// will not need to grow back to a similar size.
func (m *Map) ResetByReallocateMap() {
	if m.opts.Size > 0 {
		m.lookup = make(map[MapHash]MapEntry, m.opts.Size)
	} else {
		m.lookup = make(map[MapHash]MapEntry)
	}
}

// inline alloc-free fnv-1a hash
func hashFnv(key MapKey) MapHash {
	var v MapHash = offset64
	for _, c := range key.Bytes() {
		v ^= MapHash(c)
		v *= prime64
	}
	return v
}
