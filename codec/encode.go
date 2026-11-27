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
// defined for primitive datatypes and inherited by those derived from them
// (§2.1's note), and never for a special one (§2.4.2 dt-special),
// each written as the package doc's "# What encodes" states. It assesses the
// document it writes, the bytes past len(dst) alone, through package validate,
// on backend and under no URI, exactly when an element declaration item 4
// examines, the document element's or one a particle reaches at any depth, has
// {identity-constraint definitions} (Structures §3.3.4.3 cvc-elt clause 6,
// §3.11.4 cvc-identity-constraint), or when a simple type item 4 examines, an
// element's, a complex type's simple content's, an attribute's, a list's item
// type or a union's basic member, is xs:ID, xs:IDREF or a type derived from
// either (§3.3.4.5 cvc-id, cvc-elt clause 7), whatever variant a union's value
// holds, or when item 4's walk reaches a wildcard particle with {process
// contents} strict or lax, or an element declaration whose {type definition} is
// xs:anyType itself, whose content Append writes raw and in which a top-level
// declaration or an xsi:type may govern an item (§3.3.4.6 key-governing-ed
// clause 3, key-governing-type-elem clause 8, §3.4.7), each of which holds for
// every value of T or for none, and runs no assessment otherwise: a caller
// wanting a verdict on such a document assesses it through package validate,
// as any document. The package doc's "# What encodes" section states the slice
// Append encodes and the document it writes. On error Append returns dst
// unextended, its bytes beyond len(dst) possibly overwritten, and the first of
// these that applies:
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
//     and {max occurs}, a choice's alternative's {min occurs} 0 read as 1 in its branch, though never a
//     flattened sequence's or choice's slice field, which holds one item per child, not per occurrence, nor
//     below its {min occurs} the occurrence field of a choice one occurrence of which is emptiable (codegen's
//     package doc, "# Particles"; §3.9.4.3 cvc-accept 3.1-3.3, §3.9.6.3 cos-group-emptiable clause 2 via
//     §3.8.6.6 cos-choice-range); the nil sealed interface of a choice occurring once, the content choice or
//     a nested one, none of whose alternatives is emptiable (Structures §3.8.4.3 cvc-model-group, §3.9.6.3
//     cos-group-emptiable), a choice of no {particles} among them; a nil item of an occurrence field's slice
//     of a choice, the content {particle}'s or a nested one's, whatever its alternatives, which Decode never
//     returns; a value of either interface whose dynamic type the [WithBranches] set in opts places at no
//     alternative, a pointer to a branch among them, which Decode never returns; a nil item of a flattened
//     sequence's or choice's slice, the content {particle}'s or a nested one's, which Decode never returns
//     either, and an item of it whose dynamic type the [WithBranches] set in opts places at no alternative, a
//     pointer to a branch among them; the nil sealed interface of an <element ref> particle's field over a
//     substitution group (codegen's package doc, "# Substitution groups"), or of an item of its slice,
//     wherever nil does not mean absent, that of a head with {abstract} true and no branch among them, and a
//     value of that interface whose dynamic type the [WithBranches] set in opts places at no branch, a
//     pointer to a branch among them, which Decode never returns; the nil sealed interface of a union simple
//     type wherever nil does not mean absent, that is, every field but an element particle's of {min occurs}
//     0 and {max occurs} 1 and an optional attribute use's to which no value constraint supplies a value
//     (codegen's package doc, "# Particles", "# Attributes"), which Decode never returns; a value of that
//     interface whose dynamic type the [WithBranches] set in opts places at no basic member, a pointer to a
//     variant among them, which Decode never returns; a value that holds itself, a struct or sealed interface
//     value its own write reaches again through a *S, []S or []I field, which no finite document holds, the
//     error naming the Go path where the value is reached again and nothing written beneath it; a string
//     holding bytes that are not UTF-8 or a character the XML 1.0 Char production does not match, U+0000
//     among them (Datatypes §3.3.1.1, §3.3.17.1, §3.2.1.2, §3.2.2.2), a held element's Space, an attribute's
//     Space or Value, a Text and a Tail among them; a string that differs from its own whiteSpace-normalized
//     form under its simple type, such as an xs:token holding " a b"; a list item, a string, that is empty or
//     holds TAB, LF, CR or a space, which no list item's lexical representation does (Datatypes §2.4.1.2); a
//     big.Rat outside the decimal value space, whose denominator in lowest terms is no 2^m·5^n (§3.3.3.1),
//     such as 1/3; an element a wildcard particle's field holds, of any {process contents}, that its
//     content model admits at no particle where it is written, a name outside the wildcard's {namespace
//     constraint} or in its {disallowed names} among them, or attributes to a particle other than that
//     wildcard, an element particle taking its name first among them, or after whose content's last child the
//     content model is not accepting, as xsd.Schema.ContentMatcher's Matcher decides (§3.9.4.3 cvc-accept,
//     §3.10.4.1 cvc-wildcard, §3.8.4.3 cvc-model-group); such an element of a wildcard particle with
//     {process contents} strict whose expanded name no top-level element declaration of schema names, that
//     has no ·locally declared type· where it is written (item 7), and whose Attributes hold no xsi:type, so
//     that neither a declaration nor a type governs it and its parent is invalid (Structures §3.3.4.6
//     key-governing-ed clause 3, key-governing-type-elem clause 8, §3.3.5.1 sic-e-outcome clause 1.1.3);
//     the Namespaces of an AnyType Append writes an element's content from, at any depth, whatever holds it,
//     that Decode never returns: an entry whose Prefix is not after the one before it, a Prefix xml or xmlns,
//     a Prefix other than "" that is no NCName, a Space "", one holding a byte that starts no XML 1.0 Char,
//     and one that is the XML namespace or the one the prefix xmlns is bound to (Namespaces in XML §3, §5
//     nsc-NoPrefixUndecl), the first such entry, then a "" entry where the element's namespace is absent,
//     which no prefix spells (codegen's package doc, "# xs:anyType"); an attribute an {attribute wildcard}'s
//     field holds whose expanded name an attribute use of the type names, the use's field nil or set, which
//     Decode never returns (§3.4.4.2 cvc-complex-type clause 2.1), or, but for xsi:type, xsi:nil,
//     xsi:schemaLocation and xsi:noNamespaceSchemaLocation, which clause 2 excepts by name, that the wildcard
//     does not admit, a name outside its {namespace constraint} or in its {disallowed names} among them
//     (clause 2.2.2, §3.10.4.1 cvc-wildcard); in such an element, in the content of an element of type
//     xs:anyType itself, its own attributes among it, at any depth, and in an {attribute wildcard}'s field, a
//     Local that is no NCName or a Space that is the namespace name the prefix xmlns is bound to, of the
//     element or of an attribute, an attribute of absent Space named xmlns (Namespaces in XML §3), and an
//     attribute of the expanded name of one before it (§6.3 nsc-AttrsUnique); an element an item of a
//     flattened sequence's or choice's slice holds that its content model admits at no particle where it is
//     written, or attributes to a basic particle other than the one whose branch the item is, or a slice
//     after whose last item the content model is not accepting, as that Matcher decides (§3.9.4.3 cvc-accept,
//     §3.10.4.1 cvc-wildcard, §3.8.4.3 cvc-model-group); a date, time or duration datatype struct value no
//     value of its datatype maps to, as codegen's package doc "# Simple types" states such values exist, the
//     error naming the first property found outside it, its fields read in order, or, ahead of its Seconds, a
//     duration's Months and Seconds of opposite signs; a PrecisionDecimal datatype struct value no value maps
//     to, as codegen's "# Simple types" states such values exist, a nil Value or the numerical arm with a
//     negative Coefficient, or a Value whose dynamic type the [WithPrecisionDecimalArms] set in opts supplies
//     in no role, a pointer to an arm among them, which Decode never returns; an xs:QName or xs:NOTATION
//     value whose namespace name holds a byte that starts no XML 1.0 Char or is the one the prefix xmlns is
//     bound to (Namespaces in XML §3), or whose Local the Parse of its primitive's mapping rejects in
//     Append's spelling, that Parse's verdict wrapped, or makes a spelling that resolves back to another
//     expanded name; a value written as an element's character content, or as an attribute, whose value
//     value.ConstraintMatches finds neither equal nor identical to the {value} of a fixed {value constraint}
//     there (Datatypes §2.2.1, §2.2.2): an element's own (Structures §3.3.4.3 cvc-elt clause 5.2.2.2.2), the
//     empty text included, which Append writes as an empty element that takes on the constraint's {lexical
//     form} (clause 5.1, §3.3.5.4 sic-eltType), and an attribute use's and its declaration's, each (§3.5.4
//     cvc-au, §3.2.4.1 cvc-attribute clause 4); the content of an element of type xs:anyType itself whose
//     declaration has a fixed {value constraint}, where it holds Children (cvc-elt clause 5.2.2.1) or a Text
//     other than "" and the constraint's {lexical form} (clause 5.2.2.2.1); or a value whose simple type
//     rejects it, a value.ValidateLexical error value.IsDatatypeVerdict reports as a verdict, a facet's
//     rejection or a backend Parse's among them, that error wrapped, but for item 7's pattern and
//     union-variant cases, a list's own facets read once every item has passed its own checks.
//  7. A value with no canonical representation or no spelling Append writes, or one Append declines at its {value
//     constraint}, the first in document order as item 6 counts: its *DeclineError. That is a governing mapping with no
//     Canonical, a Canonical that fails for the value, such as strict's for the zero xs:yearMonthDuration (Datatypes
//     §3.4.26.1's note), or a value with a lexical other than the one Append writes, or a list holding at least one such
//     item, whose canonical representation, or for an xs:QName or xs:NOTATION value its spelling, a pattern facet of its
//     type rejects (§2.1's note), such as 3/2 under an xs:decimal pattern \d+\.\d\d, which admits 1.50 but not 1.5,
//     whatever other facet rejects it; a canonical representation that does not map back to the value; a union's variant
//     whose member value Append writes as text whose ·active basic member· (Datatypes §2.4.1.3 dt-active-basic-member,
//     §4.1.4 cvc-datatype-valid clause 2.3) is not that variant's member, or that no member of the union accepts, so that
//     Decode returns another variant or none, by the construct of the clause before, or for an xs:QName or xs:NOTATION
//     member's spelling by its spelling's construct: xs:decimal 1 under a union of xs:int and xs:decimal is written 1,
//     which Decode returns as the xs:int variant, though 1.0 decodes into the xs:decimal variant holding 1, so that a
//     valid document holds the value and item 6 does not apply, whatever else rejects it; or an xs:QName or xs:NOTATION
//     value whose namespace name is absent, written on an element whose namespace is not, which no unprefixed spelling
//     resolves to there and no prefix can be bound to (Namespaces in XML §5 nsc-NoPrefixUndecl), whatever else rejects
//     it; or xsi:type, xsi:nil, xsi:schemaLocation or xsi:noNamespaceSchemaLocation in an {attribute wildcard}'s field,
//     whatever its value, since Decode never returns one of the four there, which cvc-complex-type clause 2 attributes to
//     no wildcard; or xsi:nil on an element a wildcard particle's field holds, or in the content of an element of type
//     xs:anyType itself, its own attributes among it, at any depth, whatever its value, since validate reads it there,
//     under cvc-elt clause 3 or the built-in nil declaration's cvc-attribute clause 3, as validate's package doc, "#
//     Contract", states, but on an element a skip wildcard particle's field holds or at any depth below one, which is not
//     assessed (§3.10.4.1 key-skipped, §3.3.4.6 cvc-assess-elt clause 2); or xsi:type among the attributes of an
//     element of type xs:anyType itself, whatever its value, since Decode declines an element carrying it there
//     (constructXsiType); or in a lax {attribute wildcard}'s field an attribute whose expanded name a top-level
//     attribute declaration of schema names, whatever its value, since the wildcard assesses it against that
//     declaration (§3.2.4.2 key-governing-ad clause 3); or an attribute the {attribute wildcard} field of a type whose
//     {attribute wildcard} has {process contents} strict holds, whatever its name and value, since the wildcard
//     assesses it against the top-level attribute declaration its expanded name resolves to, where there is one, and
//     §3.10.1 requires that there be one (§3.10.1, §3.2.4.2 key-governing-ad clause 3); or an element a strict or lax
//     wildcard particle's field holds whose expanded name no top-level element declaration of schema names and that has
//     a ·locally declared type· in the complex type whose content it is written into, an element declaration of its
//     expanded name that type's content model, or that of a type on its {base type definition} chain, contains,
//     whatever its content and whether or not it holds xsi:type, since that type governs it (§3.3.4.6
//     key-governing-type-elem clauses 6 and 7, §3.4.6.4 key-ldt-elem); or a token, split at XML white space, of a Text,
//     a Tail or an attribute's Value held in an AnyType, spelled NCName:NCName, whose prefix, other than xml, no entry
//     of that AnyType's Namespaces binds but a binding Append declares, a fresh pN or one an ancestor's, puts in scope
//     where it is written, so that the token would resolve where the value says nothing does, in a region holding every
//     place it may be typed, and more: the Text, Tails and attribute Values of an element a top-level element
//     declaration or an xsi:type it holds governs, in a strict or lax wildcard particle's match or an xs:anyType
//     element's content, or at any depth below one, every element of a strict match among them, the Value of an
//     attribute there a top-level attribute declaration governs, and every xsi:type's in any such hold, a skip one's
//     among them (Datatypes §3.3.18); or an ·ENTITY value· in content a document element's walk reaching a strict or
//     lax wildcard particle or an xs:anyType element holds, which the assessment of item 8 finds no ·declared entity
//     name· (Structures §3.16.4 cvc-simple-type clause 3), the document Append writes having no DTD to declare one, its
//     detail that violation's text, ahead of item 8; or a value item 6 holds to a fixed {value constraint} whose
//     comparison value.ConstraintMatches leaves undecided, such as a NOTATION {lexical form} naming no declared
//     notation, and the empty text of an element whose default's {value} it is not, or whose comparison is undecided,
//     since Append writes it as an empty element that takes on the default (cvc-elt clause 5.1), though another
//     lexical, such as a single space for an xs:token, may hold the value. The values with a lexical other than the one
//     Append writes, the ones whose rejection by a pattern facet is this item, a union's own pattern facet among them,
//     read once the variant's member is its text's ·active basic member·, are: a union's value whose member value is
//     one; every xs:boolean, xs:decimal-primitive, xs:dateTime, xs:dateTimeStamp, xs:time and xs:duration-primitive
//     value; every xs:float, xs:double and xs:precisionDecimal value other than NaN and -INF; an xs:date, xs:gYearMonth
//     or xs:gYear value whose timezoneOffset is 0, spelled Z, +00:00 or -00:00 (§D.2.2's note), or whose year is 0,
//     spelled 0000 or -0000; an xs:gMonthDay, xs:gDay or xs:gMonth value whose timezoneOffset is 0; an xs:hexBinary
//     value one of whose octets holds a hex digit A to F (§3.3.15.2); every xs:base64Binary value but the empty one
//     (§3.3.16.2); an xs:QName or xs:NOTATION value whose namespace name is not absent; and no xs:string- or
//     xs:anyURI-primitive value, each being its own whiteSpace-normalized form. NaN and -INF, of xs:float, xs:double
//     and xs:precisionDecimal alike, are written NaN and -INF, their only lexicals (§3.3.4.2, §3.3.5.2,
//     precisionDecimal §3.2), so a pattern facet of their own simple type that rejects one of them is item 6, not this
//     item.
//  8. An assessed document validate charges: the error item 3 of [Decode]'s
//     doc states for it, which errors.Is also reports as [ErrInvalidValue],
//     each violation charged to the written document, a
//     cvc-identity-constraint one naming its constraint and the clause it
//     breaks, 3, 4.1, 4.2.1, 4.2.2 or 4.3, and a cvc-id one naming the ID and
//     the clause it breaks, 1, an ID no element is bound to or an IDREF no ID
//     matches, or 2, an ID bound to more than one element (§3.17.5.2 sic-id).
//     Where Append assesses by neither condition but the first two above, a
//     violation of any other rule is a fault of Append's, reported so all the
//     same; otherwise every violation is the value's.
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
	// v's own Value, of Kind Interface where T is one, so that a nil T and a
	// branch's dynamic type both reach the write as T holds them.
	return encodeValue(dst, schema, backend, name, reflect.ValueOf(&v).Elem(), opts)
}

// encodeValue is Append with T as v's Go type, read by reflection, as
// decodeValue is Decode's.
func encodeValue(dst []byte, schema *xsd.Schema, backend value.Backend, name xsd.QName, v reflect.Value, opts []Option) ([]byte, error) {
	if schema == nil {
		return dst, errors.New("codec: Append: nil *xsd.Schema")
	}
	if backend == nil {
		return dst, errors.New("codec: Append: nil value.Backend")
	}
	t := v.Type()
	cfg := newConfig(opts)
	decl, f, assess, err := encodeStatic(schema, name, t, cfg)
	if err != nil {
		return dst, err
	}
	e := &encoder{schema: schema, backend: backend, decimals: cfg.decimals, branches: cfg.branches, out: dst}
	if err := encoded[writer](f).write(e, v, writeAt{el: decl, path: typePath(t)}); err != nil {
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
// in either channel validate/xmlsrc.Validate carries one in; or, ahead of
// item 8, item 7's constructEntityType decline for the first violation whose
// chain holds a cvc-simple-type verdict (undeclaredEntity).
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
		if i := slices.IndexFunc(res.Violations(), undeclaredEntity); i >= 0 {
			return encodeDecline(constructEntityType, fmt.Sprintf("an ·ENTITY value· held raw, which the document Append writes, having no DTD, declares no unparsed entity for: %v", res.Violations()[i]))
		}
		return &invalidValueError{&chargedError{violations: res.Violations()}}
	}
	return verdict(res)
}

// ruleSimpleType is String Valid (Structures §3.16.4 cvc-simple-type), whose
// verdict package validate charges, wrapped under a delegating rule, only for
// its clause 3, an ·ENTITY value· that is no ·declared entity name·.
const ruleSimpleType xsderr.Rule = "cvc-simple-type"

// undeclaredEntity reports whether v's chain, v included, holds an
// *xsderr.Error of ruleSimpleType.
func undeclaredEntity(v *xsderr.Error) bool {
	for err := error(v); err != nil; err = errors.Unwrap(err) {
		if rule, _ := xsderr.RuleOf(err); rule == ruleSimpleType {
			return true
		}
	}
	return false
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
// implementers are exactly the arms the slice the package doc's "# What encodes" states
// reaches: the root arms, *simpleRoot, *boundBinding, *boundChoiceRoot and rawContent,
// writing the element at.el; and the content arms, *boundSequence, *boundContentChoice,
// boundOccurrence and boundFlattened, writing at.el's content, each member's elements by
// its bound state (memberWriter). Append reaches each through encoded.
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
// a fieldSink's elementReader, the memberWriter capability of a boundMember, the
// flatHolder capability of a contentMember, the emptier capability of a branchArm,
// or a closed sum's arm, a boundMember's boundParticle or fieldSink, a fieldArm's
// scalarArm or a scalarElem's atomic. It is the write's one assertion of the binder's
// arms. Invariant: where item 4 of [Append]'s doc admits a construct, every arm the
// binder builds for it that the write reaches is such a C. A failed assertion breaks
// it, an internal bug, never input, and panics naming x's type.
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
// tag, then each attribute field's attribute, in field order, named by
// writeRawAttribute, the fields after the content's (binding.lead,
// elementContent.width), a nil *V's written as none, and so is a nil union
// interface of a use whose {required} is false and to which no value
// constraint supplies a value (supply), each written value held to its use's
// and its declaration's fixed {value constraint} (encoder.attributeFixed),
// then, for a type with an {attribute wildcard}, the attributes its field
// holds (encoder.wildcardAttributes), then the content: for simple content the
// value field's text (texter), computed before the start tag's '>', and
// otherwise the content's elements (b.content's write), an element with no
// content written as an empty-element tag; or nothing, where v holds itself
// (encoder.enter).
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
		// meaning absent (codegen's package doc, "# Attributes"), but where a
		// value constraint supplies the attribute's value, which Decode then
		// holds there (Structures §3.4.5.1 sic-attrDefault), so that a nil one
		// reaches union, item 6 of Append's doc.
		if f.Kind() == reflect.Interface && f.IsNil() && !a.use.Required() && a.supplied == nil {
			continue
		}
		w, err := e.scalarText(a.scalar, f, fpath, simpleSubject(a.st, attributeSubject(a.decl)))
		if err != nil {
			return err
		}
		e.attributeFixed(a, w, fpath)
		e.writeRawAttribute(a.decl.Name(), w.text, nil)
	}
	if bd.anyAttribute != nil {
		i := first + len(bd.attrs)
		e.wildcardAttributes(bd, v.Field(i), at.path+"."+v.Type().Field(i).Name, at.el)
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
// A union's nil interface is item 6 there: a value field is never absent. The
// text is held to at.el's {value constraint} under that type
// (encoder.elementConstraint), the type cvc-elt clause 5.2.2.2.2 compares it
// under.
func (b boundValue) text(e *encoder, v reflect.Value, at writeAt) (string, error) {
	s := encoded[scalarArm](b.value.value.arm).scalar
	path := at.path + "." + v.Type().Field(b.index).Name
	subject := contentSubject(typeSubject(at.el, at.typ), s.st)
	w, err := e.scalarText(s, v.Field(b.index), path, subject)
	if err != nil {
		return "", err
	}
	e.elementConstraint(at.el, s.st, w, path, subject)
	return w.text, nil
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
	alts := c.root.sealed.alts
	check := e.attribution(c.root.index, at.el, c.root.typ, altsFlatten(alts))
	if err := writeChoice(e, alts, c.choice, v, writeAt{el: at.el, typ: c.root.typ, path: at.path}, contentChoiceSubject(at.el, c.root.typ), attributionSlot{check: check}); err != nil {
		return err
	}
	check.end(e, at.path)
	e.endTag(name.Local, start, outer, mark)
	return nil
}

// write writes the elements of each of b's members from its field of v, each
// member by its bound state, in the order sequence.writeBound states, under
// the content's attributionCheck (encoder.attribution), its first target the
// sequence's base.
func (b *boundSequence) write(e *encoder, v reflect.Value, at writeAt) error {
	check := e.attribution(b.sequence.index, at.el, at.typ, membersFlatten(b.sequence.members))
	if err := b.sequence.writeBound(e, b.members, v, at, attributionSlot{check: check, target: b.sequence.base}); err != nil {
		return err
	}
	check.end(e, at.path)
	return nil
}

// writeBound writes the elements of each of s's members from its field of v,
// the struct holding them, bound[i] member i's bound state: an all group's
// (sequence.all) by writeAll, any other sequence's in member order
// (writeMembers).
func (s *sequence) writeBound(e *encoder, bound []boundMember, v reflect.Value, at writeAt, slot attributionSlot) error {
	if s.all {
		return writeAll(e, bound, v, at, slot)
	}
	return writeMembers(e, bound, v, at, slot)
}

// writeMembers writes the elements of each member of a sequence, of a
// choice's alternative or of an occurrence field's item from its field of v,
// the struct holding them, in member order (writeMembersWhere).
func writeMembers(e *encoder, bound []boundMember, v reflect.Value, at writeAt, slot attributionSlot) error {
	return writeMembersWhere(e, bound, v, at, slot, func(boundMember) bool { return true })
}

// writeAll writes the elements of each member of an all group, the content's
// spliced one or an occurrence field's item, from its field of v: every
// element particle's member first, in member order, then every wildcard
// particle's, in member order (particleWriter.wildcard), each at the
// attribution target member order gives it (writeMembersWhere). Each order
// lies in the all group's language, the interleave of its particles'
// (§3.8.4.1.3); the elements go first because a child goes to the wildcard
// particle only where no element particle could take it (codegen/doc.go
// "# Particles"; xsd.Schema.ContentMatcher's order, xsd/contentmatcher.go),
// and slot.check, set wherever the group holds a wildcard particle
// (particleSite.matcher), stays the check of every value whose attribution
// the order does not reproduce (attributionCheck). An all group's members are
// each an element or wildcard particle's (§3.8.6.2 cos-all-limited), its
// {open content} field declined by item 4 of [Append]'s doc (contentEncodes),
// so that any other member is an internal bug, which panics, as encoded's
// assertion does.
func writeAll(e *encoder, bound []boundMember, v reflect.Value, at writeAt, slot attributionSlot) error {
	for _, wildcards := range [...]bool{false, true} {
		if err := writeMembersWhere(e, bound, v, at, slot, func(b boundMember) bool {
			return encoded[boundParticle](b).writer.wildcard() == wildcards
		}); err != nil {
			return err
		}
	}
	return nil
}

// writeMembersWhere writes the elements of each member of bound keep reports
// true for, in member order, from its field of v, each by its own bound state
// (memberWriter), each taken by slot.check: member i's first attribution
// target is slot.target, the first member's, plus the targets every earlier
// member takes, kept or not, one per sink (boundMember.sinks).
func writeMembersWhere(e *encoder, bound []boundMember, v reflect.Value, at writeAt, slot attributionSlot, keep func(boundMember) bool) error {
	target := slot.target
	for _, b := range bound {
		if keep(b) {
			if err := encoded[memberWriter](b).writeMember(e, v, at, attributionSlot{check: slot.check, target: target}); err != nil {
				return err
			}
		}
		target += len(b.sinks())
	}
	return nil
}

// A memberWriter is a boundMember Append writes, the bound state of a member
// of a sequence, a choice's alternative or an occurrence field's item, which
// holds its own member: writeMember writes the member's elements from its
// field of v, the struct holding it, at.path being v's Go path, its first
// attribution target slot.target in slot.check, and returns a backend fault.
// Its implementers are boundParticle, an element or wildcard particle's, and
// the bound states of a nested model group particle: boundNestedChoice, a
// choice occurring once; boundFlatSlice, a flattened particle; and
// boundNestedOccurrence, an occurrence field. A fieldSink is none: item 4
// declines the open-content field's {open content} (contentEncodes) and a
// governing type's branch field's {type table}, and Append writes a
// substitution group member's branch field through its reader
// (groupReader.writeElement).
type memberWriter interface {
	writeMember(e *encoder, v reflect.Value, at writeAt, slot attributionSlot) error
}

// writeMember writes the elements of b's particle from field b.sink.index of
// v (particleWriter.writeElements), read by b.sink, taken at slot.
func (b boundParticle) writeMember(e *encoder, v reflect.Value, at writeAt, slot attributionSlot) error {
	f, field := fieldOf(v, b.sink.index, at)
	return b.writer.writeElements(e, b.sink, f, field, slot)
}

// fieldOf returns field index of v, the struct at.path names, and its own
// writeAt, its Go path the struct's and its name.
func fieldOf(v reflect.Value, index int, at writeAt) (reflect.Value, writeAt) {
	return v.Field(index), writeAt{el: at.el, typ: at.typ, path: at.path + "." + v.Type().Field(index).Name}
}

// writeMember writes the branch field b.index of v holds, the nested choice's
// sealed interface (writeChoice), its alternatives' targets from slot.target.
func (b boundNestedChoice) writeMember(e *encoder, v reflect.Value, at writeAt, slot attributionSlot) error {
	f, field := fieldOf(v, b.index, at)
	return writeChoice(e, b.choice.sealed.alts, b.bound, f, field, "a nested choice particle of "+typeSubject(at.el, at.typ), slot)
}

// writeMember writes field b.index of v, the nested flattened particle's
// slice, one element per item in slice order (writeFlatItems), alternative j
// taking target slot.target+j.
func (b boundFlatSlice) writeMember(e *encoder, v reflect.Value, at writeAt, slot attributionSlot) error {
	f, field := fieldOf(v, b.index, at)
	return writeFlatItems(e, b.bound, f, field, "the nested flattened particle of "+typeSubject(at.el, at.typ), slot)
}

// writeMember writes field b.index of v, the nested occurrence field, as
// writeOccurring states, its length held to the occurrence its shape was
// computed at (nestedOccurrence.held) under the item's floor
// (boundItem.floor) and its error naming the particle's own where the two
// differ: each item it holds, in slice order, as its model
// group's particles in order (boundItem.writeItem), its targets from
// slot.target, under slot.check, the content's.
func (b boundNestedOccurrence) writeMember(e *encoder, v reflect.Value, at writeAt, slot attributionSlot) error {
	o := b.occurrence
	subject := fmt.Sprintf("a nested %s model group particle of %s", o.group.Compositor(), typeSubject(at.el, at.typ))
	f, field := fieldOf(v, b.index, at)
	return e.writeOccurring(o.shape, o.own, o.held, b.item.floor(), subject, f, field, func(item reflect.Value, path string) error {
		return b.item.writeItem(e, item, writeAt{el: at.el, typ: at.typ, path: path}, subject, slot)
	})
}

// A flatHolder is a contentMember whose content's attributionCheck Append
// reads: holdsFlat reports whether the member holds a nested flattened
// particle's slice, itself or at any depth inside it, whose items' order is
// the caller's, so that the content model's Matcher must check it
// (encoder.attribution). Its implementers are field, false, and through it
// elementMember and wildcardMember; *flatSlice, true; *choice, where one of
// its alternatives' members holds one (altsFlatten); and *nestedOccurrence,
// false, since item 4 of [Append]'s doc declines a flattened particle inside
// an occurrence field's item (encodeWalk.groupEncodes), which writeFlatItems'
// nil-check panic guards.
type flatHolder interface {
	holdsFlat() bool
}

// holdsFlat is false: a field holds no model group.
func (field) holdsFlat() bool { return false }

// holdsFlat is true: f is a flattened particle's slice.
func (*flatSlice) holdsFlat() bool { return true }

// holdsFlat is whether one of c's alternatives holds one (altsFlatten).
func (c *choice) holdsFlat() bool { return altsFlatten(c.sealed.alts) }

// holdsFlat is false: no flattened particle lies inside an occurrence field's
// item that item 4 admits.
func (*nestedOccurrence) holdsFlat() bool { return false }

// membersFlatten reports whether one of members holds a nested flattened
// particle's slice (flatHolder).
func membersFlatten(members []contentMember) bool {
	return slices.ContainsFunc(members, func(m contentMember) bool { return encoded[flatHolder](m).holdsFlat() })
}

// altsFlatten reports whether the members of one of alts, a choice's
// alternatives, hold a nested flattened particle's slice (membersFlatten).
func altsFlatten(alts []branchArm) bool {
	return slices.ContainsFunc(alts, func(a branchArm) bool { return membersFlatten(a.branchMembers()) })
}

// write writes the branch the choice's sealed interface holds, field b.index
// of v (writeChoice), then the elements of each member of the rest, whose
// fields follow it, each by its bound state (writeMembers), all under the
// content's attributionCheck (encoder.attribution).
func (b *boundContentChoice) write(e *encoder, v reflect.Value, at writeAt) error {
	rest := b.content.rest
	check := e.attribution(rest.index, at.el, at.typ, b.content.choice.holdsFlat() || membersFlatten(rest.members))
	f, field := fieldOf(v, b.index, at)
	if err := writeChoice(e, b.content.choice.sealed.alts, b.choice, f, field, contentChoiceSubject(at.el, at.typ), attributionSlot{check: check}); err != nil {
		return err
	}
	if err := writeMembers(e, b.rest, v, at, attributionSlot{check: check, target: rest.base}); err != nil {
		return err
	}
	check.end(e, at.path)
	return nil
}

// write writes the occurrence field of v, field b.index, as writeOccurring
// states, the field's shape and its particle's occurrence read from the
// occurrence (occurrence.shape, occurrence.own), its length's floor from the
// item (boundItem.floor): each item it holds, in slice order, as
// boundItem.writeItem writes its model group's particles, all under one
// attributionCheck of the content (encoder.attribution), since the content
// model's Matcher spans every occurrence, the item's targets from 0. An item
// holds no flattened particle (flatHolder).
func (b boundOccurrence) write(e *encoder, v reflect.Value, at writeAt) error {
	o := b.occurrence
	check := e.attribution(o.index, at.el, at.typ, false)
	subject := fmt.Sprintf("the %s content particle of %s", o.group.Compositor(), typeSubject(at.el, at.typ))
	group := contentChoiceSubject(at.el, at.typ)
	f, field := fieldOf(v, b.index, at)
	if err := e.writeOccurring(o.shape, o.own, o.own, b.bound.floor(), subject, f, field, func(item reflect.Value, path string) error {
		return b.bound.writeItem(e, item, writeAt{el: at.el, typ: at.typ, path: path}, group, attributionSlot{check: check})
	}); err != nil {
		return err
	}
	check.end(e, at.path)
	return nil
}

// contentChoiceSubject names the content choice of el's complex type typ,
// whose item 6 errors of [Append]'s doc name it so: a choice occurring once,
// T itself or a content field, or one occurrence of a content choice held in
// an occurrence field.
func contentChoiceSubject(el xsd.ElementDeclaration, typ xsd.ComplexType) string {
	return "the content choice of " + typeSubject(el, typ)
}

// writeItem writes the elements of each member of the sequence or all group
// from its field of v, the item's struct, member i's field i, each by its
// bound state, in the order sequence.writeBound states, the first target
// slot.target, the sequence's base.
func (b boundSequenceItem) writeItem(e *encoder, v reflect.Value, at writeAt, _ string, slot attributionSlot) error {
	return b.sequence.writeBound(e, b.members, v, at, slot)
}

// floor is floorMin: no occurrence of a sequence or all group whose slice
// writeOccurring's length check reads matches the empty sequence. That slice
// is a shapeRepeated field's, of {max occurs} above 1, which no all group
// takes (§3.8.6.2 cos-all-limited) and no sequence one occurrence of which is
// emptiable takes either, codegen flattening it (codegen/doc.go "# Particles",
// goshape.ParticleHold's flattens, its second gate).
func (boundSequenceItem) floor() lengthFloor { return floorMin }

// floor is floorWaived where one occurrence of the choice matches the empty
// sequence (choiceEmptiable), and floorMin otherwise.
func (c boundChoiceItem) floor() lengthFloor {
	if choiceEmptiable(c.choice.sealed.alts) {
		return floorWaived
	}
	return floorMin
}

// writeItem writes the branch v, a value of the choice's sealed interface,
// holds (writeChoice), the choice named group, its alternatives' targets from
// slot.target. A nil v is item 6 of [Append]'s doc, whatever the choice's
// alternatives, and writes nothing: an item is written only as an item of a
// []I, since writeOccurring writes no nil V of {min occurs} 0, and Decode
// returns no nil item, each item holding the children of one non-empty
// sub-sequence of the ·partition· (boundOccurrence.fill), so that the empty
// match would leave the item out of the slice Decode returns.
func (c boundChoiceItem) writeItem(e *encoder, v reflect.Value, at writeAt, group string, slot attributionSlot) error {
	if v.IsNil() {
		e.invalidate(fmt.Errorf("codec: Append: %s is nil, an item holding no child element, which Decode never returns in the occurrence field of %s", at.path, group))
		return nil
	}
	return writeChoice(e, c.choice.sealed.alts, c.bound, v, at, group, slot)
}

// writeChoice writes the branch v, a value of the sealed interface of a
// choice occurring once, the content's or a nested one, or of one occurrence
// of a choice held in an occurrence field, whose alternatives are alts and
// which subject names, holds, bc that interface as item 5's check bound it: its
// branches[j], the branch type placed at alts[j] (sealed.checkSite), is
// matched to v's dynamic type, and the members of alts[j] are written from the
// branch struct's fields, in field order, each by its bound state in
// branches[j] (writeMembers), the first target slot.target, the choice's
// first, plus the targets of every earlier alternative (branchTargets), under
// slot.check, the attributionCheck of the content the choice is written into,
// nil for none. The nil interface writes nothing, the empty match, but is item
// 6 of [Append]'s doc where no alternative is emptiable (choiceEmptiable), and
// so is a dynamic type bc places at no alternative, a pointer to a branch
// among them.
func writeChoice(e *encoder, alts []branchArm, bc *boundChoice, v reflect.Value, at writeAt, subject string, slot attributionSlot) error {
	if v.IsNil() {
		if !choiceEmptiable(alts) {
			e.invalidate(fmt.Errorf("codec: Append: %s is nil, the empty match, which %s does not accept: %s", at.path, subject, unemptiable(alts)))
		}
		return nil
	}
	b := v.Elem()
	j := bc.branchOf(b.Type())
	if j < 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, which the WithBranches set of %s places at no alternative of %s, so that Decode returns no such value", at.path, b.Type(), v.Type(), subject))
		return nil
	}
	first := slot.target
	for _, a := range alts[:j] {
		first += branchTargets(a)
	}
	branch := writeAt{el: at.el, typ: at.typ, path: at.path + ".(" + typePath(b.Type()) + ")"}
	return writeMembers(e, bc.branches[j].members, b, branch, attributionSlot{check: slot.check, target: first})
}

// branchOf returns the index in bc.branches of the branch whose type is t, the
// dynamic type of a value of bc's sealed interface, and -1 where the
// WithBranches set placed none at an alternative, a pointer to a branch among
// them: the alternative Decode would have returned the value from.
func (bc *boundChoice) branchOf(t reflect.Type) int {
	return slices.IndexFunc(bc.branches, func(bb boundBranch) bool { return bb.t == t })
}

// write writes the slice field of v, field b.index, which holds at.typ's
// flattened content {particle}, as writeFlatItems states, all under one
// attributionCheck of the content (flattenedCheck), never nil, alternative j
// taking target j, the index its place in b.flattened.places holds, then
// records item 6 of [Append]'s doc where the content model is not accepting at
// the slice's end (attributionCheck.end). b.open is never written: item 4
// declines {open content} (contentEncodes).
func (b boundFlattened) write(e *encoder, v reflect.Value, at writeAt) error {
	check := e.flattenedCheck(b.flattened, at)
	f, field := fieldOf(v, b.index, at)
	if err := writeFlatItems(e, b.bound, f, field, "the flattened content of "+check.subject, attributionSlot{check: check}); err != nil {
		return err
	}
	check.end(e, at.path)
	return nil
}

// writeFlatItems writes slice, at at, the slice field of a flattened
// particle, the content {particle} or nested, which subject names, bc its
// interface bound, one element per item in slice order: each item's branch,
// matched to its dynamic type (boundChoice.branchOf), writes its
// alternative's one field by the boundParticle the branch's one member is
// bound to (boundParticle.writeMember), alternative j taking target slot.target+j
// in slot.check, the content's attributionCheck, since the order of a
// flattened slice's items is the caller's. It records item 6 of [Append]'s
// doc for the slice: a nil item and an item of a dynamic type placed at no
// alternative, and the content model's verdict on each item through
// attributionCheck.take; the slice's length it never reads, since no
// writeOccurring runs over the slice. A nil slot.check, a content whose
// attributionCheck item 4 left nil (encoder.attribution, flatHolder), is an
// internal bug, which panics, as encoded's assertion does.
func writeFlatItems(e *encoder, bc *boundChoice, slice reflect.Value, at writeAt, subject string, slot attributionSlot) error {
	if slot.check == nil {
		panic(fmt.Sprintf("codec: Append reached %s with no attributionCheck of its content", subject))
	}
	for i := range slice.Len() {
		item, ipath := slice.Index(i), at.path+"["+strconv.Itoa(i)+"]"
		if item.IsNil() {
			e.invalidate(fmt.Errorf("codec: Append: %s is nil, an item of %s that holds no child element, so that Decode returns no such item", ipath, subject))
			continue
		}
		br := item.Elem()
		j := bc.branchOf(br.Type())
		if j < 0 {
			e.invalidate(fmt.Errorf("codec: Append: %s holds a %s, which the WithBranches set of %s places at no alternative of %s, so that Decode returns no such value", ipath, br.Type(), item.Type(), subject))
			continue
		}
		branch := writeAt{el: at.el, typ: at.typ, path: ipath + ".(" + typePath(br.Type()) + ")"}
		if err := encoded[boundParticle](bc.branches[j].members[0]).writeMember(e, br, branch, attributionSlot{check: slot.check, target: slot.target + j}); err != nil {
			return err
		}
	}
	return nil
}

// An emptier is a choice's alternative as Append reads its emptiability, an
// elementMember, a wildcardMember or a groupBranch: emptiable reports whether
// one occurrence of the alternative's particle matches the empty sequence,
// which §3.9.6.3 cos-group-emptiable decides by its clause 1, the particle's
// own {min occurs} 0, or its clause 2, its term's range minimum 0. Clause 2
// never holds for an alternative of a choice codegen does not flatten, which
// is every choice writeChoice writes: an element or wildcard term has range
// minimum 1, and goshape.ParticleHold flattens a choice one of whose
// alternatives' terms is emptiable, its flattens' first gate (codegen/doc.go
// "# Particles"). So each answers own.Min() == 0.
type emptier interface {
	emptiable() bool
}

// emptiable is whether m.own's {min occurs} is 0.
func (m elementMember) emptiable() bool { return m.own.Min() == 0 }

// emptiable is whether m.own's {min occurs} is 0.
func (m wildcardMember) emptiable() bool { return m.own.Min() == 0 }

// emptiable is whether g.own's {min occurs} is 0.
func (g groupBranch) emptiable() bool { return g.own.Min() == 0 }

// choiceEmptiable reports whether one occurrence of the choice whose
// alternatives are alts matches the empty sequence: one of them is emptiable
// (emptier, §3.9.6.3 cos-group-emptiable).
func choiceEmptiable(alts []branchArm) bool {
	return slices.ContainsFunc(alts, func(a branchArm) bool { return encoded[emptier](a).emptiable() })
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
// writes, a contentMember's or a branchArm's, held by the boundParticle it
// binds (boundParticle.writeMember): writeElements writes the elements of f,
// the field of its particle, read by s, the field's sink, at at, each taken
// by slot's attributionCheck at slot's target, and returns a backend fault;
// wildcard reports whether the particle is a wildcard particle, which
// writeAll reads. Its implementers are elementMember and wildcardMember.
type particleWriter interface {
	writeElements(e *encoder, s fieldSink, f reflect.Value, at writeAt, slot attributionSlot) error
	wildcard() bool
}

// wildcard is false: the member is an element particle's.
func (elementMember) wildcard() bool { return false }

// wildcard is true: the member is a wildcard particle's.
func (wildcardMember) wildcard() bool { return true }

// writeElements writes the elements of f, the field of m's particle, as
// writeOccurring states, each by the elementWriter s, the field's sink, reads
// it with, as an element of m.decl or, for a groupReader, of the member its
// value holds, taken by slot.
func (m elementMember) writeElements(e *encoder, s fieldSink, f reflect.Value, at writeAt, slot attributionSlot) error {
	w := encoded[elementWriter](s.read)
	return e.writeOccurring(m.shape, m.own, m.held, floorMin, "the particle of "+declSubject(m.decl), f, at, func(v reflect.Value, path string) error {
		return w.writeElement(e, m.decl, v, path, slot)
	})
}

// writeElements writes the elements of f, the field of m's particle, as
// writeOccurring states, each a value of AnyElement's shape written raw
// (encoder.rawElement) in the hold of m.process, skipMatch for skip and
// assessedMatch for strict and lax, its name taken by slot.check
// (attributionCheck.take) once rawName admits it, then, for strict and lax,
// held to what governs it (encoder.matchGoverned).
func (m wildcardMember) writeElements(e *encoder, _ fieldSink, f reflect.Value, at writeAt, slot attributionSlot) error {
	var hold rawHold = skipMatch(m.subject)
	if m.process != xsd.ProcessSkip {
		hold = assessedMatch(m.subject)
	}
	return e.writeOccurring(m.shape, m.own, m.held, floorMin, m.subject, f, at, func(v reflect.Value, path string) error {
		name, ok := e.rawName(v, path)
		if !ok {
			return nil
		}
		slot.check.take(e, name, slot.target, path)
		typed := m.process != xsd.ProcessSkip && e.matchGoverned(name, v.Field(2), m.process == xsd.ProcessStrict, path, hold, at)
		e.rawElement(name, v.Field(2), path, hold, typed)
		return nil
	})
}

// matchGoverned holds the element name, the match of a strict wildcard
// particle where strict holds and of a lax one otherwise, held at path in
// hold, its content x, written into at.typ, to what governs it, in this order
// (Structures §3.3.4.6 key-governing-ed clause 3, key-governing-type-elem):
// a top-level element declaration of its name, against which the document's
// assessment holds it; else a ·locally declared type· in at.typ
// (xsd.Schema.LocallyDeclaredElementType), item 7 of [Append]'s doc,
// constructLocallyTypedMatch, where no decline is recorded yet; else, for
// strict, an xsi:type x's Attributes hold, the assessment holding it to that
// type, and with none item 6, its parent invalid (§3.3.5.1 sic-e-outcome
// clause 1.1.3); and else, for lax, nothing. It reports whether the content
// lies in that region, which the captured-prefix check reads
// (encoder.capturedIn): for strict always, and for lax where a declaration, a
// locally declared type or an xsi:type governs it. Only the match itself has
// a locally declared type: below it no item does (§3.4.6.4 key-ldt-elem
// clause 1).
func (e *encoder) matchGoverned(name xsd.QName, x reflect.Value, strict bool, path string, hold rawHold, at writeAt) bool {
	if _, ok := e.schema.Element(name); ok {
		return true
	}
	if _, ok := e.schema.LocallyDeclaredElementType(at.typ, name); ok {
		if e.declined == nil {
			e.declined = encodeDecline(constructLocallyTypedMatch, fmt.Sprintf("element %s at %s, %s, governed by its locally declared type in %s", name.Display(), path, hold.in(), typeSubject(at.el, at.typ)))
		}
		return true
	}
	xsiTyped := holdsXsiType(x)
	if strict && !xsiTyped {
		e.invalidate(fmt.Errorf("codec: Append: %s holds element %s, %s, which no top-level element declaration names and whose Attributes hold no xsi:type, so that neither a declaration nor a type governs it and its parent is invalid (Structures §3.3.4.6 key-governing-ed clause 3, key-governing-type-elem clause 8, §3.3.5.1 sic-e-outcome clause 1.1.3)", path, name.Display(), hold.in()))
	}
	return strict || xsiTyped
}

// holdsXsiType reports whether x, a struct of AnyType's shape, holds xsi:type
// among its Attributes.
func holdsXsiType(x reflect.Value) bool {
	attrs := x.Field(xAttributes)
	for i := range attrs.Len() {
		if a := attrs.Index(i); a.Field(0).String() == xsd.XMLSchemaInstanceNS && a.Field(1).String() == "type" {
			return true
		}
	}
	return false
}

// writeOccurring writes the elements of f, the field of the particle subject
// names, occurring as own, at.path its Go path, each by write, by sh, the
// shape item 5 bound the field in (field.shape), computed at held, never by
// f's Kind, since a list's V is itself a slice: one per item of a []V, which
// is item 6 of [Append]'s doc where its length lies outside held under floor
// (lengthFloor.permits), the error naming own where a choice's alternative
// raised it, none for a nil *V, and one for a V, but none for a nil V where
// held's {min occurs} is 0, which shape V takes only over a sealed interface
// (goshape.ParticleShape), nil meaning absent.
func (e *encoder) writeOccurring(sh shape, own, held xsd.Occurs, floor lengthFloor, subject string, f reflect.Value, at writeAt, write func(v reflect.Value, path string) error) error {
	switch sh {
	case shapeOptional:
		if f.IsNil() {
			return nil
		}
		return write(f.Elem(), at.path)
	case shapeRepeated:
		if !floor.permits(held, f.Len()) {
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

// A lengthFloor is the lower bound writeOccurring holds a []V's length to:
// floorMin, held's {min occurs}, every field's but an occurrence field's
// whose item reads otherwise (boundItem.floor); and floorWaived, none, for
// the occurrence field of a choice one occurrence of which matches the empty
// sequence, whose slice's length codegen/doc.go "# Particles" states. Neither
// waives held's {max occurs}.
type lengthFloor int

const (
	floorMin lengthFloor = iota
	floorWaived
)

// permits reports whether n items lie within held under f: held.Permits(n),
// a count below held's {min occurs} read as that minimum for floorWaived.
func (f lengthFloor) permits(held xsd.Occurs, n int) bool {
	if f == floorWaived {
		n = max(n, held.Min())
	}
	return held.Permits(n)
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
	w := encoded[elementWriter](encoded[fieldSink](g.bound.branches[j].members[0]).read)
	return w.writeElement(e, g.members[j], br.Field(0), path+".("+typePath(br.Type())+")."+br.Type().Field(0).Name, slot)
}

// elementEncodes returns the error item 4 of [Append]'s doc states for the
// first construct of decl, the document element's declaration, outside the
// slice the package doc's "# What encodes" states, in the order the construct
// constants list, or a plain error for a component that does not resolve, and
// otherwise nil: encodeWalk.element's for decl. With nil it returns
// encodeWalk.assess, whether Append assesses what it writes (items 8 and 9).
// It reads the schema alone, never the binder, and builds nothing.
func elementEncodes(schema *xsd.Schema, decl xsd.ElementDeclaration) (bool, error) {
	w := &encodeWalk{schema: schema, walked: map[unitKey]bool{}}
	if err := w.element(decl); err != nil {
		return false, err
	}
	return w.assess, nil
}

// encodeWalk is one pass of elementEncodes: schema; walked, the complex types
// whose constructs it has examined or is examining, each under the key
// Decode's bind indexes its unit by (nestedKey); and assess, whether Append
// assesses what it writes, set where an element declaration the pass examined
// has {identity-constraint definitions} (encodeWalk.declEncodes) or a {type
// definition} that is xs:anyType itself (encodeWalk.element), where a simple
// type it examined is derived from xs:ID or xs:IDREF (encodeWalk.atomicEncodes),
// and where a wildcard particle it examined has {process contents} strict or
// lax (encodeWalk.particleEncodes), and never cleared, so that a type walked
// answers for its second reach too. An entry is stored before its type's content is
// examined, so that a later reach of the key, a recursive one included, examines
// nothing again: the first reach answers for the type, and every other component it
// reaches carries no key of its own only where it lies inside that type (bindRun).
// walked lives for one pass only and is a lookup index never iterated (STYLE D2);
// an entry left by a pass that declines is never read, since the decline ends the
// pass.
type encodeWalk struct {
	schema *xsd.Schema
	walked map[unitKey]bool
	assess bool
}

// element returns elementEncodes' error for decl, the document element's
// declaration or a particle's element declaration, examined at every reach:
// decl's own properties (encodeWalk.declEncodes), then its {type definition},
// which encodes as xs:anyType itself, its content written raw (rawContent's write),
// setting w.assess, is examined as a simple type (encodeWalk.simpleEncodes) or, for
// a complex type, declines by its {abstract} true, and else is examined by
// complexEncodes, unless walked already holds its key. complexEncodes and the
// functions under it read decl only to name a decline's subject, so that a type
// reached again through another declaration answers as it did.
func (w *encodeWalk) element(decl xsd.ElementDeclaration) error {
	if err := w.declEncodes(decl); err != nil {
		return err
	}
	td, ok := w.schema.ResolvedType(decl.TypeDefinition())
	if !ok {
		return fmt.Errorf("codec: the {type definition} of %s does not resolve", declSubject(decl))
	}
	if goshape.AnyTypeItself(td) {
		w.assess = true
		return nil
	}
	if st, isSimple := td.(*xsd.SimpleType); isSimple {
		return w.simpleEncodes(st, declSubject(decl))
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
// table}, an identity constraint whose paths icpath does not compile
// (encodeWalk.identityEncodes), a member of its {substitution group
// affiliations} that names no top-level element declaration of the schema, the
// first in document order, an ·absent· member Finalize keeps (§5.3), under
// which validate assesses the element laxly, as if cvc-elt clause 1 had
// failed, and {abstract} true, which no element validates against (cvc-elt
// clause 2): the document element's is item 3 of [Append]'s doc, found before
// declEncodes runs, so that only a particle's declines so, one the parser
// never builds but xsd.NewElementDeclaration does. A decl whose
// {identity-constraint definitions} it admits sets w.assess.
func (w *encodeWalk) declEncodes(decl xsd.ElementDeclaration) error {
	if goshape.Nillable(decl) {
		return encodeDecline(constructNillable, declSubject(decl))
	}
	if _, has := decl.TypeTable(); has {
		return encodeDecline(constructTypeTable, declSubject(decl))
	}
	for _, ic := range decl.IdentityConstraints() {
		if err := w.identityEncodes(decl, ic); err != nil {
			return err
		}
		w.assess = true
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
// {simple type definition} as encodeWalk.typeEncodes examines it, named after ct as
// Decode's simpleValueOf names it (contentSubject), and for element-only or
// mixed content its content {particle}, {max occurs} 0 declined before its
// hold is asked (#3794), then each particle in order, of a flattened one
// each basic particle it reaches (contentEncodes); then each attribute use
// codegen gives a field (goshape.FieldAttributeUses), then its {assertions}.
// An {attribute wildcard} declines nothing here: what its field holds is
// examined per value (encoder.wildcardAttributes).
func (w *encodeWalk) complexEncodes(decl xsd.ElementDeclaration, ct xsd.ComplexType) error {
	subject := typeSubject(decl, ct)
	// {content type} is a closed sum, so the switch is STYLE T2's sealed-sum
	// exception.
	switch content := ct.ContentType().(type) {
	case xsd.SimpleContent:
		if err := w.typeEncodes(content.SimpleType, contentSubject(subject, content.SimpleType)); err != nil {
			return err
		}
	case xsd.ElementContent:
		if err := w.contentEncodes(ct, content, subject); err != nil {
			return err
		}
	}
	for _, u := range goshape.FieldAttributeUses(ct) {
		if err := w.attributeEncodes(u); err != nil {
			return err
		}
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
// examines it; or each particle of a spliced sequence or all group, of a
// choice occurring once that codegen does not flatten, or of a sequence,
// choice or all group held in an occurrence field, as particleEncodes admits
// it at its particleSite; then, where a particle the walk examined sets the
// site's matcher, a wildcard particle, a nested flattened particle or a nested
// occurrence field at any depth, whose children Append attributes through a
// Matcher (attributionCheck), ct's content model where
// xsd.Schema.ContentMatcher does not track it.
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
		if group.Compositor() == xsd.CompositorChoice {
			label = alternative
		}
	case goshape.Spliced:
		// A spliced sequence's or all group's particles keep label.
	}
	var matcher bool
	site := particleSite{item: hold == goshape.Occurrence, alternative: label == alternative, matcher: &matcher}
	for i, p := range group.Particles() {
		if err := w.particleEncodes(p, fmt.Sprintf(label, i+1, subject), site); err != nil {
			return err
		}
	}
	if !matcher {
		return nil
	}
	if _, tracked := w.schema.ContentMatcher(ct); !tracked {
		return encodeDecline(constructUntrackedModel, subject)
	}
	return nil
}

// particleSite is where particleEncodes examines a particle of a content
// model: item, whether the particle lies inside an occurrence field's item,
// the content {particle}'s or a nested one's, as Decode's bind carries it
// (sequence.nested, bindItem); alternative, whether it is a choice's
// alternative, whose model group's hold is asked at goshape.BranchOccurs'
// occurrence (bindChoice); and matcher, set where Append's write builds a
// Matcher of the content model (encoder.attribution): for a wildcard particle
// at any depth, for a nested flattened particle, whose bind gives the content
// its matcherSlot (sequence.flatten), and for a nested occurrence field,
// whose bind builds one (sequence.occurrence), so that contentEncodes
// declines a content model xsd.Schema.ContentMatcher does not track.
type particleSite struct {
	item, alternative bool
	matcher           *bool
}

// occurs is the occurrence a model group particle p at s is read at: its own,
// or goshape.BranchOccurs' for a choice's alternative.
func (s particleSite) occurs(p xsd.Particle) xsd.Occurs {
	if s.alternative {
		return goshape.BranchOccurs(p.Occurs())
	}
	return p.Occurs()
}

// particlesEncode returns particleEncodes' error for each of particles, the
// {particles} of a model group subject names, in order, particle i named kind
// i+1 of subject ("particle" or "alternative"), at site.
func (w *encodeWalk) particlesEncode(particles []xsd.Particle, kind, subject string, site particleSite) error {
	for i, p := range particles {
		if err := w.particleEncodes(p, fmt.Sprintf("%s %d of %s", kind, i+1, subject), site); err != nil {
			return err
		}
	}
	return nil
}

// groupEncodes returns particleEncodes' error for a nested model group
// particle of group, the inline one or the {model group} of the definition a
// <group ref> names, occurring as occurs, its own or, for a choice's
// alternative, goshape.BranchOccurs' (§3.7.2 declare-namedModelGroup), at
// site, subject naming it, by the hold goshape.ParticleHold answers for it,
// as Decode's sequence.nested binds it. Spliced examines group's particles in
// its place; OnceChoice examines its alternatives, as alternatives; Flattened,
// inside an occurrence field's item, declines by constructNestedGroup, as
// Decode's bind does, since the item's partition reads no repetition of a
// model group inside it, and otherwise examines each basic particle of its
// reach (reachEach); Occurrence, inside an item, declines by
// constructNestedGroup too, whatever its occurrence, and otherwise examines
// its particles, a choice's as alternatives, each inside the field's item.
// Each particle examined inside group keeps site's item, and a Flattened or
// Occurrence group sets site.matcher. ParticleHold's errors are plain,
// prefixed by subject.
func (w *encodeWalk) groupEncodes(group xsd.ModelGroup, occurs xsd.Occurs, subject string, site particleSite) error {
	hold, err := goshape.ParticleHold(w.schema, group, occurs)
	if err != nil {
		return fmt.Errorf("codec: %s: %w", subject, err)
	}
	switch hold {
	case goshape.Spliced:
		return w.particlesEncode(group.Particles(), "particle", subject, particleSite{item: site.item, matcher: site.matcher})
	case goshape.OnceChoice:
		return w.particlesEncode(group.Particles(), "alternative", subject, particleSite{item: site.item, alternative: true, matcher: site.matcher})
	case goshape.Flattened:
		if site.item {
			return encodeDecline(constructNestedGroup, subject)
		}
		*site.matcher = true
		return w.reachEach(group, "the flattened "+subject, subject)
	case goshape.Occurrence:
		if site.item {
			return encodeDecline(constructNestedGroup, subject)
		}
		*site.matcher = true
		inner := particleSite{item: true, alternative: group.Compositor() == xsd.CompositorChoice, matcher: site.matcher}
		kind := "particle"
		if inner.alternative {
			kind = "alternative"
		}
		return w.particlesEncode(group.Particles(), kind, subject, inner)
	}
	// The zero Hold is an internal bug, never input.
	panic(fmt.Sprintf("codec: invalid goshape.Hold %d", hold))
}

// reachEncodes returns contentEncodes' error for group, the model group of
// ct's content {particle}, which codegen flattens, subject naming ct: each
// basic particle goshape.Reach visits (reachEach), the subject counting the
// alternatives the slice's sealed interface holds (bindReach), as bindReach's
// error is prefixed; then ct's content model where xsd.Schema.ContentMatcher
// does not track it, with or without a wildcard particle in the reach, since
// Append attributes every child of a flattened content model through a
// Matcher (flattenedCheck), as Decode's bindFlattened does.
func (w *encodeWalk) reachEncodes(ct xsd.ComplexType, group xsd.ModelGroup, subject string) error {
	if err := w.reachEach(group, "the flattened content {particle} of "+subject, subject); err != nil {
		return err
	}
	if _, tracked := w.schema.ContentMatcher(ct); !tracked {
		return encodeDecline(constructUntrackedModel, subject)
	}
	return nil
}

// reachEach returns particleEncodes' error for each basic particle
// goshape.Reach visits in group, a model group codegen flattens, the content
// {particle}'s or a nested one's, in reach order, alternative n named
// "alternative n of" flattened, which names the flattened particle, a model
// group particle with {max occurs} 0 declining by particleEncodes too, and
// goshape.Reach's own error plain, prefixed by subject.
func (w *encodeWalk) reachEach(group xsd.ModelGroup, flattened, subject string) error {
	alt := 0
	// visitErr is particleEncodes' error, which already carries its context
	// and, set, ended the walk as goshape.Reach's error, so that only
	// goshape.Reach's own is wrapped. The reach's site needs no matcher of its
	// own: its caller builds the flattened particle's.
	var visitErr error
	var matcher bool
	err := goshape.Reach(w.schema, group, func(p xsd.Particle, _ []int) error {
		alt++
		visitErr = w.particleEncodes(p, fmt.Sprintf("alternative %d of %s", alt, flattened), particleSite{matcher: &matcher})
		return visitErr
	})
	if visitErr != nil {
		return visitErr
	}
	if err != nil {
		return fmt.Errorf("codec: %s: %w", subject, err)
	}
	return nil
}

// particleEncodes returns elementEncodes' error for the particle p of a
// spliced content sequence, of a content choice's alternative, of a
// sequence or choice held in an occurrence field, of a nested model group
// (encodeWalk.groupEncodes), or a basic particle a flattened content
// {particle} or a nested one reaches (reachEach), at site, subject naming p:
// {max occurs} 0, then a model group term, inline or the {model group} of the
// definition a <group ref> names (goshape.ModelGroupTerm), as groupEncodes
// examines it at the occurrence site reads it at (particleSite.occurs), a
// <group ref> naming none a plain error, prefixed by subject; a wildcard term,
// which sets site's matcher, sets w.assess with {process contents} strict or
// lax, and is examined no further; and otherwise the element
// declaration, examined as the document element's is (encodeWalk.element). The
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
// through encodeWalk.walked. {term} is a closed sum, so the switches are STYLE
// T2's sealed-sum exception.
func (w *encodeWalk) particleEncodes(p xsd.Particle, subject string, site particleSite) error {
	if upper, bounded := p.Occurs().Max(); bounded && upper == 0 {
		return encodeDecline(constructVacuousParticle, subject)
	}
	var el xsd.ElementDeclaration
	switch term := p.Term().(type) {
	case xsd.ModelGroupRef:
		group, _, err := goshape.ModelGroupTerm(w.schema, term)
		if err != nil {
			return fmt.Errorf("codec: %s: %w", subject, err)
		}
		return w.groupEncodes(group, site.occurs(p), subject, site)
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
			return w.groupEncodes(term, site.occurs(p), subject, site)
		case xsd.Wildcard:
			*site.matcher = true
			// A strict or lax match a declaration or a type governs is assessed
			// against it (§3.3.4.6 cvc-assess-elt clause 1), which the
			// assessment of the written document decides; a skip match is not
			// assessed (§3.10.4.1 key-skipped).
			w.assess = w.assess || term.ProcessContents() != xsd.ProcessSkip
			return nil
		case xsd.ElementDeclaration:
			el = term
		}
	}
	return w.element(el)
}

// attributeEncodes returns elementEncodes' error for the attribute use u: a
// declaration whose {target namespace} is the XSI namespace, but for a
// location hint's (locationHint), whose field Append would write unassessed
// (an xsi:type it never holds to Structures §3.3.4.3 cvc-elt clause 4), or
// xmlnsNamespace, which no prefix declaredPrefix declares may be bound to
// (Namespaces in XML §3), then its simple type.
func (w *encodeWalk) attributeEncodes(u xsd.AttributeUse) error {
	decl, ok := w.schema.ResolvedAttributeDeclaration(u)
	if !ok {
		return fmt.Errorf("codec: the {attribute declaration} of attribute use %s does not resolve", u.DeclarationName().Display())
	}
	subject := attributeSubject(decl)
	if space := decl.Name().Space; space == xsd.XMLSchemaInstanceNS && !locationHint(decl.Name()) || space == xmlnsNamespace {
		return encodeDecline(constructReservedAttributeUse, subject)
	}
	st, ok := w.schema.ResolvedSimpleType(decl.TypeDefinition())
	if !ok {
		return fmt.Errorf("codec: the {type definition} of %s does not resolve", subject)
	}
	return w.simpleEncodes(st, subject)
}

// The expanded names encodeWalk.simpleEncodes reads a simple type's primitive
// and base chain by, with booleanName: encodedPrimitives, the one encoding of
// the primitives an atomic type that encodes has (encodeWalk.atomicEncodes),
// in the order of Datatypes §3.3 and then xs:precisionDecimal, which §3.3 does
// not list, the order constructEncodePrimitive's name lists them in;
// idNames, the types a type derived from sets encodeWalk.assess; and
// entityName, the one a type derived from declines.
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
	}
	entityName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "ENTITY"}
)

// simpleEncodes returns nil where the simple type st, the type of the
// declaration owner names, encodes as the package doc's "# What encodes"
// states, and otherwise the decline of the first construct it finds: for a
// list, the first construct of its {item type definition} (atomicEncodes),
// the detail naming the item after st (itemSubject), and an assertions facet
// among st's own effective facets, its own or inherited; for a special st,
// xs:anySimpleType or xs:anyAtomicType itself (Datatypes §2.4.2 dt-special),
// nil; for any other atomic or union st, atomicEncodes's. An error reading
// st is returned as is, and a list with no {item type definition} is a plain
// error.
func (w *encodeWalk) simpleEncodes(st *xsd.SimpleType, owner string) error {
	return w.typeEncodes(st, simpleSubject(st, owner))
}

// typeEncodes is simpleEncodes for st, subject naming it already: a
// declaration's type as simpleSubject names it, a complex type's {content
// type}'s {simple type definition} as contentSubject names it after that type
// (complexEncodes), or a union's basic member as memberSubject names it after
// its union (encodeWalk.unionEncodes).
func (w *encodeWalk) typeEncodes(st *xsd.SimpleType, subject string) error {
	if st.IsSpecial() {
		return nil
	}
	variety, err := st.Variety(w.schema)
	if err != nil {
		return err
	}
	if _, isList := variety.(xsd.List); !isList {
		return w.atomicEncodes(st, subject)
	}
	item, err := st.Item(w.schema)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("codec: the list type %s has no {item type definition}", subject)
	}
	if err := w.atomicEncodes(item, itemSubject(subject, item)); err != nil {
		return err
	}
	return assertionsDecline(w.schema, st, subject)
}

// atomicEncodes returns nil where st, an atomic or union simple type subject
// names, a list's {item type definition} among them, which cos-st-restricts
// clause 2.1 holds to one of those two, encodes, and otherwise the decline
// of the first construct it finds: for a union, encodeWalk.unionEncodes's,
// and for an atomic type a primitive not in encodedPrimitives, a type on
// whose {base type definition} chain xs:ENTITY lies, then an assertions facet
// among its effective facets, its own or inherited. A type on whose chain
// xs:ID or xs:IDREF lies sets w.assess, the cvc-id condition of [Append]'s
// doc: a list over one, xs:IDREFS among them, reaches here as its item, and
// a union as each of its basic members. An error reading st is returned as
// is.
func (w *encodeWalk) atomicEncodes(st *xsd.SimpleType, subject string) error {
	variety, err := st.Variety(w.schema)
	if err != nil {
		return err
	}
	if _, isUnion := variety.(xsd.Union); isUnion {
		return w.unionEncodes(st, subject)
	}
	primitive, err := st.Primitive(w.schema)
	if err != nil {
		return err
	}
	if primitive == nil || !slices.Contains(encodedPrimitives, primitive.Name()) {
		return encodeDecline(constructEncodePrimitive, subject)
	}
	for step := st; step != nil; {
		if step.Name() == entityName {
			return encodeDecline(constructEntityType, subject)
		}
		if slices.Contains(idNames, step.Name()) {
			w.assess = true
		}
		step, err = step.Base(w.schema)
		if err != nil {
			return err
		}
	}
	return assertionsDecline(w.schema, st, subject)
}

// unionEncodes returns atomicEncodes's answer for st, a union simple type
// subject names: the first construct of its basic members, in basic-member
// order (goshape.BasicMembers), each examined by typeEncodes, the detail
// naming the member after subject (memberSubject), as Decode's unionDecline
// examines them; then unionAssertionsDecline's for st. An error reading st is
// returned as is.
func (w *encodeWalk) unionEncodes(st *xsd.SimpleType, subject string) error {
	basic, err := goshape.BasicMembers(w.schema, st)
	if err != nil {
		return err
	}
	for _, m := range basic {
		if err := w.typeEncodes(m, memberSubject(subject, m)); err != nil {
			return err
		}
	}
	return unionAssertionsDecline(w.schema, st, subject)
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
// the prefixed bindings the open start tags declare, outermost first, an inner
// one of a prefix shadowing an outer one (writtenBindings); open, the values
// whose element's write is open, outermost first (enter); and the first item 6
// and item 7 errors of [Append]'s doc it has found, in document order. Every
// value is written whatever was found before it, so that the walk finds an
// item 6 error after an item 7 one.
type encoder struct {
	schema   *xsd.Schema
	backend  value.Backend
	decimals decimalArmSets
	branches branchSets
	out      []byte
	ns       string
	bound    []nsBinding
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
// value's text (scalarText), which may first declare a prefix on that start
// tag (qname), held to decl's {value constraint} (elementConstraint), an
// empty text written as an empty-element tag (textTag). The error is a
// backend fault.
func (e *encoder) simpleElement(decl xsd.ElementDeclaration, s scalar, v reflect.Value, path string) error {
	name := decl.Name()
	outer, mark := e.startTag(name)
	subject := simpleSubject(s.st, declSubject(decl))
	w, err := e.scalarText(s, v, path, subject)
	if err != nil {
		return err
	}
	e.elementConstraint(decl, s.st, w, path, subject)
	e.textTag(name.Local, w.text, outer, mark)
	return nil
}

// elementConstraint holds w, what Append writes as the character content of
// decl's element for the value at path, of st, the element's simple type or
// its type's {content type}.{simple type definition}, subject naming st, to
// decl's {value constraint} (heldTo), where w recorded no item 6 or 7 error
// of [Append]'s doc: a non-empty text under a fixed constraint, as
// Structures §3.3.4.3 cvc-elt clause 5.2.2.2.2 compares it, and the empty
// text under either variety, which Append writes as an empty-element tag, to
// which clause 5.1 and §3.3.5.4 sic-eltType supply the constraint's {lexical
// form}. A non-empty text under a default is written whatever it holds:
// clause 5.2 compares nothing to a default.
func (e *encoder) elementConstraint(decl xsd.ElementDeclaration, st *xsd.SimpleType, w written, path, subject string) {
	vc, has := decl.ValueConstraint()
	if !has || !w.ok {
		return
	}
	if w.text == "" {
		e.heldTo(vc, declSubject(decl), "§3.3.4.3 cvc-elt clause 5.1, §3.3.5.4 sic-eltType", st, w.text, path, subject)
		return
	}
	if vc.Kind() == xsd.ValueFixed {
		e.heldTo(vc, declSubject(decl), "§3.3.4.3 cvc-elt clause 5.2.2.2.2", st, w.text, path, subject)
	}
}

// attributeFixed holds w, what Append writes as a's attribute for the value
// at path, to a's fixed {value constraint} (heldTo), where w recorded no
// item 6 or 7 error of [Append]'s doc: the use's own, which Structures
// §3.5.4 cvc-au reads, then its declaration's, which §3.2.4.1 cvc-attribute
// clause 4 reads, each separately and never the ·effective value constraint·
// alone (xsd.Schema.EffectiveValueConstraint), since a use's default would
// hide its declaration's fixed one. A default is read by neither rule, and
// Append writes every attribute a value holds, so that no attribute it
// writes is ·defaulted· (§3.4.4.2 key-dflt-att).
func (e *encoder) attributeFixed(a attribute, w written, path string) {
	if !w.ok {
		return
	}
	subject := simpleSubject(a.st, attributeSubject(a.decl))
	if vc, has := a.use.ValueConstraint(); has && vc.Kind() == xsd.ValueFixed {
		e.heldTo(vc, "the attribute use of "+attributeSubject(a.decl), "§3.5.4 cvc-au", a.st, w.text, path, subject)
	}
	if vc, has := a.decl.ValueConstraint(); has && vc.Kind() == xsd.ValueFixed {
		e.heldTo(vc, attributeSubject(a.decl), "§3.2.4.1 cvc-attribute clause 4", a.st, w.text, path, subject)
	}
}

// heldTo records the item 6 or item 7 error of [Append]'s doc that vc, the
// {value constraint} of the component on names, which the Structures rule
// rule reads, raises for text, written for the value at path, of st, subject
// naming st. It compares them by value.ConstraintMatches under st, text under
// the bindings in scope where it is written (writtenBindings), in the value
// space, equal or identical (Datatypes §2.2.1, §2.2.2), as validate compares
// them. The same is written; not the same is item 6 under a fixed vc, whose
// value every valid document holds there, and item 7 by
// constructValueConstraint under a default, which elementConstraint passes
// only for the empty text, since another lexical of the value may be valid
// there, as a single space is for an empty xs:token under default "abc"; and
// a comparison ConstraintMatches leaves undecided, such as a NOTATION
// {lexical form} naming no declared notation, is item 7 by the same
// construct. A comparison found the same has validated vc's {lexical form}
// under st, so that cvc-elt clause 5.1.2 holds for the empty text written.
func (e *encoder) heldTo(vc xsd.ValueConstraint, on, rule string, st *xsd.SimpleType, text, path, subject string) {
	variety := "default"
	if vc.Kind() == xsd.ValueFixed {
		variety = "fixed"
	}
	shown := fmt.Sprintf("the value written %q", text)
	same, decided := value.ConstraintMatches(e.backend, e.schema, st, text, writtenBindings{e}, vc, noAssertions{})
	switch {
	case !decided:
		e.declineValue(constructValueConstraint, subject, path, shown, fmt.Sprintf("value.ConstraintMatches does not decide whether its value is equal or identical to %q, the {value} of the %s {value constraint} of %s (Datatypes §2.2.1, §2.2.2; Structures %s)", vc.LexicalForm(), variety, on, rule))
	case same:
	case vc.Kind() == xsd.ValueFixed:
		e.invalidate(fmt.Errorf("codec: Append: %s, of %s, is written %q, whose value is neither equal nor identical to %q, the {value} of the fixed {value constraint} of %s, which every valid document holds there (Structures %s)", path, subject, text, vc.LexicalForm(), on, rule))
	default:
		e.declineValue(constructValueConstraint, subject, path, shown, fmt.Sprintf("its value is neither equal nor identical to %q, the {value} of the default {value constraint} of %s, which the empty element it is written as takes on (Structures %s)", vc.LexicalForm(), on, rule))
	}
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
// c, constructNoCanonical, constructQNameSpelling or
// constructValueConstraint, for the value shown at path, of the simple type
// subject names, cause saying why, unless an earlier value's is recorded.
func (e *encoder) declineValue(c construct, subject, path, shown, cause string) {
	if e.declined == nil {
		e.declined = encodeDecline(c, fmt.Sprintf("%s, at %s holding %s: %s", subject, path, shown, cause))
	}
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
// s.st, at path, subject naming s.st: a list by list, and any other value,
// of an atomic or union simple type or of a special type, xs:anySimpleType's
// {variety} being absent (Datatypes §2.4.2 dt-special, §4.1.1), by value, under s's
// element.
func (e *encoder) scalarText(s scalar, v reflect.Value, path, subject string) (written, error) {
	if s.item != nil {
		return e.list(s, v, path, subject)
	}
	return e.value(s.elem, s.st, v, path, subject)
}

// value returns what Append writes for v, a value of st, an atomic or union
// simple type or a special type, xs:anySimpleType's {variety} being absent
// (Datatypes §2.4.2 dt-special, §4.1.1), whose scalar element is elem, at path,
// subject naming st: a union's by union, and every other's by atom, as held
// reads it, under its atomic Go type (encoded).
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

// atom returns the text Append writes for v, a value of st, an atomic simple
// type or a special type, xs:anySimpleType's {variety} being absent (Datatypes
// §2.4.2 dt-special, §4.1.1), held in a, its atomic Go type, at path, subject naming
// st: the value of an atomic formed holds for in its canonical representation
// (spelled), a string, a special type's among them, as itself (str), an xs:QName or
// xs:NOTATION value under a prefix in scope (qname), a float32 or float64 in its
// canonical representation (double), and a precisionDecimal, v the decimalValue held
// translated it into, in its canonical representation (precisionDecimal). ok reports
// that it recorded no item 6 or 7 error of [Append]'s doc for v.
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
// or xs:anyURI or that is a special type, after checking, in this order, that
// it is UTF-8 of XML 1.0 Char characters only, which both value spaces and
// both special types' lexical spaces are sequences of (Datatypes §3.3.1.1,
// §3.3.17.1, §3.2.1.2, §3.2.2.2), and then, for st other than a special type,
// which it returns after that check alone, that value.ValidateLexical accepts
// it under st, and that it is the canonical representation of the value that
// returns, its own whiteSpace-normalized form, as the package doc's "# What
// encodes" states a string value is written. Canonical renders the value
// ValidateLexical returns, never one Parse maps s to: Parse takes a
// normalized lexical, and s may be none. ok is atom's.
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
		// encodeWalk.atomicEncodes declines a type with no {primitive type definition}.
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
// other than the XML namespace and e.ns that no open start tag declares a
// prefix for, it declares one on that start tag (declaredPrefix). Where none
// spells it — a space "" under a non-absent e.ns, then a space holding a byte
// that starts no XML 1.0 Char, then xmlnsNamespace — it records that value's
// item 7 or item 6 error of [Append]'s doc, in that order, and ok is false.
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
// declares for it (inScopePrefix), or else the first of p1, p2 and so on that
// no open start tag binds (freshPrefix), whose declaration it appends to the
// start tag that is open and pushes onto e.bound, so that e.bound holds only
// bindings a declaration makes.
func (e *encoder) declaredPrefix(space string) string {
	if prefix, ok := e.inScopePrefix(space); ok {
		return prefix
	}
	b := nsBinding{prefix: e.freshPrefix(), space: space}
	e.bound = append(e.bound, b)
	e.out = b.declare(e.out)
	return b.prefix
}

// inScopePrefix returns the prefix of the first binding of e.bound,
// outermost first, to space that no binding after it of the same prefix
// shadows, and true, or false where none is.
func (e *encoder) inScopePrefix(space string) (string, bool) {
	for _, b := range e.bound {
		if b.space != space {
			continue
		}
		if bound, _ := e.boundSpace(b.prefix); bound == space {
			return b.prefix, true
		}
	}
	return "", false
}

// freshPrefix returns the first of p1, p2 and so on that no binding of
// e.bound declares.
func (e *encoder) freshPrefix() string {
	for i := 1; ; i++ {
		if prefix := "p" + strconv.Itoa(i); !slices.ContainsFunc(e.bound, func(b nsBinding) bool { return b.prefix == prefix }) {
			return prefix
		}
	}
}

// boundSpace returns the namespace name the innermost binding of e.bound of
// prefix binds it to, and true, or false where none binds it.
func (e *encoder) boundSpace(prefix string) (string, bool) {
	for i := len(e.bound) - 1; i >= 0; i-- {
		if e.bound[i].prefix == prefix {
			return e.bound[i].space, true
		}
	}
	return "", false
}

// nsBinding is a prefixed namespace binding a start tag Append writes
// declares, or one a held AnyType's Namespaces holds (heldNamespaces): prefix
// bound to space.
type nsBinding struct{ prefix, space string }

// declare appends to dst the declaration of b, xmlns:prefix="space", led by a
// space.
func (b nsBinding) declare(dst []byte) []byte {
	dst = append(dst, " xmlns:"+b.prefix+`="`...)
	dst = appendEscaped(dst, b.space, true)
	return append(dst, '"')
}

// attributionCheck is how Append holds the children it writes into one
// element's content, whose content model holds a wildcard particle or a
// nested flattened particle, or is flattened, to that content model (§3.9.4.2
// cvc-particle, §3.9.4.3 cvc-accept, §3.8.4.3 cvc-model-group): a, a
// matchAttributor over a fresh Matcher of the content model's {content type}
// and its wildcard particles' places, or a flattened one's every basic
// particle's (flattenedCheck), which decides for each name cvc-wildcard
// (§3.10.4.1), clauses 1 to 3, {disallowed names} included, and cvc-accept's
// element particle before wildcard particle precedence; elements, the content
// model's element particles' places (childIndex.elements), or a flattened
// one's every basic particle's; and subject, the content's complex type and
// element as typeSubject names them.
type attributionCheck struct {
	a        *matchAttributor
	elements []place
	subject  string
}

// attribution returns the attributionCheck of the content of el, of the
// complex type typ, whose children x attributes, or nil where x places no
// wildcard particle and flat is false, a content model whose children Append
// writes unchecked. flat is whether the content holds a nested flattened
// particle's slice (flatHolder), whose items' order is the caller's, and
// whose bind gave x its matcherSlot (sequence.flatten). Item 4 of [Append]'s
// doc declines a content model holding a wildcard particle or a nested
// flattened particle that xsd.Schema.ContentMatcher does not track
// (contentEncodes), so that a matcherSlot.attributor error here is an
// internal bug, which panics.
func (e *encoder) attribution(x childIndex, el xsd.ElementDeclaration, typ xsd.ComplexType, flat bool) *attributionCheck {
	if x.matcher == nil || len(x.matcher.places) == 0 && !flat {
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
// (§3.9.4.3 cvc-accept clause 3.3, §3.8.4.3 cvc-model-group). A nil c checks
// nothing.
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
// at any depth, or the attributes it writes raw (encoder.wildcardAttributes):
// a match of a wildcard with {process contents} skip (skipMatch), nothing in
// which is assessed (§3.10.4.1 key-skipped), a match of one with {process
// contents} strict or lax (assessedMatch), the content of an element of type
// xs:anyType itself (anyTypeContent), each of the last two assessed against
// what governs an item, and laxly where nothing does (§3.3.4.6 key-lva,
// §3.4.7), the attributes of an element of type xs:anyType itself
// (anyTypeElement), or the {attribute wildcard} field of a complex type's
// struct (attributeWildcardHold). in names the hold for a decline's detail;
// assessed reports whether an assessment of the document reads what it
// holds, false for skipMatch alone of the holds rawContent writes;
// writesHints reports whether Append writes a held xsi:schemaLocation or
// xsi:noNamespaceSchemaLocation, true for the holds rawContent writes and
// false for attributeWildcardHold; writesType reports whether Append writes a
// held xsi:type, true for skipMatch, assessedMatch and anyTypeContent and
// false for anyTypeElement and attributeWildcardHold; writesNil reports
// whether Append writes a held xsi:nil, true for skipMatch alone.
type rawHold interface {
	in() string
	assessed() bool
	writesHints() bool
	writesType() bool
	writesNil() bool
}

// skipMatch is the match of the wildcard particle it names, of {process
// contents} skip.
type skipMatch string

// in names the wildcard particle.
func (m skipMatch) in() string { return "in the match of " + string(m) }

// assessed reports false: a skipped item has no governing declaration,
// whatever its name (§3.10.4.1 key-skipped).
func (skipMatch) assessed() bool { return false }

// writesHints reports true: a skipped hint is not assessed.
func (skipMatch) writesHints() bool { return true }

// writesType reports true: a skipped xsi:type is not assessed.
func (skipMatch) writesType() bool { return true }

// writesNil reports true: a skipped element is not assessed, as item 7 of [Append]'s doc states.
func (skipMatch) writesNil() bool { return true }

// anyTypeContent is the content of an element of decl, whose {type
// definition} is xs:anyType itself, the element's own attributes among it.
type anyTypeContent struct{ decl xsd.ElementDeclaration }

// in names c.decl.
func (c anyTypeContent) in() string { return "in the content of " + declSubject(c.decl) }

// assessed reports true: xs:anyType's wildcards are lax (§3.4.7).
func (anyTypeContent) assessed() bool { return true }

// writesHints reports true: a hint's built-in declaration admits every value
// rawAttribute writes (locationHint).
func (anyTypeContent) writesHints() bool { return true }

// writesType reports true: the assessment holds an xsi:type to cvc-elt clause
// 4.
func (anyTypeContent) writesType() bool { return true }

// writesNil reports false: validate reads a held xsi:nil there, as item 7 of
// [Append]'s doc states.
func (anyTypeContent) writesNil() bool { return false }

// anyTypeElement is the attributes of an element of decl, whose {type
// definition} is xs:anyType itself, held as anyTypeContent holds its content
// but for an xsi:type among them.
type anyTypeElement struct{ decl xsd.ElementDeclaration }

// in names e.decl, as anyTypeContent's does.
func (e anyTypeElement) in() string { return anyTypeContent(e).in() }

// assessed reports true, as anyTypeContent's does.
func (anyTypeElement) assessed() bool { return true }

// writesHints reports true, as anyTypeContent's does.
func (anyTypeElement) writesHints() bool { return true }

// writesType reports false, as item 7 of [Append]'s doc states.
func (anyTypeElement) writesType() bool { return false }

// writesNil reports false, as anyTypeContent's does.
func (anyTypeElement) writesNil() bool { return false }

// assessedMatch is the match of the wildcard particle it names, of {process
// contents} strict or lax.
type assessedMatch string

// in names the wildcard particle.
func (m assessedMatch) in() string { return "in the match of " + string(m) }

// assessed reports true: the match is assessed against what governs it, and
// below it each item is assessed against what governs it or laxly against
// xs:anyType (§3.3.4.6 key-governing-ed clause 3, key-lva).
func (assessedMatch) assessed() bool { return true }

// writesHints reports true, as anyTypeContent's does.
func (assessedMatch) writesHints() bool { return true }

// writesType reports true, as anyTypeContent's does.
func (assessedMatch) writesType() bool { return true }

// writesNil reports false, as anyTypeContent's does.
func (assessedMatch) writesNil() bool { return false }

// attributeWildcardHold is the {attribute wildcard} field of the struct of
// typ, the complex type of the element of el, which holds that element's
// attributes no attribute use takes. Its wildcard is typ's {attribute
// wildcard}, the effective one (§3.4.2.5 dcl.ctd.anyatt), read from typ at
// each ask.
type attributeWildcardHold struct {
	el  xsd.ElementDeclaration
	typ xsd.ComplexType
}

// in names the field and h.typ.
func (h attributeWildcardHold) in() string {
	return "in the {attribute wildcard} field of " + typeSubject(h.el, h.typ)
}

// assessed reports whether the wildcard's {process contents} is other than
// skip (§3.10.4.1 key-skipped).
func (h attributeWildcardHold) assessed() bool {
	wild, _ := h.typ.AttributeWildcard()
	return wild.ProcessContents() != xsd.ProcessSkip
}

// governing returns the subject of the top-level attribute declaration in
// schema of the expanded name name, and true, where h is assessed, an
// attribute a strict or lax wildcard admits being governed by the one its
// expanded name resolves to (§3.2.4.2 key-governing-ad clause 3), or false
// where none governs it.
func (h attributeWildcardHold) governing(schema *xsd.Schema, name xsd.QName) (string, bool) {
	if !h.assessed() {
		return "", false
	}
	d, ok := schema.Attribute(name)
	if !ok {
		return "", false
	}
	return attributeSubject(d), true
}

// writesHints reports false: Decode never returns a hint in an {attribute
// wildcard} field ([Decode]'s doc), so that one written there would not decode
// back.
func (attributeWildcardHold) writesHints() bool { return false }

// writesType reports false: Decode never returns xsi:type in an {attribute
// wildcard} field ([Decode]'s doc), so that one written there would not decode
// back.
func (attributeWildcardHold) writesType() bool { return false }

// writesNil reports false: Decode never returns xsi:nil in an {attribute
// wildcard} field ([Decode]'s doc), so that one written there would not
// decode back.
func (attributeWildcardHold) writesNil() bool { return false }

// write writes at.el, the document element, of type xs:anyType itself,
// holding v (writeElement).
func (c rawContent) write(e *encoder, v reflect.Value, at writeAt) error {
	return c.writeElement(e, at.el, v, at.path, attributionSlot{})
}

// writeElement writes the element of decl, of type xs:anyType itself, whose
// content is v, a struct of AnyType's shape or a named type over one, at
// path: its start tag (rawStartTag), then v as rawContent writes it under
// anyTypeContent, as the package doc's "# What encodes" states, its content
// held to decl's fixed {value constraint} after its Text (anyTypeFixed); or
// nothing, where v holds itself (encoder.enter), taken by slot first either
// way, its own attributes held in anyTypeElement. It reads no simple value, so
// it returns no backend fault.
func (rawContent) writeElement(e *encoder, decl xsd.ElementDeclaration, v reflect.Value, path string, slot attributionSlot) error {
	slot.take(e, decl, path)
	if !e.enter(v, path) {
		return nil
	}
	defer e.leave()
	tag, held, outer, mark := e.rawStartTag(decl.Name(), v, path)
	start := e.rawAttributes(v, held, path, anyTypeElement{decl: decl}, false)
	e.rawText(v.Field(xText).String(), held, path+".Text", false)
	e.anyTypeFixed(decl, v, path)
	e.rawChildren(v, held, path, anyTypeContent{decl: decl}, false)
	e.endTag(tag, start, outer, mark)
	return nil
}

// anyTypeFixed records item 6 of [Append]'s doc where decl, of type
// xs:anyType itself, has a fixed {value constraint} and x, the AnyType
// content of its element at path, is written as content no valid element
// holds: x holds Children, an element child cvc-elt clause 5.2.2.1 admits
// none of, or a Text other than "" and the constraint's {lexical form}, which
// clause 5.2.2.2.1 matches the ·initial value· to as a string, xs:anyType's
// {content type} being mixed (Structures §3.4.7, §3.3.4.3). With no children
// and an empty Text the element is written empty, valid by clause 5.1. A
// default constrains nothing Append writes there, and Decode supplies no
// value to xs:anyType, so that x decodes as itself; its attributes are read
// by no clause of cvc-elt 5.
func (e *encoder) anyTypeFixed(decl xsd.ElementDeclaration, x reflect.Value, path string) {
	vc, has := decl.ValueConstraint()
	if !has || vc.Kind() != xsd.ValueFixed {
		return
	}
	if n := x.Field(xChildren).Len(); n > 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s.Children holds %d, though the fixed {value constraint} %q of %s admits no element child (Structures §3.3.4.3 cvc-elt clause 5.2.2.1)", path, n, vc.LexicalForm(), declSubject(decl)))
		return
	}
	if text := x.Field(xText).String(); text != "" && text != vc.LexicalForm() {
		e.invalidate(fmt.Errorf("codec: Append: %s.Text holds %q, which does not match %q, the {lexical form} of the fixed {value constraint} of %s (Structures §3.3.4.3 cvc-elt clause 5.2.2.2.1)", path, text, vc.LexicalForm(), declSubject(decl)))
	}
}

// rawElement writes the element name, held raw at path in hold, rawName
// having admitted name, whose content is x, a struct of AnyType's shape, as
// the package doc's "# What encodes" states: its start tag (rawStartTag),
// then its content (rawContent) in the same hold, typed reporting whether it
// may be typed (encoder.capturedIn); or nothing, where x holds itself
// (encoder.enter).
func (e *encoder) rawElement(name xsd.QName, x reflect.Value, path string, hold rawHold, typed bool) {
	if !e.enter(x, path+".Content") {
		return
	}
	defer e.leave()
	tag, held, outer, mark := e.rawStartTag(name, x, path+".Content")
	start := e.rawContent(x, held, path+".Content", hold, typed)
	e.endTag(tag, start, outer, mark)
}

// rawStartTag appends the start tag of the element name, whose content is x,
// a struct of AnyType's shape at path, but for its attributes and its '>', its
// name spelled (namePrefix) and its bindings declared under x's Namespaces
// (heldNamespaces), as the package doc's "# What encodes" states, the
// default's declaration first, then the prefixed ones, the fresh prefix's
// last. It makes that default e.ns and pushes those bindings onto e.bound, for
// its attributes and content, and returns the tag's name, Namespaces'
// prefixed entries, and the e.ns and length of e.bound it replaced, which
// endTag restores.
func (e *encoder) rawStartTag(name xsd.QName, x reflect.Value, path string) (tag string, held []nsBinding, outer string, mark int) {
	def, held := e.heldNamespaces(name, x.Field(xNamespaces), path)
	outer, mark = e.ns, len(e.bound)
	for _, b := range held {
		if space, ok := e.boundSpace(b.prefix); !ok || space != b.space {
			e.bound = append(e.bound, b)
		}
	}
	tag = name.Local
	if prefix := e.namePrefix(name.Space, def, held); prefix != "" {
		tag = prefix + ":" + name.Local
	}
	e.out = append(e.out, '<')
	e.out = append(e.out, tag...)
	if def != outer {
		e.out = append(e.out, ` xmlns="`...)
		e.out = appendEscaped(e.out, def, true)
		e.out = append(e.out, '"')
	}
	for _, b := range e.bound[mark:] {
		e.out = b.declare(e.out)
	}
	e.ns = def
	return tag, held, outer, mark
}

// namePrefix returns the prefix rawStartTag spells an element of namespace
// space with, "" for none, where def is the default namespace and held the
// prefixed bindings its Namespaces gives it, pushing a fresh one onto e.bound,
// undeclared yet, where it needs one.
func (e *encoder) namePrefix(space, def string, held []nsBinding) string {
	switch space {
	case xmltree.XMLNamespaceURI:
		return "xml"
	case def:
		return ""
	}
	if i := slices.IndexFunc(held, func(b nsBinding) bool { return b.space == space }); i >= 0 {
		return held[i].prefix
	}
	if prefix, ok := e.inScopePrefix(space); ok {
		return prefix
	}
	b := nsBinding{prefix: e.freshPrefix(), space: space}
	e.bound = append(e.bound, b)
	return b.prefix
}

// heldNamespaces returns the default namespace, "" for none, and the prefixed
// bindings, in order, that ns, the Namespaces of the AnyType at path holding
// the content of the element name, gives that element, or records item 6 of
// [Append]'s doc for the first entry of ns no Decode returns, and then for
// name, and returns none: an entry whose Prefix is not after the one before it,
// so that ns is not sorted by Prefix with one entry per prefix; a Prefix xml
// or xmlns, which Namespaces never holds; a Prefix other than "" holding a
// byte that starts no XML 1.0 Char or that is no NCName; a Space "", which no
// prefix is bound to (Namespaces in XML §5 nsc-NoPrefixUndecl) and which no
// "" entry holds, an absent default being no entry; a Space holding a byte
// that starts no XML 1.0 Char; and a Space that is the XML namespace or
// xmlnsNamespace, which no other prefix is bound to and no default is
// (Namespaces in XML §3); then an element of absent namespace under a ""
// entry, which no prefix spells (nsc-NoPrefixUndecl) (codegen's package doc,
// "# xs:anyType").
func (e *encoder) heldNamespaces(name xsd.QName, ns reflect.Value, path string) (string, []nsBinding) {
	def := ""
	var held []nsBinding
	for i := range ns.Len() {
		prefix, space := ns.Index(i).Field(0).String(), ns.Index(i).Field(1).String()
		at := path + ".Namespaces[" + strconv.Itoa(i) + "]"
		if err := heldBinding(prefix, space, ns, i); err != nil {
			e.invalidate(fmt.Errorf("codec: Append: %s holds the binding of %s to %s, %w", at, strconv.Quote(prefix), strconv.Quote(space), err))
			return "", nil
		}
		if prefix == "" {
			def = space
			continue
		}
		held = append(held, nsBinding{prefix: prefix, space: space})
	}
	if name.Space == "" && def != "" {
		e.invalidate(fmt.Errorf("codec: Append: %s.Namespaces makes %s the default namespace of %s, an element of no namespace, which no prefix can be bound to, so that no spelling names it (Namespaces in XML §5 nsc-NoPrefixUndecl)", path, strconv.Quote(def), name.Display()))
		return "", nil
	}
	return def, held
}

// heldBinding returns why the binding of prefix to space, item i of ns, a
// held AnyType's Namespaces, is one no Decode returns, heldNamespaces states
// which, or nil.
func heldBinding(prefix, space string, ns reflect.Value, i int) error {
	switch {
	case i > 0 && prefix <= ns.Index(i-1).Field(0).String():
		return fmt.Errorf("whose Prefix is not after %s, the one of the item before it, so that Namespaces is not sorted by Prefix with one entry per prefix (codegen's package doc, \"# xs:anyType\")", strconv.Quote(ns.Index(i-1).Field(0).String()))
	case prefix == "xml" || prefix == "xmlns":
		return errors.New("which Namespaces never holds, xml being bound everywhere and xmlns never declared (Namespaces in XML §3)")
	case prefix != "" && (xmlchar.IndexNonChar(prefix) >= 0 || !xmlname.IsNCName(prefix)):
		return errors.New("whose Prefix is no NCName, so that no declaration binds it (Namespaces in XML [4] NCName)")
	case space == "" && prefix != "":
		return errors.New("which undeclares the prefix, which no namespace declaration of XML 1.0 may do (Namespaces in XML §5 nsc-NoPrefixUndecl)")
	case space == "":
		return errors.New(`whose Space "" is no default namespace, an absent one being no entry (codegen's package doc, "# xs:anyType")`)
	case xmlchar.IndexNonChar(space) >= 0:
		return fmt.Errorf("whose Space's byte offset %d starts no XML 1.0 Char", xmlchar.IndexNonChar(space))
	case space == xmltree.XMLNamespaceURI || space == xmlnsNamespace:
		return errors.New("a namespace name no other prefix may be bound to and no default namespace may be (Namespaces in XML §3)")
	}
	return nil
}

// rawContent appends to the start tag that is open the attributes of x, a
// struct of AnyType's shape at path, held raw in hold, then the tag's '>',
// whose offset it returns, which endTag reads, then x's Text and each of its
// Children's Element, written by rawElement, followed by its Tail, each as the
// package doc's "# What encodes" states, an attribute by writeRawAttribute. It
// records the item 6 and item 7 errors of [Append]'s doc these raise, in
// document order (rawAttribute, rawName, capturedIn, rawText), and leaves
// each such attribute or text, and each element rawName refuses, out of what
// it writes. It is rawAttributes, an attribute's prefix read from held, the
// prefixed bindings of x's Namespaces, then x's Text (rawText), then
// rawChildren, typed reporting whether the content may be typed
// (encoder.capturedIn).
func (e *encoder) rawContent(x reflect.Value, held []nsBinding, path string, hold rawHold, typed bool) int {
	start := e.rawAttributes(x, held, path, hold, typed)
	e.rawText(x.Field(xText).String(), held, path+".Text", typed)
	e.rawChildren(x, held, path, hold, typed)
	return start
}

// rawAttributes is rawContent's write of x's attributes and the tag's '>',
// whose offset it returns. It writes each attribute rawAttribute admits, held
// and typed being rawContent's; once the start tag holds every declaration it
// makes, it holds the Value of each written attribute in that region to
// capturedIn: every one where typed holds, an xsi:type in any hold, and,
// in a hold an assessment reads, one a top-level attribute declaration of
// schema governs (§3.2.4.2 key-governing-ad clause 3).
func (e *encoder) rawAttributes(x reflect.Value, held []nsBinding, path string, hold rawHold, typed bool) int {
	attrs := x.Field(xAttributes)
	written := make([]bool, attrs.Len())
	for i := range attrs.Len() {
		a := attrs.Index(i)
		name := xsd.QName{Space: a.Field(0).String(), Local: a.Field(1).String()}
		value := a.Field(2).String()
		if !e.rawAttribute(name, value, attrs, i, path+".Attributes["+strconv.Itoa(i)+"]", hold) {
			continue
		}
		e.writeRawAttribute(name, value, held)
		written[i] = true
	}
	for i := range attrs.Len() {
		a := attrs.Index(i)
		name := xsd.QName{Space: a.Field(0).String(), Local: a.Field(1).String()}
		_, governed := e.schema.Attribute(name)
		if written[i] && (typed || name == xsiTypeName || hold.assessed() && governed) {
			e.capturedIn(a.Field(2).String(), held, path+".Attributes["+strconv.Itoa(i)+"].Value")
		}
	}
	start := len(e.out)
	e.out = append(e.out, '>')
	return start
}

// xsiTypeName is the expanded name of xsi:type.
var xsiTypeName = xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}

// xsiNilName is the expanded name of xsi:nil.
var xsiNilName = xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}

// capturedIn records item 7 of [Append]'s doc, constructCapturedPrefix, where
// no decline is recorded yet and text, held at path in an element written
// from an AnyType whose Namespaces' prefixed entries are held, holds a token,
// split at XML white space (listSeparators), spelled NCName:NCName whose
// prefix, other than xml, no entry of held binds but a binding of e.bound, one
// Append declares, puts in scope where it is written, so that the token would
// resolve where the value says nothing does (Datatypes §3.3.18). A token whose
// prefix nothing binds is written: an assessment that types it charges it, item
// 8.
func (e *encoder) capturedIn(text string, held []nsBinding, path string) {
	if e.declined != nil {
		return
	}
	for _, token := range strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune(listSeparators, r) }) {
		prefix, local, ok := strings.Cut(token, ":")
		if !ok || prefix == "xml" || !xmlname.IsNCName(prefix) || !xmlname.IsNCName(local) || slices.ContainsFunc(held, func(b nsBinding) bool { return b.prefix == prefix }) {
			continue
		}
		if space, bound := e.boundSpace(prefix); bound {
			e.declined = encodeDecline(constructCapturedPrefix, fmt.Sprintf("%s holds the token %q, whose prefix %s no entry of its AnyType's Namespaces binds, though Append's declaration binds it to %s where it is written", path, token, prefix, strconv.Quote(space)))
			return
		}
	}
}

// rawChildren is rawContent's write of each of x's Children, each followed by
// its Tail, held, hold and typed being rawContent's: a child may be typed
// where typed holds, or where hold is one an assessment reads and a top-level
// element declaration of schema names it or its Attributes hold xsi:type
// (§3.3.4.6 key-governing-ed clause 3, key-governing-type-elem clause 8).
func (e *encoder) rawChildren(x reflect.Value, held []nsBinding, path string, hold rawHold, typed bool) {
	children := x.Field(xChildren)
	for i := range children.Len() {
		c := children.Index(i)
		cpath := path + ".Children[" + strconv.Itoa(i) + "]"
		if name, ok := e.rawName(c.Field(0), cpath+".Element"); ok {
			content := c.Field(0).Field(2)
			_, declared := e.schema.Element(name)
			e.rawElement(name, content, cpath+".Element", hold, typed || hold.assessed() && (declared || holdsXsiType(content)))
		}
		e.rawText(c.Field(1).String(), held, cpath+".Tail", typed)
	}
}

// writeRawAttribute appends to the start tag that is open the attribute name
// of value value, held raw or an attribute use's, as the package doc's "# What
// encodes" states: of absent namespace unprefixed, in the XML namespace
// xml:Local, and otherwise under the first prefix of held, the prefixed
// bindings of the Namespaces of the AnyType holding it, nil for an attribute
// held in no AnyType, bound to its namespace, or else under declaredPrefix's,
// whose declaration precedes the attribute's ' '. name.Space is never
// xmlnsNamespace: rawAttribute (writableName) refuses it upstream, and
// attributeEncodes declines it.
func (e *encoder) writeRawAttribute(name xsd.QName, value string, held []nsBinding) {
	qualified := name.Local
	i := slices.IndexFunc(held, func(b nsBinding) bool { return b.space == name.Space })
	switch {
	case name.Space == xmltree.XMLNamespaceURI:
		qualified = "xml:" + name.Local
	case name.Space == "":
	case i >= 0:
		qualified = held[i].prefix + ":" + name.Local
	default:
		qualified = e.declaredPrefix(name.Space) + ":" + name.Local
	}
	e.out = append(e.out, ' ')
	e.out = append(e.out, qualified...)
	e.out = append(e.out, `="`...)
	e.out = appendEscaped(e.out, value, true)
	e.out = append(e.out, '"')
}

// governed reports whether hold governs the attribute name it holds at path
// (attributeWildcardHold.governing), and records item 7 of [Append]'s doc,
// constructGovernedContent, where it does and no decline is recorded yet.
// wildcardAttributes writes no governed attribute.
func (e *encoder) governed(name xsd.QName, path string, hold attributeWildcardHold) bool {
	governor, ok := hold.governing(e.schema, name)
	if !ok {
		return false
	}
	if e.declined == nil {
		e.declined = encodeDecline(constructGovernedContent, fmt.Sprintf("attribute %s at %s, %s, governed by %s", name.Display(), path, hold.in(), governor))
	}
	return true
}

// rawAttribute reports whether the attribute name of value value, attrs' item
// i at path, held in hold, can be written raw, recording its item 6 or item 7
// error of [Append]'s doc where it cannot: writableName's, an unqualified
// xmlns, an expanded name an attribute before it holds, one of the four
// instanceAttribute names (constructXsiAttribute), but for a hint
// (locationHint) in a hold that writes them (rawHold.writesHints), xsi:type
// in a hold that writes it (rawHold.writesType) and xsi:nil in a hold that
// writes it (rawHold.writesNil), and a Value of a byte that starts no Char, in that
// order. wildcardAttributes asks whether hold governs it (encoder.governed)
// after it.
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
	if instanceAttribute(name) && (!locationHint(name) || !hold.writesHints()) && (name != xsiTypeName || !hold.writesType()) && (name != xsiNilName || !hold.writesNil()) {
		if e.declined == nil {
			e.declined = encodeDecline(constructXsiAttribute, fmt.Sprintf("attribute %s at %s, %s", name.Display(), path, hold.in()))
		}
		return false
	}
	if j := xmlchar.IndexNonChar(value); j >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s.Value holds %s, whose byte offset %d starts no XML 1.0 Char", path, strconv.Quote(value), j))
		return false
	}
	return true
}

// wildcardAttributes appends to the start tag that is open, after its
// attribute uses' attributes, each attribute attrs, the {attribute wildcard}
// field of bd's struct at path, holds, in field order, written raw
// (writeRawAttribute), the field the element of el holds. It reads the
// wildcard from bd.typ, its effective {attribute wildcard} (§3.4.2.5
// dcl.ctd.anyatt). For each item it records, in this order, item 6 of
// [Append]'s doc where an attribute use of bd.typ names its expanded name,
// which then takes it (§3.4.4.2 cvc-complex-type clause 2.1), whether the
// use's field is nil or set, and, but for xsi:type, xsi:nil,
// xsi:schemaLocation and xsi:noNamespaceSchemaLocation, which clause 2
// excepts by name, where the wildcard does not admit it (clause 2.2.2,
// §3.10.4.1 cvc-wildcard, xsd.Schema.AllowsAttributeWildcardName); then
// rawAttribute's errors; then item 7, constructStrictAttributeWildcard, where
// the wildcard's {process contents} is strict, whatever its name, and
// constructGovernedContent where it is lax and a top-level attribute
// declaration governs it (encoder.governed); and writes the item where none
// applies. It writes no attribute for an empty field, whatever the
// {process contents}.
func (e *encoder) wildcardAttributes(bd *binding, attrs reflect.Value, path string, el xsd.ElementDeclaration) {
	wild, _ := bd.typ.AttributeWildcard()
	hold := attributeWildcardHold{el: el, typ: bd.typ}
	for i := range attrs.Len() {
		a := attrs.Index(i)
		apath := path + "[" + strconv.Itoa(i) + "]"
		name := xsd.QName{Space: a.Field(0).String(), Local: a.Field(1).String()}
		value := a.Field(2).String()
		if _, used := bd.attributes[name]; used {
			e.invalidate(fmt.Errorf("codec: Append: %s holds the attribute %s, whose expanded name an attribute use of %s names, so that the use takes it, not the {attribute wildcard} (Structures §3.4.4.2 cvc-complex-type clause 2.1)", apath, name.Display(), typeSubject(el, bd.typ)))
			continue
		}
		if !instanceAttribute(name) && !e.schema.AllowsAttributeWildcardName(wild, name) {
			e.invalidate(fmt.Errorf("codec: Append: %s holds the attribute %s, which the {attribute wildcard} of %s does not admit (Structures §3.4.4.2 cvc-complex-type clause 2.2.2, §3.10.4.1 cvc-wildcard)", apath, name.Display(), typeSubject(el, bd.typ)))
			continue
		}
		if !e.rawAttribute(name, value, attrs, i, apath, hold) {
			continue
		}
		if wild.ProcessContents() == xsd.ProcessStrict {
			if e.declined == nil {
				e.declined = encodeDecline(constructStrictAttributeWildcard, fmt.Sprintf("attribute %s at %s, %s", name.Display(), apath, hold.in()))
			}
			continue
		}
		if e.governed(name, apath, hold) {
			continue
		}
		e.writeRawAttribute(name, value, nil)
	}
}

// rawText appends text, character data held raw at path, escaped as element
// content, or records item 6 of [Append]'s doc where it holds a byte that
// starts no XML 1.0 Char, and writes nothing. Where typed holds, the content
// holding it may be typed, and text is held to capturedIn under held, the
// prefixed bindings of its element's Namespaces.
func (e *encoder) rawText(text string, held []nsBinding, path string, typed bool) {
	if i := xmlchar.IndexNonChar(text); i >= 0 {
		e.invalidate(fmt.Errorf("codec: Append: %s holds %s, whose byte offset %d starts no XML 1.0 Char", path, strconv.Quote(text), i))
		return
	}
	if typed {
		e.capturedIn(text, held, path)
	}
	e.out = appendEscaped(e.out, text, false)
}

// xmlnsNamespace is the namespace name the prefix xmlns is bound to by
// definition (Namespaces in XML §3); xml's is xmltree.XMLNamespaceURI.
const xmlnsNamespace = "http://www.w3.org/2000/xmlns/"

// writtenBindings is the value.Context of the namespace bindings in scope at
// the element whose start tag e has open, the ones the document Append writes
// declares there: the empty prefix bound to e.ns, "" where it is absent; xml
// to the XML namespace; each prefix e.bound binds to the namespace name its
// innermost binding there gives it (encoder.boundSpace); and every other
// prefix, xmlns among them, unbound, as parser/xmltree answers xmlns. It reads e live,
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
	return b.e.boundSpace(prefix)
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
// encodeWalk.unionEncodes).
type noAssertions struct{}

// Evaluate declines every {test}.
func (noAssertions) Evaluate(value.Backend, xsd.TypeResolver, *xsd.SimpleType, xsd.XPathExpression, value.Value) value.AssertionAnswer {
	return value.AssertionDeclinedBecause("codec's encoder evaluates no assertion")
}
