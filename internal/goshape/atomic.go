package goshape

import (
	"fmt"

	"github.com/kud360/goxsd8/xsd"
)

// Atomic is the atomic Go type codegen/doc.go "# Simple types" gives an
// atomic simple type, or the string a special type gets: string, bool,
// float32, float64, big.Rat, big.Int, a bounded Go integer, []byte for
// xs:hexBinary and, apart from it, for xs:base64Binary, the datatype struct of
// a date, time or duration datatype, QName's datatype struct, which NOTATION
// shares, or PrecisionDecimal. The zero value is invalid (STYLE T1).
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
// none, st's {primitive type definition} being an ·implementation-defined·
// one other than xs:precisionDecimal (an xsd.NewPrimitiveType one, or a
// restriction of it), which each caller declines in its own terms. A special
// type, known by identity ([xsd.SimpleType.IsSpecial]) ahead of its
// {variety}, gives AtomicString. An atomic st's Atomic is that of the first
// builtin of chainAtomics or datatypeAtomics met walking its {base type
// definition} chain from st itself, an anonymous step's zero name meeting
// none, and failing one, its primitive's in primitiveAtomics: every bounded
// builtin lies below xs:integer, and dateTimeStamp, dayTimeDuration and
// yearMonthDuration below the primitive they restrict, so the nearest wins;
// a primitive of datatypeAtomics is met only at the top of its chain, and no
// chain meets both an integer builtin and a temporal one.
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
