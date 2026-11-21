package codec

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/internal/xmlchar"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ErrInvalidValue is the sentinel errors.Is finds in item 6's error of
// [Append], whose text is its cause's alone, never this sentinel's. In the
// last of item 6's cases that error wraps the value.ValidateLexical rejection
// and carries whatever *xsderr.Error the rejection carries; in the others it
// carries none.
var ErrInvalidValue = errors.New("codec: Append: a value no valid document holds")

// Append appends to dst the XML document that encodes v as the document
// element of the top-level element declaration name in schema, and returns
// the extended slice; dst may be nil. Read name from the header comment of the
// type codegen emits for that declaration, the header the "# Naming" section
// of codegen's package doc describes. It writes each simple value in its
// canonical representation (Datatypes §2.3.1 dt-canonical-representation),
// the form the Canonical of its governing mapping on backend
// (value.GoverningMapping) renders, and runs no assessment by package validate
// of the document it writes: a caller wanting a verdict on it assesses it
// through package validate, as any document. The package doc's
// "# What encodes" section states the slice Append encodes and the document
// it writes. On error Append returns dst unextended, its bytes beyond
// len(dst) possibly overwritten, and the first of these that applies:
//
//  1. A nil schema or a nil backend: a plain error.
//  2. No top-level element declaration of schema has the expanded name name:
//     a plain error naming name.
//  3. The declaration has {abstract} true: a plain error.
//  4. A construct outside the encoded slice: its *DeclineError, charged to
//     no instance Loc ([DeclineError.Loc]).
//  5. A T that is not the type codegen emits for the declaration, item 6 of
//     [Decode]'s doc: its error, which errors.Is reports as [ErrTypeMismatch].
//  6. A value no valid document holds, the first in document order, an
//     element's attributes before its content: an error, which errors.Is
//     reports as [ErrInvalidValue], naming the Go path of its field from T
//     and, for a simple value, its simple type. It is the slice field of a
//     particle with {max occurs} above 1 whose length lies outside its {min
//     occurs} and {max occurs}; a string holding bytes that are not UTF-8 or
//     a character the XML 1.0 Char production does not match, U+0000 among
//     them (Datatypes §3.3.1.1, §3.3.17.1); a string that differs from its own
//     whiteSpace-normalized form under its simple type, such as an xs:token
//     holding "  a  b"; a list item, a string, that is empty or holds TAB,
//     LF, CR or a space, which no list item's lexical representation does
//     (Datatypes §2.4.1.2); a big.Rat outside the decimal value space, whose
//     denominator in lowest terms is no 2^m·5^n (§3.3.3.1), such as 1/3; a
//     date, time or duration datatype struct value no value of its datatype
//     maps to, as codegen's package doc "# Simple types" states such values
//     exist, the error naming the first property found outside it, its
//     fields read in order, or, ahead of its Seconds, a duration's Months
//     and Seconds of opposite signs; or a value whose simple type rejects it, a
//     value.ValidateLexical error value.IsDatatypeVerdict reports as a
//     verdict, a facet's rejection or a backend Parse's among them, that
//     error wrapped, but for item 7's pattern case, a list's own facets read
//     once every item has passed its own checks.
//  7. A value with no canonical representation, the first in document order
//     as item 6 counts: its *DeclineError. That is a governing mapping with no
//     Canonical, a Canonical that fails for the value, such as strict's for
//     the zero xs:yearMonthDuration (Datatypes §3.4.26.1's note), or a value
//     with a lexical other than its canonical representation, or a list
//     holding at least one such item, whose canonical representation a
//     pattern facet of its type rejects (§2.1's note), such as 3/2 under an
//     xs:decimal pattern \d+\.\d\d, which admits 1.50 but not 1.5, whatever
//     other facet rejects it; or a canonical representation that does not map
//     back to the value. Every xs:boolean, xs:decimal-primitive, xs:dateTime,
//     xs:dateTimeStamp, xs:time and xs:duration-primitive value has another
//     lexical; an xs:float or xs:double value has one but for NaN and -INF
//     (§3.3.4.2, §3.3.5.2); an xs:date, xs:gYearMonth or xs:gYear value only
//     where its timezoneOffset is 0, spelled Z, +00:00 or -00:00 (§D.2.2's
//     note), or its year is 0, spelled 0000 or -0000; an xs:gMonthDay, xs:gDay
//     or xs:gMonth value only where its timezoneOffset is 0; an xs:hexBinary
//     value only where one of its octets holds a hex digit A to F
//     (§3.3.15.2); an xs:base64Binary value but for the empty one
//     (§3.3.16.2); and an xs:string- or xs:anyURI-primitive value never, being
//     its own whiteSpace-normalized form.
//
// A plain error is no *DeclineError and wraps none of [ErrTypeMismatch],
// [ErrInvalidValue] and errors.ErrUnsupported; whether it wraps an
// *xsderr.Error does not tell its item.
//
// A fault of backend found at a value, in place of that value's item 6 or 7, is
// a plain error too, returned at once and naming the value's Go path and simple
// type: no mapping of the value's primitive, no governing mapping, a Parse that
// rejects the spelling Append gives the value, the mapping's of the type's
// {primitive type definition} for a value of none of xs:string, xs:anyURI,
// xs:float and xs:double primitive and the governing mapping's for an xs:float
// or xs:double one, a Parse of the governing mapping that rejects the
// whiteSpace-normalized canonical representation an xs:float or xs:double
// value's Canonical renders, a Canonical that renders a value of none of
// xs:string, xs:anyURI, xs:float and xs:double primitive as other text than
// its canonical representation (Datatypes §E.1 f-decimalCanmap, §E.4
// f-booleanCanmap, §E.2 f-durationCanMap, §E.3.5 vp-dateTimeCanRep,
// vp-timeCanRep, vp-dateCanRep, vp-gYearMonthCanRep, vp-gYearCanRep,
// vp-gMonthDayCanRep, vp-gDayCanRep and vp-gMonthCanRep, §E.4.1
// f-hexBinaryCanonical and §3.3.16.2's Canonical-base64Binary), or an error
// of value.ValidateLexical that value.IsDatatypeVerdict does not report as a
// verdict.
//
// Items 2 to 5 read no value; [CheckAppend] returns them for a declaration
// and T.
func Append[T any](dst []byte, schema *xsd.Schema, backend value.Backend, name xsd.QName, v T) ([]byte, error) {
	if schema == nil {
		return dst, errors.New("codec: Append: nil *xsd.Schema")
	}
	if backend == nil {
		return dst, errors.New("codec: Append: nil value.Backend")
	}
	t := reflect.TypeFor[T]()
	plan, err := encodeStatic(schema, name, t)
	if err != nil {
		return dst, err
	}
	root := t.Name()
	if root == "" {
		root = t.String()
	}
	e := &encoder{schema: schema, backend: backend, out: dst}
	if err := e.element(plan, reflect.ValueOf(v), root, xsd.QName{}, true); err != nil {
		return dst, err
	}
	if e.invalid != nil {
		return dst, e.invalid
	}
	if e.uncanonical != nil {
		return dst, e.uncanonical
	}
	return e.out, nil
}

// CheckAppend returns the error of the first of items 2 to 5 of [Append]'s
// doc that applies to a value of T encoded as the document element of the
// top-level element declaration name in schema, in the class that item
// states, or nil when none applies; for a nil schema it returns a plain
// error. Item 4's slice is the one the package doc's "# What encodes" states.
// Items 6 and 7 and a fault of the backend depend on the value, so they lie
// outside CheckAppend's answer, and nil leaves them open. The backend is no
// input, so a nil backend, which Append's item 1 refuses, lies outside it
// too. For a non-nil schema, a non-nil error is the one Append returns, of
// equal Error text and class, for every value of T and every non-nil backend.
func CheckAppend[T any](schema *xsd.Schema, name xsd.QName) error {
	if schema == nil {
		return errors.New("codec: CheckAppend: nil *xsd.Schema")
	}
	_, err := encodeStatic(schema, name, reflect.TypeFor[T]())
	return err
}

// encodeStatic returns the encodeElement of the top-level element
// declaration name in schema, a non-nil schema, encoded from a value of Go
// type t, or the error of the first of items 2 to 5 of [Append]'s doc that
// applies. It reads no value, so its answer holds for every value of t.
func encodeStatic(schema *xsd.Schema, name xsd.QName, t reflect.Type) (encodeElement, error) {
	decl, ok := schema.Element(name)
	if !ok {
		return encodeElement{}, fmt.Errorf("codec: no top-level element declaration has the expanded name %q", name.Display())
	}
	// An answer about the schema, no violation of an instance, so it carries
	// no xsderr rule (STYLE E2).
	if decl.Abstract() {
		return encodeElement{}, fmt.Errorf("codec: %s has {abstract} true, so no document element validates against it (cvc-elt clause 2)", declSubject(decl))
	}
	plan, err := encodePlanOf(schema, decl)
	if err != nil {
		return encodeElement{}, err
	}
	b, err := bindDecl(&pipeline{schema: schema}, decl)
	if err != nil {
		return encodeElement{}, err
	}
	if _, err := b.check(t, newCheckContext(newConfig(nil))); err != nil {
		return encodeElement{}, err
	}
	return plan, nil
}

// encodeElement is an element declaration of the encoded slice, the
// document element's or a particle's, as Append writes it: of a simple
// {type definition}, simple; of a complex one, its particles, each a local
// element declaration, and its attribute uses, in the order codegen gives
// their fields (codegen's package doc, "# Particles" and "# Attributes").
type encodeElement struct {
	decl       xsd.ElementDeclaration
	simple     *encodeSimple
	particles  []encodeParticle
	attributes []encodeAttribute
}

// encodeParticle is an element particle of a spliced content sequence: its
// occurrence and its local element declaration.
type encodeParticle struct {
	occurs  xsd.Occurs
	element encodeElement
}

// encodeAttribute is an unqualified attribute use with no value constraint:
// its attribute declaration and that declaration's simple type.
type encodeAttribute struct {
	decl   xsd.AttributeDeclaration
	simple encodeSimple
}

// encodeSimple is a simple type of the encoded slice: st itself and, where st
// is a list, item, its {item type definition}, nil where st is atomic, and
// atom, the atomic Go type codegen gives st where it is atomic and item where
// st is a list (goshape.AtomicType), which the encoder reads a value by.
// simpleEncodes is its one constructor.
type encodeSimple struct {
	st   *xsd.SimpleType
	item *xsd.SimpleType
	atom atomic
}

// encodePlanOf returns the encodeElement of decl, the document element's
// declaration, or the error item 4 of [Append]'s doc states for the first
// construct outside the slice the package doc's "# What encodes" states, in
// the order the construct constants list, or a plain error for a component
// that does not resolve.
func encodePlanOf(schema *xsd.Schema, decl xsd.ElementDeclaration) (encodeElement, error) {
	if err := declEncodes(decl); err != nil {
		return encodeElement{}, err
	}
	td, ok := schema.ResolvedType(decl.TypeDefinition())
	if !ok {
		return encodeElement{}, fmt.Errorf("codec: the {type definition} of %s does not resolve", declSubject(decl))
	}
	if goshape.AnyTypeItself(td) {
		return encodeElement{}, encodeDecline(constructAnyType, declSubject(decl))
	}
	if st, isSimple := td.(*xsd.SimpleType); isSimple {
		simple, err := simpleEncodes(schema, st, declSubject(decl))
		if err != nil {
			return encodeElement{}, err
		}
		return encodeElement{decl: decl, simple: &simple}, nil
	}
	ct, _ := td.(xsd.ComplexType)
	if ct.Abstract() {
		return encodeElement{}, encodeDecline(constructAbstractType, typeSubject(decl, ct))
	}
	return complexPlanOf(schema, decl, ct)
}

// declEncodes returns the decline of the first of decl's own properties the
// slice excludes: {nillable} as codegen reads it (goshape.Nillable), a {type
// table}, a {value constraint} and {identity-constraint definitions}.
func declEncodes(decl xsd.ElementDeclaration) error {
	if goshape.Nillable(decl) {
		return encodeDecline(constructNillable, declSubject(decl))
	}
	if _, has := decl.TypeTable(); has {
		return encodeDecline(constructTypeTable, declSubject(decl))
	}
	if _, has := decl.ValueConstraint(); has {
		return encodeDecline(constructValueConstraint, declSubject(decl))
	}
	if len(decl.IdentityConstraints()) > 0 {
		return encodeDecline(constructIdentityConstraint, declSubject(decl))
	}
	return nil
}

// complexPlanOf returns the encodeElement of decl, whose {type definition}
// is the complex type ct, or the error encodePlanOf states: ct's content,
// then its content {particle}, {max occurs} 0 declined before
// goshape.ParticleHold is asked (#3794), then each particle in order, then
// each attribute use codegen gives a field (goshape.FieldAttributeUses), then
// its {attribute wildcard}, then its {assertions}.
func complexPlanOf(schema *xsd.Schema, decl xsd.ElementDeclaration, ct xsd.ComplexType) (encodeElement, error) {
	subject := typeSubject(decl, ct)
	plan := encodeElement{decl: decl}
	// {content type} is a closed sum, so the switch is STYLE T2's sealed-sum
	// exception.
	switch content := ct.ContentType().(type) {
	case xsd.SimpleContent:
		return encodeElement{}, encodeDecline(constructSimpleContent, subject)
	case xsd.ElementContent:
		particles, err := contentPlanOf(schema, content, subject)
		if err != nil {
			return encodeElement{}, err
		}
		plan.particles = particles
	}
	for _, u := range goshape.FieldAttributeUses(ct) {
		a, err := attributePlanOf(schema, u)
		if err != nil {
			return encodeElement{}, err
		}
		plan.attributes = append(plan.attributes, a)
	}
	if _, has := ct.AttributeWildcard(); has {
		return encodeElement{}, encodeDecline(constructAttributeWildcard, subject)
	}
	if len(ct.Assertions()) > 0 {
		return encodeElement{}, encodeDecline(constructComplexAssertions, subject)
	}
	return plan, nil
}

// contentPlanOf returns the encodeParticles of content, element-only or
// mixed content of the complex type subject names, or complexPlanOf's error.
func contentPlanOf(schema *xsd.Schema, content xsd.ElementContent, subject string) ([]encodeParticle, error) {
	if content.Mixed {
		return nil, encodeDecline(constructMixed, subject)
	}
	if content.OpenContent != nil {
		return nil, encodeDecline(constructOpenContent, subject)
	}
	occurs := content.Particle.Occurs()
	if upper, bounded := occurs.Max(); bounded && upper == 0 {
		return nil, encodeDecline(constructVacuousParticle, "the content {particle} of "+subject)
	}
	group, isGroup, err := goshape.ModelGroupTerm(schema, content.Particle.Term())
	if err != nil {
		return nil, fmt.Errorf("codec: %s: %w", subject, err)
	}
	if !isGroup {
		return nil, fmt.Errorf("codec: the content {particle} of %s holds no model group", subject)
	}
	hold, err := goshape.ParticleHold(schema, group, occurs)
	if err != nil {
		return nil, fmt.Errorf("codec: %s: %w", subject, err)
	}
	switch hold {
	case goshape.Flattened, goshape.OnceChoice:
		return nil, encodeDecline(constructSealedInterface, subject)
	case goshape.Occurrence:
		return nil, encodeDecline(constructOccurrenceField, subject)
	case goshape.Spliced:
		if group.Compositor() == xsd.CompositorAll {
			return nil, encodeDecline(constructAllGroup, subject)
		}
	}
	var particles []encodeParticle
	for i, p := range group.Particles() {
		el, err := particlePlanOf(schema, p, fmt.Sprintf("particle %d of the content {particle} of %s", i+1, subject))
		if err != nil {
			return nil, err
		}
		particles = append(particles, encodeParticle{occurs: p.Occurs(), element: el})
	}
	return particles, nil
}

// particlePlanOf returns the encodeElement of the particle p of a spliced
// content sequence, subject naming p, or the error encodePlanOf states:
// {max occurs} 0, then a term other than a local element declaration, then
// the declaration's own properties and type, of which a complex one other
// than xs:anyType itself declines whole. {term} is a closed sum, so the
// switches are STYLE T2's sealed-sum exception.
func particlePlanOf(schema *xsd.Schema, p xsd.Particle, subject string) (encodeElement, error) {
	if upper, bounded := p.Occurs().Max(); bounded && upper == 0 {
		return encodeElement{}, encodeDecline(constructVacuousParticle, subject)
	}
	var el xsd.ElementDeclaration
	switch term := p.Term().(type) {
	case xsd.ModelGroupRef:
		return encodeElement{}, encodeDecline(constructNestedGroup, subject)
	case xsd.ElementDeclarationRef:
		return encodeElement{}, encodeDecline(constructElementRef, fmt.Sprintf("%s, naming element %s", subject, term.Name.Display()))
	case xsd.ResolvedTerm:
		switch term := term.Term.(type) {
		case xsd.ModelGroup:
			return encodeElement{}, encodeDecline(constructNestedGroup, subject)
		case xsd.Wildcard:
			return encodeElement{}, encodeDecline(constructWildcard, subject)
		case xsd.ElementDeclaration:
			el = term
		}
	}
	if err := declEncodes(el); err != nil {
		return encodeElement{}, err
	}
	td, ok := schema.ResolvedType(el.TypeDefinition())
	if !ok {
		return encodeElement{}, fmt.Errorf("codec: the {type definition} of %s does not resolve", declSubject(el))
	}
	if goshape.AnyTypeItself(td) {
		return encodeElement{}, encodeDecline(constructAnyType, declSubject(el))
	}
	st, isSimple := td.(*xsd.SimpleType)
	if !isSimple {
		return encodeElement{}, encodeDecline(constructComplexParticle, declSubject(el))
	}
	simple, err := simpleEncodes(schema, st, declSubject(el))
	if err != nil {
		return encodeElement{}, err
	}
	return encodeElement{decl: el, simple: &simple}, nil
}

// attributePlanOf returns the encodeAttribute of the attribute use u, or the
// error encodePlanOf states: a qualified declaration, whose name the default
// namespace binding Append writes cannot spell (Namespaces in XML §6.2), then
// an effective value constraint ([xsd.Schema.EffectiveValueConstraint]), then
// its simple type.
func attributePlanOf(schema *xsd.Schema, u xsd.AttributeUse) (encodeAttribute, error) {
	decl, ok := schema.ResolvedAttributeDeclaration(u)
	if !ok {
		return encodeAttribute{}, fmt.Errorf("codec: the {attribute declaration} of attribute use %s does not resolve", u.DeclarationName().Display())
	}
	subject := fmt.Sprintf("attribute %s declared at %s", decl.Name().Display(), decl.Loc())
	if decl.Name().Space != "" {
		return encodeAttribute{}, encodeDecline(constructQualifiedAttribute, subject)
	}
	if _, has := schema.EffectiveValueConstraint(u); has {
		return encodeAttribute{}, encodeDecline(constructValueConstraint, subject)
	}
	st, ok := schema.ResolvedSimpleType(decl.TypeDefinition())
	if !ok {
		return encodeAttribute{}, fmt.Errorf("codec: the {type definition} of %s does not resolve", subject)
	}
	simple, err := simpleEncodes(schema, st, subject)
	if err != nil {
		return encodeAttribute{}, err
	}
	return encodeAttribute{decl: decl, simple: simple}, nil
}

// The expanded names simpleEncodes reads a simple type's primitive and base
// chain by, with booleanName: encodedPrimitives, the one encoding of the
// primitives an atomic type that encodes has (atomicEncodes), in the order of
// Datatypes §3.3, which constructEncodePrimitive's name lists; and idNames.
var (
	stringName        = xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}
	doubleName        = xsd.QName{Space: xsd.XMLSchemaNS, Local: "double"}
	decimalName       = xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"}
	encodedPrimitives = []xsd.QName{
		stringName,
		booleanName,
		decimalName,
		{Space: xsd.XMLSchemaNS, Local: "float"},
		doubleName,
		{Space: xsd.XMLSchemaNS, Local: "duration"},
		{Space: xsd.XMLSchemaNS, Local: "dateTime"},
		{Space: xsd.XMLSchemaNS, Local: "time"},
		{Space: xsd.XMLSchemaNS, Local: "date"},
		{Space: xsd.XMLSchemaNS, Local: "gYearMonth"},
		{Space: xsd.XMLSchemaNS, Local: "gYear"},
		{Space: xsd.XMLSchemaNS, Local: "gMonthDay"},
		{Space: xsd.XMLSchemaNS, Local: "gDay"},
		{Space: xsd.XMLSchemaNS, Local: "gMonth"},
		{Space: xsd.XMLSchemaNS, Local: "hexBinary"},
		{Space: xsd.XMLSchemaNS, Local: "base64Binary"},
		{Space: xsd.XMLSchemaNS, Local: "anyURI"},
	}
	idNames = []xsd.QName{
		{Space: xsd.XMLSchemaNS, Local: "ID"},
		{Space: xsd.XMLSchemaNS, Local: "IDREF"},
		{Space: xsd.XMLSchemaNS, Local: "ENTITY"},
	}
)

// simpleEncodes returns the encodeSimple of the simple type st, the type of
// the declaration owner names, where st encodes as the package doc's "# What
// encodes" states, and otherwise the decline of the first construct it
// finds: a special type, then, for a list, the first construct of its {item
// type definition} (atomicEncodes), the detail naming the item after st
// (itemSubject), and an assertions facet among st's own effective facets,
// its own or inherited; for an atomic or union st, atomicEncodes's. An error
// reading st is returned as is, and a list with no {item type definition} is
// a plain error.
func simpleEncodes(r xsd.TypeResolver, st *xsd.SimpleType, owner string) (encodeSimple, error) {
	subject := simpleSubject(st, owner)
	if st.IsSpecial() {
		return encodeSimple{}, encodeDecline(constructSpecialType, subject)
	}
	variety, err := st.Variety(r)
	if err != nil {
		return encodeSimple{}, err
	}
	if _, isList := variety.(xsd.List); !isList {
		atom, err := atomicEncodes(r, st, subject)
		if err != nil {
			return encodeSimple{}, err
		}
		return encodeSimple{st: st, atom: atom}, nil
	}
	item, err := st.Item(r)
	if err != nil {
		return encodeSimple{}, err
	}
	if item == nil {
		return encodeSimple{}, fmt.Errorf("codec: the list type %s has no {item type definition}", subject)
	}
	atom, err := atomicEncodes(r, item, itemSubject(subject, item))
	if err != nil {
		return encodeSimple{}, err
	}
	if err := assertionsDecline(r, st, subject); err != nil {
		return encodeSimple{}, err
	}
	return encodeSimple{st: st, item: item, atom: atom}, nil
}

// atomicEncodes returns the atomic Go type codegen gives st
// (goshape.AtomicType) where st, an atomic or union simple type subject
// names, a list's {item type definition} among them, which cos-st-restricts
// clause 2.1 holds to one of those two, encodes, and otherwise the decline
// of the first construct it finds: a special type, a union, a primitive
// not in encodedPrimitives, a type on whose {base type definition} chain
// xs:ID, xs:IDREF or xs:ENTITY lies, then an assertions facet among its
// effective facets, its own or inherited. An error reading st is returned as
// is.
func atomicEncodes(r xsd.TypeResolver, st *xsd.SimpleType, subject string) (atomic, error) {
	if st.IsSpecial() {
		return 0, encodeDecline(constructSpecialType, subject)
	}
	variety, err := st.Variety(r)
	if err != nil {
		return 0, err
	}
	if _, isUnion := variety.(xsd.Union); isUnion {
		return 0, encodeDecline(constructUnion, subject)
	}
	primitive, err := st.Primitive(r)
	if err != nil {
		return 0, err
	}
	if primitive == nil || !slices.Contains(encodedPrimitives, primitive.Name()) {
		return 0, encodeDecline(constructEncodePrimitive, subject)
	}
	for step := st; step != nil; {
		for _, id := range idNames {
			if step.Name() == id {
				return 0, encodeDecline(constructIDType, fmt.Sprintf("%s, derived from %s", subject, id.Display()))
			}
		}
		step, err = step.Base(r)
		if err != nil {
			return 0, err
		}
	}
	if err := assertionsDecline(r, st, subject); err != nil {
		return 0, err
	}
	a, ok, err := goshape.AtomicType(r, st)
	if err != nil {
		return 0, err
	}
	if !ok {
		// goshape.AtomicType gives each primitive of encodedPrimitives an
		// atomic Go type, so no schema reaches this.
		return 0, fmt.Errorf("codec: codegen gives %s no atomic Go type", subject)
	}
	return atomic(a), nil
}

// assertionsDecline returns constructAssertions's decline, subject naming
// st, where an assertions facet is among st's effective facets, its own or
// inherited, an error reading them as is, and otherwise nil.
func assertionsDecline(r xsd.TypeResolver, st *xsd.SimpleType, subject string) error {
	facets, err := st.EffectiveFacets(r)
	if err != nil {
		return err
	}
	for _, f := range facets {
		if f.Facet().Kind() == xsd.FacetAssertions {
			return encodeDecline(constructAssertions, subject)
		}
	}
	return nil
}

// simpleSubject names the simple type st of the declaration owner names, by
// its QName where it has one, and its schema Loc where located(st) holds.
func simpleSubject(st *xsd.SimpleType, owner string) string {
	return "simple type " + typeLabel(st, owner) + definedAt(" ", st)
}

// encoder is one Append's write of a document: out, the document so far,
// and the first item 6 and item 7 errors of [Append]'s doc it has found, in
// document order. Every value is written whatever was found before it, so
// that the walk finds an item 6 error after an item 7 one.
type encoder struct {
	schema      *xsd.Schema
	backend     value.Backend
	out         []byte
	invalid     *invalidValueError
	uncanonical error
}

// element writes the element el, holding v, whose Go path from T is path:
// its start tag, with a default namespace declaration where its namespace
// differs from parent's, the document element's from no namespace, then its
// attributes in field order, then its value or its particles' elements, an
// element with no content written as an empty-element tag. The error is a
// backend fault (simple).
func (e *encoder) element(el encodeElement, v reflect.Value, path string, parent xsd.QName, root bool) error {
	name := el.decl.Name()
	e.out = append(e.out, '<')
	e.out = append(e.out, name.Local...)
	if name.Space != parent.Space || root && name.Space != "" {
		e.out = append(e.out, ` xmlns="`...)
		e.out = appendEscaped(e.out, name.Space, true)
		e.out = append(e.out, '"')
	}
	if el.simple != nil {
		text, err := e.simple(*el.simple, v, path, declSubject(el.decl))
		if err != nil {
			return err
		}
		start := len(e.out)
		e.out = append(e.out, '>')
		e.out = appendEscaped(e.out, text, false)
		e.close(name, start)
		return nil
	}
	first := len(el.particles)
	for i, a := range el.attributes {
		f, fpath := v.Field(first+i), path+"."+v.Type().Field(first+i).Name
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		text, err := e.simple(a.simple, f, fpath, fmt.Sprintf("attribute %s declared at %s", a.decl.Name().Display(), a.decl.Loc()))
		if err != nil {
			return err
		}
		e.out = append(e.out, ' ')
		e.out = append(e.out, a.decl.Name().Local...)
		e.out = append(e.out, `="`...)
		e.out = appendEscaped(e.out, text, true)
		e.out = append(e.out, '"')
	}
	start := len(e.out)
	e.out = append(e.out, '>')
	for i, p := range el.particles {
		if err := e.particle(p, v.Field(i), path+"."+v.Type().Field(i).Name, name); err != nil {
			return err
		}
	}
	e.close(name, start)
	return nil
}

// close ends the element name whose start tag's '>' is e.out's byte at
// start: with its end tag, or, where nothing follows that '>', by turning the
// start tag into an empty-element tag.
func (e *encoder) close(name xsd.QName, start int) {
	if len(e.out) == start+1 {
		e.out = append(e.out[:start], "/>"...)
		return
	}
	e.out = append(e.out, "</"...)
	e.out = append(e.out, name.Local...)
	e.out = append(e.out, '>')
}

// particle writes the elements of the field f of the particle p, path its Go
// path, its parent element named parent, by the shape codegen gives the field
// (goshape.ParticleShape), never by f's Kind, since a list's V is itself a
// slice: one per item of a []V, which is item 6 of [Append]'s doc where its
// length lies outside p's occurrence, none for a nil *V, and one for a V.
// The error is element's.
func (e *encoder) particle(p encodeParticle, f reflect.Value, path string, parent xsd.QName) error {
	// ok is false only for {max occurs} 0, which particlePlanOf declines
	// (constructVacuousParticle), and sealed is false since a union declines
	// (constructUnion).
	shape, _ := goshape.ParticleShape(p.occurs, false)
	switch shape {
	case goshape.Pointer:
		if f.IsNil() {
			return nil
		}
		return e.element(p.element, f.Elem(), path, parent, false)
	case goshape.Slice:
		if !p.occurs.Permits(f.Len()) {
			e.invalidate(fmt.Errorf("codec: Append: %s holds %d items, outside the occurrence %s of the particle of %s", path, f.Len(), p.occurs, declSubject(p.element.decl)))
		}
		for j := range f.Len() {
			if err := e.element(p.element, f.Index(j), path+"["+strconv.Itoa(j)+"]", parent, false); err != nil {
				return err
			}
		}
		return nil
	default:
		return e.element(p.element, f, path, parent, false)
	}
}

// invalidate records err as the document's item 6 error of [Append]'s doc,
// marked ErrInvalidValue, unless an earlier value's is recorded.
func (e *encoder) invalidate(err error) {
	if e.invalid == nil {
		e.invalid = &invalidValueError{err}
	}
}

// invalidValueError is item 6's error of [Append]'s doc: err, the cause one
// of invalidate's callers built, marked ErrInvalidValue.
type invalidValueError struct{ err error }

// Error returns the cause's text unchanged.
func (e *invalidValueError) Error() string { return e.err.Error() }

// Unwrap returns ErrInvalidValue and the cause, for errors.Is and for the
// errors.As and xsderr.RuleOf reads of the cause's chain.
func (e *invalidValueError) Unwrap() []error { return []error{ErrInvalidValue, e.err} }

// noCanonical records the item 7 decline of [Append]'s doc for the value
// shown at path, of the simple type subject names, cause saying why, unless
// an earlier value's is recorded.
func (e *encoder) noCanonical(subject, path, shown, cause string) {
	if e.uncanonical == nil {
		e.uncanonical = encodeDecline(constructNoCanonical, fmt.Sprintf("%s, at %s holding %s: %s", subject, path, shown, cause))
	}
}

// simple returns the text Append writes for v, a value of the simple type s
// held in the Go type the slice gives it, at path, owner naming s's
// declaration: an atomic value by atom, a list by list. It records the
// value's item 6 or 7 error of [Append]'s doc, and returns a backend fault.
func (e *encoder) simple(s encodeSimple, v reflect.Value, path, owner string) (string, error) {
	subject := simpleSubject(s.st, owner)
	if s.item != nil {
		return e.list(s, v, path, subject)
	}
	text, _, err := e.atom(s.st, s.atom, v, path, subject)
	return text, err
}

// atom returns the text Append writes for v, a value of the atomic simple
// type st held in a, its atomic Go type, at path, subject naming st: the
// value of an atomic formed holds for in its canonical representation
// (spelled), a string as itself (str), and a float32 or float64 in its
// canonical representation (double). ok reports that it recorded no item 6 or
// 7 error of [Append]'s doc for v.
func (e *encoder) atom(st *xsd.SimpleType, a atomic, v reflect.Value, path, subject string) (text string, ok bool, err error) {
	if a.formed() {
		return e.spelled(st, a, v, path, subject)
	}
	if a == atomicString {
		return e.str(st, v.String(), path, subject)
	}
	return e.double(st, v.Float(), a.bare().Bits(), path, subject)
}

// list returns the text the package doc's "# What encodes" states for v, a []V
// holding a value of the list type s.st, at path, subject naming s.st, each
// item written by atom under s.item at path "[i]", and records the item 6 and
// 7 errors [Append]'s doc states for a list: an item's white space is checked
// once its own checks pass, and s.st's own facets, through
// value.ValidateLexical, once every item has. An empty item is no list item
// since the list's whiteSpace collapse (Datatypes §4.3.6) leaves no empty
// token. A pattern rejection is item 7 only where an item with another
// lexical is held, which a pattern may match (§2.1's note, otherLexical); the
// empty list has no other lexical, nor a list none of whose items has one, so
// every lexical of such a value normalizes to the text written.
func (e *encoder) list(s encodeSimple, v reflect.Value, path, subject string) (string, error) {
	isubject := itemSubject(subject, s.item)
	items := make([]string, v.Len())
	clean := true
	for i := range v.Len() {
		ipath := path + "[" + strconv.Itoa(i) + "]"
		text, ok, err := e.atom(s.item, s.atom, v.Index(i), ipath, isubject)
		if err != nil {
			return "", err
		}
		if ok && (text == "" || strings.ContainsAny(text, listSeparators)) {
			e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, which is empty or holds white space, so no list item's lexical representation spells it (Datatypes §2.4.1.2)", ipath, strconv.Quote(text), isubject))
			ok = false
		}
		clean = clean && ok
		items[i] = text
	}
	text := strings.Join(items, " ")
	if !clean {
		return text, nil
	}
	shown := fmt.Sprintf("%v", v)
	if s.atom.formed() {
		// fmt spells a big.Rat, a big.Int or a datatype struct by its fields, and a
		// []byte by its octets in decimal; each item's text is its form (spelled).
		shown = "[" + text + "]"
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, s.st, text, nil, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && otherLexical(s.atom, v) {
			e.noCanonical(subject, path, shown, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", text, err))
			return text, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, written %q, which %s rejects: %w", path, shown, text, subject, err))
	}
	return text, nil
}

// listSeparators are the white space characters that separate a list's items
// (Datatypes §2.4.1.2, the S production of XML 1.0 §2.3): #x9, #xA, #xD and
// #x20, not unicode.IsSpace's set.
const listSeparators = "\t\n\r "

// str returns s, a value of the simple type st whose primitive is xs:string
// or xs:anyURI, after checking, in this order, that it is UTF-8 of XML 1.0
// Char characters only, which both value spaces are sequences of (Datatypes
// §3.3.1.1, §3.3.17.1), that value.ValidateLexical accepts it under st, and
// that it is the canonical representation of the value that returns, its own
// whiteSpace-normalized form, as the package doc's "# What encodes" states a
// string value is written. Canonical renders the value ValidateLexical
// returns, never one Parse maps s to: Parse takes a normalized lexical, and s
// may be none. ok is atom's.
func (e *encoder) str(st *xsd.SimpleType, s, path, subject string) (text string, ok bool, err error) {
	shown := strconv.Quote(s)
	if i := xmlchar.IndexNonChar(s); i >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, whose byte offset %d starts no XML 1.0 Char (Datatypes §3.3.1.1, §3.3.17.1)", path, shown, subject, i))
		return s, false, nil
	}
	v, err := value.ValidateLexical(e.backend, e.schema, st, s, nil, noAssertions{})
	if err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, shown, subject, err))
		return s, false, nil
	}
	m, err := e.mapping(st, path, subject)
	if err != nil {
		return "", false, err
	}
	if m.Canonical == nil {
		e.noCanonical(subject, path, shown, "its governing mapping has no canonical mapping")
		return s, false, nil
	}
	c, err := m.Canonical(v)
	if err != nil {
		e.noCanonical(subject, path, shown, err.Error())
		return s, false, nil
	}
	if c != s {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, which differs from its whiteSpace-normalized form %q", path, shown, subject, c))
		return s, false, nil
	}
	return s, true, nil
}

// double returns the canonical representation of f, a value of the simple
// type st whose primitive is xs:float, bits 32, or xs:double, bits 64, f
// holding a float32 exactly where bits is 32: f spelled in floatRep or
// doubleRep (doubleLexical) mapped by st's governing mapping's Parse,
// rendered by its Canonical, that rendering whiteSpace-normalized
// (value.ApplyWhiteSpace) and accepted by Parse, then held to st's facets by
// value.ValidateLexical and read back at bits (mapsBack), every NaN equal to
// every NaN. A pattern facet's rejection is item 7 of [Append]'s doc, since
// another lexical of the value may match it (Datatypes §2.1's note), but for
// NaN and -INF, which have no other (soleLexical), item 6; any other facet's
// is item 6. ok is atom's.
func (e *encoder) double(st *xsd.SimpleType, f float64, bits int, path, subject string) (text string, ok bool, err error) {
	shown := strconv.FormatFloat(f, 'g', -1, bits)
	m, err := e.mapping(st, path, subject)
	if err != nil {
		return "", false, err
	}
	lexical := doubleLexical(f, bits)
	v, err := m.Parse(lexical, nil)
	if err != nil {
		rep := "doubleRep"
		if bits == 32 {
			rep = "floatRep"
		}
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, whose %s form %q the backend's Parse rejects: %w", path, shown, subject, rep, lexical, err)
	}
	if m.Canonical == nil {
		e.noCanonical(subject, path, shown, "its governing mapping has no canonical mapping")
		return "", false, nil
	}
	c, err := m.Canonical(v)
	if err != nil {
		e.noCanonical(subject, path, shown, err.Error())
		return "", false, nil
	}
	normalized, err := value.ApplyWhiteSpace(e.schema, st, c)
	if err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
	}
	if _, err := m.Parse(normalized, nil); err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, whose canonical representation %q the backend's Parse rejects: %w", path, shown, subject, c, err)
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, st, c, nil, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && !soleLexical(f) {
			e.noCanonical(subject, path, shown, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", c, err))
			return c, false, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, shown, subject, err))
		return c, false, nil
	}
	if !mapsBack(c, f, bits) {
		e.noCanonical(subject, path, shown, fmt.Sprintf("its canonical representation %q does not map back to it", c))
		return c, false, nil
	}
	return c, true, nil
}

// spelled returns the canonical representation of v, a value of the simple
// type st whose primitive is one of encodedPrimitives, held in a, its atomic
// Go type, an atomic formed holds for, at path, subject naming st:
// a.form's spelling of v, which Parse of the mapping of st's {primitive type
// definition} on e's backend must accept, held to st's facets by
// value.ValidateLexical, the value that returns rendered by st's governing
// mapping's Canonical, which must be the spelling itself. a.form runs first and
// is the only check that v lies in a's value space, so that the backend's
// Parse and Canonical see no value outside it. The governing mapping's Parse
// is reached inside ValidateLexical alone, so a rejection by it is a verdict
// on the value, never a backend fault: the governing mapping may be a derived
// builtin's (value.Backend), narrower than the Go type codegen gives it, as
// big.Int holds 0 though xs:positiveInteger does not. It records the item 6
// or 7 error [Append]'s doc states for v, a pattern facet's rejection item 7
// where v has another lexical (a.otherLexicalOf). ok is atom's.
func (e *encoder) spelled(st *xsd.SimpleType, a atomic, v reflect.Value, path, subject string) (text string, ok bool, err error) {
	lexical, err := a.form(v)
	if err != nil {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, no value of its datatype: %w", path, a.shown(v), subject, err))
		return "", false, nil
	}
	primitive, err := st.Primitive(e.schema)
	if err != nil {
		return "", false, fmt.Errorf("codec: Append: %s, of %s: %w", path, subject, err)
	}
	if primitive == nil {
		// atomicEncodes declines a type with no {primitive type definition}.
		return "", false, fmt.Errorf("codec: Append: %s, of %s: the type has no {primitive type definition}", path, subject)
	}
	pm, has := e.backend.Mapping(primitive.Name())
	if !has {
		return "", false, fmt.Errorf("codec: Append: %s, of %s: the backend has no mapping of its primitive %s", path, subject, primitive.Name().Display())
	}
	if _, err := pm.Parse(lexical, nil); err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, whose canonical spelling the backend's Parse of %s rejects: %w", path, lexical, subject, primitive.Name().Display(), err)
	}
	val, err := value.ValidateLexical(e.backend, e.schema, st, lexical, nil, noAssertions{})
	if err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, lexical, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && a.otherLexicalOf(v) {
			e.noCanonical(subject, path, lexical, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", lexical, err))
			return lexical, false, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, lexical, subject, err))
		return lexical, false, nil
	}
	m, err := e.mapping(st, path, subject)
	if err != nil {
		return "", false, err
	}
	if m.Canonical == nil {
		e.noCanonical(subject, path, lexical, "its governing mapping has no canonical mapping")
		return "", false, nil
	}
	c, err := m.Canonical(val)
	if err != nil {
		e.noCanonical(subject, path, lexical, err.Error())
		return "", false, nil
	}
	if c != lexical {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend's Canonical renders %q, other text than its canonical representation (Datatypes Appendix E, the canonical mapping of %s)", path, lexical, subject, c, primitive.Name().Display())
	}
	return c, true, nil
}

// mapsBack reports whether strconv.ParseFloat, rounding at bits, reads c,
// double's canonical representation of f, back as f, every NaN equal to every
// NaN. Where bits is 32 both are float32 values held in a float64, which the
// widening holds bit for bit, so comparing their float64 bits compares their
// float32 ones.
func mapsBack(c string, f float64, bits int) bool {
	back, err := strconv.ParseFloat(c, bits)
	if err != nil {
		return false
	}
	return math.Float64bits(back) == math.Float64bits(f) || math.IsNaN(back) && math.IsNaN(f)
}

// soleLexical reports whether floatRep or doubleRep spells f by its canonical
// representation alone, as each does NaN and -INF (Datatypes §3.3.4.2,
// §3.3.5.2).
func soleLexical(f float64) bool { return math.IsNaN(f) || math.IsInf(f, -1) }

// otherLexical reports whether v, the []V of a list whose items a, the item's
// atomic Go type, holds, has a lexical other than its canonical one for a
// pattern facet to match: v holds an item a.otherLexicalOf holds of, since the
// list's text has another lexical exactly where one of its items has.
func otherLexical(a atomic, v reflect.Value) bool {
	for i := range v.Len() {
		if a.otherLexicalOf(v.Index(i)) {
			return true
		}
	}
	return false
}

// mapping returns the governing mapping of st on e's backend
// (value.GoverningMapping), or a backend fault naming the value at path,
// subject naming st.
func (e *encoder) mapping(st *xsd.SimpleType, path, subject string) (value.Mapping, error) {
	m, ok, err := value.GoverningMapping(e.backend, e.schema, st)
	if err != nil {
		return value.Mapping{}, fmt.Errorf("codec: Append: %s, of %s: %w", path, subject, err)
	}
	if !ok {
		return value.Mapping{}, fmt.Errorf("codec: Append: %s, of %s: the backend has no mapping governing the type", path, subject)
	}
	return m, nil
}

// rulePatternValid is pattern Valid (Datatypes §4.3.4.4 cvc-pattern-valid),
// the rule whose rejection of the canonical representation of a value other
// than an xs:string- or xs:anyURI-primitive one, or of a list holding one,
// spelled, double and list read as item 7 of [Append]'s doc where the value
// has another lexical: where otherLexicalOf holds for spelled, soleLexical
// does not hold for double and otherLexical holds for list.
const rulePatternValid xsderr.Rule = "cvc-pattern-valid"

// doubleLexical spells f in floatRep where bits is 32 and in doubleRep where
// it is 64 (Datatypes §3.3.4.2, §3.3.5.2): a finite f as strconv's shortest
// 'E' form at bits, which the decimal-and-exponent production the two share
// matches, and the infinities and NaN as INF, -INF and NaN, which Go spells
// otherwise.
func doubleLexical(f float64, bits int) string {
	switch {
	case math.IsInf(f, 1):
		return "INF"
	case math.IsInf(f, -1):
		return "-INF"
	case math.IsNaN(f):
		return "NaN"
	}
	return strconv.FormatFloat(f, 'E', -1, bits)
}

// appendEscaped appends s to dst escaped for an attribute value delimited by
// '"' (attr) or for element content: '<', '&' and '>' always, so that "]]>"
// is never written; '"' and TAB and LF in an attribute value, which XML 1.0
// §3.3.3 attribute-value normalization would otherwise rewrite to a space;
// and CR everywhere, which XML 1.0 §2.11 end-of-line handling would otherwise
// rewrite to LF or remove. Every other character is written as itself, the
// apostrophe among them.
func appendEscaped(dst []byte, s string, attr bool) []byte {
	for i := range len(s) {
		c := s[i]
		switch {
		case c == '<':
			dst = append(dst, "&lt;"...)
		case c == '&':
			dst = append(dst, "&amp;"...)
		case c == '>':
			dst = append(dst, "&gt;"...)
		case c == '\r':
			dst = append(dst, "&#xD;"...)
		case attr && c == '"':
			dst = append(dst, "&quot;"...)
		case attr && c == '\t':
			dst = append(dst, "&#x9;"...)
		case attr && c == '\n':
			dst = append(dst, "&#xA;"...)
		default:
			dst = append(dst, c)
		}
	}
	return dst
}

// noAssertions is the value.AssertionEvaluator Append passes
// value.ValidateLexical, which requires one: it decides no {test}. No call
// reaches it, since a simple type with an assertions facet declines before
// any value is read (constructAssertions).
type noAssertions struct{}

// Evaluate declines every {test}.
func (noAssertions) Evaluate(value.Backend, xsd.TypeResolver, *xsd.SimpleType, xsd.XPathExpression, value.Value) value.AssertionAnswer {
	return value.AssertionDeclinedBecause("codec's encoder evaluates no assertion")
}
