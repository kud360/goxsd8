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

// seededIndex seeds the builtin datatypes from the strict backend and indexes
// them by local name, so a test can restrict a real builtin.
func seededIndex(t *testing.T) (map[string]*xsd.SimpleType, value.Backend) {
	t.Helper()
	backend := strict.New()
	types, err := builtin.Seed(backend)
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	idx := make(map[string]*xsd.SimpleType, len(types))
	for _, st := range types {
		idx[st.Name().Local] = st
	}
	return idx, backend
}

// restrictBuiltin builds a restriction of the named builtin carrying ownFacets
// and runs the whole cos-st-restricts entry point over it.
func restrictBuiltin(t *testing.T, local string, ownFacets ...xsd.Facet) error {
	t.Helper()
	idx, backend := seededIndex(t)
	base, ok := idx[local]
	if !ok {
		t.Fatalf("builtin xs:%s not seeded", local)
	}
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "derived"},
		base.Variety(), base, ownFacets, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	return builtin.CheckSimpleTypeRestriction(backend, st)
}

// TestCheckSimpleTypeRestrictionApplicability covers cos-st-restricts clause
// 1.3.1 against the GENERATED per-primitive table: a facet the primitive's row
// lists is accepted, one it does not is rejected under cos-st-restricts.
func TestCheckSimpleTypeRestrictionApplicability(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		facet    xsd.Facet
		rejected bool
	}{
		{"maxLength on string", "string", xsd.NewFacet(xsd.FacetMaxLength, []string{"4"}, false), false},
		{"maxInclusive on string", "string", xsd.NewFacet(xsd.FacetMaxInclusive, []string{"4"}, false), true},
		{"totalDigits on decimal", "decimal", xsd.NewFacet(xsd.FacetTotalDigits, []string{"4"}, false), false},
		{"length on decimal", "decimal", xsd.NewFacet(xsd.FacetLength, []string{"4"}, false), true},
		{"fractionDigits on precisionDecimal", "precisionDecimal", xsd.NewFacet(xsd.FacetFractionDigits, []string{"1"}, false), true},
		{"maxScale on precisionDecimal", "precisionDecimal", xsd.NewFacet(xsd.FacetMaxScale, []string{"3"}, true), false},
		{"explicitTimezone on dateTime", "dateTime", xsd.NewFacet(xsd.FacetExplicitTimezone, []string{"required"}, false), false},
		{"explicitTimezone on string", "string", xsd.NewFacet(xsd.FacetExplicitTimezone, []string{"required"}, false), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := restrictBuiltin(t, c.base, c.facet)
			if !c.rejected {
				if err != nil {
					t.Fatalf("applicable facet rejected: %v", err)
				}
				return
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

// TestCheckSimpleTypeRestrictionInheritedFacetsApplicable guards against a
// false-reject: a restriction's {facets} include everything inherited from the
// builtin base, and every one of those must still be applicable. xs:integer
// carries an inherited fractionDigits and pattern from the decimal chain.
func TestCheckSimpleTypeRestrictionInheritedFacetsApplicable(t *testing.T) {
	if err := restrictBuiltin(t, "integer", xsd.NewFacet(xsd.FacetTotalDigits, []string{"3"}, false)); err != nil {
		t.Fatalf("restriction of xs:integer rejected: %v", err)
	}
}

// TestCheckSimpleTypeRestrictionDelegatesToValue proves the entry point does not
// stop at applicability: a facet that IS applicable but widens the base's value
// space is still rejected, under the bound facet's own rule from package value.
func TestCheckSimpleTypeRestrictionDelegatesToValue(t *testing.T) {
	// xs:byte's value space is bounded [-128, 127] by inherited minInclusive /
	// maxInclusive facets, so a maxInclusive of 200 widens it.
	err := restrictBuiltin(t, "byte", xsd.NewFacet(xsd.FacetMaxInclusive, []string{"200"}, false))
	rule, ok := xsderr.RuleOf(err)
	if !ok || rule != "maxInclusive-valid-restriction" {
		t.Fatalf("rule = %q (ok=%v), want maxInclusive-valid-restriction; err=%v", rule, ok, err)
	}
	if err := restrictBuiltin(t, "byte", xsd.NewFacet(xsd.FacetMaxInclusive, []string{"100"}, false)); err != nil {
		t.Fatalf("narrowing maxInclusive on xs:byte rejected: %v", err)
	}
}

// TestCheckSimpleTypeRestrictionSeededBuiltins is the regression guard for the
// whole seam: every builtin datatype Seed produces must itself pass the check it
// gates other types with, or the entry point is mis-specified rather than the
// schemas it rejects being invalid.
func TestCheckSimpleTypeRestrictionSeededBuiltins(t *testing.T) {
	idx, backend := seededIndex(t)
	for _, name := range builtinNames(idx) {
		if err := builtin.CheckSimpleTypeRestriction(backend, idx[name]); err != nil {
			t.Errorf("seeded builtin xs:%s fails CheckSimpleTypeRestriction: %v", name, err)
		}
	}
}

// builtinNames returns the seeded local names in Types order (plus
// anySimpleType), so the loop above reports deterministically.
func builtinNames(idx map[string]*xsd.SimpleType) []string {
	names := make([]string, 0, len(idx))
	if _, ok := idx["anySimpleType"]; ok {
		names = append(names, "anySimpleType")
	}
	for i := range builtin.Types {
		if _, ok := idx[builtin.Types[i].Name]; ok {
			names = append(names, builtin.Types[i].Name)
		}
	}
	return names
}

// TestStringAcceptsEveryApplicableFacet walks the generated xs:string row and
// restricts xs:string with each facet that row declares applicable. It is the
// mutation guard on the FacetKind→FacetName bridge: a bridge case dropped for
// any of these kinds would read as "not applicable" and false-reject the very
// facet the spec table says applies.
func TestStringAcceptsEveryApplicableFacet(t *testing.T) {
	spec, ok := typeSpecByName("string")
	if !ok {
		t.Fatal("no generated row for xs:string")
	}
	for _, f := range spec.Facets {
		if f.Name == "assertions" {
			continue // {value} is a sequence of Assertion components, not lexical
		}
		var facet xsd.Facet
		if f.Name == "enumeration" {
			facet = xsd.NewEnumerationFacet([]xsd.EnumerationMember{xsd.NewEnumerationMember("a", nil, nil)})
		} else {
			facet = xsd.NewFacet(kindOf(t, f.Name), []string{stringFacetValue(f.Name)}, false)
		}
		if err := restrictBuiltin(t, "string", facet); err != nil {
			t.Errorf("xs:string restriction with its own applicable facet %s rejected: %v", f.Name, err)
		}
	}
}

// typeSpecByName finds a generated row by name through the exported table.
func typeSpecByName(name string) (builtin.TypeSpec, bool) {
	for i := range builtin.Types {
		if builtin.Types[i].Name == name {
			return builtin.Types[i], true
		}
	}
	return builtin.TypeSpec{}, false
}

// kindOf maps a spec facet name to its xsd.FacetKind for the table-driven test.
func kindOf(t *testing.T, name builtin.FacetName) xsd.FacetKind {
	t.Helper()
	for k := xsd.FacetLength; k <= xsd.FacetMinScale; k++ {
		if k.String() == string(name) {
			return k
		}
	}
	t.Fatalf("no xsd.FacetKind for facet %q", name)
	return 0
}

// stringFacetValue returns a legal {value} for a facet applicable to xs:string.
func stringFacetValue(name builtin.FacetName) string {
	switch name {
	case "whiteSpace":
		return "collapse"
	case "pattern":
		return "a*"
	default:
		return "1"
	}
}
