package builtin

// NCNamePattern returns the value of xs:NCName's pattern facet as the generated
// Types row carries it (Datatypes §3.4.7.1), so a consumer checking the NCName
// production compiles the spec's own string and never a hand-typed copy of it.
//
// The string is XSD regular-expression syntax (Datatypes §G), unanchored, and
// meant for regex.Translate. With regex.FlavorXSD the translated pattern matches
// only the whole string; a regex.FlavorFO prefix scan must prepend "^" itself.
//
// Its consumers are the parser's declaration-name check and builtin/strict's
// QName lexical check. xpath and icpath keep a local "^"-anchored FO copy and
// pin it to this value in a test.
//
// It scans Types on each call rather than caching: both consumers call it once,
// at package initialization (STYLE D3). It panics if the row or its pattern
// facet is missing, which means tools/typespecgen emitted a table without them.
func NCNamePattern() string {
	spec, ok := typeSpecOf(qname("NCName"))
	if !ok {
		panic("builtin: generated Types has no xs:NCName row")
	}
	for _, f := range spec.Facets {
		if f.Name == "pattern" {
			return f.Default
		}
	}
	panic("builtin: generated xs:NCName row carries no pattern facet")
}
