package xsd

import "slices"

// This file decides ONE relation, which two derivation constraints state in
// byte-for-byte identical words: "B.{assertions} is a prefix of T.{assertions}"
// — derivation-ok-restriction (§3.4.6.3) clause 5 and cos-ct-extends (§3.4.6.2)
// case 1 clause 1.7. One relation, one encoding (STYLE T4); the two callers
// differ only in the rule they charge and the sentence they charge it with.
//
// The relation holds BY CONSTRUCTION for a complex type mapped from a schema
// document: §3.4.2.1 (dcl.ctd.common) clause 1 makes a type's {assertions} the
// {base type definition}'s own followed by clause 2's <assert> children, and
// parser/produce_xpath.go's assertionsWithBase performs exactly that fold on
// every derivation alternant. Both clauses are nonetheless CHARGED rather than
// assumed, on the footing checkExtensionAttributeUses records for clause 1.2:
// they are the executable statement of the constraint for a component the
// SchemaBuilder assembled by other means, and the guard that would turn a fold
// which stopped carrying the base's members forward into a verdict rather than
// into silence.
//
// cos-ct-extends has no counterpart clause under case 2 (a simple {base type
// definition}) and needs none: a Simple Type Definition has no {assertions}
// property at all, so there is nothing for a prefix to be taken of.

// assertionsPrefix decides whether base is a prefix of derived: no longer, and
// identical member for member from position 0 up.
func assertionsPrefix(base, derived []Assertion) bool {
	if len(base) > len(derived) {
		return false // a longer list is no prefix
	}
	for i, a := range base {
		if !assertionsIdentical(derived[i], a) {
			return false
		}
	}
	return true
}

// assertionsIdentical decides property identity between two Assertions
// (§3.13.1) by their one schema-significant property, {test}, an XPathExpression
// property record compared field for field. {annotations} is not compared — it
// carries no schema-significant content, the same omission attributeUsesIdentical
// makes for the same reason.
func assertionsIdentical(a, b Assertion) bool {
	return xpathExpressionsIdentical(a.test, b.test)
}

// xpathExpressionsIdentical compares two XPath Expression property records
// (§3.13.1) field for field: {expression} verbatim (it is carried as text and
// never parsed here, so textual equality is the only reading available),
// {default namespace} and {base URI} including their presence, and {namespace
// bindings}.
func xpathExpressionsIdentical(a, b XPathExpression) bool {
	if a.expression != b.expression {
		return false
	}
	if !optionalStringsEqual(a.defaultNamespace, a.hasDefaultNamespace, b.defaultNamespace, b.hasDefaultNamespace) {
		return false
	}
	if !optionalStringsEqual(a.baseURI, a.hasBaseURI, b.baseURI, b.hasBaseURI) {
		return false
	}
	return namespaceBindingsIdentical(a.namespaceBindings, b.namespaceBindings)
}

// namespaceBindingsIdentical compares two {namespace bindings} properties as the
// SETS §3.13.1 defines them to be: equal cardinality, and every member of one a
// member of the other.
//
// Position is deliberately NOT compared, though the property is modeled as an
// ordered slice: that order is a determinism device with no spec weight
// (xpathexpression.go), so an order-sensitive comparison could call two equal
// sets different — and both callers read "not identical" as "reject", making
// that direction a FALSE REJECT, the one direction PRINCIPLES 9 forbids. The
// residual looseness (a hypothetical multiset with repeats compares equal to a
// set of the same cardinality) runs the other way, which is fail-open.
func namespaceBindingsIdentical(a, b []NamespaceBinding) bool {
	if len(a) != len(b) {
		return false
	}
	for _, binding := range a {
		if !slices.Contains(b, binding) {
			return false
		}
	}
	return true
}
