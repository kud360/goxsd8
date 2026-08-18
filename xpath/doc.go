// Package xpath is the XPath 2.0 engine serving conditional type
// assignment (CTA), assertions, and identity-constraint paths. Full
// XPath 2.0 is the destination; the engine grows outward from the
// XSD-required subset, tracked by its own conformance lane.
//
// # Growth tiers
//
//  1. The CTA restricted subset (the `test` attribute of
//     xs:alternative) — M6. SHIPPED: CompileCTATest parses Structures
//     §3.12.6's productions [8] ta-Test through [18]
//     ta-ConstructorFunction and CTATest.Evaluate decides one against
//     an element's attributes, casts included: [15] ta-CastExpr's
//     `cast as` tail and [18] ta-ConstructorFunction are one node,
//     evaluated through value's facet pipeline, which is what
//     xpath-functions.md §17.1.1 makes a cast. ONE shape inside that
//     grammar compile-time-DECLINES rather than evaluating — a
//     wildcard NameTest (#859), declined lexically before any
//     production sees it — and so does a cast whose TARGET is not a
//     builtin datatype, which is the required subset's own boundary
//     (§3.12.6 clause 4) rather than a construct of the grammar.
//  2. Assertion essentials: axes, predicates, quantified expressions,
//     typed comparisons, the F&O function core — M6. PLANNED; nothing
//     of it is exported.
//  3. The full grammar (docs/specs/md/xpath20.md) and function library
//     (docs/specs/md/xpath-functions.md) — M7 onward, ratcheted.
//     PLANNED.
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
// An unsupported construct must NEVER cause a false rejection, and each
// consumer takes that in the direction its own rule allows. A CTA
// {test} the implemented subset cannot evaluate is DECLINED at compile
// time, and the caller withholds the element's ·governing type
// definition· rather than assessing it against a guess — never
// "unmatched", which would fall through to another alternative or to
// the {default type definition} and select a type the rule may not have
// selected. An assertion whose expression falls outside the implemented
// subset evaluates as satisfied. Every fail-open site carries a
// greppable marker:
//
//	// GAP(xpath): <construct>
//
// Direction matters, and the two directions are not the same rule. A
// DYNAMIC error — type mismatch, uncastable value, bad or inexpressible
// regex/flag — is a real verdict, not a decline: it makes an assertion
// definitively UNSATISFIED, and it makes a CTA {test} FALSE, which
// Structures §3.12.4 key-cta-ta-select clause 2 states outright ("the
// {test} is treated as if it had evaluated (without error) to false").
// Confusing that with a decline flips false-accepts into false-rejects
// or vice versa.
//
// # Static context
//
//   - $value binds a typed atom {Lexical, Kind}, not a bare string
//     (PRINCIPLES 17). It is an ASSERTION binding: ta-props-correct
//     adds no variable, so no CTA {test} sees one.
//   - xpathDefaultNamespace supplies the default ELEMENT/TYPE namespace
//     for unprefixed element steps (never attribute steps) in
//     assertions and IDC selector/field paths (PRINCIPLES 15), and for
//     an unprefixed cast TARGET (xpath20.md §3.10.2). The CTA subset
//     reaches no element step, so a cast target is the only thing it
//     consults the default namespace for.
//   - fn:matches / fn:replace / fn:tokenize bind to regex flavor FO,
//     never the pattern-facet flavor.
//
// Numbers follow the XDM model the subset needs; comparisons over typed
// atoms delegate to value capabilities so backend values participate.
package xpath
