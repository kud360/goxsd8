package goname

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Namer scopes one set of identifiers that must not collide: a package's type
// names, or one struct's field names. Claims are made in order, and that
// order settles every collision among the candidates it is handed.
type Namer struct {
	// claimed indexes the identifiers handed out so far; it is a lookup
	// index only and never iterated (STYLE D2).
	claimed map[string]bool
}

// NewNamer returns a Namer that has claimed nothing.
func NewNamer() *Namer {
	return &Namer{claimed: map[string]bool{}}
}

// Claim returns a fresh identifier in this scope for the path of XSD local
// names: Render's candidate, claimed as ClaimCandidate claims it.
func (n *Namer) Claim(path ...string) string {
	return n.ClaimCandidate(Render(path))
}

// ClaimCandidate returns a fresh identifier in this scope for candidate, an
// identifier already rendered, by codegen's package doc "# Naming" rule for
// collisions.
func (n *Namer) ClaimCandidate(candidate string) string {
	name := candidate
	for i := 2; n.claimed[name]; i++ {
		name = candidate + strconv.Itoa(i)
	}
	n.claimed[name] = true
	return name
}

// Render joins the Identifier of each XSD local name in path.
func Render(path []string) string {
	var b strings.Builder
	for _, local := range path {
		b.WriteString(Identifier(local))
	}
	return b.String()
}

// Identifier maps one XSD local name (an NCName) to an exported Go
// identifier by codegen's package doc "# Naming" rules for one name.
func Identifier(local string) string {
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
