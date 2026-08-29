package xsd

import (
	"strings"

	"github.com/kud360/goxsd8/xsderr"
)

// ruleCosValidDefault is Element Default Valid (Immediate) (Structures §3.3.6.2,
// cos-valid-default) charged AS ITSELF, which only [Schema.ElementDefaultValid]
// does: every in-package caller charges it as a clause of its own rule instead
// and cites that one.
const ruleCosValidDefault xsderr.Rule = "cos-valid-default"

// This file is the element-side leaf of Phase E (resolve.go): Element
// Declaration Properties Correct (§3.3.6.1, e-props-correct) clause 2 — "If E has
// a ·non-absent· {value constraint}, then E.{value constraint} is a valid default
// with respect to E.{type definition} as defined in Element Default Valid
// (Immediate) (§3.3.6.2)" — and the predicate it defers to, cos-valid-default.
//
// The predicate is charged from outside Phase E too, and so is exported
// ([Schema.ElementDefaultValid]): cvc-elt clause 5.1.1 asks the same question of
// an empty element's ·governing type definition· at assessment time, over the
// same (T, {value constraint}) pair shape and a different T. Everything below is
// generic over that pair and over the citation, which is the CALLER's rule and
// clause for the in-package callers that charge cos-valid-default as a clause of
// their own, and cos-valid-default itself across the package boundary — the
// predicate cites the predicate, and the caller cites itself (defaultCharge).
//
// cos-valid-default is a CASE ANALYSIS, not a conjunction, and reading it as one
// would be the easy mistake: its two clauses are guarded by mutually exclusive
// antecedents on T.{content type}.{variety}, and exactly one arm runs for any T.
//
//   - clause 1 — T is simple, or complex with {content type}.{variety} = simple.
//     V must be a valid default with respect to T itself (if simple) or to
//     T.{content type}.{simple type definition} (if complex), as defined by Simple
//     Default Valid (§3.2.6.2). That predicate already exists — checkSimpleDefault
//     (valueconstraintvalid.go, #371) — and this is its THIRD caller, not a third
//     copy (STYLE T4). All this arm contributes is picking which of the two simple
//     types the spec names.
//   - clause 2 — T is complex with {content type}.{variety} ≠ simple. Then BOTH
//     2.1 ({content type}.{variety} = mixed) and 2.2 (the particle
//     T.{content type}.{particle} is ·emptiable· per §3.9.6.3) must hold. 2.2 is
//     particleEmptiable (effectivetotalrange.go), likewise reused rather than
//     re-derived.
//
// Clause 2.1 is where this landing adds a REJECTION SHAPE nothing else in the
// package produced: a complex type whose {content type} is element-only or empty
// admits no default at all, whatever its particle allows. The two are one branch
// here because they fail the one requirement 2.1 states — "{variety} = mixed" —
// and differ only in which non-mixed token they carry, which the message reports.
//
// The rule is charged uniformly over GLOBAL and LOCAL element declarations. §3.3.6
// heads the constraint "All element declarations ... must satisfy the following
// constraint" and the rule opens "For any element declaration E", with no scope
// or nesting qualification anywhere below it; clause 3's explicit test on
// {scope}.{variety} shows the rule text says so when it means to. The walk that
// delivers on that quantifier — and the measurement that decided its shape — is
// documented at the head of valueconstraintvalid.go.

// checkElementDefaultValid charges e-props-correct (§3.3.6.1) clause 2 against one
// element declaration, deciding Element Default Valid (Immediate) (§3.3.6.2,
// cos-valid-default) over the declaration's {value constraint} and its RESOLVED
// {type definition}.
//
// The declaration is charged at its OWN Loc, which an ElementDeclaration retains
// (doc.go), so a reader is sent to the <xs:element> that wrote the default rather
// than to whichever enclosing component the walk happened to arrive from.
//
// Two gates accept rather than reject, matching the arms of its attribute-side
// counterpart (checkAttributeDeclarationValueConstraint) exactly:
//
//   - no {value constraint}: the clause's antecedent ("If E has a ·non-absent·
//     {value constraint}") fails, so cos-valid-default is not reached at all —
//     never reached-and-vacuously-satisfied, which is why the value space must not
//     be consulted either;
//   - a {type definition} that is absent or unresolvable: ResolvedType's documented "not
//     decidable by this clause". A dangling name was already charged src-resolve
//     by Phase A, so reaching here with one means a genuinely absent slot, and
//     cos-valid-default predicates over a T that must be there to be read.
//
// Both are fail-open and neither can mask a failure the spec states.
func (s *Schema) checkElementDefaultValid(e ElementDeclaration) error {
	vc, present := e.ValueConstraint()
	if !present {
		return nil
	}
	t, ok := s.ResolvedType(e.TypeDefinition())
	if !ok {
		return nil
	}
	return s.elementDefaultValid(s.valueSpace, ruleEPropsCorrect, "2", e.Loc(),
		"element declaration "+e.Name().String(), t, vc)
}

// ElementDefaultValid decides Element Default Valid (Immediate) (§3.3.6.2,
// cos-valid-default) over one type definition and one {value constraint}: nil
// where vc is a valid default with respect to t, and the rejection — charged
// under cos-valid-default, the predicate's OWN rule — otherwise.
//
// It is exported for the ONE caller that charges cos-valid-default over a type
// definition no ElementDeclaration supplies: cvc-elt (§3.3.4.3) clause 5.1.1,
// which asks it at ASSESSMENT time of an empty element's ·governing type
// definition· where that type is an ·instance-specified type definition·
// (§3.3.4.1, key-itd — the xsi:type-driven case alone). The predicate is the same
// one Phase E charges and is deliberately not re-derived there (STYLE T4); what
// differs is only the (t, vc) pair and the location.
//
// The rule the CALLER charges is its own and does not cross this seam: a
// cvc-elt clause 5.1.1 charge names itself in its own message and carries this
// verdict as the cause, which is the contract [ValueSpace.ValidDefault] already
// publishes one layer down — it hands back a cvc-datatype-valid verdict and
// leaves a-props-correct to say so.
//
// vs is the value space clause 1's datatype question is decided in, and is a
// PARAMETER rather than this schema's own installed one because the two are not
// the same space at assessment time: a schema assembled through
// [SchemaBuilder.Finalize] carries undecidedValueSpace{}, which answers every
// question undecided, so reading s.valueSpace would leave cvc-elt clause 5.1.1
// permanently undecidable for such a schema, while the walk charging that clause
// holds the space its own backend defines. It is the same seam validate's
// cvc-complex-type clause 4 charge reads for the same reason. A nil vs panics on
// [SchemaBuilder.FinalizeWith]'s grounds, this being the one door that can carry
// one in.
//
// loc and owner locate and name the item charged: the element information item
// and its ·expanded name·.
//
// t is the RESOLVED type definition, never a [TypeDefinitionRef]: cos-valid-default
// predicates over a T that must be there to be read, and a caller holding a slot
// resolves it first (checkElementDefaultValid) or declines.
func (s *Schema) ElementDefaultValid(vs ValueSpace, loc xsderr.Loc, owner string, t TypeDefinition, vc ValueConstraint) error {
	if vs == nil {
		panic("xsd: Schema.ElementDefaultValid: nil ValueSpace")
	}
	return s.elementDefaultValid(vs, ruleCosValidDefault, "", loc, owner, t, vc)
}

// elementDefaultValid is cos-valid-default's case analysis, over the citation its
// caller charges it under: an in-package rule and clause, or the predicate's own
// rule with no clause of another's to name (defaultCharge).
func (s *Schema) elementDefaultValid(vs ValueSpace, rule xsderr.Rule, clause string, loc xsderr.Loc, owner string, t TypeDefinition, vc ValueConstraint) error {
	switch td := t.(type) {
	case *SimpleType:
		return s.checkSimpleDefault(vs, rule, clause, loc, owner, td, vc) // clause 1, T simple
	case ComplexType:
		return s.checkComplexDefaultValid(vs, rule, clause, loc, owner, td, vc)
	default:
		panic("xsd: elementDefaultValid: non-exhaustive TypeDefinition switch")
	}
}

// defaultCharge closes a cos-valid-default rejection with the citation it is
// CHARGED under: the caller's own rule and clause where it charges the predicate
// as a clause of one of its rules, and nothing at all where it charges
// cos-valid-default as itself — that caller's rule is the message's own, which
// [xsderr.Error] already renders, and every message here names the
// cos-valid-default or cos-valid-simple-default clause it decided on inline
// (STYLE E4).
//
// also carries whatever else the message cites in the same parenthetical, so one
// encoding serves the three rejection sites rather than each spelling its own
// bracket (STYLE T4).
func defaultCharge(rule xsderr.Rule, clause string, also ...string) string {
	cites := also
	if clause != "" {
		cites = append([]string{string(rule) + " clause " + clause}, also...)
	}
	if len(cites) == 0 {
		return ""
	}
	return " (" + strings.Join(cites, ", ") + ")"
}

// checkComplexDefaultValid decides cos-valid-default for a COMPLEX T, switching
// on the {content type} sealed sum — which is the spec's own case split, since
// each clause's antecedent is a test on {content type}.{variety} and the sum's
// three variants partition that property (SimpleContent is simple, EmptyContent
// is empty, ElementContent is mixed or element-only by its Mixed bool).
//
// The default arm asserts the sealed-sum invariant and is unreachable for any
// value an outside package can produce, since contentType is unexported (the same
// footing componentWalk.walkParticle's Term switch stands on).
func (s *Schema) checkComplexDefaultValid(vs ValueSpace, rule xsderr.Rule, clause string, loc xsderr.Loc, owner string, c ComplexType, vc ValueConstraint) error {
	switch ct := c.ContentType().(type) {
	case SimpleContent:
		// Clause 1's complex arm: the governing type is
		// T.{content type}.{simple type definition}, never T itself.
		return s.checkSimpleDefault(vs, rule, clause, loc, owner, ct.SimpleType, vc)
	case ElementContent:
		if !ct.Mixed {
			return notMixedDefault(rule, clause, loc, owner, c, vc) // clause 2.1
		}
		if s.particleEmptiable(ct.Particle) {
			return nil // clause 2.2 holds, and 2.1 held above
		}
		return xsderr.New(rule, loc,
			"%s has a {value constraint} of %q and a mixed complex {type definition} %s, but that type's {content type}.{particle} is not ·emptiable· (§3.9.6.3 cos-group-emptiable), which cos-valid-default clause 2.2 requires of a default on a non-simple content type%s",
			owner, vc.LexicalForm(), typeDefinitionLabel(c), defaultCharge(rule, clause))
	case EmptyContent:
		return notMixedDefault(rule, clause, loc, owner, c, vc) // clause 2.1
	default:
		panic("xsd: checkComplexDefaultValid: non-exhaustive ContentType switch")
	}
}

// notMixedDefault is the shared cos-valid-default clause 2.1 rejection (STYLE T4):
// a complex {type definition} whose {content type}.{variety} is neither simple
// (clause 1's arm) nor mixed admits no {value constraint} at all. It serves both
// non-mixed varieties — empty and element-only — because clause 2.1 states ONE
// requirement over both, and reports which one the type carries by reading the
// {variety} token back off c rather than spelling either literal. It takes the
// complex type, not the content type its caller already has in hand: the token is
// derivable from c, and taking both would let a caller pass a variety that is not
// c's (STYLE D3).
func notMixedDefault(rule xsderr.Rule, clause string, loc xsderr.Loc, owner string, c ComplexType, vc ValueConstraint) error {
	return xsderr.New(rule, loc,
		"%s has a {value constraint} of %q, but its {type definition} %s is a complex type whose {content type}.{variety} is %s, and cos-valid-default clause 2.1 admits a default only on mixed content%s",
		owner, vc.LexicalForm(), typeDefinitionLabel(c), c.ContentType().Variety(), defaultCharge(rule, clause))
}
