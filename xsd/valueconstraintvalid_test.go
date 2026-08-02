package xsd

import (
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
type stubValueSpace struct {
	same    bool
	decided bool
	calls   int
}

func (s *stubValueSpace) Identical(*SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	s.calls++
	return s.same, s.decided
}

func (s *stubValueSpace) EqualOrIdentical(*SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	s.calls++
	return s.same, s.decided
}

// vcSchema is bSchema with a ValueSpace installed: it finalizes through
// FinalizeWith so the {value} halves of au-props-correct clause 3 and
// loc-testSubP clauses 4.2/5.2.2 are decidable.
func vcSchema(t *testing.T, vs ValueSpace, build func(*SchemaBuilder)) (*Schema, error) {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	str := dSimple(t, uq("str"), AnyAtomicType())
	b.AddType(str)
	b.AddType(dSimple(t, uq("narrow"), str))
	if build != nil {
		build(b)
	}
	return b.FinalizeWith(vs)
}

// vcGlobalFixed adds a global attribute declaration g of type str whose {value
// constraint} is fixed to lexical.
func vcGlobalFixed(t *testing.T, lexical string) func(*SchemaBuilder) {
	t.Helper()
	return func(b *SchemaBuilder) {
		vc := NewValueConstraint(ValueFixed, lexical)
		d, err := NewAttributeDeclaration(xsderr.Loc{}, uq("g"), TypeDefinitionRef{Name: uq("str")}, ScopeGlobal, &vc, false, nil)
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
	vc := NewValueConstraint(kind, lexical)
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
	declVC := NewValueConstraint(ValueFixed, "7")
	decl, err := NewAttributeDeclaration(xsderr.Loc{}, uq("a"), TypeDefinitionRef{Name: uq("str")}, ScopeLocal, &declVC, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	useVC := NewValueConstraint(ValueFixed, "07")
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
			declVC := NewValueConstraint(tc.declKind, "7")
			_, err := vcSchema(t, vs, func(b *SchemaBuilder) {
				d, err := NewAttributeDeclaration(xsderr.Loc{}, uq("g"), TypeDefinitionRef{Name: uq("str")}, ScopeGlobal, &declVC, false, nil)
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
		b.AddType(dSimple(t, uq("str"), AnyAtomicType()))
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
	if _, err := b.FinalizeWith(&stubValueSpace{same: false, decided: true}); err == nil {
		t.Fatal("FinalizeWith a decided-not-identical value space must reject the same schema")
	}
}

// TestFinalizeWithNilValueSpacePanics pins the nil guard: a nil capability is a
// caller bug, not a schema-validity condition.
func TestFinalizeWithNilValueSpacePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("FinalizeWith(nil) must panic")
		}
	}()
	_, _ = NewSchemaBuilder().FinalizeWith(nil)
}
