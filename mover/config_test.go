package mover

import (
	"errors"
	"strings"
	"testing"

	"github.com/andreich/docsync/config"
	"github.com/google/uuid"
)

func TestConfig(t *testing.T) {
	oldReadFile := config.ReadFile
	defer func() { config.ReadFile = oldReadFile }()

	for _, tc := range []struct {
		desc string

		readContent string
		readError   error
		hasErr      bool
		errContains string
	}{{
		desc:      "missing file",
		readError: errors.New("missing"),
		hasErr:    true,
	}, {
		desc:        "no mover",
		readContent: "{}",
		hasErr:      true,
		errContains: "mover",
	}, {
		desc:        "no from",
		readContent: `{"mover": {}}`,
		hasErr:      true,
		errContains: "from field",
	}, {
		desc:        "from invalid - not existent",
		readContent: `{"mover": {"from": ["/this/directory/does/not/exist"]}}`,
		hasErr:      true,
		errContains: "no such file",
	}, {
		desc:        "from invalid - not a directory",
		readContent: `{"mover": {"from": ["/dev/random"]}}`,
		hasErr:      true,
		errContains: "is not a directory",
	}, {
		desc:        "no rules",
		readContent: `{"mover": {"from": ["."]}}`,
		hasErr:      true,
		errContains: "rules",
	}, {
		desc: "rule without pattern",
		readContent: `
{
    "mover": {
        "from": [
            "."
        ],
        "rules": [
            {
                "to": "/a/nice/place"
            }
        ]
    }
}`,
		hasErr:      true,
		errContains: "pattern",
	}, {
		desc: "rule with invalid pattern",
		readContent: `
{
    "mover": {
        "from": [
            "."
        ],
        "rules": [
            {
	    	"patterns": ["this should[ be rejected"],
                "to": "/a/nice/place"
            }
        ]
    }
}`,
		hasErr:      true,
		errContains: "regexp",
	}, {
		desc: "rule without to",
		readContent: `
{
    "mover": {
        "from": [
            "."
        ],
        "rules": [
            {
	    	"patterns": ["this should be in the text"]
            }
        ]
    }
}`,
		hasErr:      true,
		errContains: "to is required",
	}, {
		desc: "rule failing to create",
		readContent: `
{
    "mover": {
        "from": [
            "."
        ],
        "rules": [
            {
	    	"patterns": ["this should be in the text"],
		"to": "/etc/this/should/fail/to/create"
            }
        ]
    }
}`,
		hasErr:      true,
		errContains: "tried to create",
	}, {
		desc: "rule failing to stat",
		readContent: `
{
    "mover": {
        "from": [
            "."
        ],
        "rules": [
            {
	    	"patterns": ["this should be in the text"],
		"to": "/root/this/should/really/fail"
            }
        ]
    }
}`,
		hasErr:      true,
		errContains: "permission denied",
	}, {
		desc: "to not an actual directory",
		readContent: `
{
    "mover": {
        "from": [
            "."
        ],
        "rules": [
            {
	    	"patterns": ["this should be in the text"],
		"to": "/dev/urandom"
            }
        ]
    }
}`,
		hasErr:      true,
		errContains: "not a directory",
	}, {
		desc: "valid configuration",
		readContent: `
{
    "mover": {
        "from": [
            "."
        ],
        "rules": [
            {
	    	"patterns": ["this should be in the text"],
		"to": "/tmp/` + uuid.New().String() + `"
            }
        ]
    }
}`,
	}} {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &EmbeddedConfig{}
			config.ReadFile = func(string) ([]byte, error) {
				return []byte(tc.readContent), tc.readError
			}

			err := cfg.Parse("config.json")
			if tc.hasErr != (err != nil) {
				t.Errorf("Parse() got error %v, want error %v", err, tc.hasErr)
				return
			}
			if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Parse() error %q does not contain %q", err.Error(), tc.errContains)
				return
			}
		})
	}
}
