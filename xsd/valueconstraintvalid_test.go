package xsd

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: Phase E is an unexported finalize helper
// (STYLE T5). Package xsd is a pure leaf, so nothing here imports package value
// — every {value} verdict comes from stubValueSpace, which is the whole point of
// the ValueSpace seam.

// stubValueSpace is a ValueSpace with a fixed verdict, plus a call counter so a
// test can pin that a clause DID consult the value space rather than deciding
// structurally. Its verdict is the same for both relations: the two are told
// apart by the caller, not by this stub.
//
// ValidDefault is answered separately (validLexicals/defaultCalls), because it is
// a DIFFERENT question from the comparisons — "is this lexical a value of this
// type" rather than "are these two the same value" — and a clause-3 test must be
// able to keep clause 2 quiet while pinning a not-identical verdict. A nil
// validLexicals means "every lexical is valid", which is the shape every
// pre-#371 test wants: those schemas exercise clause 3 and must not trip clause
// 2 in passing.
type stubValueSpace struct {
	same    bool
	decided bool
	calls   int

	// validLexicals, when non-nil, is the set of {lexical form}s ValidDefault
	// reports valid; every other lexical is a DECIDED reject.
	validLexicals map[string]bool
	// undecidedDefault makes ValidDefault answer undecided for every lexical,
	// the fail-open arm.
	undecidedDefault bool
	defaultCalls     int
}

func (s *stubValueSpace) Identical(TypeResolver, *SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	s.calls++
	return s.same, s.decided
}

func (s *stubValueSpace) EqualOrIdentical(TypeResolver, *SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	s.calls++
	return s.same, s.decided
}

func (s *stubValueSpace) ValidDefault(_ TypeResolver, _ *SimpleType, vc ValueConstraint) (error, bool) {
	s.defaultCalls++
	if s.undecidedDefault {
		return nil, false
	}
	if s.validLexicals == nil || s.validLexicals[vc.LexicalForm()] {
		return nil, true
	}
	// The real seam hands back the datatype-layer verdict, so the stub does
	// too: a caller that wrapped it would otherwise be pinned against a cause
	// no implementation produces.
	return xsderr.New("cvc-datatype-valid", xsderr.Loc{},
		"the literal %q is not in the ·lexical space· of the type", vc.LexicalForm()), true
}

// vcSchema is bSchema with a ValueSpace installed: it finalizes through
// FinalizeWith so the {value} halves of au-props-correct clause 3 and
// loc-testSubP clauses 4.2/5.2.2 are decidable. The other capability the seam
// takes is the undecided one — cos-st-restricts' facet-value half needs the
// generated applicability table, which this leaf cannot reach (its coverage is
// pinned from package builtin instead).
func vcSchema(t *testing.T, vs ValueSpace, build func(*SchemaBuilder)) (*Schema, error) {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	str := dPrimitive(t, uq("str"))
	b.AddType(str)
	b.AddType(dSimple(t, uq("narrow"), str))
	if build != nil {
		build(b)
	}
	return b.FinalizeWith(vs, undecidedRestrictionChecker{})
}

// vcGlobalFixed adds a global attribute declaration g of type str whose {value
// constraint} is fixed to lexical.
func vcGlobalFixed(t *testing.T, lexical string) func(*SchemaBuilder) {
	t.Helper()
	return func(b *SchemaBuilder) {
		vc := NewValueConstraint(ValueFixed, lexical, nil, nil)
		d, err := NewAttributeDeclaration(xsderr.Loc{}, uq("g"), TypeDefinitionRef{Name: uq("str")}, NewAttributeGlobalScope(), &vc, false, nil)
		if err != nil {
			t.Fatalf("NewAttributeDeclaration: %v", err)
		}
		b.AddAttribute(d)
	}
}

// vcRefUse builds an attribute use referencing the global declaration g and
// carrying its own {value constraint}.
func vcRefUse(t *testing.T, kind ValueConstraintKind, lexical string) AttributeUse {
	t.Helper()
	vc := NewValueConstraint(kind, lexical, nil, nil)
	u, err := NewAttributeUse(xsderr.Loc{}, false, AttributeDeclarationRef{Name: uq("g")}, &vc, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	return u
}

// TestPhaseEClause3VarietyHalfRefVariant pins the half NewAttributeUse cannot
// decide: for the AttributeDeclarationRef variant the declaration is an
// unresolved QName at construction, so "the declaration fixes it, therefore the
// use must fix it too" (au-props-correct clause 3) is decidable only once the
// schema is finalized. No value space is needed — this half is structural.
func TestPhaseEClause3VarietyHalfRefVariant(t *testing.T) {
	for _, tc := range []struct {
		name    string
		use     AttributeUse
		wantErr bool
	}{
		{"a default use against a fixed declaration is rejected", vcRefUse(t, ValueDefault, "7"), true},
		{"a fixed use against a fixed declaration is not", vcRefUse(t, ValueFixed, "7"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vcSchema(t, undecidedValueSpace{}, func(b *SchemaBuilder) {
				vcGlobalFixed(t, "7")(b)
				b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{tc.use}, nil))
			})
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected rejection: %v", err)
				}
				return
			}
			expectRule(t, err, ruleAuPropsCorrect)
		})
	}
}

// TestPhaseEClause3ValueHalf pins the {value}-identity half: with both varieties
// fixed, the verdict is the ValueSpace's, and an UNDECIDED verdict accepts. The
// two lexical forms deliberately differ, so a lexical comparison would decide
// where the spec says only the value space may.
func TestPhaseEClause3ValueHalf(t *testing.T) {
	for _, tc := range []struct {
		name    string
		vs      *stubValueSpace
		wantErr bool
	}{
		{"decided identical accepts", &stubValueSpace{same: true, decided: true}, false},
		{"decided NOT identical rejects", &stubValueSpace{same: false, decided: true}, true},
		{"undecided accepts (fail-open)", &stubValueSpace{same: false, decided: false}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vcSchema(t, tc.vs, func(b *SchemaBuilder) {
				vcGlobalFixed(t, "7")(b)
				b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{vcRefUse(t, ValueFixed, "07")}, nil))
			})
			if tc.vs.calls == 0 {
				t.Fatal("clause 3 decided without consulting the ValueSpace")
			}
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected rejection: %v", err)
				}
				return
			}
			expectRule(t, err, ruleAuPropsCorrect)
		})
	}
}

// TestPhaseEClause3LocalVariant pins that clause 3's {value} half fires for the
// LocalAttributeDeclaration variant too. NewAttributeUse already rejects the
// VARIETY mismatch there, so only the value half can reach Phase E — the rule
// text draws no variant distinction and neither does the check.
func TestPhaseEClause3LocalVariant(t *testing.T) {
	declVC := NewValueConstraint(ValueFixed, "7", nil, nil)
	decl, err := NewAttributeDeclaration(xsderr.Loc{}, uq("a"), TypeDefinitionRef{Name: uq("str")}, aLocalScope(t), &declVC, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	useVC := NewValueConstraint(ValueFixed, "07", nil, nil)
	use, err := NewAttributeUse(xsderr.Loc{}, false, LocalAttributeDeclaration{Declaration: decl}, &useVC, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	vs := &stubValueSpace{same: false, decided: true}
	_, err = vcSchema(t, vs, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{use}, nil))
	})
	expectRule(t, err, ruleAuPropsCorrect)
}

// TestPhaseEClause3AntecedentNotMet pins that clause 3 is DISCHARGED, without
// consulting the value space, whenever either conjunct of its antecedent fails:
// the use has no {value constraint} of its own, or the declaration's is not
// fixed. A use with no own constraint is the ·effective value constraint· case
// (key-evc) — clause 3 is not about that.
func TestPhaseEClause3AntecedentNotMet(t *testing.T) {
	bare, err := NewAttributeUse(xsderr.Loc{}, false, AttributeDeclarationRef{Name: uq("g")}, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	for _, tc := range []struct {
		name     string
		declKind ValueConstraintKind
		use      AttributeUse
	}{
		{"the use carries no {value constraint} of its own", ValueFixed, bare},
		{"the declaration's {value constraint} is default, not fixed", ValueDefault, vcRefUse(t, ValueDefault, "9")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vs := &stubValueSpace{same: false, decided: true}
			declVC := NewValueConstraint(tc.declKind, "7", nil, nil)
			_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
				d, err := NewAttributeDeclaration(xsderr.Loc{}, uq("g"), TypeDefinitionRef{Name: uq("str")}, NewAttributeGlobalScope(), &declVC, false, nil)
				if err != nil {
					t.Fatalf("NewAttributeDeclaration: %v", err)
				}
				b.AddAttribute(d)
				b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{tc.use}, nil))
			})
			if err != nil {
				t.Fatalf("clause 3's antecedent is not met, so it must be discharged: %v", err)
			}
			if vs.calls != 0 {
				t.Fatalf("the ValueSpace was consulted %d time(s) for a discharged clause", vs.calls)
			}
		})
	}
}

// TestPhaseEReachesEveryAttributeUseSite pins the walk: Phase E must reach an
// attribute use wherever the component model can hold one, not only on a
// top-level complex type. The offending use is identical in each case, so any
// site that goes unwalked shows up as a missing rejection.
func TestPhaseEReachesEveryAttributeUseSite(t *testing.T) {
	bad := func() []AttributeUse { return []AttributeUse{vcRefUse(t, ValueDefault, "7")} }
	inlineType := func(name string) ElementDeclaration {
		return dOwnInline(t, uq(name), dType(t, QName{}, anyTypeName, EmptyContent{}, bad(), nil), NewGlobalScope())
	}

	for _, tc := range []struct {
		name  string
		build func(*SchemaBuilder)
	}{
		{"a top-level complex type's own {attribute uses}", func(b *SchemaBuilder) {
			b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, bad(), nil))
		}},
		{"an inline anonymous complex type under a top-level element", func(b *SchemaBuilder) {
			b.AddElement(inlineType("e"))
		}},
		{"an inline complex type nested in a content model", func(b *SchemaBuilder) {
			inner := uOne(t, ResolvedTerm{Term: inlineType("nested")})
			b.AddType(dType(t, uq("t"), anyTypeName, dElementContent(t, false, uGroup(t, CompositorSequence, inner)), nil, nil))
		}},
		{"an inline complex type inside a top-level model group definition", func(b *SchemaBuilder) {
			inner := uOne(t, ResolvedTerm{Term: inlineType("nested")})
			mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("mg"), uGroup(t, CompositorSequence, inner), nil)
			if err != nil {
				t.Fatalf("NewModelGroupDefinition: %v", err)
			}
			b.AddModelGroup(mgd)
		}},
		{"a top-level attribute group definition", func(b *SchemaBuilder) {
			g, err := NewAttributeGroupDefinition(xsderr.Loc{}, uq("ag"), bad(), nil, nil)
			if err != nil {
				t.Fatalf("NewAttributeGroupDefinition: %v", err)
			}
			b.AddAttributeGroup(g)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vcSchema(t, undecidedValueSpace{}, func(b *SchemaBuilder) {
				vcGlobalFixed(t, "7")(b)
				tc.build(b)
			})
			expectRule(t, err, ruleAuPropsCorrect)
		})
	}
}

// TestPhaseEOwnerNamesAnonymousComplexType pins the owner phrase Phase E puts in
// its message. An inline <xs:complexType> has no {name}, and the zero QName
// renders as "", so concatenating it would leave a hole ("complex type  gives
// …"); the anonymous case must be described instead.
func TestPhaseEOwnerNamesAnonymousComplexType(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(*SchemaBuilder)
		want  string
	}{
		{"named", func(b *SchemaBuilder) {
			b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{vcRefUse(t, ValueDefault, "7")}, nil))
		}, "complex type {urn:upa}t "},
		{"anonymous", func(b *SchemaBuilder) {
			ct := dType(t, QName{}, anyTypeName, EmptyContent{}, []AttributeUse{vcRefUse(t, ValueDefault, "7")}, nil)
			b.AddElement(dOwnInline(t, uq("e"), ct, NewGlobalScope()))
		}, "anonymous complex type "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vcSchema(t, undecidedValueSpace{}, func(b *SchemaBuilder) {
				vcGlobalFixed(t, "7")(b)
				tc.build(b)
			})
			expectRule(t, err, ruleAuPropsCorrect)
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("message %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

// TestPhaseEUnresolvableRefIsSkipped pins the fail-open branch inside an
// attribute group: Phase A never walks {attribute group definitions}, so a
// dangling <attribute ref> there reaches Phase E unvetted and must be SKIPPED,
// not charged — au-props-correct clause 3 has no declaration to read.
func TestPhaseEUnresolvableRefIsSkipped(t *testing.T) {
	vs := &stubValueSpace{same: false, decided: true}
	_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
		use := vcRefUse(t, ValueDefault, "7") // g is never declared here
		g, err := NewAttributeGroupDefinition(xsderr.Loc{}, uq("ag"), []AttributeUse{use}, nil, nil)
		if err != nil {
			t.Fatalf("NewAttributeGroupDefinition: %v", err)
		}
		b.AddAttributeGroup(g)
	})
	if err != nil {
		t.Fatalf("an unresolvable <attribute ref> must be skipped, not charged: %v", err)
	}
	if vs.calls != 0 {
		t.Fatalf("the ValueSpace was consulted %d time(s) for a use with no resolvable declaration", vs.calls)
	}
}

// TestFinalizeWithoutValueSpaceFailsOpen pins that plain Finalize installs the
// undecided value space rather than nil: the same schema Phase E rejects under a
// decided-not-identical value space is ACCEPTED with no value space at all, and
// no nil dereference occurs.
func TestFinalizeWithoutValueSpaceFailsOpen(t *testing.T) {
	build := func(b *SchemaBuilder) {
		b.AddType(dAnyType(t))
		b.AddType(dPrimitive(t, uq("str")))
		vcGlobalFixed(t, "7")(b)
		b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{vcRefUse(t, ValueFixed, "07")}, nil))
	}
	b := NewSchemaBuilder()
	build(b)
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize must fail open on the {value} half: %v", err)
	}
	b = NewSchemaBuilder()
	build(b)
	if _, err := b.FinalizeWith(&stubValueSpace{same: false, decided: true}, undecidedRestrictionChecker{}); err == nil {
		t.Fatal("FinalizeWith a decided-not-identical value space must reject the same schema")
	}
}

// TestFinalizeWithNilCapabilityPanics pins the nil guard on BOTH parameters: a
// nil capability is a caller bug, not a schema-validity condition, and one seam
// installing two capabilities must not accept a half-installed pair.
func TestFinalizeWithNilCapabilityPanics(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func()
	}{
		{"nil ValueSpace", func() { _, _ = NewSchemaBuilder().FinalizeWith(nil, undecidedRestrictionChecker{}) }},
		{"nil SimpleTypeRestrictionChecker", func() { _, _ = NewSchemaBuilder().FinalizeWith(undecidedValueSpace{}, nil) }},
		{"both nil", func() { _, _ = NewSchemaBuilder().FinalizeWith(nil, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("FinalizeWith with a %s must panic", tc.name)
				}
			}()
			tc.call()
		})
	}
}

// vcLoc is a recognizable position, so a clause-2 test can pin that the rejection
// is charged where the reader must edit rather than at the zero Loc.
var vcLoc = xsderr.Loc{URI: "decl.xsd", Line: 12, Col: 3}

// vcOnly is the stub configuration clause 2's tests want: exactly one lexical is
// a valid default, every other is a DECIDED reject.
func vcOnly(valid string) *stubValueSpace {
	return &stubValueSpace{validLexicals: map[string]bool{valid: true}}
}

// vcGlobalDecl adds a global attribute declaration g of type str whose {value
// constraint} is present (kind/lexical) or absent (vc nil), at vcLoc.
func vcGlobalDecl(t *testing.T, vc *ValueConstraint) func(*SchemaBuilder) {
	t.Helper()
	return func(b *SchemaBuilder) {
		d, err := NewAttributeDeclaration(vcLoc, uq("g"), TypeDefinitionRef{Name: uq("str")}, NewAttributeGlobalScope(), vc, false, nil)
		if err != nil {
			t.Fatalf("NewAttributeDeclaration: %v", err)
		}
		b.AddAttribute(d)
	}
}

// TestPhaseEAPropsCorrectClause2 pins a-props-correct (§3.2.6.1) clause 2 over the
// declaration-side walk: a GLOBAL attribute declaration whose own {value
// constraint} is not a valid default with respect to its {type definition} is
// rejected (cos-valid-simple-default §3.2.6.2), charged to that rule at the
// DECLARATION's own Loc — not at some enclosing component's.
//
// The declaration is referenced by no attribute use at all, which is the case the
// use-side walk cannot reach: a global declaration is charged in its own right.
func TestPhaseEAPropsCorrectClause2(t *testing.T) {
	for _, kind := range []ValueConstraintKind{ValueDefault, ValueFixed} {
		t.Run(kind.String(), func(t *testing.T) {
			vs := vcOnly("7")
			vc := NewValueConstraint(kind, "not a value of str", nil, nil)
			_, err := vcSchema(t, vs, vcGlobalDecl(t, &vc))
			expectRule(t, err, ruleAPropsCorrect)
			if vs.defaultCalls == 0 {
				t.Fatal("clause 2 decided without consulting the ValueSpace")
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) && xe.Loc != vcLoc {
				t.Errorf("charged at %s, want the declaration's own %s", xe.Loc, vcLoc)
			}
			if !strings.Contains(err.Error(), "a-props-correct clause 2") {
				t.Errorf("message %q does not name the clause", err.Error())
			}
		})
	}
}

// TestPhaseEAPropsCorrectClause2Accepts pins the two ways clause 2 must NOT fire:
// a valid default passes cleanly (no false positive), and a declaration with NO
// {value constraint} does not reach the clause at all — "if there is a {value
// constraint}" is a presence gate, not a vacuously-satisfied test, so the value
// space must not even be consulted.
func TestPhaseEAPropsCorrectClause2Accepts(t *testing.T) {
	valid := NewValueConstraint(ValueFixed, "7", nil, nil)
	for _, tc := range []struct {
		name      string
		vc        *ValueConstraint
		wantCalls bool
	}{
		{"a valid default passes", &valid, true},
		{"no {value constraint}: the clause is not reached", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vs := vcOnly("7")
			if _, err := vcSchema(t, vs, vcGlobalDecl(t, tc.vc)); err != nil {
				t.Fatalf("unexpected rejection: %v", err)
			}
			if got := vs.defaultCalls > 0; got != tc.wantCalls {
				t.Errorf("ValueSpace consulted %d time(s), want consulted=%t", vs.defaultCalls, tc.wantCalls)
			}
		})
	}
}

// TestPhaseEAPropsCorrectClause2LocalDeclaration pins the other half of the split:
// a LOCAL attribute declaration is in no global table — its owning Attribute Use
// is its sole owner — so the declaration-side walk cannot see it and the use-side
// walk must charge it, still under a-props-correct and still at the DECLARATION's
// own Loc.
func TestPhaseEAPropsCorrectClause2LocalDeclaration(t *testing.T) {
	declVC := NewValueConstraint(ValueDefault, "not a value of str", nil, nil)
	decl, err := NewAttributeDeclaration(vcLoc, uq("a"), TypeDefinitionRef{Name: uq("str")}, aLocalScope(t), &declVC, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	use, err := NewAttributeUse(xsderr.Loc{}, false, LocalAttributeDeclaration{Declaration: decl}, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	vs := vcOnly("7")
	_, err = vcSchema(t, vs, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{use}, nil))
	})
	expectRule(t, err, ruleAPropsCorrect)
	var xe *xsderr.Error
	if errors.As(err, &xe) && xe.Loc != vcLoc {
		t.Errorf("charged at %s, want the local declaration's own %s", xe.Loc, vcLoc)
	}
}

// TestPhaseEAuPropsCorrectClause2 pins au-props-correct (§3.5.6) clause 2: an
// Attribute Use whose OWN {value constraint} is not a valid default with respect
// to the RESOLVED {attribute declaration}.{type definition} is rejected, charged
// to au-props-correct at the ENCLOSING component's Loc (an Attribute Use retains
// none of its own).
//
// The declaration carries NO {value constraint} in the first row, so clause 3's
// antecedent fails outright: clause 2 must fire on its own, not as a side effect
// of the clause-3 comparison.
func TestPhaseEAuPropsCorrectClause2(t *testing.T) {
	for _, tc := range []struct {
		name   string
		declVC *ValueConstraint
	}{
		{"the declaration has no {value constraint}: clause 3 cannot fire", nil},
		{"the declaration is fixed to a valid lexical", func() *ValueConstraint {
			vc := NewValueConstraint(ValueFixed, "7", nil, nil)
			return &vc
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vs := vcOnly("7")
			_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
				vcGlobalDecl(t, tc.declVC)(b)
				bad := vcRefUse(t, ValueFixed, "not a value of str")
				b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{bad}, nil))
			})
			expectRule(t, err, ruleAuPropsCorrect)
			if !strings.Contains(err.Error(), "au-props-correct clause 2") {
				t.Errorf("message %q does not name the clause", err.Error())
			}
			if !strings.Contains(err.Error(), "complex type {urn:upa}t attribute {urn:upa}g") {
				t.Errorf("message %q does not name the owner and the attribute", err.Error())
			}
		})
	}
}

// TestPhaseEClause2PrecedesClause3 pins the spec order (§3.5.6 lists 2 before 3):
// a use that violates BOTH — its own fixed lexical is not a valid default AND it
// differs from the declaration's fixed {value} — reports clause 2, the more basic
// failure. "Is it the same value as the declaration's" is moot for a lexical that
// denotes no value at all.
func TestPhaseEClause2PrecedesClause3(t *testing.T) {
	vs := vcOnly("7")
	vs.same, vs.decided = false, true
	_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
		vcGlobalFixed(t, "7")(b)
		b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{vcRefUse(t, ValueFixed, "not a value of str")}, nil))
	})
	expectRule(t, err, ruleAuPropsCorrect)
	if !strings.Contains(err.Error(), "au-props-correct clause 2") {
		t.Errorf("message %q reports a later clause; clause 2 must win", err.Error())
	}
	if vs.calls != 0 {
		t.Errorf("clause 3 consulted the value space %d time(s) after clause 2 rejected", vs.calls)
	}
}

// TestPhaseEClause2ClauseDerivesFromRule pins checkSimpleDefault's message to the
// citation it is charged under: the rule's own name plus its caller's own clause
// number, never a hardcoded pair with one of them as the default (STYLE D3). All
// four of today's callers are asserted, so the four phrases the message may carry
// are nailed down byte for byte.
//
// The fourth is the one that broke the constant " clause 2" suffix the first three
// shared: cvc-elt clause 5.1.1 charges the same predicate at ASSESSMENT time
// ([Schema.ElementDefaultValid], #853), under a different rule AND a different
// clause number, so the clause travels as its own parameter.
//
// It calls the helper directly rather than through a schema because the callers
// reach it from different walks, and what is under test is the message, not any of
// the walks.
func TestPhaseEClause2ClauseDerivesFromRule(t *testing.T) {
	for _, tc := range []struct {
		rule   xsderr.Rule
		clause string
		want   string
	}{
		{ruleAPropsCorrect, "2", "a-props-correct clause 2"},
		{ruleAuPropsCorrect, "2", "au-props-correct clause 2"},
		{ruleEPropsCorrect, "2", "e-props-correct clause 2"},
		{"cvc-elt", "5.1.1", "cvc-elt clause 5.1.1"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			s := &Schema{}
			err := s.checkSimpleDefault(vcOnly("7"), tc.rule, tc.clause, vcLoc, "attribute declaration g", nil, NewValueConstraint(ValueDefault, "not a value of str", nil, nil))
			expectRule(t, err, tc.rule)
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("message %q does not name %q", err.Error(), tc.want)
			}
		})
	}
}

// TestPhaseEClause2FailsOpenWhenUndecided pins the fail-open contract at both call
// sites: an UNDECIDED ValueSpace verdict accepts, and so does the undecided value
// space a plain Finalize installs. That is the branch protecting every ungoverned
// type — xs:anySimpleType, which §3.2.2.2's third tier gives a typeless attribute
// — and every QName/NOTATION default, which no ValueConstraint can decide.
func TestPhaseEClause2FailsOpenWhenUndecided(t *testing.T) {
	build := func(b *SchemaBuilder) {
		declVC := NewValueConstraint(ValueDefault, "not a value of str", nil, nil)
		vcGlobalDecl(t, &declVC)(b)
		bad := vcRefUse(t, ValueDefault, "also not a value of str")
		b.AddType(dType(t, uq("t"), anyTypeName, EmptyContent{}, []AttributeUse{bad}, nil))
	}
	undecidedStub := &stubValueSpace{undecidedDefault: true}
	for _, tc := range []struct {
		name string
		vs   ValueSpace
	}{
		{"a ValueSpace that answers undecided", undecidedStub},
		{"no ValueSpace installed at all", undecidedValueSpace{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := vcSchema(t, tc.vs, build); err != nil {
				t.Fatalf("an undecided verdict must accept, never reject: %v", err)
			}
		})
	}
	if undecidedStub.defaultCalls == 0 {
		t.Fatal("clause 2 accepted without consulting the ValueSpace at all")
	}
}
