package main

import (
	"regexp"
	"sort"
	"strings"
)

// selectorPattern finds the selectors an added sentence names, which the
// package doc states. The first group is the character before the qualifier,
// so a qualifier never starts inside a word, a path or another selector.
var selectorPattern = regexp.MustCompile(`(^|[^\w./-])((?:[\w.-]+/)*[a-z_]\w*)\.([A-Z]\w*)(?:\.(\w+))?`)

// selector is one selector of an added sentence: its qualifier, and the name
// it selects as ident.name spells it.
type selector struct {
	qual string
	name string
}

// String is the selector as the sentence spells it, pointer and brackets
// aside.
func (s selector) String() string { return s.qual + "." + s.name }

// selectors returns the distinct selectors text names, in text order. A
// member is read whatever its case, so a selector of an unexported member
// resolves to nothing: no identifier is indexed under it.
func selectors(text string) []selector {
	var out []selector
	seen := make(map[selector]bool)
	for _, m := range selectorPattern.FindAllStringSubmatch(text, -1) {
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
// root holds the go.mod declaring its path module. It drains docs' keys into
// sorted dirs, which resolve ranges over (STYLE D2), rather than readTree
// keeping a slice alongside: readTree reads files in sorted path order, which
// splits a directory's files ("a/b.go", "a/b/c.go", "a/c.go"), so a slice it
// kept would need this same set and sort.
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

// resolve returns the identifiers s names, as the package doc states, from a
// sentence in the directory own, in directory order.
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
