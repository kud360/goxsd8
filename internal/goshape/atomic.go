package goshape

import (
	"fmt"
	"math/big"
	"reflect"
	"slices"

	"github.com/kud360/goxsd8/xsd"
)

// Atomic is one of the atomic Go types codegen/doc.go "# Simple types" names,
// which owns the list, a special type's string among them: one constant per
// Go type, but two for []byte, xs:hexBinary's and xs:base64Binary's, whose
// values are read apart, and none of NOTATION's own, which shares QName's
// datatype struct. The zero value is invalid (STYLE T1).
type Atomic uint8

// The Atomics.
const (
	AtomicString Atomic = iota + 1
	AtomicBool
	AtomicFloat32
	AtomicFloat64
	AtomicBigRat
	AtomicBigInt
	AtomicInt8
	AtomicInt16
	AtomicInt32
	AtomicInt64
	AtomicUint8
	AtomicUint16
	AtomicUint32
	AtomicUint64
	AtomicDateTime
	AtomicDateTimeStamp
	AtomicTime
	AtomicDate
	AtomicGYearMonth
	AtomicGYear
	AtomicGMonthDay
	AtomicGDay
	AtomicGMonth
	AtomicDuration
	AtomicDayTimeDuration
	AtomicYearMonthDuration
	AtomicHexBinary
	AtomicBase64Binary
	AtomicQName
	AtomicPrecisionDecimal
)

// datatypeNames holds, indexed by an Atomic whose Go type is a datatype
// struct, the local name, in the XSD namespace, of the datatype whose
// datatype struct it is, the one encoding of that pairing (Datatype reads
// it, datatypeAtomics indexes it by name). xs:NOTATION, which shares QName's
// struct, is no entry: primitiveAtomics keys it to AtomicQName.
var datatypeNames = [...]string{
	AtomicDateTime:          "dateTime",
	AtomicDateTimeStamp:     "dateTimeStamp",
	AtomicTime:              "time",
	AtomicDate:              "date",
	AtomicGYearMonth:        "gYearMonth",
	AtomicGYear:             "gYear",
	AtomicGMonthDay:         "gMonthDay",
	AtomicGDay:              "gDay",
	AtomicGMonth:            "gMonth",
	AtomicDuration:          "duration",
	AtomicDayTimeDuration:   "dayTimeDuration",
	AtomicYearMonthDuration: "yearMonthDuration",
	AtomicQName:             "QName",
	AtomicPrecisionDecimal:  "precisionDecimal",
}

// Datatype returns the XSD datatype whose datatype struct is a's Go type,
// which codegen names the struct from (codegen/doc.go, the datatype struct
// names), and true; or false where a's Go type is no datatype struct.
func (a Atomic) Datatype() (xsd.QName, bool) {
	if int(a) >= len(datatypeNames) || datatypeNames[a] == "" {
		return xsd.QName{}, false
	}
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: datatypeNames[a]}, true
}

// DatatypeField is one field of a datatype struct, or of PrecisionDecimal's
// numerical arm, as codegen/doc.go "# Simple types" states it: its Go name and
// its Go type, which codegen spells by Type.String and codec builds a struct
// of by reflect.StructOf.
type DatatypeField struct {
	Name string
	Type reflect.Type
}

// The fields the datatype structs share: uint8 is the narrowest Go integer
// holding month 1–12, day 1–31, hour 0–23 and minute 0–59, int16 the narrowest
// holding a timezoneOffset of −840..840 (§D.2.1), and year, second, months and
// seconds are unbounded (§D.2.1, §3.3.6.1).
var (
	yearField           = DatatypeField{"Year", reflect.TypeFor[big.Int]()}
	monthField          = DatatypeField{"Month", reflect.TypeFor[uint8]()}
	dayField            = DatatypeField{"Day", reflect.TypeFor[uint8]()}
	hourField           = DatatypeField{"Hour", reflect.TypeFor[uint8]()}
	minuteField         = DatatypeField{"Minute", reflect.TypeFor[uint8]()}
	secondField         = DatatypeField{"Second", reflect.TypeFor[big.Rat]()}
	optionalOffsetField = DatatypeField{"TimezoneOffset", reflect.TypeFor[*int16]()}
	requiredOffsetField = DatatypeField{"TimezoneOffset", reflect.TypeFor[int16]()}
	monthsField         = DatatypeField{"Months", reflect.TypeFor[big.Int]()}
	secondsField        = DatatypeField{"Seconds", reflect.TypeFor[big.Rat]()}
)

// datatypeFields holds, indexed by Atomic, the fields of each datatype struct
// other than PrecisionDecimal, one per property its values carry, in the order
// of the seven-property model (§D.2.1), of duration's two properties
// (§3.3.6.1), or of QName's {namespace name, local part} (§3.3.18): one struct
// per date, time or duration primitive, one per builtin derived from one whose
// values carry fewer properties or a required one (§3.4.26 yearMonthDuration,
// §3.4.27 dayTimeDuration, §3.4.28 dateTimeStamp), and one for QName, which
// NOTATION shares (§3.3.18, §3.3.19). Every Atomic Datatype answers for but
// AtomicPrecisionDecimal has an entry, which TestDatatypeFields holds.
var datatypeFields = [...][]DatatypeField{
	AtomicDateTime:          {yearField, monthField, dayField, hourField, minuteField, secondField, optionalOffsetField}, // Datatypes §3.3.7
	AtomicDateTimeStamp:     {yearField, monthField, dayField, hourField, minuteField, secondField, requiredOffsetField}, // Datatypes §3.4.28
	AtomicTime:              {hourField, minuteField, secondField, optionalOffsetField},                                  // Datatypes §3.3.8
	AtomicDate:              {yearField, monthField, dayField, optionalOffsetField},                                      // Datatypes §3.3.9
	AtomicGYearMonth:        {yearField, monthField, optionalOffsetField},                                                // Datatypes §3.3.10
	AtomicGYear:             {yearField, optionalOffsetField},                                                            // Datatypes §3.3.11
	AtomicGMonthDay:         {monthField, dayField, optionalOffsetField},                                                 // Datatypes §3.3.12
	AtomicGDay:              {dayField, optionalOffsetField},                                                             // Datatypes §3.3.13
	AtomicGMonth:            {monthField, optionalOffsetField},                                                           // Datatypes §3.3.14
	AtomicDuration:          {monthsField, secondsField},                                                                 // Datatypes §3.3.6
	AtomicDayTimeDuration:   {secondsField},                                                                              // Datatypes §3.4.27
	AtomicYearMonthDuration: {monthsField},                                                                               // Datatypes §3.4.26
	AtomicQName:             {{"Space", reflect.TypeFor[string]()}, {"Local", reflect.TypeFor[string]()}},                // Datatypes §3.3.18, §3.3.19
}

// Fields returns the fields of a's datatype struct in declaration order, a
// copy the caller may keep; or nil where a's Go type is no datatype struct,
// and for AtomicPrecisionDecimal, whose struct's one field Value is of the
// caller's own sealed interface, its numerical arm's fields being
// NumericalFields'.
func (a Atomic) Fields() []DatatypeField {
	if int(a) >= len(datatypeFields) {
		return nil
	}
	return slices.Clone(datatypeFields[a])
}

// NumericalFields returns the fields of PrecisionDecimal's numerical arm in
// declaration order, a copy the caller may keep: Coefficient, Scale, its
// ·scale· (precisionDecimal vp-pd-precision), each a big.Int, and Negative,
// its ·sign· (vp-pd-sign) negative.
func NumericalFields() []DatatypeField {
	return []DatatypeField{
		{"Coefficient", reflect.TypeFor[big.Int]()},
		{"Scale", reflect.TypeFor[big.Int]()},
		{"Negative", reflect.TypeFor[bool]()},
	}
}

// chainAtomics maps each integer builtin that decides an atomic type's
// Atomic from anywhere on its {base type definition} chain to that Atomic:
// the eight bounded integer builtins (Datatypes §3.4.16–§3.4.19,
// §3.4.21–§3.4.24) to the Go integer whose range is their value space, and
// xs:integer (§3.4.13) to big.Int. It is a lookup index only and never
// iterated (STYLE D2).
var chainAtomics = map[xsd.QName]Atomic{
	{Space: xsd.XMLSchemaNS, Local: "long"}:          AtomicInt64,
	{Space: xsd.XMLSchemaNS, Local: "int"}:           AtomicInt32,
	{Space: xsd.XMLSchemaNS, Local: "short"}:         AtomicInt16,
	{Space: xsd.XMLSchemaNS, Local: "byte"}:          AtomicInt8,
	{Space: xsd.XMLSchemaNS, Local: "unsignedLong"}:  AtomicUint64,
	{Space: xsd.XMLSchemaNS, Local: "unsignedInt"}:   AtomicUint32,
	{Space: xsd.XMLSchemaNS, Local: "unsignedShort"}: AtomicUint16,
	{Space: xsd.XMLSchemaNS, Local: "unsignedByte"}:  AtomicUint8,
	{Space: xsd.XMLSchemaNS, Local: "integer"}:       AtomicBigInt,
}

// datatypeAtomics maps the QName of each datatype of datatypeNames to its
// datatype struct's Atomic, built from datatypeNames once; it is a lookup
// index only and never iterated (STYLE D2).
var datatypeAtomics = func() map[xsd.QName]Atomic {
	m := map[xsd.QName]Atomic{}
	for a := range datatypeNames {
		if name, ok := Atomic(a).Datatype(); ok {
			m[name] = Atomic(a)
		}
	}
	return m
}()

// primitiveAtomics maps each {primitive type definition} that
// datatypeAtomics does not hold, and that gives an Atomic, to the Atomic it
// gives a type whose {base type definition} chain meets no builtin of
// chainAtomics or datatypeAtomics: xs:NOTATION among them, to QName's
// datatype struct, which it shares. It is a lookup index only and never
// iterated (STYLE D2).
var primitiveAtomics = map[xsd.QName]Atomic{
	{Space: xsd.XMLSchemaNS, Local: "string"}:       AtomicString,
	{Space: xsd.XMLSchemaNS, Local: "anyURI"}:       AtomicString,
	{Space: xsd.XMLSchemaNS, Local: "boolean"}:      AtomicBool,
	{Space: xsd.XMLSchemaNS, Local: "float"}:        AtomicFloat32,
	{Space: xsd.XMLSchemaNS, Local: "double"}:       AtomicFloat64,
	{Space: xsd.XMLSchemaNS, Local: "decimal"}:      AtomicBigRat,
	{Space: xsd.XMLSchemaNS, Local: "hexBinary"}:    AtomicHexBinary,
	{Space: xsd.XMLSchemaNS, Local: "base64Binary"}: AtomicBase64Binary,
	{Space: xsd.XMLSchemaNS, Local: "NOTATION"}:     AtomicQName,
}

// AtomicType returns the Atomic codegen/doc.go "# Simple types" gives the
// simple type st, which owns the rule, and true; or false where it gives
// none, st's {primitive type definition} being none of the twenty the rule
// names, an ·implementation-defined· one other than xs:precisionDecimal such
// as one xsd.NewPrimitiveType mints, which each caller declines in its own
// terms. A special type, known by identity ([xsd.SimpleType.IsSpecial])
// ahead of its {variety}, gives AtomicString. An atomic st's Atomic is that
// of the first builtin of chainAtomics or datatypeAtomics met walking its
// {base type definition} chain from st itself, an anonymous step's zero name
// meeting none, and failing one, its primitive's in primitiveAtomics: every
// bounded builtin lies below xs:integer, and dateTimeStamp, dayTimeDuration
// and yearMonthDuration below the primitive they restrict, so the nearest
// wins; a primitive of datatypeAtomics is met only at the top of its chain,
// and no chain meets both an integer builtin and a temporal one.
//
// The error is xsd's, returned bare, or a plain one, prefixed "goshape:",
// for a list or union st, whose Go type is no Atomic, and for any other st
// with no {primitive type definition}, which st-props-correct clause 1
// (Structures §3.16.6.1) and the Datatypes §4.1.1 tableau give no simple
// type but the special ones. {variety} is a closed sum, so the switch is
// STYLE T2's sealed-sum exception.
func AtomicType(r xsd.TypeResolver, st *xsd.SimpleType) (a Atomic, ok bool, err error) {
	if st.IsSpecial() {
		return AtomicString, true, nil
	}
	variety, err := st.Variety(r)
	if err != nil {
		return 0, false, err
	}
	switch variety.(type) {
	case xsd.Union:
		return 0, false, fmt.Errorf("goshape: the simple type defined at %s is a union where an atomic type is required", st.Loc())
	case xsd.List:
		return 0, false, fmt.Errorf("goshape: the simple type defined at %s is a list where an atomic type is required", st.Loc())
	}
	primitive, err := st.Primitive(r)
	if err != nil {
		return 0, false, err
	}
	if primitive == nil {
		return 0, false, fmt.Errorf("goshape: the simple type defined at %s is of absent or atomic {variety} with no {primitive type definition}, which st-props-correct clause 1 (Structures §3.16.6.1) and the Datatypes §4.1.1 tableau give no simple type but the special types xs:anySimpleType and xs:anyAtomicType", st.Loc())
	}
	for step := st; step != nil; {
		if found, hit := chainAtomics[step.Name()]; hit {
			return found, true, nil
		}
		if found, hit := datatypeAtomics[step.Name()]; hit {
			return found, true, nil
		}
		base, err := step.Base(r)
		if err != nil {
			return 0, false, err
		}
		step = base
	}
	if found, hit := primitiveAtomics[primitive.Name()]; hit {
		return found, true, nil
	}
	return 0, false, nil
}
