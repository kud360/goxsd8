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
// decides that with the first end-to-end producer (parser.Produce + Finalize,
// issue #174), which maps ONLY top-level <simpleType>/<element>/<attribute>
// declarations into xsd components, resolves cross-references, and rejects
// duplicate top-level names within a kind. Every other top-level representation
// (complexType/group/attributeGroup/notation/import/include/redefine/override)
// is SILENTLY SKIPPED by Produce (§3.1.2 permits ignoring a not-yet-produced
// representation), NOT rejected.
//
// # Why "Produce returns nil" is not, by itself, evidence of validity
//
// Because Produce silently skips the representations it does not yet build, a
// document whose top-level content includes (say) an invalid <complexType>
// alongside valid simpleType/element/attribute would still Produce+Finalize with
// no error — a FALSE ACCEPT. §3.1.2's licence to ignore a representation is an
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
//     xsd:simpleType, xsd:element, or xsd:attribute — anything else at top level
//     (complexType/group/attributeGroup/notation/import/include/redefine/override,
//     any non-xsd element, or an out-of-set local name) closes the false-accept
//     gap above by DECLINING the whole case. Within the allowed kinds:
//     - element: must carry type= AND have no inline <simpleType>/<complexType>
//       child. A bare element (no type=) defaults to xs:anyType (§3.3.2.1 case 4),
//       a Complex Type Definition this narrow producer never seeds, so a
//       genuinely-valid bare element would be WRONGLY rejected at src-resolve
//       (§3.17.6.2) — a false reject, DECLINED. An inline anonymous type (with or
//       without type=) is an explicit src-element clause 3 (§3.3.3) rejection that
//       conflates a genuine both-present violation with a mere not-yet-supported
//       inline-only form — indistinguishable here, DECLINED.
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
//     src-simple-type §3.16.3, src-resolve §3.17.6.2, st-props-correct), never a
//     fabricated one — the shape allowlist excludes every case whose rejection
//     would be a limitation-in-disguise. The case Passes iff observed agrees with
//     the suite's declared validity.
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
// list/union/enumeration/assertion simpleTypes, bare elements) where Produce's
// rejection would be a limitation rather than a spec violation. The remaining
// risk the allowlist specifically closes is the VACUOUS pass — a document of
// entirely skipped top-level content (e.g. all complexType) that would otherwise
// always "pass" through Produce doing nothing — which is why step 3 confines the
// whole top level to the four processed kinds.
//
// # Still deferred
//
// Bare-element-defaulting cases (xs:anyType), inline anonymous types on
// element/attribute, list/union/enumeration/assertion simpleTypes, and every
// other top-level declaration kind widen in with later producer slices (#176
// onward, exactly as the datatypes lane grew across #15/#57/#80); they stay
// DECLINED (Fail) recorded gaps here, never guessed.

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
		default:
			// complexType/group/attributeGroup/notation/import/include/redefine/
			// override or any other local name: silently skipped by Produce, so a
			// nil verdict there would be vacuous — decline the whole case.
			return false
		}
	}
	return true
}

// elementDecidable reports whether a top-level <element> is in the type=-form the
// producer decides genuinely: it must carry a type= attribute AND have no inline
// <simpleType>/<complexType> child. A bare element (no type=) defaults to
// xs:anyType (§3.3.2.1 case 4), a Complex Type Definition never seeded here, so it
// would be wrongly rejected at src-resolve — a false reject; an inline anonymous
// type is an explicit src-element clause 3 rejection (§3.3.3) that conflates a
// genuine violation with an unsupported form. Both are declined.
func elementDecidable(el *parser.Element) bool {
	if !hasAttr(el, "type") {
		return false
	}
	return childXSD(el, "simpleType") == nil && childXSD(el, "complexType") == nil
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

// hasAttr reports whether el carries the unprefixed (no-namespace) attribute
// local — XSD schema-element attributes carry no namespace.
func hasAttr(el *parser.Element, local string) bool {
	for _, a := range el.Attributes() {
		if a.Name().Space() == "" && a.Name().Local() == local {
			return true
		}
	}
	return false
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
