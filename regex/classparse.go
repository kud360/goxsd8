package regex

import "unicode/utf8"

// parseClassBody parses a charClassExpr (Datatypes productions [75]-[82])
// starting at '[' and returns a member set together with whether emitClass
// negates it. A negCharGroup with no subtraction returns its positive set and
// true, so a common negated class stays a compact RE2 [^...]. Production [76]
// charGroup ::= ( posCharGroup | negCharGroup ) ( '-' charClassExpr )? makes
// G-C "the set of all characters in C(G) that are not in C(C)" with G possibly
// a negCharGroup, so the subtraction applies after the negation: [^P-[C]]
// materializes (complement of P) minus C and returns false (cvc-pattern-valid,
// #1856).
//
// GAP(regex): under FlavorFO with flag i (p.foldCase) [^P-[C]] keeps the
// subtraction inside the negation, so it matches the complement of (P minus C)
// and admits every member of C, which the spec's set excludes. A materialized
// complement is no fix there: RE2's (?i) folds it, so [^Q-[x]]'s complement
// carries q and matches Q again. FlavorFO with flag i therefore keeps its
// earlier emission on every route, nested operands included, until #2148 makes
// the subtraction fold-aware; the rule-4 deviation that (?i) folds \p{..} and
// \w is #2147's.
func (p *parser) parseClassBody() (runeSet, bool, error) {
	open := p.pos
	p.pos++ // consume '['
	neg := false
	if p.peek() == '^' {
		neg = true
		p.pos++
	}
	var set runeSet
	count := 0
	for {
		c := p.peek()
		if c == -1 {
			return nil, false, p.errf(open, "unclosed character class '['")
		}
		if c == ']' {
			break
		}
		if c == '-' && p.peekAt(1) == '[' {
			break
		}
		part, err := p.charGroupPart(open)
		if err != nil {
			return nil, false, err
		}
		set = set.union(part)
		count++
	}
	if count == 0 {
		return nil, false, p.errf(open, "empty character class")
	}
	if p.peek() == '-' && p.peekAt(1) == '[' {
		p.pos++ // consume the subtraction '-'
		sub, err := p.classMatchedSet()
		if err != nil {
			return nil, false, err
		}
		if neg && !p.foldCase {
			set = set.complement()
			neg = false
		}
		set = set.subtract(sub)
	}
	if p.peek() != ']' {
		return nil, false, p.errf(p.pos, "expected ']' to close a character class")
	}
	p.pos++
	return set, neg, nil
}

// classMatchedSet parses a nested charClassExpr used as a subtraction operand
// and returns the actual set of characters it matches (its own negation
// applied).
func (p *parser) classMatchedSet() (runeSet, error) {
	if p.peek() != '[' {
		return nil, p.errf(p.pos, "expected '[' after a character-class subtraction operator")
	}
	set, neg, err := p.parseClassBody()
	if err != nil {
		return nil, err
	}
	if neg {
		return set.complement(), nil
	}
	return set, nil
}

// charGroupPart parses one charGroupPart (production [79]): a single character
// (escaped or not, possibly starting a range) or a character-class escape.
func (p *parser) charGroupPart(open int) (runeSet, error) {
	if p.peek() == '\\' {
		r, isRune, set, err := p.parseClassEscape(open)
		if err != nil {
			return nil, err
		}
		if !isRune {
			return set, nil
		}
		return p.maybeRange(r, true, open)
	}
	if p.peek() == '[' {
		return nil, p.errf(p.pos, "'[' must be escaped inside a character class")
	}
	r, size := utf8.DecodeRuneInString(p.in[p.pos:])
	if r == utf8.RuneError && size <= 1 {
		return nil, p.errf(p.pos, "invalid UTF-8 in character class")
	}
	p.pos += size
	return p.maybeRange(r, false, open)
}

// maybeRange decides whether the singleChar just read (lo) begins a charRange,
// applying the hyphen-disambiguation rules of Datatypes §G.4.1: a '-' before
// '[', ']', or '-[' is not a range operator, and an unescaped '-' may be
// neither endpoint of a range (which excludes strings like [--z]).
func (p *parser) maybeRange(lo rune, loEsc bool, open int) (runeSet, error) {
	if p.peek() != '-' {
		return runeSet{{lo, lo}}, nil
	}
	n1 := p.peekAt(1)
	if n1 == '[' || n1 == ']' {
		return runeSet{{lo, lo}}, nil
	}
	if n1 == '-' && p.peekAt(2) == '[' {
		return runeSet{{lo, lo}}, nil
	}
	if n1 == -1 {
		return nil, p.errf(p.pos, "'-' at end of character class")
	}
	dash := p.pos
	if !loEsc && lo == '-' {
		return nil, p.errf(dash, "unescaped '-' cannot start a character range")
	}
	p.pos++ // consume the range '-'
	hi, hiEsc, err := p.singleCharInClass(open)
	if err != nil {
		return nil, err
	}
	if !hiEsc && hi == '-' {
		return nil, p.errf(dash, "unescaped '-' cannot end a character range")
	}
	if hi < lo {
		return nil, p.errf(dash, "character range %q-%q is out of order", string(lo), string(hi))
	}
	return runeSet{{lo, hi}}, nil
}

// singleCharInClass reads a single character (escaped or not) serving as the
// upper endpoint of a charRange; a multi-character class escape is not a valid
// endpoint.
func (p *parser) singleCharInClass(open int) (rune, bool, error) {
	if p.peek() == -1 {
		return 0, false, p.errf(open, "unclosed character class")
	}
	if p.peek() == '\\' {
		r, isRune, _, err := p.parseClassEscape(open)
		if err != nil {
			return 0, false, err
		}
		if !isRune {
			return 0, false, p.errf(open, "a character-class escape cannot be a range endpoint")
		}
		return r, true, nil
	}
	if p.peek() == '[' || p.peek() == ']' {
		return 0, false, p.errf(p.pos, "%q must be escaped inside a character class", string(rune(p.peek())))
	}
	r, size := utf8.DecodeRuneInString(p.in[p.pos:])
	if r == utf8.RuneError && size <= 1 {
		return 0, false, p.errf(p.pos, "invalid UTF-8 in character class")
	}
	p.pos += size
	return r, false, nil
}

// parseClassEscape parses a '\'-led token inside a character class, returning
// either a single code point (isRune true) or the set denoted by a
// character-class escape. A '\'-digit sequence is invalid here (F&O §7.6.1).
func (p *parser) parseClassEscape(open int) (rune, bool, runeSet, error) {
	start := p.pos
	p.pos++
	if p.pos >= len(p.in) {
		return 0, false, nil, p.errf(start, "trailing backslash in character class")
	}
	c := p.in[p.pos]
	switch {
	case c == 'p' || c == 'P':
		name, err := p.parseCharProp(start)
		if err != nil {
			return 0, false, nil, err
		}
		set, err := propSet(name, c == 'P')
		if err != nil {
			return 0, false, nil, p.errCause(start, err)
		}
		return 0, false, set, nil
	case c >= '0' && c <= '9':
		return 0, false, nil, p.errf(start, "\\%c is not allowed inside a character class", c)
	case c == 's' || c == 'S' || c == 'd' || c == 'D' || c == 'w' || c == 'W':
		p.pos++
		return 0, false, multiEscSet(c), nil
	case c == 'i' || c == 'I' || c == 'c' || c == 'C':
		p.pos++
		return 0, false, multiEscSet(c), nil
	case isSingleCharEscByte(c, p.flavor):
		p.pos++
		return singleCharEscRune(c), true, nil, nil
	default:
		return 0, false, nil, p.errf(start, "\\%c is not a valid escape", c)
	}
}
