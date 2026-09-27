package builtin

import "testing"

// TestNCNamePatternIsTheSpecFacet pins NCNamePattern to the value Datatypes
// Appendix E gives xs:NCName's pattern facet (id="NCName.pattern", §3.4.7.1),
// so a lookup that reads the wrong row or the wrong facet fails here rather
// than in a consumer's compiled matcher.
func TestNCNamePatternIsTheSpecFacet(t *testing.T) {
	if got, want := NCNamePattern(), `[\i-[:]][\c-[:]]*`; got != want {
		t.Errorf("NCNamePattern() = %q, want %q", got, want)
	}
}
