// Package xpath is the XPath 2.0 engine serving conditional type
// assignment (CTA), assertions, and identity-constraint paths. Full
// XPath 2.0 is the destination; the engine grows outward from the
// XSD-required subset, tracked by its own conformance lane.
//
// # Growth tiers
//
//  1. The CTA restricted subset (the `test` attribute of
//     xs:alternative) — M6.
//  2. Assertion essentials: axes, predicates, quantified expressions,
//     typed comparisons, the F&O function core — M6.
//  3. The full grammar (docs/specs/md/xpath20.md) and function library
//     (docs/specs/md/xpath-functions.md) — M7 onward, ratcheted.
//
// # One parser, one AST
//
// One lexer and one recursive-descent parser build one AST that serves
// both static analysis (schema-time: syntax errors, type references)
// and evaluation (instance-time). There is never a second, lenient
// parser (STYLE T4).
//
// # The fail-open contract (PRINCIPLES 20)
//
// An unsupported construct must NEVER cause a false rejection: an
// assertion whose expression falls outside the implemented subset
// evaluates as satisfied; an alternative's test as unmatched. Every
// fail-open site carries a greppable marker:
//
//	// GAP(xpath): <construct>
//
// Direction matters: a DYNAMIC error — type mismatch, uncastable value,
// bad or inexpressible regex/flag — makes the assertion definitively
// UNSATISFIED (a real false), not fail-open. Confusing the two flips
// false-accepts into false-rejects or vice versa.
//
// # Static context
//
//   - $value binds a typed atom {Lexical, Kind}, not a bare string
//     (PRINCIPLES 17).
//   - xpathDefaultNamespace supplies the default ELEMENT namespace for
//     unprefixed element steps (never attribute steps) in assertions
//     and IDC selector/field paths (PRINCIPLES 15).
//   - fn:matches / fn:replace / fn:tokenize bind to regex flavor FO,
//     never the pattern-facet flavor.
//
// Numbers follow the XDM model the subset needs; comparisons over typed
// atoms delegate to value capabilities so backend values participate.
package xpath
