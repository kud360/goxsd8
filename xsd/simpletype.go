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
// primitive ancestor — which for a primitive datatype is the type itself
// (§3.16.1: "the {primitive type definition} of a primitive datatype is that
// datatype itself"), wired via NewPrimitiveType — or nil for the one exception,
// xs:anyAtomicType, whose {primitive type definition} is ·absent·. The field is
// read-only by convention; do not mutate it after construction.
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

// FacetKind is the closed set of facet kinds the Facet/EffectiveFacets/
// restriction-merging machinery treats uniformly — 16 members: the 14 core
// Constraining Facets of Datatypes §4.3.1–4.3.14, plus the two
// precisionDecimal-only extension facets maxScale and minScale
// (xsd-precisionDecimal.md §4.2/§4.3). The two extension facets are a
// deliberate, documented cross-spec inclusion: they combine, overlay, and check
// through the very same generic path as the core kinds (NewFacet,
// EffectiveFacets, the value/facets.go pipeline), so excluding them would force
// a parallel mechanism for no gain — the constants themselves cite
// xsd-precisionDecimal.md for provenance. The zero value is invalid (an unset
// field is a caught bug, STYLE T1/T7); constants start at iota+1 and carry the
// verbatim spec token, returned by String().
type FacetKind uint8

// The FacetKind values: the 14 core facets (Datatypes §4.3.1–4.3.14), then the
// two precisionDecimal extension facets (xsd-precisionDecimal.md §4.2/§4.3), in
// spec order.
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
	// FacetMaxScale is the "maxScale" facet (xsd-precisionDecimal.md §4.2,
	// dc-maxScale), a precisionDecimal-only extension facet — not one of the
	// core §4.3 facets. Its {value} is a REQUIRED xs:integer (may be negative)
	// and it carries a REQUIRED {fixed} xs:boolean.
	FacetMaxScale
	// FacetMinScale is the "minScale" facet (xsd-precisionDecimal.md §4.3,
	// dc-minScale), a precisionDecimal-only extension facet — not one of the
	// core §4.3 facets. Its {value} is a REQUIRED xs:integer (may be negative)
	// and it carries a REQUIRED {fixed} xs:boolean.
	FacetMinScale
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
	case FacetMaxScale:
		return "maxScale"
	case FacetMinScale:
		return "minScale"
	default:
		return "FacetKind(" + strconv.Itoa(int(k)) + ")"
	}
}

// HasFixed reports whether facets of this kind carry a {fixed} property. It is
// derived from the kind, never stored (STYLE D3). It is false only for
// FacetPattern, FacetEnumeration, and FacetAssertions: their tableaux (§4.3.4,
// §4.3.5, §4.3.13) give only {annotations} and {value}, with no {fixed}. Every
// other kind's tableau carries a required {fixed} xs:boolean — including the two
// precisionDecimal extension facets FacetMaxScale and FacetMinScale
// (xsd-precisionDecimal.md §4.2/§4.3 both give a REQUIRED {fixed}), which fall
// into the default branch deliberately.
func (k FacetKind) HasFixed() bool {
	switch k {
	case FacetPattern, FacetEnumeration, FacetAssertions:
		return false
	default:
		return true
	}
}

// Facet is a Constraining Facet component (Datatypes §4.3): a facet kind, its
// {value}, and its {fixed} flag where the kind has one. How {value} is modeled
// depends on the kind:
//
//   - For every kind except FacetAssertions, {value} is one or more normalized
//     lexical strings — a single string for the single-valued kinds such as
//     length or whiteSpace, or several for the set/sequence-valued kinds pattern
//     and enumeration — held in values and read through Values.
//   - For FacetAssertions (§4.3.13) {value} is "a sequence of Assertion
//     components" (Structures §3.13.1, id="as"), each carrying a Required {test}
//     XPathExpression that a lexical string cannot represent. It is held in
//     assertions and read through Assertions; values stays nil. kind is the sole
//     discriminant between the two representations.
//
// Construct a non-assertions facet through NewFacet and an assertions facet
// through NewAssertionsFacet; NewFacet normalizes away the illegal combination
// of a set {fixed} on a kind that has no {fixed} property, so that state is
// unrepresentable (STYLE T1). Facet is immutable after construction.
type Facet struct {
	kind       FacetKind
	values     []string
	assertions []Assertion
	fixed      bool
}

// NewFacet builds a Facet of the given kind carrying values as its {value}. The
// values slice is copied; the caller's backing array is not aliased.
//
// It panics if kind is FacetAssertions: the assertions facet models {value} as
// a sequence of Assertion components, not lexical strings, so it must be built
// through NewAssertionsFacet. That is a programmer error (the wrong
// constructor), not user-supplied invalid data, so a panic — not an xsderr
// validation error — is the right guard per this package's convention.
//
// fixed is honored only when kind.HasFixed() is true; for FacetPattern and
// FacetEnumeration (which have no {fixed} property, §4.3.4/.5) it is normalized
// to false so that "fixed set on a kind with no {fixed}" cannot be stored
// (STYLE T1). FacetAssertions likewise has no {fixed} (§4.3.13) but is
// unreachable here. Read {fixed} back through Fixed, whose second result
// reports whether the kind has the property at all.
func NewFacet(kind FacetKind, values []string, fixed bool) Facet {
	if kind == FacetAssertions {
		panic("xsd: NewFacet cannot build an assertions facet; use NewAssertionsFacet")
	}
	f := Facet{kind: kind}
	if len(values) > 0 {
		f.values = append([]string(nil), values...)
	}
	if kind.HasFixed() {
		f.fixed = fixed
	}
	return f
}

// NewAssertionsFacet builds the assertions Constraining Facet (Datatypes
// §4.3.13) whose {value} is a sequence of Assertion components (Structures
// §3.13.1, id="as") rather than lexical strings. The assertions slice is copied
// in document order; the caller's backing array is not aliased. The result's
// kind is FacetAssertions, its values stays nil, and its {fixed} is false —
// §4.3.13 gives the assertions facet no {fixed} property (HasFixed reports
// false for it). Read the assertions back through Assertions.
func NewAssertionsFacet(assertions []Assertion) Facet {
	f := Facet{kind: FacetAssertions}
	if len(assertions) > 0 {
		f.assertions = append([]Assertion(nil), assertions...)
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

// Assertions returns the facet's {value} as a sequence of Assertion components
// in document order. The second result reports whether this is an assertions
// facet (Kind() == FacetAssertions): when it is false the facet models {value}
// as lexical strings instead (use Values), and the first result is nil. It
// returns a copy: mutating the result does not affect f. An assertions facet
// with no assertions yields nil.
func (f Facet) Assertions() (assertions []Assertion, ok bool) {
	if f.kind != FacetAssertions {
		return nil, false
	}
	if len(f.assertions) == 0 {
		return nil, true
	}
	return append([]Assertion(nil), f.assertions...), true
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
	if err := checkSTProps(loc, ownFacets, final); err != nil {
		return nil, err
	}
	t := &SimpleType{name: name, variety: variety, base: base}
	t.setOwnFacetsFinal(ownFacets, final)
	return t, nil
}

// NewPrimitiveType builds a primitive datatype (Datatypes §2.4.2): one of the
// types whose {base type definition} is xs:anyAtomicType (§3.16.1). It fixes the
// {base type definition} to the canonical xs:anyAtomicType anchor (see
// AnyAtomicType) so IsPrimitive reports true and pointer identity holds across
// every graph, and it wires the self-referential {primitive type definition} —
// a primitive's {primitive type definition} is itself (§3.16.1) — so the
// returned node's {variety} is an Atomic whose Primitive points back at that
// same node.
//
// The self-reference is established inside this constructor, before the node
// escapes, so the node is immutable to every external caller. ownFacets and
// final follow NewSimpleType's contract and are validated identically
// (st-props-correct); they are copied, not aliased. loc is charged to any
// rejection.
func NewPrimitiveType(loc xsderr.Loc, name QName, ownFacets []Facet, final []DerivationMethod) (*SimpleType, error) {
	if err := checkSTProps(loc, ownFacets, final); err != nil {
		return nil, err
	}
	t := &SimpleType{name: name, base: anyAtomicType}
	t.variety = Atomic{Primitive: t}
	t.setOwnFacetsFinal(ownFacets, final)
	return t, nil
}

// checkSTProps enforces the st-props-correct (§3.16.6.1) construction-time
// rejections shared by NewSimpleType and NewPrimitiveType: no {final} token
// outside the legal simple-type subset, and no two ownFacets of the same
// FacetKind (clause 4). loc is charged to any rejection.
func checkSTProps(loc xsderr.Loc, ownFacets []Facet, final []DerivationMethod) error {
	for _, d := range final {
		switch d {
		case DerivationRestriction, DerivationExtension, DerivationList, DerivationUnion:
			// legal simple-type {final} token
		default:
			return xsderr.New(ruleSTPropsCorrect, loc,
				"simple type {final} token %s is not one of restriction, extension, list, union", d)
		}
	}

	seen := make(map[FacetKind]struct{}, len(ownFacets))
	for _, f := range ownFacets {
		if _, dup := seen[f.kind]; dup {
			return xsderr.New(ruleSTPropsCorrect, loc,
				"simple type has more than one %s facet", f.kind)
		}
		seen[f.kind] = struct{}{}
	}
	return nil
}

// setOwnFacetsFinal copies ownFacets and final onto t during construction. It
// is called only by the constructors before t escapes; SimpleType is immutable
// thereafter. The caller's backing arrays are not aliased.
func (t *SimpleType) setOwnFacetsFinal(ownFacets []Facet, final []DerivationMethod) {
	if len(ownFacets) > 0 {
		t.ownFacets = append([]Facet(nil), ownFacets...)
	}
	if len(final) > 0 {
		t.final = append([]DerivationMethod(nil), final...)
	}
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

// EffectiveFacet pairs a Facet with the {name} QName of the type on the base
// chain that declared it — the operand type of the §3.16.6.4 overlay that
// contributed that facet-kind to the winning overlay. It is what EffectiveFacets
// yields, so a consumer keeps facet provenance instead of a flattened final
// value.
//
// Provenance is load-bearing for a value-space consumer: the widest-space rule
// requires an inherited enumeration or bound facet to be compared in the value
// space of the type that DECLARES it, not the type that inherits it — a
// consumer building the facet pipeline cannot honor that if the effective view
// flattens away which ancestor contributed the facet. (Package xsd itself does
// NOT depend on package value; this is forward-looking motivation for that
// future consumer, stated as rationale only, not an implemented dependency.)
//
// Declaring is the zero QName when the declaring type is anonymous, per this
// package's zero-value-means-anonymous convention (see QName and SimpleType's
// {name} godoc). That is a legitimate value, not a missing one: an inherited
// facet can genuinely come from an unnamed ancestor on the chain.
//
// FacetAssertions is an explicit EXCEPTION to the "the type that declared it"
// contract above. Its {value} accumulates across the base chain (Datatypes
// §4.3.13.2: the base's Assertions then each restriction's own, appended), so a
// single merged assertions facet spans multiple declaring types and no lone
// QName is truthful. For it, Declaring reflects ONLY the most-derived
// contributor's position — chosen for positional consistency with the
// replace-kind facets — and per-assertion provenance (which type each
// individual Assertion came from) is NOT recoverable from an EffectiveFacet; a
// caller needing it must track it separately. This is acceptable because
// downstream evaluation reads each Assertion's own {test} XPathExpression and
// context, not facet-level Declaring, so the widest-space rationale that makes
// Declaring load-bearing for enumeration and bound facets does not apply here.
//
// EffectiveFacet is immutable after construction; it is produced only by
// EffectiveFacets.
type EffectiveFacet struct {
	facet     Facet
	declaring QName
}

// Facet returns the Constraining Facet in force.
func (f EffectiveFacet) Facet() Facet {
	return f.facet
}

// Declaring returns the {name} QName of the type on the base chain that
// declared the facet. It is the zero QName when that type is anonymous (the
// zero-value-means-anonymous convention, not a missing value).
//
// For a merged FacetAssertions facet — whose {value} accumulates across the
// chain (§4.3.13.2) — Declaring reflects only the most-derived contributor's
// position; the individual Assertions may originate from several ancestors and
// their provenance is not recoverable here (see EffectiveFacet's godoc).
func (f EffectiveFacet) Declaring() QName {
	return f.declaring
}

// EffectiveFacets computes and returns the spec's {facets} property (Structures
// §3.16.1): the Constraining Facets in force on this type, accumulated through
// the whole {base type definition} chain. It is computed on demand, never
// cached or stored (STYLE D3), by walking Base() from this type up to
// xs:anySimpleType and overlaying each level's OwnFacets per the §3.16.6.4
// overlay rule: a facet contributed by a more-derived level supersedes any
// same-kind facet from a less-derived level, and every non-superseded facet
// survives. Two facet kinds are exceptions to the supersede rule (see
// overlayFacet): FacetAssertions accumulates — its {value} is the base's
// Assertions then the restriction's own, appended into one facet (§4.3.13.2);
// FacetPattern keeps both — the base's pattern facet and the restriction's own
// survive as two separate entries (§4.3.4.2 xr-pattern), because patterns at
// different derivation steps are ANDed, not superseded and not merged.
//
// Each result element is an EffectiveFacet, pairing the surviving Facet with
// the {name} QName of the type on the chain that declared it. That provenance
// is required by a downstream value-space consumer's widest-space rule (see
// EffectiveFacet), which must compare an inherited facet in the value space of
// the type that declared it — a bare []Facet would flatten that away. A facet
// declared by an anonymous type reports the zero QName as its declaring name.
//
// The result is deterministic (STYLE D2/D3) and ordered base-to-derived:
// facets from a less-derived type come first, and within one type in declared
// order; when a more-derived type overrides a facet kind, the overriding facet
// replaces the base one and takes its own (more-derived) position. FacetPattern
// is the carve-out to that replace: a pattern facet from a more-derived step
// does NOT replace the base's — both survive as separate entries, base before
// derived (§4.3.4.2 xr-pattern), so EffectiveFacets can return several
// FacetPattern entries for a multi-step-pattern chain. It returns a fresh slice
// each call; mutating it does not affect t.
func (t *SimpleType) EffectiveFacets() []EffectiveFacet {
	// Collect the base chain most-derived first (t, then its base, ...).
	var chain []*SimpleType
	for s := t; s != nil; s = s.base {
		chain = append(chain, s)
	}

	// Overlay least-derived first so more-derived facets win. Each facet
	// carries the declaring type's {name} as its provenance QName.
	var result []EffectiveFacet
	for i := len(chain) - 1; i >= 0; i-- {
		for _, f := range chain[i].ownFacets {
			result = overlayFacet(result, EffectiveFacet{facet: f, declaring: chain[i].name})
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// overlayFacet applies a single more-derived facet onto acc per the §3.16.6.4
// key-facets-overlay rule (which facet component survives per kind): any
// same-kind facet already in acc is dropped, and f is appended, so f both wins
// and takes the more-derived position.
//
// Two facet kinds are exceptions to that replace rule: FacetAssertions merges,
// and FacetPattern keeps both. They differ because their spec combine rules
// differ. FacetAssertions concatenates into ONE facet: the assertions {value}
// is a single sequence whose members are all ANDed, so appending the
// restriction's own onto the base's yields the correct combined {value}.
// FacetPattern must instead keep the base and derived facets as SEPARATE
// entries: each pattern facet's {value} is an OR-set (the branches declared at
// one step), and patterns at DIFFERENT steps are ANDed, not ORed (§4.3.4.2
// xr-pattern, and its summary Note: same-step OR, cross-step AND). Merging the
// base's OR-set into the derived facet's {value} the way assertions merge would
// wrongly OR the two steps together, collapsing the cross-step AND into an OR
// and false-accepting a literal that matches the derived pattern but violates
// the base's. So when acc already holds a FacetPattern entry and f is also
// FacetPattern, the base entry is kept in place (base before derived, for
// determinism) and f is appended after it — both survive, each independently
// checked by the consumer. cos-pattern-restriction (§4.3.4.5,
// id="cos-pattern-restriction"; cataloged in xsderr/catalog.go) — every member
// of the base pattern facet's {value} must remain a member of the derived's —
// holds by construction here: the base entry survives verbatim, so its {value}
// members are trivially still present. No runtime rejection path is needed.
//
// FacetAssertions ACCUMULATES rather than replacing (Datatypes §4.3.13.2,
// id="xr-assertions"). The assertions {value}
// on a restriction is the base type's Assertions followed by the restriction's
// own new Assertions, in that order — an append, never a set-union and never
// deduplicated. So when acc already holds a FacetAssertions entry (the base's
// accumulated {value}) and f is also FacetAssertions, the two are merged into a
// single NewAssertionsFacet whose sequence is the existing (base) Assertions
// PREPENDED before f's own, and that merged facet takes f's more-derived
// position. The merged facet's declaring QName is f's — the most-derived
// contributor (see EffectiveFacet's godoc for why per-assertion provenance is
// not recoverable from the result). With no prior FacetAssertions entry in acc,
// f is appended unchanged, exactly as for every other kind.
//
// Because the base's already-accumulated {value} is unconditionally PREPENDED
// before f's own, the base type's assertions {value} is always a prefix of the
// derived type's: cos-assertions-restriction (§4.3.13.4,
// id="cos-assertions-restriction"; cataloged in xsderr/catalog.go) holds by
// construction. That guarantee rests on this prepend mechanic — NOT on the
// producer-side ownFacets "own-only" convention, which the type system does not
// enforce. Even a caller that smuggled a pre-merged assertions facet into a
// derived type's ownFacets would still have the base prepended, keeping the
// base a prefix. There is therefore no runtime rejection path to add here; the
// constraint is structurally unfalsifiable, so no xsderr call site wraps it.
func overlayFacet(acc []EffectiveFacet, f EffectiveFacet) []EffectiveFacet {
	out := make([]EffectiveFacet, 0, len(acc)+1)
	for _, existing := range acc {
		if existing.facet.kind != f.facet.kind {
			out = append(out, existing)
			continue
		}
		switch f.facet.kind {
		case FacetAssertions:
			f.facet = mergeAssertions(existing.facet, f.facet)
		case FacetPattern:
			// keep-both: the base pattern facet survives as a separate entry so
			// it stays independently checkable (AND-across-steps, §4.3.4.2
			// xr-pattern). f (the more-derived step) is appended below.
			out = append(out, existing)
		default:
			// replace-kind (the other 12 facet kinds): the existing same-kind
			// entry is intentionally dropped here; f is appended below and wins,
			// taking the more-derived position.
		}
	}
	return append(out, f)
}

// mergeAssertions builds the accumulated assertions facet for a restriction per
// §4.3.13.2: base's Assertions first, then own's, in that order — an append,
// never a set-union and never deduplicated. Both arguments are FacetAssertions
// facets. NewAssertionsFacet copies the merged sequence, so the result aliases
// neither operand's backing array.
func mergeAssertions(base, own Facet) Facet {
	merged := make([]Assertion, 0, len(base.assertions)+len(own.assertions))
	merged = append(merged, base.assertions...)
	merged = append(merged, own.assertions...)
	return NewAssertionsFacet(merged)
}

// AnySimpleType returns the canonical xs:anySimpleType anchor (§3.16.1): the
// single shared root of the simple-type hierarchy, an immutable package
// singleton whose {variety} and {base type definition} are both absent. A
// producer that builds the simple-type graph (e.g. builtin.Seed) roots every
// chain on THIS node so the whole graph has one anySimpleType identity — pointer
// identity is load-bearing (see SimpleType). The returned node is read-only; do
// not mutate it.
func AnySimpleType() *SimpleType { return anySimpleType }

// AnyAtomicType returns the canonical xs:anyAtomicType anchor (Datatypes
// §4.1.6): the special atomic type that is the {base type definition} of every
// primitive datatype, an immutable package singleton. A producer roots every
// primitive on THIS node (NewPrimitiveType does so) so IsPrimitive — which tests
// {base type definition} == xs:anyAtomicType by pointer — holds across the whole
// graph. Its own {primitive type definition} is absent (Atomic{Primitive: nil},
// §3.16.1). The returned node is read-only; do not mutate it.
func AnyAtomicType() *SimpleType { return anyAtomicType }

// anySimpleType is the xs:anySimpleType anchor (§3.16.1): the root of the
// simple-type hierarchy. Its {variety} and {base type definition} are both
// absent (nil) — its real base, xs:anyType, is a Complex Type Definition
// outside this package's scope. Exposed to producers through AnySimpleType.
var anySimpleType = &SimpleType{
	name: QName{Space: XMLSchemaNS, Local: "anySimpleType"},
}

// anyAtomicType is the xs:anyAtomicType anchor (Datatypes §4.1.6): the special
// atomic type that is the {base type definition} of every primitive datatype.
// Its {base type definition} is anySimpleType, and it is the one atomic type
// whose {primitive type definition} is itself absent (Atomic{Primitive: nil}).
// Exposed to producers through AnyAtomicType.
var anyAtomicType = &SimpleType{
	name:    QName{Space: XMLSchemaNS, Local: "anyAtomicType"},
	variety: Atomic{Primitive: nil},
	base:    anySimpleType,
}
