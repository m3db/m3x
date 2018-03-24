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

package fnv

import (
	stdfnv "hash/fnv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHash1a(t *testing.T) {
	tests := []struct {
		in string
	}{
		{""},
		{"a"},
		{"ab"},
		{"abc"},
		{"the quick brown fox jumps over the lazy dog"},
		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit"},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			actual := Hash64a([]byte(test.in))

			h := stdfnv.New64a()
			_, err := h.Write([]byte(test.in))
			require.NoError(t, err)
			expected := h.Sum64()

			assert.Equal(t, actual, expected)
		})
	}
}
