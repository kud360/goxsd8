package codegen

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// namer is the one component that turns XSD names into Go identifiers
// (package doc, "# Naming"). One namer scopes one set of identifiers that
// must not collide: a package's type names, or one struct's field names.
// Claims are made in document order, and that order alone settles every
// collision.
type namer struct {
	// claimed indexes the identifiers handed out so far; it is a lookup
	// index only and never iterated (STYLE D2).
	claimed map[string]bool
}

func newNamer() *namer {
	return &namer{claimed: map[string]bool{}}
}

// claim returns a fresh identifier in this scope for the path of XSD local
// names, by the package doc's "# Naming" rules for paths and collisions.
func (n *namer) claim(path ...string) string {
	var b strings.Builder
	for _, local := range path {
		b.WriteString(identifier(local))
	}
	candidate := b.String()
	name := candidate
	for i := 2; n.claimed[name]; i++ {
		name = candidate + strconv.Itoa(i)
	}
	n.claimed[name] = true
	return name
}

// identifier maps one XSD local name (an NCName) to an exported Go
// identifier by the package doc's "# Naming" rules for one name.
func identifier(local string) string {
	var b strings.Builder
	breakWord := true
	for _, r := range local {
		if r == '_' || !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			breakWord = true
			continue
		}
		if breakWord {
			r = unicode.ToUpper(r)
			breakWord = false
		}
		b.WriteRune(r)
	}
	s := b.String()
	first, _ := utf8.DecodeRuneInString(s)
	if !unicode.IsUpper(first) {
		return "X" + s
	}
	return s
}
