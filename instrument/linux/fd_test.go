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

// +build linux

package linux

// We explicitly set build tags for linux to only test this on linux.

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/m3db/m3x/instrument"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
)

func TestFileDescriptorsReporter(t *testing.T) {
	scope := tally.NewTestScope("", nil)
	iopts := instrument.NewOptions().SetMetricsScope(scope)

	countOpen := 32
	for i := 0; i < countOpen; i++ {
		tmpFile, err := ioutil.TempFile("", "example")
		if err != nil {
			require.FailNow(t, fmt.Sprintf("could not open temp file: %v", err))
		}
		defer os.Remove(tmpFile.Name())
	}

	r := NewFileDescriptorsReporter(iopts)

	every := 10 * time.Millisecond

	require.NoError(t, r.Start(every))

	time.Sleep(2 * every)

	v, ok := scope.Snapshot().Gauges()["process.fds+platform=linux"]
	if !ok {
		require.FailNow(t, "metric for fds not found after waiting 2x interval")
	}

	assert.True(t, v.Value() >= float64(countOpen))

	require.NoError(t, r.Stop())
}

func TestFileDescriptorsReporterDoesNotOpenMoreThanOnce(t *testing.T) {
	r := NewFileDescriptorsReporter(instrument.NewOptions())
	assert.NoError(t, r.Start(10*time.Millisecond))
	assert.Error(t, r.Start(10*time.Millisecond))
	assert.NoError(t, r.Stop())
}

func TestFileDescriptorsReporterDoesNotCloseMoreThanOnce(t *testing.T) {
	r := NewFileDescriptorsReporter(instrument.NewOptions())
	assert.NoError(t, r.Start(10*time.Millisecond))
	assert.NoError(t, r.Stop())
	assert.Error(t, r.Stop())
}

func TestFileDescriptorsReporterDoesNotOpenWithInvalidReportInterval(t *testing.T) {
	r := NewFileDescriptorsReporter(instrument.NewOptions())
	assert.Error(t, r.Start(0))
}
