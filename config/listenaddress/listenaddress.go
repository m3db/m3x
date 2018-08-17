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

// Package listenaddress provides a configuration struct for resolving
// a listen address from YAML.
package listenaddress

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultHostname = "0.0.0.0"
)

// Resolver is a type of port resolver
type Resolver string

const (
	// ConfigResolver resolves port using a value provided in config
	ConfigResolver Resolver = "config"
	// EnvironmentResolver resolves port using an environment variable
	// of which the name is provided in config
	EnvironmentResolver Resolver = "environment"
)

// Configuration is the configuration for resolving a listen address.
type Configuration struct {
	// PortType is the port type for the port
	PortType Resolver `yaml:"portType" validate:"nonzero"`

	// Value is the config specified listen address if using config port type.
	Value *string `yaml:"value"`

	// EnvListenPort specifies the environment variable name for the listen address port.
	EnvListenPort *string `yaml:"envListenPort"`

	// EnvListenPort specifies the environment variable name for the listen address hostname.
	EnvListenHost *string `yaml:"envListenHost"`
}

// Resolve returns the resolved listen address given the configuration.
func (c Configuration) Resolve() (string, error) {

	portType := c.PortType

	var listenAddress string
	switch portType {
	case ConfigResolver:
		if c.Value == nil {
			err := fmt.Errorf("missing port type using: resolver=%s",
				string(portType))
			return "", err
		}
		listenAddress = *c.Value

	case EnvironmentResolver:
		// environment variable for port is required
		if c.EnvListenPort == nil {
			err := fmt.Errorf("missing port env var name using: resolver=%s",
				string(portType))
			return "", err
		}
		portStr := os.Getenv(*c.EnvListenPort)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			err := fmt.Errorf("invalid port env var value using: resolver=%s, name=%s",
				string(portType), *c.EnvListenPort)
			return "", err
		}
		// if environment variable for hostname is not set, use the default
		if c.EnvListenHost == nil {
			listenAddress = fmt.Sprintf("%s:%d", defaultHostname, port)
		} else {
			envHost := os.Getenv(*c.EnvListenHost)
			listenAddress = fmt.Sprintf("%s:%d", envHost, port)
		}

	default:
		return "", fmt.Errorf("unknown port type: resolver=%s",
			string(portType))
	}

	return listenAddress, nil
}
