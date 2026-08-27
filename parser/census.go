package parser

import (
	"iter"

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
// Its top level is nonetheless not a SECOND walk: both passes range
// topLevelDecls (produce.go), which is what keeps the census's idea of "children
// considered" from drifting away from run's the way the harness's allowlist
// drifted away from the dispatch (STYLE T4).
//
// # What a site vocabulary states
//
// Below the top level the census walks region by region, and each region states
// the ONE thing that decides it: the set of child names some pass of this
// producer READS at that position. It is not the schema for schema documents'
// content model for the position — the two differ, and where they do it is the
// mapping that governs. xs:simpleRestrictionType admits <assertions> in its
// facet position and restrictionFacets folds it into nothing, so the name is
// s4s-legal and unmapped at once (mappedFacetElement records the seam).
//
// A name outside the vocabulary is reported UnmappedNoDispatch only where no
// pass rejects it either. Two shapes therefore stay out of the census although
// no component comes of them, and both are rejections rather than silences: a
// model group's non-particle child (groupParticles' default arm) and a
// <simpleType> <restriction>'s out-of-model child
// (rejectOutOfModelFacetChildren).
//
// # The census is NAME-based
//
// A child dropped because the POSITION its name fills was already filled is not
// reported — a second <all> in a named <group>'s body, a second <anyAttribute>
// in an <attributeGroup>. Reporting it would make the census's answer depend on
// how many siblings a name has, which no reader can act on without re-walking
// the document itself.
//
// # Scope
//
// Censused: the top level of <schema>; the child vocabulary of the four
// complex-type derivation alternants and of the implicit-content form; the
// derivation tail — annotation?, (attribute | attributeGroup)*, anyAttribute?,
// assert* (§3.4.2) — wherever those five containers carry it.
//
// NOT censused, each a widening of its own and each a region with its own
// dispatch:
//
//   - the model-group region, the named <group> and <attributeGroup> bodies, and
//     the simple-type alternatives, which the next commits of #1030 carry;
//   - <element>, <attribute> and <simpleType>'s OWN children: each drops a name
//     its content model does not admit in silence, so each holds real unmapped
//     constructs, but reporting them flags documents conformance's shape gate
//     still admits — a latent false accept of that gate, filed rather than
//     ridden along here;
//   - the <complexType> wrapper's own children once it has chosen <simpleContent>
//     or <complexContent>, and those two wrappers' own children, on the same
//     footing: xs:complexTypeModel's other disjunct admits <attribute> and a
//     model group, checkS4SChildOrder deliberately does not charge a child the
//     CHOSEN alternative excludes, and nothing maps it;
//   - an <override>'s or <redefine>'s children as seen from HERE. An <override>'s
//     need no walk from here at all: §F.2 clause 1 substitutes them into the
//     OVERRIDDEN document, whose own producer censuses them through
//     topLevelDecls. A <redefine>'s are definitions of THIS document (§4.2.4
//     clause 4.1.1) and are a region still to come.
//
// A narrow census is SOUND but incomplete: it never names a construct the
// producer does map, so a consumer may act on what it reports and must not read
// silence as coverage.

// census reports, in document order, every element of this producer's document
// that no pass of it maps and none of its passes rejects — the complement of the
// dispatch vocabulary at each position the walk reaches, taken over exactly the
// children those dispatches walk.
//
// The children topLevelDecls does not yield are not reported as unmapped
// constructs, and the reasons they are not are stated there — including the
// foreign-namespace child, which its GAP(parser) marker records as settled in
// neither direction (#1036).
//
// The name and location of a top-level child come from the OVERRIDING
// declaration where §F.2 clause 1 substitutes one, exactly as run's dispatch
// reads it, and the walk below descends into that declaration's subtree for the
// same reason: the overriding declaration is the one this document's producer
// maps.
func (p *producer) census() []UnmappedConstruct {
	var w censusWalk
	for decl := range p.topLevelDecls {
		w.topLevel(decl)
	}
	return w.unmapped
}

// censusWalk accumulates one document's census. It holds no producer state: the
// census is taken before the first pass that can fail (assembly.compile), so
// every question it answers is a question about the document tree alone.
type censusWalk struct {
	unmapped []UnmappedConstruct
}

// report records el as mapped by nothing. Callers report BEFORE descending into
// a sibling, so the slice comes out in document order (STYLE D2).
func (w *censusWalk) report(el *Element) {
	w.unmapped = append(w.unmapped, UnmappedConstruct{
		Name:   xsd.QName{Space: el.Name().Space(), Local: el.Name().Local()},
		Reason: UnmappedNoDispatch,
		At:     el.Loc(),
	})
}

// xsdChildren yields el's child ELEMENTS in the XSD namespace, in document
// order. Every site below walks its children through it: a non-element child
// maps to nothing anywhere and is not a construct, and a foreign-namespace one
// is the open question topLevelDecls' GAP(parser) marker owns (#1036), left the
// same answer at every depth.
func xsdChildren(el *Element) iter.Seq[*Element] {
	return func(yield func(*Element) bool) {
		for _, child := range el.Children() {
			c, ok := child.(*Element)
			if !ok || c.Name().Space() != xsd.XMLSchemaNS {
				continue
			}
			if !yield(c) {
				return
			}
		}
	}
}

// topLevel censuses one <schema> child run's dispatch considers, and descends
// into the declarations whose own regions the census carries.
func (w *censusWalk) topLevel(decl *Element) {
	if !topLevelMapped(decl.Name().Local()) {
		w.report(decl)
		return
	}
	switch decl.Name().Local() {
	case "element":
		w.element(decl)
	case "complexType":
		w.complexType(decl)
	}
}

// element descends into the type definitions an <element> OWNS — its inline
// <simpleType>/<complexType> (§3.3.2.1 dcl.elt.common clause 1) and the ones its
// <alternative> children own (§3.12.2 declare-ta) — and reports nothing of its
// own, per the Scope note above.
func (w *censusWalk) element(el *Element) {
	for c := range xsdChildren(el) {
		switch c.Name().Local() {
		case "complexType":
			w.complexType(c)
		case "alternative":
			for a := range xsdChildren(c) {
				if a.Name().Local() == "complexType" {
					w.complexType(a)
				}
			}
		}
	}
}

// complexType censuses one <complexType> by the disjunct of xs:complexTypeModel
// (§3.4.2, xmlschema11-1.md:1649) it wrote: the implicit-content form is a
// container of its own, and the two wrapped forms delegate to the alternant
// under the wrapper.
//
// A <complexType> carrying BOTH a <simpleContent> and a <complexContent>, or two
// of either, is rejected outright (repeatedContentAlternative), so nothing under
// it is silently skipped and the walk stops.
func (w *censusWalk) complexType(el *Element) {
	if repeatedContentAlternative(el) != nil {
		return
	}
	sc := childElement(el, xsd.XMLSchemaNS, "simpleContent")
	cc := childElement(el, xsd.XMLSchemaNS, "complexContent")
	if sc == nil && cc == nil {
		w.complexContentContainer(el)
		return
	}
	if sc != nil {
		w.simpleContentAlternant(sc)
		return
	}
	w.complexContentAlternant(cc)
}

// simpleContentAlternant censuses the <restriction> or <extension> under a
// <simpleContent> (§3.4.2.2). A wrapper carrying both is rejected
// (repeatedDerivationAlternant) and one carrying neither is too
// (produceSimpleContent), so the walk stops for either.
func (w *censusWalk) simpleContentAlternant(sc *Element) {
	if repeatedDerivationAlternant(sc) != nil {
		return
	}
	if ext := childElement(sc, xsd.XMLSchemaNS, "extension"); ext != nil {
		w.container(ext, simpleExtensionChildMapped)
		return
	}
	if r := childElement(sc, xsd.XMLSchemaNS, "restriction"); r != nil {
		w.container(r, simpleRestrictionChildMapped)
	}
}

// complexContentAlternant censuses the <restriction> or <extension> under a
// <complexContent> (§3.4.2.3), which carries the same children the
// implicit-content form does. Both-or-neither stops the walk for
// simpleContentAlternant's reasons (produceComplexContent charges the neither
// case).
func (w *censusWalk) complexContentAlternant(cc *Element) {
	if repeatedDerivationAlternant(cc) != nil {
		return
	}
	derivation, _ := derivationAlternant(cc)
	if derivation == nil {
		return
	}
	w.complexContentContainer(derivation)
}

// complexContentContainer censuses a container whose children are the structural
// positions "openContent?, (group | all | choice | sequence)?" followed by the
// derivation tail: the implicit-content <complexType> itself (§3.4.2.3.2) and
// each <complexContent> alternant (§3.4.2.3.3).
func (w *censusWalk) complexContentContainer(parent *Element) {
	w.container(parent, complexContentChildMapped)
}

// container reports every child of parent that mapped declines. It is the one
// shape every site below the top level takes, so a region is added by naming its
// vocabulary rather than by writing another walk (STYLE T4).
func (w *censusWalk) container(parent *Element, mapped func(local string) bool) {
	for c := range xsdChildren(parent) {
		if !mapped(c.Name().Local()) {
			w.report(c)
		}
	}
}

// derivationTailMapped reports whether a child named local of a complex-type
// derivation alternant is read by some pass of this producer where the alternant
// ends — xs:attrDecls followed by xs:assertions (xmlschema11-1.md:4720, :4743),
// "((attribute | attributeGroup)*, anyAttribute?), assert*", plus the
// <annotation> every one of them opens with.
//
// <attribute>, <attributeGroup> and <anyAttribute> are read by
// collectAttributeContent (§3.4.2.4 clause 3, §3.4.2.5 clause 2,
// produce_complex.go); <assert> by assertionsOf (§3.4.2.1 clause 2,
// produce_xpath.go). <annotation> is read by NO pass, and is admitted for
// topLevelMapped's reason: §3.15.1 puts annotations outside ·validation·
// altogether, so no verdict is short by the Annotation that is never built.
//
// The tail is one vocabulary and is stated once, so a site that gains a child
// kind cannot quietly disagree with the others about what it holds — the shape
// attrDeclsDecidable holds in conformance/schema.go, moved to the side that
// knows the answer.
func derivationTailMapped(local string) bool {
	switch local {
	case "annotation", "attribute", "attributeGroup", "anyAttribute", "assert":
		return true
	}
	return false
}

// simpleExtensionChildMapped reports whether a child named local of a
// <simpleContent> <extension> is read. The tail is the WHOLE vocabulary:
// xs:simpleExtensionType (:4979) is "(annotation?, ((attribute |
// attributeGroup)*, anyAttribute?), assert*)" and §3.4.2.2 computes {content
// type} from the resolved base alone (cases 3-5), so a model group written here
// is read by nothing — neither produceSimpleContent, which never reaches a
// particle, nor checkS4SChildOrder, which skips a name no position of the model
// admits.
func simpleExtensionChildMapped(local string) bool {
	return derivationTailMapped(local)
}

// simpleRestrictionChildMapped reports whether a child named local of a
// <simpleContent> <restriction> is read. xs:simpleRestrictionType (:4960) puts
// two positions in front of the tail, and both are mapped: the inline
// <simpleType> and the facet children that §3.4.2.2 cases 1-2 synthesize the
// anonymous simple type from (simpleContentSimpleType, produce_complex.go).
//
// The facet membership is the MAPPING one (mappedFacetElement), not the ordering
// one — see that predicate for the two names that part.
func simpleRestrictionChildMapped(local string) bool {
	return derivationTailMapped(local) || local == "simpleType" || mappedFacetElement(local)
}

// complexContentChildMapped reports whether a child named local of the
// implicit-content <complexType> form, or of a <complexContent>
// <restriction>/<extension>, is read. It is the tail behind the two structural
// positions xs:complexRestrictionType (:4850) and xs:extensionType (:4873) share
// with it, "openContent?, (group | all | choice | sequence)?" — <openContent>
// read by openContentType (§3.4.2.3.3 clauses 5-6) and the model group by
// buildComplexContentType (clause 2).
func complexContentChildMapped(local string) bool {
	switch local {
	case "openContent", "group", "all", "choice", "sequence":
		return true
	}
	return derivationTailMapped(local)
}
