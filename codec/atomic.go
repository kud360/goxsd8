package codec

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/xsd"
)

// atomic is the atomic Go type codegen gives an atomic simple type Decode
// decodes (codegen/doc.go "# Simple types"): string, bool, float32, float64,
// big.Rat, big.Int, a bounded Go integer, or the datatype struct of a date,
// time or duration datatype. The zero value is invalid (STYLE T1).
type atomic uint8

const (
	atomicString atomic = iota + 1
	atomicBool
	atomicFloat32
	atomicFloat64
	atomicBigRat
	atomicBigInt
	atomicInt8
	atomicInt16
	atomicInt32
	atomicInt64
	atomicUint8
	atomicUint16
	atomicUint32
	atomicUint64
	atomicDateTime
	atomicDateTimeStamp
	atomicTime
	atomicDate
	atomicGYearMonth
	atomicGYear
	atomicGMonthDay
	atomicGDay
	atomicGMonth
	atomicDuration
	atomicDayTimeDuration
	atomicYearMonthDuration
)

// The mirrors of the twelve date, time and duration datatype structs codegen
// emits (codegen/doc.go "# Simple types"): each holds codegen's exported
// fields exactly, in name, order and Go type, so that the struct codegen
// emits in the caller's package, which codec cannot name, converts to it and
// back (mirrored). TestDatatypeMirrorsAreCodegens pins each to codegen's.
type (
	dateTimeMirror struct {
		Year           big.Int
		Month          uint8
		Day            uint8
		Hour           uint8
		Minute         uint8
		Second         big.Rat
		TimezoneOffset *int16
	}
	dateTimeStampMirror struct {
		Year           big.Int
		Month          uint8
		Day            uint8
		Hour           uint8
		Minute         uint8
		Second         big.Rat
		TimezoneOffset int16
	}
	timeMirror struct {
		Hour           uint8
		Minute         uint8
		Second         big.Rat
		TimezoneOffset *int16
	}
	dateMirror struct {
		Year           big.Int
		Month          uint8
		Day            uint8
		TimezoneOffset *int16
	}
	gYearMonthMirror struct {
		Year           big.Int
		Month          uint8
		TimezoneOffset *int16
	}
	gYearMirror struct {
		Year           big.Int
		TimezoneOffset *int16
	}
	gMonthDayMirror struct {
		Month          uint8
		Day            uint8
		TimezoneOffset *int16
	}
	gDayMirror struct {
		Day            uint8
		TimezoneOffset *int16
	}
	gMonthMirror struct {
		Month          uint8
		TimezoneOffset *int16
	}
	durationMirror struct {
		Months  big.Int
		Seconds big.Rat
	}
	dayTimeDurationMirror struct {
		Seconds big.Rat
	}
	yearMonthDurationMirror struct {
		Months big.Int
	}
)

// atomicTypes holds, indexed by atomic, each one's Go type: for a datatype
// struct, codec's mirror of it.
var atomicTypes = [...]reflect.Type{
	atomicString:            reflect.TypeFor[string](),
	atomicBool:              reflect.TypeFor[bool](),
	atomicFloat32:           reflect.TypeFor[float32](),
	atomicFloat64:           reflect.TypeFor[float64](),
	atomicBigRat:            reflect.TypeFor[big.Rat](),
	atomicBigInt:            reflect.TypeFor[big.Int](),
	atomicInt8:              reflect.TypeFor[int8](),
	atomicInt16:             reflect.TypeFor[int16](),
	atomicInt32:             reflect.TypeFor[int32](),
	atomicInt64:             reflect.TypeFor[int64](),
	atomicUint8:             reflect.TypeFor[uint8](),
	atomicUint16:            reflect.TypeFor[uint16](),
	atomicUint32:            reflect.TypeFor[uint32](),
	atomicUint64:            reflect.TypeFor[uint64](),
	atomicDateTime:          reflect.TypeFor[dateTimeMirror](),
	atomicDateTimeStamp:     reflect.TypeFor[dateTimeStampMirror](),
	atomicTime:              reflect.TypeFor[timeMirror](),
	atomicDate:              reflect.TypeFor[dateMirror](),
	atomicGYearMonth:        reflect.TypeFor[gYearMonthMirror](),
	atomicGYear:             reflect.TypeFor[gYearMirror](),
	atomicGMonthDay:         reflect.TypeFor[gMonthDayMirror](),
	atomicGDay:              reflect.TypeFor[gDayMirror](),
	atomicGMonth:            reflect.TypeFor[gMonthMirror](),
	atomicDuration:          reflect.TypeFor[durationMirror](),
	atomicDayTimeDuration:   reflect.TypeFor[dayTimeDurationMirror](),
	atomicYearMonthDuration: reflect.TypeFor[yearMonthDurationMirror](),
}

// datatypeNames holds, indexed by a mirrored atomic, the local name of the
// XSD datatype codegen names its datatype struct from (codegen/doc.go, the
// datatype struct names): dateTime's is DateTime.
var datatypeNames = [...]string{
	atomicDateTime:          "dateTime",
	atomicDateTimeStamp:     "dateTimeStamp",
	atomicTime:              "time",
	atomicDate:              "date",
	atomicGYearMonth:        "gYearMonth",
	atomicGYear:             "gYear",
	atomicGMonthDay:         "gMonthDay",
	atomicGDay:              "gDay",
	atomicGMonth:            "gMonth",
	atomicDuration:          "duration",
	atomicDayTimeDuration:   "dayTimeDuration",
	atomicYearMonthDuration: "yearMonthDuration",
}

// bare returns a's Go type itself, unnamed, or for a datatype struct codec's
// mirror of it.
func (a atomic) bare() reflect.Type {
	if a == 0 || int(a) >= len(atomicTypes) {
		// The zero atomic is an internal bug, never input.
		panic(fmt.Sprintf("codec: invalid atomic %d", a))
	}
	return atomicTypes[a]
}

// mirrored reports whether the Go type codegen gives a is a datatype struct,
// which codegen declares in the caller's package, so that codec holds only a
// struct of the identical underlying type, its mirror (bare). It is read from
// the mirror's package: big.Rat and big.Int are structs too, of math/big.
func (a atomic) mirrored() bool {
	return a.bare().PkgPath() == reflect.TypeFor[atomic]().PkgPath()
}

// readsLexical reports whether a's value is read from the normalized lexical
// its governing mapping's Parse accepted, never through that mapping's
// Canonical: yearMonthDuration alone, whose zero value has no canonical
// representation (Datatypes §3.4.26.1, §2.3.1 dt-canonical-mapping's "where
// possible").
func (a atomic) readsLexical() bool { return a == atomicYearMonthDuration }

// spell spells a's Go type for an error message: its own name, or for a
// datatype struct "D for the datatype struct D of" the datatype's QName and
// the struct's fields, never the mirror's name.
func (a atomic) spell() string {
	if !a.mirrored() {
		return a.bare().String()
	}
	t := a.bare()
	fields := make([]string, t.NumField())
	for i := range fields {
		fields[i] = t.Field(i).Name + " " + t.Field(i).Type.String()
	}
	return fmt.Sprintf("D for the datatype struct D of xs:%s (%s)", datatypeNames[a], strings.Join(fields, ", "))
}

// convert returns the value of a's bare type that form denotes, read as
// Decode's doc states, or the error for a form a's Go type does not read:
// form is a canonical lexical form the governing mapping's Canonical
// rendered, or where a readsLexical the normalized lexical Parse accepted. A
// boolean is never read by strconv.ParseBool, which accepts more than
// f-booleanCanmap's two forms. atomicString has no conversion and panics.
func (a atomic) convert(form string) (reflect.Value, error) {
	if a.mirrored() {
		return convertTemporal(a, form)
	}
	v := reflect.New(a.bare()).Elem()
	switch a {
	case atomicBool:
		switch form {
		case "true":
			v.SetBool(true)
		case "false":
		default:
			return reflect.Value{}, errors.New("a boolean's canonical form is true or false (f-booleanCanmap)")
		}
	case atomicFloat32, atomicFloat64:
		f, err := strconv.ParseFloat(form, a.bare().Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		v.SetFloat(f)
	case atomicBigRat:
		r, ok := new(big.Rat).SetString(form)
		if !ok {
			return reflect.Value{}, errors.New("big.Rat does not read it")
		}
		v.Set(reflect.ValueOf(r).Elem())
	case atomicBigInt:
		i, ok := new(big.Int).SetString(form, 10)
		if !ok {
			return reflect.Value{}, errors.New("big.Int does not read it in base 10")
		}
		v.Set(reflect.ValueOf(i).Elem())
	case atomicInt8, atomicInt16, atomicInt32, atomicInt64:
		n, err := strconv.ParseInt(form, 10, a.bare().Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		v.SetInt(n)
	case atomicUint8, atomicUint16, atomicUint32, atomicUint64:
		n, err := strconv.ParseUint(form, 10, a.bare().Bits())
		if err != nil {
			return reflect.Value{}, err
		}
		v.SetUint(n)
	default:
		// atomicString's value is the normalized lexical itself.
		panic(fmt.Sprintf("codec: atomic %s has no conversion", a.bare()))
	}
	return v, nil
}

// chainAtomics maps each builtin that decides an atomic type's atomic from
// anywhere on its {base type definition} chain to that atomic: the eight
// bounded integer builtins (Datatypes §3.4.16–§3.4.19, §3.4.21–§3.4.24) to
// the Go integer whose range is their value space, xs:integer (§3.4.13) to
// big.Int, and dateTimeStamp, dayTimeDuration and yearMonthDuration
// (§3.4.26–§3.4.28) to their datatype structs, the twelve codegen's
// builtinType walks for. It is a lookup index only and never iterated (STYLE
// D2).
var chainAtomics = map[xsd.QName]atomic{
	{Space: xsd.XMLSchemaNS, Local: "long"}:              atomicInt64,
	{Space: xsd.XMLSchemaNS, Local: "int"}:               atomicInt32,
	{Space: xsd.XMLSchemaNS, Local: "short"}:             atomicInt16,
	{Space: xsd.XMLSchemaNS, Local: "byte"}:              atomicInt8,
	{Space: xsd.XMLSchemaNS, Local: "unsignedLong"}:      atomicUint64,
	{Space: xsd.XMLSchemaNS, Local: "unsignedInt"}:       atomicUint32,
	{Space: xsd.XMLSchemaNS, Local: "unsignedShort"}:     atomicUint16,
	{Space: xsd.XMLSchemaNS, Local: "unsignedByte"}:      atomicUint8,
	{Space: xsd.XMLSchemaNS, Local: "integer"}:           atomicBigInt,
	{Space: xsd.XMLSchemaNS, Local: "dateTimeStamp"}:     atomicDateTimeStamp,
	{Space: xsd.XMLSchemaNS, Local: "dayTimeDuration"}:   atomicDayTimeDuration,
	{Space: xsd.XMLSchemaNS, Local: "yearMonthDuration"}: atomicYearMonthDuration,
}

// primitiveAtomics maps each {primitive type definition} Decode decodes to
// the atomic it gives a type no builtin of chainAtomics decides
// (codegen/doc.go "# Simple types"); it is a lookup index only and never
// iterated (STYLE D2).
var primitiveAtomics = map[xsd.QName]atomic{
	{Space: xsd.XMLSchemaNS, Local: "string"}:     atomicString,
	{Space: xsd.XMLSchemaNS, Local: "anyURI"}:     atomicString,
	{Space: xsd.XMLSchemaNS, Local: "boolean"}:    atomicBool,
	{Space: xsd.XMLSchemaNS, Local: "float"}:      atomicFloat32,
	{Space: xsd.XMLSchemaNS, Local: "double"}:     atomicFloat64,
	{Space: xsd.XMLSchemaNS, Local: "decimal"}:    atomicBigRat,
	{Space: xsd.XMLSchemaNS, Local: "dateTime"}:   atomicDateTime,
	{Space: xsd.XMLSchemaNS, Local: "time"}:       atomicTime,
	{Space: xsd.XMLSchemaNS, Local: "date"}:       atomicDate,
	{Space: xsd.XMLSchemaNS, Local: "gYearMonth"}: atomicGYearMonth,
	{Space: xsd.XMLSchemaNS, Local: "gYear"}:      atomicGYear,
	{Space: xsd.XMLSchemaNS, Local: "gMonthDay"}:  atomicGMonthDay,
	{Space: xsd.XMLSchemaNS, Local: "gDay"}:       atomicGDay,
	{Space: xsd.XMLSchemaNS, Local: "gMonth"}:     atomicGMonth,
	{Space: xsd.XMLSchemaNS, Local: "duration"}:   atomicDuration,
}

// chainAtomic returns the atomic of st as codegen/doc.go "# Simple types"
// picks it from st's {base type definition} chain: walking it from st itself,
// the first builtin of chainAtomics met gives its atomic, and false where the
// walk meets none. Every bounded builtin lies below xs:integer, so the nearest
// wins over it, and no chain meets both an integer builtin and a temporal one.
func chainAtomic(r xsd.TypeResolver, st *xsd.SimpleType) (atomic, bool, error) {
	for step := st; step != nil; {
		if a, ok := chainAtomics[step.Name()]; ok {
			return a, true, nil
		}
		base, err := step.Base(r)
		if err != nil {
			return 0, false, err
		}
		step = base
	}
	return 0, false, nil
}
