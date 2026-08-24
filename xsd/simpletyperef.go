package xsd

import "github.com/kud360/goxsd8/xsderr"

// SimpleTypeOrRef is the slot holding one of a Simple Type Definition's three
// type-valued properties (§3.16.1) — {base type definition}, {item type
// definition} and each member of {member type definitions} — as either a
// DEFERRED reference by expanded name, resolved at finalize, or a component the
// slot already holds. It is a sealed sum (STYLE T2): SimpleTypeRef and
// OwnedSimpleType are its only implementations, sealed by the unexported
// simpleTypeOrRef method, so consumers exhaustively switch the two branches and
// no third variant is representable.
//
// WHY A SECOND SUM beside TypeDefinitionOrRef (typedefinition.go), which STYLE
// T4 would otherwise refuse. This slot's arm × slot legality table is genuinely
// narrower: a simple type's base is never a ComplexType, so there is no
// InlineTypeDefinition-wrapping-a-complex-type state to reject at runtime, and
// §3.3.2.1 dcl.elt.common clause 3 has no analog here, so there is no
// SubstitutionGroupHeadTypeRef arm. Reusing TypeDefinitionOrRef was rejected on
// two counts: it has NO arm for a component already held that also has a {name}
// (checkTypeDefinitionOrRef rejects an InlineTypeDefinition wrapping a named
// type outright), and adding a fourth arm to it would re-open the arm-by-slot
// legality table for all four slots that sum already serves.
//
// THE ARM × SLOT LEGALITY TABLE. Both arms are legal in all three slots; nil is
// legal in exactly one, and every other absent-encoding is illegal everywhere:
//
//	slot                       nil    SimpleTypeRef{Name}   OwnedSimpleType{Definition}
//	{base type definition}     legal  legal, Name present   legal, Definition present
//	{item type definition}     ILLEGAL  "        "          "        "
//	{member type definitions}[i]  ILLEGAL  "     "          "        "
//
// nil in the base slot is the single encoding of an ·absent· {base type
// definition}: the type IS xs:anySimpleType, whose real base xs:anyType is a
// Complex Type Definition outside this package's scope, and IsAnySimpleType is
// exactly that predicate. A list always HAS an item and a membership never holds
// an absent member (§3.16.1), so nil there encodes nothing — as do the two
// forgeable near-misses, a zero-named SimpleTypeRef and an OwnedSimpleType
// wrapping nil. NewSimpleType rejects all three in the item and member slots
// (checkSimpleTypeOrRefPresent) and the latter two in the base slot
// (checkSimpleTypeOrRef), so absence is decided ONCE, at construction, and
// simpleTypeOfRef never multiplexes it per caller: a nil it sees came from the
// base slot and means xs:anySimpleType.
//
// That construction-time discharge is what keeps this sum's contract narrower
// than its sibling TypeDefinitionOrRef's, whose doc must instead make "the
// meaning of nil is a property of the SUM, not of the slot the caller reached it
// through" a doctrine, because ResolvedType and resolveTypeDefinitionSlot answer for four
// slots at once at READ time (STYLE T6, so the two do not drift silently).
type SimpleTypeOrRef interface{ simpleTypeOrRef() }

// SimpleTypeRef is the DEFERRED by-name arm: the base= of a §3.16.2.1
// map.std.restriction alternative, the itemType= of a map.std.list one, or one
// entry of a map.std.union memberTypes=, each naming a top-level simple type
// definition which may be forward-referenced, so only a QName is available at
// mapping time. It is resolved once, at finalize, through simpleTypeOfRef. The
// field is read-only by convention; do not mutate it after construction.
//
// EVERY by-name base, item and member is this arm — see OwnedSimpleType for the
// prohibition that makes that binding rather than customary.
//
// Name is a PRESENT reference, never the absent (zero) QName: a reference that
// names nothing cannot be followed, so it is an illegal representation rather
// than a resolution failure to defer to finalize. NewSimpleType rejects one,
// exactly as the TypeDefinitionRef precedent (typedefinition.go) is rejected by
// NewElementDeclaration and its siblings.
type SimpleTypeRef struct{ Name QName }

// OwnedSimpleType is the arm carrying a *SimpleType the slot already holds,
// reached through no symbol table and so needing no resolution.
//
// OWNERSHIP, NOT ANONYMITY, is this arm's invariant — the same word
// InlineTypeDefinition's doc uses for its own arm (typedefinition.go), so the
// two read as deliberate variants of one idea rather than as a slip. It differs
// from InlineTypeDefinition in exactly one respect, and the difference is why
// that arm's anonymity invariant cannot be carried across: a *SimpleType chain
// can be fully assembled before any Schema exists — builtin.Seed builds the
// whole builtin graph with named components and no Schema anywhere, and the
// conformance datatypes lane synthesizes named types the same way — a state
// TypeDefinitionOrRef has no slot for. So this arm ADMITS a named definition,
// and Definition.Name() may be the zero QName or not.
//
// THAT LATITUDE IS BOUNDED, and the bound is part of this arm's contract rather
// than a convention some producer happens to follow: a producer mapping a schema
// document emits SimpleTypeRef for EVERY by-name base, itemType= and
// memberTypes= entry, and OwnedSimpleType only for a slot-owned inline
// <simpleType>, the §4.2.4 src-expredef ORIGINAL a redefining <simpleType> is
// paired with, a component a MAPPING RULE names outright — §3.4.2.2's cases 1.2
// and 2 hand the anonymous simple type a <simpleContent> <restriction>
// synthesizes a B read off the already-resolved base complex type, which the
// source names by no QName of its own — or a pre-assembled Schema-less graph.
// Without that prohibition this arm is an escape hatch that lets any producer
// opt out of deferred resolution one call site at a time, and nothing detects
// the drift; it is pinned by tests asserting a produced named base= and a
// produced named itemType= each store SimpleTypeRef.
//
// Definition is always present: in the base slot nil-the-interface is the only
// encoding of absent, and in the item and member slots nothing encodes absent at
// all, so an OwnedSimpleType wrapping nil is a second encoding of absent or of
// nothing (STYLE D3). NewSimpleType rejects one in every slot. The field is
// read-only by convention; do not mutate it after construction.
//
// It carries a *SimpleType and not a TypeDefinition, which makes original item
// 7's runtime rejection — a ComplexType written into the simple-type base slot —
// a COMPILE-time impossibility instead.
type OwnedSimpleType struct{ Definition *SimpleType }

// simpleTypeOrRef marks SimpleTypeRef as a SimpleTypeOrRef; see the
// SimpleTypeOrRef doc.
func (SimpleTypeRef) simpleTypeOrRef() {}

// simpleTypeOrRef marks OwnedSimpleType as a SimpleTypeOrRef; see the
// SimpleTypeOrRef doc.
func (OwnedSimpleType) simpleTypeOrRef() {}

// checkSimpleTypeOrRef rejects the encodings the {base type definition} slot —
// the ONE slot of the three where nil is legal (SimpleTypeOrRef's arm × slot
// table) — may not hold. A nil ref is that slot's absent encoding and passes;
// anything else must be present, which is checkSimpleTypeOrRefPresent's verdict.
func checkSimpleTypeOrRef(loc xsderr.Loc, ref SimpleTypeOrRef) error {
	if ref == nil {
		return nil
	}
	return checkSimpleTypeOrRefPresent(loc, ref, "{base type definition}")
}

// checkSimpleTypeOrRefPresent rejects every encoding of ABSENCE in a slot that
// must hold a type: a nil ref, a SimpleTypeRef naming nothing, and an
// OwnedSimpleType holding nothing. It is charged to
// xsderr.RuleComponentInvariant: these are representation invariants this
// package owns, not spec clauses a schema author can violate — the same footing
// checkTypeDefinitionOrRef rejects a zero-named TypeDefinitionRef on.
//
// slot names the property for the message ("{item type definition}",
// "{member type definitions}[2]"), and is what lets the item and member slots
// discharge their absence at CONSTRUCTION rather than leaving simpleTypeOfRef to
// answer "absent" differently per caller at read time.
func checkSimpleTypeOrRefPresent(loc xsderr.Loc, ref SimpleTypeOrRef, slot string) error {
	switch r := ref.(type) {
	case nil:
		return xsderr.New(xsderr.RuleComponentInvariant, loc,
			"simple type %s is absent, but that slot must hold a simple type definition", slot)
	case SimpleTypeRef:
		if r.Name == (QName{}) {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"simple type %s is a SimpleTypeRef carrying the absent (zero) QName, but that variant names a simple type definition reachable by name", slot)
		}
		return nil
	case OwnedSimpleType:
		if r.Definition == nil {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"simple type %s is an OwnedSimpleType with no definition, but that variant holds the component outright", slot)
		}
		return nil
	default:
		panic("xsd: checkSimpleTypeOrRefPresent: non-exhaustive SimpleTypeOrRef switch")
	}
}

// simpleTypeOfRef is the ONE way this package turns a SimpleTypeOrRef slot into
// a component, exhaustively over the sum's two arms — the narrowed sibling of
// ResolvedType (typedefinition.go), and the single site charging src-resolve clause
// 1.1 for a simple type's base, item and members. It is NOT (*Schema).ResolvedSimpleType
// (defaultbinding.go), which narrows ResolvedType over the OTHER sum,
// TypeDefinitionOrRef, for the {type definition} slots; the names are kept apart
// deliberately so a reader grepping either finds one thing.
//
// The arms:
//
//   - nil is an ·absent· base (the type IS xs:anySimpleType): (nil, nil), which
//     every caller reads as the end of the chain, never as a failure. It reaches
//     here from the base slot alone — NewSimpleType rejects a nil item or member
//     — so the answer needs no per-caller multiplexing (SimpleTypeOrRef's arm ×
//     slot table).
//   - OwnedSimpleType IS the component; it is in no by-name symbol table, so a
//     lookup would miss it.
//   - SimpleTypeRef is the r.Type lookup. BOTH a miss and a wrong-kind hit (the
//     name resolves to a ComplexType) are charged src-resolve clause 1.1: they
//     are the same failure seen twice — the kind-specific lookup simply misses —
//     which is the argument ruleSrcResolve's own doc already makes.
//
// It returns an ERROR rather than a comma-ok, because an unresolvable base is
// exactly the silently short chain a resolver-threaded reader must never
// produce: a caller that read a missing base as "the chain ended" would compute
// {variety}, {primitive type definition} or {facets} off a truncated chain and
// ACCEPT what the full chain forbids. The complex-type precedent (ResolvedType's
// comma-ok) discharges that with "Phase A already rejected a dangling one", an
// argument that does not transfer here because these readers are also called
// from outside finalize, where no phase has run.
//
// loc positions a rejection at the REFERRING component — the type whose base=
// names nothing — following resolveReferences' referrer-Loc convention; the
// target is exactly what does not exist, so it has no position. ctx names that
// referring site for the message.
func simpleTypeOfRef(r TypeResolver, ref SimpleTypeOrRef, loc xsderr.Loc, ctx string) (*SimpleType, error) {
	switch b := ref.(type) {
	case nil:
		return nil, nil
	case OwnedSimpleType:
		return b.Definition, nil
	case SimpleTypeRef:
		t, ok := r.Type(b.Name)
		if !ok {
			return nil, xsderr.New(ruleSrcResolve, loc,
				"%s references simple type %s, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)", ctx, b.Name)
		}
		st, ok := t.(*SimpleType)
		if !ok {
			return nil, xsderr.New(ruleSrcResolve, loc,
				"%s references simple type %s, but that expanded name is a complex type definition, so the simple-type lookup finds nothing (src-resolve clause 1.1)", ctx, b.Name)
		}
		return st, nil
	default:
		panic("xsd: simpleTypeOfRef: non-exhaustive SimpleTypeOrRef switch")
	}
}

// ownedSimpleType returns the component a slot holds OUTRIGHT, or nil for a
// by-name arm or an absent slot. It is the one encoding of "descend this slot
// only if nothing has to be looked up to follow it" (STYLE T4), which is the
// test both finalize descents make on all three slots — the by-name arms name
// top-level types those passes reach in their own right.
func ownedSimpleType(ref SimpleTypeOrRef) *SimpleType {
	owned, ok := ref.(OwnedSimpleType)
	if !ok {
		return nil
	}
	return owned.Definition
}

// simpleTypeLabel names t for a rejection message, mirroring
// complexTypeOwner/typeDefinitionLabel: a named type by its expanded name, an
// anonymous one by that fact (the zero QName is anonymity, not a missing value —
// see SimpleType's {name} godoc).
func simpleTypeLabel(t *SimpleType) string {
	if t.name == (QName{}) {
		return "anonymous simple type"
	}
	return "simple type " + t.name.String()
}
