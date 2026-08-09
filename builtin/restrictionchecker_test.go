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
// holds must still be charged the facet-VALUE half of cos-st-restricts — clause
// 1.3.1's atomic applicability, and clause 1.3.2's value-space constraints
// (§4.3.7.4–§4.3.10.4) — when the schema is finalized with a checker installed.
//
// It is deliberately built on the SchemaBuilder/FinalizeWith lane rather than by
// parsing a schema document. On the parser lane the same fixture would prove
// nothing about the descent: the anonymous type reaches the checker through
// whichever component holds it, and a parsed fixture rejects identically whether
// the charge happens at construction or at finalize. Here the ONLY thing that can
// reach the type is checkSimpleTypeDerivations' walk.
//
// Each case places the offending type in a slot an s.types-only walk cannot
// reach, and each proves that twice over: the control run finalizes the same
// shape with a CONFORMING facet and asserts that the anonymous type is a member
// of no {type definitions}, and the rejected run's message must name a facet
// {value} that only the anonymous type carries. A walk over s.types alone
// therefore accepts every case below.
//
// Being unindexed is necessary but NOT sufficient for a case to discriminate,
// which is why the two cases carry DIFFERENT faults. Clause 1.3.1 reads
// {facets}, which EffectiveFacets accumulates over the whole {base type
// definition} chain, so an inapplicable facet on a base is equally visible on
// every type derived from it: an applicability fault discriminates only where no
// type on the chain is indexed, i.e. the inline-declaration case. The base-hop
// case therefore faults the anonymous base's OWN facets instead — the value-space
// half reads OwnFacets and never the inherited ones (value.CheckFacetRestriction)
// — so the named type derived from it is charged CLEAN, and the base hop is the
// only way the fault is reached at all.
func TestRestrictionCheckerRejectsAnonymousInlineSimpleType(t *testing.T) {
	for _, tc := range []struct {
		slot string
		// add places a schema holding the anonymous type into the builder.
		add func(t *testing.T, b *xsd.SchemaBuilder, anon *xsd.SimpleType)
		// base is the local name of the builtin the anonymous type restricts,
		// and bad/good the facet it carries in the rejected and control runs.
		base      string
		bad, good xsd.Facet
		// wantRule is the rule ID the rejected run must carry, and wantMessage a
		// substring of its message that only the anonymous type can explain.
		wantRule    xsderr.Rule
		wantMessage string
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
			// maxInclusive is not applicable to xs:string (cos-applicable-facets
			// §4.1.5), so clause 1.3.1 rejects; maxLength is, so the control run
			// finalizes. Nothing but the inline type is unindexed here, so the
			// accumulating {facets} fault is reached only through slot 6.
			base:        "string",
			bad:         xsd.NewFacet(xsd.FacetMaxInclusive, []string{"4"}, false),
			good:        xsd.NewFacet(xsd.FacetMaxLength, []string{"4"}, false),
			wantRule:    "cos-st-restricts",
			wantMessage: "1.3.1",
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
			// derived IS indexed, so this case discriminates only because the
			// fault is invisible from it: maxInclusive="1000" widens xs:byte's
			// own maxInclusive="127" (maxInclusive valid restriction, §4.3.7.4),
			// a constraint charged against the OWN facets of whichever type is
			// being checked. derived declares none, and it inherits a facet that
			// was already charged where it was declared, so charging derived
			// alone accepts. "1000" appears in no other component.
			base:        "byte",
			bad:         xsd.NewFacet(xsd.FacetMaxInclusive, []string{"1000"}, false),
			good:        xsd.NewFacet(xsd.FacetMaxInclusive, []string{"100"}, false),
			wantRule:    "maxInclusive-valid-restriction",
			wantMessage: `"1000"`,
		},
	} {
		t.Run(tc.slot, func(t *testing.T) {
			s, backend := seededBuilder(t, tc.base, tc.add, tc.good)
			for _, td := range s.Types() {
				if st, ok := td.(*xsd.SimpleType); ok && st.Name() == (xsd.QName{}) {
					t.Fatal("the anonymous type must not be a member of {type definitions}, or an s.types-only walk would reach it")
				}
			}

			_, err := finalizeSeeded(t, backend, tc.base, tc.add, tc.bad)
			if err == nil {
				t.Fatalf("a faulty facet on an anonymous simple type in %s was accepted", tc.slot)
			}
			rule, ok := xsderr.RuleOf(err)
			if !ok || rule != tc.wantRule {
				t.Fatalf("rule = %q (ok=%v), want %q; err=%v", rule, ok, tc.wantRule, err)
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("message %q does not contain %s", err.Error(), tc.wantMessage)
			}
		})
	}
}

// seededBuilder is finalizeSeeded with the finalize expected to succeed; it
// returns the finalized schema and the backend both runs share.
func seededBuilder(t *testing.T, base string, add func(*testing.T, *xsd.SchemaBuilder, *xsd.SimpleType), facet xsd.Facet) (*xsd.Schema, value.Backend) {
	t.Helper()
	backend := strict.New()
	s, err := finalizeSeeded(t, backend, base, add, facet)
	if err != nil {
		t.Fatalf("the control fixture must finalize: %v", err)
	}
	return s, backend
}

// finalizeSeeded seeds the builtin datatypes into a fresh builder, hands add an
// ANONYMOUS restriction of the builtin whose local name is base, carrying facet,
// and finalizes with both capabilities installed.
func finalizeSeeded(t *testing.T, backend value.Backend, base string, add func(*testing.T, *xsd.SchemaBuilder, *xsd.SimpleType), facet xsd.Facet) (*xsd.Schema, error) {
	t.Helper()
	types, err := builtin.Seed(backend)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	var baseType *xsd.SimpleType
	for _, bt := range types {
		b.AddType(bt)
		if bt.Name().Local == base {
			baseType = bt
		}
	}
	if baseType == nil {
		t.Fatalf("xs:%s not seeded", base)
	}
	anon, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{}, xsd.RestrictionDerivation{}, baseType, []xsd.Facet{facet}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(anonymous): %v", err)
	}
	add(t, b, anon)
	return b.FinalizeWith(value.NewValueSpace(backend), builtin.NewRestrictionChecker(backend))
}
