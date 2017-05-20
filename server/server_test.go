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

package xserver

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m3db/m3x/retry"
	"github.com/stretchr/testify/require"
)

const (
	testListenAddress = "127.0.0.1:0"
)

func testServer(addr string) (*server, *mockHandler, *int32, *int32) {
	var (
		numAdded   int32
		numRemoved int32
	)

	opts := NewOptions().SetRetrier(xretry.NewRetrier(xretry.NewOptions().SetMaxRetries(2)))
	opts = opts.SetInstrumentOptions(opts.InstrumentOptions().SetReportInterval(time.Second))

	h := newMockHandler()
	s := NewServer(addr, h, opts).(*server)

	s.addConnectionFn = func(conn net.Conn) bool {
		atomic.AddInt32(&numAdded, 1)
		ret := s.addConnection(conn)
		return ret
	}

	s.removeConnectionFn = func(conn net.Conn) {
		atomic.AddInt32(&numRemoved, 1)
		s.removeConnection(conn)
	}

	return s, h, &numAdded, &numRemoved
}

func TestServerListenAndClose(t *testing.T) {
	s, h, numAdded, numRemoved := testServer(testListenAddress)

	numClients := 9

	err := s.ListenAndServe()
	require.NoError(t, err)
	listenAddr := s.listener.Addr().String()

	var expectedRes []interface{}

	for i := 1; i <= numClients; i++ {
		conn, err := net.Dial("tcp", listenAddr)
		require.NoError(t, err)

		send := []byte(fmt.Sprintf("%d", i))
		expectedRes = append(expectedRes, send)
		_, err = conn.Write(send)
		require.NoError(t, err)
	}

	for {
		handleCalled, res := h.res()
		if handleCalled != numClients {
			time.Sleep(50 * time.Microsecond)
			continue
		}

		require.Equal(t, expectedRes, res)
		break
	}

	s.Close()

	require.Equal(t, int32(numClients), atomic.LoadInt32(numAdded))
	require.Equal(t, int32(numClients), atomic.LoadInt32(numRemoved))
}

type mockHandler struct {
	sync.Mutex

	called             int
	received           []interface{}
	handleConnectionFn func(conn net.Conn)
}

func newMockHandler() *mockHandler { return &mockHandler{} }

func (h *mockHandler) Handle(conn net.Conn) {
	h.Lock()
	b := make([]byte, 16)

	n, _ := conn.Read(b)
	h.called++
	h.received = append(h.received, b[:n])
	h.Unlock()
}

func (h *mockHandler) res() (int, []interface{}) {
	h.Lock()
	defer h.Unlock()

	return h.called, h.received
}
