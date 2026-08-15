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
// (cvc-* rule + instance and/or schema Loc), in document order. Six rules
// are charged today, at the ·validation root· and at every descendant whose
// ·governing element declaration· the descent determines.
//
// Two come from [Validator.Assess]'s dispatch on the root's ·governing
// element declaration·: cvc-assess-elt (§3.3.4.6) for a root that
// determines no declaration, and cvc-elt (§3.3.4.3) clause 2 for one whose
// declaration is abstract.
//
// Three more are the root's attribute half, against its ·governing
// type definition·'s {attribute uses}. cvc-complex-type (§3.4.4.2) clauses
// 2 and 3 decide EXISTENCE and need no value space. Clause 4 and the two
// rules clause 2.1 dispatches to — cvc-attribute (§3.2.4.1) clauses 3 and 4
// and cvc-au (§3.5.4) — decide VALUES, and read them through the
// value.Backend [New] takes: an attribute's lexical against its
// declaration's {type definition} per String Valid (§3.16.4), its ·actual
// value· against a fixed {value constraint} on the declaration and on the
// use (two independent rules over two properties, both charged), and a
// ·defaulted attribute·'s own {lexical form} against its type.
//
// The sixth is the root's content half, against the same type's {content
// type}. cvc-complex-type clause 1 decides what its {variety} admits —
// no [[children]] at all for empty, no element ones for simple, no
// non-white-space character ones for element-only — and clause 1.4 sends
// the sequence of element information items to cvc-complex-content
// (§3.4.4.3) over xsd.Schema.ContentMatcher, which charges an item no
// particle admits at its position against that item's own Loc, and a
// sequence ending short of a {min occurs} against the root's. Clause 1.2
// additionally reads a VALUE, through the same backend the attribute charges
// use: a simple {content type} has the root's ·initial value· — every
// character information item [[child]] concatenated in order — validated
// against its {simple type definition} per String Valid, charged against the
// root's own Loc.
//
// Everything not decidable is left undecided rather than guessed at: an
// {attribute wildcard} to evaluate, a ·governing type definition· that is
// not determinable or whose {attribute uses}/{attribute wildcard} are not
// yet the spec's (the attribute half alone: a {content type} needs no such
// fold), an element that may be ·nilled·, a {content type} whose shape
// xsd.Schema.ContentMatcher declines, an element with no character content for
// cvc-elt clause 5.1 to have supplied a default in place of, a declaration
// whose {type definition} is not a simple type, and — the decline that matters
// most — a value.ValidateLexical error that is a fault of the type or of the
// backend rather than a verdict about the lexical (value.IsDatatypeVerdict),
// which is what keeps a typeless attribute (xs:anySimpleType, §3.2.2.2), or
// simple content of a type this backend does not map, from being rejected by
// every document that carries one.
//
// Every one of those charges reaches a DESCENDANT on the same terms, against
// the ·governing type definition· the particle its parent's {content type}
// ·attributes· it to supplies (§3.3.4.6 clause 3.1): an element particle's
// {term}, or — for a strict or lax wildcard particle, and for an item admitted
// as a member of a ·substitution group· — the top-level declaration its
// ·expanded name· ·resolves· to. Two shapes stop it: a child ·attributed to· a
// skip wildcard, which is ·skipped· along with every element beneath it (clause
// 3.2), and a child whose declaration is not determinable, whose own subtree is
// then assessed against nothing in its turn.
//
// The rest of the cvc- decisions land on the walk [Validator.Assess]
// already makes. Non-fatal warnings get an accessor of their own the day
// something produces one.
package validate
