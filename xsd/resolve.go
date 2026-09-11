package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleSrcResolve is Schema Representation Constraint: QName resolution (Schema
// Document) (Structures §3.17.6.2, id="src-resolve"): for a QName to resolve to
// a schema component of a specified kind, that component must be a member of the
// appropriate {…definitions}/{…declarations} property (clause 1, kind-specific:
// 1.1 type, 1.2 attribute decl, 1.3 element decl, 1.4 attribute group, 1.5 model
// group, 1.7 identity constraint) with a matching {name}/{target namespace}
// (clauses 2–3). A dangling reference (no such component) and a wrong-kind
// reference (the name exists only in another kind's table) are the SAME failure
// — the kind-specific lookup simply misses — so both are charged this rule,
// differing only in message. Clause 4 (namespace reachability from the referring
// document) is a distinct precondition that needs the schema-document import
// graph, which the compiled component model does not carry; it is out of #173's
// scope and left to the producer (#176).
const ruleSrcResolve xsderr.Rule = "src-resolve"

// anyTypeName is the expanded name of xs:anyType, the one Complex Type
// Definition permitted to be its own {base type definition} (§3.4.7,
// any-type-itself). This package models no built-in anyType anchor (it is
// outside the simple-type graph, per simpletype.go), so checkComplexBaseAcyclic
// detects the permitted self-derivation by this name rather than by pointer
// identity.
var anyTypeName = QName{Space: XMLSchemaNS, Local: "anyType"}

// resolve is the finalize resolution pass (Structures §3.17.3 assembly, §3.17.6.2
// src-resolve). It runs in two phases, both driven from the compiled Schema's
// document-order slices (STYLE D2 — the by-name indexes are used only for point
// lookups, never ranged to walk or to pick which failure to report, so the first
// reported failure is deterministic):
//
//   - Phase A (resolved-target verdicts): walk every in-scope QName reference
//     site and charge the verdicts that need the target IN HAND. It charges
//     NOTHING for a target that is absent: §5.3 (Missing Sub-components) makes an
//     unresolvable reference an ·absent· value rather than a schema error, and
//     §4.2.3 requires the ·QName· to be retained through construction, so this
//     phase resolves in order to judge and never in order to reject (#434; the
//     residue §5.3 still owes is #250's, recorded at resolveElementDecl). What it
//     does charge: the owner-of-owner representation invariant on a
//     SubstitutionGroupHeadTypeRef slot (resolveTypeDefinitionSlot), and at the
//     keyref site the two c-props-correct checks that only the resolved target
//     makes decidable — clause 1 for a keyref pointing at another keyref, and
//     clause 2, the {fields} cardinality match between keyref and {referenced
//     key} (resolveKeyref).
//   - Phase B (circularity): reject the spec-forbidden named circularities that
//     become representable only across the assembled set — the complex-type base
//     chain (ct-props-correct clause 3), the SIMPLE-type base chain
//     (st-props-correct clause 2), union membership (cos-st-restricts clause
//     3.3), the <group ref> graph (mg-props-correct clause 2), and the
//     substitution-group affiliation graph (e-props-correct clause 5). Every
//     unguarded chain walk in derivation.go presupposes its two simple-type
//     halves; see the paragraph below. The membership check runs after the
//     base-chain one, whose verdict it uses: a ·restriction· reports its base's
//     membership, so walking a membership walks base chains too.
//   - Phase C (content-model validity): reject the three §3.8.6 Model Group
//     constraints that read a whole content model with its <group ref>s expanded
//     and its <element ref>s and substitution groups followed — All Group Limited
//     (cos-all-limited, allgrouplimited.go), Unique Particle Attribution
//     (cos-nonambig, particleattribution.go) and Element Declarations Consistent
//     (cos-element-consistent, elementconsistent.go). cos-all-limited runs first
//     within the phase, for the reason checkAllGroupsLimited records.
//   - Phase D (derivation validity), in five steps. It OPENS on the simple-type
//     side: checkSimpleTypeDerivations puts every Simple Type Definition the
//     finalized schema reaches — anonymous inline ones included, which no index
//     holds — to SimpleType.CheckDerivation and then to the installed
//     SimpleTypeRestrictionChecker, charging between them the graph half and the
//     facet-VALUE half of Derivation Valid (Restriction, Simple) (§3.16.6.2,
//     cos-st-restricts) plus st-props-correct clauses 1, 3 and 5, against every
//     type the usable gate admits. That step needs Phase B's simple-type
//     acyclicity and nothing else; it runs first within the phase so a schema whose simple
//     types are themselves invalid says so before any complex-type derivation
//     verdict computed over them, and so that the four steps after it may treat
//     every simple-type base chain as already resolved. Its walk carries no
//     visited set, for the reason its own doc records. It then MATERIALISES the
//     two attribute-side properties whose mapping rules a producer cannot
//     finish, because each needs the resolved base: {attribute uses}, whose §3.4.2.4
//     clause 3 folds the {base type definition}'s uses into every complex type's own
//     (attributeusefold.go, #401), and {attribute wildcard}, whose §3.4.2.5 clause 2.2
//     unions an EXTENSION's own ·complete wildcard· with its ·base wildcard·
//     (attributewildcardfold.go, #265). The two are independent properties, so their
//     relative order carries no verdict; both precede the checks. It then rejects the
//     derivation-relative constraints that need that resolved base — the ct-props-correct
//     (§3.4.6.1) clauses 2 and 4, derivation-ok-restriction (§3.4.6.3) for every
//     restriction-derived complex type, and cos-ct-extends (§3.4.6.2) for every
//     extension-derived one (complexderivation.go, complexextension.go, defaultbinding.go,
//     effectivetotalrange.go). Immediately after those, over the same c-ran clause 3
//     apparatus, it charges the one constraint that compares two ATTRIBUTE GROUP
//     definitions — src-redefine (§4.2.4) clause 7.2.2, which requires a redefining
//     <attributeGroup> carrying no self-reference to RESTRICT the definition it redefines
//     (redefinition.go). It finishes on the one derivation verdict quantified over ELEMENT
//     declarations rather than types — e-props-correct (§3.3.6.1) clause 4 (c-vs-sg),
//     which requires a declaration's {type definition} to be ·validly substitutable· for
//     that of each member of its {substitution group affiliations}, subject to that
//     member's {substitution group exclusions} (substitutiongrouptypes.go). It shares the
//     phase because it is the same cos-ct-derived-ok/cos-st-derived-ok engine pair under a
//     different quantifier; see checkSubstitutionGroupTypes for the ordering argument. It
//     CLOSES on that engine's third quantifier — e-props-correct clause 7, which requires
//     each Type Alternative's {type definition} in a declaration's {type table}, and the
//     {default type definition}'s, to be ·validly substitutable· for the declaration's own
//     {type definition} subject to its {disallowed substitutions}, unless it is ·xs:error·
//     (typetablesubstitutable.go). Clause 7 runs after clause 4 so a declaration failing
//     both reports the failure that does not depend on its type table.
//   - Phase E (value-constraint validity), in two walks over the same file
//     (valueconstraintvalid.go). The DESCENDING walk rejects an Attribute Use whose
//     own {value constraint} contradicts its resolved {attribute declaration}'s
//     fixed one — au-props-correct (§3.5.6) clause 3, both the variety half and
//     the {value}-identity half — and an Attribute Use or LOCAL Attribute
//     Declaration whose own {value constraint} is not a valid default with
//     respect to the resolved {type definition} (au-props-correct clause 2 and
//     a-props-correct (§3.2.6.1) clause 2, both over the one shared
//     cos-valid-simple-default (§3.2.6.2) predicate, #371). The same descent
//     charges e-props-correct (§3.3.6.1) clause 2 against every Element
//     Declaration it passes through, global or local, deciding cos-valid-default
//     (§3.3.6.2) — clause 1 through that same shared predicate, clause 2 through
//     particleEmptiable (elementdefaultvalid.go, #463). The DECLARATION-side
//     walk charges a-props-correct clause 2 against every GLOBAL Attribute
//     Declaration. Both walks are needed and neither duplicates the other: a
//     local declaration is reachable only through its owning use, a global one
//     only through the schema's {attribute declarations} — where it is charged
//     once, not once per referencing use.
//
// EVERY UNGUARDED CHAIN WALK IN derivation.go PRESUPPOSES PHASE B'S SIMPLE-TYPE
// PROOFS, and the dependency is stated here rather than left implicit.
// SimpleType.Variety, .Primitive, .Item, .Members and .EffectiveFacets, and the
// derivedOKSimple relation over them, follow {base type definition} with no
// visited set (STYLE D4). That was discharged BY CONSTRUCTION while the slot
// held a live pointer which had to pre-exist the type naming it (PRINCIPLES 9);
// with the slot deferred to a name it is discharged by checkSimpleBaseAcyclic
// running in Phase B, before any later phase walks such a chain. The MEMBERSHIP
// walks — derivedOKSimple's descent through a union's members, and
// unionMembershipHasList — are unguarded on the same footing, and rest on
// checkUnionMembershipAcyclic in the same phase; value's needsContext follows
// the same edges and rests on the same proof.
//
// Phase C runs strictly after Phase B, and that ordering is load-bearing rather
// than cosmetic: both checks expand <group ref>s and walk {substitution group
// affiliations} with NO cycle guard, which is licensed only because
// checkModelGroupsAcyclic and checkSubstitutionGroupsAcyclic have already
// rejected a circular graph of each kind (PRINCIPLES 9). Deciding
// cos-equiv-derived-ok-rec clause 2.3 makes them follow {base type definition}
// too, so they rest on checkComplexBaseAcyclic as well — plus the explicit
// xs:anyType test §3.4.7's one self-based type needs, which no acyclicity check
// can supply (substitutiongroup.go).
//
// Phase D runs last, and its position after each earlier phase is load-bearing
// too — its COMPLEX-type steps need Phase B's acyclicity (they walk {base type
// definition} chains and <group ref> edges with no visited set) and Phase C's
// cos-element-consistent (which is what makes the ·locally declared type· of an
// element name within one content model a function rather than a relation,
// without which derivation-ok-restriction clause 4 and its cos-ct-extends
// clause-1.6 twin would not be statable). See checkComplexDerivations' own doc
// for the full statement. Its SIMPLE-type step is exempt from Phase C alone: it
// needs Phase B's simple-type acyclicity (it follows a deferred {base type
// definition} with no visited set), but decides nothing about content models.
//
// NO PHASE DEPENDS ON PHASE A FOR RESOLVABILITY, and none may be written to.
// Phase A charges nothing for an unresolvable reference (§5.3), so every later
// reader reaches an ·absent· target on a schema that will be ACCEPTED: the
// complex side reads through the ok-checked accessors ResolvedType,
// ResolvedSimpleType, ResolvedAttributeDeclaration and the modelGroupIndex/
// s.Element lookups, each of which skips and contributes nothing, and the simple
// side is gated by usable before anything is charged against it.
//
// Phase E runs LAST. Its position is not load-bearing the way Phase D's is — it
// reads one component at a time and follows no chain, apart from the ·emptiable·
// verdict cos-valid-default clause 2.2 takes over a particle's own subtree — and
// it charges the narrowest, most component-local failure of the five, so
// reporting it after the structural phases keeps the first reported failure the
// most structural one. Its two walks run descending first, declaration-side
// second, which is arbitrary — no verdict depends on the order, only which of two
// independent failures is reported first.
//
// resolve stores no RESOLUTION result: a consumer that later wants the component
// behind a reference obtains it by a read-time index lookup
// (schema.Type/Element/Attribute), never from a resolved pointer this pass
// produced — that pointer would be state derivable from the QName plus the index
// (STYLE D3). Its only mutations are Phase D's two folds, which are mapping rules
// finished rather than references cached: each overwrites a property with the
// value its §3.4.2 rule defines, leaving one encoding of it and not two (see
// foldAttributeUses and foldAttributeWildcards). Each writes into every root that
// can OWN a complex type — {type definitions}, {element declarations}, {model
// group definitions} and the <redefine> originals, with the by-name index derived
// from each — because a folded type is observable only through the slot holding
// it (ownedtypefold.go).
//
// An absent reference and a present-but-unresolvable one are BOTH skipped, and
// they are different facts reaching the same verdict, not one fact. Absence — a
// zero QName in a bare-QName slot, a nil TypeDefinitionOrRef in a {type
// definition} slot — means "no reference", which src-resolve has nothing to
// resolve, and it is reachable only from the genuinely OPTIONAL reference slots
// (a ComplexType with no {base type definition} name, an ElementDeclaration with
// no {type definition}, for instance); it can never mask a mandatory reference,
// because the four ref-only sum variants — AttributeDeclarationRef,
// ElementDeclarationRef, ModelGroupRef, TypeDefinitionRef — cannot hold a zero
// QName in the first place: NewAttributeUse, NewParticle, NewElementDeclaration
// and NewAttributeDeclaration reject one at construction (STYLE T1). A
// MANDATORY reference that resolves to nothing is §5.3's ·absent· value, which
// the retained ·QName· records and ·assessment· answers for; construction
// charges nothing either way.
//
// FOLLOW-COST ASYMMETRY (recorded deliberately, not silently): every reference
// is followed by a read-time lookup, never by a resolved pointer some pass
// stored, so the cost of following one falls on each reader. The three Query
// views (Type/Element/Attribute Resolvers) and Schema.ModelGroup(QName) (#307)
// serve four of the five kinds; a ModelGroupRef, like a ModelGroupScopeParent
// (elementdeclaration.go), needed no new Resolver interface (no consumer takes
// one, unlike Type/Element/Attribute). Because resolution is still
// validation-only, this package exposes no Schema.IdentityConstraint(name)
// accessor (STYLE 8 — export nothing without a consumer): the cost of
// following a keyref at read time is shifted onto the future Walker/Matcher
// and instance validator, which will need exactly that accessor. That
// remaining asymmetry is intentional, discharged by the consumer issue that
// adds it.
func (s *Schema) resolve() error {
	if err := s.resolveReferences(); err != nil {
		return err
	}
	if err := s.checkComplexBaseAcyclic(); err != nil {
		return err
	}
	if err := s.checkSimpleBaseAcyclic(); err != nil {
		return err
	}
	if err := s.checkUnionMembershipAcyclic(); err != nil {
		return err
	}
	if err := s.checkModelGroupsAcyclic(); err != nil {
		return err
	}
	if err := s.checkSubstitutionGroupsAcyclic(); err != nil {
		return err
	}
	if err := s.checkAllGroupsLimited(); err != nil {
		return err
	}
	if err := s.checkContentModelsUnambiguous(); err != nil {
		return err
	}
	if err := s.checkElementDeclarationsConsistent(); err != nil {
		return err
	}
	if err := s.checkSimpleTypeDerivations(); err != nil {
		return err
	}
	if err := s.foldAttributeUses(); err != nil {
		return err
	}
	if err := s.foldAttributeWildcards(); err != nil {
		return err
	}
	if err := s.checkComplexDerivations(); err != nil {
		return err
	}
	if err := s.checkAttributeGroupRedefinitions(); err != nil {
		return err
	}
	if err := s.checkModelGroupRedefinitions(); err != nil {
		return err
	}
	if err := s.checkSubstitutionGroupTypes(); err != nil {
		return err
	}
	if err := s.checkTypeTableSubstitutability(); err != nil {
		return err
	}
	if err := s.checkComponentValueConstraints(); err != nil {
		return err
	}
	return s.checkAttributeDeclarationDefaults()
}

// resolveReferences is Phase A: it walks every reference site in document order
// and charges the first verdict that needs the resolved target. An unresolvable
// target is NOT one of those verdicts — §5.3 retains the ·QName· and defers the
// consequence to ·assessment· — so the walk carries exactly two charges: the
// owner-of-owner representation invariant (resolveTypeDefinitionSlot) and the
// keyref's two c-props-correct clauses (resolveKeyref).
//
// REFERRER-LOC CONVENTION. Every rejection here is charged to the REFERRING
// component's position, never to the target's — where the target is missing
// outright it has none. Each helper therefore takes a loc alongside its ctx phrase,
// and the descent threads down the position of the nearest ENCLOSING component that
// retains one: a ComplexType, ElementDeclaration, AttributeDeclaration or
// ModelGroupDefinition re-roots it at its own Loc() as the walk enters it, while a
// Particle, AttributeUse, ModelGroup, TypeTable or TypeAlternative inherits the
// enclosing component's, because none of those five retains a position or exposes
// an accessor for one — nothing consumed their positions when they were built, so
// none was minted (xsd doc.go, STYLE T5). Inheriting is not an approximation of the
// wrong thing: an inline particle tree or attribute use belongs to exactly one such
// enclosing component, so its position names the declaration a reader must open,
// one enclosing element out.
func (s *Schema) resolveReferences() error {
	w := s.referenceWalk()
	for _, t := range s.types {
		if err := w.walkTypeRoot(t); err != nil {
			return err
		}
	}
	for _, e := range s.elements {
		if err := w.walkElementDeclaration(e); err != nil {
			return err
		}
	}
	for _, a := range s.attributes {
		if err := w.walkAttributeDeclaration(a); err != nil {
			return err
		}
	}
	// {attribute group definitions} (§3.17.1), whose reference sites are an
	// <attribute ref> and a local <attribute>'s type= (src-resolve clauses 1.2 and
	// 1.1). A definition's {attribute uses} are its own components: the producer
	// builds them for every top-level <attributeGroup> and splices a COPY into each
	// complex type referencing it, since §3.6.2.1 inlines the ref at mapping time.
	// Those copies are walked with their types above, so this loop is what reaches
	// a definition NOTHING references. Re-charging the same names once per
	// referencing type is bounded and harmless, and no visited set belongs here
	// (PRINCIPLES 9): that same inlining leaves an Attribute Group Definition
	// holding no edge to another one.
	for _, g := range s.attributeGroups {
		if err := w.walkAttributeGroupDefinition(g); err != nil {
			return err
		}
	}
	// The S2 originals of the <attributeGroup> redefinitions, in pairing order,
	// for the reason the <group> originals below are walked: they are in no
	// property and no index (§4.2.4 clause 4.1.2), so the loop above reaches none
	// of them, and this phase's owner-of-owner invariant would go uncharged on the
	// {type definition} slot of every local <attribute> inside one.
	for _, r := range s.attributeGroupRedefinitions {
		if err := w.walkAttributeGroupDefinition(r.original); err != nil {
			return err
		}
	}
	for _, mgd := range s.modelGroups {
		if err := w.walkModelGroup(mgd.ModelGroup(), mgd.Loc()); err != nil {
			return err
		}
	}
	// The S2 originals of the <group> redefinitions, in pairing order, for the same
	// reason: they are in no property and no index (§4.2.4 clause 4.1.2), so the
	// loop above reaches none of them and the inline <complexType>s their local
	// <element> children own would go unvisited.
	for _, r := range s.modelGroupRedefinitions {
		if err := w.walkModelGroup(r.original.ModelGroup(), r.original.Loc()); err != nil {
			return err
		}
	}
	for _, ic := range s.identityConstraints {
		if err := s.resolveKeyref(ic); err != nil {
			return err
		}
	}
	return nil
}

// referenceWalk is Phase A's set of charges for the shared component descent
// (componentwalk.go): the owner-of-owner representation invariant at a {type
// definition}/{base type definition} slot (resolveTypeDefinitionSlot), and the
// two slots the descent does not itself enter — an element declaration's {type
// table} and its keyrefs (resolveElementDecl).
//
// The hooks NOT wired are the point of this landing (#434). A particle's by-name
// {term}, an <attribute ref> and a simple type's three SimpleTypeOrRef slots each
// used to carry a src-resolve verdict here; §5.3 makes an unresolvable one an
// ·absent· value rather than a schema error, so with the charge gone each hook
// had nothing left to do and is not wired. The descent still enters every
// component they used to be charged on, through the arms componentWalk follows
// on its own — a LocalAttributeDeclaration through walkAttributeUse, an inline
// simple type through the InlineTypeDefinition arm — so no slot lost its
// owner-of-owner verdict along with its src-resolve one.
func (s *Schema) referenceWalk() componentWalk {
	return componentWalk{
		typeDefinitionSlot: s.resolveTypeDefinitionSlot,
		elementDeclaration: s.resolveElementDecl,
	}
}

// resolveTypeDefinitionSlot charges the {type definition}/{base type definition}
// slot of a type, element declaration or attribute declaration (§3.3.2.1
// dcl.elt.common, §3.2.2.2 dcl.att.local), exhaustively over
// TypeDefinitionOrRef's three arms. ctx names the referring site for the message
// and loc positions it at the component holding the slot.
//
//   - nil is an absent {type definition}: there is nothing to resolve.
//   - TypeDefinitionRef is the by-name arm. It is NOT charged src-resolve clause
//     1.1 when it names nothing — §5.3 retains the ·QName· as an ·absent· {type
//     definition} and defers the consequence to ·assessment·, the same reading
//     the head arm below already got — so the arm charges nothing at all and the
//     miss surfaces at every read through ResolvedType's ok.
//   - InlineTypeDefinition is already the component, reached through no symbol
//     table, so the SLOT itself needs no resolution, and neither do the
//     references inside it: a *SimpleType's three SimpleTypeOrRef slots and a
//     ComplexType's by-name {base type definition} are §5.3 slots like this one.
//     The shared descent still enters the component, so this phase's remaining
//     charges are applied inside it exactly as at a top-level one — which is what
//     reaches the src-expredef clause 1.1 original of a <redefine>, an anonymous
//     component held by no index and named by nothing.
//   - SubstitutionGroupHeadTypeRef names the element declaration that OWNS the
//     inherited anonymous type. It is NOT charged src-resolve clause 1.3 when it
//     names nothing — see below — and the descent does not enter it either: the
//     head is itself an entry of s.elements, so its own inline type is walked
//     when its turn comes, exactly once.
//
// The head name reaches this component through {substitution group
// affiliations}, which #281 aligned with §5.3 first; the by-name arm above now
// reads the same way, so the two no longer need telling apart. A
// substitutionGroup naming nothing is a VALID schema whose members are ·absent·
// (§5.3 Missing Sub-components; W3C saxonData/Missing missing002 pins it, and
// resolveElementDecl carries the full argument), the miss returns nil and clause
// 3 simply contributes no type — which is also what ResolvedType answers, and
// what checkElementSubstitutableForHeads skips on.
//
// The ONE rejection this function carries is a representation invariant, not a
// spec clause: an OWNER-OF-OWNER chain, where the named head's own {type
// definition} is itself a SubstitutionGroupHeadTypeRef. The producer walks to
// the TERMINAL head precisely so that never happens, and ResolvedType's read is
// DEPTH-1 on the strength of it; rejecting the chain here is what keeps that
// read one hop rather than a silent fail-open (STYLE P3). It is also the whole
// reason this hook is still wired into referenceWalk at all.
//
// ctx is retained for that one rejection's message and is unused by the other
// arms, which charge nothing.
func (s *Schema) resolveTypeDefinitionSlot(ref TypeDefinitionOrRef, loc xsderr.Loc, ctx string) error {
	switch r := ref.(type) {
	case nil:
		return nil
	case TypeDefinitionRef:
		return nil // an ·absent· {type definition} (§5.3); ResolvedType answers ok=false
	case InlineTypeDefinition:
		return nil // the descent enters it; the slot itself resolves nothing
	case SubstitutionGroupHeadTypeRef:
		head, ok := s.Element(r.Head)
		if !ok {
			return nil // an ·absent· head (§5.3), not a src-resolve clause 1.3 failure
		}
		if _, chained := head.TypeDefinition().(SubstitutionGroupHeadTypeRef); chained {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s inherits its type from the substitution group head %s (§3.3.2.1 dcl.elt.common clause 3), but that head's own {type definition} is itself inherited from a further head, and this variant must name the declaration that OWNS the anonymous type", ctx, r.Head)
		}
		return nil
	default:
		panic("xsd: resolveTypeDefinitionSlot: non-exhaustive TypeDefinitionOrRef switch")
	}
}

// resolveKeyref resolves an identity constraint's {referenced key} against
// idcIndex directly, but only for a keyref (a key/unique carries no reference),
// and charges the two c-props-correct (§3.11.6.1) requirements that need the
// RESOLVED target. The split with NewIdentityConstraint is: the constructor owns
// clause 1's LOCAL presence-iff-keyref shape only, and finalize (here) owns
// clause 1's category and clause 2's cardinality.
//
//   - clause 1 (category): the referenced constraint must be a key or unique, NOT
//     another keyref. A same-kind lookup resolves (both are IDCs), so the
//     keyref→keyref mismatch is a c-props-correct fault about a target that IS
//     there, not a resolution failure.
//   - clause 2 (cardinality): the keyref's {fields} count must equal the
//     {referenced key}'s.
//
// A refer= naming NOTHING is charged nothing. §3.11.2 declare-key maps
// {referenced key} through the ordinary ·resolution· convention, so a miss is
// §5.3's ·absent· value under c-props-correct clause 1's own "modulo the impact
// of Missing Sub-components (§5.3)"; the ·QName· is retained (§4.2.3) and
// ·assessment· answers for it (#434, and see resolveElementDecl). The two
// clauses below are the half §5.3 does NOT forgive — clause 2 carries no modulo
// qualifier at all — and they stay, which is why the existence test and the two
// checks had to be told apart rather than dropped together.
//
// Both rejections are positioned at the KEYREF (ic.Loc()), never at the target:
// the keyref is the component whose {referenced key} is wrong and the one the
// schema author must edit. resolveKeyref therefore needs no threaded loc — an
// IdentityConstraint retains its own, top-level or nested alike.
//
// The category check runs first, so a target that is both the wrong category and
// the wrong cardinality reports the category failure — one deterministic first
// failure (STYLE D1). The lengths are read off the unexported fields of two
// same-package values rather than through Fields(), whose defensive copy would
// be allocated only to be discarded.
func (s *Schema) resolveKeyref(ic IdentityConstraint) error {
	ref, isKeyref := ic.ReferencedKeyName()
	if !isKeyref || ref == (QName{}) {
		return nil
	}
	target, ok := s.idcIndex[ref]
	if !ok {
		return nil // an ·absent· {referenced key} (§5.3); neither clause is competent
	}
	if target.Category() == IdentityConstraintKeyref {
		return xsderr.New(ruleICProps, ic.Loc(),
			"keyref %s references %s, which is itself a keyref, but c-props-correct clause 1 requires a keyref's {referenced key} to be a key or unique", ic.Name(), ref)
	}
	if len(ic.fields) != len(target.fields) {
		return xsderr.New(ruleICProps, ic.Loc(),
			"keyref %s has %d {fields} but its {referenced key} %s has %d, and c-props-correct clause 2 requires equal cardinality", ic.Name(), len(ic.fields), ref, len(target.fields))
	}
	return nil
}

// resolveElementDecl reaches the two reference-bearing slots of an element
// declaration that the shared descent does not itself enter: each type-table
// alternative's {type definition} and each nested {identity-constraint
// definitions} keyref. Its own {type definition} is reached by the descent,
// before this runs (resolveTypeDefinitionSlot).
//
// THIS IS THE PACKAGE'S §5.3 STATEMENT, and every reference slot now reads the
// same way. Resolution failure is not a schema-construction error: "the
// ·resolution· of such QNames can fail, resulting in one or more values of or
// containing ·absent· where a component is mandated", and §5.3 then defers the
// consequence to ·assessment· — an element item validated against a component
// with an ·absent· value fails cvc-elt clause 1 and the processor falls back to
// ·lax assessment·; an attribute item fails cvc-attribute clause 1 with no
// fallback. §4.2.3 makes the construction-time half mandatory in as many words:
// "During schema construction, implementations must retain ·QName· values for
// such references". It is why e-props-correct clause 1 — and the a-, ct-, mgd-,
// c- and st-props-correct clause 1s with it — reads "as described in the property
// tableau ... modulo the impact of Missing Sub-components (§5.3)". W3C
// saxonData/Missing missing002 pins it for {substitution group affiliations}:
// substitutionGroup="rotten" with no `rotten` declared is a VALID schema whose
// only invalid instance is the one that uses the affected declaration.
//
// #281 aligned {substitution group affiliations} and #434 the rest: a dangling
// {type definition}, <element ref>, <attribute ref>, <group ref>, keyref refer=,
// simple-type base=, itemType= or memberTypes= entry is retained and charged
// nothing. The reader pattern this slot established is what every one of them
// now follows — affiliationChainReaches (substitutiongroup.go) skips a member it
// cannot look up, so no chain runs through an absent component, and
// checkSubstitutionGroupsAcyclic contributes no edges for one.
//
// GAP(xsd): §5.3's ·assessment·-time half does not exist. A schema now CONSTRUCTS
// with ·absent· values in it, and nothing yet turns one into the cvc clause-1
// failure and ·lax assessment· fallback quoted above — there is no validator to
// charge them from. Until there is, an ·absent· value simply withholds every
// verdict predicated on the component it stands for. #250 owns that half, and
// the two GAP(xsd) markers at usable record what it owes on the simple-type side.
func (s *Schema) resolveElementDecl(e ElementDeclaration) error {
	if tt, ok := e.TypeTable(); ok {
		if err := s.resolveTypeTable(tt, e.Loc()); err != nil {
			return err
		}
	}
	for _, ic := range e.IdentityConstraints() {
		if err := s.resolveKeyref(ic); err != nil {
			return err
		}
	}
	return nil
}

// resolveTypeTable reaches each Type Alternative's {type definition} slot
// (§3.12.2 declare-ta maps the type/@type of an <alternative> via [·resolved·]). Both the
// {alternatives} members and the {default type definition} carry the same
// TypeDefinitionOrRef slot, so both go through the shared descent's walkTypeDefinition —
// the one implementation that is total over the sum. A by-name arm naming nothing is
// §5.3's ·absent· {type definition} and is charged nothing; declare-ta's INLINE arm is
// "the type definition corresponding to the complexType or simpleType among the
// children", a direct structural mapping, and the same call enters that anonymous type's
// OWN references instead. A {default type definition} §3.3.2.1 case 2 synthesized carries
// the declaring element's own slot, so resolveElementDecl reaches that component twice;
// the descent writes nothing and answers the same either way.
//
// loc is the owning element declaration's position: neither TypeTable nor
// TypeAlternative retains one, and both live inside the <element> the position
// names.
//
// EXISTENCE ONLY. e-props-correct clause 7 (§3.3.6.1) predicates over the very
// alternatives this resolves — each must be ·validly substitutable· for the
// declaration's own {type definition}, or be ·xs:error· — and is charged in
// Phase D instead (checkTypeTableSubstitutability, typetablesubstitutable.go).
// It cannot be charged here: ValidlySubstitutable walks {base type definition}
// chains and union membership with no visited set, which is licensed only after
// Phase B's acyclicity checks, and a circular chain is still representable while
// this phase runs.
func (s *Schema) resolveTypeTable(tt TypeTable, loc xsderr.Loc) error {
	w := s.referenceWalk()
	for _, alt := range tt.Alternatives() {
		if err := w.walkTypeDefinition(alt.TypeDefinition(), loc, "type alternative {type definition}"); err != nil {
			return err
		}
	}
	return w.walkTypeDefinition(tt.DefaultTypeDefinition().TypeDefinition(), loc, "type table {default type definition}")
}

// checkSimpleTypeDerivations is Phase D's simple-type step: it puts every Simple
// Type Definition the finalized Schema reaches to TWO charges, in this order.
//
//  1. [SimpleType.CheckDerivation] (derivation.go), the graph half — the
//     cross-reference constraints that need the RESOLVED {base type
//     definition}: st-props-correct clauses 1, 3 and 5, and the structural
//     sub-clauses of Derivation Valid (Restriction, Simple) (§3.16.6.2,
//     cos-st-restricts). Those ran inside NewSimpleType while the base was a
//     live pointer; a deferred base cannot be followed at construction, so they
//     moved here, to the one pass that holds a resolver.
//  2. the installed [SimpleTypeRestrictionChecker], which charges the
//     facet-VALUE half — clause 1.3.1's atomic applicability and the value-space
//     constraints of clauses 1.3.2 / 2.2.2.5 / 3.2.2.5.
//
// The order is not arbitrary: the checker's implementation reads {variety},
// {primitive type definition} and {facets} off the same chain, so running the
// graph half first means it is handed a type whose chain has already been
// proved well formed, and a schema breaking both says so under the more
// structural rule.
//
// Consolidating the charge here is this project's architecture, not a spec
// mandate: §4.1 is explicitly agnostic about when a processor assembles a
// schema, and the reason to make it a finalize pass is PRINCIPLES 9 — one place
// that sees the whole assembled graph beats a charge scattered over whichever
// producer happened to build a component.
//
// THE DESCENT INVENTORY IS SIX SLOTS, and it is exhaustive over the places a
// *SimpleType can live. It is written out here, with one reason per slot, so that
// the next slot added to this package is forced to be considered against it — a
// walk that reaches only the indexed types is a FALSE ACCEPT, since an anonymous
// simple type is in no index at all:
//
//  1. s.types roots — the named types. The only slot an index-only walk reaches,
//     and the only one from which every other is entered.
//  2. the {base type definition} hop, walked through its OWNED arm only. An
//     ANONYMOUS inline base hangs off this field and nowhere else: no index holds
//     it and no declaration slot names it, so dropping this hop would lose
//     coverage of every <simpleType><restriction><simpleType> base. A BY-NAME
//     arm is deliberately not followed — it names a top-level type this pass
//     reaches through slot 1 in its own right, so following it would re-charge
//     the same component once per type deriving from it.
//  3. SimpleContent.{simple type definition} (complextype.go). The shared
//     descent hands this slot to Phase A too, which enters it to reach the
//     owner-of-owner invariant on any {type definition} slot below; this pass
//     descends it to CHARGE the two derivation halves. Neither visit substitutes
//     for the other. On the <simpleContent> <extension> alternant the slot holds
//     an EXISTING component (parser/produce_complex.go's
//     simpleContentSimpleType, tableau cases 3-5), usually a named one, so this
//     descent re-charges a component slot 1 also reaches — harmless, and already
//     licensed by the NO VISITED SET paragraph below. On the <restriction>
//     alternant it holds the ANONYMOUS type §3.4.2.2 cases 1-2 synthesize from
//     that restriction's facet children, which no index reaches at all, so
//     dropping this hop would be a false accept for every one of them —
//     including the case-2 shape with no inline <simpleType>, whose whole
//     rejection is the CheckDerivation charged from here (§3.4.2.2's own Note:
//     it "fails to obey the constraints on simple type definitions").
//  4. ListDerivation.Item (simpletype.go). An anonymous item type is in no index.
//  5. UnionDerivation.Members (simpletype.go). Ditto, for every member, walked in
//     the declared order the property preserves (STYLE D2).
//  6. the InlineTypeDefinition/*SimpleType arm of a {type definition} or {base
//     type definition} slot — the declaration-slot inline hop: an <element> or
//     <attribute> whose type is written out in place. The shared descent enters
//     the same arm for Phase A, and slot 3's split applies here unchanged: that
//     arm resolves the inline type's own by-name base, this pass charges its
//     derivation.
//
// Reaching slots 3 and 6 means walking the same tree Phase A and Phase E walk,
// and this pass walks it by the same code (componentwalk.go), filling in only the
// simpleType charge. Its ROOTS are its own: types, then element declarations,
// then attribute declarations, then model group definitions, then attribute group
// definitions. Attribute group definitions are rooted for the reason Phase A and
// Phase E root them (valueconstraintvalid.go) plus one of this pass's own: the
// produce-time call this pass replaces charged every simple type the producer
// CONSTRUCTED, whatever slot it ended up in, so a walk that skipped a slot the
// producer can fill would be a silent regression.
//
// NO VISITED SET (STYLE D4, and STYLE D3's no-memoized-cache-without-a-profile).
// A shared base is re-visited once per type that derives from it, which is
// O(n·depth) and correct; memoizing the verdict would be a cache with no measured
// hot path, and one whose keys are the very pointers the walk is enumerating.
//
// TERMINATION, in two halves. This pass's own DESCENT follows only OWNED
// pointers — the owned arm of each of the three SimpleTypeOrRef slots (base,
// ListDerivation.Item, UnionDerivation.Members) and the by-value nesting of the
// complex-type slots — and an owned component must pre-exist the slot holding
// it, so the descent is a finite tree (PRINCIPLES 9). The by-NAME edges are
// never followed here: a SimpleTypeRef base, item or member, a TypeDefinitionRef
// base, an <element ref>, a <group ref> each name a component this pass reaches
// in its own right. What CheckDerivation does with the chain is the other half,
// and it is NOT by construction: it walks the by-name base chain unguarded,
// which is licensed by Phase B's checkSimpleBaseAcyclic having run first (see
// resolve's phase narration). The complex-type inline base IS followed, which
// Phase B has likewise made acyclic.
//
// Roots are walked in document order and a base chain bottom-up — a type's base,
// item and members are charged before the type itself — so the first reported
// failure is deterministic (STYLE D1/D2) and is the most basic one: a fault in a
// base is reported against the base, not against everything derived from it.
func (s *Schema) checkSimpleTypeDerivations() error {
	w := componentWalk{simpleType: s.checkSimpleTypeGraph}
	for _, t := range s.types {
		if err := w.walkTypeRoot(t); err != nil {
			return err
		}
	}
	for _, e := range s.elements {
		if err := w.walkElementDeclaration(e); err != nil {
			return err
		}
	}
	for _, a := range s.attributes {
		if err := w.walkAttributeDeclaration(a); err != nil {
			return err
		}
	}
	for _, mgd := range s.modelGroups {
		if err := w.walkModelGroup(mgd.ModelGroup(), mgd.Loc()); err != nil {
			return err
		}
	}
	for _, g := range s.attributeGroups {
		if err := w.walkAttributeGroupDefinition(g); err != nil {
			return err
		}
	}
	return nil
}

// checkSimpleTypeGraph charges both halves against t and everything reachable
// from it: inventory slots 2 (the OWNED {base type definition} hop), 4
// (ListDerivation.Item) and 5 (UnionDerivation.Members). The derived readers are
// deliberately not used to enumerate them — Item and Members report the property
// a restriction INHERITS from its base, which this walk already reaches through
// the base hop, whereas the stored arm is the only thing that names the component
// this type itself owns.
//
// All three slots are descended by OWNERSHIP alone (ownedSimpleType): a by-name
// item, member or base names a top-level type this pass walks in its own right,
// exactly as the base slot has done since #636, so following one would re-walk
// it once per referring type — and would recurse forever on a schema whose
// itemType= names the very list declaring it, a shape CheckDerivation rejects
// under cos-st-restricts clause 2.1 without descending at all.
//
// THE OWNED-CHILD DESCENT RUNS BEFORE THE usable GATE, and the order is the
// whole point of the gate's position: an unusable type is charged nothing (§5.3),
// but the components it OWNS are separate Simple Type Definitions that the schema
// reaches, and each is judged on its own usability when the descent hands it
// here. Gating the descent too would lose every verdict below an unusable type.
//
// A nil t is an owned arm that was absent, not a fault: only the base slot may be
// absent, and st-props-correct clause 1 owns that verdict inside CheckDerivation,
// so re-charging it here would name a rule this pass does not own (STYLE E2).
func (s *Schema) checkSimpleTypeGraph(t *SimpleType) error {
	if t == nil {
		return nil
	}
	if err := s.checkSimpleTypeGraph(ownedSimpleType(t.base)); err != nil {
		return err
	}
	switch d := t.derivation.(type) {
	case ListDerivation:
		if err := s.checkSimpleTypeGraph(ownedSimpleType(d.Item)); err != nil {
			return err
		}
	case UnionDerivation:
		for _, m := range d.Members {
			if err := s.checkSimpleTypeGraph(ownedSimpleType(m)); err != nil {
				return err
			}
		}
	}
	if !usable(s, t) {
		return nil // §5.3: an unusable simple type is charged nothing at construction
	}
	if err := t.CheckDerivation(s); err != nil {
		return err
	}
	return s.restrictionChecker.CheckRestriction(s, t)
}

// checkComplexBaseAcyclic is Phase B for the complex-type base chain
// (ct-props-correct §3.4.6.1 clause 3): a complex type's {base type definition}
// chain must terminate, the sole permitted self-derivation being xs:anyType
// (§3.4.7). Because each type has at most one base, the "graph" is functional
// (out-degree ≤ 1), so a cycle is a repeated name on a single chain walk.
//
// The path map is a per-walk, finalize-scoped cycle guard: it lives entirely
// inside this function and is discarded when resolve returns (PRINCIPLES 9). It
// is NEVER threaded into any later traversal — doc.go promises the Walker needs
// "no visited set beyond the path-scoped guard" — so no runtime traversal
// inherits it.
//
// The walk follows the {base type definition} SLOT, both of its arms. Descending
// the InlineTypeDefinition arm is not optional and is not defensive: a redefining
// complex type's base is the anonymous src-expredef clause 1.1 original, whose
// OWN base is a name again, so stopping at the inline hop would leave every cycle
// running through a redefinition undetected — and the two attribute folds, which
// do follow that hop, would then recurse without bound (#505). It costs no extra
// guard: an inline base is a value owned by exactly one slot, so the nesting is
// finite by construction and cannot itself close a loop.
//
// Roots are iterated in document order (STYLE D2); an anonymous node (zero name)
// is walked but never recorded (it can be no NAMED base's target, having no name
// to be referenced by, and its one owner is the node the walk just came from), so
// the first reported cycle is deterministic. path is read only by key, never
// ranged, so which cycle member is named does not depend on Go map iteration
// order either.
//
// The cycle is symmetric, so any member on it would be a defensible position;
// nextCT is the one already in hand at the rejection — the ComplexType the
// offending base= resolved to, whose {name} the message also prints — so it is
// charged without a second lookup.
//
// This is ONE of two entry points on clause 3, with identical verdicts, for the
// two construction paths. The other is parser's buildComplexType, whose on-stack
// memo sentinel must reject a cycle to TERMINATE demand-driven base construction
// before a component ever reaches a builder. This one is what keeps xsd
// self-defending on the programmatic SchemaBuilder path, which has no producer;
// it stays whether or not the parser guards its own recursion.
func (s *Schema) checkComplexBaseAcyclic() error {
	for _, t := range s.types {
		ct, ok := t.(ComplexType)
		if !ok {
			continue // *SimpleType base chains are acyclic by construction
		}
		path := map[QName]bool{}
		cur := ct
		for {
			name := cur.Name()
			if ref, byName := cur.Base().(TypeDefinitionRef); byName && name == anyTypeName && ref.Name == anyTypeName {
				break // §3.4.7: xs:anyType is the one permitted self-based type
			}
			if name != (QName{}) {
				path[name] = true
			}
			next, ok := s.ResolvedType(cur.Base())
			if !ok {
				break // an absent base ends the chain, and an ·absent· one (§5.3) too
			}
			nextCT, ok := next.(ComplexType)
			if !ok {
				break // base is a simple type: chain terminates
			}
			if nextName := nextCT.Name(); nextName != (QName{}) && path[nextName] {
				return xsderr.New(ruleCTPropsCorrect, nextCT.Loc(),
					"complex type %s participates in a circular {base type definition} chain, but ct-props-correct clause 3 forbids it (only xs:anyType may be its own base)", nextName)
			}
			cur = nextCT
		}
	}
	return nil
}

// checkSimpleBaseAcyclic is Phase B for the SIMPLE-type base chain
// (st-props-correct §3.16.6.1 clause 2): "All simple type definitions are, or
// are ·derived· ultimately from, ·xs:anySimpleType· (so circular definitions are
// disallowed). That is, it is possible to reach a primitive datatype or
// ·xs:anySimpleType· by following the {base type definition} zero or more
// times." It is structurally the complex-type twin, checkComplexBaseAcyclic
// (below), and copies its colour-map idiom deliberately; the two differ only in
// that no simple type is permitted to be its own base, so there is no
// xs:anyType-style exemption here.
//
// The rule became a real check with this landing and not before: while the base
// slot held a live pointer that had to pre-exist the type naming it, a cycle was
// unconstructible (PRINCIPLES 9), so the clause had nothing to reject. A
// SimpleTypeRef base is resolved by name at finalize, which makes A-derives-from-B
// -derives-from-A representable, and unguarded chain walks are exactly what
// derivation.go is built out of.
//
// It TRAVERSES BOTH ARMS of the slot, through SimpleType.Base. An owned arm
// alone cannot close a cycle, but a MIXED owned-then-named chain can — an
// anonymous inline base whose own base= names the type that owns it — so
// stopping at the owned hop would miss every cycle running through a
// redefinition, which is the same reason checkComplexBaseAcyclic descends its
// own inline hop (§4.2.4 src-expredef clause 1.1 makes a redefining type's base
// an anonymous original with a by-name base of its own).
//
// Because each type has at most one base, the graph is functional (out-degree ≤
// 1), so a cycle is a repeated NODE on a single chain walk. Node identity is the
// POINTER, not the {name}: an anonymous node has no name to be keyed by, and
// pointer identity is what SimpleType's contract already makes load-bearing. The
// path map is a per-walk, finalize-scoped guard that lives entirely inside this
// function and is discarded when resolve returns (PRINCIPLES 9); it is never
// threaded into any later traversal.
//
// Roots are iterated in document order (STYLE D2) and path is read only by key,
// never ranged, so the first reported cycle is deterministic and which member it
// names does not depend on Go map iteration order. The member reported is the
// one the walk RE-ENTERS, which is on the cycle by construction and is the node
// already in hand.
//
// A resolution failure ends the walk rather than being charged: it is §5.3's
// ·absent· {base type definition}, which nothing charges (usable), and charging
// it under THIS function's rule would report a missing component as a
// circularity (STYLE E2).
func (s *Schema) checkSimpleBaseAcyclic() error {
	for _, t := range s.types {
		st, ok := t.(*SimpleType)
		if !ok {
			continue // a ComplexType's chain is checkComplexBaseAcyclic's
		}
		path := map[*SimpleType]bool{}
		for cur := st; cur != nil; {
			if path[cur] {
				return xsderr.New(ruleSTPropsCorrect, cur.Loc(),
					"%s participates in a circular {base type definition} chain, but st-props-correct clause 2 requires every simple type to reach a primitive datatype or xs:anySimpleType by following {base type definition} zero or more times", simpleTypeLabel(cur))
			}
			path[cur] = true
			next, err := cur.Base(s)
			if err != nil {
				break // an ·absent· base (§5.3) ends the chain; no cycle runs through it
			}
			cur = next
		}
	}
	return nil
}

// checkUnionMembershipAcyclic is Phase B for union-membership circularity
// (cos-st-restricts §3.16.6.2 clause 3.3, no-self-membership): "Neither D nor
// any type ·derived· from it is a member of its own transitive membership." Both
// conjuncts are read against the SAME set — D's ·transitive membership·
// (Datatypes dt-transitivemembership: D's own {member type definitions}, and
// theirs, recursively) — and "derived from D" is base-chain reachability alone,
// never the membership or item-type relation, which other clauses govern
// (src-simple-type clause 4, cos-st-restricts clause 2.1).
//
// So the two conjuncts collapse into ONE test, which is what this pass runs: for
// every member M in D's transitive membership, M's {base type definition} chain
// — M ITSELF first — must not reach D. M = D at step zero is conjunct 1 (D is
// its own member), a later step is conjunct 2 (a type derived from D is), and no
// third reading is admitted: "derived or constructed from", the broader wording
// dt-transitivemembership's companion sentence uses, would reject the
// list-of-union and union-member-of-union shapes clause 2.1 and clause 3.1
// already judge on their own terms.
//
// IT IS NOT checkSimpleBaseAcyclic's CHAIN WALK generalized. Membership has
// out-degree > 1, so a cycle is not a repeated node on a single walk and a path
// map keyed by the walk's own chain cannot see one; this is a recursive DFS over
// the membership graph, like checkModelGroupsAcyclic's, with
// checkSimpleBaseAcyclic's POINTER identity for nodes — an anonymous inline
// member has no {name} to be keyed by.
//
// Its map is a plain PER-ROOT visited set rather than that DFS's shared
// onStack/done colouring, and both halves of that are load-bearing. A colouring
// answers "does this graph contain a cycle", which is conjunct 1 alone; the test
// run here is per D and answers both conjuncts together, so what it needs
// recorded is "already tested under THIS D". And it cannot be shared across
// roots for the same reason: conjunct 2's verdict on M depends on which D the
// walk reached M from, so a node marked finished under one root would carry that
// root's answer to every later one — an M first reached from a root whose chain
// it does not touch would never be re-tested against the D that it does. Each
// map is created and discarded inside this pass, threaded into no later
// traversal (PRINCIPLES 9).
//
// ROOTS ARE THE NAMED TOP-LEVEL TYPES, in document order (STYLE D2), and no
// violation escapes that. A cycle needs an edge INTO an already-visited node; an
// anonymous type is held by exactly one owning slot, and only a by-name
// SimpleTypeRef — which resolves through the type index — can name a second
// referrer, so every membership cycle runs through a type this loop roots at. A
// conjunct-2 violation whose D is an ANONYMOUS union is charged against the type
// that OWNS D as its {base type definition}: a ·restriction· reports its base's
// membership, so that type's transitive membership is D's, and the offending
// member's chain passes through it on the way to D. Every map here is read only
// by key and never ranged, so the first reported rejection is deterministic
// (STYLE D1).
//
// THIS PASS IS WHAT MAKES THE UNGUARDED MEMBERSHIP WALKS TERMINATE —
// unionMembershipHasList and derivedOKSimple (derivation.go) and needsContext
// (value/valuespace.go) all follow {member type definitions} with no visited set
// — exactly as checkSimpleBaseAcyclic licenses the unguarded base-chain walks.
// It runs after checkSimpleBaseAcyclic, whose verdict it presupposes in turn:
// SimpleType.Members reports a ·restriction·'s membership by following its base,
// and the conjunct-2 test walks that chain outright.
func (s *Schema) checkUnionMembershipAcyclic() error {
	for _, t := range s.types {
		st, ok := t.(*SimpleType)
		if !ok {
			continue // only a simple type has a membership
		}
		if err := s.checkOwnTransitiveMembership(st); err != nil {
			return err
		}
	}
	return nil
}

// checkOwnTransitiveMembership charges cos-st-restricts clause 3.3 against one
// D: it walks D's ·transitive membership· depth-first in declared member order
// and rejects the first member whose {base type definition} chain reaches D. A
// type with no membership — every {variety} but union — walks nothing, so no
// {variety} test gates the call (STYLE D3).
//
// A resolution failure ends that branch of the walk rather than being charged
// again (membersOrNone): an unresolvable member is §5.3's ·absent· value, which
// nothing charges (usable), and charging it under THIS function's rule would
// report a missing component as a membership cycle (STYLE E2). The base-chain
// walk needs no guard of its own — checkSimpleBaseAcyclic has already rejected
// a circular chain.
func (s *Schema) checkOwnTransitiveMembership(d *SimpleType) error {
	visited := map[*SimpleType]bool{}
	var walk func(t *SimpleType) error
	walk = func(t *SimpleType) error {
		for _, m := range membersOrNone(s, t) {
			if visited[m] {
				continue
			}
			visited[m] = true
			if err := checkMemberNotDerivedFrom(s, m, d); err != nil {
				return err
			}
			if err := walk(m); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(d)
}

// usable reports whether every by-name reference reachable from t's {base type
// definition}, {item type definition} and {member type definitions} resolves
// against r. A false answer is §5.3 ·absent· — "unusable" is §5.3's own word —
// and NOT a schema error: the ·QName· is retained (§4.2.3) and the consequence is
// deferred to ·assessment· (#250).
//
// REACHABLE MEANS EXACTLY WHAT THE SIX READERS REACH, which is what makes this
// predicate decide their error and not something adjacent to it. Variety,
// Primitive, Base and EffectiveFacets follow the {base type definition} chain;
// Item and Members follow that chain to the nearest list or union alternative and
// then read that alternative's own slot, without following IT. So the walk below
// is the base chain, testing each level's own item and member slots as it passes,
// and nothing else. It is the same "resolve the slot without following it" split
// the rest of this file makes, in predicate form.
//
// It needs no visited set (STYLE D4, PRINCIPLES 9): the one edge it FOLLOWS is
// the base chain, which Phase B's checkSimpleBaseAcyclic has already made
// acyclic, and the item and member slots are tested in place. A transitive
// reading — following a resolved item or member and asking whether IT is usable —
// would have neither property, because no phase makes the ITEM edge acyclic: a
// list whose itemType= names itself is representable right up to
// CheckDerivation's cos-st-restricts clause 2.1 verdict, which is charged after
// this gate, not before it.
//
// GAP(xsd): a simple type whose item or member RESOLVES to a type that is itself
// unusable is still charged here, where §5.3's closing paragraph puts "any types
// derived or constructed from them" in the same ·absent· class. The residue is
// the transitive reading above, and closing it needs the item edge proved acyclic
// first; it rides with the rest of §5.3 on #250.
//
// GAP(xsd): the six readers this predicate is defined over still return an
// `error` for a state that is not one. At construction the gate below answers for
// it — an unusable type is charged nothing — but an out-of-package consumer
// reaching one gets that error where §5.3 mandates a cvc clause-1 failure
// (Attribute Locally Valid §3.2.4.1 clause 1, Element Locally Valid (Element)
// §3.3.4.3 clause 1) and, for an element item, a fallback to ·lax assessment·.
// Those cvc clauses do not exist in this module yet, so there is nothing for such
// a consumer to be judged against; #250 owns both the clauses and the signature
// question they reopen.
//
// The one-error-source invariant the six readers document is what this predicate
// rests on: their only non-nil error is simpleTypeOfRef's, so a false answer here
// can only ever be a §5.3 ·absent· reference and never a swallowed rule failure.
// TestSimpleTypeReadersHaveOneErrorSource pins it.
func usable(r TypeResolver, t *SimpleType) bool {
	for cur := t; cur != nil; {
		switch d := cur.derivation.(type) {
		case ListDerivation:
			if _, err := simpleTypeOfRef(r, d.Item, cur.loc, ""); err != nil {
				return false
			}
		case UnionDerivation:
			for _, m := range d.Members {
				if _, err := simpleTypeOfRef(r, m, cur.loc, ""); err != nil {
					return false
				}
			}
		}
		next, err := simpleTypeOfRef(r, cur.base, cur.loc, "")
		if err != nil {
			return false
		}
		cur = next
	}
	return true
}

// membersOrNone returns t's {member type definitions}, or none when one of them
// cannot be resolved; baseOrNone is its {base type definition} twin. Neither
// swallows a verdict: an unresolvable member or base is §5.3's ·absent· value,
// which no phase charges (usable), so the branch exists to END the walk rather
// than to decide anything — charging a rule of this function's own against a slot
// that holds no component would name a rule it does not own (STYLE E2).
func membersOrNone(r TypeResolver, t *SimpleType) []*SimpleType {
	members, err := t.Members(r)
	if err != nil {
		return nil
	}
	return members
}

// baseOrNone returns t's resolved {base type definition}, or nil when it is
// absent or unresolvable — the end of the chain either way. See membersOrNone.
func baseOrNone(r TypeResolver, t *SimpleType) *SimpleType {
	base, err := t.Base(r)
	if err != nil {
		return nil
	}
	return base
}

// checkMemberNotDerivedFrom rejects m — a member of d's ·transitive membership· —
// when m IS d or is ·derived· from d, which is the whole of cos-st-restricts
// clause 3.3 once the transitive membership is in hand (see
// checkUnionMembershipAcyclic). The two are told apart in the message but not in
// the verdict: one clause, one rule ID (STYLE E2). The rejection is charged to
// the position of the member that closes the loop, which is the node in hand and
// on the loop by construction.
func checkMemberNotDerivedFrom(r TypeResolver, m, d *SimpleType) error {
	for cur := m; cur != nil; cur = baseOrNone(r, cur) {
		if cur != d {
			continue
		}
		if m == d {
			return xsderr.New(ruleCosSTRestricts, m.Loc(),
				"%s is a member of its own transitive membership, but cos-st-restricts clause 3.3 forbids a union to be a member of it", simpleTypeLabel(d))
		}
		return xsderr.New(ruleCosSTRestricts, m.Loc(),
			"%s is a member of the transitive membership of %s, which it is derived from, but cos-st-restricts clause 3.3 forbids a type derived from a union to be a member of that union's transitive membership", simpleTypeLabel(m), simpleTypeLabel(d))
	}
	return nil
}

// checkModelGroupsAcyclic is Phase B for <group ref> circularity
// (mg-props-correct §3.8.6.1 clause 2, no-circular-groups): within the
// {particles} of a group there is no particle at any depth whose {term} is the
// group itself. Nodes are the top-level model group definitions; an edge M→N
// exists for each ModelGroupRef to N reachable through M's particle tree
// (inline model groups are descended; a ModelGroupRef is a leaf edge, resolved
// by visiting its target definition, not descended in place).
//
// The color map is a finalize-scoped cycle guard (0 unvisited, 1 on the current
// DFS stack, 2 finished): it lives only in this function and is discarded when
// resolve returns (PRINCIPLES 9), never threaded into any later traversal.
// Definitions are iterated, and each definition's out-refs collected, in
// document order (STYLE D2), so the first reported cycle is deterministic. The
// color map is read only by key and never ranged, so neither the reported name nor
// the reported position depends on Go map iteration order.
func (s *Schema) checkModelGroupsAcyclic() error {
	// The map's zero value is the implicit "unvisited" state; onStack/done are
	// the two recorded states.
	const (
		onStack = 1
		done    = 2
	)
	color := map[QName]int{}
	var visit func(name QName) error
	visit = func(name QName) error {
		switch color[name] {
		case done:
			return nil
		case onStack:
			// A name is onStack only between its own visit's two color writes, and
			// recursion in that window runs solely through its out-edges — which the
			// index lookup below supplies only for a name the index HOLDS. So a name
			// absent from the index goes straight to done and can never be
			// re-encountered onStack: the lookup here always hits, and the position is
			// the offending definition's own <group> element.
			return xsderr.New(ruleMgPropsCorrect, s.modelGroupIndex[name].Loc(),
				"model group definition %s participates in a circular <group ref> chain, but mg-props-correct clause 2 forbids circular groups", name)
		}
		color[name] = onStack
		if mgd, ok := s.modelGroupIndex[name]; ok {
			for _, ref := range groupRefsIn(mgd.ModelGroup()) {
				if err := visit(ref); err != nil {
					return err
				}
			}
		}
		color[name] = done
		return nil
	}
	for _, mgd := range s.modelGroups {
		if err := visit(mgd.Name()); err != nil {
			return err
		}
	}
	return nil
}

// groupRefsIn returns, in document order, the name of every <group ref>
// (ModelGroupRef) reachable through g's particle tree without descending into a
// referenced definition (that edge is followed by the DFS, not inlined here).
func groupRefsIn(g ModelGroup) []QName {
	var refs []QName
	for _, p := range g.Particles() {
		collectGroupRefs(p.Term(), &refs)
	}
	return refs
}

// collectGroupRefs appends the ModelGroupRef names in t's subtree (document
// order). Inline model groups are descended; an element (declaration or ref) is
// not — a nested element's own references belong to that element's resolution,
// not this group's reference graph.
func collectGroupRefs(t TermOrRef, refs *[]QName) {
	switch t := t.(type) {
	case ResolvedTerm:
		switch inner := t.Term.(type) {
		case ModelGroup:
			for _, p := range inner.Particles() {
				collectGroupRefs(p.Term(), refs)
			}
		case ElementDeclaration, Wildcard:
			// not a group reference
		default:
			panic("xsd: collectGroupRefs: non-exhaustive Term switch")
		}
	case ModelGroupRef:
		*refs = append(*refs, t.Name)
	case ElementDeclarationRef:
		// not a group reference
	default:
		panic("xsd: collectGroupRefs: non-exhaustive TermOrRef switch")
	}
}

// checkSubstitutionGroupsAcyclic is Phase B for substitution-group circularity
// (e-props-correct §3.3.6.1 clause 5): it must not be possible to return to an
// element E by repeatedly following any member of its {substitution group
// affiliations}. Nodes are the top-level element declarations; an edge E→F
// exists for each affiliation name F of E.
//
// The color map is a finalize-scoped cycle guard (same 0/1/2 scheme as
// checkModelGroupsAcyclic): it lives only in this function and is discarded when
// resolve returns (PRINCIPLES 9), never threaded into a later traversal.
// Elements are iterated, and each element's affiliations followed, in document
// order (STYLE D2), so the first reported cycle is deterministic. The color map is
// read only by key and never ranged, so neither the reported name nor the reported
// position depends on Go map iteration order.
func (s *Schema) checkSubstitutionGroupsAcyclic() error {
	// The map's zero value is the implicit "unvisited" state; onStack/done are
	// the two recorded states.
	const (
		onStack = 1
		done    = 2
	)
	color := map[QName]int{}
	var visit func(name QName) error
	visit = func(name QName) error {
		switch color[name] {
		case done:
			return nil
		case onStack:
			// The index lookup always hits, by checkModelGroupsAcyclic's argument: a
			// name the index does not hold contributes no out-edges, so it reaches done
			// before any recursion can return to it. That matters more here than there,
			// because a §5.3 dangling affiliation legitimately names no declaration and
			// Phase A does not reject it (resolveElementDecl).
			return xsderr.New(ruleEPropsCorrect, s.elementIndex[name].Loc(),
				"element declaration %s participates in a circular {substitution group affiliations} chain, but e-props-correct clause 5 forbids circular substitution groups", name)
		}
		color[name] = onStack
		if e, ok := s.elementIndex[name]; ok {
			for _, aff := range e.SubstitutionGroupAffiliationNames() {
				if err := visit(aff); err != nil {
					return err
				}
			}
		}
		color[name] = done
		return nil
	}
	for _, e := range s.elements {
		if err := visit(e.Name()); err != nil {
			return err
		}
	}
	return nil
}
