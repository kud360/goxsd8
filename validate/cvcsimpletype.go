package validate

import (
	"slices"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xpath"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCvcSimpleType is String Valid (Structures §3.16.4, cvc-simple-type). This
// package charges it only as the verdict a delegating rule wraps — cvc-attribute
// clause 3, cvc-type clause 3.1.3 and cvc-complex-type clauses 1.2 and 4 — and
// only for clause 3: clauses 1 and 2 are Datatype Valid's, whose verdict
// carries ruleCvcDatatypeValid or a facet rule under it.
const ruleCvcSimpleType xsderr.Rule = "cvc-simple-type"

// ruleCvcDatatypeValid is Datatype Valid (Datatypes §4.1.4,
// cvc-datatype-valid). value.ValidateLexical charges it for every literal the
// backend's mapping and facets reject; this package charges it for the one
// clause-2 condition no backend can decide, a NOTATION value naming no notation
// declared in the schema ([walk.notationsDeclared]).
const ruleCvcDatatypeValid xsderr.Rule = "cvc-datatype-valid"

// stringValid runs String Valid (§3.16.4) over lexical against st, for the
// item at loc whose element is owner: clauses 1 and 2 through
// value.ValidateLexical, then clause 3 ([walk.entitiesDeclared]) and the
// declared-notation half of clause 2 ([walk.notationsDeclared]) over the
// ·actual value· value.ValidateLexical accepted. It reports decided false where
// this package withholds a verdict, and otherwise a nil verdict for a ·valid·
// lexical and the rejection for an invalid one, which the caller charges under
// its own rule with the verdict as the wrapped cause. Where decided is false,
// verdict is instead the value.ValidateLexical error that withheld it, nil
// where clause 3 or the NOTATION check did — which [walk.declineAssertions]
// reads to decline an assertions facet under its own rule.
//
// A ·special· st (isSpecial) passes clause 2 without asking ValidateLexical,
// which no backend answers for one: no clause 1 normalization can move a string
// out of either type's lexical space, and neither type is NOTATION or has it in
// its closure. A ValidateLexical error that is not a VERDICT
// ([value.IsDatatypeVerdict]) withholds one; each caller states that decline's
// GAP on its own terms. The pipeline evaluates st's assertions facets, and
// those of every type it recurses into, through [xpath.FacetAssertions].
func (w *walk) stringValid(st *xsd.SimpleType, lexical string, owner Element, loc xsderr.Loc) (decided bool, verdict error) {
	if isSpecial(st) {
		return w.entitiesDeclared(st, lexical, owner, loc)
	}
	_, err := value.ValidateLexical(w.backend, w.schema, st, lexical, elementContext{owner: owner}, xpath.FacetAssertions())
	if err != nil && !value.IsDatatypeVerdict(err) {
		return false, err
	}
	if err != nil {
		return true, err
	}
	decided, verdict = w.entitiesDeclared(st, lexical, owner, loc)
	if !decided || verdict != nil {
		return decided, verdict
	}
	return w.notationsDeclared(st, lexical, owner, loc)
}

// isSpecial reports whether st is one of the two ·special· datatypes,
// xs:anySimpleType and xs:anyAtomicType (Datatypes §2.4, dt-special), by the
// pointer identity [xsd.AnySimpleType] and [xsd.AnyAtomicType] make
// load-bearing. Datatype Valid (Datatypes §4.1.4, cvc-datatype-valid) holds
// UNCONDITIONALLY for every literal against either — its own first disjunct is
// "T corresponds to a ·special· datatype" — so a value this package reads
// against one is decided, never declined, whatever the backend maps: both
// lexical spaces are every Char sequence and both {facets} are empty (#1788).
// Their value spaces are another matter, the lexical mapping "not a function"
// (Datatypes §3.2.1.2, §3.2.2.2), so a ·key-sequence· member of a special type
// is compared by its lexical, decided only where two such lexicals are
// byte-identical ([sameSpecialMember]). A fixed-value comparison
// ([walk.fixedAgreement], [contentCheck.fixedActualValue]) is decided by
// [value.ConstraintMatches] over the mapping's union of primitive and list
// mappings, and declines only where a member of it cannot answer.
func isSpecial(st *xsd.SimpleType) bool {
	return st == xsd.AnySimpleType() || st == xsd.AnyAtomicType()
}

// entitiesDeclared settles String Valid clause 3 — "Let V be the ·actual value·
// of N with respect to T. Then: Every ·ENTITY value· in V is a ·declared entity
// name·" — over a lexical clauses 1 and 2 have accepted against st. It is the
// ONE check site for both ways the clause fails, and both yield a
// cvc-simple-type verdict at loc:
//
//   - the source does not implement [UnparsedEntities] (walk.entities is nil).
//     Appendix D makes non-support of [unparsedEntities] itself the failure:
//     "all items of type ENTITY or ENTITIES will fail to ·validate·".
//   - the value names no unparsed entity in [unparsedEntities], so it is not a
//     ·declared entity name· (key-vde). Where the source reports through
//     [DeclarationsProcessed] that its DTD was not fully read (walk.declsUnread),
//     the message says so, since the name may be declared where the source did
//     not read; the verdict is the same.
//
// Which values are ·ENTITY values· is key-TYPE-value's, read through the same
// ·validating type· machinery the [ID/IDREF table] uses (roleValues, cvcid.go):
// an item whose ·governing type definition· has no ENTITY or ENTITIES in its
// closure is admitted by no candidacy and costs no member scan, an ENTITIES
// list is checked per item (key-vtype clause 2), and a union of ENTITY and
// string checks only the values its ENTITY member validated. The first value
// that fails is the verdict.
//
// GAP(validate): a ·validating type· this package cannot decide — a candidacy
// or member scan that errors, on validatingType's terms — withholds the verdict
// rather than charging, the same decline idRecord states for the [ID/IDREF
// table]. RULED permanent by #774 (STYLE P3b), on the terms of
// [walk.declaredAttribute]'s ungoverned-type decline: a candidacy or member scan
// that errors is backend or type-scan coverage, not this package's.
func (w *walk) entitiesDeclared(st *xsd.SimpleType, lexical string, owner Element, loc xsderr.Loc) (decided bool, verdict error) {
	candidate, decided := w.candidate(st, valueRole.isEntity)
	if !decided {
		return false, nil
	}
	if !candidate {
		return true, nil
	}
	values, decided := w.roleValues(st, lexical, owner)
	if !decided {
		return false, nil
	}
	for _, v := range values {
		if !v.role.isEntity() {
			continue
		}
		if w.entities == nil {
			return true, xsderr.New(ruleCvcSimpleType, loc,
				"the ·ENTITY value· %q is not a ·declared entity name·: the source presents no [unparsedEntities] property of the document information item, and Appendix D makes every ·ENTITY value· of such a source fail cvc-simple-type clause 3",
				v.value)
		}
		if w.entities.HasUnparsedEntity(v.value) {
			continue
		}
		if w.declsUnread {
			return true, xsderr.New(ruleCvcSimpleType, loc,
				"the ·ENTITY value· %q names no unparsed entity in the document's [unparsedEntities], so it is not a ·declared entity name· (key-vde), which cvc-simple-type clause 3 requires: the document's DTD was not fully read — an external DTD subset or a parameter entity went unread — and the name may be declared there, but a declaration that was not read declares nothing",
				v.value)
		}
		return true, xsderr.New(ruleCvcSimpleType, loc,
			"the ·ENTITY value· %q names no unparsed entity in the document's [unparsedEntities], so it is not a ·declared entity name· (key-vde), which cvc-simple-type clause 3 requires",
			v.value)
	}
	return true, nil
}

// notationsDeclared settles the half of String Valid clause 2 no backend can:
// NOTATION's ·value space· and ·lexical space· are "the set of QNames of
// notations declared in the current schema" (Datatypes §3.3.19), so a lexical
// the backend's QName mapping accepted is still not Datatype Valid (§4.1.4)
// against st where its QName names no notation declaration of the schema. It
// runs over a lexical value.ValidateLexical has accepted, so an enumeration
// subtype's facet (cvc-enumeration-valid, §4.3.5.4) has already been checked by
// the backend, and it covers a type derived from NOTATION by enumeration and
// xs:NOTATION itself alike. A parsed schema never gives the enumeration arm a
// verdict to reach: finalize rejects an enumeration member naming no declared
// notation under enumeration valid restriction (§4.3.5.5, value's
// declaredNotationBackend), so a value the enumeration admits names a
// declaration. The verdict is cvc-datatype-valid at loc; it is not
// enumeration-required-notation, a schema component constraint this package
// does not charge.
//
// Which values are NOTATION values is key-TYPE-value's, read through the same
// ·validating type· machinery [walk.entitiesDeclared] uses (roleValues,
// cvcid.go), so a list or union checks only the values its NOTATION-derived
// member validated. Each value is re-resolved from its token with
// [resolveInstanceQName] against owner's in-scope bindings — an unprefixed
// value takes the default namespace (§3.3.18), not the no-namespace convention
// of an unprefixed attribute NAME — and looked up among
// [xsd.Schema.Notations]. The first value in list order that names no
// declaration is the verdict.
//
// GAP(validate): a ·validating type· this package cannot decide declines, on
// [walk.entitiesDeclared]'s terms, and so does a token [resolveInstanceQName]
// turns away after value.ValidateLexical accepted it — a disagreement between
// the backend's QName mapping and this package's split, not a fact about the
// document. Each withholds the verdict rather than charging, and the caller
// records the decline. RULED permanent by #774 (STYLE P3b), on
// [walk.entitiesDeclared]'s terms.
func (w *walk) notationsDeclared(st *xsd.SimpleType, lexical string, owner Element, loc xsderr.Loc) (decided bool, verdict error) {
	candidate, decided := w.candidate(st, valueRole.isNotation)
	if !decided {
		return false, nil
	}
	if !candidate {
		return true, nil
	}
	values, decided := w.roleValues(st, lexical, owner)
	if !decided {
		return false, nil
	}
	notations := w.schema.Notations()
	for _, v := range values {
		if !v.role.isNotation() {
			continue
		}
		name, resolved := resolveInstanceQName(owner, v.value)
		if !resolved {
			return false, nil
		}
		if slices.ContainsFunc(notations, func(n xsd.Notation) bool { return n.Name() == name }) {
			continue
		}
		return true, xsderr.New(ruleCvcDatatypeValid, loc,
			"the NOTATION value %q resolves to the QName %s, which names no notation declaration of the schema, so it is outside the ·value space· of NOTATION, \"the set of QNames of notations declared in the current schema\" (Datatypes §3.3.19), and not Datatype Valid against %s",
			v.value, name, typeName(st))
	}
	return true, nil
}
