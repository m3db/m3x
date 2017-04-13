package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	goodConfig = `
listen_address: localhost:4385
buffer_space: 1024
servers:
    - server1:8090
    - server2:8010
`
)

type configuration struct {
	ListenAddress string   `yaml:"listen_address" validate:"nonzero"`
	BufferSpace   int      `yaml:"buffer_space" validate:"min=255"`
	Servers       []string `validate:"nonzero"`
}

func TestLoadWithInvalidFile(t *testing.T) {
	var cfg configuration

	// no file provided
	err := LoadFiles(&cfg)
	require.Error(t, err)
	require.Equal(t, errNoFilesToLoad, err)

	// non-exist file provided
	err = LoadFiles(&cfg, "./no-config.yaml")
	require.Error(t, err)

	fname := writeFile(t, goodConfig)
	defer os.Remove(fname)

	// invalid yaml file
	err = LoadFiles(&cfg, "./config.go")
	require.Error(t, err)

	// non-exist file in the file list
	err = LoadFiles(&cfg, fname, "./no-config.yaml")
	require.Error(t, err)

	// invalid file in the file list
	err = LoadFiles(&cfg, fname, "./config.go")
	require.Error(t, err)
}

func TestLoadFilesExtends(t *testing.T) {
	fname := writeFile(t, goodConfig)
	defer os.Remove(fname)

	partialConfig := `
buffer_space: 8080
servers:
    - server3:8080
    - server4:8080
`
	partial := writeFile(t, partialConfig)
	defer os.Remove(partial)

	var cfg configuration
	err := LoadFiles(&cfg, fname, partial)
	require.NoError(t, err)

	require.Equal(t, "localhost:4385", cfg.ListenAddress)
	require.Equal(t, 8080, cfg.BufferSpace)
	require.Equal(t, []string{"server3:8080", "server4:8080"}, cfg.Servers)
}

func TestLoadFilesValidateOnce(t *testing.T) {
	const invalidConfig1 = `
    listen_address:
    buffer_space: 256
    servers:
    `

	const invalidConfig2 = `
    listen_address: "localhost:8080"
    servers:
      - server2:8010
    `

	fname1 := writeFile(t, invalidConfig1)
	defer os.Remove(fname1)

	fname2 := writeFile(t, invalidConfig2)
	defer os.Remove(invalidConfig2)

	// Either config by itself will not pass validation.
	var cfg1 configuration
	err := LoadFiles(&cfg1, fname1)
	require.Error(t, err)

	var cfg2 configuration
	err = LoadFiles(&cfg2, fname2)
	require.Error(t, err)

	// But merging load has no error.
	var mergedCfg configuration
	err = LoadFiles(&mergedCfg, fname1, fname2)
	require.NoError(t, err)

	require.Equal(t, "localhost:8080", mergedCfg.ListenAddress)
	require.Equal(t, 256, mergedCfg.BufferSpace)
	require.Equal(t, []string{"server2:8010"}, mergedCfg.Servers)
}

func writeFile(t *testing.T, contents string) string {
	f, err := ioutil.TempFile("", "configtest")
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(contents))
	require.NoError(t, err)

	return f.Name()
}
