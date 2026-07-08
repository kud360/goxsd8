package regex

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
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

// addTable folds every interval of a unicode.RangeTable into the set.
func (s runeSet) addTable(t *unicode.RangeTable) runeSet {
	for _, r := range t.R16 {
		for c := rune(r.Lo); c <= rune(r.Hi); c += rune(r.Stride) {
			s = s.add(c, c)
		}
	}
	for _, r := range t.R32 {
		for c := rune(r.Lo); c <= rune(r.Hi); c += rune(r.Stride) {
			s = s.add(c, c)
		}
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
// (\d \D \s \S \w \W) as it appears inside a character-class expression, where
// no symbolic RE2 form is available and the set must be materialized.
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
	}
	return nil
}

// propSet returns the code-point set named by a \p{...}/\P{...} property body
// (Datatypes §G.4.2.2/§G.4.2.3): a Unicode general category (L, Lu, Nd, …) or a
// block escape (IsBasicLatin, …). An unrecognized category or block name is a
// hard error, never a fail-open to "all characters": goxsd8 must not accept a
// literal that the pattern as written would reject (PRINCIPLES 20; the spec's
// §G.4.2.4 permissive default is deliberately not taken).
func propSet(name string) (runeSet, error) {
	if rest, ok := strings.CutPrefix(name, "Is"); ok {
		return blockSet(rest)
	}
	t, ok := unicode.Categories[name]
	if !ok {
		return nil, fmt.Errorf("unrecognized Unicode category %q", name)
	}
	return runeSet{}.addTable(t), nil
}

// isCategoryName reports whether name is a Unicode general category that RE2
// can express directly as \p{name}, so a standalone category escape can pass
// through instead of being materialized.
func isCategoryName(name string) bool {
	_, ok := unicode.Categories[name]
	return ok
}

// blockSet returns the code points of the Unicode block whose normalized name
// (Datatypes §G.4.2.3: whitespace and underbars stripped, hyphens and case
// retained) matches nm. Go's standard library exposes categories and scripts
// but not blocks, and the block ranges are drawn from the Unicode database
// rather than from the local goxsd8 specs, so unicodeBlocks is a curated,
// hand-authored subset of high-frequency Appendix G blocks. An unrecognized
// block name is an error (see propSet).
func blockSet(nm string) (runeSet, error) {
	key := normalizeBlockName(nm)
	r, ok := unicodeBlocks[key]
	if !ok {
		return nil, fmt.Errorf("unrecognized or unsupported Unicode block %q", nm)
	}
	return runeSet{r}, nil
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

// unicodeBlocks maps normalized Unicode block names to their code-point ranges.
// Hand-authored (see blockSet) subset of the blocks enumerated in the Unicode
// database referenced by Datatypes Appendix G; the superseded XSD 1.0 aliases
// the spec lists (Greek, PrivateUse, CombiningMarksforSymbols) are included for
// compatibility.
var unicodeBlocks = map[string]runeRange{
	"BasicLatin":                {0x0000, 0x007F},
	"Latin-1Supplement":         {0x0080, 0x00FF},
	"LatinExtended-A":           {0x0100, 0x017F},
	"LatinExtended-B":           {0x0180, 0x024F},
	"IPAExtensions":             {0x0250, 0x02AF},
	"SpacingModifierLetters":    {0x02B0, 0x02FF},
	"CombiningDiacriticalMarks": {0x0300, 0x036F},
	"GreekandCoptic":            {0x0370, 0x03FF},
	"Greek":                     {0x0370, 0x03FF},
	"Cyrillic":                  {0x0400, 0x04FF},
	"Hebrew":                    {0x0590, 0x05FF},
	"Arabic":                    {0x0600, 0x06FF},
	"GeneralPunctuation":        {0x2000, 0x206F},
	"SuperscriptsandSubscripts": {0x2070, 0x209F},
	"CurrencySymbols":           {0x20A0, 0x20CF},
	"CombiningMarksforSymbols":  {0x20D0, 0x20FF},
	"LetterlikeSymbols":         {0x2100, 0x214F},
	"NumberForms":               {0x2150, 0x218F},
	"Arrows":                    {0x2190, 0x21FF},
	"MathematicalOperators":     {0x2200, 0x22FF},
	"Hiragana":                  {0x3040, 0x309F},
	"Katakana":                  {0x30A0, 0x30FF},
	"CJKUnifiedIdeographs":      {0x4E00, 0x9FFF},
	"PrivateUse":                {0xE000, 0xF8FF},
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
