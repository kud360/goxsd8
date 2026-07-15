// Package regex implements XML Schema regular expressions by translating
// them to Go's RE2 — one engine, two flavors.
//
// The F&O (XPath/XQuery Functions and Operators) regex grammar is a
// superset of the XSD pattern-facet grammar, so a single recursive-
// descent parser/translator core serves both; a flavor flag selects the
// semantics (PRINCIPLES 10). The package sits just above the leaves: it
// imports only xsderr (so its FORX0001/FORX0002/src-pattern-value failures
// are rule-tagged per STYLE T2), otherwise stdlib.
//
// # Flavors
//
//	XSD (pattern facets; Datatypes Appendix G):
//	  - the whole pattern is implicitly anchored (\A…\z);
//	  - ^ and $ are LITERAL characters;
//	  - groups are non-capturing;
//	  - no flags;
//	  - . excludes both \n and \r.
//
//	FO (fn:matches / fn:replace / fn:tokenize; F&O §7.6.1):
//	  - unanchored — the pattern matches a substring unless it anchors
//	    itself;
//	  - ^ and $ are real anchors;
//	  - groups CAPTURE in opening-parenthesis order (so fn:replace can
//	    reference $1, $2; the longest-valid-group-number resolution of a
//	    multi-digit $N is the replacement caller's concern, not Translate's);
//	  - flags: i (case-insensitive), s (dot-all), and m (multi-line) map to
//	    the RE2 inline flags (?i)/(?s)/(?m); x (extended) strips insignificant
//	    whitespace before parsing. Any other flag character — including q,
//	    which this local F&O edition does not define — is an err:FORX0001
//	    error;
//	  - back-references (\N) are legal F&O grammar but have no RE2 form, so
//	    they are an err:FORX0002 error — surfaced, never silently accepted (a
//	    wrong answer is worse than a refusal);
//	  - . excludes only \n by default (it matches \r); the s flag lifts that.
//
// Character-class handling — \d \w \s, the XML name-character escapes
// \i \c (NameStartChar/NameChar, §G.4.2.5) and their complements \I \C,
// \p{…}/\P{…} including Unicode blocks (\p{IsBasicLatin}), and class
// subtraction ([a-z-[m]], including name-escape bases like [\i-[:]]) — is
// shared between flavors. Go RE2's counted-repeat limit (1000) is a
// documented deviation surfaced as a translation error, not a silent
// truncation.
//
// # Contract (implemented from M3)
//
//	func Translate(pattern string, flavor Flavor, flags string) (goRE string, err error)
//	    Deterministic; errors carry the offending construct and offset.
//	    Compilation of the result is the caller's concern (callers cache
//	    compiled patterns alongside the facet/function that owns them).
//
// Callers: the pattern facet uses flavor XSD; xpath's fn:matches/
// fn:replace/fn:tokenize use flavor FO. Never cross them.
package regex
