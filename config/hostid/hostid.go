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

// Package hostid provides a configuration struct for resolving
// a host ID from YAML.
package hostid

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	defaultFileCheckInterval = 10 * time.Second
)

// Resolver is a type of host ID resolver
type Resolver string

const (
	// HostnameResolver resolves host using the hostname returned by OS
	HostnameResolver Resolver = "hostname"
	// ConfigResolver resolves host using a value provided in config
	ConfigResolver Resolver = "config"
	// EnvironmentResolver resolves host using an environment variable
	// of which the name is provided in config
	EnvironmentResolver Resolver = "environment"
	// KeyValueFileResolver parses a file of key-value pairs (formatted
	// `KEY=VALUE`) and resolves host ID from a given key.
	KeyValueFileResolver Resolver = "kvFile"
)

// IDResolver represents a method of resolving host identity.
type IDResolver interface {
	ID() (string, error)
}

// Configuration is the configuration for resolving a host ID.
type Configuration struct {
	// Resolver is the resolver for the host ID.
	Resolver Resolver `yaml:"resolver"`

	// Value is the config specified host ID if using config host ID resolver.
	Value *string `yaml:"value"`

	// EnvVarName is the environment specified host ID if using environment host ID resolver.
	EnvVarName *string `yaml:"envVarName"`

	// KVFile is the kv file config.
	KVFile *KVFileConfig `yaml:"kvFile"`
}

// KVFileConfig contains the info needed to construct a KeyValueFileResolver.
type KVFileConfig struct {
	// Path of the file containing the KV config.
	Path string `yaml:"path"`

	// Key to search for in the file.
	Key string `yaml:"key"`

	// Timeout to wait to find the key in the file.
	Timeout time.Duration `yaml:"timeout"`
}

func (c Configuration) resolver() (IDResolver, error) {
	switch c.Resolver {
	case HostnameResolver:
		return hostnameResolver{}, nil
	case ConfigResolver:
		return &configResolver{value: c.Value}, nil
	case EnvironmentResolver:
		return &environmentResolver{envVarName: c.EnvVarName}, nil
	case KeyValueFileResolver:
		if c.KVFile == nil {
			return nil, errors.New("KV file config cannot be nil")
		}
		return &kvFile{
			key:     c.KVFile.Key,
			path:    c.KVFile.Path,
			timeout: c.KVFile.Timeout,
		}, nil
	}
	return nil, fmt.Errorf("unknown host ID resolver: resolver=%s",
		string(c.Resolver))
}

// Resolve returns the resolved host ID given the configuration.
func (c Configuration) Resolve() (string, error) {
	r, err := c.resolver()
	if err != nil {
		return "", err
	}
	return r.ID()
}

type hostnameResolver struct{}

func (hostnameResolver) ID() (string, error) {
	return os.Hostname()
}

type configResolver struct {
	value *string
}

func (c *configResolver) ID() (string, error) {
	if c.value == nil {
		err := fmt.Errorf("missing host ID using: resolver=%s", string(ConfigResolver))
		return "", err
	}
	return *c.value, nil
}

type environmentResolver struct {
	envVarName *string
}

func (c *environmentResolver) ID() (string, error) {
	if c.envVarName == nil {
		err := fmt.Errorf("missing host ID env var name using: resolver=%s",
			string(EnvironmentResolver))
		return "", err
	}
	v := os.Getenv(*c.envVarName)
	if v == "" {
		err := fmt.Errorf("missing host ID env var value using: resolver=%s, name=%s",
			string(EnvironmentResolver), *c.envVarName)
		return "", err
	}
	return v, nil
}

type kvFile struct {
	path     string
	key      string
	interval time.Duration
	timeout  time.Duration
}

// ID attempts to parse an ID from a KV config file. It will optionally wait a
// timeout to find the value, as in some environments the file may be
// dynamically generated from external metadata and not immediately available
// when the instance starts up.
func (c *kvFile) ID() (string, error) {
	checkF := func() (string, error) {
		f, err := os.Open(c.path)
		if err != nil {
			return "", err
		}

		return c.parseKV(f)
	}

	if c.timeout == 0 {
		return checkF()
	}

	interval := c.interval
	if interval == 0 {
		interval = defaultFileCheckInterval
	}

	startT := time.Now()
	for time.Since(startT) < c.timeout {
		v, err := checkF()
		if err == nil {
			return v, nil
		}

		time.Sleep(interval)
	}

	return "", fmt.Errorf("did not find value in %s within %s", c.path, c.timeout)
}

func (c *kvFile) parseKV(r io.Reader) (string, error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		l := strings.Trim(s.Text(), " ")
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			continue
		}

		if parts[0] == c.key {
			return parts[1], nil
		}
	}

	if err := s.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("did not find key '%s' in %s", c.key, c.path)
}
