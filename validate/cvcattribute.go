package validate

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file holds the ATTRIBUTE charges that need a value space: an attribute
// information item's lexical read through its declaration's {type definition}
// (cvc-attribute clause 3), the two independent fixed-value agreements
// (cvc-attribute clause 4 and cvc-au), and a ·defaulted attribute·'s own
// {lexical form} (cvc-complex-type clause 4). All four are reached from
// [walk.attributes]; the walk topology itself is assess.go's, and the content
// half's own value charge — an element's ·initial value· under cvc-complex-type
// clause 1.2 — is cvccomplexcontent.go's.
//
// Every one of them delegates to String Valid (§3.16.4, cvc-simple-type) and
// so inherits its three clauses: whiteSpace normalization (cl.1) and Datatype
// Valid (cl.2, Datatypes §4.1.4) are what value.ValidateLexical runs, in that
// order, and clause 3 is the residue below.
//
// GAP(validate): String Valid clause 3 — "every ·ENTITY value· in V is a
// ·declared entity name·" — is not checked, so an xs:ENTITY-valued attribute,
// or an element ·initial value· of that type, naming an entity the document
// never declared is accepted. A ·declared entity name· is the [name] of an
// unparsed entity information item in the DOCUMENT's [unparsedEntities]
// property (key-vde), which the abstract infoset does not carry: [Element],
// [Attribute] and [Text] are element-level views with no document above them,
// and the property arrives as a capability interface of its own rather than as
// a method added to any of the three (PRINCIPLES 3, doc.go). The withheld
// value's whole consumer set is Result.violations and its one reader
// Result.Violations, both of which decidedNotValid reads as violations
// PRESENT, so withholding one can only cost a rejection and never manufacture
// one (#773).

// ruleCvcAttribute is Attribute Locally Valid (Structures §3.2.4.1,
// cvc-attribute). The clause charged goes in the message on ruleCvcElt's
// terms: the catalog carries the bare name, so "cvc-attribute.3" is not a
// valid [xsderr.Rule].
const ruleCvcAttribute xsderr.Rule = "cvc-attribute"

// ruleCvcAu is Attribute Locally Valid (Use) (Structures §3.5.4, cvc-au). It
// is a rule of its own and not a clause of cvc-attribute: it tests the
// ATTRIBUTE USE's {value constraint}, where cvc-attribute clause 4 tests the
// DECLARATION's, and au-props-correct clause 3 reconciles the two at schema
// assembly rather than making either instance check redundant.
const ruleCvcAu xsderr.Rule = "cvc-au"

// matchedAttribute settles clause 2.1 for an attribute information item that
// matched an attribute use, which is where cvc-complex-type sends it: to
// cvc-attribute against the use's {attribute declaration} and to cvc-au
// against the use itself. Both are charged here, in that order, and both can
// fire for one attribute — they read two different {value constraint}s.
//
// The assertions facets clause 3's String Valid reaches on the declaration's
// {type definition} are recorded before the lexical is read at all
// ([walk.simpleAssertions], cvcassertion.go): they are a property of the type
// and not of the verdict, so a charge, a pass and a decline record the same
// sites.
//
// The three declines below withhold a verdict rather than guess one:
//
//   - a use whose {attribute declaration} does not resolve. Unreachable on a
//     *xsd.Schema that exists (Phase A charges src-resolve for a dangling
//     Ref), and a decline rather than a charge because a resolution that failed
//     says nothing about the attribute.
//   - a {type definition} that is absent, unresolvable, or COMPLEX. cvc-attribute
//     clause 2 makes an absent one a violation in its own right, which this
//     package does not charge: the property is read off an assembled schema,
//     where an absent {type definition} on an Attribute Declaration is not
//     representable at all (§3.2.1 makes the slot required), so charging clause 2
//     here would report the component model's own shape rather than the
//     document's.
//   - a lexical whose ValidateLexical error is not a VERDICT
//     ([value.IsDatatypeVerdict]). GAP(validate): an ungoverned type is the
//     live case — §3.2.2.2's third tier types an <attribute> with no @type as
//     xs:anySimpleType, which no backend maps, and the resulting error carries
//     cvc-datatype-valid exactly as a genuine rejection does. Charging it would
//     reject every typeless attribute in existence (#774).
func (w *walk) matchedAttribute(a Attribute, e Element, u xsd.AttributeUse) {
	d, resolved := w.schema.ResolvedAttributeDeclaration(u)
	if !resolved {
		w.logAttribute(a, ruleCvcAttribute, "3", "declined")
		return
	}
	st, simple := w.schema.ResolvedSimpleType(d.TypeDefinition())
	if !simple {
		w.logAttribute(a, ruleCvcAttribute, "3", "declined")
		return
	}
	w.simpleAssertions(st, a.Loc())
	if _, err := value.ValidateLexical(w.backend, w.schema, st, a.Value(), elementContext{owner: e}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			w.logAttribute(a, ruleCvcAttribute, "3", "declined")
			return
		}
		w.res.violations = append(w.res.violations, causedBy(ruleCvcAttribute, a.Loc(), err,
			"the ·initial value· of the attribute %s is not ·valid· with respect to its declaration's {type definition} %s, which cvc-attribute clause 3 requires as per String Valid (§3.16.4)",
			a.Name(), st.Name()))
		w.logAttribute(a, ruleCvcAttribute, "3", "charged")
		return
	}
	w.logAttribute(a, ruleCvcAttribute, "3", "satisfied")
	if f, fixed := declarationFixed(d); fixed {
		w.fixedAgreement(a, e, st, f)
	}
	if f, fixed := useFixed(u); fixed {
		w.fixedAgreement(a, e, st, f)
	}
}

// instanceTypeResolves charges cvc-attribute (§3.2.4.1) clause 5 against e's
// xsi:type attribute: where D is the built-in declaration for the type
// attribute (§3.2.7.1), A's ·actual value· must ·resolve· to a type definition.
//
// It is reached from [walk.attributes] (assess.go) and never through
// [walk.matchedAttribute], because no attribute use ever matches xsi:type —
// cvc-complex-type clause 2 excepts the four Built-in Attribute Declarations by
// name (isInstanceAttribute) and §3.2.6 a-props-correct forbids a schema to
// redeclare them. That exception is clause 2's own quantifier and nothing
// wider: the Note under §3.2.4.2 keeps an xsi:type attribute governed by its
// built-in declaration whatever the element's ·governing type definition· is,
// so cvc-assess-elt clause 2 (§3.3.4.6) walks the item through cvc-attribute
// regardless, and clause 5 is the part of that walk the built-in declaration
// adds. It is called for BOTH arms of cvc-type clause 3 for the same reason:
// the governing type decides nothing about an attribute it never governs.
//
// The charge is the ATTRIBUTE's alone and the ELEMENT is untouched by it: a
// lexical that does not ·resolve· leaves E with no ·instance-specified type
// definition· at all, so cvc-elt clause 4 stays vacuous and the element is
// assessed against its ·selected type definition· (the Note under cvc-elt,
// [walk.instanceTypeDefinition]). Clause 5 carries no such fallback wording and
// charges whether or not that fallback succeeds.
//
// GAP(validate): clause 5 is the only clause any xsi:type attribute is charged
// under, and a lexical outside xs:QName's lexical space falls the wrong side of
// it in EITHER direction, according to where [resolveInstanceQName] (cvcelt.go)
// leaves it. That function splits on emptiness and colon placement and never
// checks either half against the NCName production, so its two sides do not
// divide QNames from non-QNames:
//
//   - DECLINED. An empty lexical, a colon structure no QName has, and a prefix
//     with no binding in scope at E stop at the split (instanceTypeNoValue) and
//     are charged by nothing. Clause 5 quantifies over A's ·actual value·, which
//     such a lexical does not have, so charging it here would report it under a
//     clause that does not hold it.
//   - CHARGED, wrongly. Every OTHER lexical outside that lexical space — 23.789,
//     not a name — clears the split, and the name it yields resolves to nothing,
//     so it is charged below as though it were a QName naming no type.
//
// Both halves are clause 3's, as String Valid against the built-in declaration's
// xs:QName {type definition}, and clause 3 has no site for xsi:type — matchedAttribute
// is its only one and no attribute use ever reaches the item. One such site,
// running value.ValidateLexical the way matchedAttribute already does, retires
// the pair: the declined lexicals gain the charge they are owed and the charged
// ones move to the clause that holds them.
func (w *walk) instanceTypeResolves(e Element) {
	a, present := instanceAttribute(e, "type")
	if !present {
		return
	}
	switch name, _, outcome := w.resolveInstanceType(e, a); outcome {
	case instanceTypeResolved:
		w.logAttribute(a, ruleCvcAttribute, "5", "satisfied")
	case instanceTypeNoValue:
		w.logAttribute(a, ruleCvcAttribute, "5", "declined")
	case instanceTypeUnresolved:
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcAttribute, a.Loc(),
			"the xsi:type attribute of the element %s has the lexical %q, and the schema declares no type definition named %q for it to ·resolve· to (§3.17.6.3, cvc-resolve-instance), which cvc-attribute clause 5 requires of an attribute governed by the built-in declaration for the type attribute (§3.2.7.1)",
			e.Name(), a.Value(), name))
		w.logAttribute(a, ruleCvcAttribute, "5", "charged")
	}
}

// fixedConstraint is one fixed {value constraint} together with the rule that
// reads it and the words that rule's message needs: the clause it charges (empty
// for cvc-au, which is one undivided sentence with no numbered clauses) and the
// component the constraint sits on. They travel WITH the constraint rather than
// being derived from the rule ID at the message site, so adding a third reader
// is one constructor and not a third arm of a mapping that can go stale.
type fixedConstraint struct {
	vc     xsd.ValueConstraint
	rule   xsderr.Rule
	clause string
	owner  string
}

// fixedAgreement charges the "equal or identical" agreement one fixed {value
// constraint} demands of a present attribute — cvc-attribute clause 4 for the
// declaration's, cvc-au for the use's — or nothing at all where there is no
// fixed constraint to agree with, which is the vacuous case both rules state.
//
// The comparison is in the VALUE space and never over lexicals:
// [value.ConstraintMatches] maps both sides through st's pipeline, the
// attribute's under the instance's namespace context and the constraint's
// under the schema document's, so "1" and "01" agree as one xs:integer and
// "a:x" and "b:x" agree exactly when both prefixes name one namespace.
//
// An undecided answer charges nothing. GAP(validate): that covers a type this
// backend does not govern and a {lexical form} outside its own type's lexical
// space (a schema fault cos-valid-simple-default charges at assembly, not the
// instance's). Charging on undecided would reject a document for a gap in the
// processor (#774).
func (w *walk) fixedAgreement(a Attribute, e Element, st *xsd.SimpleType, f fixedConstraint) {
	same, decided := value.ConstraintMatches(w.backend, w.schema, st, a.Value(), elementContext{owner: e}, f.vc)
	if !decided {
		w.logAttribute(a, f.rule, f.clause, "declined")
		return
	}
	if same {
		w.logAttribute(a, f.rule, f.clause, "satisfied")
		return
	}
	w.res.violations = append(w.res.violations, xsderr.New(f.rule, a.Loc(),
		"the ·actual value· of the attribute %s is neither equal nor identical to the {value} of the fixed {value constraint} %q on its %s, which %s requires",
		a.Name(), f.vc.LexicalForm(), f.owner, citation(f.rule, f.clause)))
	w.logAttribute(a, f.rule, f.clause, "charged")
}

// declarationFixed reports D.{value constraint} where its {variety} is fixed —
// the only shape cvc-attribute clause 4 tests.
func declarationFixed(d xsd.AttributeDeclaration) (fixedConstraint, bool) {
	vc, has := d.ValueConstraint()
	if !has || vc.Kind() != xsd.ValueFixed {
		return fixedConstraint{}, false
	}
	return fixedConstraint{vc: vc, rule: ruleCvcAttribute, clause: "4", owner: "attribute declaration"}, true
}

// useFixed reports U.{value constraint} where its {variety} is fixed — the only
// shape cvc-au tests. It reads the USE's own property and never the ·effective
// value constraint·, which would substitute the declaration's fixed value for a
// use that carries none and charge cvc-au for an agreement the rule does not
// demand.
func useFixed(u xsd.AttributeUse) (fixedConstraint, bool) {
	vc, has := u.ValueConstraint()
	if !has || vc.Kind() != xsd.ValueFixed {
		return fixedConstraint{}, false
	}
	return fixedConstraint{vc: vc, rule: ruleCvcAu, owner: "attribute use"}, true
}

// citation renders the rule and clause a message names inline (STYLE E4), and
// the bare rule for one that has no numbered clauses at all.
func citation(rule xsderr.Rule, clause string) string {
	if clause == "" {
		return string(rule)
	}
	return string(rule) + " clause " + clause
}

// defaultedAttributes charges cvc-complex-type (§3.4.4.2) clause 4: for each
// ·defaulted attribute· of e, the {lexical form} of its ·effective value
// constraint· must be ·valid· with respect to the DECLARATION's {type
// definition} per String Valid. It validates a value the schema supplies, not
// one the document carries — the attribute is absent, which is what makes it
// defaulted — so it is the one charge here that a schema alone determines.
//
// The five conjuncts of ·defaulted attribute· (key-dflt-att) are applied in
// their own order: membership in {attribute uses} is the range, then
// {required} = false (clause 2), then a non-·absent· ·effective value
// constraint· (clause 3), then not one of the four Built-in Attribute
// Declarations (clause 4, isInstanceAttribute — §3.2.6 a-props-correct forbids
// a schema to redeclare those names, so the name settles it), then no
// attribute information item matching the declaration per clause 2.1
// (clause 5).
//
// The scan is over governing.AttributeUses() in document order, never a map, so
// several charges on one element arrive in a fixed order (STYLE D2). attrs is
// e.[[attributes]] as [walk.attributes] read it, so clause 5 quantifies over
// the same reading of the source clauses 2 and 3 did.
//
// A charge here is rare by construction rather than by accident: a schema
// finalized with a value space installed has already charged
// cos-valid-simple-default (§3.2.6.2) against the same {lexical form}, under
// a-props-correct clause 2 or au-props-correct clause 2, so only a schema
// assembled through [xsd.SchemaBuilder.Finalize] — which installs none —
// reaches this clause with an invalid default still in it.
func (w *walk) defaultedAttributes(e Element, attrs []Attribute, governing xsd.ComplexType) {
	for _, u := range governing.AttributeUses() {
		if u.Required() { // clause 2
			continue
		}
		vc, constrained := w.schema.EffectiveValueConstraint(u)
		if !constrained { // clause 3
			continue
		}
		if isInstanceAttribute(u.DeclarationName()) { // clause 4
			continue
		}
		if hasAttributeNamed(attrs, u.DeclarationName()) { // clause 5
			continue
		}
		w.defaultedAttribute(e, u, vc)
	}
}

// defaultedAttribute charges clause 4 for one ·defaulted attribute·. The type
// is read off A.{attribute declaration}.{type definition} and never off the
// use, which the clause is explicit about.
//
// The question is Datatype Valid over one {lexical form} against one type,
// which is what [xsd.ValueSpace]'s ValidDefault decides — the same decision
// a-props-correct clause 2 and au-props-correct clause 2 charge at assembly, and
// reached here through the same seam rather than re-derived, over the one value
// space [Validator.Assess] built for this walk. Its undecided answer carries the
// whole fail-open gate: an ungoverned type, a context-dependent one, a
// construction-stage facet failure and a facet-pipeline precondition fault each
// charge nothing. A decided rejection hands back the Datatype Valid verdict
// itself, which the charge carries as its wrapped cause (validate.go's causedBy).
//
// The type's assertion sites are recorded at the ELEMENT's location, on
// [walk.simpleAssertions]'s terms: the attribute is absent, which is what makes
// it defaulted, so there is no attribute item to carry a Loc.
func (w *walk) defaultedAttribute(e Element, u xsd.AttributeUse, vc xsd.ValueConstraint) {
	d, resolved := w.schema.ResolvedAttributeDeclaration(u)
	if !resolved {
		return
	}
	st, simple := w.schema.ResolvedSimpleType(d.TypeDefinition())
	if !simple {
		return
	}
	w.simpleAssertions(st, e.Loc())
	cause, decided := w.values.ValidDefault(w.schema, st, vc)
	if !decided || cause == nil {
		return
	}
	w.res.violations = append(w.res.violations, causedBy(ruleCvcComplexType, e.Loc(), cause,
		"the element %s carries no attribute information item named %s, and the {lexical form} %q of the ·effective value constraint· of the ·defaulted attribute· it would supply is not ·valid· with respect to that declaration's {type definition} %s, which cvc-complex-type clause 4 requires as per String Valid (§3.16.4)",
		e.Name(), u.DeclarationName(), vc.LexicalForm(), st.Name()))
}
