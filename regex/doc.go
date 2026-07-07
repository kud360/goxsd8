// Package regex implements XML Schema regular expressions by translating
// them to Go's RE2 — one engine, two flavors.
//
// The F&O (XPath/XQuery Functions and Operators) regex grammar is a
// superset of the XSD pattern-facet grammar, so a single recursive-
// descent parser/translator core serves both; a flavor flag selects the
// semantics (PRINCIPLES 10). The package is a pure leaf: stdlib only.
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
//	FO (fn:matches / fn:replace / fn:tokenize; F&O Appendix F):
//	  - unanchored — the pattern matches a substring unless it anchors
//	    itself;
//	  - ^ and $ are real anchors (\A, \z);
//	  - groups CAPTURE, so fn:replace can reference $1, $2 (with the
//	    longest-valid-group-number rule for multi-digit references);
//	  - flags: i (case-insensitive) and s (dot-all) are honored;
//	    m, x, q, and back-references are not expressible in RE2 and are
//	    ERRORS — surfaced to the caller, never silently accepted or
//	    ignored (a wrong answer is worse than a refusal);
//	  - . excludes only \n by default (it matches \r).
//
// Character-class handling — \d \w \s, \p{…}/\P{…} including Unicode
// blocks (\p{IsBasicLatin}), and class subtraction ([a-z-[m]]) — is
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
