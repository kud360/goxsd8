package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleAgPropsCorrect is Attribute Group Definition Properties Correct
// (Structures §3.6.6, id="ag-props-correct"): an attribute group definition's
// properties must match the §3.6.1 property tableau. This file enforces:
//
//   - clause 1 (tableau shape): {name} is present. The rest is satisfied by
//     construction — the sum and optional-slot machinery already make an
//     ill-formed {attribute uses} member or {attribute wildcard}
//     unrepresentable, so no extra check is needed there.
//   - clause 2: no two {attribute uses} members have {attribute declaration}s
//     with the same expanded name. This is cheaply computable now — BOTH sum
//     variants expose a QName WITHOUT resolution (a local declaration's own
//     Name(), or a ref's Name directly) — so it is enforced structurally here,
//     paralleling NewTypeTable enforcing e-props-correct clause 6.
//
// The §3.6.2.2 aspect of {attribute uses} — the union that folds in the
// {attribute uses} of referenced <attributeGroup>s — is resolved at PRODUCER
// mapping time, not finalize: §3.6.2.1 inlines each <attributeGroup ref> when the
// group is mapped, so the producer hands this constructor the already-computed
// union (matching xsd/resolve.go's "inlined at producer mapping time with no
// persistent ref component"). This constructor therefore validates the complete
// members it is given, not just a container's direct <attribute> children.
const ruleAgPropsCorrect xsderr.Rule = "ag-props-correct"

// AttributeGroupDefinition is the Attribute Group Definition component
// (Structures §3.6.1, id="agd"): a kind of Annotated Component with {name}
// (bundled with {target namespace} as an xsd.QName per this package's "Names are
// expanded QNames" convention — doc.go), {attribute uses}, {attribute wildcard}
// (Optional), and {annotations}.
//
// {attribute uses} is spec-worded a SET of Attribute Use components (§3.6.1
// tableau, §3.6.2.1 "union of ... sets"); this package represents it as a
// document-order slice per its standing convention (determinism, STYLE D2/D3) —
// the order carries no spec significance. ag-props-correct clause 2 forbids two
// members whose {attribute declaration}s share an expanded name, which
// NewAttributeGroupDefinition enforces.
//
// Construct only through NewAttributeGroupDefinition, which rejects the states
// ag-props-correct (§3.6.6) clauses 1 and 2 forbid so they are unrepresentable
// (STYLE T1). AttributeGroupDefinition is immutable after construction.
type AttributeGroupDefinition struct {
	loc           xsderr.Loc // source position; provenance, not a §3.6.1 property
	name          QName
	attributeUses []AttributeUse
	wildcard      Wildcard
	hasWildcard   bool
	annotations   []Annotation
}

// NewAttributeGroupDefinition builds an AttributeGroupDefinition, rejecting the
// state Attribute Group Definition Properties Correct (§3.6.6, ag-props-correct)
// clause 1 forbids for the {name} slot — name must be present, its local part
// may not be empty. The §3.6.1 tableau types {name} as a Required xs:NCName, and
// NCName's value space (Datatypes §3.4.7, pattern \i\c*) excludes the empty
// string, so a zero-Local QName is categorically not a legal {name}. The §5.3
// Missing Sub-components escape hatch does not cover it: §5.3 is scoped to
// properties whose value is another component reached by QName ·resolution·, and
// {name} is the identity other components resolve AGAINST. The guard is
// unconditional because an attribute group definition has NO anonymous form:
// per §3.6.2.1 an <attributeGroup> maps to this component only as a child of
// <schema>/<redefine>, where name is required; the ref usage under
// <complexType>/<attributeGroup> "does not correspond to any component as such".
// That reasoning is deliberately NOT generalized to NewComplexType /
// NewSimpleType, whose components have a genuine anonymous form ({name}
// Optional). Testing the local part, not name == QName{}, is deliberate: the
// latter would admit QName{Space: "urn:x", Local: ""} as a named definition.
// Same idiom as NewElementDeclaration's e-props-correct clause 1 check.
//
// It also rejects the state clause 2 forbids: two {attribute uses} members whose
// {attribute declaration}s have the same expanded name. The scan is
// deterministic (STYLE D2) — the members are walked in document order and
// membership is tested against a map[QName]struct{} seen-set, so the first
// duplicate found by index is the one rejected (never ranging the map itself for
// the scan). Both sum variants expose the expanded name without resolution: a
// LocalAttributeDeclaration via its Declaration.Name(), an
// AttributeDeclarationRef via its Name.
//
// The REST of clause 1 is satisfied by construction (the sum and optional-slot
// machinery already make ill-formed members unrepresentable); the §3.6.2.2
// referenced-group union is folded in by the producer (§3.6.2.1, mapping time)
// before it calls this constructor, so the members passed here are already
// complete.
//
// It also rejects one state Wildcard Properties Correct (§3.10.6.1,
// w-props-correct) clause 5 forbids, charged to that rule rather than to
// ag-props-correct: an {attribute wildcard} whose {namespace constraint}
// carries the sibling keyword. This slot is one of the two places an attribute
// wildcard is identifiable as such; see rejectSiblingOnAttributeWildcard.
//
// attributeUses and annotations are copied; the caller's backing arrays are not
// aliased, and an empty input is held as nil. wildcard is a pointer so absence
// (nil) is distinct from a present zero record (mirroring elementdeclaration.go's
// *TypeTable and the wildcard.go optional-slot pattern); when non-nil the
// pointed-to value is COPIED into the struct and hasWildcard is set — the pointer
// itself is never stored, so the caller's value is not aliased.
//
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the definition's provenance. Pass the position of this
// definition's own declaring element, never a convenient nearby one (a parent
// element's, say) — it is observable, not merely an error-charging convenience.
// A caller with no real parser position — a synthesized or programmatically
// built definition — passes the zero xsderr.Loc{}, which reads as "unknown".
func NewAttributeGroupDefinition(loc xsderr.Loc, name QName, attributeUses []AttributeUse, wildcard *Wildcard, annotations []Annotation) (AttributeGroupDefinition, error) {
	if name.Local == "" {
		return AttributeGroupDefinition{}, xsderr.New(ruleAgPropsCorrect, loc,
			"attribute group definition has an absent {name}, but the §3.6.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (ag-props-correct clause 1)")
	}
	seen := make(map[QName]struct{}, len(attributeUses))
	for i, use := range attributeUses {
		expanded := attributeUseDeclarationName(use)
		if _, dup := seen[expanded]; dup {
			return AttributeGroupDefinition{}, xsderr.New(ruleAgPropsCorrect, loc,
				"attribute group definition {attribute uses}[%d] repeats the expanded name %s, but ag-props-correct clause 2 forbids two attribute uses whose {attribute declaration}s share an expanded name", i, expanded)
		}
		seen[expanded] = struct{}{}
	}
	if err := rejectSiblingOnAttributeWildcard(loc, wildcard); err != nil {
		return AttributeGroupDefinition{}, err
	}
	g := AttributeGroupDefinition{loc: loc, name: name}
	if len(attributeUses) > 0 {
		g.attributeUses = append([]AttributeUse(nil), attributeUses...)
	}
	if wildcard != nil {
		g.wildcard, g.hasWildcard = *wildcard, true
	}
	if len(annotations) > 0 {
		g.annotations = append([]Annotation(nil), annotations...)
	}
	return g, nil
}

// attributeUseDeclarationName returns the expanded name of a use's {attribute
// declaration} without resolution: a LocalAttributeDeclaration's own
// Declaration.Name(), or an AttributeDeclarationRef's Name directly (both are
// available at shape time). It exists only for the in-package ag-props-correct
// clause 2 scan; the sum is deliberately NOT given an exported ExpandedName
// method (STYLE T5/8 — keep the surface minimal, no consumer needs it yet).
func attributeUseDeclarationName(use AttributeUse) QName {
	switch d := use.AttributeDeclaration().(type) {
	case LocalAttributeDeclaration:
		return d.Declaration.Name()
	case AttributeDeclarationRef:
		return d.Name
	default:
		// Unreachable: the sum is sealed to exactly these two variants, and
		// NewAttributeUse rejects a nil {attribute declaration}. Return the zero
		// QName rather than panic so a future variant fails visibly in the scan
		// instead of crashing.
		return QName{}
	}
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
func (g AttributeGroupDefinition) Name() QName {
	return g.name
}

// Loc reports the source position of the declaring element — provenance, not a
// §3.6.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown.
func (g AttributeGroupDefinition) Loc() xsderr.Loc {
	return g.loc
}

// AttributeUses returns the {attribute uses} property in document order. It
// returns a copy: mutating the result does not affect g. An empty {attribute
// uses} yields nil.
//
// The spec property is a set (§3.6.1); the document order here is an
// implementation choice for determinism and carries no spec significance.
func (g AttributeGroupDefinition) AttributeUses() []AttributeUse {
	if len(g.attributeUses) == 0 {
		return nil
	}
	return append([]AttributeUse(nil), g.attributeUses...)
}

// AttributeWildcard returns the {attribute wildcard} property (Optional); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
func (g AttributeGroupDefinition) AttributeWildcard() (Wildcard, bool) {
	return g.wildcard, g.hasWildcard
}

// Annotations returns the {annotations} property in document order. It returns
// a copy: mutating the result does not affect g. An empty {annotations} yields
// nil.
func (g AttributeGroupDefinition) Annotations() []Annotation {
	if len(g.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), g.annotations...)
}
