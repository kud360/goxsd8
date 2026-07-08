package regex

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kud360/goxsd8/xsderr"
)

// Flavor selects the regular-expression dialect Translate parses. The two
// flavors share one grammar family but differ in anchoring, the meaning of
// ^/$, group capture, and flag handling (PRINCIPLES 10). The zero value is
// invalid so an unset Flavor is a caught bug (see builtin.Ordered); use
// FlavorXSD or FlavorFO.
type Flavor uint8

// The Flavor values.
const (
	// FlavorXSD is the XML Schema pattern-facet dialect (Datatypes Appendix G):
	// the whole pattern is implicitly anchored, ^ and $ are literal characters,
	// groups are non-capturing, no flags are accepted, and . excludes \n and \r.
	FlavorXSD Flavor = iota + 1
	// FlavorFO is the XPath/XQuery Functions & Operators dialect used by
	// fn:matches, fn:replace, and fn:tokenize (F&O §7.6.1): unanchored unless
	// the pattern anchors itself, ^ and $ are real anchors, groups capture, the
	// flags i/s/m/x are honored, and . excludes only \n unless the s flag is set.
	FlavorFO
)

// Rule IDs attached to translation failures (STYLE E2). XSD-flavor failures are
// schema-authoring-time pattern errors (Datatypes §4.3.4.3); FO-flavor failures
// are the F&O dynamic errors for an invalid pattern or invalid flags.
const (
	ruleXSDPattern xsderr.Rule = "src-pattern-value"
	ruleFOPattern  xsderr.Rule = "err:FORX0002"
	ruleFOFlags    xsderr.Rule = "err:FORX0001"
)

// maxRepeat is the largest counted-repetition bound Go's regexp/syntax accepts;
// {n,m} with n or m above it is rejected by the RE2 compiler. The translator
// enforces the ceiling itself so the failure is attributable to goxsd8 and so a
// too-large count never silently truncates into a false accept (a truncated
// bound would admit literals the pattern as written rejects). Not spec-mandated
// (see .agent/grounding-issue-12.md §4).
const maxRepeat = 1000

// Translate converts a pattern in the given flavor to an equivalent Go RE2
// (regexp) source string, or returns an *xsderr.Error carrying the offending
// construct and its byte offset. The result is deterministic; compiling and
// caching it is the caller's concern. flags is honored only for FlavorFO; it
// must be empty for FlavorXSD. An invalid Flavor, or a non-empty flags with
// FlavorXSD, is a caller-contract violation (a programming error, not a
// pattern error) and panics rather than returning an error.
func Translate(pattern string, flavor Flavor, flags string) (string, error) {
	switch flavor {
	case FlavorXSD:
		if flags != "" {
			panic(fmt.Sprintf("regex: XSD-flavor Translate accepts no flags, got %q", flags))
		}
		p := &parser{in: pattern, flavor: flavor}
		if err := p.translateBody(); err != nil {
			return "", err
		}
		return `\A(?:` + p.out.String() + `)\z`, nil
	case FlavorFO:
		prefix, stripX, err := foFlags(flags)
		if err != nil {
			return "", err
		}
		in := pattern
		if stripX {
			in = stripInsignificantWhitespace(pattern)
		}
		p := &parser{in: in, flavor: flavor}
		if err := p.translateBody(); err != nil {
			return "", err
		}
		return prefix + p.out.String(), nil
	default:
		panic(fmt.Sprintf("regex: invalid Flavor %d", flavor))
	}
}

// foFlags validates an F&O $flags string and returns the RE2 inline-flag prefix
// to prepend, whether x-mode whitespace stripping applies, or an err:FORX0001
// for any character that is not one of the defined flags i/s/m/x (F&O §7.6.1.1).
// There is no q flag in the local spec, so q takes the generic unrecognized-flag
// path like any other bad character (see .agent/grounding-issue-12.md).
func foFlags(flags string) (prefix string, stripX bool, err error) {
	var i, s, m bool
	for off, c := range flags {
		switch c {
		case 'i':
			i = true
		case 's':
			s = true
		case 'm':
			m = true
		case 'x':
			stripX = true
		default:
			return "", false, xsderr.New(ruleFOFlags, xsderr.Loc{}, "regex: unrecognized flag %q at byte offset %d", string(c), off)
		}
	}
	var b strings.Builder
	if i {
		b.WriteByte('i')
	}
	if s {
		b.WriteByte('s')
	}
	if m {
		b.WriteByte('m')
	}
	if b.Len() == 0 {
		return "", stripX, nil
	}
	return "(?" + b.String() + ")", stripX, nil
}

// stripInsignificantWhitespace removes #x9/#xA/#xD/#x20 from a pattern except
// inside character-class expressions, implementing the F&O x flag (§7.6.1.1). A
// backslash escapes the following byte so an escaped bracket does not change
// class depth; whitespace is significant at any bracket depth above zero.
func stripInsignificantWhitespace(s string) string {
	var b strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			b.WriteByte(c)
			b.WriteByte(s[i+1])
			i++
			continue
		}
		if c == '[' {
			depth++
		}
		if c == ']' && depth > 0 {
			depth--
		}
		if depth == 0 && (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// parser is a single-pass recursive-descent translator over one pattern. It
// reads in[pos:] and appends the RE2 translation to out.
type parser struct {
	in     string
	pos    int
	flavor Flavor
	out    strings.Builder
}

// translateBody parses a whole regExp and requires that it consume the entire
// input; a leftover ')' or other trailing text is a translation error.
func (p *parser) translateBody() error {
	if err := p.regExp(); err != nil {
		return err
	}
	if p.pos != len(p.in) {
		return p.errf(p.pos, "unexpected %q", string(p.in[p.pos]))
	}
	return nil
}

func (p *parser) peek() int {
	if p.pos >= len(p.in) {
		return -1
	}
	return int(p.in[p.pos])
}

func (p *parser) peekAt(n int) int {
	i := p.pos + n
	if i >= len(p.in) {
		return -1
	}
	return int(p.in[i])
}

func (p *parser) rule() xsderr.Rule {
	if p.flavor == FlavorXSD {
		return ruleXSDPattern
	}
	return ruleFOPattern
}

func (p *parser) errf(off int, format string, args ...any) error {
	return xsderr.New(p.rule(), xsderr.Loc{}, "regex: %s (offset %d)", fmt.Sprintf(format, args...), off)
}

// regExp ::= branch ( '|' branch )* (Datatypes production [64]).
func (p *parser) regExp() error {
	if err := p.branch(); err != nil {
		return err
	}
	for p.peek() == '|' {
		p.pos++
		p.out.WriteByte('|')
		if err := p.branch(); err != nil {
			return err
		}
	}
	return nil
}

// branch ::= piece* (production [65]); it stops at end of input, '|', or the
// ')' that closes an enclosing group.
func (p *parser) branch() error {
	for {
		c := p.peek()
		if c == -1 || c == '|' || c == ')' {
			return nil
		}
		if err := p.piece(); err != nil {
			return err
		}
	}
}

// piece ::= atom quantifier? (production [66]).
func (p *parser) piece() error {
	if err := p.atom(); err != nil {
		return err
	}
	return p.quantifier()
}

// atom ::= NormalChar | charClass | '(' regExp ')' (production [72]); the F&O
// flavor adds ^/$ anchors and rejects the back-reference it also adds.
func (p *parser) atom() error {
	start := p.pos
	c := p.peek()
	switch c {
	case '(':
		p.pos++
		if p.flavor == FlavorXSD {
			p.out.WriteString("(?:")
		}
		if p.flavor == FlavorFO {
			p.out.WriteByte('(')
		}
		if err := p.regExp(); err != nil {
			return err
		}
		if p.peek() != ')' {
			return p.errf(start, "unclosed group '('")
		}
		p.pos++
		p.out.WriteByte(')')
		return nil
	case '[':
		set, neg, err := p.parseClassBody()
		if err != nil {
			return err
		}
		emitClass(&p.out, set, neg)
		return nil
	case '.':
		p.pos++
		if p.flavor == FlavorXSD {
			p.out.WriteString(`[^\n\r]`)
			return nil
		}
		p.out.WriteByte('.')
		return nil
	case '\\':
		return p.atomEscape()
	case '^', '$':
		p.pos++
		if p.flavor == FlavorFO {
			p.out.WriteByte(byte(c))
			return nil
		}
		p.out.WriteByte('\\')
		p.out.WriteByte(byte(c))
		return nil
	case '*', '+', '?', '{', '}', ']', ')', '|':
		return p.errf(start, "%q is not a valid atom", string(rune(c)))
	default:
		r, size := utf8.DecodeRuneInString(p.in[p.pos:])
		if r == utf8.RuneError && size <= 1 {
			return p.errf(start, "invalid UTF-8 in pattern")
		}
		p.pos += size
		writeLiteralRune(&p.out, r)
		return nil
	}
}

// atomEscape handles a '\'-led atom: category escapes pass through when RE2
// supports them and are materialized otherwise, multi-character escapes expand
// to their defining classes, single-character escapes become literals, and a
// back-reference (\N) is rejected as untranslatable (F&O §7.6.1: legal grammar,
// but no RE2 equivalent, so a real error rather than a fail-open).
func (p *parser) atomEscape() error {
	start := p.pos
	p.pos++
	if p.pos >= len(p.in) {
		return p.errf(start, "trailing backslash")
	}
	c := p.in[p.pos]
	switch {
	case c == 'p' || c == 'P':
		return p.atomCategoryEscape(c == 'P', start)
	case c >= '1' && c <= '9':
		return p.errf(start, "back-reference \\%c is not expressible in RE2", c)
	case c == '0':
		return p.errf(start, "\\0 is not a valid escape")
	case strings.IndexByte("sSdDwW", c) >= 0:
		p.pos++
		p.out.WriteString(atomMultiEsc(c))
		return nil
	case c == 'i' || c == 'I' || c == 'c' || c == 'C':
		return p.errf(start, "\\%c (XML NameChar class) is not supported", c)
	case isSingleCharEscByte(c, p.flavor):
		p.pos++
		writeLiteralRune(&p.out, singleCharEscRune(c))
		return nil
	default:
		return p.errf(start, "\\%c is not a valid escape", c)
	}
}

// atomCategoryEscape emits a standalone \p{...}/\P{...} escape. A general
// category passes through to RE2 unchanged; a block escape (\p{IsX}) has no RE2
// form and is emitted as an explicit range class.
func (p *parser) atomCategoryEscape(negate bool, start int) error {
	name, err := p.parseCharProp(start)
	if err != nil {
		return err
	}
	if strings.HasPrefix(name, "Is") {
		set, err := blockSet(strings.TrimPrefix(name, "Is"))
		if err != nil {
			return p.errf(start, "%v", err)
		}
		emitClass(&p.out, set, negate)
		return nil
	}
	if !isCategoryName(name) {
		return p.errf(start, "unrecognized Unicode category \\p{%s}", name)
	}
	p.out.WriteByte('\\')
	if negate {
		p.out.WriteByte('P')
	}
	if !negate {
		p.out.WriteByte('p')
	}
	p.out.WriteByte('{')
	p.out.WriteString(name)
	p.out.WriteByte('}')
	return nil
}

// parseCharProp reads the '{name}' body of a \p/\P escape, leaving pos after the
// closing brace. It expects pos at the 'p' or 'P'.
func (p *parser) parseCharProp(start int) (string, error) {
	p.pos++
	if p.peek() != '{' {
		return "", p.errf(start, "expected '{' after \\%c", p.in[start+1])
	}
	p.pos++
	j := strings.IndexByte(p.in[p.pos:], '}')
	if j < 0 {
		return "", p.errf(start, "unterminated \\p{...}")
	}
	name := p.in[p.pos : p.pos+j]
	p.pos += j + 1
	if name == "" {
		return "", p.errf(start, "empty \\p{}")
	}
	return name, nil
}

// quantifier ::= ( [?*+] | '{' quantity '}' ) with an optional trailing '?' for
// the F&O reluctant form (production [67], F&O §7.6.1). RE2 accepts the greedy
// and reluctant operators directly.
func (p *parser) quantifier() error {
	c := p.peek()
	switch c {
	case '?', '*', '+':
		p.pos++
		p.out.WriteByte(byte(c))
	case '{':
		if err := p.quantity(); err != nil {
			return err
		}
	default:
		return nil
	}
	if p.flavor == FlavorFO && p.peek() == '?' {
		p.pos++
		p.out.WriteByte('?')
	}
	return nil
}

// quantity ::= QuantExact ( ',' QuantExact? )? (productions [68]-[71]), with the
// maxRepeat ceiling enforced before the bounds reach the RE2 compiler.
func (p *parser) quantity() error {
	start := p.pos
	p.pos++
	n, ok := p.readInt()
	if !ok {
		return p.errf(start, "expected a repetition count after '{'")
	}
	if n > maxRepeat {
		return p.errf(start, "repetition count %d exceeds the RE2 limit of %d", n, maxRepeat)
	}
	p.out.WriteByte('{')
	p.out.WriteString(strconv.Itoa(n))
	if p.peek() == ',' {
		p.pos++
		p.out.WriteByte(',')
		if p.peek() >= '0' && p.peek() <= '9' {
			m, _ := p.readInt()
			if m > maxRepeat {
				return p.errf(start, "repetition count %d exceeds the RE2 limit of %d", m, maxRepeat)
			}
			if m < n {
				return p.errf(start, "repetition range {%d,%d} is out of order", n, m)
			}
			p.out.WriteString(strconv.Itoa(m))
		}
	}
	if p.peek() != '}' {
		return p.errf(p.pos, "expected '}' to close a repetition")
	}
	p.pos++
	p.out.WriteByte('}')
	return nil
}

// readInt reads a run of digits as a non-negative int, clamping above maxRepeat
// so an oversized count is caught by the ceiling check rather than overflowing.
func (p *parser) readInt() (int, bool) {
	start := p.pos
	v := 0
	for p.peek() >= '0' && p.peek() <= '9' {
		v = v*10 + int(p.in[p.pos]-'0')
		if v > maxRepeat {
			v = maxRepeat + 1
		}
		p.pos++
	}
	return v, p.pos > start
}

// atomMultiEsc returns the RE2 form of a standalone multi-character escape
// (Datatypes §G.4.2.5). \d/\D map to \p{Nd}/\P{Nd}; \w/\W expand to the
// complement/union of [\p{P}\p{Z}\p{C}] (\w's spec definition subtracts these
// from the whole Unicode range, which is exactly a complement); \s/\S expand to
// the explicit whitespace class.
func atomMultiEsc(c byte) string {
	switch c {
	case 'd':
		return `\p{Nd}`
	case 'D':
		return `\P{Nd}`
	case 's':
		return `[\x{9}\x{a}\x{d}\x{20}]`
	case 'S':
		return `[^\x{9}\x{a}\x{d}\x{20}]`
	case 'w':
		return `[^\p{P}\p{Z}\p{C}]`
	case 'W':
		return `[\p{P}\p{Z}\p{C}]`
	}
	return ""
}

// isSingleCharEscByte reports whether \c is a SingleCharEsc (Datatypes
// production [84]); the F&O flavor additionally allows \$ (F&O §7.6.1).
func isSingleCharEscByte(c byte, flavor Flavor) bool {
	if strings.IndexByte(`nrt\|.?*+(){}-[]^`, c) >= 0 {
		return true
	}
	return flavor == FlavorFO && c == '$'
}

// singleCharEscRune returns the code point a SingleCharEsc denotes.
func singleCharEscRune(c byte) rune {
	switch c {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	}
	return rune(c)
}

// writeLiteralRune writes a code point as a literal atom, backslash-escaping RE2
// metacharacters and hex-escaping anything outside printable ASCII so the output
// is unambiguous regardless of the input character.
func writeLiteralRune(b *strings.Builder, r rune) {
	if r < 0x20 || r == 0x7F {
		fmt.Fprintf(b, `\x{%x}`, r)
		return
	}
	if r < 0x7F {
		if strings.ContainsRune(`.\+*?()|[]{}^$`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
		return
	}
	b.WriteRune(r)
}
