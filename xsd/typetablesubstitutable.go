package xsd

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// This file is the fifth step of Phase D of the finalize resolution pass
// (resolve.go): Element Declaration Properties Correct (§3.3.6.1,
// e-props-correct) clause 7 — "If E.{type table} exists, then for each {type
// definition} T in E.{type table}.{alternatives}, and also for E.{type
// table}.{default type definition}.{type definition}, one of the following is
// true 7.1 T is ·validly substitutable· for E.{type definition}, subject to the
// blocking keywords of E.{disallowed substitutions}. 7.2 T is the type
// ·xs:error·."
//
// It shares Phase D with clause 4 (substitutiongrouptypes.go) because it is the
// same cos-ct-derived-ok / cos-st-derived-ok engine pair under a third
// quantifier — Type Alternatives rather than substitution group affiliations or
// complex types.
//
// IT CANNOT RUN IN PHASE A, where resolveTypeTable resolves the same
// alternatives' QNames (src-resolve clause 1.1). ValidlySubstitutable walks
// {base type definition} chains and a union's transitive membership with NO
// visited set, which resolve.go's invariant licenses only for a phase running
// after Phase B's checkComplexBaseAcyclic, checkSimpleBaseAcyclic and
// checkUnionMembershipAcyclic. A circular base chain is fully representable
// while Phase A runs, so charging clause 7 there would not walk a cycle once and
// answer wrongly — it would not terminate.
//
// THE UNION-BASE READING OF cos-st-derived-ok IS SETTLED, and it is the shape
// conditional type assignment normally takes: an alternative naming a MEMBER of
// the declaration's own union {type definition}. key-val-sub-type (§3.4.6.5)
// routes a SIMPLE candidate T to cos-st-derived-ok (§3.16.6.3) whatever the
// target E.{type definition} is, and that constraint's clause 2.2.4 decomposes a
// union TARGET member-wise rather than treating it as opaque: 2.2.4.1 requires
// B.{variety} = union, 2.2.4.2 requires T to be validly derived from some member
// M of B's TRANSITIVE membership by this same constraint, and 2.2.4.3 requires
// the {facets} of B and of any intervening union to be empty. §3.16.6.3's Note
// states the consequence outright — the relation "can hold between a Simple Type
// Definition in the transitive membership of a union type, and the union type,
// even though neither is actually ·derived· from the other". derivedOKSimple
// (derivation.go) already implements all three sub-clauses, so a member-naming
// alternative passes clause 7.1 rather than being rejected by it.
//
// 2.2.4.1 tests the TARGET, never the candidate. An alternative whose own type
// is a union, against a non-union E.{type definition}, is decided by the
// ordinary clauses 2.2.1/2.2.2 and gets no membership escape.
//
// Clause 7.2 is a PLAIN TYPE-IDENTITY CHECK against the ·xs:error· component
// (§3.16.7.3), never routed through the substitutability machinery: it is a
// direct disjunct of clause 7, and no derivation-relation carve-out for
// xs:error exists in cos-st-derived-ok or key-val-sub-type. Running xs:error's
// own zero-member union through clause 7.1 would fail every time, which is the
// opposite of what the type is FOR (§3.16.7.3 key-error: "it can be used in
// conditional type assignment to cause elements which satisfy certain
// conditions to be invalid").

// errorTypeName is the expanded name of ·xs:error· (§3.16.7.3), clause 7.2's
// referent. The component is seeded by builtin.Seed and reached by expanded
// name, exactly as resolveTypeName reaches it; identity by name rather than by
// pointer is sound because a Schema's {type definitions} hold at most one
// component per expanded name, and it needs no xsd.Error() accessor for a
// singleton nothing else roots on (STYLE T5).
var errorTypeName = QName{Space: XMLSchemaNS, Local: "error"}

// checkTypeTableSubstitutability is Phase D's fifth step: e-props-correct
// clause 7 over every element declaration the compiled schema holds that carries
// a {type table}.
//
// It walks by DESCENT, not over s.elements alone, and the difference is not
// cosmetic: §3.3.2.1's {type table} row is a COMMON mapping rule that
// parser/produce_typetable.go serves from the global <element> path and both
// local ones, so a local declaration carries a table exactly as a top-level one
// does, and Phase A's resolveTypeTable already reaches both. This is the same
// conclusion checkComponentValueConstraints reaches for clause 2 and the
// opposite of checkSubstitutionGroupTypes' for clause 4, whose s.elements-only
// quantifier rests on clause 3 confining a {substitution group affiliations} to
// a global scope. No clause confines a {type table}.
//
// The descent is componentwalk.go's, shared with every other phase that has one,
// so this step supplies only the clause-7 charge and inherits which components
// exist — including a redefine original reached through {base type definition},
// which the hand-written copy this replaces missed (#843). Its roots are its own:
// top-level types, then top-level element declarations (whose inline anonymous
// complex types hold particles of their own), then top-level model group
// definitions.
//
// Components are walked in document order (STYLE D1/D2 — no index map is
// ranged), so the first reported failure is deterministic.
func (s *Schema) checkTypeTableSubstitutability() error {
	w := componentWalk{elementDeclaration: s.checkElementTypeTable}
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
	return nil
}

// checkElementTypeTable charges e-props-correct clause 7 against one element
// declaration, at its own Loc. A declaration with no {type table} owes the clause
// nothing: "If E.{type table} exists" is its whole antecedent.
func (s *Schema) checkElementTypeTable(e ElementDeclaration) error {
	tt, ok := e.TypeTable()
	if !ok {
		return nil
	}
	return s.checkTypeTableAlternatives(e, tt)
}

// checkTypeTableAlternatives charges clause 7 over e's whole {type table}: each
// {alternatives} member in document order, then the {default type definition}.
// The clause names both, and the order is spec order, so a table with two
// offending entries reports the earlier one.
//
// An ABSENT or unresolvable E.{type definition} skips the whole table rather
// than charging it: clause 7.1 predicates over that component, so there is
// nothing for T to be ·validly substitutable· FOR. A dangling type name was
// already charged src-resolve by Phase A, so reaching this point with one means
// a genuinely absent slot (§5.3), which checkElementSubstitutableForHeads and
// declaredTypeRestricts skip identically. The skip is fail-open — it withholds a
// rejection, never invents one.
//
// The blocking-keyword set is read off the unexported field rather than through
// DisallowedSubstitutions(), whose defensive copy would be allocated only to be
// discarded: ValidlySubstitutable never retains or mutates the slice. Same
// reason checkElementSubstitutableForHeads reads substitutionGroupExclusions
// directly.
//
// {disallowed substitutions} is a subset of {substitution, extension,
// restriction} (§3.3.1) while key-val-sub-type's blocking set is drawn from
// {substitution, extension, restriction, list, union}, so it is threaded through
// unchanged: the members the narrower constraint does not name are inert there,
// not errors, and cos-st-derived-ok says of its own set that "only restriction
// is actually relevant".
func (s *Schema) checkTypeTableAlternatives(e ElementDeclaration, tt TypeTable) error {
	declared, ok := s.ResolvedType(e.TypeDefinition())
	if !ok {
		return nil
	}
	for i, alt := range tt.alternatives {
		if err := s.checkTypeAlternativeSubstitutable(e, declared, alt, "{alternatives}["+strconv.Itoa(i)+"]"); err != nil {
			return err
		}
	}
	if isDeclaredTypeItself(e.TypeDefinition(), tt.defaultTypeDefinition.TypeDefinition()) {
		return nil
	}
	return s.checkTypeAlternativeSubstitutable(e, declared, tt.defaultTypeDefinition, "{default type definition}")
}

// isDeclaredTypeItself reports whether a {default type definition}'s {type
// definition} slot holds the declaring element's OWN {type definition} — the
// component §3.3.2.1 dcl.elt.common's case 2 copies into it verbatim, "the {type
// definition} property of the parent Element Declaration". Clause 7.1 is then
// satisfied with no derivation walk at all: cos-ct-derived-ok clause 2.1 (B = D)
// holds outright, and §3.4.6.5's no-identity Note names "the same by
// construction" among the cases where component identity IS determined by this
// specification — the same footing SubstitutionGroupHeadTypeRef stands on.
//
// It answers by ARM, because the arm is where that construction is visible:
//
//   - two TypeDefinitionRefs naming one type, and two SubstitutionGroupHeadTypeRefs
//     naming one owner, each denote one component;
//   - two OWNED anonymous types are read as one component. That is the case-2
//     copy, and it is also the only reading this component model can give: a
//     ComplexType is a value, and the {context} both carry is the owning
//     declaration itself (§3.4.2.1 dcl.ctd.common), so nothing distinguishes
//     them. The one other slot that can put an owned type here — a TRAILING
//     untested <alternative> on the inline arm, under a declaration whose own
//     type is anonymous too — is therefore read as the declared type as well.
//     That shape is a narrow FALSE ACCEPT, not an absent verdict: an anonymous
//     type is unnameable, so it can never appear in another anonymous type's
//     {base type definition} chain, and sameTypeDefinition (complexderivation.go)
//     would have rejected it correctly on that ground. checkTypeAlternativeSubstitutable,
//     the one caller, is the sole consumer this costs a clause-7 charge it should
//     have made; nothing else reads this function.
//
// The {alternatives} members are deliberately NOT routed through this. §3.12.2
// declare-ta mints each of their owned types from its own <alternative> element,
// so no member is the declaration's own type by construction, and each is charged
// clause 7 in full.
func isDeclaredTypeItself(declared, dflt TypeDefinitionOrRef) bool {
	switch d := declared.(type) {
	case nil:
		return false
	case TypeDefinitionRef:
		ref, sameArm := dflt.(TypeDefinitionRef)
		return sameArm && ref.Name == d.Name
	case InlineTypeDefinition:
		_, sameArm := dflt.(InlineTypeDefinition)
		return sameArm
	case SubstitutionGroupHeadTypeRef:
		head, sameArm := dflt.(SubstitutionGroupHeadTypeRef)
		return sameArm && head.Head == d.Head
	default:
		panic("xsd: isDeclaredTypeItself: non-exhaustive TypeDefinitionOrRef switch")
	}
}

// checkTypeAlternativeSubstitutable decides clause 7 for ONE Type Alternative,
// clause 7.2 first: an alternative naming ·xs:error· satisfies the clause
// outright and never reaches the substitutability call, which its zero-member
// union would fail. slot names the entry for the message.
//
// The alternative's {type definition} is followed through ResolvedType, so the
// clause charges the COMPONENT the slot holds and not merely a name it carries.
// An alternative on §3.12.2's INLINE arm is therefore charged exactly as a
// by-name one is: clause 7 quantifies over T1.{type definition}, which every arm
// of the slot supplies. The one entry that reaches this without being charged is
// the {default type definition} that IS the declaration's own type, which
// isDeclaredTypeItself discharges before the call.
//
// An alternative whose {type definition} reaches no component is SKIPPED. A
// present name that resolves to nothing was already charged src-resolve by Phase
// A's resolveTypeTable; an ABSENT slot — nil, which a SchemaBuilder caller may
// leave — has no T for the clause to quantify over. Both are fail-open.
func (s *Schema) checkTypeAlternativeSubstitutable(e ElementDeclaration, declared TypeDefinition, alt TypeAlternative, slot string) error {
	t, ok := s.ResolvedType(alt.TypeDefinition())
	if !ok {
		return nil
	}
	if t.Name() == errorTypeName {
		return nil // clause 7.2
	}
	substitutable, err := s.ValidlySubstitutable(t, declared, e.disallowedSubstitutions)
	if err != nil {
		return err
	}
	if substitutable {
		return nil // clause 7.1
	}
	return xsderr.New(ruleEPropsCorrect, e.Loc(),
		"element declaration %s has a {type table} whose %s is typed %s, which is neither ·validly substitutable· for the declaration's own {type definition} %s subject to its {disallowed substitutions} %v nor the type ·xs:error·; e-props-correct clause 7 requires one of the two",
		e.Name(), slot, typeDefinitionLabel(t), typeDefinitionLabel(declared), e.disallowedSubstitutions)
}
