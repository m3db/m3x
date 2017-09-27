// Copyright (c) 2017 Uber Technologies, Inc.
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

package id

import (
	"math/rand"
	"testing"
	"time"
)

const (
	testShortIDSize  = 8
	testMediumIDSize = 200
	testLongIDSize   = 8192
)

var (
	testShortID, testMediumID, testLongID []byte
)

func BenchmarkMD5ShortID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashFn(testShortID)
	}
}

func BenchmarkMD5MediumID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashFn(testMediumID)
	}
}

func BenchmarkMD5LongID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashFn(testLongID)
	}
}

func BenchmarkMurmur3Hash128ShortID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Murmur3Hash128(testShortID)
	}
}

func BenchmarkMurmur3Hash128MediumID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Murmur3Hash128(testMediumID)
	}
}

func BenchmarkMurmur3Hash128LongID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Murmur3Hash128(testLongID)
	}
}

func generateTestID(s *rand.Rand, size int) []byte {
	id := make([]byte, size)
	for i := 0; i < size; i++ {
		id[i] = byte(s.Int63() % 256)
	}
	return id
}

func init() {
	s := rand.New(rand.NewSource(time.Now().UnixNano()))
	testShortID = generateTestID(s, testShortIDSize)
	testMediumID = generateTestID(s, testMediumIDSize)
	testLongID = generateTestID(s, testLongIDSize)
}
