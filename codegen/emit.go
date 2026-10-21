package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"slices"
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
)

// decline reports a construct the generator does not cover. It is a plain
// error wrapping errors.ErrUnsupported, never an *xsderr.Error: no schema
// rule is violated (STYLE E2).
//
// GAP(codegen): every construct outside the package doc's "# Coverage" slice
// declines here; that section lists them under the same marker. No tracking
// issue owns lifting a decline yet.
func decline(loc xsderr.Loc, format string, args ...any) error {
	return fmt.Errorf("%s at %s is not supported by codegen, a generator limitation, not a schema fault: %w",
		fmt.Sprintf(format, args...), loc, errors.ErrUnsupported)
}

// emitter builds one target's source. decls holds each emitted type
// declaration in emission order: a top-level element's type, then the
// structs nested in its type definition that no earlier element emitted,
// depth first in particle order.
type emitter struct {
	schema *xsd.Schema
	types  *namer
	decls  []string
	// emitted names the struct emitted for each anonymous complex type nested
	// in a named complex type, so that every top-level element whose type is
	// that named type shares it; it is a lookup index only and never iterated
	// (STYLE D2).
	emitted map[nestedKey]string
}

// scope is where the structs nested in one struct take their names from: the
// path of XSD local names from their nearest named ancestor, and that
// ancestor's name when it is a named complex type (zero when it is a
// top-level element whose type is anonymous).
type scope struct {
	namedType xsd.QName
	path      []string
}

// nestedKey identifies an anonymous complex type nested in a named complex
// type: the named type, and the path of local names from it joined by a
// space, which no NCName contains.
type nestedKey struct {
	namedType xsd.QName
	path      string
}

// emit returns the gofmt-formatted source of package pkg for schema, or the
// first decline in document order.
func emit(schema *xsd.Schema, pkg string) ([]byte, error) {
	e := &emitter{schema: schema, types: newNamer(), emitted: map[nestedKey]string{}}
	for _, decl := range schema.Elements() {
		if err := e.topLevel(decl); err != nil {
			return nil, err
		}
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s\n\npackage %s\n", generatedHeader, pkg)
	for _, d := range e.decls {
		b.WriteString("\n")
		b.WriteString(d)
	}
	src, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting the emitted source: %w", err)
	}
	return src, nil
}

// header is the doc comment of an emitted type: the declaration's expanded
// name, whose namespace is its {target namespace} (for a local declaration,
// as form and elementFormDefault set it, §3.3.2.3), and its Loc.
func header(name string, decl xsd.ElementDeclaration) string {
	return fmt.Sprintf("// %s is generated from element %s declared at %s.\n", name, decl.Name(), decl.Loc())
}

// topLevel emits the type of one top-level element declaration: a named type
// over a builtin for an atomic simple {type definition}, a struct for an
// accepted complex one, named or anonymous. The struct is named from the
// element; the structs nested in it are named from the type when it is named,
// and from the element when it is anonymous.
func (e *emitter) topLevel(decl xsd.ElementDeclaration) error {
	td, err := e.declaredType(decl)
	if err != nil {
		return err
	}
	switch t := td.(type) {
	case *xsd.SimpleType:
		builtin, err := e.builtinType(decl, t)
		if err != nil {
			return err
		}
		name := e.types.claim(decl.Name().Local)
		e.decls = append(e.decls, header(name, decl)+fmt.Sprintf("type %s %s\n", name, builtin))
		return nil
	case xsd.ComplexType:
		nested := scope{path: []string{decl.Name().Local}}
		if t.Name() != (xsd.QName{}) {
			nested = scope{namedType: t.Name(), path: []string{t.Name().Local}}
		}
		return e.structType(e.types.claim(decl.Name().Local), nested, decl, t)
	}
	return fmt.Errorf("element %s at %s: unexpected type definition %T", decl.Name(), decl.Loc(), td)
}

// declaredType checks the element declaration properties the slice declines
// on every declaration it emits from, and returns the resolved {type
// definition}, declining xs:anyType.
func (e *emitter) declaredType(decl xsd.ElementDeclaration) (xsd.TypeDefinition, error) {
	subject := "element " + decl.Name().String()
	switch {
	case decl.Nillable():
		return nil, decline(decl.Loc(), "%s with {nillable} true", subject)
	case decl.Abstract():
		return nil, decline(decl.Loc(), "%s with {abstract} true", subject)
	case len(decl.SubstitutionGroupAffiliationNames()) > 0:
		return nil, decline(decl.Loc(), "%s with {substitution group affiliations}", subject)
	case len(decl.IdentityConstraints()) > 0:
		return nil, decline(decl.Loc(), "%s with {identity-constraint definitions}", subject)
	}
	if _, ok := decl.ValueConstraint(); ok {
		return nil, decline(decl.Loc(), "%s with a {value constraint}", subject)
	}
	if _, ok := decl.TypeTable(); ok {
		return nil, decline(decl.Loc(), "%s with a {type table}", subject)
	}
	td, ok := e.schema.ResolvedType(decl.TypeDefinition())
	if !ok {
		return nil, fmt.Errorf("%s at %s: its {type definition} does not resolve", subject, decl.Loc())
	}
	if td.Name() == anyTypeName {
		return nil, decline(decl.Loc(), "%s with {type definition} xs:anyType", subject)
	}
	return td, nil
}

// builtinType is the Go builtin a simple-typed element's field or named type
// is declared over, chosen from t's {primitive type definition} (§3.16.1) by
// the package doc's "# Coverage". decl is the element the type is read from,
// and its Loc is a decline's.
func (e *emitter) builtinType(decl xsd.ElementDeclaration, t *xsd.SimpleType) (string, error) {
	subject := fmt.Sprintf("element %s with {type definition} %s", decl.Name(), typeName(t))
	if t.IsSpecial() {
		return "", decline(decl.Loc(), "%s, a special type", subject)
	}
	variety, err := t.Variety(e.schema)
	if err != nil {
		return "", fmt.Errorf("element %s at %s: %w", decl.Name(), decl.Loc(), err)
	}
	switch variety.(type) {
	case xsd.Atomic:
	case xsd.List:
		return "", decline(decl.Loc(), "%s, a list type", subject)
	case xsd.Union:
		return "", decline(decl.Loc(), "%s, a union type", subject)
	default:
		return "", decline(decl.Loc(), "%s, of absent {variety}", subject)
	}
	primitive, err := t.Primitive(e.schema)
	if err != nil {
		return "", fmt.Errorf("element %s at %s: %w", decl.Name(), decl.Loc(), err)
	}
	if primitive == nil {
		return "", decline(decl.Loc(), "%s, with no {primitive type definition}", subject)
	}
	switch primitive.Name() {
	case stringPrimitive, anyURIPrimitive:
		return "string", nil
	case booleanPrimitive:
		return "bool", nil
	}
	return "", decline(decl.Loc(), "%s, whose {primitive type definition} is %s", subject, primitive.Name())
}

// typeName names a type definition in a decline message.
func typeName(t xsd.TypeDefinition) string {
	if t.Name() == (xsd.QName{}) {
		return "(anonymous)"
	}
	return t.Name().String()
}

// structType emits the struct name for decl's complex {type definition} t,
// then the structs of its anonymous-typed particles, named in nested.
func (e *emitter) structType(name string, nested scope, decl xsd.ElementDeclaration, t xsd.ComplexType) error {
	subject := "complex type " + typeName(t) + " of element " + decl.Name().String()
	particles, err := e.sequence(subject, t)
	if err != nil {
		return err
	}
	slot := len(e.decls)
	e.decls = append(e.decls, "")
	fields := newNamer()
	var body bytes.Buffer
	for _, p := range particles {
		field, err := e.field(nested, subject, t.Loc(), p)
		if err != nil {
			return err
		}
		fmt.Fprintf(&body, "\t%s %s\n", fields.claim(field.local), field.goType)
	}
	e.decls[slot] = header(name, decl) + fmt.Sprintf("type %s struct {\n%s}\n", name, body.String())
	return nil
}

// exactlyOnce is the {min occurs} = {max occurs} = 1 range the slice requires
// of a complex type's content particle.
var exactlyOnce = mustOccurs(xsd.NewOccurs(xsderr.Loc{}, 1, 1))

// mustOccurs unwraps a constructor result that cannot fail on constant
// arguments.
func mustOccurs(o xsd.Occurs, err error) xsd.Occurs {
	if err != nil {
		panic(err)
	}
	return o
}

// sequence checks the complex type t, named in declines by subject, against
// the package doc's "# Coverage" struct shape — a restriction of xs:anyType,
// implicit (§3.4.2.3.2 dcl.ctd.ctcc.implicit) or explicit (§3.4.2.3.1
// dcl.ctd.ctcc.explicit), with the content §3.4.2.3.3 dcl.ctd.ctcc.common
// maps — and returns the particles of its sequence. Particles and model
// groups carry no Loc, so their declines cite t's.
func (e *emitter) sequence(subject string, t xsd.ComplexType) ([]xsd.Particle, error) {
	if t.DerivationMethod() != xsd.DerivationRestriction {
		return nil, decline(t.Loc(), "%s, derived by %s", subject, t.DerivationMethod())
	}
	base, ok := e.schema.ResolvedType(t.Base())
	if !ok {
		return nil, fmt.Errorf("%s at %s: its {base type definition} does not resolve", subject, t.Loc())
	}
	if base.Name() != anyTypeName {
		return nil, decline(t.Loc(), "%s, a restriction of %s rather than of xs:anyType", subject, typeName(base))
	}
	switch {
	case t.Abstract():
		return nil, decline(t.Loc(), "%s with {abstract} true", subject)
	case len(t.AttributeUses()) > 0:
		return nil, decline(t.Loc(), "%s with {attribute uses}", subject)
	case len(t.Assertions()) > 0:
		return nil, decline(t.Loc(), "%s with {assertions}", subject)
	}
	if _, ok := t.AttributeWildcard(); ok {
		return nil, decline(t.Loc(), "%s with an {attribute wildcard}", subject)
	}
	content, ok := t.ContentType().(xsd.ElementContent)
	if !ok {
		return nil, decline(t.Loc(), "%s with %s content", subject, t.ContentType().Variety())
	}
	if content.Mixed {
		return nil, decline(t.Loc(), "%s with mixed content", subject)
	}
	if content.OpenContent != nil {
		return nil, decline(t.Loc(), "%s with {open content}", subject)
	}
	if occurs := content.Particle.Occurs(); occurs != exactlyOnce {
		return nil, decline(t.Loc(), "%s whose content particle occurs %s times rather than exactly once", subject, occurs)
	}
	resolved, ok := content.Particle.Term().(xsd.ResolvedTerm)
	if !ok {
		return nil, decline(t.Loc(), "%s whose content particle is a <group ref>", subject)
	}
	group, ok := resolved.Term.(xsd.ModelGroup)
	if !ok {
		return nil, fmt.Errorf("%s at %s: unexpected content particle term %T", subject, t.Loc(), resolved.Term)
	}
	if group.Compositor() != xsd.CompositorSequence {
		return nil, decline(t.Loc(), "%s with a %s compositor", subject, group.Compositor())
	}
	return group.Particles(), nil
}

// fieldSpec is one struct field before its name is claimed.
type fieldSpec struct {
	local  string
	goType string
}

// field returns the field of particle p in a struct whose nested structs are
// named in nested, for the complex type at loc named in declines by subject,
// typed by the package doc's "# Coverage" field rules.
func (e *emitter) field(nested scope, subject string, loc xsderr.Loc, p xsd.Particle) (fieldSpec, error) {
	decl, err := localElement(subject, loc, p.Term())
	if err != nil {
		return fieldSpec{}, err
	}
	td, err := e.declaredType(decl)
	if err != nil {
		return fieldSpec{}, err
	}
	var goType string
	switch ft := td.(type) {
	case *xsd.SimpleType:
		goType, err = e.builtinType(decl, ft)
	case xsd.ComplexType:
		goType, err = e.nestedStruct(nested, decl, ft)
	default:
		err = fmt.Errorf("element %s at %s: unexpected type definition %T", decl.Name(), decl.Loc(), td)
	}
	if err != nil {
		return fieldSpec{}, err
	}
	occurs := p.Occurs()
	upper, bounded := occurs.Max()
	switch {
	case !bounded || upper > 1:
		goType = "[]" + goType
	case upper == 0:
		return fieldSpec{}, decline(loc, "a particle of element %s with {max occurs} 0 in %s", decl.Name(), subject)
	case upper == 1 && occurs.Min() == 0:
		goType = "*" + goType
	}
	return fieldSpec{local: decl.Name().Local, goType: goType}, nil
}

// nestedStruct returns the struct name of a local element declaration's
// anonymous complex type, named by the path from its nearest named ancestor in
// parent. Under a named complex type the struct is emitted once, by the first
// top-level element that reaches it, and later ones share it.
func (e *emitter) nestedStruct(parent scope, decl xsd.ElementDeclaration, t xsd.ComplexType) (string, error) {
	if t.Name() != (xsd.QName{}) {
		return "", decline(decl.Loc(), "local element %s with named complex {type definition} %s", decl.Name(), t.Name())
	}
	child := scope{namedType: parent.namedType, path: append(slices.Clone(parent.path), decl.Name().Local)}
	key := nestedKey{namedType: child.namedType, path: strings.Join(child.path, " ")}
	if name, ok := e.emitted[key]; ok {
		return name, nil
	}
	name := e.types.claim(child.path...)
	if child.namedType != (xsd.QName{}) {
		e.emitted[key] = name
	}
	return name, e.structType(name, child, decl, t)
}

// localElement returns the local element declaration a sequence particle's
// term holds, declining every other term; a reference is declined, never
// dereferenced. subject and loc name the complex type holding the particle.
func localElement(subject string, loc xsderr.Loc, term xsd.TermOrRef) (xsd.ElementDeclaration, error) {
	switch term := term.(type) {
	case xsd.ElementDeclarationRef:
		return xsd.ElementDeclaration{}, decline(loc, "an <element ref> particle to %s in %s", term.Name, subject)
	case xsd.ModelGroupRef:
		return xsd.ElementDeclaration{}, decline(loc, "a <group ref> particle to %s in %s", term.Name, subject)
	case xsd.ResolvedTerm:
		switch inner := term.Term.(type) {
		case xsd.ElementDeclaration:
			return inner, nil
		case xsd.ModelGroup:
			return xsd.ElementDeclaration{}, decline(loc, "a nested %s model group particle in %s", inner.Compositor(), subject)
		case xsd.Wildcard:
			return xsd.ElementDeclaration{}, decline(loc, "a wildcard particle in %s", subject)
		}
	}
	return xsd.ElementDeclaration{}, fmt.Errorf("%s at %s: unexpected particle term %T", subject, loc, term)
}
