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
// A name outside the vocabulary is reported UnmappedNoDispatch wherever this walk
// reaches it, whether or not another pass goes on to REJECT the document over the
// same name: a rejected document decides nothing vacuously, so a report redundant
// with a rejection misleads no consumer. Two shapes are not reported at all, since
// a rejection covers every name they can hold and neither can be a silence: a
// model group's non-particle child (groupParticles' default arm) and a
// <simpleType> <restriction>'s out-of-model child (rejectOutOfModelFacetChildren).
//
// # The census is NAME-based
//
// A child dropped because the POSITION its name fills was already filled is not
// reported — a second <anyAttribute> in an <attributeGroup>. Reporting it would
// make the census's answer depend on how many siblings a name has, which no
// reader can act on without re-walking the document itself.
//
// # Scope
//
// Censused: the top level of <schema>; the child vocabulary of the four
// complex-type derivation alternants and of the implicit-content form; the
// derivation tail — annotation?, (attribute | attributeGroup)*, anyAttribute?,
// assert* (§3.4.2) — wherever those five containers carry it; the body of a
// top-level <group> and of a top-level <attributeGroup>; the <list> and <union>
// alternatives of every <simpleType> any of those reaches.
//
// Since #1047 those five complex-type vocabularies report only names
// checkS4SChildOrder rejects as well, save the ONE name s4sFacetElement admits
// and mappedFacetElement does not — <assertions> under a <simpleContent>
// <restriction>. Since #1380 the top level of <schema> is in that condition too:
// rejectUnmappedTopLevel (produce.go) charges every name topLevelMapped
// declines, so a top-level report now only ever rides on a document run went on
// to reject. The reports are kept: retiring a region is a census change with a
// measurement of its own, and [UnmappedConstruct] states what the top-level arm
// is for once no accepted document can carry one.
//
// NOT censused, each a widening of its own and each a region with its own
// dispatch — the first three covered by a rejection at the position, the fourth
// answered by another producer's census or still to come, and the last six each
// a live uncaught silence. None of the six is TOTAL. rejectS4SFaults
// (produce.go) walks every XSD-namespace element of the document outside
// <appinfo> and <documentation>, so at any depth a <notation> is charged for
// where it stands (rejectMisplacedNotation) and a second <annotation> under one
// parent for its cardinality (rejectRepeatedAnnotations). Every OTHER
// XSD-namespace name is what survives at the six — an <xs:element> written under
// an <openContent>, an <import>, a <group ref>, an <assert>, an <annotation> or
// an <any> is reported by nothing and rejected by nothing.
//
//   - the <complexType> wrapper's own children once it has chosen <simpleContent>
//     or <complexContent>, and those two wrappers' own children: checkS4SChildOrder
//     charges every XSD-namespace child the CHOSEN xs:complexTypeModel disjunct
//     does not admit (#1047), so no silence is possible at those three positions
//     and there is nothing left there for a census to report;
//   - <element>, <attribute> and <simpleType>'s OWN children, on the same footing
//     since #1076, and an <alternative>'s since #1275: each of the four is now
//     ordered against the one content model Appendix A gives its every form
//     (s4sElement, s4sAttribute, s4sSimpleType, s4sAlternative), so a name those
//     models do not admit is charged rather than dropped and no silence is left at
//     those positions either. The <alternative> case is the one whose walk stands
//     beside a src-* clause over the same children: xs:altType is "(annotation?,
//     (simpleType | complexType)?)" (:5137, §3.12.2 :3210) and checkSrcTA charges
//     only HOW MANY of a type attribute, a <complexType> child and a <simpleType>
//     child are present (src-ta, §3.12.3), so it is checkS4SChildOrder alone that
//     answers a further child beside them;
//   - a <notation>'s own children, covered by a rejection written for that one
//     position. topLevelMapped admits the name and censusWalk.topLevel holds no
//     arm for it, but xs:notation extends xs:annotated with three attributes
//     (:5696), leaving "(annotation?)" (§3.14.2, :3376), and rejectNotationContent
//     charges every child element but <annotation> — and every non-whitespace
//     text node besides — so nothing written there is a silence;
//   - an <override>'s or <redefine>'s children as seen from HERE. An <override>'s
//     need no walk from here at all: §F.2 clause 1 substitutes them into the
//     OVERRIDDEN document, whose own producer censuses them through
//     topLevelDecls. A <redefine>'s are definitions of THIS document (§4.2.4
//     clause 4.1.1) and are a region still to come;
//   - an <openContent>'s own children and a <defaultOpenContent>'s, and those of
//     the <any> under either — one silence reached over one code path.
//     xs:openContent is "(annotation?, any?)" (§3.4.2, :1728) and
//     xs:defaultOpenContent is "(annotation?, any)" (§3.17.2, :3787); the <any> of
//     either has type xs:wildcard, which extends xs:annotated with attributes
//     alone (Appendix A, :5356), leaving "(annotation?)" (§3.10.2, :2838). No
//     s4sModel exists for any of the four positions, so checkS4SChildOrder is
//     never invoked against them. checkOpenContentAny and openContentOf find the
//     <any> by name and read the wrapper's mode attribute; wildcardElement reads
//     appliesToEmpty off a <defaultOpenContent>; checkDefaultOpenContent charges
//     that element for carrying no <any>, and for a mode outside
//     interleave|suffix, and for nothing else; produceWildcard reads the <any>'s
//     namespace, notNamespace, notQName and processContents attributes.
//     complexContentChildMapped admits <openContent> while container holds no arm
//     for it, and topLevelMapped admits <defaultOpenContent> while
//     censusWalk.topLevel holds none, so a stray XSD-namespace sibling of either
//     <any> — or a child of an <any> itself — is reported by nothing;
//   - an <include>'s or an <import>'s own children. Both hold "(annotation?)"
//     alone — §4.2.3's summary at :4056 and §4.2.6's at :4181, Appendix A's
//     xs:include (:5535) and xs:import (:5585) extending xs:annotated with the
//     schemaLocation and namespace ATTRIBUTES and no element position — so there
//     is nothing here to descend INTO, and what is missing is again the REPORT:
//     topLevelMapped admits both names, censusWalk.topLevel holds an arm for
//     neither, run's dispatch has an empty arm for both (the assembler consumed
//     them during discovery, parse.go), and no s4sModel orders either.
//     AssemblyReport does not already answer this. Its Unfollowed half records
//     the ·inter-schema-document references· that yielded no DOCUMENT, as an
//     UnfollowedReason and the directive element's own location (#272), and says
//     nothing about that element's CHILDREN: a FOLLOWED <include> yields no
//     UnfollowedDirective at all, and the report's other half carries this very
//     census;
//   - a nested <group ref>'s or <attributeGroup ref>'s own children, uncaught the
//     same way. Both ref forms hold "(annotation?)" alone: Appendix A's
//     xs:groupRef, for the <group> §3.7.2 xr.mgd3 maps to a particle, and
//     xs:attributeGroupRef, which §3.4.2.4's summary spells the same way. They are
//     the REFERENCES, not the top-level DEFINITIONS censused above — xs:namedGroup
//     is "(annotation?, (all | choice | sequence))" and xs:namedAttributeGroup
//     carries the attribute tail — so there is nothing here to descend INTO. What
//     is missing is the REPORT: modelGroup has no <group> arm and container none
//     for <attributeGroup>; produceGroupRefParticle reads ref, minOccurs and
//     maxOccurs, and collectReferencedGroup reads ref, each of them reading name
//     as well through rejectProhibitedRefAttrs (produce.go), which charges the
//     definition-form attribute the reference form prohibits; no s4sModel orders
//     either ref form; and groupParticles' default arm charges a name written
//     DIRECTLY under an <all>/<choice>/<sequence>, not one nested inside a <group
//     ref> it has already recognized;
//   - an <assert>'s own children. xs:assertion extends xs:annotated with the test
//     and xpathDefaultNamespace ATTRIBUTES and no element position of its own,
//     leaving "(annotation?)" (§3.13.2). assertionsOf reads the test attribute
//     through buildXPathExpression, which reads xpathDefaultNamespace off the
//     <assert> too — falling back to this document's <schema>, xpathDefaultNamespace
//     (produce_xpath.go) — and neither of them descends into the element's
//     children; no s4sModel orders an <assert>, and container holds no arm for the
//     name;
//   - an <annotation>'s own children, wherever one stands. No site of this walk
//     descends into an <annotation> or reports one: topLevel, container and
//     namedGroup each admit the name and hold no arm for it, and element,
//     attributeDecl, simpleType and modelGroup report nothing of their own at any
//     position. Nothing maps one either — this producer builds an xsd.Annotation
//     nowhere, which is why topLevelMapped admits <annotation> for §3.15.1's
//     reason rather than for a dispatch's — and no s4sModel orders one.
//     xs:annotation is "(appinfo | documentation)*" (§3.15.2, :3480; Appendix A
//     :5747), and rejectRepeatedAnnotations charges the two shapes it owns, a
//     nested <annotation> at any cardinality and a second <annotation> under one
//     parent, so any other XSD-namespace child of an <annotation> is what
//     survives. The <appinfo> and <documentation> below one are never reached
//     either, nor is anything inside them: that is <xs:any processContents="lax">
//     content (:5727, :5740), no construct of this producer at any depth, and the
//     content rejectS4SFaults stops at for the same reason.
//   - every remaining element the walk STOPS at, by the rule that generates this
//     half of the list rather than by another enumeration: a site descends only
//     where the Censused side above says it does, so an element it reaches and
//     does not descend keeps its OWN children out of the census. No s4sModel
//     orders any of the three positions that rule leaves, and each is an element
//     some pass reaches by name and reads ATTRIBUTES off: a particle <any> and an
//     <anyAttribute> — groupParticles' "any" arm reaches produceWildcard through
//     produceAnyParticle and collectAttributeContent hands it the <anyAttribute>
//     it finds by name, xs:wildcard and the xs:anyAttribute element built on it
//     (Appendix A :5356, :4729) leave "(annotation?)" (§3.10.2, :2838, :2846),
//     and modelGroup holds no "any" arm while container holds none for
//     "anyAttribute"; a <unique>, <key> or <keyref> with the <selector> and
//     <field> under it — xs:keybase is "(annotation?, (selector, field+)?)"
//     (:5648, §3.11.2 :2991) over two "(annotation?)" children (:5599, :5624,
//     §3.11.2 :3010, :3016), constructIdentityConstraint finds the <selector> and
//     the <field>s by name and reads xpath off each, and element holds no arm for
//     any of the three constraint names; and every FACET element under a
//     <simpleType> or <simpleContent> <restriction> — xs:facet is xs:annotated
//     plus value and fixed (xmlschema11-2.md:4001), which xs:noFixedFacet spells
//     "(annotation?)" outright (xmlschema11-2.md:4010), restrictionFacets reads
//     value, fixed and an <assertion>'s test off each and never a facet's
//     children, and neither site that reaches one reports its child —
//     simpleType's <restriction> arm descends into the inline base <simpleType>
//     alone, and container admits every mapped facet name while holding no arm
//     for one.
//
// A narrow census is SOUND but incomplete: it never names a construct the
// producer does map, so a consumer may act on what it reports and must not read
// silence as coverage.

// census reports, in document order, every element of this producer's document
// that no pass of it maps — the complement of the dispatch vocabulary at each
// position the walk reaches, taken over exactly the children those dispatches
// walk. Whether some pass also REJECTS the document over that element is not
// consulted, for the reason the file comment above gives.
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
	case "attribute":
		w.attributeDecl(decl)
	case "simpleType":
		w.simpleType(decl)
	case "complexType":
		w.complexType(decl)
	case "group":
		w.namedGroup(decl)
	case "attributeGroup":
		w.container(decl, attributeGroupChildMapped)
	}
}

// namedGroup censuses a top-level <group> definition (§3.7.2). xs:namedGroup's
// content model (xmlschema11-1.md:5187) is (annotation?, (all | choice |
// sequence)), and buildDefinitionModelGroup reads the compositor child and
// nothing else, so any other name is dropped in silence.
//
// It is dropped ONLY when a body admits a read at all. With no compositor,
// rejectNamedGroupBody charges the first child xs:namedGroup does not admit, so
// that child is named in a verdict rather than skipped and the census must not
// claim it. With TWO, the definition is rejected outright
// (repeatedCompositorChild, #1048) and the walk stops for complexType's reason:
// nothing under a refused document is silently skipped, and which compositor to
// descend is not a census's to choose.
func (w *censusWalk) namedGroup(el *Element) {
	if repeatedCompositorChild(el) != nil {
		return
	}
	compositor := compositorChild(el)
	if compositor == nil {
		return
	}
	for c := range xsdChildren(el) {
		switch c.Name().Local() {
		case "annotation", "all", "choice", "sequence":
		default:
			w.report(c)
		}
	}
	w.modelGroup(compositor)
}

// modelGroup descends through an <all>/<choice>/<sequence> (§3.8.2) and reports
// nothing of its own: groupParticles' dispatch has a default arm that REJECTS
// every name it holds no arm for, so nothing at this position is ever a silence.
// What the descent reaches is the type definitions a local <element> owns.
func (w *censusWalk) modelGroup(group *Element) {
	for c := range xsdChildren(group) {
		switch c.Name().Local() {
		case "element":
			w.element(c)
		case "all", "choice", "sequence":
			w.modelGroup(c)
		}
	}
}

// element descends into the type definitions an <element> OWNS — its inline
// <simpleType>/<complexType> (§3.3.2.1 dcl.elt.common clause 1) and the ones its
// <alternative> children own (§3.12.2 declare-ta) — and reports nothing of its
// own: checkS4SChildOrder charges every XSD-namespace child s4sElement does not
// admit (#1076), and every one s4sAlternative does not admit under an
// <alternative> it reaches (#1275), so no name at either position is a silence.
func (w *censusWalk) element(el *Element) {
	for c := range xsdChildren(el) {
		switch c.Name().Local() {
		case "simpleType":
			w.simpleType(c)
		case "complexType":
			w.complexType(c)
		case "alternative":
			for a := range xsdChildren(c) {
				switch a.Name().Local() {
				case "simpleType":
					w.simpleType(a)
				case "complexType":
					w.complexType(a)
				}
			}
		}
	}
}

// attributeDecl descends into the anonymous simple type an <attribute>
// declaration owns — §3.2.2.1 dcl.att.global and §3.2.2.2 dcl.att.local state
// the same tier-1 {type definition} chain and declaredType maps both — and
// reports nothing of its own: checkS4SChildOrder charges every XSD-namespace child
// s4sAttribute does not admit (#1076), so no name at this position is a silence.
func (w *censusWalk) attributeDecl(el *Element) {
	for c := range xsdChildren(el) {
		if c.Name().Local() == "simpleType" {
			w.simpleType(c)
		}
	}
}

// simpleType censuses one <simpleType> by the §3.16.2.1 alternative it chose, and
// reports nothing of its own at that position: checkS4SChildOrder charges every
// XSD-namespace child s4sSimpleType does not admit (#1076).
//
// The choice is simpleTypeBody's own, so a <simpleType> naming none of the three
// or more than one stops the walk: the producer rejects that document either way,
// under §5.1's first bullet and no numbered rule — none as an unfilled
// alternative position, more than one as a repeat of that same single position —
// and nothing under a rejected alternative is a silence.
//
// The <restriction> alternative reports NOTHING of its own.
// rejectOutOfModelFacetChildren (#972) rejects every XSD-namespace child
// Datatypes §4.1.2's content model has no position for, so the spec's own
// silent-non-mapping carve-out — map.std.common case 2 (xmlschema11-1.md:3635),
// where an unrecognized facet makes the whole <simpleType> "map to no component
// at all" without being in error — is not reachable in this processor: that
// document is refused before it can map to nothing. Only the inline base
// <simpleType> of §3.16.3 clause 2 is descended into.
func (w *censusWalk) simpleType(el *Element) {
	body, err := simpleTypeBody(el)
	if err != nil {
		return
	}
	if body.Name().Local() == "restriction" {
		for c := range xsdChildren(body) {
			if c.Name().Local() == "simpleType" {
				w.simpleType(c)
			}
		}
		return
	}
	w.container(body, simpleTypeAlternativeChildMapped)
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
		w.container(el, complexContentChildMapped)
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
	w.container(derivation, complexContentChildMapped)
}

// container reports every child of parent that mapped declines, and descends
// through the ones it admits into the regions of their own. It is the one shape
// every site below the top level takes, so a region is added by naming its
// vocabulary rather than by writing another walk (STYLE T4).
//
// Reporting and descent share ONE pass over the children, so the census comes
// out in document order however deep a construct sits (STYLE D2).
//
// The switch is over names ACROSS the site vocabularies, and each site's own
// predicate is what makes an arm reachable from it: only the two complex-content
// containers admit a model group, only the <simpleContent> <restriction> and the
// two simple-type alternatives admit a <simpleType>.
func (w *censusWalk) container(parent *Element, mapped func(local string) bool) {
	for c := range xsdChildren(parent) {
		local := c.Name().Local()
		if !mapped(local) {
			w.report(c)
			continue
		}
		switch local {
		case "attribute":
			w.attributeDecl(c)
		case "simpleType":
			w.simpleType(c)
		case "all", "choice", "sequence":
			w.modelGroup(c)
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
// is read by nothing: produceSimpleContent never reaches a particle. The document
// does not survive it either — checkS4SChildOrder rejects a child no position of
// xs:simpleExtensionType admits (#1047) — and the report stands beside that
// rejection rather than instead of it, per the Scope note above.
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

// attributeGroupChildMapped reports whether a child named local of a TOP-LEVEL
// <attributeGroup> definition (§3.6.2) is read. buildAttributeGroup folds its
// body in through the very collectAttributeContent a complex type's tail goes
// through, so the three attribute names are the same ones — but <assert> is NOT
// among them: xs:namedAttributeGroup (xmlschema11-1.md:5502, whose body is the
// xs:attrDecls group at :4720) is "(annotation?, ((attribute | attributeGroup)*,
// anyAttribute?))" with no xs:assertions position, and assertionsOf is reached
// from a complex type alone. The tail vocabulary is therefore not reused whole
// here; a group's <assert> child maps to nothing.
func attributeGroupChildMapped(local string) bool {
	switch local {
	case "annotation", "attribute", "attributeGroup", "anyAttribute":
		return true
	}
	return false
}

// simpleTypeAlternativeChildMapped reports whether a child named local of a
// <list> or a <union> is read. Both alternatives carry the one position
// listItem and unionMembers read — the inline <simpleType> of §3.16.3 clauses 3
// and 4 — so one vocabulary serves both (xs:list at xmlschema11-2.md:3957 and
// xs:union at :3977 differ only in that position's maxOccurs, which is
// cardinality and not vocabulary).
func simpleTypeAlternativeChildMapped(local string) bool {
	return local == "annotation" || local == "simpleType"
}
