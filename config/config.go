package config

import (
	"errors"
	"io/ioutil"

	validator "gopkg.in/validator.v2"
	yaml "gopkg.in/yaml.v2"
)

// errNoFilesToLoad is return when you attemp to call LoadFiles with no file paths
var errNoFilesToLoad = errors.New("attempt to load configuration with no files")

// LoadFiles loads a list of files, deep-merging values.
// Validation is done after merging all values
func LoadFiles(config interface{}, fnames ...string) error {
	if len(fnames) == 0 {
		return errNoFilesToLoad
	}
	for _, fname := range fnames {
		data, err := ioutil.ReadFile(fname)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return err
		}
	}

	if err := validator.Validate(config); err != nil {
		return err
	}
	return nil
}
