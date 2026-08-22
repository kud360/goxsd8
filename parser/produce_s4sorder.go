package parser

import (
	"fmt"
	"slices"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
)

// s4sSlot is ONE position in a schema-for-schema-documents content model: the
// child element local names that fill it, and whether the position repeats.
//
// admits is a predicate rather than a name list because one position — the facet
// choice of xs:simpleRestrictionModel (xmlschema11-2.md:3929) — is the
// substitution group of xs:facet, whose membership this package already owns a
// bridge for and may not retype (s4sFacetElement, STYLE T4).
type s4sSlot struct {
	admits   func(local string) bool
	repeated bool
}

// s4sModel is one element's schema-for-schema-documents content model, as the
// ordered positions its children fill. grammar names the Appendix A type or group
// the model belongs to, since the element name alone does not identify it —
// <restriction> carries three different models — spec is the line model is quoted
// from, and all three reach the diagnostic.
type s4sModel struct {
	grammar string
	spec    string
	model   string
	slots   []s4sSlot
}

// s4sNames returns a slot predicate admitting exactly the listed local names. The
// list is scanned linearly, never mapped: no position holds more than four names
// and a schema document's children are counted in tens (STYLE D3).
func s4sNames(names ...string) func(string) bool {
	return func(local string) bool { return slices.Contains(names, local) }
}

// s4sFacetElement reports whether local names a member of the xs:facet
// substitution group, the repeated position of xs:simpleRestrictionModel
// (xmlschema11-2.md:3929). It asks [builtin.FacetKindByName], the ONE name→kind
// bridge this package consults (facetKindOf records the same reuse), so a facet
// added to that table is ordered here without a second list to update.
//
// xs:assertion is the one name the bridge cannot answer for: §4.3.13 spells the
// facet "assertions" while the element contributing to it is <assertion>, the
// exception restrictionFacets makes at the same seam.
//
// The bridge also answers yes for "assertions" itself, which names no element at
// all, and for the precisionDecimal extension facets maxScale/minScale
// (xsd-precisionDecimal.md §4.2/§4.3), which the 2012 summary predates and this
// processor produces. Both are over-admissions in a check that only ORDERS the
// names its model admits somewhere: an element the grammar admits NOWHERE is a
// different fault, and an unowned one (#928).
func s4sFacetElement(local string) bool {
	if local == "assertion" {
		return true
	}
	_, ok := builtin.FacetKindByName(builtin.FacetName(local))
	return ok
}

// s4sAttributeTail is xs:attrDecls followed by xs:assertions
// (xmlschema11-1.md:4720, :4743) — "((attribute | attributeGroup)*,
// anyAttribute?), assert*" — the tail EVERY complex-type derivation alternant and
// the implicit-content form end with, held once so the five models cannot
// disagree about it (STYLE T4).
//
// The attribute block is ONE repeated position over two names, not two positions,
// because xs:attrDecls wraps them in an xs:choice with maxOccurs="unbounded": any
// interleaving of <attribute> and <attributeGroup> is legal, and only the block's
// contiguity and its precedence over <anyAttribute> are ordered facts.
var s4sAttributeTail = []s4sSlot{
	{admits: s4sNames("attribute", "attributeGroup"), repeated: true},
	{admits: s4sNames("anyAttribute")},
	{admits: s4sNames("assert"), repeated: true},
}

// s4sStructuralTail is s4sAttributeTail behind the two positions the three
// complex-content models put in front of it: "openContent?, (group | all | choice
// | sequence)?" — xs:openContent and xs:typeDefParticle (:4645).
//
// The two are held as SEPARATELY optional positions, following the XML
// Representation Summaries (:1649, :1718, :1723). Appendix A's
// xs:complexRestrictionType (:4850) is narrower — it wraps them in one optional
// sequence whose particle is required, so an <openContent> with no model group
// beside it fails there — and the summary's reading is taken because this check
// may only ever reject what BOTH readings reject.
var s4sStructuralTail = slices.Concat([]s4sSlot{
	{admits: s4sNames("openContent")},
	{admits: s4sNames("group", "all", "choice", "sequence")},
}, s4sAttributeTail)

// s4sAnnotationFirst is the "annotation?" every one of these models opens with,
// inherited from xs:annotated (:4426).
var s4sAnnotationFirst = []s4sSlot{{admits: s4sNames("annotation")}}

// The eight models checkS4SChildOrder is charged with, one per element position a
// complex type is written through — xs:complexTypeModel appearing twice, once for
// each of its disjuncts a <complexType> can be dispatched on. Each model is quoted
// verbatim from its XML Representation Summary, and its slots are that quotation
// read left to right.
//
// These are TRANSCRIBED from the spec, not generated (PRINCIPLES 26). Generating
// them means flattening Appendix A itself — resolving xs:group refs and the
// xs:restriction/xs:extension chains through xs:annotated — for the whole schema
// for schema documents rather than these eight, which is its own tool and its own
// grounding; the eight here are pinned against their quoted model text and against
// the disjointness their fault classification rests on (the tests beside this
// file), and rejectProhibitedAttrs (produce.go) already transcribes s4s facts on
// the same footing.
var (
	// s4sComplexTypeWrapped is xs:complexTypeModel (:4757) on the two disjuncts
	// that delegate: a <complexType> carrying a <simpleContent> or a
	// <complexContent> child holds nothing else but a leading <annotation>.
	s4sComplexTypeWrapped = s4sModel{
		grammar: "xs:complexTypeModel",
		spec:    "xmlschema11-1.md:1649",
		model:   "(annotation?, (simpleContent | complexContent | (openContent?, (group | all | choice | sequence)?, ((attribute | attributeGroup)*, anyAttribute?), assert*)))",
		slots: slices.Concat(s4sAnnotationFirst, []s4sSlot{
			{admits: s4sNames("simpleContent", "complexContent")},
		}),
	}

	// s4sComplexTypeImplicit is the SAME group's third disjunct, the
	// implicit-content form (§3.4.2.3.2, :1737-1741). §3.4.2.3.3's opening note
	// (:1744) is what makes it a site of its own: wherever a mapping rule says
	// "the [[children]]", this form means the <complexType>'s own, so the tail
	// ordered here is the one xs:complexRestrictionType orders under a
	// <complexContent>, inlined.
	s4sComplexTypeImplicit = s4sModel{
		grammar: "xs:complexTypeModel",
		spec:    "xmlschema11-1.md:1649",
		model:   "(annotation?, (simpleContent | complexContent | (openContent?, (group | all | choice | sequence)?, ((attribute | attributeGroup)*, anyAttribute?), assert*)))",
		slots:   slices.Concat(s4sAnnotationFirst, s4sStructuralTail),
	}

	// s4sSimpleContentWrapper is the xs:simpleContent element's own model (:4995).
	s4sSimpleContentWrapper = s4sModel{
		grammar: "xs:simpleContent",
		spec:    "xmlschema11-1.md:1687",
		model:   "(annotation?, (restriction | extension))",
		slots: slices.Concat(s4sAnnotationFirst, []s4sSlot{
			{admits: s4sNames("restriction", "extension")},
		}),
	}

	// s4sComplexContentWrapper is the xs:complexContent element's own model
	// (:4887), the same shape at a different line — quoted twice rather than
	// shared, because the two are separate grammar declarations and a future
	// divergence must show up as an edit here.
	s4sComplexContentWrapper = s4sModel{
		grammar: "xs:complexContent",
		spec:    "xmlschema11-1.md:1713",
		model:   "(annotation?, (restriction | extension))",
		slots: slices.Concat(s4sAnnotationFirst, []s4sSlot{
			{admits: s4sNames("restriction", "extension")},
		}),
	}

	// s4sSimpleRestriction is xs:simpleRestrictionType (:4960). Its facet position
	// is repeated, so a facet written twice is ADMITTED here and rejected by
	// repeatedFacetChild instead: src-ct clause 2 is the constraint that bounds
	// facet repetition, and this check may not usurp it (STYLE E2).
	s4sSimpleRestriction = s4sModel{
		grammar: "xs:simpleRestrictionType",
		spec:    "xmlschema11-1.md:1692",
		model:   "(annotation?, (simpleType?, (minExclusive | minInclusive | maxExclusive | maxInclusive | totalDigits | fractionDigits | length | minLength | maxLength | enumeration | whiteSpace | pattern | assertion | {any with namespace: ##other})*)?, ((attribute | attributeGroup)*, anyAttribute?), assert*)",
		slots: slices.Concat(s4sAnnotationFirst, []s4sSlot{
			{admits: s4sNames("simpleType")},
			{admits: s4sFacetElement, repeated: true},
		}, s4sAttributeTail),
	}

	// s4sSimpleExtension is xs:simpleExtensionType (:4979), the one alternant with
	// no structural position at all: no <openContent>, no model group, no facets.
	s4sSimpleExtension = s4sModel{
		grammar: "xs:simpleExtensionType",
		spec:    "xmlschema11-1.md:1697",
		model:   "(annotation?, ((attribute | attributeGroup)*, anyAttribute?), assert*)",
		slots:   slices.Concat(s4sAnnotationFirst, s4sAttributeTail),
	}

	// s4sComplexRestriction is xs:complexRestrictionType (:4850).
	s4sComplexRestriction = s4sModel{
		grammar: "xs:complexRestrictionType",
		spec:    "xmlschema11-1.md:1718",
		model:   "(annotation?, openContent?, (group | all | choice | sequence)?, ((attribute | attributeGroup)*, anyAttribute?), assert*)",
		slots:   slices.Concat(s4sAnnotationFirst, s4sStructuralTail),
	}

	// s4sComplexExtension is xs:extensionType (:4873). Its summary brackets the
	// tail differently from xs:complexRestrictionType's, which changes no
	// admissible sequence: the bracketing groups positions that are already
	// consecutive.
	s4sComplexExtension = s4sModel{
		grammar: "xs:extensionType",
		spec:    "xmlschema11-1.md:1723",
		model:   "(annotation?, openContent?, ((group | all | choice | sequence)?, ((attribute | attributeGroup)*, anyAttribute?), assert*))",
		slots:   slices.Concat(s4sAnnotationFirst, s4sStructuralTail),
	}
)

// checkS4SChildOrder rejects a child of owner that m's content model does not
// admit where it is written: a child before a position it must follow, or a
// second child in a position that does not repeat. It charges §5.1's FIRST bullet
// (:4296) — "not fully valid with respect to a schema corresponding to the Schema
// for Schema Documents" — which carries NO numbered rule ID: §2.3's gloss-src
// defines a Schema Representation Constraint as a condition "beyond those which
// are expressed in" the schema for schema documents, §2.4 states sd-valid and
// sd-supervalid as two separate conformance clauses, and src-ct's own preamble
// (§3.4.3, :1945) opens "In addition to the conditions imposed … by the schema for
// schema documents" over five clauses none of which concerns child order or
// cardinality. Charging src-ct — or any other src-*/cvc-*/cos-* — here would be a
// fabricated verdict (STYLE E2). The plain wrapped error is the footing
// rejectProhibitedAttrs (produce.go) already stands on for the same fault class.
//
// The match is a genuine left-to-right walk of the model's positions, not a fixed
// linear order over element names (PRINCIPLES 14). That is the whole difference on
// the two faults that look alike: <attribute> and <attributeGroup> share ONE
// repeated position, so any interleaving of them is legal, while <anyAttribute>
// holds the position after it, so an <attribute> written once <anyAttribute> has
// matched cannot re-enter the block behind it.
//
// A child whose name NO position of m admits is skipped, not rejected. What that
// leaves unrejected is an element the grammar does not admit in this position at
// all — a <complexContent> under a <simpleContent>, an <element> beside a facet —
// which is a different fault with an owner of its own (#928), and one this check
// must not pre-empt by charging an order verdict over it. Children outside the XSD
// namespace are skipped for the same reason plus one: they are what
// xs:simpleRestrictionModel's "{any with namespace: ##other}" position admits.
func checkS4SChildOrder(owner *Element, m s4sModel) error {
	// last is the position the previous matched child filled, and the search start
	// is derived from it rather than tracked beside it (STYLE D3): a repeated
	// position stays open, any other closes behind the child that filled it.
	last := -1
	for _, child := range owner.Children() {
		el, ok := child.(*Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		from := last + 1
		if last >= 0 && m.slots[last].repeated {
			from = last
		}
		local := el.Name().Local()
		if at := s4sSlotAt(m.slots, from, local); at >= 0 {
			last = at
			continue
		}
		// The name is admitted somewhere, but not from here on. It fills the
		// position it already filled — a maxOccurs fault — or one now behind the
		// walk, an order fault. No name appears in two positions of any model
		// above, so the first position admitting it is the only one.
		filled := s4sSlotAt(m.slots, 0, local)
		if filled < 0 {
			continue
		}
		if filled == last {
			return fmt.Errorf("parser: <%s> at %s repeats a position the schema for schema documents admits at most once among the children of the <%s> at %s: %s's content model (%s) is %s",
				local, el.Loc(), owner.Name().Local(), owner.Loc(), m.grammar, m.spec, m.model)
		}
		return fmt.Errorf("parser: <%s> at %s is out of the child order the schema for schema documents requires of the <%s> at %s: %s's content model (%s) is %s, and a <%s> may not follow the children written before it here",
			local, el.Loc(), owner.Name().Local(), owner.Loc(), m.grammar, m.spec, m.model, local)
	}
	return nil
}

// s4sSlotAt returns the index of the first position at or after from that admits
// local, or -1 when none does.
func s4sSlotAt(slots []s4sSlot, from int, local string) int {
	for i := from; i < len(slots); i++ {
		if slots[i].admits(local) {
			return i
		}
	}
	return -1
}
