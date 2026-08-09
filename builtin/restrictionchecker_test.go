package builtin_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestRestrictionCheckerRejectsAnonymousInlineSimpleType is the acceptance test
// for the capability seam: an ANONYMOUS simple type that no §3.17.1 symbol table
// holds must still be charged cos-st-restricts clause 1.3.1 when the schema is
// finalized with a checker installed.
//
// It is deliberately built on the SchemaBuilder/FinalizeWith lane rather than by
// parsing a schema document. On the parser lane the same fixture would prove
// nothing about the descent: the anonymous type reaches the checker through
// whichever component holds it, and a parsed fixture rejects identically whether
// the charge happens at construction or at finalize. Here the ONLY thing that can
// reach the type is checkSimpleTypeDerivations' walk.
//
// Each case places the offending type in a slot an s.types-only walk cannot
// reach, and proves it: the control run finalizes the same shape with an
// APPLICABLE facet and asserts that the type appears nowhere in the finalized
// schema's {type definitions}. A walk over that property alone therefore accepts
// every case below.
func TestRestrictionCheckerRejectsAnonymousInlineSimpleType(t *testing.T) {
	for _, tc := range []struct {
		slot string
		// add places a schema holding the anonymous type into the builder.
		add func(t *testing.T, b *xsd.SchemaBuilder, anon *xsd.SimpleType)
	}{
		{
			slot: "an element declaration's inline {type definition} (inventory slot 6)",
			add: func(t *testing.T, b *xsd.SchemaBuilder, anon *xsd.SimpleType) {
				t.Helper()
				e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "e"},
					xsd.InlineTypeDefinition{Definition: anon}, nil, xsd.NewGlobalScope(),
					nil, false, nil, nil, nil, false, nil, nil)
				if err != nil {
					t.Fatalf("NewElementDeclaration: %v", err)
				}
				b.AddElement(e)
			},
		},
		{
			slot: "the anonymous {base type definition} of a named type (inventory slot 2)",
			add: func(t *testing.T, b *xsd.SchemaBuilder, anon *xsd.SimpleType) {
				t.Helper()
				derived, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "derived"},
					xsd.RestrictionDerivation{}, anon, nil, nil)
				if err != nil {
					t.Fatalf("NewSimpleType(derived): %v", err)
				}
				b.AddType(derived)
			},
		},
	} {
		t.Run(tc.slot, func(t *testing.T) {
			// maxInclusive is not applicable to xs:string (cos-applicable-facets
			// §4.1.5), so clause 1.3.1 rejects; maxLength is, so the control run
			// finalizes.
			bad := xsd.NewFacet(xsd.FacetMaxInclusive, []string{"4"}, false)
			good := xsd.NewFacet(xsd.FacetMaxLength, []string{"4"}, false)

			s, backend := seededBuilder(t, tc.add, good)
			for _, td := range s.Types() {
				if st, ok := td.(*xsd.SimpleType); ok && st.Name() == (xsd.QName{}) {
					t.Fatal("the anonymous type must not be a member of {type definitions}, or an s.types-only walk would reach it")
				}
			}

			_, err := finalizeSeeded(t, backend, tc.add, bad)
			if err == nil {
				t.Fatalf("an inapplicable facet on an anonymous simple type in %s was accepted", tc.slot)
			}
			rule, ok := xsderr.RuleOf(err)
			if !ok || rule != "cos-st-restricts" {
				t.Fatalf("rule = %q (ok=%v), want cos-st-restricts; err=%v", rule, ok, err)
			}
			if !strings.Contains(err.Error(), "1.3.1") {
				t.Errorf("message %q does not name clause 1.3.1", err.Error())
			}
		})
	}
}

// seededBuilder is finalizeSeeded with the finalize expected to succeed; it
// returns the finalized schema and the backend both runs share.
func seededBuilder(t *testing.T, add func(*testing.T, *xsd.SchemaBuilder, *xsd.SimpleType), facet xsd.Facet) (*xsd.Schema, value.Backend) {
	t.Helper()
	backend := strict.New()
	s, err := finalizeSeeded(t, backend, add, facet)
	if err != nil {
		t.Fatalf("the control fixture must finalize: %v", err)
	}
	return s, backend
}

// finalizeSeeded seeds the builtin datatypes into a fresh builder, hands add an
// ANONYMOUS restriction of xs:string carrying facet, and finalizes with both
// capabilities installed.
func finalizeSeeded(t *testing.T, backend value.Backend, add func(*testing.T, *xsd.SchemaBuilder, *xsd.SimpleType), facet xsd.Facet) (*xsd.Schema, error) {
	t.Helper()
	types, err := builtin.Seed(backend)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	var str *xsd.SimpleType
	for _, bt := range types {
		b.AddType(bt)
		if bt.Name().Local == "string" {
			str = bt
		}
	}
	if str == nil {
		t.Fatal("xs:string not seeded")
	}
	anon, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{}, xsd.RestrictionDerivation{}, str, []xsd.Facet{facet}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(anonymous): %v", err)
	}
	add(t, b, anon)
	return b.FinalizeWith(value.NewValueSpace(backend), builtin.NewRestrictionChecker(backend))
}
