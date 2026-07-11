package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleICProps is Identity-Constraint Definition Properties Correct
// (Structures §3.11.6.1, id="c-props-correct"): an identity-constraint
// definition's properties must match the §3.11.1 tableau. This constructor
// enforces clause 1 (a {name}/{identity-constraint category}/{selector}/
// {fields} shape with at least one field, and a {referenced key} present
// exactly when the category is "keyref").
//
// Clause 2 — the cross-component check that a "keyref"'s {fields}
// cardinality matches its resolved {referenced key}'s {fields} — is a
// finalize-phase constraint that needs the resolved {referenced key}
// component, which this package does not resolve yet; it is deferred to the
// later schema-assembly issue that introduces phased construction (doc.go's
// "parse → resolve → finalize") and is NOT enforced here.
const ruleICProps xsderr.Rule = "c-props-correct"

// IdentityConstraint is the Identity-Constraint Definition component
// (Structures §3.11.1, id="icd"): a kind of Annotated Component with {name}
// (bundled with {target namespace} as an xsd.QName, per this package's
// "Names are expanded QNames" convention — doc.go), {identity-constraint
// category} ("key"/"keyref"/"unique"), {selector} (an XPath Expression
// property record), {fields} (a non-empty sequence of XPath Expression
// property records), {referenced key} (present only for "keyref"), and
// {annotations}.
//
// {selector} and {fields} reuse xsd.XPathExpression verbatim: §3.13.1
// (id="x") defines the XPath Expression property record once, and §3.11.2's
// XML mapping reuses it by reference for both the <selector> and <field>
// xpath attributes — exactly as Assertion's {test} does. Like Assertion,
// IdentityConstraint is a STRUCTURAL, opaque holder: the selector/field
// XPaths are preserved verbatim (see XPathExpression's doc), never compiled
// or evaluated here. Evaluation (c-selector-xpath, c-fields-xpaths,
// cvc-identity-constraint) is deferred to the M6+ XPath engine and is out of
// scope.
//
// Construct only through NewIdentityConstraint, which rejects the states
// c-props-correct clause 1 (§3.11.6.1) forbids so they are unrepresentable
// (STYLE T1). IdentityConstraint is immutable after construction.
type IdentityConstraint struct {
	name          QName
	category      IdentityConstraintCategory
	selector      XPathExpression
	fields        []XPathExpression
	referencedKey QName // zero value when category != IdentityConstraintKeyref
	annotations   []Annotation
}

// NewIdentityConstraint builds an IdentityConstraint, rejecting the states
// Identity-Constraint Definition Properties Correct clause 1 (§3.11.6.1,
// c-props-correct) forbids: an unknown {identity-constraint category}, an
// empty {fields}, and a {referenced key} whose presence disagrees with the
// category (present iff the category is "keyref"). referencedKey is a pointer
// so that its absence (nil) is distinct from a present zero/absent QName.
// fields and annotations are copied; the caller's backing arrays are not
// aliased.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built definition — may
// legitimately pass the zero xsderr.Loc{}.
func NewIdentityConstraint(loc xsderr.Loc, name QName, category IdentityConstraintCategory, selector XPathExpression, fields []XPathExpression, referencedKey *QName, annotations []Annotation) (IdentityConstraint, error) {
	switch category {
	case IdentityConstraintKey, IdentityConstraintKeyref, IdentityConstraintUnique:
	default:
		return IdentityConstraint{}, xsderr.New(ruleICProps, loc,
			"identity-constraint definition has an unknown {identity-constraint category}: %s", category)
	}
	if len(fields) == 0 {
		return IdentityConstraint{}, xsderr.New(ruleICProps, loc,
			"identity-constraint definition must have at least one {field}")
	}
	if (referencedKey != nil) != (category == IdentityConstraintKeyref) {
		return IdentityConstraint{}, xsderr.New(ruleICProps, loc,
			"identity-constraint definition has a {referenced key} if and only if its {identity-constraint category} is keyref")
	}
	ic := IdentityConstraint{
		name:     name,
		category: category,
		selector: selector,
		fields:   append([]XPathExpression(nil), fields...),
	}
	if referencedKey != nil {
		ic.referencedKey = *referencedKey
	}
	if len(annotations) > 0 {
		ic.annotations = append([]Annotation(nil), annotations...)
	}
	return ic, nil
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
func (c IdentityConstraint) Name() QName {
	return c.name
}

// Category returns the {identity-constraint category} property.
func (c IdentityConstraint) Category() IdentityConstraintCategory {
	return c.category
}

// Selector returns the {selector} property: the Required XPathExpression that
// selects the target node set (evaluated once the M6+ engine exists).
func (c IdentityConstraint) Selector() XPathExpression {
	return c.selector
}

// Fields returns the {fields} property in document order. It returns a copy:
// mutating the result does not affect c. Construction guarantees at least one
// field, so the result is never nil or empty.
func (c IdentityConstraint) Fields() []XPathExpression {
	return append([]XPathExpression(nil), c.fields...)
}

// ReferencedKeyName returns the pre-resolution refer QName — the input from
// §3.11.2's refer attribute — and whether it is present (true exactly when
// the category is "keyref"); when false the first result is not meaningful.
//
// This is NOT the resolved {referenced key} component (§3.11.1). The resolved
// component pointer, and c-props-correct clause 2 (the {fields} cardinality
// match), are deferred to the future schema-assembly/finalize-phase issue
// that first introduces phased construction (per doc.go's "parse → resolve →
// finalize"); nothing in this package resolves it yet.
func (c IdentityConstraint) ReferencedKeyName() (QName, bool) {
	return c.referencedKey, c.category == IdentityConstraintKeyref
}

// Annotations returns the {annotations} property in document order. It
// returns a copy: mutating the result does not affect c. An empty
// {annotations} yields nil.
func (c IdentityConstraint) Annotations() []Annotation {
	if len(c.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), c.annotations...)
}
