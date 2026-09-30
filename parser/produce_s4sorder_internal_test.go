package parser

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

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
	{"alternative", s4sAlternative},
	{"keybase", s4sKeybase},
	{"selector", s4sSelector},
	{"field", s4sField},
	{"namedGroup", s4sNamedGroup},
	{"namedAttributeGroup", s4sNamedAttributeGroup},
	{"list", s4sList},
	{"simpleTypeRestriction", s4sSimpleTypeRestriction},
	{"openContent", s4sOpenContent},
	{"defaultOpenContent", s4sDefaultOpenContent},
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
	"totalDigits", "fractionDigits", "assertion", "assertions", "explicitTimezone",
	"maxScale", "minScale", "selector", "field", "any",
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

// s4sModelName is one element name as a model's quoted content model spells it;
// s4sModelWildcard is the "{any with namespace: ##other}" wildcard term, whose
// words name no element.
var (
	s4sModelName     = regexp.MustCompile(`[A-Za-z]+`)
	s4sModelWildcard = regexp.MustCompile(`\{[^}]*\}`)
)

// s4sVocabulary is every element name any model positions: each name its quoted
// content model spells, and each s4sProbe name one of its slots admits — the
// second reaches the facet names a model's quotation can omit (<explicitTimezone>
// under s4sSimpleRestriction) or never spell (<assertions>).
func s4sVocabulary() []string {
	var names []string
	for _, m := range s4sModels {
		names = append(names, s4sModelName.FindAllString(s4sModelWildcard.ReplaceAllString(m.model.model, ""), -1)...)
		for _, local := range s4sProbe {
			if s4sSlotAt(m.model.slots, 0, local) >= 0 {
				names = append(names, local)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// TestS4SVowelNamesHaveArticles pins #1098 over the whole model vocabulary: every
// name a model positions that opens with a vowel LETTER has its article decided
// by s4sVowelArticles, since the letter alone does not decide it (<union>,
// <unique>). A model added later with a new vowel-letter name fails here until the
// table decides it, rather than printing "a <…>" by default. The table carries no
// row for a name no model positions.
func TestS4SVowelNamesHaveArticles(t *testing.T) {
	vocabulary := s4sVocabulary()
	for _, local := range vocabulary {
		if !strings.ContainsRune("aeiouAEIOU", rune(local[0])) {
			continue
		}
		if !slices.ContainsFunc(s4sVowelArticles, func(e struct{ local, article string }) bool { return e.local == local }) {
			t.Errorf("<%s> is positioned by a model and opens with a vowel letter, but s4sVowelArticles does not decide its article", local)
		}
	}
	for _, e := range s4sVowelArticles {
		if !slices.Contains(vocabulary, e.local) {
			t.Errorf("s4sVowelArticles decides <%s>, which no model positions", e.local)
		}
		if got := s4sArticle(e.local); got != e.article {
			t.Errorf("s4sArticle(%q) = %q, want the table's %q", e.local, got, e.article)
		}
	}
}
