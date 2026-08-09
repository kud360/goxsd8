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
// (§4.2.3), chameleon coercion included (§F.1), an <xs:override> child (§4.2.5,
// which composes the overridden document as an include of its §F.2-transformed
// self), or an <xs:import> child (§4.2.6.2, which coerces nothing: the imported
// document keeps its own namespace) — and maps top-level
// <simpleType>/<element>/<attribute>/<attributeGroup>/<group>/<notation> and the
// decidable subset of <complexType> (implicit content, both <complexContent>
// alternants, and <simpleContent> <extension>; its particles including <group
// ref>, local element/attribute declarations, attribute uses including
// <attributeGroup ref>, wildcards, <openContent> and the schema-level
// <defaultOpenContent> it falls back to, and <assert> assertions) into xsd
// components, maps the name= identity constraints of global and local <element>s
// (#178), seeds the ur-type xs:anyType, resolves cross-references, and rejects
// duplicate top-level names within a kind. The ref= identity-constraint form and
// the not-yet-produced complexType forms (<simpleContent> <restriction>, inline
// anonymous local types) are SILENTLY SKIPPED or declined (§3.1.2 permits
// ignoring a not-yet-produced representation), NOT rejected.
//
// The EXTENSION forms are DECIDED as of #336: Parse builds <complexContent>
// <extension> and <simpleContent> <extension> types (#228; §3.4.2.3.3 clause 4.2,
// §3.4.2.2 cases 3-5), cos-ct-extends (§3.4.6.2) judges them (#264), and every
// §3.4.2 base fold its case-1 clauses read is now done — clause 1.2's
// {attribute uses} (§3.4.2.4 clause 3, #401), clause 1.3's {attribute wildcard}
// (§3.4.2.5 clause 2.2, #265) and clause 1.7's {assertions} (§3.4.2.1 clause 1,
// #346). THREE cos-ct-extends clauses cannot reject inside the admitted shape —
// 1.5, 1.1 and 2.2 — and they cannot for two DIFFERENT reasons, which this
// inventory keeps apart.
//
// Clause 1.5 (two-step derivability) is an ENGINE approximation: it is proven
// only for a pure-extension chain, and a chain that mixes extension and
// restriction steps is accepted unconditionally (GAP(xsd) in
// xsd/complexextension.go, follow-up #392). That marker, not this comment,
// carries its retirement.
//
// Clauses 1.1 and 2.2 — "B.{final} does not contain extension", for a complex
// and a simple base respectively — are instead DATA-DEAD: both checks are
// written, correct and tested, but nothing maps {final} from final=/finalDefault=
// (every producer construction site passes a literal nil: parser/produce_complex.go
// for complex types, parser/produce.go for simple ones), so finalContains never
// sees extension and both clauses pass VACUOUSLY. That is a producer MAPPING
// gap, not an engine one. Until #436 maps {final}, these two are
// unreachable-to-reject and the count here is three; when #436 lands they go
// live and it drops back to one — clause 1.5 alone. Case 2 (simple base) is
// therefore complete only in its clause 2.1.
//
// All three are UNDER-rejections — the lane can report "valid" for a schema a
// complete processor rejects, never "invalid" for a valid one — so they are
// recorded gaps of the same shape as the ones the admitted <restriction> path
// already carries, not fabricated verdicts.
//
// # Why "Parse returns nil" is not, by itself, evidence of validity
//
// Because the producer silently skips the representations it does not yet build,
// a document whose top-level content includes (say) an invalid <group> or an
// undecidable <complexType> form alongside valid simpleType/element/attribute
// would still assemble with no error — a FALSE ACCEPT. §3.1.2's licence
// to ignore a representation is an implementation choice about what to BUILD; it
// does not make the spec consider such a document valid: the invalid complexType
// still makes the document schema-INVALID under sch-props-correct clause 1
// (§3.17.6.1), whichever cvc-complex-type/cos-* rule it violates (oracle
// grounding, issue #175). So "Parse returns nil" is genuine evidence of validity
// ONLY when the top-level content is PROVABLY CONFINED to what the producer
// actually processes.
//
// Since #242 that qualifier binds over a CLOSURE, not one document. The assembly
// reads the whole <include>/<override>/<import> closure, so a composed document
// holding a skipped representation false-accepts exactly as a root one would. So
// the harness runs parser.ParseReport, which reports the DOCUMENT SET it
// assembled, and gates every document of that set on the allowlist below
// (closureDecidable, conformance/schema_closure.go). Until #272 the harness
// instead re-walked §4.2's composition edges itself, because parser.Parse could
// not be asked which documents it read; the gated set is now the assembled set by
// construction rather than by two walks agreeing, which is what closes the
// under-gating hazard — a document the harness missed but the parser read would
// be a document whose shape was never gated, the false accept back again.
//
// # The decidable shape (the strict top-level allowlist)
//
// execSchemaCase therefore decides a case only after confirming the whole shape
// of every document in its closure is confined to what the producer checks, and
// DECLINES (Fail) anything else:
//
//  1. Readability. parser.ReadDocument is run on the root before anything else,
//     and the assembly reads every composed document through it. ANY error
//     DECLINES the case (Fail), never a validity verdict: a ReadDocument error
//     does not distinguish a genuine XML well-formedness fault from a parser
//     encoding LIMITATION. Well-formed UTF-16 input (BOM FF FE) is currently
//     rejected as "invalid UTF-8" because UTF-16 decoding is not yet implemented,
//     so treating that as observed-invalid would fabricate an "invalid" verdict
//     for a well-formed document — a wrong-reason pass that would flip pass→fail
//     once UTF-16 decoding lands (a separate change). So malformed XML is NOT a
//     claimed schema-well-formedness sub-cohort here; it is a declined recorded
//     gap. For a COMPOSED document the decline comes through the report: a
//     document that resolved but could not be read is one the assembly never took
//     in, recorded as parser.UnfollowedUnreadable, and the error it also returns
//     completes the conjunction in step 4.
//  2. Root identity. If the root is not <schema> (IsSchema false) the case is
//     DECLINED: §3.17.2 explicitly does NOT require <schema> to be the document
//     root, so Parse's error there is a plain non-xsderr Go precondition fault,
//     not a sch-props-correct rejection — not decidable for this lane. Inventing
//     a "root must be <schema>" rejection would overreach (oracle grounding). An
//     INCLUDED document that is not a <schema> is the opposite case and is NOT
//     declined: src-include clause 1 makes that a genuine rejection, which the
//     assembly emits, and the document never enters the report to be gated.
//  3. Top-level allowlist. Every top-level child element must be xsd:annotation,
//     xsd:include, xsd:import, xsd:override, xsd:simpleType, xsd:element,
//     xsd:attribute, xsd:complexType, xsd:attributeGroup (named definition),
//     xsd:group (named definition), xsd:defaultOpenContent (only in the shape
//     its declaration allows: with the <any> child its content model requires
//     and, if mode= is present at all, a value from its interleave|suffix
//     enumeration) or xsd:notation or xsd:redefine — anything else at top level
//     (a non-xsd element, or an out-of-set local name) closes the false-accept
//     gap above by DECLINING the whole case. Within the allowed kinds:
//     - include: always admitted (#242). Its own content model is (annotation?),
//       so it contributes nothing the producer could silently skip; the
//       decidability of the document it POINTS AT is established by the closure
//       gate, which runs this same allowlist over every document the assembly
//       reported reading (and so, transitively, over that document's own
//       <include>s and <import>s) — not by this allowlist entry. src-include
//       (§4.2.3) itself imposes no shape constraint on the included document,
//       only existence and targetNamespace agreement, both of which the assembly
//       decides genuinely. An <include> whose schemaLocation does not resolve
//       yields no document to gate: the assembly reports that
//       (parser.UnfollowedLocationUnresolved) exactly as it does for an <import>
//       that yields no D2, and execSchemaCase declines only if the parse then
//       fails (#276).
//     - override: admitted (#183) when every child of it is a decidable source
//       declaration in its own right (overrideDecidable), because §F.2 clause 1
//       makes those children top-level declarations of the OVERRIDDEN document.
//       The document it points at, and the rest of §4.2.5's ·target set·, is
//       gated by the same closure gate that gates an <include>'s target, being in
//       the same reported document set; src-override's own clauses are then
//       enforced genuinely by the assembly.
//     - import: admitted at top level (#182) on the same reasoning. As for
//       include, a directive that yields no D2 (no schemaLocation, or one that
//       does not resolve) is REPORTED as unfollowed, and declines the case
//       only when the parse also fails — that failure being the fabricated
//       src-resolve rejection the missing components would produce. See the
//       Composition section below and parser.AssemblyReport.Unfollowed.
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
//     - complexType (top-level, or a <complexContent>/<simpleContent> derivation
//       reached transitively): must lie within the producer's decidable subset per
//       complexTypeDecidable — implicit content, either <complexContent>
//       alternant, or <simpleContent> <extension>, whose content model is
//       element/any/sequence/choice/all/<group ref>, whose attributes are local
//       <attribute>/<anyAttribute>/<attributeGroup ref>, whose <openContent> maps to
//       {open content} (#230) and whose <assert> children map to {assertions} (#178),
//       with no <simpleContent> <restriction> and no inline anonymous local type.
//       <simpleContent> <restriction> and the inline forms need a later producer slice,
//       so Produce declines them with a plain limitation error, not a spec verdict. A
//       <group ref>/ <attributeGroup ref> IS produced (#177): its target resolves (or
//       fails src-resolve) genuinely. A real structural violation inside an admitted
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
//       per restriction, Datatypes §4.3.13) and is admitted. An inline
//       <simpleType> base child (the genuinely-supported anonymous nested base,
//       §3.16.3 clause 2) is recursed into with the same two checks. The
//       restriction's base=/inline-child exactly-one arrangement is NOT
//       pre-checked: that IS the genuine src-simple-type clause 2 rule Produce
//       correctly enforces, so a violation flows through as a real decidable
//       rejection.
//     - annotation: always allowed, no further check.
//  4. Decide. When every document of the closure passes, observed =
//     (parser.ParseReport's err == nil): a nil error is genuine evidence of
//     validity (no document of the assembly has any of the violations checked
//     above, so a real one would surface), and a non-nil error is a REAL,
//     implemented rejection (src-include §4.2.3, src-import and
//     src-import-noselfimport §4.2.6.2, sch-props-correct clause 2 duplicate-name
//     §3.17.6.1, src-element §3.3.3, src-attribute §3.2.3, src-simple-type
//     §3.16.3, src-override §4.2.5, src-resolve §3.17.6.2, st-props-correct,
//     src-identity-constraint §3.11.3, c-props-correct §3.11.6.1, n-props-correct
//     §3.14.6, and for the complex-type subset src-ct §3.4.3, cos-all-limited
//     §3.8.6, src-wildcard §3.10.3, p-props-correct §3.9.6, cos-nonambig §3.8.6.4,
//     cos-element-consistent §3.8.6.3, ct-props-correct §3.4.6.1 and
//     derivation-ok-restriction §3.4.6.3), never a fabricated one — the shape
//     allowlist excludes every case whose rejection would be a
//     limitation-in-disguise. The case Passes iff observed agrees with the suite's
//     declared validity.
//
//     No error-type discrimination (errors.As over *xsderr.Error) is needed to
//     make that trustworthy, because steps 1-3 have ruled out the
//     non-verdict failure modes ACROSS THE WHOLE CLOSURE: the root was
//     independently confirmed resolvable, readable and <schema>-rooted before the
//     assembly ran, and every document the assembly did take in is reported and
//     shape-gated. The plain non-xsderr errors ParseReport can otherwise return —
//     an unresolvable root, an I/O or encoding failure, a non-schema root, and an
//     <include>/<override> carrying no schemaLocation at all, which is a grammar
//     fault no Schema Representation Constraint covers (parse.go's compose) — are
//     exactly the modes the root pre-check and the Unfollowed conjunction
//     eliminate: the last two are reported as parser.UnfollowedUnreadable and
//     parser.UnfollowedNoSchemaLocation, so a case failing on either declines
//     rather than being read as a verdict. What remains is spec verdicts.
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
// producer artifact. The executor relies on Finalize's per-kind indexByName for
// exactly this, so no cross-kind duplicate check is done here (that would be a
// false-INVALID verdict, a ratchet regression risk).
//
// # Why no false ratchet-corrupting pass is possible
//
// Every "invalid" verdict this lane emits comes from ONE source: parser.Parse
// rejecting an assembly EVERY document of which already passed the allowlist.
// ReadDocument errors never produce an "invalid" verdict — they decline (step 1)
// — precisely because a ReadDocument error can be a parser encoding limitation
// (well-formed UTF-16 misread as invalid UTF-8) rather than a real violation, and
// turning that into "invalid" would fabricate a verdict for a well-formed
// document.
//
// A "valid" verdict coincides only with a truly-valid ground truth: a truly-valid
// assembly (by definition) has none of the checked violations, so Parse
// correctly finds none. An "invalid" verdict coincides only with truly-invalid
// ground truth via a REAL implemented violation — never a fabricated one, since
// the shape allowlist excludes every form (inline element/attribute types,
// list/union/enumeration simpleTypes, ref= identity constraints, the
// not-yet-produced complexType forms — <simpleContent> <restriction>, inline
// anonymous local types — and the produced-but-unjudged extension
// forms) where the producer's rejection would be a limitation rather than a spec
// violation, or its silence a missing rejection. A suite-invalid case whose only
// defect is a rule finalize does NOT yet check (cos-content-act-restrict —
// derivation-ok-restriction clause 2.4.2, #263 — cos-ns-subset, #265, or the Open
// Content half of derivation-ok-restriction clause 2.4, which xsd.Schema's
// content-model automaton fails open on: xsd/contentrestricts.go's GAP(xsd), live
// for a produced {open content} since #230) is produced cleanly, so the lane
// observes "valid", disagrees with the suite, and records a still-failing gap —
// never a wrong "invalid" pass. The remaining risk the allowlist closes is the
// VACUOUS pass — a document of entirely skipped top-level content that would
// otherwise always "pass" through a producer doing nothing — which is why step 3
// confines the whole top level of EVERY document in the closure to the processed
// kinds and the decidable complexType subset.
//
// # Composition: <include>, <import> and <override> decided, redefine deferred
//
// <xs:include>, chameleon inclusion included, is DECIDED as of #242,
// <xs:import> as of #182 and <xs:override> as of #183: the assembly follows all
// three closures (#179/#182/#183) and the closure gate covers every document in them,
// so an include/chameleon/import/override case is now decided for the same reason a
// single-document case is, not guessed. A composition directive that yields no D2 —
// an <import> with no schemaLocation, or an <include>, <override> or <import> whose
// schemaLocation does not resolve — is REPORTED as unfollowed
// (parser.AssemblyReport.Unfollowed) and declines the case only when the parse ALSO
// fails: the missing document's components are then absent from the assembly and the
// reference that wanted them failed src-resolve clauses 1-3 at finalize, a FABRICATED
// "invalid" verdict, the one direction that can corrupt the ratchet. The conjunction
// is the whole hazard (#276): where the parse succeeds despite the missing document,
// §4.2.3 clause 2.4's "not an error ... the inclusion must not be performed" is
// simply in force and the case is decided normally — the suite's own cl. 2.4 tests
// (MS-Schema schD8 and friends) depend on that. The directives are one hazard, not
// two: src-include clause 2.4 and src-import's "not an error" text are parallel, and
// src-resolve clause 4 (cl.qnr.nsdeclared) licenses a same-namespace reference
// (4.2.1) and a reference into a namespace with a PRESENT <import> element (4.2.2)
// alike, whether or not that import's schemaLocation resolved.
//
// Two import-adjacent gaps are the OTHER direction — fabricated "valid" — which
// can only cost wins, never corrupt: src-resolve clause 4 (cl.qnr.nsdeclared,
// §3.17.6.2) is not enforced, so a reference into a namespace the containing
// document never imported still resolves if another document of the assembly
// contributed it; and a namespace whose components are genuinely missing is not
// reported as a §5.3 missing component. Both make the lane observe "valid" where
// the suite says "invalid", which records a gap rather than a pass.
//
// An <xs:override> is admitted when every one of its children is itself a
// decidable source declaration (overrideDecidable): those children become
// top-level declarations of the overridden document (§F.2 clause 1), so an
// undecidable one would be an undecidable top-level declaration by another route.
// The document it points at, and every document that one <include>s or
// <override>s — §4.2.5's ·target set· — is gated by the same closure gate that
// covers an <include>'s target: all of them are documents the assembly read, so
// all of them are in its report. Verdicts on an admitted override case are
// genuine in both directions: the substituted declarations are really produced (a
// violation among them, such as a simple type left restricting itself, surfaces
// as the rule it breaks), and an unmatched override child is really ignored
// (§4.2.5) rather than added.
//
// <xs:redefine> is DECIDED as of #286, and for ALL FOUR redefinable kinds as of
// #505. The assembly follows it, so the document it names is in the
// closure and gated like any other, and the redefinition's own clauses —
// src-redefine clause 1 (a non-empty <redefine> whose schemaLocation does not
// resolve is an ERROR, unlike <include>), clause 5, clauses 6.1.1/6.1.2, clause
// 7.1, and src-expredef's requirement of a top-level definition item of the
// appropriate name and kind — are enforced genuinely. A redefining
// <simpleType>/<complexType> is paired with the {name}-·absent· original of
// src-expredef clause 1.1, so its self-derivation is a real derivation rather
// than the circularity the pairing exists to avoid, and each redefining child is
// gated by the predicate its own element type is gated by anywhere else
// (redefineDecidable). §F.2 clause 1's "or <redefine>" scope is no longer
// empty either: an <override> in force over a document now substitutes for that
// document's <redefine> children too, restricted to the four element types
// <redefine>'s own content model admits.
//
// # Still deferred
//
// Inline anonymous types on element/attribute, list/union/enumeration
// simpleTypes, ref= identity constraints and the not-yet-produced complexType
// forms named above widen in with later slices (exactly as the datatypes lane
// grew across #15/#57/#80); they stay DECLINED (Fail) recorded gaps here, never
// guessed. UPA and EDC landed with #180, derivation-ok-restriction with #262
// and cos-ct-extends with #264, so the admitted complexType cases those rules
// reject — restriction and extension alike, the latter admitted by #336 — are
// now decided; the cases still turning on cos-content-act-restrict (#263) stay
// failing gaps rather than wins until that lands.
//
// A schemaTest with MORE THAN ONE <ts:schemaDocument> child declares a SET of
// documents to be loaded "one by one, in order" (xsts.xsd, the suite's own
// catalog schema); the runner now carries all of them (caseSpec.extraDocs)
// instead of silently keeping one. This lane decides such a case only when the
// assembly from the FIRST document provably consumed every other declared one
// — which requires the first document's own <include>/<override>/<import> to
// name them, those being the only directives the assembly follows, and is then
// just the composition case above. A <redefine> can never supply that reachability: a
// document carrying one is DECLINED outright above, so nothing it names is ever
// reached. Documents genuinely independent of each other need several roots merged
// into one schema, which neither parser.ParseReport (one root) nor this harness
// offers, so those cases are DECLINED (extraDocsInClosure) rather than decided
// against a schema the suite did not declare.

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
// reads the root document to check the two preconditions parser.ParseReport
// answers with a plain Go error rather than a validity verdict, runs
// parser.ParseReport, gates the WHOLE <xs:include>/<xs:override>/<xs:import>
// closure it reports on the decidable top-level shape (closureDecidable, which
// runs schemaShapeDecidable on every document the assembly consumed), and agrees
// or disagrees with the suite's declared validity. A root it cannot resolve OR
// cannot read (any ReadDocument error, including a parser encoding limitation
// such as unsupported UTF-16), whose root element is not <schema>, or any
// document of whose closure falls outside the producer's decidable subset is
// DECLINED (Fail) as a recorded gap, never guessed. So is a case that both
// carries a directive naming no document and fails to parse — see the
// Unfollowed check below (#276).
//
// The resolver is a loader.Dir rooted at the case document's own directory and the
// root is named by its BASE name, because parser.ParseReport reads the root under
// exactly the location string it is handed (readRootDocument in parser/parse.go):
// passing the full path would give the root document a base URI of
// "…/sunData/SType/x" instead of "x", and every <include> in it would then resolve
// one directory tree away from where the resolver serves. The harness's own
// precondition read below therefore uses the SAME resolver and the SAME location
// string, so it reads byte-identically the document the assembly roots at.
func execSchemaCase(backend value.Backend, c caseSpec) Status {
	resolver := loader.Dir(filepath.Dir(c.doc))
	location := filepath.Base(c.doc)
	rc, _, err := resolver.Resolve("", location)
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
	// ParseReport, not Parse: the verdict needs the DOCUMENT SET the assembly
	// consumed, not only its components (#272).
	_, report, perr := parser.ParseReport(location, parser.WithResolver(resolver), parser.WithBackend(backend))
	// Only decide when EVERY document of the <include>/<override>/<import> closure
	// is confined to what the producer processes; otherwise a silently-skipped
	// invalid representation, in the root or in any composed document, could
	// false-accept.
	if !closureDecidable(report) {
		return Fail()
	}
	if !extraDocsInClosure(report, resolver, c) {
		return Fail()
	}
	// A directive that named no document is only half the fabricated-rejection
	// hazard (#276): the missing components matter solely when something referred
	// to them, and that shows up here, as a failed parse whose src-resolve clause
	// 1-3 error the spec does not attach to the missing document (§5.3). A parse
	// that SUCCEEDED past an unfollowed directive fabricated nothing — §4.2.3
	// clause 2.4's "not an error ... the inclusion must not be performed" is
	// exactly that outcome — so the case is still decided. Only the conjunction
	// declines.
	if len(report.Unfollowed()) > 0 && perr != nil {
		return Fail()
	}
	return decideSchema(perr == nil, c.expect.wantsValid())
}

// extraDocsInClosure reports whether every FURTHER document the case declares
// beyond c.doc was consumed by the assembly rooted at c.doc — the only condition
// under which parser.ParseReport, which is handed one root, nonetheless
// assembles the whole declared set.
//
// A schemaTest may list several <schemaDocument> children, and the suite's own
// catalog schema defines that as "run as if the schema documents given were
// loaded one by one, in order": the case is the SET, not any member of it. The
// condition holds when the first document's own
// <include>/<override>/<import>/<redefine> names the others — the four
// directives the assembly follows — so it composes them. The check exists so
// that a case whose documents the parser's OWN composition constructs already
// link is decided on the declared set rather than declined for its member count.
// When the condition does not hold (documents genuinely independent), the
// harness has no mechanism to merge several roots into one schema, so any
// verdict it emitted would be a verdict on a DIFFERENT schema than the one
// declared. It therefore DECLINES, as it declines every other shape it cannot
// decide for the right reason, rather than loading an arbitrary member or
// ignoring the rest.
//
// Each extra document is resolved through the SAME resolver the parse used, under
// a location string relative to the same root directory, so the resolved identity
// compared against the report is in the report's own format (closureReached). A
// path that will not resolve at all declines for the same reason as an
// unresolvable root: an unreadable document is a gap, never a validity verdict.
func extraDocsInClosure(report *parser.AssemblyReport, resolver loader.Resolver, c caseSpec) bool {
	root := filepath.Dir(c.doc)
	for _, extra := range c.extraDocs {
		location, err := filepath.Rel(root, extra)
		if err != nil {
			return false
		}
		rc, resolved, err := resolver.Resolve("", filepath.ToSlash(location))
		if err != nil {
			return false
		}
		// Read-only handle: a close failure cannot change what the assembly read,
		// so it cannot affect the verdict (STYLE S3).
		_ = rc.Close()
		if !closureReached(report, resolved) {
			return false
		}
	}
	return true
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
		case "include", "import":
			// Admitted (#242 for include, #182 for import). Neither contributes a
			// component of its own — <include>'s content model is (annotation?) and
			// <import>'s likewise — so there is nothing here for the producer to
			// silently skip. What each POINTS AT is the thing that could be skipped,
			// and that is gated by closureDecidable, which runs this same function
			// over every document the assembly reported reading. src-include's
			// (§4.2.3) and src-import's (§4.2.6.2) own clauses are then enforced
			// genuinely by the assembly. A directive that yields no D2 at all — an
			// <import> with no schemaLocation, or either directive with one that does
			// not resolve — is reported as unfollowed, not judged here, and declines
			// the case only alongside a failed parse: see
			// parser.AssemblyReport.Unfollowed and execSchemaCase.
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
		case "override":
			// Admitted (#183). The document it points at is gated by the closure
			// walk, exactly as an <include>'s target is; its own children become top-
			// level declarations of THAT document (§4.2.5, §F.2 clause 1) and are
			// gated here.
			if !overrideDecidable(el) {
				return false
			}
		case "defaultOpenContent":
			// Read (#230) by every complex type of this document that has no
			// <openContent> of its own (§3.4.2.3.3 clause 5.2), so it is no longer
			// silently skipped. It is admitted only with the <any> child its content
			// model makes mandatory: without one the producer rejects it as a grammar
			// fault, which is a real verdict, but only for a document that also holds
			// a complex type reaching clause 5.2 — admitting the childless form would
			// therefore make the verdict depend on unrelated content.
			if childXSD(el, "any") == nil {
				return false
			}
			// Its mode enumeration is (interleave|suffix) — none is legal on a type's
			// OWN <openContent> but not here, and every other token is out of the
			// enumeration outright. Both are rejected by the same lazy path as the
			// childless form, so the same decline applies: one principle, applied to
			// the whole shape rather than half of it.
			if mode, present := el.Attr("mode"); present && mode != "interleave" && mode != "suffix" {
				return false
			}
		case "redefine":
			// Admitted (#286, all four redefinable kinds as of #505). Like
			// <override> the document it points at is gated by the closure walk,
			// and its own children are gated here, each by the predicate its own
			// element type is gated by anywhere else.
			if !redefineDecidable(el) {
				return false
			}
		default:
			// Any other local name: silently skipped by the producer, so a nil
			// verdict there would be vacuous — decline the whole case.
			return false
		}
	}
	return true
}

// redefineDecidable reports whether every child of an <xs:redefine> (§4.2.4) is
// a definition the producer decides genuinely once the redefinition has been
// applied (#286). A redefining child becomes a top-level definition of the
// REDEFINING document (src-expredef clause 1.2 / clause 2, §4.2.4 clause 4.1.1),
// so each is gated by the very predicate a top-level child of any document is
// gated by — with one subtraction.
//
// There is no subtraction as of #505: a redefining <complexType> is produced,
// paired with the {name}-·absent· original src-expredef clause 1.1 defines, so it
// is gated by complexTypeDecidable exactly as a top-level one is. The pairing
// puts the original in no by-name index, which narrows nothing this predicate can
// see — every clause a redefinition turns on reads the base COMPONENT, and
// finalize reaches it through the {base type definition} slot. What it does NOT
// reach is the original itself: Phase D's derivation checks quantify over the
// schema's type definitions and so produce no verdict for it (GAP(xsd),
// xsd/complexderivation.go). That is an under-rejection on a component the
// redefinition's own verdict already covers structurally, the same safe direction
// as the two fail-open clauses below.
//
// A child with no name= declines rather than being waved through: the pairing is
// keyed on (element type, name), and src-expredef requires a top-level definition
// item of that name and kind in the redefined document, so a nameless child is
// reported as a grammar fault rather than decided.
//
// Two fail-open clauses remain inside an admitted case, both in the SAFE
// direction: src-redefine clause 6.2.2 (the redefining group must accept a subset
// of the original's element sequences) and clause 7.2.2 (the redefining attribute
// group must satisfy derivation-ok-restriction clause 3 against the original) are
// not decided, so a case turning on either observes "valid" where the suite says
// "invalid" — a recorded gap, never a pass. The document the <redefine> POINTS AT
// is gated by closureDecidable, exactly as an <include>'s target is.
func redefineDecidable(el *parser.Element) bool {
	for _, child := range el.Children() {
		c, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		if c.Name().Space() != xsd.XMLSchemaNS {
			return false
		}
		if c.Name().Local() == "annotation" {
			continue
		}
		if !hasAttr(c, "name") {
			return false
		}
		switch c.Name().Local() {
		case "simpleType":
			if !simpleTypeDecidable(c) {
				return false
			}
		case "complexType":
			if !complexTypeDecidable(c) {
				return false
			}
		case "group":
			if !groupDecidable(c) {
				return false
			}
		case "attributeGroup":
			if !attributeGroupDecidable(c) {
				return false
			}
		default:
			// Every element type §4.2.4's content model does not admit at all.
			return false
		}
	}
	return true
}

// overrideDecidable reports whether every child of an <xs:override> (§4.2.5) is a
// source declaration the producer decides genuinely once ·override
// pre-processing· has substituted it into the overridden document (#183). A child
// that overrides something "will become a top-level declaration" of that document
// (§3.1.2's note on <override> children), so each is gated by the very predicate
// a top-level child of any document is gated by — the substituted declaration is
// produced by the overridden document's producer, through the same code path.
//
// A child with no name= declines rather than being waved through: §F.2 clause 1
// matches on (element type, name), so the parser can only ignore such a child,
// and ignoring a declaration is precisely the vacuous shape this allowlist
// exists to refuse. Any other element type declines for the same reason.
//
// The document the <override> POINTS AT is NOT gated here; that is the closure
// gate's job (closureDecidable), which runs schemaShapeDecidable over it exactly
// as it does for an <include>'s target, both being documents the assembly
// reported reading.
func overrideDecidable(el *parser.Element) bool {
	for _, child := range el.Children() {
		c, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		if c.Name().Space() != xsd.XMLSchemaNS {
			return false
		}
		if c.Name().Local() == "annotation" {
			continue
		}
		if !hasAttr(c, "name") {
			return false
		}
		switch c.Name().Local() {
		case "notation":
			// Always decidable, as at top level.
		case "element":
			if !elementDecidable(c) {
				return false
			}
		case "attribute":
			if !attributeDecidable(c) {
				return false
			}
		case "simpleType":
			if !simpleTypeDecidable(c) {
				return false
			}
		case "complexType":
			if !complexTypeDecidable(c) {
				return false
			}
		case "group":
			if !groupDecidable(c) {
				return false
			}
		case "attributeGroup":
			if !attributeGroupDecidable(c) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// elementDecidable reports whether a TOP-LEVEL <element> is in the form the
// producer decides genuinely. A bare element (no type=) defaults to xs:anyType
// (§3.3.2.1 case 4), which the producer seeds as a Complex Type Definition
// (§3.4.7), so it resolves at finalize and is decided genuinely — type= is not
// required.
//
// An inline <complexType> child IS produced (§3.3.2.1 dcl.elt.common clause 1,
// #340) and is admitted when the anonymous type it maps to is itself decidable
// — see anonymousComplexTypeDecidable, which is narrower than
// complexTypeDecidable and says why. An inline <simpleType> child is still
// declined: #229 and #340 widened tier 1 for the LOCAL <simpleType> and for the
// <complexType> on both paths, leaving the GLOBAL inline <simpleType> as the one
// unproduced tier-1 shape, which the producer declines as the limitation it is.
//
// Its <unique>/<key>/<keyref> children impose no condition of their own: #178
// produced the name= form and #240 the ref= form, so BOTH are mapped and both
// src-identity-constraint (§3.11.3) and c-props-correct (§3.11.6.1) rejections on
// them are genuine.
func elementDecidable(el *parser.Element) bool {
	if childXSD(el, "simpleType") != nil {
		return false
	}
	inline := childXSD(el, "complexType")
	return inline == nil || anonymousComplexTypeDecidable(inline)
}

// anonymousComplexTypeDecidable reports whether the ANONYMOUS complex type of an
// inline <complexType> child of an <element> is decidable. It is deliberately
// NARROWER than complexTypeDecidable, which judges a type the finalized Schema
// holds in its {type definitions}: an anonymous type is never a member of that
// set (§3.17.2 scopes the property to the <complexType> children OF <schema>;
// xsd/typedefinition.go records why registering one would be worse), so every
// finalize pass that quantifies over it silently produces NO verdict for the
// anonymous type — see the GAP marker on parser's produceComplexType for the
// exact list and its issues (#414, #438).
//
// This gate therefore admits only the IMPLICIT-CONTENT form — no <simpleContent>
// and no <complexContent> child — because on that shape the two MISSING
// attribute folds are provably the identity: §3.4.2.3.2 makes the {base type
// definition} xs:anyType with {derivation method} restriction, §3.4.7 gives
// xs:anyType an empty {attribute uses} so §3.4.2.4 clause 3's fold adds nothing,
// and §3.4.2.5 clause 2 unions the base's wildcard only for an EXTENSION, which
// this form is not. What stays genuinely unenforced on the admitted shape is
// narrower but real: cos-nonambig and cos-element-consistent inside the
// anonymous content model, and ct-props-correct clause 4's attribute-name
// uniqueness. All three are UNDER-rejections — this lane can report "valid" for
// a schema a complete processor rejects, never "invalid" for a valid one — which
// is the safe direction for a ratchet.
//
// The nesting recursion is contentDecidable's: an inline <complexType> deeper in
// the tree is reached through modelGroupDecidable → localElementDecidable and
// comes back through here, so the narrowing applies at every depth.
func anonymousComplexTypeDecidable(el *parser.Element) bool {
	if childXSD(el, "simpleContent") != nil || childXSD(el, "complexContent") != nil {
		return false
	}
	return contentDecidable(el)
}

// complexTypeDecidable reports whether a <complexType> (top-level, or a nested
// derivation reached through <complexContent>/<simpleContent>) lies within the
// producer's decidable subset — the shapes it fully builds AND finalize fully
// judges, so any Produce error on it is a REAL structural violation
// (src-ct/cos-all-limited/src-wildcard/src-attribute/p-props-correct/src-resolve)
// and its silence is a real absence of one. It declines:
//
//   - every shape the producer declines with a plain "not yet produced"
//     limitation error — <simpleContent> <restriction> (§3.4.2.2 cases 1-2
//     synthesize an anonymous simple type from the restriction's facets) and a
//     bare <group>/<attributeGroup> lacking a ref (a nested one is always a
//     reference, so a bare one is malformed). An inline anonymous <simpleType>
//     on a local element or attribute IS produced (#229) and an inline anonymous
//     <complexType> on a local OR global element is produced too (#340); neither
//     declines here — see localElementDecidable/elementDecidable, and
//     anonymousComplexTypeDecidable for the separate, narrower gate the
//     anonymous complex-type shape passes through. An <openContent> is produced
//     as of #230 (§3.4.2.3.3 clauses 5-6) and is admitted by contentDecidable
//     wherever the schema for schema documents allows one; a MISPLACED one —
//     beside <simpleContent>/<complexContent>, or directly under
//     <complexContent> — is rejected by the producer as the grammar fault it is,
//     so it needs no decline of its own;
//   - a <complexContent> carrying NEITHER alternant. §3.4.2.3 requires one of
//     them, and the producer says so, but as a grammar fault about the source
//     item rather than a rule verdict, so it is declined like any limitation.
//
// Both EXTENSION forms are ADMITTED as of #336. The producer builds them (#228),
// cos-ct-extends (§3.4.6.2) judges them (#264), and its case-1 clauses read only
// folds that are now done — 1.2 over §3.4.2.4 clause 3's {attribute uses}
// (#401), 1.3 over §3.4.2.5 clause 2.2's {attribute wildcard} (#265) and 1.7
// over §3.4.2.1 clause 1's {assertions} (#346). Three of its clauses still
// cannot reject: case 1's 1.5 (two-step derivability), proven only for a
// pure-extension chain and otherwise accepted unconditionally, an engine
// approximation (GAP(xsd), xsd/complexextension.go, #392); and — until #436 maps
// {final} — case 1's 1.1 and case 2's 2.2, whose "B.{final} does not contain
// extension" test reads a property the producer never populates and so passes
// vacuously, leaving case 2 complete only in its clause 2.1. The file comment
// above carries the full inventory and why the two kinds differ. All three are
// UNDER-rejections, the same safe direction as cos-nonambig and ct-props-correct
// clause 4 already are on the admitted <restriction> path, so they bound what
// this lane can claim rather than letting it fabricate an "invalid".
//
// A <group ref>/<attributeGroup ref> IS produced (#177) and admitted: its target
// resolves genuinely at finalize (or fails src-resolve). Real structural
// violations the producer DOES reject (a nested <all>, a mixed
// mismatch, a both-namespace-forms wildcard, a bad occurrence) are NOT declined:
// admitting them is safe because the producer's rejection is the right reason.
func complexTypeDecidable(el *parser.Element) bool {
	if sc := childXSD(el, "simpleContent"); sc != nil {
		ext := childXSD(sc, "extension")
		if ext == nil {
			return false // <restriction>, or neither alternant — see above
		}
		return simpleContentExtensionDecidable(ext)
	}
	if cc := childXSD(el, "complexContent"); cc != nil {
		derivation := childXSD(cc, "restriction")
		if derivation == nil {
			derivation = childXSD(cc, "extension")
		}
		if derivation == nil {
			return false // neither alternant — see above
		}
		return contentDecidable(derivation)
	}
	return contentDecidable(el)
}

// simpleContentExtensionDecidable reports whether a <simpleContent> <extension>
// is in the shape the producer decides genuinely. It is contentDecidable MINUS
// the particles: xs:simpleExtensionType's content model is (annotation?,
// ((attribute | attributeGroup)*, anyAttribute?), assert*), so a
// <sequence>/<choice>/<all>/<group> child is a grammar fault — but §3.4.2.2
// computes {content type} from the resolved base alone (cases 3-5) and
// produceSimpleContent never reads a particle, so the producer SILENTLY DROPS
// such a child and returns no error at all. That silence is exactly the false
// accept this allowlist exists to refuse, so the shape declines here, while the
// attribute children the producer really does fold in (§3.4.2.4) go through
// contentDecidable unchanged.
func simpleContentExtensionDecidable(ext *parser.Element) bool {
	for _, child := range ext.Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch el.Name().Local() {
		case "sequence", "choice", "all", "group":
			return false
		}
	}
	return contentDecidable(ext)
}

// contentDecidable reports whether the content-model child and attribute children
// of a <complexType> (implicit content) or <restriction> (explicit complex
// content) are all within the producer's decidable subset. A <group ref> content
// child and an <attributeGroup ref> are admitted (produced, #177); an
// <openContent> is admitted (produced, #230); a bare <group>/<attributeGroup>
// without a ref, or a stray <simpleContent> at this level, declines. A local
// <attribute>'s inline anonymous <simpleType> is admitted when the inline type
// itself is decidable — see localAttributeDecidable.
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
			if !localAttributeDecidable(el) {
				return false
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
		case "openContent":
			// Produced (#230) into {open content} (§3.4.2.3.3 clauses 5-6). Its own
			// src-ct clause 3/4 violations and its <any>'s src-wildcard/w-props-correct
			// ones are genuine rejections, so nothing here is limitation-shaped.
		default:
			// simpleContent/complexContent or any other name at this level: not
			// produced — decline.
			return false
		}
	}
	return true
}

// modelGroupDecidable reports whether every particle child of a model group
// (<sequence>/<choice>/<all>) is within the producer's decidable subset: nested
// model groups recurse, <element> must be locally decidable
// (localElementDecidable) — its identity constraints impose no condition of
// their own, both forms being produced for local declarations too (#178, #240) —
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
			if !localElementDecidable(el) {
				return false
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
// name, and its children must be only <attribute> (locally decidable — the group's
// <attribute> children map to LOCAL declarations, §3.6.2.1), <attributeGroup ref>,
// or <anyAttribute> — the shapes the producer folds in (§3.6.2.1/§3.6.2.2). A
// dangling or circular ref is decided genuinely at producer/finalize time, so it
// is admitted.
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
			if !localAttributeDecidable(c) {
				return false
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

// attributeDecidable reports whether a TOP-LEVEL <attribute> is decidable: it
// must have no inline <simpleType> child. type= is NOT required — a bare
// attribute defaults to xs:anySimpleType (§3.2.2.1), which builtin.Seed always
// seeds, so it resolves and is decided genuinely.
//
// The inline decline is deliberately ASYMMETRIC with localAttributeDecidable,
// which admits one: #229 widened the producer's §3.2.2.2 dcl.att.local mapping
// only. The global mapping dcl.att.global (§3.2.2.1) still declines an inline
// <simpleType> with a limitation-shaped src-attribute error, so admitting a
// global one here would report a fabricated "invalid".
func attributeDecidable(el *parser.Element) bool {
	return childXSD(el, "simpleType") == nil
}

// localElementDecidable reports whether a LOCAL <element> (a model group's
// particle child) is within the producer's decidable subset. The asymmetry with
// elementDecidable, its top-level sibling, is deliberate and is #229's whole
// point: the producer maps §3.3.2.1 dcl.elt.common clause 1 for a LOCAL element,
// so an inline anonymous <simpleType> is genuinely produced and admitted here —
// provided the inline type is itself in the produced simple-type subset, since
// otherwise the decline would move inside constructSimpleType and re-shape a
// limitation as a verdict.
//
// An inline <complexType> is produced too as of #340, on this path and on the
// global one alike (§3.3.2.1's tier 1 is a COMMON rule), and is admitted under
// the same proviso — see anonymousComplexTypeDecidable for why that gate is
// narrower than complexTypeDecidable.
func localElementDecidable(el *parser.Element) bool {
	if ct := childXSD(el, "complexType"); ct != nil && !anonymousComplexTypeDecidable(ct) {
		return false
	}
	inline := childXSD(el, "simpleType")
	return inline == nil || simpleTypeDecidable(inline)
}

// localAttributeDecidable reports whether a LOCAL <attribute> — a child of a
// <complexType>/<restriction> or of an <attributeGroup> definition, both of
// which map through §3.2.2.2 dcl.att.local — is within the producer's decidable
// subset. Like localElementDecidable it admits an inline anonymous <simpleType>
// whose own shape is produced (#229), and for the same reason declines one that
// is not.
func localAttributeDecidable(el *parser.Element) bool {
	inline := childXSD(el, "simpleType")
	return inline == nil || simpleTypeDecidable(inline)
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
// is the presence-only face of parser.Element.Attr, not a second scan of its own
// (STYLE D3) — the lookup itself lives in the parser package, which is the one
// implementation both this harness and the assembly use (#272).
func hasAttr(el *parser.Element, local string) bool {
	_, ok := el.Attr(local)
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
