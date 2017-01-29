package config

import (
	"errors"
	"testing"
)

func TestParseSyncConfiguration(t *testing.T) {
	oldReadFile := readFile
	defer func() { readFile = oldReadFile }()

	fakeReadFile := func(data string) func(string) ([]byte, error) {
		return func(filename string) ([]byte, error) {
			if data == "error" {
				return nil, errors.New("readFile error")
			}
			return []byte(data), nil
		}
	}

	for _, test := range []struct {
		desc   string
		config string

		err bool
	}{{
		"empty contents",
		"",
		true,
	}, {
		"read file error",
		"error",
		true,
	}, {
		"unparseable json",
		"\x11\x22\x33\x44",
		true,
	}, {
		"point to non-existing directory",
		`
{
  "dirs": {
		"/does/not/exist/locally": "sample/remote/dir"
	}
}
 `,
		true,
	}, {
		"point to file instead of dir",
		`
{
  "dirs": {
		"/dev/random": "sample/remote/dir"
	}
}
		`,
		true,
	}, {
		"interval too small",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
	"interval": "0s"
}
`,
		true,
	}, {
		"manifest file missing",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
	"interval": "1h"
}
`,
		true,
	}, {
		"remote manifest file missing",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
	"interval": "1h",
	"manifest_file": "/tmp/manifest"
}
`,
		true,
	}, {
		"aes_passphrase missing",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
	"interval": "1h",
	"manifest_file": "/tmp/manifest",
	"remote_manifest_file": "remote/manifest"
}
`,
		true,
	}, {
		"credentials_file missing",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
	"interval": "1h",
	"manifest_file": "/tmp/manifest",
	"remote_manifest_file": "remote/manifest",
	"aes_passphrase": "sample passphrase"
}
`,
		true,
	}, {
		"simple include/exclude regexps",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
  "interval": "1h",
  "manifest_file": "/tmp/manifest",
  "remote_manifest_file": "manifest",
  "aes_passphrase": "This is safe",
  "credentials_file": "writer.json",
  "include": [".*\\.go"],
  "exclude": [".*\\.swp", "^\\..*"]
}
`,
		false,
	}, {
		"include regexp invalid",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
  "interval": "1h",
  "manifest_file": "/tmp/manifest",
  "remote_manifest_file": "manifest",
  "aes_passphrase": "This is safe",
  "credentials_file": "writer.json",
  "include": ["*.go"]
}
		`,
		true,
	}, {
		"exclude regexp invalid",
		`
{
  "dirs": {
		".": "sample/remote/dir"
	},
  "interval": "1h",
  "manifest_file": "/tmp/manifest",
  "remote_manifest_file": "manifest",
  "aes_passphrase": "This is safe",
  "credentials_file": "writer.json",
  "exclude": ["*.go"]
}
		`,
		true,
	}} {
		readFile = fakeReadFile(test.config)
		cfg := &Sync{}
		err := cfg.Parse(test.desc)
		t.Logf("%s: %v", test.desc, err)
		if test.err != (err != nil) {
			t.Errorf("%s: Parse() want error %v, got (%+v, %v)", test.desc, test.err, cfg, err)
		}
	}
}
