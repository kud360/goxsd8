// Package validate assesses instance documents against a compiled schema
// set, over an abstract infoset so XML, JSON, and BER sources plug in as
// adapters.
//
// # The abstract infoset
//
// The engine consumes marker interfaces — element, attribute, and node
// views with name, character content, namespace lookup, and Loc — never
// a concrete decoder's types (PRINCIPLES 8). Adapters construct infoset
// values and hand them over:
//
//	validate/xmlsrc   XML instances via parser/xmltree      (M5)
//	validate/jsonsrc  JSON instances                        (M8)
//	validate/bersrc   BER-encoded instances                 (M11)
//
// The engine imports none of encoding/xml, encoding/json, or a BER
// decoder; only the adapters do.
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
// # Planned contract (M5 — not yet implemented)
//
//	func New(set *xsd.SchemaSet, opts ...Option) (*Validator, error)
//	    Options: WithLogger. The Validator is immutable and reusable.
//
//	func (v *Validator) Assess(root Element) *Result
//	    Result carries every violation as an *xsderr.Error (cvc-* rule +
//	    instance and/or schema Loc), in document order, plus non-fatal
//	    warnings. Streaming-oriented: assessment walks the source once.
//
// The Validator exposes a minimal read-only schema view (STYLE T3) so
// adapters can resolve root element declarations without reaching into
// compiled internals.
package validate
