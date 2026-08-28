package validate

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file visits every {assertions} site the assessment reaches and records
// it as an [Unevaluated], never as a satisfied check. Nothing here charges a
// violation and nothing here reads an instance value: there is no XPath
// evaluator for an assertion's {test} yet (#1042), and the two GAP markers
// below state, separately, what each of the two hooks withholds.
//
// The two rules are DISTINCT and are never conflated. cvc-assertion
// (§3.13.4.1) is the complex-type variety, reached from cvc-complex-type
// (§3.4.4.2) clause 6 and from nowhere else. cvc-assertions-valid (Datatypes
// §4.3.13.3) is the simple-type one, reached from cvc-datatype-valid (§4.1.4)
// clause 3 (dv_vfacets) over the assertions facet of a simple type. One record
// type carries both, discriminated by [Unevaluated.Rule].
//
// The simple-type sites are collected STATICALLY off the type
// ([walk.assertionSites]): being unevaluated is a property of the compiled
// type, not of a value, and the cvc-datatype-valid variety recursion PRINCIPLES
// 12 quantifies over lives in package value behind a (Value, error) signature
// with no channel for a non-verdict. The collection therefore OVER-reports
// against what a real evaluator would have touched — every member of a union
// gets a record, not only the ·validating· one, and a list gets one record for
// its {item type definition} rather than one per item — and that is the safe
// direction: the added records' whole consumer set is Result.unevaluated and
// its one reader [Result.Unevaluated], which carry sites PRESENT, so an extra
// record costs a decline and manufactures no charge. A record count is not a
// claim about how many evaluations were skipped.
//
// The recording sites are the four places the assessment decides an instance
// lexical against a simple type — cvc-attribute clause 3 ([walk.matchedAttribute]),
// cvc-complex-type clause 4 over a ·defaulted attribute·'s {lexical form}
// ([walk.defaultedAttribute]), and cvc-type clause 3.1.3 / cvc-complex-type clause
// 1.2 over an element's ·initial value· ([contentCheck.stringValid]) — plus
// [walk.wildcardAttributeAssertions] for the one path that reaches a simple type's
// facets without passing any of them. cvcid.go and cvcidentityconstraint.go re-run
// the datatype pipeline over lexicals those sites already recorded, and record
// nothing of their own.

// ruleCvcAssertion is Assertion Satisfied (Structures §3.13.4.1,
// cvc-assertion), whose single caller is cvc-complex-type clause 6. The clause
// charged goes in the message on ruleCvcElt's terms: the catalog carries the
// bare name.
const ruleCvcAssertion xsderr.Rule = "cvc-assertion"

// ruleCvcAssertionsValid is Assertions Valid (Datatypes §4.3.13.3,
// cvc-assertions-valid), the assertions facet's own per-facet specification
// under Facet Valid. A simple type's assertions answer to this rule and never
// to ruleCvcAssertion: the two differ in what they bind $value to and in what
// context item they evaluate against, so one ID standing for both would
// misreport every simple-type assertion the day evaluation lands.
const ruleCvcAssertionsValid xsderr.Rule = "cvc-assertions-valid"

// elementAssertions records one site per assertion in the {assertions} of e's
// ·governing type definition·, which is cvc-complex-type (§3.4.4.2) clause 6:
// "E is ·valid· with respect to each of the assertions in T.{assertions} as
// per Assertion Satisfied (§3.13.4.1)". A governing type that is not a Complex
// Type Definition has no {assertions} property at all — a simple one's
// assertions are facets, and reach ruleCvcAssertionsValid instead.
//
// {assertions} is read whole and its base chain is never walked: cos-ct-extends
// clause 1.7 and derivation-ok-restriction clause 5 both make B.{assertions} a
// prefix of T.{assertions}, so unioning the chain here would report every
// inherited assertion once per derivation step.
//
// GAP(validate): cvc-complex-type clause 6 is not evaluated — cvc-assertion
// (§3.13.4.1) needs an XPath evaluator for an assertion's {test} and this
// module has none (#1042) — so the element is neither charged under
// cvc-assertion nor shown ·valid· with respect to it. Fail-open: the withheld
// value is clause 6's own verdict, whose whole consumer set inside this
// package is w.res.violations and its one reader [Result.Violations]: neither
// is written here, both charge on a violation PRESENT, and no other reader of
// the walk consults clause 6's outcome. The skip can therefore only cost a
// rejection and can manufacture none.
func (w *walk) elementAssertions(e Element, g governance) {
	ct := g.complexType()
	if ct == nil {
		return
	}
	assertions := ct.Assertions()
	for i, a := range assertions {
		w.res.unevaluated = append(w.res.unevaluated, newUnevaluated(ruleCvcAssertion, e.Loc(),
			"assertion %d of %d in the {assertions} of the ·governing type definition· %s, whose {test} is %q, was not evaluated, so the element %s is not shown ·valid· with respect to it as cvc-complex-type clause 6 requires (Assertion Satisfied, §3.13.4.1)",
			i+1, len(assertions), typeName(*ct), a.Test().Expression(), e.Name()))
	}
}

// simpleAssertions records every assertions-facet site st carries, at loc — the
// location of the attribute or element whose lexical is being decided against
// st, an assertion component carrying no Loc of its own (#35).
//
// GAP(validate): DIRECTION UNESTABLISHED. cvc-assertions-valid (§4.3.13.3) is
// not evaluated (#1042), so the assertions facet contributes nothing to the
// Datatype Valid (§4.1.4) verdict its clause 3 folds it into. The withheld
// value is a conjunct of datatype-validity, and its readers are NOT only
// w.res.violations and [Result.Violations] — which charge on a violation
// PRESENT and so lose a rejection. [walk.validatingType] and [walk.idBindings]
// (cvcid.go) classify a value by its ·validating type·, which cvc-datatype-valid
// clause 2.3 makes the FIRST member of a union the value is Datatype Valid
// against: an unchecked assertion can leave an earlier member ·validating· that
// the spec rejects, binding an ·ID value· the spec's §3.17.5.2 table has none
// of, which cvc-id clause 2 then charges as a duplicate. [walk.keyMember]
// (cvcidentityconstraint.go) reads a PRESENT [schema actual value] where the
// spec's is ·absent· for the same reason, lengthening a ·key-sequence· into the
// duplicate arm of cvc-identity-constraint clause 4. Both of those are FALSE
// REJECTS, so this hook is not fail-open, and the direction over the whole
// consumer set is not established here (STYLE P3a).
func (w *walk) simpleAssertions(st *xsd.SimpleType, loc xsderr.Loc) {
	w.res.unevaluated = append(w.res.unevaluated, w.assertionSites(st, loc)...)
}

// assertionSites is the site set of one simple type, in cvc-datatype-valid
// (§4.1.4)'s own order: the constituents clause 2 recurses into first, then the
// type's own facets under clause 3. A list yields its {item type definition}'s
// sites then its own, a union yields each {member type definition}'s in
// document order then its own, and an atomic type yields its own alone — the
// three levels PRINCIPLES 12 names, all of which can carry assertions
// independently.
//
// It carries no visited set (STYLE D4). A union whose membership is circular is
// rejected at construction (xsd.Schema's checkUnionMembershipAcyclic) and
// §3.16.1 std-item_type_definition forbids a list of lists, which is why
// package value's own list and union recursions carry none either.
//
// An unresolvable hop — a {variety} whose base chain breaks, an itemType= or
// memberTypes= naming a type the schema cannot resolve — yields st's OWN
// assertions-facet sites rather than an error. Those sites survive the break:
// [walk.ownAssertionSites] reads [xsd.SimpleType.EffectiveFacets] and depends
// on none of the three hops, so a list whose itemType= resolves to nothing
// still has the list type's own assertions facet readable, while its
// constituents' sites are unreachable through a type nothing can map. There is
// no charge here to be wrong about: the same type fails value.ValidateLexical
// at every recording site, which declines there.
//
// No test drives those three arms and none can: src-resolve clause 1.1 rejects
// an itemType= or memberTypes= that resolves to nothing (or to a complex type)
// when the Schema is finalized, so a finalized w.schema reaches none of them.
// They are the safe answer for a resolver that is not one, which
// [xsd.SimpleType.Variety] admits by taking a TypeResolver.
func (w *walk) assertionSites(st *xsd.SimpleType, loc xsderr.Loc) []Unevaluated {
	if st == nil {
		return nil
	}
	variety, err := st.Variety(w.schema)
	if err != nil {
		return w.ownAssertionSites(st, loc)
	}
	var sites []Unevaluated
	switch variety.(type) {
	case xsd.List:
		item, err := st.Item(w.schema)
		if err != nil {
			return w.ownAssertionSites(st, loc)
		}
		sites = w.assertionSites(item, loc)
	case xsd.Union:
		members, err := st.Members(w.schema)
		if err != nil {
			return w.ownAssertionSites(st, loc)
		}
		for _, m := range members {
			sites = append(sites, w.assertionSites(m, loc)...)
		}
	}
	return append(sites, w.ownAssertionSites(st, loc)...)
}

// ownAssertionSites is one type's OWN assertions-facet sites, read off
// [xsd.SimpleType.EffectiveFacets] and never assembled by walking the base
// chain: §4.3.13.2 xr-assertions builds the facet's {value} as the base's
// Assertions followed by the restriction's own, and cos-assertions-restriction
// (§4.3.13.4) requires that prefix, so the effective facet already carries
// every inherited assertion exactly once.
func (w *walk) ownAssertionSites(st *xsd.SimpleType, loc xsderr.Loc) []Unevaluated {
	facets, err := st.EffectiveFacets(w.schema)
	if err != nil {
		return nil
	}
	var sites []Unevaluated
	for _, ef := range facets {
		assertions, isAssertions := ef.Facet().Assertions()
		if !isAssertions {
			continue
		}
		for i, a := range assertions {
			sites = append(sites, newUnevaluated(ruleCvcAssertionsValid, loc,
				"assertion %d of %d in the {value} of the assertions facet of the simple type %s, whose {test} is %q, was not evaluated, so the value at this location is not shown facet-valid with respect to that facet as cvc-assertions-valid requires (Datatypes §4.3.13.3, reached from cvc-datatype-valid clause 3)",
				i+1, len(assertions), typeName(st), a.Test().Expression()))
		}
	}
	return sites
}

// wildcardAttributeAssertions records the sites of an attribute information
// item that matches no {attribute use} and is left to an {attribute wildcard}
// this package does not evaluate (cvc-complex-type clause 2.2, assess.go's
// unmatchedAttribute). Under a ***strict*** or ***lax*** wildcard — and under
// those two only — the spec's ·attribute assessment· of such an item runs
// cvc-attribute clause 3 against the top-level declaration its ·expanded name·
// ·resolves· to, and [walk.attributeType] names exactly that type for cvcid.go
// and cvcidentityconstraint.go, which decide the lexical against it — so the
// site is reached here even though [walk.matchedAttribute], the ordinary
// clause-3 recording site, never runs for it.
//
// Under ***skip*** none of that holds: §3.10.4.1's Note performs QName
// resolution only for an item ·attributed to· a strict or lax wildcard, so a
// skipped item has NO ·governing· declaration, cvc-assess-elt (§3.3.4.6)
// clause 2.2 leaves its schema-validity unassessed, and no facet of any type is
// reached over its lexical. [walk.attributeType] declines such an attribute for
// exactly that reason (#1043), so under skip cvcid.go and
// cvcidentityconstraint.go no longer decide its lexical against anything. This
// call is DELIBERATELY NOT GATED on {process contents} all the same: dropping
// the record would make [Result.Unevaluated] a claim about WHICH wildcard
// admitted the item, which is cvc-wildcard's question and one this package does
// not evaluate at all (#717, assess.go's unmatchedAttribute). Under skip the
// record is therefore an over-report, in the direction this file's header
// states — a decline, never a charge.
//
// An attribute the schema resolves no top-level declaration for records
// nothing: it has no ·governing type definition·, so §3.17.5.2 clause 3
// excludes it and no facet of any type is reached over its lexical.
func (w *walk) wildcardAttributeAssertions(a Attribute) {
	st, typed := w.topLevelAttributeType(a)
	if !typed {
		return
	}
	w.simpleAssertions(st, a.Loc())
}
