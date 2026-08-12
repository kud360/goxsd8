package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleSrcRedefine is Schema Representation Constraint: Redefinition Constraints
// and Semantics (Structures §4.2.4, id="src-redefine"). This package charges ONE
// of its clauses — 7.2.2, the obligation that a redefining <attributeGroup> with
// no self-reference RESTRICT the definition it replaces — because that clause
// invokes derivation-ok-restriction (§3.4.6.3) clause 3 over two components, and
// a constraint over assembled components is decided at finalize whatever
// document-level rule states it. Every other src-redefine clause is a fact about
// a schema DOCUMENT's element tree and is charged by the producer, which holds
// that tree and declares the same rule ID for it.
//
// Charging a §4.2.4 constraint from here is the footing ruleSrcResolve
// (resolve.go) already stands on: §3.17.6.2 is a Schema Representation
// Constraint too, and is charged here for the same reason — the fact it decides
// is only stateable over the assembled set.
const ruleSrcRedefine xsderr.Rule = "src-redefine"

// attributeGroupRedefinition pairs a redefining attribute group definition with
// the definition of the same expanded name in S2, the redefined schema document,
// so finalize can charge src-redefine clause 7.2.2 against the pair.
//
// name is the handle: the redefinition itself is in {attribute group
// definitions} like any other top-level definition, so it is looked up rather
// than stored twice (STYLE D3). original is stored, because it is in no property
// and no index — §4.2.4 clause 4.1.2 excepts an explicitly redefined definition
// from the components D1's schema includes, so this field is the only thing that
// holds it.
type attributeGroupRedefinition struct {
	name     QName                    // the redefinition's {name}: the handle into attributeGroupIndex
	original AttributeGroupDefinition // S2's definition; in no index, in no property
}

// AddRedefiningAttributeGroup appends g — a top-level attribute group definition
// that is a ·redefinition· (§4.2.4) of original — in document order, exactly as
// [SchemaBuilder.AddAttributeGroup] does, and records the pairing src-redefine
// clause 7.2.2 decides at Finalize: g's {attribute uses}/{attribute wildcard},
// viewed as a Complex Type Definition's, must satisfy clause 3 of
// derivation-ok-restriction (§3.4.6.3, c-ran) against original's.
//
// original is the definition of the same expanded name in S2, the redefined
// schema document. It contributes NO component: §4.2.4 clause 4.1.2 excepts an
// explicitly redefined definition from the components D1's schema includes, so
// original enters neither {attribute group definitions} nor the by-name index,
// and is reachable through no accessor. Passing it to AddAttributeGroup instead
// would fabricate a sch-props-correct (§3.17.6.1) clause 2 duplicate against the
// very definition it replaces.
//
// Call this only on clause 7.2's branch — the redefining <attributeGroup> with
// NO self-reference. Clause 7.1's branch (exactly one self-reference) states no
// restriction obligation and takes plain AddAttributeGroup; which branch applies
// is the producer's verdict (src-redefine clause 7's own dispatch), and this
// package does not infer it.
//
// It panics if either definition has an absent {name}, on the same grounds as
// [SchemaBuilder.AddType]'s nil guard: a redefinition pairs two TOP-LEVEL
// definitions, which §3.6.2.1 gives no anonymous form, so an absent name is a
// caller/producer bug and not a schema-validity condition.
func (b *SchemaBuilder) AddRedefiningAttributeGroup(g, original AttributeGroupDefinition) {
	if g.Name().Local == "" {
		panic("xsd: SchemaBuilder.AddRedefiningAttributeGroup: redefinition with an absent {name}")
	}
	if original.Name().Local == "" {
		panic("xsd: SchemaBuilder.AddRedefiningAttributeGroup: redefined original with an absent {name}")
	}
	b.attributeGroups = append(b.attributeGroups, g)
	b.attributeGroupRedefinitions = append(b.attributeGroupRedefinitions, attributeGroupRedefinition{name: g.Name(), original: original})
}

// checkAttributeGroupRedefinitions is src-redefine clause 7.2.2, charged over
// every pairing AddRedefiningAttributeGroup recorded, in the order they were
// added (STYLE D2 — never over attributeGroupIndex, so the first reported
// failure is deterministic).
//
// The clause is worded as a ROLE SUBSTITUTION rather than as a derivation: the
// redefinition's two attribute properties "viewed as the {attribute uses} and
// {attribute wildcard} of a Complex Type Definition" and the original's "viewed
// as … of the {base type definition}" must satisfy c-ran clause 3. So the
// redefinition plays T and the original plays B, and no complex type is minted
// to view them through (attributerestriction.go).
//
// NO FOLD APPLIES TO EITHER SIDE, and none is missing. The Note under clause
// 7.2.2 settles it for the redefinition — "An attribute group restrictively
// redefined per clause 7.2 corresponds to an attribute group whose {attribute
// uses} consist all and only of those attribute uses corresponding to
// <attribute>s explicitly present among the [children] of the <redefine>ing
// <attributeGroup> … No inheritance from the <redefine>d attribute group occurs"
// — and the same holds of the original, which is an ordinary top-level
// definition of S2. Neither has a {base type definition}, so the §3.4.2.4 clause
// 3 fold checkRestrictionAttributes depends on has nothing to reach through
// here; the §3.6.2.2 union of any <attributeGroup ref> children is already in
// both components, applied by the producer at mapping time (§3.6.2.1).
//
// PHASE ORDER: it runs immediately after checkComplexDerivations, and needs only
// what that step needs — Phase A's resolvability, so an AttributeDeclarationRef
// inside the REDEFINITION resolves, and Phase B's simple-type acyclicity, since
// loc-testSubP clause 5.1 walks a {base type definition} chain with no visited
// set. It follows no chain of its own: the pairing is a single edge from a named
// definition to an off-index component, and §3.6.2.1 has already inlined every
// <attributeGroup ref> at mapping time, so an AttributeGroupDefinition holds no
// edge to another one and no visited set belongs here (PRINCIPLES 9).
//
// GAP(xsd): a reference INSIDE the original — an <attribute ref>, or a type= on
// one of its local <attribute> children — that names nothing the assembled
// schema holds gets no src-resolve (§3.17.6.2) verdict, and no issue owns its
// retirement yet. Phase A walks the schema's own §3.17.1 properties, and clause
// 4.1.2 puts the original in none of them, so nothing charges it.
//
// DIRECTION, per reader of the value that goes missing — the resolved
// {attribute declaration} or {type definition} behind an original-side use
// (STYLE P3a):
//
//   - attributeUseName and findAttributeUse read the use's own QName with NO
//     resolution, so the original still REPORTS the use. That is what matters
//     most: under-reporting B's {attribute uses} is what would make this check
//     fail-CLOSED, and it does not happen.
//   - checkAttributeTypeDerivedOK (defaultbinding.go) gets not-ok from
//     attributeUseType and returns nil, leaving loc-testSubP clause 5.1
//     undecided: FAIL-OPEN.
//   - checkAttributeValueConstraintSubsumes reads the same miss as an ·absent·
//     ·effective value constraint· and discharges clause 5.2.1: FAIL-OPEN.
//   - checkAttributeUseSubsumes' clause 5.3 ({inheritable}) and
//     checkAttributeRestrictionRequired ({required}) read the Attribute Use
//     itself, never its declaration: UNAFFECTED.
func (s *Schema) checkAttributeGroupRedefinitions() error {
	for _, r := range s.attributeGroupRedefinitions {
		g, ok := s.attributeGroupIndex[r.name]
		if !ok {
			panic("xsd: checkAttributeGroupRedefinitions: no attribute group definition named " + r.name.String() +
				": AddRedefiningAttributeGroup records the pairing and the component together")
		}
		if err := s.checkAttributeRestriction(attributeGroupRedefinitionRestriction(g, r.original)); err != nil {
			return err
		}
	}
	return nil
}

// attributeGroupRedefinitionRestriction is the c-ran clause 3 comparison
// src-redefine clause 7.2.2 states: the redefinition as the spec's T, the
// redefined document's definition as its B, charged to src-redefine at the
// redefinition's own position — the redefining <attributeGroup> is what a reader
// must edit, and the original is in the document they are redefining.
//
// The two labels name the same expanded name (a redefinition redefines the name
// it replaces), so each says which side it is — "the redefining" against "the
// original", the §4.2.4 vocabulary — and the verb says which direction the
// obligation runs. Neither ends in a clause, so a message reading "%s's ·default
// binding·" of either stays grammatical.
func attributeGroupRedefinitionRestriction(g, original AttributeGroupDefinition) attributeRestriction {
	return attributeRestriction{
		rule:    ruleSrcRedefine,
		loc:     g.Loc(),
		verb:    "must restrict",
		clause:  "src-redefine clause 7.2.2 and derivation-ok-restriction clause 3, c-ran",
		derived: attributeGroupAttributeSide(g, "the redefining attribute group "+g.Name().String()),
		base:    attributeGroupAttributeSide(original, "the original attribute group "+original.Name().String()),
	}
}
