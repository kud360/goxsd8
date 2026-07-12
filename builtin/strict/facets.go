package strict

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/kud360/goxsd8/regex"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file drives the pattern-facet and value-facet pipeline stages that run
// after the whiteSpace stage (whitespace.go) over the strict primitive cohort
// (decimal/boolean/string). The fixed stage sequence (value/doc.go "The facet
// pipeline", ARCHITECTURE.md) is:
//
//	whiteSpace → pattern (lexical) → lexical mapping (Parse) → value facets
//
// Pattern operates on the whiteSpace-normalized LEXICAL literal
// (cvc-pattern-valid, §4.3.4.4); enumeration/bounds/digits/length operate on
// the parsed VALUE (cvc-enumeration-valid §4.3.5.4, cvc-*Inclusive/*Exclusive
// §4.3.7–4.3.10, cvc-totalDigits/fractionDigits §4.3.11–4.3.12,
// cvc-length/minLength/maxLength §4.3.1–4.3.3). Each stage failure carries the
// SPECIFIC per-facet rule ID, never the cvc-facet-valid umbrella (§4.1.4).
//
// Compile-time assertions that the concrete checkers satisfy the pre-declared
// pipeline-stage interfaces (value/backend.go): compile builds
// []value.LexicalFacet and []value.ValueFacet and ValidateLexical ranges over
// them polymorphically, so the interface satisfaction has real call sites.
var (
	_ value.LexicalFacet = patternFacet{}
	_ value.ValueFacet   = enumFacet{}
	_ value.ValueFacet   = boundFacet{}
	_ value.ValueFacet   = digitsFacet{}
	_ value.ValueFacet   = lengthFacet{}
)

// ValidateLexical validates the lexical string rawLexical against st's effective
// facets through the full facet pipeline (whiteSpace → pattern → lexical mapping
// → value facets), returning the parsed value on success or the first
// *xsderr.Error a stage produces (stop-on-first-failure; this does not collect
// all facet violations). ctx is the VALIDATED INSTANCE's context, threaded to
// the governing mapping's Parse for the candidate value; the strict cohort
// (decimal/boolean/string) is context-free, so nil is fine here.
//
// PRECONDITION (caller-guarded, NOT checked here): every facet on st must be
// APPLICABLE to st's primitive ancestor (cos-applicable-facets §4.1.5), and st
// must have a primitive ancestor b maps (the decimal/boolean/string cohort).
// ValidateLexical PANICS — it does not return an error — when a value facet is
// paired with a value lacking the capability that facet needs (a bound facet on
// a non-Ordered value, a length facet on a non-Lengthed value, a digit facet on
// a non-DigitCounted value), or when st has no primitive ancestor. Those are
// schema-construction errors (st-restrict-facets / cos-applicable-facets) the
// caller must have already rejected, never instance data, so they surface as
// programming-error panics, not validity verdicts.
//
// Facet {value} parsing is a separate concern with its own scope: an inherited
// enumeration/bound facet's lexical {value} is parsed in the DECLARING SCHEMA's
// context (see newEnumFacet/newBoundFacet), which for this cohort is also
// context-free (nil). Future QName/NOTATION enumeration facets must not
// silently inherit the instance context here — that would resolve a facet
// literal's prefixes against the wrong scope.
func ValidateLexical(b value.Backend, st *xsd.SimpleType, rawLexical string, ctx value.Context) (value.Value, error) {
	lexFacets, valFacets, err := compile(b, st)
	if err != nil {
		return nil, err
	}

	// whiteSpace stage (§4.3.6): normalize using st's cohort mode, resolved from
	// its primitive ancestor (whiteSpace is fixed at the primitive for this
	// cohort — string=preserve, decimal/boolean=collapse).
	normalized := normalizeWhiteSpace(rawLexical, whiteSpaceOfType(st))

	// pattern (lexical) stage (cvc-pattern-valid, §4.3.4.4): checked on the
	// normalized lexical, before the value even exists.
	for _, lf := range lexFacets {
		if err := lf.CheckLexical(normalized); err != nil {
			return nil, err
		}
	}

	// lexical mapping: the candidate value is produced by st's OWN governing
	// mapping (its own, or its nearest mapped ancestor's — the widest-space rule
	// governs facet {value}s, not the application-facing candidate).
	m, ok := governingMapping(b, st)
	if !ok {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"strict: no backend mapping governs type %s", st.Name())
	}
	v, err := m.Parse(normalized, ctx)
	if err != nil {
		return nil, err
	}

	// value-facet stage: enumeration/bounds/digits/length on the parsed value.
	for _, vf := range valFacets {
		if err := vf.CheckValue(v); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// compile builds the pattern (lexical) and value facet checkers for st from its
// EffectiveFacets once, so pattern regexes are compiled and facet {value}s are
// parsed at this construction point — not per validated literal. A bad pattern
// or an unmappable declaring type surfaces here as an *xsderr.Error.
//
// The whiteSpace facet is consumed by the normalize stage, not as a checker;
// assertions and explicitTimezone are out of this cohort's scope (they never
// apply to decimal/boolean/string, cos-applicable-facets §4.1.5) and are
// skipped.
func compile(b value.Backend, st *xsd.SimpleType) ([]value.LexicalFacet, []value.ValueFacet, error) {
	var lexFacets []value.LexicalFacet
	var valFacets []value.ValueFacet
	for _, ef := range st.EffectiveFacets() {
		switch ef.Facet().Kind() {
		case xsd.FacetWhiteSpace:
			// Consumed by the whiteSpace normalize stage, not a checker.
		case xsd.FacetPattern:
			pf, err := newPatternFacet(ef.Facet())
			if err != nil {
				return nil, nil, err
			}
			lexFacets = append(lexFacets, pf)
		case xsd.FacetEnumeration:
			enf, err := newEnumFacet(b, st, ef)
			if err != nil {
				return nil, nil, err
			}
			valFacets = append(valFacets, enf)
		case xsd.FacetMaxInclusive, xsd.FacetMaxExclusive, xsd.FacetMinInclusive, xsd.FacetMinExclusive:
			bf, err := newBoundFacet(b, st, ef)
			if err != nil {
				return nil, nil, err
			}
			valFacets = append(valFacets, bf)
		case xsd.FacetTotalDigits, xsd.FacetFractionDigits:
			df, err := newDigitsFacet(ef.Facet())
			if err != nil {
				return nil, nil, err
			}
			valFacets = append(valFacets, df)
		case xsd.FacetLength, xsd.FacetMinLength, xsd.FacetMaxLength:
			lf, err := newLengthFacet(ef.Facet())
			if err != nil {
				return nil, nil, err
			}
			valFacets = append(valFacets, lf)
		case xsd.FacetAssertions, xsd.FacetExplicitTimezone:
			// Out of this cohort's scope (never applicable to decimal/boolean/
			// string; assertions are a separate later stage).
		}
	}
	return lexFacets, valFacets, nil
}

// primitiveOf returns st's primitive ancestor by walking the base chain until
// IsPrimitive (§2.4.2). It is nil only for the anySimpleType/anyAtomicType
// anchors, which are not in this cohort.
func primitiveOf(st *xsd.SimpleType) *xsd.SimpleType {
	for s := st; s != nil; s = s.Base() {
		if s.IsPrimitive() {
			return s
		}
	}
	return nil
}

// whiteSpaceOfType resolves st's whiteSpace mode from its primitive ancestor's
// per-type default in builtin.Types (§4.3.6). For this cohort whiteSpace is
// fixed at the primitive, so a derived type never overrides it; reading it off
// the primitive keeps the fact in one place (STYLE D3) rather than requiring a
// whiteSpace facet on every hand-built derived node. A type with no primitive
// ancestor is an internal-consistency failure (not a cohort type), never user
// input, so it panics.
func whiteSpaceOfType(st *xsd.SimpleType) whiteSpace {
	prim := primitiveOf(st)
	if prim == nil {
		panic(fmt.Sprintf("strict: type %s has no primitive ancestor", st.Name()))
	}
	return whiteSpaceOf(prim.Name().Local)
}

// governingMapping walks from node (inclusive) up the base chain and returns
// the first ancestor's Mapping the backend supplies. This is the widest-space
// resolution (value/backend.go, st-restrict-facets §3.16.6.4): a derived type
// without its own mapping is governed by its nearest mapped ancestor's.
func governingMapping(b value.Backend, node *xsd.SimpleType) (value.Mapping, bool) {
	for s := node; s != nil; s = s.Base() {
		if m, ok := b.Mapping(s.Name()); ok {
			return m, true
		}
	}
	return value.Mapping{}, false
}

// declaringMapping implements the widest-space rule (st-restrict-facets
// §3.16.6.4, value/backend.go) for an inherited facet: it finds the type named
// declaring on leaf's base chain, then returns the governing mapping FROM that
// type (its own, or its nearest mapped ancestor's) — never leaf's. A facet's
// lexical {value} is parsed in the value space of the type that DECLARES it, so
// a narrow derived representation can never corrupt an inherited bound/enum
// comparison (overflow, collapsed precision, different ordering).
//
// Types are matched by QName; anonymous declaring types (the zero QName) are
// outside this cohort's manually-built scope.
func declaringMapping(b value.Backend, leaf *xsd.SimpleType, declaring xsd.QName) (value.Mapping, bool) {
	for s := leaf; s != nil; s = s.Base() {
		if s.Name() == declaring {
			return governingMapping(b, s)
		}
	}
	return value.Mapping{}, false
}

// patternFacet is the pattern (lexical) stage (cvc-pattern-valid, §4.3.4.4).
// EffectiveFacets folds every derivation step's pattern union into at most one
// pattern facet whose {value} is the full OR-set; a literal is pattern-valid if
// it matches ANY member. The RE2 regexes are compiled once at construction.
type patternFacet struct {
	res []*regexp.Regexp
}

// newPatternFacet translates each XSD-flavor pattern value to RE2 and compiles
// it (regex.FlavorXSD is implicitly whole-string anchored; ^ and $ are literal
// characters, not anchors). A bad pattern surfaces here, not mid-validation.
func newPatternFacet(f xsd.Facet) (patternFacet, error) {
	values := f.Values()
	res := make([]*regexp.Regexp, 0, len(values))
	for _, p := range values {
		goRE, err := regex.Translate(p, regex.FlavorXSD, "")
		if err != nil {
			return patternFacet{}, err // already an *xsderr.Error (src-pattern-value)
		}
		re, err := regexp.Compile(goRE)
		if err != nil {
			return patternFacet{}, xsderr.Wrap("src-pattern-value", xsderr.Loc{}, err)
		}
		res = append(res, re)
	}
	return patternFacet{res: res}, nil
}

// CheckLexical accepts the normalized literal iff it matches at least one
// pattern in the OR-set (cvc-pattern-valid, §4.3.4.4).
func (p patternFacet) CheckLexical(normalized string) error {
	for _, re := range p.res {
		if re.MatchString(normalized) {
			return nil
		}
	}
	return xsderr.New("cvc-pattern-valid", xsderr.Loc{},
		"value %q matches no member of the pattern facet (cvc-pattern-valid, §4.3.4.4)", normalized)
}

// enumFacet is the enumeration value-facet stage (cvc-enumeration-valid,
// §4.3.5.4): a candidate is valid iff it is "equal or identical to one of the
// values specified in {value}". The members are parsed once, in the value space
// of the type that DECLARES the enumeration (widest-space rule).
type enumFacet struct {
	members []value.Value
}

// newEnumFacet parses each enumeration {value} lexical via the declaring type's
// mapping (widest-space rule, st-restrict-facets §3.16.6.4). The declaring
// schema's context is nil for this context-free cohort (see ValidateLexical).
func newEnumFacet(b value.Backend, st *xsd.SimpleType, ef xsd.EffectiveFacet) (enumFacet, error) {
	m, ok := declaringMapping(b, st, ef.Declaring())
	if !ok {
		return enumFacet{}, xsderr.New("cvc-enumeration-valid", xsderr.Loc{},
			"enumeration: no backend mapping governs declaring type %s", ef.Declaring())
	}
	values := ef.Facet().Values()
	members := make([]value.Value, 0, len(values))
	for _, lex := range values {
		v, err := m.Parse(lex, nil)
		if err != nil {
			return enumFacet{}, err
		}
		members = append(members, v)
	}
	return enumFacet{members: members}, nil
}

// CheckValue accepts v iff it is equal or identical to a member
// (cvc-enumeration-valid, §4.3.5.4).
func (e enumFacet) CheckValue(v value.Value) error {
	for _, member := range e.members {
		if enumMatch(v, member) {
			return nil
		}
	}
	return xsderr.New("cvc-enumeration-valid", xsderr.Loc{},
		"value is not equal or identical to any enumeration member (cvc-enumeration-valid, §4.3.5.4)")
}

// enumMatch reports the "equal or identical" relation cvc-enumeration-valid
// needs (§4.3.5.4). It prefers value.Identical (the identity relation:
// NaN identical to itself, +0 not identical to -0 — value/doc.go) when the
// candidate implements it, and unions it with value.Eq so an equal-but-not-
// identical member (e.g. +0 vs -0) still matches. A candidate with neither
// capability matches nothing.
func enumMatch(candidate, member value.Value) bool {
	if id, ok := candidate.(value.Identical); ok && id.Identical(member) {
		return true
	}
	if eq, ok := candidate.(value.Eq); ok && eq.Eq(member) {
		return true
	}
	return false
}

// boundFacet is one of the four bound value-facet stages
// (cvc-maxInclusive/maxExclusive/minInclusive/minExclusive-valid, §4.3.7–4.3.10).
// Only decimal (of this cohort) can carry these (cos-applicable-facets §4.1.5),
// so the limit and candidate both assert value.Ordered; an Incomparable Cmp is
// an internal-consistency bug (facet applicability not enforced upstream), a
// panic — never a legitimate spec rejection.
type boundFacet struct {
	limit value.Ordered
	kind  xsd.FacetKind
}

// newBoundFacet parses the single bound {value} via the declaring type's
// mapping (widest-space rule) and requires it to be value.Ordered.
func newBoundFacet(b value.Backend, st *xsd.SimpleType, ef xsd.EffectiveFacet) (boundFacet, error) {
	kind := ef.Facet().Kind()
	rule := boundRule(kind)
	m, ok := declaringMapping(b, st, ef.Declaring())
	if !ok {
		return boundFacet{}, xsderr.New(rule, xsderr.Loc{},
			"%s: no backend mapping governs declaring type %s", kind, ef.Declaring())
	}
	values := ef.Facet().Values()
	if len(values) != 1 {
		return boundFacet{}, xsderr.New(rule, xsderr.Loc{},
			"%s facet must carry exactly one value, has %d", kind, len(values))
	}
	v, err := m.Parse(values[0], nil)
	if err != nil {
		return boundFacet{}, err
	}
	ord, ok := v.(value.Ordered)
	if !ok {
		panic(fmt.Sprintf("strict: %s facet value %q is not value.Ordered (cos-applicable-facets §4.1.5 not enforced upstream)", kind, values[0]))
	}
	return boundFacet{limit: ord, kind: kind}, nil
}

// CheckValue rejects a candidate that violates the bound (§4.3.7–4.3.10).
func (bf boundFacet) CheckValue(v value.Value) error {
	cand, ok := v.(value.Ordered)
	if !ok {
		panic(fmt.Sprintf("strict: candidate %T under a %s facet is not value.Ordered (cos-applicable-facets §4.1.5 not enforced upstream)", v, bf.kind))
	}
	ord := cand.Cmp(bf.limit)
	if ord == value.Incomparable {
		panic(fmt.Sprintf("strict: %s facet comparison is Incomparable (facet applicability not enforced upstream)", bf.kind))
	}
	if bf.violates(ord) {
		return xsderr.New(boundRule(bf.kind), xsderr.Loc{},
			"value violates the %s facet (%s, §4.3.7–4.3.10)", bf.kind, boundRule(bf.kind))
	}
	return nil
}

// violates maps the candidate-vs-limit ordering to a bound violation per kind.
func (bf boundFacet) violates(ord value.Ordering) bool {
	switch bf.kind {
	case xsd.FacetMaxInclusive:
		return ord == value.Greater
	case xsd.FacetMaxExclusive:
		return ord == value.Greater || ord == value.Equal
	case xsd.FacetMinInclusive:
		return ord == value.Less
	case xsd.FacetMinExclusive:
		return ord == value.Less || ord == value.Equal
	default:
		panic(fmt.Sprintf("strict: violates: %s is not a bound facet", bf.kind))
	}
}

// boundRule maps a bound facet kind to its per-facet rule ID (§4.3.7–4.3.10).
func boundRule(k xsd.FacetKind) xsderr.Rule {
	switch k {
	case xsd.FacetMaxInclusive:
		return "cvc-maxInclusive-valid"
	case xsd.FacetMaxExclusive:
		return "cvc-maxExclusive-valid"
	case xsd.FacetMinInclusive:
		return "cvc-minInclusive-valid"
	case xsd.FacetMinExclusive:
		return "cvc-minExclusive-valid"
	default:
		panic(fmt.Sprintf("strict: boundRule: %s is not a bound facet", k))
	}
}

// digitsFacet is the totalDigits/fractionDigits value-facet stage
// (cvc-totalDigits-valid §4.3.11.3, cvc-fractionDigits-valid §4.3.12.3),
// decimal-only (cos-applicable-facets §4.1.5). Both are UPPER-BOUND (magnitude)
// constraints, not exact-count matches: violation is candidate digits > limit.
type digitsFacet struct {
	limit int
	kind  xsd.FacetKind
}

// newDigitsFacet reads the facet's plain nonNegativeInteger {value} — a count,
// not a value in the declaring type's space, so no declaring-mapping lookup.
func newDigitsFacet(f xsd.Facet) (digitsFacet, error) {
	rule := digitsRule(f.Kind())
	n, err := facetCount(f, rule)
	if err != nil {
		return digitsFacet{}, err
	}
	return digitsFacet{limit: n, kind: f.Kind()}, nil
}

// CheckValue rejects a candidate whose digit count exceeds the limit
// (§4.3.11.3/§4.3.12.3).
func (df digitsFacet) CheckValue(v value.Value) error {
	dc, ok := v.(value.DigitCounted)
	if !ok {
		panic(fmt.Sprintf("strict: candidate %T under a %s facet is not value.DigitCounted (cos-applicable-facets §4.1.5 not enforced upstream)", v, df.kind))
	}
	got := dc.TotalDigits()
	if df.kind == xsd.FacetFractionDigits {
		got = dc.FractionDigits()
	}
	if got > df.limit {
		return xsderr.New(digitsRule(df.kind), xsderr.Loc{},
			"value has %d %s, exceeds facet limit %d (%s)", got, df.kind, df.limit, digitsRule(df.kind))
	}
	return nil
}

// digitsRule maps a digit facet kind to its per-facet rule ID.
func digitsRule(k xsd.FacetKind) xsderr.Rule {
	switch k {
	case xsd.FacetTotalDigits:
		return "cvc-totalDigits-valid"
	case xsd.FacetFractionDigits:
		return "cvc-fractionDigits-valid"
	default:
		panic(fmt.Sprintf("strict: digitsRule: %s is not a digit facet", k))
	}
}

// lengthFacet is the length/minLength/maxLength value-facet stage
// (cvc-length-valid §4.3.1.3, cvc-minLength-valid §4.3.2.3, cvc-maxLength-valid
// §4.3.3.3). For string the unit is Unicode codepoints (value.Lengthed.Len).
type lengthFacet struct {
	limit int
	kind  xsd.FacetKind
}

// newLengthFacet reads the facet's plain nonNegativeInteger {value} (a count),
// so no declaring-mapping lookup.
func newLengthFacet(f xsd.Facet) (lengthFacet, error) {
	rule := lengthRule(f.Kind())
	n, err := facetCount(f, rule)
	if err != nil {
		return lengthFacet{}, err
	}
	return lengthFacet{limit: n, kind: f.Kind()}, nil
}

// CheckValue rejects a candidate whose length violates the facet
// (§4.3.1.3–4.3.3.3).
func (lf lengthFacet) CheckValue(v value.Value) error {
	l, ok := v.(value.Lengthed)
	if !ok {
		panic(fmt.Sprintf("strict: candidate %T under a %s facet is not value.Lengthed (cos-applicable-facets §4.1.5 not enforced upstream)", v, lf.kind))
	}
	if lf.violates(l.Len()) {
		return xsderr.New(lengthRule(lf.kind), xsderr.Loc{},
			"value length %d violates the %s facet limit %d (%s)", l.Len(), lf.kind, lf.limit, lengthRule(lf.kind))
	}
	return nil
}

// violates maps a length to a violation per kind.
func (lf lengthFacet) violates(n int) bool {
	switch lf.kind {
	case xsd.FacetLength:
		return n != lf.limit
	case xsd.FacetMinLength:
		return n < lf.limit
	case xsd.FacetMaxLength:
		return n > lf.limit
	default:
		panic(fmt.Sprintf("strict: violates: %s is not a length facet", lf.kind))
	}
}

// lengthRule maps a length facet kind to its per-facet rule ID.
func lengthRule(k xsd.FacetKind) xsderr.Rule {
	switch k {
	case xsd.FacetLength:
		return "cvc-length-valid"
	case xsd.FacetMinLength:
		return "cvc-minLength-valid"
	case xsd.FacetMaxLength:
		return "cvc-maxLength-valid"
	default:
		panic(fmt.Sprintf("strict: lengthRule: %s is not a length facet", k))
	}
}

// facetCount parses a single-valued facet's plain xs:nonNegativeInteger {value}
// (a bare count for the digit and length facets), charging rule on rejection.
func facetCount(f xsd.Facet, rule xsderr.Rule) (int, error) {
	values := f.Values()
	if len(values) != 1 {
		return 0, xsderr.New(rule, xsderr.Loc{},
			"%s facet must carry exactly one value, has %d", f.Kind(), len(values))
	}
	n, err := strconv.Atoi(values[0])
	if err != nil || n < 0 {
		return 0, xsderr.New(rule, xsderr.Loc{},
			"%s facet value %q is not a nonNegativeInteger", f.Kind(), values[0])
	}
	return n, nil
}
