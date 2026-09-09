package parser

import (
	"fmt"
	"slices"
	"strings"
)

// s4sAttrRoster is every NO-NAMESPACE attribute name the schema for schema
// documents declares on one s4s element name, together with the Appendix A
// production the names are transcribed from and the line that production begins
// at. All three reach the diagnostic, exactly as s4sModel's grammar, spec and
// model do (produce_s4sorder.go).
//
// declares is the UNION over every production the element name is written
// through, which is what makes it the right roster for the one question
// rejectUndeclaredAttrs asks — is this name admitted ANYWHERE on this element?
// Where an element name has more than one production the productions are
// restrictions of one base type, so the base's own declarations ARE that union
// and one citation is exact: xs:topLevelElement and xs:localElement both restrict
// xs:element (:5086, :5110), xs:topLevelAttribute restricts xs:attribute (:4703),
// xs:topLevelComplexType and xs:localComplexType restrict xs:complexType (:4804,
// :4816), xs:topLevelSimpleType and xs:localSimpleType restrict xs:simpleType
// (xmlschema11-2.md:3876, :3894), xs:namedGroup and xs:groupRef restrict xs:group
// through xs:realGroup (:5187, :5217, :5171), and xs:namedAttributeGroup and
// xs:attributeGroupRef restrict xs:attributeGroup (:5502, :5516).
//
// A restriction NARROWING the base to use="prohibited" is not a narrowing of this
// roster: the name is still declared by the base, so it is a name the grammar
// admits somewhere on the element and rejectUndeclaredAttrs may not claim it.
// rejectProhibitedAttrs (produce.go) and rejectProhibitedRefAttrs are the checks
// that own it, and the roster below is the invariant that keeps the two apart —
// every name either of them prohibits appears here (the tests beside this file
// pin it).
//
// The one place that reading does NOT hold is where NO production of the element
// name declares the attribute, only some base type shared with other element
// names. <choice> and <sequence> are written through xs:explicitGroup (:5229) and
// xs:simpleExplicitGroup (:5246) alone, both of which set name and ref to
// use="prohibited"; the xs:group they descend from (:5155) declares the pair for
// <group>'s sake, and <choice name="x"> is admitted by nothing. So those two, and
// <all> (:5286, and the further-restricted inline declaration at :5192), take
// xs:explicitGroup and xs:all as their citation rather than xs:group.
//
// declares is in the production's own declaration order, read top to bottom, with
// the id xs:annotated contributes (:4426, :4438) last — so the leading run is
// diffable line by line against the cited production, and the trailing id is
// visibly absent from the two productions not built on xs:annotated.
type s4sAttrRoster struct {
	grammar  string
	spec     string
	declares []string
}

// s4sAttrRosters is the roster for every element name Appendix A declares a
// production for, keyed by local name. The map is READ by key alone and never
// ranged into output (STYLE D2); the tests beside this file sort its keys.
//
// These are TRANSCRIBED from Appendix A, not generated, on the footing
// produce_s4sorder.go's s4sModel tables and rejectProhibitedAttrs' own roster
// already stand on: generating them means flattening Appendix A itself —
// resolving xs:attributeGroup refs and the xs:restriction/xs:extension chains
// through xs:annotated — which is its own tool and its own grounding (PRINCIPLES
// 26 targets spec DATA tables: builtin properties, hfn definitions, regex and
// facet tables). Each entry names the production and the line it is quoted from,
// so an entry is diffable against the spec text one lookup away.
//
// A name absent from this table has NO roster and is never charged: an element in
// the XSD namespace whose local name Appendix A declares nothing for is a fault of
// its own — the element-side hole #1380 and #1036 own — and the precisionDecimal
// extension facets <maxScale> and <minScale> (xsd-precisionDecimal.md §4.2, §4.3)
// have an XML Representation Summary and no Appendix A production at all, so they
// are transcribed nowhere here and their attributes go unchecked. That is the
// direction this check may err in, and it is the same one s4sFacetElement's own
// over-admission of the pair takes.
var s4sAttrRosters = map[string]s4sAttrRoster{
	// xs:schema extends xs:openAttrs DIRECTLY rather than through xs:annotated —
	// the carve-out xs:annotated's own documentation states, "extended by all types
	// which allow annotation other than <schema> itself" (:4429) — so its id is its
	// own declaration, not an inherited one. xml:lang is declared here too
	// (:4581) and is a PREFIXED name this check never reaches.
	"schema": {"xs:schema", "xmlschema11-1.md:4546", []string{
		"targetNamespace", "version", "finalDefault", "blockDefault",
		"attributeFormDefault", "elementFormDefault", "defaultAttributes",
		"xpathDefaultNamespace", "id",
	}},

	// xs:element pulls in xs:defRef (:4636, name and ref) and xs:occurs (:4627,
	// minOccurs and maxOccurs); every one of the five xs:topLevelElement prohibits
	// is among them.
	"element": {"xs:element", "xmlschema11-1.md:5044", []string{
		"name", "ref", "type", "substitutionGroup", "minOccurs", "maxOccurs",
		"default", "fixed", "nillable", "abstract", "final", "block", "form",
		"targetNamespace", "id",
	}},

	// xs:attribute pulls in xs:defRef (:4636) and declares no occurrence pair at
	// any level, the asymmetry rejectProhibitedAttrs' doc records.
	"attribute": {"xs:attribute", "xmlschema11-1.md:4677", []string{
		"name", "ref", "type", "use", "default", "fixed", "form",
		"targetNamespace", "inheritable", "id",
	}},

	"complexType": {"xs:complexType", "xmlschema11-1.md:4778", []string{
		"name", "mixed", "abstract", "final", "block", "defaultAttributesApply",
		"id",
	}},

	"simpleType": {"xs:simpleType", "xmlschema11-2.md:3861", []string{
		"final", "name", "id",
	}},

	// <restriction> is written through FOUR productions and base is the only
	// attribute any of them declares: xs:restrictionType (:4831, base required),
	// its two restrictions xs:complexRestrictionType (:4850) and
	// xs:simpleRestrictionType (:4960), and the datatypes-side <restriction>
	// element (xmlschema11-2.md:3940, base optional).
	"restriction": {"xs:restrictionType", "xmlschema11-1.md:4831", []string{
		"base", "id",
	}},

	// xs:simpleExtensionType (:4979) restricts xs:extensionType and declares no
	// attribute of its own.
	"extension": {"xs:extensionType", "xmlschema11-1.md:4873", []string{
		"base", "id",
	}},

	"complexContent": {"xs:complexContent", "xmlschema11-1.md:4887", []string{
		"mixed", "id",
	}},

	"simpleContent": {"xs:simpleContent", "xmlschema11-1.md:4995", []string{
		"id",
	}},

	"openContent": {"xs:openContent", "xmlschema11-1.md:4909", []string{
		"mode", "id",
	}},

	"defaultOpenContent": {"xs:defaultOpenContent", "xmlschema11-1.md:4934", []string{
		"appliesToEmpty", "mode", "id",
	}},

	// xs:group is the base of every group form, and its xs:defRef pair is what
	// xs:namedGroup requires and xs:groupRef prohibits, and the reverse; xs:occurs
	// is what xs:namedGroup prohibits and xs:groupRef keeps.
	"group": {"xs:group", "xmlschema11-1.md:5155", []string{
		"name", "ref", "minOccurs", "maxOccurs", "id",
	}},

	// xs:explicitGroup sets name and ref to use="prohibited" and xs:simpleExplicitGroup
	// (:5246) goes on to prohibit the occurrence pair, so the union is xs:occurs alone.
	"choice":   {"xs:explicitGroup", "xmlschema11-1.md:5229", []string{"minOccurs", "maxOccurs", "id"}},
	"sequence": {"xs:explicitGroup", "xmlschema11-1.md:5229", []string{"minOccurs", "maxOccurs", "id"}},

	// xs:all restricts xs:explicitGroup and re-declares the occurrence pair over
	// the 0|1 enumerations; the inline declaration xs:namedGroup writes at :5192
	// prohibits both.
	"all": {"xs:all", "xmlschema11-1.md:5286", []string{"minOccurs", "maxOccurs", "id"}},

	"attributeGroup": {"xs:attributeGroup", "xmlschema11-1.md:5492", []string{
		"name", "ref", "id",
	}},

	// The <any> ELEMENT (:5364) extends xs:wildcard with notQName and xs:occurs;
	// the <any> inside an <openContent> or a <defaultOpenContent> has type
	// xs:wildcard bare (:4918, :4942), so the union is the element's. xs:wildcard
	// (:5356) is xs:annotated plus xs:anyAttrGroup (:5336).
	"any": {"xs:any", "xmlschema11-1.md:5364", []string{
		"namespace", "notNamespace", "processContents", "notQName",
		"minOccurs", "maxOccurs", "id",
	}},

	// <anyAttribute> is xs:wildcard plus notQName and NO occurrence pair — an
	// attribute wildcard is not a particle.
	"anyAttribute": {"xs:anyAttribute", "xmlschema11-1.md:4729", []string{
		"namespace", "notNamespace", "processContents", "notQName", "id",
	}},

	"assert": {"xs:assertion", "xmlschema11-1.md:4749", []string{
		"test", "xpathDefaultNamespace", "id",
	}},

	// The <assertion> FACET (xmlschema11-2.md:4182) has the same xs:assertion type
	// as an <assert>, quoted at its own line because the two are separate element
	// declarations.
	"assertion": {"xs:assertion", "xmlschema11-2.md:4182", []string{
		"test", "xpathDefaultNamespace", "id",
	}},

	"alternative": {"xs:altType", "xmlschema11-1.md:5137", []string{
		"test", "type", "xpathDefaultNamespace", "id",
	}},

	"include":  {"xs:include", "xmlschema11-1.md:5535", []string{"schemaLocation", "id"}},
	"import":   {"xs:import", "xmlschema11-1.md:5585", []string{"namespace", "schemaLocation", "id"}},
	"redefine": {"xs:redefine", "xmlschema11-1.md:5548", []string{"schemaLocation", "id"}},
	"override": {"xs:override", "xmlschema11-1.md:5567", []string{"schemaLocation", "id"}},

	"selector": {"xs:selector", "xmlschema11-1.md:5599", []string{"xpath", "xpathDefaultNamespace", "id"}},
	"field":    {"xs:field", "xmlschema11-1.md:5624", []string{"xpath", "xpathDefaultNamespace", "id"}},

	// <unique> and <key> have type xs:keybase outright (:5672, :5678); <keyref>
	// extends it with refer.
	"unique": {"xs:keybase", "xmlschema11-1.md:5648", []string{"name", "ref", "id"}},
	"key":    {"xs:keybase", "xmlschema11-1.md:5648", []string{"name", "ref", "id"}},
	"keyref": {"xs:keyref", "xmlschema11-1.md:5683", []string{"name", "ref", "refer", "id"}},

	"notation": {"xs:notation", "xmlschema11-1.md:5696", []string{"name", "public", "system", "id"}},

	// <annotation> extends xs:openAttrs and declares its own id (:5760), the same
	// carve-out <schema> takes.
	"annotation": {"xs:annotation", "xmlschema11-1.md:5747", []string{"id"}},

	// <appinfo> and <documentation> are the TWO productions built on no named base
	// at all — a bare mixed complex type over <xs:any processContents="lax">
	// content — so neither carries an id and neither inherits xs:openAttrs' own
	// wildcard; each declares its own <xs:anyAttribute namespace="##other"> instead
	// (:5729, :5743). <documentation> also declares xml:lang (:5742), a PREFIXED
	// name this check never reaches.
	"appinfo":       {"xs:appinfo", "xmlschema11-1.md:5720", []string{"source"}},
	"documentation": {"xs:documentation", "xmlschema11-1.md:5733", []string{"source"}},

	"list":  {"xs:list", "xmlschema11-2.md:3957", []string{"itemType", "id"}},
	"union": {"xs:union", "xmlschema11-2.md:3977", []string{"memberTypes", "id"}},

	// The facet elements. xs:facet (xmlschema11-2.md:4001) is xs:annotated plus a
	// required value and an optional fixed, and every facet below is that type or a
	// restriction of it. Only xs:noFixedFacet (:4010) sets fixed to
	// use="prohibited"; xs:numFacet (:4053), xs:intFacet (:4066) and the four
	// elements with inline restrictions narrow value's TYPE and leave fixed
	// declared, so fixed stays in their rosters.
	"minExclusive":     {"xs:facet", "xmlschema11-2.md:4001", []string{"value", "fixed", "id"}},
	"minInclusive":     {"xs:facet", "xmlschema11-2.md:4001", []string{"value", "fixed", "id"}},
	"maxExclusive":     {"xs:facet", "xmlschema11-2.md:4001", []string{"value", "fixed", "id"}},
	"maxInclusive":     {"xs:facet", "xmlschema11-2.md:4001", []string{"value", "fixed", "id"}},
	"totalDigits":      {"xs:totalDigits", "xmlschema11-2.md:4078", []string{"value", "fixed", "id"}},
	"fractionDigits":   {"xs:numFacet", "xmlschema11-2.md:4053", []string{"value", "fixed", "id"}},
	"length":           {"xs:numFacet", "xmlschema11-2.md:4053", []string{"value", "fixed", "id"}},
	"minLength":        {"xs:numFacet", "xmlschema11-2.md:4053", []string{"value", "fixed", "id"}},
	"maxLength":        {"xs:numFacet", "xmlschema11-2.md:4053", []string{"value", "fixed", "id"}},
	"whiteSpace":       {"xs:whiteSpace", "xmlschema11-2.md:4136", []string{"value", "fixed", "id"}},
	"explicitTimezone": {"xs:explicitTimezone", "xmlschema11-2.md:4189", []string{"value", "fixed", "id"}},

	// <enumeration> has type xs:noFixedFacet and <pattern> restricts it further
	// (xmlschema11-2.md:4162), so fixed is prohibited on both and value is the
	// whole roster.
	"enumeration": {"xs:noFixedFacet", "xmlschema11-2.md:4010", []string{"value", "id"}},
	"pattern":     {"xs:pattern", "xmlschema11-2.md:4162", []string{"value", "id"}},
}

// rejectUndeclaredAttrs rejects el for carrying a NO-NAMESPACE attribute that no
// Appendix A production of el's element name declares — a typo for a real
// attribute name, or an invented one. el is an element in the XSD namespace; the
// walk that reaches it (rejectS4SFaults, produce.go) is what establishes that.
//
// The rule is xs:openAttrs (:4412-:4425): its whole content is <xs:anyAttribute
// namespace="##other" processContents="lax"/>, so an element admits the attributes
// its own type declares plus attributes from OTHER namespaces, and nothing else.
// Every s4s production either reaches that type or writes the same ##other
// wildcard inline, which is what <appinfo> and <documentation> do (:5729, :5743)
// since they extend nothing. An unprefixed name the type does not declare is
// admitted by neither half, and the document is not ·valid· with respect to the
// schema for schema documents: §2.4 clause 1 (sd-valid, :615), restated as an
// error condition at §5.1 (:4289, :4296).
//
// processContents="lax" on that wildcard governs how DEEPLY an admitted
// other-namespace attribute is then validated. It is not an admission rule and
// says nothing about a no-namespace attribute, which the wildcard's ##other never
// reaches in the first place.
//
// The fault carries NO numbered rule ID, on the footing rejectProhibitedAttrs and
// checkS4SChildOrder already derive in full and xsderr/doc.go's s4s-grammar
// section states: sd-valid is anchored but absent from Appendix B's three tables,
// and no src-* covers the grammar by construction (§2.3's gloss-src, :607).
// Charging src-element, src-attribute or any other cataloged ID here would be a
// fabricated verdict, and so would putting "sd-valid" itself in the Rule position
// (STYLE E2). The plain wrapped error is the whole class's shape.
//
// This is NOT a widening of rejectProhibitedAttrs and does not subsume it. That
// check asks whether a name the grammar admits SOMEWHERE on this element is
// forbidden on the FORM in hand — ref on a top-level <element>, use on a top-level
// <attribute> — and this one asks whether the name is admitted on the element at
// all. Every name that check prohibits is in this one's roster, so the two never
// answer the same attribute, and neither is a partial implementation of the other.
//
// A PREFIXED attribute is left alone whatever its local name, which is the
// ##other half of the wildcard doing its work: xml:lang on a <schema>, xml:base
// anywhere, a vc:* attribute conditional-inclusion pre-processing did not strip,
// an application's own annotation attribute. Element.Attr (tree.go) is the
// primitive that draws the line and this check reads the same attrs list it does
// (STYLE T4).
//
// The attributes are checked in the DOCUMENT's own order, so an element writing
// more than one undeclared attribute is always reported at the first of them
// (STYLE D2).
func rejectUndeclaredAttrs(el *Element) error {
	local := el.Name().Local()
	roster, ok := s4sAttrRosters[local]
	if !ok {
		return nil
	}
	for _, a := range el.attrs {
		if a.Name().Space() != "" {
			continue
		}
		name := a.Name().Local()
		if slices.Contains(roster.declares, name) {
			continue
		}
		return fmt.Errorf("parser: <%s> at %s carries a %s attribute, which the schema for schema documents declares nowhere on <%s>: %s (%s) declares %s, and beyond those it admits attributes from other namespaces alone (the ##other wildcard of xs:openAttrs, xmlschema11-1.md:4412)",
			local, el.Loc(), name, local, roster.grammar, roster.spec, strings.Join(roster.declares, ", "))
	}
	return nil
}
