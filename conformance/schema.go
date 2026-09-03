package conformance

import (
	"path/filepath"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file activates the schema lane (issue #175) by giving the schema entry of
// defaultLanes a real executor. It touches nothing else in the runner (the #6
// seam, STYLE T2): the lane's selector stays selectsKind(kindSchema), so the
// executor is handed EVERY schemaTest case and either decides it or honestly
// DECLINES it (records a Fail gap) — a case it cannot decide for the right
// reason never flips to pass. It is package-internal conformance support: it
// exports nothing and no library code imports it.
//
// The gate this file documents is assembleCase, and it is SHARED: the instance
// lane (conformance/instance.go, #713) needs the very same assurance before it
// may assess anything against the schema an assembly built, so it asks the same
// question of the same closure and declines on the same answers.
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
// decidable subset of <complexType> (implicit content, and both alternants of
// each of <complexContent> and <simpleContent>; its particles including <group
// ref>, local element/attribute declarations, attribute uses including
// <attributeGroup ref>, wildcards, <openContent> and the schema-level
// <defaultOpenContent> it falls back to, and <assert> assertions) into xsd
// components, maps the name= identity constraints of global and local <element>s
// (#178), seeds the ur-type xs:anyType, resolves cross-references, and rejects
// duplicate top-level names within a kind. The ref= identity-constraint form and
// the inline anonymous local complexType forms outside the implicit-content
// shape are SILENTLY SKIPPED or declined (§3.1.2 permits ignoring a
// not-yet-produced representation), NOT rejected.
//
// The EXTENSION forms are DECIDED as of #336: Parse builds <complexContent>
// <extension> and <simpleContent> <extension> types (#228; §3.4.2.3.3 clause 4.2,
// §3.4.2.2 cases 3-5), cos-ct-extends (§3.4.6.2) judges them (#264), and every
// §3.4.2 base fold its case-1 clauses read is now done — clause 1.2's
// {attribute uses} (§3.4.2.4 clause 3, #401), clause 1.3's {attribute wildcard}
// (§3.4.2.5 clause 2.2, #265) and clause 1.7's {assertions} (§3.4.2.1 clause 1,
// #346). ONE cos-ct-extends clause cannot reject inside the admitted shape:
// clause 1.5.
//
// Clause 1.5 (two-step derivability) is an ENGINE approximation: it is proven
// only for a pure-extension chain, and a chain that mixes extension and
// restriction steps is accepted unconditionally (GAP(xsd) in
// xsd/complexextension.go, follow-up #392). That marker, not this comment,
// carries its retirement. Case 2 (simple base) is complete.
//
// It is an UNDER-rejection — the lane can report "valid" for a schema a complete
// processor rejects, never "invalid" for a valid one — so it is a recorded gap
// of the same shape as the ones the admitted <restriction> path already carries,
// not a fabricated verdict.
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
// assembleCase therefore admits a case only after confirming the whole shape
// of every document in its closure is confined to what the producer checks, and
// the lane DECLINES (Fail) anything else:
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
//  3. Top-level allowlist, unless the document is UNCONDITIONALLY REJECTED
//     first. A document holding a MISPLACED <xs:notation> — one whose parent is
//     neither <xs:schema> nor <xs:override>, the only two content models that
//     reach the xs:schemaTop group <notation> is declared in — is admitted
//     whatever else it holds (#945, holdsMisplacedNotation). Nothing in it can
//     be silently skipped into a vacuous accept, because the parse cannot return
//     nil: rejectS4SFaults walks the whole document before any producer runs and
//     charges the fault wherever it stands. The verdict it buys is "invalid",
//     which §5.1's first bullet (xmlschema11-1.md:4296) makes the right one for
//     such a document however its remaining shape reads, and no later slice can
//     flip it — that permanence is what separates this from the ReadDocument
//     limitation step 1 declines. Only a construct the producer rejects
//     GENUINELY and UNCONDITIONALLY may be admitted this way; a non-nil error is
//     NOT decidability on its own, since the producer also rejects for
//     not-yet-implemented reasons and the allowlist below is what guards that
//     direction.
//
//     Every top-level child element must otherwise be xsd:annotation,
//     xsd:include, xsd:import, xsd:override, xsd:simpleType, xsd:element,
//     xsd:attribute, xsd:complexType, xsd:attributeGroup (named definition),
//     xsd:group (named definition), xsd:defaultOpenContent (in every shape, as
//     of #352: the two its declaration forbids are rejected by the producer for
//     any document that declares one, so they are real verdicts rather than
//     content-dependent ones) or xsd:notation or xsd:redefine — anything else at
//     top level (a non-xsd element, or an out-of-set local name) closes the
//     false-accept gap above by DECLINING the whole case. Within the allowed
//     kinds:
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
//       that yields no D2, and assembleCase declines only if the parse then fails
//       with the src-resolve rejection that shortfall could have fabricated
//       (#276, narrowed by #404 — see fabricatedRejection).
//     - override: admitted (#183) when every child of it is a decidable source
//       declaration in its own right (overrideDecidable), because §F.2 clause 1
//       makes those children top-level declarations of the OVERRIDDEN document.
//       The document it points at, and the rest of §4.2.5's ·target set·, is
//       gated by the same closure gate that gates an <include>'s target, being in
//       the same reported document set; src-override's own clauses are then
//       enforced genuinely by the assembly.
//     - import: admitted at top level (#182) on the same reasoning. As for
//       include, a directive that yields no D2 (no schemaLocation, or one that
//       does not resolve) is REPORTED as unfollowed, and declines the case only
//       when the parse also fails with the src-resolve rejection the missing
//       components would produce. See the Composition section below,
//       fabricatedRejection and parser.AssemblyReport.Unfollowed.
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
//       fails src-resolve) genuinely. A <simpleContent>/<complexContent> carrying
//       NEITHER alternant is admitted too (#868): §3.4.2.2 and §3.4.2.3 each require
//       one, and the producer rejects a document carrying neither. A real structural
//       violation inside an admitted shape (src-ct, cos-all-limited, src-wildcard, …)
//       flows through as a genuine rejection.
//     - attributeGroup (top-level, §3.6.2): children only <attribute> (no inline
//       anonymous type), <attributeGroup>, and <anyAttribute> — the shapes the
//       producer folds in (§3.6.2.1/§3.6.2.2). A dangling ref (src-resolve clause
//       1.4) and a circular ref chain (spec-legal, §3.6.2.1) are both decided
//       genuinely.
//     - group (top-level, §3.7.2): a single all/choice/sequence body whose
//       particles are decidable; the body maps to {model group} genuinely (a
//       missing body is the s4s-grammar rejection xs:namedGroup's content model
//       requires, #884).
//     - Neither of those two requires the DEFINITION form (#1182). A top-level
//       <group>/<attributeGroup> with no name=, or one carrying the ref= that
//       xs:namedGroup/xs:namedAttributeGroup restricts to use="prohibited", is
//       admitted here and REJECTED by the producer under §2.4 clause 1 sd-valid
//       (§5.1's first bullet for schema construction) — an s4s attribute-use
//       fault, which carries no numbered rule ID. The nested reference forms are
//       the mirror image and are admitted on the same terms: a <group> in a
//       content model or an <attributeGroup> in an attribute-declaration tail
//       with no ref=, or carrying the prohibited name=, is the producer's
//       rejection to make, not this gate's to pre-empt.
//     - attribute: an inline <simpleType> child is admitted when its own shape is
//       produced (#733 maps §3.2.2.1 dcl.att.global's tier 1, which dcl.att.local
//       states in the same words), so a type= beside it is a genuine src-attribute
//       clause 4 (§3.2.3) verdict. A bare attribute is FINE too: it defaults to
//       xs:anySimpleType (§3.2.2.1), which builtin.Seed always seeds, so type= is
//       NOT required.
//     - simpleType (top-level or any anonymous inline one reached transitively
//       through a restriction, list, union, element or attribute chain): all
//       three §3.16.2.1 alternatives are admitted. <list> is produced in both its
//       forms (#447) and <union> in all three of its (#738), and every inline
//       <simpleType> item or member child is recursed into. <restriction> is
//       admitted outright: every constraining facet it can carry is produced,
//       the two folded ones included — one assertions facet per restriction
//       (#178, Datatypes §4.3.13) and one enumeration facet per restriction
//       (#740, §4.3.5.2 xr-enumeration) — and an inline <simpleType> base child
//       (§3.16.3 clause 2) is recursed into. A <simpleType> naming NONE of the
//       three is admitted too (#786): simpleTypeBody rejects that document
//       unconditionally, so the rejection is the producer's own and not a
//       limitation. No alternative's member-source arrangement is pre-checked:
//       those ARE the genuine src-simple-type clauses 2, 3 and 4 Produce
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
//     Steps 1-3 rule out the non-verdict failure modes ACROSS THE WHOLE CLOSURE:
//     the root was independently confirmed resolvable, readable and
//     <schema>-rooted before the assembly ran, and every document the assembly
//     did take in is reported and shape-gated. Two of those modes reach this
//     step — an I/O or encoding failure on a COMPOSED document, reported as
//     parser.UnfollowedUnreadable, and a rejection fabricated by components a
//     directive did not bring in — and fabricatedRejection eliminates both, as
//     the ONE site that discriminates on the error, by rule (xsderr.RuleOf)
//     rather than by type. Every OTHER failure this step reads as a verdict, an
//     unruled plain error included, with the single exception the GAP below
//     states: an <include>/<override> carrying no schemaLocation at all is a
//     §2.4 clause 1 grammar fault no Schema Representation Constraint covers
//     (parse.go's compose), which STYLE E2 requires be charged WITHOUT a rule ID
//     — an unruled error is this processor's spelling of "not valid against the
//     schema for schema documents", not a signal that no verdict was reached
//     (#404).
//
//     GAP(conformance): a THIRD non-verdict mode survives to this step, and
//     reading it as a verdict scores in the ratchet-corrupting direction.
//     parser/parse.go's fetch has TWO error-returning exits and only one of them
//     is a read: a resolver fault that is not loader.ErrNotFound — a permission
//     or transport error, ENOTDIR among them — records
//     parser.UnfollowedLocationUnresolved, NOT parser.UnfollowedUnreadable, and
//     returns an unruled plain Go error. fabricatedRejection therefore finds no
//     unreadable directive and no src-resolve rule, and this step decides
//     "invalid" off a pure I/O failure on a NON-ROOT document — the fault
//     assembleCase's own step 1 declines when it hits the ROOT. #1201 owns the
//     fix, which is parser's rather than this file's; both shapes it may take
//     are stated there. No suite case takes that exit today — every
//     schemaLocation in the corpus was scanned through the loader.Dir open path
//     and none yields a non-ErrNotFound error (#404) — which is exactly why
//     neither the gate nor the ratchet can see it. Do not close it by declining
//     on "unruled AND
//     UnfollowedLocationUnresolved": that predicate is annotB020, this lane's
//     own witness, and it gives back a banked win (#404).
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
// the shape allowlist excludes every form (an <override> child carrying no
// name=, an <override> child of a kind <override>'s own content model does not
// admit — both passed over in silence — an <assertions> facet child folding into
// nothing, the produced-but-unjudged extension forms) where the producer's
// rejection would be a limitation rather than a spec violation, or its silence a
// missing rejection. A particle under a <simpleContent> alternant is excluded
// CONSERVATIVELY instead, on neither of those grounds: the producer REJECTS it,
// the s4s order check charging that xs:simpleExtensionType is (annotation?,
// ((attribute | attributeGroup)*, anyAttribute?), assert*)
// (xmlschema11-1.md:1697), which admits no particle in any position — the
// reading schema_closure_test.go's undecidable const records. Declining over an
// explicit rejection forgoes a verdict and cannot fabricate one, so that
// exclusion is a harvest candidate on #1182's reasoning rather than a soundness
// requirement. A suite-invalid case whose only defect is a rule finalize does NOT
// yet check (cos-content-act-restrict — derivation-ok-restriction clause 2.4.2,
// #263 — cos-ns-subset, #265, or the Open Content half of
// derivation-ok-restriction clause 2.4, which xsd.Schema's content-model
// automaton fails open on: xsd/contentrestricts.go's GAP(xsd), live for a
// produced {open content} since #230) is produced cleanly, so the lane observes
// "valid", disagrees with the suite, and records a still-failing gap — never a
// wrong "invalid" pass. The remaining risk the allowlist closes is the VACUOUS
// pass — a document of entirely skipped top-level content that would otherwise
// always "pass" through a producer doing nothing — which is why step 3 confines
// the whole top level of EVERY document in the closure to the processed kinds and
// the decidable complexType subset.
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
// fails src-resolve: the missing document's components are then absent from the
// assembly and the reference that wanted them failed src-resolve clauses 1-3 at
// finalize, a FABRICATED "invalid" verdict, the one direction that can corrupt the
// ratchet. That conjunction is the whole hazard (#276): where the parse succeeds
// despite the missing document, §4.2.3 clause 2.4's "not an error ... the inclusion
// must not be performed" is simply in force and the case is decided normally — the
// suite's own cl. 2.4 tests (MS-Schema schD8 and friends) depend on that. Where it
// fails on a rule the shortfall cannot reach, the rejection is the document's own
// and the case is likewise decided (#404, fabricatedRejection). The directives are
// one hazard, not two: src-include clause 2.4 and src-import's "not an error" text
// are parallel, and src-resolve clause 4 (cl.qnr.nsdeclared) licenses a
// same-namespace reference (4.2.1) and a reference into a namespace with a PRESENT
// <import> element (4.2.2) alike, whether or not that import's schemaLocation
// resolved.
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
// The not-yet-produced complexType forms named above widen in with later slices
// (exactly as the datatypes lane grew across #15/#57/#80, and as a global
// attribute's inline anonymous type did with #733); they stay DECLINED (Fail)
// recorded gaps here, never guessed. UPA and EDC landed with #180,
// derivation-ok-restriction with #262 and cos-ct-extends with #264, so the
// admitted complexType cases those rules reject — restriction and extension
// alike, the latter admitted by #336 — are now decided; the cases still turning
// on cos-content-act-restrict (#263) stay failing gaps rather than wins until
// that lands.
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

// execSchemaCase decides one schemaTest case, or honestly declines it (Fail).
// The whole decidability gate is assembleCase, over the case's own declared
// document set; what is left here is the one comparison of the assembly's
// outcome with the suite's declared validity.
func execSchemaCase(backend value.Backend, c caseSpec) Status {
	_, decidable, perr := assembleCase(backend, c.doc, c.extraDocs)
	if !decidable {
		return Fail()
	}
	return decideAgreement(perr == nil, c.expect.wantsValid())
}

// assembleCase assembles the schema rooted at doc, together with any FURTHER
// documents extraDocs declares, and reports whether the outcome may be read as a
// verdict at all. It is the schema lane's decidability gate and the instance
// lane's precondition alike (conformance/instance.go).
//
// It reads the root document to check the two preconditions parser.ParseReport
// answers with a plain Go error rather than a validity verdict, runs
// parser.ParseReport, and gates the WHOLE <xs:include>/<xs:override>/<xs:import>
// closure it reports on the decidable top-level shape (closureDecidable, which
// runs schemaShapeDecidable on every document the assembly consumed). The second
// result, decidable, is false — and the caller DECLINES — under any of five
// conditions: a root it cannot resolve; a root it cannot read (any ReadDocument
// error, including a parser encoding limitation such as unsupported UTF-16); a
// root element that is not <schema>; a closure holding one document outside the
// producer's decidable subset, or missing one the case further declared; and a
// case whose parse failed with a rejection its own unfollowed directives could
// have fabricated (fabricatedRejection, #276/#404). The other two results say
// nothing then.
//
// Where decidable is true, the third result is the assembly's OWN error: nil is
// genuine evidence of validity — no document of the assembly has any of the
// violations the allowlist confines it to, so a real one would surface — and
// non-nil is a REAL implemented rejection. The first result is what the assembly
// built. The schema lane reads the error alone; the instance lane needs the
// schema, and takes it only where the error is nil, a schema the assembly
// rejected being not the schema the suite declared.
//
// The resolver is a loader.Dir rooted at doc's own directory and the root is
// named by its BASE name, because parser.ParseReport reads the root under
// exactly the location string it is handed (readRootDocument in parser/parse.go):
// passing the full path would give the root document a base URI of
// "…/sunData/SType/x" instead of "x", and every <include> in it would then resolve
// one directory tree away from where the resolver serves. The harness's own
// precondition read below therefore uses the SAME resolver and the SAME location
// string, so it reads byte-identically the document the assembly roots at.
func assembleCase(backend value.Backend, doc string, extraDocs []string) (*xsd.Schema, bool, error) {
	resolver := loader.Dir(filepath.Dir(doc))
	location := filepath.Base(doc)
	rc, _, err := resolver.Resolve("", location)
	if err != nil {
		// Unreadable document: an honest recorded gap, not a validity verdict.
		return nil, false, nil
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	root, err := parser.ReadDocument(location, rc)
	if err != nil {
		// A ReadDocument error is DECLINED, never treated as an observed-invalid
		// verdict. The error does not distinguish a genuine XML well-formedness
		// fault from a parser encoding LIMITATION: well-formed UTF-16 input (BOM
		// FF FE) is currently rejected as "[xml-wf] invalid UTF-8" because UTF-16
		// decoding is not yet implemented, so an "invalid" verdict here would be
		// fabricated for a well-formed document — a wrong-reason pass that would
		// silently flip pass→fail once UTF-16 decoding lands (a separate change).
		// Declining on ANY ReadDocument error keeps the lane's verdicts honest.
		return nil, false, nil
	}
	// §3.17.2 does not require <schema> to be the document root, so a non-schema
	// root is a Parse precondition fault (a plain Go error, not a
	// sch-props-correct rejection), not decidable for this lane — decline.
	if !root.IsSchema() {
		return nil, false, nil
	}
	// ParseReport, not Parse: the verdict needs the DOCUMENT SET the assembly
	// consumed, not only its components (#272).
	schema, report, perr := parser.ParseReport(location, parser.WithResolver(resolver), parser.WithBackend(backend))
	// Only decide when EVERY document of the <include>/<override>/<import> closure
	// is confined to what the producer processes; otherwise a silently-skipped
	// invalid representation, in the root or in any composed document, could
	// false-accept.
	if !closureDecidable(report) {
		return nil, false, nil
	}
	if !extraDocsInClosure(report, resolver, doc, extraDocs) {
		return nil, false, nil
	}
	// A directive that named no document is only half the fabricated-rejection
	// hazard (#276): the missing components matter solely when something referred
	// to them, and that shows up here, as a failed parse whose src-resolve clause
	// 1-3 error the spec does not attach to the missing document (§5.3). A parse
	// that SUCCEEDED past an unfollowed directive fabricated nothing — §4.2.3
	// clause 2.4's "not an error ... the inclusion must not be performed" is
	// exactly that outcome — so the case is still decided, and so is a parse that
	// failed for a reason no unfollowed directive could have produced (#404).
	if fabricatedRejection(report, perr) {
		return nil, false, nil
	}
	return schema, true, perr
}

// ruleSrcResolve is the rule the producer charges for a reference the assembled
// schema holds no component for (QName resolution, §3.17.6.2 clauses 1-3). It is
// matched in its BARE catalog form, the clause living in the message text rather
// than in a dotted rule ID — the instance lane's decidableRules idiom.
const ruleSrcResolve xsderr.Rule = "src-resolve"

// fabricatedRejection reports whether perr is a rejection this assembly's own
// unfollowed directives could have FABRICATED: an "invalid" resting on a
// document this processor did not read rather than on the document under test.
// That is the one direction able to corrupt the ratchet, and the whole reason
// assembleCase declines rather than deciding (#276).
//
// The correlation it draws is an ENGINEERING HEURISTIC over the rules THIS
// producer charges, not a spec rule, and there is no spec rule to have: §3.17.6.2
// src-resolve tests a QName against the composed schema's aggregate properties
// and back-references no directive, §4.2.3 clause 2.4 instructs retention of
// unresolved QName references rather than attribution of blame, and §5.3
// describes the ·absent· consequence generically. Nothing in either document
// correlates a specific unfollowed directive with a specific downstream failure,
// so the reasoning below is about this producer's charge sites and is revisited
// whenever they change (#404).
//
// Two directive reasons carry a hazard, and they are different hazards:
//
//   - A location that did not resolve, or a directive naming none at all
//     (parser.UnfollowedLocationUnresolved, parser.UnfollowedNoLocation), leaves
//     the assembly SHORT of components. Having FEWER components surfaces at
//     exactly one rule: the producer charges src-resolve for every reference
//     whose target the schema does not hold, and no other rule it charges can be
//     reached by a shortfall — sch-props-correct clause 2 needs two components to
//     collide, cos-nonambig needs particles that already resolved,
//     src-include/src-import/src-redefine are the directive's OWN faults and so
//     are causes of the shortfall rather than consequences of it, and the §2.4
//     clause 1 grammar faults are properties of one document's element tree.
//   - A location that resolved to a document that could not be READ
//     (parser.UnfollowedUnreadable) is charged src-include/src-import/
//     src-redefine "not well-formed" — and a read failure does not distinguish a
//     genuine well-formedness fault from a reader LIMITATION, an encoding this
//     parser does not decode. That is step 1's own reasoning for declining an
//     unreadable ROOT, and it holds whatever rule carries the failure, so any
//     failure alongside such a directive declines.
//
// Every other reason fabricates nothing, parser.UnfollowedNoSchemaLocation most
// clearly: §4.2.1 makes that attribute mandatory ("not hints: conforming
// processors must attempt to de-reference"), so the assembly rejects the
// DIRECTIVE ITSELF against the §2.4 clause 1 grammar (STYLE E2's unruled plain
// error) — a genuine rejection of the document under test, naming no missing
// component at all. Reading it as a verdict is what the lane already does for
// every other grammar fault.
//
// The shortfall arm stays SCAN-scoped: ANY unfollowed directive plus a
// src-resolve failure declines, without asking whether that directive is the one
// that would have supplied the component the reference wanted. Narrowing it
// needs the reference's namespace on the error and the directive's namespace on
// the report, and neither xsderr.Error nor parser.UnfollowedDirective carries
// one — a data-shape change, where this is not (#404).
//
// GAP(conformance): one non-verdict failure is invisible to BOTH arms, and the
// caller then scores it "invalid". parser/parse.go's fetch records
// parser.UnfollowedLocationUnresolved and returns an UNRULED plain error for a
// resolver fault that is not loader.ErrNotFound — a permission or transport
// error. The reason is not parser.UnfollowedUnreadable, so the read arm misses
// it; the error carries no rule, so xsderr.RuleOf yields no src-resolve and the
// shortfall arm misses it too. The two arms above assume every non-verdict a
// composed document can produce is a READ failure; a RESOLVE failure that is not
// mere absence is the one that is not. It cannot be told from the shape this
// predicate reads: parser.UnfollowedLocationUnresolved with an unruled error is
// also annotB020, whose <include> names a document that plain does not exist and
// whose unruled error is an unrelated §2.4 clause 1 grammar fault that must
// DECIDE. #1201 owns the fix and states why it belongs in parser rather than
// here. Zero suite cases reach that exit today, so this predicate's measured
// behavior over the whole corpus is unaffected (#404).
func fabricatedRejection(report *parser.AssemblyReport, perr error) bool {
	if perr == nil || len(report.Unfollowed()) == 0 {
		return false
	}
	for _, u := range report.Unfollowed() {
		if u.Reason == parser.UnfollowedUnreadable {
			return true
		}
	}
	rule, ok := xsderr.RuleOf(perr)
	return ok && rule == ruleSrcResolve
}

// extraDocsInClosure reports whether every FURTHER document the case declares
// beyond doc was consumed by the assembly rooted at doc — the only condition
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
func extraDocsInClosure(report *parser.AssemblyReport, resolver loader.Resolver, doc string, extraDocs []string) bool {
	root := filepath.Dir(doc)
	for _, extra := range extraDocs {
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

// decideAgreement Passes iff the observed validity agrees with the suite's
// declared XSD 1.1 expectation. It is the ONE closing comparison every deciding
// executor of the schema and instance lanes reaches, and every decline returns
// before it — which is exactly what the decline census probes (declines.go).
func decideAgreement(observed, expected bool) Status {
	if observed == expected {
		return Pass()
	}
	return Fail()
}

// schemaShapeDecidable reports whether this document contributes no false-accept
// hazard to the assembly that read it: either it is UNCONDITIONALLY REJECTED
// (holdsMisplacedNotation), or every top-level child lies within the producer's
// decidable subset (the step-3 allowlist documented above). A single
// out-of-subset child declines the whole case, since Produce would silently skip
// it (or reject it for a not-yet-supported reason) rather than decide it
// genuinely.
//
// The misplaced-<notation> short-circuit comes FIRST because it makes every
// other question moot: a document carrying one is rejected before any producer
// dispatches, so no shape below can be silently skipped into a vacuous accept
// and no shape below can be rejected for a limitation instead. The remaining
// allowlist runs unchanged on every other document, this one included once the
// notation is moved into a slot the grammar admits.
func schemaShapeDecidable(doc *parser.Document) bool {
	if holdsMisplacedNotation(doc.Root()) {
		return true
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
			// parser.AssemblyReport.Unfollowed and assembleCase.
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
			// silently skipped — and admitted in EVERY shape as of #352, malformed
			// ones included. The two the schema for schema documents forbids (no
			// <any> child; a mode outside interleave|suffix) are now rejected by the
			// producer's pre-produce pass for any document that declares one, so both
			// verdicts are real and unconditional. The narrower arm #230 wrote
			// declined them only because that rejection used to be lazy — reachable
			// solely through a complex type that got as far as clause 5.2 — which
			// would have made the verdict depend on unrelated content.
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

// holdsMisplacedNotation reports whether the subtree under parent holds an
// <xs:notation> the schema for schema documents admits nowhere: <notation> is
// declared in the xs:schemaTop group alone (xmlschema11-1.md:4462), and only
// <schema> (:4562) and <override> (:5577) reference that group. <redefine> is
// not a third — its own model reaches xs:redefinable (:5558), which omits
// <notation> (:4465-4477) — so a <notation> child of one is misplaced too, by a
// different production and the same umbrella fault. The fault carries NO rule
// ID: §3.14.3 and §3.14.4 both answer "None as such" (:3409, :3413), so it
// stands on §5.1's first, unnumbered bullet (:4296) directly, and charging a
// src-* verdict here would fabricate one (STYLE E2).
//
// A document holding one is DECIDABLE whatever else it holds (#945), because the
// producer rejects it GENUINELY and UNCONDITIONALLY: parser's rejectS4SFaults
// walks the whole document before any producer dispatches, so no position can
// reach around the charge — and a <notation> under <redefine> is charged earlier
// still, at discovery, by <redefine>'s own content-model guard. Both are grammar
// verdicts the spec licenses, not limitations awaiting a slice, so the "invalid"
// they buy cannot flip once some later slice lands. That permanence is the whole
// admission argument; a merely non-nil parse error is NOT one, since the
// producer also rejects for not-yet-implemented reasons.
//
// The walk MIRRORS rejectS4SFaults' — XSD-namespace elements only, never
// descending into <appinfo> or <documentation>, whose <xs:any
// processContents="lax"> content (:5727, :5740) makes an element merely NAMED
// {XMLSchemaNS}notation ordinary content that no guard governs. The two must
// agree exactly: a shape this predicate claimed and that walk did not would be
// admitted on a rejection that never comes (MS-Notations2006-07-15/notatF009,
// suite-VALID, is precisely such a document).
//
// parent is the element whose children are examined, so the top-level call
// passes the <schema> root and every notation child of it is legal.
func holdsMisplacedNotation(parent *parser.Element) bool {
	name := parent.Name()
	legal := name.Space() == xsd.XMLSchemaNS && (name.Local() == "schema" || name.Local() == "override")
	for _, child := range parent.Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		local := el.Name().Local()
		if local == "appinfo" || local == "documentation" {
			continue
		}
		if local == "notation" && !legal {
			return true
		}
		if holdsMisplacedNotation(el) {
			return true
		}
	}
	return false
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
// reach is the original itself: Phase D's walk enters an owned {base type
// definition} to reach the declarations inside it but charges the base no
// verdict of its own (GAP(xsd), xsd/complexderivation.go, #584). That is an
// under-rejection on a component the redefinition's own verdict already covers
// structurally, the same safe direction as the residue below.
//
// A child with no name= declines rather than being waved through: the pairing is
// keyed on (element type, name), and src-expredef requires a top-level definition
// item of that name and kind in the redefined document, so a nameless child is
// reported as a grammar fault rather than decided.
//
// No fail-open src-redefine clause remains inside an admitted case: clause 7.2.2
// is charged at finalize as of #503 and clause 6.2.2 as of #504, each over the
// pairing the producer records between the redefinition and the original in S2.
// The document the <redefine> POINTS AT is gated by closureDecidable, exactly as
// an <include>'s target is.
//
// ONE fail-CLOSED residue is kept deliberately: a CHAINED <redefine> of a
// <group>/<attributeGroup> — the redefined document redefining a document of its
// own for the same name — is refused under src-expredef's closing requirement, so
// such a case observes "invalid" whatever the suite says. The producer carries the
// marker (parser/redefine.go's chainedOriginal) and #744 owns its retirement,
// which needs an xsd entry point for a redefinition that contributes no component:
// charging both clauses now is NOT enough, because the two suite cases in that
// shape, MS-Schema2006-07-15/schL10 and /schM5, are each invalid at the MIDDLE
// level of their chain, which is the level clause 4.1.2 withholds. So the residue
// is still the whole reason those two agree with the suite. A chained <simpleType>
// or <complexType> composes and is decided genuinely
// (MS-Additional2006-07-15/addB007).
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
// Both inline type children ARE produced (§3.3.2.1 dcl.elt.common clause 1) and
// each is admitted when the anonymous type it maps to is itself decidable: a
// <complexType> through complexTypeDecidable, the very predicate a NAMED one
// takes (#340, widened to the explicit-content forms by #1126 — that function's
// doc comment carries why anonymity narrows nothing), and a <simpleType>
// through simpleTypeDecidable, exactly as localElementDecidable gates the local
// form (#229/#442). This function and localElementDecidable are therefore no longer
// asymmetric about tier 1 — the tier is a COMMON mapping rule and the producer
// now maps it on both paths, so a difference here would be the fabrication.
//
// This gate does NOT need a condition for a substitutionGroup= whose head is
// typed by an inline <simpleType>: the producer maps that clause-3 shape to an
// xsd.SubstitutionGroupHeadTypeRef naming the head, like the inline
// <complexType> head beside it, so no member of such a head reaches a limitation
// error (#442; before it, the head's own decline here is what kept the shape out
// of reach).
//
// Its <alternative> children are gated by alternativeTypesDecidable, on the same
// terms and through the same two functions.
//
// Its <unique>/<key>/<keyref> children impose no condition of their own: #178
// produced the name= form and #240 the ref= form, so BOTH are mapped and both
// src-identity-constraint (§3.11.3) and c-props-correct (§3.11.6.1) rejections on
// them are genuine.
func elementDecidable(el *parser.Element) bool {
	if st := childXSD(el, "simpleType"); st != nil && !simpleTypeDecidable(st) {
		return false
	}
	if inline := childXSD(el, "complexType"); inline != nil && !complexTypeDecidable(inline) {
		return false
	}
	return alternativeTypesDecidable(el)
}

// alternativeTypesDecidable reports whether every type an <element>'s
// <alternative> children own is decidable. §3.12.2 declare-ta's inline arm maps
// a <complexType> or <simpleType> child of an <alternative> to a type definition
// the element declaration OWNS (#851); an <alternative> on the type= arm imposes
// no condition, its reference being resolved at finalize like any other with a
// genuine src-resolve clause 1.1 failure.
//
// The <simpleType> child goes through simpleTypeDecidable, the same gate an
// element's own inline one takes.
//
// The <complexType> child goes through complexTypeDecidable, which is where the
// anonymity residual every inline complex type carries is stated once. This site
// took that predicate from the start (#851) and the two element paths joined it
// at #1126; the three no longer differ, so no argument for an asymmetry is owed
// here any more.
//
// It serves both element paths (STYLE T4): §3.3.2.1's {type table} row is a
// COMMON mapping rule, so a top-level and a local <element> carry the same
// children through the same producer.
func alternativeTypesDecidable(el *parser.Element) bool {
	for _, alt := range childrenXSD(el, "alternative") {
		if ct := childXSD(alt, "complexType"); ct != nil && !complexTypeDecidable(ct) {
			return false
		}
		if st := childXSD(alt, "simpleType"); st != nil && !simpleTypeDecidable(st) {
			return false
		}
	}
	return true
}

// complexTypeDecidable reports whether a <complexType> — top-level, ANONYMOUS,
// or a nested derivation reached through <complexContent>/<simpleContent> —
// lies within the producer's decidable subset — the shapes it fully builds AND
// finalize fully judges, so any Produce error on it is a REAL structural violation
// (src-ct/cos-all-limited/src-wildcard/src-attribute/p-props-correct/src-resolve)
// and its silence is a real absence of one. Finalize judges every shape here
// alike with ONE exception, which is anonymity's and is stated in the anonymity
// paragraphs below. It declines:
//
//   - every shape the producer declines with a plain "not yet produced"
//     limitation error — today NONE at this level. The bare <group>/
//     <attributeGroup> lacking a ref was the last one named here and is no
//     longer declined: the producer REJECTS it as the s4s attribute-use fault it
//     is (#1182), so the verdict is genuine and the shape is admitted.
//     <simpleContent> <restriction> was the other one and is no longer declined:
//     #909 built §3.4.2.2 cases 1-2, so the shape goes through
//     simpleContentRestrictionDecidable, which admits what that mapping reads
//     plus every child the s4s order check rejects outright (#1181), leaving the
//     plural <assertions> as its one decline. An inline anonymous <simpleType>
//     on a local element or attribute IS produced (#229) and an inline anonymous
//     <complexType> on a local OR global element is produced too (#340); neither
//     declines here — see localElementDecidable/elementDecidable, which route
//     the anonymous complex-type shape into THIS function, and the anonymity
//     paragraphs below. An <openContent> is produced as of #230 (§3.4.2.3.3
//     clauses 5-6) and is admitted by contentDecidable wherever the schema for
//     schema documents allows one; a MISPLACED one — beside
//     <simpleContent>/<complexContent>, or directly under <complexContent> — is
//     rejected by the producer as the grammar fault it is, so it needs no
//     decline of its own.
//
// A <simpleContent>/<complexContent> carrying NEITHER alternant is ADMITTED
// (#868). §3.4.2.2 and §3.4.2.3 each require one of them, and the producer
// rejects a document that carries neither; that the rejection is a grammar fault
// about the source item rather than a rule verdict does not make it a
// limitation, so it needs no decline any more than a misplaced <openContent>
// does.
//
// Both EXTENSION forms are ADMITTED as of #336. The producer builds them (#228),
// cos-ct-extends (§3.4.6.2) judges them (#264), and its case-1 clauses read only
// folds that are now done — 1.2 over §3.4.2.4 clause 3's {attribute uses}
// (#401), 1.3 over §3.4.2.5 clause 2.2's {attribute wildcard} (#265) and 1.7
// over §3.4.2.1 clause 1's {assertions} (#346). One of its clauses still cannot
// reject: case 1's 1.5 (two-step derivability), proven only for a pure-extension
// chain and otherwise accepted unconditionally, an engine approximation
// (GAP(xsd), xsd/complexextension.go, #392). Case 2 is complete. The file comment
// above carries the full inventory. That clause is an UNDER-rejection, the same
// safe direction as cos-nonambig and ct-props-correct clause 4 already are on the
// admitted <restriction> path, so it bounds what this lane can claim rather than
// letting it fabricate an "invalid".
//
// A <group ref>/<attributeGroup ref> IS produced (#177) and admitted: its target
// resolves genuinely at finalize (or fails src-resolve). Real structural
// violations the producer DOES reject (a nested <all>, a mixed
// mismatch, a both-namespace-forms wildcard, a bad occurrence) are NOT declined:
// admitting them is safe because the producer's rejection is the right reason.
//
// ANONYMITY NARROWS NOTHING HERE, and until #1126 a separate predicate said
// otherwise by admitting the implicit-content form alone. §3.3.2.1 dcl.elt.common
// clause 1 is a COMMON mapping rule, so an <element>'s inline <complexType> maps
// through §3.4.2 exactly as a top-level named one does; §3.4.2.1 makes {name},
// {target namespace} and {context} the only properties anonymity fixes; §3.4.2.4
// opens "This mapping rule is the same for all complex type definitions" and
// §3.4.2.5 says it of its own; and no clause of §3.4.2.2, §3.4.2.3, §3.4.6.1
// (ct-props-correct), §3.4.6.2 (cos-ct-extends) or §3.4.6.3
// (derivation-ok-restriction) reads {name} or {context} at all. A gate narrower
// for the anonymous case would therefore state a distinction the spec does not
// make. All three sites that own an anonymous complex type reach it through this
// one function — elementDecidable and localElementDecidable for §3.3.2.1's tier
// 1 at every nesting depth (contentDecidable → modelGroupDecidable →
// localElementDecidable comes back here), alternativeTypesDecidable for
// §3.12.2's inline arm, which took this predicate from the start (#851).
//
// The residual anonymity used to leave — every finalize pass quantifying over
// {type definitions} reaching NO verdict for a type in no such set (§3.17.2
// scopes the property to the <complexType> children OF <schema>;
// xsd/typedefinition.go records why registering one would be worse) — is GONE
// for both slots this predicate reaches. Since #438 checkContentModelsUnambiguous
// (cos-nonambig), checkElementDeclarationsConsistent (cos-element-consistent)
// and checkComplexDerivations with the ct-props-correct clause 4 and
// cos-ct-extends clause 1.2 checks it drives all walk the owning slots
// (xsd/complextypewalk.go), so an anonymous type here is charged what a named
// one is. What is left uncharged is the redefine ORIGINAL alone (#584, and the
// GAP marker on parser's produceComplexType), which no <element> and no
// <alternative> can seat. That remainder is an UNDER-rejection: this lane can
// report "valid" for a schema a complete processor rejects, never "invalid" for
// a valid one. Nothing an INSTANCE case turns on is approximate on
// either newly admitted alternant — {content type} is computed at PRODUCE time
// (produceSimpleContent for §3.4.2.2, produceComplexContent for §3.4.2.3.3,
// the extension merge of clause 4.2 through xsd.ExtensionContentType), and
// §3.4.2.4 clause 3 and §3.4.2.5 clause 2 reach an owned anonymous type through
// the slot that owns it (xsd/ownedtypefold.go, #414) — so cvc-complex-type
// clause 2 is charged against the spec's own {attribute uses}/{attribute
// wildcard} and clause 3 against the spec's own {simple type definition}.
//
// What admits these shapes is that under-rejection, and NOT §3.4.6.3's
// provisional-accept carve-out, which is scoped to a T whose {content
// type}.{particle}.{term}.{compositor} is all and is keyed to the content
// model's shape rather than to whether T is named.
func complexTypeDecidable(el *parser.Element) bool {
	if sc := childXSD(el, "simpleContent"); sc != nil {
		if ext := childXSD(sc, "extension"); ext != nil {
			return simpleContentExtensionDecidable(ext)
		}
		if r := childXSD(sc, "restriction"); r != nil {
			return simpleContentRestrictionDecidable(r)
		}
		// Neither alternant is a grammar fault the producer rejects genuinely, so
		// it is admitted — see above.
		return true
	}
	if cc := childXSD(el, "complexContent"); cc != nil {
		derivation := childXSD(cc, "restriction")
		if derivation == nil {
			derivation = childXSD(cc, "extension")
		}
		if derivation == nil {
			return true // neither alternant — a genuine rejection, see above
		}
		return contentDecidable(derivation)
	}
	return contentDecidable(el)
}

// simpleContentExtensionDecidable reports whether a <simpleContent> <extension>
// is in the shape the producer decides genuinely. xs:simpleExtensionType
// (xmlschema11-1.md:1697) is (annotation?, ((attribute | attributeGroup)*,
// anyAttribute?), assert*) — the one derivation alternant with no structural
// position at all — so attrDeclsDecidable's vocabulary is the whole of this
// content model and the attribute children the producer folds in (§3.4.2.4) are
// the only ones with a shape to judge.
//
// Every OTHER XSD-namespace child is ADMITTED because the producer REJECTS it
// (#1181), never because it is harmless: checkS4SChildOrder walks this element
// against that model and charges §2.4 clause 1 sd-valid (:615, restated at
// §5.1's first bullet, :4296) for a name no position admits — a
// <sequence>/<choice>/<all>, a <group>, a stray <simpleContent>/<complexContent>
// alike — and an <openContent> is charged earlier still by misplacedOpenContent.
// §3.4.2.2 cases 3-5 do compute {content type} from the resolved base alone, but
// no document carrying such a child reaches that tableau, so NOTHING is dropped
// in silence at this position.
func simpleContentExtensionDecidable(ext *parser.Element) bool {
	for _, child := range ext.Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if decidable, shared := attrDeclsDecidable(el); shared && !decidable {
			return false
		}
	}
	return true
}

// simpleContentRestrictionDecidable reports whether a <simpleContent>
// <restriction> is in the shape the producer decides genuinely (#909). Its
// content model is xs:simpleRestrictionType — (annotation?, (simpleType?,
// facet*)?, ((attribute | attributeGroup)*, anyAttribute?), assert*) — and every
// item of it is mapped: the facet children and the optional inline <simpleType>
// into the anonymous simple type §3.4.2.2 cases 1-2 synthesize, the attribute
// children into {attribute uses}/{attribute wildcard}, the trailing <assert>
// children into the CTD's own {assertions}. So the shared tail goes through
// attrDeclsDecidable unchanged and a facet child is admitted outright, as it
// already is under <simpleType> <restriction> (simpleTypeDecidable) — the two
// sites map facets through the very same restrictionFacets.
//
// A name OUTSIDE that content model is ADMITTED, and the two mechanisms behind
// that must not be conflated (#1181). A model group, a <group>, a stray
// <simpleContent>/<complexContent> or any other name no position of
// xs:simpleRestrictionType admits is REJECTED: checkS4SChildOrder charges §2.4
// clause 1 sd-valid (:615, restated at §5.1's first bullet, :4296) over it, so
// the verdict is the producer's own. The PLURAL <assertions> is the one child
// that really is DROPPED IN SILENCE — it fills the facet position and folds into
// no facet — and it alone still declines here, through
// silentlyDroppedFacetElement. An inline <simpleType> outside the produced
// simple-type subset moves the decline inward through simpleTypeDecidable, as
// every other inline-type site does.
//
// Nothing here narrows for the tableau's case 5 (a base with element-only or
// empty content, or a simple type base, under <restriction>), although that case
// drops the facet children and any inline <simpleType> by the tableau's own
// instruction: every case-5 restriction shape is REJECTED at finalize —
// ct-props-correct clause 2 for a simple type base, derivation-ok-restriction
// clause 2 for the rest — so no verdict rides on what the mapping dropped. Since
// #438 that holds for an anonymous complex type owned by an <alternative> too:
// Phase D reaches it through the owning slot (xsd/complextypewalk.go) and makes
// the same rejection.
func simpleContentRestrictionDecidable(restriction *parser.Element) bool {
	for _, child := range restriction.Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if decidable, shared := attrDeclsDecidable(el); shared {
			if !decidable {
				return false
			}
			continue
		}
		local := el.Name().Local()
		if local == "simpleType" {
			if !simpleTypeDecidable(el) {
				return false
			}
			continue
		}
		if silentlyDroppedFacetElement(local) {
			return false
		}
	}
	return true
}

// silentlyDroppedFacetElement reports whether local names a child the parser's
// s4s order check lets THROUGH the facet position of xs:simpleRestrictionType
// (xmlschema11-1.md:1692) and the mapping then folds into NO facet — the false
// accept this allowlist exists to refuse.
//
// That is exactly the PLURAL <assertions>. §4.3.13 is where the two spellings
// part: the facet is "assertions" (a single facet holding the sequence) while
// the element contributing to it is <assertion>, which is what the facet choice
// spells — no element is named "assertions" anywhere in the s4s grammar. The
// parser's s4sFacetElement nevertheless admits the name into that position,
// because it asks builtin.FacetKindByName and that bridge answers for FACET
// names; its restrictionFacets then matches only the singular and its
// facetKindOf excludes xsd.FacetAssertions, so the child reaches no component
// and no error.
//
// It asks the same bridge, the ONE name→kind bridge the module owns (STYLE
// T4/D3), rather than spelling "assertions" here where the two would drift.
// <assert> is the CTD-level assertion of the same content model's trailing
// position, not a facet at all, and attrDeclsDecidable takes it before this is
// reached.
func silentlyDroppedFacetElement(local string) bool {
	kind, ok := builtin.FacetKindByName(builtin.FacetName(local))
	return ok && kind == xsd.FacetAssertions
}

// attrDeclsDecidable reports whether one child from the tail every complex-type
// derivation alternant ends with — annotation?, (attribute | attributeGroup)*,
// anyAttribute?, assert* (§3.4.2) — is within the producer's decidable subset.
// shared is false when el is none of those names, which leaves the verdict to
// the caller's own site vocabulary: a model group, a <group> or an
// <openContent> for contentDecidable, a facet or an inline <simpleType> for
// simpleContentRestrictionDecidable.
//
// The tail is one vocabulary and is judged in one place, so a site that gains a
// child kind cannot quietly disagree with the others about what an <attribute>
// or an <attributeGroup> may be.
func attrDeclsDecidable(el *parser.Element) (decidable, shared bool) {
	switch el.Name().Local() {
	case "annotation":
		return true, true // harmless
	case "attribute":
		return localAttributeDecidable(el), true
	case "attributeGroup":
		// Admitted in both shapes (#1182). The ref form is produced (#177); the
		// ref-less one is REJECTED by collectReferencedGroup, either for carrying
		// the name= xs:attributeGroupRef prohibits or for missing the ref= it
		// requires, so the verdict is the producer's own and nothing is fabricated.
		return true, true
	case "anyAttribute":
		return true, true // an attribute wildcard is produced
	case "assert":
		// Produced (#178) into {assertions}. An Assertion carries only an opaque
		// XPath Expression record with no rejectable state, so admitting it can
		// never turn into a fabricated "invalid" verdict.
		return true, true
	}
	return false, false
}

// contentDecidable reports whether the content-model child and attribute children
// of a <complexType> (implicit content) or of a <complexContent>
// <restriction>/<extension> are all within the producer's decidable subset. A
// <group ref> content child and an <attributeGroup ref> are admitted (produced,
// #177); an <openContent> is admitted (produced, #230); a ref-less <group>/
// <attributeGroup> is admitted too, the producer rejecting it as the s4s
// attribute-use fault it is (#1182). A local <attribute>'s inline anonymous
// <simpleType> is admitted when the inline type itself is decidable — see
// localAttributeDecidable.
//
// The attribute/assert/annotation tail is judged by attrDeclsDecidable, shared
// with simpleContentRestrictionDecidable; what stays here is the vocabulary only
// a complex-content alternant carries.
//
// ITS TWO CALL POSITIONS SHARE ONE GRAMMAR, and that is what lets an unrecognized
// name be admitted rather than declined (#1181). xs:complexTypeModel's
// implicit-content disjunct (:1649), xs:complexRestrictionType (:1718) and
// xs:extensionType (:1723) each end with "openContent?, (group | all | choice |
// sequence)?, ((attribute | attributeGroup)*, anyAttribute?), assert*" behind an
// "annotation?" — exactly the names the arms below spell — so a name reaching
// the default arm fills no position of whichever model the producer ordered this
// element against, and checkS4SChildOrder charges §2.4 clause 1 sd-valid (:615,
// restated at §5.1's first bullet, :4296) over it. A stray <simpleContent>/
// <complexContent> is that fault at BOTH positions and no third one:
// complexTypeDecidable dispatches on either wrapper before it ever calls here,
// so neither name can reach the implicit-content position at all.
func contentDecidable(parent *parser.Element) bool {
	for _, child := range parent.Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if decidable, shared := attrDeclsDecidable(el); shared {
			if !decidable {
				return false
			}
			continue
		}
		switch el.Name().Local() {
		case "sequence", "choice", "all":
			if !modelGroupDecidable(el) {
				return false
			}
		case "group":
			// Produced in the ref form (#177) and REJECTED in the ref-less one
			// (#1182): produceGroupRefParticle charges the name= xs:groupRef
			// prohibits, or the absent ref= it requires. Both verdicts are genuine,
			// so both shapes are admitted.
		case "openContent":
			// Produced (#230) into {open content} (§3.4.2.3.3 clauses 5-6). Its own
			// src-ct clause 3/4 violations and its <any>'s src-wildcard/w-props-correct
			// ones are genuine rejections, so nothing here is limitation-shaped.
		default:
			// Any other name — a stray <simpleContent>/<complexContent> included —
			// fills no position of the model this element is ordered against, so
			// checkS4SChildOrder rejects the document. The verdict is the producer's
			// own, so the shape is admitted (#1181); see above for why the two call
			// positions agree about that.
		}
	}
	return true
}

// modelGroupDecidable reports whether every particle child of a model group
// (<sequence>/<choice>/<all>) is within the producer's decidable subset: nested
// model groups recurse, <element> must be locally decidable
// (localElementDecidable) — its identity constraints impose no condition of
// their own, both forms being produced for local declarations too (#178, #240) —
// <any> is fine, and a <group> is admitted in both shapes — produced in the ref
// form (#177), rejected by the producer in the ref-less one (#1182). Any other
// child declines.
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
			// Produced in the ref form (#177) and REJECTED in the ref-less one
			// (#1182), by the same produceGroupRefParticle arms contentDecidable's
			// <group> child reaches.
		default:
			// any other child: not produced — decline.
			return false
		}
	}
	return true
}

// groupDecidable reports whether a top-level <group> (§3.7.2) is within the
// producer's decidable subset: its single all/choice/sequence body's particles
// must all be decidable. A missing body still produces genuinely (the producer
// rejects it against xs:namedGroup's content model, #884), so it is admitted.
//
// The DEFINITION FORM is not required either (#1182). A top-level <group> with
// no name=, or one carrying the ref= xs:namedGroup restricts to
// use="prohibited", is REJECTED by the producer — parser.topLevelName and
// rejectProhibitedAttrs respectively — under §2.4 clause 1 sd-valid, with no
// numbered rule ID because an s4s attribute-use fault has none. Admitting those
// two shapes therefore fabricates no verdict; it only lets the producer's own
// rejection be observed.
func groupDecidable(el *parser.Element) bool {
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

// attributeGroupDecidable reports whether a top-level <attributeGroup> (§3.6.2)
// is within the producer's decidable subset: its children must be only
// <attribute> (locally decidable — the group's <attribute> children map to LOCAL
// declarations, §3.6.2.1), <attributeGroup>, or <anyAttribute> — the shapes the
// producer folds in (§3.6.2.1/§3.6.2.2). A dangling or circular ref is decided
// genuinely at producer/finalize time, so it is admitted.
//
// The DEFINITION FORM is not required, on exactly groupDecidable's terms
// (#1182): xs:namedAttributeGroup makes name= use="required" and ref=
// use="prohibited", and the producer charges both faults itself, so a top-level
// <attributeGroup> missing the one or carrying the other is admitted here and
// rejected there.
func attributeGroupDecidable(el *parser.Element) bool {
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
			// Produced in the ref form (#177) and REJECTED in the ref-less one
			// (#1182), by the same collectReferencedGroup arms attrDeclsDecidable's
			// <attributeGroup> child reaches.
		default:
			return false
		}
	}
	return true
}

// attributeDecidable reports whether a TOP-LEVEL <attribute> is decidable.
// type= is NOT required — a bare attribute defaults to xs:anySimpleType
// (§3.2.2.1), which builtin.Seed always seeds, so it resolves and is decided
// genuinely.
//
// An inline anonymous <simpleType> is admitted on the same terms as
// localAttributeDecidable admits one, and declined on the same terms too when
// the inline type's own shape is outside the produced subset. The two are no
// longer asymmetric: §3.2.2.1 dcl.att.global states the same three-tier {type
// definition} chain as §3.2.2.2 dcl.att.local word for word, and the producer
// maps both through declaredType (#733), so a difference here would be the
// fabrication.
func attributeDecidable(el *parser.Element) bool {
	inline := childXSD(el, "simpleType")
	return inline == nil || simpleTypeDecidable(inline)
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
// the same proviso — through complexTypeDecidable, which since #1126 judges an
// anonymous type on the same terms as a named one and says there why anonymity
// narrows nothing. Its <alternative> children are gated by
// alternativeTypesDecidable, shared with elementDecidable.
func localElementDecidable(el *parser.Element) bool {
	if ct := childXSD(el, "complexType"); ct != nil && !complexTypeDecidable(ct) {
		return false
	}
	if inline := childXSD(el, "simpleType"); inline != nil && !simpleTypeDecidable(inline) {
		return false
	}
	return alternativeTypesDecidable(el)
}

// localAttributeDecidable reports whether a LOCAL <attribute> — a child of a
// <complexType>/<restriction> or of an <attributeGroup> definition, both of
// which map through §3.2.2.2 dcl.att.local — is within the producer's decidable
// subset. Like localElementDecidable, and like attributeDecidable since #733, it
// admits an inline anonymous <simpleType> whose own shape is produced (#229),
// and for the same reason declines one that is not.
func localAttributeDecidable(el *parser.Element) bool {
	inline := childXSD(el, "simpleType")
	return inline == nil || simpleTypeDecidable(inline)
}

// simpleTypeDecidable reports whether a <simpleType> (top-level, or an anonymous
// inline one reached through a restriction, list, union or element/attribute
// chain) is decidable. Every §3.16.2.1 alternative is, and so is a document
// naming none of them:
//
//   - <list> (#447): produced in both its forms, so it is admitted. Its
//     itemType= is a deferred by-name reference resolved at finalize, whose
//     src-resolve clause 1.1 failure is genuine; its inline <simpleType> child
//     is recursed into with these same checks.
//   - <union> (#738): produced in all three of its forms — memberTypes= alone,
//     inline <simpleType> children alone, and both at once — so it is admitted
//     on the same terms as <list>. Each memberTypes= item is a deferred by-name
//     reference; every inline child is recursed into. The membership-acyclicity
//     verdict a by-name member makes reachable is xsd's, charged at finalize
//     under cos-st-restricts clause 3.3, so a cyclic membership is a genuine
//     rejection here and not a decline.
//   - <restriction>: admitted outright. Every constraining facet it can carry is
//     produced, including the two folded ones — one assertions facet per
//     restriction (#178, Datatypes §4.3.13) and one enumeration facet per
//     restriction (#740, §4.3.5.2 xr-enumeration). An inline <simpleType> base
//     child (§3.16.3 clause 2) is recursed into.
//   - NONE of the three (#786): admitted. simpleTypeBody rejects that document
//     unconditionally — under §5.1's first bullet and no numbered rule, an
//     unfilled alternative position, which is why the rejection carries no
//     rule ID — so the harness observes "invalid" and no verdict is fabricated.
//
// A <simpleType> naming TWO alternatives is admitted through whichever branch is
// tested first, and correctly: the producer rejects that document outright
// (simpleTypeBody), so the verdict is genuine whichever branch let it through.
//
// EVERY return here is therefore true and the recursion only propagates that:
// the predicate is a constant, and no call site of it gates anything. Retiring
// it collapses attributeDecidable and localAttributeDecidable to constants too
// and rewrites the justification every call site carries, so it is a refactor of
// its own rather than part of the widening that made it one (#786).
func simpleTypeDecidable(el *parser.Element) bool {
	if list := childXSD(el, "list"); list != nil {
		inline := childXSD(list, "simpleType")
		return inline == nil || simpleTypeDecidable(inline)
	}
	if union := childXSD(el, "union"); union != nil {
		for _, inline := range childrenXSD(union, "simpleType") {
			if !simpleTypeDecidable(inline) {
				return false
			}
		}
		return true
	}
	restriction := childXSD(el, "restriction")
	if restriction == nil {
		return true
	}
	for _, child := range restriction.Children() {
		r, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		if r.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if r.Name().Local() == "simpleType" && !simpleTypeDecidable(r) {
			return false
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

// childrenXSD returns every child element of el with expanded name
// {XMLSchemaNS}local, in document order. It is childXSD's repeatable twin, for
// the one gated content model that admits several — <union>'s inline
// <simpleType> children (§3.16.2.4 map.std.union case 1b).
func childrenXSD(el *parser.Element, local string) []*parser.Element {
	var found []*parser.Element
	for _, child := range el.Children() {
		c, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		if name := c.Name(); name.Space() == xsd.XMLSchemaNS && name.Local() == local {
			found = append(found, c)
		}
	}
	return found
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
