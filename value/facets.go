package value

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/kud360/goxsd8/regex"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file drives the backend-generic facet pipeline: the pattern (lexical)
// and value-facet stages that run after the whiteSpace stage (whitespace.go)
// over an atomic type's effective facets. The fixed stage sequence (doc.go "The
// facet pipeline", ARCHITECTURE.md) is:
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
// Compile-time assertions that the concrete checkers satisfy the pipeline-stage
// interfaces (backend.go): compile builds []LexicalFacet and []ValueFacet and
// ValidateLexical ranges over them polymorphically, so the interface
// satisfaction has real call sites.
var (
	_ LexicalFacet = patternFacet{}
	_ ValueFacet   = enumFacet{}
	_ ValueFacet   = boundFacet{}
	_ ValueFacet   = digitsFacet{}
	_ ValueFacet   = lengthFacet{}
	_ ValueFacet   = explicitTimezoneFacet{}
)

// ValidateLexical validates the lexical string rawLexical against st's effective
// facets through the full facet pipeline (whiteSpace → pattern → lexical mapping
// → value facets), returning the parsed value on success or the first
// *xsderr.Error a stage produces (stop-on-first-failure; this does not collect
// all facet violations). ctx is the VALIDATED INSTANCE's context, threaded to
// the governing mapping's Parse for the candidate value; a context-free cohort
// (decimal/boolean/string) passes nil here.
//
// PRECONDITION (caller-guarded, NOT checked here): every facet on st must be
// APPLICABLE to st's primitive ancestor (cos-applicable-facets §4.1.5), st must
// be an atomic type whose effective facets carry a whiteSpace facet (§3.16.7.4),
// and b must map st's primitive ancestor. ValidateLexical PANICS — it does not
// return an error — when a value facet is paired with a value lacking the
// capability that facet needs (a bound facet on a non-Ordered value, a length
// facet on a non-Lengthed value, a digit facet on a non-DigitCounted value), or
// when st has no whiteSpace facet in force (see effectiveWhiteSpace). Those are
// schema-construction errors (st-restrict-facets / cos-applicable-facets) the
// caller must have already rejected, never instance data, so they surface as
// programming-error panics, not validity verdicts.
//
// Facet {value} parsing is a separate concern with its own scope: an inherited
// enumeration/bound facet's lexical {value} is parsed in the DECLARING SCHEMA's
// context (see newEnumFacet/newBoundFacet), which for a context-free cohort is
// also nil. Future QName/NOTATION enumeration facets must not silently inherit
// the instance context here — that would resolve a facet literal's prefixes
// against the wrong scope.
func ValidateLexical(b Backend, st *xsd.SimpleType, rawLexical string, ctx Context) (Value, error) {
	lexFacets, valFacets, err := compile(b, st)
	if err != nil {
		return nil, err
	}

	// whiteSpace stage (§4.3.6): normalize using st's effective whiteSpace facet,
	// resolved off EffectiveFacets (the ordinary same-kind overlay, §3.16.6.4).
	normalized := normalizeWhiteSpace(rawLexical, effectiveWhiteSpace(st))

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
			"value: no backend mapping governs type %s", st.Name())
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
// explicitTimezone is a value facet handled here (cvc-explicitTimezone-valid,
// §4.3.14.3). assertions remain out of this runner's scope — they are a separate
// later stage, not an atomic value facet — and are skipped.
func compile(b Backend, st *xsd.SimpleType) ([]LexicalFacet, []ValueFacet, error) {
	var lexFacets []LexicalFacet
	var valFacets []ValueFacet
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
		case xsd.FacetExplicitTimezone:
			tf, err := newExplicitTimezoneFacet(ef.Facet())
			if err != nil {
				return nil, nil, err
			}
			valFacets = append(valFacets, tf)
		case xsd.FacetAssertions:
			// Out of this runner's scope: assertions are a separate later stage,
			// not an atomic value facet.
		}
	}
	return lexFacets, valFacets, nil
}

// governingMapping walks from node (inclusive) up the base chain and returns
// the first ancestor's Mapping the backend supplies. This is the widest-space
// resolution (backend.go, st-restrict-facets §3.16.6.4): a derived type without
// its own mapping is governed by its nearest mapped ancestor's.
func governingMapping(b Backend, node *xsd.SimpleType) (Mapping, bool) {
	for s := node; s != nil; s = s.Base() {
		if m, ok := b.Mapping(s.Name()); ok {
			return m, true
		}
	}
	return Mapping{}, false
}

// declaringMapping implements the widest-space rule (st-restrict-facets
// §3.16.6.4, backend.go) for an inherited facet: it finds the type named
// declaring on leaf's base chain, then returns the governing mapping FROM that
// type (its own, or its nearest mapped ancestor's) — never leaf's. A facet's
// lexical {value} is parsed in the value space of the type that DECLARES it, so
// a narrow derived representation can never corrupt an inherited bound/enum
// comparison (overflow, collapsed precision, different ordering).
//
// Types are matched by QName; anonymous declaring types (the zero QName) are
// outside this runner's manually-built scope.
func declaringMapping(b Backend, leaf *xsd.SimpleType, declaring xsd.QName) (Mapping, bool) {
	for s := leaf; s != nil; s = s.Base() {
		if s.Name() == declaring {
			return governingMapping(b, s)
		}
	}
	return Mapping{}, false
}

// patternFacet is the pattern (lexical) stage (cvc-pattern-valid, §4.3.4.4).
// Each FacetPattern EffectiveFacet returned by EffectiveFacets represents ONE
// derivation step's OR-set (the branches declared at that step, ORed within its
// Values()); patterns at different steps are NOT folded into one facet — they
// stay as separate EffectiveFacets (§4.3.4.2 xr-pattern: cross-step patterns
// are ANDed, never merged into one flat OR-set). compile() builds one
// patternFacet per such EffectiveFacet, and ValidateLexical requires EVERY one
// to pass (AND-across-steps); within a single patternFacet a literal is
// pattern-valid if it matches ANY member (the same-step OR-set). The RE2
// regexes are compiled once at construction.
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
	members []Value
}

// newEnumFacet parses each enumeration {value} lexical via the declaring type's
// mapping (widest-space rule, st-restrict-facets §3.16.6.4). The declaring
// schema's context is nil for a context-free cohort (see ValidateLexical).
func newEnumFacet(b Backend, st *xsd.SimpleType, ef xsd.EffectiveFacet) (enumFacet, error) {
	m, ok := declaringMapping(b, st, ef.Declaring())
	if !ok {
		return enumFacet{}, xsderr.New("cvc-enumeration-valid", xsderr.Loc{},
			"enumeration: no backend mapping governs declaring type %s", ef.Declaring())
	}
	values := ef.Facet().Values()
	members := make([]Value, 0, len(values))
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
func (e enumFacet) CheckValue(v Value) error {
	for _, member := range e.members {
		if enumMatch(v, member) {
			return nil
		}
	}
	return xsderr.New("cvc-enumeration-valid", xsderr.Loc{},
		"value is not equal or identical to any enumeration member (cvc-enumeration-valid, §4.3.5.4)")
}

// enumMatch reports the "equal or identical" relation cvc-enumeration-valid
// needs (§4.3.5.4). It prefers Identical (the identity relation: NaN identical
// to itself, +0 not identical to -0 — doc.go) when the candidate implements it,
// and unions it with Eq so an equal-but-not-identical member (e.g. +0 vs -0)
// still matches. A candidate with neither capability matches nothing.
func enumMatch(candidate, member Value) bool {
	if id, ok := candidate.(Identical); ok && id.Identical(member) {
		return true
	}
	if eq, ok := candidate.(Eq); ok && eq.Eq(member) {
		return true
	}
	return false
}

// boundFacet is one of the four bound value-facet stages
// (cvc-maxInclusive/maxExclusive/minInclusive/minExclusive-valid, §4.3.7–4.3.10).
// The limit and candidate both assert Ordered (every bound-applicable primitive
// is ordered, cos-applicable-facets §4.1.5). An Incomparable Cmp is a legitimate
// spec outcome for a PARTIALLY ordered primitive (float/double): a value
// incomparable with a bounding facet's value is EXCLUDED from the restricted
// value space (§3.3.4.3/§3.3.5.3 Note — e.g. NaN against any numeric bound, or
// any value when the bound itself is NaN), so CheckValue REJECTS it rather than
// panicking.
type boundFacet struct {
	limit Ordered
	kind  xsd.FacetKind
}

// newBoundFacet parses the single bound {value} via the declaring type's
// mapping (widest-space rule) and requires it to be Ordered.
func newBoundFacet(b Backend, st *xsd.SimpleType, ef xsd.EffectiveFacet) (boundFacet, error) {
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
	ord, ok := v.(Ordered)
	if !ok {
		panic(fmt.Sprintf("value: %s facet value %q is not Ordered (cos-applicable-facets §4.1.5 not enforced upstream)", kind, values[0]))
	}
	return boundFacet{limit: ord, kind: kind}, nil
}

// CheckValue rejects a candidate that violates the bound (§4.3.7–4.3.10).
func (bf boundFacet) CheckValue(v Value) error {
	cand, ok := v.(Ordered)
	if !ok {
		panic(fmt.Sprintf("value: candidate %T under a %s facet is not Ordered (cos-applicable-facets §4.1.5 not enforced upstream)", v, bf.kind))
	}
	ord := cand.Cmp(bf.limit)
	if ord == Incomparable {
		// A value incomparable with the bound is excluded from the restricted
		// value space (§3.3.4.3/§3.3.5.3 Note): a real facet rejection, e.g. a
		// NaN candidate against a numeric bound, or any candidate when the bound
		// value is itself NaN (the restricted space is then empty).
		return xsderr.New(boundRule(bf.kind), xsderr.Loc{},
			"value is incomparable with the %s facet bound, so it is excluded from the restricted value space (%s, §4.3.7–4.3.10)", bf.kind, boundRule(bf.kind))
	}
	if bf.violates(ord) {
		return xsderr.New(boundRule(bf.kind), xsderr.Loc{},
			"value violates the %s facet (%s, §4.3.7–4.3.10)", bf.kind, boundRule(bf.kind))
	}
	return nil
}

// violates maps the candidate-vs-limit ordering to a bound violation per kind.
func (bf boundFacet) violates(ord Ordering) bool {
	switch bf.kind {
	case xsd.FacetMaxInclusive:
		return ord == Greater
	case xsd.FacetMaxExclusive:
		return ord == Greater || ord == Equal
	case xsd.FacetMinInclusive:
		return ord == Less
	case xsd.FacetMinExclusive:
		return ord == Less || ord == Equal
	default:
		panic(fmt.Sprintf("value: violates: %s is not a bound facet", bf.kind))
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
		panic(fmt.Sprintf("value: boundRule: %s is not a bound facet", k))
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
func (df digitsFacet) CheckValue(v Value) error {
	dc, ok := v.(DigitCounted)
	if !ok {
		panic(fmt.Sprintf("value: candidate %T under a %s facet is not DigitCounted (cos-applicable-facets §4.1.5 not enforced upstream)", v, df.kind))
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
		panic(fmt.Sprintf("value: digitsRule: %s is not a digit facet", k))
	}
}

// lengthFacet is the length/minLength/maxLength value-facet stage
// (cvc-length-valid §4.3.1.3, cvc-minLength-valid §4.3.2.3, cvc-maxLength-valid
// §4.3.3.3). For string the unit is Unicode codepoints (Lengthed.Len).
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
func (lf lengthFacet) CheckValue(v Value) error {
	l, ok := v.(Lengthed)
	if !ok {
		panic(fmt.Sprintf("value: candidate %T under a %s facet is not Lengthed (cos-applicable-facets §4.1.5 not enforced upstream)", v, lf.kind))
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
		panic(fmt.Sprintf("value: violates: %s is not a length facet", lf.kind))
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
		panic(fmt.Sprintf("value: lengthRule: %s is not a length facet", k))
	}
}

// tzRequirement is the explicitTimezone {value} domain — exactly the three
// tokens required/prohibited/optional (§4.3.14.1), normalized from the facet's
// single NCName {value} at construction so CheckValue never re-parses a string.
type tzRequirement int

const (
	tzRequired tzRequirement = iota
	tzProhibited
	tzOptional
)

// explicitTimezoneFacet is the explicitTimezone value-facet stage
// (cvc-explicitTimezone-valid, §4.3.14.3), applicable to the date/time family
// only (cos-applicable-facets §4.1.5). Its {value} is one of required/prohibited/
// optional (§4.3.14.1), resolved once at construction into a tzRequirement.
type explicitTimezoneFacet struct {
	requirement tzRequirement
}

// newExplicitTimezoneFacet reads the facet's single {value} token
// (required/prohibited/optional) — a plain NCName from the facet's XML
// representation (§4.3.14.2), not a value in the declaring type's space, so no
// declaring-mapping lookup (the digitsFacet/lengthFacet shape). Any other shape
// is a malformed facet, rejected here as an *xsderr.Error, not at check time.
func newExplicitTimezoneFacet(f xsd.Facet) (explicitTimezoneFacet, error) {
	values := f.Values()
	if len(values) != 1 {
		return explicitTimezoneFacet{}, xsderr.New("cvc-explicitTimezone-valid", xsderr.Loc{},
			"explicitTimezone facet must carry exactly one value, has %d", len(values))
	}
	switch values[0] {
	case "required":
		return explicitTimezoneFacet{requirement: tzRequired}, nil
	case "prohibited":
		return explicitTimezoneFacet{requirement: tzProhibited}, nil
	case "optional":
		return explicitTimezoneFacet{requirement: tzOptional}, nil
	}
	return explicitTimezoneFacet{}, xsderr.New("cvc-explicitTimezone-valid", xsderr.Loc{},
		"explicitTimezone facet value %q is not one of required/prohibited/optional (§4.3.14.1)", values[0])
}

// CheckValue enforces cvc-explicitTimezone-valid (§4.3.14.3): required demands a
// non-absent ·timezoneOffset·, prohibited demands an absent one, optional always
// passes (a real always-succeeding branch, not a dropped stage). The candidate
// must be TimezoneAware for the required/prohibited cases; a non-TimezoneAware
// value under an explicitTimezone facet is a schema-construction error (the facet
// is not applicable to it, cos-applicable-facets §4.1.5), never instance data, so
// it PANICS rather than returning a validity verdict — the boundFacet convention.
func (tf explicitTimezoneFacet) CheckValue(v Value) error {
	if tf.requirement == tzOptional {
		return nil
	}
	ta, ok := v.(TimezoneAware)
	if !ok {
		panic(fmt.Sprintf("value: candidate %T under an explicitTimezone facet is not TimezoneAware (cos-applicable-facets §4.1.5 not enforced upstream)", v))
	}
	if tf.requirement == tzRequired && !ta.HasTimezone() {
		return xsderr.New("cvc-explicitTimezone-valid", xsderr.Loc{},
			"value has no explicit timezone but the explicitTimezone facet is required (cvc-explicitTimezone-valid, §4.3.14.3)")
	}
	if tf.requirement == tzProhibited && ta.HasTimezone() {
		return xsderr.New("cvc-explicitTimezone-valid", xsderr.Loc{},
			"value has an explicit timezone but the explicitTimezone facet prohibits one (cvc-explicitTimezone-valid, §4.3.14.3)")
	}
	return nil
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
