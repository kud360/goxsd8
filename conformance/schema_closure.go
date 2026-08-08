package conformance

import (
	"github.com/kud360/goxsd8/parser"
)

// This file holds the schema lane's closure gate: the two questions
// execSchemaCase asks about the set of schema documents an assembly actually
// consumed (issues #242, #182, #183, #276, #272).
//
// # Why the closure, not the root
//
// schema.go's false-accept guard (schemaShapeDecidable) must hold for EVERY
// document parser.ParseReport assembles, not just the root. A single-document
// lane could gate on the one document it read; a multi-document lane cannot,
// because an <include>d document holding a representation the producer silently
// SKIPS (§3.1.2) — an inline anonymous type, a list/union simpleType — or one it
// builds with a rule judging it only in part would let a schema-INVALID assembly
// "Parse" cleanly, a FALSE ACCEPT of exactly the kind schema.go's step-3
// allowlist exists to prevent.
//
// # Why the parser reports the closure instead of the harness re-walking it
//
// Until #272 this file carried its own <include>/<override>/<import> walk,
// because parser.Parse could not be asked which documents it read. That walk was
// a second implementation of §4.2's composition edges, and under-discovering was
// not a harmless conservatism: a document the harness missed but the parser read
// was a document whose shape was never gated — the false accept back again. So
// the walk was kept in step with parser/parse.go by hand, and every new
// composition edge had to be ported twice.
//
// parser.ParseReport now reports the assembly directly (parser.AssemblyReport):
// every document read, in discovery order, and every ·inter-schema-document
// reference· (§4.2.1) it could not follow to one. The gate below reads that
// report, so the gated set is the assembled set BY CONSTRUCTION rather than by
// two walks agreeing — no §4.3.2 clause 4 base-URI algorithm, no resolver
// protocol and no load-once identity is re-implemented here.
//
// # Why running the shape check on the RAW document is correct
//
// Chameleon inclusion (§4.2.3 clause 2.3, §F.1) is purely a NAMESPACE-level
// transformation: per the c-chamvalidi note it "(a) adds a targetNamespace
// [attribute] to D2 ... and (b) updates all unqualified QName references so that
// their namespace names become the ·actual value· of the targetNamespace
// [attribute]" (oracle grounding, issue #242). It restructures no elements, so the
// decidability verdict on the raw document is the verdict on the coerced one and
// the gate needs no chameleon awareness of its own. src-include itself likewise
// imposes no shape constraint on D2 — only existence and targetNamespace
// agreement — so shape decidability is a property this gate establishes
// independently of anything parser.ParseReport checks.
//
// # The report is a SUBSET of what the old walk reached, which is the safe side
//
// The walk deliberately kept going past conditions on which the parser gives up
// (a targetNamespace mismatch, a D2 that is not a <schema>), so it gated
// documents the parser never read. The report cannot: a document the assembly
// abandoned before reaching is not in it. Those cases are now decided on the
// genuine src-include/src-override/src-import rejection the parser really
// emitted, where before the harness declined on an unreached document's shape —
// a decline becoming a decision, never the reverse (issue #272).
//
// Under-gating in the other direction is impossible for the same reason the
// report exists: every document the parse consumed is in it.

// closureDecidable reports whether every schema document the assembly consumed
// lies within the producer's decidable subset (schemaShapeDecidable, the step-3
// allowlist schema.go documents). One out-of-subset document declines the whole
// case, wherever in the closure it sits — that is the false-accept guard.
//
// The same document can appear twice in the report (once per namespace or
// ·override pre-processing· it was reached under, parser.AssemblyReport's own
// documentation); checking it twice is the same verdict twice, so no dedup index
// is kept here (STYLE D3/D4).
func closureDecidable(report *parser.AssemblyReport) bool {
	for _, d := range report.Documents() {
		if !schemaShapeDecidable(d.Doc) {
			return false
		}
	}
	return true
}

// closureReached reports whether the assembly consumed the document at the
// resolved location resolved. It exists for the multi-document schemaTest, whose
// extra declared documents may only be decided when the assembly from the first
// one provably reached them (extraDocsInClosure).
//
// The comparison is against parser.AssembledDocument.Location — the RESOLVED
// location, which is the loader.Resolver's own dedup identity — so the caller
// must resolve its candidate through the SAME resolver the parse used before
// asking. The namespace a document was reached under is deliberately not part of
// the question: one document can be reached under several (a chameleon
// <include> and a bare <import>), and what is asked here is only whether this
// location was assembled.
//
// The linear scan is the whole implementation on purpose — one case's closure is
// scanned once per extra document, so an index would be redundant state (STYLE
// D3) for no measured hot path (STYLE D4).
func closureReached(report *parser.AssemblyReport, resolved string) bool {
	for _, d := range report.Documents() {
		if d.Location == resolved {
			return true
		}
	}
	return false
}
