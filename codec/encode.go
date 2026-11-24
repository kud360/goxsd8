package codec

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/icpath"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/internal/xmlchar"
	"github.com/kud360/goxsd8/internal/xmlname"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ErrInvalidValue is the sentinel errors.Is finds in the errors of items 6 and
// 8 of [Append], whose text is its cause's alone, never this sentinel's.
// Item 6's error wraps the value.ValidateLexical rejection in the last of item
// 6's cases, and the verdict of its primitive mapping's Parse in its case of
// an xs:QName or xs:NOTATION Local that Parse rejects, and carries whatever
// *xsderr.Error that rejection carries; in the others it carries none. Item
// 8's carries the violations it states.
var ErrInvalidValue = errors.New("codec: Append: a value no valid document holds")

// Append appends to dst the XML document that encodes v as the document
// element of the top-level element declaration name in schema, and returns
// the extended slice; dst may be nil. Read name from the header comment of the
// type codegen emits for that declaration, the header the "# Naming" section
// of codegen's package doc describes. It writes each simple value in its
// canonical representation (Datatypes §2.3.1 dt-canonical-representation),
// the form the Canonical of its governing mapping on backend
// (value.GoverningMapping) renders, which for an xs:precisionDecimal zero,
// whose canonical representation precisionDecimal §6 f-precDecCanmap does not
// decide, is that form alone, but for an xs:QName or xs:NOTATION value,
// which has none (§3.3.18, §3.3.19), and a value of xs:anySimpleType or
// xs:anyAtomicType itself, which has none either, canonical mappings being
// defined for primitive datatypes alone (§2.1's note, §2.4.2 dt-special),
// each written as the package doc's "# What encodes" states. It assesses the
// document it writes, the bytes past len(dst) alone, through package validate,
// on backend and under no URI, exactly when an element declaration item 4
// examines, the document element's or one a particle reaches at any depth, has
// {identity-constraint definitions} (Structures §3.3.4.3 cvc-elt clause 6,
// §3.11.4 cvc-identity-constraint), which holds for every value of T or for
// none, and runs no assessment otherwise: a caller wanting a verdict on such a
// document assesses it through package validate, as any document. The package
// doc's "# What encodes" section states the slice Append encodes and the
// document it writes. On error Append returns dst unextended, its bytes beyond
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
//  6. A value no valid document holds, the first in document order, an element's attributes
//     before its content: an error, which errors.Is reports as [ErrInvalidValue], naming the
//     Go path of its field from T and, for a simple value, its simple type. It is the slice
//     field of a particle with {max occurs} above 1 whose length lies outside its {min occurs}
//     and {max occurs}, a choice's alternative's {min occurs} 0 read as 1 in its branch,
//     though never a flattened sequence's or choice's slice field, which holds one item per
//     child, not per occurrence (codegen's package doc, "# Particles"); the nil sealed interface
//     of a content choice, or of an item of an occurrence field's slice of a choice, none of whose
//     alternatives is emptiable (Structures §3.8.4.3 cvc-model-group, §3.9.6.3 cos-group-emptiable), a
//     choice of no {particles} among them; a value of that interface whose dynamic type the
//     [WithBranches] set in opts places at no alternative, a pointer to a branch among them, which
//     Decode never returns; a nil item of a flattened sequence's or choice's slice, which Decode never
//     returns either, and an item of it whose dynamic type the [WithBranches] set in opts places at no
//     alternative, a pointer to a branch among them; the nil sealed interface of an <element ref>
//     particle's field over a substitution group (codegen's package doc, "# Substitution groups"), or of
//     an item of its slice, wherever nil does not mean absent, that of a head with {abstract} true and
//     no branch among them, and a value of that interface whose dynamic type the [WithBranches] set in
//     opts places at no branch, a pointer to a branch among them, which Decode never returns; the nil
//     sealed interface of a union simple type wherever nil does not mean absent, that is, every field
//     but an element particle's of {min occurs} 0 and {max occurs} 1 and an optional attribute use's
//     (codegen's package doc, "# Particles", "# Attributes"); a value of that interface whose dynamic
//     type the [WithBranches] set in opts places at no basic member, a pointer to a variant among them,
//     which Decode never returns; a value that holds itself, a struct or sealed interface value its own
//     write reaches again through a *S, []S or []I field, which no finite document holds, the error
//     naming the Go path where the value is reached again and nothing written beneath it; a string
//     holding bytes that are not UTF-8 or a character the XML 1.0 Char production does not match, U+0000
//     among them (Datatypes §3.3.1.1, §3.3.17.1, §3.2.1.2, §3.2.2.2), a held element's Space, an
//     attribute's Space or Value, a Text and a Tail among them; a string that differs from its own
//     whiteSpace-normalized form under its simple type, such as an xs:token holding "  a b"; a list
//     item, a string, that is empty or holds TAB, LF, CR or a space, which no list item's lexical
//     representation does (Datatypes §2.4.1.2); a big.Rat outside the decimal value space, whose
//     denominator in lowest terms is no 2^m·5^n (§3.3.3.1), such as 1/3; an element a wildcard
//     particle's field holds, of {process contents} skip or lax, that its content model admits at no
//     particle where it is written, a name outside the wildcard's {namespace constraint} or in its
//     {disallowed names} among them, or attributes to a particle other than that wildcard, an element
//     particle taking its name first among them, or after whose content's last child the content model
//     is not accepting, as xsd.Schema.ContentMatcher's Matcher decides (§3.9.4.3 cvc-accept, §3.10.4.1
//     cvc-wildcard, §3.8.4.3 cvc-model-group); in such an element, and in the content of an element of
//     type xs:anyType itself, its own attributes among it, at any depth, a Local that is no NCName or a
//     Space that is the namespace name the prefix xmlns is bound to, of the element or of an attribute,
//     an attribute of absent Space named xmlns (Namespaces in XML §3), and an attribute of the expanded
//     name of one before it (§6.3 nsc-AttrsUnique); an element an item of a flattened sequence's or
//     choice's slice holds that its content model admits at no particle where it is written, or
//     attributes to a basic particle other than the one whose branch the item is, or a slice after whose
//     last item the content model is not accepting, as that Matcher decides (§3.9.4.3 cvc-accept,
//     §3.10.4.1 cvc-wildcard, §3.8.4.3 cvc-model-group); a date, time or duration datatype struct value
//     no value of its datatype maps to, as codegen's package doc "# Simple types" states such values
//     exist, the error naming the first property found outside it, its fields read in order, or, ahead
//     of its Seconds, a duration's Months and Seconds of opposite signs; a PrecisionDecimal datatype
//     struct value no value maps to, as codegen's "# Simple types" states such values exist, a nil Value
//     or the numerical arm with a negative Coefficient, or a Value whose dynamic type the
//     [WithPrecisionDecimalArms] set in opts supplies in no role, a pointer to an arm among them, which
//     Decode never returns; an xs:QName or xs:NOTATION value whose namespace name holds a byte that
//     starts no XML 1.0 Char or is the one the prefix xmlns is bound to (Namespaces in XML §3), or whose
//     Local the Parse of its primitive's mapping rejects in Append's spelling, that Parse's verdict
//     wrapped, or makes a spelling that resolves back to another expanded name; or a value whose simple
//     type rejects it, a value.ValidateLexical error value.IsDatatypeVerdict reports as a verdict, a
//     facet's rejection or a backend Parse's among them, that error wrapped, but for item 7's pattern
//     and union-variant cases, a list's own facets read once every item has passed its own checks.
//  7. A value with no canonical representation or no spelling Append writes,
//     the first in document order as item 6 counts: its *DeclineError. That is
//     a governing mapping with no Canonical, a Canonical that fails for the
//     value, such as strict's for the zero xs:yearMonthDuration (Datatypes
//     §3.4.26.1's note), or a value with a lexical other than the one Append
//     writes, or a list holding at least one such item, whose canonical
//     representation, or for an xs:QName or xs:NOTATION value its spelling, a
//     pattern facet of its type rejects (§2.1's note), such as 3/2 under an
//     xs:decimal pattern \d+\.\d\d, which admits 1.50 but not 1.5, whatever
//     other facet rejects it; a canonical representation that does not map back
//     to the value; a union's variant whose member value Append writes as text
//     whose ·active basic member· (Datatypes §2.4.1.3 dt-active-basic-member,
//     §4.1.4 cvc-datatype-valid clause 2.3) is not that variant's member, or
//     that no member of the union accepts, so that Decode returns another
//     variant or none, by the construct of the clause before, or for an
//     xs:QName or xs:NOTATION member's spelling by its spelling's construct:
//     xs:decimal 1 under a union of xs:int and xs:decimal is written 1, which
//     Decode returns as the xs:int variant, though 1.0 decodes into the
//     xs:decimal variant holding 1, so that a valid document holds the value
//     and item 6 does not apply, whatever else rejects it; or an xs:QName or
//     xs:NOTATION value whose namespace name is absent, written on an element
//     whose namespace is not, which no unprefixed spelling resolves to there
//     and no prefix can be bound to (Namespaces in XML §5 nsc-NoPrefixUndecl),
//     whatever else rejects it; or an attribute in the XSI namespace on an
//     element a wildcard particle's field holds, or in the content of an
//     element of type xs:anyType itself, its own attributes among it, at any
//     depth, whatever its value, since an xsi:type's may hold a prefix no
//     binding the struct holds resolves; or, in such content of an xs:anyType
//     element, or in an element a lax wildcard particle's field holds, the
//     element itself among it, at any depth, an element whose expanded name a
//     top-level element declaration of schema names, or an attribute whose
//     expanded name a top-level attribute declaration of schema names,
//     whatever its content or value, since a lax wildcard, xs:anyType's among
//     them, assesses it against that declaration (Structures §3.4.7, §3.3.4.6
//     key-governing-ed clause 3, §3.2.4.2 key-governing-ad clause 3); or an
//     element a lax wildcard particle's field holds that has a ·locally
//     declared type· in the complex type whose content it is written into, an
//     element declaration of its expanded name that type's content model, or
//     that of a type on its {base type definition} chain, contains, whatever
//     its content, since that type governs it (§3.3.4.6 key-governing-type-elem
//     clause 7, §3.4.6.4 key-ldt-elem). The values with a lexical other than
//     the one Append writes, the ones whose rejection by a pattern facet is
//     this item, a union's own pattern facet among them, read once the
//     variant's member is its text's ·active basic member·, are: a union's
//     value whose member value is one; every xs:boolean, xs:decimal-primitive,
//     xs:dateTime, xs:dateTimeStamp, xs:time and xs:duration-primitive value;
//     every xs:float, xs:double and xs:precisionDecimal value other than NaN
//     and -INF; an xs:date, xs:gYearMonth or xs:gYear value whose
//     timezoneOffset is 0, spelled Z, +00:00 or -00:00 (§D.2.2's note), or
//     whose year is 0, spelled 0000 or -0000; an xs:gMonthDay, xs:gDay or
//     xs:gMonth value whose timezoneOffset is 0; an xs:hexBinary value one of
//     whose octets holds a hex digit A to F (§3.3.15.2); every xs:base64Binary
//     value but the empty one (§3.3.16.2); an xs:QName or xs:NOTATION value
//     whose namespace name is not absent; and no xs:string- or
//     xs:anyURI-primitive value, each being its own whiteSpace-normalized form.
//     NaN and -INF, of xs:float, xs:double and xs:precisionDecimal alike, are
//     written NaN and -INF, their only lexicals (§3.3.4.2, §3.3.5.2,
//     precisionDecimal §3.2), so a pattern facet of their own simple type that
//     rejects one of them is item 6, not this item.
//  8. An assessed document validate charges: the error item 3 of [Decode]'s
//     doc states for it, which errors.Is also reports as [ErrInvalidValue],
//     each violation charged to the written document, a
//     cvc-identity-constraint one naming its constraint and the clause it
//     breaks, 3, 4.1, 4.2.1, 4.2.2 or 4.3. A violation of any other rule is a
//     fault of Append's, reported so all the same.
//  9. An assessed document validate leaves undecided: the error item 4 of
//     [Decode]'s doc states for it.
//
// A plain error is no *DeclineError and wraps none of [ErrTypeMismatch],
// [ErrInvalidValue] and errors.ErrUnsupported; whether it wraps an
// *xsderr.Error does not tell its item. A source fault validate/xmlsrc.Validate
// reports in the document an assessment reads, in either channel it carries
// one in, is a plain error in place of items 8 and 9.
//
// A fault of backend found at a value, in place of that value's item 6 or 7, is
// a plain error too, returned at once and naming the value's Go path and simple
// type: no mapping of the value's primitive, no governing mapping, a Parse that
// rejects the spelling Append gives the value, the mapping's of the type's
// {primitive type definition} for a value of none of xs:string, xs:anyURI,
// xs:float, xs:double, xs:QName, xs:NOTATION and xs:precisionDecimal
// primitive, the governing mapping's for an xs:float, xs:double or
// xs:precisionDecimal one and, for an xs:QName or xs:NOTATION one, its
// primitive's mapping's with an error value.IsDatatypeVerdict does not report
// as a verdict, a Parse of the governing mapping that rejects the
// whiteSpace-normalized canonical representation an xs:float, xs:double or
// xs:precisionDecimal value's Canonical renders, a Canonical that renders a
// value of none of xs:string, xs:anyURI, xs:float, xs:double, xs:QName,
// xs:NOTATION and xs:precisionDecimal primitive as other text than
// its canonical representation (Datatypes §E.1 f-decimalCanmap, §E.4
// f-booleanCanmap, §E.2 f-durationCanMap, §E.3.5 vp-dateTimeCanRep,
// vp-timeCanRep, vp-dateCanRep, vp-gYearMonthCanRep, vp-gYearCanRep,
// vp-gMonthDayCanRep, vp-gDayCanRep and vp-gMonthCanRep, §E.4.1
// f-hexBinaryCanonical and §3.3.16.2's Canonical-base64Binary), or an error
// of value.ValidateLexical or, identifying a union value's ·active basic
// member·, of value.ValidatingType that value.IsDatatypeVerdict does not
// report as a verdict.
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
	cfg := newConfig(opts)
	decl, f, assess, err := encodeStatic(schema, name, t, cfg)
	if err != nil {
		return dst, err
	}
	e := &encoder{schema: schema, backend: backend, decimals: cfg.decimals, branches: cfg.branches, out: dst}
	// v's own Value, of Kind Interface where T is one, so that a nil T and a
	// branch's dynamic type both reach the write as T holds them.
	if err := encoded[writer](f).write(e, reflect.ValueOf(&v).Elem(), writeAt{el: decl, path: typePath(t)}); err != nil {
		return dst, err
	}
	if e.invalid != nil {
		return dst, e.invalid
	}
	if e.declined != nil {
		return dst, e.declined
	}
	if !assess {
		return e.out, nil
	}
	if err := assessWritten(schema, backend, e.out[len(dst):]); err != nil {
		return dst, err
	}
	return e.out, nil
}

// assessWritten returns the error items 8 and 9 of [Append]'s doc state for
// doc, the document Append wrote, assessed through package validate on
// backend under no URI, or a plain error: validate.New's, or a source fault
// in either channel validate/xmlsrc.Validate carries one in.
func assessWritten(schema *xsd.Schema, backend value.Backend, doc []byte) error {
	v, err := validate.New(schema, backend)
	if err != nil {
		return err
	}
	res, err := xmlsrc.Validate(v, bytes.NewReader(doc))
	if err != nil {
		return err
	}
	if res.Err() == nil && res.Standing() == validate.Charged {
		return &invalidValueError{&chargedError{violations: res.Violations()}}
	}
	return verdict(res)
}

// CheckAppend returns the error of the first of items 2 to 5 of [Append]'s
// doc that applies to a value of T encoded as the document element of the
// top-level element declaration name in schema under opts, in the class
// that item states, or nil when none applies; for a nil schema it returns a
// plain error. Item 4's slice is the one the package doc's "# What encodes"
// states. Items 6 to 9 and a fault of the backend depend on the value, so
// they lie outside CheckAppend's answer, and nil leaves them open. The
// backend is no input, so a nil backend, which Append's item 1 refuses, lies
// outside it too. For a non-nil schema, a non-nil error is the one Append
// returns, of equal Error text and class, for every value of T, every non-nil
// backend and the same opts.
func CheckAppend[T any](schema *xsd.Schema, name xsd.QName, opts ...Option) error {
	if schema == nil {
		return errors.New("codec: CheckAppend: nil *xsd.Schema")
	}
	_, _, _, err := encodeStatic(schema, name, reflect.TypeFor[T](), newConfig(opts))
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
// non-nil schema, the filler item 5's check returns for it under cfg, opts
// resolved once by its caller, Decode's for a T of Go type t, and whether
// Append assesses what it writes (elementEncodes), or the error of the first
// of items 2 to 5 of [Append]'s doc that applies. It reads no value, so its
// answer holds for every value of t. Item 4's pass (elementEncodes) walks the
// schema and returns before bindDecl is called, so that its declines come
// first; it builds nothing, and Append writes from the filler alone (writer).
func encodeStatic(schema *xsd.Schema, name xsd.QName, t reflect.Type, cfg config) (xsd.ElementDeclaration, filler, bool, error) {
	decl, ok := schema.Element(name)
	if !ok {
		return xsd.ElementDeclaration{}, nil, false, fmt.Errorf("codec: no top-level element declaration has the expanded name %q", name.Display())
	}
	// An answer about the schema, no violation of an instance, so it carries
	// no xsderr rule (STYLE E2).
	if decl.Abstract() {
		return xsd.ElementDeclaration{}, nil, false, fmt.Errorf("codec: %s has {abstract} true, so no document element validates against it (cvc-elt clause 2)", declSubject(decl))
	}
	assess, err := elementEncodes(schema, decl)
	if err != nil {
		return xsd.ElementDeclaration{}, nil, false, err
	}
	b, err := bindDecl(&pipeline{schema: schema}, decl)
	if err != nil {
		return xsd.ElementDeclaration{}, nil, false, err
	}
	f, err := b.check(t, newCheckContext(cfg))
	if err != nil {
		return xsd.ElementDeclaration{}, nil, false, namedMismatch(decl, err)
	}
	return decl, f, assess, nil
}

// A writer is a bound arm of the filler item 5's check returns that Append
// writes a value from: write appends to e.out the XML encoding v, a value of
// the Go type the arm is bound to, at at, and returns a backend fault. Its
// implementers are exactly the arms the slice the package doc's "# What
// encodes" states reaches: the root arms, *simpleRoot, *boundBinding, *boundChoiceRoot and
// rawContent, writing the element at.el; and the content arms, *boundSequence,
// *boundContentChoice, boundOccurrence and boundFlattened, writing at.el's content, each
// particle's elements by its member (particleWriter). Append reaches each through encoded.
type writer interface {
	write(e *encoder, v reflect.Value, at writeAt) error
}

// A texter is the bound simple content of a complex type, which Append
// writes as character content rather than as elements: text returns the text
// Append writes for v, the struct at.el's element holds, at at, and a backend
// fault. *boundBinding's write reads it before the start tag's '>', so that a
// prefix an xs:QName or xs:NOTATION value's spelling declares lands on that
// start tag. Its one implementer is boundValue.
type texter interface {
	text(e *encoder, v reflect.Value, at writeAt) (string, error)
}

// writeAt is what a writer, a texter or a particleWriter reads beside its
// value: el, the element declaration of the element the value is written as,
// for a root arm, or of the element whose content holds it, for every other
// writer, a texter and a particleWriter; typ, for a content arm, a texter or a
// particleWriter, the complex type whose content it writes, el's, which a
// choice's item 6 errors and a texter's value name (typeSubject) and a lax
// wildcard's match reads its locally declared type in
// (wildcardMember.writeElements), and the zero ComplexType for every other
// writer, which reads none; and path, the value's Go path from T.
type writeAt struct {
	el   xsd.ElementDeclaration
	typ  xsd.ComplexType
	path string
}

// attributionSlot is the attributionCheck of the content a particle's
// elements are written into, nil for none, and the particle's attribution
// target there (childIndex).
type attributionSlot struct {
	check  *attributionCheck
	target int
}

// take puts the element of decl, written at path, to s.check at s.target
// (attributionCheck.take), by decl's own expanded name.
func (s attributionSlot) take(e *encoder, decl xsd.ElementDeclaration, path string) {
	s.check.take(e, decl.Name(), s.target, path)
}

// encoded returns x, an arm of the filler item 5's check returns or of the
// binder under it, as C, what Append's write reads it as: the writer capability, the
// texter capability of simple content's contentFiller, the elementWriter capability of
// a fieldSink's elementReader, the particleWriter capability of a contentMember's or a
// branchArm's elementMember or wildcardMember, or a closed sum's arm, a sink's
// fieldSink, a fieldArm's scalarArm or a scalarElem's atomic. It is the write's one
// assertion of the binder's arms. Invariant: where item 4 of [Append]'s doc admits a
// construct, every arm the binder builds for it that the write reaches is such a C. A
// failed assertion breaks it, an internal bug, never input, and panics naming x's type.
func encoded[C any](x any) C {
	c, ok := x.(C)
	if !ok {
		panic(fmt.Sprintf("codec: Append reached a %T, which is no %v: item 4 of Append's doc admitted a construct whose bound arm Append does not write", x, reflect.TypeFor[C]()))
	}
	return c
}

// write writes at.el, the document element, holding v, a value of s's simple
// type (encoder.simpleElement); only the document element's type is a
// simpleRoot.
func (s *simpleRoot) write(e *encoder, v reflect.Value, at writeAt) error {
	return e.simpleElement(at.el, s.scalar, v, at.path)
}

// write writes the element at.el holding v, a value of b's struct: its start
// tag, then each attribute field's attribute, in field order, the fields after
// the content's (binding.lead, elementContent.width), a nil *V's written as
// none, and so is a nil union interface of a use whose {required} is false,
// then the content: for simple content the value field's text (texter),
// computed before the start tag's '>', and otherwise the content's elements
// (b.content's write), an element with no content written as an empty-element
// tag; or nothing, where v holds itself (encoder.enter).
func (b *boundBinding) write(e *encoder, v reflect.Value, at writeAt) error {
	if !e.enter(v, at.path) {
		return nil
	}
	defer e.leave()
	bd := b.binding
	name := at.el.Name()
	outer, mark := e.startTag(name)
	first := bd.lead() + bd.content.width()
	for i, a := range bd.attrs {
		f, fpath := v.Field(first+i), at.path+"."+v.Type().Field(first+i).Name
		if a.shape == shapeOptional {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		}
		// An optional union use's field is its bare sealed interface, nil
		// meaning absent (codegen's package doc, "# Attributes").
		if f.Kind() == reflect.Interface && f.IsNil() && !a.required {
			continue
		}
		text, err := e.simple(a.scalar, f, fpath, attributeSubject(a.decl))
		if err != nil {
			return err
		}
		e.out = append(e.out, ' ')
		e.out = append(e.out, a.decl.Name().Local...)
		e.out = append(e.out, `="`...)
		e.out = appendEscaped(e.out, text, true)
		e.out = append(e.out, '"')
	}
	content := writeAt{el: at.el, typ: bd.typ, path: at.path}
	// {content type} is a closed sum, so the switch is STYLE T2's sealed-sum
	// exception.
	switch bd.typ.ContentType().(type) {
	case xsd.SimpleContent:
		text, err := encoded[texter](b.content).text(e, v, content)
		if err != nil {
			return err
		}
		e.textTag(name.Local, text, outer, mark)
		return nil
	}
	start := len(e.out)
	e.out = append(e.out, '>')
	if err := encoded[writer](b.content).write(e, v, content); err != nil {
		return err
	}
	e.endTag(name.Local, start, outer, mark)
	return nil
}

// text returns the text Append writes for v's value field, field b.index of
// its struct, under the field's scalar (encoder.scalarText), the scalar
// Decode's simpleValueOf bound, of {content type}.{simple type definition}, so
// that its facets, an anonymous restriction's own among them, reach the value
// through value.ValidateLexical; the field's item 6 and 7 errors of [Append]'s
// doc name that type after at.typ as simpleValueOf names it (contentSubject).
// A union's nil interface is item 6 there: a value field is never absent.
func (b boundValue) text(e *encoder, v reflect.Value, at writeAt) (string, error) {
	s := encoded[scalarArm](b.value.value.arm).scalar
	path := at.path + "." + v.Type().Field(b.index).Name
	w, err := e.scalarText(s, v.Field(b.index), path, contentSubject(typeSubject(at.el, at.typ), s.st))
	return w.text, err
}

// write writes the element at.el holding v, a value of the choice's sealed
// interface: its start tag, then the branch v holds (writeChoice), an empty
// match written as an empty-element tag; or nothing, where v holds itself
// (encoder.enter).
func (c *boundChoiceRoot) write(e *encoder, v reflect.Value, at writeAt) error {
	if !e.enter(v, at.path) {
		return nil
	}
	defer e.leave()
	name := at.el.Name()
	outer, mark := e.startTag(name)
	start := len(e.out)
	e.out = append(e.out, '>')
	check := e.attribution(c.root.index, at.el, c.root.typ)
	if err := writeChoice(e, c.root.sealed.alts, c.choice, v, writeAt{el: at.el, typ: c.root.typ, path: at.path}, check); err != nil {
		return err
	}
	check.end(e, at.path)
	e.endTag(name.Local, start, outer, mark)
	return nil
}

// write writes the elements of each of b's members from its field, member i's
// field b.first+i of v, read by b.sinks[i] (writeMembers), under the content's
// attributionCheck (encoder.attribution).
func (b *boundSequence) write(e *encoder, v reflect.Value, at writeAt) error {
	check := e.attribution(b.sequence.index, at.el, at.typ)
	if err := writeMembers(e, b.sequence, b.sinks, v, b.first, at, check); err != nil {
		return err
	}
	check.end(e, at.path)
	return nil
}

// writeMembers writes the elements of each of s's members from its field of
// v, member i's field first+i, read by sinks[i], in member order, each taken
// by check: each member is an element or wildcard particle's field, which
// takes one attribution target (field.targets), so member i's sink is
// sinks[i] and its target s.base+i.
func writeMembers(e *encoder, s *sequence, sinks []sink, v reflect.Value, first int, at writeAt, check *attributionCheck) error {
	for i, m := range s.members {
		j := first + i
		if err := encoded[particleWriter](m).writeElements(e, sinks[i], v.Field(j), writeAt{el: at.el, typ: at.typ, path: at.path + "." + v.Type().Field(j).Name}, attributionSlot{check: check, target: s.base + i}); err != nil {
			return err
		}
	}
	return nil
}

// write writes the branch the choice's sealed interface holds, field b.index
// of v (writeChoice), then the elements of each member of the rest, whose
// fields follow it, member i read by b.rest[i] (writeMembers), all under the
// content's attributionCheck (encoder.attribution).
func (b *boundContentChoice) write(e *encoder, v reflect.Value, at writeAt) error {
	rest := b.content.rest
	check := e.attribution(rest.index, at.el, at.typ)
	if err := writeChoice(e, b.content.choice.sealed.alts, b.choice, v.Field(b.index), writeAt{el: at.el, typ: at.typ, path: at.path + "." + v.Type().Field(b.index).Name}, check); err != nil {
		return err
	}
	if err := writeMembers(e, rest, b.rest, v, b.index+1, at, check); err != nil {
		return err
	}
	check.end(e, at.path)
	return nil
}

// write writes the occurrence field of v, field b.index, as writeOccurring
// states, the field's shape and its particle's occurrence read from the
// occurrence (occurrence.shape, occurrence.own): each item it holds, in slice
// order, as its model group's particles in order (boundItem.writeItem), all
// under one attributionCheck of the content (encoder.attribution), since the
// content model's Matcher spans every occurrence.
func (b boundOccurrence) write(e *encoder, v reflect.Value, at writeAt) error {
	o := b.occurrence
	check := e.attribution(o.index, at.el, at.typ)
	subject := fmt.Sprintf("the %s content particle of %s", o.group.Compositor(), typeSubject(at.el, at.typ))
	field := writeAt{el: at.el, typ: at.typ, path: at.path + "." + v.Type().Field(b.index).Name}
	if err := e.writeOccurring(o.shape, o.own, o.own, subject, v.Field(b.index), field, func(item reflect.Value, path string) error {
		return b.bound.writeItem(e, item, writeAt{el: at.el, typ: at.typ, path: path}, check)
	}); err != nil {
		return err
	}
	check.end(e, at.path)
	return nil
}

// writeItem writes the elements of each member of the sequence or all group
// from its field of v, the item's struct, member i's field i, read by
// b.fields[i] (writeMembers).
func (b boundSequenceItem) writeItem(e *encoder, v reflect.Value, at writeAt, check *attributionCheck) error {
	return writeMembers(e, b.sequence, b.fields, v, 0, at, check)
}

// writeItem writes the branch v, a value of the choice's sealed interface,
// holds (writeChoice).
func (c boundChoiceItem) writeItem(e *encoder, v reflect.Value, at writeAt, check *attributionCheck) error {
	return writeChoice(e, c.choice.sealed.alts, c.bound, v, at, check)
}

// writeChoice writes the branch v, a value of the sealed interface of a
// choice occurring once or of one occurrence of a choice held in an
// occurrence field, whose alternatives are alts, holds, bc that interface
// as item 5's check bound it: its branches[j], the branch type placed at
// alts[j] (sealed.checkSite), is matched to v's dynamic type, and the members
// of alts[j] are written from the branch struct's fields, in field order,
// member i read by branches[j].sinks[i] as boundSequence's write pairs them,
// its target the targets of every earlier alternative (branchTargets) plus i,
// under check, the attributionCheck of the content the choice is written
// into, nil for none. The nil interface writes nothing, the empty match, but is
// item 6 of [Append]'s doc where no alternative is emptiable
// (choiceEmptiable), and so is a dynamic type bc places at no alternative, a
// pointer to a branch among them; at.typ is the complex type whose content the
// choice is.
func writeChoice(e *encoder, alts []branchArm, bc *boundChoice, v reflect.Value, at writeAt, check *attributionCheck) error {
	if v.IsNil() {
		if !choiceEmptiable(alts) {
			e.invalidate(fmt.Errorf("codec: Append: %s is nil, the empty match, which the content choice of %s does not accept: %s", at.path, typeSubject(at.el, at.typ), unemptiable(alts)))
		}
		return nil
	}
	b := v.Elem()
	j := bc.branchOf(b.Type())
	if j < 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, which the WithBranches set of %s places at no alternative of the content choice of %s, so that Decode returns no such value", at.path, b.Type(), v.Type(), typeSubject(at.el, at.typ)))
		return nil
	}
	first := 0
	for _, a := range alts[:j] {
		first += branchTargets(a)
	}
	for i, m := range alts[j].branchMembers() {
		if err := encoded[particleWriter](m).writeElements(e, bc.branches[j].sinks[i], b.Field(i), writeAt{el: at.el, typ: at.typ, path: at.path + ".(" + typePath(b.Type()) + ")." + b.Type().Field(i).Name}, attributionSlot{check: check, target: first + i}); err != nil {
			return err
		}
	}
	return nil
}

// branchOf returns the index in bc.branches of the branch whose type is t, the
// dynamic type of a value of bc's sealed interface, and -1 where the
// WithBranches set placed none at an alternative, a pointer to a branch among
// them: the alternative Decode would have returned the value from.
func (bc *boundChoice) branchOf(t reflect.Type) int {
	return slices.IndexFunc(bc.branches, func(bb boundBranch) bool { return bb.t == t })
}

// write writes the slice field of v, field b.index, which holds at.typ's
// flattened content {particle}, one element per item in slice order, all
// under one attributionCheck of the content (flattenedCheck), never nil, since
// the order of a flattened slice's items is the caller's: each item's branch,
// matched to its dynamic type (boundChoice.branchOf), writes its
// alternative's one field (particleWriter.writeElements), taken at the
// alternative's index, the one its place in b.flattened.places holds. It
// records item 6 of [Append]'s doc for the slice: a nil item and an item of a
// dynamic type placed at no alternative here, and the content model's verdict
// on each item and on the slice's end through attributionCheck.take and
// attributionCheck.end; the slice's length it never reads, since no
// writeOccurring runs over the slice. b.open is never written: item 4 declines
// {open content} (contentEncodes).
func (b boundFlattened) write(e *encoder, v reflect.Value, at writeAt) error {
	alts := b.flattened.slice.sealed.alts
	check := e.flattenedCheck(b.flattened, at)
	slice := v.Field(b.index)
	path := at.path + "." + v.Type().Field(b.index).Name
	for i := range slice.Len() {
		item, ipath := slice.Index(i), path+"["+strconv.Itoa(i)+"]"
		if item.IsNil() {
			e.invalidate(fmt.Errorf("codec: Append: %s is nil, an item of the flattened content of %s that holds no child element, so that Decode returns no such item", ipath, check.subject))
			continue
		}
		br := item.Elem()
		j := b.bound.branchOf(br.Type())
		if j < 0 {
			e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, which the WithBranches set of %s places at no alternative of the flattened content of %s, so that Decode returns no such value", ipath, br.Type(), item.Type(), check.subject))
			continue
		}
		field := writeAt{el: at.el, typ: at.typ, path: ipath + ".(" + typePath(br.Type()) + ")." + br.Type().Field(0).Name}
		if err := alts[j].writeElements(e, b.bound.branches[j].sinks[0], br.Field(0), field, attributionSlot{check: check, target: j}); err != nil {
			return err
		}
	}
	check.end(e, at.path)
	return nil
}

// choiceEmptiable reports whether one occurrence of the choice whose
// alternatives are alts, each an element or wildcard particle, matches the
// empty sequence: one of them has {min occurs} 0 (§3.9.6.3
// cos-group-emptiable).
func choiceEmptiable(alts []branchArm) bool {
	return slices.ContainsFunc(alts, func(a branchArm) bool { return encoded[particleWriter](a).occurs().Min() == 0 })
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

// A particleWriter is the member of an element or wildcard particle Append
// writes, a contentMember's or a branchArm's: writeElements writes the
// elements of f, the field of its particle, read by s, the field's sink, at
// at, each taken by slot's attributionCheck at slot's target, and returns a
// backend fault; occurs is the particle's own {min occurs}
// and {max occurs}. Its implementers are elementMember and wildcardMember.
type particleWriter interface {
	writeElements(e *encoder, s sink, f reflect.Value, at writeAt, slot attributionSlot) error
	occurs() xsd.Occurs
}

// occurs is m.own.
func (m elementMember) occurs() xsd.Occurs { return m.own }

// occurs is m.own.
func (m wildcardMember) occurs() xsd.Occurs { return m.own }

// writeElements writes the elements of f, the field of m's particle, as
// writeOccurring states, each by the elementWriter s, the field's sink, reads
// it with, as an element of m.decl or, for a groupReader, of the member its
// value holds, taken by slot.
func (m elementMember) writeElements(e *encoder, s sink, f reflect.Value, at writeAt, slot attributionSlot) error {
	w := encoded[elementWriter](encoded[fieldSink](s).read)
	return e.writeOccurring(m.shape, m.own, m.held, "the particle of "+declSubject(m.decl), f, at, func(v reflect.Value, path string) error {
		return w.writeElement(e, m.decl, v, path, slot)
	})
}

// writeElements writes the elements of f, the field of m's particle, as
// writeOccurring states, each a value of AnyElement's shape written raw
// (encoder.rawElement) in the hold of m.process, skipMatch for skip and
// laxMatch for lax, its name taken by slot.check (attributionCheck.take) once
// rawName admits it, then held to its hold's governing declaration
// (encoder.heldElement) and, for lax, to its locally declared type in at.typ
// (encoder.locallyTyped), each a decline item 7 of [Append]'s doc states.
// Item 4 declines a strict wildcard particle (particleEncodes), so that any
// other m.process is an internal bug, which panics, as encoded's assertion
// does.
func (m wildcardMember) writeElements(e *encoder, _ sink, f reflect.Value, at writeAt, slot attributionSlot) error {
	var hold rawHold
	// {process contents} is a closed set, so the switch is STYLE T2's
	// sealed-sum exception.
	switch m.process {
	case xsd.ProcessSkip:
		hold = skipMatch(m.subject)
	case xsd.ProcessLax:
		hold = laxMatch(m.subject)
	default:
		panic(fmt.Sprintf("codec: Append reached %s, of {process contents} %s, which item 4 of Append's doc declines", m.subject, m.process))
	}
	return e.writeOccurring(m.shape, m.own, m.held, m.subject, f, at, func(v reflect.Value, path string) error {
		name, ok := e.rawName(v, path)
		if !ok {
			return nil
		}
		slot.check.take(e, name, slot.target, path)
		e.heldElement(name, path, hold)
		if m.process == xsd.ProcessLax {
			e.locallyTyped(name, path, hold, at)
		}
		e.rawElement(name, v.Field(2), path, hold)
		return nil
	})
}

// writeOccurring writes the elements of f, the field of the particle subject
// names, occurring as own, at.path its Go path, each by write, by sh, the
// shape item 5 bound the field in (field.shape), computed at held, never by
// f's Kind, since a list's V is itself a slice: one per item of a []V, which
// is item 6 of [Append]'s doc where its length lies outside held, the error
// naming own where a choice's alternative raised it, none for a nil *V, and
// one for a V, but none for a nil V where held's {min occurs} is 0, which
// shape V takes only over a sealed interface (goshape.ParticleShape), nil
// meaning absent.
func (e *encoder) writeOccurring(sh shape, own, held xsd.Occurs, subject string, f reflect.Value, at writeAt, write func(v reflect.Value, path string) error) error {
	switch sh {
	case shapeOptional:
		if f.IsNil() {
			return nil
		}
		return write(f.Elem(), at.path)
	case shapeRepeated:
		if !held.Permits(f.Len()) {
			raised := ""
			if held != own {
				raised = fmt.Sprintf(", the occurrence %s with {min occurs} 0 read as 1 in its branch,", own)
			}
			e.invalidate(fmt.Errorf("codec: Append: %s holds %d items, outside the occurrence %s%s of %s", at.path, f.Len(), held, raised, subject))
		}
		for j := range f.Len() {
			if err := write(f.Index(j), at.path+"["+strconv.Itoa(j)+"]"); err != nil {
				return err
			}
		}
		return nil
	default:
		if held.Min() == 0 && f.IsNil() {
			return nil
		}
		return write(f, at.path)
	}
}

// An elementWriter is the elementReader of an element particle's field whose
// element Append writes: writeElement appends the element of decl holding v,
// a V the reader reads, whose Go path from T is path, taken first by slot
// (attributionSlot.take), and returns a backend fault. Its implementers are
// scalarArm, of a simple type, nestedReader, of a complex type, a structArm's
// or an ifaceArm's, rawContent, of xs:anyType itself, an anyTypeArm's, and
// groupReader, of a substitution group's sealed interface, a groupArm's,
// which writes the element of the member its value holds in decl's place.
type elementWriter interface {
	writeElement(e *encoder, decl xsd.ElementDeclaration, v reflect.Value, path string, slot attributionSlot) error
}

// writeElement writes the element of decl holding v, a value of a's simple
// type (encoder.simpleElement), taken by slot.
func (a scalarArm) writeElement(e *encoder, decl xsd.ElementDeclaration, v reflect.Value, path string, slot attributionSlot) error {
	slot.take(e, decl, path)
	return e.simpleElement(decl, a.scalar, v, path)
}

// writeElement writes the element of decl holding v, a value of n.t, by the
// writer n.fill is, taken by slot.
func (n nestedReader) writeElement(e *encoder, decl xsd.ElementDeclaration, v reflect.Value, path string, slot attributionSlot) error {
	slot.take(e, decl, path)
	return encoded[writer](n.fill).write(e, v, writeAt{el: decl, path: path})
}

// writeElement writes v, a value of g.t, the sealed interface over the
// substitution group of head, the declaration an <element ref> particle
// names: the branch v holds, matched to its dynamic type
// (boundChoice.branchOf), as an element of the member that branch is given
// for, g.members at the branch's index, under the member's own expanded name
// (§3.9.4.3 cvc-accept clause 2.3.2, §3.3.4.3 cvc-elt clause 1), from the
// branch's one field by the elementWriter its sink reads it with, taken by
// slot. A nil v, which writeOccurring passes only where nil does not mean
// absent, and a dynamic type the WithBranches set places at no branch, a
// pointer to a branch among them, are item 6 of [Append]'s doc, and write
// nothing: no element of an abstract head's own name is valid (cvc-elt clause
// 2), and Decode returns neither.
func (g groupReader) writeElement(e *encoder, head xsd.ElementDeclaration, v reflect.Value, path string, slot attributionSlot) error {
	if v.IsNil() {
		e.invalidate(fmt.Errorf("codec: Append: %s is nil, the sealed interface over the substitution group of %s, which holds no element where one is written", path, declSubject(head)))
		return nil
	}
	br := v.Elem()
	j := g.bound.branchOf(br.Type())
	if j < 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, which the WithBranches set of %s places at no branch of the substitution group of %s, so that Decode returns no such value", path, br.Type(), v.Type(), declSubject(head)))
		return nil
	}
	w := encoded[elementWriter](encoded[fieldSink](g.bound.branches[j].sinks[0]).read)
	return w.writeElement(e, g.members[j], br.Field(0), path+".("+typePath(br.Type())+")."+br.Type().Field(0).Name, slot)
}

// elementEncodes returns the error item 4 of [Append]'s doc states for the
// first construct of decl, the document element's declaration, outside the
// slice the package doc's "# What encodes" states, in the order the construct
// constants list, or a plain error for a component that does not resolve, and
// otherwise nil: encodeWalk.element's for decl. With nil it returns whether
// an element declaration the pass examined has {identity-constraint
// definitions}, which is when Append assesses what it writes (items 8 and 9).
// It reads the schema alone, never the binder, and builds nothing.
func elementEncodes(schema *xsd.Schema, decl xsd.ElementDeclaration) (bool, error) {
	w := &encodeWalk{schema: schema, walked: map[unitKey]bool{}}
	if err := w.element(decl); err != nil {
		return false, err
	}
	return w.identity, nil
}

// encodeWalk is one pass of elementEncodes: schema; walked, the complex types
// whose constructs it has examined or is examining, each under the key
// Decode's bind indexes its unit by (nestedKey); and identity, whether a
// declaration it examined has {identity-constraint definitions}
// (encodeWalk.declEncodes). An entry is stored before its type's content is
// examined, so that a later reach of the key, a recursive one included,
// examines nothing again: the first reach answers for the type, and every
// other component it reaches carries no key of its own only where it lies
// inside that type (bindRun). walked lives for one pass only and is a lookup
// index never iterated (STYLE D2); an entry left by a pass that declines is
// never read, since the decline ends the pass.
type encodeWalk struct {
	schema   *xsd.Schema
	walked   map[unitKey]bool
	identity bool
}

// element returns elementEncodes' error for decl, the document element's
// declaration or a particle's element declaration, examined at every reach:
// decl's own properties (encodeWalk.declEncodes), then its {type definition},
// which encodes as xs:anyType itself, its content written raw (rawContent's write),
// is examined as a simple type (simpleEncodes) or, for a complex type,
// declines by its {abstract} true, and else is examined by complexEncodes,
// unless walked already holds its key. complexEncodes and the functions under
// it read decl only to name a decline's subject, so that a type reached again
// through another declaration answers as it did.
func (w *encodeWalk) element(decl xsd.ElementDeclaration) error {
	if err := w.declEncodes(decl); err != nil {
		return err
	}
	td, ok := w.schema.ResolvedType(decl.TypeDefinition())
	if !ok {
		return fmt.Errorf("codec: the {type definition} of %s does not resolve", declSubject(decl))
	}
	if goshape.AnyTypeItself(td) {
		return nil
	}
	if st, isSimple := td.(*xsd.SimpleType); isSimple {
		return simpleEncodes(w.schema, st, declSubject(decl))
	}
	ct, _ := td.(xsd.ComplexType)
	if ct.Abstract() {
		return encodeDecline(constructAbstractType, typeSubject(decl, ct))
	}
	if key := nestedKey(decl, ct); key != nil {
		if w.walked[key] {
			return nil
		}
		w.walked[key] = true
	}
	return w.complexEncodes(decl, ct)
}

// declEncodes returns the decline of the first of decl's own properties the
// slice excludes: {nillable} as codegen reads it (goshape.Nillable), a {type
// table}, a {value constraint}, an identity constraint whose paths icpath
// does not compile (encodeWalk.identityEncodes), a member of its {substitution
// group affiliations} that names no top-level element declaration of the
// schema, the first in document order, an ·absent· member Finalize keeps
// (§5.3), under which validate assesses the element laxly, as if cvc-elt
// clause 1 had failed, and {abstract} true, which no element validates against
// (cvc-elt clause 2): the document element's is item 3 of [Append]'s doc,
// found before declEncodes runs, so that only a particle's declines so, one
// the parser never builds but xsd.NewElementDeclaration does. A decl whose
// {identity-constraint definitions} it admits sets w.identity.
func (w *encodeWalk) declEncodes(decl xsd.ElementDeclaration) error {
	if goshape.Nillable(decl) {
		return encodeDecline(constructNillable, declSubject(decl))
	}
	if _, has := decl.TypeTable(); has {
		return encodeDecline(constructTypeTable, declSubject(decl))
	}
	if _, has := decl.ValueConstraint(); has {
		return encodeDecline(constructValueConstraint, declSubject(decl))
	}
	for _, ic := range decl.IdentityConstraints() {
		if err := w.identityEncodes(decl, ic); err != nil {
			return err
		}
		w.identity = true
	}
	for _, head := range decl.SubstitutionGroupAffiliationNames() {
		if _, ok := w.schema.Element(head); !ok {
			return encodeDecline(constructAbsentAffiliation, fmt.Sprintf("%s, whose {substitution group affiliations} member %s names no top-level element declaration", declSubject(decl), head.Display()))
		}
	}
	if decl.Abstract() {
		return encodeDecline(constructAbstractElement, declSubject(decl))
	}
	return nil
}

// identityEncodes returns the decline by constructIdentityConstraint of ic, an
// identity constraint of decl, whose {selector} icpath.CompileSelector or one
// of whose {fields} icpath.CompileField does not compile, or, for a keyref,
// whose {referenced key}'s do not, the first in that order; a {referenced key}
// that names no identity constraint of the schema is a plain error.
func (w *encodeWalk) identityEncodes(decl xsd.ElementDeclaration, ic xsd.IdentityConstraint) error {
	subject := fmt.Sprintf("%s, its identity constraint %s", declSubject(decl), ic.Name().Display())
	if part, ok := pathsCompile(ic); !ok {
		return encodeDecline(constructIdentityConstraint, subject+"'s "+part)
	}
	ref, isKeyref := ic.ReferencedKeyName()
	if !isKeyref {
		return nil
	}
	all := w.schema.IdentityConstraints()
	i := slices.IndexFunc(all, func(c xsd.IdentityConstraint) bool { return c.Name() == ref })
	if i < 0 {
		return fmt.Errorf("codec: %s's {referenced key} %s does not resolve", subject, ref.Display())
	}
	if part, ok := pathsCompile(all[i]); !ok {
		return encodeDecline(constructIdentityConstraint, fmt.Sprintf("%s's {referenced key} %s's %s", subject, ref.Display(), part))
	}
	return nil
}

// pathsCompile reports whether icpath compiles ic's {selector}
// (icpath.CompileSelector) and each of its {fields} (icpath.CompileField),
// and otherwise names the first part it does not compile.
func pathsCompile(ic xsd.IdentityConstraint) (string, bool) {
	if _, ok := icpath.CompileSelector(ic.Selector()); !ok {
		return "{selector}", false
	}
	for i, f := range ic.Fields() {
		if _, ok := icpath.CompileField(f); !ok {
			return fmt.Sprintf("{fields} member %d", i+1), false
		}
	}
	return "", true
}

// complexEncodes returns elementEncodes' error for decl, whose {type
// definition} is the complex type ct: ct's content, for simple content its
// {simple type definition} as typeEncodes examines it, named after ct as
// Decode's simpleValueOf names it (contentSubject), and for element-only or
// mixed content its content {particle}, {max occurs} 0 declined before its
// hold is asked (#3794), then each particle in order, of a flattened one
// each basic particle it reaches (contentEncodes); then each attribute use
// codegen gives a field (goshape.FieldAttributeUses), then its {attribute
// wildcard}, then its {assertions}.
func (w *encodeWalk) complexEncodes(decl xsd.ElementDeclaration, ct xsd.ComplexType) error {
	subject := typeSubject(decl, ct)
	// {content type} is a closed sum, so the switch is STYLE T2's sealed-sum
	// exception.
	switch content := ct.ContentType().(type) {
	case xsd.SimpleContent:
		if err := typeEncodes(w.schema, content.SimpleType, contentSubject(subject, content.SimpleType)); err != nil {
			return err
		}
	case xsd.ElementContent:
		if err := w.contentEncodes(ct, content, subject); err != nil {
			return err
		}
	}
	for _, u := range goshape.FieldAttributeUses(ct) {
		if err := attributeEncodes(w.schema, u); err != nil {
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
// mixed content of ct, the complex type subject names: its {variety}, its
// {open content}, its {particle}'s {max occurs} 0, then its model group and
// hold as contentGroup, Decode's, answers them, contentGroup's errors returned
// as they are and its construct, which §3.4.2.3.3 leaves no schema to reach,
// as a plain error; then a model group codegen flattens as reachEncodes
// examines it; or each particle of a spliced sequence, of a choice
// occurring once that codegen does not flatten, or of a sequence or choice
// held in an occurrence field, as particleEncodes admits it, an all group
// spliced or held in an occurrence field declining before any particle is
// examined; then, for a model group holding a wildcard particle, whose
// children Append attributes through a Matcher (attributionCheck), ct's
// content model where xsd.Schema.ContentMatcher does not track it.
func (w *encodeWalk) contentEncodes(ct xsd.ComplexType, content xsd.ElementContent, subject string) error {
	if content.Mixed {
		return encodeDecline(constructMixed, subject)
	}
	if content.OpenContent != nil {
		return encodeDecline(constructOpenContent, subject)
	}
	if upper, bounded := content.Particle.Occurs().Max(); bounded && upper == 0 {
		return encodeDecline(constructVacuousParticle, "the content {particle} of "+subject)
	}
	group, hold, d, err := contentGroup(w.schema, content, subject)
	if err != nil {
		return err
	}
	if d != 0 {
		return fmt.Errorf("codec: the content {particle} of %s holds no model group", subject)
	}
	label := "particle %d of the content {particle} of %s"
	const alternative = "alternative %d of the content {particle} of %s"
	switch hold {
	case goshape.Flattened:
		return w.reachEncodes(ct, group, subject)
	case goshape.OnceChoice:
		label = alternative
	case goshape.Occurrence:
		if group.Compositor() == xsd.CompositorAll {
			return encodeDecline(constructAllGroup, subject)
		}
		if group.Compositor() == xsd.CompositorChoice {
			label = alternative
		}
	case goshape.Spliced:
		if group.Compositor() == xsd.CompositorAll {
			return encodeDecline(constructAllGroup, subject)
		}
	}
	for i, p := range group.Particles() {
		if err := w.particleEncodes(p, fmt.Sprintf(label, i+1, subject)); err != nil {
			return err
		}
	}
	if !slices.ContainsFunc(group.Particles(), wildcardParticle) {
		return nil
	}
	if _, tracked := w.schema.ContentMatcher(ct); !tracked {
		return encodeDecline(constructUntrackedModel, subject)
	}
	return nil
}

// reachEncodes returns contentEncodes' error for group, the model group of
// ct's content {particle}, which codegen flattens, subject naming ct: each
// basic particle goshape.Reach visits, in reach order, as particleEncodes
// admits it, the subject counting the alternatives the slice's sealed
// interface holds (bindReach), a model group particle with {max occurs} 0
// declining by particleEncodes too, and goshape.Reach's own error plain,
// prefixed by subject, as bindReach's is; then ct's content model where
// xsd.Schema.ContentMatcher does not track it, with or without a wildcard
// particle in the reach, since Append attributes every child of a flattened
// content model through a Matcher (flattenedCheck), as Decode's bindFlattened
// does.
func (w *encodeWalk) reachEncodes(ct xsd.ComplexType, group xsd.ModelGroup, subject string) error {
	alt := 0
	// visitErr is particleEncodes' error, which already carries its context
	// and, set, ended the walk as goshape.Reach's error, so that only
	// goshape.Reach's own is wrapped.
	var visitErr error
	err := goshape.Reach(w.schema, group, func(p xsd.Particle, _ []int) error {
		alt++
		visitErr = w.particleEncodes(p, fmt.Sprintf("alternative %d of the flattened content {particle} of %s", alt, subject))
		return visitErr
	})
	if visitErr != nil {
		return visitErr
	}
	if err != nil {
		return fmt.Errorf("codec: %s: %w", subject, err)
	}
	if _, tracked := w.schema.ContentMatcher(ct); !tracked {
		return encodeDecline(constructUntrackedModel, subject)
	}
	return nil
}

// wildcardParticle reports whether p's {term} is a wildcard. TermOrRef and
// Term are closed sums, so the assertions are STYLE T2's sealed-sum exception.
func wildcardParticle(p xsd.Particle) bool {
	resolved, ok := p.Term().(xsd.ResolvedTerm)
	if !ok {
		return false
	}
	_, ok = resolved.Term.(xsd.Wildcard)
	return ok
}

// particleEncodes returns elementEncodes' error for the particle p of a
// spliced content sequence, of a content choice's alternative, of a
// sequence or choice held in an occurrence field, or a basic particle a
// flattened content {particle} reaches (reachEncodes), subject naming p: {max
// occurs} 0, then a term other than an element declaration or a wildcard with
// {process contents} skip or lax, which is examined no further, then the
// declaration as the document element's is examined (encodeWalk.element). The
// declaration of an <element ref> is the top-level one its name resolves to
// through [xsd.Schema.Element] (§3.3.2.4 ref.elt.global), as Decode's
// particleTerm follows it, a name resolving to none a plain error. Where
// codegen gives its field a sealed interface (goshape.SubstitutionSum), an
// abstract declaration's always, the declarations examined are the members
// that interface has a branch for, in group order, the first decline winning,
// since Append writes each under its own expanded name (§3.9.4.3 cvc-accept
// clause 2.3.2), and the declaration itself only as one of them, so that with
// none nothing is examined. Otherwise the declaration alone is examined, as a
// local one is, an element of its own expanded name matching the particle
// (clause 2.3.1). A member whose type reaches the declaration again answers
// through encodeWalk.walked. {term} is a closed sum, so the switches are
// STYLE T2's sealed-sum exception.
func (w *encodeWalk) particleEncodes(p xsd.Particle, subject string) error {
	if upper, bounded := p.Occurs().Max(); bounded && upper == 0 {
		return encodeDecline(constructVacuousParticle, subject)
	}
	var el xsd.ElementDeclaration
	switch term := p.Term().(type) {
	case xsd.ModelGroupRef:
		return encodeDecline(constructNestedGroup, subject)
	case xsd.ElementDeclarationRef:
		decl, ok := w.schema.Element(term.Name)
		if !ok {
			return fmt.Errorf("codec: the <element ref> particle to %s does not resolve", term.Name.Display())
		}
		members, sum := goshape.SubstitutionSum(w.schema, decl)
		if !sum {
			members = []xsd.ElementDeclaration{decl}
		}
		for _, m := range members {
			if err := w.element(m); err != nil {
				return err
			}
		}
		return nil
	case xsd.ResolvedTerm:
		switch term := term.Term.(type) {
		case xsd.ModelGroup:
			return encodeDecline(constructNestedGroup, subject)
		case xsd.Wildcard:
			// A strict match with no governing declaration or type invalidates
			// its parent (§3.3.5.1 sic-e-outcome clause 1.1.3) and one with
			// either is assessed against it (§3.3.4.6 cvc-assess-elt clause 1),
			// which no check of Append's decides; a lax match Append holds to
			// both (wildcardMember.writeElements), and a skip match is not
			// assessed (§3.10.4.1 key-skipped).
			if term.ProcessContents() == xsd.ProcessStrict {
				return encodeDecline(constructWildcard, subject)
			}
			return nil
		case xsd.ElementDeclaration:
			el = term
		}
	}
	return w.element(el)
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
	subject := attributeSubject(decl)
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
// Datatypes §3.3 and then xs:precisionDecimal, which §3.3 does not list, the
// order constructEncodePrimitive's name lists them in; and idNames.
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
		{Space: xsd.XMLSchemaNS, Local: "QName"},
		{Space: xsd.XMLSchemaNS, Local: "NOTATION"},
		{Space: xsd.XMLSchemaNS, Local: "precisionDecimal"},
	}
	idNames = []xsd.QName{
		{Space: xsd.XMLSchemaNS, Local: "ID"},
		{Space: xsd.XMLSchemaNS, Local: "IDREF"},
		{Space: xsd.XMLSchemaNS, Local: "ENTITY"},
	}
)

// simpleEncodes returns nil where the simple type st, the type of the
// declaration owner names, encodes as the package doc's "# What encodes"
// states, and otherwise the decline of the first construct it finds: for a
// list, the first construct of its {item type definition} (atomicEncodes),
// the detail naming the item after st (itemSubject), and an assertions facet
// among st's own effective facets, its own or inherited; for an atomic or
// union st, atomicEncodes's. An error reading st is returned as is, and a
// list with no {item type definition} is a plain error.
func simpleEncodes(r xsd.TypeResolver, st *xsd.SimpleType, owner string) error {
	return typeEncodes(r, st, simpleSubject(st, owner))
}

// typeEncodes is simpleEncodes for st, subject naming it already: a
// declaration's type as simpleSubject names it, a complex type's {content
// type}'s {simple type definition} as contentSubject names it after that type
// (complexEncodes), or a union's basic member as memberSubject names it after
// its union (unionEncodes).
func typeEncodes(r xsd.TypeResolver, st *xsd.SimpleType, subject string) error {
	if st.IsSpecial() {
		return nil
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
// of the first construct it finds: for a union, unionEncodes's, and for an
// atomic type a primitive not in encodedPrimitives, a type on whose {base
// type definition} chain xs:ID, xs:IDREF or xs:ENTITY lies, then an
// assertions facet among its effective facets, its own or inherited. An error
// reading st is returned as is.
func atomicEncodes(r xsd.TypeResolver, st *xsd.SimpleType, subject string) error {
	variety, err := st.Variety(r)
	if err != nil {
		return err
	}
	if _, isUnion := variety.(xsd.Union); isUnion {
		return unionEncodes(r, st, subject)
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

// unionEncodes returns atomicEncodes's answer for st, a union simple type
// subject names: the first construct of its basic members, in basic-member
// order (goshape.BasicMembers), each examined by typeEncodes, the detail
// naming the member after subject (memberSubject), as Decode's unionDecline
// examines them; then unionAssertionsDecline's for st. An error reading st is
// returned as is.
func unionEncodes(r xsd.TypeResolver, st *xsd.SimpleType, subject string) error {
	basic, err := goshape.BasicMembers(r, st)
	if err != nil {
		return err
	}
	for _, m := range basic {
		if err := typeEncodes(r, m, memberSubject(subject, m)); err != nil {
			return err
		}
	}
	return unionAssertionsDecline(r, st, subject)
}

// unionAssertionsDecline returns constructAssertions's decline for the first
// union with an assertions facet among its effective facets, its own or
// inherited, of st, a union simple type subject names, and every union its
// {member type definitions} reach at any depth, st first and then each member
// union in order, depth first, named after subject by nestedUnionSubject.
// goshape.BasicMembers flattens a member union away, and value.ValidatingType
// reads its facets on every literal Append's identification hands it
// (encoder.activeMember), so its assertions facet would reach noAssertions.
// An error reading a type is returned as is.
func unionAssertionsDecline(r xsd.TypeResolver, st *xsd.SimpleType, subject string) error {
	if err := assertionsDecline(r, st, subject); err != nil {
		return err
	}
	members, err := st.Members(r)
	if err != nil {
		return err
	}
	for _, m := range members {
		variety, err := m.Variety(r)
		if err != nil {
			return err
		}
		// {variety} is a closed sum, so the assertion is STYLE T2's sealed-sum
		// exception.
		if _, isUnion := variety.(xsd.Union); !isUnion {
			continue
		}
		if err := unionAssertionsDecline(r, m, nestedUnionSubject(subject, m)); err != nil {
			return err
		}
	}
	return nil
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

// encoder is one Append's write of a document: decimals, the arm sets
// [WithPrecisionDecimalArms] supplied, and branches, the sets [WithBranches]
// supplied, each from the config item 5's check read (encodeStatic), by
// which a PrecisionDecimal's Value is read (decimalOf) and a union's variant
// placed at its basic member (encoder.union); out, the document so far; ns,
// the in-scope default namespace, that of the element whose start tag is open
// or whose content is being written, "" outside the document element; bound,
// the namespace names qname has declared a prefix for on the open elements,
// outermost first, bound[i] under boundPrefix(i), which no field stores
// (writtenBindings); open, the values whose element's write is open,
// outermost first (enter); and the first item 6 and item 7 errors of
// [Append]'s doc it has found, in document order. Every value is written
// whatever was found before it, so that the walk finds an item 6 error after
// an item 7 one.
type encoder struct {
	schema   *xsd.Schema
	backend  value.Backend
	decimals decimalArmSets
	branches branchSets
	out      []byte
	ns       string
	bound    []string
	open     []openValue
	invalid  *invalidValueError
	declined error
}

// openValue is a value of complex type whose element's write is open: ptr,
// its address, a pointer to its Go type, or the zero Value for a value that
// is not addressable, and path, its Go path from T.
type openValue struct {
	ptr  reflect.Value
	path string
}

// enter opens the write of v, a struct or sealed interface value of complex
// type, or the AnyType content of an element held raw, at path, and reports
// whether to write it: false, after recording item 6 of [Append]'s doc,
// where v is addressable and a value at its address of its Go type is open,
// a value that holds itself, which the write would reach again without end.
// Only a *S, []S or []I field leads to a value already open, and each of
// them yields an addressable value, so a value that is not addressable is
// opened unchecked; a value written twice outside one path, shared, is no
// cycle. Where it answers true, leave closes v.
func (e *encoder) enter(v reflect.Value, path string) bool {
	var ptr reflect.Value
	if v.CanAddr() {
		ptr = v.Addr()
		for _, o := range e.open {
			if o.ptr.IsValid() && o.ptr.Type() == ptr.Type() && o.ptr.Pointer() == ptr.Pointer() {
				e.invalidate(fmt.Errorf("codec: Append: %s is the value at %s, whose element is still being written, a value that holds itself, which no finite document holds", path, o.path))
				return false
			}
		}
	}
	e.open = append(e.open, openValue{ptr: ptr, path: path})
	return true
}

// leave closes the innermost value enter opened.
func (e *encoder) leave() { e.open = e.open[:len(e.open)-1] }

// startTag appends the start tag of the element name, but for its '>', with a
// default namespace declaration where name's namespace differs from e.ns, the
// document element's from no namespace, and makes name's namespace e.ns for
// its attributes and content, returning the one it replaced and mark, the
// length of e.bound, both of which endTag restores, so that a prefix qname
// declares in this start tag is in scope until endTag.
func (e *encoder) startTag(name xsd.QName) (outer string, mark int) {
	e.out = append(e.out, '<')
	e.out = append(e.out, name.Local...)
	if name.Space != e.ns {
		e.out = append(e.out, ` xmlns="`...)
		e.out = appendEscaped(e.out, name.Space, true)
		e.out = append(e.out, '"')
	}
	outer, e.ns = e.ns, name.Space
	return outer, len(e.bound)
}

// endTag ends the element whose start tag, which names it tag, has its '>'
// at e.out's byte start: with its end tag, or, where nothing follows that
// '>', by turning the start tag into an empty-element tag; and restores outer
// and mark, startTag's answers, as e.ns and the length of e.bound.
func (e *encoder) endTag(tag string, start int, outer string, mark int) {
	e.ns = outer
	e.bound = e.bound[:mark]
	if len(e.out) == start+1 {
		e.out = append(e.out[:start], "/>"...)
		return
	}
	e.out = append(e.out, "</"...)
	e.out = append(e.out, tag...)
	e.out = append(e.out, '>')
}

// textTag ends the start tag of the element tag names, whose startTag
// answered outer and mark, with text, escaped, as its character content, an
// empty text written as an empty-element tag (endTag).
func (e *encoder) textTag(tag, text, outer string, mark int) {
	start := len(e.out)
	e.out = append(e.out, '>')
	e.out = appendEscaped(e.out, text, false)
	e.endTag(tag, start, outer, mark)
}

// simpleElement writes the element of decl, holding v, a value of s's simple
// type, whose Go path from T is path: its start tag (startTag), then its
// value's text (simple), which may first declare a prefix on that start tag
// (qname), an empty text written as an empty-element tag (textTag). The
// error is a backend fault.
func (e *encoder) simpleElement(decl xsd.ElementDeclaration, s scalar, v reflect.Value, path string) error {
	name := decl.Name()
	outer, mark := e.startTag(name)
	text, err := e.simple(s, v, path, declSubject(decl))
	if err != nil {
		return err
	}
	e.textTag(name.Local, text, outer, mark)
	return nil
}

// invalidate records err as the document's item 6 error of [Append]'s doc,
// marked ErrInvalidValue (invalidValueError), unless an earlier value's is
// recorded.
func (e *encoder) invalidate(err error) {
	if e.invalid == nil {
		e.invalid = &invalidValueError{err}
	}
}

// invalidValueError is the error of items 6 and 8 of [Append]'s doc: err, the
// cause one of invalidate's callers built or, for item 8, a chargedError
// (assessWritten), marked ErrInvalidValue.
type invalidValueError struct{ err error }

// Error returns the cause's text unchanged.
func (e *invalidValueError) Error() string { return e.err.Error() }

// Unwrap returns ErrInvalidValue and the cause, for errors.Is and for the
// errors.As and xsderr.RuleOf reads of the cause's chain.
func (e *invalidValueError) Unwrap() []error { return []error{ErrInvalidValue, e.err} }

// declineValue records the item 7 decline of [Append]'s doc by the construct
// c, constructNoCanonical or constructQNameSpelling, for the value shown at
// path, of the simple type subject names, cause saying why, unless an earlier
// value's is recorded.
func (e *encoder) declineValue(c construct, subject, path, shown, cause string) {
	if e.declined == nil {
		e.declined = encodeDecline(c, fmt.Sprintf("%s, at %s holding %s: %s", subject, path, shown, cause))
	}
}

// simple returns the text Append writes for v, a value of s's simple type,
// s.st, held in the Go type the slice gives it, at path, owner naming s.st's
// declaration (scalarText). It records the value's item 6 or 7 error of
// [Append]'s doc, and returns a backend fault.
func (e *encoder) simple(s scalar, v reflect.Value, path, owner string) (string, error) {
	w, err := e.scalarText(s, v, path, simpleSubject(s.st, owner))
	return w.text, err
}

// written is what Append's write of one simple value found: text, the text it
// writes; ok, that it recorded no item 6 or 7 error of [Append]'s doc for the
// value; other, that the value has a lexical other than text, which a pattern
// facet may match where it rejects text (Datatypes §2.1's note); and
// spelling, that text spells an xs:QName or xs:NOTATION value, which has no
// canonical representation, or holds a list item or a union's member value
// that does.
type written struct {
	text            string
	ok              bool
	other, spelling bool
}

// scalarText returns what Append writes for v, a value of s's simple type,
// s.st, at path, subject naming s.st: a list by list, and an atomic or union
// value by value, under s's element.
func (e *encoder) scalarText(s scalar, v reflect.Value, path, subject string) (written, error) {
	if s.item != nil {
		return e.list(s, v, path, subject)
	}
	return e.value(s.elem, s.st, v, path, subject)
}

// value returns what Append writes for v, a value of st, an atomic or union
// simple type whose scalar element is elem, at path, subject naming st: a
// union's by union, and an atomic's by atom, as held reads it, under its
// atomic Go type (encoded).
func (e *encoder) value(elem scalarElem, st *xsd.SimpleType, v reflect.Value, path, subject string) (written, error) {
	// scalarElem is a closed sum, so the assertion is STYLE T2's sealed-sum
	// exception.
	if u, isUnion := elem.(*unionScalar); isUnion {
		return e.union(u, st, v, path, subject)
	}
	a := encoded[atomic](elem)
	hv, ok := e.held(a, v, path, subject)
	if !ok {
		return written{}, nil
	}
	text, ok, err := e.atom(st, a, hv, path, subject)
	return written{text: text, ok: ok, other: a.otherLexicalOf(hv), spelling: a == atomicQName}, err
}

// union returns what Append writes for v, a value of u's sealed interface, of
// the union simple type st, at path, subject naming st, deciding in this order
// (Datatypes §2.4.1.3, §4.1.4 cvc-datatype-valid): the nil interface, which
// reaches union only where a value is required, item 6 of [Append]'s doc;
// then a dynamic type the [WithBranches] set e.branches holds for the
// interface places at no basic member (unionScalar.boundIn), a pointer to a
// variant among them, item 6; then the variant's member value, written by
// that member's scalar (scalarText), whose own item 6 and 7 errors it
// records under the member's subject (memberSubject); then the ·active basic
// member· of that text (activeMember), item 7 where it is not the variant's
// member or no member accepts the text, by constructNoCanonical, or by
// constructQNameSpelling for a spelling, since another lexical of the value
// may activate its member where the one Append writes does not; then st's own
// facets, through value.ValidateLexical, a pattern rejection item 7 where the
// member value has another lexical (written.other), any other rejection item
// 6. other and spelling are the member value's.
func (e *encoder) union(u *unionScalar, st *xsd.SimpleType, v reflect.Value, path, subject string) (written, error) {
	if v.IsNil() {
		e.invalidate(fmt.Errorf("codec: Append: %s is nil, of %s, where a value is required, and no variant of the union's sealed interface is nil", path, subject))
		return written{}, nil
	}
	vt := v.Elem().Type()
	j := slices.Index(u.boundIn(e.branches, v.Type()), vt)
	if j < 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, which the WithBranches set of %s places at no basic member of %s, so that Decode returns no such value", path, vt, v.Type(), subject))
		return written{}, nil
	}
	m := u.members[j]
	w, err := e.scalarText(m, v.Elem(), path+".("+typePath(vt)+")", memberSubject(subject, m.st))
	if err != nil || !w.ok {
		return w, err
	}
	shown := fmt.Sprintf("a %s written %q", vt, w.text)
	form := "canonical representation"
	c := constructNoCanonical
	if w.spelling {
		form, c = "spelling", constructQNameSpelling
	}
	active, err := e.activeMember(st, w.text, path, subject)
	if err != nil {
		return written{}, err
	}
	if active == nil {
		e.declineValue(c, subject, path, shown, fmt.Sprintf("no member of the union accepts its %s %q (Datatypes §4.1.4 cvc-datatype-valid clause 2.3)", form, w.text))
		w.ok = false
		return w, nil
	}
	if active != m.st {
		e.declineValue(c, subject, path, shown, fmt.Sprintf("the ·active basic member· of its %s %q is %s, not its variant's member %s, so that Decode returns another variant (Datatypes §2.4.1.3 dt-active-basic-member, §4.1.4 cvc-datatype-valid clause 2.3)",
			form, w.text, memberName(active), memberName(m.st)))
		w.ok = false
		return w, nil
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, st, w.text, writtenBindings{e}, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return written{}, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		w.ok = false
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && w.other {
			e.patternDecline(w.spelling, subject, path, shown, w.text, err)
			return w, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, shown, subject, err))
	}
	return w, nil
}

// memberName names m, a basic member of a union, in a cause: by its QName, or
// for an anonymous m as one and by its schema Loc.
func memberName(m *xsd.SimpleType) string {
	if m.Name() == (xsd.QName{}) {
		return "the anonymous simple type" + definedAt(" ", m)
	}
	return m.Name().Display()
}

// activeMember returns the ·active basic member· of the union st for text,
// under the bindings of the element whose start tag is open
// (writtenBindings): the first of st's {member type definitions}, in order,
// that value.ValidatingType accepts text under, whose answer for a member
// that is itself a union is that member's own ·active basic member·
// (Datatypes dt-active-member, dt-active-basic-member, §4.1.4
// cvc-datatype-valid clause 2.3), or nil where none accepts it. Each member is
// handed text itself, as value.ValidatingType hands its members the raw
// literal, never a normalized form. A ValidatingType error
// value.IsDatatypeVerdict does not report as a verdict, or an error reading
// st's members, is a backend fault naming the value at path, subject naming
// st.
func (e *encoder) activeMember(st *xsd.SimpleType, text, path, subject string) (*xsd.SimpleType, error) {
	members, err := st.Members(e.schema)
	if err != nil {
		return nil, fmt.Errorf("codec: Append: %s, of %s: %w", path, subject, err)
	}
	for _, mt := range members {
		active, _, err := value.ValidatingType(e.backend, e.schema, mt, text, writtenBindings{e}, noAssertions{})
		if err == nil {
			return active, nil
		}
		if !value.IsDatatypeVerdict(err) {
			return nil, fmt.Errorf("codec: Append: %s, written %q, of %s, which the backend cannot assess: %w", path, text, subject, err)
		}
	}
	return nil, nil
}

// patternDecline records the item 7 decline of [Append]'s doc for the value
// shown at path, of the simple type subject names, whose text a pattern facet
// rejects with err, the value having another lexical: by
// constructQNameSpelling where spelling holds, text being a spelling, and by
// constructNoCanonical otherwise.
func (e *encoder) patternDecline(spelling bool, subject, path, shown, text string, err error) {
	if spelling {
		e.declineValue(constructQNameSpelling, subject, path, shown, fmt.Sprintf("a pattern facet rejects its spelling %q: %v", text, err))
		return
	}
	e.declineValue(constructNoCanonical, subject, path, shown, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", text, err))
}

// atom returns the text Append writes for v, a value of the atomic simple
// type st held in a, its atomic Go type, at path, subject naming st: the
// value of an atomic formed holds for in its canonical representation
// (spelled), a string as itself (str), an xs:QName or xs:NOTATION value under
// a prefix in scope (qname), a float32 or float64 in its canonical
// representation (double), and a precisionDecimal, v the decimalValue held
// translated it into, in its canonical representation (precisionDecimal). ok
// reports that it recorded no item 6 or 7 error of [Append]'s doc for v.
func (e *encoder) atom(st *xsd.SimpleType, a atomic, v reflect.Value, path, subject string) (text string, ok bool, err error) {
	if a == atomicPrecisionDecimal {
		return e.precisionDecimal(st, v.Interface().(decimalValue), path, subject)
	}
	if a.formed() {
		return e.spelled(st, a, v, path, subject)
	}
	if a == atomicString {
		return e.str(st, v.String(), path, subject)
	}
	if a == atomicQName {
		return e.qname(st, v, path, subject)
	}
	return e.double(st, v.Float(), a.bare().Bits(), path, subject)
}

// list returns what Append writes for v, a []V holding a value of s's list
// type, s.st, at path, subject naming s.st: the text the package doc's
// "# What encodes" states, each item written by value under s.item.st, its
// {item type definition}, and s's element, at path "[i]", and records the item
// 6 and 7 errors [Append]'s doc states for a list: an item's white space is
// checked once its own checks pass, and s.st's own facets, through
// value.ValidateLexical, once every item has. An empty item is no list item
// since the list's whiteSpace collapse (Datatypes §4.3.6) leaves no empty
// token. A pattern rejection is item 7 only where an item with another lexical
// is held, which a pattern may match (§2.1's note, written.other), since the
// list's text has another lexical exactly where one of its items has; the
// empty list has no other lexical, nor a list none of whose items has one, so
// every lexical of such a value normalizes to the text written. A list holding
// a spelling of an xs:QName or xs:NOTATION value records that item 7 by
// constructQNameSpelling (patternDecline). A union item's own facets and
// ·active basic member· are its own (encoder.union); its token holds no white
// space, so a member's whiteSpace is the identity on it.
func (e *encoder) list(s scalar, v reflect.Value, path, subject string) (written, error) {
	isubject := itemSubject(subject, s.item.st)
	items := make([]string, v.Len())
	out := written{ok: true}
	for i := range v.Len() {
		ipath := path + "[" + strconv.Itoa(i) + "]"
		w, err := e.value(s.elem, s.item.st, v.Index(i), ipath, isubject)
		if err != nil {
			return written{}, err
		}
		if w.ok && (w.text == "" || strings.ContainsAny(w.text, listSeparators)) {
			e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, which is empty or holds white space, so no list item's lexical representation spells it (Datatypes §2.4.1.2)", ipath, strconv.Quote(w.text), isubject))
			w.ok = false
		}
		out.ok = out.ok && w.ok
		out.other = out.other || w.other
		out.spelling = out.spelling || w.spelling
		items[i] = w.text
	}
	out.text = strings.Join(items, " ")
	if !out.ok {
		return out, nil
	}
	shown := fmt.Sprintf("%v", v)
	// scalarElem is a closed sum, so the assertion is STYLE T2's sealed-sum
	// exception.
	if a, isAtomic := s.elem.(atomic); !isAtomic || a.formed() || a == atomicPrecisionDecimal {
		// fmt spells a big.Rat, a big.Int or a datatype struct by its fields, a
		// []byte by its octets in decimal, and a union's variant by its member
		// value as those; each item's text is its form (spelled), its canonical
		// representation, or a union member's text.
		shown = "[" + out.text + "]"
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, s.st, out.text, writtenBindings{e}, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return written{}, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		out.ok = false
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && out.other {
			e.patternDecline(out.spelling, subject, path, shown, out.text, err)
			return out, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, written %q, which %s rejects: %w", path, shown, out.text, subject, err))
	}
	return out, nil
}

// listSeparators are the white space characters that separate a list's items
// (Datatypes §2.4.1.2, the S production of XML 1.0 §2.3): #x9, #xA, #xD and
// #x20, not unicode.IsSpace's set.
const listSeparators = "\t\n\r "

// str returns s, a value of the simple type st whose primitive is xs:string
// or xs:anyURI, or a special type, which it returns after the Char check
// alone, after checking, in this order, that it is UTF-8 of XML 1.0 Char
// characters only, which both value spaces and both special types' lexical
// spaces are sequences of (Datatypes §3.3.1.1, §3.3.17.1, §3.2.1.2,
// §3.2.2.2), that value.ValidateLexical accepts it under st, and
// that it is the canonical representation of the value that returns, its own
// whiteSpace-normalized form, as the package doc's "# What encodes" states a
// string value is written. Canonical renders the value ValidateLexical
// returns, never one Parse maps s to: Parse takes a normalized lexical, and s
// may be none. ok is atom's.
func (e *encoder) str(st *xsd.SimpleType, s, path, subject string) (text string, ok bool, err error) {
	shown := strconv.Quote(s)
	if i := xmlchar.IndexNonChar(s); i >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, whose byte offset %d starts no XML 1.0 Char (Datatypes §3.3.1.1, §3.3.17.1, §3.2.1.2, §3.2.2.2)", path, shown, subject, i))
		return s, false, nil
	}
	if st.IsSpecial() {
		return s, true, nil
	}
	v, err := value.ValidateLexical(e.backend, e.schema, st, s, writtenBindings{e}, noAssertions{})
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
		e.declineValue(constructNoCanonical, subject, path, shown, "its governing mapping has no canonical mapping")
		return s, false, nil
	}
	c, err := m.Canonical(v)
	if err != nil {
		e.declineValue(constructNoCanonical, subject, path, shown, err.Error())
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
		e.declineValue(constructNoCanonical, subject, path, shown, "its governing mapping has no canonical mapping")
		return "", false, nil
	}
	c, err := m.Canonical(v)
	if err != nil {
		e.declineValue(constructNoCanonical, subject, path, shown, err.Error())
		return "", false, nil
	}
	normalized, err := value.ApplyWhiteSpace(e.schema, st, c)
	if err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
	}
	if _, err := m.Parse(normalized, nil); err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, whose canonical representation %q the backend's Parse rejects: %w", path, shown, subject, c, err)
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, st, c, writtenBindings{e}, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && !soleLexical(f) {
			e.declineValue(constructNoCanonical, subject, path, shown, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", c, err))
			return c, false, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, shown, subject, err))
		return c, false, nil
	}
	if !mapsBack(c, f, bits) {
		e.declineValue(constructNoCanonical, subject, path, shown, fmt.Sprintf("its canonical representation %q does not map back to it", c))
		return c, false, nil
	}
	return c, true, nil
}

// held returns v, a value of a Go type over a's, at path, subject naming its
// simple type, as atom reads it: for atomicPrecisionDecimal, the decimalValue
// value of decimalValueType its PrecisionDecimal denotes (decimalOf), and ok
// false where it denotes none, item 6 of [Append]'s doc recorded; every other
// a's v itself.
func (e *encoder) held(a atomic, v reflect.Value, path, subject string) (reflect.Value, bool) {
	if a != atomicPrecisionDecimal {
		return v, true
	}
	d, ok := e.decimalOf(v, path, subject)
	if !ok {
		return reflect.Value{}, false
	}
	dv := reflect.New(decimalValueType).Elem()
	dv.Set(reflect.ValueOf(d))
	return dv, true
}

// decimalOf returns the decimalValue that v, a struct of codegen's PrecisionDecimal
// shape (decimalShape), at path, subject naming its simple type, holds in its Value
// field, by the role e.decimals' set for that field's interface gives its dynamic type,
// pipeline.decimal's inverse: the numerical arm converted to decimalNumerical, which
// checkDecimalArms held it to, and each special arm its own. Where v is item 6's
// PrecisionDecimal case of [Append]'s doc, it records that error, and ok is false. Item
// 5 held the arms first (checkDecimalArms), so an interface with none is an internal
// bug, which panics.
func (e *encoder) decimalOf(v reflect.Value, path, subject string) (decimalValue, bool) {
	field := v.Field(0)
	fpath := path + "." + v.Type().Field(0).Name
	arms, ok := e.decimals.of(field.Type())
	if !ok {
		panic(fmt.Sprintf("codec: no arms of %s reached the write", field.Type()))
	}
	if field.IsNil() {
		e.invalidate(fmt.Errorf("codec: Append: %s is nil, of %s, which no precisionDecimal value maps to (codegen's package doc, \"# Simple types\")", fpath, subject))
		return nil, false
	}
	arm := field.Elem()
	if arm.Type() == arms.numerical {
		n := arm.Convert(reflect.TypeFor[decimalNumerical]()).Interface().(decimalNumerical)
		if n.Coefficient.Sign() < 0 {
			e.invalidate(fmt.Errorf("codec: Append: %s holds the Coefficient %s, of %s, negative, which no precisionDecimal value maps to (codegen's package doc, \"# Simple types\")", fpath, n.Coefficient.String(), subject))
			return nil, false
		}
		return n, true
	}
	for _, s := range decimalSpecials {
		if s.armOf(arms) == arm.Type() {
			return s, true
		}
	}
	e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, of %s, which the WithPrecisionDecimalArms set of %s supplies in no role, so that Decode returns no such value", fpath, arm.Type(), subject, field.Type()))
	return nil, false
}

// precisionDecimal returns the canonical representation of d, a value of the
// simple type st whose primitive is xs:precisionDecimal, at path, subject
// naming st, as double writes a float's: d's pDecimalRep (decimalValue.lexical)
// mapped by st's governing mapping's Parse, rendered by its Canonical, that
// rendering whiteSpace-normalized (value.ApplyWhiteSpace) and accepted by
// Parse, then held to st's facets by value.ValidateLexical, and the normalized
// rendering read back by convertDecimal to d itself (sameDecimal). The
// rendering is held to mapping back alone, never to text of the codec's own, so
// the codec writes what the backend renders where precisionDecimal §6
// f-precDecCanmap decides no canonical representation, at a zero. A pattern
// facet's rejection is item 7 of [Append]'s doc where d has another lexical
// (decimalValue.otherLexical), and item 6 otherwise; any other facet's is item
// 6. ok is atom's.
func (e *encoder) precisionDecimal(st *xsd.SimpleType, d decimalValue, path, subject string) (text string, ok bool, err error) {
	shown := d.lexical()
	m, err := e.mapping(st, path, subject)
	if err != nil {
		return "", false, err
	}
	v, err := m.Parse(shown, nil)
	if err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, whose pDecimalRep form the backend's Parse rejects: %w", path, shown, subject, err)
	}
	if m.Canonical == nil {
		e.declineValue(constructNoCanonical, subject, path, shown, "its governing mapping has no canonical mapping")
		return "", false, nil
	}
	c, err := m.Canonical(v)
	if err != nil {
		e.declineValue(constructNoCanonical, subject, path, shown, err.Error())
		return "", false, nil
	}
	normalized, err := value.ApplyWhiteSpace(e.schema, st, c)
	if err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
	}
	if _, err := m.Parse(normalized, nil); err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, whose canonical representation %q the backend's Parse rejects: %w", path, shown, subject, c, err)
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, st, c, writtenBindings{e}, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && d.otherLexical() {
			e.declineValue(constructNoCanonical, subject, path, shown, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", c, err))
			return c, false, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, which %s rejects: %w", path, shown, subject, err))
		return c, false, nil
	}
	if !decimalMapsBack(normalized, d) {
		e.declineValue(constructNoCanonical, subject, path, shown, fmt.Sprintf("its canonical representation %q does not map back to it", c))
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
	pm, primitive, err := e.primitiveMapping(st, path, subject)
	if err != nil {
		return "", false, err
	}
	if _, err := pm.Parse(lexical, nil); err != nil {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, whose canonical spelling the backend's Parse of %s rejects: %w", path, lexical, subject, primitive.Display(), err)
	}
	val, err := value.ValidateLexical(e.backend, e.schema, st, lexical, writtenBindings{e}, noAssertions{})
	if err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, lexical, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && a.otherLexicalOf(v) {
			e.declineValue(constructNoCanonical, subject, path, lexical, fmt.Sprintf("a pattern facet rejects its canonical representation %q: %v", lexical, err))
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
		e.declineValue(constructNoCanonical, subject, path, lexical, "its governing mapping has no canonical mapping")
		return "", false, nil
	}
	c, err := m.Canonical(val)
	if err != nil {
		e.declineValue(constructNoCanonical, subject, path, lexical, err.Error())
		return "", false, nil
	}
	if c != lexical {
		return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend's Canonical renders %q, other text than its canonical representation (Datatypes Appendix E, the canonical mapping of %s)", path, lexical, subject, c, primitive.Display())
	}
	return c, true, nil
}

// primitiveMapping returns the mapping on e's backend of the {primitive type
// definition} of st, and that primitive's expanded name, or a backend fault
// naming the value at path, subject naming st, where st has no primitive or
// the backend no mapping of it; an error reading st is returned wrapped.
func (e *encoder) primitiveMapping(st *xsd.SimpleType, path, subject string) (value.Mapping, xsd.QName, error) {
	primitive, err := st.Primitive(e.schema)
	if err != nil {
		return value.Mapping{}, xsd.QName{}, fmt.Errorf("codec: Append: %s, of %s: %w", path, subject, err)
	}
	if primitive == nil {
		// atomicEncodes declines a type with no {primitive type definition}.
		return value.Mapping{}, xsd.QName{}, fmt.Errorf("codec: Append: %s, of %s: the type has no {primitive type definition}", path, subject)
	}
	pm, has := e.backend.Mapping(primitive.Name())
	if !has {
		return value.Mapping{}, xsd.QName{}, fmt.Errorf("codec: Append: %s, of %s: the backend has no mapping of its primitive %s", path, subject, primitive.Name().Display())
	}
	return pm, primitive.Name(), nil
}

// qname returns the spelling of v, QName's datatype struct holding {Space,
// Local}, of the simple type st whose primitive is xs:QName or xs:NOTATION, at
// path, subject naming st, written on the element whose start tag is open
// (spellQName, which records the error where no spelling exists), and records
// v's item 6 or 7 error of [Append]'s doc for the spelling, deciding in this
// order: the Parse of st's primitive's mapping under writtenBindings, the
// resolution back to {Space, Local} (expandedName), and value.ValidateLexical
// under writtenBindings, a pattern rejection item 7 where
// atomicQName.otherLexicalOf holds. A Parse or ValidateLexical error
// value.IsDatatypeVerdict does not report as a verdict is a backend fault. No
// Canonical is read. ok is atom's.
func (e *encoder) qname(st *xsd.SimpleType, v reflect.Value, path, subject string) (text string, ok bool, err error) {
	q := v.Convert(atomicQName.bare())
	space, local := q.FieldByName("Space").String(), q.FieldByName("Local").String()
	shown := atomicQName.shown(v)
	text, ok = e.spellQName(space, local, path, shown, subject)
	if !ok {
		return "", false, nil
	}
	scope := writtenBindings{e}
	pm, primitive, err := e.primitiveMapping(st, path, subject)
	if err != nil {
		return "", false, err
	}
	if _, err := pm.Parse(text, scope); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, written %q, which the backend cannot assess: %w", path, shown, subject, text, err)
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, written %q, which the backend's Parse of %s rejects: %w", path, shown, subject, text, primitive.Display(), err))
		return text, false, nil
	}
	back, err := expandedName(text, scope)
	if err == nil && back.Interface() != q.Interface() {
		err = fmt.Errorf("it resolves to %s", atomicQName.shown(back))
	}
	if err != nil {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, written %q, which does not resolve back to it: %w", path, shown, subject, text, err))
		return text, false, nil
	}
	if _, err := value.ValidateLexical(e.backend, e.schema, st, text, scope, noAssertions{}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			return "", false, fmt.Errorf("codec: Append: %s holds %s, of %s, which the backend cannot assess: %w", path, shown, subject, err)
		}
		if rule, _ := xsderr.RuleOf(err); rule == rulePatternValid && atomicQName.otherLexicalOf(v) {
			e.declineValue(constructQNameSpelling, subject, path, shown, fmt.Sprintf("a pattern facet rejects its spelling %q: %v", text, err))
			return text, false, nil
		}
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, written %q, which %s rejects: %w", path, shown, text, subject, err))
		return text, false, nil
	}
	return text, true, nil
}

// spellQName returns the spelling the package doc's "# What encodes" states
// for the expanded name {space, local}, the value shown at path of the simple
// type subject names, on the element whose start tag is open: for a space
// other than the XML namespace and e.ns that e.bound does not hold, it appends
// the declaration of boundPrefix(len(e.bound)) to that start tag and pushes
// space onto e.bound, so that e.bound holds only namespace names a declaration
// binds. Where none spells it — a space "" under a non-absent e.ns, then a
// space holding a byte that starts no XML 1.0 Char, then xmlnsNamespace — it
// records that value's item 7 or item 6 error of [Append]'s doc, in that
// order, and ok is false.
func (e *encoder) spellQName(space, local, path, shown, subject string) (text string, ok bool) {
	if space == "" && e.ns != "" {
		e.declineValue(constructQNameSpelling, subject, path, shown, fmt.Sprintf("its namespace name is absent where the default namespace in scope, its element's own, is %s, so that no unprefixed spelling resolves to it, and no prefix can be bound to no namespace (Namespaces in XML §5 nsc-NoPrefixUndecl)", e.ns))
		return "", false
	}
	if i := xmlchar.IndexNonChar(space); i >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, whose namespace name's byte offset %d starts no XML 1.0 Char, so that no namespace declaration binds it (Datatypes §3.3.17.1)", path, shown, subject, i))
		return "", false
	}
	if space == xmlnsNamespace {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, of %s, whose namespace name is the one the prefix xmlns is bound to, which no other prefix may be bound to and no lexical's prefix resolves to (Namespaces in XML §3)", path, shown, subject))
		return "", false
	}
	switch space {
	case xmltree.XMLNamespaceURI:
		return "xml:" + local, true
	case e.ns:
		return local, true
	}
	return e.declaredPrefix(space) + ":" + local, true
}

// declaredPrefix returns the prefix of space, a namespace name other than
// the XML namespace, xmlnsNamespace and "": the one an open start tag
// declares for it, boundPrefix(i) where e.bound[i] is space, or else, where
// e.bound does not hold it, boundPrefix(len(e.bound)), whose declaration it
// appends to the start tag that is open and pushes space onto e.bound, so
// that e.bound holds only namespace names a declaration binds.
func (e *encoder) declaredPrefix(space string) string {
	i := slices.Index(e.bound, space)
	if i < 0 {
		i = len(e.bound)
		e.bound = append(e.bound, space)
		e.out = append(e.out, " xmlns:"+boundPrefix(i)+`="`...)
		e.out = appendEscaped(e.out, space, true)
		e.out = append(e.out, '"')
	}
	return boundPrefix(i)
}

// attributionCheck is how Append holds the children it writes into one
// element's content, whose content model holds a wildcard particle or is
// flattened, to that content model (§3.9.4.2 cvc-particle, §3.9.4.3
// cvc-accept, §3.8.4.3 cvc-model-group): a, a matchAttributor over a fresh
// Matcher of the content model's {content type} and its wildcard particles'
// places, or a flattened one's every basic particle's (flattenedCheck), which
// decides for each name cvc-wildcard (§3.10.4.1), clauses 1 to 3, {disallowed
// names} included, and cvc-accept's element particle before wildcard particle
// precedence; elements, the content model's element particles' places
// (childIndex.elements), or a flattened one's every basic particle's; and
// subject, the content's complex type and element as typeSubject names them.
type attributionCheck struct {
	a        *matchAttributor
	elements []place
	subject  string
}

// attribution returns the attributionCheck of the content of el, of the
// complex type typ, whose children x attributes, or nil where x places no
// wildcard particle, a content model whose children Append writes unchecked.
// Item 4 of [Append]'s doc declines a content model holding a wildcard
// particle that xsd.Schema.ContentMatcher does not track (contentEncodes), so
// that a matcherSlot.attributor error here is an internal bug, which panics.
func (e *encoder) attribution(x childIndex, el xsd.ElementDeclaration, typ xsd.ComplexType) *attributionCheck {
	if x.matcher == nil || len(x.matcher.places) == 0 {
		return nil
	}
	a, err := x.matcher.attributor(e.schema)
	if err != nil {
		panic(fmt.Sprintf("codec: Append reached a content model item 4 of Append's doc admitted: %v", err))
	}
	return &attributionCheck{a: a, elements: x.elements, subject: typeSubject(el, typ)}
}

// take puts the child named name, written at path from the field of the
// particle of attribution target target, to c's Matcher, in document order,
// and records item 6 of [Append]'s doc where the Matcher admits it at no
// particle or attributes it to a particle other than that one (attributeAt).
// A nil c checks nothing. A Matcher position c's places do not hold is a
// defect of the premise matcherSlot.attributor states, which panics.
func (c *attributionCheck) take(e *encoder, name xsd.QName, target int, path string) {
	if c == nil {
		return
	}
	t, pos, _ := c.a.m.NextParticle(name)
	got, ok, err := attributeAt(t, pos, c.a.places, c.elements, name, path)
	if err != nil {
		panic(err.Error())
	}
	if !ok {
		e.invalidate(fmt.Errorf("codec: Append: %s holds element %s, which no particle of the content model of %s admits where it is written (§3.9.4.3 cvc-accept, §3.10.4.1 cvc-wildcard)", path, name.Display(), c.subject))
		return
	}
	if got.index != target {
		e.invalidate(fmt.Errorf("codec: Append: %s holds element %s, which the content model of %s attributes to a particle other than the one whose field holds it (§3.9.4.3 cvc-accept, §3.4.4.4 key-att-to)", path, name.Display(), c.subject))
	}
}

// flattenedCheck returns the attributionCheck of the content of at.el, of the
// complex type at.typ, whose {particle} f is: a matchAttributor over a fresh
// Matcher of f.ct's {content type} and f.places, every alternative's place,
// element and wildcard alike, also its elements. Item 4 of [Append]'s doc
// declines a flattened content model xsd.Schema.ContentMatcher does not
// track (contentEncodes), so that a decline here is an internal bug, which
// panics, as attribution's does.
func (e *encoder) flattenedCheck(f *flattened, at writeAt) *attributionCheck {
	m, ok := e.schema.ContentMatcher(f.ct)
	if !ok {
		panic(fmt.Sprintf("codec: Append reached a flattened content model item 4 of Append's doc admitted, which xsd.Schema.ContentMatcher declines: %s", complexSubject(f.ct)))
	}
	return &attributionCheck{a: &matchAttributor{m: m, places: f.places}, elements: f.places, subject: typeSubject(at.el, at.typ)}
}

// end records item 6 of [Append]'s doc, at path, the content's Go path, where
// c's Matcher is not accepting once the content's every child is taken
// (§3.9.4.3 cvc-accept clause 1). A nil c checks nothing.
func (c *attributionCheck) end(e *encoder, path string) {
	if c == nil || c.a.m.Accepting() {
		return
	}
	e.invalidate(fmt.Errorf("codec: Append: %s holds content the content model of %s does not accept: it ends short of a particle it must satisfy (§3.9.4.3 cvc-accept)", path, c.subject))
}

// rawName returns the expanded name {Space, Local} v, a struct of
// AnyElement's shape at path, holds, or records item 6 of [Append]'s doc
// where no element is so named (writableName), and ok is false.
func (e *encoder) rawName(v reflect.Value, path string) (xsd.QName, bool) {
	name := xsd.QName{Space: v.Field(0).String(), Local: v.Field(1).String()}
	return name, e.writableName(name, path)
}

// writableName reports whether an element or attribute can be named name, the
// expanded name held at path, in the document Append writes, and otherwise
// records item 6 of [Append]'s doc for its Local or Space and reports false.
// A Local holding a byte that is not UTF-8 is checked as no Char first, since
// xmlname.IsNCName reads it as U+FFFD, a NameStartChar.
func (e *encoder) writableName(name xsd.QName, path string) bool {
	if xmlchar.IndexNonChar(name.Local) >= 0 || !xmlname.IsNCName(name.Local) {
		e.invalidate(fmt.Errorf("codec: Append: %s holds the local name %s, which is no NCName, so that nothing is so named (Namespaces in XML [4] NCName)", path, strconv.Quote(name.Local)))
		return false
	}
	if i := xmlchar.IndexNonChar(name.Space); i >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds the namespace name %s, whose byte offset %d starts no XML 1.0 Char", path, strconv.Quote(name.Space), i))
		return false
	}
	if name.Space == xmlnsNamespace {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, whose namespace name is the one the prefix xmlns is bound to, which no element or attribute is in (Namespaces in XML §3)", path, name.Display()))
		return false
	}
	return true
}

// A rawHold is what holds the content Append writes raw (encoder.rawContent),
// at any depth: a match of a wildcard with {process contents} skip
// (skipMatch), nothing in which is assessed (§3.10.4.1 key-skipped), a match
// of one with {process contents} lax (laxMatch), or the content of an element
// of type xs:anyType itself (anyTypeContent), each of the last two laxly
// assessed but for what a declaration governs (§3.3.4.6 key-lva, §3.4.7). in
// names the hold for a decline's detail; governing returns the subject of the
// top-level declaration in schema that governs a held element, or for
// attribute a held attribute, of the expanded name name, and true, or false
// where none does.
type rawHold interface {
	in() string
	governing(schema *xsd.Schema, name xsd.QName, attribute bool) (string, bool)
}

// skipMatch is the match of the wildcard particle it names, of {process
// contents} skip.
type skipMatch string

// in names the wildcard particle.
func (m skipMatch) in() string { return "in the match of " + string(m) }

// governing reports false: a skipped item has no governing declaration,
// whatever its name (§3.10.4.1 key-skipped).
func (skipMatch) governing(*xsd.Schema, xsd.QName, bool) (string, bool) { return "", false }

// anyTypeContent is the content of an element of decl, whose {type
// definition} is xs:anyType itself, the element's own attributes among it.
type anyTypeContent struct{ decl xsd.ElementDeclaration }

// in names c.decl.
func (c anyTypeContent) in() string { return "in the content of " + declSubject(c.decl) }

// governing is topLevelGoverning's: xs:anyType's wildcards are lax (§3.4.7).
func (anyTypeContent) governing(schema *xsd.Schema, name xsd.QName, attribute bool) (string, bool) {
	return topLevelGoverning(schema, name, attribute)
}

// laxMatch is the match of the wildcard particle it names, of {process
// contents} lax.
type laxMatch string

// in names the wildcard particle.
func (m laxMatch) in() string { return "in the match of " + string(m) }

// governing is topLevelGoverning's, at every depth: below the match each item
// is laxly assessed against xs:anyType (§3.3.4.6 key-lva), whose content has
// no ·locally declared type· (§3.4.6.4 key-ldt-elem clause 1).
func (laxMatch) governing(schema *xsd.Schema, name xsd.QName, attribute bool) (string, bool) {
	return topLevelGoverning(schema, name, attribute)
}

// topLevelGoverning returns the subject of the top-level element declaration,
// or for attribute the top-level attribute declaration, of schema named name,
// and true, or false where schema declares none: an item attributed to a lax
// wildcard so named is assessed against it (§3.3.4.6 key-governing-ed clause
// 3, §3.2.4.2 key-governing-ad clause 3), and every other one laxly again
// (§3.3.4.6 key-lva) or not at all.
func topLevelGoverning(schema *xsd.Schema, name xsd.QName, attribute bool) (string, bool) {
	if attribute {
		d, ok := schema.Attribute(name)
		if !ok {
			return "", false
		}
		return attributeSubject(d), true
	}
	d, ok := schema.Element(name)
	if !ok {
		return "", false
	}
	return declSubject(d), true
}

// write writes at.el, the document element, of type xs:anyType itself,
// holding v (writeElement).
func (c rawContent) write(e *encoder, v reflect.Value, at writeAt) error {
	return c.writeElement(e, at.el, v, at.path, attributionSlot{})
}

// writeElement writes the element of decl, of type xs:anyType itself, whose
// content is v, a struct of AnyType's shape or a named type over one, at
// path: its start tag, then v as rawContent writes it under anyTypeContent,
// as the package doc's "# What encodes" states; or nothing, where v holds
// itself (encoder.enter), taken by slot first either way. It reads no simple
// value, so it returns no backend fault.
func (rawContent) writeElement(e *encoder, decl xsd.ElementDeclaration, v reflect.Value, path string, slot attributionSlot) error {
	slot.take(e, decl, path)
	if !e.enter(v, path) {
		return nil
	}
	defer e.leave()
	name := decl.Name()
	outer, mark := e.startTag(name)
	start := e.rawContent(v, path, anyTypeContent{decl: decl})
	e.endTag(name.Local, start, outer, mark)
	return nil
}

// rawElement writes the element name, held raw at path in hold, rawName
// having admitted name, whose content is x, a struct of AnyType's shape, as
// the package doc's "# What encodes" states: its start tag (rawStartTag),
// then its content (rawContent) in the same hold; or nothing, where x holds
// itself (encoder.enter).
func (e *encoder) rawElement(name xsd.QName, x reflect.Value, path string, hold rawHold) {
	if !e.enter(x, path+".Content") {
		return
	}
	defer e.leave()
	tag, outer, mark := e.rawStartTag(name)
	start := e.rawContent(x, path+".Content", hold)
	e.endTag(tag, start, outer, mark)
}

// rawStartTag appends the start tag of the element name, but for its '>', and
// returns the tag's name and startTag's answers: startTag's, but for a name in
// the XML namespace, written as the package doc's "# What encodes" states,
// which moves no default namespace, since that namespace may not be the
// default one (Namespaces in XML §3).
func (e *encoder) rawStartTag(name xsd.QName) (tag, outer string, mark int) {
	if name.Space != xmltree.XMLNamespaceURI {
		outer, mark = e.startTag(name)
		return name.Local, outer, mark
	}
	tag = "xml:" + name.Local
	e.out = append(e.out, '<')
	e.out = append(e.out, tag...)
	return tag, e.ns, len(e.bound)
}

// rawContent appends to the start tag that is open the attributes of x, a
// struct of AnyType's shape at path, held raw in hold, then the tag's '>',
// whose offset it returns, which endTag reads, then x's Text and each of its
// Children's Element, written by rawElement, followed by its Tail, each as the
// package doc's "# What encodes" states, a prefix by declaredPrefix. It
// records the item 6 and item 7 errors of [Append]'s doc these raise, in
// document order (rawAttribute, rawName, heldElement, rawText), and leaves
// each such attribute or text, and each element rawName refuses, out of what
// it writes; an element heldElement declines is written, so that its content
// is examined for item 6 too, though Append returns no byte of it.
func (e *encoder) rawContent(x reflect.Value, path string, hold rawHold) int {
	attrs := x.Field(0)
	for i := range attrs.Len() {
		a := attrs.Index(i)
		apath := path + ".Attributes[" + strconv.Itoa(i) + "]"
		name := xsd.QName{Space: a.Field(0).String(), Local: a.Field(1).String()}
		value := a.Field(2).String()
		if !e.rawAttribute(name, value, attrs, i, apath, hold) {
			continue
		}
		qualified := name.Local
		switch {
		case name.Space == xmltree.XMLNamespaceURI:
			qualified = "xml:" + name.Local
		case name.Space != "":
			qualified = e.declaredPrefix(name.Space) + ":" + name.Local
		}
		e.out = append(e.out, ' ')
		e.out = append(e.out, qualified...)
		e.out = append(e.out, `="`...)
		e.out = appendEscaped(e.out, value, true)
		e.out = append(e.out, '"')
	}
	start := len(e.out)
	e.out = append(e.out, '>')
	e.rawText(x.Field(1).String(), path+".Text")
	children := x.Field(2)
	for i := range children.Len() {
		c := children.Index(i)
		cpath := path + ".Children[" + strconv.Itoa(i) + "]"
		if name, ok := e.rawName(c.Field(0), cpath+".Element"); ok {
			e.heldElement(name, cpath+".Element", hold)
			e.rawElement(name, c.Field(0).Field(2), cpath+".Element", hold)
		}
		e.rawText(c.Field(1).String(), cpath+".Tail")
	}
	return start
}

// heldElement records item 7 of [Append]'s doc, constructGovernedContent,
// where hold governs the element name it holds at path and no decline is
// recorded yet.
func (e *encoder) heldElement(name xsd.QName, path string, hold rawHold) {
	if governor, ok := hold.governing(e.schema, name, false); ok && e.declined == nil {
		e.declined = encodeDecline(constructGovernedContent, fmt.Sprintf("element %s at %s, %s, governed by %s", name.Display(), path, hold.in(), governor))
	}
}

// locallyTyped records item 7 of [Append]'s doc, constructLocallyTypedMatch,
// where the element name, the match of a lax wildcard particle held at path
// in hold, has a ·locally declared type· in at.typ, the complex type whose
// content it is written into, at.el's (xsd.Schema.LocallyDeclaredElementType),
// and no decline is recorded yet. Only the match itself has one: below it no
// item does (laxMatch.governing).
func (e *encoder) locallyTyped(name xsd.QName, path string, hold rawHold, at writeAt) {
	if e.declined != nil {
		return
	}
	if _, ok := e.schema.LocallyDeclaredElementType(at.typ, name); !ok {
		return
	}
	e.declined = encodeDecline(constructLocallyTypedMatch, fmt.Sprintf("element %s at %s, %s, governed by its locally declared type in %s", name.Display(), path, hold.in(), typeSubject(at.el, at.typ)))
}

// rawAttribute reports whether rawContent writes the attribute name of value
// value, attrs' item i at path, held in hold, recording its item 6 or item 7
// error of [Append]'s doc where it does not: writableName's, an unqualified
// xmlns, an expanded name an attribute before it holds, the XSI namespace's
// (constructXsiAttribute), a Value of a byte that starts no Char, and a name
// hold governs (constructGovernedContent), in that order.
func (e *encoder) rawAttribute(name xsd.QName, value string, attrs reflect.Value, i int, path string, hold rawHold) bool {
	if !e.writableName(name, path) {
		return false
	}
	if name.Space == "" && name.Local == "xmlns" {
		e.invalidate(fmt.Errorf("codec: Append: %s holds the attribute xmlns, a namespace declaration, which no element holds as an attribute (Namespaces in XML §3)", path))
		return false
	}
	for j := range i {
		if b := attrs.Index(j); b.Field(0).String() == name.Space && b.Field(1).String() == name.Local {
			e.invalidate(fmt.Errorf("codec: Append: %s holds the attribute %s, as item %d before it does, and no element holds two attributes of one expanded name (Namespaces in XML §6.3 nsc-AttrsUnique)", path, name.Display(), j))
			return false
		}
	}
	if name.Space == xsd.XMLSchemaInstanceNS {
		if e.declined == nil {
			e.declined = encodeDecline(constructXsiAttribute, fmt.Sprintf("attribute %s at %s, %s", name.Display(), path, hold.in()))
		}
		return false
	}
	if j := xmlchar.IndexNonChar(value); j >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s.Value holds %s, whose byte offset %d starts no XML 1.0 Char", path, strconv.Quote(value), j))
		return false
	}
	if governor, ok := hold.governing(e.schema, name, true); ok {
		if e.declined == nil {
			e.declined = encodeDecline(constructGovernedContent, fmt.Sprintf("attribute %s at %s, %s, governed by %s", name.Display(), path, hold.in(), governor))
		}
		return false
	}
	return true
}

// rawText appends text, character data held raw at path, escaped as element
// content, or records item 6 of [Append]'s doc where it holds a byte that
// starts no XML 1.0 Char, and writes nothing.
func (e *encoder) rawText(text, path string) {
	if i := xmlchar.IndexNonChar(text); i >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, whose byte offset %d starts no XML 1.0 Char", path, strconv.Quote(text), i))
		return
	}
	e.out = appendEscaped(e.out, text, false)
}

// boundPrefix is the prefix of e.bound[i]: "p" and i+1 in decimal.
func boundPrefix(i int) string { return "p" + strconv.Itoa(i+1) }

// xmlnsNamespace is the namespace name the prefix xmlns is bound to by
// definition (Namespaces in XML §3); xml's is xmltree.XMLNamespaceURI.
const xmlnsNamespace = "http://www.w3.org/2000/xmlns/"

// writtenBindings is the value.Context of the namespace bindings in scope at
// the element whose start tag e has open, the ones the document Append writes
// declares there: the empty prefix bound to e.ns, "" where it is absent; xml
// to the XML namespace; boundPrefix(i) to e.bound[i]; and every other prefix,
// xmlns among them, unbound, as parser/xmltree answers xmlns. It reads e live,
// so that a prefix qname declares is bound for the checks after it. Decode's
// elementBindings reads a parsed start tag's scope instead, which the writer
// has none of.
type writtenBindings struct{ e *encoder }

// LookupNamespace resolves prefix against the bindings writtenBindings states.
func (b writtenBindings) LookupNamespace(prefix string) (string, bool) {
	switch prefix {
	case "":
		return b.e.ns, true
	case "xml":
		return xmltree.XMLNamespaceURI, true
	}
	for i, space := range b.e.bound {
		if boundPrefix(i) == prefix {
			return space, true
		}
	}
	return "", false
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

// decimalMapsBack reports whether convertDecimal, the lexical mapping
// (precisionDecimal §6 f-precDecLexmap), reads c, precisionDecimal's
// whiteSpace-normalized canonical representation of d, back as d
// (sameDecimal).
func decimalMapsBack(c string, d decimalValue) bool {
	back, err := convertDecimal(c)
	return err == nil && sameDecimal(back.Interface().(decimalValue), d)
}

// soleLexical reports whether floatRep or doubleRep spells f by its canonical
// representation alone, as item 7 of [Append]'s doc states each does NaN and
// -INF.
func soleLexical(f float64) bool { return math.IsNaN(f) || math.IsInf(f, -1) }

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
// than an xs:string- or xs:anyURI-primitive one, of the spelling of an
// xs:QName or xs:NOTATION one, or of a list or union value holding one,
// spelled, double, precisionDecimal, qname, list and union read as item 7 of
// [Append]'s doc where the value has another lexical: where otherLexicalOf
// holds for spelled and qname, soleLexical does not hold for double,
// decimalValue.otherLexical holds for precisionDecimal and written.other holds
// for list and union.
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
// value.ValidateLexical and value.ValidatingType, which require one: it
// decides no {test}. No call reaches it, since a simple type with an
// assertions facet, a union's basic member or member union at any depth
// among them, declines before any value is read (constructAssertions,
// unionEncodes).
type noAssertions struct{}

// Evaluate declines every {test}.
func (noAssertions) Evaluate(value.Backend, xsd.TypeResolver, *xsd.SimpleType, xsd.XPathExpression, value.Value) value.AssertionAnswer {
	return value.AssertionDeclinedBecause("codec's encoder evaluates no assertion")
}
