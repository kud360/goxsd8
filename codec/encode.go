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
//  5. A T that is not the type codegen emits for the declaration under opts,
//     item 6 of [Decode]'s doc: its error, which errors.Is reports as
//     [ErrTypeMismatch].
//  6. A value no valid document holds, the first in document order, an
//     element's attributes before its content: an error, which errors.Is
//     reports as [ErrInvalidValue], naming the Go path of its field from T
//     and, for a simple value, its simple type. It is the slice field of a
//     particle with {max occurs} above 1 whose length lies outside its {min
//     occurs} and {max occurs}, a choice's alternative's {min occurs} 0 read
//     as 1 in its branch; the nil sealed interface of a content choice none of
//     whose alternatives is emptiable (Structures §3.8.4.3 cvc-model-group,
//     §3.9.6.3 cos-group-emptiable), a choice of no {particles} among them; a
//     value of that interface whose dynamic type the [WithBranches] set in
//     opts places at no alternative, a pointer to a branch among them, which
//     Decode never returns; a string holding bytes that are not UTF-8 or
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
// Of opts Append reads only the sets [WithBranches] and
// [WithPrecisionDecimalArms] supply. Items 2 to 5 read no value;
// [CheckAppend] returns them for a declaration, T and opts.
func Append[T any](dst []byte, schema *xsd.Schema, backend value.Backend, name xsd.QName, v T, opts ...Option) ([]byte, error) {
	if schema == nil {
		return dst, errors.New("codec: Append: nil *xsd.Schema")
	}
	if backend == nil {
		return dst, errors.New("codec: Append: nil value.Backend")
	}
	t := reflect.TypeFor[T]()
	decl, f, err := encodeStatic(schema, name, t, opts)
	if err != nil {
		return dst, err
	}
	e := &encoder{schema: schema, backend: backend, out: dst}
	// v's own Value, of Kind Interface where T is one, so that a nil T and a
	// branch's dynamic type both reach the write as T holds them.
	if err := encoded[writer](f).write(e, reflect.ValueOf(&v).Elem(), writeAt{el: decl, path: typePath(t)}); err != nil {
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
// top-level element declaration name in schema under opts, in the class
// that item states, or nil when none applies; for a nil schema it returns a
// plain error. Item 4's slice is the one the package doc's "# What encodes"
// states. Items 6 and 7 and a fault of the backend depend on the value, so
// they lie outside CheckAppend's answer, and nil leaves them open. The
// backend is no input, so a nil backend, which Append's item 1 refuses, lies
// outside it too. For a non-nil schema, a non-nil error is the one Append
// returns, of equal Error text and class, for every value of T, every non-nil
// backend and the same opts.
func CheckAppend[T any](schema *xsd.Schema, name xsd.QName, opts ...Option) error {
	if schema == nil {
		return errors.New("codec: CheckAppend: nil *xsd.Schema")
	}
	_, _, err := encodeStatic(schema, name, reflect.TypeFor[T](), opts)
	return err
}

// typePath names t as the Go path of a value from T starts it: by its name,
// or, for a type with none, its String.
func typePath(t reflect.Type) string {
	if t.Name() == "" {
		return t.String()
	}
	return t.Name()
}

// encodeStatic returns the top-level element declaration name in schema, a
// non-nil schema, and the filler item 5's check returns for it under opts,
// Decode's for a T of Go type t, or the error of the first of items 2 to 5 of
// [Append]'s doc that applies. It reads no value, so its answer holds for
// every value of t. Item 4's pass (elementEncodes) walks the schema and
// returns before bindDecl is called, so that its declines come first; it
// builds nothing, and Append writes from the filler alone (writer).
func encodeStatic(schema *xsd.Schema, name xsd.QName, t reflect.Type, opts []Option) (xsd.ElementDeclaration, filler, error) {
	decl, ok := schema.Element(name)
	if !ok {
		return xsd.ElementDeclaration{}, nil, fmt.Errorf("codec: no top-level element declaration has the expanded name %q", name.Display())
	}
	// An answer about the schema, no violation of an instance, so it carries
	// no xsderr rule (STYLE E2).
	if decl.Abstract() {
		return xsd.ElementDeclaration{}, nil, fmt.Errorf("codec: %s has {abstract} true, so no document element validates against it (cvc-elt clause 2)", declSubject(decl))
	}
	if err := elementEncodes(schema, decl); err != nil {
		return xsd.ElementDeclaration{}, nil, err
	}
	b, err := bindDecl(&pipeline{schema: schema}, decl)
	if err != nil {
		return xsd.ElementDeclaration{}, nil, err
	}
	f, err := b.check(t, newCheckContext(newConfig(opts)))
	if err != nil {
		return xsd.ElementDeclaration{}, nil, err
	}
	return decl, f, nil
}

// A writer is a bound arm of the filler item 5's check returns that Append
// writes a value from: write appends to e.out the XML encoding v, a value of
// the Go type the arm is bound to, at at, and returns a backend fault. Its
// implementers are exactly the arms the slice the package doc's "# What
// encodes" states reaches: the root arms, *simpleRoot, *boundBinding and
// *boundChoiceRoot, writing at.el itself, the document element; the content
// arms, *boundSequence and *boundContentChoice, writing at.el's content; and
// elementMember, writing an element particle's elements, a sequence's or a
// choice alternative's, children of at.el. Append reaches each through
// encoded.
type writer interface {
	write(e *encoder, v reflect.Value, at writeAt) error
}

// writeAt is what a writer reads beside its value: el, the element
// declaration of the element the value is written as, for a root arm, or of
// the element whose content holds it, for every other writer; typ, the complex
// type a content arm writes the content of, el's, which its item 6 errors name
// (typeSubject); and path, the value's Go path from T.
type writeAt struct {
	el   xsd.ElementDeclaration
	typ  xsd.ComplexType
	path string
}

// encoded returns x, an arm of the filler item 5's check returns, as C, the
// capability (writer) or closed-sum arm (scalarArm, elementMember, atomic)
// Append's write reads it as. Invariant: where item 4 of [Append]'s doc admits
// a construct, every arm the binder builds for it that the write reaches is
// such a C. A failed assertion breaks it, an internal bug, never input, and
// panics naming x's type.
func encoded[C any](x any) C {
	c, ok := x.(C)
	if !ok {
		panic(fmt.Sprintf("codec: Append reached a %T, which is no %v: item 4 of Append's doc admitted a construct whose bound arm Append does not write", x, reflect.TypeFor[C]()))
	}
	return c
}

// write writes at.el, the document element, holding v, a value of s's simple
// type (encoder.simpleElement).
func (s *simpleRoot) write(e *encoder, v reflect.Value, at writeAt) error {
	return e.simpleElement(at.el, s.scalar, v, at.path)
}

// write writes at.el, the document element, holding v, a value of b's struct:
// its start tag, then each attribute field's attribute, in field order, the
// fields after the content's (binding.lead, elementContent.width), a nil
// *V's written as none, then the content's elements (b.content's write), an
// element with no content written as an empty-element tag.
func (b *boundBinding) write(e *encoder, v reflect.Value, at writeAt) error {
	bd := b.binding
	name := at.el.Name()
	outer := e.startTag(name)
	first := bd.lead() + bd.content.width()
	for i, a := range bd.attrs {
		f, fpath := v.Field(first+i), at.path+"."+v.Type().Field(first+i).Name
		if a.shape == shapeOptional {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		text, err := e.simple(a.scalar, f, fpath, fmt.Sprintf("attribute %s declared at %s", a.decl.Name().Display(), a.decl.Loc()))
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
	if err := encoded[writer](b.content).write(e, v, writeAt{el: at.el, typ: bd.typ, path: at.path}); err != nil {
		return err
	}
	e.endTag(name, start, outer)
	return nil
}

// write writes at.el, the document element, holding v, a value of the
// choice's sealed interface: its start tag, then the branch v holds
// (writeChoice), an empty match written as an empty-element tag.
func (c *boundChoiceRoot) write(e *encoder, v reflect.Value, at writeAt) error {
	name := at.el.Name()
	outer := e.startTag(name)
	start := len(e.out)
	e.out = append(e.out, '>')
	if err := writeChoice(e, c.root.sealed.alts, c.choice, v, writeAt{el: at.el, typ: c.root.typ, path: at.path}); err != nil {
		return err
	}
	e.endTag(name, start, outer)
	return nil
}

// write writes the elements of each of b's members from its field, member i's
// field b.first+i of v, in member order.
func (b *boundSequence) write(e *encoder, v reflect.Value, at writeAt) error {
	for i, m := range b.sequence.members {
		j := b.first + i
		if err := encoded[writer](m).write(e, v.Field(j), writeAt{el: at.el, path: at.path + "." + v.Type().Field(j).Name}); err != nil {
			return err
		}
	}
	return nil
}

// write writes the branch the choice's sealed interface holds, field b.index
// of v (writeChoice), then the elements of each member of the rest, whose
// fields follow it.
func (b *boundContentChoice) write(e *encoder, v reflect.Value, at writeAt) error {
	if err := writeChoice(e, b.content.choice.sealed.alts, b.choice, v.Field(b.index), writeAt{el: at.el, typ: at.typ, path: at.path + "." + v.Type().Field(b.index).Name}); err != nil {
		return err
	}
	for i, m := range b.content.rest.members {
		j := b.index + 1 + i
		if err := encoded[writer](m).write(e, v.Field(j), writeAt{el: at.el, path: at.path + "." + v.Type().Field(j).Name}); err != nil {
			return err
		}
	}
	return nil
}

// writeChoice writes the branch v, a value of the sealed interface of a
// choice occurring once whose alternatives are alts, holds, bc that interface
// as item 5's check bound it: its branches[j], the branch type placed at
// alts[j] (sealed.checkSite), is matched to v's dynamic type, and the members
// of alts[j] are written from the branch struct's fields, in field order. The
// nil interface writes nothing, the empty match, but is item 6 of [Append]'s
// doc where no alternative is emptiable (choiceEmptiable), and so is a
// dynamic type bc places at no alternative, a pointer to a branch among them;
// at.typ is the complex type whose content the choice is.
func writeChoice(e *encoder, alts []branchArm, bc *boundChoice, v reflect.Value, at writeAt) error {
	if v.IsNil() {
		if !choiceEmptiable(alts) {
			e.invalidate(fmt.Errorf("codec: Append: %s is nil, the empty match, which the content choice of %s does not accept: %s", at.path, typeSubject(at.el, at.typ), unemptiable(alts)))
		}
		return nil
	}
	b := v.Elem()
	j := slices.IndexFunc(bc.branches, func(bb boundBranch) bool { return bb.t == b.Type() })
	if j < 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, which the WithBranches set of %s places at no alternative of the content choice of %s, so that Decode returns no such value", at.path, b.Type(), v.Type(), typeSubject(at.el, at.typ)))
		return nil
	}
	for i, m := range alts[j].branchMembers() {
		if err := encoded[writer](m).write(e, b.Field(i), writeAt{el: at.el, path: at.path + ".(" + typePath(b.Type()) + ")." + b.Type().Field(i).Name}); err != nil {
			return err
		}
	}
	return nil
}

// choiceEmptiable reports whether one occurrence of the choice whose
// alternatives are alts, each an element particle, matches the empty
// sequence: one of them has {min occurs} 0 (§3.9.6.3 cos-group-emptiable).
func choiceEmptiable(alts []branchArm) bool {
	return slices.ContainsFunc(alts, func(a branchArm) bool { return encoded[elementMember](a).own.Min() == 0 })
}

// unemptiable says why the empty sequence lies outside the language of the
// choice whose alternatives are alts, none of them emptiable: it has none
// (§3.8.4.3's note), or each has {min occurs} at least 1.
func unemptiable(alts []branchArm) string {
	if len(alts) == 0 {
		return "the choice has no alternative, so that it accepts no sequence (§3.8.4.3 cvc-model-group)"
	}
	return fmt.Sprintf("none of its %d alternatives is emptiable (§3.8.4.3 cvc-model-group, §3.9.6.3 cos-group-emptiable)", len(alts))
}

// write writes the elements of f, the field of m's particle, at.path its Go
// path, each an element of m.decl, of a simple type (scalarArm), by the shape
// item 5 bound the field in (field.shape), computed at m.held, never by f's
// Kind, since a list's V is itself a slice: one per item of a []V, which is
// item 6 of [Append]'s doc where its length lies outside m.held, the error
// naming m.own where a choice's alternative raised it, none for a nil *V, and
// one for a V.
func (m elementMember) write(e *encoder, f reflect.Value, at writeAt) error {
	a := encoded[scalarArm](m.arm)
	switch m.shape {
	case shapeOptional:
		if f.IsNil() {
			return nil
		}
		return e.simpleElement(m.decl, a.scalar, f.Elem(), at.path)
	case shapeRepeated:
		if !m.held.Permits(f.Len()) {
			raised := ""
			if m.held != m.own {
				raised = fmt.Sprintf(", the occurrence %s with {min occurs} 0 read as 1 in its branch,", m.own)
			}
			e.invalidate(fmt.Errorf("codec: Append: %s holds %d items, outside the occurrence %s%s of the particle of %s", at.path, f.Len(), m.held, raised, declSubject(m.decl)))
		}
		for j := range f.Len() {
			if err := e.simpleElement(m.decl, a.scalar, f.Index(j), at.path+"["+strconv.Itoa(j)+"]"); err != nil {
				return err
			}
		}
		return nil
	default:
		return e.simpleElement(m.decl, a.scalar, f, at.path)
	}
}

// elementEncodes returns the error item 4 of [Append]'s doc states for the
// first construct of decl, the document element's declaration, outside the
// slice the package doc's "# What encodes" states, in the order the construct
// constants list, or a plain error for a component that does not resolve, and
// otherwise nil. It reads the schema alone, never the binder, and builds
// nothing.
func elementEncodes(schema *xsd.Schema, decl xsd.ElementDeclaration) error {
	if err := declEncodes(decl); err != nil {
		return err
	}
	td, ok := schema.ResolvedType(decl.TypeDefinition())
	if !ok {
		return fmt.Errorf("codec: the {type definition} of %s does not resolve", declSubject(decl))
	}
	if goshape.AnyTypeItself(td) {
		return encodeDecline(constructAnyType, declSubject(decl))
	}
	if st, isSimple := td.(*xsd.SimpleType); isSimple {
		return simpleEncodes(schema, st, declSubject(decl))
	}
	ct, _ := td.(xsd.ComplexType)
	if ct.Abstract() {
		return encodeDecline(constructAbstractType, typeSubject(decl, ct))
	}
	return complexEncodes(schema, decl, ct)
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

// complexEncodes returns elementEncodes' error for decl, whose {type
// definition} is the complex type ct: ct's content, then its content
// {particle}, {max occurs} 0 declined before its hold is asked (#3794), then
// each particle in order, then each attribute use codegen gives a field
// (goshape.FieldAttributeUses), then its {attribute wildcard}, then its
// {assertions}.
func complexEncodes(schema *xsd.Schema, decl xsd.ElementDeclaration, ct xsd.ComplexType) error {
	subject := typeSubject(decl, ct)
	// {content type} is a closed sum, so the switch is STYLE T2's sealed-sum
	// exception.
	switch content := ct.ContentType().(type) {
	case xsd.SimpleContent:
		return encodeDecline(constructSimpleContent, subject)
	case xsd.ElementContent:
		if err := contentEncodes(schema, content, subject); err != nil {
			return err
		}
	}
	for _, u := range goshape.FieldAttributeUses(ct) {
		if err := attributeEncodes(schema, u); err != nil {
			return err
		}
	}
	if _, has := ct.AttributeWildcard(); has {
		return encodeDecline(constructAttributeWildcard, subject)
	}
	if len(ct.Assertions()) > 0 {
		return encodeDecline(constructComplexAssertions, subject)
	}
	return nil
}

// contentEncodes returns complexEncodes' error for content, element-only or
// mixed content of the complex type subject names: its {variety}, its {open
// content}, its {particle}'s {max occurs} 0, then its model group and hold as
// contentGroup, Decode's, answers them, contentGroup's errors returned as
// they are and its construct, which §3.4.2.3.3 leaves no schema to reach, as
// a plain error; then each particle of a spliced sequence, or of a choice
// occurring once that codegen does not flatten, as particleEncodes admits it.
func contentEncodes(schema *xsd.Schema, content xsd.ElementContent, subject string) error {
	if content.Mixed {
		return encodeDecline(constructMixed, subject)
	}
	if content.OpenContent != nil {
		return encodeDecline(constructOpenContent, subject)
	}
	if upper, bounded := content.Particle.Occurs().Max(); bounded && upper == 0 {
		return encodeDecline(constructVacuousParticle, "the content {particle} of "+subject)
	}
	group, hold, d, err := contentGroup(schema, content, subject)
	if err != nil {
		return err
	}
	if d != 0 {
		return fmt.Errorf("codec: the content {particle} of %s holds no model group", subject)
	}
	label := "particle %d of the content {particle} of %s"
	switch hold {
	case goshape.Flattened:
		return encodeDecline(constructSealedInterface, subject)
	case goshape.OnceChoice:
		label = "alternative %d of the content {particle} of %s"
	case goshape.Occurrence:
		return encodeDecline(constructOccurrenceField, subject)
	case goshape.Spliced:
		if group.Compositor() == xsd.CompositorAll {
			return encodeDecline(constructAllGroup, subject)
		}
	}
	for i, p := range group.Particles() {
		if err := particleEncodes(schema, p, fmt.Sprintf(label, i+1, subject)); err != nil {
			return err
		}
	}
	return nil
}

// particleEncodes returns elementEncodes' error for the particle p of a
// spliced content sequence or of a content choice's alternative, subject
// naming p: {max occurs} 0, then a term other than a local element
// declaration, then the declaration's own properties and type, of which a
// complex one other than xs:anyType itself declines whole. {term} is a closed
// sum, so the switches are STYLE T2's sealed-sum exception.
func particleEncodes(schema *xsd.Schema, p xsd.Particle, subject string) error {
	if upper, bounded := p.Occurs().Max(); bounded && upper == 0 {
		return encodeDecline(constructVacuousParticle, subject)
	}
	var el xsd.ElementDeclaration
	switch term := p.Term().(type) {
	case xsd.ModelGroupRef:
		return encodeDecline(constructNestedGroup, subject)
	case xsd.ElementDeclarationRef:
		return encodeDecline(constructElementRef, fmt.Sprintf("%s, naming element %s", subject, term.Name.Display()))
	case xsd.ResolvedTerm:
		switch term := term.Term.(type) {
		case xsd.ModelGroup:
			return encodeDecline(constructNestedGroup, subject)
		case xsd.Wildcard:
			return encodeDecline(constructWildcard, subject)
		case xsd.ElementDeclaration:
			el = term
		}
	}
	if err := declEncodes(el); err != nil {
		return err
	}
	td, ok := schema.ResolvedType(el.TypeDefinition())
	if !ok {
		return fmt.Errorf("codec: the {type definition} of %s does not resolve", declSubject(el))
	}
	if goshape.AnyTypeItself(td) {
		return encodeDecline(constructAnyType, declSubject(el))
	}
	st, isSimple := td.(*xsd.SimpleType)
	if !isSimple {
		return encodeDecline(constructComplexParticle, declSubject(el))
	}
	return simpleEncodes(schema, st, declSubject(el))
}

// attributeEncodes returns elementEncodes' error for the attribute use u: a
// qualified declaration, whose name the default namespace binding Append
// writes cannot spell (Namespaces in XML §6.2), then an effective value
// constraint ([xsd.Schema.EffectiveValueConstraint]), then its simple type.
func attributeEncodes(schema *xsd.Schema, u xsd.AttributeUse) error {
	decl, ok := schema.ResolvedAttributeDeclaration(u)
	if !ok {
		return fmt.Errorf("codec: the {attribute declaration} of attribute use %s does not resolve", u.DeclarationName().Display())
	}
	subject := fmt.Sprintf("attribute %s declared at %s", decl.Name().Display(), decl.Loc())
	if decl.Name().Space != "" {
		return encodeDecline(constructQualifiedAttribute, subject)
	}
	if _, has := schema.EffectiveValueConstraint(u); has {
		return encodeDecline(constructValueConstraint, subject)
	}
	st, ok := schema.ResolvedSimpleType(decl.TypeDefinition())
	if !ok {
		return fmt.Errorf("codec: the {type definition} of %s does not resolve", subject)
	}
	return simpleEncodes(schema, st, subject)
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

// simpleEncodes returns nil where the simple type st, the type of the
// declaration owner names, encodes as the package doc's "# What encodes"
// states, and otherwise the decline of the first construct it finds: a
// special type, then, for a list, the first construct of its {item type
// definition} (atomicEncodes), the detail naming the item after st
// (itemSubject), and an assertions facet among st's own effective facets, its
// own or inherited; for an atomic or union st, atomicEncodes's. An error
// reading st is returned as is, and a list with no {item type definition} is
// a plain error.
func simpleEncodes(r xsd.TypeResolver, st *xsd.SimpleType, owner string) error {
	subject := simpleSubject(st, owner)
	if st.IsSpecial() {
		return encodeDecline(constructSpecialType, subject)
	}
	variety, err := st.Variety(r)
	if err != nil {
		return err
	}
	if _, isList := variety.(xsd.List); !isList {
		return atomicEncodes(r, st, subject)
	}
	item, err := st.Item(r)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("codec: the list type %s has no {item type definition}", subject)
	}
	if err := atomicEncodes(r, item, itemSubject(subject, item)); err != nil {
		return err
	}
	return assertionsDecline(r, st, subject)
}

// atomicEncodes returns nil where st, an atomic or union simple type subject
// names, a list's {item type definition} among them, which cos-st-restricts
// clause 2.1 holds to one of those two, encodes, and otherwise the decline
// of the first construct it finds: a special type, a union, a primitive
// not in encodedPrimitives, a type on whose {base type definition} chain
// xs:ID, xs:IDREF or xs:ENTITY lies, then an assertions facet among its
// effective facets, its own or inherited. An error reading st is returned as
// is.
func atomicEncodes(r xsd.TypeResolver, st *xsd.SimpleType, subject string) error {
	if st.IsSpecial() {
		return encodeDecline(constructSpecialType, subject)
	}
	variety, err := st.Variety(r)
	if err != nil {
		return err
	}
	if _, isUnion := variety.(xsd.Union); isUnion {
		return encodeDecline(constructUnion, subject)
	}
	primitive, err := st.Primitive(r)
	if err != nil {
		return err
	}
	if primitive == nil || !slices.Contains(encodedPrimitives, primitive.Name()) {
		return encodeDecline(constructEncodePrimitive, subject)
	}
	for step := st; step != nil; {
		for _, id := range idNames {
			if step.Name() == id {
				return encodeDecline(constructIDType, fmt.Sprintf("%s, derived from %s", subject, id.Display()))
			}
		}
		step, err = step.Base(r)
		if err != nil {
			return err
		}
	}
	return assertionsDecline(r, st, subject)
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

// encoder is one Append's write of a document: out, the document so far; ns,
// the in-scope default namespace, that of the element whose content is being
// written, "" outside the document element; and the first item 6 and item 7
// errors of [Append]'s doc it has found, in document order. Every value is
// written whatever was found before it, so that the walk finds an item 6
// error after an item 7 one.
type encoder struct {
	schema      *xsd.Schema
	backend     value.Backend
	out         []byte
	ns          string
	invalid     *invalidValueError
	uncanonical error
}

// startTag appends the start tag of the element name, but for its '>', with a
// default namespace declaration where name's namespace differs from e.ns, the
// document element's from no namespace, and makes name's namespace e.ns for
// its content, returning the one it replaced, which endTag restores.
func (e *encoder) startTag(name xsd.QName) (outer string) {
	e.out = append(e.out, '<')
	e.out = append(e.out, name.Local...)
	if name.Space != e.ns {
		e.out = append(e.out, ` xmlns="`...)
		e.out = appendEscaped(e.out, name.Space, true)
		e.out = append(e.out, '"')
	}
	outer, e.ns = e.ns, name.Space
	return outer
}

// endTag ends the element name whose start tag's '>' is e.out's byte at
// start: with its end tag, or, where nothing follows that '>', by turning the
// start tag into an empty-element tag; and restores outer, startTag's
// answer, as e.ns.
func (e *encoder) endTag(name xsd.QName, start int, outer string) {
	e.ns = outer
	if len(e.out) == start+1 {
		e.out = append(e.out[:start], "/>"...)
		return
	}
	e.out = append(e.out, "</"...)
	e.out = append(e.out, name.Local...)
	e.out = append(e.out, '>')
}

// simpleElement writes the element of decl, holding v, a value of s's simple
// type, whose Go path from T is path: its start tag (startTag), then its
// value's text (simple), an empty text written as an empty-element tag. The
// error is a backend fault.
func (e *encoder) simpleElement(decl xsd.ElementDeclaration, s scalar, v reflect.Value, path string) error {
	name := decl.Name()
	outer := e.startTag(name)
	text, err := e.simple(s, v, path, declSubject(decl))
	if err != nil {
		return err
	}
	start := len(e.out)
	e.out = append(e.out, '>')
	e.out = appendEscaped(e.out, text, false)
	e.endTag(name, start, outer)
	return nil
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

// simple returns the text Append writes for v, a value of s's simple type,
// s.st, held in the Go type the slice gives it, at path, owner naming s.st's
// declaration: an atomic value by atom, a list by list, each under the atomic
// Go type of s's element (encoded). It records the value's item 6 or 7 error
// of [Append]'s doc, and returns a backend fault.
func (e *encoder) simple(s scalar, v reflect.Value, path, owner string) (string, error) {
	subject := simpleSubject(s.st, owner)
	a := encoded[atomic](s.elem)
	if s.item != nil {
		return e.list(s.st, s.item.st, a, v, path, subject)
	}
	text, _, err := e.atom(s.st, a, v, path, subject)
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
// holding a value of the list type st, at path, subject naming st, each item
// written by atom under item, st's {item type definition}, held in a, its
// atomic Go type, at path "[i]", and records the item 6 and 7 errors
// [Append]'s doc states for a list: an item's white space is checked once its
// own checks pass, and st's own facets, through value.ValidateLexical, once
// every item has. An empty item is no list item since the list's whiteSpace
// collapse (Datatypes §4.3.6) leaves no empty token. A pattern rejection is
// item 7 only where an item with another lexical is held, which a pattern may
// match (§2.1's note, otherLexical); the empty list has no other lexical, nor
// a list none of whose items has one, so every lexical of such a value
// normalizes to the text written.
func (e *encoder) list(st, item *xsd.SimpleType, a atomic, v reflect.Value, path, subject string) (string, error) {
	isubject := itemSubject(subject, item)
	items := make([]string, v.Len())
	clean := true
	for i := range v.Len() {
		ipath := path + "[" + strconv.Itoa(i) + "]"
		text, ok, err := e.atom(item, a, v.Index(i), ipath, isubject)
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
	if a.formed() {
		// fmt spells a big.Rat, a big.Int or a datatype struct by its fields, and a
		// []byte by its octets in decimal; each item's text is its form (spelled).
		shown = "[" + text + "]"
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, st, text, nil, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && otherLexical(a, v) {
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
