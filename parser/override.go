package parser

import (
	"strings"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file implements ·override pre-processing· (§4.2.5, transformation §F.2):
// the rewriting an <xs:override> element applies to the schema document it points
// at, before that document's contents are mapped to components.
//
// §F.2 states the rewrite as a stylesheet over D2, the overridden document, given
// O1, the overriding <override> element. For each element child E2 of a <schema>
// (or <redefine>) in D2:
//
//	clause 1  E2 is a named source declaration — <simpleType>, <complexType>,
//	          <group>, <attributeGroup>, <element>, <attribute>, <notation> — and
//	          O1 has a child of the SAME element type with the SAME name: D2′ has
//	          a copy of O1's child in E2's place.
//	clause 2  no such child of O1: D2′ has a copy of E2.
//	clause 3  E2 is an <include>: D2′ has an <override> pointing at the same
//	          schemaLocation with copies of O1's children — the override CASCADES
//	          into every included document, which is what makes the ·target set·
//	          of §4.2.5 transitive over inclusion.
//	clause 4  E2 is an <override>: its children are merged with O1's — O1 wins on
//	          a match (4.1), E2's unmatched children are kept in place (4.2), and
//	          O1's unmatched children are appended (4.3).
//	clause 5  anything else — notably <import> — is copied unchanged, which is
//	          what stops the ·target set· at a namespace boundary.
//
// NO TREE IS REWRITTEN HERE. The parser holds each document as an immutable raw
// tree read once (parser/document.go), so the transformation is carried as DATA —
// an overrideSet threaded through discovery beside the effective target namespace
// — and applied as a lookup at the two places that walk a document's top level
// (producer.prescan and producer.run). That yields the same components §F.2
// prescribes with no copying and no synthesized nodes, and it keeps the
// document-level defaults right by construction: a substituted declaration is
// produced by the OVERRIDDEN document's producer, so §4.2.5's note that
// elementFormDefault and friends "are applied not in the context of the document
// containing the <override> (Dnew) but in the context of the document containing
// the original overridden declaration (Dold)" holds without a special case
// (PRINCIPLES 16).
//
// §F.2's governing sentence scopes the case analysis to "each element
// information item E2 in the [children] of any <schema> OR <redefine> element
// information item within D2", and since #286 that second half is live: a
// <redefine> IS followed, so an overridden document's inline <redefine> children
// are candidate E2s and clause 1 substitutes for them exactly as it does for a
// <schema> child. That substitution is applied where the redefine's children are
// read — producer.prescanRedefine and producer.produceRedefine — through the same
// [overrideSet.replacement] lookup used at a document's top level. Only four of
// clause 1's seven element types can occur there, since <redefine>'s own content
// model (§4.2.4) admits no <element>, <attribute> or <notation>.
//
// This is distinct from, and not in tension with, §4.2.5's ·target set·
// (key-targetset), which governs which further DOCUMENTS an <override> reaches
// and which "does not include schema documents which are pointed to by <import>
// or <redefine> elements": the override does not cascade into the document a
// <redefine> names (parse.go's assembly.redefine passes no override on), only
// into the redefinition items written inline in the document it already reaches.
//
// GAP(parser): a redefining declaration REPLACED under clause 1 loses its
// self-reference resolution (#286). src-expredef pairs a redefining
// <simpleType>/<group>/<attributeGroup> with the original it replaces by walking
// up from the reference to the declaration that contains it, and a substituted
// declaration is written under the <override>, not under the <redefine>, so the
// walk does not find it and the self-reference resolves to the visible
// redefinition instead — reported as a circular derivation or a circular group.
// It OVER-rejects: an override-of-a-redefine-child assembly can be lost, never
// wrongly accepted. Nothing else about the substitution is affected.

// componentKey identifies an overridable source declaration by exactly the pair
// §F.2 clause 1 matches on: the ELEMENT TYPE (the local name of the declaration
// element, always in the XSD namespace) and the ·actual value· of its name
// attribute. Two declarations of different kinds never override one another even
// when they share a name, mirroring sch-props-correct clause 2's per-property
// reading (§3.17.6.1).
type componentKey struct {
	kind string
	name string
}

// overrideEntry is one child of an <override> element paired with the key it
// overrides on. Entries are kept as a SLICE in document order, never as a bare
// map, because §F.2 clause 4.3 appends the outer override's unmatched children to
// a nested one in that order — an order that reaches the user through the
// resulting component set and through duplicate reports (STYLE D2).
type overrideEntry struct {
	key  componentKey
	elem *Element
}

// overrideSet is the ·override pre-processing· in force over one schema document:
// the substitutions an <override> element (composed with any override already in
// force over the document that contains it) makes in the document it points at.
//
// The nil *overrideSet is the identity — it substitutes nothing — so a document
// reached by a plain <include>/<import>, and the root, need no special case.
type overrideSet struct {
	// entries are the substitutions in document order (see overrideEntry).
	entries []overrideEntry

	// index is the lookup entries are consulted through, derived from them at
	// construction and never ranged to produce output (STYLE D2/D3).
	index map[componentKey]*Element

	// id is the set's canonical identity, likewise derived from entries at
	// construction: the ordered (kind, name, source location) triples. It is the
	// override half of docKey, so that one document overridden two DIFFERENT ways
	// is read twice — the spec's own outcome ("the resulting schema will have
	// duplicate and conflicting versions of some components", §4.2.5) — while one
	// overridden the same way twice, or reached again around an <include>/<override>
	// cycle, is read once. Because every reachable set is drawn from the finitely
	// many children of the finitely many <override> elements of the assembly, the
	// space of ids is finite and the load-once index that keys on it terminates the
	// walk §4.2.5's note requires processors to close (STYLE D4: identity, not a
	// cycle guard).
	id string
}

// newOverrideSet reads the substitutions an <override> element declares (§F.2
// clause 1). It returns nil for an <override> that declares none: §4.2.5's
// termination note names "the fact that E is empty" as a condition making
// override(E,D) idempotent, and a nil set leaves the overridden document's
// load-once identity equal to a plain <include>'s, so a document reached both
// ways is read once rather than tripping sch-props-correct clause 2 on itself.
func newOverrideSet(el *Element) (*overrideSet, error) {
	var entries []overrideEntry
	seen := make(map[componentKey]*Element)
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if !overridableKind(c.Name().Local()) {
			// <annotation> — the only other child the §4.2.5 content model allows —
			// is not one of the seven element types §F.2 clause 1 matches on, so it
			// substitutes for nothing.
			continue
		}
		name, ok := c.Attr("name")
		if !ok {
			// The schema for schema documents makes name required on every
			// non-annotation child of <override>. One without it matches no source
			// declaration under clause 1, so it is ignored exactly as §4.2.5 ignores
			// a child that "matches nothing in the target set".
			continue
		}
		key := componentKey{kind: c.Name().Local(), name: name}
		if prev, dup := seen[key]; dup {
			// §F.2's NORMATIVE stylesheet resolves a repeated (element type, name)
			// pair as first-match-wins — clause 1 selects ($replacement, $original)[1]
			// — so the REC as published makes this case VALID. It is rejected here
			// because the Working Group resolved otherwise (W3C Bugzilla 17617,
			// 2012-06-29: "an erratum is needed to make this situation an error"), the
			// erratum was never filed, and W3C Override/over021 is `accepted` on the
			// strength of that intent. The rejection loses a valid assembly, never
			// accepts an invalid one.
			return nil, xsderr.New(ruleSrcOverride, c.Loc(),
				"<override> has two <%s> children named %q (the first at %s): a duplicate override target, which the XML Schema WG resolved is an error (W3C Bugzilla 17617) though §F.2 clause 1's stylesheet would resolve it to the first",
				key.kind, key.name, prev.Loc())
		}
		seen[key] = c
		entries = append(entries, overrideEntry{key: key, elem: c})
	}
	return buildOverrideSet(entries), nil
}

// buildOverrideSet completes an overrideSet from its document-ordered entries,
// deriving BOTH the lookup index and the identity string from them so neither is
// a second encoding of the same fact (STYLE D3). An empty entry list is the nil
// set.
func buildOverrideSet(entries []overrideEntry) *overrideSet {
	if len(entries) == 0 {
		return nil
	}
	index := make(map[componentKey]*Element, len(entries))
	var id strings.Builder
	for _, e := range entries {
		index[e.key] = e.elem
		id.WriteString(e.key.kind)
		id.WriteByte(' ')
		id.WriteString(e.key.name)
		id.WriteByte('@')
		id.WriteString(e.elem.Loc().String())
		id.WriteByte('\n')
	}
	return &overrideSet{entries: entries, index: index, id: id.String()}
}

// mergedUnder composes this override set — the one a nested <override> element
// declares — with outer, the one already in force over the document that contains
// it, per §F.2 clause 4: outer's matching children replace this set's (4.1), this
// set's unmatched children stay where they are (4.2), and outer's unmatched
// children are appended (4.3). Both operands are ranged as SLICES, so the merged
// order is fixed by document order alone (STYLE D2/S3).
//
// Either operand may be the nil (identity) set: composing with nothing is the
// other operand unchanged.
func (s *overrideSet) mergedUnder(outer *overrideSet) *overrideSet {
	if outer == nil {
		return s
	}
	if s == nil {
		return outer
	}
	entries := make([]overrideEntry, 0, len(s.entries)+len(outer.entries))
	for _, e := range s.entries {
		if repl, matched := outer.index[e.key]; matched {
			entries = append(entries, overrideEntry{key: e.key, elem: repl}) // clause 4.1
			continue
		}
		entries = append(entries, e) // clause 4.2
	}
	for _, e := range outer.entries {
		if _, matched := s.index[e.key]; matched {
			continue
		}
		entries = append(entries, e) // clause 4.3
	}
	return buildOverrideSet(entries)
}

// replacement returns the declaration that stands in for the top-level source
// declaration el under this override set: the overriding child when §F.2 clause 1
// matches (same element type, same name), el itself under clause 2. A directive
// child — <include>, <import>, <override>, <annotation> — carries no name and so
// is never substituted, which is clause 3/4/5's copy-unchanged behaviour for
// everything this parser reads off a document's top level.
func (s *overrideSet) replacement(el *Element) *Element {
	if s == nil {
		return el
	}
	name, ok := el.Attr("name")
	if !ok {
		return el
	}
	repl, matched := s.index[componentKey{kind: el.Name().Local(), name: name}]
	if !matched {
		return el
	}
	return repl
}

// key is this override set's contribution to a discovered document's load-once
// identity (see [overrideSet].id). The nil set contributes the empty string, so a
// plainly <include>d document and one reached under an empty <override> share a
// key.
func (s *overrideSet) key() string {
	if s == nil {
		return ""
	}
	return s.id
}

// overridableKind reports whether local is one of the seven element types §F.2
// clause 1 matches an <override> child against. It is deliberately NOT the
// <override> content model (which also admits <annotation>): it answers "can a
// child with this element type substitute for a source declaration", which is the
// only question the transformation asks.
func overridableKind(local string) bool {
	switch local {
	case "simpleType", "complexType", "group", "attributeGroup", "element", "attribute", "notation":
		return true
	}
	return false
}
