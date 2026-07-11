package xsd

// ValueConstraint is the {value constraint} property record (Structures
// §3.2.1 vc_a / §3.3.1 vc_e / §3.5.1 vc_au): a {variety} (ValueConstraintKind,
// default or fixed) paired with {lexical form} — the normalized lexical
// string. It deliberately omits the record's third field, {value} (the
// ·actual value·): parsing it requires a value.Mapping, and xsd is a pure
// leaf (see doc.go) that cannot import package value; which type's mapping
// governs parsing (declaring type vs. nearest mapped ancestor) is also a
// decision only a value-aware consumer can make, so baking a parsed value in
// here would freeze that decision at the wrong layer. Follows Facet.Values's
// "normalized lexical strings" convention.
//
// The zero value is NOT a valid constraint (its {variety} is the invalid
// zero ValueConstraintKind) — unlike QName, whose zero IS a legitimate
// ·absent·. Construct only through NewValueConstraint. A future consumer
// (M4 element/attribute/attribute-use declarations) models an absent value
// constraint via (ValueConstraint, bool) or *ValueConstraint, never a zero
// ValueConstraint. Immutable after construction.
type ValueConstraint struct {
	kind        ValueConstraintKind
	lexicalForm string
}

// NewValueConstraint builds a ValueConstraint pairing kind with lexicalForm
// — the {lexical form} property, i.e. the normalized lexical string (see
// Facet.Values's identical convention), never the raw unprocessed
// schema-document string.
func NewValueConstraint(kind ValueConstraintKind, lexicalForm string) ValueConstraint {
	return ValueConstraint{kind: kind, lexicalForm: lexicalForm}
}

// Kind returns the {variety} property.
func (v ValueConstraint) Kind() ValueConstraintKind {
	return v.kind
}

// LexicalForm returns the {lexical form} property: the normalized lexical
// string. It deliberately does NOT return {value} (the ·actual value·) —
// see the type doc.
func (v ValueConstraint) LexicalForm() string {
	return v.lexicalForm
}
