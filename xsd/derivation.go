package xsd

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// ruleCosSTRestricts is Derivation Valid (Restriction, Simple) (Structures
// §3.16.6.2, id="cos-st-restricts"): the per-variety constraints relating a
// Simple Type Definition D to its {base type definition} B. This package charges
// its structural/variety-shape sub-clauses (1.1, 2.1, 2.2.1.1, 2.2.2.1, 2.2.2.3,
// 3.1, 3.2.1.1, 3.2.2.1, 3.2.2.3) at construction time; the facet-value
// sub-clauses are deferred (see checkSTGraph).
const ruleCosSTRestricts xsderr.Rule = "cos-st-restricts"

// The precisionDecimal scale-facet Schema Component Constraints, charged at
// schema construction against the abstract-model {facets} property
// (SimpleType.EffectiveFacets) — the construction-time complements of the
// instance-time cvc-maxScale-valid / cvc-minScale-valid facet stages (#133,
// package value). Only precisionDecimal and its restrictions carry maxScale /
// minScale (xsd-precisionDecimal.md §3.3), so these are vacuous on every other
// type.
const (
	// ruleMaxScaleValidRestriction is maxScale valid restriction
	// (xsd-precisionDecimal.md §4.2.4, id="maxScale-valid-restriction"): a
	// restriction's maxScale {value} may not be greater than the {value} of the
	// {base type definition}'s effective maxScale — maxScale may only move down.
	ruleMaxScaleValidRestriction xsderr.Rule = "maxScale-valid-restriction"
	// ruleMinScaleValidRestriction is minScale valid restriction
	// (xsd-precisionDecimal.md §4.3.4, id="minScale-valid-restriction"): the
	// mirror image — a restriction's minScale {value} may not be less than the
	// base's effective minScale — minScale may only move up.
	ruleMinScaleValidRestriction xsderr.Rule = "minScale-valid-restriction"
	// ruleMinScaleLEMaxScale is the "minScale less than or equal to maxScale"
	// consistency SCC (xsd-precisionDecimal.md §4.3.4). It is NOT
	// restriction-specific: it constrains any type's {facets}. WHY the constant
	// carries the string "minScale-totalDigits": the spec's anchor id for this
	// SCC is a copy-paste bug that names the totalDigits constraint, so that is
	// the only string extractable from the spec text for the catalog; error
	// messages cite the SCC by its true title instead, and the spec's own Note
	// disclaims any relation to totalDigits.
	ruleMinScaleLEMaxScale xsderr.Rule = "minScale-totalDigits"
	// ruleMaxScaleFixed is the maxScale {fixed}-inheritance SCC
	// (xsd-precisionDecimal.md §4.2.1 dc-maxScale, id="f-ms-fixed"): if the base's
	// effective maxScale is {fixed}, a restriction may not specify any maxScale
	// {value} other than the base's — checked independently of the value SCC, as a
	// further-narrowing value satisfies maxScale-valid-restriction yet still
	// violates {fixed}.
	ruleMaxScaleFixed xsderr.Rule = "f-ms-fixed"
	// ruleMinScaleFixed is the minScale {fixed}-inheritance SCC
	// (xsd-precisionDecimal.md §4.3.1 dc-minScale, id="f-mns-fixed"): the mirror
	// of ruleMaxScaleFixed for minScale.
	ruleMinScaleFixed xsderr.Rule = "f-mns-fixed"
)

// checkSTGraph enforces the cross-reference Simple Type Definition constraints
// that need t's resolved {base type definition}, {item type definition}, and
// {member type definitions} pointers — the checks checkSTProps (simpletype.go)
// cannot make at the pure-property layer. NewSimpleType and NewPrimitiveType call
// it after t.variety/t.base/t.ownFacets are wired, when those pointers are
// already live (a simple type references its base/item/members by pointer, set
// once at construction with no setter).
//
// It charges, per clause:
//
//   - st-props-correct clause 5 (each member of {facets} is supported by the
//     processor) — via checkFacetsSupported.
//   - st-props-correct clause 3 (D.{base type definition}.{final} does not
//     contain restriction). This single site also discharges cos-st-restricts
//     clauses 1.2, 2.2.2.2, and 3.2.2.2 ("B.{final} does not contain
//     restriction"): B is by definition D's {base type definition}, so those
//     clauses are the identical predicate on the identical component and are not
//     re-checked here — a second, unreachable rejection would be charge-imprecise
//     (STYLE E2).
//   - the per-variety shape and cos-st-restricts case constraints — via
//     checkAtomicGraph / checkListGraph / checkUnionGraph.
//   - the precisionDecimal scale-facet Schema Component Constraints
//     (maxScale-valid-restriction, minScale-valid-restriction, minScale ≤
//     maxScale, f-ms-fixed, f-mns-fixed) — via checkScaleFacets, which compares
//     t's {facets} against the base's through EffectiveFacets.
//
// st-props-correct clause 2 (the {base} chain terminates at a primitive or
// xs:anySimpleType — no circular derivation) is a documented no-op: a cyclic
// {base} chain is unconstructible via this package's constructors, because
// NewSimpleType demands a live base pointer that must already exist, so a type
// cannot appear on its own base chain. cos-st-restricts clause 3.3
// (no-self-membership, checkUnionGraph) is retired by the same argument — a
// union's members must pre-exist the union, so the union cannot be in its own
// transitive membership. Tightening the still-exported
// Atomic.Primitive/List.Item/Union.Members fields to make even a
// post-construction mutation-induced cycle impossible is tracked in #215; until
// then the honest claim is "unconstructible via constructors", not
// "structurally unrepresentable".
//
// Deferred, out of scope for this pure-leaf package (which does not depend on
// package value): the facet-applicability and facet-constraint sub-clauses
// cos-st-restricts 1.3, 2.2.1.2, 2.2.2.4, 2.2.2.5, 3.2.1.2, 3.2.2.4, and 3.2.2.5,
// which compare facet {value}s in the value space (a derived minInclusive within
// the base's range, enumeration ⊆ base, "only whiteSpace collapse fixed", "facets
// empty", ...). They await a value-aware finalize pass.
func checkSTGraph(loc xsderr.Loc, t *SimpleType) error {
	if err := checkFacetsSupported(loc, t.ownFacets); err != nil {
		return err
	}
	if t.base != nil && finalContains(t.base.final, DerivationRestriction) {
		return xsderr.New(ruleSTPropsCorrect, loc,
			"simple type {base type definition} %s has restriction in its {final}, which blocks derivation (st-props-correct clause 3)", t.base.name)
	}
	if err := checkScaleFacets(loc, t); err != nil {
		return err
	}
	switch t.variety.(type) {
	case Atomic:
		return checkAtomicGraph(loc, t)
	case List:
		return checkListGraph(loc, t)
	case Union:
		return checkUnionGraph(loc, t)
	}
	return nil
}

// checkFacetsSupported enforces st-props-correct clause 5: each member of
// {facets} is supported by the processor. goxsd8 supports exactly the closed
// FacetKind set (the 14 core Constraining Facets plus the two precisionDecimal
// extension facets), so a facet whose kind falls outside that contiguous enum is
// unsupported. Every Facet built through this package carries a supported kind
// under the current static facet catalog, so this rejection is expected to be
// unreachable; it is emitted rather than skipped so an implementation-defined
// facet introduced later fails here with the right rule (§3.16.6.1 Note) instead
// of silently.
func checkFacetsSupported(loc xsderr.Loc, facets []Facet) error {
	for _, f := range facets {
		if f.kind < FacetLength || f.kind > FacetMinScale {
			return xsderr.New(ruleSTPropsCorrect, loc,
				"simple type {facets} contains an unsupported facet %s (st-props-correct clause 5)", f.kind)
		}
	}
	return nil
}

// checkAtomicGraph enforces the atomic-variety constraint on t (whose {variety}
// is Atomic): cos-st-restricts clause 1.1 — either t is xs:anyAtomicType, or its
// {base type definition} is itself an atomic simple type definition. This is the
// same requirement as the Datatypes §4.1.1 shape prose ("if {variety} is atomic
// then the {variety} of {base type definition} must be atomic, unless the base is
// anySimpleType"): the sole base=anySimpleType exception applies only to
// xs:anyAtomicType (whose base xs:anySimpleType has an absent {variety}), and
// xs:anyAtomicType is a package singleton never built through this constructor.
//
// Clause 1.2 (B.{final} does not contain restriction) is discharged by
// checkSTGraph's clause-3 site (B is D's {base}); clause 1.3 (facet
// applicability/constraints) is deferred (see checkSTGraph). It reads only
// t.base — never the Atomic.Primitive pointer, which self-references on a
// primitive datatype (§3.16.1) and so cannot drive a terminating base walk.
func checkAtomicGraph(loc xsderr.Loc, t *SimpleType) error {
	if t == anyAtomicType {
		return nil
	}
	if t.base == nil {
		return xsderr.New(ruleSTPropsCorrect, loc,
			"atomic simple type has an absent {base type definition} (st-props-correct clause 1)")
	}
	if _, ok := t.base.variety.(Atomic); ok {
		return nil
	}
	return xsderr.New(ruleCosSTRestricts, loc,
		"atomic simple type {base type definition} %s is not an atomic simple type definition (cos-st-restricts clause 1.1)", t.base.name)
}

// checkListGraph enforces the list-variety constraints on t (whose {variety} is
// List): cos-st-restricts clause 2. Clause 2.1 fixes the {item type definition}
// shape (not a special type; {variety} atomic, or union with no list type in its
// transitive membership). Clause 2.2 then branches on the constructed-vs-restricted
// discriminant B == xs:anySimpleType (grounding part D: the abstract model keys
// this off the resolved base, not off which XML element produced the type):
//
//   - constructed (B is xs:anySimpleType): clause 2.2.1.1 — the item's {final}
//     does not contain list. Clause 2.2.1.2 (facets shape) is deferred.
//   - restricted (B is a real list): clause 2.2.2.1 — B.{variety} is list; clause
//     2.2.2.3 — the item is validly derived from B's item (cos-st-derived-ok,
//     §3.16.6.3). Clause 2.2.2.2 (B.{final}) is discharged by checkSTGraph's
//     clause-3 site; clauses 2.2.2.4/.5 (facets) are deferred.
func checkListGraph(loc xsderr.Loc, t *SimpleType) error {
	item := t.variety.(List).Item
	if item == nil {
		return xsderr.New(ruleSTPropsCorrect, loc,
			"list simple type has an absent {item type definition} (st-props-correct clause 1)")
	}
	if isSpecialType(item) {
		return xsderr.New(ruleCosSTRestricts, loc,
			"list {item type definition} %s is a special type definition (cos-st-restricts clause 2.1)", item.name)
	}
	switch item.variety.(type) {
	case Atomic:
	case Union:
		if unionMembershipHasList(item) {
			return xsderr.New(ruleCosSTRestricts, loc,
				"list {item type definition} %s is a union with a list type in its transitive membership (cos-st-restricts clause 2.1)", item.name)
		}
	default:
		return xsderr.New(ruleCosSTRestricts, loc,
			"list {item type definition} %s has a {variety} that is neither atomic nor union (cos-st-restricts clause 2.1)", item.name)
	}

	if t.base == nil {
		return xsderr.New(ruleSTPropsCorrect, loc,
			"list simple type has an absent {base type definition} (st-props-correct clause 1)")
	}
	if t.base == anySimpleType {
		if finalContains(item.final, DerivationList) {
			return xsderr.New(ruleCosSTRestricts, loc,
				"list {item type definition} %s has list in its {final}, blocking its use as a list item (cos-st-restricts clause 2.2.1.1)", item.name)
		}
		return nil
	}
	baseList, ok := t.base.variety.(List)
	if !ok {
		return xsderr.New(ruleCosSTRestricts, loc,
			"list simple type restricts base %s whose {variety} is not list (cos-st-restricts clause 2.2.2.1)", t.base.name)
	}
	if !derivedOKSimple(item, baseList.Item) {
		return xsderr.New(ruleCosSTRestricts, loc,
			"list {item type definition} %s is not validly derived from the base list's item type (cos-st-restricts clause 2.2.2.3 via cos-st-derived-ok §3.16.6.3)", item.name)
	}
	return nil
}

// checkUnionGraph enforces the union-variety constraints on t (whose {variety}
// is Union): cos-st-restricts clause 3. Clause 3.1 excludes special type
// definitions from {member type definitions}. Clause 3.2 branches on the
// constructed-vs-restricted discriminant B == xs:anySimpleType:
//
//   - constructed (B is xs:anySimpleType): clause 3.2.1.1 — every member's
//     {final} does not contain union. Clause 3.2.1.2 (facets empty) is deferred.
//   - restricted (B is a real union): clause 3.2.2.1 — B.{variety} is union;
//     clause 3.2.2.3 — each member is validly derived from the CORRESPONDING
//     (positional, PRINCIPLES 11) base member (cos-st-derived-ok, §3.16.6.3).
//     Clause 3.2.2.2 (B.{final}) is discharged by checkSTGraph's clause-3 site;
//     clauses 3.2.2.4/.5 (facets) are deferred.
//
// Clause 3.3 (no-self-membership) is a documented no-op — see checkSTGraph.
func checkUnionGraph(loc xsderr.Loc, t *SimpleType) error {
	members := t.variety.(Union).Members
	for _, m := range members {
		if m == nil {
			return xsderr.New(ruleSTPropsCorrect, loc,
				"union {member type definitions} contains an absent member (st-props-correct clause 1)")
		}
		if isSpecialType(m) {
			return xsderr.New(ruleCosSTRestricts, loc,
				"union {member type definitions} contains special type definition %s (cos-st-restricts clause 3.1)", m.name)
		}
	}

	if t.base == nil {
		return xsderr.New(ruleSTPropsCorrect, loc,
			"union simple type has an absent {base type definition} (st-props-correct clause 1)")
	}
	if t.base == anySimpleType {
		for _, m := range members {
			if finalContains(m.final, DerivationUnion) {
				return xsderr.New(ruleCosSTRestricts, loc,
					"union member %s has union in its {final}, blocking its use as a union member (cos-st-restricts clause 3.2.1.1)", m.name)
			}
		}
		return nil
	}
	baseUnion, ok := t.base.variety.(Union)
	if !ok {
		return xsderr.New(ruleCosSTRestricts, loc,
			"union simple type restricts base %s whose {variety} is not union (cos-st-restricts clause 3.2.2.1)", t.base.name)
	}
	if len(members) != len(baseUnion.Members) {
		return xsderr.New(ruleCosSTRestricts, loc,
			"union restriction has %d member type definitions but base union %s has %d (cos-st-restricts clause 3.2.2.3)",
			len(members), t.base.name, len(baseUnion.Members))
	}
	for i, m := range members {
		if !derivedOKSimple(m, baseUnion.Members[i]) {
			return xsderr.New(ruleCosSTRestricts, loc,
				"union member %s is not validly derived from the corresponding base member (cos-st-restricts clause 3.2.2.3 via cos-st-derived-ok §3.16.6.3)", m.name)
		}
	}
	return nil
}

// derivedOKSimple reports whether d is validly derived from b per Type
// Derivation OK (Simple) (Structures §3.16.6.3, cos-st-derived-ok) under the
// empty set of blocking keywords — the "validly derived" relation invoked by
// cos-st-restricts clauses 2.2.2.3 and 3.2.2.3. It is a relation, not a rejection
// point: a false result is charged by its caller as a cos-st-restricts violation.
//
// With the empty blocking set, clause 2.1 (restriction not in S, or in
// d.{base}.{final}) is vacuously satisfied, so only clause 1 (same type) and
// clause 2.2's alternatives remain: 2.2.1 d.{base} = b; 2.2.2 d.{base} (never
// xs:anyType, a Complex Type Definition absent from this package) is itself
// validly derived from b; 2.2.3 d is a list or union and b is xs:anySimpleType;
// 2.2.4 b is a union whose {facets} are empty and d is validly derived from a
// member of b (recursion descends b's transitive membership, checking each
// intervening union's {facets} emptiness at its own level, clause 2.2.4.3).
//
// It walks d's {base} chain and b's members with no visited set: both are finite
// and acyclic on any constructor-built graph, so the recursion terminates
// (unconstructible-via-constructors, see checkSTGraph; #215).
func derivedOKSimple(d, b *SimpleType) bool {
	if d == nil || b == nil {
		return false
	}
	if d == b {
		return true
	}
	if d.base == b {
		return true
	}
	if b == anySimpleType {
		switch d.variety.(type) {
		case List, Union:
			return true
		}
	}
	if d.base != nil && derivedOKSimple(d.base, b) {
		return true
	}
	if bv, ok := b.variety.(Union); ok && len(b.EffectiveFacets()) == 0 {
		for _, m := range bv.Members {
			if derivedOKSimple(d, m) {
				return true
			}
		}
	}
	return false
}

// isSpecialType reports whether t is one of the two special datatypes,
// xs:anySimpleType or xs:anyAtomicType (Datatypes §2.4.2, id="dt-special"),
// tested by identity against the package singletons.
func isSpecialType(t *SimpleType) bool {
	return t == anySimpleType || t == anyAtomicType
}

// finalContains reports whether the {final} set contains derivation method d.
func finalContains(final []DerivationMethod, d DerivationMethod) bool {
	for _, f := range final {
		if f == d {
			return true
		}
	}
	return false
}

// checkScaleFacets enforces the five precisionDecimal scale-facet Schema
// Component Constraints at construction (see the rule constants above). It reads
// the {facets} property directly through SimpleType.EffectiveFacets (the
// §3.16.6.4 overlay), so a facet inherited unchanged through several restriction
// levels is compared transitively with no manual ancestor walk. t.base is nil
// only for xs:anySimpleType, which carries no facets, so the base-relative SCCs
// are vacuous there; the minScale ≤ maxScale consistency SCC is not
// restriction-specific and runs on every type's own effective {facets}.
func checkScaleFacets(loc xsderr.Loc, t *SimpleType) error {
	if t.base != nil {
		baseEff := t.base.EffectiveFacets()
		if err := checkScaleValueRestriction(loc, t, baseEff, FacetMaxScale, ruleMaxScaleValidRestriction); err != nil {
			return err
		}
		if err := checkScaleValueRestriction(loc, t, baseEff, FacetMinScale, ruleMinScaleValidRestriction); err != nil {
			return err
		}
		if err := checkScaleFixed(loc, t, baseEff, FacetMaxScale, ruleMaxScaleFixed); err != nil {
			return err
		}
		if err := checkScaleFixed(loc, t, baseEff, FacetMinScale, ruleMinScaleFixed); err != nil {
			return err
		}
	}
	return checkScaleConsistency(loc, t)
}

// checkScaleValueRestriction charges maxScale-valid-restriction (§4.2.4) or
// minScale-valid-restriction (§4.3.4): a restriction's own scale facet {value}
// may not relax the base's effective same-kind {value}. maxScale may only move
// down (own > base is the violation), minScale only up (own < base). Both are
// vacuous when the base has no effective facet of this kind, or when t declares
// no own facet of this kind (an inherited-only facet equals the base's effective
// value and cannot cross it).
func checkScaleValueRestriction(loc xsderr.Loc, t *SimpleType, baseEff []EffectiveFacet, kind FacetKind, rule xsderr.Rule) error {
	baseF, ok := findEffectiveFacet(baseEff, kind)
	if !ok {
		return nil
	}
	ownF, ok := findFacet(t.ownFacets, kind)
	if !ok {
		return nil
	}
	ownV, err := scaleValue(ownF, loc, rule)
	if err != nil {
		return err
	}
	baseV, err := scaleValue(baseF, loc, rule)
	if err != nil {
		return err
	}
	if !scaleRelaxes(kind, ownV, baseV) {
		return nil
	}
	return xsderr.New(rule, loc,
		"simple type restriction's own %s {value} %d relaxes the {base type definition}'s effective %s {value} %d, which restriction may not do (%s)",
		kind, ownV, kind, baseV, rule)
}

// scaleRelaxes reports whether an own scale {value} widens (relaxes) the base's,
// which restriction forbids: for maxScale a larger value widens the space, for
// minScale a smaller value does.
func scaleRelaxes(kind FacetKind, own, base int) bool {
	if kind == FacetMaxScale {
		return own > base
	}
	return own < base
}

// checkScaleFixed charges f-ms-fixed (§4.2.1) or f-mns-fixed (§4.3.1): if the
// base's effective scale facet of this kind is {fixed}, a restriction may not
// specify its own scale facet with ANY value other than the base's — this is
// distinct from checkScaleValueRestriction, which a further-narrowing value can
// satisfy while still overriding a {fixed} base facet. Vacuous when the base has
// no such effective facet, the base facet is not {fixed}, or t declares no own
// facet of this kind.
func checkScaleFixed(loc xsderr.Loc, t *SimpleType, baseEff []EffectiveFacet, kind FacetKind, rule xsderr.Rule) error {
	baseF, ok := findEffectiveFacet(baseEff, kind)
	if !ok {
		return nil
	}
	if fixed, _ := baseF.Fixed(); !fixed {
		return nil
	}
	ownF, ok := findFacet(t.ownFacets, kind)
	if !ok {
		return nil
	}
	ownV, err := scaleValue(ownF, loc, rule)
	if err != nil {
		return err
	}
	baseV, err := scaleValue(baseF, loc, rule)
	if err != nil {
		return err
	}
	if ownV == baseV {
		return nil
	}
	return xsderr.New(rule, loc,
		"simple type restriction sets %s {value} %d but the {base type definition}'s effective %s is {fixed} at %d and may not be overridden (%s)",
		kind, ownV, kind, baseV, rule)
}

// checkScaleConsistency charges the "minScale less than or equal to maxScale"
// SCC (spec anchor id minScale-totalDigits, a copy-paste bug — see
// ruleMinScaleLEMaxScale): it is not restriction-specific, so it runs against
// t's OWN effective {facets} after overlay. It rejects when both facets are in
// force and minScale's {value} exceeds maxScale's. The spec's Note explicitly
// disclaims any cross-check against totalDigits.
func checkScaleConsistency(loc xsderr.Loc, t *SimpleType) error {
	eff := t.EffectiveFacets()
	minF, hasMin := findEffectiveFacet(eff, FacetMinScale)
	maxF, hasMax := findEffectiveFacet(eff, FacetMaxScale)
	if !hasMin || !hasMax {
		return nil
	}
	// A malformed {value} on either facet is charged under that facet's own
	// valid-restriction rule (the rule a bad literal on it would otherwise hit).
	minV, err := scaleValue(minF, loc, ruleMinScaleValidRestriction)
	if err != nil {
		return err
	}
	maxV, err := scaleValue(maxF, loc, ruleMaxScaleValidRestriction)
	if err != nil {
		return err
	}
	if minV <= maxV {
		return nil
	}
	return xsderr.New(ruleMinScaleLEMaxScale, loc,
		"simple type {facets} has minScale {value} %d greater than maxScale {value} %d, violating \"minScale less than or equal to maxScale\" (its spec anchor id %s is a copy-paste bug)",
		minV, maxV, ruleMinScaleLEMaxScale)
}

// findFacet returns the own Facet of the given kind and whether it is present.
func findFacet(facets []Facet, kind FacetKind) (Facet, bool) {
	for _, f := range facets {
		if f.kind == kind {
			return f, true
		}
	}
	return Facet{}, false
}

// findEffectiveFacet returns the in-force Facet of the given kind from an
// EffectiveFacets result and whether it is present.
func findEffectiveFacet(facets []EffectiveFacet, kind FacetKind) (Facet, bool) {
	for _, ef := range facets {
		if ef.facet.kind == kind {
			return ef.facet, true
		}
	}
	return Facet{}, false
}

// scaleValue reads a scale facet's single xs:integer {value} (which may be
// negative — no nonNegativeInteger constraint). That {value} is user-supplied
// schema lexical data reachable through the public NewFacet/NewSimpleType API,
// which accepts arbitrary lexical strings for scale kinds, so a wrong value count
// or non-integer literal is a real validity rejection charged as an
// *xsderr.Error, not a package logic error — mirroring value/facets.go's facetInt
// for the exact same maxScale/minScale {value} parsing at instance-validation
// time.
func scaleValue(f Facet, loc xsderr.Loc, rule xsderr.Rule) (int, error) {
	if len(f.values) != 1 {
		return 0, xsderr.New(rule, loc,
			"%s facet must carry exactly one value, has %d", f.kind, len(f.values))
	}
	n, err := strconv.Atoi(f.values[0])
	if err != nil {
		return 0, xsderr.New(rule, loc,
			"%s facet value %q is not an integer", f.kind, f.values[0])
	}
	return n, nil
}

// unionMembershipHasList reports whether any type in u's transitive membership
// (Datatypes id="dt-transitivemembership") has {variety} = list; it is the
// negative form of clause 2.1's "no types whose {variety} is list among the
// union's transitive membership". u is expected to be a union; a non-union u
// yields false. It recurses through member unions with no visited set: on any
// constructor-built graph a union's members pre-exist the union, so the
// transitive membership is finite and acyclic and the recursion terminates (a
// mutation-induced cycle is unconstructible via constructors; #215).
func unionMembershipHasList(u *SimpleType) bool {
	uv, ok := u.variety.(Union)
	if !ok {
		return false
	}
	for _, m := range uv.Members {
		if m == nil {
			continue
		}
		switch m.variety.(type) {
		case List:
			return true
		case Union:
			if unionMembershipHasList(m) {
				return true
			}
		}
	}
	return false
}
