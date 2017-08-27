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

package xsync_test

import (
	"log"
	"net/http"
	"sync"

	"github.com/m3db/m3x/sync"
)

func ExampleWorkerPool() {
	var (
		wg          sync.WaitGroup
		workers     = xsync.NewWorkerPool(3)
		errorCh     = make(chan error, 1)
		numRequests = 9
		responses   = make([]*http.Response, numRequests)
	)

	wg.Add(numRequests)
	workers.Init()

	for i := 0; i < numRequests; i++ {
		// Capture loop variable.
		i := i

		// Execute request on worker pool.
		workers.Go(func() {
			defer wg.Done()

			resp, err := http.Get("http://example.com/")
			if err != nil {
				// Return the first error that is encountered.
				select {
				case errorCh <- err:
				default:
				}

				return
			}

			// Can concurrently modify responses since each iteration updates a
			// different index.
			responses[i] = resp
		})

		_ = i
	}

	// Wait for all requests to finish.
	wg.Wait()

	close(errorCh)
	if err := <-errorCh; err != nil {
		log.Fatal(err)
	}
}
