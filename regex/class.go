package regex

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/kud360/goxsd8/internal/xmlname"
)

// maxRune is the largest Unicode code point; the universe for complementing a
// character set spans [0, maxRune]. Surrogate code points are included: Go's
// regexp accepts them inside character classes, and they never decode from
// well-formed UTF-8 input, so their presence in a class body is inert.
const maxRune = 0x10FFFF

// runeRange is an inclusive code-point interval with lo <= hi.
type runeRange struct {
	lo, hi rune
}

// runeSet is a set of code points held as a sorted, disjoint, non-adjacent list
// of intervals. It is the internal currency for character-class algebra:
// character-class subtraction ([a-z-[m]]), negation ([^...]), and the
// multi-character escapes whose spec definitions are complements (\S, \D, \w)
// have no native RE2 representation, so the translator computes the flattened
// set here and emits explicit ranges. Kept as a slice (never a map) so emission
// is deterministic (STYLE D2/D1).
type runeSet []runeRange

// add appends an interval; the set is re-normalized by normalize before use.
func (s runeSet) add(lo, hi rune) runeSet {
	return append(s, runeRange{lo, hi})
}

// addTable folds every interval of a unicode.RangeTable into the set. A stride
// of 1 covers [Lo, Hi] contiguously and is added as the one interval it is;
// only a wider stride needs the code-point walk, because the covered points are
// then non-adjacent and no single runeRange spans them.
//
// Adding contiguous runs whole is what keeps the set proportional to the
// table's intervals rather than to its code points: unicode.Categories["C"]
// covers 965,096 code points in 609 intervals, so walking it materialized
// 965,096 single-point runeRanges for normalize to sort back down to a few
// hundred — 70ms per \w or \W translation, against 1.5µs for a class with no
// property escape in it (#913).
func (s runeSet) addTable(t *unicode.RangeTable) runeSet {
	for _, r := range t.R16 {
		s = s.addStrided(rune(r.Lo), rune(r.Hi), rune(r.Stride))
	}
	for _, r := range t.R32 {
		s = s.addStrided(rune(r.Lo), rune(r.Hi), rune(r.Stride))
	}
	return s
}

// addStrided folds one unicode.RangeTable interval into the set: whole when the
// stride is contiguous, one code point at a time otherwise.
func (s runeSet) addStrided(lo, hi, stride rune) runeSet {
	if stride == 1 {
		return s.add(lo, hi)
	}
	for c := lo; c <= hi; c += stride {
		s = s.add(c, c)
	}
	return s
}

// normalize returns the set as sorted, merged, disjoint intervals.
func (s runeSet) normalize() runeSet {
	if len(s) == 0 {
		return nil
	}
	cp := make(runeSet, len(s))
	copy(cp, s)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].lo != cp[j].lo {
			return cp[i].lo < cp[j].lo
		}
		return cp[i].hi < cp[j].hi
	})
	out := cp[:1]
	for _, r := range cp[1:] {
		last := &out[len(out)-1]
		if r.lo <= last.hi+1 {
			if r.hi > last.hi {
				last.hi = r.hi
			}
			continue
		}
		out = append(out, r)
	}
	return out
}

// union returns the normalized union of two sets.
func (s runeSet) union(o runeSet) runeSet {
	return append(append(runeSet{}, s...), o...).normalize()
}

// complement returns [0, maxRune] minus the set.
func (s runeSet) complement() runeSet {
	n := s.normalize()
	var out runeSet
	next := rune(0)
	for _, r := range n {
		if r.lo > next {
			out = out.add(next, r.lo-1)
		}
		if r.hi+1 > next {
			next = r.hi + 1
		}
	}
	if next <= maxRune {
		out = out.add(next, maxRune)
	}
	return out
}

// subtract returns the set minus o (set difference).
func (s runeSet) subtract(o runeSet) runeSet {
	// A \ B = A ∩ complement(B); complement is over the whole universe, and
	// intersection with A restores the bound, so the result stays within A.
	return s.intersect(o.complement())
}

// intersect returns the normalized intersection of two sets.
func (s runeSet) intersect(o runeSet) runeSet {
	a := s.normalize()
	b := o.normalize()
	var out runeSet
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		lo := a[i].lo
		if b[j].lo > lo {
			lo = b[j].lo
		}
		hi := a[i].hi
		if b[j].hi < hi {
			hi = b[j].hi
		}
		if lo <= hi {
			out = out.add(lo, hi)
		}
		if a[i].hi < b[j].hi {
			i++
			continue
		}
		j++
	}
	return out
}

// wsSet is the set \s denotes: [#x20\t\n\r] (Datatypes §G.4.2.5).
func wsSet() runeSet {
	return runeSet{{0x9, 0x9}, {0xA, 0xA}, {0xD, 0xD}, {0x20, 0x20}}
}

// wordExcludedSet is [\p{P}\p{Z}\p{C}] — the classes \w subtracts from the full
// Unicode range. \w == [#x0000-#x10FFFF]-[\p{P}\p{Z}\p{C}] is a subtraction from
// the entire universe, which is exactly the complement of this union; \W is its
// re-complement, i.e. this union itself. So \w = wordExcludedSet.complement()
// and \W = wordExcludedSet, with no residual subtraction to represent.
func wordExcludedSet() runeSet {
	return runeSet{}.
		addTable(unicode.Categories["P"]).
		addTable(unicode.Categories["Z"]).
		addTable(unicode.Categories["C"])
}

// multiEscSet returns the set matched by a multi-character escape letter
// (\d \D \s \S \w \W \i \I \c \C) as it appears inside a character-class
// expression, where no symbolic RE2 form is available and the set must be
// materialized.
func multiEscSet(c byte) runeSet {
	switch c {
	case 'd':
		return runeSet{}.addTable(unicode.Categories["Nd"])
	case 'D':
		return runeSet{}.addTable(unicode.Categories["Nd"]).complement()
	case 's':
		return wsSet()
	case 'S':
		return wsSet().complement()
	case 'w':
		return wordExcludedSet().complement()
	case 'W':
		return wordExcludedSet()
	case 'i':
		return nameStartSet()
	case 'I':
		return nameStartSet().complement()
	case 'c':
		return nameCharSet()
	case 'C':
		return nameCharSet().complement()
	}
	return nil
}

// nameStartSet is the set \i denotes: the XML NameStartChar, production [4]
// (Datatypes §G.4.2.5: "\i | the set of initial name characters, those matched
// by NameStartChar"). \I is its complement. The production, and the XML
// edition it is taken from (Datatypes §H.1 clause 1), are internal/xmlname's.
func nameStartSet() runeSet {
	return runeSet{}.addTable(xmlname.NameStartChar)
}

// nameCharSet is the set \c denotes: the XML NameChar, production [4a], i.e.
// NameStartChar plus the continuation characters (Datatypes §G.4.2.5: "\c |
// the set of name characters, those matched by NameChar"). \C is its
// complement.
func nameCharSet() runeSet {
	return nameStartSet().addTable(xmlname.NameCharExtra)
}

// propSet returns the code-point set denoted by a \p{...} property body, or by a
// \P{...} one when negate is set (Datatypes §G.4.2.2/§G.4.2.3): a Unicode
// general category (L, Lu, Nd, …) or a block escape (IsBasicLatin, …). The
// polarity goes in rather than being applied by the caller, because a block
// name that names no block denotes all characters under \p and \P alike
// (§G.4.2.4; see blockSet). An unrecognized general category matches neither
// IsCategory nor IsBlock, so it is no regExp at all (§G.4.2.4) and an error.
func propSet(name string, negate bool) (runeSet, error) {
	if rest, ok := strings.CutPrefix(name, "Is"); ok {
		set, neg, err := blockSet(rest, negate)
		if err != nil {
			return nil, err
		}
		if neg {
			return set.complement(), nil
		}
		return set, nil
	}
	t, ok := unicode.Categories[name]
	if !ok {
		return nil, fmt.Errorf("unrecognized Unicode category %q", name)
	}
	set := runeSet{}.addTable(t)
	if negate {
		return set.complement(), nil
	}
	return set, nil
}

// isCategoryName reports whether name is a Unicode general category that RE2
// can express directly as \p{name}, so a standalone category escape can pass
// through instead of being materialized.
func isCategoryName(name string) bool {
	_, ok := unicode.Categories[name]
	return ok
}

// errUnsupported marks a construct this module RECOGNIZES as well-formed per
// Datatypes Appendix G but does not implement, as opposed to one Appendix G's
// grammar genuinely excludes. The two are indistinguishable in a translation
// FAILURE — both stop the translation — but not in a verdict about the pattern
// AUTHOR: only the second is a src-pattern-value defect. Its one producer is
// regex.go's maxRepeat ceiling, which production [71]'s uncapped QuantExact
// puts on the recognized side. [CheckSyntax] tells the two classes apart
// through this sentinel, so a schema-construction pass rejects malformed
// patterns eagerly without false-rejecting one the spec allows. Unexported: the
// distinction is CheckSyntax's to make, and a caller reaching past it would be
// asserting the classification itself.
var errUnsupported = errors.New("not supported by this implementation")

// blockSet resolves the block escape \p{Isnm}, or \P{Isnm} when negate is set,
// and returns emitClass's operands: the set, and whether the emitted class
// negates it. A name whose normalized form (Datatypes §G.4.2.3: whitespace and
// underbars stripped, hyphens and case retained) is a key of the generated
// unicodeBlocks — every block of the Unicode version tools/blockgen reads, plus
// §G.4.2.3's superseded Unicode 3.1 names — yields that block and negate
// unchanged, \P being the block's complement (dt-ccesblock).
//
// A name production [96] admits that the table does not hold — \p{IsaA0-a9} —
// is a block this module does not recognize, and §G.4.2.4 makes \p{IsX} and
// \P{IsX} EACH denote the set of all characters, so both yield all characters
// and no negation: \P{IsX} is not the complement of \p{IsX} here, because
// dt-ccesblock's complement relation holds only for a recognized X. The error
// and the empty set §G.4.2.4 also allows are ·at user option· only, and this
// module offers no such option. §G.4.2.4's warning is a "should" with no
// mechanism named, and none is issued, as parser/conditional.go issues none for
// src-cip's encouraged warning on an unknown versioning attribute.
func blockSet(nm string, negate bool) (set runeSet, neg bool, err error) {
	key := normalizeBlockName(nm)
	if !matchesIsBlock(key) {
		// A name outside production [96] matches neither IsBlock nor IsCategory,
		// so "\p{Is}", "\p{IsThai$}" and "\p{Is.}" are no regExp at all
		// (§G.4.2.4). Such a name denotes no block, which puts it outside both
		// what §G.4.2.4 allows an unrecognized name and what errUnsupported
		// covers: a defect, not an unrecognized block.
		return nil, false, fmt.Errorf("malformed Unicode block name in \\p{Is%s}", nm)
	}
	r, ok := unicodeBlocks[key]
	if !ok {
		return runeSet{}.complement(), false, nil
	}
	return runeSet{r}, negate, nil
}

// matchesIsBlock reports whether name is admissible as the tail of an IsBlock:
// production [96] is IsBlock ::= 'Is' [a-zA-Z0-9#x2D]+, so one or more hyphens,
// digits and Basic Latin letters follow the 'Is' and nothing else. Applied to
// the name AFTER normalizeBlockName, so a whitespace or underbar this module
// strips per §G.4.2.3 does not itself make the pattern a defect.
func matchesIsBlock(name string) bool {
	const blockNameChars = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"
	if name == "" {
		return false
	}
	for _, r := range name {
		if !strings.ContainsRune(blockNameChars, r) {
			return false
		}
	}
	return true
}

// normalizeBlockName strips whitespace (#x9/#xA/#xD/#x20) and underbars while
// retaining hyphens and case (Datatypes §G.4.2.3).
func normalizeBlockName(nm string) string {
	var b strings.Builder
	for _, r := range nm {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// emitClass writes an RE2 character-class expression for the matched set. When
// neg is true the emitted class is negated (RE2 [^...]); the empty positive and
// empty negated cases can render as [^] / [], both invalid in RE2, so they are
// emitted as a never-match / match-any class over the whole universe instead.
func emitClass(b *strings.Builder, set runeSet, neg bool) {
	n := set.normalize()
	if len(n) == 0 {
		if neg {
			b.WriteString(`[\x{0}-\x{10ffff}]`)
			return
		}
		b.WriteString(`[^\x{0}-\x{10ffff}]`)
		return
	}
	b.WriteByte('[')
	if neg {
		b.WriteByte('^')
	}
	for _, r := range n {
		writeClassRune(b, r.lo)
		if r.hi != r.lo {
			b.WriteByte('-')
			writeClassRune(b, r.hi)
		}
	}
	b.WriteByte(']')
}

// writeClassRune writes a single code point inside a character class, escaping
// the class metacharacters and rendering anything outside printable ASCII as a
// \x{...} hex escape so the emitted class is unambiguous.
func writeClassRune(b *strings.Builder, r rune) {
	if r >= 0x20 && r < 0x7F && !strings.ContainsRune(`\^]-[`, r) {
		b.WriteRune(r)
		return
	}
	fmt.Fprintf(b, `\x{%x}`, r)
}
