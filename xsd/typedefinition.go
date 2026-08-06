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

// TypeDefinitionOrRef is the {type definition} slot of an Element Declaration
// (§3.3.1) or an Attribute Declaration (§3.2.1). It is a sealed sum (STYLE
// T2/T7): TypeDefinitionRef and InlineTypeDefinition are its only
// implementations, sealed by the unexported typeDefinitionOrRef method, so
// consumers exhaustively switch the two branches and no third variant is
// representable. It copies attributeuse.go's AttributeDeclarationOrRef
// precedent exactly, and for the same reason: the two XML mappings that
// populate the slot differ fundamentally in OWNERSHIP of the type definition
// they yield.
//
//   - TypeDefinitionRef covers §3.3.2.1 dcl.elt.common clauses 2-4 and §3.2.2.2
//     dcl.att.local's type= and xs:anySimpleType tiers: the type is a top-level
//     component reachable BY NAME, possibly forward-referenced, so only a
//     deferred QName is available at parse time.
//   - InlineTypeDefinition covers §3.3.2.1 clause 1 and §3.2.2.2's first tier:
//     the anonymous type corresponding to an inline <simpleType>/<complexType>
//     child, built in the same producer call, in no symbol table, so this slot
//     is its SOLE owner.
//
// A nil TypeDefinitionOrRef is the single encoding of an ABSENT {type
// definition} — the state a programmatically built declaration is in before the
// §3.3.2.1/§3.2.2.2 defaulting tiers are applied. It replaces the zero QName
// that used to carry that meaning, which was ambiguous: the same QName{} also
// had to stand for "anonymous type" (STYLE D3, one fact one encoding).
// Anonymity is now DERIVED from which arm the slot holds, never separately
// stored: an InlineTypeDefinition is anonymous by construction, a
// TypeDefinitionRef never is.
type TypeDefinitionOrRef interface{ typeDefinitionOrRef() }

// TypeDefinitionRef is the {type definition} variant naming a top-level type
// definition: the type/@type of §3.3.2/§3.2.2, the {type definition} an element
// inherits from its substitution-group head (§3.3.2.1 clause 3), or the
// xs:anyType / xs:anySimpleType default (§3.3.2.1 clause 4, §3.2.2.2). All four
// are reachable by name, so all four are this arm. The field is read-only by
// convention; do not mutate it after construction.
//
// Name is a PRESENT reference, never the absent (zero) QName: a reference that
// names nothing could not be followed, so it is not a resolution failure to
// defer to finalize but an illegal representation. NewElementDeclaration and
// NewAttributeDeclaration reject one, exactly as NewAttributeUse rejects a
// zero-named AttributeDeclarationRef.
type TypeDefinitionRef struct{ Name QName }

// InlineTypeDefinition is the {type definition} variant owning the anonymous
// type definition corresponding to an inline <simpleType>/<complexType> child
// (§3.3.2.1 dcl.elt.common clause 1, §3.2.2.2 dcl.att.local's first tier),
// carried by value because the declaration is its sole owner.
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
// caller supplies it. The caller mints one xsd.ComponentID for the declaration
// before either component exists and threads it into both — into the wrapped
// type through NewAnonymousComplexType's context argument, and into the
// declaration through NewElementDeclarationOwningType, which is the only entry
// point that accepts a wrapped COMPLEX type and checks the two agree (#340).
// That property is §3.4.1 ctd-context; see ComplexTypeContext. A wrapped SIMPLE
// type's own {context} (§3.16.1 std-context) is a separate property and is still
// unmodeled here.
//
// Definition is always present and always ANONYMOUS (its Name() is the zero
// QName): a named type is reachable by name and so is always the
// TypeDefinitionRef arm. NewElementDeclaration and NewAttributeDeclaration
// reject both violations. The field is read-only by convention; do not mutate
// it after construction.
type InlineTypeDefinition struct{ Definition TypeDefinition }

// typeDefinitionOrRef marks TypeDefinitionRef as a TypeDefinitionOrRef; see the
// TypeDefinitionOrRef doc.
func (TypeDefinitionRef) typeDefinitionOrRef() {}

// typeDefinitionOrRef marks InlineTypeDefinition as a TypeDefinitionOrRef; see
// the TypeDefinitionOrRef doc.
func (InlineTypeDefinition) typeDefinitionOrRef() {}

// checkTypeDefinitionOrRef rejects the encodings of a {type definition} slot
// that TypeDefinitionOrRef's doc declares illegal, charged to
// xsderr.RuleComponentInvariant: these are representation invariants this
// package owns, not spec clauses a schema author can violate, which is the same
// footing NewAttributeUse rejects a zero-named AttributeDeclarationRef on. ctx
// names the declaring component kind for the message. A nil ref is the legal
// encoding of an absent {type definition} and passes.
func checkTypeDefinitionOrRef(loc xsderr.Loc, ref TypeDefinitionOrRef, ctx string) error {
	switch r := ref.(type) {
	case nil:
		return nil
	case TypeDefinitionRef:
		if r.Name == (QName{}) {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s {type definition} is a TypeDefinitionRef carrying the absent (zero) QName, but that variant names a type definition reachable by name; an absent {type definition} is the nil slot and an anonymous one is InlineTypeDefinition", ctx)
		}
		return nil
	case InlineTypeDefinition:
		if r.Definition == nil {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s {type definition} is an InlineTypeDefinition with no definition, but that variant owns the anonymous type definition outright; an absent {type definition} is the nil slot", ctx)
		}
		if r.Definition.Name() != (QName{}) {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s {type definition} is an InlineTypeDefinition wrapping the NAMED type %s, but a named type definition is reachable by name and so is always the TypeDefinitionRef variant", ctx, r.Definition.Name())
		}
		return nil
	default:
		panic("xsd: checkTypeDefinitionOrRef: non-exhaustive TypeDefinitionOrRef switch")
	}
}
