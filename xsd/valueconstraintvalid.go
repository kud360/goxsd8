package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file is Phase E of the finalize resolution pass (resolve.go): the
// value-constraint validity constraints neither NewAttributeDeclaration,
// NewAttributeUse, nor any earlier phase can decide, because each needs a
// RESOLVED {type definition} or {attribute declaration}. Four clauses land here,
// three of them reading the same two components:
//
//   - a-props-correct (§3.2.6.1) clause 2: "if there is a {value constraint},
//     then it is a valid default with respect to the {type definition} as
//     defined in Simple Default Valid (§3.2.6.2)."
//   - au-props-correct (§3.5.6) clause 2: "If U.{value constraint} is not
//     ·absent·, then it is a valid default with respect to U.{attribute
//     declaration}.{type definition} as defined in Simple Default Valid."
//   - au-props-correct clause 3, below.
//   - e-props-correct (§3.3.6.1) clause 2, the ELEMENT-side counterpart of the
//     first two, which routes through Element Default Valid (Immediate)
//     (§3.3.6.2, cos-valid-default) rather than reaching Simple Default Valid
//     directly. Its predicate lives in elementdefaultvalid.go; only its walk
//     sites are here, because they are the SAME descent (STYLE T4).
//
// The two attribute clause 2s are ONE predicate — cos-valid-simple-default over a
// (type, value constraint) pair — differing only in which type and which
// constraint they pair. They therefore share one implementation, checkSimpleDefault
// (STYLE T4, #371); a second, parallel one for either call site would be a design
// failure. cos-valid-default's clause 1 is the THIRD caller of that same
// implementation (#463), reached once the element-side case analysis has picked
// out which simple type governs. All are gated on PRESENCE: with no {value
// constraint} the clause is not reached at all, which is not the same as
// reached-and-vacuously-satisfied.
//
// Clause 3, verbatim: "If U.{attribute declaration} has {value constraint}.
// {variety} = fixed and U itself has a {value constraint}, then U.{value
// constraint}.{variety} = fixed and U.{value constraint}.{value} is identical to
// U.{attribute declaration}.{value constraint}.{value}."
//
// Two things put it here rather than in the constructor:
//
//   - the {attribute declaration} half. For the AttributeDeclarationRef variant
//     the declaration is a deferred QName at construction time, so its {value
//     constraint} is unreadable there; only a finalized Schema can resolve it.
//     NewAttributeUse therefore decides the variety half for the Local variant
//     ONLY, and this phase decides it for both — the rule text draws no variant
//     distinction, so the Local case is re-tested here (vacuously, since the
//     constructor already rejected it) rather than special-cased away (STYLE T4).
//   - the {value} half. {value} is an ·actual value· (key-vv §3.2.1), a member of
//     the type's value space, not the {lexical form} ValueConstraint carries — so
//     deciding "identical" needs a lexical→value mapping, which this package
//     cannot have (PRINCIPLES 1). It asks the installed ValueSpace instead, and
//     accepts whenever the answer is undecided.
//
// The two walks. A GLOBAL Attribute Declaration is charged a-props-correct clause
// 2 by its own walk over the schema's {attribute declarations}
// (checkAttributeDeclarationDefaults), never again per use that references it: one
// charge per component keeps the first reported failure deterministic and does not
// re-run the same verdict once per referencing site. A LOCAL declaration is not in
// that table at all — its sole owner is the AttributeUse holding it — so it is
// charged from the use side instead. Together the two reach every declaration at
// least once, at a stable Loc; neither alone does. Not exactly once: since #401
// materialised inherited attribute uses, a LOCAL declaration owned by an
// inherited use is re-checked at every complex type that inherits it (see
// checkComponentValueConstraints below, which spells out why the repeats are
// harmless — the charge is at the declaration's OWN Loc, so neither the verdict
// nor the reported position moves).
//
// THE ELEMENT SIDE DOES NOT SPLIT THAT WAY, and the reason is a measured fact
// about this codebase, not a symmetry argument (#463). SchemaBuilder.AddElement
// has exactly one producer caller — the top-level <element> arm of
// parser/produce.go — so s.elements holds GLOBAL declarations only; a local
// <element name="..."> becomes a sibling Element Declaration inside its
// particle's {term} (parser/produce_complex.go's produceLocalElement) and enters
// no table at all. It nonetheless carries a {value constraint}, mapped by the
// same valueConstraintOf the global path uses, and e-props-correct is charged
// against "all element declarations" with no scope exemption (§3.3.6, oracle
// grounding on #463). Measuring the W3C suite settles the size of the gap: of
// the 242 <element> items carrying default= or fixed= across testdata/xsdtests,
// 102 (in 52 schema documents, at nesting depths 4 to 6) are local. So clause 2
// is charged by DESCENT — the walk below, which already reaches every element
// declaration on its way to their inline types' attribute uses — and never by a
// second loop over s.elements, which would miss two fifths of the declarations
// the rule quantifies over. This is the opposite conclusion from
// checkSubstitutionGroupTypes' (clause 4 IS complete over s.elements alone), and
// deliberately so: that argument rests on clause 3 confining a {substitution
// group affiliations} to a global scope, and no clause confines a {value
// constraint}.
//
// D4 (no traversal state): the shared descent carries no visited set, for the
// reason componentwalk.go states — it follows only BY-VALUE structure and never
// a by-name ref, which is what makes the structure a finite tree.

// checkComponentValueConstraints is Phase E's DESCENDING walk: it charges
// au-props-correct clauses 2 and 3 — and, for a use owning a LOCAL declaration,
// that declaration's own a-props-correct clause 2 — against every Attribute Use
// the compiled schema holds, and e-props-correct clause 2 against every Element
// Declaration it holds, walking in document order so the first reported failure
// is deterministic (STYLE D1/D2 — no index map is ranged).
//
// The two duties share ONE descent rather than getting a walk each (STYLE T4):
// every element declaration the attribute-use side descends THROUGH on its way to
// an inline complex type is exactly an element declaration clause 2 quantifies
// over, so a second traversal of the same tree would be a parallel copy of five
// functions that visits the same components in the same order.
//
// The descent itself is componentwalk.go's, shared with every other phase that
// has one, so this pass supplies only the two charges above and inherits which
// components exist — including a redefine original reached through {base type
// definition}, which a hand-written copy of this walk missed (#843). The roots
// are this phase's own: top-level type definitions, then top-level element
// declarations (whose inline anonymous complex types carry attribute uses of
// their own), then top-level model group definitions (whose particles can carry
// element declarations with inline complex types), and finally — where Phase A
// stops — top-level attribute group definitions. Between those roots and that
// descent, every element declaration in the schema is charged exactly once, at
// its own Loc, which is what makes clause 2's quantifier ("all element
// declarations", §3.3.6) complete; see this file's head for the measurement
// behind that shape.
//
// Since #401 materialised §3.4.2.4 clause 3, an INHERITED attribute use is a
// member of the deriving type's {attribute uses} too, so it is re-checked at
// every type that inherits it and charged against THAT type's Loc rather than
// the ancestor's — deliberate, because clause 3 makes the use genuinely a
// property of the derived type and that is the position a reader is looking at.
// The extra passes cannot change the verdict: the walk is over a set, and a use
// that passed once passes again. The same repetition reaches the element side by
// a different route, an extension's {content type} particle containing the
// base's, and is harmless for the same reason and one more — e-props-correct
// clause 2 is charged at the DECLARATION's own Loc (checkElementDefaultValid),
// which no enclosing type can move, so neither the verdict nor the reported
// position depends on which route reached it.
//
// Phase A does NOT walk {attribute group definitions} (see resolve.go's
// FOLLOW-COST ASYMMETRY note): every <attributeGroup ref> is inlined at producer
// mapping time, so a group's uses are already folded into each complex type that
// references it and walking types alone would suffice for the parser path. This
// phase walks them anyway, because an UNREFERENCED group — and any group a
// SchemaBuilder caller adds directly — holds Attribute Use components the spec
// constrains all the same, and a folded use is simply re-tested with the same
// verdict. The price of going where Phase A did not is that a group's <attribute
// ref> was never vetted for resolvability: an unresolvable one is SKIPPED here
// (ResolvedAttributeDeclaration reports no declaration), not charged src-resolve,
// which is fail-open and never a false reject.
func (s *Schema) checkComponentValueConstraints() error {
	w := componentWalk{
		attributeUse:       s.checkAttributeUseValueConstraint,
		elementDeclaration: s.checkElementDefaultValid,
	}
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
	for _, mgd := range s.modelGroups {
		if err := w.walkModelGroup(mgd.ModelGroup(), mgd.Loc()); err != nil {
			return err
		}
	}
	for _, g := range s.attributeGroups {
		for _, u := range g.AttributeUses() {
			if err := w.walkAttributeUse(u, g.Loc(), attributeGroupOwner(g)); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkAttributeUseValueConstraint charges au-props-correct (§3.5.6) clauses 2 and
// 3 against one Attribute Use, plus a-props-correct clause 2 against the LOCAL
// declaration a use owns. owner names the enclosing component and loc is its
// position: an Attribute Use is not one of the eight kinds that retain a Loc
// (doc.go), so the rejection is charged where a reader must edit — the complex
// type or attribute group that holds the use — and names the attribute in the
// message. The owned local declaration is charged at its OWN Loc, which it does
// retain.
//
// Clause 2 is decided first, in spec order, so a use violating both clauses
// reports the more basic failure: its {value constraint} is not a valid default
// at all, which makes "is it identical to the declaration's" moot. It is checked
// against the RESOLVED {attribute declaration}.{type definition} — never a type
// on the Use, which has none — and only when the use has its OWN {value
// constraint}, never the §3.5.4 ·effective value constraint· (that one is the
// declaration's, already charged a-props-correct clause 2 in its own right).
//
// Clause 3's antecedent is then read in the rule's own order. Both conjuncts must
// hold for it to bite: U must have its OWN {value constraint} (never the
// ·effective value constraint·, which would make the test vacuously self-
// comparing), and U.{attribute declaration}.{value constraint} must be present
// with {variety} = fixed.
//
// Two consequents follow, in spec order so the first reported failure is
// deterministic (STYLE D1): the use's {variety} must be fixed, and its {value}
// must be identical to the declaration's. Both {value}s belong to ONE value
// space — the declaration's {type definition} governs each, since the use's
// {value constraint} constrains that very declaration — so the same type is
// handed to ValueSpace.Identical on both sides.
//
// Every non-decision is fail-open and never a false reject: an unresolvable
// declaration (a dangling Ref, already charged src-resolve by Phase A on the
// paths Phase A walks), a {type definition} that is absent, unresolvable, or
// complex, and an undecided ValueSpace verdict all accept.
func (s *Schema) checkAttributeUseValueConstraint(u AttributeUse, loc xsderr.Loc, owner string) error {
	d, ok := s.ResolvedAttributeDeclaration(u)
	if !ok {
		return nil
	}
	// The Local variant's declaration is in no global table, so this use is the
	// only site that can charge it (see the two walks, above). The Ref variant's
	// target IS in that table and checkAttributeDeclarationDefaults charges it
	// there, exactly once, however many uses reference it.
	if owned, isOwned := s.ownedAttributeDeclaration(u); isOwned {
		if err := s.checkAttributeDeclarationValueConstraint(owned); err != nil {
			return err
		}
	}
	uvc, own := u.ValueConstraint()
	if !own {
		// Both clauses are gated on U having its OWN {value constraint} —
		// clause 2's "U.{value constraint} is not ·absent·" and clause 3's "U
		// itself has a {value constraint}" — so neither is reached.
		return nil
	}
	n := u.DeclarationName()
	t, hasType := s.ResolvedSimpleType(d.TypeDefinition())
	if hasType {
		if err := s.checkSimpleDefault(ruleAuPropsCorrect, "2", loc, owner+" attribute "+n.String(), t, uvc); err != nil {
			return err
		}
	}
	dvc, present := d.ValueConstraint()
	if !present || dvc.Kind() != ValueFixed {
		return nil // the declaration is not fixed: clause 3 is discharged
	}
	if uvc.Kind() != ValueFixed {
		return xsderr.New(ruleAuPropsCorrect, loc,
			"%s gives attribute %s a %s value constraint, but the attribute declaration fixes it to %q, and au-props-correct clause 3 requires the use's {value constraint}.{variety} to be fixed too", owner, n, uvc.Kind(), dvc.LexicalForm())
	}
	if !hasType {
		return nil
	}
	identical, decided := s.valueSpace.Identical(s, t, uvc, t, dvc)
	if decided && !identical {
		return xsderr.New(ruleAuPropsCorrect, loc,
			"%s fixes attribute %s to %q, but the attribute declaration fixes it to %q, and au-props-correct clause 3 requires the two {value}s to be identical", owner, n, uvc.LexicalForm(), dvc.LexicalForm())
	}
	return nil
}

// checkAttributeDeclarationDefaults is Phase E's declaration-side walk: it charges
// a-props-correct (§3.2.6.1) clause 2 against every GLOBAL Attribute Declaration,
// in document order so the first reported failure is deterministic (STYLE D1/D2 —
// s.attributes is the document-ordered slice, not the by-name index).
//
// It walks the GLOBAL table only. A LocalAttributeDeclaration is never a member of
// it — the dcl.att.local mapping (§3.2.2.2) hands the declaration to its sibling
// Attribute Use, which is its sole owner (attributeuse.go) — so it is unreachable
// from here and is charged the same clause from the use side instead, by
// checkAttributeUseValueConstraint. Neither walk alone reaches every declaration:
// a global one may be referenced by no use at all, and a local one appears in no
// table.
func (s *Schema) checkAttributeDeclarationDefaults() error {
	for _, d := range s.attributes {
		if err := s.checkAttributeDeclarationValueConstraint(d); err != nil {
			return err
		}
	}
	return nil
}

// checkAttributeDeclarationValueConstraint charges a-props-correct (§3.2.6.1)
// clause 2 against one Attribute Declaration: "if there is a {value constraint},
// then it is a valid default with respect to the {type definition}". The
// declaration is charged at its own Loc, which an AttributeDeclaration retains
// (doc.go), so a reader is sent to the <xs:attribute> that wrote the default.
//
// Both gates accept rather than reject: no {value constraint} means the clause is
// not reached at all (never reached-and-satisfied), and a {type definition} that
// is absent, unresolvable, or complex is ResolvedSimpleType's documented "not decidable
// by this clause".
func (s *Schema) checkAttributeDeclarationValueConstraint(d AttributeDeclaration) error {
	dvc, present := d.ValueConstraint()
	if !present {
		return nil // "if there is a {value constraint}" fails: clause 2 is not reached
	}
	t, ok := s.ResolvedSimpleType(d.TypeDefinition())
	if !ok {
		return nil
	}
	return s.checkSimpleDefault(ruleAPropsCorrect, "2", d.Loc(), "attribute declaration "+d.Name().String(), t, dvc)
}

// checkSimpleDefault is Simple Default Valid (§3.2.6.2, cos-valid-simple-default)
// — the ONE implementation of it in this package (STYLE T4), shared by
// a-props-correct clause 2, au-props-correct clause 2, and — through
// cos-valid-default clause 1, which defers to it for both of its own arms —
// e-props-correct clause 2. The three differ only in which (type, value
// constraint) pair they hand it and which rule the failure is charged to; the
// element-side caller does a case analysis first (elementdefaultvalid.go) to work
// out which simple type that is.
//
// rule and clause together are the citation the message closes with, and the two
// travel as separate parameters because the clause number is NOT constant across
// the callers: the three Phase E rules above all charge it as their clause 2,
// and cvc-elt charges the same predicate at ASSESSMENT time as its clause 5.1.1
// ([Schema.ElementDefaultValid]). Neither is derivable from the other, and
// spelling either rule's name a second time here would silently mislabel any
// further caller (STYLE D3).
//
// The verdict is the installed ValueSpace's, and an UNDECIDED verdict ACCEPTS.
// That is not laxity, it is the fail-open contract (ValueSpace, PRINCIPLES 20):
// undecided covers a type no backend mapping governs — xs:anySimpleType and
// xs:anyAtomicType included, which is what §3.2.2.2's third tier gives every
// typeless <xs:attribute default="…">, and for which Datatype Valid is
// unconditionally true — a QName/NOTATION-governed value space, whose lexical
// mapping needs namespace bindings a ValueConstraint does not carry (PRINCIPLES
// 19), and a construction-stage failure in the TYPE's facets, which is not a
// verdict about this value constraint at all.
//
// A decided rejection's cause — the datatype-layer cvc-* verdict ValueSpace hands
// back — is DISCARDED here rather than wrapped, and this is the one place that
// says why: the rejection is charged at the owning component's Loc under a
// schema-component rule ID, so carrying a cvc-* verdict under it would blame the
// wrong component under the wrong rule. A user who wants the datatype-layer
// detail gets it from instance validation, which charges cvc-* and does wrap it.
//
// This is the first finalize-phase check that enters the installed value space's
// FACET pipeline rather than its Mapping alone, so it is also the first that can
// meet a facet-PRECONDITION fault of the type it is handed. There are two such
// classes, and naming only the first would understate the exposure:
//
//   - a facet paired with a value lacking the capability it needs
//     (cos-applicable-facets §4.1.5). §4.1.5 is ONE constraint whose cases split
//     on {variety}, not two provisions; the split below it is this package's, not
//     the spec's. checkVarietyApplicableFacets (derivation.go) discharges the list
//     and union cases here, where the applicable set is a fixed literal, and the
//     ATOMIC case's per-primitive table lookup is discharged by the installed
//     SimpleTypeRestrictionChecker (restrictionchecker.go), whose Phase D pass
//     runs before this one.
//   - a type with NO whiteSpace facet in force where §3.16.7.4 (every primitive's
//     {facets} carries one) and §4.3.6.1 (a list's materialized fixed collapse
//     facet) guarantee one, which value/whitespace.go's effectiveWhiteSpace
//     charges. Only atomic and list are exposed: a union has no whiteSpace facet
//     at all under §4.1.5, so its absence there is spec-mandated, not a fault.
//
// Both are reachable only for a *SimpleType in a Schema finalized with NO
// SimpleTypeRestrictionChecker installed — the second, for instance, from a
// NewPrimitiveType(loc, name, nil, nil) that some backend then maps. Neither is
// reachable for a type the PARSER built: it installs one, so the first fault is
// already rejected by Phase D, and its atomic types inherit a primitive's
// whiteSpace facet while its lists carry the materialized fixed collapse one.
// Either fault is a fault of the TYPE, and the value space reports it undecided
// rather than as a verdict (#321 settled that contract: the pipeline returns an
// *xsderr.Error, it does not panic), so it lands in the accepting branch below:
// this clause never rejects a schema for it, and never crashes on it.
func (s *Schema) checkSimpleDefault(rule xsderr.Rule, clause string, loc xsderr.Loc, owner string, t *SimpleType, vc ValueConstraint) error {
	cause, decided := s.valueSpace.ValidDefault(s, t, vc)
	if !decided || cause == nil {
		return nil
	}
	return xsderr.New(rule, loc,
		"%s has a {value constraint} of %q, which is not Datatype Valid with respect to its {type definition} (Datatypes §4.1.4 cvc-datatype-valid), so it is not a valid default (%s clause %s, cos-valid-simple-default §3.2.6.2)", owner, vc.LexicalForm(), rule, clause)
}
