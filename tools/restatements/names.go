package main

import (
	"bufio"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// selectorPattern finds the selectors an added sentence names, as the package
// doc states: a qualifier (an import path's last element, or a slash-separated
// tail of one) and a name, optionally a member's after a second dot. The
// first group is the character before the qualifier, so a qualifier never
// starts inside a word, a path or another selector.
var selectorPattern = regexp.MustCompile(`(^|[^\w./-])((?:[\w.-]+/)*[a-z_]\w*)\.(\w+)(?:\.(\w+))?`)

// selector is one selector of an added sentence: its qualifier, and the name
// it selects as ident.name spells it.
type selector struct {
	qual string
	name string
}

// String is the selector as the sentence spells it, pointer and brackets
// aside.
func (s selector) String() string { return s.qual + "." + s.name }

// selectors returns the distinct selectors text names, in text order, less
// those selecting an unexported name or member.
func selectors(text string) []selector {
	var out []selector
	seen := make(map[selector]bool)
	for _, m := range selectorPattern.FindAllStringSubmatch(text, -1) {
		if !token.IsExported(m[3]) || m[4] != "" && !token.IsExported(m[4]) {
			continue
		}
		s := selector{qual: m[2], name: m[3]}
		if m[4] != "" {
			s.name += "." + m[4]
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// namesIndex resolves selectors against the exported identifiers of a tree's
// packages outside package main.
type namesIndex struct {
	module string
	dirs   []string
	docs   map[ident][]sentence
}

// newNamesIndex indexes docs, the packages being those of the module whose
// root holds the go.mod declaring its path module.
func newNamesIndex(module string, docs map[ident][]sentence) namesIndex {
	seen := make(map[string]bool)
	var dirs []string
	for k := range docs {
		if !seen[k.dir] {
			seen[k.dir] = true
			dirs = append(dirs, k.dir)
		}
	}
	sort.Strings(dirs)
	return namesIndex{module: module, dirs: dirs, docs: docs}
}

// resolve returns the identifiers s names outside the directory own, in
// directory order: each package whose import path is s's qualifier or ends in
// a slash and the qualifier, and that documents s's name.
func (x namesIndex) resolve(s selector, own string) []ident {
	var out []ident
	for _, dir := range x.dirs {
		if dir == own {
			continue
		}
		path := x.module
		if dir != "" {
			path += "/" + dir
		}
		if path != s.qual && !strings.HasSuffix(path, "/"+s.qual) {
			continue
		}
		k := ident{dir: dir, name: s.name}
		if len(x.docs[k]) > 0 {
			out = append(out, k)
		}
	}
	return out
}

// modulePath reads the module path from the go.mod at root.
func modulePath(root string) (string, error) {
	name := filepath.Join(root, "go.mod")
	f, err := os.Open(name)
	if err != nil {
		return "", fmt.Errorf("reading the module path: %w", err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			return strings.Trim(strings.TrimSpace(rest), `"`), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("reading %s: %w", name, err)
	}
	return "", fmt.Errorf("%s declares no module path", name)
}
