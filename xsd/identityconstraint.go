package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleICProps is Identity-Constraint Definition Properties Correct
// (Structures §3.11.6.1, id="c-props-correct"): an identity-constraint
// definition's properties must match the §3.11.1 tableau. This constructor
// enforces clause 1 (a present {name}, a legal {identity-constraint category},
// a {fields} with at least one field, and a {referenced key} present exactly
// when the category is "keyref").
//
// This constructor enforces only clause 1's LOCAL part (the presence-iff-keyref
// shape). Everything that needs the RESOLVED {referenced key} is enforced at
// finalize instead, by resolve.go's resolveKeyref: the rest of clause 1 (the
// referenced name must resolve, and must resolve to a key/unique rather than
// another keyref) and clause 2 (a keyref's {fields} cardinality must equal its
// {referenced key}'s). Neither is enforced in this constructor.
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
// or evaluated here. cvc-identity-constraint (§3.11.4), the Validation Rule
// that evaluates {selector}/{fields} against an instance, is implemented in
// validate (validate/cvcidentityconstraint.go, validate/icpath.go), landed by
// #718. Still unimplemented are c-selector-xpath (§3.11.6.2) and
// c-fields-xpaths (§3.11.6.3), the Schema Component Constraints that check
// {selector}/{fields}'s own {expression} against the restricted selector/field
// path grammar those sections define — a subset of the path axes, not XPath
// 2.0 — owned by #812.
//
// Construct only through NewIdentityConstraint, which rejects the states
// c-props-correct clause 1 (§3.11.6.1) forbids so they are unrepresentable
// (STYLE T1). IdentityConstraint is immutable after construction.
type IdentityConstraint struct {
	loc           xsderr.Loc // source position; provenance, not a §3.11.1 property
	name          QName
	category      IdentityConstraintCategory
	selector      XPathExpression
	fields        []XPathExpression
	referencedKey QName // zero value when category != IdentityConstraintKeyref
	annotations   []Annotation
}

// NewIdentityConstraint builds an IdentityConstraint, rejecting the states
// Identity-Constraint Definition Properties Correct clause 1 (§3.11.6.1,
// c-props-correct) forbids: an absent {name}, an unknown {identity-constraint
// category}, an empty {fields}, and a {referenced key} whose presence disagrees
// with the category (present iff the category is "keyref").
//
// name must be present: its local part may not be empty. The §3.11.1 tableau
// types {name} as a Required xs:NCName, and NCName's value space (Datatypes
// §3.4.7, pattern \i\c*) excludes the empty string, so a zero-Local QName is
// categorically not a legal {name}. The §5.3 Missing Sub-components escape hatch
// does not cover it: §5.3 is scoped to properties whose value is another
// component reached by QName ·resolution·, and {name} is the identity other
// components resolve AGAINST — here quite literally, since a keyref's
// {referenced key} names a key/unique by exactly this QName. The guard is
// unconditional because an identity-constraint definition has NO anonymous form:
// §3.11.2 requires name on <key>, <unique> and <keyref> alike. That reasoning is
// deliberately NOT generalized to NewComplexType / NewSimpleType, whose
// components have a genuine anonymous form ({name} Optional). Testing the local
// part, not name == QName{}, is deliberate: the latter would admit
// QName{Space: "urn:x", Local: ""} as a named definition. Same idiom as
// NewElementDeclaration's e-props-correct clause 1 check.
//
// referencedKey is a pointer so that its absence (nil) is distinct from a
// present zero/absent QName. fields and annotations are copied; the caller's
// backing arrays are not aliased.
//
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the definition's provenance. Pass the position of this
// definition's own declaring element, never a convenient nearby one (a parent
// element's, say) — it is observable, not merely an error-charging convenience.
// A caller with no real parser position — a synthesized or programmatically
// built definition — passes the zero xsderr.Loc{}, which reads as "unknown".
func NewIdentityConstraint(loc xsderr.Loc, name QName, category IdentityConstraintCategory, selector XPathExpression, fields []XPathExpression, referencedKey *QName, annotations []Annotation) (IdentityConstraint, error) {
	if name.Local == "" {
		return IdentityConstraint{}, xsderr.New(ruleICProps, loc,
			"identity-constraint definition has an absent {name}, but the §3.11.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (c-props-correct clause 1)")
	}
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
		loc:      loc,
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

// Loc reports the source position of the declaring element — provenance, not a
// §3.11.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown.
func (c IdentityConstraint) Loc() xsderr.Loc {
	return c.loc
}

// Category returns the {identity-constraint category} property.
func (c IdentityConstraint) Category() IdentityConstraintCategory {
	return c.category
}

// Selector returns the {selector} property: the Required XPathExpression that
// selects the target node set (evaluated directly by validate against the
// restricted selector subset, not through the xpath package — see this
// package's doc comment above).
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
// This is NOT the resolved {referenced key} component (§3.11.1). Finalize
// validates that the name resolves to an identity-constraint definition
// (src-resolve clause 1.7) that is a key or unique, not another keyref
// (c-props-correct clause 1), and that this constraint's {fields} cardinality
// equals that resolved target's (c-props-correct clause 2). It adds no
// resolved-component accessor: the QName is retained, and a consumer follows it
// by a read-time lookup.
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
