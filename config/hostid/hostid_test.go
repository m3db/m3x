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

package hostid

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostnameResolver(t *testing.T) {
	cfg := Configuration{
		Resolver: "hostname",
	}

	value, err := cfg.Resolve()
	require.NoError(t, err)

	expected, err := os.Hostname()
	require.NoError(t, err)

	assert.Equal(t, expected, value)
}

func TestConfigResolver(t *testing.T) {
	expected := "foo"

	cfg := Configuration{
		Resolver: "config",
		Value:    &expected,
	}

	value, err := cfg.Resolve()
	require.NoError(t, err)

	assert.Equal(t, expected, value)
}

func TestConfigResolverErrorWhenMissing(t *testing.T) {
	cfg := Configuration{
		Resolver: "config",
	}

	_, err := cfg.Resolve()
	require.Error(t, err)
}

func TestEnvironmentVariableResolver(t *testing.T) {
	varName := "HOST_ENV_NAME_" + strconv.Itoa(int(time.Now().UnixNano()))
	expected := "foo"

	require.NoError(t, os.Setenv(varName, expected))

	cfg := Configuration{
		Resolver:   "environment",
		EnvVarName: &varName,
	}

	value, err := cfg.Resolve()
	require.NoError(t, err)

	assert.Equal(t, expected, value)
}

func TestEnvironmentResolverErrorWhenNameMissing(t *testing.T) {
	cfg := Configuration{
		Resolver: "environment",
	}

	_, err := cfg.Resolve()
	require.Error(t, err)
}

func TestEnvironmentResolverErrorWhenValueMissing(t *testing.T) {
	varName := "HOST_ENV_NAME_" + strconv.Itoa(int(time.Now().UnixNano()))

	cfg := Configuration{
		Resolver:   "environment",
		EnvVarName: &varName,
	}

	_, err := cfg.Resolve()
	require.Error(t, err)
}

func TestKeyValueFileResolver(t *testing.T) {
	cfg := Configuration{
		Resolver: "kvFile",
		KVFile: &KVFileConfig{
			Path: "foobarbaz",
			Key:  "foo",
		},
	}

	_, err := cfg.Resolve()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file"), "expected error to be due to file not found")
}

func TestUnknownResolverError(t *testing.T) {
	cfg := Configuration{
		Resolver: "some-unknown-type",
	}

	_, err := cfg.Resolve()
	require.Error(t, err)
}

func TestKVFileConfig(t *testing.T) {
	c := &kvFile{
		path: "foobarpath",
	}

	_, err := c.ID()
	assert.Error(t, err)

	f, err := ioutil.TempFile("", "hostid-test")
	require.NoError(t, err)

	defer os.Remove(f.Name())

	_, err = f.WriteString("k1=v1")
	require.NoError(t, err)

	c = &kvFile{
		path: f.Name(),
		key:  "k1",
	}

	v, err := c.ID()
	assert.NoError(t, err)
	assert.Equal(t, "v1", v)
}

func TestKVFileConfig_Timeout(t *testing.T) {
	f, err := ioutil.TempFile("", "hostid-test")
	require.NoError(t, err)

	defer os.Remove(f.Name())

	c := &kvFile{
		path:     f.Name(),
		key:      "k1",
		interval: 10 * time.Millisecond,
		timeout:  time.Minute,
	}

	valC := make(chan string)

	go func() {
		v, err := c.ID()
		if err == nil {
			valC <- v
			close(valC)
		}
	}()

	time.Sleep(time.Second)
	_, err = f.WriteString("k1=v1")
	require.NoError(t, err)

	select {
	case v := <-valC:
		assert.Equal(t, "v1", v)
	case <-time.After(5 * time.Second):
		t.Fatal("expected to see value within 5s")
	}
}

func TestKVFileConfig_TimeoutErr(t *testing.T) {
	f, err := ioutil.TempFile("", "hostid-test")
	require.NoError(t, err)

	defer os.Remove(f.Name())

	c := &kvFile{
		path:     f.Name(),
		key:      "k1",
		interval: 10 * time.Millisecond,
		timeout:  time.Second,
	}

	_, err = c.ID()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "within 1s"))
}

func TestParseKV(t *testing.T) {
	for _, test := range []struct {
		content string
		key     string
		exp     string
		expErr  bool
	}{
		{
			content: `k1=v1
k2=v2`,
			key: "k2",
			exp: "v2",
		},
		{
			content: `k1=v1
k2=v2=foo=bar`,
			key: "k2",
			exp: "v2=foo=bar",
		},
		{
			content: `k1=v1
   k2=v2=foo=bar  `,
			key: "k2",
			exp: "v2=foo=bar",
		},
		{
			content: `k1=v1
k2=v2`,
			key:    "k3",
			expErr: true,
		},
	} {
		c := &kvFile{
			key: test.key,
		}
		r := strings.NewReader(test.content)

		val, err := c.parseKV(r)
		if test.expErr {
			assert.Error(t, err)
			continue
		}

		assert.NoError(t, err)
		assert.Equal(t, test.exp, val)
	}
}
