package validate

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCvcSimpleType is String Valid (Structures §3.16.4, cvc-simple-type). This
// package charges it only as the verdict a delegating rule wraps — cvc-attribute
// clause 3, cvc-type clause 3.1.3 and cvc-complex-type clauses 1.2 and 4 — and
// only for clause 3: clauses 1 and 2 are value.ValidateLexical's, whose verdict
// carries Datatype Valid's own rule.
const ruleCvcSimpleType xsderr.Rule = "cvc-simple-type"

// stringValid runs String Valid (§3.16.4) over lexical against st, for the
// item at loc whose element is owner: clauses 1 and 2 through
// value.ValidateLexical, then clause 3 ([walk.entitiesDeclared]) over the
// ·actual value· those two accepted. It reports decided false where this
// package withholds a verdict, and otherwise a nil verdict for a ·valid·
// lexical and the rejection for an invalid one, which the caller charges under
// its own rule with the verdict as the wrapped cause.
//
// A ValidateLexical error that is not a VERDICT ([value.IsDatatypeVerdict])
// withholds one; each caller states that decline's GAP on its own terms.
func (w *walk) stringValid(st *xsd.SimpleType, lexical string, owner Element, loc xsderr.Loc) (decided bool, verdict error) {
	_, err := value.ValidateLexical(w.backend, w.schema, st, lexical, elementContext{owner: owner})
	if err == nil {
		return w.entitiesDeclared(st, lexical, owner, loc)
	}
	if !value.IsDatatypeVerdict(err) {
		return false, nil
	}
	return true, err
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
//     ·declared entity name· (key-vde).
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
// [walk.matchedAttribute]'s ungoverned-type decline: a candidacy or member scan
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
		if !w.entities.HasUnparsedEntity(v.value) {
			return true, xsderr.New(ruleCvcSimpleType, loc,
				"the ·ENTITY value· %q names no unparsed entity in the document's [unparsedEntities], so it is not a ·declared entity name· (key-vde), which cvc-simple-type clause 3 requires",
				v.value)
		}
	}
	return true, nil
}
