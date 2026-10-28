package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The names of the components the slice reads by identity.
var (
	anyTypeName      = xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}
	stringPrimitive  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}
	anyURIPrimitive  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyURI"}
	booleanPrimitive = xsd.QName{Space: xsd.XMLSchemaNS, Local: "boolean"}
	floatPrimitive   = xsd.QName{Space: xsd.XMLSchemaNS, Local: "float"}
	doublePrimitive  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "double"}
	decimalPrimitive = xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"}

	hexBinaryPrimitive    = xsd.QName{Space: xsd.XMLSchemaNS, Local: "hexBinary"}
	base64BinaryPrimitive = xsd.QName{Space: xsd.XMLSchemaNS, Local: "base64Binary"}

	integerBuiltin = xsd.QName{Space: xsd.XMLSchemaNS, Local: "integer"}

	longBuiltin          = xsd.QName{Space: xsd.XMLSchemaNS, Local: "long"}
	intBuiltin           = xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}
	shortBuiltin         = xsd.QName{Space: xsd.XMLSchemaNS, Local: "short"}
	byteBuiltin          = xsd.QName{Space: xsd.XMLSchemaNS, Local: "byte"}
	unsignedLongBuiltin  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedLong"}
	unsignedIntBuiltin   = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedInt"}
	unsignedShortBuiltin = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedShort"}
	unsignedByteBuiltin  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedByte"}
)

// The math/big types "# Coverage" maps to; a field or named type of either
// makes the generated file import math/big (emitter.usesBig).
const (
	bigInt = "big.Int"
	bigRat = "big.Rat"
)

// datatypeStruct is the struct "# Coverage" maps one date, time or duration
// datatype to (Datatypes §3.3.6–§3.3.14, §3.4.26–§3.4.28): one field per
// property its values carry, in the order of the seven-property model
// (§D.2.1) or of duration's two properties (§3.3.6.1). doc is the header's
// line after its first, "" for none.
type datatypeStruct struct {
	datatype xsd.QName
	doc      string
	fields   []datatypeField
}

// datatypeField is one field of a datatypeStruct: its Go name, which is its
// property's, and its Go type.
type datatypeField struct {
	name, goType string
}

// The header lines the datatype structs share.
const (
	optionalOffsetDoc = "TimezoneOffset is in minutes from UTC, nil when absent."
	requiredOffsetDoc = "TimezoneOffset is in minutes from UTC."
)

// The fields the datatype structs share: uint8 is the narrowest Go integer
// holding month 1–12, day 1–31, hour 0–23 and minute 0–59, int16 the narrowest
// holding a timezoneOffset of −840..840 (§D.2.1), and year, second, months and
// seconds are unbounded (§D.2.1, §3.3.6.1).
var (
	yearField           = datatypeField{"Year", bigInt}
	monthField          = datatypeField{"Month", "uint8"}
	dayField            = datatypeField{"Day", "uint8"}
	hourField           = datatypeField{"Hour", "uint8"}
	minuteField         = datatypeField{"Minute", "uint8"}
	secondField         = datatypeField{"Second", bigRat}
	optionalOffsetField = datatypeField{"TimezoneOffset", "*int16"}
	requiredOffsetField = datatypeField{"TimezoneOffset", "int16"}
	monthsField         = datatypeField{"Months", bigInt}
	secondsField        = datatypeField{"Seconds", bigRat}
)

// The twelve datatype structs: one per date, time or duration primitive, and
// one per builtin derived from one whose values carry fewer properties or a
// required one (§3.4.26 yearMonthDuration, §3.4.27 dayTimeDuration, §3.4.28
// dateTimeStamp).
var (
	dateTimeStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTime"}, optionalOffsetDoc,
		[]datatypeField{yearField, monthField, dayField, hourField, minuteField, secondField, optionalOffsetField}}
	dateTimeStampStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTimeStamp"}, requiredOffsetDoc,
		[]datatypeField{yearField, monthField, dayField, hourField, minuteField, secondField, requiredOffsetField}}
	timeStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "time"}, optionalOffsetDoc,
		[]datatypeField{hourField, minuteField, secondField, optionalOffsetField}}
	dateStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "date"}, optionalOffsetDoc,
		[]datatypeField{yearField, monthField, dayField, optionalOffsetField}}
	gYearMonthStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "gYearMonth"}, optionalOffsetDoc,
		[]datatypeField{yearField, monthField, optionalOffsetField}}
	gYearStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "gYear"}, optionalOffsetDoc,
		[]datatypeField{yearField, optionalOffsetField}}
	gMonthDayStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "gMonthDay"}, optionalOffsetDoc,
		[]datatypeField{monthField, dayField, optionalOffsetField}}
	gDayStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "gDay"}, optionalOffsetDoc,
		[]datatypeField{dayField, optionalOffsetField}}
	gMonthStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "gMonth"}, optionalOffsetDoc,
		[]datatypeField{monthField, optionalOffsetField}}
	durationStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "duration"}, "Months and Seconds share one sign.",
		[]datatypeField{monthsField, secondsField}}
	dayTimeDurationStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "dayTimeDuration"}, "",
		[]datatypeField{secondsField}}
	yearMonthDurationStruct = datatypeStruct{xsd.QName{Space: xsd.XMLSchemaNS, Local: "yearMonthDuration"}, "",
		[]datatypeField{monthsField}}
)

// emitter builds one target's source. decls holds each emitted type
// declaration in emission order: a top-level element's type, then the types
// its particles reach that no earlier element emitted — anonymous complex
// types nested in its type definition or in a model group definition a
// <group ref> particle names, and named complex types of local elements —
// depth first in particle order; the type for one occurrence of a
// struct's content model group follows that struct, before the types its
// particles reach, and a sealed interface's branch structs follow it, each
// before the types its alternative reaches.
type emitter struct {
	schema *xsd.Schema
	types  *namer
	// base is the target's Dir made absolute, the directory each header
	// renders its declaration's location against (headerLoc).
	base  string
	decls []string
	// emitted holds the type emitted for each named complex type a local
	// element reaches, so that every later local element of that type shares
	// it, and for each anonymous complex type nested in a named complex type
	// or in a model group definition, and each type for one occurrence of a
	// content model group in either (occurrenceType), so that every element
	// whose walk reaches that named type, and every <group ref> particle to
	// that group, shares it; it is a lookup index only and never iterated
	// (STYLE D2). nestedType and occurrenceType store an entry before walking
	// its type's particles, so a type that reaches itself reads it, and remove
	// it when that walk fails, so that it holds only types emitted whole or
	// still being walked. A top-level element's type is named from the element
	// and is never stored, nor is a type nested in an anonymous top-level type
	// outside a model group definition (scope.shared), and a branch struct is
	// never stored: it rides with its interface.
	emitted map[nestedKey]emittedType
	// elements holds the type emitted for each top-level element declaration,
	// keyed by its expanded name and stored by elementType when the walk first
	// reaches the declaration, at its own topLevel visit or at an <element ref>
	// particle to it, so that a ref field reads its name and whether it is a
	// sealed interface; it is a lookup index only and never iterated (STYLE
	// D2).
	elements map[xsd.QName]emittedType
	// acyclic holds the chain nodes whose value fields valueCycle followed to
	// the end with the node in its chain (valueChain), so that no value cycle
	// passes through them and valueCycle never walks one again; it is a lookup
	// index only and never iterated (STYLE D2), kept for a measured hot path:
	// without it a schema whose named types each reach the next twice costs
	// double per level (#3037).
	acyclic map[valueNode]struct{}
	// usesBig records that goType produced a math/big type, or emitted a
	// datatype struct with a math/big field, so that emit imports math/big;
	// emit builds source only for a target that declines nothing, so there a
	// set flag means an emitted declaration names the package.
	usesBig bool
	// datatypes holds the name claimed for each datatype struct emitted, so
	// that each is emitted once per package; it is a lookup index only and
	// never iterated (STYLE D2). tailDecls holds those structs' declarations,
	// and the raw-content structs' (anyTypeType), in first-use order, emitted
	// after decls.
	datatypes map[xsd.QName]string
	tailDecls []string
	// raw holds the names claimed for the four raw-content structs, which
	// anyTypeType emits once per package; its zero value means they are
	// unclaimed and unemitted.
	raw rawNames
}

// rawNames is the names claimed for the four raw-content structs "# Coverage"
// gives an element of type xs:anyType (§3.4.7 any-type-itself) and a wildcard
// particle of a sequence, in claim order.
type rawNames struct {
	anyType, attribute, child, element string
}

// isAnyType reports whether td is xs:anyType itself, by identity: every other
// type definition is derived from it (§2.2.1.1 key-derived), whatever its
// shape, and so is never xs:anyType.
func isAnyType(td xsd.TypeDefinition) bool { return td.Name() == anyTypeName }

// emittedType is the Go type emitted for one complex type definition or
// top-level element declaration: its name, and whether it is a sealed
// interface, which a field holds bare rather than through a pointer.
type emittedType struct {
	name   string
	sealed bool
}

// scope is where the structs nested in one struct take their names from: the
// path of expanded names from their nearest named ancestor, or from the named
// complex type or model group definition that declares their element
// (nestedType), that ancestor's first, and the kind of component that
// ancestor is (scopeRoot). A struct is named from the path's local names and
// keyed by its root and its expanded names (nestedKey).
type scope struct {
	root scopeRoot
	path []xsd.QName
}

// scopeRoot is the kind of component a scope's path starts at: a top-level
// element declaration whose type is anonymous, a named complex type
// definition, or a model group definition. The three are separate symbol
// spaces that may share an expanded name (§3.3.2, §3.17.1). The zero value is
// invalid (STYLE T1).
type scopeRoot uint8

// The scopeRoot values.
const (
	elementRoot scopeRoot = iota + 1
	typeRoot
	groupRoot
)

// shared reports whether a type emitted under s is emitted once and shared by
// every walk that reaches s's path (e.emitted): under a named complex type,
// whichever elements have that type, and under a model group definition,
// whichever particles reference it, each element declared in the group being
// one component (§3.7.2). A top-level element's anonymous type is reached from
// that element alone.
func (s scope) shared() bool { return s.root != elementRoot }

// locals returns the local names of the path, the names the namer joins.
func (s scope) locals() []string {
	locals := make([]string, len(s.path))
	for i, name := range s.path {
		locals[i] = name.Local
	}
	return locals
}

// nestedKey identifies a named complex type by its expanded name, and an
// anonymous complex type nested in one or in a model group definition by the
// expanded names along its path: the named type's or group's, then each
// element's {target namespace} and {name} (§3.3.2.3), each Go-quoted so that
// the concatenation is unambiguous and a named type's one-name key differs
// from every nested type's, all after the path's scopeRoot in decimal, so that
// a type and a group of one expanded name never share a key; and the type for
// one occurrence of the content model group of either by that key and one
// quoted empty string more (groupKey).
type nestedKey string

// key returns the nestedKey of the struct whose path is s's.
func (s scope) key() nestedKey {
	var b strings.Builder
	b.WriteString(strconv.Itoa(int(s.root)))
	for _, name := range s.path {
		b.WriteString(strconv.Quote(name.Space))
		b.WriteString(strconv.Quote(name.Local))
	}
	return nestedKey(b.String())
}

// groupKey returns the nestedKey of the type for one occurrence of the content
// model group of the struct whose path is s's (occurrenceType): s's key
// followed by one Go-quoted empty string. A path key holds an even number of
// quoted strings, so the two never meet.
func (s scope) groupKey() nestedKey { return s.key() + nestedKey(strconv.Quote("")) }

// emit returns the gofmt-formatted source of package pkg for schema, or the
// declines and the failure that is not a decline that Generate's doc states
// for one target's walk, one turn per top-level element declaration. It
// returns source only when it collects no decline, and err carries no
// decline. base is the absolute directory the source is written into.
func emit(schema *xsd.Schema, pkg, base string) (src []byte, declines []*DeclineError, err error) {
	e := &emitter{schema: schema, types: newNamer(), base: base, emitted: map[nestedKey]emittedType{}, elements: map[xsd.QName]emittedType{}, acyclic: map[valueNode]struct{}{}, datatypes: map[xsd.QName]string{}}
	for _, decl := range schema.Elements() {
		err := e.topLevel(decl)
		if err == nil {
			continue
		}
		var de *DeclineError
		if !errors.As(err, &de) {
			return nil, declines, err
		}
		if !slices.ContainsFunc(declines, func(d *DeclineError) bool { return *d == *de }) {
			declines = append(declines, de)
		}
	}
	if len(declines) > 0 {
		return nil, declines, nil
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s\n\npackage %s\n", generatedHeader, pkg)
	if e.usesBig {
		b.WriteString("\nimport \"math/big\"\n")
	}
	for _, d := range slices.Concat(e.decls, e.tailDecls) {
		b.WriteString("\n")
		b.WriteString(d)
	}
	src, err = format.Source(b.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("formatting the emitted source: %w", err)
	}
	return src, nil, nil
}

// header is the doc comment of the emitted type name, generated from the
// declaration from renders (elementFrom, typeOrigin).
func header(name, from string) string {
	return fmt.Sprintf("// %s is generated from %s.\n", name, from)
}

// elementFrom renders an element declaration for a header: its expanded name,
// whose namespace is its {target namespace} (for a local declaration, as form
// and elementFormDefault set it, §3.3.2.3), and its Loc as headerLoc renders
// it against e.base. Only a header renders a Loc so: every decline and error
// names the Loc as the declaration carries it.
func (e *emitter) elementFrom(decl xsd.ElementDeclaration) string {
	return fmt.Sprintf("element %s declared at %s", decl.Name(), headerLoc(e.base, decl.Loc()))
}

// origin is what one complex type's emitted declarations are generated from:
// from renders it for their headers, and subject names the complex type in
// declines.
type origin struct {
	from, subject string
}

// elementOrigin is the origin of the type emitted for decl's complex {type
// definition} t and named from decl: a top-level element's type, or a local
// element's anonymous one.
func (e *emitter) elementOrigin(decl xsd.ElementDeclaration, t xsd.ComplexType) origin {
	return origin{from: e.elementFrom(decl), subject: "complex type " + typeName(t) + " of element " + decl.Name().String()}
}

// typeOrigin is the origin of the type emitted for the named complex type t
// that a local element reaches and named from t: t's expanded name and its
// Loc, rendered as elementFrom renders an element's.
func (e *emitter) typeOrigin(t xsd.ComplexType) origin {
	return origin{
		from:    fmt.Sprintf("complex type %s declared at %s", t.Name(), headerLoc(e.base, t.Loc())),
		subject: "complex type " + typeName(t),
	}
}

// headerLoc renders loc for a header against the absolute directory base, by
// the rule the package doc's "Naming" states for a target's Dir.
func headerLoc(base string, loc xsderr.Loc) string {
	if !filepath.IsAbs(loc.URI) {
		return loc.String()
	}
	rel, err := filepath.Rel(base, loc.URI)
	if err != nil {
		return loc.String()
	}
	loc.URI = filepath.ToSlash(rel)
	return loc.String()
}

// topLevel emits the type of one top-level element declaration: a named type
// over the raw-content struct AnyType ("# Coverage") for xs:anyType, over an
// atomic Go type for an atomic simple {type definition}, over a slice of one
// for a list, a struct or sealed interface for an accepted complex one, named
// or anonymous. That type is named from the element (elementType), and a local
// element of the same named type never shares it (nestedType), while every
// <element ref> particle to the element does; the types nested in it are named
// from the type when it is named, and shared with that type's every other use,
// and from the element when it is anonymous, but for those of the elements a
// model group definition declares, named from the group and shared with its
// every <group ref> particle (nestedType).
func (e *emitter) topLevel(decl xsd.ElementDeclaration) error {
	td, err := e.declaredType(decl)
	if err != nil {
		return err
	}
	typ, err := e.elementType(decl, td)
	if err != nil {
		return err
	}
	if isAnyType(td) {
		e.decls = append(e.decls, header(typ.name, e.elementFrom(decl))+fmt.Sprintf("type %s %s\n", typ.name, e.raw.anyType))
		return nil
	}
	switch t := td.(type) {
	case *xsd.SimpleType:
		underlying, err := e.simpleGoType("element "+decl.Name().String(), decl.Loc(), t)
		if err != nil {
			return err
		}
		e.decls = append(e.decls, header(typ.name, e.elementFrom(decl))+fmt.Sprintf("type %s %s\n", typ.name, underlying))
		return nil
	case xsd.ComplexType:
		nested := scope{root: elementRoot, path: []xsd.QName{decl.Name()}}
		if t.Name() != (xsd.QName{}) {
			nested = scope{root: typeRoot, path: []xsd.QName{t.Name()}}
		}
		chain, err := valueChain(nil, false, decl, t)
		if err != nil {
			return err
		}
		o := e.elementOrigin(decl, t)
		_, content, err := e.shape(typ.name, o, t)
		if err != nil {
			return err
		}
		return e.body(typ, nested, chain, o, t, content)
	}
	return fmt.Errorf("element %s at %s: unexpected type definition %T", decl.Name(), decl.Loc(), td)
}

// elementType returns the type emitted for the top-level element declaration
// decl, whose {type definition} declaredType resolved to td, claiming its name
// and checking its type when the walk first reaches decl: at its own topLevel
// visit, or at an <element ref> particle to it met earlier. For xs:anyType
// there is no check, and anyTypeType claims the raw-content structs' names
// before decl's is claimed ("# Naming"); for a simple td the check is
// simpleGoType's, run before the name is claimed so that a datatype struct
// claims its name first; for a complex one it is shape's, run after. It emits no
// declaration of its own, only the datatype or raw-content structs it first
// needs: topLevel emits the type's declaration at decl's own visit, in document
// order, whichever reached decl first.
func (e *emitter) elementType(decl xsd.ElementDeclaration, td xsd.TypeDefinition) (emittedType, error) {
	if typ, ok := e.elements[decl.Name()]; ok {
		return typ, nil
	}
	var typ emittedType
	st, isSimple := td.(*xsd.SimpleType)
	ct, isComplex := td.(xsd.ComplexType)
	switch {
	case isAnyType(td):
		e.anyTypeType()
		typ = emittedType{name: e.types.claim(decl.Name().Local)}
	case isSimple:
		if _, err := e.simpleGoType("element "+decl.Name().String(), decl.Loc(), st); err != nil {
			return emittedType{}, err
		}
		typ = emittedType{name: e.types.claim(decl.Name().Local)}
	case isComplex:
		shaped, _, err := e.shape(e.types.claim(decl.Name().Local), e.elementOrigin(decl, ct), ct)
		if err != nil {
			return emittedType{}, err
		}
		typ = shaped
	default:
		return emittedType{}, fmt.Errorf("element %s at %s: unexpected type definition %T", decl.Name(), decl.Loc(), td)
	}
	e.elements[decl.Name()] = typ
	return typ, nil
}

// declaredType checks the element declaration properties the slice declines
// on every declaration it emits from, and returns the resolved {type
// definition}.
func (e *emitter) declaredType(decl xsd.ElementDeclaration) (xsd.TypeDefinition, error) {
	subject := "element " + decl.Name().String()
	switch {
	case decl.Nillable():
		return nil, decline(decl.Loc(), constructElementNillable, subject)
	case decl.Abstract():
		return nil, decline(decl.Loc(), constructElementAbstract, subject)
	case len(decl.SubstitutionGroupAffiliationNames()) > 0:
		return nil, decline(decl.Loc(), constructSubstitutionGroup, subject)
	}
	if _, ok := decl.ValueConstraint(); ok {
		return nil, decline(decl.Loc(), constructElementValueConstraint, subject)
	}
	if _, ok := decl.TypeTable(); ok {
		return nil, decline(decl.Loc(), constructTypeTable, subject)
	}
	td, ok := e.schema.ResolvedType(decl.TypeDefinition())
	if !ok {
		return nil, fmt.Errorf("%s at %s: its {type definition} does not resolve", subject, decl.Loc())
	}
	return td, nil
}

// simpleGoType is the atomic Go type, or the slice of one, that a
// simple-typed element's or attribute's field, or an element's named type, is
// declared over, chosen by the package doc's "# Coverage" (goType). subject
// names the declaration the type is read from, and loc, its Loc, is a
// decline's.
func (e *emitter) simpleGoType(subject string, loc xsderr.Loc, t *xsd.SimpleType) (string, error) {
	return e.goType(subject, fmt.Sprintf("%s with {type definition} %s", subject, typeName(t)), loc, t)
}

// goType is the Go type "# Coverage" maps the simple type t to: for an atomic
// t, its atomic Go type, chosen through builtinType and then t's {primitive
// type definition} (Datatypes §4.1.1), setting e.usesBig for a math/big type
// or a datatype struct holding one; for a list (Datatypes §2.4.1.2, a finite
// sequence of its item type's values), a slice of the type its {item type
// definition} (§3.16.1) maps to, read through [xsd.SimpleType.Item], which
// takes a restriction's item from its base (§3.16.2.3), never by walking t to
// a builtin. typed names t in a decline, and a list's item is named in it
// after the list, so an anonymous item still has a named context. subject and
// loc are simpleGoType's.
func (e *emitter) goType(subject, typed string, loc xsderr.Loc, t *xsd.SimpleType) (string, error) {
	if t.IsSpecial() {
		return "", decline(loc, constructSpecialType, typed)
	}
	variety, err := t.Variety(e.schema)
	if err != nil {
		return "", fmt.Errorf("%s at %s: %w", subject, loc, err)
	}
	switch variety.(type) {
	case xsd.Atomic:
	case xsd.List:
		item, err := t.Item(e.schema)
		if err != nil {
			return "", fmt.Errorf("%s at %s: %w", subject, loc, err)
		}
		elem, err := e.goType(subject, typed+", whose {item type definition} is "+typeName(item), loc, item)
		if err != nil {
			return "", err
		}
		return "[]" + elem, nil
	case xsd.Union:
		return "", decline(loc, constructUnionType, typed)
	default:
		return "", decline(loc, constructAbsentVariety, typed)
	}
	mapped, err := e.builtinType(t)
	if err != nil {
		return "", fmt.Errorf("%s at %s: %w", subject, loc, err)
	}
	if mapped != "" {
		return mapped, nil
	}
	primitive, err := t.Primitive(e.schema)
	if err != nil {
		return "", fmt.Errorf("%s at %s: %w", subject, loc, err)
	}
	if primitive == nil {
		return "", decline(loc, constructNoPrimitive, typed)
	}
	switch primitive.Name() {
	case stringPrimitive, anyURIPrimitive:
		return "string", nil
	case booleanPrimitive:
		return "bool", nil
	case floatPrimitive: // Datatypes §3.3.4.1, as "# Coverage" states
		return "float32", nil
	case doublePrimitive: // Datatypes §3.3.5.1, as "# Coverage" states
		return "float64", nil
	case decimalPrimitive: // Datatypes §3.3.3, as "# Coverage" states
		return e.useBig(bigRat), nil
	case hexBinaryPrimitive, base64BinaryPrimitive: // Datatypes §3.3.15.1, §3.3.16.1, as "# Coverage" states
		return "[]byte", nil
	case dateTimeStruct.datatype: // Datatypes §3.3.7
		return e.datatypeType(dateTimeStruct), nil
	case timeStruct.datatype: // Datatypes §3.3.8
		return e.datatypeType(timeStruct), nil
	case dateStruct.datatype: // Datatypes §3.3.9
		return e.datatypeType(dateStruct), nil
	case gYearMonthStruct.datatype: // Datatypes §3.3.10
		return e.datatypeType(gYearMonthStruct), nil
	case gYearStruct.datatype: // Datatypes §3.3.11
		return e.datatypeType(gYearStruct), nil
	case gMonthDayStruct.datatype: // Datatypes §3.3.12
		return e.datatypeType(gMonthDayStruct), nil
	case gDayStruct.datatype: // Datatypes §3.3.13
		return e.datatypeType(gDayStruct), nil
	case gMonthStruct.datatype: // Datatypes §3.3.14
		return e.datatypeType(gMonthStruct), nil
	case durationStruct.datatype: // Datatypes §3.3.6
		return e.datatypeType(durationStruct), nil
	}
	return "", decline(loc, constructUncoveredPrimitive, typed, primitive.Name())
}

// useBig returns the math/big type typ, setting e.usesBig.
func (e *emitter) useBig(typ string) string {
	e.usesBig = true
	return typ
}

// datatypeType returns the name of the struct ds, emitting it when no earlier
// declaration did: its name is claimed in e.types at that first use, by the
// package doc's "Naming", and its declaration joins e.tailDecls. Emitting a
// struct with a math/big field sets e.usesBig.
func (e *emitter) datatypeType(ds datatypeStruct) string {
	if name, ok := e.datatypes[ds.datatype]; ok {
		return name
	}
	name := e.types.claim(ds.datatype.Local)
	e.datatypes[ds.datatype] = name
	var b strings.Builder
	fmt.Fprintf(&b, "// %s is generated from datatype %s.\n", name, ds.datatype)
	if ds.doc != "" {
		fmt.Fprintf(&b, "// %s\n", ds.doc)
	}
	fmt.Fprintf(&b, "type %s struct {\n", name)
	for _, f := range ds.fields {
		if f.goType == bigInt || f.goType == bigRat {
			e.usesBig = true
		}
		fmt.Fprintf(&b, "\t%s %s\n", f.name, f.goType)
	}
	b.WriteString("}\n")
	e.tailDecls = append(e.tailDecls, b.String())
	return name
}

// anyTypeType returns the name of the raw-content struct AnyType, which holds
// the content of an element of type xs:anyType: mixed, a lax ##any wildcard
// any number of times, and a lax ##any {attribute wildcard} (§3.4.7
// any-type-itself). When no earlier declaration or wildcard particle needed
// them, it claims the four raw-content structs' names in e.types, in rawNames'
// order, by the package doc's "Naming", and appends their declarations to
// e.tailDecls in that order. No field names math/big.
func (e *emitter) anyTypeType() string {
	if e.raw.anyType != "" {
		return e.raw.anyType
	}
	n := rawNames{
		anyType:   e.types.claim("anyType"),
		attribute: e.types.claim("anyAttribute"),
		child:     e.types.claim("anyChild"),
		element:   e.types.claim("anyElement"),
	}
	e.raw = n
	from := "complex type " + anyTypeName.String()
	e.tailDecls = append(e.tailDecls,
		header(n.anyType, from)+
			"// Text is the characters before the first of Children, \"\" for none.\n"+
			fmt.Sprintf("type %s struct {\n\tAttributes []%s\n\tText string\n\tChildren []%s\n}\n", n.anyType, n.attribute, n.child),
		header(n.attribute, from)+
			fmt.Sprintf("type %s struct {\n\tSpace, Local, Value string\n}\n", n.attribute),
		header(n.child, from)+
			"// Tail is the characters after Element, before the next child or the end.\n"+
			fmt.Sprintf("type %s struct {\n\tElement %s\n\tTail string\n}\n", n.child, n.element),
		header(n.element, from)+
			fmt.Sprintf("type %s struct {\n\tSpace, Local string\n\tContent %s\n}\n", n.element, n.anyType))
	return n.anyType
}

// builtinType is the Go type "# Coverage" maps t to through the eight bounded
// builtin integer types (Datatypes §3.4.16–§3.4.19, §3.4.21–§3.4.24),
// xs:integer (§3.4.13), and the three builtins derived from a date, time or
// duration primitive that get a datatype struct of their own (§3.4.26–§3.4.28),
// walking t's {base type definition} chain (Structures §3.16.1) from t itself
// and stopping at the first of the twelve it meets, or "" when the walk ends
// at xs:anySimpleType's absent base without meeting one. Every bounded type
// lies below xs:integer on its chain, so the nearest bounded type wins over
// it; no chain meets both an integer builtin and a temporal one. Returning
// big.Int, or a datatype struct with a math/big field, sets e.usesBig.
func (e *emitter) builtinType(t *xsd.SimpleType) (string, error) {
	for step := t; step != nil; {
		switch step.Name() {
		case integerBuiltin:
			return e.useBig(bigInt), nil
		case dateTimeStampStruct.datatype: // Datatypes §3.4.28
			return e.datatypeType(dateTimeStampStruct), nil
		case dayTimeDurationStruct.datatype: // Datatypes §3.4.27
			return e.datatypeType(dayTimeDurationStruct), nil
		case yearMonthDurationStruct.datatype: // Datatypes §3.4.26
			return e.datatypeType(yearMonthDurationStruct), nil
		case longBuiltin:
			return "int64", nil
		case intBuiltin:
			return "int32", nil
		case shortBuiltin:
			return "int16", nil
		case byteBuiltin:
			return "int8", nil
		case unsignedLongBuiltin:
			return "uint64", nil
		case unsignedIntBuiltin:
			return "uint32", nil
		case unsignedShortBuiltin:
			return "uint16", nil
		case unsignedByteBuiltin:
			return "uint8", nil
		}
		base, err := step.Base(e.schema)
		if err != nil {
			return "", err
		}
		step = base
	}
	return "", nil
}

// typeName names a type definition in a decline message.
func typeName(t xsd.TypeDefinition) string {
	if t.Name() == (xsd.QName{}) {
		return "(anonymous)"
	}
	return t.Name().String()
}

// shape checks the complex type t against the package doc's "# Coverage"
// shape, naming it in declines by o's subject, and returns the type it emits
// as name — a sealed interface for a choice content particle that occurs
// exactly once, which declines {attribute uses}, and a struct otherwise — and
// its content particle (contentGroup), nil for empty content. It emits
// nothing: body does, so that a caller can store the type in e.emitted before
// its body is walked.
func (e *emitter) shape(name string, o origin, t xsd.ComplexType) (emittedType, *contentParticle, error) {
	content, err := e.contentGroup(o.subject, t)
	if err != nil {
		return emittedType{}, nil, err
	}
	if content == nil || content.group.Compositor() != xsd.CompositorChoice || content.occurs != exactlyOnce {
		return emittedType{name: name}, content, nil
	}
	if len(t.AttributeUses()) > 0 {
		return emittedType{}, nil, decline(t.Loc(), constructChoiceAttributeUses, o.subject)
	}
	return emittedType{name: name, sealed: true}, content, nil
}

// body emits typ, which shape returned for the complex type t with content,
// generated from o: a struct holding particleFields' fields, or for content
// occurring other than exactly once the one field groupField returns, or for
// simple content (§3.4.2.2 dcl.ctd.ctsc) the value field, named from the local
// name value and holding, never through a pointer, what goType maps its
// {simple type definition} to, and then one per attribute use; or the sealed
// interface sealedType emits; then the types its fields reach, the anonymous
// ones named in nested. chain is valueChain's for a walk of t's particles.
func (e *emitter) body(typ emittedType, nested scope, chain []valueNode, o origin, t xsd.ComplexType, content *contentParticle) error {
	if typ.sealed {
		return e.sealedType(typ.name, nested, o, t, content.group.Particles())
	}
	slot := len(e.decls)
	e.decls = append(e.decls, "")
	var fields []fieldSpec
	if simple, ok := t.ContentType().(xsd.SimpleContent); ok {
		goType, err := e.goType(o.subject, fmt.Sprintf("%s with {simple type definition} %s", o.subject, typeName(simple.SimpleType)), t.Loc(), simple.SimpleType)
		if err != nil {
			return err
		}
		fields = append(fields, fieldSpec{local: "value", goType: goType})
	}
	if content != nil && content.occurs == exactlyOnce {
		particleFields, err := e.particleFields(nested, chain, o.subject, t.Loc(), content.group.Particles())
		if err != nil {
			return err
		}
		fields = particleFields
	}
	if content != nil && content.occurs != exactlyOnce {
		field, err := e.groupField(nested, o, t, *content)
		if err != nil {
			return err
		}
		fields = append(fields, field)
	}
	for _, u := range t.AttributeUses() {
		field, err := e.attributeField(o.subject, t.Loc(), u)
		if err != nil {
			return err
		}
		fields = append(fields, field)
	}
	e.decls[slot] = header(typ.name, o.from) + structDecl(typ.name, fields)
	return nil
}

// groupField returns the field of the model-group particle c of the complex
// type t generated from o, in a struct whose nested types are named in nested:
// named from c's compositor, "sequence" or "choice", it holds the type for one
// occurrence of c's group (occurrenceType) by c's occurrence (occursGoType).
// Its one caller is body, for a content particle occurring other than exactly
// once; a nested model group particle, which field declines, is the same
// particle shape.
func (e *emitter) groupField(nested scope, o origin, t xsd.ComplexType, c contentParticle) (fieldSpec, error) {
	typ, err := e.occurrenceType(nested, o, t, c.group)
	if err != nil {
		return fieldSpec{}, err
	}
	goType, ok := occursGoType(typ, c.occurs)
	if !ok {
		return fieldSpec{}, fmt.Errorf("%s at %s: a %s model group particle with {max occurs} 0 reached groupField", o.subject, t.Loc(), c.group.Compositor())
	}
	return fieldSpec{local: c.group.Compositor().String(), goType: goType}, nil
}

// occurrenceType returns the type for one occurrence of the model group group
// of the complex type t generated from o, in a struct whose nested types are
// named in nested: a struct of a field per particle, with no attribute field,
// for a sequence, and the sealed interface sealedType emits for a choice. It
// is named by nested's path with the compositor last, generated from the
// declaration that path is named from — t when t is named (typeOrigin), and
// o's element otherwise — and emitted after the struct holding its field; its
// particles take their nested types' names from nested as if it were absent.
// Under a named complex type or a model group definition (scope.shared) it is
// stored in e.emitted under nested.groupKey before its particles are walked,
// shared by every later walk of nested's path, and removed when that walk
// fails, as nestedType stores a type.
func (e *emitter) occurrenceType(nested scope, o origin, t xsd.ComplexType, group xsd.ModelGroup) (emittedType, error) {
	key := nested.groupKey()
	if typ, ok := e.emitted[key]; ok && nested.shared() {
		return typ, nil
	}
	typ := emittedType{
		name:   e.types.claim(append(nested.locals(), group.Compositor().String())...),
		sealed: group.Compositor() == xsd.CompositorChoice,
	}
	if nested.shared() {
		e.emitted[key] = typ
	}
	if t.Name() != (xsd.QName{}) {
		o.from = e.typeOrigin(t).from
	}
	if err := e.occurrence(typ, nested, o, t, group.Particles()); err != nil {
		if nested.shared() {
			delete(e.emitted, key)
		}
		return emittedType{}, err
	}
	return typ, nil
}

// occurrence emits typ, the type occurrenceType names for one occurrence of
// the model group of particles in the complex type t generated from o: the
// sealed interface sealedType emits, or a struct of particleFields' fields.
// The particles' walk starts with an empty chain (valueChain): the occurrence
// lies behind a slice, a pointer or an interface, so no value field passes
// through it.
func (e *emitter) occurrence(typ emittedType, nested scope, o origin, t xsd.ComplexType, particles []xsd.Particle) error {
	if typ.sealed {
		return e.sealedType(typ.name, nested, o, t, particles)
	}
	slot := len(e.decls)
	e.decls = append(e.decls, "")
	fields, err := e.particleFields(nested, nil, o.subject, t.Loc(), particles)
	if err != nil {
		return err
	}
	e.decls[slot] = header(typ.name, o.from) + structDecl(typ.name, fields)
	return nil
}

// particleFields returns the fields of particles, one per particle spliced
// returns for them, in that order, in a struct whose nested types are named in
// nested and whose particles chain is valueChain's for, for the complex type
// at loc named in declines by subject (field).
func (e *emitter) particleFields(nested scope, chain []valueNode, subject string, loc xsderr.Loc, particles []xsd.Particle) ([]fieldSpec, error) {
	leaves, err := e.spliced(subject, loc, particles)
	if err != nil {
		return nil, err
	}
	fields := make([]fieldSpec, 0, len(leaves))
	for _, p := range leaves {
		field, err := e.field(nested, chain, subject, loc, p)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

// spliced returns particles in order with each particle whose term is a
// sequence model group (modelGroupTerm), inline or the {model group} of the
// definition a <group ref> names (§3.7.2), and which occurs exactly once
// replaced, recursively, by that group's own particles: one occurrence of such
// a particle is one pass through its particles in order (§3.8.4.1.1
// sec-sequences), so its fields are theirs. A complex-content extension's
// content particle (§3.4.2.3.3 dcl.ctd.ctcc.common clause 4.2.3.3) is one such
// sequence, nesting its base's when the base is itself an extension, so the
// splice recurses; the package doc's "# Coverage" states the fields that
// gives. Every other particle is kept as is, and field declines a choice and
// a sequence occurring other than exactly once, inline or referenced alike.
// subject and loc name the complex type holding particles in the error
// modelGroupTerm returns.
func (e *emitter) spliced(subject string, loc xsderr.Loc, particles []xsd.Particle) ([]xsd.Particle, error) {
	var leaves []xsd.Particle
	for _, p := range particles {
		group, isGroup, err := e.modelGroupTerm(subject, loc, p.Term())
		if err != nil {
			return nil, err
		}
		if !isGroup || group.Compositor() != xsd.CompositorSequence || p.Occurs() != exactlyOnce {
			leaves = append(leaves, p)
			continue
		}
		inner, err := e.spliced(subject, loc, group.Particles())
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, inner...)
	}
	return leaves, nil
}

// modelGroupTerm returns the model group a particle's term holds and true: an
// inline <sequence>, <choice> or <all>, or the {model group} of the top-level
// model group definition a <group ref> names (§3.7.2 declare-namedModelGroup),
// followed through [xsd.Schema.ModelGroup], so that every site reads the two
// alike. It returns false for an element declaration, an <element ref> or a
// wildcard. A <group ref> that does not resolve is an error naming subject and
// loc, the complex type holding the particle, and never a decline: finalize
// guarantees it resolves (src-resolve clause 1.5). The term is a closed sum,
// so the switch is STYLE T2's sealed-sum exception, as elementTerm's is.
func (e *emitter) modelGroupTerm(subject string, loc xsderr.Loc, term xsd.TermOrRef) (xsd.ModelGroup, bool, error) {
	switch term := term.(type) {
	case xsd.ModelGroupRef:
		def, ok := e.schema.ModelGroup(term.Name)
		if !ok {
			return xsd.ModelGroup{}, false, fmt.Errorf("%s at %s: its <group ref> particle to %s does not resolve", subject, loc, term.Name)
		}
		return def.ModelGroup(), true, nil
	case xsd.ResolvedTerm:
		group, isGroup := term.Term.(xsd.ModelGroup)
		return group, isGroup, nil
	}
	return xsd.ModelGroup{}, false, nil
}

// structDecl renders the declaration of the struct name holding fields in
// order, their names claimed in a namer of its own, and spelled struct{} when
// there is no field.
func structDecl(name string, fields []fieldSpec) string {
	if len(fields) == 0 {
		return fmt.Sprintf("type %s struct{}\n", name)
	}
	names := newNamer()
	var b strings.Builder
	for _, f := range fields {
		fmt.Fprintf(&b, "\t%s %s\n", names.claim(f.local), f.goType)
	}
	return fmt.Sprintf("type %s struct {\n%s}\n", name, b.String())
}

// sealedType emits the sealed interface name for the complex type t generated
// from o, whose content particle's term is a choice of particles (§3.8.4.1.2
// sec-choices: one alternative per pass), in the package doc's "# Coverage"
// shape: the interface, then each branch struct in particle order, followed
// by the types its particle reaches, the anonymous ones named in nested.
//
// Whatever the content particle's own {min occurs}, the gate reads the
// alternatives alone: a particle of this choice with {min occurs} 1 has as the
// minimum of its effective total range (§3.8.6.6 cos-choice-range) the least
// {min occurs} or range minimum among the alternatives, or 0 when there are no
// {particles}, and is emptiable (§3.9.6.3 cos-group-emptiable) exactly when
// that minimum is 0, so one occurrence the interface holds is never empty and
// absence is only a nil interface or an empty slice. An alternative with
// {min occurs} 0 makes it 0 whatever the others are; any alternative but an
// element declaration, local or the top-level one an <element ref> particle
// names (elementTerm), is declined in particle order, so when no decline fires
// every alternative is an element particle with {min occurs} at least 1.
func (e *emitter) sealedType(name string, nested scope, o origin, t xsd.ComplexType, particles []xsd.Particle) error {
	if len(particles) == 0 {
		return decline(t.Loc(), constructEmptyChoice, o.subject)
	}
	for _, p := range particles {
		alt, err := e.elementTerm(o.subject, t.Loc(), p.Term())
		if err != nil {
			return err
		}
		if p.Occurs().Min() == 0 {
			return decline(t.Loc(), constructEmptiableChoice, o.subject, alt.Name())
		}
	}
	marker := "is" + name
	e.decls = append(e.decls, header(name, o.from)+fmt.Sprintf("type %s interface {\n\t%s()\n}\n", name, marker))
	for _, p := range particles {
		alt, err := e.elementTerm(o.subject, t.Loc(), p.Term())
		if err != nil {
			return err
		}
		branch := e.types.claim(name, alt.Name().Local)
		slot := len(e.decls)
		e.decls = append(e.decls, "")
		field, err := e.field(nested, nil, o.subject, t.Loc(), p)
		if err != nil {
			return err
		}
		e.decls[slot] = header(branch, e.elementFrom(alt)) + fmt.Sprintf("type %s struct {\n\t%s %s\n}\n\nfunc (%s) %s() {}\n",
			branch, newNamer().claim(field.local), field.goType, branch, marker)
	}
	return nil
}

// exactlyOnce is the {min occurs} = {max occurs} = 1 range under which a
// complex type's content model group is emitted as the type itself rather
// than through an occurrence field, and under which a particle gives a value
// field.
var exactlyOnce = mustOccurs(xsd.NewOccurs(xsderr.Loc{}, 1, 1))

// mustOccurs unwraps a constructor result that cannot fail on constant
// arguments.
func mustOccurs(o xsd.Occurs, err error) xsd.Occurs {
	if err != nil {
		panic(err)
	}
	return o
}

// contentParticle is a complex type's content {particle} as contentGroup
// accepted it (§3.4.2.3.3 dcl.ctd.ctcc.common): group, the sequence or choice
// model group that is its term, inline or the {model group} a <group ref>
// names (§3.7.2), never the source <choice> itself, and occurs,
// its {min occurs} and {max occurs}, {max occurs} never 0.
type contentParticle struct {
	group  xsd.ModelGroup
	occurs xsd.Occurs
}

// contentGroup checks the complex type t, named in declines by subject,
// against the package doc's "# Coverage" complex-type shape. It reads t's
// finalized component alone, never its base, so a base that itself declines
// does not block t, and an extension is checked as a restriction is: its
// {content type} already holds what it takes from its base (§3.4.2.3.3
// dcl.ctd.ctcc.common clause 4.2, §3.4.2.2 dcl.ctd.ctsc). For element-only content
// it returns t's content particle, whatever its {min occurs} and {max occurs}
// but a {max occurs} of 0, which declines, its group the particle's own model
// group or, for a <group ref>, the referenced definition's {model group}
// (modelGroupTerm), read before the all and {max occurs} tests, so that a
// referenced group is checked and emitted as an inline one with the ref's
// {min occurs} and {max occurs} (§3.7.2); for {variety} empty (§3.4.1),
// which has no {particle}, and for {variety} simple, whose value field body
// gives, it returns nil and a nil error. Mixed content declines. Particles
// and model groups carry no Loc, so their declines cite t's.
func (e *emitter) contentGroup(subject string, t xsd.ComplexType) (*contentParticle, error) {
	switch {
	case t.Abstract():
		return nil, decline(t.Loc(), constructComplexTypeAbstract, subject)
	case len(t.Assertions()) > 0:
		return nil, decline(t.Loc(), constructAssertions, subject)
	}
	if _, has := t.AttributeWildcard(); has {
		return nil, decline(t.Loc(), constructAttributeWildcard, subject)
	}
	content, isElement := t.ContentType().(xsd.ElementContent)
	if !isElement {
		switch variety := t.ContentType().Variety(); variety {
		case xsd.ContentEmpty, xsd.ContentSimple:
			return nil, nil
		default:
			return nil, fmt.Errorf("%s at %s: unexpected content type %T of {variety} %s", subject, t.Loc(), t.ContentType(), variety)
		}
	}
	if content.Mixed {
		return nil, decline(t.Loc(), constructMixedContent, subject)
	}
	if content.OpenContent != nil {
		return nil, decline(t.Loc(), constructOpenContent, subject)
	}
	group, isGroup, err := e.modelGroupTerm(subject, t.Loc(), content.Particle.Term())
	if err != nil {
		return nil, err
	}
	if !isGroup {
		return nil, fmt.Errorf("%s at %s: unexpected content particle term %T", subject, t.Loc(), content.Particle.Term())
	}
	if group.Compositor() == xsd.CompositorAll {
		return nil, decline(t.Loc(), constructAll, subject)
	}
	occurs := content.Particle.Occurs()
	if upper, bounded := occurs.Max(); bounded && upper == 0 {
		return nil, decline(t.Loc(), constructVacuousParticle, fmt.Sprintf("a %s model group", group.Compositor()), subject)
	}
	return &contentParticle{group: group, occurs: occurs}, nil
}

// fieldSpec is one struct field before its name is claimed.
type fieldSpec struct {
	local  string
	goType string
}

// field returns the field of particle p in a struct whose nested types are
// named in nested and whose particles chain is valueChain's for, for the
// complex type at loc named in declines by subject, typed by the package doc's
// "# Coverage" field rules. The field of a wildcard particle (wildcardTerm) is
// named from the local name any and holds the raw-content struct AnyElement
// (anyTypeType) by p's occurrence (occursGoType), whatever the wildcard's
// {namespace constraint} and {process contents}, which it never reads; that
// path never calls shape, contentGroup, nestedType or valueChain, and is
// reached only through particleFields, since sealedType declines a wildcard
// alternative in elementTerm first. The field of an <element ref> particle,
// whose declaration is global, holds the type elementType returns for the
// referenced declaration, after declaredType has checked that declaration as
// it checks every one; the walk does not enter that type's body, which
// topLevel emits, but valueCycle checks it when the particle occurs exactly
// once and the type is a complex type other than xs:anyType emitted as a
// struct; valueCycle ends at once for a type whose content particle occurs
// other than exactly once. A local declaration of type xs:anyType holds the
// raw-content struct AnyType (anyTypeType). Neither a global declaration nor
// xs:anyType ever reaches nestedType.
func (e *emitter) field(nested scope, chain []valueNode, subject string, loc xsderr.Loc, p xsd.Particle) (fieldSpec, error) {
	if wildcardTerm(p.Term()) {
		e.anyTypeType()
		goType, ok := occursGoType(emittedType{name: e.raw.element}, p.Occurs())
		if !ok {
			return fieldSpec{}, decline(loc, constructVacuousParticle, "a wildcard", subject)
		}
		return fieldSpec{local: "any", goType: goType}, nil
	}
	decl, err := e.elementTerm(subject, loc, p.Term())
	if err != nil {
		return fieldSpec{}, err
	}
	td, err := e.declaredType(decl)
	if err != nil {
		return fieldSpec{}, err
	}
	value := p.Occurs() == exactlyOnce
	st, isSimple := td.(*xsd.SimpleType)
	ct, isComplex := td.(xsd.ComplexType)
	var typ emittedType
	switch {
	case decl.ScopeVariety() == xsd.ScopeGlobal:
		typ, err = e.refType(chain, value, decl, td)
	case isAnyType(td):
		typ.name = e.anyTypeType()
	case isSimple:
		typ.name, err = e.simpleGoType("element "+decl.Name().String(), decl.Loc(), st)
	case isComplex:
		typ, err = e.nestedType(nested, chain, value, decl, ct)
	default:
		err = fmt.Errorf("element %s at %s: unexpected type definition %T", decl.Name(), decl.Loc(), td)
	}
	if err != nil {
		return fieldSpec{}, err
	}
	goType, ok := occursGoType(typ, p.Occurs())
	if !ok {
		return fieldSpec{}, decline(loc, constructVacuousParticle, "element "+decl.Name().String(), subject)
	}
	return fieldSpec{local: decl.Name().Local, goType: goType}, nil
}

// occursGoType is the Go type of a field holding typ under occurs, by the
// package doc's "# Coverage" pointer and slice rules: a slice for {max occurs}
// above 1 or unbounded, a pointer for {min occurs} 0 with {max occurs} 1
// unless typ is a sealed interface, and typ itself otherwise. ok is false, and
// goType "", exactly when {max occurs} is 0; the caller declines that particle
// naming its own term.
func occursGoType(typ emittedType, occurs xsd.Occurs) (goType string, ok bool) {
	upper, bounded := occurs.Max()
	switch {
	case !bounded || upper > 1:
		return "[]" + typ.name, true
	case upper == 0:
		return "", false
	case upper == 1 && occurs.Min() == 0 && !typ.sealed:
		return "*" + typ.name, true
	}
	return typ.name, true
}

// wildcardTerm reports whether a particle's term is a wildcard (§3.10.1): an
// xsd.ResolvedTerm whose Term is an xsd.Wildcard. The particle term is a
// closed sum, so this assertion is STYLE T2's sealed-sum exception, as
// elementTerm's switch is.
func wildcardTerm(term xsd.TermOrRef) bool {
	resolved, ok := term.(xsd.ResolvedTerm)
	if !ok {
		return false
	}
	_, ok = resolved.Term.(xsd.Wildcard)
	return ok
}

// refType returns the type of the field of an <element ref> particle to the
// top-level element declaration decl, whose {type definition} declaredType
// resolved to td: elementType's. When value is true, the particle occurring
// exactly once, and that type is a struct emitted for a complex td other than
// xs:anyType, valueCycle walks td's body from the particles whose chain is
// chain, as it does whether or not decl's type was emitted earlier. A td of
// xs:anyType gets no walk: AnyType reaches itself only through slices.
func (e *emitter) refType(chain []valueNode, value bool, decl xsd.ElementDeclaration, td xsd.TypeDefinition) (emittedType, error) {
	typ, err := e.elementType(decl, td)
	if err != nil {
		return emittedType{}, err
	}
	ct, isComplex := td.(xsd.ComplexType)
	if isAnyType(td) || !value || !isComplex || typ.sealed {
		return typ, nil
	}
	return typ, e.valueCycle(chain, decl, e.elementOrigin(decl, ct).subject, ct)
}

// attributeField returns the field of attribute use u (§3.5.1) of the complex
// type at loc named in declines by subject, typed and declined by the package
// doc's "# Coverage" attribute-use rule. It reads the effective value
// constraint through Schema.EffectiveValueConstraint, never u.ValueConstraint,
// which holds the use's own alone. An attribute use carries no Loc, so its
// declines cite the declaration it resolves to, the global one for an
// <attribute ref>.
func (e *emitter) attributeField(subject string, loc xsderr.Loc, u xsd.AttributeUse) (fieldSpec, error) {
	name := u.DeclarationName()
	attr := "attribute " + name.String() + " of " + subject
	decl, ok := e.schema.ResolvedAttributeDeclaration(u)
	if !ok {
		return fieldSpec{}, fmt.Errorf("%s at %s: its {attribute declaration} does not resolve", attr, loc)
	}
	if vc, ok := e.schema.EffectiveValueConstraint(u); ok {
		return fieldSpec{}, decline(decl.Loc(), constructAttributeValueConstraint, attr, vc.Kind())
	}
	td, ok := e.schema.ResolvedType(decl.TypeDefinition())
	if !ok {
		return fieldSpec{}, fmt.Errorf("%s at %s: its {type definition} does not resolve", attr, decl.Loc())
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		return fieldSpec{}, fmt.Errorf("%s at %s: unexpected type definition %T", attr, decl.Loc(), td)
	}
	goType, err := e.simpleGoType(attr, decl.Loc(), st)
	if err != nil {
		return fieldSpec{}, err
	}
	if !u.Required() {
		goType = "*" + goType
	}
	return fieldSpec{local: name.Local, goType: goType}, nil
}

// nestedType returns the type emitted for a local element declaration's complex
// type t, whose particle occurs exactly once when value is true and whose
// parent's particles chain is valueChain's for. A named t gets one type, named
// from t and generated from it, whose nested types take their names from the
// path at t, as under a top-level element of type t (topLevel). An anonymous t
// gets a type named by the path from the named complex type or the model group
// definition that declares decl ({scope}.{parent}, §3.3.2.3 dcl.elt.local)
// when decl has one, so that every type extending that complex type, and every
// particle referencing that group, shares it, and otherwise from its nearest
// named ancestor in parent. The {parent} is a closed sum, so the switch is
// STYLE T2's sealed-sum exception, as wildcardTerm's assertion is. Each
// named type, and each anonymous type under one or under a model group
// definition, is emitted once per nestedKey, by the first element whose walk
// reaches it, and later ones share it; its entry is stored before its body is
// walked, and removed when that body fails, so that a later turn's walk, which
// emit runs after a decline, enters the type again rather than sharing a type
// never emitted, and so that a walk reaching the type again inside its own
// body, through a <group ref> to the group that declares decl, ends there. A
// shared type reached by a value field is walked again by valueCycle, which the
// emitting walk does not do. The emitted lookup, the store and the removal run
// only under a shared scope (scope.shared): a key whose path starts at a
// top-level element is reached from that element alone.
func (e *emitter) nestedType(parent scope, chain []valueNode, value bool, decl xsd.ElementDeclaration, t xsd.ComplexType) (emittedType, error) {
	child := scope{root: parent.root, path: append(slices.Clone(parent.path), decl.Name())}
	if sp, ok := decl.Scope().Parent(); ok {
		switch owner := sp.(type) {
		case xsd.ComplexTypeScopeParent:
			child = scope{root: typeRoot, path: []xsd.QName{owner.Name, decl.Name()}}
		case xsd.ModelGroupScopeParent:
			child = scope{root: groupRoot, path: []xsd.QName{owner.Name, decl.Name()}}
		}
	}
	locals := child.locals()
	if t.Name() != (xsd.QName{}) {
		child = scope{root: typeRoot, path: []xsd.QName{t.Name()}}
		locals = []string{t.Name().Local}
	}
	o := e.localOrigin(decl, t)
	key := child.key()
	if typ, ok := e.emitted[key]; ok && child.shared() {
		if !value || typ.sealed {
			return typ, nil
		}
		return typ, e.valueCycle(chain, decl, o.subject, t)
	}
	typ, content, err := e.shape(e.types.claim(locals...), o, t)
	if err != nil {
		return emittedType{}, err
	}
	next, err := valueChain(chain, value && !typ.sealed, decl, t)
	if err != nil {
		return emittedType{}, err
	}
	if child.shared() {
		e.emitted[key] = typ
	}
	if err := e.body(typ, child, next, o, t, content); err != nil {
		if child.shared() {
			delete(e.emitted, key)
		}
		return emittedType{}, err
	}
	return typ, nil
}

// localOrigin is the origin of the type emitted for the local element decl's
// complex type t: typeOrigin's for a named t, elementOrigin's for an anonymous
// one.
func (e *emitter) localOrigin(decl xsd.ElementDeclaration, t xsd.ComplexType) origin {
	if t.Name() != (xsd.QName{}) {
		return e.typeOrigin(t)
	}
	return e.elementOrigin(decl, t)
}

// symbolSpace is the symbol space a valueNode's name is in: top-level element
// declarations, type definitions, or the local element declarations of model
// group definitions, which may share an expanded name (§3.3.2, §3.17.1). The
// zero value is invalid (STYLE T1).
type symbolSpace uint8

// The symbolSpace values.
const (
	elementSpace symbolSpace = iota + 1
	typeSpace
	groupElementSpace
)

// valueNode is one node of a chain (valueChain): a top-level element
// declaration or a named complex type definition, by its expanded name, or
// the anonymous complex type of a local element declaration that a model
// group definition declares, by the group's expanded name and the element's,
// the path nestedType keys that type's struct by. Build one with elementNode,
// typeNode or groupElementNode; element is the zero QName but in the last.
type valueNode struct {
	space   symbolSpace
	name    xsd.QName
	element xsd.QName
}

// elementNode returns the valueNode of the top-level element declaration name.
func elementNode(name xsd.QName) valueNode { return valueNode{space: elementSpace, name: name} }

// typeNode returns the valueNode of the named complex type definition name.
func typeNode(name xsd.QName) valueNode { return valueNode{space: typeSpace, name: name} }

// groupElementNode returns the valueNode of the anonymous complex type of the
// local element declaration element that the model group definition group
// declares.
func groupElementNode(group, element xsd.QName) valueNode {
	return valueNode{space: groupElementSpace, name: group, element: element}
}

// String names the node in a struct-value recursion decline.
func (n valueNode) String() string {
	switch n.space {
	case elementSpace:
		return "element " + n.name.String()
	case typeSpace:
		return "complex type " + n.name.String()
	case groupElementSpace:
		return "element " + n.element.String() + " of model group " + n.name.String()
	}
	return fmt.Sprintf("valueNode(%d)", n.space)
}

// valueNodes returns the nodes a walk's step through the element declaration
// decl to its complex type t pushes, in order: a global decl, a top-level
// element or the declaration an <element ref> particle names, pushes its
// element node, and a named t its type node; a local decl of an anonymous t
// pushes its groupElementNode when a model group definition declares decl
// ({scope}.{parent} a xsd.ModelGroupScopeParent, §3.3.2.3 dcl.elt.local),
// and none when a complex type declares it. That node bounds a cycle that
// reaches decl again through a <group ref> to the group declaring it, which
// mg-props-correct clause 2 leaves legal: it bounds group containment alone,
// not the path from a group through an element's type back to it. It names
// decl as well as the group, since a step through a sibling of decl in the
// same group reaches no cycle by that alone. The {parent} is a closed sum, so
// the assertion is STYLE T2's sealed-sum exception, as nestedType's switch is.
func valueNodes(decl xsd.ElementDeclaration, t xsd.ComplexType) []valueNode {
	var nodes []valueNode
	if decl.ScopeVariety() == xsd.ScopeGlobal {
		nodes = append(nodes, elementNode(decl.Name()))
	}
	if t.Name() != (xsd.QName{}) {
		return append(nodes, typeNode(t.Name()))
	}
	if sp, ok := decl.Scope().Parent(); ok {
		if owner, inGroup := sp.(xsd.ModelGroupScopeParent); inGroup {
			nodes = append(nodes, groupElementNode(owner.Name, decl.Name()))
		}
	}
	return nodes
}

// valueChain returns the chain for a walk of the particles of the complex
// type t, reached through the element declaration decl from particles whose
// chain is chain, or declines a struct-value recursion. A chain is the nodes
// (valueNodes) a walk has pushed through value fields only since its last
// other field, in path order: a value field is one "# Coverage" types by the
// bare struct, which needs a particle that occurs exactly once (value) and a
// t that is not a choice. A value field appends the step's nodes, and
// declines on one already in chain, since the emitted Go types would hold one
// another as struct values, which Go rejects; it keeps chain when the step
// pushes none. Any other field starts afresh, holding the step's nodes alone.
// A top-level element's walk starts as a step that is not a value field, and
// each branch of a sealed interface and the particles of a content model
// group occurring other than exactly once start empty. The decline names decl and
// the repeated node, and cites decl's Loc: for an <element ref> particle,
// which carries none, the Loc of the top-level declaration it names.
func valueChain(chain []valueNode, value bool, decl xsd.ElementDeclaration, t xsd.ComplexType) ([]valueNode, error) {
	nodes := valueNodes(decl, t)
	if !value {
		return nodes, nil
	}
	for _, n := range nodes {
		if slices.Contains(chain, n) {
			return nil, decline(decl.Loc(), constructValueCycle, reacher(decl), n)
		}
	}
	return slices.Concat(chain, nodes), nil
}

// reacher names the element declaration decl, reached by a value field, in a
// struct-value recursion decline: a global decl is reached so only through an
// <element ref> particle, since a top-level element's own step is not a value
// field.
func reacher(decl xsd.ElementDeclaration) string {
	if decl.ScopeVariety() == xsd.ScopeGlobal {
		return "<element ref> particle to element " + decl.Name().String()
	}
	return "local element " + decl.Name().String()
}

// valueCycle walks, through value fields only, the particles of the complex
// type t that the element declaration decl, local or the top-level one an
// <element ref> particle names, reaches by a value field from particles whose
// chain is chain, applying valueChain at each field; subject names t to the
// checks it calls. It walks the particles spliced returns for t's content
// particle's group, a <group ref> content particle followed by contentGroup,
// the list particleFields gives fields for, so a sequence spliced into its
// parent, inline or through a <group ref>, is a value edge that pushes no
// node, and every other nested model group particle ends its path at elementTerm, whose
// decline the emitting walk meets in field. A choice t, a t whose content particle
// occurs other than exactly once, and a step whose first node (valueNodes) is in
// e.acyclic, end the walk, and it never steps to an element of type xs:anyType, whose
// AnyType reaches itself only through slices. A wildcard particle (wildcardTerm) is a
// leaf, passed before elementTerm: its field holds AnyElement, which reaches AnyType,
// which reaches itself only through the slice Children. It reads the schema, never
// e.emitted or e.elements, so a type the emitting walk shares, and the type of a
// declaration an <element ref> particle names, is walked again here unless e.acyclic
// holds its step's first node; that node joins e.acyclic once the walk of t's particles
// reaches the end, and a step that pushes none, a local decl of an anonymous t that a
// complex type declares, is never memoized. It returns only a struct-value recursion:
// every other failure on its path ends that path, since the emitting walk meets it.
func (e *emitter) valueCycle(chain []valueNode, decl xsd.ElementDeclaration, subject string, t xsd.ComplexType) error {
	nodes := valueNodes(decl, t)
	if len(nodes) > 0 {
		if _, done := e.acyclic[nodes[0]]; done {
			return nil
		}
	}
	content, err := e.contentGroup(subject, t)
	if err != nil || content != nil && (content.group.Compositor() == xsd.CompositorChoice || content.occurs != exactlyOnce) {
		//nolint:nilerr // a decline of t ends this path: the emitting walk meets it and returns it in walk order.
		return nil
	}
	chain, err = valueChain(chain, true, decl, t)
	if err != nil {
		return err
	}
	var particles []xsd.Particle
	if content != nil {
		particles, err = e.spliced(subject, t.Loc(), content.group.Particles())
	}
	if err != nil {
		//nolint:nilerr // a <group ref> that does not resolve ends this path: the emitting walk meets it in particleFields.
		return nil
	}
	for _, p := range particles {
		if p.Occurs() != exactlyOnce || wildcardTerm(p.Term()) {
			continue
		}
		el, err := e.elementTerm(subject, t.Loc(), p.Term())
		if err != nil {
			continue
		}
		td, err := e.declaredType(el)
		if err != nil {
			continue
		}
		ct, isComplex := td.(xsd.ComplexType)
		if isAnyType(td) || !isComplex {
			continue
		}
		elSubject := e.localOrigin(el, ct).subject
		if el.ScopeVariety() == xsd.ScopeGlobal {
			elSubject = e.elementOrigin(el, ct).subject
		}
		if err := e.valueCycle(chain, el, elSubject, ct); err != nil {
			return err
		}
	}
	if len(nodes) > 0 {
		e.acyclic[nodes[0]] = struct{}{}
	}
	return nil
}

// elementTerm returns the element declaration a sequence or choice particle's
// term holds: a local declaration, or the top-level one an <element ref>
// particle names (§3.3.2.4 ref.elt.global), followed through
// [xsd.Schema.Element]. It declines every other term: a model group, inline
// or the {model group} a <group ref> names (modelGroupTerm), as a nested
// model group particle of that group's compositor, and a wildcard. subject
// and loc name the complex type holding the particle.
func (e *emitter) elementTerm(subject string, loc xsderr.Loc, term xsd.TermOrRef) (xsd.ElementDeclaration, error) {
	group, isGroup, err := e.modelGroupTerm(subject, loc, term)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	if isGroup {
		return xsd.ElementDeclaration{}, decline(loc, constructNestedGroup, group.Compositor(), subject)
	}
	switch term := term.(type) {
	case xsd.ElementDeclarationRef:
		// No substitution check: every member of the named declaration's
		// substitution group is top-level, so emit's walk of every top-level
		// declaration declines it in declaredType.
		decl, ok := e.schema.Element(term.Name)
		if !ok {
			return xsd.ElementDeclaration{}, fmt.Errorf("%s at %s: its <element ref> particle to %s does not resolve", subject, loc, term.Name)
		}
		return decl, nil
	case xsd.ResolvedTerm:
		switch inner := term.Term.(type) {
		case xsd.ElementDeclaration:
			return inner, nil
		case xsd.Wildcard:
			return xsd.ElementDeclaration{}, decline(loc, constructWildcard, subject)
		}
	}
	return xsd.ElementDeclaration{}, fmt.Errorf("%s at %s: unexpected particle term %T", subject, loc, term)
}
