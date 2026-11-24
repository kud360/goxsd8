package codec

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/internal/xmlqname"
	"github.com/kud360/goxsd8/value"
)

// atomic is goshape.Atomic, with how codec holds each one's value (bare) and
// reads it (reading, convert). The zero value is invalid (STYLE T1).
type atomic goshape.Atomic

// The atomics, each goshape's of its name.
const (
	atomicString            = atomic(goshape.AtomicString)
	atomicBool              = atomic(goshape.AtomicBool)
	atomicFloat32           = atomic(goshape.AtomicFloat32)
	atomicFloat64           = atomic(goshape.AtomicFloat64)
	atomicBigRat            = atomic(goshape.AtomicBigRat)
	atomicBigInt            = atomic(goshape.AtomicBigInt)
	atomicInt8              = atomic(goshape.AtomicInt8)
	atomicInt16             = atomic(goshape.AtomicInt16)
	atomicInt32             = atomic(goshape.AtomicInt32)
	atomicInt64             = atomic(goshape.AtomicInt64)
	atomicUint8             = atomic(goshape.AtomicUint8)
	atomicUint16            = atomic(goshape.AtomicUint16)
	atomicUint32            = atomic(goshape.AtomicUint32)
	atomicUint64            = atomic(goshape.AtomicUint64)
	atomicDateTime          = atomic(goshape.AtomicDateTime)
	atomicDateTimeStamp     = atomic(goshape.AtomicDateTimeStamp)
	atomicTime              = atomic(goshape.AtomicTime)
	atomicDate              = atomic(goshape.AtomicDate)
	atomicGYearMonth        = atomic(goshape.AtomicGYearMonth)
	atomicGYear             = atomic(goshape.AtomicGYear)
	atomicGMonthDay         = atomic(goshape.AtomicGMonthDay)
	atomicGDay              = atomic(goshape.AtomicGDay)
	atomicGMonth            = atomic(goshape.AtomicGMonth)
	atomicDuration          = atomic(goshape.AtomicDuration)
	atomicDayTimeDuration   = atomic(goshape.AtomicDayTimeDuration)
	atomicYearMonthDuration = atomic(goshape.AtomicYearMonthDuration)
	atomicHexBinary         = atomic(goshape.AtomicHexBinary)
	atomicBase64Binary      = atomic(goshape.AtomicBase64Binary)
	atomicQName             = atomic(goshape.AtomicQName)
	atomicPrecisionDecimal  = atomic(goshape.AtomicPrecisionDecimal)
)

// atomicTypes holds, indexed by atomic, each one's bare Go type: for a
// datatype struct other than PrecisionDecimal (codegen/doc.go "# Simple
// types"), the unnamed struct reflect.StructOf builds of goshape.Atomic.Fields'
// fields, to which the struct codegen emits in the caller's package, which
// codec cannot name, converts and back, its underlying type being identical;
// and for PrecisionDecimal's, whose one field is of the caller's own sealed
// interface, so that no struct of codec's converts to it, codec's
// decimalValue.
var atomicTypes = func() []reflect.Type {
	types := []reflect.Type{
		atomicString:           reflect.TypeFor[string](),
		atomicBool:             reflect.TypeFor[bool](),
		atomicFloat32:          reflect.TypeFor[float32](),
		atomicFloat64:          reflect.TypeFor[float64](),
		atomicBigRat:           reflect.TypeFor[big.Rat](),
		atomicBigInt:           reflect.TypeFor[big.Int](),
		atomicInt8:             reflect.TypeFor[int8](),
		atomicInt16:            reflect.TypeFor[int16](),
		atomicInt32:            reflect.TypeFor[int32](),
		atomicInt64:            reflect.TypeFor[int64](),
		atomicUint8:            reflect.TypeFor[uint8](),
		atomicUint16:           reflect.TypeFor[uint16](),
		atomicUint32:           reflect.TypeFor[uint32](),
		atomicUint64:           reflect.TypeFor[uint64](),
		atomicHexBinary:        reflect.TypeFor[[]byte](),
		atomicBase64Binary:     reflect.TypeFor[[]byte](),
		atomicPrecisionDecimal: decimalValueType,
	}
	for a := range types {
		if fields := goshape.Atomic(a).Fields(); fields != nil {
			types[a] = structOf(fields)
		}
	}
	return types
}()

// structOf returns the unnamed struct of fields, in order, none embedded.
func structOf(fields []goshape.DatatypeField) reflect.Type {
	sf := make([]reflect.StructField, len(fields))
	for i, f := range fields {
		sf[i] = reflect.StructField{Name: f.Name, Type: f.Type}
	}
	return reflect.StructOf(sf)
}

// bare returns a's Go type itself, unnamed, or for a datatype struct the
// unnamed struct of its fields (atomicTypes), or decimalValue for
// PrecisionDecimal's, which converts to no caller type.
func (a atomic) bare() reflect.Type {
	if a == 0 || int(a) >= len(atomicTypes) || atomicTypes[a] == nil {
		// The zero atomic is an internal bug, never input, and so is a
		// datatype struct goshape gives no fields.
		panic(fmt.Sprintf("codec: invalid atomic %d", a))
	}
	return atomicTypes[a]
}

// build returns the value of a's bare type, a datatype struct's other than
// PrecisionDecimal's, whose fields hold props, each set by its field's name:
// props names every field of it once, in any order. A field props does not
// name, a name it holds twice or that no field has, and a value of another
// type than its field's are internal bugs, which panic.
func (a atomic) build(props ...prop) reflect.Value {
	v := reflect.New(a.bare()).Elem()
	if len(props) != v.NumField() {
		panic(fmt.Sprintf("codec: %d properties for the %d fields of %s", len(props), v.NumField(), a.bare()))
	}
	set := make([]bool, v.NumField())
	for _, p := range props {
		f, ok := v.Type().FieldByName(p.name)
		if !ok || set[f.Index[0]] {
			panic(fmt.Sprintf("codec: %s has no field %s, or it is set twice", a.bare(), p.name))
		}
		set[f.Index[0]] = true
		v.Field(f.Index[0]).Set(reflect.ValueOf(p.value))
	}
	return v
}

// prop is one property of a datatype struct's value for build: the name of
// its field and the value it holds.
type prop struct {
	name  string
	value any
}

// datatypeStruct reports whether the Go type codegen gives a is a datatype
// struct, PrecisionDecimal's included: whether goshape.Atomic.Datatype
// names its datatype.
func (a atomic) datatypeStruct() bool {
	_, ok := goshape.Atomic(a).Datatype()
	return ok
}

// shapes reports whether v is a struct of the shape of the datatype struct
// codegen gives a, a datatypeStruct atomic, its name never compared: for any
// but PrecisionDecimal's, one that converts to a's bare type, so of its
// exported fields exactly, in name, order and Go type, struct tags aside; for
// PrecisionDecimal's, one decimalShape holds.
func (a atomic) shapes(v reflect.Type) bool {
	if a == atomicPrecisionDecimal {
		return decimalShape(v)
	}
	return v.ConvertibleTo(a.bare())
}

// reading is what an atomic's value is read from once its governing
// mapping's Parse has accepted the normalized lexical. The zero value is
// invalid (STYLE T1).
type reading uint8

const (
	// readCanonical is the form the governing mapping's Canonical renders the
	// value in (Datatypes §2.3.1 dt-canonical-mapping).
	readCanonical reading = iota + 1
	// readLexical is the normalized lexical Parse accepted, never Canonical:
	// yearMonthDuration's, whose zero value has no canonical representation
	// (§3.4.26.1, §2.3.1 dt-canonical-mapping's "where possible"), and
	// precisionDecimal's, read by the lexical mapping (f-precDecLexmap),
	// since its canonical mapping decides no rendering of a zero
	// (precisionDecimal §6 f-precDecCanmap), so a Canonical cannot be relied
	// on to keep a zero's ·scale· or ·sign·.
	readLexical
	// readExpanded is the expanded name the normalized lexical resolves to
	// against the bindings Parse was given, never Canonical: QName's, which
	// NOTATION shares, neither having a canonical representation (§3.3.18,
	// §3.3.19).
	readExpanded
)

// reading returns what a's value is read from.
func (a atomic) reading() reading {
	if a == atomicYearMonthDuration || a == atomicPrecisionDecimal {
		return readLexical
	}
	if a == atomicQName {
		return readExpanded
	}
	return readCanonical
}

// spell spells a's Go type for an error message: its own name, []byte for
// either binary arm (reflect spells it []uint8), or for a datatype struct "D
// for the datatype struct D of" the datatype's QName and the struct's fields,
// never the bare type's spelling, PrecisionDecimal's Value field by its role,
// its type being the caller's interface.
func (a atomic) spell() string {
	if a == atomicHexBinary || a == atomicBase64Binary {
		return "[]byte"
	}
	if a == atomicPrecisionDecimal {
		return "D for the datatype struct D of xs:precisionDecimal (Value I, I a sealed interface whose arms WithPrecisionDecimalArms supplies)"
	}
	if !a.datatypeStruct() {
		return a.bare().String()
	}
	t := a.bare()
	fields := make([]string, t.NumField())
	for i := range fields {
		fields[i] = t.Field(i).Name + " " + t.Field(i).Type.String()
	}
	datatype, _ := goshape.Atomic(a).Datatype()
	return fmt.Sprintf("D for the datatype struct D of xs:%s (%s)", datatype.Local, strings.Join(fields, ", "))
}

// convert returns the value of a's bare type that form denotes, read as
// Decode's doc states, or the error for a form a's Go type does not read:
// form is a canonical lexical form the governing mapping's Canonical
// rendered, or where a's reading is readLexical the normalized lexical Parse
// accepted. A boolean is never read by strconv.ParseBool, which accepts more
// than f-booleanCanmap's two forms. PrecisionDecimal's is convertDecimal's,
// and every atomic convert does not name is convertTemporal's, which converts
// the date, time and duration ones and panics for the rest, atomicString and
// atomicQName, read by expandedName, having no conversion.
func (a atomic) convert(form string) (reflect.Value, error) {
	switch a {
	case atomicPrecisionDecimal:
		return convertDecimal(form)
	case atomicBool:
		switch form {
		case "true":
			return reflect.ValueOf(true), nil
		case "false":
			return reflect.ValueOf(false), nil
		}
		return reflect.Value{}, errors.New("a boolean's canonical form is true or false (f-booleanCanmap)")
	case atomicFloat32, atomicFloat64:
		f, err := strconv.ParseFloat(form, a.bare().Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(f).Convert(a.bare()), nil
	case atomicBigRat:
		r, ok := new(big.Rat).SetString(form)
		if !ok {
			return reflect.Value{}, errors.New("big.Rat does not read it")
		}
		return reflect.ValueOf(r).Elem(), nil
	case atomicBigInt:
		i, ok := new(big.Int).SetString(form, 10)
		if !ok {
			return reflect.Value{}, errors.New("big.Int does not read it in base 10")
		}
		return reflect.ValueOf(i).Elem(), nil
	case atomicInt8, atomicInt16, atomicInt32, atomicInt64:
		n, err := strconv.ParseInt(form, 10, a.bare().Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(n).Convert(a.bare()), nil
	case atomicUint8, atomicUint16, atomicUint32, atomicUint64:
		n, err := strconv.ParseUint(form, 10, a.bare().Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(n).Convert(a.bare()), nil
	case atomicHexBinary:
		// form is f-hexBinaryCanonical's (Datatypes §E.4.1); its letter case
		// is not compared, hex.DecodeString reading either.
		b, err := hex.DecodeString(form)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(b), nil
	case atomicBase64Binary:
		// form is Canonical-base64Binary's (Datatypes §3.3.16.2).
		b, err := base64.StdEncoding.DecodeString(form)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(b), nil
	default:
		return convertTemporal(a, form)
	}
}

// formed reports whether form spells a's values, so that spelled writes them:
// every atomic but atomicString, which str writes, atomicFloat32 and
// atomicFloat64, which double writes, atomicQName, which qname writes, and
// atomicPrecisionDecimal, which precisionDecimal writes. atom, list and
// TestFormArms read it.
func (a atomic) formed() bool {
	return a != atomicString && a != atomicFloat32 && a != atomicFloat64 && a != atomicQName && a != atomicPrecisionDecimal
}

// form returns, as convert's inverse, the canonical representation of v, a
// value of a Go type that converts to a's bare type, codegen's named type over
// it among them, a being the atomic Go type of a type whose primitive is one
// of encodedPrimitives and an atomic formed holds for, and a nil error: a bool
// as true or false (f-booleanCanmap); an integer in base 10 with no decimal
// point, no '+' and no leading zero (f-decimalCanmap through f-noDecCanMap);
// any other big.Rat as the shortest decimal expansion that is exact, a digit
// at least on each side of the point (f-decCanFragMap); a []byte of
// xs:hexBinary as two upper-case hex digits per octet (Datatypes §E.4.1
// f-hexBinaryCanonical) and of xs:base64Binary as padded Base64 with no white
// space (§3.3.16.2 Canonical-base64Binary); and a date, time or duration
// datatype struct value as temporalForm states. A non-nil error is the cause
// v lies outside a's value space, as item 6 of [Append]'s doc states it,
// which no lexical spells: a big.Rat whose denominator in lowest terms is no
// 2^m·5^n, or a datatype struct value temporalForm refuses; it names the
// property, its value and the bound, and carries no *xsderr.Error, since no
// rule charges it. Any other a is an internal bug, which panics.
func (a atomic) form(v reflect.Value) (string, error) {
	v = v.Convert(a.bare())
	switch a {
	case atomicBool:
		return strconv.FormatBool(v.Bool()), nil
	case atomicBigRat:
		r := v.Interface().(big.Rat)
		return decimalForm(&r)
	case atomicBigInt:
		i := v.Interface().(big.Int)
		return i.String(), nil
	case atomicInt8, atomicInt16, atomicInt32, atomicInt64:
		return strconv.FormatInt(v.Int(), 10), nil
	case atomicUint8, atomicUint16, atomicUint32, atomicUint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case atomicHexBinary:
		return strings.ToUpper(hex.EncodeToString(v.Bytes())), nil
	case atomicBase64Binary:
		return base64.StdEncoding.EncodeToString(v.Bytes()), nil
	default:
		return temporalForm(a, v)
	}
}

// shown spells v, a value of a Go type that converts to a's bare type, for an
// error message about a value form refuses: a big.Rat as a fraction
// (big.Rat.RatString), a datatype struct as its fields, each its name and its
// value, a nil *int16 as nil, and any other value as fmt's %v.
func (a atomic) shown(v reflect.Value) string {
	v = v.Convert(a.bare())
	if !a.datatypeStruct() {
		return shownField(v)
	}
	fields := make([]string, v.NumField())
	for i := range fields {
		fields[i] = v.Type().Field(i).Name + " " + shownField(v.Field(i))
	}
	return "{" + strings.Join(fields, ", ") + "}"
}

// shownField is shown's spelling of one value or field.
func shownField(v reflect.Value) string {
	switch x := v.Interface().(type) {
	case big.Rat:
		return x.RatString()
	case big.Int:
		return x.String()
	case *int16:
		if x == nil {
			return "nil"
		}
		return strconv.Itoa(int(*x))
	}
	return fmt.Sprint(v.Interface())
}

// otherLexicalOf reports whether v, a value of a Go type that converts to a's
// bare type, has a lexical other than its canonical representation, which a
// pattern facet may match where it rejects the canonical one (Datatypes
// §2.1's note), so that the rejection is item 7 of [Append]'s doc and not item
// 6, for each primitive as item 7 states it. The other lexical is, for a
// float32 or float64, any floatRep or doubleRep spelling but for NaN and -INF
// (soleLexical); for an xs:boolean value 1 or 0 (§3.3.2.2); for an
// xs:decimal-primitive or duration-primitive one a leading zero; for a
// dateTime, dateTimeStamp or time one a trailing zero of its seconds; for a
// date or g* value, one item 7 spells; for an xs:hexBinary value a lower-case
// hex digit (§3.3.15.2's hexDigit production); and for an xs:base64Binary
// value a single space between two of its characters (§3.3.16.2's B64
// production and its note); for an xs:QName or xs:NOTATION value, which
// has no canonical representation, its Local under a prefix other than
// spellQName's bound to its Space; and for an xs:precisionDecimal value, v
// being of atomicPrecisionDecimal's bare type, the decimalValue encoder.held
// translates the caller's into, any pDecimalRep spelling but for NaN and -INF
// (decimalValue.otherLexical). An a of a primitive atomicEncodes declines is
// an internal bug, which panics.
func (a atomic) otherLexicalOf(v reflect.Value) bool {
	switch a {
	case atomicString:
		return false
	case atomicFloat32, atomicFloat64:
		return !soleLexical(v.Float())
	case atomicHexBinary:
		return hexLetter(v.Bytes())
	case atomicBase64Binary:
		return v.Len() > 0
	case atomicBool, atomicBigRat, atomicBigInt, atomicInt8, atomicInt16, atomicInt32, atomicInt64, atomicUint8, atomicUint16, atomicUint32, atomicUint64,
		atomicDateTime, atomicDateTimeStamp, atomicTime, atomicDuration, atomicDayTimeDuration, atomicYearMonthDuration:
		return true
	case atomicDate, atomicGYearMonth, atomicGYear:
		v = v.Convert(a.bare())
		year := v.FieldByName("Year").Interface().(big.Int)
		return zeroOffset(v) || year.Sign() == 0
	case atomicGMonthDay, atomicGDay, atomicGMonth:
		return zeroOffset(v.Convert(a.bare()))
	case atomicQName:
		return v.Convert(a.bare()).FieldByName("Space").String() != ""
	case atomicPrecisionDecimal:
		return v.Interface().(decimalValue).otherLexical()
	default:
		panic(fmt.Sprintf("codec: no other-lexical answer for atomic %d", a))
	}
}

// hexLetter reports whether one of b's octets has a nibble of 0xA or above,
// which f-hexDigitCanonical spells by a letter (Datatypes §E.4.1).
func hexLetter(b []byte) bool {
	for _, o := range b {
		if o>>4 >= 0xA || o&0xF >= 0xA {
			return true
		}
	}
	return false
}

// zeroOffset reports whether v, a datatype struct value of a's bare type with
// a *int16 TimezoneOffset, holds the timezoneOffset 0.
func zeroOffset(v reflect.Value) bool {
	off := v.FieldByName("TimezoneOffset").Interface().(*int16)
	return off != nil && *off == 0
}

// decimalForm is form's for a big.Rat r: its digits after the point number
// the larger of the exponents of 2 and 5 in its denominator, the fewest that
// spell r exactly, and none for an integer; or form's error for an r outside
// the decimal value space.
func decimalForm(r *big.Rat) (string, error) {
	if r.IsInt() {
		return r.Num().String(), nil
	}
	d := new(big.Int).Set(r.Denom())
	twos := d.TrailingZeroBits()
	d.Rsh(d, twos)
	fives := uint(0)
	q, m := new(big.Int), new(big.Int)
	for {
		q.QuoRem(d, bigFive, m)
		if m.Sign() != 0 {
			break
		}
		d.Set(q)
		fives++
	}
	if d.Cmp(bigOne) != 0 {
		return "", errOutsideDecimal
	}
	return r.FloatString(int(max(twos, fives))), nil
}

// errOutsideDecimal is decimalForm's error, a plain one, since no rule
// charges it.
var errOutsideDecimal = errors.New("its denominator in lowest terms is no 2^m·5^n, so it lies outside the decimal value space (Datatypes §3.3.3.1)")

// bigOne and bigFive are the constants decimalForm compares with and divides
// by; nothing writes them.
var (
	bigOne  = big.NewInt(1)
	bigFive = big.NewInt(5)
)

// expandedName returns the value of QName's datatype struct, as a value of
// atomicQName's bare type, that lexical denotes: the expanded name it
// resolves to against ctx, the bindings the governing mapping's Parse
// accepted it under (Datatypes §3.3.18, §3.3.19), Space its namespace
// name, "" for none, and Local its local part, or the error where it
// resolves to none.
func expandedName(lexical string, ctx value.Context) (reflect.Value, error) {
	space, local, ok := xmlqname.Resolve(lexical, ctx.LookupNamespace)
	if !ok {
		return reflect.Value{}, errors.New("it resolves to no expanded name against the bindings in scope")
	}
	return atomicQName.build(prop{"Space", space}, prop{"Local", local}), nil
}
