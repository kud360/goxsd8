// Package validate assesses instance documents against a compiled schema
// set, over an abstract infoset so XML, JSON, and BER sources plug in as
// adapters.
//
// # The abstract infoset
//
// The engine consumes [Element], [Attribute], [Text] and the [Children]
// cursor over the [Child] sum — never a concrete decoder's types
// (PRINCIPLES 8). They carry the Appendix D properties a cvc- rule reads
// off an information item, plus the Loc a diagnostic cites. Adapters
// construct infoset values and hand them over:
//
//	validate/xmlsrc   XML instances via parser/xmltree      (M5)
//	validate/jsonsrc  JSON instances                        (M8)
//	validate/bersrc   BER-encoded instances                 (M11)
//
// No package of this module in the engine's import closure imports
// encoding/xml, encoding/json, or a BER decoder; only the adapters do.
// (log/slog carries encoding/json into every closure in the module for
// its JSONHandler — the test that pins this boundary says which form of
// the ban each path is held to.)
//
// Assessment is streaming: [Children] is a pull cursor, so a source
// yields one child at a time and the walk never holds a document.
//
// A later infoset property arrives as a NEW optional capability
// interface the engine narrows to — interface{ BaseURI() string },
// falling back to Loc().URI where an adapter does not implement it —
// and NEVER as a method added to Element, Attribute or Text
// (PRINCIPLES 3). Those three are implemented by adapter packages
// outside this module, so a method added to one breaks every adapter at
// once. [[prefix]], [[base URI]] and [[attribute type]] are the
// Appendix D properties waiting on that route: no cvc- rule reads one
// today.
//
// # Assessment semantics designed in from the start
//
//   - Content-model matching is GREEDY and deterministic — UPA makes the
//     model unambiguous, so the matcher never backtracks — and explicit
//     content beats an open-content wildcard at the current state
//     (PRINCIPLES 14). The matcher is xsd's pull walk driver.
//   - Empty content is stricter than element-only: a type whose particle
//     can never match an element admits no character content at all, not
//     even whitespace (PRINCIPLES 13).
//   - Parent element context is threaded through the whole chain: ID
//     harvesting under value constraints, EDC's post-xsi:type governing
//     type, and namespace context for identity constraints all need it.
//   - Identity constraints: node tables propagate UPWARD — a keyref on
//     element E resolves only against key sequences sourced within E's
//     own subtree; selector/field paths honor xpathDefaultNamespace for
//     element steps (PRINCIPLES 15).
//   - Union values validate against DirectMembers in order, with the
//     validating member's whiteSpace driving pattern normalization
//     (PRINCIPLES 11).
//   - Assertions run at every variety level, fail-open per xpath's
//     contract.
//
// # Contract (M5, landing rule by rule)
//
// [Result] carries every violation charged so far as an *xsderr.Error
// (cvc-* rule + instance and/or schema Loc), in document order. Two are
// charged today, both at the ·validation root· and both from
// [Validator.Assess]'s dispatch on the root's ·governing element
// declaration·: cvc-assess-elt (§3.3.4.6) for a root that determines no
// declaration, and cvc-elt (§3.3.4.3) clause 2 for one whose declaration
// is abstract. The rest of the cvc- decisions land on the walk
// [Validator.Assess] already makes. Non-fatal warnings get an accessor of
// their own the day something produces one.
package validate
