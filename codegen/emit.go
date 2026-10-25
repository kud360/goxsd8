package codegen

import (
	"bytes"
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

	longBuiltin          = xsd.QName{Space: xsd.XMLSchemaNS, Local: "long"}
	intBuiltin           = xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}
	shortBuiltin         = xsd.QName{Space: xsd.XMLSchemaNS, Local: "short"}
	byteBuiltin          = xsd.QName{Space: xsd.XMLSchemaNS, Local: "byte"}
	unsignedLongBuiltin  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedLong"}
	unsignedIntBuiltin   = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedInt"}
	unsignedShortBuiltin = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedShort"}
	unsignedByteBuiltin  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "unsignedByte"}
)

// emitter builds one target's source. decls holds each emitted type
// declaration in emission order: a top-level element's type, then the types
// nested in its type definition that no earlier element emitted, depth first
// in particle order; a sealed interface's branch structs follow it, each
// before the types nested in its alternative.
type emitter struct {
	schema *xsd.Schema
	types  *namer
	// base is the target's Dir made absolute, the directory each header
	// renders its declaration's location against (headerLoc).
	base  string
	decls []string
	// emitted holds the type emitted for each anonymous complex type nested in
	// a named complex type, so that every top-level element whose type is that
	// named type shares it; it is a lookup index only and never iterated
	// (STYLE D2). It is keyed by anonymous complex types, and a branch struct
	// is not one: a branch is never stored here, and it rides with its
	// interface.
	emitted map[nestedKey]emittedType
}

// emittedType is the Go type emitted for one complex type definition: its
// name, and whether it is a sealed interface, which a field holds bare rather
// than through a pointer.
type emittedType struct {
	name   string
	sealed bool
}

// scope is where the structs nested in one struct take their names from: the
// path of expanded names from their nearest named ancestor, that ancestor's
// first, and whether that ancestor is a named complex type (false when it is a
// top-level element whose type is anonymous). A struct is named from the
// path's local names and keyed by its expanded names (nestedKey).
type scope struct {
	underNamedType bool
	path           []xsd.QName
}

// locals returns the local names of the path, the names the namer joins.
func (s scope) locals() []string {
	locals := make([]string, len(s.path))
	for i, name := range s.path {
		locals[i] = name.Local
	}
	return locals
}

// nestedKey identifies an anonymous complex type nested in a named complex
// type by the expanded names along its path: the named type's, then each
// element's {target namespace} and {name} (§3.3.2.3), each Go-quoted so that
// the concatenation is unambiguous.
type nestedKey string

// key returns the nestedKey of the struct whose path is s's.
func (s scope) key() nestedKey {
	var b strings.Builder
	for _, name := range s.path {
		b.WriteString(strconv.Quote(name.Space))
		b.WriteString(strconv.Quote(name.Local))
	}
	return nestedKey(b.String())
}

// emit returns the gofmt-formatted source of package pkg for schema, or the
// first decline in document order. base is the absolute directory the source
// is written into.
func emit(schema *xsd.Schema, pkg, base string) ([]byte, error) {
	e := &emitter{schema: schema, types: newNamer(), base: base, emitted: map[nestedKey]emittedType{}}
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
// as form and elementFormDefault set it, §3.3.2.3), and its Loc as headerLoc
// renders it against e.base. Only the header renders a Loc so: every decline
// and error names the Loc as the declaration carries it.
func (e *emitter) header(name string, decl xsd.ElementDeclaration) string {
	return fmt.Sprintf("// %s is generated from element %s declared at %s.\n", name, decl.Name(), headerLoc(e.base, decl.Loc()))
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
// over a builtin for an atomic simple {type definition}, a struct or sealed
// interface for an accepted complex one, named or anonymous. That type is
// named from the element; the types nested in it are named from the type when
// it is named, and from the element when it is anonymous.
func (e *emitter) topLevel(decl xsd.ElementDeclaration) error {
	td, err := e.declaredType(decl)
	if err != nil {
		return err
	}
	switch t := td.(type) {
	case *xsd.SimpleType:
		builtin, err := e.builtinType("element "+decl.Name().String(), decl.Loc(), t)
		if err != nil {
			return err
		}
		name := e.types.claim(decl.Name().Local)
		e.decls = append(e.decls, e.header(name, decl)+fmt.Sprintf("type %s %s\n", name, builtin))
		return nil
	case xsd.ComplexType:
		nested := scope{path: []xsd.QName{decl.Name()}}
		if t.Name() != (xsd.QName{}) {
			nested = scope{underNamedType: true, path: []xsd.QName{t.Name()}}
		}
		_, err := e.complexType(e.types.claim(decl.Name().Local), nested, decl, t)
		return err
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
	if td.Name() == anyTypeName {
		return nil, decline(decl.Loc(), constructAnyTypeElement, subject)
	}
	return td, nil
}

// builtinType is the Go builtin a simple-typed element's or attribute's field,
// or an element's named type, is declared over, chosen by the package doc's
// "# Coverage" through boundedInteger and then t's {primitive type definition}
// (Datatypes §4.1.1). subject names the declaration the type is read from, and
// loc, its Loc, is a decline's.
func (e *emitter) builtinType(subject string, loc xsderr.Loc, t *xsd.SimpleType) (string, error) {
	typed := fmt.Sprintf("%s with {type definition} %s", subject, typeName(t))
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
		return "", decline(loc, constructListType, typed)
	case xsd.Union:
		return "", decline(loc, constructUnionType, typed)
	default:
		return "", decline(loc, constructAbsentVariety, typed)
	}
	integer, err := e.boundedInteger(t)
	if err != nil {
		return "", fmt.Errorf("%s at %s: %w", subject, loc, err)
	}
	if integer != "" {
		return integer, nil
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
	}
	return "", decline(loc, constructUncoveredPrimitive, typed, primitive.Name())
}

// boundedInteger is the Go integer "# Coverage" maps t to through the eight
// bounded builtin integer types (Datatypes §3.4.16–§3.4.19, §3.4.21–§3.4.24),
// walking t's {base type definition} chain (Structures §3.16.1) from t itself,
// or "" when the walk ends at xs:anySimpleType's absent base without meeting
// one. Each Go type's range equals its XSD type's value space.
func (e *emitter) boundedInteger(t *xsd.SimpleType) (string, error) {
	for step := t; step != nil; {
		switch step.Name() {
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

// complexType emits the type name for decl's complex {type definition} t — a
// struct for sequence or empty content, holding a field per particle and then
// one per attribute use, or a sealed interface for choice content, which
// declines {attribute uses} — then the types of its anonymous-typed particles,
// named in nested, in the package doc's "# Coverage" shape.
func (e *emitter) complexType(name string, nested scope, decl xsd.ElementDeclaration, t xsd.ComplexType) (emittedType, error) {
	subject := "complex type " + typeName(t) + " of element " + decl.Name().String()
	group, hasParticle, err := e.contentGroup(subject, t)
	if err != nil {
		return emittedType{}, err
	}
	if hasParticle && group.Compositor() == xsd.CompositorChoice {
		if len(t.AttributeUses()) > 0 {
			return emittedType{}, decline(t.Loc(), constructChoiceAttributeUses, subject)
		}
		return emittedType{name: name, sealed: true}, e.sealedType(name, nested, subject, decl, t, group.Particles())
	}
	slot := len(e.decls)
	e.decls = append(e.decls, "")
	fields := newNamer()
	var body bytes.Buffer
	if hasParticle {
		for _, p := range group.Particles() {
			field, err := e.field(nested, subject, t.Loc(), p)
			if err != nil {
				return emittedType{}, err
			}
			fmt.Fprintf(&body, "\t%s %s\n", fields.claim(field.local), field.goType)
		}
	}
	for _, u := range t.AttributeUses() {
		field, err := e.attributeField(subject, t.Loc(), u)
		if err != nil {
			return emittedType{}, err
		}
		fmt.Fprintf(&body, "\t%s %s\n", fields.claim(field.local), field.goType)
	}
	typeDecl := fmt.Sprintf("type %s struct {\n%s}\n", name, body.String())
	if body.Len() == 0 {
		typeDecl = fmt.Sprintf("type %s struct{}\n", name)
	}
	e.decls[slot] = e.header(name, decl) + typeDecl
	return emittedType{name: name}, nil
}

// sealedType emits the sealed interface name for decl's complex {type
// definition} t, whose content particle's term is a choice of particles
// (§3.8.4.1.2 sec-choices: one alternative per pass), in the package doc's
// "# Coverage" shape: the interface, then each branch struct in particle
// order, followed by the types of its anonymous-typed particle, named in
// nested.
//
// contentGroup has already required the content particle's {min occurs} to be
// 1, so the minimum of its effective total range (§3.8.6.6 cos-choice-range)
// is the least {min occurs} or range minimum among the alternatives, or 0 when
// there are no {particles}, and the particle is emptiable (§3.9.6.3
// cos-group-emptiable) exactly when that minimum is 0. An alternative with
// {min occurs} 0 makes it 0 whatever the others are; any alternative but a
// local element declaration is declined in particle order, so when no decline
// fires every alternative is a local element with {min occurs} at least 1.
func (e *emitter) sealedType(name string, nested scope, subject string, decl xsd.ElementDeclaration, t xsd.ComplexType, particles []xsd.Particle) error {
	if len(particles) == 0 {
		return decline(t.Loc(), constructEmptyChoice, subject)
	}
	for _, p := range particles {
		alt, err := localElement(subject, t.Loc(), p.Term())
		if err != nil {
			return err
		}
		if p.Occurs().Min() == 0 {
			return decline(t.Loc(), constructEmptiableChoice, subject, alt.Name())
		}
	}
	marker := "is" + name
	e.decls = append(e.decls, e.header(name, decl)+fmt.Sprintf("type %s interface {\n\t%s()\n}\n", name, marker))
	for _, p := range particles {
		alt, err := localElement(subject, t.Loc(), p.Term())
		if err != nil {
			return err
		}
		branch := e.types.claim(name, alt.Name().Local)
		slot := len(e.decls)
		e.decls = append(e.decls, "")
		field, err := e.field(nested, subject, t.Loc(), p)
		if err != nil {
			return err
		}
		e.decls[slot] = e.header(branch, alt) + fmt.Sprintf("type %s struct {\n\t%s %s\n}\n\nfunc (%s) %s() {}\n",
			branch, newNamer().claim(field.local), field.goType, branch, marker)
	}
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

// contentGroup checks the complex type t, named in declines by subject,
// against the package doc's "# Coverage" complex-type shape — a restriction
// of xs:anyType, implicit (§3.4.2.3.2 dcl.ctd.ctcc.implicit) or explicit
// (§3.4.2.3.1 dcl.ctd.ctcc.explicit), with the content §3.4.2.3.3
// dcl.ctd.ctcc.common maps. ok answers whether t's {content type} has a
// {particle}: for element-only content it is true and group is the sequence or
// choice model group that is the {particle}'s term, never the source <choice>
// itself; for {variety} empty (§3.4.1), which has no {particle}, it is false,
// err is nil and group is the zero value, which callers must not read. Every
// other content declines. Particles and model groups carry no Loc, so their
// declines cite t's.
func (e *emitter) contentGroup(subject string, t xsd.ComplexType) (group xsd.ModelGroup, ok bool, err error) {
	if t.DerivationMethod() != xsd.DerivationRestriction {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructDerivation, subject, t.DerivationMethod())
	}
	base, found := e.schema.ResolvedType(t.Base())
	if !found {
		return xsd.ModelGroup{}, false, fmt.Errorf("%s at %s: its {base type definition} does not resolve", subject, t.Loc())
	}
	if base.Name() != anyTypeName {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructRestrictionBase, subject, typeName(base))
	}
	switch {
	case t.Abstract():
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructComplexTypeAbstract, subject)
	case len(t.Assertions()) > 0:
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructAssertions, subject)
	}
	if _, has := t.AttributeWildcard(); has {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructAttributeWildcard, subject)
	}
	content, isElement := t.ContentType().(xsd.ElementContent)
	if !isElement {
		switch variety := t.ContentType().Variety(); variety {
		case xsd.ContentEmpty:
			return xsd.ModelGroup{}, false, nil
		case xsd.ContentSimple:
			return xsd.ModelGroup{}, false, decline(t.Loc(), constructSimpleContent, subject)
		default:
			return xsd.ModelGroup{}, false, fmt.Errorf("%s at %s: unexpected content type %T of {variety} %s", subject, t.Loc(), t.ContentType(), variety)
		}
	}
	if content.Mixed {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructMixedContent, subject)
	}
	if content.OpenContent != nil {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructOpenContent, subject)
	}
	if occurs := content.Particle.Occurs(); occurs != exactlyOnce {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructContentOccurs, subject, occurs)
	}
	resolved, isResolved := content.Particle.Term().(xsd.ResolvedTerm)
	if !isResolved {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructContentGroupRef, subject)
	}
	group, isGroup := resolved.Term.(xsd.ModelGroup)
	if !isGroup {
		return xsd.ModelGroup{}, false, fmt.Errorf("%s at %s: unexpected content particle term %T", subject, t.Loc(), resolved.Term)
	}
	if group.Compositor() == xsd.CompositorAll {
		return xsd.ModelGroup{}, false, decline(t.Loc(), constructAll, subject)
	}
	return group, true, nil
}

// fieldSpec is one struct field before its name is claimed.
type fieldSpec struct {
	local  string
	goType string
}

// field returns the field of particle p in a struct whose nested types are
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
	var typ emittedType
	switch ft := td.(type) {
	case *xsd.SimpleType:
		typ.name, err = e.builtinType("element "+decl.Name().String(), decl.Loc(), ft)
	case xsd.ComplexType:
		typ, err = e.nestedType(nested, decl, ft)
	default:
		err = fmt.Errorf("element %s at %s: unexpected type definition %T", decl.Name(), decl.Loc(), td)
	}
	if err != nil {
		return fieldSpec{}, err
	}
	goType := typ.name
	occurs := p.Occurs()
	upper, bounded := occurs.Max()
	switch {
	case !bounded || upper > 1:
		goType = "[]" + goType
	case upper == 0:
		return fieldSpec{}, decline(loc, constructVacuousParticle, decl.Name(), subject)
	case upper == 1 && occurs.Min() == 0 && !typ.sealed:
		goType = "*" + goType
	}
	return fieldSpec{local: decl.Name().Local, goType: goType}, nil
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
	goType, err := e.builtinType(attr, decl.Loc(), st)
	if err != nil {
		return fieldSpec{}, err
	}
	if !u.Required() {
		goType = "*" + goType
	}
	return fieldSpec{local: name.Local, goType: goType}, nil
}

// nestedType returns the type emitted for a local element declaration's
// anonymous complex type, named by the path from its nearest named ancestor in
// parent. Under a named complex type the type is emitted once per nestedKey,
// by the first top-level element that reaches it, and later ones share it.
// Both the emitted lookup and the store run only under a named complex type:
// a top-level element may share a complex type's expanded name, since the two
// are separate symbol spaces (§3.3.2), so a key whose path starts at an
// element must never read an entry stored from a type.
func (e *emitter) nestedType(parent scope, decl xsd.ElementDeclaration, t xsd.ComplexType) (emittedType, error) {
	if t.Name() != (xsd.QName{}) {
		return emittedType{}, decline(decl.Loc(), constructNamedComplexLocal, decl.Name(), t.Name())
	}
	child := scope{underNamedType: parent.underNamedType, path: append(slices.Clone(parent.path), decl.Name())}
	key := child.key()
	if typ, ok := e.emitted[key]; ok && child.underNamedType {
		return typ, nil
	}
	typ, err := e.complexType(e.types.claim(child.locals()...), child, decl, t)
	if err != nil {
		return emittedType{}, err
	}
	if child.underNamedType {
		e.emitted[key] = typ
	}
	return typ, nil
}

// localElement returns the local element declaration a sequence or choice
// particle's term holds, declining every other term; a reference is declined, never
// dereferenced. subject and loc name the complex type holding the particle.
func localElement(subject string, loc xsderr.Loc, term xsd.TermOrRef) (xsd.ElementDeclaration, error) {
	switch term := term.(type) {
	case xsd.ElementDeclarationRef:
		return xsd.ElementDeclaration{}, decline(loc, constructElementRef, term.Name, subject)
	case xsd.ModelGroupRef:
		return xsd.ElementDeclaration{}, decline(loc, constructGroupRef, term.Name, subject)
	case xsd.ResolvedTerm:
		switch inner := term.Term.(type) {
		case xsd.ElementDeclaration:
			return inner, nil
		case xsd.ModelGroup:
			return xsd.ElementDeclaration{}, decline(loc, constructNestedGroup, inner.Compositor(), subject)
		case xsd.Wildcard:
			return xsd.ElementDeclaration{}, decline(loc, constructWildcard, subject)
		}
	}
	return xsd.ElementDeclaration{}, fmt.Errorf("%s at %s: unexpected particle term %T", subject, loc, term)
}
