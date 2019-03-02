// Package mover provides a structure which can be used to move files from
// a directory to another based on rules matching content.
package mover

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type seenRecord struct {
	modified time.Time
	hash     string
}

// M is the actual mover, able to scan directories and move the matched files.
type M struct {
	cfg *Config

	seen map[string]*seenRecord
}

// New creates a new mover with the given config (should have been validated
// before).
func New(cfg *Config) *M {
	return &M{
		cfg:  cfg,
		seen: map[string]*seenRecord{},
	}
}

func supported(filename string) bool {
	for _, suffix := range []string{".pdf"} {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}
	return false
}

var osOpen = func(fn string) (io.ReadCloser, error) {
	return os.Open(fn)
}

func hashFile(filename string) (string, error) {
	fp, err := osOpen(filename)
	if err != nil {
		return "", err
	}
	defer fp.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fp); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (m *M) alreadySeen(filename string, modified time.Time) (bool, error) {
	record, found := m.seen[filename]
	var hash string
	var err error
	if found {
		if record.modified == modified {
			return true, nil
		}
	}
	hash, err = hashFile(filename)
	if err != nil {
		return false, err
	}
	if record != nil && hash == record.hash {
		record.modified = modified
		return true, nil
	}
	m.seen[filename] = &seenRecord{
		modified: modified,
		hash:     hash,
	}
	return false, nil
}

func doMove(moves map[string][]string, dryRun bool) error {
	for dst, files := range moves {
		for _, file := range files {
			to := path.Join(dst, path.Base(file))
			if dryRun {
				log.Printf("%q -> %q", file, to)
				continue
			}
			if _, err := osStat(to); os.IsNotExist(err) {
				if err := os.Rename(file, to); err != nil {
					return fmt.Errorf("moving %q to %q: %v", file, to, err)
				}
			} else {
				return fmt.Errorf("moving %q to %q: declined as destination already exists", file, to)
			}
		}
	}
	return nil
}

// Scan runs once through all the configured From directories and returns a
// map keyed on the destination directory with the list of files to be moved
// there. If dryRun is false, then the files are also moved.
func (m *M) Scan(dryRun bool) (map[string][]string, error) {
	moves := map[string][]string{}
	for _, d := range m.cfg.From {
		localMoves, err := m.internalScan(d)
		if err != nil {
			log.Printf("%q: could not scan: %v", d, err)
			continue
		}
		merge(moves, localMoves)
	}
	return moves, doMove(moves, dryRun)
}

func merge(dst, src map[string][]string) {
	for to, files := range src {
		dst[to] = append(dst[to], files...)
		sort.Strings(dst[to])
	}
}

var readdir = ioutil.ReadDir

func (m *M) internalScan(d string) (map[string][]string, error) {
	files, err := readdir(d)
	if err != nil {
		return nil, err
	}
	moves := map[string][]string{}
	for _, entry := range files {
		fullPath := path.Join(d, entry.Name())
		if entry.IsDir() {
			localMoves, err := m.internalScan(fullPath)
			if err != nil {
				return nil, err
			}
			merge(moves, localMoves)
			continue
		}
		if !supported(entry.Name()) {
			continue
		}
		if seen, err := m.alreadySeen(fullPath, entry.ModTime()); seen {
			continue
		} else if err != nil {
			return nil, err
		}
		to, matches, err := m.match(fullPath)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}
		moves[to] = append(moves[to], fullPath)
	}
	return moves, nil
}

var extractTextFn = extractText

func (m *M) match(filename string) (string, bool, error) {
	pages, err := extractTextFn(filename)
	if err != nil {
		log.Printf("%q: %v", filename, err)
		return "", false, nil
	}
	content := strings.Join(pages, "\n")
	for _, entry := range m.cfg.Rules {
		matches := true
		for _, re := range entry.PatternsRegexp {
			matches = matches && re.MatchString(content)
			if !matches {
				break
			}
		}
		if !matches {
			continue
		}
		return entry.To, true, nil
	}
	return "", false, nil
}
