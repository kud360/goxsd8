package conformance

import (
	"path/filepath"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
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
// decides that with the end-to-end assembler (parser.Parse, issue #179, which
// subsumes the producer and Finalize of issues #174/#176/#177/#178). Parse
// implements §4.2.1's schema(D) — the root document's components plus,
// transitively, those of every document reached through an <xs:include> child
// (§4.2.3), chameleon coercion included (§F.1) — and maps top-level
// <simpleType>/<element>/<attribute>/<attributeGroup>/<group>/<notation> and the
// produce-time-decidable subset of <complexType> (implicit and <complexContent>
// <restriction> content, its particles including <group ref>, local
// element/attribute declarations, attribute uses including <attributeGroup ref>,
// wildcards, and <assert> assertions) into xsd components, maps the name=
// identity constraints of global and local <element>s (#178), seeds the ur-type
// xs:anyType, resolves cross-references, and rejects duplicate top-level names
// within a kind. The remaining top-level representations
// (import/redefine/override), the ref= identity-constraint form, and the
// not-yet-produced complexType forms (<simpleContent>, <complexContent>
// <extension>, inline anonymous local types, <openContent>) are SILENTLY SKIPPED
// or declined (§3.1.2 permits ignoring a not-yet-produced representation), NOT
// rejected.
//
// # Why "Parse returns nil" is not, by itself, evidence of validity
//
// Because the producer silently skips the representations it does not yet build,
// a document whose top-level content includes (say) an invalid <group> or an
// undecidable <complexType> form alongside valid simpleType/element/attribute
// would still assemble with no error — a FALSE ACCEPT. §3.1.2's licence
// to ignore a representation is an
// implementation choice about what to BUILD; it does not make the spec consider
// such a document valid: the invalid complexType still makes the document
// schema-INVALID under sch-props-correct clause 1 (§3.17.6.1), whichever
// cvc-complex-type/cos-* rule it violates (oracle grounding, issue #175). So
// "Parse returns nil" is genuine evidence of validity ONLY when the top-level
// content is PROVABLY CONFINED to what the producer actually processes.
//
// Since #242 that qualifier binds over a CLOSURE, not one document. Parse reads
// the whole <include> closure, so an included document holding a skipped
// representation false-accepts exactly as a root one would — and Parse cannot be
// asked which documents it read (its discovery is unexported, and the *xsd.Schema
// it returns carries components, not provenance). The harness therefore performs
// its OWN discovery walk first (closureScan, conformance/schema_closure.go),
// independently finding every document of the closure and gating each one on the
// allowlist below. Only when the whole closure is decidable does the case reach
// parser.Parse. That walk resolves every schemaLocation exactly as the parser
// does — same resolver, same root location string, same §4.3.2 clause 4
// base-URI algorithm — because a document it under-discovered would be a document
// whose shape was never gated, which is the false accept back again.
//
// # The decidable shape (the strict top-level allowlist)
//
// execSchemaCase therefore decides a case only after confirming the whole shape
// of every document in its closure is confined to what the producer checks, and
// DECLINES (Fail) anything else:
//
//  1. Readability. parser.ReadDocument is run first — on the root, and again on
//     every document the discovery walk reaches. ANY error DECLINES the case
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
//     root, so Parse's error there is a plain non-xsderr Go precondition fault,
//     not a sch-props-correct rejection — not decidable for this lane. Inventing
//     a "root must be <schema>" rejection would overreach (oracle grounding). An
//     INCLUDED document that is not a <schema> is the opposite case and is NOT
//     declined: src-include clause 1 makes that a genuine rejection, which Parse
//     emits, so the walk leaves it alone (schema_closure.go).
//  3. Top-level allowlist. Every top-level child element must be xsd:annotation,
//     xsd:include, xsd:simpleType, xsd:element, xsd:attribute, xsd:complexType,
//     xsd:attributeGroup (named definition), xsd:group (named definition), or
//     xsd:notation — anything else at top level (import/redefine/override/
//     defaultOpenContent, any non-xsd element, or an out-of-set local name) closes
//     the false-accept gap above by DECLINING the whole case. Within the allowed
//     kinds:
//     - include: always admitted (#242). Its own content model is (annotation?),
//       so it contributes nothing the producer could silently skip; the
//       decidability of the document it POINTS AT is established by the discovery
//       walk, which reads that document and runs this same allowlist over it
//       (and over its own <include>s, transitively) before anything is decided —
//       not by this allowlist entry. src-include (§4.2.3) itself imposes no shape
//       constraint on the included document, only existence and targetNamespace
//       agreement, both of which Parse decides genuinely.
//     - element: must have no inline <simpleType>/<complexType> child, and every
//       <unique>/<key>/<keyref> child must use the name= form. A bare
//       element (no type=) defaults to xs:anyType (§3.3.2.1 case 4), now seeded as
//       a Complex Type Definition (§3.4.7), so it resolves and is decided
//       genuinely; type= is no longer required. An inline anonymous type is an
//       explicit src-element clause 3 (§3.3.3) rejection that conflates a genuine
//       both-present violation with a mere not-yet-supported inline-only form —
//       indistinguishable here, DECLINED. A name= identity constraint IS produced
//       (#178) — its src-identity-constraint (§3.11.3) and c-props-correct
//       (§3.11.6.1) rejections and its finalize-time keyref resolution
//       (src-resolve clause 1.7) are all genuine — while the ref= form names an
//       existing definition the producer does not yet resolve, so it is DECLINED.
//     - notation: always allowed. Its content is (annotation?) and its whole
//       tableau is decided at produce time (n-props-correct §3.14.6 rejects both
//       identifiers absent), so nothing is silently skipped.
//     - complexType (top-level, or a <complexContent> <restriction> reached
//       transitively): must lie within the producer's decidable subset per
//       complexTypeDecidable — implicit or <restriction> complex content whose
//       content model is element/any/sequence/choice/all/<group ref> and whose
//       attributes are local <attribute>/<anyAttribute>/<attributeGroup ref> and
//       whose <assert> children map to {assertions} (#178), with
//       no <simpleContent>, no <complexContent> <extension>, no <openContent>, and
//       no inline anonymous local type. Those excluded forms need the resolved base
//       or a later slice, so Produce declines them with a plain limitation error,
//       not a spec verdict — DECLINED to avoid a wrong-reason pass. A <group ref>/
//       <attributeGroup ref> IS produced (#177): its target resolves (or fails
//       src-resolve) genuinely. A real structural violation inside an admitted
//       shape (src-ct, cos-all-limited, src-wildcard, …) flows through as a genuine
//       rejection.
//     - attributeGroup (top-level named definition, §3.6.2): children only
//       <attribute> (no inline anonymous type), <attributeGroup ref>, and
//       <anyAttribute> — the shapes the producer folds in (§3.6.2.1/§3.6.2.2). A
//       dangling ref (src-resolve clause 1.4) and a circular ref chain (spec-legal,
//       §3.6.2.1) are both decided genuinely.
//     - group (top-level named definition, §3.7.2): must carry a name and a single
//       all/choice/sequence body whose particles are decidable; the body maps to
//       {model group} genuinely (mgd-props-correct rejects a missing body).
//     - attribute: must have no inline <simpleType> child (src-attribute clause 4,
//       §3.2.3). A bare attribute is FINE: it defaults to xs:anySimpleType
//       (§3.2.2.1), which builtin.Seed always seeds, so type= is NOT required.
//     - simpleType (top-level or any anonymous inline base reached transitively
//       through a restriction chain): must have exactly one <restriction> child
//       (no <list>/<union> — their absence of a <restriction> is an explicit
//       src-simple-type rejection that conflates genuine invalidity with an
//       unsupported variety, DECLINED) whose children include no <enumeration>
//       (still a not-yet-produced facet rejected by src-simple-type §3.16.3;
//       DECLINED). An <assertion> facet IS produced (#178, one assertions facet
//       per restriction, Datatypes §4.3.13) and is admitted. An inline <simpleType> base child (the
//       genuinely-supported anonymous nested base, §3.16.3 clause 2) is recursed
//       into with the same two checks. The restriction's base=/inline-child
//       exactly-one arrangement is NOT pre-checked: that IS the genuine
//       src-simple-type clause 2 rule Produce correctly enforces, so a violation
//       flows through as a real decidable rejection.
//     - annotation: always allowed, no further check.
//  4. Decide. When every document of the closure passes, parser.Parse is run and
//     observed = (err == nil): a nil error is genuine evidence of validity (no
//     document of the assembly has any of the violations checked above, so a real
//     one would surface), and a non-nil error is a REAL, implemented rejection
//     (src-include §4.2.3, sch-props-correct clause 2
//     duplicate-name §3.17.6.1, src-element §3.3.3, src-attribute §3.2.3,
//     src-simple-type §3.16.3, src-resolve §3.17.6.2, st-props-correct,
//     src-identity-constraint §3.11.3, c-props-correct §3.11.6.1,
//     n-props-correct §3.14.6, and for
//     the complex-type subset src-ct §3.4.3, cos-all-limited §3.8.6, src-wildcard
//     §3.10.3, p-props-correct §3.9.6), never a fabricated one — the shape
//     allowlist excludes every case whose rejection would be a
//     limitation-in-disguise. The case Passes iff observed agrees with the suite's
//     declared validity.
//
//     No error-type discrimination (errors.As over *xsderr.Error) is needed to
//     make that trustworthy, because step 1-3's walk has already ruled out the
//     non-verdict failure modes ACROSS THE WHOLE CLOSURE: every document Parse
//     will read has been independently confirmed resolvable, readable,
//     <schema>-rooted (or deliberately left to src-include clause 1) and
//     shape-decidable. The plain non-xsderr errors Parse can otherwise return —
//     an unresolvable root, an I/O or encoding failure, a non-schema root — are
//     exactly what the walk already eliminated, so what remains is spec verdicts.
//
// # sch-props-correct clause 2 is per-kind
//
// The duplicate-name rejection (sch-props-correct §3.17.6.1 clause 2) is checked
// PER KIND ({type definitions}, {element declarations}, {attribute declarations}
// are distinct properties, §3.17.1): two simpleTypes sharing an expanded name
// collide, but a simpleType and an element sharing a name do NOT.
// {identity-constraint definitions} is one such property too, and it collects the
// constraints of every <key>/<keyref>/<unique> ANYWHERE in the document
// (§3.17.1), so two identically-named identity constraints under two different
// element declarations DO collide — a genuine clause-2 rejection, not a
// producer artifact. The executor
// relies on Finalize's per-kind indexByName for exactly this, so no cross-kind
// duplicate check is done here (that would be a false-INVALID verdict, a ratchet
// regression risk).
//
// # Why no false ratchet-corrupting pass is possible
//
// Every "invalid" verdict this lane emits comes from ONE source: parser.Parse
// rejecting an assembly EVERY document of which already passed the allowlist.
// ReadDocument
// errors never produce an "invalid" verdict — they decline (step 1) — precisely
// because a ReadDocument error can be a parser encoding limitation (well-formed
// UTF-16 misread as invalid UTF-8) rather than a real violation, and turning that
// into "invalid" would fabricate a verdict for a well-formed document.
//
// A "valid" verdict coincides only with a truly-valid ground truth: a truly-valid
// assembly (by definition) has none of the checked violations, so Parse
// correctly finds none. An "invalid" verdict coincides only with truly-invalid
// ground truth via a REAL implemented violation — never a fabricated one, since
// the shape allowlist excludes every form (inline element/attribute types,
// list/union/enumeration simpleTypes, ref= identity constraints, and the
// not-yet-produced complexType forms — <simpleContent>, <complexContent>
// <extension>, inline anonymous local types, <openContent>) where the producer's
// rejection would be a limitation rather than a spec violation. A
// suite-invalid case whose only defect is a rule this slice does NOT yet check
// (UPA cos-nonambig, EDC, derivation-ok-restriction) is produced cleanly, so the
// lane observes "valid", disagrees with the suite, and records a still-failing
// gap — never a wrong "invalid" pass. The remaining risk the allowlist closes is
// the VACUOUS pass — a document of entirely skipped top-level content that would
// otherwise always "pass" through a producer doing nothing — which is why step 3
// confines the whole top level of EVERY document in the closure to the processed
// kinds and the decidable complexType subset.
//
// # Composition: <include> decided, import/redefine/override still deferred
//
// <xs:include>, chameleon inclusion included, is DECIDED as of #242: parser.Parse
// follows the closure (#179) and the harness's discovery walk gates every document
// in it, so an include/chameleon case is now decided for the same reason a
// single-document case is, not guessed.
//
// <xs:import>, <xs:redefine> and <xs:override> stay DECLINED. Parse does not
// follow them — like any other not-yet-produced representation they are skipped,
// not rejected (§3.1.2, #182/#183 unlanded) — so a document carrying one
// assembles SHORT: the components of the document it names never enter the
// builder, and any violation among them is invisible. That is precisely the
// vacuous pass step 3 exists to refuse, so their mere presence at top level
// declines the case until the parser follows them.
//
// # Still deferred
//
// Inline anonymous types on element/attribute, list/union/enumeration
// simpleTypes, ref= identity constraints, and the not-yet-produced complexType
// forms named above widen in with later producer slices (exactly
// as the datatypes lane grew across #15/#57/#80); they stay DECLINED (Fail)
// recorded gaps here, never guessed. The derivation-validity, UPA, and EDC rules
// (#180/#181) that would newly reject some admitted complexType cases as invalid
// are separate slices; until they land, those suite-invalid cases stay failing
// gaps rather than wins.
//
// A schemaTest with MORE THAN ONE <ts:schemaDocument> child is decided against the
// wrong document (the runner keeps only one, #238, unlanded). That defect is
// orthogonal to the closure walk — those cases were mis-decided before #242 and
// still are; assembling a case's several root documents is the runner's business,
// not this lane's.

// newSchemaExec builds the schema lane's executor. The strict backend is built
// once here (mirroring newDatatypesExec's strictBackend := strict.New()): it maps
// all 20 primitives, so parser.Parse's internal builtin.Seed precondition holds
// for every case. Parse seeds from the backend on each call, so no symbol table is
// captured here — the executor only threads the backend and reads the documents.
func newSchemaExec() executor {
	backend := strict.New()
	return func(c caseSpec) Status {
		return execSchemaCase(backend, c)
	}
}

// execSchemaCase decides one schemaTest case, or honestly declines it (Fail). It
// reads the root document, gates the WHOLE <xs:include> closure on the decidable
// top-level shape (closureScan.decidable, which runs schemaShapeDecidable on every
// document it discovers), then runs parser.Parse and agrees or disagrees with the
// suite's declared validity. A document it cannot resolve OR cannot read (any
// ReadDocument error, including a parser encoding limitation such as unsupported
// UTF-16), whose root is not <schema>, or any document of whose closure falls
// outside the producer's decidable subset is DECLINED (Fail) as a recorded gap,
// never guessed.
//
// The resolver is a loader.Dir rooted at the case document's own directory and the
// root is named by its BASE name, because parser.Parse reads the root under
// exactly the location string it is handed (readRootDocument in parser/parse.go):
// passing the full path would give the root document a base URI of
// "…/sunData/SType/x" instead of "x", and every <include> in it would then resolve
// one directory tree away from where the resolver serves. The harness's own read
// below therefore uses the SAME resolver and the SAME location string, so its
// closure walk resolves byte-identically to the assembly Parse builds.
func execSchemaCase(backend value.Backend, c caseSpec) Status {
	resolver := loader.Dir(filepath.Dir(c.doc))
	location := filepath.Base(c.doc)
	rc, resolved, err := resolver.Resolve("", location)
	if err != nil {
		// Unreadable document: an honest recorded gap, not a validity verdict.
		return Fail()
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	doc, err := parser.ReadDocument(location, rc)
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
	// root is a Parse precondition fault (a plain Go error, not a
	// sch-props-correct rejection), not decidable for this lane — decline.
	if !doc.IsSchema() {
		return Fail()
	}
	// Only decide when EVERY document of the <include> closure is confined to what
	// the producer processes; otherwise a silently-skipped invalid representation,
	// in the root or in any included document, could false-accept.
	if !newClosureScan(resolver, doc, resolved).decidable(doc) {
		return Fail()
	}
	_, perr := parser.Parse(location, parser.WithResolver(resolver), parser.WithBackend(backend))
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
//
// The root ELEMENT itself is checked first for defaultAttributes (§3.4.2.4): that
// attribute names an attribute group whose {attribute uses} must be folded into
// every complex type that does not set defaultAttributesApply="false". The
// producer does not model defaultAttributes at all, so the fold is silently
// skipped and a document invalid only because of the folded-in uses (a duplicate
// attribute, an ID clash, a src-resolve failure on the named group) would
// false-ACCEPT. Declining on the attribute's mere presence closes the gap: with
// no defaultAttributes on <schema>, defaultAttributesApply on any complexType has
// nothing to apply and skips nothing.
func schemaShapeDecidable(doc *parser.Document) bool {
	if hasAttr(doc.Root(), "defaultAttributes") {
		return false
	}
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
		case "include":
			// Admitted (#242). <include> contributes no component of its own — its
			// content model is (annotation?) — so there is nothing here for the
			// producer to silently skip. What it POINTS AT is the thing that could
			// be skipped, and that is gated by closureScan.decidable, which reads
			// the included document and runs this same function over it before any
			// case is decided. src-include's own clauses (§4.2.3) are then enforced
			// genuinely by parser.Parse.
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
		case "group":
			if !groupDecidable(el) {
				return false
			}
		case "attributeGroup":
			if !attributeGroupDecidable(el) {
				return false
			}
		case "notation":
			// Produced (#178) with no undecidable sub-shape: <notation>'s content is
			// (annotation?) and its whole property tableau is settled at produce time
			// (n-props-correct §3.14.6 rejects both identifiers absent), so it is
			// always admitted, like annotation.
		default:
			// import/redefine/override, defaultOpenContent, or any other local name:
			// silently skipped by the producer AND not followed by the assembly
			// (#182/#183 unlanded), so a nil verdict there would be vacuous —
			// decline the whole case.
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
// Its <unique>/<key>/<keyref> children must also be decidable (#178).
func elementDecidable(el *parser.Element) bool {
	return childXSD(el, "simpleType") == nil && childXSD(el, "complexType") == nil &&
		identityConstraintsDecidable(el)
}

// identityConstraintsDecidable reports whether every <unique>/<key>/<keyref>
// child of an <element> (global or local) is in the form the producer builds
// (#178): the name= form, whose whole mapping — category, selector, fields,
// refer — is settled at produce time, so src-identity-constraint (§3.11.3) and
// c-props-correct (§3.11.6.1) rejections on it are genuine. The ref= form
// corresponds to no new component (§3.11.2: it names an existing definition,
// resolved at finalize) and the producer declines it as not yet produced, so its
// presence declines the case rather than risking a limitation-shaped verdict.
func identityConstraintsDecidable(el *parser.Element) bool {
	for _, child := range el.Children() {
		c, ok := child.(*parser.Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch c.Name().Local() {
		case "unique", "key", "keyref":
			if hasAttr(c, "ref") {
				return false
			}
		}
	}
	return true
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
//   - an inline anonymous <simpleType>/<complexType> on a local element/attribute
//     (not yet produced), or a bare <group>/<attributeGroup> lacking a ref (a
//     nested one is always a reference, so a bare one is malformed — declined).
//
// A <group ref>/<attributeGroup ref> IS produced (#177) and admitted: its target
// resolves genuinely at finalize (or fails src-resolve). Real structural
// violations the producer DOES reject (a nested <all>, a mixed
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
// content) are all within the producer's decidable subset. A <group ref> content
// child and an <attributeGroup ref> are admitted (produced, #177); a bare
// <group>/<attributeGroup> without a ref, or a stray <simpleContent>/<openContent>
// at this level, declines.
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
		case "assert":
			// Produced (#178) into {assertions}. An Assertion carries only an opaque
			// XPath Expression record with no rejectable state, so admitting it can
			// never turn into a fabricated "invalid" verdict.
		case "group":
			if !hasAttr(el, "ref") {
				return false // a content <group> is always a reference; a bare one is malformed — decline
			}
		case "attributeGroup":
			if !hasAttr(el, "ref") {
				return false // a nested <attributeGroup> is always a reference; a bare one is malformed — decline
			}
		default:
			// simpleContent/complexContent/openContent or any other name at this
			// level: not produced — decline.
			return false
		}
	}
	return true
}

// modelGroupDecidable reports whether every particle child of a model group
// (<sequence>/<choice>/<all>) is within the producer's decidable subset: nested
// model groups recurse, <element> must carry no inline anonymous type and only
// decidable identity constraints (produced for local declarations too, #178),
// <any> is fine, and a <group ref> is produced (#177). A bare <group> without a
// ref (a nested group is always a reference) or any other child declines.
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
			if !identityConstraintsDecidable(el) {
				return false // ref= identity constraint — not yet produced (#178)
			}
		case "sequence", "choice", "all":
			if !modelGroupDecidable(el) {
				return false
			}
		case "group":
			if !hasAttr(el, "ref") {
				return false // a nested <group> is always a reference; a bare one is malformed — decline
			}
		default:
			// any other child: not produced — decline.
			return false
		}
	}
	return true
}

// groupDecidable reports whether a top-level named <group> definition (§3.7.2) is
// within the producer's decidable subset: it must carry a name (the definition
// form; a top-level <group ref> is malformed) and its single all/choice/sequence
// body's particles must all be decidable. A missing body still produces genuinely
// (mgd-props-correct rejects it), so it is admitted.
func groupDecidable(el *parser.Element) bool {
	if !hasAttr(el, "name") || hasAttr(el, "ref") {
		return false
	}
	for _, child := range el.Children() {
		c, ok := child.(*parser.Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch c.Name().Local() {
		case "annotation":
			// Harmless.
		case "all", "sequence", "choice":
			if !modelGroupDecidable(c) {
				return false
			}
		default:
			// A <group> body is only all/choice/sequence; anything else is out of
			// the produced shape — decline.
			return false
		}
	}
	return true
}

// attributeGroupDecidable reports whether a top-level named <attributeGroup>
// definition (§3.6.2) is within the producer's decidable subset: it must carry a
// name, and its children must be only <attribute> (no inline anonymous type),
// <attributeGroup ref>, or <anyAttribute> — the shapes the producer folds in
// (§3.6.2.1/§3.6.2.2). A dangling or circular ref is decided genuinely at
// producer/finalize time, so it is admitted.
func attributeGroupDecidable(el *parser.Element) bool {
	if !hasAttr(el, "name") || hasAttr(el, "ref") {
		return false
	}
	for _, child := range el.Children() {
		c, ok := child.(*parser.Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch c.Name().Local() {
		case "annotation", "anyAttribute":
			// Harmless / produced.
		case "attribute":
			if childXSD(c, "simpleType") != nil {
				return false // inline anonymous attribute type — not yet produced
			}
		case "attributeGroup":
			if !hasAttr(c, "ref") {
				return false // a nested <attributeGroup> is always a reference; a bare one is malformed — decline
			}
		default:
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
// <enumeration> facet (still not produced). An <assertion> facet IS produced
// (#178, one assertions facet per restriction, Datatypes §4.3.13) and is
// admitted. An inline <simpleType> base child (the supported anonymous nested
// base, §3.16.3 clause 2) is recursed into with the same checks.
// src-simple-type §3.16.3.
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
		case "enumeration":
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
// local, as XSD schema-element attributes (name, ref, …) carry no namespace. It
// is the presence-only face of elementAttr, not a second scan of its own (STYLE
// D3).
func hasAttr(el *parser.Element, local string) bool {
	_, ok := elementAttr(el, local)
	return ok
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
