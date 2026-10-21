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

// claim returns a fresh identifier for the path of XSD local names: the
// ancestor rule names an anonymous type by concatenating its nearest named
// ancestor's name with every element name below it (purchaseOrder, shipTo →
// PurchaseOrderShipTo). A candidate already claimed in this scope — two names
// that fold to one identifier, like a and A — takes the smallest numeric
// suffix from 2 up that is still free, so the earlier claim in document order
// keeps the bare name.
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
// identifier. A rune Go does not admit in an identifier — '-', '.', '·', a
// combining mark — and '_' are word breaks: each is dropped and the next
// letter is upper-cased, as the first letter is. A result whose first rune is
// not an upper-case letter (an NCName starting with '_' then a digit, or with
// a letter that has no upper case) is prefixed with X. Every result therefore
// begins with an upper-case letter, which is what keeps it exported and keeps
// it from ever spelling a Go keyword, every one of which is lower-case.
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
