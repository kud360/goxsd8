package codec

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// A DeclineError is a construct [Decode] does not decode or [Append] does not
// encode; it wraps errors.ErrUnsupported, and the docs of Decode,
// [CheckDecode], Append and [CheckAppend] state when each returns one.
type DeclineError struct {
	construct construct
	// path is the path that declined, Decode's or Append's.
	path path
	// loc is the instance Loc the decline is charged to, nil for none.
	loc *xsderr.Loc
	// detail is the text Error's doc states after the construct.
	detail string
}

// Construct returns the construct the decline declines.
func (e *DeclineError) Construct() Construct { return Construct{e.construct} }

// Loc returns the instance Loc of the element the decline is charged to: the
// document element for a construct [Decode]'s item 5 finds, of its declaration or type
// or of a declaration or type its particles reach at any depth, and the element carrying
// it for xsi:type, and true. For a decline [CheckDecode], [Append] or [CheckAppend]
// returns, which reads no instance, it returns the zero Loc and false.
func (e *DeclineError) Loc() (xsderr.Loc, bool) {
	if e.loc == nil {
		return xsderr.Loc{}, false
	}
	return *e.loc, true
}

// Error opens with "codec does not decode" for a decline of [Decode] or
// [CheckDecode] and "codec does not encode" for one of [Append] or
// [CheckAppend]. For the first it names the instance Loc, where Loc reports
// one, and the construct, then the declining declaration, the document
// element's, a particle's element declaration's, local or a top-level one an
// <element ref> names, or a member of that one's substitution group, at any
// depth, or an attribute's, and, for a construct of a complex type, the
// document element's or one a particle's element declaration has at any depth,
// or of the document element's simple type, that type labelled as its
// declaration's {type definition}, with no path from the document element, and,
// for a construct of a complex type's simple content, its {content type}'s
// {simple type definition}, and of a list type's {item type definition} or a
// union's basic member, that item or member, by its QName where it has one,
// each with its schema Loc; for a wildcard particle, it names the particle as
// [Decode]'s item 5 states; for xsi:type, it names the element and the type the
// xsi:type names. For the second it names the construct, then the declining
// element declaration, attribute use or simple type, by its QName where it has
// one, with its schema Loc, and, for a construct of a complex type's simple
// content, that type labelled as its declaration's {type definition}, then its
// {content type}'s {simple type definition}, by its QName where it has one,
// with its schema Loc, and, for a construct of a list type's {item type
// definition}, a union's basic member or a union among a union's {member type
// definitions} at any depth, that item, member or member union, by its QName
// where it has one, with its schema Loc, and, for an identity constraint whose
// paths icpath does not compile, that constraint and the part, for a keyref's
// {referenced key} that key and its part, and, for a value with no canonical
// representation or no spelling Append writes, the Go path of its field from T
// and its value, and, for an attribute in the XSI namespace an element held in
// a wildcard's match carries, or the content of an element of type xs:anyType
// itself holds, that attribute, its Go path from T and the wildcard particle,
// as [Decode]'s item 5 names it, or that element's declaration, and, for an
// element such content holds whose expanded name a top-level element
// declaration names, or an attribute it holds whose expanded name a top-level
// attribute declaration names, that item, its Go path from T, that element's
// declaration and the top-level declaration, with its schema Loc. A type
// definition named in the XML Schema namespace whose schema Loc is the zero Loc
// is named by its QName alone, in place of the schema Loc the sentences above
// give it.
func (e *DeclineError) Error() string {
	text := "codec does not decode " + e.construct.name() + ": " + e.detail +
		"; a decoder refusal, not a schema or instance fault"
	if e.path == pathEncode {
		text = "codec does not encode " + e.construct.name() + ": " + e.detail +
			"; an encoder refusal, not a schema or value fault"
	}
	if e.loc == nil {
		return text
	}
	return e.loc.String() + ": " + text
}

// Unwrap returns errors.ErrUnsupported.
func (e *DeclineError) Unwrap() error { return errors.ErrUnsupported }

// A Construct is a construct Decode or [Append] declines: two declines of the same
// construct have equal Constructs; the zero Construct names none. A Construct
// is comparable: == reports whether two name the same construct, and it may be
// used as a map key. The package exports no Construct value, because a
// release that lifts a construct stops returning its Construct; compare a
// Construct only with another obtained from [DeclineError.Construct]. The
// package doc's "What decodes" and "What encodes" sections state what declines.
type Construct struct{ c construct }

// String returns the construct's name for display, distinct for each
// Construct; it is not the decline's message. The names are not stable across
// releases: a release may rename a construct, split one into several, or stop
// returning a name once Decode or Append covers its construct, and no name is reserved.
// To tell constructs apart, use the Constructs themselves, as [Construct]
// states, never a name.
func (c Construct) String() string { return c.c.name() }

// path is the closed set of the paths a DeclineError declines on: Decode's,
// [CheckDecode]'s among it, and Append's, [CheckAppend]'s among it. A
// construct does not imply its path, since both paths decline
// constructNillable and constructNestedGroup. The zero value is invalid
// (STYLE T1).
type path uint8

// The paths.
const (
	pathDecode path = iota + 1
	pathEncode
)

// construct is the closed set of constructs Decode or Append declines, one per
// declining site; one may be declined by both. The zero value is invalid
// (STYLE T1).
type construct uint8

// The construct values, in the order Decode finds them: the document element's
// declaration, its type's content, each particle in order, its attribute uses in order,
// and the instance. A particle's complex type, examined at the particle, declines by the
// same constructs from constructNestedGroup on. The one of a simple type,
// constructOtherPrimitive, sits with the particle's, though a document element of simple
// type, a complex type's simple content, by its {simple type definition}, an attribute
// use, a list type, by its {item type definition}, and a union, by a basic member,
// decline by it too. constructUntrackedModel follows every particle of a flattened
// content model or an occurrence field's, ahead of the type's attribute uses, and every
// basic particle a nested flattened particle reaches and every particle of a nested
// occurrence field's item, ahead of the particles after it. The constructs after
// constructXsiType are Append's alone, in the order Append finds them: the document
// element's declaration, after constructNillable, its type's {abstract}, its type's
// content, its content {particle}, each particle in order, its attribute uses in order,
// its {attribute wildcard}, its {assertions}, and each value. Append declines by
// constructUntrackedModel too, once every particle of a content model holding a wildcard
// particle, or every basic particle a flattened content {particle} reaches, is examined.
// A particle declines by constructVacuousParticle first, as the content {particle} does,
// then by constructNestedGroup where its term is a model group; its element declaration,
// local or the top-level one an <element ref> names, or, for an <element ref> whose field
// codegen gives a sealed interface (goshape.SubstitutionSum), each declaration that
// interface has a branch for, in group order, declines by the document element's
// declaration's constructs, constructNillable first and constructAbstractElement last,
// then by its type's as the document element's type does, from constructAbstractType on,
// a complex type's content and attribute uses examined at the particle; and an attribute
// use declines by constructValueConstraint after constructQualifiedAttribute. The ones of
// a simple type, constructSpecialType to constructAssertions, sit after the
// {assertions}'s, though the document element's, a particle's and an attribute use's
// simple type declines by them where that declaration or use is examined, and a complex
// type's simple content, by its {simple type definition}, where its type's content is,
// ahead of its attribute uses.
const (
	constructNillable construct = iota + 1
	constructNestedGroup
	constructSharedBranch
	constructOtherPrimitive
	constructUntrackedModel
	constructXsiType
	constructTypeTable
	constructValueConstraint
	constructIdentityConstraint
	constructAbsentAffiliation
	constructAbstractElement
	constructAbstractType
	constructMixed
	constructOpenContent
	constructVacuousParticle
	constructAllGroup
	constructWildcard
	constructQualifiedAttribute
	constructAttributeWildcard
	constructComplexAssertions
	constructSpecialType
	constructEncodePrimitive
	constructIDType
	constructAssertions
	constructNoCanonical
	constructQNameSpelling
	constructXsiAttribute
	constructGovernedContent
)

// constructNames holds, indexed by construct, each construct's name.
var constructNames = [...]string{
	constructNillable:           "an element declaration with {nillable} true",
	constructNestedGroup:        "a nested model group particle",
	constructSharedBranch:       "two branches of a sealed interface whose structs have one shape",
	constructOtherPrimitive:     "an atomic simple type whose primitive is ·implementation-defined·, other than xs:precisionDecimal",
	constructUntrackedModel:     "a content model the content matcher does not track",
	constructXsiType:            "an element carrying xsi:type",
	constructTypeTable:          "an element declaration with a {type table}",
	constructValueConstraint:    "a value constraint, default or fixed",
	constructIdentityConstraint: "an identity constraint whose {selector} or {fields}, or whose {referenced key}'s, lie outside the path subset icpath compiles (Structures §3.11.6.2 ·selector subset·, §3.11.6.3 ·field subset·)",
	constructAbsentAffiliation:  "an element declaration with an ·absent· member of its {substitution group affiliations}",
	constructAbstractElement:    "an element declaration with {abstract} true",
	constructAbstractType:       "an element declaration whose {type definition} is a complex type with {abstract} true",
	constructMixed:              "a complex type with mixed content",
	constructOpenContent:        "an {open content}",
	constructVacuousParticle:    "a particle with {max occurs} 0",
	constructAllGroup:           "an all group",
	constructWildcard:           "a wildcard particle with {process contents} strict or lax",
	constructQualifiedAttribute: "an attribute use whose declaration has a {target namespace}",
	constructAttributeWildcard:  "an {attribute wildcard}",
	constructComplexAssertions:  "a complex type with {assertions}",
	constructSpecialType:        "xs:anySimpleType or xs:anyAtomicType as a simple type",
	constructEncodePrimitive:    "an atomic simple type whose primitive is other than " + displayList(encodedPrimitives),
	constructIDType:             "a simple type derived from xs:ID, xs:IDREF or xs:ENTITY",
	constructAssertions:         "a simple type with an assertions facet",
	constructNoCanonical:        "a value with no canonical representation",
	constructQNameSpelling:      "an xs:QName or xs:NOTATION value with no spelling Append writes where it is written",
	constructXsiAttribute:       "an attribute in the XSI namespace held in a wildcard's match or an xs:anyType element's content",
	constructGovernedContent:    "an element held in an xs:anyType element's content whose expanded name a top-level element declaration names, or an attribute it holds whose expanded name a top-level attribute declaration names",
}

// displayList spells names, at least two, each by its Display, separated by
// ", " but for " or " before the last.
func displayList(names []xsd.QName) string {
	s := names[0].Display()
	for _, n := range names[1 : len(names)-1] {
		s += ", " + n.Display()
	}
	return s + " or " + names[len(names)-1].Display()
}

// name returns the construct's name.
func (c construct) name() string {
	if c == 0 || int(c) >= len(constructNames) {
		return "construct(" + strconv.Itoa(int(c)) + ")"
	}
	return constructNames[c]
}

// decline returns the *DeclineError of Decode for construct c, charged to no
// instance Loc, detail the text Error's doc states after the construct. No
// schema or instance rule is violated, so it carries no xsderr rule (STYLE
// E2).
func decline(c construct, detail string) *DeclineError {
	return &DeclineError{construct: c, path: pathDecode, detail: detail}
}

// encodeDecline returns the *DeclineError of Append for construct c, detail
// the text Error's doc states after the construct. Append never charges one
// to an instance Loc (at), and it carries no xsderr rule, as decline's does
// not.
func encodeDecline(c construct, detail string) *DeclineError {
	return &DeclineError{construct: c, path: pathEncode, detail: detail}
}

// at returns a copy of e charged to the instance Loc loc.
func (e *DeclineError) at(loc xsderr.Loc) *DeclineError {
	charged := *e
	charged.loc = &loc
	return &charged
}

// located reports whether a message names td with its schema Loc, which
// [DeclineError.Error]'s doc states: every type definition except one named in
// the XML Schema namespace whose Loc is the zero Loc, whose location such a
// message would print as "?".
func located(td xsd.TypeDefinition) bool {
	return td.Name().Space != xsd.XMLSchemaNS || td.Loc() != (xsderr.Loc{})
}

// definedAt returns lead, then "defined at" and td's schema Loc, where
// located(td) holds, and otherwise "": every subject that renders a type's Loc
// renders it here, and one naming td by its Loc alone names it by its QName
// where located(td) does not hold (typeSubject, complexSubject).
func definedAt(lead string, td xsd.TypeDefinition) string {
	if !located(td) {
		return ""
	}
	return fmt.Sprintf("%sdefined at %s", lead, td.Loc())
}
