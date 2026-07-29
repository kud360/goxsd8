package xsd

import "github.com/kud360/goxsd8/xsderr"

// TypeDefinition is the sealed sum of the two kinds that populate a schema's
// {type definitions} property (Structures §3.17.1): a ComplexType (§3.4.1) or
// a *SimpleType (§3.16.1 / Datatypes §4.1.1). §3.17.6.2 clause 1.1 unifies both
// into one lookup bucket — a shared name between a simple and a complex type in
// one namespace is exactly the sch-props-correct (§3.17.6.1 clause 2)
// collision, so one sum keyed once makes the two-map illegal state
// unrepresentable (STYLE T7). The unexported typeDefinition marker method seals
// it (STYLE T2/T7, the PRINCIPLES 7 sealed-sum exception): consumers
// exhaustively switch these two variants and no third is representable,
// mirroring term.go's Term and complextype.go's ContentType sealed sums.
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
	// Loc is the type definition's source position — provenance, not a §3.17.1
	// component property (see the package doc's Components section). Both
	// variants already expose it (ComplexType.Loc, (*SimpleType).Loc), so it is
	// promoted into the sum to let the one {type definitions} bucket cite a
	// position without a type switch (STYLE T7).
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
