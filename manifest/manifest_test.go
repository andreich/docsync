package manifest

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"
)

type file struct {
	name  string
	mod   time.Time
	bytes []byte
}

type dirOrFile struct {
	file
	files []dirOrFile
}

func (d dirOrFile) Name() string {
	return d.name
}

func (d dirOrFile) Size() int64 {
	if len(d.files) > 0 {
		return 0
	}
	return int64(len(d.bytes))
}

func (d dirOrFile) Mode() os.FileMode {
	return 0
}

func (d dirOrFile) ModTime() time.Time {
	return d.mod
}

func (d dirOrFile) IsDir() bool {
	return len(d.files) > 0
}

func (d dirOrFile) Sys() interface{} {
	return nil
}

type fileSystem map[string]dirOrFile

func (f fileSystem) init() {
	var queue []string
	for k := range f {
		queue = append(queue, k)
	}
	for len(queue) > 0 {
		elem := queue[0]
		queue = queue[1:]
		dof := f[elem]
		if dof.IsDir() {
			for _, fe := range dof.files {
				fn := path.Join(elem, fe.Name())
				f[fn] = fe
				if fe.IsDir() {
					queue = append(queue, fn)
				}
			}
		}
	}
}

func (f fileSystem) readDir(fn string) ([]os.FileInfo, error) {
	if strings.Contains(fn, "error-dir") {
		return nil, fmt.Errorf("error reading dir %q", fn)
	}
	dof, found := f[fn]
	if !found {
		return nil, fmt.Errorf("%q not found", fn)
	}
	var ret []os.FileInfo
	for _, f := range dof.files {
		ret = append(ret, f)
	}
	return ret, nil
}

func (f fileSystem) readFile(fn string) ([]byte, error) {
	if strings.Contains(fn, "with-error") {
		return nil, fmt.Errorf("error reading %q", fn)
	}
	dof, found := f[fn]
	if !found {
		return nil, fmt.Errorf("%q not found", fn)
	}
	if len(dof.bytes) == 0 {
		return nil, fmt.Errorf("permission denied %q", fn)
	}
	return dof.bytes, nil
}

func TestManifestUpdates(t *testing.T) {
	oldReadDir, oldReadFile := readDir, readFile
	defer func() {
		readDir, readFile = oldReadDir, oldReadFile
	}()
	now := time.Now()
	for _, test := range []struct {
		desc string
		fs   fileSystem
		dir  string

		changed []string
		err     bool
	}{{
		"readdir error",
		fileSystem{},
		"/directory/not/found",
		nil,
		true,
	}, {
		"empty directory",
		fileSystem{
			"/root": dirOrFile{},
		},
		"/root",
		nil,
		false,
	}, {
		"one file directory",
		fileSystem{
			"/root": dirOrFile{
				files: []dirOrFile{{file: file{name: "sample", mod: now, bytes: []byte{1, 2, 3}}}},
			},
		},
		"/root",
		[]string{"/root/sample"},
		false,
	}, {
		"more complex file system",
		fileSystem{
			"/root": dirOrFile{files: []dirOrFile{
				{file: file{name: "f1", mod: now, bytes: []byte{1}}},
				{file: file{name: "f2", mod: now, bytes: []byte{2}}},
				{file: file{name: "d1"}, files: []dirOrFile{
					{file: file{name: "f3", mod: now, bytes: []byte{3}}},
					{file: file{name: "f3", mod: now, bytes: []byte{3}}},
					{file: file{name: "f3", mod: now, bytes: []byte{3}}},
					{file: file{name: "f3", mod: now.Add(1 * time.Second), bytes: []byte{3}}},
					{file: file{name: "f3", mod: now.Add(2 * time.Second), bytes: []byte{3, 3, 3}}}},
				},
				{file: file{name: "d2"}, files: []dirOrFile{
					{file: file{name: "f4", mod: now, bytes: []byte{4}}},
					{file: file{name: "f5", mod: now, bytes: []byte{5}}},
				}},
			}},
		},
		"/root",
		[]string{"/root/d1/f3", "/root/d2/f4", "/root/d2/f5", "/root/f1", "/root/f2"},
		false,
	}, {
		"filesystem with errors",
		fileSystem{
			"/root": dirOrFile{files: []dirOrFile{
				{file: file{name: "f1", mod: now, bytes: []byte{1}}},
				{file: file{name: "f1", mod: now.Add(1 * time.Second), bytes: []byte{}}},
				{file: file{name: "error-dir"}, files: []dirOrFile{
					{file: file{name: "unreachable"}},
				}},
			}},
		},
		"/root",
		nil,
		true,
	}} {
		test.fs.init()
		readDir, readFile = test.fs.readDir, test.fs.readFile
		m := New(nil, nil)
		changed, err := m.Update(test.dir)
		if test.err != (err != nil) {
			t.Errorf("%s: m.Update(%q) want error %v, got %v", test.desc, test.dir, test.err, err)
		}
		if !reflect.DeepEqual(changed, test.changed) {
			t.Errorf("%s: m.Update(%q) want %v, got %v", test.desc, test.dir, test.changed, changed)
		}
	}
}

func TestManifestDumpLoad(t *testing.T) {
	oldReadDir, oldReadFile := readDir, readFile
	defer func() {
		readDir, readFile = oldReadDir, oldReadFile
	}()
	fs := fileSystem{
		"/root": dirOrFile{
			files: []dirOrFile{
				{file: file{name: "sample-00", mod: time.Now(), bytes: []byte{1, 2, 3}}},
				{file: file{name: "sample-01", mod: time.Now(), bytes: []byte{1, 2, 3}}},
				{file: file{name: "sample-01.excl", mod: time.Now(), bytes: []byte{1, 2, 3}}},
				{file: file{name: ".excluded", mod: time.Now(), bytes: []byte{1}}},
			},
		},
	}
	fs.init()
	readDir, readFile = fs.readDir, fs.readFile
	m := New([]string{"sample-\\d{2}"}, []string{"\\.excluded", ".*.excl"})
	changed, err := m.Update("/root")
	want := []string{"/root/sample-00", "/root/sample-01"}
	if err != nil || !reflect.DeepEqual(changed, want) {
		t.Errorf("m.Update(/root) want (%v, nil) got (%v, %v)", want, changed, err)
	}
	var buf bytes.Buffer
	if err := m.Dump(&buf); err != nil {
		t.Errorf("Dump want nil, got error %v", err)
	}
	newM := New(nil, nil)
	if err := newM.Load(&buf); err != nil {
		t.Errorf("Load want nil, got error %v", err)
	}
	changed, err = newM.Update("/root")
	want = nil
	if err != nil || !reflect.DeepEqual(changed, want) {
		t.Errorf("newM.Update(/root) want (%v, nil) got (%v, %v)", want, changed, err)
	}
}
