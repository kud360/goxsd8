package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleSrcRedefine is Schema Representation Constraint: Redefinition Constraints
// and Semantics (Structures §4.2.4, id="src-redefine"). This package charges TWO
// of its clauses — 7.2.2, the obligation that a redefining <attributeGroup> with
// no self-reference RESTRICT the definition it replaces, and 6.2.2, the
// obligation that a redefining <group> with no self-reference accept a SUBSET of
// the element sequences the definition it replaces accepts — because each states
// a relation over two assembled components (derivation-ok-restriction §3.4.6.3
// clause 3 for the one, language containment for the other), and a constraint
// over assembled components is decided at finalize whatever document-level rule
// states it. Every other src-redefine clause is a fact about a schema DOCUMENT's
// element tree and is charged by the producer, which holds that tree and declares
// the same rule ID for it.
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
// PHASE ORDER: it runs immediately after checkComplexDerivations, and needs
// Phase B's simple-type acyclicity, since loc-testSubP clause 5.1 walks a {base
// type definition} chain with no visited set. It follows no chain of its own:
// the pairing is a single edge from a named definition to an off-index
// component, and §3.6.2.1 has already inlined every <attributeGroup ref> at
// mapping time, so an AttributeGroupDefinition holds no edge to another one and
// no visited set belongs here (PRINCIPLES 9). It draws no resolvability
// guarantee from Phase A for either side — see the GAP below.
//
// GAP(xsd): a reference on EITHER side — an <attribute ref>, or a type= on a
// local <attribute> child, of the redefinition or of the original — that names
// nothing the assembled schema holds gets no src-resolve (§3.17.6.2) verdict,
// and #725 owns its retirement. This is not clause 4.1.2's doing:
// resolveReferences (resolve.go) never walks s.attributeGroups for ANY
// top-level attribute group, redefinition or not, so the redefinition side (in
// {attribute group definitions} like any other component) is exactly as
// unresolved as the original (which 4.1.2 additionally keeps out of every
// property and index, but that exclusion costs it nothing here — it never had
// Phase A coverage to lose).
//
// DIRECTION, per reader of the value that goes missing — the resolved
// {attribute declaration} or {type definition} behind a use on either side
// (STYLE P3a):
//
//   - AttributeUse.DeclarationName and findAttributeUse read the use's own
//     QName with NO resolution, so the side still REPORTS the use. That is
//     what matters most: under-reporting either side's {attribute uses} is
//     what would make this check fail-CLOSED, and it does not happen.
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

// modelGroupRedefinition pairs a redefining model group definition with the
// definition of the same expanded name in S2, the redefined schema document, so
// finalize can charge src-redefine clause 6.2.2 against the pair.
//
// name is the handle: the redefinition itself is in {model group definitions}
// like any other top-level definition, so it is looked up rather than stored
// twice (STYLE D3). original is stored, because it is in no property and no
// index — §4.2.4 clause 4.1.2 excepts an explicitly redefined definition from the
// components D1's schema includes, so this field is the only thing that holds it.
type modelGroupRedefinition struct {
	name     QName                // the redefinition's {name}: the handle into modelGroupIndex
	original ModelGroupDefinition // S2's definition; in no index, in no property
}

// AddRedefiningModelGroup appends d — a top-level model group definition that is
// a ·redefinition· (§4.2.4) of original — in document order, exactly as
// [SchemaBuilder.AddModelGroup] does, and records the pairing src-redefine clause
// 6.2.2 decides at Finalize: d's {model group} must accept a subset of the
// element sequences accepted by original's (§3.7.2).
//
// original is the definition of the same expanded name in S2, the redefined
// schema document. It contributes NO component: §4.2.4 clause 4.1.2 excepts an
// explicitly redefined definition from the components D1's schema includes, so
// original enters neither {model group definitions} nor the by-name index, and is
// reachable through no accessor. Passing it to AddModelGroup instead would
// fabricate a sch-props-correct (§3.17.6.1) clause 2 duplicate against the very
// definition it replaces.
//
// Call this only on clause 6.2's branch — the redefining <group> with NO
// self-reference. Clause 6.1's branch states no containment obligation and takes
// plain AddModelGroup; which branch applies is the producer's verdict
// (src-redefine clause 6's own dispatch, under 6.1's <element>-ancestor
// exclusion), and this package does not infer it.
//
// It panics if either definition has an absent {name}, which is the zero
// ModelGroupDefinition and no other value: NewModelGroupDefinition rejects the
// other route (mgd-props-correct). A zero original would present an empty {model
// group} as B and reject every redefinition against it, so the guard is on the
// fail-closed direction and not decorative.
func (b *SchemaBuilder) AddRedefiningModelGroup(d, original ModelGroupDefinition) {
	if d.Name().Local == "" {
		panic("xsd: SchemaBuilder.AddRedefiningModelGroup: redefinition with an absent {name}")
	}
	if original.Name().Local == "" {
		panic("xsd: SchemaBuilder.AddRedefiningModelGroup: redefined original with an absent {name}")
	}
	b.modelGroups = append(b.modelGroups, d)
	b.modelGroupRedefinitions = append(b.modelGroupRedefinitions, modelGroupRedefinition{name: d.Name(), original: original})
}

// checkModelGroupRedefinitions is src-redefine clause 6.2.2, charged over every
// pairing AddRedefiningModelGroup recorded, in the order they were added (STYLE
// D2 — never over modelGroupIndex, so the first reported failure is
// deterministic).
//
// The clause is LANGUAGE CONTAINMENT and nothing else: the redefinition's {model
// group} "accepts a subset of the element sequences accepted by that model group
// definition in S2". §3.8.4's key-lvip fixes what "accepts" means — V(P), the
// sequences ·locally valid· with respect to a particle — so the question is
// V(R.{model group}) ⊆ V(B.{model group}), which is cos-content-act-restrict
// (§3.4.6.4) clause 1 asked of two Model Groups instead of two Content Types. It
// is decided by that constraint's own engine (contentrestricts.go), never by a
// second traversal of the same automaton (STYLE T4).
//
// WRAPPING each Model Group in a synthetic Content Type is the reduction
// cvc-complex-content (§3.4.4.3) clause 1 licenses: with {open content} absent, a
// Content Type's ·locally valid· sequences are exactly its {particle}'s, so an
// element-only ElementContent whose {particle} is the group at 1..1 accepts
// exactly V(g). Both sides are wrapped IDENTICALLY, because the wrapper is a
// reduction device and any asymmetry in it becomes an asymmetry in the verdict.
//
// The scope is restrictsLanguage. 6.2.2 states clause 1's question and invokes
// cos-content-act-restrict for nothing, so clause 2's ·default binding·
// subsumption — which derivation-ok-restriction clause 2.4.2 does invoke — is not
// charged here: a redefining group that retypes an element while accepting the
// same sequences satisfies 6.2.2 and clause 2 would reject it.
//
// PHASE ORDER: it runs immediately after checkAttributeGroupRedefinitions, its
// clause-7.2.2 twin, and needs both Phase A (every <element ref>/<group ref> on
// EITHER side resolves — resolveReferences walks the originals for exactly this
// reason) and Phase B's checkModelGroupsAcyclic, since the automaton construction
// follows <group ref> edges with no visited set (PRINCIPLES 9). It follows no
// chain of its own: the pairing is a single edge from a named definition to an
// off-index component.
//
// GAP(xsd): contentTypeRestricts provisionally accepts whenever an ·all· group is
// reachable in R's content model, on a licence §3.4.6.3 grants
// derivation-ok-restriction clause 2.4.2 BY NAME. §4.2.4 grants clause 6.2.2 no
// such licence, so the leniency is an implementation incompleteness at this call
// site rather than a spec-licensed one: a redefining <group> reaching an <all> is
// accepted undecided. The direction is fail-open — a missed rejection, never a
// fabricated one, since the star addAll models an ·all· with over-approximates
// its language — and #743 owns the retirement. The same function's open-content
// arm is inert here: the wrapper below never sets OpenContent.
func (s *Schema) checkModelGroupRedefinitions() error {
	for _, r := range s.modelGroupRedefinitions {
		d, ok := s.modelGroupIndex[r.name]
		if !ok {
			panic("xsd: checkModelGroupRedefinitions: no model group definition named " + r.name.String() +
				": AddRedefiningModelGroup records the pairing and the component together")
		}
		redefining, err := modelGroupContent(d.ModelGroup(), d.Loc())
		if err != nil {
			return err
		}
		original, err := modelGroupContent(r.original.ModelGroup(), r.original.Loc())
		if err != nil {
			return err
		}
		if s.contentTypeRestricts(redefining, original, restrictsLanguage) {
			continue
		}
		return xsderr.New(ruleSrcRedefine, d.Loc(),
			"the redefining <group> %s accepts element sequences the original model group definition it replaces (at %s) does not, but src-redefine clause 6.2.2 requires its {model group} to accept a SUBSET of the sequences that definition in S2 accepts (§3.7.2)", r.name, r.original.Loc())
	}
	return nil
}

// modelGroupContent is the synthetic Content Type checkModelGroupRedefinitions
// compares through: element-only, {open content} ·absent·, {particle} the Model
// Group at exactly 1..1. Its ·locally valid· sequences are V(g) and nothing else
// (cvc-complex-content clause 1), so the containment the engine decides over it
// is the containment src-redefine clause 6.2.2 states over g.
//
// loc positions the two constructor rejections, neither of which a Model Group
// already inside a ModelGroupDefinition can produce — NewOccurs rejects only an
// out-of-range pair and NewParticle only an absent {term} — so the errors decide
// this verdict rather than being dropped (STYLE S3), charged to the definition
// the group was read from.
func modelGroupContent(g ModelGroup, loc xsderr.Loc) (ElementContent, error) {
	once, err := NewOccurs(loc, 1, 1)
	if err != nil {
		return ElementContent{}, err
	}
	p, err := NewParticle(loc, once, ResolvedTerm{Term: g}, nil)
	if err != nil {
		return ElementContent{}, err
	}
	return ElementContent{Particle: p}, nil
}
