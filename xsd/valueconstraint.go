package xsd

// ValueConstraint is the {value constraint} property record (Structures
// §3.2.1 vc_a / §3.3.1 vc_e / §3.5.1 vc_au): a {variety} (ValueConstraintKind,
// default or fixed) paired with {lexical form} — the normalized lexical
// string — and the namespace context in force where that lexical form was
// written. It deliberately omits the record's third field, {value} (the
// ·actual value·): parsing it requires a value.Mapping, and xsd is a pure
// leaf (see doc.go) that cannot import package value; which type's mapping
// governs parsing (declaring type vs. nearest mapped ancestor) is also a
// decision only a value-aware consumer can make, so baking a parsed value in
// here would freeze that decision at the wrong layer. Follows Facet.Values's
// "normalized lexical strings" convention.
//
// The namespace context is load-bearing for a QName- or NOTATION-governed
// constraint, whose lexical→value mapping resolves the literal's prefix
// against the in-scope namespace bindings AT THE LITERAL (Datatypes §3.3.18,
// adopted verbatim by §3.3.19, PRINCIPLES 19): "a:x" and "b:x" denote the same
// {value} when both prefixes name one namespace, and "p:x" denotes different
// {value}s in two documents that bind "p" differently. §3.2.6.2
// cos-valid-simple-default clause 2 fixes that context at the moment the
// {lexical form} is validated, so it is a property OF the literal and travels
// WITH it — including through §3.4.2.4 clause 3's fold, which copies attribute
// uses and element declarations wholesale. This package carries the context
// opaquely (like EnumerationMember and XPathExpression) and defers prefix
// resolution to a value-space consumer.
//
// The zero value is NOT a valid constraint (its {variety} is the invalid
// zero ValueConstraintKind) — unlike QName, whose zero IS a legitimate
// ·absent·. Construct only through NewValueConstraint. A future consumer
// (M4 element/attribute/attribute-use declarations) models an absent value
// constraint via (ValueConstraint, bool) or *ValueConstraint, never a zero
// ValueConstraint. Immutable after construction.
//
// Holding a slice, a ValueConstraint is not comparable with ==; in-package
// property comparison goes through sameRecord.
type ValueConstraint struct {
	kind                ValueConstraintKind
	lexicalForm         string
	namespaceBindings   []NamespaceBinding
	defaultNamespace    string
	hasDefaultNamespace bool
}

// NewValueConstraint builds a ValueConstraint pairing kind with lexicalForm
// — the {lexical form} property, i.e. the normalized lexical string (see
// Facet.Values's identical convention), never the raw unprocessed
// schema-document string — and the namespace context in scope where that
// lexical form was written.
//
// namespaceBindings is copied in document order; the caller's backing array is
// not aliased. A nil defaultNamespace means the {default namespace} is absent;
// a non-nil pointer (including to "") means it is present, because "" is a
// legal anyURI and cannot double as an absence sentinel (mirrors
// NewEnumerationMember's and NewXPathExpression's discipline).
//
// A producer that captures no context yields a constraint whose unprefixed
// literals resolve to no namespace and whose prefixed ones resolve to nothing
// — so every producer of a QName/NOTATION-governed constraint must pass the
// bindings in scope at its schema-document element, which is what this
// signature exists to force.
func NewValueConstraint(kind ValueConstraintKind, lexicalForm string, namespaceBindings []NamespaceBinding, defaultNamespace *string) ValueConstraint {
	v := ValueConstraint{kind: kind, lexicalForm: lexicalForm}
	if len(namespaceBindings) > 0 {
		v.namespaceBindings = append([]NamespaceBinding(nil), namespaceBindings...)
	}
	if defaultNamespace != nil {
		v.defaultNamespace, v.hasDefaultNamespace = *defaultNamespace, true
	}
	return v
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

// NamespaceBindings returns the constraint's namespace context in document
// order: the prefixed in-scope namespace bindings at the schema-document
// element carrying its default=/fixed= attribute (§3.3.18). It returns a copy:
// mutating the result does not affect v. An empty context yields nil.
func (v ValueConstraint) NamespaceBindings() []NamespaceBinding {
	if len(v.namespaceBindings) == 0 {
		return nil
	}
	return append([]NamespaceBinding(nil), v.namespaceBindings...)
}

// DefaultNamespace returns the constraint's {default namespace} — the
// namespace an unprefixed QName or NOTATION literal binds to in scope where
// the {lexical form} was written. The second result is false when it is absent
// (no default namespace in scope), in which case the first result is not
// meaningful.
func (v ValueConstraint) DefaultNamespace() (string, bool) {
	return v.defaultNamespace, v.hasDefaultNamespace
}

// sameRecord reports whether v and o carry the same {variety} and {lexical
// form}, standing in for the == the property comparisons in
// complexextension.go used before this record held a slice. It compares those
// two fields ONLY: the namespace context is deliberately excluded, because
// those comparisons read "identical" as "the clause is satisfied" and every
// omission makes the relation looser. Folding the bindings in would instead
// TIGHTEN it and false-reject a legal extension whose copy reaches the same
// namespace through a different prefix.
func (v ValueConstraint) sameRecord(o ValueConstraint) bool {
	return v.kind == o.kind && v.lexicalForm == o.lexicalForm
}
