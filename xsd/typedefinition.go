package xsd

import "github.com/kud360/goxsd8/xsderr"

// TypeDefinition is the sealed sum of the two kinds that populate a schema's
// {type definitions} property (Structures §3.17.1): a ComplexType (§3.4.1) or
// a *SimpleType (§3.16.1 / Datatypes §4.1.1). §3.17.6.2 clause 1.1 unifies both
// into one lookup bucket — a shared name between a simple and a complex type in
// one namespace is exactly the sch-props-correct (§3.17.6.1 clause 2)
// collision, so one sum keyed once makes the two-map illegal state
// unrepresentable (STYLE T7). The unexported typeDefinition marker method seals
// it (STYLE T2/T7): consumers exhaustively switch these two variants and no
// third is representable, mirroring term.go's Term and complextype.go's
// ContentType sealed sums.
//
// The two variants satisfy TypeDefinition with different receiver kinds, a
// deliberate asymmetry a consumer must respect: ComplexType satisfies it BY
// VALUE (its own methods are value-receiver), while *SimpleType satisfies it BY
// POINTER (SimpleType's own methods are pointer-receiver, and its component
// identity is load-bearing — see SimpleType's doc). An exhaustive type switch
// over a TypeDefinition therefore switches on `ComplexType` and `*SimpleType`,
// never on `SimpleType`.
type TypeDefinition interface {
	typeDefinition()
	// Name is the {name} property bundled with {target namespace} as a QName;
	// the zero QName marks an anonymous type definition. Both variants already
	// expose it (ComplexType.Name, (*SimpleType).Name), so it is promoted into
	// the sum for name-keyed lookup without a type switch.
	Name() QName
	// Loc is the type definition's source position — provenance, not a
	// component property of either variant (§3.4.1, §3.16.1; see the package
	// doc's Components section). Both variants already expose it
	// (ComplexType.Loc, (*SimpleType).Loc), so it is promoted into the sum to
	// let the one {type definitions} bucket cite a position without a type
	// switch (STYLE T7).
	Loc() xsderr.Loc
}

// typeDefinition marks ComplexType as a TypeDefinition (§3.17.1); see the
// TypeDefinition doc. ComplexType satisfies the sum by value.
func (ComplexType) typeDefinition() {}

// typeDefinition marks *SimpleType as a TypeDefinition (§3.17.1); see the
// TypeDefinition doc. *SimpleType satisfies the sum by pointer — its component
// identity is load-bearing (see SimpleType's doc), so the marker is on the
// pointer, not the value.
func (*SimpleType) typeDefinition() {}

// TypeDefinitionOrRef is a slot holding a reference to a type definition. It
// serves TWO §3 properties, which is why its doc is written in terms of the slot
// rather than of one property:
//
//   - the {type definition} of an Element Declaration (§3.3.1) or an Attribute
//     Declaration (§3.2.1);
//   - the {base type definition} of a Complex Type Definition (§3.4.1), which
//     src-expredef clause 1.1 (§4.2.4) makes able to hold an anonymous
//     already-resolved component (see ComplexType.Base).
//
// It is a sealed sum (STYLE T2/T7): TypeDefinitionRef, InlineTypeDefinition and
// SubstitutionGroupHeadTypeRef are its only implementations, sealed by the
// unexported typeDefinitionOrRef method, so consumers exhaustively switch the
// three branches and no fourth variant is representable. It copies
// attributeuse.go's AttributeDeclarationOrRef precedent exactly, and for the same
// reason: the XML mappings that populate the slot differ fundamentally in
// OWNERSHIP of the type definition they yield.
//
//   - TypeDefinitionRef covers §3.3.2.1 dcl.elt.common clauses 2 and 4, clause
//     3's NAMED-head case, §3.2.2.2 dcl.att.local's type= and xs:anySimpleType
//     tiers, and every ordinary base= of §3.4.2: the type is a top-level
//     component reachable BY NAME, possibly forward-referenced, so only a
//     deferred QName is available at parse time.
//   - InlineTypeDefinition covers the three mapping rules that OWN the type
//     they yield outright; see that type.
//   - SubstitutionGroupHeadTypeRef covers §3.3.2.1 clause 3's ANONYMOUS-head
//     case: the one mapping that references a type definition the slot neither
//     owns nor can name, because the OWNER is another Element Declaration. See
//     that type.
//
// NOT EVERY ARM IS LEGAL IN EVERY SLOT, and the whole arm × slot legality table
// lives in exactly one place, checkTypeDefinitionOrRef's typeDefinitionSlot
// switch: SubstitutionGroupHeadTypeRef is admitted only by an ELEMENT
// declaration's {type definition}, since neither §3.2.2.2 nor §3.4.2 has a
// clause-3 analog to produce one. Making that unrepresentable at COMPILE time
// would take a second sealed interface for the element slot alone, which forks
// ResolvedType and checkTypeDefinitionOrRef (STYLE T4) and changes
// ElementDeclaration.TypeDefinition()'s return type; it is deliberately not
// done. A runtime rejection charged to xsderr.RuleComponentInvariant is the
// established precedent here — NewElementDeclaration rejects an
// InlineTypeDefinition wrapping a ComplexType the same way.
//
// A nil TypeDefinitionOrRef is the single encoding of an ABSENT slot, in EVERY
// property this sum serves — the state a programmatically built declaration is
// in before the §3.3.2.1/§3.2.2.2 defaulting tiers are applied, and the state a
// ComplexType built with no base name is in. The meaning of nil is a property of
// the SUM, not of the slot the caller reached it through: a consumer that
// followed a slot must never read nil as "the ur-type" or as any other present
// component, because ResolvedType and resolveTypeDefinition answer for all slots at
// once and cannot be made arm-meaning-dependent on the caller's origin. It
// replaces the zero QName that used to carry that meaning, which was ambiguous:
// the same QName{} also had to stand for "anonymous type" (STYLE D3, one fact
// one encoding). Anonymity is now DERIVED from which arm the slot holds, never
// separately stored: an InlineTypeDefinition is anonymous by construction, a
// TypeDefinitionRef never is.
type TypeDefinitionOrRef interface{ typeDefinitionOrRef() }

// TypeDefinitionRef is the variant naming a top-level type definition: the
// type/@type of §3.3.2/§3.2.2, the {type definition} an element inherits from a
// substitution-group head whose own type is NAMED (§3.3.2.1 clause 3), the
// xs:anyType / xs:anySimpleType default (§3.3.2.1 clause 4, §3.2.2.2), and the
// base= of a §3.4.2 derivation. All are reachable by name, so all are this arm.
// The field is read-only by convention; do not mutate it after construction.
//
// Clause 3's other half is NOT this arm: a head whose own {type definition} is
// the ANONYMOUS type of its own inline <complexType> has no name for this arm to
// carry, and the member does not own it either, so that case is
// SubstitutionGroupHeadTypeRef.
//
// Name is a PRESENT reference, never the absent (zero) QName: a reference that
// names nothing could not be followed, so it is not a resolution failure to
// defer to finalize but an illegal representation. NewElementDeclaration,
// NewAttributeDeclaration and the ComplexType constructors reject one, exactly
// as NewAttributeUse rejects a zero-named AttributeDeclarationRef.
type TypeDefinitionRef struct{ Name QName }

// InlineTypeDefinition is the variant that OWNS the anonymous type definition it
// wraps, carried by value because the slot holding it is its sole owner.
//
// Ownership, not XML provenance, is this arm's invariant, and all four
// properties below are stated in those terms. THREE mapping rules produce it,
// and a fourth would be admitted on the same footing — but only a fourth that
// OWNS what it yields. §3.3.2.1 clause 3's anonymous-head case is the mapping
// that does not: the head's inline <complexType> already has an owner (the head
// declaration, named by its {context}), so a member inheriting it gets the
// non-owning SubstitutionGroupHeadTypeRef instead of a second owner here. The
// copy that would let it be this arm is literally unconstructible —
// NewElementDeclarationOwningType rejects a type whose {context} names another
// declaration.
//
// The three rules that do own what they yield:
//
//   - §3.3.2.1 dcl.elt.common clause 1 — the anonymous type of an inline
//     <simpleType>/<complexType> child of an <element>;
//   - §3.2.2.2 dcl.att.local's first tier — the same for an <attribute>;
//   - §4.2.4 src-expredef clause 1.1 — the {name}-·absent· ORIGINAL a redefining
//     <simpleType>/<complexType> is paired with, which is not an XML child of
//     anything in the redefining document at all: it corresponds to the
//     REDEFINED document's own top-level definition item, "as defined in Schema
//     Component Details (§3), except that its {name} is ·absent· and its
//     {context} is the redefining component". It satisfies every property below
//     — it is registered nowhere, it is built in the same producer call, the
//     {base type definition} slot holding it is its sole owner, and its {name}
//     is absent — so it is this arm rather than a fourth one recording
//     provenance the component model does not otherwise carry (STYLE T4).
//
// The wrapped definition is deliberately NOT registered in the schema's
// {type definitions} (SchemaBuilder.AddType). §3.17.2's XML Mapping Summary
// scopes that property to "the <simpleType> and <complexType> element
// information items in the [children]" of <schema> — top level only — where
// {identity-constraint definitions} beside it deliberately widens to "anywhere
// within the [children]", so the spec makes the narrowing for exactly one
// property and not this one. §3.17.1 agrees from the other side: it collects the
// components a QName can resolve to, and an anonymous type has no name to be
// resolved by, so registering it would key the by-name index on QName{} and let
// unrelated anonymous types collide. Registering would also fork the component
// in two: ComplexType is a value type, this slot holds its own copy, and
// finalize's attribute folds write back through the Schema's own type slice —
// so a registered copy would DIVERGE from the one a consumer reads here (STYLE
// D3), which is worse than the fold not running. It is reachable only through
// the owning declaration's TypeDefinition accessor; see Schema.Types.
//
// This slot does not populate the wrapped type's own {context} property, and it
// is not the place to: {context} is the BACK-pointer of this same edge, and the
// caller supplies it. The caller mints one xsd.ComponentID for the OWNER before
// either component exists and threads it into both — into the wrapped type
// through NewAnonymousComplexType's context argument, and into the owner through
// the one entry point that accepts a wrapped COMPLEX type and checks the two
// agree: NewElementDeclarationOwningType for a declaration (#340),
// NewComplexTypeOwningBase for a redefining complex type (#505). That property
// is §3.4.1 ctd-context; see ComplexTypeContext. A wrapped SIMPLE type's own
// {context} (§3.16.1 std-context) is a separate property and is still unmodeled
// here.
//
// Definition is always present and always ANONYMOUS (its Name() is the zero
// QName): a named type is reachable by name and so is always the
// TypeDefinitionRef arm. NewElementDeclaration, NewAttributeDeclaration and the
// ComplexType constructors reject both violations. The field is read-only by
// convention; do not mutate it after construction.
type InlineTypeDefinition struct{ Definition TypeDefinition }

// SubstitutionGroupHeadTypeRef is the variant carrying the {type definition} an
// element inherits BY CONSTRUCTION from the substitution-group head that OWNS an
// anonymous type: §3.3.2.1 dcl.elt.common clause 3, "the declared {type
// definition} of the Element Declaration ·resolved· to by the first QName in the
// ·actual value· of the substitutionGroup attribute", in the case where that
// declared type is the head's own inline <complexType> child.
//
// OWNERSHIP is why this is a third arm rather than either existing one.
// InlineTypeDefinition means the slot OWNS the type it holds; a member does not
// own the head's, and the owning copy is not merely undesirable but literally
// unconstructible — an anonymous complex type's {context} names exactly one
// Element Declaration (§3.4.2.1 dcl.ctd.common), and
// NewElementDeclarationOwningType rejects a definition whose {context} names
// another declaration. TypeDefinitionRef is unavailable for the opposite reason:
// an anonymous type has no {name} for a by-name lookup to find, which is exactly
// what {context} exists to substitute for (§3.4.1, Appendix G.1.11).
//
// §3.4.6.5's no-identity Note settles that this must be the SAME component and
// not a structurally equal copy: it lists "when an element's type definition
// defaults to being the same type definition as that of its substitution-group
// head" among the cases where component identity IS determined by this
// specification. This arm is how that identity is encoded — one component, one
// owner, one {context}, referenced from a second slot (STYLE D3).
//
// Head names an ELEMENT DECLARATION, a different symbol space from
// TypeDefinitionRef.Name: it is followed through Schema.Element, never
// Schema.Type. It is the TERMINAL head — the declaration that actually carries
// the anonymous type — which for a chain of affiliations may differ from this
// declaration's own substitutionGroup[0]. That is what makes ResolvedType's read
// DEPTH-1 by definition rather than by producer convention: the owner is the
// only component whose own {type definition} slot can hold the type, so
// following Head once always lands on it.
//
// Head is a PRESENT reference, never the absent (zero) QName, on the same
// footing as TypeDefinitionRef.Name. The arm is legal ONLY in an Element
// Declaration's {type definition} slot; see TypeDefinitionOrRef for the arm ×
// slot table and checkTypeDefinitionOrRef for its one implementation. The field
// is read-only by convention; do not mutate it after construction.
type SubstitutionGroupHeadTypeRef struct{ Head QName }

// typeDefinitionOrRef marks TypeDefinitionRef as a TypeDefinitionOrRef; see the
// TypeDefinitionOrRef doc.
func (TypeDefinitionRef) typeDefinitionOrRef() {}

// typeDefinitionOrRef marks InlineTypeDefinition as a TypeDefinitionOrRef; see
// the TypeDefinitionOrRef doc.
func (InlineTypeDefinition) typeDefinitionOrRef() {}

// typeDefinitionOrRef marks SubstitutionGroupHeadTypeRef as a
// TypeDefinitionOrRef; see the TypeDefinitionOrRef doc.
func (SubstitutionGroupHeadTypeRef) typeDefinitionOrRef() {}

// ResolvedType is the one way a TypeDefinitionOrRef slot becomes a component,
// exhaustively over the sum's three arms: an InlineTypeDefinition IS
// the component (it is in no by-name symbol table, so a lookup would miss it),
// a TypeDefinitionRef is the by-name Schema.Type lookup, and a
// SubstitutionGroupHeadTypeRef is a Schema.ELEMENT lookup followed by a single
// read of that head's own {type definition} slot. ok is false for an absent
// (nil) slot and for an unresolvable name — the cases every caller treats as
// "not decidable by this clause", never as a violation (a dangling name was
// already charged src-resolve by resolve.go's Phase A).
//
// It is exported for the instance validator, which reads an element
// declaration's {type definition} slot to reach the ·selected type definition·
// §3.3.4.6 makes the ·governing type definition· of a validation root (#714).
// A consumer outside this package must not re-derive the lookup: the three arms
// do not agree on where the component lives, and the head arm is not a name
// lookup at all.
//
// The head read is DEPTH-1 and carries no recursion and no visited set (STYLE
// D4): SubstitutionGroupHeadTypeRef.Head names the OWNER of the anonymous type,
// so the owner's own slot holds the component itself and never a second head
// reference. That is enforced rather than assumed — an owner-of-owner chain
// answers ok=false here, and Phase A rejects one outright
// (resolveTypeDefinition), so for any schema that survived finalize this branch
// is unreachable. A read-time accessor must not loop on a programmatically built
// graph that checkSubstitutionGroupsAcyclic has not yet seen, which is why the
// non-recursion is a rule of this function and not a caller's obligation.
//
// It answers for EVERY slot the sum serves — an element or attribute
// declaration's {type definition} and a complex type's {base type definition}
// alike — and deliberately takes the slot rather than the owning component, so
// it can never become arm-meaning-dependent on where the caller came from. Every
// consumer goes through this helper, through its complex-only narrowing
// baseComplexType (complexderivation.go), or through its narrowed sibling
// ResolvedSimpleType; none re-derives a bare-name lookup of its own (STYLE T4).
func (s *Schema) ResolvedType(ref TypeDefinitionOrRef) (TypeDefinition, bool) {
	switch r := ref.(type) {
	case nil:
		return nil, false
	case TypeDefinitionRef:
		return s.Type(r.Name)
	case InlineTypeDefinition:
		return r.Definition, true
	case SubstitutionGroupHeadTypeRef:
		head, ok := s.Element(r.Head)
		if !ok {
			return nil, false // an ·absent· head (§5.3); see resolveElementDecl
		}
		if _, chained := head.TypeDefinition().(SubstitutionGroupHeadTypeRef); chained {
			return nil, false // an owner-of-owner chain; Phase A rejects it
		}
		// The guard above leaves only the nil, by-name and inline arms, so this
		// re-entry cannot reach this case again: one hop, then the ordinary read.
		return s.ResolvedType(head.TypeDefinition())
	default:
		panic("xsd: ResolvedType: non-exhaustive TypeDefinitionOrRef switch")
	}
}

// typeDefinitionSlot names WHICH of the three properties a TypeDefinitionOrRef
// is filling. It exists because the sum's arms are not uniformly legal across
// them — §3.3.2.1 dcl.elt.common clause 3 has no analog in §3.2.2.2 or §3.4.2,
// so SubstitutionGroupHeadTypeRef belongs to the element slot alone — and the
// arm × slot legality table must be TOTAL and live in ONE place, so that a
// FOURTH slot added later has to answer for every arm rather than silently
// admitting one by omission.
//
// It also supplies the property label the rejection messages carry, which used
// to be passed in as a string literal by each caller (STYLE T1: a closed set is
// a type, not a string).
//
// The zero value is invalid, mirroring the exported closed sets in
// closedsets.go; the constants start at iota+1.
type typeDefinitionSlot uint8

const (
	// elementTypeSlot is an Element Declaration's {type definition} (§3.3.1).
	elementTypeSlot typeDefinitionSlot = iota + 1
	// attributeTypeSlot is an Attribute Declaration's {type definition} (§3.2.1).
	attributeTypeSlot
	// baseTypeSlot is a Complex Type Definition's {base type definition}
	// (§3.4.1).
	baseTypeSlot
)

// property is the §3 property name this slot fills, as it appears in a
// rejection message.
func (s typeDefinitionSlot) property() string {
	switch s {
	case elementTypeSlot, attributeTypeSlot:
		return "{type definition}"
	case baseTypeSlot:
		return "{base type definition}"
	default:
		panic("xsd: typeDefinitionSlot.property: unknown slot")
	}
}

// admitsHeadInherited reports whether this slot may hold a
// SubstitutionGroupHeadTypeRef. Only the element slot may: §3.3.2.1
// dcl.elt.common clause 3 is the sole mapping rule that yields one, and neither
// §3.2.2.2 dcl.att.local (three tiers, no clause-3 analog) nor §3.4.2's base=
// has anything like it.
func (s typeDefinitionSlot) admitsHeadInherited() bool { return s == elementTypeSlot }

// checkTypeDefinitionOrRef rejects the encodings of a TypeDefinitionOrRef slot
// that the sum's doc declares illegal, charged to xsderr.RuleComponentInvariant:
// these are representation invariants this package owns, not spec clauses a
// schema author can violate, which is the same footing NewAttributeUse rejects a
// zero-named AttributeDeclarationRef on. A nil ref is the legal encoding of an
// absent slot and passes.
//
// It decides two things at once, and deliberately in one place: whether the ARM
// is well formed, and whether that arm is legal in THIS slot. owner names the
// owning component — "element declaration {urn:x}e", "complex type {urn:x}T" —
// and slot supplies the property name that completes the phrase, so no caller
// spells "{type definition}" as a literal: hard-coding it made every message lie
// the moment {base type definition} adopted the sum (#505).
func checkTypeDefinitionOrRef(loc xsderr.Loc, ref TypeDefinitionOrRef, slot typeDefinitionSlot, owner string) error {
	ctx := owner + " " + slot.property()
	switch r := ref.(type) {
	case nil:
		return nil
	case TypeDefinitionRef:
		if r.Name == (QName{}) {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s is a TypeDefinitionRef carrying the absent (zero) QName, but that variant names a type definition reachable by name; an absent slot is the nil one and an anonymous type is InlineTypeDefinition", ctx)
		}
		return nil
	case InlineTypeDefinition:
		if r.Definition == nil {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s is an InlineTypeDefinition with no definition, but that variant owns the anonymous type definition outright; an absent slot is the nil one", ctx)
		}
		if r.Definition.Name() != (QName{}) {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s is an InlineTypeDefinition wrapping the NAMED type %s, but a named type definition is reachable by name and so is always the TypeDefinitionRef variant", ctx, r.Definition.Name())
		}
		return nil
	case SubstitutionGroupHeadTypeRef:
		if !slot.admitsHeadInherited() {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s is a SubstitutionGroupHeadTypeRef naming the substitution group head %s, but that variant encodes §3.3.2.1 dcl.elt.common clause 3, which only an ELEMENT declaration's {type definition} has; this property has no clause-3 analog", ctx, r.Head)
		}
		if r.Head == (QName{}) {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s is a SubstitutionGroupHeadTypeRef carrying the absent (zero) QName, but that variant names the element declaration that OWNS the inherited anonymous type; an absent slot is the nil one", ctx)
		}
		return nil
	default:
		panic("xsd: checkTypeDefinitionOrRef: non-exhaustive TypeDefinitionOrRef switch")
	}
}
