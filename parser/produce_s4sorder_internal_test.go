package parser

import "testing"

// s4sModels is every content model checkS4SChildOrder is charged with, named as
// the tests below report them.
var s4sModels = []struct {
	name  string
	model s4sModel
}{
	{"complexTypeWrapped", s4sComplexTypeWrapped},
	{"complexTypeImplicit", s4sComplexTypeImplicit},
	{"simpleContentWrapper", s4sSimpleContentWrapper},
	{"complexContentWrapper", s4sComplexContentWrapper},
	{"simpleRestriction", s4sSimpleRestriction},
	{"simpleExtension", s4sSimpleExtension},
	{"complexRestriction", s4sComplexRestriction},
	{"complexExtension", s4sComplexExtension},
	{"element", s4sElement},
	{"attribute", s4sAttribute},
	{"simpleType", s4sSimpleType},
}

// s4sProbe is the vocabulary the models above draw on: every element name they
// position, and every facet name the xs:facet substitution group carries. It is
// test data rather than a table the package ships — a name missing from it can
// only weaken the check below, never make it pass wrongly.
var s4sProbe = []string{
	"annotation", "simpleContent", "complexContent", "restriction", "extension",
	"openContent", "group", "all", "choice", "sequence",
	"simpleType", "complexType", "element", "list", "union",
	"alternative", "unique", "key", "keyref",
	"attribute", "attributeGroup", "anyAttribute", "assert",
	"length", "minLength", "maxLength", "pattern", "enumeration", "whiteSpace",
	"maxInclusive", "maxExclusive", "minInclusive", "minExclusive",
	"totalDigits", "fractionDigits", "assertion", "explicitTimezone",
	"maxScale", "minScale",
}

// TestS4SModelPositionsAreDisjoint pins the invariant checkS4SChildOrder's fault
// classification rests on: within one model, no element name is admitted by two
// positions, so the FIRST position admitting a name is the only one and a name
// that no longer matches from the walk's position is unambiguously either a
// repeat of the position it already filled or a return to one behind it. A model
// edit that puts a name in two positions turns that reasoning false silently, and
// fails here instead.
func TestS4SModelPositionsAreDisjoint(t *testing.T) {
	for _, m := range s4sModels {
		t.Run(m.name, func(t *testing.T) {
			for _, local := range s4sProbe {
				var at []int
				for i, slot := range m.model.slots {
					if slot.admits(local) {
						at = append(at, i)
					}
				}
				if len(at) > 1 {
					t.Errorf("%s admits <%s> at positions %v, want at most one", m.name, local, at)
				}
			}
		})
	}
}

// TestS4SFacetElementSeparatesAssertionFromAssert pins the one name pair the
// facet position could swallow: <assertion> is a facet of xs:simpleRestrictionModel
// and <assert> is the {assertions} position that closes the model. Admitting
// <assert> among the facets would put the last position of the model inside a
// repeated one two slots earlier, and every "nothing follows assert*" rejection
// would quietly stop firing.
func TestS4SFacetElementSeparatesAssertionFromAssert(t *testing.T) {
	if !s4sFacetElement("assertion") {
		t.Error("s4sFacetElement rejects <assertion>, which xs:facet's substitution group carries")
	}
	if s4sFacetElement("assert") {
		t.Error("s4sFacetElement admits <assert>, which is the {assertions} position and not a facet")
	}
}
