// Package manifest keeps track of changes to interesting files within a given
// set of directories.
package manifest

import (
	"crypto/md5"
	"encoding/gob"
	"io"
	"io/ioutil"
	"log"
	"path"
	"regexp"
	"sort"
	"time"
)

type value struct {
	Mod  time.Time
	Hash [md5.Size]byte
}
type index struct {
	Data    map[string]value
	Include []string
	Exclude []string
	include []*regexp.Regexp
	exclude []*regexp.Regexp
}

// Manifest provides the interface for monitoring changes on a directory.
type Manifest interface {
	// Update tracks changes in the given directory.
	Update(dir string) (changed []string, err error)
	// Dump allows serialization of the manifest state.
	Dump(io.Writer) error
	// Load allows deserialization of a manifest state in the current object.
	Load(io.Reader) error
}

// New creates a manifest with the provided include/exclude rules.
// include & exclude should parse to valid Regexp.
func New(include, exclude []string) Manifest {
	i := &index{
		Data:    make(map[string]value),
		Include: include,
		Exclude: exclude,
	}
	i.filtersForImport()
	return i
}

func filtersForExport(f []*regexp.Regexp) []string {
	var res []string
	for _, e := range f {
		res = append(res, e.String())
	}
	return res
}

func (i *index) filtersForExport() {
	i.Include = filtersForExport(i.include)
	i.Exclude = filtersForExport(i.exclude)
}

func filtersForImport(f []string) []*regexp.Regexp {
	var res []*regexp.Regexp
	for _, e := range f {
		res = append(res, regexp.MustCompile(e))
	}
	return res
}

func (i *index) filtersForImport() {
	i.include = filtersForImport(i.Include)
	i.exclude = filtersForImport(i.Exclude)
}

var (
	// For faking in tests.
	readDir  = ioutil.ReadDir
	readFile = ioutil.ReadFile
	hash     = md5.Sum
)

func (i index) Dump(w io.Writer) error {
	i.filtersForExport()
	return gob.NewEncoder(w).Encode(i)
}

func (i *index) Load(r io.Reader) error {
	err := gob.NewDecoder(r).Decode(i)
	i.filtersForImport()
	return err
}

func matchesAny(s string, res []*regexp.Regexp) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func (i index) tracks(filepath string) bool {
	name := path.Base(filepath)
	if matchesAny(name, i.exclude) {
		return false
	}
	if len(i.include) == 0 {
		return true
	}
	return matchesAny(name, i.include)
}

func (i index) Update(d string) ([]string, error) {
	files, err := readDir(d)
	if err != nil {
		return nil, err
	}
	changed := make(map[string]bool)
	for _, f := range files {
		fn := path.Join(d, f.Name())
		if f.IsDir() {
			new, err := i.Update(fn)
			if err != nil {
				return nil, err
			}
			for _, fc := range new {
				changed[fc] = true
			}
			continue
		}
		if v, found := i.Data[fn]; found && f.ModTime().Equal(v.Mod) {
			continue
		}
		if !i.tracks(fn) {
			continue
		}
		bytes, err := readFile(fn)
		if err != nil {
			log.Printf("Could not read %s: %v\n", fn, err)
			continue
		}
		changed[fn] = true
		i.Data[fn] = value{
			Mod:  f.ModTime(),
			Hash: hash(bytes),
		}
	}
	var res []string
	for k := range changed {
		res = append(res, k)
	}
	sort.Strings(res)
	return res, nil
}
