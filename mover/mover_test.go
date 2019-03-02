package mover

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/andreich/docsync/config"
)

type fileInfo struct {
	name     string
	modified time.Time
	dir      bool
}

func (f *fileInfo) Name() string {
	return f.name
}

func (f *fileInfo) Size() int64 {
	panic("not implemented")
}

func (f *fileInfo) Mode() os.FileMode {
	panic("not implemented")
}

func (f *fileInfo) ModTime() time.Time {
	return f.modified
}

func (f *fileInfo) IsDir() bool {
	return f.dir
}

func (f *fileInfo) Sys() interface{} {
	return nil
}

type fsEntry struct {
	children map[string]bool
	content  string
	info     os.FileInfo
	err      error
}

type fs struct {
	info map[string]*fsEntry
}

func (f *fs) addEntry(filePath string, e *fsEntry) {
	f.info[filePath] = e
	parts := strings.Split(filePath, "/")
	prev := ""
	for i := 0; i < len(parts)-1; i++ {
		cur := path.Join(prev, parts[i])
		node, found := f.info[cur]
		if found {
			if parts[i+1] != "" {
				node.children[parts[i+1]] = true
			}
			prev = cur
			continue
		}
		f.info[cur] = &fsEntry{
			children: map[string]bool{},
			info: &fileInfo{
				name: parts[i],
				dir:  true,
			},
		}
		if parts[i+1] != "" {
			f.info[cur].children[parts[i+1]] = true
		}
		prev = cur
	}
}

func newFileInfo(filePath string, modified time.Time, dir bool) *fileInfo {
	return &fileInfo{
		name:     path.Base(filePath),
		modified: modified,
		dir:      dir,
	}
}

func (f *fs) add(filePath string, modified time.Time, content string) {
	f.addEntry(filePath, &fsEntry{
		content: content,
		info:    newFileInfo(filePath, modified, false),
	})
}

func (f *fs) addError(filePath string, err error) {
	f.addEntry(filePath, &fsEntry{
		err: err,
		info: &fileInfo{
			name: path.Base(filePath),
		},
	})
}

type openCloser struct {
	bytes.Reader
}

func (oc *openCloser) Close() error {
	return nil
}

func (f *fs) open(filePath string) (io.ReadCloser, error) {
	node, found := f.info[filePath]
	if !found {
		return nil, os.ErrNotExist
	}
	if node.err != nil {
		return nil, node.err
	}
	if node.info.IsDir() {
		return nil, os.ErrInvalid
	}
	return &openCloser{Reader: *bytes.NewReader([]byte(node.content))}, nil
}

func (f *fs) readdir(filePath string) ([]os.FileInfo, error) {
	node, found := f.info[filePath]
	if !found {
		return nil, os.ErrNotExist
	}
	var res []os.FileInfo
	for entry := range node.children {
		res = append(res, f.info[path.Join(filePath, entry)].info)
	}
	return res, nil
}

func (f *fs) extractText(filePath string) ([]string, error) {
	if strings.HasSuffix(filePath, "extraction-failure.pdf") {
		return nil, errors.New("extraction failure")
	}
	node, found := f.info[filePath]
	if !found {
		return nil, os.ErrNotExist
	}
	return []string{node.content}, nil
}

func (f *fs) readfile(filePath string) ([]byte, error) {
	node, found := f.info[filePath]
	if !found {
		return nil, os.ErrNotExist
	}
	return []byte(node.content), nil
}

func (f *fs) stat(filePath string) (os.FileInfo, error) {
	node, found := f.info[filePath]
	if !found {
		return nil, os.ErrNotExist
	}
	return node.info, nil
}

func TestMover(t *testing.T) {
	f := &fs{
		info: make(map[string]*fsEntry),
	}
	old := time.Now().AddDate(0, -6, 0).UTC()
	f.add("user/config.json", old, `{
	"mover": {
		"from": ["user/root/A", "user/root/B", "user/root/C"],
		"rules": [{
			"patterns": ["bank llc", "#123456"],
			"to": "to/bank"
		}, {
			"patterns": ["electricity", "invoice no."],
			"to": "to/invoices/electricity"
		}]
	}
}`)
	f.addError("user/root/C/permission-denied/file.pdf", errors.New("permission denied"))
	f.add("user/root/A/subdir/file.pdf", old, "interesting")
	f.add("user/root/A/subdir/second-file.pdf", old, "from bank llc, client #123456, account statement")
	f.add("user/root/A/subdir/third-file.pdf", old, "electricity company, invoice no. 123")
	f.add("user/root/A/another-file.pdf", old, "from bank llc, client #123456, credit card statement")
	f.add("user/root/A/partial-match.pdf", old, "details about bank llc")
	f.add("user/root/A/no-match.pdf", old, "for sure not matching")
	f.add("user/root/B/unsupported", old, "should not appear")
	f.add("user/root/B/extraction-failure.pdf", old, "")
	f.add("to/bank/", old, "")
	f.add("to/invoices/electricity/", old, "")

	osOpen = f.open
	osStat = f.stat
	readdir = f.readdir
	extractTextFn = f.extractText
	config.ReadFile = f.readfile

	cfg := &EmbeddedConfig{}
	if err := cfg.Parse("user/config.json"); err != nil {
		t.Fatalf("Could not parse configuration: %v", err)
	}
	m := New(cfg.Mover)

	// First scan - initial detection.
	entries, err := m.Scan(true)
	if err != nil {
		t.Fatalf("Could not perform a scan: %v", err)
	}
	want := map[string][]string{
		"to/bank":                 []string{"user/root/A/another-file.pdf", "user/root/A/subdir/second-file.pdf"},
		"to/invoices/electricity": []string{"user/root/A/subdir/third-file.pdf"},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Errorf("Scan got %+v, want %+v", entries, want)
	}

	entries, err = m.Scan(true)
	if err != nil {
		t.Fatalf("Could not perform a scan(2): %v", err)
	}
	want = map[string][]string{}
	if !reflect.DeepEqual(entries, want) {
		t.Errorf("Scan(2) got %+v, want %+v", entries, want)
	}
	// Add one more file to electricity, update one from bank and update but without content change another from bank.
	newer := old.AddDate(0, 1, 0)
	f.add("user/root/A/electricity.pdf", newer, "electricity company, invoice no. 124")
	f.add("user/root/A/subdir/second-file.pdf", newer, "from bank llc, client #123456, account statement update")
	f.add("user/root/A/another-file.pdf", newer, "from bank llc, client #123456, credit card statement")

	entries, err = m.Scan(true)
	if err != nil {
		t.Fatalf("Could not perform a scan(3): %v", err)
	}
	want = map[string][]string{
		"to/bank":                 []string{"user/root/A/subdir/second-file.pdf"},
		"to/invoices/electricity": []string{"user/root/A/electricity.pdf"},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Errorf("Scan(3) got %+v, want %+v", entries, want)
	}
}
