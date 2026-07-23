package conformance

import (
	"os"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file activates the schema lane (issue #175) by giving the schema entry of
// defaultLanes a real executor. It touches nothing else in the runner (the #6
// seam, STYLE T2): the lane's selector stays selectsKind(kindSchema), so the
// executor is handed EVERY schemaTest case and either decides it or honestly
// DECLINES it (records a Fail gap) — a case it cannot decide for the right
// reason never flips to pass. It is package-internal conformance support: it
// exports nothing and no library code imports it.
//
// # What a schemaTest asserts
//
// A schemaTest asks: is THIS schema document itself schema-valid? The lane
// decides that with the end-to-end producer (parser.Produce + Finalize, issues
// #174/#176), which maps top-level <simpleType>/<element>/<attribute> and the
// produce-time-decidable subset of <complexType> (implicit and <complexContent>
// <restriction> content, its particles, local element/attribute declarations,
// attribute uses, and wildcards) into xsd components, seeds the ur-type
// xs:anyType, resolves cross-references, and rejects duplicate top-level names
// within a kind. The remaining top-level representations (group/attributeGroup/
// notation/import/include/redefine/override) and the not-yet-produced complexType
// forms (<simpleContent>, <complexContent> <extension>, group/attributeGroup
// references, inline anonymous local types, <openContent>) are SILENTLY SKIPPED
// or declined by Produce (§3.1.2 permits ignoring a not-yet-produced
// representation), NOT rejected.
//
// # Why "Produce returns nil" is not, by itself, evidence of validity
//
// Because Produce silently skips the representations it does not yet build, a
// document whose top-level content includes (say) an invalid <group> or an
// undecidable <complexType> form alongside valid simpleType/element/attribute
// would still Produce+Finalize with no error — a FALSE ACCEPT. §3.1.2's licence
// to ignore a representation is an
// implementation choice about what to BUILD; it does not make the spec consider
// such a document valid: the invalid complexType still makes the document
// schema-INVALID under sch-props-correct clause 1 (§3.17.6.1), whichever
// cvc-complex-type/cos-* rule it violates (oracle grounding, issue #175). So
// "Produce returns nil" is genuine evidence of validity ONLY when the document's
// top-level content is PROVABLY CONFINED to what Produce actually processes.
//
// # The decidable shape (the strict top-level allowlist)
//
// execSchemaCase therefore decides a case only after confirming its whole shape
// is confined to what the producer checks, and DECLINES (Fail) anything else:
//
//  1. Readability. parser.ReadDocument is run first. ANY error DECLINES the case
//     (Fail), never a validity verdict: a ReadDocument error does not distinguish
//     a genuine XML well-formedness fault from a parser encoding LIMITATION.
//     Well-formed UTF-16 input (BOM FF FE) is currently rejected as "invalid
//     UTF-8" because UTF-16 decoding is not yet implemented, so treating that as
//     observed-invalid would fabricate an "invalid" verdict for a well-formed
//     document — a wrong-reason pass that would flip pass→fail once UTF-16
//     decoding lands (a separate change). So malformed XML is NOT a claimed
//     schema-well-formedness sub-cohort here; it is a declined recorded gap.
//  2. Root identity. If the root is not <schema> (IsSchema false) the case is
//     DECLINED: §3.17.2 explicitly does NOT require <schema> to be the document
//     root, so Produce's error there is a plain non-xsderr Go precondition fault,
//     not a sch-props-correct rejection — not decidable for this lane. Inventing
//     a "root must be <schema>" rejection would overreach (oracle grounding).
//  3. Top-level allowlist. Every top-level child element must be xsd:annotation,
//     xsd:simpleType, xsd:element, xsd:attribute, or xsd:complexType — anything
//     else at top level (group/attributeGroup/notation/import/include/redefine/
//     override/defaultOpenContent, any non-xsd element, or an out-of-set local
//     name) closes the false-accept gap above by DECLINING the whole case. Within
//     the allowed kinds:
//     - element: must have no inline <simpleType>/<complexType> child. A bare
//       element (no type=) defaults to xs:anyType (§3.3.2.1 case 4), now seeded as
//       a Complex Type Definition (§3.4.7), so it resolves and is decided
//       genuinely; type= is no longer required. An inline anonymous type is an
//       explicit src-element clause 3 (§3.3.3) rejection that conflates a genuine
//       both-present violation with a mere not-yet-supported inline-only form —
//       indistinguishable here, DECLINED.
//     - complexType (top-level, or a <complexContent> <restriction> reached
//       transitively): must lie within the producer's decidable subset per
//       complexTypeDecidable — implicit or <restriction> complex content whose
//       content model is element/any/sequence/choice/all and whose attributes are
//       local <attribute>/<anyAttribute>, with no <simpleContent>, no
//       <complexContent> <extension>, no <openContent>, no group/attributeGroup
//       reference, and no inline anonymous local type. Those excluded forms need
//       the resolved base or a later slice, so Produce declines them with a plain
//       limitation error, not a spec verdict — DECLINED to avoid a wrong-reason
//       pass. A real structural violation inside an admitted shape (src-ct,
//       cos-all-limited, src-wildcard, …) flows through as a genuine rejection.
//     - attribute: must have no inline <simpleType> child (src-attribute clause 4,
//       §3.2.3). A bare attribute is FINE: it defaults to xs:anySimpleType
//       (§3.2.2.1), which builtin.Seed always seeds, so type= is NOT required.
//     - simpleType (top-level or any anonymous inline base reached transitively
//       through a restriction chain): must have exactly one <restriction> child
//       (no <list>/<union> — their absence of a <restriction> is an explicit
//       src-simple-type rejection that conflates genuine invalidity with an
//       unsupported variety, DECLINED) whose children include no <enumeration>/
//       <assertion> (likewise not-yet-produced facets rejected by src-simple-type
//       §3.16.3 clause; DECLINED). An inline <simpleType> base child (the
//       genuinely-supported anonymous nested base, §3.16.3 clause 2) is recursed
//       into with the same two checks. The restriction's base=/inline-child
//       exactly-one arrangement is NOT pre-checked: that IS the genuine
//       src-simple-type clause 2 rule Produce correctly enforces, so a violation
//       flows through as a real decidable rejection.
//     - annotation: always allowed, no further check.
//  4. Decide. When the whole shape passes, parser.Produce is run and observed =
//     (err == nil): a nil error is genuine evidence of validity (the shape has
//     none of the violations checked above, so a real one would surface), and a
//     non-nil error is a REAL, implemented rejection (sch-props-correct clause 2
//     duplicate-name §3.17.6.1, src-element §3.3.3, src-attribute §3.2.3,
//     src-simple-type §3.16.3, src-resolve §3.17.6.2, st-props-correct, and for
//     the complex-type subset src-ct §3.4.3, cos-all-limited §3.8.6, src-wildcard
//     §3.10.3, p-props-correct §3.9.6), never a fabricated one — the shape
//     allowlist excludes every case whose rejection would be a
//     limitation-in-disguise. The case Passes iff observed agrees with the suite's
//     declared validity.
//
// # sch-props-correct clause 2 is per-kind
//
// The duplicate-name rejection (sch-props-correct §3.17.6.1 clause 2) is checked
// PER KIND ({type definitions}, {element declarations}, {attribute declarations}
// are distinct properties, §3.17.1): two simpleTypes sharing an expanded name
// collide, but a simpleType and an element sharing a name do NOT. The executor
// relies on Finalize's per-kind indexByName for exactly this, so no cross-kind
// duplicate check is done here (that would be a false-INVALID verdict, a ratchet
// regression risk).
//
// # Why no false ratchet-corrupting pass is possible
//
// Every "invalid" verdict this lane emits comes from ONE source: parser.Produce
// rejecting a document whose shape already passed the allowlist. ReadDocument
// errors never produce an "invalid" verdict — they decline (step 1) — precisely
// because a ReadDocument error can be a parser encoding limitation (well-formed
// UTF-16 misread as invalid UTF-8) rather than a real violation, and turning that
// into "invalid" would fabricate a verdict for a well-formed document.
//
// A "valid" verdict coincides only with a truly-valid ground truth: a truly-valid
// document (by definition) has none of the checked violations, so Produce
// correctly finds none. An "invalid" verdict coincides only with truly-invalid
// ground truth via a REAL implemented violation — never a fabricated one, since
// the shape allowlist excludes every form (inline element/attribute types,
// list/union/enumeration/assertion simpleTypes, and the not-yet-produced
// complexType forms — <simpleContent>, <complexContent> <extension>, group/
// attributeGroup references, inline anonymous local types, <openContent>) where
// Produce's rejection would be a limitation rather than a spec violation. A
// suite-invalid case whose only defect is a rule this slice does NOT yet check
// (UPA cos-nonambig, EDC, derivation-ok-restriction) is produced cleanly, so the
// lane observes "valid", disagrees with the suite, and records a still-failing
// gap — never a wrong "invalid" pass. The remaining risk the allowlist closes is
// the VACUOUS pass — a document of entirely skipped top-level content that would
// otherwise always "pass" through Produce doing nothing — which is why step 3
// confines the whole top level to the processed kinds and the decidable
// complexType subset.
//
// # Still deferred
//
// Inline anonymous types on element/attribute, list/union/enumeration/assertion
// simpleTypes, the not-yet-produced complexType forms named above, and every
// other top-level declaration kind widen in with later producer slices (exactly
// as the datatypes lane grew across #15/#57/#80); they stay DECLINED (Fail)
// recorded gaps here, never guessed. The derivation-validity, UPA, and EDC rules
// (#180/#181) that would newly reject some admitted complexType cases as invalid
// are separate slices; until they land, those suite-invalid cases stay failing
// gaps rather than wins.

// newSchemaExec builds the schema lane's executor. The strict backend is built
// once here (mirroring newDatatypesExec's strictBackend := strict.New()): it maps
// all 20 primitives, so parser.Produce's internal builtin.Seed precondition holds
// for every case. Produce seeds from the backend on each call, so no symbol table
// is captured here — the executor only threads the backend and reads the document.
func newSchemaExec() executor {
	backend := strict.New()
	return func(c caseSpec) Status {
		return execSchemaCase(backend, c)
	}
}

// execSchemaCase decides one schemaTest case, or honestly declines it (Fail). It
// reads the document, gates on the decidable top-level shape (schemaShapeDecidable),
// then runs parser.Produce and agrees or disagrees with the suite's declared
// validity. A document it cannot open OR cannot read (any ReadDocument error,
// including a parser encoding limitation such as unsupported UTF-16), whose root is
// not <schema>, or whose shape falls outside the producer's decidable subset is
// DECLINED (Fail) as a recorded gap, never guessed.
func execSchemaCase(backend value.Backend, c caseSpec) Status {
	f, err := os.Open(c.doc)
	if err != nil {
		// Unreadable document: an honest recorded gap, not a validity verdict.
		return Fail()
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the verdict
	doc, err := parser.ReadDocument(c.doc, f)
	if err != nil {
		// A ReadDocument error is DECLINED, never treated as an observed-invalid
		// verdict. The error does not distinguish a genuine XML well-formedness
		// fault from a parser encoding LIMITATION: well-formed UTF-16 input (BOM
		// FF FE) is currently rejected as "[xml-wf] invalid UTF-8" because UTF-16
		// decoding is not yet implemented, so an "invalid" verdict here would be
		// fabricated for a well-formed document — a wrong-reason pass that would
		// silently flip pass→fail once UTF-16 decoding lands (a separate change).
		// Declining on ANY ReadDocument error keeps the lane's verdicts honest.
		return Fail()
	}
	// §3.17.2 does not require <schema> to be the document root, so a non-schema
	// root is a producer precondition fault (a plain Go error, not a
	// sch-props-correct rejection), not decidable for this lane — decline.
	if !doc.IsSchema() {
		return Fail()
	}
	// Only decide when the whole top level is confined to what Produce processes;
	// otherwise a silently-skipped invalid representation could false-accept.
	if !schemaShapeDecidable(doc) {
		return Fail()
	}
	_, perr := parser.Produce(doc, backend)
	return decideSchema(perr == nil, c.expectValid)
}

// decideSchema Passes iff the observed validity agrees with the suite's declared
// XSD 1.1 expectation.
func decideSchema(observed, expected bool) Status {
	if observed == expected {
		return Pass()
	}
	return Fail()
}

// schemaShapeDecidable reports whether every top-level child of the <schema> root
// lies within the producer's decidable subset (the step-3 allowlist documented
// above). A single out-of-subset child declines the whole case, since Produce
// would silently skip it (or reject it for a not-yet-supported reason) rather
// than decide it genuinely.
func schemaShapeDecidable(doc *parser.Document) bool {
	for _, child := range doc.Root().Children() {
		el, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		name := el.Name()
		if name.Space() != xsd.XMLSchemaNS {
			return false
		}
		switch name.Local() {
		case "annotation":
			// Harmless, always allowed.
		case "element":
			if !elementDecidable(el) {
				return false
			}
		case "attribute":
			if !attributeDecidable(el) {
				return false
			}
		case "simpleType":
			if !simpleTypeDecidable(el) {
				return false
			}
		case "complexType":
			if !complexTypeDecidable(el) {
				return false
			}
		default:
			// group/attributeGroup/notation/import/include/redefine/override,
			// defaultOpenContent, or any other local name: silently skipped by
			// Produce, so a nil verdict there would be vacuous — decline the whole
			// case.
			return false
		}
	}
	return true
}

// elementDecidable reports whether a top-level <element> is in the form the
// producer decides genuinely: it must have no inline <simpleType>/<complexType>
// child. A bare element (no type=) now defaults to xs:anyType (§3.3.2.1 case 4),
// which the producer seeds as a Complex Type Definition (§3.4.7), so it resolves
// at finalize and is decided genuinely — type= is no longer required. An inline
// anonymous type is an explicit src-element clause 3 rejection (§3.3.3) that
// conflates a genuine violation with an unsupported form, so it stays declined.
func elementDecidable(el *parser.Element) bool {
	return childXSD(el, "simpleType") == nil && childXSD(el, "complexType") == nil
}

// complexTypeDecidable reports whether a <complexType> (top-level, or a nested
// <restriction> reached through <complexContent>) lies within the producer's
// decidable subset — the shapes it fully builds, so any Produce error on it is a
// REAL structural violation (src-ct/cos-all-limited/src-wildcard/src-attribute/
// p-props-correct/src-resolve), never a limitation-in-disguise. It declines every
// shape the producer declines with a plain "not yet produced" limitation error:
//
//   - <simpleContent> (its {simple type definition} needs the resolved base,
//     §3.4.2.2 — finalize-time);
//   - <complexContent> whose derivation is <extension>, not <restriction> (its
//     {content type} needs the resolved base particle, §3.4.2.3.3 clause 4.2);
//   - <openContent> anywhere (its {open content} needs <defaultOpenContent>
//     fallback, §3.4.2.3.3, not yet built);
//   - a <group> reference, an <attributeGroup> reference, or an inline anonymous
//     <simpleType>/<complexType> on a local element/attribute (all not yet
//     produced).
//
// Real structural violations the producer DOES reject (a nested <all>, a mixed
// mismatch, a both-namespace-forms wildcard, a bad occurrence) are NOT declined:
// admitting them is safe because the producer's rejection is the right reason.
func complexTypeDecidable(el *parser.Element) bool {
	if childXSD(el, "simpleContent") != nil || childXSD(el, "openContent") != nil {
		return false
	}
	if cc := childXSD(el, "complexContent"); cc != nil {
		restriction := childXSD(cc, "restriction")
		if restriction == nil {
			return false // <extension> (or a bare/absent derivation) — not produced
		}
		if childXSD(cc, "openContent") != nil || childXSD(restriction, "openContent") != nil {
			return false
		}
		return contentDecidable(restriction)
	}
	return contentDecidable(el)
}

// contentDecidable reports whether the content-model child and attribute children
// of a <complexType> (implicit content) or <restriction> (explicit complex
// content) are all within the producer's decidable subset. Anything unexpected at
// this level — a <group> content reference, an <attributeGroup>, a stray
// <simpleContent>/<openContent> — declines.
func contentDecidable(parent *parser.Element) bool {
	for _, child := range parent.Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch el.Name().Local() {
		case "annotation":
			// Harmless.
		case "sequence", "choice", "all":
			if !modelGroupDecidable(el) {
				return false
			}
		case "attribute":
			if childXSD(el, "simpleType") != nil {
				return false // inline anonymous attribute type — not yet produced
			}
		case "anyAttribute":
			// An attribute wildcard is produced.
		default:
			// group/attributeGroup/simpleContent/complexContent/openContent or any
			// other name at this level: not produced — decline.
			return false
		}
	}
	return true
}

// modelGroupDecidable reports whether every particle child of a model group
// (<sequence>/<choice>/<all>) is within the producer's decidable subset: nested
// model groups recurse, <element> must carry no inline anonymous type, and <any>
// is fine. A <group> reference or any other child declines.
func modelGroupDecidable(group *parser.Element) bool {
	for _, child := range group.Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch el.Name().Local() {
		case "annotation", "any":
			// Harmless / produced.
		case "element":
			if childXSD(el, "simpleType") != nil || childXSD(el, "complexType") != nil {
				return false // inline anonymous element type — not yet produced
			}
		case "sequence", "choice", "all":
			if !modelGroupDecidable(el) {
				return false
			}
		default:
			// group reference or any other child: not produced — decline.
			return false
		}
	}
	return true
}

// attributeDecidable reports whether a top-level <attribute> is decidable: it must
// have no inline <simpleType> child (src-attribute clause 4, §3.2.3). type= is NOT
// required — a bare attribute defaults to xs:anySimpleType (§3.2.2.1), which
// builtin.Seed always seeds, so it resolves and is decided genuinely.
func attributeDecidable(el *parser.Element) bool {
	return childXSD(el, "simpleType") == nil
}

// simpleTypeDecidable reports whether a <simpleType> (top-level or an anonymous
// inline base reached through a restriction chain) is decidable: it must have
// exactly one <restriction> child (no <list>/<union>, whose absence of a
// <restriction> is an unsupported-variety rejection) whose children carry no
// <enumeration>/<assertion> facet (not-yet-produced facets). An inline <simpleType>
// base child (the supported anonymous nested base, §3.16.3 clause 2) is recursed
// into with the same checks. src-simple-type §3.16.3.
func simpleTypeDecidable(el *parser.Element) bool {
	restriction := childXSD(el, "restriction")
	if restriction == nil {
		return false
	}
	for _, child := range restriction.Children() {
		r, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		if r.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch r.Name().Local() {
		case "enumeration", "assertion":
			return false
		case "simpleType":
			if !simpleTypeDecidable(r) {
				return false
			}
		}
	}
	return true
}

// childXSD returns el's first child element with expanded name {XMLSchemaNS}local,
// or nil.
func childXSD(el *parser.Element, local string) *parser.Element {
	for _, child := range el.Children() {
		c, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		if name := c.Name(); name.Space() == xsd.XMLSchemaNS && name.Local() == local {
			return c
		}
	}
	return nil
}
