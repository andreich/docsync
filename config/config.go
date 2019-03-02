// Package config provides convenient ways of parsing Upload/Download/Sync
// configs by grouping them together, handling validation and parsing from
// JSON.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"time"
)

// Encryption holds the minimum configuration needed to configure
// crypt.Encryption.
type Encryption struct {
	AESPassphrase string `json:"aes_passphrase"`
}

// Duration is a nasty hack to go around serializing/deserializig duration from
// json.
type Duration struct {
	time.Duration
}

// UnmarshalJSON implements the json.Unmarshaler interface. The duration is
// expected to be a quoted-string of a duration in the format accepted by
// time.ParseDuration.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	tmp, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	d.Duration = tmp

	return nil
}

// Sync is the configuration for syncing a set of directories to cloud.
type Sync struct {
	Upload

	// Directory structure to sync, with keys being the local directories and
	// value being the directory in to upload the sync'ed files.
	Dirs               map[string]string
	Interval           Duration `json:"interval"`
	ManifestFile       string   `json:"manifest_file"`
	RemoteManifestFile string   `json:"remote_manifest_file"`

	Include []string
	Exclude []string
}

// Upload is the minimum configuration to upload files to cloud.
type Upload struct {
	Encryption
	Storage
}

// Storage is the minumum configuration to connect to the cloud.
type Storage struct {
	Credentials map[string]string `json:"credentials"`
	BucketName  string            `json:"bucket_name"`
}

// C provides the methods to be implemented by all configurations.
type C interface {
	// Validate performs the all checks of a configuration.
	// If a call to Validate returns nil, then the configuration should be fully
	// set up.
	Validate() error
	// Parse updates the configuration with values from the provided filename,
	// calling Validate() as well on the resulting configuration.
	Parse(filename string) error
}

// Validate satisfies interface C.
func (c *Encryption) Validate() error {
	if c.AESPassphrase == "" {
		return errors.New("aes_passphrase empty")
	}
	return nil
}

// Validate satisfies interface C.
func (c *Storage) Validate() error {
	if len(c.Credentials) == 0 {
		return errors.New("credentials empty")
	}
	for _, key := range []string{"type", "project_id", "private_key"} {
		if _, found := c.Credentials[key]; !found {
			return fmt.Errorf("key %q missing from credentials", key)
		}
	}
	return nil
}

// Validate satisfies interface C.
func (c *Sync) Validate() error {
	if len(c.Dirs) == 0 {
		return errors.New("dirs empty: at least one dir needs to be provided")
	}
	for e := range c.Dirs {
		fs, err := os.Stat(e)
		if err != nil {
			return fmt.Errorf("dirs entry %q invalid: %v", e, err)
		}
		if !fs.IsDir() {
			return fmt.Errorf("dirs entry %q invalid: file, not directory", e)
		}
	}
	if c.Interval.Duration < 10*time.Second {
		return errors.New("interval too small: at least 10 seconds")
	}
	if c.ManifestFile == "" {
		return errors.New("manifest_file empty")
	}
	if c.RemoteManifestFile == "" {
		return errors.New("remote_manifest_file empty")
	}
	for _, e := range c.Include {
		_, err := regexp.Compile(e)
		if err != nil {
			return fmt.Errorf("%q is not a valid regexp in include: %v", e, err)
		}
	}
	for _, e := range c.Exclude {
		_, err := regexp.Compile(e)
		if err != nil {
			return fmt.Errorf("%q is not a valid regexp in exclude: %v", e, err)
		}
	}
	return c.Upload.Validate()
}

// Validate satisfies interface C.
func (c *Upload) Validate() error {
	if err := c.Encryption.Validate(); err != nil {
		return err
	}
	if err := c.Storage.Validate(); err != nil {
		return err
	}
	return nil
}

// ReadFile is the normal ioutil.ReadFile, but having it like this enables
// testing configurations by faking the reading. See config_test.go.
var ReadFile = ioutil.ReadFile

// ParseConfig parses and validate a given filename in a configuration.
func ParseConfig(c C, filename string) error {
	contents, err := ReadFile(filename)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(contents, c); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}
	return nil
}

// Parse satisfies interface C.
func (c *Encryption) Parse(filename string) error {
	return ParseConfig(c, filename)
}

// Parse satisfies interface C.
func (c *Storage) Parse(filename string) error {
	return ParseConfig(c, filename)
}

// Parse satisfies interface C.
func (c *Sync) Parse(filename string) error {
	return ParseConfig(c, filename)
}

// Parse satisfies interface C.
func (c *Upload) Parse(filename string) error {
	return ParseConfig(c, filename)
}
