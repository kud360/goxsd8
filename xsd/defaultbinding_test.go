package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: key-dft-binding and loc-testSubP are
// unexported finalize-phase helpers (STYLE T5). The component builders come from
// particleattribution_test.go and complexderivation_test.go (STYLE T4).

// bSchema finalizes a schema carrying xs:anyType, the named simple types the
// attribute helpers refer to, and whatever build adds, returning the *Schema the
// binding helpers are methods on.
func bSchema(t *testing.T, build func(*SchemaBuilder)) *Schema {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	str := dPrimitive(t, uq("str"))
	b.AddType(str)
	b.AddType(dSimple(t, uq("narrow"), str))
	if build != nil {
		build(b)
	}
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// TestAttributeDefaultBindingCases covers the key-dft-binding cases this
// rendering decides: case 2 (a matching {attribute use}), cases 4/5/6 (the
// {attribute wildcard}'s keyword), and the "no binding at all" answer the caller
// charges.
func TestAttributeDefaultBindingCases(t *testing.T) {
	s := bSchema(t, nil)
	use := dAttr(t, uq("a"), uq("str"))
	strict := uWildcard(t, NamespaceConstraintAny, nil, ProcessStrict)
	skip := uWildcard(t, NamespaceConstraintAny, nil, ProcessSkip)

	for _, tc := range []struct {
		name    string
		ct      ComplexType
		query   QName
		wantOK  bool
		wantUse bool            // expect an attributeUseBinding
		wantKey ProcessContents // expect a wildcardKeywordBinding with this keyword
	}{
		{"case 2: a matching attribute use wins",
			dType(t, uq("c"), anyTypeName, EmptyContent{}, []AttributeUse{use}, &strict),
			uq("a"), true, true, 0},
		{"case 4: a strict wildcard with no matching use yields the keyword",
			dType(t, uq("c"), anyTypeName, EmptyContent{}, []AttributeUse{use}, &strict),
			uq("z"), true, false, ProcessStrict},
		{"case 6: a skip wildcard yields skip",
			dType(t, uq("c"), anyTypeName, EmptyContent{}, nil, &skip),
			uq("z"), true, false, ProcessSkip},
		{"no use and no wildcard: no binding",
			dType(t, uq("c"), anyTypeName, EmptyContent{}, nil, nil),
			uq("z"), false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := s.attributeDefaultBinding(tc.ct, tc.query)
			if ok != tc.wantOK {
				t.Fatalf("attributeDefaultBinding ok = %t, want %t", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if tc.wantUse {
				if _, isUse := got.(attributeUseBinding); !isUse {
					t.Fatalf("binding = %T, want attributeUseBinding", got)
				}
				return
			}
			k, isKeyword := got.(wildcardKeywordBinding)
			if !isKeyword {
				t.Fatalf("binding = %T, want wildcardKeywordBinding", got)
			}
			if k.keyword != tc.wantKey {
				t.Fatalf("keyword = %s, want %s", k.keyword, tc.wantKey)
			}
		})
	}
}

// TestBindingSubsumesKeywords covers loc-testSubP clauses 1-3 plus the
// documented fail-open on a strict G. The two complex types are only message
// context here, so a pair of trivial ones is enough.
func TestBindingSubsumesKeywords(t *testing.T) {
	s := bSchema(t, nil)
	tt := dType(t, uq("t"), anyTypeName, EmptyContent{}, nil, nil)
	bb := dType(t, uq("b"), anyTypeName, EmptyContent{}, nil, nil)
	use := attributeUseBinding{use: dAttr(t, uq("a"), uq("str"))}
	decl := elementDeclarationBinding{decl: uLocal(t, uq("a"), uq("str"))}

	for _, tc := range []struct {
		name     string
		general  defaultBinding
		specific defaultBinding
		wantOK   bool
	}{
		{"clause 1: skip subsumes an attribute use",
			wildcardKeywordBinding{keyword: ProcessSkip}, use, true},
		{"clause 1: skip subsumes skip",
			wildcardKeywordBinding{keyword: ProcessSkip}, wildcardKeywordBinding{keyword: ProcessSkip}, true},
		{"clause 2: lax subsumes an attribute use",
			wildcardKeywordBinding{keyword: ProcessLax}, use, true},
		{"clause 2: lax does NOT subsume skip",
			wildcardKeywordBinding{keyword: ProcessLax}, wildcardKeywordBinding{keyword: ProcessSkip}, false},
		{"clause 3: strict subsumes strict",
			wildcardKeywordBinding{keyword: ProcessStrict}, wildcardKeywordBinding{keyword: ProcessStrict}, true},
		{"strict over an attribute use is the documented fail-open",
			wildcardKeywordBinding{keyword: ProcessStrict}, use, true},
		{"an Element Declaration does not subsume an Attribute Use",
			decl, use, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := s.checkBindingSubsumes(uq("a"), tt, bb, tc.general, tc.specific)
			if tc.wantOK && err != nil {
				t.Fatalf("expected ·subsumes·, got %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestBindingSubsumesAttributeUses covers loc-testSubP clause 5's three
// sub-conditions and the pass case.
func TestBindingSubsumesAttributeUses(t *testing.T) {
	s := bSchema(t, nil)
	tt := dType(t, uq("t"), anyTypeName, EmptyContent{}, nil, nil)
	bb := dType(t, uq("b"), anyTypeName, EmptyContent{}, nil, nil)
	fixed := NewValueConstraint(ValueFixed, "7")
	other := NewValueConstraint(ValueDefault, "7")

	inheritable := func(u AttributeUse) AttributeUse {
		u.inheritable = true
		return u
	}

	for _, tc := range []struct {
		name     string
		general  AttributeUse
		specific AttributeUse
		wantOK   bool
	}{
		{"identical uses subsume",
			dAttr(t, uq("a"), uq("str")), dAttr(t, uq("a"), uq("str")), true},
		{"5.1: a narrower type is validly derived",
			dAttr(t, uq("a"), uq("str")), dAttr(t, uq("a"), uq("narrow")), true},
		{"5.1: a wider type is not",
			dAttr(t, uq("a"), uq("narrow")), dAttr(t, uq("a"), uq("str")), false},
		{"5.2: keeping the base's fixed value is fine",
			dAttrUse(t, uq("a"), uq("str"), false, &fixed), dAttrUse(t, uq("a"), uq("str"), false, &fixed), true},
		{"5.2: dropping the base's fixed value is not",
			dAttrUse(t, uq("a"), uq("str"), false, &fixed), dAttr(t, uq("a"), uq("str")), false},
		{"5.2: replacing fixed with default is not",
			dAttrUse(t, uq("a"), uq("str"), false, &fixed), dAttrUse(t, uq("a"), uq("str"), false, &other), false},
		{"5.2.1: a base default constraint imposes nothing",
			dAttrUse(t, uq("a"), uq("str"), false, &other), dAttr(t, uq("a"), uq("str")), true},
		{"5.3: {inheritable} must be equal",
			dAttr(t, uq("a"), uq("str")), inheritable(dAttr(t, uq("a"), uq("str"))), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := s.checkBindingSubsumes(uq("a"), tt, bb,
				attributeUseBinding{use: tc.general}, attributeUseBinding{use: tc.specific})
			if tc.wantOK && err != nil {
				t.Fatalf("expected ·subsumes·, got %v", err)
			}
			if !tc.wantOK {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestEffectiveValueConstraintFallback pins key-evc's three-step fallback,
// including the Ref variant, which is why the helper lives on *Schema at all.
func TestEffectiveValueConstraintFallback(t *testing.T) {
	declFixed := NewValueConstraint(ValueFixed, "7")
	useDefault := NewValueConstraint(ValueDefault, "9")
	s := bSchema(t, func(b *SchemaBuilder) {
		global, err := NewAttributeDeclaration(xsderr.Loc{}, uq("g"), TypeDefinitionRef{Name: uq("str")}, NewAttributeGlobalScope(), &declFixed, false, nil)
		if err != nil {
			t.Fatalf("NewAttributeDeclaration: %v", err)
		}
		b.AddAttribute(global)
	})

	own, err := NewAttributeUse(xsderr.Loc{}, false, AttributeDeclarationRef{Name: uq("g")}, &useDefault, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	if vc, ok := s.effectiveValueConstraint(own); !ok || vc.Kind() != ValueDefault || vc.LexicalForm() != "9" {
		t.Fatalf("the use's OWN {value constraint} must win: got %+v ok=%t", vc, ok)
	}

	inherited, err := NewAttributeUse(xsderr.Loc{}, false, AttributeDeclarationRef{Name: uq("g")}, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	if vc, ok := s.effectiveValueConstraint(inherited); !ok || vc.Kind() != ValueFixed || vc.LexicalForm() != "7" {
		t.Fatalf("a use with no own constraint must fall back to the resolved declaration's: got %+v ok=%t", vc, ok)
	}

	bare := dAttr(t, uq("a"), uq("str"))
	if _, ok := s.effectiveValueConstraint(bare); ok {
		t.Fatalf("a use whose declaration has no {value constraint} must be ·absent·")
	}
}

// vcElem builds a global element declaration of the named type carrying vc as
// its {value constraint}.
func vcElem(t *testing.T, local string, typeName QName, vc *ValueConstraint) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, uq(local), TypeDefinitionRef{Name: typeName}, nil,
		NewGlobalScope(), vc, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", local, err)
	}
	return e
}

// TestElementDeclarationSubsumesFixedValues pins loc-testSubP clause 4.2's
// fourth outcome — both {value constraint}s fixed — as a VALUE-space test
// delegated to the installed ValueSpace, and pins that an UNDECIDED verdict
// still accepts. The two lexical forms differ, so a lexical comparison would
// decide where only the value space may.
func TestElementDeclarationSubsumesFixedValues(t *testing.T) {
	gvc := NewValueConstraint(ValueFixed, "7")
	svc := NewValueConstraint(ValueFixed, "07")

	for _, tc := range []struct {
		name string
		vs   *stubValueSpace
		want bool
	}{
		{"decided equal-or-identical subsumes", &stubValueSpace{same: true, decided: true}, true},
		{"decided NOT equal-or-identical does not", &stubValueSpace{same: false, decided: true}, false},
		{"undecided subsumes (fail-open)", &stubValueSpace{same: false, decided: false}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := vcSchema(t, tc.vs, nil)
			if err != nil {
				t.Fatalf("FinalizeWith: %v", err)
			}
			got := s.elementDeclarationSubsumes(vcElem(t, "g", uq("str"), &gvc), vcElem(t, "s", uq("narrow"), &svc))
			if got != tc.want {
				t.Fatalf("elementDeclarationSubsumes = %t, want %t", got, tc.want)
			}
			if tc.vs.calls == 0 {
				t.Fatal("clause 4.2 decided without consulting the ValueSpace")
			}
		})
	}
}

// TestElementDeclarationSubsumesFixedValuesSkipsUnresolvedType pins clause 4.2's
// other fail-open branch: a {type definition} that names no simple type leaves
// the clause with no value space to compare in, so it accepts without consulting
// the ValueSpace at all.
func TestElementDeclarationSubsumesFixedValuesSkipsUnresolvedType(t *testing.T) {
	gvc := NewValueConstraint(ValueFixed, "7")
	svc := NewValueConstraint(ValueFixed, "07")
	vs := &stubValueSpace{same: false, decided: true}
	s, err := vcSchema(t, vs, nil)
	if err != nil {
		t.Fatalf("FinalizeWith: %v", err)
	}
	if !s.fixedValueConstraintSubsumes(vcElem(t, "g", anyTypeName, &gvc), vcElem(t, "s", anyTypeName, &svc)) {
		t.Fatal("a complex {type definition} names no value space, so clause 4.2 must accept")
	}
	if vs.calls != 0 {
		t.Fatalf("the ValueSpace was consulted %d time(s) with no simple type to compare in", vs.calls)
	}
}

// TestAttributeValueConstraintSubsumesFixedValues pins loc-testSubP clause
// 5.2.2 the same way: two fixed ·effective value constraints· are compared in
// the value space, and an undecided verdict accepts.
func TestAttributeValueConstraintSubsumesFixedValues(t *testing.T) {
	gvc := NewValueConstraint(ValueFixed, "7")
	svc := NewValueConstraint(ValueFixed, "07")

	for _, tc := range []struct {
		name    string
		vs      *stubValueSpace
		wantErr bool
	}{
		{"decided equal-or-identical subsumes", &stubValueSpace{same: true, decided: true}, false},
		{"decided NOT equal-or-identical does not", &stubValueSpace{same: false, decided: true}, true},
		{"undecided subsumes (fail-open)", &stubValueSpace{same: false, decided: false}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := vcSchema(t, tc.vs, nil)
			if err != nil {
				t.Fatalf("FinalizeWith: %v", err)
			}
			tt := dType(t, uq("t"), anyTypeName, EmptyContent{}, nil, nil)
			bb := dType(t, uq("b"), anyTypeName, EmptyContent{}, nil, nil)
			err = s.checkBindingSubsumes(uq("a"), tt, bb,
				attributeUseBinding{use: dAttrUse(t, uq("a"), uq("str"), false, &gvc)},
				attributeUseBinding{use: dAttrUse(t, uq("a"), uq("narrow"), false, &svc)})
			if tc.vs.calls == 0 {
				t.Fatal("clause 5.2.2 decided without consulting the ValueSpace")
			}
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("expected ·subsumes·, got %v", err)
				}
				return
			}
			expectRule(t, err, ruleDerivationOKRestriction)
		})
	}
}
