package mover

import (
	"strings"
	"testing"
)

func TestExtractText(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		in       string
		hasErr   bool
		contains []string
	}{{
		desc:   "no input file",
		in:     "",
		hasErr: true,
	}, {
		desc:   "missing file",
		in:     "file-not-there.pdf",
		hasErr: true,
	}, {
		desc: "good extraction",
		in:   "migros.pdf",
		contains: []string{
			"dividends from companies operating in the retail",
			"326 371",
			"CHF 120 million in 2016",
		},
	}} {
		t.Run(tc.desc, func(t *testing.T) {
			content, err := extractText(tc.in)
			if tc.hasErr != (err != nil) {
				t.Errorf("got (%v, %v), want error %v", content, err, tc.hasErr)
			}
			if err != nil {
				return
			}
			cnt := strings.Join(content, "\n")
			for _, entry := range tc.contains {
				if !strings.Contains(cnt, entry) {
					t.Errorf("missing %q in content", entry)
				}
			}
		})
	}
}
