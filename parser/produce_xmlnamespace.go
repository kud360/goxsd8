package parser

import (
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// languageName, stringName, ncNameName and idName are the expanded names of the
// builtin simple types the XML namespace's four attribute declarations take
// their {type definition} from, directly or as the base of an anonymous one;
// anyURIName (produce_complex.go) is the fifth. [builtin.Seed] always seeds all
// of them.
var (
	languageName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "language"}
	stringName   = xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}
	ncNameName   = xsd.QName{Space: xsd.XMLSchemaNS, Local: "NCName"}
	idName       = xsd.QName{Space: xsd.XMLSchemaNS, Local: "ID"}
)

// composedDocument is one document of an assembly as supplyXMLNamespace reads
// it: the <schema> element information item whose own <import> children license
// namespaces (src-resolve clause 4.2.2), and the EFFECTIVE target namespace the
// document's components are minted in.
//
// It is that predicate's PARAMETER shape and nothing's state (STYLE D3):
// assembly.compile projects each [discovered] onto it and Produce projects the
// lone document it was handed, so the one rule §4.2.6.1 licenses is encoded
// once for both entry points instead of once per entry point.
type composedDocument struct {
	schema *Element
	target string
}

// composedDocuments projects the assembly's discoveries onto what
// supplyXMLNamespace reads. It walks the docs SLICE, never the loaded index, so
// nothing here depends on map iteration order (STYLE D2) — the predicate's
// answer would not, but the walk is the one every other reader of a.docs uses.
func (a *assembly) composedDocuments() []composedDocument {
	docs := make([]composedDocument, 0, len(a.docs))
	for _, d := range a.docs {
		docs = append(docs, composedDocument{schema: d.doc.Root(), target: d.tns})
	}
	return docs
}

// supplyXMLNamespace reports whether this assembly must be handed the built-in
// components for the XML namespace — §1.3.2's first ·namespace with special
// status· (xmlschema11-1.md:217) — which is true exactly when some document of
// it carries <import namespace="http://www.w3.org/XML/1998/namespace"> AND no
// document of it was composed under that namespace.
//
// Both halves are load-bearing.
//
// The IMPORT half is what separates this supply from §3.2.7's xsi: seeding
// (seedInstanceAttributes, which is unconditional). src-resolve (§3.17.6.2)
// clause 4.2 hardcodes exactly two namespaces a reference may name with no
// <import> at all — clause 4.2.3 the XSD namespace, clause 4.2.4 the XSI one —
// and the XML namespace is pointedly absent from that pair, so a QName like
// xml:base is licensed only through clause 4.2.2, by an <import> in its own
// document. These four are not "present in every schema by definition"; they
// live in a well-known schema document a schema has to ask for, which is what
// the s4s itself does (xmlschema11-1.md:4403).
//
// The COMPOSED half is what keeps the supply a SUBSTITUTE for that document
// rather than an addition to it. §4.2.6.1 licenses the substitute — "the
// processor is free to access or construct components using means of its own
// choosing, whether or not a schemaLocation hint is provided"
// (xmlschema11-1.md:4199) — but a document actually composed under the
// namespace has already contributed its own, user-authored declarations, and
// adding these on top is two components of one kind sharing an expanded name,
// charged sch-props-correct (§3.17.6.1) clause 2 at finalize. The W3C suite
// holds exactly that shape twice: saxonData/Override/over030b.xsd and
// msData/additional/test264908_1a.xsd each declare the namespace's attributes
// themselves and are reached by a RESOLVABLE relative schemaLocation.
//
// target is the EFFECTIVE target namespace, so a document coerced into the XML
// namespace by chameleon inclusion (§F.1 task a) counts as composed under it
// without this predicate re-deriving the coercion.
func supplyXMLNamespace(docs []composedDocument) bool {
	imported := false
	for _, d := range docs {
		if d.target == xmltree.XMLNamespaceURI {
			return false
		}
		if importsNamespace(d.schema, xmltree.XMLNamespaceURI) {
			imported = true
		}
	}
	return imported
}

// addXMLNamespace hands builder the declarations xmlNamespaceAttributes builds
// when supplyXMLNamespace says this assembly needs them, and does nothing
// otherwise.
//
// It runs ONCE per assembly, beside newSymbols' unconditional seeding and for
// the same duplication reason: the declarations enter the very {attribute
// declarations} set a document's own top-level <attribute> declarations reach,
// so a second helping would collide with the first under sch-props-correct
// clause 2. It runs AFTER the whole <include>/<import> closure is discovered,
// because supplyXMLNamespace's composed half is a fact about the finished
// document set and not about any one <import> element.
//
// GAP(parser): the xml:specialAttrs attribute group definition — the component
// the s4s itself imports the namespace for (xmlschema11-1.md:4406) — is NOT
// supplied, so <attributeGroup ref="xml:specialAttrs"/> is still charged
// src-resolve clause 1.4. Supplying it as a COMPONENT would not help: a group
// reference is resolved by splicing the referenced top-level <attributeGroup>
// ELEMENT's own uses and wildcards into the referring type
// (spliceAttributeGroup, §3.6.2.2), which reads symbols.attributeGroups and
// never {attribute group definitions}, so a seeded component is unreachable by
// reference and teaching that path to accept one would mint a second resolution
// mechanism beside it (STYLE T4). No suite schema case turns on it: the one
// fixture naming the group, msData/additional/test264908_1a.xsd, declares the
// group itself.
func addXMLNamespace(builder *xsd.SchemaBuilder, docs []composedDocument) error {
	if !supplyXMLNamespace(docs) {
		return nil
	}
	decls, err := xmlNamespaceAttributes()
	if err != nil {
		return err
	}
	for _, d := range decls {
		builder.AddAttribute(d)
	}
	return nil
}

// xmlNamespaceAttributes builds the four Attribute Declarations the schema
// document for the XML namespace declares — xml:lang, xml:space, xml:base and
// xml:id, returned in that order, which is the order they enter {attribute
// declarations}. Each carries {target namespace} the XML namespace, {scope}
// global with {parent} ·absent·, {value constraint} ·absent· and {inheritable}
// false; they differ only in {type definition}.
//
// THE REVISION SERVED IS THE 2009 ONE, the revision of
// http://www.w3.org/2001/xml.xsd whose xml:lang is a UNION of xs:language with
// an anonymous empty-string type rather than the plain xs:language of the 2001
// original. §1.3.2 constrains what may be served — the components "should agree
// with the descriptions given in the relevant specifications and with the
// declarations given in any applicable XSD schema documents maintained by the
// World Wide Web Consortium for these namespaces" — without naming a revision,
// and the local spec corpus picks this one: Datatypes §3.4.3's note on language
// quotes that union declaration verbatim off that very URI
// (xmlschema11-2.md:1742). The 2001 original is not a candidate anyway — it
// predates xml:id and declares three attributes, not four.
//
// Two consequences of the revision, both observable:
//
//   - {value constraint} is ·absent· on xml:space. The default="preserve" the
//     2001 original wrote on it is not part of what is served.
//   - xml:space's {type definition} is an anonymous restriction of xs:NCName
//     and xml:lang's an anonymous union, so neither is reachable as a by-name
//     xsd.TypeDefinitionRef; xml:base (xs:anyURI) and xml:id (xs:ID) are.
//
// The table is HAND-BUILT rather than generated, which PRINCIPLES 26 forbids
// for a spec data table, because there is no local input a generator could
// read: no copy of xml.xsd exists under docs/specs/md — the corpus quotes the
// one xml:lang declaration cited above and nothing else of it — nor under
// testdata/xsdtests, whose only documents for the namespace are fixtures a test
// author wrote. seedInstanceAttributes is the precedent for the same situation,
// building §3.2.7's four from a section's property tables through these same
// constructors. Giving generation a source means bringing xml.xsd itself into
// the repository first.
//
// Every {type definition} that CAN be a by-name reference is one, on
// seedInstanceAttributes' reasoning: finalize's src-resolve ladder already
// walks the schema's {type definitions} and resolves exactly this slot, so
// threading pointers in from symbols.builtins would mint a second resolution
// mechanism beside it (STYLE T4). The same holds for the anonymous types' own
// {base type definition} and {member type definitions}.
func xmlNamespaceAttributes() ([]xsd.AttributeDeclaration, error) {
	// xml:lang's anonymous union member: the type whose only value is the empty
	// string, which is what lets xml:lang="" un-declare an inherited language
	// (xmlschema11-2.md:1742's own note — "the empty string is not a member of
	// the ·value space· of language").
	//
	// Its {name} is the zero QName, absence's encoding here, and the type is
	// inline so no by-name index holds it.
	emptyLang, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{},
		xsd.RestrictionDerivation{}, xsd.SimpleTypeRef{Name: stringName},
		[]xsd.Facet{xsd.NewEnumerationFacet([]xsd.EnumerationMember{
			xsd.NewEnumerationMember("", nil, nil),
		})}, nil)
	if err != nil {
		return nil, err
	}
	// {base type definition} xs:anySimpleType and {facets} the EMPTY set, which
	// §3.16.2.1 map.std.common gives every <union> alternative (case 2 and case
	// 4): cos-st-restricts clause 3.2.1.2 admits nothing else for a union
	// constructed directly on xs:anySimpleType, so a facet here is refused at
	// finalize by xsd's checkUnionGraph. constructUnionType builds a document's
	// own <union> the same way.
	langType, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{},
		xsd.UnionDerivation{Members: []xsd.SimpleTypeOrRef{
			xsd.SimpleTypeRef{Name: languageName},
			xsd.OwnedSimpleType{Definition: emptyLang},
		}},
		xsd.OwnedSimpleType{Definition: xsd.AnySimpleType()}, nil, nil)
	if err != nil {
		return nil, err
	}
	// xml:space is an anonymous restriction of xs:NCName enumerating the two
	// values XML §2.10 gives the attribute and no others: "This specification
	// does not give meaning to any value of xml:space other than 'default' and
	// 'preserve'" (xml.md:576).
	spaceType, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{},
		xsd.RestrictionDerivation{}, xsd.SimpleTypeRef{Name: ncNameName},
		[]xsd.Facet{xsd.NewEnumerationFacet([]xsd.EnumerationMember{
			xsd.NewEnumerationMember("default", nil, nil),
			xsd.NewEnumerationMember("preserve", nil, nil),
		})}, nil)
	if err != nil {
		return nil, err
	}

	seeds := []struct {
		local    string
		typeSlot xsd.TypeDefinitionOrRef
	}{
		{"lang", xsd.InlineTypeDefinition{Definition: langType}},
		{"space", xsd.InlineTypeDefinition{Definition: spaceType}},
		{"base", xsd.TypeDefinitionRef{Name: anyURIName}},
		{"id", xsd.TypeDefinitionRef{Name: idName}},
	}
	decls := make([]xsd.AttributeDeclaration, 0, len(seeds))
	for _, s := range seeds {
		d, err := xsd.NewAttributeDeclaration(xsderr.Loc{},
			xsd.QName{Space: xmltree.XMLNamespaceURI, Local: s.local}, s.typeSlot,
			xsd.NewAttributeGlobalScope(), nil, false, nil)
		if err != nil {
			return nil, err
		}
		decls = append(decls, d)
	}
	return decls, nil
}
