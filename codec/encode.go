package codec

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"unicode/utf8"

	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/internal/xmlchar"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// Append appends to dst the XML document that encodes v as the document
// element of the top-level element declaration name in schema, and returns
// the extended slice; dst may be nil. It writes each simple value in its
// canonical representation (Datatypes §2.3.1 dt-canonical-representation),
// the form the Canonical of its governing mapping on backend
// (value.GoverningMapping) renders, and validates nothing: a caller assesses
// the document through package validate, as any document. The package doc's
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
//     element's attributes before its content: a plain error naming the Go
//     path of its field from T and, for a simple value, its simple type. It
//     is a slice field whose length lies outside its particle's {min occurs}
//     and {max occurs}; a string holding bytes that are not UTF-8 or a
//     character the XML 1.0 Char production does not match, U+0000 among them
//     (Datatypes §3.3.1); a string that differs from its own whiteSpace-normalized
//     form under its simple type, such as an xs:token holding "  a  b"; or a
//     value a facet of its simple type rejects, that facet's *xsderr.Error
//     wrapped, but for item 7's pattern case.
//  7. A value with no canonical representation, the first in document order
//     as item 6 counts: its *DeclineError. That is a governing mapping with no
//     Canonical, a Canonical that fails for the value, an xs:double value
//     whose canonical representation a pattern facet of its type rejects
//     (Datatypes §2.1's note), whatever other facet rejects it, or a
//     canonical representation that does not map back to the value.
//
// A plain error is no *DeclineError and wraps neither [ErrTypeMismatch] nor
// errors.ErrUnsupported; only item 6's facet case carries an xsderr rule.
//
// A fault of backend found at a value, in place of that value's item 6 or 7,
// is a plain error too, returned at once and naming the value's Go path and
// simple type: no governing mapping for the type, a Parse that rejects the
// form Append spells an xs:double value in, or an error of
// value.ValidateLexical that value.IsDatatypeVerdict does not report as a
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
		return encodeElement{}, fmt.Errorf("codec: no top-level element declaration has the expanded name %q", name.String())
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
	simple     *xsd.SimpleType
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
	simple *xsd.SimpleType
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
		return encodeElement{decl: decl, simple: st}, simpleEncodes(schema, st, declSubject(decl))
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
		return encodeElement{}, encodeDecline(constructElementRef, fmt.Sprintf("%s, naming element %s", subject, term.Name))
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
	return encodeElement{decl: el, simple: st}, simpleEncodes(schema, st, declSubject(el))
}

// attributePlanOf returns the encodeAttribute of the attribute use u, or the
// error encodePlanOf states: a qualified declaration, whose name the default
// namespace binding Append writes cannot spell (Namespaces in XML §6.2), then
// an effective value constraint ([xsd.Schema.EffectiveValueConstraint]), then
// its simple type.
func attributePlanOf(schema *xsd.Schema, u xsd.AttributeUse) (encodeAttribute, error) {
	decl, ok := schema.ResolvedAttributeDeclaration(u)
	if !ok {
		return encodeAttribute{}, fmt.Errorf("codec: the {attribute declaration} of attribute use %s does not resolve", u.DeclarationName())
	}
	subject := fmt.Sprintf("attribute %s declared at %s", decl.Name(), decl.Loc())
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
	return encodeAttribute{decl: decl, simple: st}, simpleEncodes(schema, st, subject)
}

// The expanded names simpleEncodes reads a simple type's primitive and base
// chain by.
var (
	stringName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}
	doubleName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "double"}
	idNames    = []xsd.QName{
		{Space: xsd.XMLSchemaNS, Local: "ID"},
		{Space: xsd.XMLSchemaNS, Local: "IDREF"},
		{Space: xsd.XMLSchemaNS, Local: "ENTITY"},
	}
)

// simpleEncodes returns nil where the simple type st, the type of the
// declaration owner names, encodes as the package doc's "# What encodes"
// states, and otherwise the decline of the first construct it finds: a
// special type, a list, a union, a primitive other than xs:string and
// xs:double, a type on whose {base type definition} chain xs:ID, xs:IDREF or
// xs:ENTITY lies, then an assertions facet among its effective facets, its
// own or inherited. An error reading st is returned as is. {variety} is a
// closed sum, so the switch is STYLE T2's sealed-sum exception.
func simpleEncodes(r xsd.TypeResolver, st *xsd.SimpleType, owner string) error {
	subject := simpleSubject(st, owner)
	if st.IsSpecial() {
		return encodeDecline(constructSpecialType, subject)
	}
	variety, err := st.Variety(r)
	if err != nil {
		return err
	}
	switch variety.(type) {
	case xsd.List:
		return encodeDecline(constructList, subject)
	case xsd.Union:
		return encodeDecline(constructUnion, subject)
	}
	primitive, err := st.Primitive(r)
	if err != nil {
		return err
	}
	if primitive == nil || primitive.Name() != stringName && primitive.Name() != doubleName {
		return encodeDecline(constructEncodePrimitive, subject)
	}
	for step := st; step != nil; {
		for _, id := range idNames {
			if step.Name() == id {
				return encodeDecline(constructIDType, fmt.Sprintf("%s, derived from %s", subject, id))
			}
		}
		step, err = step.Base(r)
		if err != nil {
			return err
		}
	}
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
	invalid     error
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
		text, err := e.simple(el.simple, v, path, declSubject(el.decl))
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
		text, err := e.simple(a.simple, f, fpath, fmt.Sprintf("attribute %s declared at %s", a.decl.Name(), a.decl.Loc()))
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
// path, its parent element named parent: one per item of a []V, which is item
// 6 of [Append]'s doc where its length lies outside p's occurrence, none for
// a nil *V, and one for a V. The error is element's.
func (e *encoder) particle(p encodeParticle, f reflect.Value, path string, parent xsd.QName) error {
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return nil
		}
		return e.element(p.element, f.Elem(), path, parent, false)
	}
	if f.Kind() != reflect.Slice {
		return e.element(p.element, f, path, parent, false)
	}
	if !p.occurs.Permits(f.Len()) {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %d items, outside the occurrence %s of the particle of %s", path, f.Len(), p.occurs, declSubject(p.element.decl)))
	}
	for j := range f.Len() {
		if err := e.element(p.element, f.Index(j), path+"["+strconv.Itoa(j)+"]", parent, false); err != nil {
			return err
		}
	}
	return nil
}

// invalidate records err as the document's item 6 error of [Append]'s doc
// unless an earlier value's is recorded.
func (e *encoder) invalidate(err error) {
	if e.invalid == nil {
		e.invalid = err
	}
}

// noCanonical records the item 7 decline of [Append]'s doc for the value
// shown at path, of the simple type subject names, cause saying why, unless
// an earlier value's is recorded.
func (e *encoder) noCanonical(subject, path, shown, cause string) {
	if e.uncanonical == nil {
		e.uncanonical = encodeDecline(constructNoCanonical, fmt.Sprintf("%s, at %s holding %s: %s", subject, path, shown, cause))
	}
}

// simple returns the text Append writes for v, a value of the simple type st
// held as string or float64, the Go types the slice gives it, at path, owner
// naming st's declaration: a string as itself (str), a float64 in its
// canonical representation (double). It records the value's item 6 or 7
// error of [Append]'s doc, and returns a backend fault.
func (e *encoder) simple(st *xsd.SimpleType, v reflect.Value, path, owner string) (string, error) {
	subject := simpleSubject(st, owner)
	if v.Kind() == reflect.Float64 {
		return e.double(st, v.Float(), path, subject)
	}
	return e.str(st, v.String(), path, subject)
}

// str returns s, a value of the simple type st whose primitive is xs:string,
// after checking, in this order, that it is UTF-8 of XML 1.0 Char characters
// only, that value.ValidateLexical accepts it under st, and that it is the
// canonical representation of the value that returns, its own
// whiteSpace-normalized form, since xs:string's canonical mapping is the
// identity (Datatypes §3.3.1, §E.4 f-stringCanmap). Canonical renders the
// value ValidateLexical returns, never one Parse maps s to: Parse takes a
// normalized lexical, and s may be none.
func (e *encoder) str(st *xsd.SimpleType, s, path, subject string) (string, error) {
	shown := strconv.Quote(s)
	if i, bad := notChars(s); bad {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, whose byte offset %d starts no XML 1.0 Char (Datatypes §3.3.1)", path, shown, subject, i))
		return s, nil
	}
	v, err := value.ValidateLexical(e.backend, e.schema, st, s, nil, noAssertions{})
	if err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, shown, subject, err))
		return s, nil
	}
	m, err := e.mapping(st, path, subject)
	if err != nil {
		return "", err
	}
	if m.Canonical == nil {
		e.noCanonical(subject, path, shown, "its governing mapping has no canonical mapping")
		return s, nil
	}
	c, err := m.Canonical(v)
	if err != nil {
		e.noCanonical(subject, path, shown, err.Error())
		return s, nil
	}
	if c != s {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, which differs from its whiteSpace-normalized form %q", path, shown, subject, c))
	}
	return s, nil
}

// double returns the canonical representation of f, a value of the simple
// type st whose primitive is xs:double: f spelled in doubleRep (doubleLexical)
// mapped by st's governing mapping's Parse, rendered by its Canonical, then
// held to st's facets by value.ValidateLexical and read back by
// strconv.ParseFloat, every NaN equal to every NaN. A pattern facet's
// rejection is item 7 of [Append]'s doc, since another lexical of the value
// may match it (Datatypes §2.1's note); any other facet's is item 6.
func (e *encoder) double(st *xsd.SimpleType, f float64, path, subject string) (string, error) {
	shown := strconv.FormatFloat(f, 'g', -1, 64)
	m, err := e.mapping(st, path, subject)
	if err != nil {
		return "", err
	}
	lexical := doubleLexical(f)
	v, err := m.Parse(lexical, nil)
	if err != nil {
		return "", fmt.Errorf("codec: Append: %s holds %s, of %s, whose doubleRep form %q the backend's Parse rejects: %w", path, shown, subject, lexical, err)
	}
	if m.Canonical == nil {
		e.noCanonical(subject, path, shown, "its governing mapping has no canonical mapping")
		return "", nil
	}
	c, err := m.Canonical(v)
	if err != nil {
		e.noCanonical(subject, path, shown, err.Error())
		return "", nil
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, st, c, nil, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid {
			e.noCanonical(subject, path, shown, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", c, err))
			return c, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, shown, subject, err))
		return c, nil
	}
	back, err := strconv.ParseFloat(c, 64)
	if err != nil || math.Float64bits(back) != math.Float64bits(f) && !(math.IsNaN(back) && math.IsNaN(f)) {
		e.noCanonical(subject, path, shown, fmt.Sprintf("its canonical representation %q does not map back to it", c))
	}
	return c, nil
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
// the rule whose rejection of an xs:double value's canonical representation
// double reads as item 7 of [Append]'s doc.
const rulePatternValid xsderr.Rule = "cvc-pattern-valid"

// doubleLexical spells f in doubleRep (Datatypes §3.3.5.2): a finite f as
// strconv's 'E' form, which doubleRep's decimal-and-exponent production
// matches, and the infinities and NaN as INF, -INF and NaN, which Go spells
// otherwise.
func doubleLexical(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "INF"
	case math.IsInf(f, -1):
		return "-INF"
	case math.IsNaN(f):
		return "NaN"
	}
	return strconv.FormatFloat(f, 'E', -1, 64)
}

// notChars reports whether s holds a byte sequence that is not UTF-8 or a
// rune xmlchar.IsChar rejects, and the byte offset of the first.
func notChars(s string) (int, bool) {
	for i, r := range s {
		if r == utf8.RuneError {
			if _, size := utf8.DecodeRuneInString(s[i:]); size <= 1 {
				return i, true
			}
		}
		if !xmlchar.IsChar(r) {
			return i, true
		}
	}
	return 0, false
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
