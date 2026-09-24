package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleAgPropsCorrect is Attribute Group Definition Properties Correct
// (Structures §3.6.6, id="ag-props-correct"): an attribute group definition's
// properties must match the §3.6.1 property tableau. This package enforces:
//
//   - clause 1 (tableau shape): {name} is present, in NewAttributeGroupDefinition.
//     The rest is satisfied by construction — the sum and optional-slot machinery
//     already make an ill-formed {attribute uses} member or {attribute wildcard}
//     unrepresentable, so no extra check is needed there.
//   - clause 2: no two {attribute uses} members have {attribute declaration}s
//     with the same expanded name, at FINALIZE (checkAttributeGroupUsesUnique),
//     over the folded property. The constructor cannot decide it: a member that
//     arrives through an <attributeGroup ref> is not known until finalize
//     resolves the reference and takes §3.6.2.1's transitive closure
//     (attributegroupfold.go), and a scan over the unfolded content would report
//     an index naming no member of {attribute uses}.
//
// Circularity is NOT among the clauses, and nothing here or in the fold rejects
// it: §3.6.2.1 says "Circular reference is not disallowed", and ag-props-correct
// has exactly the two clauses above, so a cycle of <attributeGroup ref>s has no
// rule ID to be charged to (PRINCIPLES 9).
const ruleAgPropsCorrect xsderr.Rule = "ag-props-correct"

// AttributeGroupDefinition is the Attribute Group Definition component
// (Structures §3.6.1, id="agd"): a kind of Annotated Component with {name}
// (bundled with {target namespace} as an xsd.QName per this package's "Names are
// expanded QNames" convention — doc.go), {attribute uses}, and {attribute
// wildcard} (Optional).
//
// {attribute uses} is spec-worded a SET of Attribute Use components (§3.6.1
// tableau, §3.6.2.1 "union of ... sets"); this package represents it as a
// document-order slice per its standing convention (determinism, STYLE D2/D3) —
// the order carries no spec significance. ag-props-correct clause 2 forbids two
// members whose {attribute declaration}s share an expanded name, which Finalize
// enforces over the folded property (checkAttributeGroupUsesUnique).
//
// attributeContent is NOT a §3.6.1 property: it is the mapping INPUT §3.6.2.1
// and §3.6.2.2 build both attribute properties from — the definition's own
// <attribute> and <attributeGroup ref> children in document order — retained
// because a referenced group is reachable only once the schema is assembled.
// Finalize folds it into attributeUses and wildcard and sets it to nil
// (attributegroupfold.go), so a finalized definition holds no reference; until
// then attributeUses holds only its ResolvedAttributeUse arms, the pre-finalize
// under-approximation AttributeUses documents. The duplication of those arms in
// both fields ends at that fold (STYLE D3).
//
// Construct only through NewAttributeGroupDefinition, which rejects the state
// ag-props-correct (§3.6.6) clause 1 forbids and the member states
// checkAttributeContent does, so they are unrepresentable (STYLE T1).
// AttributeGroupDefinition is immutable after construction; Finalize's fold
// writes only the copies the assembled Schema holds.
type AttributeGroupDefinition struct {
	loc              xsderr.Loc // source position; provenance, not a §3.6.1 property
	name             QName
	attributeUses    []AttributeUse
	attributeContent []AttributeUseOrGroupRef
	wildcard         Wildcard
	hasWildcard      bool
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
// attributeContent is the definition's OWN attribute content in document order:
// a ResolvedAttributeUse per <attribute> child, an AttributeGroupRef per
// <attributeGroup ref> child, and a ResolvedAttributeGroup for a redefining
// definition's self-reference (see that type). checkAttributeContent rejects a
// member no arm describes. Clause 2 — no two members of {attribute uses} sharing
// an expanded name — is NOT checked here: a referenced group's members are not
// known until Finalize folds them in, so Finalize charges it over the folded
// property (checkAttributeGroupUsesUnique).
//
// wildcard is the definition's OWN <anyAttribute> only, §3.6.2.2's ·local
// wildcard·. The intersection with the referenced groups' {attribute wildcard}s
// that §3.6.2.2 makes the property is taken at Finalize, beside the {attribute
// uses} closure (attributegroupfold.go).
//
// It also rejects one state Wildcard Properties Correct (§3.10.6.1,
// w-props-correct) clause 5 forbids, charged to that rule rather than to
// ag-props-correct: an {attribute wildcard} whose {namespace constraint}
// carries the sibling keyword. This slot is one of the two places an attribute
// wildcard is identifiable as such; see rejectSiblingOnAttributeWildcard.
//
// attributeContent is copied; the caller's backing array is not aliased, and an
// empty input is held as nil. wildcard is a pointer so absence
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
func NewAttributeGroupDefinition(loc xsderr.Loc, name QName, attributeContent []AttributeUseOrGroupRef, wildcard *Wildcard) (AttributeGroupDefinition, error) {
	if name.Local == "" {
		return AttributeGroupDefinition{}, xsderr.New(ruleAgPropsCorrect, loc,
			"attribute group definition has an absent {name}, but the §3.6.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (ag-props-correct clause 1)")
	}
	if err := checkAttributeContent(loc, "attribute group definition "+name.String(), attributeContent); err != nil {
		return AttributeGroupDefinition{}, err
	}
	if err := rejectSiblingOnAttributeWildcard(loc, wildcard); err != nil {
		return AttributeGroupDefinition{}, err
	}
	g := AttributeGroupDefinition{loc: loc, name: name, attributeUses: directAttributeUses(attributeContent)}
	if len(attributeContent) > 0 {
		g.attributeContent = append([]AttributeUseOrGroupRef(nil), attributeContent...)
	}
	if wildcard != nil {
		g.wildcard, g.hasWildcard = *wildcard, true
	}
	return g, nil
}

// checkAttributeGroupUsesUnique is ag-props-correct (§3.6.6) clause 2, charged at
// Finalize over every Attribute Group Definition's FOLDED {attribute uses}: the
// {attribute group definitions} in document order, then the <redefine>
// originals in pairing order, which §4.2.4 clause 4.1.2 keeps out of every
// property yet checkAttributeGroupRedefinitions reads as the B side of clause
// 7.2.2. A definition held by value in a ResolvedAttributeGroup needs no charge
// of its own: the fold expands it into its holder, so every duplicate among its
// members is one among the holder's.
//
// It runs after attributegroupfold.go's fold, the first point at which the
// members reached through an <attributeGroup ref> exist. The first duplicate by
// index is the one reported (duplicateAttributeUseName), and the fold lays the
// members out in the document order the producer-time check this replaced
// scanned, so the message names the same member it did (#479).
func (s *Schema) checkAttributeGroupUsesUnique() error {
	for _, g := range s.attributeGroups {
		if err := checkAttributeGroupDefinitionUsesUnique(g); err != nil {
			return err
		}
	}
	for _, r := range s.attributeGroupRedefinitions {
		if err := checkAttributeGroupDefinitionUsesUnique(r.original); err != nil {
			return err
		}
	}
	return nil
}

// checkAttributeGroupDefinitionUsesUnique charges ag-props-correct clause 2
// against one folded definition, at its own position.
func checkAttributeGroupDefinitionUsesUnique(g AttributeGroupDefinition) error {
	i, name, duplicate := duplicateAttributeUseName(g.attributeUses)
	if !duplicate {
		return nil
	}
	return xsderr.New(ruleAgPropsCorrect, g.Loc(),
		"%s {attribute uses}[%d] repeats the expanded name %s, but ag-props-correct clause 2 forbids two attribute uses whose {attribute declaration}s share an expanded name", attributeGroupOwner(g), i, name)
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
//
// On a definition reached through a finalized [Schema] this is the §3.6.2.1
// property: the union of the definition's own attribute uses with those of
// every attribute group its <attributeGroup ref>s reach, transitively and
// through any reference cycle, folded in at Finalize (attributegroupfold.go).
// On a definition a caller built with [NewAttributeGroupDefinition] and has not
// yet finalized, it is only the ResolvedAttributeUse members that caller passed
// in: a referenced group is reachable only from the assembled schema.
func (g AttributeGroupDefinition) AttributeUses() []AttributeUse {
	if len(g.attributeUses) == 0 {
		return nil
	}
	return append([]AttributeUse(nil), g.attributeUses...)
}

// AttributeWildcard returns the {attribute wildcard} property (Optional); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
//
// On a definition reached through a finalized [Schema] this is the §3.6.2.2
// property: the definition's own <anyAttribute> intersected (cos-aw-intersect)
// with the {attribute wildcard}s of every attribute group its <attributeGroup
// ref>s reach, folded in at Finalize (attributegroupfold.go). On a definition
// not yet finalized it is only the ·local wildcard· its caller passed in.
func (g AttributeGroupDefinition) AttributeWildcard() (Wildcard, bool) {
	return g.wildcard, g.hasWildcard
}
