package parser

import (
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the COVERAGE CENSUS: what one schema document holds that this
// producer maps to no component (#1029).
//
// # Why the producer answers instead of the consumer
//
// A consumer that must not read a vacuous verdict out of this parser — the
// conformance harness is the standing one — needs to know which constructs the
// producer silently skipped, and until now could only find out by re-walking the
// document against a hand-kept allowlist of what the producer happens to read
// (conformance/schema.go). That allowlist is a second implementation of this
// package's own dispatch (topLevelMapped, parser/produce.go), kept in step by
// hand, and the failure mode of drifting out of step is a FALSE ACCEPT: a
// construct the producer stopped reading, or never read, that the allowlist
// still admits. Reporting the census from the producer makes the answer the
// dispatch's own, exactly as parser.AssemblyReport made the document closure the
// assembly's own rather than the harness's (#272).
//
// # Why ONE pass, not instrumentation at the skip sites
//
// The census is a walk of its own, taken before any component is built, and not
// a side effect of run:
//
//   - run returns at its FIRST error (§4.2's assembly stops there too), so a
//     census accumulated inside it would be truncated at whatever the document's
//     first fault happens to be, and would say "nothing unmapped" for the
//     documents most likely to hold something.
//   - a complex type's base is built ON DEMAND out of document order
//     (buildComplexType's memo), so records appended at skip sites would reach
//     the slice in build order rather than document order (STYLE D2).
//
// It is nonetheless not a SECOND walk: both passes range topLevelDecls
// (produce.go), which is what keeps the census's idea of "children considered"
// from drifting away from run's the way the harness's allowlist drifted away
// from the dispatch (STYLE T4).
//
// # Scope
//
// The top level of <schema> only: the same children run's dispatch walks. Every
// other region — a complex type's content model, a simple type's facets, an
// <override>'s or <redefine>'s children as seen from HERE — is a region of its
// own with its own dispatch, and each is a widening this census does not yet
// carry (#1030). A narrow census is SOUND but incomplete: it never names a
// construct the producer does map, so a consumer may act on what it reports and
// must not read silence as coverage.

// census reports, in document order, every top-level <schema> child of this
// producer's document that topLevelMapped declines — the complement of run's
// dispatch vocabulary, taken over exactly the children run walks.
//
// The children topLevelDecls does not yield are not reported as unmapped
// constructs, and the reasons they are not are stated there — including the
// foreign-namespace child, which its GAP(parser) marker records as settled in
// neither direction (#1036).
//
// The name and location come from the OVERRIDING declaration where §F.2 clause 1
// substitutes one, exactly as run's dispatch reads it, which is never an
// unmapped construct in practice: replacement substitutes only a NAMED top-level
// declaration of the seven kinds §F.2 clause 1 lists, and topLevelMapped admits
// every one of them.
func (p *producer) census() []UnmappedConstruct {
	var unmapped []UnmappedConstruct
	for decl := range p.topLevelDecls {
		if topLevelMapped(decl.Name().Local()) {
			continue
		}
		unmapped = append(unmapped, UnmappedConstruct{
			Name:   xsd.QName{Space: decl.Name().Space(), Local: decl.Name().Local()},
			Reason: UnmappedNoDispatch,
			At:     decl.Loc(),
		})
	}
	return unmapped
}
