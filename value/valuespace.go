package value

import "github.com/kud360/goxsd8/xsd"

// This file is the adapter that lets package xsd — a pure leaf that cannot
// import this one (PRINCIPLES 1) — answer the Structures constraints that reach
// into a value space. Two of them COMPARE the {value}s of two Value Constraints:
//
//   - au-props-correct (§3.5.6) clause 3: "U.{value constraint}.{value} is
//     IDENTICAL to U.{attribute declaration}.{value constraint}.{value}" —
//     Datatypes §2.2.1's identity relation.
//   - loc-testSubP (§3.4.6.4) clauses 4.2 and 5.2.2: "an EQUAL OR IDENTICAL
//     value" — the §2.2.1/§2.2.2 union, which §2.2.2 makes the default reading
//     of every sameness test the specification prescribes.
//
// Both are VALUE-space tests, not lexical ones: {value} is an ·actual value·
// (key-vv §3.2.1), so "1" and "01" are the same xs:integer {value} under two
// {lexical form}s, and xsd.ValueConstraint carries only the latter.
//
// The third, cos-valid-simple-default (§3.2.6.2) — charged by a-props-correct
// (§3.2.6.1) clause 2 and au-props-correct clause 2 — is one-sided: it asks
// whether ONE {lexical form} denotes a value of ONE type at all, which is
// Datatype Valid (§4.1.4) and so [ValidateLexical]'s whole job. It needs no
// shared mapping across two types, so it decides the list and union varieties
// the comparisons refuse; what it must NOT do is turn the pipeline's non-verdict
// errors into rejections (ValidDefault says which, and why).

// NewValueSpace returns the [xsd.ValueSpace] backed by b: it maps a Value
// Constraint's {lexical form} through the governing mapping of the type that
// constrains it, compares the resulting values with the capability relations the
// values themselves carry ([Identical], [Eq]), and validates one against its type
// through the full facet pipeline ([ValidateLexical]).
//
// It honors the fail-open contract [xsd.ValueSpace] states: every question it
// cannot answer reports decided=false rather than a verdict, so an unsupported
// type, an unmappable lexical, a cross-type comparison, or an error that belongs
// to the type rather than to the value constraint can never turn into a schema
// rejection.
//
// It panics if b is nil, matching parser.WithBackend's guard: a nil backend is a
// caller bug, not a schema-validity condition.
func NewValueSpace(b Backend) xsd.ValueSpace {
	if b == nil {
		panic("value: NewValueSpace: nil Backend")
	}
	return valueSpace{b: b}
}

// valueSpace is NewValueSpace's implementation. It holds only the backend: the
// governing mapping and the whiteSpace mode in force are recomputed per
// comparison from the types handed in, never cached (STYLE D3 — a value
// constraint is compared at most a handful of times per finalize, so there is no
// measured hot path to cache for).
type valueSpace struct{ b Backend }

// Identical is Datatypes §2.2.1's identity relation, which au-props-correct
// clause 3 compares two {value}s under.
func (vs valueSpace) Identical(r xsd.TypeResolver, ta *xsd.SimpleType, a xsd.ValueConstraint, tb *xsd.SimpleType, b xsd.ValueConstraint) (bool, bool) {
	av, bv, ok := vs.values(r, ta, a, tb, b)
	if !ok {
		return false, false
	}
	return identical(av, bv)
}

// EqualOrIdentical is the §2.2.1/§2.2.2 equal-or-identical union, which
// loc-testSubP clauses 4.2 and 5.2.2 compare two {value}s under.
func (vs valueSpace) EqualOrIdentical(r xsd.TypeResolver, ta *xsd.SimpleType, a xsd.ValueConstraint, tb *xsd.SimpleType, b xsd.ValueConstraint) (bool, bool) {
	av, bv, ok := vs.values(r, ta, a, tb, b)
	if !ok {
		return false, false
	}
	return equalOrIdentical(av, bv)
}

// ValidDefault is Simple Default Valid (§3.2.6.2, cos-valid-simple-default),
// which a-props-correct clause 2 and au-props-correct clause 2 both charge: is
// vc.{lexical form} Datatype Valid (§4.1.4) with respect to t? [ValidateLexical]
// decides exactly that rule — all three of its clauses, over all three varieties
// — so the whole of this method is the four gates that separate the errors it
// returns which ARE that verdict from the ones that are not. Every gate answers
// undecided, never invalid, per the [xsd.ValueSpace] fail-open contract.
//
// The first three gates run BEFORE the pipeline, in this order; the fourth reads
// the pipeline's own error, because the fault it catches is only detectable once a
// facet meets a value:
//
//  1. GAP(value): needsContext. A QName- or NOTATION-governed value space
//     anywhere in t's variety closure needs the namespace bindings in scope at
//     the literal (§3.3.18/§3.3.19, PRINCIPLES 19). vc now CARRIES those
//     bindings, and the comparisons resolve them (sharedMapping), so what is
//     still unclaimed here is the routing: this method does not refuse the list
//     and union varieties, and threading a [Context] to the literal through
//     validateUnion and listMapping — which re-derive their own dispatch — is a
//     rejection surface of its own (a list OF QName, a union WITH a QName
//     member), with its own grounding and its own ratchet attribution. Until
//     that lands the whole closure stays undecided, because a nil [Context]
//     makes a backend reject EVERY QName lexical and every QName default would
//     be a false reject.
//  2. GAP(value): governingMapping. An ungoverned type — no backend Mapping on
//     it or any ancestor, an ungoverned item type, an ungoverned union member —
//     makes validateLexical return a cvc-datatype-valid "no backend mapping
//     governs type" error that is a BACKEND gap, not instance data. It is also
//     how xs:anySimpleType and xs:anyAtomicType arrive here: no backend maps the
//     ·special· datatypes (Datatypes §4.1), for which Datatype Valid is
//     unconditionally TRUE, and §3.2.2.2's third tier types every attribute
//     declaration with no @type as xs:anySimpleType — so reading that error as a
//     verdict would reject every typeless attribute default in existence. The
//     union branch of this gate (unionGoverned) is the same test validateUnion
//     applies before dispatching, so this pre-check and the pipeline agree on
//     exactly which unions are governed.
//  3. GAP(value): compile. A construction-stage failure in t's OWN facets — a
//     pattern regex.Translate cannot express (src-pattern-value), an enumeration
//     or bound facet whose DECLARING type the backend does not map
//     (src-enumeration-value) — says nothing about vc.{lexical form}. Charging
//     it as clause 2 would reject a schema for an unrelated facet, under the
//     wrong rule ID and against the wrong component.
//  4. [IsFacetPrecondition]. A facet paired with a value lacking the capability it
//     needs (cos-applicable-facets §4.1.5), or a type with no usable whiteSpace mode
//     where §3.16.7.4 guarantees one, is a fault in T ITSELF — the first half
//     discharged wherever an xsd.SimpleTypeRestrictionChecker is installed, as the
//     parser installs one, and the caller's own where none is. Unlike
//     gates 1–3 it cannot be pre-checked here: the pairing is only observable once a
//     facet meets a parsed value inside the pipeline. Reading it as clause 2 would
//     reject the schema for a fault of the component rather than of the value
//     constraint — under a rule that has nothing to say about it, and, since
//     [ValidateLexical] would report the same fault for EVERY literal, for a default
//     that no lexical could have satisfied.
//
// One residue is recorded rather than papered over. GAP(value): item/member
// facet compilation. The compile gate covers T's OWN effective facets only: a
// list's ITEM type and a union's MEMBER types compile inside the dispatch
// (listMapping's Parse recurses through validateLexical; dispatchUnion folds
// every member's rejection into one cvc-datatype-valid error), so a
// construction-stage failure down there still reaches the caller as a decided
// reject. Closing that needs the pipeline itself to separate its construction
// and verdict stages per member, which is a change to package value's own error
// model, not to this adapter. Gate 4 is NOT part of that residue: a precondition
// fault propagates out of the item/member dispatch unchanged (list.go, union.go),
// so it is caught here wherever in T's closure it arose.
//
// nil is passed as the [Context] because gate 1 has already excluded every
// context-dependent literal — unlike values, which parses each side under the
// context its own value constraint captured.
//
// An UNRESOLVABLE {base type definition} in t's own chain — the src-resolve
// error r's readers produce — is undecided, not a verdict: gates 1-3 each walk
// that chain and each answers undecided on the error, which is the right
// polarity, since the fault says t could not be READ and says nothing about
// vc.{lexical form}. The residue is a break deeper inside a list's item or a
// union's member closure than gates 1-3 reach, which would surface from
// [ValidateLexical] and be read as decided-invalid. It is unreachable for the
// caller this interface exists for: an xsd.Schema charges src-resolve for
// every such break in Phase A and again in Phase D's first step, both of which
// run before Phase E asks this question at all.
func (vs valueSpace) ValidDefault(r xsd.TypeResolver, t *xsd.SimpleType, vc xsd.ValueConstraint) (bool, bool) {
	needs, err := needsContext(r, t)
	if err != nil || needs {
		return false, false
	}
	if _, ok, err := governingMapping(vs.b, r, t); err != nil || !ok {
		return false, false
	}
	if _, _, err := compile(vs.b, r, t); err != nil {
		return false, false
	}
	if _, err := ValidateLexical(vs.b, r, t, vc.LexicalForm(), nil); err != nil {
		if IsFacetPrecondition(err) {
			return false, false // gate 4
		}
		return false, true
	}
	return true, true
}

// values maps both {lexical form}s to ·actual values· IN ONE VALUE SPACE, or
// reports ok=false when it cannot — the single fail-open gate both methods pass
// through (STYLE T4).
//
// The one space is the shared governing mapping of ta and tb (sharedMapping).
// Each side is whiteSpace-normalized under ITS OWN type's effective mode before
// being parsed, because normalization is the first stage of that type's lexical
// mapping (key-nv §3.1.4, cvc-simple-type §3.16.4) and a restriction may narrow
// the mode its base declared. A type with no mode in force (xs:anySimpleType, a
// union) yields the zero mode, which normalizeWhiteSpace refuses, so it is
// reported undecided instead.
//
// Each side is also parsed under ITS OWN [Context], built from the namespace
// bindings that side's value constraint captured where its {lexical form} was
// written (§3.3.18/§3.3.19, PRINCIPLES 19). The two literals may come from
// schema documents binding the same prefix differently, or different prefixes
// to one namespace, so a single shared context would decide the comparison
// wrongly in both directions.
//
// A Parse failure is undecided, never a verdict: a {lexical form} outside the
// type's lexical space is au-props-correct clause 2 / cos-valid-simple-default's
// business (§3.2.6.2), a separate obligation this adapter must not pre-empt by
// reporting "not the same value" for what is really "not a value at all".
//
// GAP(value): a QName or NOTATION prefix with no binding in the context its own
// constraint captured takes that same path and stays undecided. It needs no
// branch and no tri-state of its own — the mapping's Parse rejects the literal,
// and this function reports the rejection undecided. Nor is it a spec category:
// such a literal already fails Datatype Valid, so cos-valid-simple-default
// clause 1 rejects the schema before either comparison is asked anything. What
// it must not become is a NOT-same verdict, which every consumer of these two
// methods charges on — xsd's checkAttributeUseValueConstraint
// (valueconstraintvalid.go, au-props-correct clause 3),
// fixedValueConstraintSubsumes and attributeValueConstraintsAgree
// (defaultbinding.go, loc-testSubP clauses 4.2 and 5.2.2) each reject on
// decided-and-not-same and accept on undecided — so a wrong NOT-same here is a
// false schema rejection at all three.
func (vs valueSpace) values(r xsd.TypeResolver, ta *xsd.SimpleType, a xsd.ValueConstraint, tb *xsd.SimpleType, b xsd.ValueConstraint) (Value, Value, bool) {
	m, ok := vs.sharedMapping(r, ta, tb)
	if !ok {
		return nil, nil, false
	}
	aws, aerr := whiteSpaceInForce(r, ta)
	bws, berr := whiteSpaceInForce(r, tb)
	if aerr != nil || berr != nil {
		return nil, nil, false // an unresolvable base: undecided, never a verdict
	}
	if aws == 0 || bws == 0 {
		return nil, nil, false
	}
	av, err := m.Parse(normalizeWhiteSpace(a.LexicalForm(), aws), constraintContext(a))
	if err != nil {
		return nil, nil, false
	}
	bv, err := m.Parse(normalizeWhiteSpace(b.LexicalForm(), bws), constraintContext(b))
	if err != nil {
		return nil, nil, false
	}
	return av, bv, true
}

// constraintContext is the [Context] a value constraint's {lexical form} is
// parsed under: the namespace bindings captured at the schema-document element
// that wrote it (§3.3.18, fixed there by cos-valid-simple-default clause 2), on
// the ONE nsContext this package resolves prefixes with (facets.go).
func constraintContext(vc xsd.ValueConstraint) nsContext {
	ns, ok := vc.DefaultNamespace()
	return newNSContext(vc.NamespaceBindings(), ns, ok)
}

// sharedMapping resolves the ONE Mapping that governs both ta's and tb's values,
// or reports ok=false when no single mapping does. It is the correctness-critical
// half of this adapter, because the two constraints it serves compare values
// across DIFFERENT types: loc-testSubP clause 5.1 requires only that S's type be
// ·derived· from G's, so a general xs:decimal and a specific xs:integer routinely
// meet here.
//
// The widest-space rule (doc.go, backend.go) is what makes such a pair
// comparable at all: a derived type without its own mapping is governed by its
// nearest mapped ancestor's, so xs:integer and xs:byte both resolve to whatever
// governs them upward, and Datatypes §2.2.1's own note settles the semantics —
// "'+2', treated as a decimal, '+2', treated as an integer, and '+2', treated as
// a byte, all denote the same value. They are not only equal but identical."
//
// When the two sides resolve to DIFFERENT governing types the comparison is
// refused. That is not conservatism, it is §2.2.1/§2.2.2's own rule: "values
// from different ·primitive· datatypes' ·value spaces· are made artificially
// distinct", and a backend's derived mapping may in addition represent values in
// a narrower space of its own, where a value of the base space has no image.
// Parsing each side under its own mapping and comparing the results across the
// two would be exactly the corruption the widest-space rule forbids, and could
// report a spurious NOT-same — the one outcome the fail-open contract rules out.
//
// The list and union varieties are refused outright, on ta or tb: their governing
// mappings are synthesized per type (listMapping over the {item type definition},
// unionMapping over the {member type definitions}), so "the same mapping"
// is not a property two distinct nodes can be checked for by identity here, and
// no comparison is better than a guessed one.
//
// The two context-dependent primitives, QName and NOTATION, are NOT refused:
// each side's value constraint carries the namespace bindings in scope AT ITS
// LITERAL (§3.3.18/§3.3.19, PRINCIPLES 19), which values resolves each side's
// prefix under, so "a:x" and "b:x" compare SAME when both prefixes name one
// namespace and "p:x" compares NOT-same across documents that bind "p"
// differently — the two verdicts a lexical comparison gets wrong in opposite
// directions. The prefix that resolves to nothing is values' residual, marked
// there.
//
// An unresolvable {base type definition} on either side is refused: the
// widest-space walk cannot finish, so there is no shared mapping to name, and
// refusing is the fail-open answer this whole adapter owes.
func (vs valueSpace) sharedMapping(r xsd.TypeResolver, ta, tb *xsd.SimpleType) (Mapping, bool) {
	ga, ok, err := governingType(vs.b, r, ta)
	if err != nil || !ok {
		return Mapping{}, false
	}
	gb, ok, err := governingType(vs.b, r, tb)
	if err != nil || !ok {
		return Mapping{}, false
	}
	if ga != gb {
		return Mapping{}, false // incommensurable: two different value spaces
	}
	return vs.b.Mapping(ga.Name())
}

// governingType is the type node whose Mapping governs node's value space: node
// itself, or — under the widest-space rule, walked by the shared governingNode
// (facets.go) that governingMapping also reads — its nearest mapped ancestor.
// The NODE is returned rather than the Mapping because a Mapping is a struct of
// funcs and so cannot be compared; identity of the governing node is what tells
// two value spaces apart, and pointer identity is the right test because
// xsd.SimpleType's whole contract is that one node per type is shared across a
// compiled schema (simpletype.go).
//
// Only the atomic {variety} resolves: see sharedMapping for why list and union
// are refused.
func governingType(b Backend, r xsd.TypeResolver, node *xsd.SimpleType) (*xsd.SimpleType, bool, error) {
	variety, err := node.Variety(r)
	if err != nil {
		return nil, false, err
	}
	if _, ok := variety.(xsd.Atomic); !ok {
		return nil, false, nil
	}
	return governingNode(b, r, node)
}

// contextDependent reports whether t's {primitive type definition} is QName or
// NOTATION — the two primitives whose lexical mapping needs a [Context] (§3.3.18,
// §3.3.19). An absent {primitive type definition} (xs:anyAtomicType alone,
// §3.16.1) is not one of them.
func contextDependent(r xsd.TypeResolver, t *xsd.SimpleType) (bool, error) {
	variety, err := t.Variety(r)
	if err != nil {
		return false, err
	}
	if _, ok := variety.(xsd.Atomic); !ok {
		return false, nil
	}
	p, err := t.Primitive(r)
	if err != nil || p == nil {
		return false, err
	}
	return p.Name() == qnameName || p.Name() == notationName, nil
}

// needsContext reports whether t's value space is governed, anywhere in its
// {item type definition}/{member type definitions} closure, by QName or NOTATION
// — the recursive form of contextDependent (above).
//
// The closure form is needed because ValidDefault, unlike Identical and
// EqualOrIdentical, does NOT refuse the list and union varieties: it needs one
// type's mapping rather than a shared one across two, so ValidateLexical's own
// list/union dispatch (facets.go, union.go) decides them, and a list OF QName or
// a union WITH a QName member would otherwise reach a backend with the nil
// Context ValidDefault passes and be rejected for want of bindings it never
// threaded — the value constraint carries them, but the dispatch does not route
// them (gate 1). governingMapping recurses the same closure for the same
// reason (facets.go). For a union, ANY context-dependent member makes the whole
// union undecided: dispatch takes the first member that accepts, so a member
// that could only be decided WITH context could change the verdict.
//
// The nil guards on Item and each member are belt-and-braces over a state no
// component can be in, NOT a reachable one: the ListDerivation.Item and
// UnionDerivation.Members slots are xsd.SimpleTypeOrRefs that admit no encoding
// of absence, rejected by xsd.NewSimpleType at construction, and an unresolvable
// by-name one is an error out of Item/Members rather than a nil. They stay
// because everything below them is nil-hostile too — the recursive Variety()
// call they gate, and governingMapping on the very next line — so removing them
// would trade a documented impossibility for a crash rather than for a verdict.
//
// No visited set is needed (PRINCIPLES 9), on two arguments that are no longer
// one. The ITEM edge cannot close a loop that reaches here: any cycle of
// itemType= references contains a list whose item is a list, which
// xsd.SimpleType.CheckDerivation rejects under cos-st-restricts clause 2.1
// before a schema finalizes. The MEMBER edge rests on nothing constructing a
// by-name union-membership cycle, which holds only because no producer emits a
// union at all — see xsd/derivation.go's CheckDerivation clause 3.3 paragraph
// and the guard #738 replaces it with.
func needsContext(r xsd.TypeResolver, t *xsd.SimpleType) (bool, error) {
	variety, err := t.Variety(r)
	if err != nil {
		return false, err
	}
	switch variety.(type) {
	case xsd.Atomic:
		return contextDependent(r, t)
	case xsd.List:
		item, err := t.Item(r)
		if err != nil || item == nil {
			return false, err
		}
		return needsContext(r, item)
	case xsd.Union:
		members, err := t.Members(r)
		if err != nil {
			return false, err
		}
		for _, m := range members {
			if m == nil {
				continue
			}
			needs, err := needsContext(r, m)
			if err != nil || needs {
				return needs, err
			}
		}
	}
	return false, nil
}

// identical reports Datatypes §2.2.1's identity relation over two values, with
// decided=false when neither relation is available on a.
//
// A value that implements [Identical] answers directly: that capability exists
// precisely for the value types whose identity DIFFERS from their equality
// (float/double, where +0 = −0 but the two are not identical and NaN is
// identical to itself while equal to nothing; dateTime and duration, equal but
// not identical across remembered timezone offsets). For every other value
// §2.2.2 settles it — "the equality relation for most datatypes IS the identity
// relation" — so order equality answers.
//
// That fallback is sound for STRICT identity only because of the obligation
// [Identical]'s own doc states: a value type whose identity differs from its
// equality MUST implement [Identical], so an [Eq]-only value has declared the
// two relations to be one. It is NOT sound merely by analogy with enumeration
// matching, which tests the equal-or-identical UNION (equalOrIdentical) and so
// can absorb an Eq answer in the union direction regardless.
//
// This is deliberately NOT equalOrIdentical: for a value that DOES distinguish
// the two, au-props-correct clause 3 says "identical", so an equal-but-not-
// identical pair must answer (false, true) here and (true, true) there.
func identical(a, b Value) (same, decided bool) {
	if id, ok := a.(Identical); ok {
		return id.Identical(b), true
	}
	if eq, ok := a.(Eq); ok {
		return eq.Eq(b), true
	}
	return false, false
}

// equalOrIdentical reports the §2.2.1/§2.2.2 "equal or identical" union — "all
// comparisons for 'sameness' prescribed by this specification test for either
// equality or identity, not for identity alone" (§2.2.2) — with decided=false
// when a carries NEITHER relation, so a caller that must distinguish "not the
// same" from "not comparable" can.
//
// It is the ONE encoding of that union in this package (STYLE T4): enumeration
// matching (cvc-enumeration-valid §4.3.5.4, enumMatch) and the Structures
// value-constraint clauses (loc-testSubP 4.2/5.2.2, via
// valueSpace.EqualOrIdentical) both read it, and neither re-derives it.
func equalOrIdentical(a, b Value) (same, decided bool) {
	id, hasIdentical := a.(Identical)
	if hasIdentical && id.Identical(b) {
		return true, true
	}
	eq, hasEq := a.(Eq)
	if hasEq && eq.Eq(b) {
		return true, true
	}
	return false, hasIdentical || hasEq
}
