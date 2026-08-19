package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// anyTypeName is the expanded name of xs:anyType, the ur-type (§3.4.7).
var anyTypeName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}

// anySimpleTypeName is the expanded name of xs:anySimpleType: the default
// {type definition} of a type=-less <attribute> (§3.2.2.1) and the fallback
// {simple type definition} of §3.4.2.2 case 5. [builtin.Seed] always seeds it,
// so symbols.built holds it before any document is produced.
var anySimpleTypeName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"}

// seedAnyType builds the ur-type Complex Type Definition xs:anyType (§3.4.7): a
// mixed complex type whose {content type} is a 1..1 sequence wrapping a single
// 0..unbounded lax ##any element wildcard, with a lax ##any attribute wildcard
// and no attribute uses, and whose {base type definition} is itself (the sole
// permitted self-derivation, any-type-itself). checkComplexBaseAcyclic (#173)
// recognises that self-derivation by name, so the seeded value passes finalize.
//
// Its {final} and {prohibited substitutions} stay EMPTY, and that is the mapped
// value rather than an unmapped one: §3.4.2.1's EBV is read off a source
// <complexType> and its ancestor <schema>, and the ur-type has neither — §3.4.7
// specifies the component outright. Nothing a document says, finalDefault=
// included, can make xs:anyType final.
func seedAnyType() (xsd.ComplexType, error) {
	anyNS, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	wildcard, err := xsd.NewWildcard(xsderr.Loc{}, anyNS, xsd.ProcessLax, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	inner, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, 0)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	wildcardParticle, err := xsd.NewParticle(xsderr.Loc{}, inner, xsd.ResolvedTerm{Term: wildcard}, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	seq, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{wildcardParticle}, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	oneOne, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	topParticle, err := xsd.NewParticle(xsderr.Loc{}, oneOne, xsd.ResolvedTerm{Term: seq}, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	content := xsd.ElementContent{Mixed: true, Particle: topParticle}
	return xsd.NewComplexType(xsderr.Loc{}, anyTypeName, anyTypeName, nil,
		xsd.DerivationRestriction, false, nil, nil, &wildcard, content, nil, nil, nil)
}

// complexTypeIdentity is what a <complexType> under production is identified by.
// It is a SEALED SUM (STYLE T1/T2), unexported and sealed by the
// complexTypeIdentity marker method, with one arm per construction path §3.4.2
// and §4.2.4 admit:
//
//   - namedComplexType — a top-level <complexType>, identified by its expanded
//     {name} (§3.4.1 makes {context} ·absent· for it);
//   - elementOwnedComplexType — an inline ANONYMOUS <complexType>, which §3.4.1
//     ctd-context leaves nameless and gives a {context} instead: the minted
//     identity of the owning <element>'s declaration (§3.4.2.1 dcl.ctd.common,
//     "the Element Declaration corresponding to the nearest <element>
//     information item among the ancestor element information items");
//   - redefiningComplexType — a <complexType> child of a <redefine>, which is
//     NAMED and ALSO carries a minted identity, because src-expredef clause 1.1
//     makes it the {context} of the anonymous original it is paired with;
//   - redefineOriginalComplexType — that original: anonymous, contexted in the
//     redefining component, carrying the same minted identity, and itself owning
//     a further original when a chained <redefine> put a redefining declaration
//     in the redefined document (#585);
//   - typeAlternativeOwnedComplexType — the inline ANONYMOUS <complexType> of an
//     <alternative> (§3.12.2 declare-ta's second arm), which §3.4.2.1
//     dcl.ctd.common contexts in the enclosing ELEMENT declaration, walking past
//     the <alternative>, and which therefore carries TWO tokens: that shared
//     context and its own per-edge container mint.
//
// The last three are why this is a sum and not a two-optional-field struct. The
// struct it replaced derived "is this anonymous?" from owner-presence, which
// classifies a redefining type — named AND owner-carrying — as anonymous and
// gives every local element it declares an AnonymousComplexTypeScopeParent
// instead of a ComplexTypeScopeParent: a wrong {scope}.{parent}, representable
// only because a derived predicate over an optional-field pair cannot see the
// difference (STYLE T1). With one arm per path the shape is unrepresentable, and
// the "carries neither a name nor an owner" panics the struct needed are gone
// with the zero value they guarded.
//
// It never escapes the parser — it is the producer's own threading value, not a
// component property — and it decides three things at once, which is why it is
// one value rather than several parameters: which xsd constructor builds the
// type, which xsd.ElementScopeParent variant its nested local element
// declarations report, and which xsd.AttributeScopeParent variant its local
// attribute declarations report.
type complexTypeIdentity interface{ complexTypeIdentity() }

// namedComplexType identifies a top-level <complexType> by its expanded {name}.
// A name whose local part is empty is a grammar fault, rejected before anything
// is built — by topLevelName on run's dispatch path and by produceComplexType
// on the on-demand base-build path (see produceComplexType).
type namedComplexType struct{ name xsd.QName }

// elementOwnedComplexType identifies an inline anonymous <complexType> by the
// minted identity of the element declaration that owns it — the same
// xsd.ComponentID that declaration is built with and that the type's own
// {context} carries (§3.4.2.1 dcl.ctd.common). One mint per inline construct.
type elementOwnedComplexType struct{ owner xsd.ComponentID }

// redefiningComplexType identifies a <complexType> child of a <redefine>:
// src-expredef clause 1.2's "component which corresponds to the information item
// itself … except that its {base type definition} is the component defined in
// clause 1.1". It is NAMED — its name is what references in both the
// <redefine>ing and the <redefine>d document resolve to — and it also carries a
// minted identity, because clause 1.1 makes the original's {context} "the
// redefining component" and a ComplexTypeContext names a component by
// xsd.ComponentID, never by QName. One mint per redefinition, threaded into both
// halves of the pairing and stored on neither (see xsd.NewComplexTypeOwningBase).
type redefiningComplexType struct {
	name  xsd.QName
	owner xsd.ComponentID
}

// redefineOriginalComplexType identifies src-expredef clause 1.1's original: the
// component "which corresponds to the top-level definition item with the same
// name in the <redefine>d schema document … except that its {name} is ·absent·
// and its {context} is the redefining component".
//
// It is produced under the REDEFINED document's own producer, so its content
// takes that document's target namespace and schema-level defaults; only the
// {name}/{context} override travels with this identity.
//
// It carries TWO identities because an original may stand at both ends of an
// ownership edge. A CHAINED <redefine> — D1 redefines D2, D2 redefines D3, one
// (kind, name) throughout — makes D2's own redefining declaration the top-level
// definition item clause 1.1 reads, so the original built from it is anonymous
// AND owns an original of its own (D3's), under clause 1.2 applied at D2's level.
// Build it through newRedefineOriginal, never as a literal, so ownedOriginal is
// never the zero token.
type redefineOriginalComplexType struct {
	// owner is the identity of the component this original's {context} names —
	// clause 1.1's "the redefining component", the same value that component's
	// own identity carries — and the token its local declarations report as
	// {scope}.{parent} (xsd.AnonymousComplexTypeScopeParent's 1:1 pairing).
	owner xsd.ComponentID
	// ownedOriginal is the identity minted for the edge running the other way:
	// the {context} of the clause-1.1 original THIS one owns, when the
	// declaration it is built from is itself a redefining declaration. One mint
	// per edge — reusing owner for both would leave the two containers
	// indistinguishable, and their {context}s equal.
	ownedOriginal xsd.ComponentID
}

// newRedefineOriginal mints the identity of one clause-1.1 original: owner is the
// identity of the component that owns it, and the original's own owner-side token
// is minted here, one per ownership edge (see redefineOriginalComplexType).
func newRedefineOriginal(owner xsd.ComponentID) redefineOriginalComplexType {
	return redefineOriginalComplexType{owner: owner, ownedOriginal: xsd.NewComponentID()}
}

// typeAlternativeOwnedComplexType identifies the inline anonymous <complexType>
// of an <alternative> child of an <element> (§3.12.2 declare-ta: "the type
// definition corresponding to the complexType or simpleType among the children
// of the <alternative> element").
//
// It carries TWO tokens because the two directions elementOwnedComplexType
// collapses into one come apart here. §3.4.2.1 dcl.ctd.common gives the type "the
// Element Declaration corresponding to the nearest <element> information item
// among the ancestor element information items" — the <alternative> is not an
// <element>, so the walk goes straight past it and the {context} is the OWNING
// declaration's identity, shared with that element's own inline type and with
// every sibling <alternative>'s. The {scope}.{parent} its nested local
// declarations report cannot be that shared token: one owner reaching several
// anonymous containers would leave them indistinguishable
// (xsd.AnonymousComplexTypeScopeParent), so each <alternative> edge gets a mint
// of its own. Build it through newTypeAlternativeOwned, never as a literal, so
// container is never the zero token.
type typeAlternativeOwnedComplexType struct {
	// owner is the identity of the element declaration this type's {context}
	// names, minted by the producer before either component exists and threaded
	// into xsd.NewElementDeclarationOwningTypes as well.
	owner xsd.ComponentID
	// container is the identity this type is known by as a CONTAINER: the token
	// its nested local element and attribute declarations report as
	// {scope}.{parent}. One mint per <alternative> ownership edge.
	container xsd.ComponentID
}

// newTypeAlternativeOwned mints the identity of one <alternative>'s inline
// anonymous complex type: owner is the enclosing element declaration's identity,
// and the container token is minted here, one per ownership edge (see
// typeAlternativeOwnedComplexType).
func newTypeAlternativeOwned(owner xsd.ComponentID) typeAlternativeOwnedComplexType {
	return typeAlternativeOwnedComplexType{owner: owner, container: xsd.NewComponentID()}
}

// redefineOriginalContext returns the identity a src-expredef clause 1.1 ORIGINAL
// built under this type takes as its {context}, and false for the two arms that
// can own no original. It is the one place the two directions of the "owner"
// field are told apart: for a redefining type it is the mint clause 1.1's
// original is paired with, and for an original that is ITSELF built from a
// redefining declaration it is the second mint that original carries.
//
// The switch is exhaustive over the sealed sum; see topLevelComplexTypeName for
// why the default arm is unreachable.
func redefineOriginalContext(id complexTypeIdentity) (xsd.ComponentID, bool) {
	switch i := id.(type) {
	case redefiningComplexType:
		return i.owner, true
	case redefineOriginalComplexType:
		return i.ownedOriginal, true
	case namedComplexType, elementOwnedComplexType, typeAlternativeOwnedComplexType:
		return xsd.ComponentID{}, false
	default:
		panic("parser: redefineOriginalContext: non-exhaustive complexTypeIdentity switch")
	}
}

func (namedComplexType) complexTypeIdentity()                {}
func (elementOwnedComplexType) complexTypeIdentity()         {}
func (redefiningComplexType) complexTypeIdentity()           {}
func (redefineOriginalComplexType) complexTypeIdentity()     {}
func (typeAlternativeOwnedComplexType) complexTypeIdentity() {}

// topLevelComplexTypeName returns the expanded {name} an identity gives the type
// it builds, and false for the three arms that give none. The two NAMED arms are
// the two that can carry an unusable (empty-local-part) name, so this is what
// produceComplexType's grammar-fault guard reads.
//
// The switch is exhaustive over the sealed sum; the default arm asserts the
// invariant and is unreachable, since complexTypeIdentity is unexported and this
// file declares every arm.
func topLevelComplexTypeName(id complexTypeIdentity) (xsd.QName, bool) {
	switch i := id.(type) {
	case namedComplexType:
		return i.name, true
	case redefiningComplexType:
		return i.name, true
	case elementOwnedComplexType, redefineOriginalComplexType, typeAlternativeOwnedComplexType:
		return xsd.QName{}, false
	default:
		panic("parser: topLevelComplexTypeName: non-exhaustive complexTypeIdentity switch")
	}
}

// scopeParent returns the {scope}.{parent} every local element declaration
// nested in this type's content model reports (§3.3.2.3 dcl.elt.local: "the
// Complex Type Definition corresponding to that item"), in the variant the
// container's own identity admits.
//
// The three anonymous arms all yield AnonymousComplexTypeScopeParent carrying
// the token of the OWNERSHIP EDGE that reaches the container, which is that
// field's invariant (see xsd.AnonymousComplexTypeScopeParent). For a redefine
// original that token names the REDEFINING complex type — not because the
// original's locals are scoped to the redefinition (they are scoped to the
// original, which is what the token identifies), but because the original has no
// name and this token is what it is identified by. For a Type-Alternative-owned
// type it is the per-alternative CONTAINER token and never the shared owner: the
// owner reaches one anonymous type per <alternative> plus possibly the element's
// own, and one token across all of them would leave their nested locals reporting
// the same {scope}.{parent}.
//
// The switch is exhaustive over the sealed sum; see topLevelComplexTypeName for
// why the default arm is unreachable.
func scopeParentOf(id complexTypeIdentity) xsd.ElementScopeParent {
	switch i := id.(type) {
	case namedComplexType:
		return xsd.ComplexTypeScopeParent{Name: i.name}
	case redefiningComplexType:
		return xsd.ComplexTypeScopeParent{Name: i.name}
	case elementOwnedComplexType:
		return xsd.AnonymousComplexTypeScopeParent{Owner: i.owner}
	case redefineOriginalComplexType:
		return xsd.AnonymousComplexTypeScopeParent{Owner: i.owner}
	case typeAlternativeOwnedComplexType:
		return xsd.AnonymousComplexTypeScopeParent{Owner: i.container}
	default:
		panic("parser: scopeParentOf: non-exhaustive complexTypeIdentity switch")
	}
}

// attributeScopeParentOf returns the {scope}.{parent} every local attribute
// declaration among this type's own attribute content reports (§3.2.2.2
// dcl.att.local: "the Complex Type Definition corresponding to that item"), in
// the variant the container's own identity admits. It is the attribute-side twin
// of scopeParentOf: sc_a's alternation is CTD | AGD where sc_e's is CTD | MGD, so
// the two sums are distinct types and this function cannot be folded into that
// one.
//
// It is NOT what an attribute reached through an <attributeGroup ref> reports:
// such an attribute's own ancestor axis has no <complexType> in it, so
// collectReferencedGroup rebinds the parent to the group at the hop.
//
// The switch is exhaustive over the sealed sum; see topLevelComplexTypeName for
// why the default arm is unreachable.
func attributeScopeParentOf(id complexTypeIdentity) xsd.AttributeScopeParent {
	switch i := id.(type) {
	case namedComplexType:
		return xsd.AttributeComplexTypeScopeParent{Name: i.name}
	case redefiningComplexType:
		return xsd.AttributeComplexTypeScopeParent{Name: i.name}
	case elementOwnedComplexType:
		return xsd.AttributeAnonymousComplexTypeScopeParent{Owner: i.owner}
	case redefineOriginalComplexType:
		return xsd.AttributeAnonymousComplexTypeScopeParent{Owner: i.owner}
	case typeAlternativeOwnedComplexType:
		return xsd.AttributeAnonymousComplexTypeScopeParent{Owner: i.container}
	default:
		panic("parser: attributeScopeParentOf: non-exhaustive complexTypeIdentity switch")
	}
}

// newComplexType builds the Complex Type Definition this identity names, through
// the xsd entry point its arm admits: xsd.NewComplexType for a plain top-level
// type, xsd.NewComplexTypeOwningBase for a redefining one (which owns its base),
// and xsd.NewAnonymousComplexType for the three anonymous arms, whose §3.4.1
// tableau makes {context} Required and which differ only in which
// ComplexTypeContext arm that is. Every other argument is common to all five,
// which is why the entry points differ only in this one dispatch.
//
// The two ELEMENT-contexted arms build the identical component and are not
// folded into one case: they take their {context} from DIFFERENT fields — the
// element-owned arm's sole token, and the Type-Alternative-owned arm's owner
// rather than its container — and joining them would hide which field each reads
// behind a shared name (§3.4.2.1 dcl.ctd.common contexts both in the enclosing
// element declaration).
//
// base is the {base type definition} SLOT as resolveBaseType built it, so the
// redefining arm receives the InlineTypeDefinition holding src-expredef clause
// 1.1's original and every other arm a name (or nil). A redefining type whose
// base= is NOT its own name has no original to own; that is exactly what
// src-redefine clause 5 forbids and checkRedefinedComplexType charges before
// production begins, so the mismatch is unreachable and is reported as a plain
// producer fault rather than a fabricated rule violation.
//
// The ORIGINAL arm takes the slot BOTH ways, and is the one arm that does. It
// owns an original of its own exactly when the declaration it is built from is
// itself a redefining declaration — a chained <redefine>, where clause 1.1's
// "as defined in Schema Component Details (§3)" carries clause 1.2 with it — and
// carries a plain by-name base otherwise. Which it is, is the slot's own shape
// and is read from it (STYLE D3): resolveBaseType already made that decision,
// from the source element, when it chose the arm.
//
// The switch is exhaustive over the sealed sum; see topLevelComplexTypeName for
// why the default arm is unreachable.
func (p *producer) newComplexType(id complexTypeIdentity, loc xsderr.Loc, base xsd.TypeDefinitionOrRef, final []xsd.DerivationMethod, derivationMethod xsd.DerivationMethod, abstract bool, attributeUses []xsd.AttributeUse, prohibitedAttributeNames []xsd.QName, attributeWildcard *xsd.Wildcard, contentType xsd.ContentType, prohibitedSubstitutions []xsd.DerivationMethod, assertions []xsd.Assertion, annotations []xsd.Annotation) (xsd.ComplexType, error) {
	switch i := id.(type) {
	case namedComplexType:
		return xsd.NewComplexType(loc, i.name, baseTypeName(base), final,
			derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
	case redefiningComplexType:
		original, owns := ownedComplexBase(base)
		if !owns {
			return xsd.ComplexType{}, fmt.Errorf("parser: the redefining <complexType> %s at %s did not resolve its own name as its {base type definition}, so src-expredef clause 1.2 has no clause-1.1 original to pair it with; src-redefine clause 5 requires that self-derivation and checkRedefinedComplexType charges it before production", i.name, loc)
		}
		return xsd.NewComplexTypeOwningBase(loc, i.owner, i.name, original, final,
			derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
	case elementOwnedComplexType:
		return xsd.NewAnonymousComplexType(loc, xsd.ElementDeclarationContext{Component: i.owner}, baseTypeName(base), final,
			derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
	case typeAlternativeOwnedComplexType:
		return xsd.NewAnonymousComplexType(loc, xsd.ElementDeclarationContext{Component: i.owner}, baseTypeName(base), final,
			derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
	case redefineOriginalComplexType:
		context := xsd.ComplexTypeDefinitionContext{Component: i.owner}
		if original, owns := ownedComplexBase(base); owns {
			return xsd.NewAnonymousComplexTypeOwningBase(loc, i.ownedOriginal, context, original, final,
				derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
		}
		return xsd.NewAnonymousComplexType(loc, context, baseTypeName(base), final,
			derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
	default:
		panic("parser: newComplexType: non-exhaustive complexTypeIdentity switch")
	}
}

// baseTypeName reads back the QName the three by-name xsd constructors take from
// a {base type definition} slot: the TypeDefinitionRef arm's name, and the zero
// QName (·absent·) for a nil slot. An InlineTypeDefinition never reaches those
// constructors — only the redefining arm builds one, and it takes the component
// — so it is reported as absent here rather than silently losing a name it has
// none of.
func baseTypeName(base xsd.TypeDefinitionOrRef) xsd.QName {
	if ref, ok := base.(xsd.TypeDefinitionRef); ok {
		return ref.Name
	}
	return xsd.QName{}
}

// ownedComplexBase unwraps the {base type definition} slot resolveBaseType builds
// for src-expredef clause 1.1: the InlineTypeDefinition owning the anonymous
// original, which for a <complexType> redefinition is always a Complex Type
// Definition (it is produced from the redefined document's own top-level
// <complexType>). ok is false for every other slot shape.
func ownedComplexBase(base xsd.TypeDefinitionOrRef) (xsd.ComplexType, bool) {
	inline, ok := base.(xsd.InlineTypeDefinition)
	if !ok {
		return xsd.ComplexType{}, false
	}
	ct, ok := inline.Definition.(xsd.ComplexType)
	return ct, ok
}

// produceComplexType maps a <complexType> element (§3.4.2) into a Complex Type
// Definition, in all four source forms: implicit complex content (§3.4.2.3.2,
// restriction from xs:anyType), explicit <complexContent> with <restriction> or
// with <extension> (§3.4.2.3.3 clauses 4.1 and 4.2), and <simpleContent> with
// <extension> (§3.4.2.2 cases 3-5). Every form that names a base= needs the
// {base type definition} COMPONENT — the two extension forms for their
// content-type tableaux, and the <complexContent> <restriction> form for
// §3.4.2.1 clause 1's {assertions} fold — and buildComplexType/resolveBaseType
// supply it by building it on demand (§3.4.2's preamble: the mapping rules
// "depend upon the {base type definition} having been identified before they
// apply"). The implicit-content form names no base=; its base is xs:anyType,
// always already seeded.
//
// <simpleContent> with <restriction> is the one form still declined: §3.4.2.2
// cases 1-2 SYNTHESIZE a new anonymous simple type restricting the base's, from
// the <restriction>'s own facet children, which this producer does not yet build.
// It is declined with a plain "not yet produced" error rather than a fabricated
// rule violation (mirroring Produce's non-schema-root precedent: no src-*/cos-*
// rule governs "this representation is not yet produced"). The conformance
// schema lane (conformance/schema.go) declines that shape, so the decline never
// reaches a validity verdict.
//
// id is BOTH what the built type is constructed from — a {name} for a top-level
// one, a {context} for an inline anonymous one — and the {scope}.{parent} that
// every local element declaration nested in this type's content model (§3.3.2.3
// dcl.elt.local) and every local attribute declaration among its own attribute
// content (§3.2.2.2 dcl.att.local) reports. It is threaded down as an explicit
// xsd.ElementScopeParent / xsd.AttributeScopeParent parameter rather than stashed
// on the producer, so nesting can never mis-attribute a declaration. The
// attribute half stops at the <attributeGroup ref> hop, which is a scope boundary
// and rebinds the parent to the referenced group — see collectReferencedGroup.
// The two sums are distinct types (§3.2.1 sc_a's alternation is CTD | AGD where
// §3.3.1 sc_e's is CTD | MGD), so complexTypeIdentity emits one of each.
//
// A missing name on the NAMED arm is rejected FIRST, before any content is
// built, with the same plain grammar fault topLevelName raises and for the same
// reason (produce.go: name is use="required" typed xs:NCName on
// xs:topLevelComplexType in the schema for schema documents, and §3.4.3 src-ct
// states no clause of its own for it, incorporating the condition only by
// reference). Since #305 that fault is unreachable from run's document-order
// dispatch, which now takes every top-level name from topLevelName, and since
// #343 it is unreachable from a schema DOCUMENT at all: the other entry path
// into buildComplexType — resolveBaseType's on-demand build, whose name comes
// from a base= lexical resolved against the prescan index — can no longer carry
// an empty local part, because bindQName now rejects one as a lexical
// cvc-datatype-valid fault at the base attribute itself. What the guard covers
// is therefore a direct in-package call alone, and it is kept as the backstop
// for one: deleting it would make that path's verdict depend on whether the
// content happens to hold a local element, whose xsd.NewLocalScope would charge
// e-props-correct, an unrelated rule, and only sometimes — the #206 defect.
// The ANONYMOUS arm cannot reach the rejection at all: it carries no name to be
// missing, and its own equivalent — an unminted owner — is unconstructible,
// since produceElement/produceLocalElement mint the identity before calling.
//
// GAP(xsd): an ANONYMOUS complex type built here enters no
// xsd.SchemaBuilder.AddType (§3.17.2 scopes {type definitions} to the
// <complexType> children OF <schema>, and registering it would fork the
// component in two — see xsd/typedefinition.go's InlineTypeDefinition). Finalize
// Phase A reaches it through the owning declaration, so src-resolve is charged
// normally, but every finalize pass that quantifies over the Schema's type
// definitions never visits it and produces NO verdict for it:
// checkContentModelsUnambiguous (xsd/particleattribution.go, cos-nonambig),
// checkElementDeclarationsConsistent (xsd/elementconsistent.go,
// cos-element-consistent), and checkComplexDerivations
// (xsd/complexderivation.go) with the checkAttributeUseNamesUnique
// (ct-props-correct clause 4) and checkExtensionAttributeUses
// (xsd/complexextension.go, cos-ct-extends clause 1.2) it drives — #438. The two
// FINALIZE-side folds (xsd/attributeusefold.go's foldAttributeUses,
// xsd/attributewildcardfold.go's foldAttributeWildcards) miss it as well —
// #414, and #438 depends on #414 because widening a read-only walk ahead of the
// folds would turn an unfolded anonymous extension into a FALSE rejection.
// §3.4.2.1 clause 1's {assertions} fold is NOT among them and needs no issue of
// its own: assertionsWithBase runs HERE, on every produced type, anonymous ones
// included (#346). The direction today is open (under-rejection), never
// fail-closed, and conformance/schema.go's anonymousComplexTypeDecidable narrows
// the conformance lane to the implicit-content shape, on which the two folds are
// provably the identity (base xs:anyType, §3.4.7 empty uses and no wildcard to
// union).
func (p *producer) produceComplexType(id complexTypeIdentity, el *Element) (xsd.ComplexType, error) {
	if name, named := topLevelComplexTypeName(id); named && name.Local == "" {
		return xsd.ComplexType{}, fmt.Errorf("parser: top-level <complexType> at %s has no usable name: its name attribute is absent or empty, and the schema for schema documents requires an xs:NCName", el.Loc())
	}
	if oc := misplacedOpenContent(el); oc != nil {
		return xsd.ComplexType{}, fmt.Errorf("parser: <openContent> at %s is in a position the schema for schema documents does not allow: it is a child of <complexType> only in the implicit-content form (no <simpleContent>/<complexContent>), under <complexContent> only of the <restriction>/<extension> alternant, and nowhere at all under <simpleContent>", oc.Loc())
	}
	if sc := childElement(el, xsd.XMLSchemaNS, "simpleContent"); sc != nil {
		return p.produceSimpleContent(id, el, sc)
	}
	if cc := childElement(el, xsd.XMLSchemaNS, "complexContent"); cc != nil {
		return p.produceComplexContent(id, el, cc)
	}
	return p.produceImplicitContent(id, el)
}

// produceImplicitContent maps a <complexType> with neither <simpleContent> nor
// <complexContent> (§3.4.2.3.2): the {base type definition} is xs:anyType and
// {derivation method} is restriction. The explicit content, attribute uses, and
// attribute wildcard come directly from the <complexType>'s own children, and
// this type is the {scope}.{parent} of every local element and every local
// attribute among them.
func (p *producer) produceImplicitContent(id complexTypeIdentity, el *Element) (xsd.ComplexType, error) {
	mixed, _ := boolAttr(el, "mixed")
	abstract, _ := boolAttr(el, "abstract")
	content, err := p.buildComplexContentType(el, mixed, scopeParentOf(id))
	if err != nil {
		return xsd.ComplexType{}, err
	}
	uses, prohibited, wildcard, err := p.produceAttributeUses(el, attributeScopeParentOf(id))
	if err != nil {
		return xsd.ComplexType{}, err
	}
	// {assertions} (§3.4.2.1 clause 2) come from the <assert> children of the
	// <complexType> itself in this implicit-content form. Clause 1's fold of the
	// base's own {assertions} is not applied through assertionsWithBase here, and
	// does not need to be: the base is unconditionally xs:anyType, whose
	// {assertions} is the empty sequence (§3.4.7, seedAnyType), so the fold is
	// PROVABLY the identity — and it is the seeded xs:anyType that any lookup
	// would find, since builtComplex holds it before any document is produced and
	// a name already in that memo is never rebuilt.
	return p.newComplexType(id, el.Loc(), xsd.TypeDefinitionRef{Name: anyTypeName}, p.complexTypeFinal(el),
		xsd.DerivationRestriction, abstract, uses, prohibited, wildcard, content, p.complexTypeProhibitedSubstitutions(el), p.assertionsOf(el), nil)
}

// produceSimpleContent maps a <complexType><simpleContent> (§3.4.2.2) into a
// Complex Type Definition whose {content type} has {variety} simple, {particle}
// and {open content} ·absent·, and {simple type definition} computed by the
// five-case tableau keyed on the resolved {base type definition} and on which
// derivation alternant is chosen.
//
// Only <extension> is produced — tableau cases 3, 4 and 5, all of which REUSE an
// already-built simple type (see simpleContentSimpleType). <restriction> (cases
// 1-2) synthesizes a NEW anonymous simple type restricting the base's with the
// facet children of <restriction>, which this producer does not build yet, so it
// is declined as a limitation, not charged a rule.
//
// A <simpleContent> with NEITHER alternant is a plain grammar fault rather than
// that limitation, and is reported as one: §3.4.2.2 states outright that "either
// <restriction> or <extension> must appear in the content of <simpleContent>",
// and that requirement lives in the schema for schema documents, which src-ct
// incorporates by reference without stating a clause of its own — the same
// footing as the <complexContent> half (produceComplexContent). Reporting it as
// the <restriction> limitation would name an element the source never carried.
//
// It enforces src-ct clause 1 (§3.4.3, simple-content-rules): with the
// <simpleContent> alternative chosen, the <complexType> must not have
// mixed="true". That is a Schema Representation Constraint on the source XML, so
// it is charged here at the <complexType>'s own position — and it is charged
// BEFORE the <restriction> decline, so a document that violates it gets the rule
// verdict rather than a limitation error.
func (p *producer) produceSimpleContent(id complexTypeIdentity, ctElem, sc *Element) (xsd.ComplexType, error) {
	if mixed, present := boolAttr(ctElem, "mixed"); present && mixed {
		return xsd.ComplexType{}, xsderr.New(ruleSrcCT, ctElem.Loc(),
			"<complexType> has mixed=\"true\" and a <simpleContent> child, but src-ct clause 1 forbids mixed=true when the <simpleContent> alternative is chosen")
	}
	restriction := childElement(sc, xsd.XMLSchemaNS, "restriction")
	ext := childElement(sc, xsd.XMLSchemaNS, "extension")
	if restriction == nil && ext == nil {
		return xsd.ComplexType{}, fmt.Errorf("parser: <simpleContent> at %s has neither a <restriction> nor an <extension> child, one of which §3.4.2.2 requires", sc.Loc())
	}
	if ext == nil {
		return xsd.ComplexType{}, fmt.Errorf("parser: <simpleContent> with <restriction> is not yet produced (§3.4.2.2 cases 1-2 synthesize a new anonymous simple type from the <restriction>'s facet children)")
	}
	baseLex, hasBase := ext.Attr("base")
	if !hasBase {
		// Appendix A declares base use="required" on xs:extensionType, and §3.4.3
		// states no Schema Representation Constraint for its absence, so a missing
		// base is a plain grammar fault carrying no rule — the same reading
		// produceGroupRefParticle gives a <group> with no ref. It is charged here
		// rather than handed to resolveQName as an empty lexical, which would report
		// an absent attribute as a base="" the author never wrote.
		return xsd.ComplexType{}, fmt.Errorf("parser: the <extension> of a <simpleContent> at %s must carry a base attribute, but none is present", ext.Loc())
	}
	baseName, err := p.resolveQName(ext, baseLex, "base")
	if err != nil {
		return xsd.ComplexType{}, err
	}
	base, baseRef, err := p.resolveBaseType(id, ext, baseName)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	abstract, _ := boolAttr(ctElem, "abstract")
	uses, prohibited, wildcard, err := p.produceAttributeUses(ext, attributeScopeParentOf(id))
	if err != nil {
		return xsd.ComplexType{}, err
	}
	content := xsd.SimpleContent{SimpleType: simpleContentSimpleType(base, p.symbols.built[anySimpleTypeName])}
	// {assertions} (§3.4.2.1): clause 1's members of the resolved base's own
	// {assertions} — nothing at all when it is a simple type, the common case
	// here — ahead of clause 2's <assert> children of <extension>.
	return p.newComplexType(id, ctElem.Loc(), baseRef, p.complexTypeFinal(ctElem),
		xsd.DerivationExtension, abstract, uses, prohibited, wildcard, content, p.complexTypeProhibitedSubstitutions(ctElem), assertionsWithBase(base, p.assertionsOf(ext)), nil)
}

// simpleContentSimpleType is the §3.4.2.2 {simple type definition} tableau for
// the <extension> alternant:
//
//   - case 3: the base is a complex type whose own {content type} has {variety}
//     simple — reuse THAT content type's {simple type definition};
//   - case 4: the base is a simple type definition — reuse it;
//   - case 5 (c-ctsc-bad): otherwise xs:anySimpleType.
//
// Every arm returns an EXISTING *xsd.SimpleType pointer; nothing is rebuilt, so
// simple-type component identity is preserved (xsd/typedefinition.go). Case 5
// deliberately MAPS rather than rejects: the tableau names a result for the
// base-is-a-complex-type-with-non-simple-content case, and its invalidity is
// cos-ct-extends' (§3.4.6.2) to charge, not the mapping's.
//
// anySimpleType is case 5's fallback, read from the seeded builtins by the
// caller; a nil one (an unseeded backend) is rejected downstream by
// xsd.NewComplexType as an absent Required {simple type definition}.
func simpleContentSimpleType(base xsd.TypeDefinition, anySimpleType *xsd.SimpleType) *xsd.SimpleType {
	switch b := base.(type) {
	case *xsd.SimpleType:
		return b // case 4
	case xsd.ComplexType:
		switch bc := b.ContentType().(type) {
		case xsd.SimpleContent:
			return bc.SimpleType // case 3
		case xsd.EmptyContent, xsd.ElementContent:
			return anySimpleType // case 5
		default:
			panic("parser: simpleContentSimpleType: non-exhaustive ContentType switch")
		}
	default:
		panic("parser: simpleContentSimpleType: non-exhaustive TypeDefinition switch")
	}
}

// produceComplexContent maps a <complexType><complexContent> (§3.4.2.3), in both
// derivation alternants: <restriction>, whose {content type} is purely structural
// (§3.4.2.3.3 clause 4.1), and <extension>, whose {content type} merges the
// resolved base's particle with this derivation's ·effective content· (clause
// 4.2). It enforces src-ct clause 5 (§3.4.3): when mixed is present on both
// <complexType> and <complexContent>, the two actual values must agree.
//
// A <complexContent> with neither alternant is a plain grammar fault, not a rule
// verdict: §3.4.2.3 states outright that "either <restriction> or <extension>
// must appear in the content of <complexContent>", and that requirement lives in
// the schema for schema documents, which src-ct incorporates by reference without
// stating a clause of its own (the same footing as a nameless top-level
// <complexType>).
func (p *producer) produceComplexContent(id complexTypeIdentity, ctElem, cc *Element) (xsd.ComplexType, error) {
	derivation, method := complexContentDerivation(cc)
	if derivation == nil {
		return xsd.ComplexType{}, fmt.Errorf("parser: <complexContent> at %s has neither a <restriction> nor an <extension> child, one of which §3.4.2.3 requires", cc.Loc())
	}
	ctMixed, ctHasMixed := boolAttr(ctElem, "mixed")
	ccMixed, ccHasMixed := boolAttr(cc, "mixed")
	if ctHasMixed && ccHasMixed && ctMixed != ccMixed {
		return xsd.ComplexType{}, xsderr.New(ruleSrcCT, cc.Loc(),
			"mixed is present on both <complexType> and <complexContent> with differing values, but src-ct clause 5 requires them to be the same")
	}
	// {effective mixed} (§3.4.2.3.3 clause 1): <complexContent>'s mixed if present,
	// else <complexType>'s, else false.
	mixed := ctMixed
	if ccHasMixed {
		mixed = ccMixed
	}
	abstract, _ := boolAttr(ctElem, "abstract")
	baseLex, hasBase := derivation.Attr("base")
	if !hasBase {
		// base is use="required" on both xs:extensionType and xs:complexRestrictionType
		// (Appendix A) and §3.4.3 states no clause for its absence, so this is a plain
		// grammar fault of the same kind as the missing alternant above — and charging
		// it here keeps an absent attribute from being reported as a base="" the
		// author never wrote.
		return xsd.ComplexType{}, fmt.Errorf("parser: the <%s> of a <complexContent> at %s must carry a base attribute, but none is present", derivation.Name().Local(), derivation.Loc())
	}
	baseName, err := p.resolveQName(derivation, baseLex, "base")
	if err != nil {
		return xsd.ComplexType{}, err
	}
	// The base COMPONENT is resolved here, for BOTH alternants, because §3.4.2.1
	// clause 1's {assertions} fold reads it on both — where §3.4.2.3.3 clause 4's
	// content-type merge needs it on the extension alternant only. Resolving it on
	// the restriction alternant too moves nothing but the moment: every named base
	// is built by run at its own document-order position anyway, and a base that is
	// on the build stack or resolves to nothing is charged the same ct-props-correct
	// clause 3 / src-resolve clause 1.1 verdict resolveBaseType always charges.
	base, baseRef, err := p.resolveBaseType(id, derivation, baseName)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	content, err := p.complexContentType(derivation, method, base, mixed, scopeParentOf(id))
	if err != nil {
		return xsd.ComplexType{}, err
	}
	uses, prohibited, wildcard, err := p.produceAttributeUses(derivation, attributeScopeParentOf(id))
	if err != nil {
		return xsd.ComplexType{}, err
	}
	// {assertions} (§3.4.2.1): clause 1's members of the resolved base's own
	// {assertions}, then clause 2's <assert> children of the derivation alternant
	// — not of the enclosing <complexType> — in this explicit complex-content form.
	return p.newComplexType(id, ctElem.Loc(), baseRef, p.complexTypeFinal(ctElem),
		method, abstract, uses, prohibited, wildcard, content, p.complexTypeProhibitedSubstitutions(ctElem), assertionsWithBase(base, p.assertionsOf(derivation)), nil)
}

// complexContentDerivation returns the <restriction> or <extension> child of a
// <complexContent> together with the {derivation method} it maps to (§3.4.2.3),
// or (nil, 0) when neither is present. <restriction> is looked for first, so a
// malformed source carrying both is mapped by the same alternant every run
// (STYLE D1).
func complexContentDerivation(cc *Element) (*Element, xsd.DerivationMethod) {
	if r := childElement(cc, xsd.XMLSchemaNS, "restriction"); r != nil {
		return r, xsd.DerivationRestriction
	}
	if e := childElement(cc, xsd.XMLSchemaNS, "extension"); e != nil {
		return e, xsd.DerivationExtension
	}
	return nil, 0
}

// complexContentType computes the ·explicit content type· of a complex content
// (§3.4.2.3.3 clause 4) from the derivation alternant's children: clause 4.1 for
// a restriction, clause 4.2 for an extension. derivation is the <restriction> or
// <extension> element, base the COMPONENT its base= names (already resolved by
// the caller, which needs it for the §3.4.2.1 clause 1 {assertions} fold on both
// alternants; only the extension branch reads it here), effectiveMixed is clause
// 1's result, and scopeParent is the enclosing Complex Type Definition every
// local element declaration in this content model is scoped to (§3.3.2.3
// dcl.elt.local).
//
// Clause 6's ·wildcard element· wrap is applied on top of that result by
// openContentType, in both branches: the derivation alternant is exactly the
// element whose <openContent> child clause 5.1 reads.
func (p *producer) complexContentType(derivation *Element, method xsd.DerivationMethod, base xsd.TypeDefinition, effectiveMixed bool, scopeParent xsd.ElementScopeParent) (xsd.ContentType, error) {
	if method == xsd.DerivationRestriction {
		return p.buildComplexContentType(derivation, effectiveMixed, scopeParent)
	}
	effective, explicitEmpty, err := p.effectiveContent(derivation, effectiveMixed, scopeParent)
	if err != nil {
		return nil, err
	}
	explicit, err := xsd.ExtensionContentType(derivation.Loc(), base, effective, explicitEmpty, effectiveMixed, p.resolveModelGroup)
	if err != nil {
		return nil, err
	}
	return p.openContentType(derivation, explicit)
}

// buildComplexContentType computes the {content type} of a restriction-derived
// (or implicit) complex content from parent's model-group child (§3.4.2.3.3
// clauses 2-4, restriction case). parent is the <complexType> (implicit) or the
// <restriction> (explicit complex content); effectiveMixed is clause 1's result;
// scopeParent is the enclosing Complex Type Definition every local element
// declaration in this content model is scoped to (§3.3.2.3 dcl.elt.local).
//
// Clauses 5-6 then fold in the ·wildcard element· (openContentType): parent is
// also the element whose <openContent> child clause 5.1 reads.
func (p *producer) buildComplexContentType(parent *Element, effectiveMixed bool, scopeParent xsd.ElementScopeParent) (xsd.ContentType, error) {
	effective, _, err := p.effectiveContent(parent, effectiveMixed, scopeParent)
	if err != nil {
		return nil, err
	}
	return p.openContentType(parent, explicitContentType(effective, effectiveMixed))
}

// explicitContentType is §3.4.2.3.3 clause 4.1 — the restriction branch's
// ·explicit content type· — as a total function of the already-computed
// ·effective content·: clause 4.1.1 (empty effective content ⇒ {variety} empty,
// which admits NO character content at all, unlike element-only) and clause 4.1.2
// (otherwise mixed iff ·effective mixed·). Clause 4.2.1 says the extension cases
// with a simple or empty/simple-content base yield "a Content Type as per clause
// 4.1.1 and clause 4.1.2 above", i.e. exactly this.
//
// It is stated a SECOND time, in package xsd, and that split was ruled at #392's
// warden pre-flight rather than overlooked. Clause 4.2 moved down to xsd because
// cos-ct-extends clause 1.5 needs it at finalize (xsd.ExtensionContentType), and
// it carries clause 4.1 with it for its own 4.2.1 arm; exporting a SECOND name
// for a two-line total function, whose one out-of-package caller would be this
// restriction branch, was more surface than the sharing is worth (STYLE T5). The
// two encodings are pinned against each other by this package's content-type
// tests, which run both arms over the same source shapes: a change to clause
// 4.1.1 or 4.1.2 is a change to both sites.
func explicitContentType(effective *xsd.Particle, effectiveMixed bool) xsd.ContentType {
	if effective == nil {
		return xsd.EmptyContent{} // clause 4.1.1
	}
	return xsd.ElementContent{Mixed: effectiveMixed, Particle: *effective} // clause 4.1.2
}

// effectiveContent computes the ·effective content· of a complex content
// (§3.4.2.3.3 clauses 2-3) from parent's model-group child, and reports whether
// the ·explicit content· (clause 2) was ***empty***. parent is the <complexType>
// (implicit content), <restriction> or <extension>.
//
// A nil particle means the ·effective content· itself is ***empty*** (clause
// 3.1.2). The two facts are distinct and both are needed: clause 4.2.2 keys on
// the EFFECTIVE content being empty, while clause 4.2.3.1 keys on the EXPLICIT
// content being empty — they differ exactly when clause 3.1.1 substitutes an
// empty 1..1 sequence for a mixed type with no model-group child, so that text is
// admitted.
//
// This is the single encoding of clauses 2-3: both the restriction branch
// (buildComplexContentType) and the extension branch (complexContentType) read
// their ·effective content· from here.
func (p *producer) effectiveContent(parent *Element, effectiveMixed bool, scopeParent xsd.ElementScopeParent) (effective *xsd.Particle, explicitEmpty bool, err error) {
	explicit, err := p.explicitContent(modelGroupChild(parent), scopeParent)
	if err != nil {
		return nil, false, err
	}
	if explicit != nil {
		return explicit, false, nil // clause 3.2
	}
	if !effectiveMixed {
		return nil, true, nil // clause 3.1.2: ·effective content· is ***empty***
	}
	// clause 3.1.1: a 1..1 particle over an empty sequence stands in, so a mixed
	// type with no model-group child still admits character content.
	part, err := emptySequenceParticle(parent.Loc())
	if err != nil {
		return nil, false, err
	}
	return &part, true, nil
}

// misplacedOpenContent returns the <openContent> element a <complexType> carries
// in a position the schema for schema documents (§3.4.2) does not allow, or nil.
// The type's own <openContent> is legal only in the IMPLICIT content form —
// xs:complexType's content model puts it in the third alternative, beside the
// model group, never beside <simpleContent>/<complexContent> — under
// <complexContent> only as a child of the <restriction>/<extension> alternant,
// since xs:complexContent's content model is (annotation?, (restriction |
// extension)), and NOWHERE under <simpleContent>: that element's content model is
// the same (annotation?, (restriction | extension)) shape (spec L1687), but
// neither of ITS alternants admits an <openContent> (L1692/L1697 — the
// simple-content <restriction> takes facets and attribute declarations, and its
// one wildcard slot is ##other, which the XSD namespace is not).
//
// All three misplacements were previously invisible: the {content type} branches
// read <openContent> from the <complexType> or the complex-content derivation
// alternant only, so a stray one elsewhere was silently dropped and a schema a
// complete processor rejects would have assembled clean. Like <complexContent>
// with neither alternant (produceComplexContent), this is a plain grammar fault,
// not an xsderr rule verdict: src-ct (§3.4.3) states no clause for it and
// incorporates the schema for schema documents' own conditions by reference.
func misplacedOpenContent(ctElem *Element) *Element {
	cc := childElement(ctElem, xsd.XMLSchemaNS, "complexContent")
	sc := childElement(ctElem, xsd.XMLSchemaNS, "simpleContent")
	if cc == nil && sc == nil {
		return nil // the implicit form: an <openContent> child here is in its legal position
	}
	if own := childElement(ctElem, xsd.XMLSchemaNS, "openContent"); own != nil {
		return own
	}
	if cc != nil {
		return childElement(cc, xsd.XMLSchemaNS, "openContent")
	}
	return simpleContentOpenContent(sc)
}

// simpleContentOpenContent returns the <openContent> a <simpleContent> carries
// either directly or under its <restriction>/<extension> alternant, or nil — every
// position inside a <simpleContent> subtree is illegal (see misplacedOpenContent).
// The alternants are searched restriction-first, matching
// complexContentDerivation's precedent, so a malformed source carrying both is
// reported at the same position every run (STYLE D1).
func simpleContentOpenContent(sc *Element) *Element {
	if oc := childElement(sc, xsd.XMLSchemaNS, "openContent"); oc != nil {
		return oc
	}
	for _, alternant := range [...]string{"restriction", "extension"} {
		alt := childElement(sc, xsd.XMLSchemaNS, alternant)
		if alt == nil {
			continue
		}
		if oc := childElement(alt, xsd.XMLSchemaNS, "openContent"); oc != nil {
			return oc
		}
	}
	return nil
}

// openContentType applies §3.4.2.3.3 (dcl.ctd.ctcc.common) clauses 5 and 6 to an
// already-computed ·explicit content type·, yielding the complex type's final
// {content type}. owner is the element whose <openContent> child clause 5.1
// looks for: the <complexType> itself (implicit content) or the <restriction>/
// <extension> alternant. It is the single encoding of the Open Content fold, and
// every {content type} branch ends in it.
//
// src-ct (§3.4.3) clauses 3-4 are charged FIRST, before clause 5 selects
// anything: they are Schema Representation Constraints on the source
// <openContent> element, so they hold whether or not the ·wildcard element·
// turns out to matter — in particular for a mode="none" element, which clause
// 6.1 otherwise discards unexamined.
func (p *producer) openContentType(owner *Element, explicit xsd.ContentType) (xsd.ContentType, error) {
	if err := checkOpenContentAny(owner); err != nil {
		return nil, err
	}
	we := p.wildcardElement(owner, explicit)
	if we == nil {
		return explicit, nil // clause 6.1: the ·wildcard element· is ·absent·
	}
	oc, err := p.openContentOf(we, explicit)
	if err != nil {
		return nil, err
	}
	if oc == nil {
		return explicit, nil // clause 6.1: the ·wildcard element· has mode="none"
	}
	return wrapOpenContent(we.Loc(), explicit, *oc)
}

// checkOpenContentAny enforces src-ct (§3.4.3) clauses 3 and 4 on owner's own
// <openContent> child: an <any> is required when mode is not "none" (clause 3,
// since clause 6.2 has no {wildcard} to build without it) and forbidden when it
// is (clause 4, since clause 6.1 would ignore it). The schema-level
// <defaultOpenContent> is NOT governed by these clauses — src-ct is a constraint
// on <complexType> — so its own mandatory <any> is checked as a grammar fault by
// checkDefaultOpenContent instead, eagerly, before anything is produced.
func checkOpenContentAny(owner *Element) error {
	oc := childElement(owner, xsd.XMLSchemaNS, "openContent")
	if oc == nil {
		return nil
	}
	none := openContentModeIsNone(oc)
	hasAny := childElement(oc, xsd.XMLSchemaNS, "any") != nil
	if none && hasAny {
		return xsderr.New(ruleSrcCT, oc.Loc(),
			`<openContent mode="none"> has an <any> child, but src-ct clause 4 forbids one`)
	}
	if !none && !hasAny {
		return xsderr.New(ruleSrcCT, oc.Loc(),
			`<openContent> whose mode is not "none" has no <any> child, but src-ct clause 3 requires one`)
	}
	return nil
}

// wildcardElement selects §3.4.2.3.3 clause 5's ·wildcard element·, returning nil
// for clause 5.3 (·absent·): owner's own <openContent> child when present (clause
// 5.1 — presence alone decides, so an <openContent mode="none"> is how a type
// opts OUT of the document's default), otherwise this document's
// <defaultOpenContent> when the ·explicit content type· is not empty (5.2.1) or
// is empty and that element carries appliesToEmpty="true" (5.2.2).
//
// appliesToEmpty is read here and nowhere else: §3.4.1's Open Content record has
// no such property, so it must not travel past this selection (STYLE D3).
func (p *producer) wildcardElement(owner *Element, explicit xsd.ContentType) *Element {
	if own := childElement(owner, xsd.XMLSchemaNS, "openContent"); own != nil {
		return own // clause 5.1
	}
	def := p.defaultOpenContentElem()
	if def == nil {
		return nil // clause 5.3: this document declares no default
	}
	if explicit.Variety() != xsd.ContentEmpty {
		return def // clause 5.2.1
	}
	if appliesToEmpty, _ := boolAttr(def, "appliesToEmpty"); appliesToEmpty {
		return def // clause 5.2.2
	}
	return nil // clause 5.3
}

// defaultOpenContentElem returns the <defaultOpenContent> child of THIS
// document's <schema> (§3.4.2.3.3 clause 5.2's "the <schema> ancestor"), or nil
// when it declares none. The <schema> is read off p.schemaElem, never by walking
// Element.Parent() from the complex type: under ·override pre-processing· a
// substituted declaration is a child of the OVERRIDING document's <override>,
// so an ancestor walk would climb into that document and read ITS default,
// whereas §4.2.5 makes the substituted declaration a top-level declaration of
// the OVERRIDDEN document and produced by its producer (override.go, PRINCIPLES
// 16) — the same reading targetNamespace and the *FormDefault attributes already
// take. The two coincide for every non-override document, which is exactly why
// the wrong one is invisible without an override fixture.
//
// The <schema> content model admits at most one <defaultOpenContent>; a
// malformed document with several is mapped by its first, so the verdict is the
// same every run (STYLE D1). This is a PURE LOOKUP with no rejection path of its
// own: checkDefaultOpenContent has already judged the element's grammar for
// every document of the assembly by the time any complex type consults it, which
// is what keeps that verdict independent of whether any type reaches clause 5.2.
func (p *producer) defaultOpenContentElem() *Element {
	return childElement(p.schemaElem, xsd.XMLSchemaNS, "defaultOpenContent")
}

// checkDefaultOpenContent enforces the two schema for schema documents grammar
// rules on this document's <defaultOpenContent>: its content model (annotation?,
// any) makes the <any> child mandatory, and its mode attribute is restricted to
// (interleave | suffix) — "none" is an <openContent> mode alone.
//
// Both are plain grammar faults rather than xsderr rule verdicts, because no
// Schema Component Constraint governs <defaultOpenContent> at all: it is pure
// schema-document-level source grammar, consumed by §3.4.2.3.3
// (dcl.ctd.ctcc.common) clause 5.2's mapping, while ct-props-correct (§3.4.6.1)
// clause 1 is scoped to a Complex Type Definition's §3.4.1 property tableau —
// and no such component is in view here — and src-ct (§3.4.3) clauses 3-4 are
// constraints on <complexType>. The mode enumeration is therefore NOT routed
// through openContentModeOf, whose out-of-enumeration charge is ct-props-correct
// and whose job is to yield a {mode} value this check has no use for.
//
// Running it once per document, ahead of production, is what makes the verdict
// content-INDEPENDENT: a malformed <defaultOpenContent> is rejected because it
// is malformed, not because some complex type of the same document happened to
// have an ·explicit content type· that reaches clause 5.2 and consults it.
//
// It is called beside the pre-scan, for every document of the assembly, rather
// than from run: a base= in the document produced FIRST builds its base type
// through the DECLARING document's producer on demand (symbols.typeSource), so a
// per-run check would let a neighbour document's still-unjudged default be
// selected by clause 5.2 first — and a childless one would then reach
// openContentOf's panic instead of any verdict at all.
func (p *producer) checkDefaultOpenContent() error {
	def := p.defaultOpenContentElem()
	if def == nil {
		return nil
	}
	if childElement(def, xsd.XMLSchemaNS, "any") == nil {
		return fmt.Errorf("parser: <defaultOpenContent> at %s has no <any> child, but the schema for schema documents makes it mandatory", def.Loc())
	}
	mode, present := def.Attr("mode")
	if !present {
		return nil
	}
	switch strings.TrimSpace(mode) {
	case "interleave", "suffix":
		return nil
	}
	return fmt.Errorf(`parser: <defaultOpenContent> at %s has mode=%q, but the schema for schema documents admits only interleave or suffix there ("none" is an <openContent> mode)`, def.Loc(), mode)
}

// openContentOf computes §3.4.2.3.3 clause 6's {open content} from the
// ·wildcard element· we, returning nil for clause 6.1 — a mode="none" element
// contributes NO Open Content record, which is why xsd.OpenContentMode has no
// third member (xsd/closedsets.go) and why the "none" token dies here rather
// than travelling as a mode.
//
// {mode} is the mode attribute's ·actual value·, defaulting to interleave, and
// {wildcard} is openContentWildcard's clause-6.2 combination.
func (p *producer) openContentOf(we *Element, explicit xsd.ContentType) (*xsd.OpenContent, error) {
	if openContentModeIsNone(we) {
		return nil, nil // clause 6.1
	}
	mode, err := openContentModeOf(we)
	if err != nil {
		return nil, err
	}
	anyElem := childElement(we, xsd.XMLSchemaNS, "any")
	if anyElem == nil {
		// Unreachable: a mode≠"none" <openContent> without an <any> was rejected by
		// checkOpenContentAny (src-ct clause 3) and a <defaultOpenContent> without
		// one by checkDefaultOpenContent before ANY document of the assembly was
		// produced, and those are the only two elements clause 5 can select.
		// Panicking names the broken invariant rather than fabricating a verdict for
		// a source shape that cannot reach here.
		panic("parser: openContentOf: ·wildcard element· with mode other than none has no <any> child")
	}
	w, err := p.produceWildcard(anyElem)
	if err != nil {
		return nil, err
	}
	wildcard, err := openContentWildcard(we.Loc(), w, explicit)
	if err != nil {
		return nil, err
	}
	oc, err := xsd.NewOpenContent(we.Loc(), mode, wildcard)
	if err != nil {
		return nil, err
	}
	return &oc, nil
}

// openContentWildcard computes clause 6.2's {wildcard}: the wildcard W
// corresponding to the ·wildcard element·'s <any> child when the ·explicit
// content type· carries no {open content}, and otherwise a wildcard whose
// {process contents} and {annotations} are W's and whose {namespace constraint}
// is the §3.10.6.3 wildcard union (xsd.UnionNamespaceConstraint) of W's with
// that of ·explicit content type·.{open content}.{wildcard}.
//
// The second arm is live for an <extension> whose base already has an Open
// Content: §3.4.2.3.3 clause 4.2.2/4.2.3 hand the base's {open content} through
// into the ·explicit content type· (xsd.ExtensionContentType), and this derivation's
// own <openContent> then widens it rather than replacing it.
func openContentWildcard(loc xsderr.Loc, w xsd.Wildcard, explicit xsd.ContentType) (xsd.Wildcard, error) {
	ec, ok := explicit.(xsd.ElementContent)
	if !ok || ec.OpenContent == nil {
		return w, nil
	}
	unioned, err := xsd.UnionNamespaceConstraint(loc, w.NamespaceConstraint(), ec.OpenContent.Wildcard().NamespaceConstraint())
	if err != nil {
		return xsd.Wildcard{}, err
	}
	return xsd.NewWildcard(loc, unioned, w.ProcessContents(), w.Annotations())
}

// wrapOpenContent folds oc into the ·explicit content type· per §3.4.2.3.3
// clause 6.2's {variety}/{particle} half: an element-only or mixed explicit
// content keeps its {variety} and {particle} and merely gains the {open
// content}, while an ***empty*** one becomes ELEMENT-ONLY over the clause's
// synthesized empty-sequence particle, so the Open Content has a content model
// to interleave with or suffix. That element-only is unconditional — clause 6.2
// names it outright, with no reading of ·effective mixed· — because the empty
// ·explicit content type· that reaches here already answered the mixed question
// by being empty (clause 4.1.1).
func wrapOpenContent(loc xsderr.Loc, explicit xsd.ContentType, oc xsd.OpenContent) (xsd.ContentType, error) {
	switch e := explicit.(type) {
	case xsd.EmptyContent:
		particle, err := emptySequenceParticle(loc)
		if err != nil {
			return nil, err
		}
		return xsd.ElementContent{Mixed: false, Particle: particle, OpenContent: &oc}, nil
	case xsd.ElementContent:
		e.OpenContent = &oc
		return e, nil
	default:
		// xsd.SimpleContent, the only other member, is unreachable twice over:
		// {open content} is a property of ElementContent alone
		// (xsd/complextype.go), and no §3.4.2.3.3 branch feeding this function can
		// yield simple content — the <simpleContent> forms never reach it.
		// Panicking on the broken sealed sum matches xsd.ExtensionContentType;
		// charging a rule the type system already makes impossible would be a
		// fabricated verdict.
		panic("parser: wrapOpenContent: non-exhaustive ContentType switch")
	}
}

// openContentModeIsNone reports whether an <openContent>/<defaultOpenContent>
// element carries mode="none" — the ·actual value· of an xs:NMTOKEN-derived
// enumeration, so surrounding whitespace is collapsed away before the compare.
func openContentModeIsNone(we *Element) bool {
	return strings.TrimSpace(attrOr(we, "mode")) == "none"
}

// openContentModeOf maps a ·wildcard element·'s mode attribute to the {mode} of
// clause 6.2's Open Content: interleave when absent (the schema for schema
// documents' default) or written out, suffix when written out. "none" never
// reaches here — openContentOf returns clause 6.1 before calling — so an
// out-of-enumeration token is what is left, charged to ct-props-correct clause 1
// (§3.4.6.1), the §3.4.1 tableau the {mode} property belongs to, exactly as
// xsd.NewOpenContent charges a mode it cannot represent.
//
// That charge is reachable for a type's OWN <openContent> alone, which is where
// a Complex Type Definition is in view to charge it against. The other
// ·wildcard element· clause 5 can select, the schema-level <defaultOpenContent>,
// still passes through here — but only ever on the interleave/suffix arms, its
// whole enumeration having been settled by checkDefaultOpenContent as a grammar
// fault, since ct-props-correct reaches no component there.
func openContentModeOf(we *Element) (xsd.OpenContentMode, error) {
	mode, ok := we.Attr("mode")
	if !ok {
		return xsd.OpenContentInterleave, nil
	}
	switch strings.TrimSpace(mode) {
	case "interleave":
		return xsd.OpenContentInterleave, nil
	case "suffix":
		return xsd.OpenContentSuffix, nil
	}
	return 0, xsderr.New(ruleCTPropsCorr, we.Loc(),
		"open content mode %q is not one of none/interleave/suffix (ct-props-correct clause 1)", mode)
}

// emptySequenceParticle builds the 1..1 particle over an empty sequence model
// group that §3.4.2.3.3 names twice in identical words: clause 3.1.1's ·effective
// content· for a mixed type with no model-group child, and clause 6.2's
// {particle} when an ***empty*** ·explicit content type· has to carry an Open
// Content. One encoding for both (STYLE T4).
func emptySequenceParticle(loc xsderr.Loc) (xsd.Particle, error) {
	seq, err := xsd.NewModelGroup(loc, xsd.CompositorSequence, nil, nil)
	if err != nil {
		return xsd.Particle{}, err
	}
	oneOne, err := xsd.NewOccurs(loc, 1, 1)
	if err != nil {
		return xsd.Particle{}, err
	}
	return xsd.NewParticle(loc, oneOne, xsd.ResolvedTerm{Term: seq}, nil)
}

// explicitContent maps the model-group child to the {explicit content} particle
// (§3.4.2.3.3 clause 2), returning nil for the ***empty*** cases: no group child
// (2.1.1), an empty <all>/<sequence> (2.1.2), a childless <choice minOccurs="0">
// (2.1.3), or a group child with maxOccurs="0" (2.1.4).
//
// An <all> child has its own occurrence grammar checked FIRST (allOccursGrammar),
// ahead of every elision test: 2.1.2/2.1.4 decide what the group maps TO, while
// the {0,1} enumeration decides whether the <all> element is well-formed at all,
// so an elided <all maxOccurs="0" minOccurs="2"> is still rejected. A <group>
// child's prohibited name (rejectProhibitedRefAttrs) is checked first for the
// same reason: 2.1.4 elides a maxOccurs="0" child without ever entering
// produceGroupRefParticle, so the charge made there would never answer.
//
// scopeParent is passed through to every local element declaration built beneath
// this content model (§3.3.2.3 dcl.elt.local). It is generic rather than a
// ComplexTypeScopeParent because this function and the ones it calls are shared
// with the named-<group> tree, where the same elements are scoped to a Model
// Group Definition instead.
func (p *producer) explicitContent(group *Element, scopeParent xsd.ElementScopeParent) (*xsd.Particle, error) {
	if group == nil {
		return nil, nil // 2.1.1
	}
	local := group.Name().Local()
	if local == "all" {
		// Before any clause-2 elision: an <all> whose occurrence attributes are
		// outside the {0,1} enumeration is invalid however it maps.
		if err := allOccursGrammar(group); err != nil {
			return nil, err
		}
	}
	if local == "group" {
		// Before any clause-2 elision, on the same footing: a <group> reference
		// carrying the name xs:groupRef prohibits is invalid however it maps, and
		// 2.1.4 below returns without ever entering produceGroupRefParticle, where
		// this same charge answers for the groupParticles path.
		if err := rejectProhibitedRefAttrs(group); err != nil {
			return nil, err
		}
	}
	hasChildren := hasParticleChild(group)
	if (local == "all" || local == "sequence") && !hasChildren {
		return nil, nil // 2.1.2
	}
	if local == "choice" && !hasChildren && minOccursZero(group) {
		return nil, nil // 2.1.3
	}
	if maxOccursZero(group) {
		return nil, nil // 2.1.4
	}
	if local == "group" {
		return p.produceGroupRefParticle(group) // 2.2, <group ref> content-model child
	}
	return p.produceGroupParticle(group, scopeParent) // 2.2
}

// produceGroupParticle maps an <all>/<choice>/<sequence> element to a Particle
// wrapping a Model Group (§3.8.2), with {particles} in document order. A
// minOccurs=maxOccurs=0 group maps to no component at all (§3.8.2) —
// produceGroupParticle returns (nil, nil) — so the caller omits it. The
// grammar's own {0,1} occurrence restriction on <all> is a separate concern,
// enforced by allOccursGrammar in explicitContent (the sole path by which an
// <all> legally reaches here), not repeated in this function.
//
// WHERE the resulting model group may appear is not decided here at all:
// cos-all-limited (§3.8.6.2) is a Schema Component Constraint over
// {term}/{particles}, and §3.7.2 xr.mgd3 makes a <group ref> particle's {term}
// the referenced definition's {model group}, so a usage site's compositor is
// invisible until references resolve. xsd's finalize pass charges it
// (xsd/allgrouplimited.go, #469).
//
// scopeParent passes straight through: a model group is not a scope boundary
// (§3.3.2.3 names only <complexType> and the named <group> as ancestors that
// determine {scope}.{parent}).
func (p *producer) produceGroupParticle(group *Element, scopeParent xsd.ElementScopeParent) (*xsd.Particle, error) {
	local := group.Name().Local()
	compositor, ok := compositorOf(local)
	if !ok {
		// A <group> reference is mapped by produceGroupRefParticle before it
		// reaches here; any other name is an unexpected model-group child.
		return nil, fmt.Errorf("parser: model group child <%s> is not a compositor (all/choice/sequence)", local)
	}
	occ, elided, err := occursOf(group)
	if err != nil {
		return nil, err
	}
	if elided {
		return nil, nil
	}
	particles, err := p.groupParticles(group, scopeParent)
	if err != nil {
		return nil, err
	}
	mg, err := xsd.NewModelGroup(group.Loc(), compositor, particles, nil)
	if err != nil {
		return nil, err
	}
	part, err := xsd.NewParticle(group.Loc(), occ, xsd.ResolvedTerm{Term: mg}, nil)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

// produceGroupRefParticle maps a <group ref> to a Particle whose {term} is a
// deferred ModelGroupRef (§3.7.2, xr.mgd3), mirroring produceElementParticle's
// <element ref> → ElementDeclarationRef mapping. A minOccurs=maxOccurs=0 <group
// ref> maps to no component at all (returns nil, §3.7.2). The reference is
// RETAINED, never rewritten to the {term} it denotes: resolution and the
// no-circular-groups check happen at finalize (#173: src-resolve clause 1.5,
// mg-props-correct clause 2), and neither VERDICT is ever duplicated here.
// Occurs-range correctness (p-props-correct §3.9.6.1 clause 2.1) is enforced
// inside xsd.NewParticle.
//
// One mapping rule nonetheless has to LOOK through a reference produced here:
// §3.4.2.3.3 clause 4.2.3 selects a sub-case by the {compositor} of the
// ·base particle·'s {term}, and that choice fixes a {content type} synthesized at
// produce time. xsd.ExtensionContentType reads it through the resolveModelGroup
// callback this producer passes it — a read, which leaves this particle untouched
// and charges nothing.
func (p *producer) produceGroupRefParticle(el *Element) (*xsd.Particle, error) {
	if err := rejectProhibitedRefAttrs(el); err != nil {
		return nil, err
	}
	occ, elided, err := occursOf(el)
	if err != nil {
		return nil, err
	}
	if elided {
		return nil, nil
	}
	ref, ok := el.Attr("ref")
	if !ok {
		// A <group> in a content model is always a reference (§3.8.2: the named
		// definition form appears only as a top-level <schema> child). The absence
		// has no dedicated SCC (§3.7.3 "None as such", Q6), so it is a plain
		// well-formedness fault, not an xsderr rule — the same footing as the
		// prohibited name rejectProhibitedRefAttrs charges just above.
		return nil, fmt.Errorf("parser: a <group> in a content model must be a reference (carry a ref attribute), but none is present")
	}
	qn, err := p.resolveQName(el, ref, "ref")
	if err != nil {
		return nil, err
	}
	if src, redefining := p.redefinedGroupOriginal(el, qn); redefining {
		// src-expredef clause 2: this is a redefining <group>'s licensed
		// self-reference, so "a component which corresponds to the top-level
		// definition item of that name and the appropriate kind in S2 is used" —
		// the ORIGINAL, which the visible name no longer reaches. The reference
		// cannot stay deferred, since resolving it by name at finalize would land on
		// the redefinition and read as a circular <group ref> graph
		// (mg-props-correct clause 2). It is resolved HERE instead, to exactly what
		// §3.7.2 xr.mgd3 says a <group ref> particle's {term} is — "the {model
		// group} of the model group definition ·resolved· to by the ·actual value·
		// of the ref attribute" — built under the REDEFINED document's producer so
		// its local declarations keep their own document's namespace and defaults.
		mg, err := src.owner.buildDefinitionModelGroup(src.elem, xsd.ModelGroupScopeParent{Name: qn})
		if err != nil {
			return nil, err
		}
		part, err := xsd.NewParticle(el.Loc(), occ, xsd.ResolvedTerm{Term: mg}, nil)
		if err != nil {
			return nil, err
		}
		return &part, nil
	}
	part, err := xsd.NewParticle(el.Loc(), occ, xsd.ModelGroupRef{Name: qn}, nil)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

// produceModelGroupDefinition maps a top-level named <group> (§3.7.2, xr.mgd1)
// into a Model Group Definition. It is reached only through
// buildModelGroupDefinition, which memoizes it so one <group> is mapped exactly
// once however many demand-driven lookups reach it. The named form has exactly
// one <all>/<choice>/<sequence> child, whose Model Group becomes {model group};
// an absent child yields the zero ModelGroup that NewModelGroupDefinition rejects
// (mgd-props-correct §3.7.6, {model group} Required). Occurrence on the child is
// irrelevant here — a model group definition carries no {min occurs}/{max occurs}
// (§3.7.2 note); those live solely on a <group ref> particle.
//
// name is also the {scope}.{parent} of every local element declaration in the
// body: §3.3.2.3 dcl.elt.local's "otherwise" branch — an <element> within a
// named <group> rather than under a <complexType> — scopes it to the Model Group
// Definition, not to whatever complex type later references the group.
//
// A missing name is rejected FIRST, before the body is built, for the reason
// produceComplexType gives: the schema for schema documents makes name
// use="required" with type xs:NCName on xs:namedGroup, and §3.7.3 states "None
// as such" for <group>'s Schema Representation Constraints, so the fault carries
// no rule ID — while deferring it would let a nameless <group> be judged by
// whether its body happens to hold a local element. Since #305 run reaches this
// only through topLevelName, which raises the identical fault a step earlier;
// what remains guarded here is the <redefine> path — produceRedefinition
// (redefine.go) mints the redefining declaration's name from the <redefine>
// child's own name attribute and never consults topLevelName, so a
// <group name=""> child of a <redefine> whose redefined document declares the
// same nameless <group> (recorded as the original, hence past src-expredef's
// closing requirement) arrives here with an empty local part — plus direct
// programmatic calls. resolveModelGroup's on-demand build does NOT reach it: its
// only caller is xsd.ExtensionContentType's group-lookup callback, reading a
// ModelGroupRef that only produceGroupRefParticle mints, and xsd.NewParticle
// rejects an empty ModelGroupRef local part as a component-invariant before any
// resolution happens — a backstop produceGroupRefParticle no longer reaches
// since #343, whose bindQName rejects the <group ref=""> lexical at the
// attribute (cvc-datatype-valid) before a ModelGroupRef is minted at all.
func (p *producer) produceModelGroupDefinition(name xsd.QName, el *Element) (xsd.ModelGroupDefinition, error) {
	if name.Local == "" {
		return xsd.ModelGroupDefinition{}, fmt.Errorf("parser: top-level <group> at %s has no usable name: its name attribute is absent or empty, and the schema for schema documents requires an xs:NCName", el.Loc())
	}
	mg, err := p.buildDefinitionModelGroup(el, xsd.ModelGroupScopeParent{Name: name})
	if err != nil {
		return xsd.ModelGroupDefinition{}, err
	}
	return xsd.NewModelGroupDefinition(el.Loc(), name, mg, nil)
}

// buildDefinitionModelGroup builds the {model group} of a top-level <group>
// definition from its single <all>/<choice>/<sequence> child (§3.7.2, xr.mgd1):
// the Model Group term itself, not a Particle (a definition carries no
// occurrence). An absent compositor child returns the zero ModelGroup, deferring
// the {model group}-Required rejection to NewModelGroupDefinition
// (mgd-props-correct). cos-all-limited is not charged here: an <all> definition
// body is legal outright (§3.8.6.2 clause 1.1), and the constraint's usage-site
// half is a component-shape verdict finalize owns (xsd/allgrouplimited.go).
// scopeParent is the enclosing definition, threaded to the local element
// declarations in the body (§3.3.2.3 dcl.elt.local).
func (p *producer) buildDefinitionModelGroup(el *Element, scopeParent xsd.ElementScopeParent) (xsd.ModelGroup, error) {
	group := compositorChild(el)
	if group == nil {
		return xsd.ModelGroup{}, nil // absent → NewModelGroupDefinition rejects
	}
	compositor, _ := compositorOf(group.Name().Local()) // compositorChild guarantees ok
	particles, err := p.groupParticles(group, scopeParent)
	if err != nil {
		return xsd.ModelGroup{}, err
	}
	return xsd.NewModelGroup(group.Loc(), compositor, particles, nil)
}

// compositorChild returns el's first <all>/<choice>/<sequence> child (a model
// group definition's body, §3.7.2), or nil. Unlike modelGroupChild it excludes
// <group>: a top-level <group> definition's body is never a nested reference.
func compositorChild(el *Element) *Element {
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch c.Name().Local() {
		case "all", "choice", "sequence":
			return c
		}
	}
	return nil
}

// groupParticles maps the particle children of a model group in document order,
// omitting each minOccurs=maxOccurs=0 child (which maps to no component, §3.9.2).
// scopeParent is the Complex Type Definition or Model Group Definition the whole
// content tree hangs under, threaded to each local element declaration.
//
// An <all> child of an <all> is refused here, and only that one. Appendix A's
// xs:allModel — the group xs:complexType "all" restricts its content to —
// admits just <element>, <any> and <group>, but the component that spelling maps
// to, an all group among another all group's {particles} at min=max=1, is
// exactly what cos-all-limited clause 1.3 permits: no Schema Component
// Constraint can name the fault, so the grammar production is the only text that
// forbids it. §3.8.3 states no Schema Representation Constraint for <all>
// either, so the fault carries no rule ID, the same footing as
// produceGroupRefParticle's missing-ref fault.
//
// A <sequence> or <choice> child is deliberately NOT refused here: it maps to
// the shape cos-all-limited clause 2 forbids, and xsd/allgrouplimited.go charges
// it at finalize with that rule ID and a positioned Loc.
func (p *producer) groupParticles(group *Element, scopeParent xsd.ElementScopeParent) ([]xsd.Particle, error) {
	var particles []xsd.Particle
	inAll := group.Name().Local() == "all"
	for _, child := range group.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		var part *xsd.Particle
		var err error
		switch el.Name().Local() {
		case "annotation":
			continue
		case "element":
			part, err = p.produceElementParticle(el, scopeParent)
		case "any":
			part, err = p.produceAnyParticle(el)
		case "sequence", "choice", "all":
			if inAll && el.Name().Local() == "all" {
				return nil, fmt.Errorf("parser: <all> at %s is a child of an <all>, whose content Appendix A's xs:allModel restricts to <element>, <any> and <group>", el.Loc())
			}
			part, err = p.produceGroupParticle(el, scopeParent)
		case "group":
			part, err = p.produceGroupRefParticle(el)
		default:
			return nil, fmt.Errorf("parser: unexpected model group child <%s>", el.Name().Local())
		}
		if err != nil {
			return nil, err
		}
		if part != nil {
			particles = append(particles, *part)
		}
	}
	return particles, nil
}

// produceElementParticle maps a local <element> to a Particle (§3.3.2.3). A
// minOccurs=maxOccurs=0 element maps to no component at all (returns nil). An
// <element ref="..."> yields a deferred ElementDeclarationRef term (resolved at
// finalize, #173); otherwise a sibling local Element Declaration is built inline,
// scoped to scopeParent (§3.3.2.3 dcl.elt.local). The ref form takes no
// scopeParent: it denotes a top-level declaration, whose own {scope} is global
// (§3.3.2.4 ref.elt.global maps only the Particle, never a declaration).
func (p *producer) produceElementParticle(el *Element, scopeParent xsd.ElementScopeParent) (*xsd.Particle, error) {
	occ, elided, err := occursOf(el)
	if err != nil {
		return nil, err
	}
	if elided {
		return nil, nil
	}
	// GAP(xsd): an <element ref="..." substitutionGroup="..."> is silently ACCEPTED,
	// the attribute simply ignored. The meta-schema prohibits substitutionGroup on
	// xs:localElement whichever form it takes (§3.3.2, use="prohibited"), but this
	// branch returns before produceLocalElement — the one place the producer charges
	// e-props-correct clause 3 on the attribute's presence — ever runs. Nothing else
	// reads it here: substitutionGroupAffiliations is called only from
	// produceElement (the global path), and an ElementDeclarationRef term has no
	// {substitution group affiliations} slot to populate, so no component property
	// is affected and no downstream rule sees a different value. The whole loss is
	// one unmade syntax rejection — an under-reject, not a false-accept of a
	// validity conclusion. No W3C suite case has this shape. Unowned: no issue
	// tracks it yet (STYLE P3 requires an issue reference only when an issue does
	// own the retirement).
	if ref, hasRef := el.Attr("ref"); hasRef {
		qn, err := p.resolveQName(el, ref, "ref")
		if err != nil {
			return nil, err
		}
		part, err := xsd.NewParticle(el.Loc(), occ, xsd.ElementDeclarationRef{Name: qn}, nil)
		if err != nil {
			return nil, err
		}
		return &part, nil
	}
	decl, err := p.produceLocalElement(el, scopeParent)
	if err != nil {
		return nil, err
	}
	part, err := xsd.NewParticle(el.Loc(), occ, xsd.ResolvedTerm{Term: decl}, nil)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

// produceLocalElement maps a local inline <element name="..."> to a local
// Element Declaration (§3.3.2.3, dcl.elt.local, {scope} = local), including its
// {identity-constraint definitions}: §3.3.2.1's Common Mapping Rules apply
// uniformly to global and local declarations, so <key>/<keyref>/<unique> children
// are mapped here exactly as in produceElement. Registering them with the schema
// builder is the caller's job (produceElementParticle).
//
// Its {type definition} is §3.3.2.1 dcl.elt.common's tier chain, mapped by
// localDeclaredType. Tier 1's inline <complexType> child needs an identity
// minted before either component exists: the anonymous complex type it maps to
// is the {scope}.{parent} of its OWN nested local element declarations, and
// having no name it is identified by an xsd.ComponentID instead
// (xsd.AnonymousComplexTypeScopeParent, #340). The remaining tiers (the inline
// <simpleType> form, the type= form, the xs:anyType default) are mapped by
// declaredType, shared with the local-attribute chain.
//
// The declaration's identity is therefore minted UNCONDITIONALLY, before any
// tier is chosen, and every path returns through
// xsd.NewElementDeclarationOwningTypes: since #851 an <alternative> child may own
// an anonymous complex type of its own (§3.12.2 declare-ta), so a declaration
// with a plain type= owns types too and there is no tier-free path left. That one
// mint is the {context} (§3.4.2.1 dcl.ctd.common) of every type this declaration
// owns, and NewElementDeclarationOwningTypes checks each of them agrees; the
// per-edge CONTAINER tokens the nested scopes report are minted separately, one
// per <alternative> (see typeAlternativeOwnedComplexType).
//
// src-element clause 3 (§3.3.3) is charged here for the both-present case, on the
// same footing produceElement charges it for a global <element>: without it,
// type= would silently win over an inline child. It covers the inline
// <complexType> arm as much as the <simpleType> one — the clause names both.
//
// An <element> carrying BOTH an inline <simpleType> and an inline <complexType>
// is rejected outright, with a plain grammar-fault error rather than a rule
// verdict: the schema for schema documents gives xs:localElement a single
// (simpleType | complexType)? slot, and no src-element clause states the
// condition (clause 3 is about a type child TOGETHER WITH type=, not about two
// type children). Letting one arm win silently would map a schema no processor
// accepts.
//
// e-props-correct clause 3 (§3.3.6.1) is charged here too, FIRST, and on the
// attribute rather than on the built component: a substitutionGroup= on a local
// <element> is prohibited by the schema for schema documents (use="prohibited"
// on xs:localElement, §3.3.2), and this producer runs no meta-schema validation
// pass ahead of mapping, so nothing else would see it. Reaching
// xsd.NewElementDeclaration's identical clause-3 check instead is not an option —
// the mapping would have to build a non-empty {substitution group affiliations}
// for a local declaration to trip it, which is exactly the state the clause
// forbids — and simply not reading the attribute here would silently ACCEPT the
// prohibited schema. It precedes the src-element test because it depends on
// nothing but the attribute's presence.
//
// The reach is the INLINE local form only, which is the only form that arrives
// here: produceElementParticle's <element ref> branch returns before this
// function runs, so a ref= local element carrying the prohibited attribute is
// accepted with the attribute ignored (GAP marker on that branch).
//
// {disallowed substitutions} comes from the same disallowedSubstitutions mapping
// the global path uses (STYLE T4): §3.3.2.1's row is a COMMON rule and the
// meta-schema leaves block= permitted on xs:localElement, unlike the three
// attributes it prohibits there (substitutionGroup, final, abstract).
//
// scopeParent is the nearest <complexType> or named <group> ancestor's component,
// supplied by the caller (never recomputed from the element here — the ancestor
// axis is not walkable from an *Element). It is a required parameter, and every
// path through this function builds the scope from it.
func (p *producer) produceLocalElement(el *Element, scopeParent xsd.ElementScopeParent) (xsd.ElementDeclaration, error) {
	if _, ok := el.Attr("substitutionGroup"); ok {
		return xsd.ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, el.Loc(),
			"a local <element> carries a substitutionGroup attribute, but e-props-correct clause 3 confines a non-empty {substitution group affiliations} to a global {scope}.{variety} — the schema for schema documents accordingly declares the attribute use=\"prohibited\" on xs:localElement (§3.3.2)")
	}
	_, hasType := el.Attr("type")
	inlineSimple := childElement(el, xsd.XMLSchemaNS, "simpleType")
	inlineComplex := childElement(el, xsd.XMLSchemaNS, "complexType")
	if hasType && (inlineSimple != nil || inlineComplex != nil) {
		return xsd.ElementDeclaration{}, xsderr.New(ruleSrcElement, el.Loc(),
			"element has both a type attribute and an inline <simpleType>/<complexType> child, but src-element clause 3 forbids both")
	}
	if err := rejectBothInlineTypes(el, inlineSimple, inlineComplex); err != nil {
		return xsd.ElementDeclaration{}, err
	}
	qname, err := declarationName(el, p.localTargetNS(el, "elementFormDefault"))
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	vc, err := valueConstraintOf(el, ruleSrcElement)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	nillable, _ := boolAttr(el, "nillable")
	constraints, err := p.identityConstraintsOf(el)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	scope, err := xsd.NewLocalScope(el.Loc(), scopeParent)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	edID := xsd.NewComponentID()
	typeDef, err := p.localDeclaredType(el, edID, inlineComplex)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	typeTable, err := p.typeTableOf(el, edID, typeDef)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	return xsd.NewElementDeclarationOwningTypes(el.Loc(), edID, qname, typeDef, typeTable, scope, vc,
		nillable, constraints, nil, nil, false, p.disallowedSubstitutions(el), nil)
}

// localDeclaredType maps §3.3.2.1 dcl.elt.common's tier chain for a LOCAL
// <element>, splitting off tier 1's inline <complexType> — which needs the
// declaration's minted identity, already in the caller's hand — and delegating
// every other tier to declaredType. inlineComplex is that child, or nil.
func (p *producer) localDeclaredType(el *Element, edID xsd.ComponentID, inlineComplex *Element) (xsd.TypeDefinitionOrRef, error) {
	if inlineComplex == nil {
		return p.declaredType(el, anyTypeName)
	}
	ct, err := p.produceComplexType(elementOwnedComplexType{owner: edID}, inlineComplex)
	if err != nil {
		return nil, err
	}
	return xsd.InlineTypeDefinition{Definition: ct}, nil
}

// rejectBothInlineTypes rejects an <element> carrying BOTH an inline
// <simpleType> and an inline <complexType> child. The schema for schema
// documents gives xs:element a single (simpleType | complexType)? slot, and
// src-element (§3.3.3) states no clause for two type children — clause 3 is
// about a type child together with type= — so this is a plain grammar fault like
// a nameless top-level <complexType>, not an xsderr rule verdict (STYLE E2:
// never fabricate a rule ID for a condition the spec states elsewhere).
//
// It is shared by the global and the local element path (STYLE T4): §3.3.2.1's
// tier 1 is a COMMON mapping rule and the meta-schema restriction is the same on
// xs:topLevelElement and xs:localElement, so one implementation serves both.
func rejectBothInlineTypes(el *Element, inlineSimple, inlineComplex *Element) error {
	if inlineSimple == nil || inlineComplex == nil {
		return nil
	}
	return fmt.Errorf("parser: <element> at %s has both an inline <simpleType> and an inline <complexType> child, but the schema for schema documents allows at most one type child on an <element>", el.Loc())
}

// declaredType maps the {type definition} of an <element> or <attribute> whose
// both-present and inline-<complexType> cases the caller has already excluded.
// It is the ONE implementation of the parallel mapping chains — §3.3.2.1
// dcl.elt.common for an element, §3.2.2.1 dcl.att.global and §3.2.2.2
// dcl.att.local for an attribute — which agree on every tier this producer
// implements (STYLE T4):
//
//	tier 1  the anonymous type corresponding to the inline <simpleType> child;
//	tier 2  the type definition the type= attribute ·resolves· to;
//	last    dflt, the caller's fallback — xs:anyType for an element (§3.3.2.1
//	        clause 4), xs:anySimpleType for an attribute (§3.2.2.1/§3.2.2.2).
//
// It serves both GLOBAL paths as well as the local ones (#442, #733): the tier
// chain is COMMON to both element forms — §3.3.2.2 dcl.elt.global supplements
// {target namespace} and {scope} alone, never {type definition} — and the two
// attribute forms state one chain each in identical words, so neither a
// top-level <element> nor a top-level <attribute> has anything to map
// differently here, and produceElement and produceAttribute call this rather
// than restating it.
//
// The anonymous type is built once, here, and handed to the declaration as an
// xsd.InlineTypeDefinition: it goes into no symbol table, so the declaration is
// its sole owner. Its {context} (§3.16.1 std-context) is a SEPARATE property
// from the {type definition} this maps and stays unpopulated on every anonymous
// simple type the producer builds, global and local alike (#206).
//
// The element chain's clause 3 — "The declared {type definition} of the Element
// Declaration ·resolved· to by the FIRST QName in the ·actual value· of the
// substitutionGroup attribute, if present" — has no tier here, and its absence
// is not a gap: no caller can reach it through this function. substitutionGroup=
// is legal only on a top-level <element> (use="prohibited" on xs:localElement,
// §3.3.2), produceLocalElement charges e-props-correct clause 3 for a local one,
// and the attribute chain has no clause-3 analog at all. The global caller
// decides it itself, BETWEEN tier 2 and dflt, through substitutionGroupHeadType
// (#395, which needed it because e-props-correct clause 4 reads the resulting
// type) — which is also why this function's dflt is never the answer on that
// path. The two mappings deliberately read the list differently: {substitution
// group affiliations} resolves EVERY item, clause 3 the first one only.
func (p *producer) declaredType(el *Element, dflt xsd.QName) (xsd.TypeDefinitionOrRef, error) {
	if inline := childElement(el, xsd.XMLSchemaNS, "simpleType"); inline != nil {
		st, err := p.constructSimpleType(xsd.QName{}, inline) // tier 1
		if err != nil {
			return nil, err
		}
		return xsd.InlineTypeDefinition{Definition: st}, nil
	}
	typeLex, hasType := el.Attr("type")
	if !hasType {
		return xsd.TypeDefinitionRef{Name: dflt}, nil
	}
	qn, err := p.resolveQName(el, typeLex, "type") // tier 2
	if err != nil {
		return nil, err
	}
	return xsd.TypeDefinitionRef{Name: qn}, nil
}

// produceAnyParticle maps an <any> to a Particle whose {term} is a Wildcard
// (§3.10.2.1). A minOccurs=maxOccurs=0 <any> maps to no component (returns nil).
func (p *producer) produceAnyParticle(el *Element) (*xsd.Particle, error) {
	occ, elided, err := occursOf(el)
	if err != nil {
		return nil, err
	}
	if elided {
		return nil, nil
	}
	wildcard, err := p.produceWildcard(el)
	if err != nil {
		return nil, err
	}
	part, err := xsd.NewParticle(el.Loc(), occ, xsd.ResolvedTerm{Term: wildcard}, nil)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

// produceAttributeUses maps the attribute-bearing children of parent (a
// <complexType>, <restriction>, or <extension>) into {attribute uses}, the
// expanded names §3.4.2.4 clause 3.2.2 blocks, and an optional {attribute
// wildcard}, following <attributeGroup ref> children transitively
// (§3.6.2.1/§3.6.2.2, the inline case). {attribute uses} is the union of parent's
// own <attribute> uses with the uses of every referenced attribute group
// (§3.6.2.1); {attribute wildcard} is the intersection of parent's own
// <anyAttribute> with the referenced groups' wildcards (§3.6.2.2, always
// intersection at one container).
//
// The BASE type's contribution to {attribute uses} is deliberately NOT folded in
// here: §3.4.2.4 clause 3 needs the resolved base component, which no producer
// holds, so xsd/attributeusefold.go completes the property at finalize (#401).
// What this function returns is therefore clauses 1 and 2 alone, and the
// component xsd.NewComplexType is handed carries exactly that until finalize
// overwrites it. The clause 3.2.2 names ride along for the same reason: the fold
// that consumes them runs there, and by then the source is gone.
//
// GAP(xsd): the base's {attribute wildcard} is not folded in either — §3.4.2.5
// clause 2.2's cos-aw-union for an extension — and unlike the uses, nothing
// completes it at finalize. An extension's {attribute wildcard} is therefore its
// own <anyAttribute> alone, which is NOT merely lenient: a name the base's
// wildcard admits reads as inadmissible on the extension, and
// xsd/defaultbinding.go's caller charges that absence as
// derivation-ok-restriction — a false reject, not a fail-open. See
// attributeDefaultBinding for the exact shape; closing it is #265 section 3's
// job, not this producer's.
//
// scopeParent is the {scope}.{parent} (§3.2.1 sc_a) of parent's OWN <attribute>
// children — the enclosing complex type, supplied by the caller as an explicit
// parameter rather than stashed on the producer, so nesting can never
// mis-attribute a declaration. It does NOT reach the attributes of a referenced
// <attributeGroup>: those are scoped to the group, and collectReferencedGroup
// rebinds the parent at that hop (§3.2.2.2 dcl.att.local).
func (p *producer) produceAttributeUses(parent *Element, scopeParent xsd.AttributeScopeParent) ([]xsd.AttributeUse, []xsd.QName, *xsd.Wildcard, error) {
	var uses []xsd.AttributeUse
	var wildcards []xsd.Wildcard
	if err := p.collectAttributeContent(parent, scopeParent, map[xsd.QName]struct{}{}, &uses, &wildcards); err != nil {
		return nil, nil, nil, err
	}
	prohibited, err := p.prohibitedAttributeNames(parent)
	if err != nil {
		return nil, nil, nil, err
	}
	wildcard, err := combineAttributeWildcards(parent.Loc(), wildcards)
	if err != nil {
		return nil, nil, nil, err
	}
	return uses, prohibited, wildcard, nil
}

// prohibitedAttributeNames is §3.4.2.4 clause 3.2.2's input: the expanded names
// of parent's own <attribute> children carrying use="prohibited", in document
// order.
//
// It reads parent's DIRECT children only, never descending <attributeGroup ref>,
// because clause 3.2.2 is written over "an <attribute> [child]" of the
// <restriction>/<extension>/<complexType> itself — a prohibited <attribute> inside
// a referenced group is one of the "other contexts" §3.4.2.4's Note calls
// pointless, and collectAttributeContent already ignores it.
//
// The name is computed rather than read off a component: §3.4.2.4's Note makes a
// prohibited <attribute> correspond to no component at all, so produceAttributeUse
// returns before building anything and there is nothing to take a name from. The
// two forms mirror that function's split — a ref= resolves as a QName (§3.2.2.3),
// a name= takes the local declaration's {target namespace} (§3.2.2.2).
//
// No src-attribute clause is charged here: produceAttributeUse charges clauses
// 1, 2, 3 and 5 upstream, unconditionally, before the same prohibited
// <attribute> declines to map, so re-charging them would put them in two
// encodings (#358). Its clause 3 charge makes the `!hasName` branch below
// unreachable — produceAttributeUses runs collectAttributeContent, which returns
// produceAttributeUse's rejection of a neither-ref-nor-name <attribute> before
// this function walks the same children — and the branch is kept as a fallback.
// The two failures that ARE surfaced here are an unresolvable ref= prefix —
// src-resolve, charged by resolveQName for every other reference in this
// producer — and a name= that is not an xs:NCName, charged cvc-datatype-valid
// by declarationName for every other declaration name in it. A prohibited
// <attribute> earns no exemption from either: both are faults of the schema
// document, and neither needs a component to exist to be a fault.
func (p *producer) prohibitedAttributeNames(parent *Element) ([]xsd.QName, error) {
	var names []xsd.QName
	for _, child := range parent.Children() {
		el, ok := child.(*Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS || el.Name().Local() != "attribute" {
			continue
		}
		if use, _ := el.Attr("use"); use != "prohibited" {
			continue
		}
		if ref, hasRef := el.Attr("ref"); hasRef {
			qn, err := p.resolveQName(el, ref, "ref")
			if err != nil {
				return nil, err
			}
			names = append(names, qn)
			continue
		}
		if _, hasName := el.Attr("name"); !hasName {
			continue // unreachable: produceAttributeUse already rejected this shape upstream
		}
		qn, err := declarationName(el, p.localTargetNS(el, "attributeFormDefault"))
		if err != nil {
			return nil, err
		}
		names = append(names, qn)
	}
	return names, nil
}

// buildAttributeGroup maps a top-level <attributeGroup> (§3.6.2) into an
// Attribute Group Definition: its {attribute uses}/{attribute wildcard} fold in
// every referenced group transitively (§3.6.2.1/§3.6.2.2). The visited set is
// seeded with name so a reference chain that loops back to this group terminates
// — a circular <attributeGroup> reference is SPEC-LEGAL (§3.6.2.1), taking the
// transitive closure, never an error (grounding Q3). ag-props-correct (§3.6.6)
// clause 2 fires inside NewAttributeGroupDefinition only on a genuine
// duplicate-name collision among the folded uses.
//
// Every <attribute> collected from elem's own body is scoped to THIS group by
// construction (§3.2.2.2 dcl.att.local: an <attribute> with no <complexType>
// ancestor takes the Attribute Group Definition corresponding to the
// <attributeGroup> it is within), so name is passed down as an
// xsd.AttributeGroupScopeParent — the same value collectReferencedGroup rebinds
// to when a complex type reaches this group by reference instead.
func (p *producer) buildAttributeGroup(name xsd.QName, elem *Element) (xsd.AttributeGroupDefinition, error) {
	var uses []xsd.AttributeUse
	var wildcards []xsd.Wildcard
	visited := map[xsd.QName]struct{}{name: {}}
	if err := p.collectAttributeContent(elem, xsd.AttributeGroupScopeParent{Name: name}, visited, &uses, &wildcards); err != nil {
		return xsd.AttributeGroupDefinition{}, err
	}
	wildcard, err := combineAttributeWildcards(elem.Loc(), wildcards)
	if err != nil {
		return xsd.AttributeGroupDefinition{}, err
	}
	return xsd.NewAttributeGroupDefinition(elem.Loc(), name, uses, wildcard, nil)
}

// collectAttributeContent appends container's own <attribute> uses and its own
// <anyAttribute> wildcard, then descends every <attributeGroup ref> child
// transitively (§3.6.2.1), appending each reached group's uses and own wildcard.
// wildcards are collected in §3.6.2.2 pre-order — a container's own <anyAttribute>
// (L) before its referenced groups' wildcards (W, in document order) — so
// wildcards[0] is the wildcard whose {process contents} the combination takes
// (L if present, else the first of W). visited guards against the spec-legal
// circular <attributeGroup> reference chains (§3.6.2.1, Q3): an already-visited
// name is not re-descended, so a cycle contributes each element once.
//
// The receiver MUST be the producer of the document that declares container: every
// child mapped here — a local <attribute>'s {target namespace} (§3.2.2.2) and its
// type=, an <attribute ref>, an <attributeGroup ref> — is resolved against the
// receiver's schemaElem and §F.1 coercion. collectReferencedGroup switches
// producers on the way in for exactly that reason.
//
// scopeParent is the {scope}.{parent} of container's OWN <attribute> children
// only (§3.2.2.2 dcl.att.local, the component container corresponds to). It is
// deliberately not forwarded across the <attributeGroup ref> hop:
// collectReferencedGroup descends into a top-level <attributeGroup>'s own body,
// whose <attribute> children have no <complexType> ancestor and are therefore
// scoped to that group whichever container reached it, so that function rebinds
// the value rather than threading this one.
func (p *producer) collectAttributeContent(container *Element, scopeParent xsd.AttributeScopeParent, visited map[xsd.QName]struct{}, uses *[]xsd.AttributeUse, wildcards *[]xsd.Wildcard) error {
	if any := childElement(container, xsd.XMLSchemaNS, "anyAttribute"); any != nil {
		wc, err := p.produceWildcard(any)
		if err != nil {
			return err
		}
		*wildcards = append(*wildcards, wc)
	}
	for _, child := range container.Children() {
		el, ok := child.(*Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch el.Name().Local() {
		case "attribute":
			use, err := p.produceAttributeUse(el, scopeParent)
			if err != nil {
				return err
			}
			if use != nil {
				*uses = append(*uses, *use)
			}
		case "attributeGroup":
			if err := p.collectReferencedGroup(el, visited, uses, wildcards); err != nil {
				return err
			}
		}
	}
	return nil
}

// collectReferencedGroup resolves one <attributeGroup ref> child and descends
// into the referenced top-level definition, splicing in its uses and wildcards
// (§3.6.2.1). A ref whose name resolves to no top-level <attributeGroup> is a
// dangling reference charged src-resolve clause 1.4 (§3.17.6.2); a nested
// <attributeGroup> with no ref, or one carrying the name xs:attributeGroupRef
// prohibits (rejectProhibitedRefAttrs), is a well-formedness fault with no
// dedicated SCC (§3.6.3 "None as such", grounding Q6), reported as a plain error.
// An already-visited target is skipped, tolerating the spec-legal cycle (Q3).
//
// The two halves of this hop belong to DIFFERENT documents, and each is resolved
// by its own producer. The ref= attribute is held by el, a child of the ASKING
// document, so p resolves it (§F.1 task (b) transforms xs:*/@ref in the document
// that carries it). The definition it names may have been contributed by any
// document of the <include> closure (§4.2.3 c-incl-incl), so its body is descended
// under src.owner: that visibility lets a foreign producer REACH the group, it
// does not transfer resolution authority over the group's own local <attribute>
// names (§3.2.2.2) or unqualified type=/ref= values (src-resolve clause 4.1.1) —
// the same split #228 established for complexTypes and #337 for simpleTypes.
//
// Descending the source elements under their owner computes exactly §3.4.2.4
// clause c-add2's component-level union of the referenced AttributeGroupDefinition's
// {attribute uses}: every element is mapped by the producer that would have mapped
// it in the top-level component, and visited makes the transitive closure §3.6.2.1
// mandates for cycles come out the same whichever group the walk started from
// (TestAttributeGroupComponentAndInlineFoldAgree pins the two foldings agreeing).
// It is preferred over folding an already-built component because that would need a
// memo the tri-state build guard cannot supply — an attributeGroup cycle is legal,
// so "on the stack" is not an error here as it is for a base chain — and because a
// diamond (two refs reaching one group) would then splice that group's uses twice
// and trip ag-props-correct on a collision the spec's set union does not have.
//
// This hop is a {scope} BOUNDARY, and takes NO scopeParent from its caller: it
// REBINDS the parent to the resolved group. §3.2.2.2 dcl.att.local reads {parent}
// off the <attribute>'s own ancestor axis, and an <attribute> child of a
// top-level <attributeGroup> has no <complexType> ancestor at all, so its
// {parent} is that Attribute Group Definition invariantly — the same value
// however many complex types reference the group, and the same value
// buildAttributeGroup passes when it builds the group's own component. Forwarding
// the referencing complex type's parent here instead would make the two foldings
// this function's doc claims are equal differ in {scope}.{parent}
// (TestAttributeGroupComponentAndInlineFoldAgree pins that they do not).
func (p *producer) collectReferencedGroup(el *Element, visited map[xsd.QName]struct{}, uses *[]xsd.AttributeUse, wildcards *[]xsd.Wildcard) error {
	if err := rejectProhibitedRefAttrs(el); err != nil {
		return err
	}
	ref, ok := el.Attr("ref")
	if !ok {
		return fmt.Errorf("parser: a nested <attributeGroup> must be a reference (carry a ref attribute), but none is present")
	}
	qn, err := p.resolveQName(el, ref, "ref")
	if err != nil {
		return err
	}
	if src, redefining := p.redefinedAttributeGroupOriginal(el, qn); redefining {
		// src-expredef clause 2 again, for the attributeGroup half: this is a
		// redefining <attributeGroup>'s self-reference, so it splices in the
		// ORIGINAL's uses and wildcard rather than nothing. It is tested BEFORE the
		// visited set, which buildAttributeGroup seeds with the group's own name and
		// which would otherwise swallow the reference silently. The descent still
		// terminates: the original lives in the redefined document, outside any
		// <redefine>, so a self-reference in ITS body is an ordinary reference and
		// meets the visited entry below. The redefined original carries the same
		// expanded name, so it is the same {scope}.{parent} either way.
		return src.owner.collectAttributeContent(src.elem, xsd.AttributeGroupScopeParent{Name: qn}, visited, uses, wildcards)
	}
	if _, seen := visited[qn]; seen {
		return nil
	}
	visited[qn] = struct{}{}
	src, ok := p.symbols.attributeGroups[qn]
	if !ok {
		return xsderr.New(ruleSrcResolve, el.Loc(),
			"<attributeGroup ref> %s does not resolve to any top-level attribute group definition (src-resolve clause 1.4)", qn)
	}
	return src.owner.collectAttributeContent(src.elem, xsd.AttributeGroupScopeParent{Name: qn}, visited, uses, wildcards)
}

// combineAttributeWildcards folds a §3.6.2.2 pre-order sequence of collected
// wildcards into the single {attribute wildcard} (or absent, nil): an empty
// sequence is absent; otherwise the result's {namespace constraint} is the left
// fold of IntersectNamespaceConstraint over every member (§3.10.6.4
// cos-aw-intersect — combination at one container is always intersection), and
// its {process contents} comes from the first member (L if the container had its
// own <anyAttribute>, else the first referenced group's wildcard). {annotations}
// is absent, matching the producer's uniform nil-annotation mapping.
func combineAttributeWildcards(loc xsderr.Loc, wildcards []xsd.Wildcard) (*xsd.Wildcard, error) {
	if len(wildcards) == 0 {
		return nil, nil
	}
	nc := wildcards[0].NamespaceConstraint()
	for _, w := range wildcards[1:] {
		combined, err := xsd.IntersectNamespaceConstraint(loc, nc, w.NamespaceConstraint())
		if err != nil {
			return nil, err
		}
		nc = combined
	}
	result, err := xsd.NewWildcard(loc, nc, wildcards[0].ProcessContents(), nil)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// produceAttributeUse maps a local <attribute> to an Attribute Use (§3.2.2.2,
// dcl.att.local). use="prohibited" maps to no Attribute Use (returns nil). An
// <attribute ref="..."> yields a deferred AttributeDeclarationRef; otherwise a
// sibling local Attribute Declaration is built inline. It enforces the structural
// src-attribute clauses (§3.2.3): 1 (default and fixed mutually exclusive, via
// valueConstraintOf), 2 and 5 (the default/fixed × use= corner, via
// useValueConstraintOK) and 3 (exactly one of ref/name; ref excludes
// simpleType/form/type).
//
// Clauses 1, 2, 3 and 5 are ALL charged BEFORE the use="prohibited" return,
// because a Schema Representation Constraint holds of the <attribute> element
// information item itself (§5.1: "any element information items which violate
// any of the relevant Schema Representation Constraints"). Mapping to no
// component at all (§3.2.2) skips the component-building half of the mapping,
// never the validation: <attribute use="prohibited" default="d" fixed="f"/>
// violates clause 1 exactly as the optional form does, and <attribute ref="a:x"
// name="x" use="prohibited"/> violates clause 3.1 exactly as the non-prohibited
// form does. This includes the neither-ref-nor-name shape (<attribute
// use="prohibited"/>): clause 3.1 has no use= precondition, so that shape is
// REJECTED under prohibited exactly as it already was under every other use=
// value, not silently accepted as a no-op element (#358).
//
// default=/fixed= on the <attribute> element map to the USE's own {value
// constraint} (§3.5.1 vc_au) for both forms: dcl.att.local (§3.2.2.2) leaves the
// sibling local declaration's own {value constraint} ·absent· unconditionally,
// and ref.att.local (§3.2.2.3) reads them off the ref-carrying element itself,
// never touching the referenced top-level declaration's.
//
// scopeParent reaches only the sibling local declaration of the name= form: the
// ref= form (§3.2.2.3 ref.att.local) builds no declaration at all, and the
// top-level one it names carries the global {scope} its own mapping gave it.
func (p *producer) produceAttributeUse(el *Element, scopeParent xsd.AttributeScopeParent) (*xsd.AttributeUse, error) {
	use := attributeUseToken(el)
	vc, err := valueConstraintOf(el, ruleSrcAttribute)
	if err != nil {
		return nil, err
	}
	if err := useValueConstraintOK(el); err != nil {
		return nil, err
	}
	ref, hasRef := el.Attr("ref")
	_, hasName := el.Attr("name")
	if hasRef && hasName {
		return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
			"attribute has both ref and name, but src-attribute clause 3 requires exactly one")
	}
	if !hasRef && !hasName {
		return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
			"attribute has neither ref nor name, but src-attribute clause 3 requires exactly one")
	}
	if hasRef {
		if childElement(el, xsd.XMLSchemaNS, "simpleType") != nil {
			return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
				"attribute has both ref and an inline <simpleType>, but src-attribute clause 3 forbids a simpleType with ref")
		}
		if _, hasForm := el.Attr("form"); hasForm {
			return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
				"attribute has both ref and form, but src-attribute clause 3 forbids a form with ref")
		}
		if _, hasType := el.Attr("type"); hasType {
			return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
				"attribute has both ref and type, but src-attribute clause 3 forbids a type with ref")
		}
	}
	if use == "prohibited" {
		return nil, nil
	}
	required := use == "required"
	inheritable, _ := boolAttr(el, "inheritable")

	if hasRef {
		qn, err := p.resolveQName(el, ref, "ref")
		if err != nil {
			return nil, err
		}
		au, err := xsd.NewAttributeUse(el.Loc(), required, xsd.AttributeDeclarationRef{Name: qn}, vc, inheritable, nil)
		if err != nil {
			return nil, err
		}
		return &au, nil
	}
	decl, err := p.produceLocalAttribute(el, scopeParent)
	if err != nil {
		return nil, err
	}
	au, err := xsd.NewAttributeUse(el.Loc(), required, xsd.LocalAttributeDeclaration{Declaration: decl}, vc, inheritable, nil)
	if err != nil {
		return nil, err
	}
	return &au, nil
}

// attributeUseToken is the ·actual value· of an <attribute>'s use=, with an
// absent attribute read as the schema for schema documents' declared default
// "optional" — the single encoding of that default, shared by produceAttribute,
// produceAttributeUse and useValueConstraintOK.
func attributeUseToken(el *Element) string {
	if use, hasUse := el.Attr("use"); hasUse {
		return use
	}
	return "optional"
}

// useValueConstraintOK charges src-attribute (§3.2.3) clauses 2 and 5 against one
// <attribute> element: clause 2, "If default and use are both present, use must
// have the ·actual value· optional", and clause 5, "If fixed and use are both
// present, use must not have the ·actual value· prohibited". Neither clause is
// guarded by the element's parent, but only the local and ref= forms reach it:
// both clauses need a use= present, and the top-level form may not carry one at
// all (xs:topLevelAttribute, xmlschema11-1.md:4712, enforced in run by
// rejectProhibitedAttrs).
//
// The check lives here rather than in valueConstraintOf because that helper is
// shared with <element> (charged src-element clause 1), and <element> has no use=
// attribute to consult. valueConstraintOf stays the single encoding of clause 1.
//
// The token is compared exactly, as the rest of produceAttributeUse compares it:
// a use= outside {optional, prohibited, required} therefore reads as
// other-than-optional here and is rejected when default is present. That is not
// src-attribute's own enumeration check — the enumeration is imposed by the
// schema for schema documents, which src-attribute is explicitly "in addition
// to" — but no schema it rejects was valid under that schema either.
func useValueConstraintOK(el *Element) error {
	use := attributeUseToken(el)
	if _, hasDefault := el.Attr("default"); hasDefault && use != "optional" {
		return xsderr.New(ruleSrcAttribute, el.Loc(),
			"attribute has default with use=%q, but src-attribute clause 2 requires use to be optional when default is present", use)
	}
	if _, hasFixed := el.Attr("fixed"); hasFixed && use == "prohibited" {
		return xsderr.New(ruleSrcAttribute, el.Loc(),
			"attribute has fixed with use=%q, but src-attribute clause 5 forbids prohibited when fixed is present", use)
	}
	return nil
}

// produceLocalAttribute maps the sibling local Attribute Declaration of a local
// <attribute> (§3.2.2.2, {scope} = local, {value constraint} always absent on the
// declaration — any default/fixed feeds the Attribute Use, #70). Its {type
// definition} is mapped by declaredType over §3.2.2.2's three tiers: the
// inline <simpleType> child (#229), the type= reference, or xs:anySimpleType.
// src-attribute clause 4 (§3.2.3) rejects the both-present case first.
//
// scopeParent is the containing <complexType>'s or <attributeGroup>'s component,
// supplied by the caller (never recomputed from the element here — the ancestor
// axis is not walkable from an *Element). It is a required parameter: every local
// attribute declaration has a {scope}.{parent} (§3.2.1 sc_a), so there is no path
// through this function that does not build the scope from it.
func (p *producer) produceLocalAttribute(el *Element, scopeParent xsd.AttributeScopeParent) (xsd.AttributeDeclaration, error) {
	_, hasType := el.Attr("type")
	if hasType && childElement(el, xsd.XMLSchemaNS, "simpleType") != nil {
		return xsd.AttributeDeclaration{}, xsderr.New(ruleSrcAttribute, el.Loc(),
			"attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both")
	}
	qname, err := declarationName(el, p.localTargetNS(el, "attributeFormDefault"))
	if err != nil {
		return xsd.AttributeDeclaration{}, err
	}
	typeDef, err := p.declaredType(el, anySimpleTypeName)
	if err != nil {
		return xsd.AttributeDeclaration{}, err
	}
	inheritable, _ := boolAttr(el, "inheritable")
	scope, err := xsd.NewAttributeLocalScope(el.Loc(), scopeParent)
	if err != nil {
		return xsd.AttributeDeclaration{}, err
	}
	return xsd.NewAttributeDeclaration(el.Loc(), qname, typeDef, scope, nil, inheritable, nil)
}

// produceWildcard maps an <any>/<anyAttribute> to a Wildcard (§3.10.2.2). It
// enforces src-wildcard (§3.10.3): namespace and notNamespace must not both be
// present.
func (p *producer) produceWildcard(el *Element) (xsd.Wildcard, error) {
	nc, err := p.namespaceConstraint(el)
	if err != nil {
		return xsd.Wildcard{}, err
	}
	process := xsd.ProcessStrict
	if pc, ok := el.Attr("processContents"); ok {
		process, err = processContentsOf(pc, el.Loc())
		if err != nil {
			return xsd.Wildcard{}, err
		}
	}
	return xsd.NewWildcard(el.Loc(), nc, process, nil)
}

// namespaceConstraint maps the namespace/notNamespace/notQName attributes of an
// <any>/<anyAttribute> to a Namespace Constraint (§3.10.2.2).
func (p *producer) namespaceConstraint(el *Element) (xsd.NamespaceConstraint, error) {
	ns, hasNS := el.Attr("namespace")
	notNS, hasNotNS := el.Attr("notNamespace")
	if hasNS && hasNotNS {
		return xsd.NamespaceConstraint{}, xsderr.New(ruleSrcWildcard, el.Loc(),
			"wildcard has both namespace and notNamespace, but src-wildcard forbids both")
	}
	disallowed, keywords, err := p.disallowedNames(el)
	if err != nil {
		return xsd.NamespaceConstraint{}, err
	}
	variety, namespaces := p.namespaceVarietyAndSet(ns, hasNS, notNS, hasNotNS)
	return xsd.NewNamespaceConstraint(el.Loc(), variety, namespaces, disallowed, keywords)
}

// namespaceVarietyAndSet computes {variety} and {namespaces} (§3.10.2.2):
//   - neither namespace nor notNamespace present → any, empty set;
//   - namespace="##any" → any, empty set;
//   - namespace="##other" → not, {·absent·} plus the target namespace if present;
//   - otherwise (a namespace/notNamespace token list) → enumeration for namespace
//     or not for notNamespace, with ##targetNamespace/##local substituted.
func (p *producer) namespaceVarietyAndSet(ns string, hasNS bool, notNS string, hasNotNS bool) (xsd.NamespaceConstraintVariety, []xsd.Namespace) {
	if !hasNS && !hasNotNS {
		return xsd.NamespaceConstraintAny, nil
	}
	if hasNS && ns == "##any" {
		return xsd.NamespaceConstraintAny, nil
	}
	if hasNS && ns == "##other" {
		set := []xsd.Namespace{xsd.NamespaceName("")}
		if p.target != "" {
			set = append(set, xsd.NamespaceName(p.target))
		}
		return xsd.NamespaceConstraintNot, set
	}
	list := ns
	variety := xsd.NamespaceConstraintEnumeration
	if hasNotNS {
		list = notNS
		variety = xsd.NamespaceConstraintNot
	}
	var set []xsd.Namespace
	for _, tok := range strings.Fields(list) {
		switch tok {
		case "##targetNamespace":
			set = append(set, xsd.NamespaceName(p.target))
		case "##local":
			set = append(set, xsd.NamespaceName(""))
		default:
			set = append(set, xsd.NamespaceName(tok))
		}
	}
	return variety, set
}

// disallowedNames maps a notQName attribute to the two halves of {disallowed
// names} (§3.10.2.2): its literal QName items, and its ##defined/##definedSibling
// keyword tokens mapped to the component keywords defined/sibling. Note the
// asymmetry the spec fixes at §3.10.2.2: ##definedSibling maps to the keyword
// sibling ALONE, not to both keywords.
//
// An unrecognized ##-prefixed token is rejected. That is not a bespoke Structures
// constraint — src-wildcard (§3.10.3) says nothing about notQName's content — but
// an ordinary datatype-validity failure of the attribute against the type the
// schema for schema documents declares for it (§3.10.2): xs:qnameList for <any>,
// whose token member type enumerates exactly ##defined and ##definedSibling, and
// xs:qnameListA for <anyAttribute>, which enumerates only ##defined. So
// notQName="##definedSibling" on an <anyAttribute> is rejected HERE, by that
// enumeration, and is the machine-checkable form of w-props-correct clause 5 —
// the parser deliberately does not also repeat clause 5 as a component check (one
// enforcement point); xsd.NewComplexType/NewAttributeGroupDefinition still charge
// clause 5 on the programmatic-construction path that bypasses this producer.
//
// The literal-QName arm is equally hard-failing: a member whose prefix has no
// in-scope binding cannot be mapped to a QName value at all (Datatypes §3.3.18),
// and §3.10.2.2 offers no fallback that omits it, so dropping it would silently
// shrink {disallowed names} into a more permissive wildcard than the schema
// declares. bindQName's src-resolve error is propagated verbatim.
//
// It is bindQName rather than resolveQName precisely because §3.10.2's mapping
// takes these items as QName VALUES and never ·resolves· them to components: a
// notQName naming a namespace this document did not <import> blocks a name, it
// does not reach for a declaration, so src-resolve clause 4's licensing test
// (licensedNamespace) does not apply to it.
func (p *producer) disallowedNames(el *Element) ([]xsd.QName, []xsd.DisallowedNameKeyword, error) {
	notQName, ok := el.Attr("notQName")
	if !ok {
		return nil, nil, nil
	}
	attributeWildcard := el.Name().Local() == "anyAttribute"
	var names []xsd.QName
	var keywords []xsd.DisallowedNameKeyword
	for _, tok := range strings.Fields(notQName) {
		if strings.HasPrefix(tok, "##") {
			kw, err := disallowedNameKeywordOf(tok, attributeWildcard, el.Loc())
			if err != nil {
				return nil, nil, err
			}
			keywords = append(keywords, kw)
			continue
		}
		qn, err := p.bindQName(el, tok, "notQName")
		if err != nil {
			return nil, nil, err
		}
		names = append(names, qn)
	}
	return names, keywords, nil
}

// disallowedNameKeywordOf maps one ##-prefixed notQName token to its §3.10.1
// component keyword. attributeWildcard selects the declared type of the notQName
// attribute being validated (§3.10.2): xs:qnameListA for <anyAttribute>, which
// enumerates only ##defined, versus xs:qnameList for <any>, which also enumerates
// ##definedSibling. A token outside the applicable enumeration fails that type,
// charged cvc-datatype-valid — the generic "attribute value is not valid against
// its declared simple type" rule, not a wildcard-specific one.
func disallowedNameKeywordOf(tok string, attributeWildcard bool, loc xsderr.Loc) (xsd.DisallowedNameKeyword, error) {
	if tok == "##defined" {
		return xsd.DisallowedNameDefined, nil
	}
	if tok == "##definedSibling" && !attributeWildcard {
		return xsd.DisallowedNameSibling, nil
	}
	declaredType := "xs:qnameList"
	enumerated := "##defined or ##definedSibling"
	if attributeWildcard {
		declaredType, enumerated = "xs:qnameListA", "##defined"
	}
	return 0, xsderr.New(ruleDatatypeValid, loc,
		"notQName token %q is not valid against %s, whose keyword member type enumerates only %s", tok, declaredType, enumerated)
}

// localTargetNS computes a local element/attribute declaration's {target
// namespace} (§3.3.2.3 / §3.2.2.2): an explicit targetNamespace attribute wins,
// else form= (qualified → the schema target, unqualified → absent), else the
// schema's *FormDefault (formDefaultAttr is "elementFormDefault" or
// "attributeFormDefault"), defaulting to absent (unqualified).
func (p *producer) localTargetNS(el *Element, formDefaultAttr string) string {
	if tns, ok := el.Attr("targetNamespace"); ok {
		return tns
	}
	if form, ok := el.Attr("form"); ok {
		if form == "qualified" {
			return p.target
		}
		return ""
	}
	if fd, ok := p.schemaElem.Attr(formDefaultAttr); ok && fd == "qualified" {
		return p.target
	}
	return ""
}

// occursOf maps the minOccurs/maxOccurs attributes to an Occurs (§3.9.2), each
// defaulting to 1. elided is true for the minOccurs=maxOccurs=0 case, which the
// XML mapping rules say "maps to no component at all" (§3.7.2/§3.8.2/§3.9.2) — the
// caller omits the particle entirely rather than building a vacuous Occurs{0,0}.
func occursOf(el *Element) (occ xsd.Occurs, elided bool, err error) {
	min := 1
	if minS, ok := el.Attr("minOccurs"); ok {
		min, err = nonNegativeInt(minS, el.Loc(), "minOccurs")
		if err != nil {
			return xsd.Occurs{}, false, err
		}
	}
	unbounded := false
	max := 1
	if maxS, ok := el.Attr("maxOccurs"); ok {
		if strings.TrimSpace(maxS) == "unbounded" {
			unbounded = true
		} else {
			max, err = nonNegativeInt(maxS, el.Loc(), "maxOccurs")
			if err != nil {
				return xsd.Occurs{}, false, err
			}
		}
	}
	if !unbounded && min == 0 && max == 0 {
		return xsd.Occurs{}, true, nil
	}
	if unbounded {
		occ, err = xsd.NewUnboundedOccurs(el.Loc(), min)
		return occ, false, err
	}
	occ, err = xsd.NewOccurs(el.Loc(), min, max)
	return occ, false, err
}

// nonNegativeInt parses an xs:nonNegativeInteger-valued occurrence attribute,
// charging p-props-correct (§3.9.6.1) on a malformed or negative value.
func nonNegativeInt(lexical string, loc xsderr.Loc, attr string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(lexical))
	if err != nil || n < 0 {
		return 0, xsderr.New(ruleParticleCorr, loc,
			"%s value %q is not a nonNegativeInteger (p-props-correct)", attr, lexical)
	}
	return n, nil
}

// allOccursGrammar enforces the occurrence grammar the schema for schema
// documents gives the <all> ELEMENT (§3.8.2's XML representation summary,
// formalized in Appendix A's xs:complexType "all"): minOccurs restricts
// xs:nonNegativeInteger and maxOccurs restricts xs:allNNI, each by an enumeration
// admitting only 0 and 1 — so "unbounded", legal on <choice>/<sequence>, is
// excluded here along with every integer above 1. An absent attribute takes the
// declared default 1 and passes.
//
// §3.8.3 lists "None as such" for <all>'s Schema Representation Constraints, so
// the fault carries no src-*/cos-* rule: it is charged cvc-datatype-valid, the
// generic "attribute value is not valid against its declared type" rule. It is
// NOT cos-all-limited, which constrains where the resulting particle may appear
// rather than what the element's attributes may say, and not p-props-correct,
// which covers lexicals that are not nonNegativeIntegers at all (or max < min).
//
// Only the content-model <all> is checked: on the <all> body of a top-level named
// <group>, Appendix A's xs:namedGroup makes both attributes use="prohibited", a
// presence fault of a different declaration that this function does not model.
func allOccursGrammar(el *Element) error {
	if lexical, ok := el.Attr("minOccurs"); ok {
		if err := allOccursEnum(lexical, el.Loc(), "minOccurs"); err != nil {
			return err
		}
	}
	lexical, ok := el.Attr("maxOccurs")
	if !ok {
		return nil
	}
	if strings.TrimSpace(lexical) == "unbounded" {
		return xsderr.New(ruleDatatypeValid, el.Loc(),
			`<all> maxOccurs is "unbounded", but the schema for schema documents restricts it to the enumeration 0, 1`)
	}
	return allOccursEnum(lexical, el.Loc(), "maxOccurs")
}

// allOccursEnum checks one numeric <all> occurrence lexical against the {0,1}
// enumeration. The enumeration facet compares VALUES, so "01" is the value 1 and
// passes, matching how occursOf reads the same attribute; a lexical that is not a
// nonNegativeInteger at all fails its base type first, charged p-props-correct by
// nonNegativeInt.
func allOccursEnum(lexical string, loc xsderr.Loc, attr string) error {
	n, err := nonNegativeInt(lexical, loc, attr)
	if err != nil {
		return err
	}
	if n > 1 {
		return xsderr.New(ruleDatatypeValid, loc,
			"<all> %s value %q is outside the enumeration 0, 1 that the schema for schema documents declares for it", attr, lexical)
	}
	return nil
}

// processContentsOf maps a processContents lexical to a ProcessContents token,
// charging w-props-correct (§3.10.6.1) on an out-of-range value.
func processContentsOf(lexical string, loc xsderr.Loc) (xsd.ProcessContents, error) {
	switch strings.TrimSpace(lexical) {
	case "skip":
		return xsd.ProcessSkip, nil
	case "strict":
		return xsd.ProcessStrict, nil
	case "lax":
		return xsd.ProcessLax, nil
	}
	return 0, xsderr.New(ruleWildcardCorr, loc,
		"wildcard processContents %q is not one of skip/strict/lax", lexical)
}

// compositorOf maps an <all>/<choice>/<sequence> local name to its Compositor.
// ok is false for <group> (a reference, mapped by produceGroupRefParticle) and
// any other name.
func compositorOf(local string) (xsd.Compositor, bool) {
	switch local {
	case "all":
		return xsd.CompositorAll, true
	case "choice":
		return xsd.CompositorChoice, true
	case "sequence":
		return xsd.CompositorSequence, true
	}
	return 0, false
}

// modelGroupChild returns el's first <all>/<choice>/<sequence>/<group> child (the
// model-group child of a <complexType>/<restriction>/<extension>), or nil.
func modelGroupChild(el *Element) *Element {
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch c.Name().Local() {
		case "all", "choice", "sequence", "group":
			return c
		}
	}
	return nil
}

// hasParticleChild reports whether group has any non-<annotation> element child
// (an element/any/group/choice/sequence), i.e. whether it is non-empty for the
// §3.4.2.3.3 clause 2 empty-content tests.
func hasParticleChild(group *Element) bool {
	for _, child := range group.Children() {
		c, ok := child.(*Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if c.Name().Local() != "annotation" {
			return true
		}
	}
	return false
}

// minOccursZero reports whether el's minOccurs actual value is 0.
func minOccursZero(el *Element) bool {
	v, ok := el.Attr("minOccurs")
	return ok && strings.TrimSpace(v) == "0"
}

// maxOccursZero reports whether el's maxOccurs actual value is 0.
func maxOccursZero(el *Element) bool {
	v, ok := el.Attr("maxOccurs")
	return ok && strings.TrimSpace(v) == "0"
}

// attrOr returns el's attribute local value, or the empty string when absent.
func attrOr(el *Element, local string) string {
	v, _ := el.Attr(local)
	return v
}

// boolAttr reads an xs:boolean-valued attribute (true/1 → true), reporting
// presence. An absent attribute is (false, false).
func boolAttr(el *Element, local string) (val bool, present bool) {
	v, ok := el.Attr(local)
	if !ok {
		return false, false
	}
	return v == "true" || v == "1", true
}
