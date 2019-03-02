package mover

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/andreich/docsync/config"
)

// RuleConfig is the mover configuration: which patterns should be moved to what
// directory.
type RuleConfig struct {
	// Patterns to be compiled to regexp and matched on the contents of
	// the observed files.
	Patterns       []string `json:"patterns"`
	PatternsRegexp []*regexp.Regexp
	// To which directory should files whose content match the above
	// patterns be moved.
	To string `json:"to"`
}

var osStat = os.Stat

// Validate satisfies the config.Config interface.
func (m *RuleConfig) Validate() error {
	m.PatternsRegexp = nil
	if len(m.Patterns) == 0 {
		return fmt.Errorf("%+v: at least one pattern is required", m)
	}
	for _, str := range m.Patterns {
		re, err := regexp.Compile(str)
		if err != nil {
			return fmt.Errorf("mover string %q is not a valid regexp: %v", str, err)
		}
		m.PatternsRegexp = append(m.PatternsRegexp, re)
	}

	if m.To == "" {
		return fmt.Errorf("%+v: to is required", m)
	}
	st, err := osStat(m.To)
	if os.IsNotExist(err) {
		err = os.MkdirAll(m.To, os.ModeDir|0744)
		if err != nil {
			return fmt.Errorf("%q in mover: tried to create: %v", m.To, err)
		}
		st, err = osStat(m.To)
	}
	if err != nil {
		return fmt.Errorf("%q in mover: %v", m.To, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("%q in mover is not a directory", m.To)
	}
	return nil
}

// Config is the mover configuration
type Config struct {
	// From is the list of directories to be scanned and from which files
	// would be moved.
	From []string `json:"from"`
	// Rules is the list of rules to be applied on the files content.
	// Rule order matters: first rule matching a file defines where that
	// file is moved and no further rules are evaluated.
	Rules []*RuleConfig `json:"rules"`
}

// Validate satisfies the config.Config interface.
func (m *Config) Validate() error {
	if len(m.From) == 0 {
		return errors.New("nothing specified in from field")
	}
	for _, dir := range m.From {
		st, err := osStat(dir)
		if err != nil {
			return fmt.Errorf("%q in mover: %v", dir, err)
		}
		if !st.IsDir() {
			return fmt.Errorf("%q in mover is not a directory", dir)
		}
	}
	if len(m.Rules) == 0 {
		return errors.New("nothing specified in rules field")
	}
	for _, move := range m.Rules {
		if err := move.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// EmbeddedConfig is the configuration to use if the mover configuration is embedded
// in a larger configuration structure.
type EmbeddedConfig struct {
	Mover *Config `json:"mover"`
}

// Parse satisfies the config.Config interface.
func (c *EmbeddedConfig) Parse(filename string) error {
	return config.ParseConfig(c, filename)
}

// Validate satisfies the config.Config interface.
func (c *EmbeddedConfig) Validate() error {
	if c.Mover == nil {
		return errors.New("mover not present")
	}
	return c.Mover.Validate()
}
