package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file is the element-side leaf of Phase E (resolve.go): Element
// Declaration Properties Correct (§3.3.6.1, e-props-correct) clause 2 — "If E has
// a ·non-absent· {value constraint}, then E.{value constraint} is a valid default
// with respect to E.{type definition} as defined in Element Default Valid
// (Immediate) (§3.3.6.2)" — and the predicate it defers to, cos-valid-default.
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
	owner := "element declaration " + e.Name().String()
	switch td := t.(type) {
	case *SimpleType:
		return s.checkSimpleDefault(ruleEPropsCorrect, e.Loc(), owner, td, vc) // clause 1, T simple
	case ComplexType:
		return s.checkComplexDefaultValid(e.Loc(), owner, td, vc)
	default:
		panic("xsd: checkElementDefaultValid: non-exhaustive TypeDefinition switch")
	}
}

// checkComplexDefaultValid decides cos-valid-default for a COMPLEX T, switching
// on the {content type} sealed sum — which is the spec's own case split, since
// each clause's antecedent is a test on {content type}.{variety} and the sum's
// three variants partition that property (SimpleContent is simple, EmptyContent
// is empty, ElementContent is mixed or element-only by its Mixed bool).
//
// The default arm asserts the sealed-sum invariant and is unreachable for any
// value an outside package can produce, since contentType is unexported (the same
// footing checkParticleValueConstraints' Term switch stands on).
func (s *Schema) checkComplexDefaultValid(loc xsderr.Loc, owner string, c ComplexType, vc ValueConstraint) error {
	switch ct := c.ContentType().(type) {
	case SimpleContent:
		// Clause 1's complex arm: the governing type is
		// T.{content type}.{simple type definition}, never T itself.
		return s.checkSimpleDefault(ruleEPropsCorrect, loc, owner, ct.SimpleType, vc)
	case ElementContent:
		if !ct.Mixed {
			return notMixedDefault(loc, owner, c, vc) // clause 2.1
		}
		if s.particleEmptiable(ct.Particle) {
			return nil // clause 2.2 holds, and 2.1 held above
		}
		return xsderr.New(ruleEPropsCorrect, loc,
			"%s has a {value constraint} of %q and a mixed complex {type definition} %s, but that type's {content type}.{particle} is not ·emptiable· (§3.9.6.3 cos-group-emptiable), which cos-valid-default clause 2.2 requires of a default on a non-simple content type (e-props-correct clause 2)",
			owner, vc.LexicalForm(), typeDefinitionLabel(c))
	case EmptyContent:
		return notMixedDefault(loc, owner, c, vc) // clause 2.1
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
func notMixedDefault(loc xsderr.Loc, owner string, c ComplexType, vc ValueConstraint) error {
	return xsderr.New(ruleEPropsCorrect, loc,
		"%s has a {value constraint} of %q, but its {type definition} %s is a complex type whose {content type}.{variety} is %s, and cos-valid-default clause 2.1 admits a default only on mixed content (e-props-correct clause 2)",
		owner, vc.LexicalForm(), typeDefinitionLabel(c), c.ContentType().Variety())
}
