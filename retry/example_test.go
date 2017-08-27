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

package xretry_test

import (
	"context"
	"log"
	"net/http"

	"github.com/m3db/m3x/retry"
)

func ExampleRetrier() {
	var (
		opts    = xretry.NewOptions()
		retrier = xretry.NewRetrier(opts)
		context = context.Background()
	)

	continueFn := func(attempt int) bool {
		// Check if the context has been canceled.
		select {
		case <-context.Done():
			return false
		default:
			return true
		}
	}

	fn := func() error {
		_, err := http.Get("http://example.com/")
		return err
	}

	if err := retrier.AttemptWhile(continueFn, fn); err != nil {
		log.Fatal(err)
	}
}