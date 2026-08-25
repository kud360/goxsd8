package parser

import (
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the COVERAGE CENSUS: what one schema document holds that this
// producer maps to no component (#846).
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
// # Scope
//
// The top level of <schema> only: the same children run's dispatch walks. Every
// other region — a complex type's content model, a simple type's facets, an
// <override>'s or <redefine>'s children as seen from HERE — is a region of its
// own with its own dispatch, and each is a widening this census does not yet
// carry. A narrow census is SOUND but incomplete: it never names a construct the
// producer does map, so a consumer may act on what it reports and must not read
// silence as coverage.

// census reports, in document order, every top-level <schema> child of this
// producer's document that topLevelMapped declines — the complement of run's
// dispatch vocabulary, taken over exactly the children run walks.
//
// The three skips it shares with run are not reported as unmapped constructs:
//
//   - a non-element child (comment, text, PI) maps to nothing anywhere and is
//     not a construct;
//   - a child outside the XSD namespace is an OPEN GAP, not a settled skip:
//     rejectS4SFaults returns at its namespace test (produce.go) and no other
//     producer pass rejects such a child either, so today it is NEITHER rejected
//     as a §5.1 grammar fault NOR reported here. A later region-widening session
//     (#846) decides which of the two it becomes; until then a consumer must not
//     read this census as covering it;
//   - a child a ·redefinition· in force over this document excepts (§4.2.4
//     clause 4.1.2) IS mapped — by the REDEFINING document's producer, in its
//     own document-order position there — so it is unmapped only from this
//     reading's vantage, which is not what this reports.
//
// The name and location come from the OVERRIDING declaration where §F.2 clause 1
// substitutes one (p.ov.replacement, as run dispatches), which is never an
// unmapped construct in practice: replacement substitutes only a NAMED top-level
// declaration of the seven kinds §F.2 clause 1 lists, and topLevelMapped admits
// every one of them.
func (p *producer) census() []UnmappedConstruct {
	var unmapped []UnmappedConstruct
	for _, child := range p.schemaElem.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if p.rd.excepts(el) {
			continue
		}
		decl := p.ov.replacement(el)
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
