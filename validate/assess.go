package validate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCvcAssessElt is Schema-Validity Assessment (Element) (Structures
// §3.3.4.6, cvc-assess-elt). Its three clauses only dispatch — strictly
// assessed, not assessed, laxly assessed — so a ·validation root· that
// determines no declaration and no type definition violates no numbered
// clause of it. The charge stands instead on §5.2 strict wildcard
// validation, whose Note has the invoking process expecting the
// ·validation root· "to be declared and valid" and otherwise reporting "an
// error to its environment"; cvc-assess-elt is the rule that policy is
// stated against, and the catalog carries the bare name.
const ruleCvcAssessElt xsderr.Rule = "cvc-assess-elt"

// ruleCvcElt is Element Locally Valid (Element) (Structures §3.3.4.3,
// cvc-elt). The clause charged goes in the message, not the rule ID: unlike
// some rules whose dotted sub-ID the spec itself anchors (cos-ct-extends.1.2,
// src-simple-type.1), cvc-elt's catalog entry is the bare name only, so
// "cvc-elt.2" is not a valid [xsderr.Rule] (see [xsderr.IsValidRule]).
const ruleCvcElt xsderr.Rule = "cvc-elt"

// ruleCvcComplexType is Element Locally Valid (Complex Type) (Structures
// §3.4.4.2, cvc-complex-type). Clauses 2 and 3 — the attribute half — are
// charged here; the clause number goes in the message on ruleCvcElt's terms.
const ruleCvcComplexType xsderr.Rule = "cvc-complex-type"

// Assess walks root's subtree once — the element, then its [[attributes]],
// then its [[children]] in document order, recursively — and reports what
// the walk found. The walk topology is cvc-assess-elt's (§3.3.4.6, ·strictly
// assessed·).
//
// It decides the ·governing element declaration· of the validation root,
// which is the declaration root's ·expanded name· ·resolves· to among the
// schema's top-level element declarations (§3.3.4.6, ·governing element
// declaration· clause 4), and charges two rules over that dispatch:
//
//   - No such declaration and no xsi:type, so no ·governing type
//     definition· can exist either and cvc-assess-elt clause 1 cannot
//     apply: cvc-assess-elt is charged and the subtree is not walked.
//     Clause 3 would have the root ·laxly assessed· against xs:anyType,
//     which this package does not implement; charging instead is §5.2
//     strict wildcard validation.
//   - A declaration whose {abstract} is true: cvc-elt clause 2 is charged
//     and the walk still runs. ·Strictly assessed· clauses 2 and 3 assess
//     [[attributes]] and [[children]] whatever clause 1.1.2's evaluation
//     returned, so an abstract root does not silence its subtree.
//
// cvc-elt clause 1 (D ·non-absent·, E and D sharing an ·expanded name·) is
// satisfied by construction — D is the declaration found BY that expanded
// name — and so is never charged.
//
// Where the root's ·governing type definition· is determinable and complex
// (see governingType), the root itself is additionally assessed against
// it, in both directions cvc-complex-type (§3.4.4.2) quantifies in. Its
// [[attributes]] go to clauses 2, 3 and 4, and through clause 2.1 to the
// cvc-attribute (§3.2.4.1) and cvc-au (§3.5.4) charges the value space
// decides (see [walk.attributes] and cvcattribute.go). Its [[children]] go to
// clause 1, and through clause 1.4 to cvc-complex-content (§3.4.4.3) over
// [xsd.Matcher] (see cvccomplexcontent.go).
//
// A DESCENDANT is assessed the same way, against the ·governing element
// declaration· the particle its parent's content model ·attributes· it to
// supplies (§3.3.4.6 clause 3.1 over §3.4.4.4, [walk.childGoverning]): its own
// [[attributes]], its own [[children]] and its own ·initial value· reach the
// same charges the root's do. Two shapes below the root are not: a child
// ·attributed to· a skip Wildcard, which is ·skipped· along with every element
// beneath it (clause 3.2), and a child whose declaration this package cannot
// determine, which is walked against no type at all — clause 3.3's ·lax
// assessment· against xs:anyType, whose {content type} and {attribute uses}
// constrain nothing any of these charges reads.
//
// Nothing else is decided: the remaining cvc-elt clauses, the rest of
// cvc-type (§3.3.4.4) and cvc-complex-type's own clauses 5 and 6 are not
// evaluated, so a [Result] carrying no violation says the root is declared,
// not abstract, and — where its type was determinable — carries no attribute
// clause 2 rejects, no required attribute clause 3 misses, no attribute whose
// value this backend could read and found invalid, and no clause 1 content
// reject its {content type}.{variety} and particle could settle, and says
// nothing else about the document.
//
// It panics if root is nil, on the same grounds as [ElementChild].
func (v *Validator) Assess(root Element) *Result {
	if root == nil {
		panic("validate: Assess: nil root Element")
	}
	w := walk{log: v.log, schema: v.schema, backend: v.backend, values: value.NewValueSpace(v.backend)}
	// GAP(xsd): an xsi:type here is DETECTED, never ·resolved·, so the
	// charge below is withheld for a root carrying one that a full
	// cvc-resolve-instance (§3.17.6.3) would find unresolvable and charge
	// anyway. The direction is fail-open across the whole consumer set of
	// the withheld value, which is Result.violations and its one reader
	// Result.Violations: both carry violations PRESENT, so withholding one
	// can only cost a rejection and never manufacture one (#716).
	d, found := v.Schema().Element(root.Name())
	if !found && !hasInstanceType(root) {
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcAssessElt, root.Loc(),
			"the validation root %s has no top-level element declaration and no xsi:type, so it determines neither a ·governing element declaration· nor a ·governing type definition· and cannot be ·strictly assessed· (§5.2 strict wildcard validation)",
			root.Name()))
		return &w.res
	}
	if found && d.Abstract() {
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcElt, root.Loc(),
			"the validation root %s is governed by an element declaration whose {abstract} is true, but cvc-elt clause 2 requires it to be false: an abstract declaration validates no element information item",
			root.Name()))
	}
	var g governance
	if found {
		g = w.declaredGovernance(root, d)
	}
	if !found {
		// Reached only for a root carrying an xsi:type, the branch above having
		// returned for every other undeclared root: its ·instance-specified type
		// definition· is the governing one and this package does not ·resolve·
		// it, so the whole subtree is typed against nothing and cvc-id clause 1
		// declines with it (cvcid.go) (#716).
		w.ids.declined = true
	}
	w.element(root, g, nil)
	w.ids.charge(&w, root)
	return &w.res
}

// governance is what cvc-assess-elt (§3.3.4.6) settles about one element
// information item before any rule reads it: the ·governing element
// declaration· the descent determined for it, and the ·governing type
// definition· that declaration supplies. It travels as one value because every
// consumer past the first needs both — the type decides cvc-complex-type's two
// halves, and the DECLARATION carries the {identity-constraint definitions}
// §3.11.4 quantifies over and the {nillable} its clause 4.2.3 reads.
//
// The zero value is an element assessed against nothing: no declaration, no
// type. hasDecl false with a nil typ is cvc-assess-elt clause 3.3's ·lax
// assessment· against xs:anyType, and hasDecl TRUE with a nil typ is this
// package declining a type it could not determine — a distinction cvcid.go
// needs, since only the second could have hidden an ID.
type governance struct {
	decl    xsd.ElementDeclaration
	hasDecl bool
	typ     xsd.TypeDefinition
}

// complexType narrows the ·governing type definition· to the Complex Type
// Definition cvc-type clause 3.2 dispatches to cvc-complex-type for, or nil for
// anything else — including a SIMPLE governing type, whose element's own
// attributes are governed by cvc-type clause 3.1.1 instead, which this package
// does not decide either.
func (g governance) complexType() *xsd.ComplexType {
	ct, isComplex := g.typ.(xsd.ComplexType)
	if !isComplex {
		return nil
	}
	return &ct
}

// simpleType narrows the ·governing type definition· to the simple type an
// element's ·initial value· is read under — the two shapes §3.11.4 clause 3
// admits as a field node, "a simple type definition or a complex type
// definition with {variety} simple" — and nil for every other, which carries no
// [schema actual value] to be a ·key-sequence· member or an ·ID value·.
func (g governance) simpleType() *xsd.SimpleType {
	switch t := g.typ.(type) {
	case *xsd.SimpleType:
		return t
	case xsd.ComplexType:
		if sc, isSimple := t.ContentType().(xsd.SimpleContent); isSimple {
			return sc.SimpleType
		}
	}
	return nil
}

// declaredGovernance pairs a ·governing element declaration· with the
// ·governing type definition· it supplies for e (governingType).
func (w *walk) declaredGovernance(e Element, d xsd.ElementDeclaration) governance {
	return governance{decl: d, hasDecl: true, typ: w.governingType(e, d)}
}

// governingType is the ·governing type definition· (§3.3.4.6) of an
// element information item whose ·governing element declaration· is d; it is nil
// wherever this package cannot determine that type, and the element's attributes
// are then assessed against nothing.
//
// Determinable here means the whole of key-governing-type-elem's clause 4
// case: no processor-stipulated type (this package stipulates none), no
// ·instance-specified type definition· overriding it, and a ·selected type
// definition· (§3.3.4.1) that is D.{type definition} outright. Each decline
// below withholds a type that could differ from the declaration's, and
// assessing the element against the WRONG type is a false reject in both
// directions — an attribute the real governing type declares looks unmatched,
// a child its real {content type} admits looks unattributable.
//
//   - GAP(xsd): an xsi:type makes the ·instance-specified type definition·
//     the governing one (clause 3) when it ·overrides· the selected type,
//     and ·resolving· it is cvc-resolve-instance (§3.17.6.3), unimplemented.
//     Presence alone is the decline, at the ·validation root· and at every
//     descendant the descent reaches alike: a descendant carrying xsi:type is
//     assessed against no type rather than against the one its
//     ·context-determined declaration· names, which the attribute may have
//     ·overridden· with a derived type admitting content the declared one
//     rejects (#716).
//   - GAP(xpath): a {type table} makes the selected type the one its
//     <alternative>s ·conditionally select· (§3.3.4.2), which means
//     evaluating each {test} as an XPath expression (#56).
//   - A {type definition} slot that resolves to nothing.
//
// The type this returns is the governing one for EVERY reader, and they narrow
// it separately from here on: governance.complexType for cvc-complex-type's two
// halves, governance.simpleType for the ·initial value· §3.11.4 clause 3 and
// §3.17.5.2 read. The attribute half narrows once more through
// attributePropertiesFolded, which the content half does not, because no
// finalize pass folds a {content type} the way §3.4.2.4 clause 3 folds
// {attribute uses} — a complex type's {content type} is whatever its producer
// built for it, named or anonymous.
func (w *walk) governingType(e Element, d xsd.ElementDeclaration) xsd.TypeDefinition {
	if hasInstanceType(e) {
		return nil
	}
	if _, tabled := d.TypeTable(); tabled {
		return nil
	}
	t, ok := w.schema.ResolvedType(d.TypeDefinition())
	if !ok {
		return nil
	}
	return t
}

// childGoverning is cvc-assess-elt (§3.3.4.6) clause 3 for one child element
// information item, read off the ·attribution· its parent's content model just
// gave it (§3.4.4.4): the ·governing element declaration· and ·governing type
// definition· to assess it against, determined exactly as the ·validation
// root·'s are (declaredGovernance), and whether it is assessed at all.
//
// assess is false for one shape alone, clause 3.2's: a child ·attributed to· a
// skip Wildcard is not ·assessed·, and neither is any element below it —
// ·skipped· (§3.10.4.1, key-skipped) holds for an item "attributed to a skip
// wildcard or if one of its ancestor elements is". cvc-wildcard makes that a
// hard stop and not a permissive pass: a skip wildcard leaves the item with no
// ·governing element declaration· at all and runs no ·QName resolution· to look
// for one, so skip and clause 3.3 are different outcomes and not two spellings
// of one.
//
// The declaration the type is read off is key-governing-ed's, one case per
// [xsd.Attribution] variant (STYLE T2's closed-sum exception):
//
//   - clause 2, the ·context-determined declaration·: an ElementDeclaration
//     attribution whose ·expanded name· is the child's own is the {term} of the
//     element particle the child was ·attributed to·.
//   - clause 2 through the ·substituting declaration·: an ElementDeclaration
//     attribution with ANOTHER name is the particle's own D, which cvc-accept
//     clause 2.3.2 admitted the child under as a member of D's ·substitution
//     group·. The ·context-determined declaration· is then that member and
//     never D ([xsd.Attribution]), and a member is top-level by construction
//     (§3.3.6.4, cos-equiv-class), so it is the resolution below.
//   - clause 3, for a strict or a lax Wildcard: the declaration the child's
//     ·expanded name· ·resolves· to among the schema's top-level element
//     declarations, which is the resolution [Validator.Assess] makes for the
//     root. The two {process contents} share it exactly — §3.10.4.1 draws no
//     distinction in the resolution step itself — and what they differ in is
//     what an UNRESOLVED name under a strict wildcard costs the PARENT's
//     [validity] (§3.3.5.1 clause 1.1.3), a property this package computes for
//     no item at all.
//
// A nil attribution is a parent that attributed the child to nothing: no
// ·governing type definition· of its own, an element already charged, or a
// child clause 1.4 declined or rejected. It leaves the child assessed against
// nothing, which is where every descendant was before the descent existed.
func (w *walk) childGoverning(e Element, a xsd.Attribution) (governance, bool) {
	switch t := a.(type) {
	case xsd.ElementDeclaration:
		if t.Name() != e.Name() {
			return w.resolvedGovernance(e), true
		}
		return w.declaredGovernance(e, t), true
	case xsd.Wildcard:
		if t.ProcessContents() == xsd.ProcessSkip {
			return governance{}, false
		}
		return w.resolvedGovernance(e), true
	default:
		return governance{}, true
	}
}

// resolvedGovernance is the ·governing element declaration· of an element that
// is the one its ·expanded name· ·resolves· to among the schema's top-level
// element declarations (cvc-resolve-instance, §3.17.6.3), together with the
// type that declaration supplies. A name that resolves to nothing leaves the
// element with no declaration to read a type off, which is cvc-assess-elt
// clause 3.3 and no charge of its own: an unresolved name is the PARENT's
// business where it is anyone's (§3.3.5.1 clause 1.1.3) and never the child's.
func (w *walk) resolvedGovernance(e Element) governance {
	d, found := w.schema.Element(e.Name())
	if !found {
		return governance{}
	}
	return w.declaredGovernance(e, d)
}

// attributePropertiesFolded reports whether ct's {attribute uses} and
// {attribute wildcard} are the §3.4.2.4 clause 3 and §3.4.2.5 clause 2
// properties cvc-complex-type clause 2 quantifies over, rather than the
// producer's pre-fold approximation of them.
//
// GAP(xsd): both folds walk a finalized schema's {type definitions} alone
// (#414), and every member of that property is NAMED — §3.17.2 scopes it to
// the <simpleType>/<complexType> children of <schema> — so a complex type
// whose {name} is ·absent· is folded for neither property and reports its own
// <attribute> children and its own ·complete wildcard· alone. Under-report the
// uses and an inherited attribute looks unmatched; under-report the wildcard
// and [walk.unmatchedAttribute]'s clause 2.2 decline does not fire. Together
// they charge clause 2 against an attribute the base admits, so an anonymous
// governing type is declined here rather than assessed.
//
// The one anonymous shape admitted is a RESTRICTION of xs:anyType, where both
// folds are provably the identity: §3.4.7 gives xs:anyType an empty {attribute
// uses}, so clause 3 inherits nothing, and clause 2 unions the base's wildcard
// for an EXTENSION only, so clause 2.1 keeps the ·complete wildcard· the
// producer already stored. §3.4.2.3.2 maps the implicit-content
// <complexType> — no <simpleContent>, no <complexContent> — to exactly that
// shape.
//
// Anonymity is read off the component's {name} and not off the arm of the
// {type definition} slot it was reached through, because
// [xsd.SubstitutionGroupHeadTypeRef] reaches an unfolded anonymous type the
// head declaration owns just as [xsd.InlineTypeDefinition] reaches its own.
// Neither arm is exotic: parser.Parse maps an inline
// <complexContent>/<simpleContent> derivation under an <element> exactly as it
// maps a top-level one (parser/produce_complex.go's produceComplexType
// dispatches on those children before it considers the type's name), so a
// document a caller parses reaches this decline without [xsd.SchemaBuilder]
// being involved at all.
func attributePropertiesFolded(ct xsd.ComplexType) bool {
	if ct.Name() != (xsd.QName{}) {
		return true
	}
	if ct.DerivationMethod() != xsd.DerivationRestriction {
		return false
	}
	base, byName := ct.Base().(xsd.TypeDefinitionRef)
	return byName && base.Name == (xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"})
}

// hasInstanceType reports whether e carries an xsi:type attribute (§2.7.1).
// It is a presence test and not a ·resolution·: establishing the
// ·instance-specified type definition· that §5.2 counts as one of the
// ·validation root·'s three determinants needs cvc-resolve-instance
// (§3.17.6.3), and presence is the upper bound on it available short of
// that.
func hasInstanceType(e Element) bool {
	for _, a := range e.Attributes() {
		if a.Name() == (xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}) {
			return true
		}
	}
	return false
}

// walk is the state of one [Validator.Assess] call, held here and not on the
// Validator so nothing survives the call that made it. schema and backend are
// the Validator's, reachable from the walk methods that need them — the one
// resolves component slots, the other reads instance lexicals — and neither is
// written here.
//
// values is [value.NewValueSpace] over that same backend, built once per
// assessment for the one charge that asks a value-constraint question of the
// SCHEMA rather than of the instance (cvc-complex-type clause 4,
// [walk.defaultedAttribute]). It is not derived state to be re-derived per call:
// the constructor is total on a non-nil backend and the result is immutable, so
// building it per ·defaulted attribute· would allocate once per use per element
// to reach the same object. nodes counts the element information items the walk
// has entered, and the count doubles as each one's IDENTITY: §3.11.5's conflict
// resolution turns on "the same key-sequence but distinct nodes" and §3.17.5.2's
// [binding] is a SET of elements, and an [Element] is an interface whose ==
// compares whatever an adapter's dynamic type compares. ids is the [ID/IDREF
// table] those ordinals bind into, assembled across the whole walk and read
// once, at the ·validation root· (cvcid.go).
type walk struct {
	log     *slog.Logger
	schema  *xsd.Schema
	backend value.Backend
	values  xsd.ValueSpace
	nodes   int
	ids     idTable
	res     Result
}

// elementContext is the [value.Context] an instance lexical is mapped under:
// the namespace bindings in scope at the element that OWNS it, which is where a
// QName- or NOTATION-valued lexical resolves its prefix (Datatypes §3.3.18,
// §3.3.19, PRINCIPLES 19). The owner is the attribute's element for an
// attribute's lexical (cvc-attribute clause 3) and the element itself for its
// ·initial value· (cvc-complex-type clause 1.2).
//
// It exists so no site passes a nil Context. A nil one makes a backend reject
// every prefixed QName lexical for want of bindings, which is a false reject of
// every xsi-free QName value in existence.
type elementContext struct{ owner Element }

func (c elementContext) LookupNamespace(prefix string) (string, bool) {
	return c.owner.LookupPrefix(prefix)
}

// element assesses one element information item: the item itself, then its
// [[attributes]], then its [[children]].
//
// governing is e's ·governing type definition· narrowed to a complex type,
// or nil where none was determined. It decides e's own [[attributes]]
// (cvc-complex-type clauses 2 to 4) and e's own [[children]] (clause 1,
// cvccomplexcontent.go), and it is not propagated to the children as it
// stands: each of them gets its OWN, off the particle e's {content type}
// ·attributes· it to (§3.4.4.4, [walk.childGoverning]), which is
// cvc-assess-elt clause 3.1's "the one identified in the course of checking
// the local validity of the parent". A nil governing type therefore ends the
// typed descent at e — a check with no type attributes nothing — and e's
// subtree is walked against nil throughout.
//
// parent is the enclosing element's identity-constraint state, nil at the
// ·validation root·. It is what carries the {selector} and {fields} evaluations
// downward and the node tables back up (cvcidentityconstraint.go), so the
// bracketing here is load-bearing: identityCheck opens e's state before its
// [[attributes]] are read, because a field path may name one of them, and
// identityExit settles §3.11.4 and §3.11.5 only once e's [[children]] are
// exhausted, because a ·key-sequence· and a node table are complete only then.
func (w *walk) element(e Element, g governance, parent *icCheck) {
	if w.log.Enabled(context.Background(), slog.LevelDebug) {
		w.log.Debug("assessing element", slog.Any("name", e.Name()), slog.Any("loc", e.Loc()))
	}
	governing := g.complexType()
	id := w.identityCheck(e, g, parent)
	w.idAttributes(id)
	w.attributes(e, governing)
	w.children(e, w.contentCheck(e, governing), id)
	if w.res.err != nil {
		// A walk that stopped on a source fault never settles §3.11.4 or
		// §3.17.5.2 for this element, on [contentCheck.end]'s grounds: the
		// [[children]] the source never finished delivering are ·target nodes·,
		// field values and ID declarations the rules would otherwise be charged
		// for the absence of.
		return
	}
	w.idElement(id)
	w.identityExit(id)
}

// attributes assesses E.[[attributes]] against governing, in the three
// directions cvc-complex-type (§3.4.4.2) quantifies in: clause 2 over the
// attribute information items PRESENT, in source order, then clause 3 over
// the {attribute uses} that must be present, then clause 4 over the
// ·defaulted attributes·. Violations reach [Result] in that order, which is
// the order they were found in.
//
// A nil governing decides nothing at all: every attribute is walked (the
// log records the visit) and none is charged or passed. A type whose two
// attribute PROPERTIES are not the spec's yet is narrowed to that same
// nothing here, and here only — its {content type} still decides its
// element's [[children]] (see attributePropertiesFolded and
// governingType).
func (w *walk) attributes(e Element, governing *xsd.ComplexType) {
	folded := governing
	if folded != nil && !attributePropertiesFolded(*folded) {
		folded = nil
	}
	attrs := e.Attributes()
	for _, a := range attrs {
		w.attribute(a, e, folded)
	}
	if folded == nil {
		return
	}
	w.requiredAttributeUses(e, attrs, *folded)
	w.defaultedAttributes(e, attrs, *folded)
}

// attribute assesses one attribute information item against clause 2, whose
// two arms are the whole rule: an attribute matching an attribute use is
// judged by cvc-attribute and cvc-au (clause 2.1, matchedAttribute), one
// matching none needs an {attribute wildcard} that admits it (clause 2.2).
// There is no third arm, so an attribute that matches neither violates clause
// 2 outright.
//
// e is the attribute's owner, carried for the namespace bindings a QName- or
// NOTATION-valued lexical resolves against (elementContext).
func (w *walk) attribute(a Attribute, e Element, governing *xsd.ComplexType) {
	if governing == nil {
		w.logAttribute(a, "", "", "")
		return
	}
	if isInstanceAttribute(a.Name()) {
		// Clause 2 excepts xsi:type, xsi:nil, xsi:schemaLocation and
		// xsi:noNamespaceSchemaLocation (§3.2.7) from its quantifier by name,
		// so no arm of it applies to one: it is neither charged nor passed.
		w.logAttribute(a, ruleCvcComplexType, "2", "exempt")
		return
	}
	u, matched := attributeUseNamed(governing.AttributeUses(), a.Name())
	if !matched {
		w.unmatchedAttribute(a, *governing)
		return
	}
	w.matchedAttribute(a, e, u)
}

// unmatchedAttribute settles clause 2 for an attribute matching no attribute
// use, where only clause 2.2 is left to satisfy.
//
// This is the one charge in the package that a property REPORTED TOO SMALL
// could fabricate: [xsd.ComplexType.AttributeUses] and
// [xsd.ComplexType.AttributeWildcard] are folded for a NAMED type definition
// alone (#414). Nothing is asserted about that here — the unfolded types are
// kept away from this charge at the source instead, by
// attributePropertiesFolded, which is where the bound and its spec grounds
// are stated. An anonymous governing type never reaches this function.
func (w *walk) unmatchedAttribute(a Attribute, governing xsd.ComplexType) {
	if _, wild := governing.AttributeWildcard(); wild {
		// GAP(validate): clause 2.2.1 holds, but clause 2.2 is a conjunction
		// and 2.2.2 sends the attribute to cvc-wildcard (§3.10.4.1), which
		// this package does not evaluate. Charging on 2.2.1 alone would reject
		// every attribute a wildcard admits, so clause 2 is left undecided for
		// an element whose type carries one (#717).
		w.logAttribute(a, ruleCvcComplexType, "2.2", "declined")
		return
	}
	w.res.violations = append(w.res.violations, xsderr.New(ruleCvcComplexType, a.Loc(),
		"the attribute %s matches no attribute use among the {attribute uses} of its element's ·governing type definition· (cvc-complex-type clause 2.1), which has no {attribute wildcard} for it to be ·valid· with respect to either (clause 2.2.1), and clause 2 has no third arm",
		a.Name()))
	w.logAttribute(a, ruleCvcComplexType, "2", "charged")
}

// requiredAttributeUses charges clause 3 for each {required} attribute use
// of governing that no attribute information item of e carries the
// ·expanded name· of. It is an existence test over names and nothing more:
// whether a matching item is itself ·valid· is clause 2's question, and an
// OPTIONAL use with no matching item is silent here whatever its ·effective
// value constraint· says.
//
// attrs is e.[[attributes]] as [walk.attributes] read it, so the two clauses
// quantify over one reading of the source. The scan is NOT filtered by
// isInstanceAttribute: clause 2's exception is clause 2's alone, and its own
// Note keeps a use of xsi:type satisfying clause 3 while being ·attributed
// to· nothing.
func (w *walk) requiredAttributeUses(e Element, attrs []Attribute, governing xsd.ComplexType) {
	for _, u := range governing.AttributeUses() {
		if !u.Required() {
			continue
		}
		name := u.DeclarationName()
		if hasAttributeNamed(attrs, name) {
			continue
		}
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcComplexType, e.Loc(),
			"the element %s carries no attribute information item named %s, but its ·governing type definition· has an attribute use for that name whose {required} is true, and cvc-complex-type clause 3 requires one to be present",
			e.Name(), name))
		if w.log.Enabled(context.Background(), slog.LevelDebug) {
			w.log.Debug("assessing attribute use", slog.Any("name", name), slog.Any("loc", e.Loc()),
				slog.String("rule", string(ruleCvcComplexType)), slog.String("clause", "3"),
				slog.String("outcome", "charged"))
		}
	}
}

// logAttribute records one attribute information item's assessment: its
// ·expanded name· and location always, and the rule, clause and outcome that
// settled it wherever anything did (STYLE L1). An attribute assessed against
// no ·governing type definition· has no rule to name and its line carries
// none — the walk visited it and decided nothing.
//
// An empty clause drops the key rather than emitting it empty: cvc-au is one
// undivided sentence with no numbered clauses, so there is no clause to name
// and a "clause=" with nothing after it would read as a missing value.
func (w *walk) logAttribute(a Attribute, rule xsderr.Rule, clause, outcome string) {
	if !w.log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	attrs := []slog.Attr{slog.Any("name", a.Name()), slog.Any("loc", a.Loc())}
	if rule != "" {
		attrs = append(attrs, slog.String("rule", string(rule)))
		if clause != "" {
			attrs = append(attrs, slog.String("clause", clause))
		}
		attrs = append(attrs, slog.String("outcome", outcome))
	}
	w.log.LogAttrs(context.Background(), slog.LevelDebug, "assessing attribute", attrs...)
}

// logSkipped records the one child the walk does not assess at all: cvc-assess-elt
// clause 3.2's, ·attributed to· a skip Wildcard. It is written here because
// [walk.element] is never reached for it, so its "assessing element" line —
// which every other element gets, whatever was or was not decided about it
// (STYLE L1) — has nowhere else to come from, and it carries the outcome that
// says the subtree below is unvisited rather than merely undecided.
func (w *walk) logSkipped(e Element) {
	if !w.log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	w.log.LogAttrs(context.Background(), slog.LevelDebug, "assessing element",
		slog.Any("name", e.Name()), slog.Any("loc", e.Loc()),
		slog.String("rule", string(ruleCvcAssessElt)), slog.String("clause", "3.2"),
		slog.String("outcome", "skipped"))
}

// attributeUseNamed reports the attribute use among uses whose {attribute
// declaration} has the ·expanded name· n, as clause 2.1 matches. The name is
// read off the use itself ([xsd.AttributeUse.DeclarationName]) and never off
// a resolved declaration: the match is by name, and a resolution that failed
// for some unrelated reason must not turn a declared attribute into an
// unmatched one.
func attributeUseNamed(uses []xsd.AttributeUse, n xsd.QName) (xsd.AttributeUse, bool) {
	for _, u := range uses {
		if u.DeclarationName() == n {
			return u, true
		}
	}
	return xsd.AttributeUse{}, false
}

// hasAttributeNamed reports whether attrs holds an attribute information
// item whose ·expanded name· is n, as clause 3 asks.
func hasAttributeNamed(attrs []Attribute, n xsd.QName) bool {
	for _, a := range attrs {
		if a.Name() == n {
			return true
		}
	}
	return false
}

// isInstanceAttribute reports whether n is one of the four attributes
// cvc-complex-type clause 2 excepts by name — xsi:type, xsi:nil,
// xsi:schemaLocation, xsi:noNamespaceSchemaLocation, the Built-in Attribute
// Declarations of §3.2.7, which §3.2.6 a-props-correct forbids a schema to
// redeclare.
func isInstanceAttribute(n xsd.QName) bool {
	if n.Space != xsd.XMLSchemaInstanceNS {
		return false
	}
	switch n.Local {
	case "type", "nil", "schemaLocation", "noNamespaceSchemaLocation":
		return true
	}
	return false
}

// text assesses one run of character information items. It reports the run's
// length rather than its content: instance data does not belong in a log.
// Under the silent default the guard leaves it with no body, on the same
// terms as attribute — the ·initial value· this run contributes to is
// assembled here once cvc-type clause 3.1.3 arrives.
func (w *walk) text(t Text) {
	if !w.log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	w.log.Debug("assessing text", slog.Int("chars", len(t.Data())), slog.Any("loc", t.Loc()))
}

// children pulls e's [[children]] in document order and assesses each against
// content, the check e's own ·governing type definition· licenses. It stops at
// the first fault — one the cursor reports, or one raised deeper in the
// subtree — and keeps it: a later child assessed after a fault would be
// assessed out of a context the source never finished delivering.
//
// A walk that stopped early never reaches [contentCheck.end], so a truncated
// [[children]] cursor cannot be charged for ending short of a particle it
// would have satisfied.
func (w *walk) children(e Element, content *contentCheck, id *icCheck) {
	kids := e.Children()
	for {
		c, ok := kids.Next()
		if !ok {
			break
		}
		w.child(c, content, id)
		if w.res.err != nil {
			return
		}
	}
	if err := kids.Err(); err != nil {
		w.res.err = fmt.Errorf("reading the children of %s at %s: %w", e.Name(), e.Loc(), err)
		return
	}
	content.end(w)
}

// child assesses one child, whichever arm it holds: against content first —
// where the item sits in its parent's {content type} is a fact about the
// PARENT — and then, for an element, as a subtree of its own, against the
// ·governing type definition· that same content check just ·attributed· it
// (cvc-assess-elt clause 3.1, [walk.childGoverning]). A Child holding
// neither arm is an adapter bug, not a fault in the source, so it panics
// rather than reaching [Result.Err] — that field means the walk stopped on a
// source fault, and a CLI would otherwise report a bug in an adapter to a user
// as a broken document.
//
// A ·skipped· child stops here and not one level down, which is what makes it
// the whole SUBTREE that is not ·assessed· (clause 3.2): [walk.element] is the
// only path to a child's own [[children]], so declining to call it leaves every
// element below the skipped one unvisited, whatever its own attribution would
// have been.
func (w *walk) child(c Child, content *contentCheck, id *icCheck) {
	if e, ok := c.Element(); ok {
		a := content.element(w, e)
		g, assess := w.childGoverning(e, a)
		if !assess {
			w.logSkipped(e)
			return
		}
		if a == nil {
			// GAP(validate): a child its parent ·attributed to· nothing is one
			// this package gave up typing and not one §3.3.4.6 leaves untyped,
			// so an ID anywhere beneath it is a declaration cvc-id never saw.
			// The whole consumer set of that decline is idTable.charge's clause
			// 1 arm, which stops charging an empty binding; clause 2 keeps
			// charging, because an unseen item can only ADD members to a
			// binding and never take one away (cvcid.go) (#716).
			w.ids.declined = true
		}
		w.element(e, g, id)
		return
	}
	t, ok := c.Text()
	if !ok {
		panic("validate: walk.child: Child holds neither arm; build one with ElementChild or TextChild")
	}
	content.text(w, t)
	id.text(t)
	w.text(t)
}
