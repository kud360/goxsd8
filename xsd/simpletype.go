package xsd

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// ruleSTPropsCorrect is Simple Type Definition Properties Correct (Structures
// §3.16.6.1, id="st-props-correct"). Clause 4 forbids more than one member of
// {facets} of the same kind; clause 5 requires each facet be supported. This
// package charges it to two construction-time rejections in NewSimpleType: a
// {final} token outside the legal simple-type subset (restriction, extension,
// list, union — §3.16.1 / Datatypes §4.1.1 tableau), and two own facets of the
// same FacetKind (clause 4).
const ruleSTPropsCorrect xsderr.Rule = "st-props-correct"

// xsdNamespace is the XSD namespace URI. It is a local unexported constant
// pending issue #39, which will export a shared named constant for it; this
// package uses it only to name the anySimpleType/anyAtomicType anchors.
const xsdNamespace = "http://www.w3.org/2001/XMLSchema"

// Variety is a Simple Type Definition's {variety} (Structures §3.16.1,
// Datatypes §2.4.1): one of atomic, list, or union. It is a sealed sum (STYLE
// T2/T7): Atomic, List, and Union are its only implementations, sealed by the
// unexported variety method, so consumers exhaustively switch the three
// branches and no fourth variety is representable. A {variety} of nil models
// the absent variety that only xs:anySimpleType has (§3.16.1: "Required for all
// Simple Type Definitions except ·xs:anySimpleType·, in which it is ·absent·").
//
// This is deliberately NOT the same type as builtin.Variety, and the two are
// not unified. builtin.Variety is the pre-resolution data-table shape: it has
// only Atomic and List (no builtin has union variety), and its List.Item is a
// bare name string, because the builtin table is generated before any type
// component exists to point at. xsd.Variety is the post-resolution component
// shape (phased construction, PRINCIPLES D4): its branches carry live
// *SimpleType pointers into the compiled model, and it adds Union, which
// user-defined simple types need. Resolving a builtin.Variety into an
// xsd.Variety is a producer's job in a later phase, not an identity.
type Variety interface{ variety() }

// Atomic is the atomic {variety} (Datatypes §2.4.1.1). Primitive is the
// {primitive type definition} (§3.16.1): a live pointer to this type's
// primitive ancestor, or nil for the one exception, xs:anyAtomicType, whose
// {primitive type definition} is ·absent·. The field is read-only by
// convention; do not mutate it after construction.
type Atomic struct{ Primitive *SimpleType }

// List is the list {variety} (Datatypes §2.4.1.2). Item is the {item type
// definition} (§3.16.1), a live pointer to the list's item type. The field is
// read-only by convention; do not mutate it after construction.
type List struct{ Item *SimpleType }

// Union is the union {variety} (Datatypes §2.4.1.3). Members are the {member
// type definitions} (§3.16.1) in document order; the sequence may be empty. The
// slice is read-only by convention; do not mutate it or its elements after
// construction.
type Union struct{ Members []*SimpleType }

func (Atomic) variety() {}
func (List) variety()   {}
func (Union) variety()  {}

// FacetKind is the kind of a Constraining Facet, the closed 14-member set of
// Datatypes §4.3.1–4.3.14. The zero value is invalid (an unset field is a
// caught bug, STYLE T1/T7); constants start at iota+1 and carry the verbatim
// §4.3 token, returned by String().
//
// The two precisionDecimal-only extension facets maxScale and minScale
// (xsd-precisionDecimal.md §4.2/§4.3) are deliberately NOT members: they are
// defined by a separate spec document, not by xmlschema11-2.md §4.3, and do not
// belong in this core enum.
type FacetKind uint8

// The FacetKind values (Datatypes §4.3.1–4.3.14, in spec order).
const (
	// FacetLength is the "length" facet (§4.3.1).
	FacetLength FacetKind = iota + 1
	// FacetMinLength is the "minLength" facet (§4.3.2).
	FacetMinLength
	// FacetMaxLength is the "maxLength" facet (§4.3.3).
	FacetMaxLength
	// FacetPattern is the "pattern" facet (§4.3.4). It has no {fixed}.
	FacetPattern
	// FacetEnumeration is the "enumeration" facet (§4.3.5). It has no {fixed}.
	FacetEnumeration
	// FacetWhiteSpace is the "whiteSpace" facet (§4.3.6).
	FacetWhiteSpace
	// FacetMaxInclusive is the "maxInclusive" facet (§4.3.7).
	FacetMaxInclusive
	// FacetMaxExclusive is the "maxExclusive" facet (§4.3.8).
	FacetMaxExclusive
	// FacetMinExclusive is the "minExclusive" facet (§4.3.9).
	FacetMinExclusive
	// FacetMinInclusive is the "minInclusive" facet (§4.3.10).
	FacetMinInclusive
	// FacetTotalDigits is the "totalDigits" facet (§4.3.11).
	FacetTotalDigits
	// FacetFractionDigits is the "fractionDigits" facet (§4.3.12).
	FacetFractionDigits
	// FacetAssertions is the "assertions" facet (§4.3.13). It has no {fixed}.
	FacetAssertions
	// FacetExplicitTimezone is the "explicitTimezone" facet (§4.3.14).
	FacetExplicitTimezone
)

// String returns the verbatim §4.3 token, or a diagnostic form for an invalid
// value (never panics).
func (k FacetKind) String() string {
	switch k {
	case FacetLength:
		return "length"
	case FacetMinLength:
		return "minLength"
	case FacetMaxLength:
		return "maxLength"
	case FacetPattern:
		return "pattern"
	case FacetEnumeration:
		return "enumeration"
	case FacetWhiteSpace:
		return "whiteSpace"
	case FacetMaxInclusive:
		return "maxInclusive"
	case FacetMaxExclusive:
		return "maxExclusive"
	case FacetMinExclusive:
		return "minExclusive"
	case FacetMinInclusive:
		return "minInclusive"
	case FacetTotalDigits:
		return "totalDigits"
	case FacetFractionDigits:
		return "fractionDigits"
	case FacetAssertions:
		return "assertions"
	case FacetExplicitTimezone:
		return "explicitTimezone"
	default:
		return "FacetKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// HasFixed reports whether facets of this kind carry a {fixed} property. It is
// derived from the kind, never stored (STYLE D3). It is false only for
// FacetPattern, FacetEnumeration, and FacetAssertions: their tableaux (§4.3.4,
// §4.3.5, §4.3.13) give only {annotations} and {value}, with no {fixed}. Every
// other kind's tableau carries a required {fixed} xs:boolean.
func (k FacetKind) HasFixed() bool {
	switch k {
	case FacetPattern, FacetEnumeration, FacetAssertions:
		return false
	default:
		return true
	}
}

// Facet is a Constraining Facet component (Datatypes §4.3): a facet kind, its
// {value} (one or more normalized lexical strings — a single string for the
// single-valued kinds such as length or whiteSpace, or several for the
// set/sequence-valued kinds pattern, enumeration, and assertions), and its
// {fixed} flag where the kind has one.
//
// Construct only through NewFacet, which normalizes away the illegal
// combination of a set {fixed} on a kind that has no {fixed} property, so that
// state is unrepresentable (STYLE T1). Facet is immutable after construction.
type Facet struct {
	kind   FacetKind
	values []string
	fixed  bool
}

// NewFacet builds a Facet of the given kind carrying values as its {value}. The
// values slice is copied; the caller's backing array is not aliased.
//
// fixed is honored only when kind.HasFixed() is true; for FacetPattern,
// FacetEnumeration, and FacetAssertions (which have no {fixed} property,
// §4.3.4/.5/.13) it is normalized to false so that "fixed set on a kind with no
// {fixed}" cannot be stored (STYLE T1). Read {fixed} back through Fixed, whose
// second result reports whether the kind has the property at all.
func NewFacet(kind FacetKind, values []string, fixed bool) Facet {
	f := Facet{kind: kind}
	if len(values) > 0 {
		f.values = append([]string(nil), values...)
	}
	if kind.HasFixed() {
		f.fixed = fixed
	}
	return f
}

// Kind returns the facet's kind.
func (f Facet) Kind() FacetKind {
	return f.kind
}

// Values returns the facet's {value} in document order. It returns a copy:
// mutating the result does not affect f. An empty {value} yields nil.
func (f Facet) Values() []string {
	if len(f.values) == 0 {
		return nil
	}
	return append([]string(nil), f.values...)
}

// Fixed returns the {fixed} property. The second result is Kind().HasFixed():
// when it is false the kind has no {fixed} property (FacetPattern,
// FacetEnumeration, FacetAssertions) and the first result is not meaningful.
func (f Facet) Fixed() (fixed bool, ok bool) {
	return f.fixed, f.kind.HasFixed()
}

// SimpleType is a Simple Type Definition component (Structures §3.16.1,
// Datatypes §4.1.1), the datatypes-facing subset: {name} (bundled with {target
// namespace} as a QName), {variety}, {base type definition}, the type's own
// contribution to {facets}, and {final}. Full complex-type breadth
// (union-of-complex, {context}, {annotations}) is out of this component's
// scope.
//
// Unlike the value-typed components in this package (Occurs, Notation),
// SimpleType is handled through a *SimpleType pointer: components reference one
// another by pointer once resolved (phased construction, PRINCIPLES D4), and
// component identity matters — two *SimpleType values denote the same component
// if and only if they are the same pointer, whereas two equal Occurs values are
// interchangeable. Construct only through NewSimpleType; a SimpleType is
// immutable after construction.
type SimpleType struct {
	name      QName
	variety   Variety
	base      *SimpleType
	ownFacets []Facet
	final     []DerivationMethod
}

// NewSimpleType builds a Simple Type Definition. base is the {base type
// definition}: nil means this type IS xs:anySimpleType (the one simple type
// whose base is xs:anyType, a Complex Type Definition outside this package's
// scope); every other simple type has a non-nil simple-type base. variety may
// be nil to model xs:anySimpleType's absent {variety}.
//
// It rejects, charging Simple Type Definition Properties Correct
// (§3.16.6.1, st-props-correct):
//
//   - any final entry that is not one of DerivationRestriction,
//     DerivationExtension, DerivationList, or DerivationUnion — the legal
//     simple-type {final} subset (§3.16.1 / Datatypes §4.1.1). In particular
//     DerivationSubstitution, which belongs to element/attribute substitution,
//     is not a member of this vocabulary.
//   - two ownFacets of the same FacetKind (clause 4: "not more than one member
//     of {facets} of the same kind").
//
// ownFacets and final are copied; the caller's backing arrays are not aliased.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built type — may
// legitimately pass the zero xsderr.Loc{}.
func NewSimpleType(loc xsderr.Loc, name QName, variety Variety, base *SimpleType, ownFacets []Facet, final []DerivationMethod) (*SimpleType, error) {
	for _, d := range final {
		switch d {
		case DerivationRestriction, DerivationExtension, DerivationList, DerivationUnion:
			// legal simple-type {final} token
		default:
			return nil, xsderr.New(ruleSTPropsCorrect, loc,
				"simple type {final} token %s is not one of restriction, extension, list, union", d)
		}
	}

	seen := make(map[FacetKind]struct{}, len(ownFacets))
	for _, f := range ownFacets {
		if _, dup := seen[f.kind]; dup {
			return nil, xsderr.New(ruleSTPropsCorrect, loc,
				"simple type has more than one %s facet", f.kind)
		}
		seen[f.kind] = struct{}{}
	}

	t := &SimpleType{name: name, variety: variety, base: base}
	if len(ownFacets) > 0 {
		t.ownFacets = append([]Facet(nil), ownFacets...)
	}
	if len(final) > 0 {
		t.final = append([]DerivationMethod(nil), final...)
	}
	return t, nil
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
// The zero QName means {name} is absent (an anonymous simple type).
func (t *SimpleType) Name() QName {
	return t.name
}

// Variety returns the {variety} property: an Atomic, List, or Union value, or
// nil for xs:anySimpleType, whose {variety} is absent (§3.16.1).
func (t *SimpleType) Variety() Variety {
	return t.variety
}

// Base returns the {base type definition} property. It is nil if and only if
// IsAnySimpleType reports true — that is, when this type IS xs:anySimpleType,
// whose real base (xs:anyType) is a Complex Type Definition outside this
// package's scope. For every other simple type Base is a non-nil *SimpleType.
func (t *SimpleType) Base() *SimpleType {
	return t.base
}

// IsAnySimpleType reports whether this type is xs:anySimpleType, the root of the
// simple-type hierarchy (§3.16.1). It is exactly the condition Base() == nil,
// exposed as a predicate so callers do not infer this identity from nil-ness.
func (t *SimpleType) IsAnySimpleType() bool {
	return t.base == nil
}

// IsPrimitive reports whether this type is a primitive datatype (Datatypes
// §2.4.2). A type is primitive if and only if its {base type definition} is
// xs:anyAtomicType (§3.16.1: "A type definition has ·xs:anyAtomicType· as its
// {base type definition} if and only if it is one of the primitive
// datatypes."). The two special types xs:anySimpleType and xs:anyAtomicType are
// themselves not primitive, and this returns false for them.
func (t *SimpleType) IsPrimitive() bool {
	return t.base == anyAtomicType
}

// Final returns the {final} property in document order. It returns a copy:
// mutating the result does not affect t. An empty {final} yields nil.
func (t *SimpleType) Final() []DerivationMethod {
	if len(t.final) == 0 {
		return nil
	}
	return append([]DerivationMethod(nil), t.final...)
}

// OwnFacets returns only the facets this type's own restriction contributes —
// the "S" operand of the §3.16.6.4 overlay — in document order. It is NOT the
// spec's {facets} property, which is the fully accumulated overlay result; for
// that, use EffectiveFacets. It returns a copy: mutating the result does not
// affect t. An empty own-facet set yields nil.
func (t *SimpleType) OwnFacets() []Facet {
	if len(t.ownFacets) == 0 {
		return nil
	}
	return append([]Facet(nil), t.ownFacets...)
}

// EffectiveFacets computes and returns the spec's {facets} property (Structures
// §3.16.1): the Constraining Facets in force on this type, accumulated through
// the whole {base type definition} chain. It is computed on demand, never
// cached or stored (STYLE D3), by walking Base() from this type up to
// xs:anySimpleType and overlaying each level's OwnFacets per the §3.16.6.4
// overlay rule: a facet contributed by a more-derived level supersedes any
// same-kind facet from a less-derived level, and every non-superseded facet
// survives.
//
// The result is deterministic (STYLE D2/D3) and ordered base-to-derived:
// facets from a less-derived type come first, and within one type in declared
// order; when a more-derived type overrides a facet kind, the overriding facet
// replaces the base one and takes its own (more-derived) position. It returns a
// fresh slice each call; mutating it does not affect t.
func (t *SimpleType) EffectiveFacets() []Facet {
	// Collect the base chain most-derived first (t, then its base, ...).
	var chain []*SimpleType
	for s := t; s != nil; s = s.base {
		chain = append(chain, s)
	}

	// Overlay least-derived first so more-derived facets win.
	var result []Facet
	for i := len(chain) - 1; i >= 0; i-- {
		for _, f := range chain[i].ownFacets {
			result = overlayFacet(result, f)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// overlayFacet applies a single more-derived facet onto acc per §3.16.6.4:
// any same-kind facet already in acc is dropped, and f is appended, so f both
// wins and takes the more-derived position.
func overlayFacet(acc []Facet, f Facet) []Facet {
	out := make([]Facet, 0, len(acc)+1)
	for _, existing := range acc {
		if existing.kind != f.kind {
			out = append(out, existing)
		}
	}
	return append(out, f)
}

// anySimpleType is the xs:anySimpleType anchor (§3.16.1): the root of the
// simple-type hierarchy. Its {variety} and {base type definition} are both
// absent (nil) — its real base, xs:anyType, is a Complex Type Definition
// outside this package's scope. It is unexported: no consumer needs it yet
// (STYLE T5).
var anySimpleType = &SimpleType{
	name: QName{Space: xsdNamespace, Local: "anySimpleType"},
}

// anyAtomicType is the xs:anyAtomicType anchor (Datatypes §4.1.6): the special
// atomic type that is the {base type definition} of every primitive datatype.
// Its {base type definition} is anySimpleType, and it is the one atomic type
// whose {primitive type definition} is itself absent (Atomic{Primitive: nil}).
// It is unexported: no consumer needs it yet (STYLE T5).
var anyAtomicType = &SimpleType{
	name:    QName{Space: xsdNamespace, Local: "anyAtomicType"},
	variety: Atomic{Primitive: nil},
	base:    anySimpleType,
}
