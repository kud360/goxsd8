package xsd

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// ruleSTPropsCorrect is Simple Type Definition Properties Correct (Structures
// §3.16.6.1, id="st-props-correct"). The package charges it across three layers:
//
//   - checkSTProps (below) makes the pure-property rejections at construction —
//     a {final} token outside the legal simple-type subset (restriction,
//     extension, list, union — §3.16.1 / Datatypes §4.1.1 tableau) and two own
//     facets of the same FacetKind (clause 4).
//   - SimpleType.CheckDerivation (derivation.go) makes the cross-reference
//     rejections, at finalize, once the {base type definition} reference is
//     resolvable: clause 3 (the base's {final} must not contain restriction),
//     clause 5 (every member of {facets} is processor-supported), the Datatypes
//     §4.1.1 per-variety shape prose, and two presence requirements from that
//     section's dc-defn property tableau — {variety}, which only
//     xs:anySimpleType may leave absent, and {primitive type definition}, whose
//     absence bars every type but a primitive datatype from naming
//     xs:anyAtomicType as its base.
//   - Schema.checkSimpleBaseAcyclic (resolve.go) charges clause 2 (no circular
//     derivation) in finalize's Phase B. It is no longer a no-op: a cyclic base
//     chain became REPRESENTABLE the moment {base type definition} became a
//     deferred SimpleTypeRef, so the rule that a live-pointer base made
//     structurally unnecessary now needs a real check.
const ruleSTPropsCorrect xsderr.Rule = "st-props-correct"

// Variety is a Simple Type Definition's {variety} (Structures §3.16.1,
// Datatypes §2.4.1): one of atomic, list, or union. It is a sealed sum (STYLE
// T2/T7): Atomic, List, and Union are its only implementations, sealed by the
// unexported variety method, so consumers exhaustively switch the three
// branches and no fourth variety is representable. A {variety} of nil models
// the absent variety that only xs:anySimpleType has (§3.16.1: "Required for all
// Simple Type Definitions except ·xs:anySimpleType·, in which it is ·absent·").
//
// The three branches are EMPTY markers: each carries the §3.16.1 token and
// nothing else. {variety} is not a stored property of a SimpleType at all — it
// is DERIVED from the component's declared SimpleTypeDerivation and its {base
// type definition} chain per §3.16.2.1 (see SimpleType.Variety), and the three
// properties that used to ride these branches are read off the owning component
// instead: {primitive type definition} through SimpleType.Primitive, {item type
// definition} through SimpleType.Item, {member type definitions} through
// SimpleType.Members. Storing a copy on the branch as well would be one fact in
// two encodings (STYLE D3), and minting a fresh data-carrying branch per
// Variety call would additionally allocate a component-identity-carrying value
// on every read with no measured hot path (STYLE D3/D4).
//
// This is deliberately NOT the same type as builtin.Variety, and the two are
// not unified. builtin.Variety is the pre-resolution data-table shape: it has
// only Atomic and List (no builtin has union variety), and its List.Item is a
// bare name string, because the builtin table is generated before any type
// component exists to point at. xsd.Variety is the post-resolution token set
// (phased construction, PRINCIPLES D4), and it adds Union, which user-defined
// simple types need. Translating a builtin.Variety into a declared
// SimpleTypeDerivation is a producer's job in a later phase, not an identity.
type Variety interface{ variety() }

// Atomic is the atomic {variety} (Datatypes §2.4.1.1). It is an empty marker:
// the associated {primitive type definition} (§3.16.1) is read off the owning
// component through SimpleType.Primitive, which derives it from the base chain.
type Atomic struct{}

// List is the list {variety} (Datatypes §2.4.1.2). It is an empty marker: the
// associated {item type definition} (§3.16.1) is read off the owning component
// through SimpleType.Item, which derives it from the base chain.
type List struct{}

// Union is the union {variety} (Datatypes §2.4.1.3). It is an empty marker: the
// associated {member type definitions} (§3.16.1) are read off the owning
// component through SimpleType.Members, which derives them from the base chain.
type Union struct{}

func (Atomic) variety() {}
func (List) variety()   {}
func (Union) variety()  {}

// SimpleTypeDerivation is a Simple Type Definition's DECLARED derivation: which
// of the <simpleType> alternatives Structures §3.16.2.1 (map.std.common, "Common
// mapping rules for Simple Type Definitions") mapped to produce the component,
// carrying only the property that alternative MINTS. It is a sealed sum (STYLE
// T2/T7) mirroring term.go's TermOrRef.
//
// It exists so {variety}, {primitive type definition}, {item type definition}
// and {member type definitions} need not be stored a second time beside the
// {base type definition} they are computed from (STYLE D3). §3.16.2.1 gives a
// ·restriction· none of the four properties of its own — it inherits every one
// from its base — so the derived readers (SimpleType.Variety, .Primitive,
// .Item, .Members) walk the base chain to the nearest ancestor whose arm mints
// the property asked for.
//
// The sum has arms this package does not export. A primitive datatype's arm is
// package-private and minted only by NewPrimitiveType, because primitiveness is
// a DECLARED fact about a builtin datatype (Datatypes §2.4.2 dt-primitive), not
// one recoverable from the graph: a primitive and a user-defined restriction of
// xs:anyAtomicType are both "a restriction whose base is the anchor", and
// telling them apart is what keeps #480's rejection (derivation.go's
// checkAtomicGraph) from becoming a false accept. An exported empty struct
// would be forgeable by any caller, so the arm — and the second package-private
// arm carrying xs:anyAtomicType's own ·absent·-primitive encoding — stay
// unexported.
//
// Consequently there is no exported reader for the stored arm: handing external
// code a sealed sum it cannot switch exhaustively would erode the capability
// (STYLE T2). The exported reader surface is exactly SimpleType.Variety,
// .Primitive, .Item, .Members and .IsPrimitive.
type SimpleTypeDerivation interface{ simpleTypeDerivation() }

// RestrictionDerivation is the ·restriction· alternative (§3.16.2.1, the
// map.std.restriction case). It carries nothing: a restriction mints none of
// {variety}, {primitive type definition}, {item type definition} or {member
// type definitions}, taking each from its {base type definition} instead, which
// is exactly what the derived readers on SimpleType walk the chain to find. It
// is why a restriction of xs:NMTOKENS reports xs:NMTOKEN as its {item type
// definition} without any producer copying the pointer down.
type RestrictionDerivation struct{}

// ListDerivation is the ·list· alternative (§3.16.2.1): {variety} is list and
// the component mints the {item type definition} carried here. Item is exported
// on the TermOrRef precedent (term.go); NewSimpleType stores the arm by value,
// and SimpleType.Item is the read path.
type ListDerivation struct{ Item *SimpleType }

// UnionDerivation is the ·union· alternative (§3.16.2.1): {variety} is union
// and the component mints the {member type definitions} carried here, in
// document order. The sequence is preserved verbatim — not sorted, not
// deduplicated, not filtered of nils — because position is load-bearing
// (cos-st-restricts clause 3.2.2.3 pairs a restriction's members with the
// base's POSITIONALLY, PRINCIPLES 11). Members is exported on the TermOrRef
// precedent (term.go); NewSimpleType COPIES it in and SimpleType.Members copies
// it out, so the stored membership cannot be mutated through a slice the caller
// still holds.
type UnionDerivation struct{ Members []*SimpleType }

// primitiveDerivation is the primitive-datatype arm (Datatypes §2.4.2,
// dt-primitive): {variety} is atomic and the {primitive type definition} is the
// component ITSELF (§3.16.1: "the {primitive type definition} of a primitive
// datatype is that datatype itself"). It is deliberately unexported and minted
// ONLY by NewPrimitiveType — see SimpleTypeDerivation's godoc for why an
// exported empty struct here would re-open #480's false accept.
type primitiveDerivation struct{}

// anyAtomicDerivation is xs:anyAtomicType's own arm (Datatypes §4.1.6), minted
// only for the anyAtomicType package singleton. It is the fifth state the other
// arms cannot express: {variety} is atomic BY FIAT — the anchor's base
// xs:anySimpleType has an ·absent· {variety}, so no restriction arm could
// inherit one — while its {primitive type definition} is ·absent· and it is not
// itself primitive. Encoding it as its own arm rather than as nil-ness of the
// derivation keeps "the derivation is absent" from acquiring a second meaning.
type anyAtomicDerivation struct{}

func (RestrictionDerivation) simpleTypeDerivation() {}
func (ListDerivation) simpleTypeDerivation()        {}
func (UnionDerivation) simpleTypeDerivation()       {}
func (primitiveDerivation) simpleTypeDerivation()   {}
func (anyAtomicDerivation) simpleTypeDerivation()   {}

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
	// FacetPattern is the "pattern" facet (§4.3.4). It has no {fixed}; for how
	// its multiple {value} members combine, see Facet.Values.
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
//   - For every kind except FacetEnumeration and FacetAssertions, {value} is one
//     or more normalized lexical strings — a single string for the single-valued
//     kinds such as length or whiteSpace, or several for the set-valued kind
//     pattern — held in values and read through Values.
//   - For FacetEnumeration (§4.3.5) {value} is a set of EnumerationMembers: each
//     is a lexical {value} member PLUS the namespace context in scope where that
//     <enumeration> was written, which a QName/NOTATION member needs to resolve
//     its prefixes (§3.3.18). The lexical strings are still read through Values
//     (derived from the members, STYLE D3 — not stored twice); the members,
//     with their context, are read through EnumerationMembers. It is held in
//     members and read through EnumerationMembers; values stays nil.
//   - For FacetAssertions (§4.3.13) {value} is "a sequence of Assertion
//     components" (Structures §3.13.1, id="as"), each carrying a Required {test}
//     XPathExpression that a lexical string cannot represent. It is held in
//     assertions and read through Assertions; values stays nil. kind is the sole
//     discriminant among the three representations.
//
// Construct a plain-lexical facet through NewFacet, an enumeration facet through
// NewEnumerationFacet, and an assertions facet through NewAssertionsFacet;
// NewFacet normalizes away the illegal combination of a set {fixed} on a kind
// that has no {fixed} property, so that state is unrepresentable (STYLE T1).
// Facet is immutable after construction.
type Facet struct {
	kind       FacetKind
	values     []string
	members    []EnumerationMember
	assertions []Assertion
	fixed      bool
}

// NewFacet builds a Facet of the given kind carrying values as its {value}. The
// values slice is copied; the caller's backing array is not aliased.
//
// It panics if kind is FacetEnumeration or FacetAssertions: those kinds model
// {value} as richer components than lexical strings — an enumeration facet's
// members carry the namespace context a QName/NOTATION member needs
// (EnumerationMember), and an assertions facet's {value} is a sequence of
// Assertion components — so they must be built through NewEnumerationFacet and
// NewAssertionsFacet respectively. That is a programmer error (the wrong
// constructor), not user-supplied invalid data, so a panic — not an xsderr
// validation error — is the right guard per this package's convention.
//
// fixed is honored only when kind.HasFixed() is true; for FacetPattern (which
// has no {fixed} property, §4.3.4) it is normalized to false so that "fixed set
// on a kind with no {fixed}" cannot be stored (STYLE T1). FacetEnumeration and
// FacetAssertions likewise have no {fixed} (§4.3.5/.13) but are unreachable
// here. Read {fixed} back through Fixed, whose second result reports whether the
// kind has the property at all.
func NewFacet(kind FacetKind, values []string, fixed bool) Facet {
	if kind == FacetEnumeration {
		panic("xsd: NewFacet cannot build an enumeration facet; use NewEnumerationFacet")
	}
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

// NewEnumerationFacet builds the enumeration Constraining Facet (Datatypes
// §4.3.5) whose {value} is a set of EnumerationMembers rather than bare lexical
// strings. Each member pairs a lexical {value} member with the namespace context
// in force where its <enumeration> was written, so a QName/NOTATION member can
// resolve its prefixes against the DECLARING schema's in-scope bindings (§3.3.18)
// rather than the wrong scope. The members slice is copied in document order;
// the caller's backing array is not aliased. The result's kind is
// FacetEnumeration, its values stays nil (Values derives the lexical strings
// from the members, STYLE D3), and its {fixed} is false — §4.3.5 gives the
// enumeration facet no {fixed} property (HasFixed reports false for it). Read the
// members back through EnumerationMembers and their lexical forms through Values.
//
// A context-free member (every non-QName/NOTATION cohort) simply carries empty
// NamespaceBindings and an absent DefaultNamespace, which resolves identically to
// a nil context — so routing every enumeration facet through members changes
// nothing for the context-free kinds.
func NewEnumerationFacet(members []EnumerationMember) Facet {
	f := Facet{kind: FacetEnumeration}
	if len(members) > 0 {
		f.members = append([]EnumerationMember(nil), members...)
	}
	return f
}

// Kind returns the facet's kind.
func (f Facet) Kind() FacetKind {
	return f.kind
}

// Values returns the facet's {value} as lexical strings in document order. It
// returns a copy: mutating the result does not affect f. An empty {value} yields
// nil.
//
// For an enumeration facet (Kind() == FacetEnumeration) the lexical strings are
// DERIVED from the members' Lexical() forms (STYLE D3 — not stored twice), so an
// existing consumer that only reads Values keeps its behavior; a consumer that
// also needs each member's namespace context reads EnumerationMembers instead.
//
// For a pattern facet (Kind() == FacetPattern) the values are ALTERNATIVE
// branches ORed with one another: they are the <pattern> siblings declared at
// ONE derivation step, and a literal satisfies this facet when it matches ANY
// of them. AND-ing happens ACROSS facets, never within one — patterns declared
// at DIFFERENT derivation steps stay separate FacetPattern entries in
// EffectiveFacets, and a literal is pattern-valid only if EVERY one of those
// entries accepts it (cvc-pattern-valid, §4.3.4.4). That is deliberately the
// mirror image of the literal {value} encoding of §4.3.4.2 (xr-pattern), which
// first joins same-step siblings into ONE regex with "|", so that a
// multi-member {value} can only have come from union ACROSS steps; a reader
// carrying that encoding over to Values would read a multi-member result as
// ANDed, which inverts its meaning. The two encodings accept exactly the same
// literals because XSD pattern regexes are implicitly whole-string anchored,
// which makes matching the single branch-set "a|b" the same as matching "a" or
// matching "b" (equivalence established on #214).
func (f Facet) Values() []string {
	if f.kind == FacetEnumeration {
		if len(f.members) == 0 {
			return nil
		}
		out := make([]string, len(f.members))
		for i, m := range f.members {
			out[i] = m.lexical
		}
		return out
	}
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

// EnumerationMembers returns the facet's {value} as a set of EnumerationMembers
// in document order. The second result reports whether this is an enumeration
// facet (Kind() == FacetEnumeration): when it is false the facet models {value}
// differently (use Values, or Assertions for FacetAssertions), and the first
// result is nil. It returns a copy: mutating the result does not affect f. An
// enumeration facet with no members yields nil.
//
// Each member carries the namespace context in force where its <enumeration> was
// written, which a downstream QName/NOTATION value-space consumer needs to
// resolve a prefixed member against the DECLARING schema's in-scope bindings
// (§3.3.18); the lexical strings alone are still available through Values.
func (f Facet) EnumerationMembers() (members []EnumerationMember, ok bool) {
	if f.kind != FacetEnumeration {
		return nil, false
	}
	if len(f.members) == 0 {
		return nil, true
	}
	return append([]EnumerationMember(nil), f.members...), true
}

// Fixed returns the {fixed} property. The second result is Kind().HasFixed():
// when it is false the kind has no {fixed} property (FacetPattern,
// FacetEnumeration, FacetAssertions) and the first result is not meaningful.
func (f Facet) Fixed() (fixed bool, ok bool) {
	return f.fixed, f.kind.HasFixed()
}

// EnumerationMember is one member of an enumeration facet's {value} (Datatypes
// §4.3.5): a lexical {value} member paired with the namespace context in force
// where its <enumeration> element was written. That context is load-bearing for
// a QName or NOTATION member, whose lexical→value mapping resolves the member's
// prefix against the in-scope namespace bindings at the point of the literal
// (§3.3.18): a member "foo:fo" denotes {namespace name, local name} only once its
// "foo" prefix is resolved, and §3.3.18 fixes the scope to the <enumeration>
// element's own [in-scope namespaces], NOT the validated instance's. Because a
// facet's {value} is computed once at schema-construction time and rides the
// §3.16.6.4 overlay unchanged, that context must travel WITH the member; this
// package carries it opaquely (like XPathExpression's {namespace bindings}) and
// defers the actual prefix resolution to a value-space consumer.
//
// The context is modeled exactly as XPathExpression models its own: a
// document-order slice of NamespaceBinding (the SAME shared record, STYLE D3),
// plus an optional {default namespace}. Unlike XPathExpression there is no {base
// URI} — QName resolution is prefix→namespace only, with no base-URI dependency.
//
// A context-free member (every non-QName/NOTATION cohort) carries no bindings and
// an absent {default namespace}, which resolves identically to a nil context, so
// routing all enumeration facets through members leaves the context-free kinds
// unchanged.
//
// Construct only through NewEnumerationMember. EnumerationMember is immutable
// after construction.
type EnumerationMember struct {
	lexical             string
	namespaceBindings   []NamespaceBinding
	defaultNamespace    string
	hasDefaultNamespace bool
}

// NewEnumerationMember builds an EnumerationMember pairing the lexical {value}
// member with its namespace context. namespaceBindings is copied in document
// order; the caller's backing array is not aliased. A nil defaultNamespace means
// the {default namespace} is absent; a non-nil pointer (including to "") means it
// is present, because "" is a legal anyURI and cannot double as an absence
// sentinel (mirrors NewXPathExpression's discipline).
//
// There is no rejectable state at this structural layer: whether a QName/NOTATION
// member's prefix actually resolves is a value-space verdict a consumer makes
// (charged to src-enumeration-value, §4.3.5.3), not something this pure-leaf
// package can or should reject.
func NewEnumerationMember(lexical string, namespaceBindings []NamespaceBinding, defaultNamespace *string) EnumerationMember {
	m := EnumerationMember{lexical: lexical}
	if len(namespaceBindings) > 0 {
		m.namespaceBindings = append([]NamespaceBinding(nil), namespaceBindings...)
	}
	if defaultNamespace != nil {
		m.defaultNamespace, m.hasDefaultNamespace = *defaultNamespace, true
	}
	return m
}

// Lexical returns the member's raw lexical {value} — the normalized value of the
// <enumeration> element's value attribute (§4.3.5.2), before any QName prefix
// resolution.
func (m EnumerationMember) Lexical() string {
	return m.lexical
}

// NamespaceBindings returns the member's namespace context in document order:
// the in-scope namespace bindings at its <enumeration> element (§3.3.18). It
// returns a copy: mutating the result does not affect m. An empty context yields
// nil.
func (m EnumerationMember) NamespaceBindings() []NamespaceBinding {
	if len(m.namespaceBindings) == 0 {
		return nil
	}
	return append([]NamespaceBinding(nil), m.namespaceBindings...)
}

// DefaultNamespace returns the member's {default namespace} — the namespace an
// unprefixed member name binds to in scope at its <enumeration> element. The
// second result is false when it is absent (no default namespace in scope), in
// which case the first result is not meaningful.
func (m EnumerationMember) DefaultNamespace() (string, bool) {
	return m.defaultNamespace, m.hasDefaultNamespace
}

// SimpleType is a Simple Type Definition component (Structures §3.16.1,
// Datatypes §4.1.1), the datatypes-facing subset: {name} (bundled with {target
// namespace} as a QName), {variety}, {base type definition}, the type's own
// contribution to {facets}, and {final}. Full complex-type breadth
// (union-of-complex, {context}, {annotations}) is out of this component's
// scope.
//
// What it STORES is narrower than what it reports: the declared
// SimpleTypeDerivation, not {variety}. That property, and {primitive type
// definition}/{item type definition}/{member type definitions} with it, are
// derived on demand from the derivation and the {base type definition} chain
// per §3.16.2.1 (STYLE D3) — see Variety, Primitive, Item and Members.
//
// The {base type definition} is a SimpleTypeOrRef, which may be a DEFERRED
// reference by name (simpletyperef.go). Every reader that walks the chain —
// Base, Variety, Primitive, Item, Members, EffectiveFacets — therefore takes a
// TypeResolver and returns an error: an unresolvable base is a failure the
// caller must see, never a silently short chain, which would be a false accept.
//
// Unlike the value-typed components in this package (Occurs, Notation),
// SimpleType is handled through a *SimpleType pointer: components reference one
// another by pointer once resolved (phased construction, PRINCIPLES D4), and
// component identity matters — two *SimpleType values denote the same component
// if and only if they are the same pointer, whereas two equal Occurs values are
// interchangeable. Construct only through NewSimpleType; a SimpleType is
// immutable after construction.
type SimpleType struct {
	loc        xsderr.Loc // source position; provenance, not a §3.16.1 property
	name       QName
	derivation SimpleTypeDerivation
	base       SimpleTypeOrRef
	ownFacets  []Facet
	final      []DerivationMethod
}

// NewSimpleType builds a Simple Type Definition. base is the {base type
// definition} slot (SimpleTypeOrRef, simpletyperef.go): nil means this type IS
// xs:anySimpleType (the one simple type whose base is xs:anyType, a Complex Type
// Definition outside this package's scope); every other simple type carries
// either a SimpleTypeRef naming its base or an OwnedSimpleType holding it.
// derivation is the declared §3.16.2.1 alternative the component was mapped
// from, and it is what {variety}, {primitive type definition}, {item type
// definition} and {member type definitions} are DERIVED from — none of the four
// is passed or stored (STYLE D3). It may be nil to model xs:anySimpleType, which
// was mapped from no alternative at all and whose {variety} is ·absent·.
//
// The two ·special· types are constrained bases. xs:anySimpleType is the base a
// ·constructed· list or union names (cos-st-restricts 2.2.1/3.2.1), and those
// declare a ListDerivation or UnionDerivation that mints a {variety} of its
// own; a caller that names it under a RestrictionDerivation inherits the absent
// {variety} instead, which is the shape st-props-correct clause 1 reserves for
// xs:anySimpleType itself. xs:anyAtomicType may be named only by a primitive
// datatype (Datatypes §2.4.2), whose absent {primitive type definition} the same
// clause reserves likewise, and which is built through NewPrimitiveType instead.
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
// and, charging xsderr.RuleComponentInvariant, the two illegal encodings of the
// base slot itself — a SimpleTypeRef naming nothing and an OwnedSimpleType
// holding nothing (checkSimpleTypeOrRef).
//
// Everything the CROSS-REFERENCE constraints decide — every cos-st-restricts
// sub-clause and st-props-correct clauses 1, 2, 3 and 5 — is charged at
// FINALIZE, not here, because a deferred base cannot be followed at
// construction: CheckDerivation (derivation.go) charges the graph clauses and
// Schema.checkSimpleBaseAcyclic charges clause 2. A component this constructor
// returns is well FORMED, not yet known to be well DERIVED; the one entry point
// that settles the latter is CheckDerivation, which finalize runs over every
// simple type an assembled Schema reaches and which a Schema-less caller runs
// itself.
//
// ownFacets and final are copied; the caller's backing arrays are not aliased,
// and so is a UnionDerivation's Members (see copyDerivation).
//
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the type's provenance. Pass the position of this type's own
// declaring element, never a convenient nearby one (a parent element's, say) —
// it is observable, not merely an error-charging convenience. A caller with no
// real parser position — a synthesized or programmatically built type, as every
// seeded built-in datatype is — passes the zero xsderr.Loc{}, which reads as
// "unknown".
func NewSimpleType(loc xsderr.Loc, name QName, derivation SimpleTypeDerivation, base SimpleTypeOrRef, ownFacets []Facet, final []DerivationMethod) (*SimpleType, error) {
	if err := checkSimpleTypeOrRef(loc, base); err != nil {
		return nil, err
	}
	if err := checkSTProps(loc, ownFacets, final); err != nil {
		return nil, err
	}
	t := &SimpleType{loc: loc, name: name, derivation: copyDerivation(derivation), base: base}
	t.setOwnFacetsFinal(ownFacets, final)
	return t, nil
}

// copyDerivation returns derivation with any caller-owned backing array copied,
// so the stored arm cannot be mutated through a slice the caller still holds.
// UnionDerivation.Members is the only such array — every other arm is empty or
// carries a single pointer — and a mutable membership is what would otherwise
// let a caller splice a cycle into a union's transitive membership after
// construction, which the recursive membership walks in derivation.go rely on
// being impossible for termination.
//
// The sequence is copied verbatim: not sorted, not deduplicated, not filtered
// of nils (cos-st-restricts clause 3.2.2.3 pairs members POSITIONALLY,
// PRINCIPLES 11). An empty membership is returned unchanged; there is nothing
// to alias, and SimpleType.Members reports it as nil either way.
func copyDerivation(derivation SimpleTypeDerivation) SimpleTypeDerivation {
	u, ok := derivation.(UnionDerivation)
	if !ok || len(u.Members) == 0 {
		return derivation
	}
	return UnionDerivation{Members: append([]*SimpleType(nil), u.Members...)}
}

// NewPrimitiveType builds a primitive datatype (Datatypes §2.4.2): one of the
// types whose {base type definition} is xs:anyAtomicType (§3.16.1). It fixes
// the {base type definition} to the canonical xs:anyAtomicType anchor (see
// AnyAtomicType) so pointer identity holds across every graph, and it declares
// the package-private primitive derivation arm — the sole mint of it, which is
// what makes primitiveness unforgeable from outside this package (see
// SimpleTypeDerivation). From that arm the returned node's {variety} derives as
// Atomic, its {primitive type definition} as the node ITSELF (§3.16.1: "the
// {primitive type definition} of a primitive datatype is that datatype
// itself"), and IsPrimitive as true.
//
// The arm is set inside this constructor, before the node escapes, so the node
// is immutable to every external caller. ownFacets and final follow
// NewSimpleType's contract and are validated identically (st-props-correct);
// they are copied, not aliased. The base is fixed as an OwnedSimpleType holding
// the anchor — never a SimpleTypeRef — because the anchor's identity is what
// checkAtomicGraph's #480 rejection keys on, and a name-based base would make
// that identity depend on which Schema resolved it. loc follows NewSimpleType's
// contract too: it is charged to any rejection AND retained, so Loc reports it
// back as the type's provenance. Like NewSimpleType it charges no
// cross-reference constraint; CheckDerivation does, at finalize.
func NewPrimitiveType(loc xsderr.Loc, name QName, ownFacets []Facet, final []DerivationMethod) (*SimpleType, error) {
	if err := checkSTProps(loc, ownFacets, final); err != nil {
		return nil, err
	}
	t := &SimpleType{loc: loc, name: name, derivation: primitiveDerivation{}, base: OwnedSimpleType{Definition: anyAtomicType}}
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

// Loc reports the source position of the declaring element — provenance, not a
// §3.16.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown, as it is for every seeded
// built-in datatype and for the xs:anySimpleType/xs:anyAtomicType anchors.
func (t *SimpleType) Loc() xsderr.Loc {
	return t.loc
}

// Variety returns the {variety} property (§3.16.1): Atomic{}, List{} or
// Union{}, or nil when the property is ·absent· — which is xs:anySimpleType.
//
// It is DERIVED, never stored (STYLE D3). §3.16.2.1 gives a ·restriction· the
// {variety} of its {base type definition}, so a RestrictionDerivation delegates
// to the base and the answer comes from the nearest ancestor whose arm mints
// one: a list or union alternative, the primitive arm (atomic), or
// xs:anyAtomicType's own arm (atomic by fiat, Datatypes §4.1.6).
//
// r follows the {base type definition} chain, which may be a deferred
// SimpleTypeRef (simpletyperef.go). An unresolvable base is returned as an
// ERROR, never as a short chain answering nil: nil is the ·absent· {variety}
// only xs:anySimpleType may carry, so reporting it for a base that merely could
// not be found would hand every caller st-props-correct clause 1's shape for a
// type that does not have it.
//
// TERMINATION: the walk carries no visited set (STYLE D4). Every base chain a
// finalized Schema holds is acyclic, which Schema.checkSimpleBaseAcyclic
// establishes in Phase B before any pass that walks one runs (resolve.go,
// PRINCIPLES 9). A caller resolving against something other than a finalized
// Schema owes the same guarantee — the Schema-less callers in this module get it
// from OwnedSimpleType-only chains, where a base must pre-exist the type naming
// it and so cannot close a loop.
func (t *SimpleType) Variety(r TypeResolver) (Variety, error) {
	switch t.derivation.(type) {
	case primitiveDerivation, anyAtomicDerivation:
		return Atomic{}, nil
	case ListDerivation:
		return List{}, nil
	case UnionDerivation:
		return Union{}, nil
	case RestrictionDerivation:
		base, err := t.Base(r)
		if err != nil || base == nil {
			return nil, err
		}
		return base.Variety(r)
	}
	return nil, nil
}

// Primitive returns the {primitive type definition} property (§3.16.1). It is
// nil when that property is ·absent·: on a list or union, on xs:anySimpleType,
// and on xs:anyAtomicType, the one atomic type the Datatypes §4.1.1 property
// tableau exempts from carrying one.
//
// It is DERIVED, never stored (STYLE D3): a primitive datatype's {primitive
// type definition} is itself, and st-restrict-facets clause 2 gives a
// restriction the same one as its base, so the answer is the nearest primitive
// ancestor on the {base type definition} chain. r resolves that chain and an
// unresolvable base is an error, for the reasons Variety's godoc states.
func (t *SimpleType) Primitive(r TypeResolver) (*SimpleType, error) {
	switch t.derivation.(type) {
	case primitiveDerivation:
		return t, nil
	case RestrictionDerivation:
		base, err := t.Base(r)
		if err != nil || base == nil {
			return nil, err
		}
		return base.Primitive(r)
	}
	return nil, nil
}

// Item returns the {item type definition} property (§3.16.1). It is nil when
// that property is ·absent·, which is every {variety} but list — a state
// checkListGraph rejects for a list, so a list that passed CheckDerivation
// always reports a non-nil item.
//
// It is DERIVED, never stored (STYLE D3): the ·list· alternative mints it and a
// ·restriction· takes its base's (§3.16.2.1), so a restriction of xs:NMTOKENS
// reports xs:NMTOKEN without any producer copying the pointer down.
//
// It takes a resolver NOT for symmetry with Variety/Primitive but because that
// inheritance hop IS the {base type definition} slot: the ListDerivation.Item
// SLOT stays a plain *SimpleType (see SimpleTypeOrRef), yet a ·restriction· of a
// named list reaches its item only THROUGH a base that may be a SimpleTypeRef.
// Answering nil there would report an absent {item type definition} for a list
// that has one, which checkListGraph turns into a false reject.
func (t *SimpleType) Item(r TypeResolver) (*SimpleType, error) {
	switch d := t.derivation.(type) {
	case ListDerivation:
		return d.Item, nil
	case RestrictionDerivation:
		base, err := t.Base(r)
		if err != nil || base == nil {
			return nil, err
		}
		return base.Item(r)
	}
	return nil, nil
}

// Members returns the {member type definitions} property (§3.16.1) in document
// order. It returns a copy: mutating the result does not affect t. An absent or
// empty membership yields nil.
//
// It is DERIVED, never stored (STYLE D3): the ·union· alternative mints it and
// a ·restriction· takes its base's (§3.16.2.1). It takes a resolver for exactly
// the reason Item does, and not for symmetry — see there.
func (t *SimpleType) Members(r TypeResolver) ([]*SimpleType, error) {
	switch d := t.derivation.(type) {
	case UnionDerivation:
		if len(d.Members) == 0 {
			return nil, nil
		}
		return append([]*SimpleType(nil), d.Members...), nil
	case RestrictionDerivation:
		base, err := t.Base(r)
		if err != nil || base == nil {
			return nil, err
		}
		return base.Members(r)
	}
	return nil, nil
}

// Base resolves and returns the {base type definition} property. It is nil, with
// a nil error, if and only if IsAnySimpleType reports true — that is, when this
// type IS xs:anySimpleType, whose real base (xs:anyType) is a Complex Type
// Definition outside this package's scope.
//
// The stored slot is a SimpleTypeOrRef (simpletyperef.go), so for a by-name base
// this is the src-resolve clause 1.1 lookup against r. It is the ONE reader of
// that slot in this package and the single encoding of the resolution (STYLE
// T4): Variety, Primitive, Item, Members and EffectiveFacets all walk the chain
// through it.
//
// A base that r cannot resolve is an ERROR, never (nil, nil): a caller reading a
// missing base as the end of the chain would compute {variety}, {primitive type
// definition} or {facets} off a truncated chain and accept what the full chain
// forbids.
func (t *SimpleType) Base(r TypeResolver) (*SimpleType, error) {
	return simpleTypeOfRef(r, t.base, t.loc, simpleTypeLabel(t)+" {base type definition}")
}

// IsAnySimpleType reports whether this type is xs:anySimpleType, the root of the
// simple-type hierarchy (§3.16.1). It is exactly the condition "the {base type
// definition} slot is absent", exposed as a predicate so callers do not infer
// this identity from nil-ness. It needs no resolver and cannot fail: absence is
// a property of the SLOT, decided without following anything (SimpleTypeOrRef).
func (t *SimpleType) IsAnySimpleType() bool {
	return t.base == nil
}

// IsPrimitive reports whether this type is a primitive datatype (Datatypes
// §2.4.2). The two special types xs:anySimpleType and xs:anyAtomicType are
// themselves not primitive, and this returns false for them.
//
// It reads the declared derivation — the package-private arm only
// NewPrimitiveType mints — rather than testing {base type definition} ==
// xs:anyAtomicType. The two agree on every component this package can build,
// because §3.16.1's "a type definition has ·xs:anyAtomicType· as its {base type
// definition} if and only if it is one of the primitive datatypes" is enforced
// from the other side: checkAtomicGraph (derivation.go) rejects any type but a
// primitive that names the anchor as its base (#480). Reading the arm is what
// keeps that rejection decidable at all — see SimpleTypeDerivation's godoc.
func (t *SimpleType) IsPrimitive() bool {
	_, ok := t.derivation.(primitiveDerivation)
	return ok
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
//
// r resolves each {base type definition} hop, which may be a deferred
// SimpleTypeRef; an unresolvable one is returned as an error rather than ending
// the chain, because a truncated overlay silently DROPS every inherited facet
// above the break, which is a false accept of any literal those facets exclude.
func (t *SimpleType) EffectiveFacets(r TypeResolver) ([]EffectiveFacet, error) {
	// Collect the base chain most-derived first (t, then its base, ...).
	var chain []*SimpleType
	for s := t; s != nil; {
		chain = append(chain, s)
		next, err := s.Base(r)
		if err != nil {
			return nil, err
		}
		s = next
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
		return nil, nil
	}
	return result, nil
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
// id="xr-assertions"). The assertions {value} on a restriction is the base
// type's Assertions followed by the restriction's own new Assertions, in that
// order — an append, never a set-union and never deduplicated. So when acc
// already holds a FacetAssertions entry (the base's accumulated {value}) and f
// is also FacetAssertions, the two are merged into a single NewAssertionsFacet
// whose sequence is the existing (base) Assertions PREPENDED before f's own,
// and that merged facet takes f's more-derived position. The merged facet's
// declaring QName is f's — the most-derived contributor (see EffectiveFacet's
// godoc for why per-assertion provenance is not recoverable from the result).
// With no prior FacetAssertions entry in acc, f is appended unchanged, exactly
// as for every other kind.
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
// primitive on THIS node (NewPrimitiveType does so) so the pointer-identity
// tests that key on the anchor — checkAtomicGraph's #480 rejection,
// isSpecialType — hold across the whole graph. The anchor reports Atomic
// {variety} while its own {primitive type definition} is ·absent· and
// IsPrimitive is false. The returned node is read-only; do not mutate it.
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
// whose {primitive type definition} is itself ·absent·. Its derivation is the
// package-private anyAtomicDerivation arm, which is the ONLY encoding of that
// triple — atomic {variety}, absent {primitive type definition}, not primitive
// — and is minted here and nowhere else. Exposed to producers through
// AnyAtomicType.
var anyAtomicType = &SimpleType{
	name:       QName{Space: XMLSchemaNS, Local: "anyAtomicType"},
	derivation: anyAtomicDerivation{},
	base:       OwnedSimpleType{Definition: anySimpleType},
}
