package jsonsrc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/kud360/goxsd8/internal/xmlname"
	"github.com/kud360/goxsd8/internal/xmlqname"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// Option configures [Validate]. The zero set of options is a complete,
// usable configuration: options only replace defaults (STYLE T1).
type Option func(*config)

// config is the resolved [Validate] configuration.
type config struct {
	uri string
}

// newConfig applies opts over the defaults: the document is unnamed, so
// every Loc it produces renders with "?" for its URI.
func newConfig(opts []Option) config {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithURI names the document being read, so every [xsderr.Loc] the
// assessment and every error of [Validate] cites identifies the file a reader
// has to open. [Validate] takes an [io.Reader], which carries no name of its
// own.
func WithURI(uri string) Option {
	return func(c *config) { c.uri = uri }
}

// Validate maps the JSON instance in r onto the validate infoset, on the
// mapping the package doc states, and assesses it against v's schema.
//
// It returns (nil, err) whenever no assessment of the whole document stands,
// and (result, nil) otherwise; a result it returns never carries
// [validate.Result.Err]. The faults are: a nil v or r; JSON that is malformed,
// or followed by further content; and every input the mapping DECLINES —
// a top level that is not an object with exactly one member naming a
// top-level element declaration, and, anywhere in the document, a member the
// implemented arms do not map (a key no child element declaration and no
// attribute use carries, a key two expanded names carry, a key both a child
// declaration and an attribute use carry, an attribute member after a child
// member or repeated, an attribute member whose value is null, an object or an
// array, a "$type" or "$xmlns" member out of its leading place or of the wrong
// shape, any other "$"-prefixed reserved member, a nested array, a declaration
// with a {type table}; the package doc lists them all). A decline names the
// arm it is not, and is never a silent drop or a guessed mapping: a member
// that would map to a wildcard's element or attribute is not part of the
// document the assessment saw, so its result is discarded rather than
// returned. The whole document is read either way, so a fault after the point
// the assessment stopped at is still reported.
//
// Every error is a plain one, prefixed "jsonsrc: " and, past the nil checks,
// the location of the construct it names. None is an [xsderr.Error]: none is a
// validity verdict about a document.
func Validate(v *validate.Validator, r io.Reader, opts ...Option) (*validate.Result, error) {
	if v == nil {
		return nil, fmt.Errorf("jsonsrc: Validate: nil *validate.Validator")
	}
	if r == nil {
		return nil, fmt.Errorf("jsonsrc: Validate: nil io.Reader")
	}
	cfg := newConfig(opts)
	w := newWalker(cfg.uri, r, v.Schema())
	root, err := w.root()
	if err != nil {
		return nil, err
	}
	res := v.Assess(root)
	if w.err != nil {
		return nil, w.err
	}
	if err := w.end(root); err != nil {
		return nil, err
	}
	return res, nil
}

// walker is the one token stream a whole mapping pulls from: a single
// json.Decoder over the positions its tokens are located by, the schema view
// members are classified against, and the one fault that can stop it.
type walker struct {
	dec    *json.Decoder
	pos    *positions
	uri    string
	schema validate.SchemaView
	// err is the walk's single latched fault, a malformed document or a
	// decline. Every cursor reports this one field and none keeps a copy of it
	// (STYLE D3).
	err error
}

func newWalker(uri string, r io.Reader, schema validate.SchemaView) *walker {
	pos := newPositions(r)
	dec := json.NewDecoder(pos)
	// A number's lexical form is its source text, never a float64 round trip:
	// 12345678901234567890 under xs:integer is that integer.
	dec.UseNumber()
	return &walker{dec: dec, pos: pos, uri: uri, schema: schema}
}

// token pulls the next token and the location it starts at.
func (w *walker) token() (json.Token, xsderr.Loc, error) {
	off := w.dec.InputOffset()
	tok, err := w.dec.Token()
	line, col := w.pos.start(off)
	loc := xsderr.Loc{URI: w.uri, Line: line, Col: col}
	if errors.Is(err, io.EOF) {
		return nil, loc, err
	}
	if err != nil {
		return nil, loc, fmt.Errorf("jsonsrc: %s: malformed JSON: %w", loc, err)
	}
	return tok, loc, nil
}

// fail latches err as the walk's fault, keeping the first one met.
func (w *walker) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

// declinef builds the error for an input the mapping declines at loc.
func (w *walker) declinef(loc xsderr.Loc, format string, args ...any) error {
	return fmt.Errorf("jsonsrc: %s: %s", loc, fmt.Sprintf(format, args...))
}

// root reads the top-level object up to the value of its one member and
// returns the element that member names: the validation root.
//
// The root is named by the member's key, matched by local name against the
// top-level element declarations; exactly one must carry it, and its expanded
// name is the root's (package doc, "The validation root").
func (w *walker) root() (*element, error) {
	tok, loc, err := w.token()
	if errors.Is(err, io.EOF) {
		return nil, w.declinef(loc, "the document holds no JSON value")
	}
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, w.declinef(loc, "the top-level value is not an object; the validation root is named by the one member of a top-level object")
	}
	tok, loc, err = w.token()
	if err != nil {
		return nil, unexpectedEOF(err, loc)
	}
	key, ok := tok.(string)
	if !ok {
		return nil, w.declinef(loc, "the top-level object has no member; its one member names the validation root")
	}
	decl, err := w.rootDeclaration(key, loc)
	if err != nil {
		return nil, err
	}
	vtok, vloc, err := w.token()
	if err != nil {
		return nil, unexpectedEOF(err, vloc)
	}
	if vtok == json.Delim('[') {
		return nil, w.declinef(vloc, "the value of root member %q is an array; the validation root is one element", key)
	}
	e, err := w.element(decl, loc, key, vtok, vloc, nil)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// rootDeclaration resolves the root member's key to the one top-level element
// declaration whose local name it is.
func (w *walker) rootDeclaration(key string, loc xsderr.Loc) (xsd.ElementDeclaration, error) {
	if isReserved(key) {
		return xsd.ElementDeclaration{}, w.declinef(loc, "root member %q is a reserved member; the top-level object's one member names the validation root, and \"$xmlns\" and \"$type\" lead an element's object", key)
	}
	var match []xsd.ElementDeclaration
	for _, d := range w.schema.Elements() {
		if d.Name().Local == key {
			match = append(match, d)
		}
	}
	switch len(match) {
	case 0:
		return xsd.ElementDeclaration{}, w.declinef(loc, "root member %q names no top-level element declaration", key)
	case 1:
		return match[0], w.typeTable(match[0], loc)
	}
	return xsd.ElementDeclaration{}, w.declinef(loc, "root member %q names top-level element declarations of %d expanded names (%s and %s); a JSON key carries no namespace to choose between them",
		key, len(match), match[0].Name(), match[1].Name())
}

// object is what an object-valued element's members are classified against,
// read once per element from the type walker.classifyingType chose — its
// declaration's {type definition}, or the type its "$type" names: the owner's
// expanded name, the child element declarations the type contains, the type's
// {attribute uses}, and whether its {content type} is simple. A simple type
// contains none and has none.
type object struct {
	owner     xsd.QName
	contained []xsd.ElementDeclaration
	uses      []xsd.AttributeUse
	// simpleContent is the type's {content type} being simple: the mapping
	// spells no text beside attribute members, so it declines one there.
	simpleContent bool
}

// member is one key's classification: an attribute use of the enclosing type
// where use is non-nil, and otherwise the child element declaration decl.
type member struct {
	decl xsd.ElementDeclaration
	use  *xsd.AttributeUse
}

// classify resolves a member's key, at loc, against o: to the child element
// declaration among o.contained whose local name it is, or to the attribute
// use among o.uses whose {attribute declaration}'s local name it is
// (cvc-complex-type clause 2.1 then matches the attribute to that use by
// expanded name). The first declaration of the name stands for every other
// (xsd.Schema.ContainedElementDeclarations says why that is sound).
//
// It DECLINES a reserved key, walker.lead having read the ones that lead the
// object; a key two expanded names carry among the declarations, or among the
// uses; a key both a declaration and a use carry, which no spec rule decides
// between; a use in the xsi namespace, xsi:nil being the null arm's to present
// and xsi:type the "$type" member's; an attribute use of a type with simple
// content, beside which the mapping spells no text; and a key neither carries,
// whatever an {attribute wildcard} or an element wildcard of the type would
// admit, a wildcard match being the arm this slice does not take.
//
// GAP(jsonsrc): a key no child element declaration and no attribute use
// carries declines, the wildcard member arm not being implemented. That covers
// a key an element wildcard of the type admits (cvc-accept), and a key its
// {attribute wildcard} admits, whose attribute cvc-complex-type clause 2.2
// would match to that wildcard.
//
// GAP(jsonsrc): a key both a child element declaration and an attribute use
// carry declines. RULED permanent by #2613: no spec rule decides between them,
// and the mapping has no warning channel to report a choice through.
//
// The xsi-namespace, two-expanded-name and simple-content declines among the
// uses are useNamed's, under its GAP(jsonsrc) markers.
func (w *walker) classify(o *object, key string, loc xsderr.Loc) (member, error) {
	if isReserved(key) {
		return member{}, w.reserved(key, loc)
	}
	decl, declared, err := w.childNamed(o, key, loc)
	if err != nil {
		return member{}, err
	}
	use, used, err := w.useNamed(o, key, loc)
	if err != nil {
		return member{}, err
	}
	if declared && used {
		return member{}, w.declinef(loc, "member %q names both the child element declaration %s and the attribute use %s of the type of %s; the mapping does not choose between them",
			key, decl.Name(), use.DeclarationName(), o.owner)
	}
	if used {
		return member{use: &use}, nil
	}
	if !declared {
		return member{}, w.declinef(loc, "member %q names no child element declaration and no attribute use of the type of %s; the wildcard member arm is not implemented",
			key, o.owner)
	}
	return member{decl: decl}, w.typeTable(decl, loc)
}

// childNamed is the first of o.contained whose local name is key, declining
// a key two expanded names carry.
func (w *walker) childNamed(o *object, key string, loc xsderr.Loc) (xsd.ElementDeclaration, bool, error) {
	var first xsd.ElementDeclaration
	found := false
	for _, d := range o.contained {
		if d.Name().Local != key {
			continue
		}
		if !found {
			first, found = d, true
			continue
		}
		if d.Name() != first.Name() {
			return xsd.ElementDeclaration{}, false, w.declinef(loc, "member %q names child element declarations of two expanded names in the type of %s (%s and %s); a JSON key carries no namespace to choose between them",
				key, o.owner, first.Name(), d.Name())
		}
	}
	return first, found, nil
}

// useNamed is the member of o.uses whose {attribute declaration}'s local name
// is key. It declines a key uses of two expanded names carry, a use in the xsi
// namespace, and a use of a type with simple content.
//
// GAP(jsonsrc): a key attribute uses of two expanded names carry declines:
// cvc-complex-type clause 2.1 matches an attribute to a use by expanded name,
// and a JSON key carries no namespace to choose between them.
//
// GAP(jsonsrc): a key naming an xsi:type attribute use declines. RULED
// permanent by #2615: a "$type" member is xsi:type's one spelling
// (walker.lead), and a second would let one object present it twice.
//
// GAP(jsonsrc): a key naming an xsi:schemaLocation or
// xsi:noNamespaceSchemaLocation attribute use declines, owned by #2627, which
// rules it a permanent decline or gives it a key spelling.
//
// GAP(jsonsrc): a key naming an xsi:nil attribute use declines. RULED
// permanent by #2614: a null value is xsi:nil's one spelling (walker.nilled),
// and a second would let one object present it twice.
//
// GAP(jsonsrc): an attribute use of a type whose {content type} is simple
// declines. RULED permanent by #2613: the XML twin, simple content beside
// attributes, is valid under cvc-complex-type clauses 1.2 and 2, and the
// mapping spells no text beside attribute members.
func (w *walker) useNamed(o *object, key string, loc xsderr.Loc) (xsd.AttributeUse, bool, error) {
	var first xsd.AttributeUse
	found := false
	for _, u := range o.uses {
		if u.DeclarationName().Local != key {
			continue
		}
		if !found {
			first, found = u, true
			continue
		}
		if u.DeclarationName() != first.DeclarationName() {
			return xsd.AttributeUse{}, false, w.declinef(loc, "member %q names attribute uses of two expanded names in the type of %s (%s and %s); a JSON key carries no namespace to choose between them",
				key, o.owner, first.DeclarationName(), u.DeclarationName())
		}
	}
	if !found {
		return xsd.AttributeUse{}, false, nil
	}
	if first.DeclarationName().Space == xsd.XMLSchemaInstanceNS {
		return xsd.AttributeUse{}, false, w.declinef(loc, "member %q names the attribute use %s of the type of %s; an xsi: attribute is presented only by the null => xsi:nil arm and the \"$type\" member",
			key, first.DeclarationName(), o.owner)
	}
	if o.simpleContent {
		return xsd.AttributeUse{}, false, w.declinef(loc, "member %q names an attribute use of the type of %s, whose {content type} is simple; the mapping spells no text beside attribute members",
			key, o.owner)
	}
	return first, true, nil
}

// typeTable declines a declaration carrying a {type table}: members are
// classified under a type the mapping reads off the declaration or a "$type"
// member, and such an element's ·selected type definition· is neither — it
// turns on the element's attributes (§3.12.4, key-selected-type clause 1), xsi:
// ones included, and is computed only inside validate, so the content
// classified would be a guess. Lifting the decline is the successor filed by
// #2615's post-land pass.
func (w *walker) typeTable(d xsd.ElementDeclaration, loc xsderr.Loc) error {
	if _, ok := d.TypeTable(); ok {
		return w.declinef(loc, "element %s carries a {type table}; conditional type assignment is not implemented", d.Name())
	}
	return nil
}

// The two reserved members the mapping reads (walker.lead).
const (
	memberXmlns = "$xmlns"
	memberType  = "$type"
)

// isReserved reports whether key is a reserved member name: every
// "$"-prefixed one. "$xmlns" and "$type" are mapped where they lead an object
// (walker.lead), and every other one is held.
func isReserved(key string) bool { return len(key) > 0 && key[0] == '$' }

// reserved declines the reserved member key at loc, which walker.lead did not
// read: a "$xmlns" or "$type" out of its place, or any other reserved name.
func (w *walker) reserved(key string, loc xsderr.Loc) error {
	if key == memberXmlns || key == memberType {
		return w.declinef(loc, "reserved member %q is out of place; \"$xmlns\" and then \"$type\" lead an element's object, each at most once, before its other members", key)
	}
	return w.declinef(loc, "reserved member %q; only the \"$type\" and \"$xmlns\" reserved members are mapped", key)
}

// element builds the element a member (or an item of a member's array) maps
// to, from its declaration, the location of its key or item, the key, the
// value's first token and its location, and the scope of the parent's object,
// which it inherits.
func (w *walker) element(decl xsd.ElementDeclaration, loc xsderr.Loc, key string, tok json.Token, tokLoc xsderr.Loc, parent *scope) (*element, error) {
	switch v := tok.(type) {
	case nil:
		return w.nilled(decl, loc, tokLoc, parent), nil
	case json.Delim:
		if v != json.Delim('{') {
			// '[' alone reaches here: Token never yields a closing delimiter
			// where a value is due.
			return nil, w.declinef(tokLoc, "an array is an item of the array of member %q; only an object or a scalar maps to one element", key)
		}
		return w.objectElement(decl, loc, tokLoc, parent)
	}
	lexical, ok := scalarLexical(tok)
	if !ok {
		panic(fmt.Sprintf("jsonsrc: Token yielded %T", tok))
	}
	return w.scalar(decl, loc, lexical, tokLoc, parent), nil
}

// objectElement builds an object-valued element whose '{' is at tokLoc: its
// leading reserved members (lead), then its attribute members (attributes),
// each classified under the type its members are (classifyingType), and a
// cursor over the rest.
func (w *walker) objectElement(decl xsd.ElementDeclaration, loc, tokLoc xsderr.Loc, parent *scope) (*element, error) {
	e := &element{w: w, decl: decl, loc: loc, scope: parent}
	typ, tok, tokAt, err := w.lead(e)
	if err != nil {
		return nil, err
	}
	td, err := w.classifyingType(e, typ, tokLoc)
	if err != nil {
		return nil, err
	}
	o := w.object(decl, td)
	attrs, head, err := w.attributes(o, tok, tokAt)
	if err != nil {
		return nil, err
	}
	if typ != nil {
		attrs = append([]validate.Attribute{typ}, attrs...)
	}
	e.attrs = attrs
	e.kids = &members{w: w, owner: e, object: o, head: &head}
	return e, nil
}

// lead reads the reserved members that lead e's object, in this order and
// each at most once: a "$xmlns", whose bindings become e's scope (xmlns), then
// a "$type", returned as e's xsi:type attribute, located at its key, whose
// [[normalized value]] is the member's string unchanged. It returns the token
// after them, a key or the object's close, and its location.
//
// Reading them first is what lets the members after them be classified under
// the type "$type" names, and "$type" resolve against the bindings "$xmlns"
// makes, without holding any of the document (STYLE P4). A "$type" before a
// "$xmlns", a repeat of either, and either after another member are left to
// classify, which declines them (walker.reserved): no spec text orders an
// element's attributes, so none of them is charged.
func (w *walker) lead(e *element) (*attribute, json.Token, xsderr.Loc, error) {
	tok, loc, err := w.token()
	if err != nil {
		return nil, nil, loc, unexpectedEOF(err, loc)
	}
	if tok == json.Token(memberXmlns) {
		bindings, err := w.xmlns(e.decl.Name())
		if err != nil {
			return nil, nil, loc, err
		}
		e.scope = &scope{parent: e.scope, bindings: bindings}
		if tok, loc, err = w.token(); err != nil {
			return nil, nil, loc, unexpectedEOF(err, loc)
		}
	}
	if tok != json.Token(memberType) {
		return nil, tok, loc, nil
	}
	vtok, vloc, err := w.token()
	if err != nil {
		return nil, nil, vloc, unexpectedEOF(err, vloc)
	}
	lexical, ok := vtok.(string)
	if !ok {
		return nil, nil, vloc, w.declinef(vloc, "the \"$type\" member of %s is not a string; its value is the lexical form of the element's xsi:type", e.decl.Name())
	}
	typ := &attribute{name: xsiType, value: lexical, loc: loc}
	if tok, loc, err = w.token(); err != nil {
		return nil, nil, loc, unexpectedEOF(err, loc)
	}
	return typ, tok, loc, nil
}

// xmlns reads the value of a "$xmlns" member of owner's object: an object
// whose every member binds its key, a prefix, to its string value, a namespace
// name. It declines any other value; a member whose value is not a string; a
// non-empty prefix that is no NCName, which no xmlns:prefix attribute can
// declare (Namespaces in XML [2] PrefixedAttName); a prefix bound twice; the
// prefixes "xml" and "xmlns", which Namespaces in XML §3 reserves, "xml" being
// bound already (element.LookupPrefix); and a non-empty prefix bound to the
// empty string, which would undeclare it. The empty prefix may be bound, and
// is shadowed: the element's default namespace stays its declaration's target
// namespace (package doc).
func (w *walker) xmlns(owner xsd.QName) ([]binding, error) {
	tok, loc, err := w.token()
	if err != nil {
		return nil, unexpectedEOF(err, loc)
	}
	if tok != json.Delim('{') {
		return nil, w.declinef(loc, "the \"$xmlns\" member of %s is not an object; its members bind prefixes to namespace names", owner)
	}
	var bindings []binding
	for {
		tok, loc, err := w.token()
		if err != nil {
			return nil, unexpectedEOF(err, loc)
		}
		prefix, ok := tok.(string)
		if !ok {
			return bindings, nil
		}
		vtok, vloc, err := w.token()
		if err != nil {
			return nil, unexpectedEOF(err, vloc)
		}
		uri, ok := vtok.(string)
		if !ok {
			return nil, w.declinef(vloc, "the \"$xmlns\" member of %s binds %q to a value that is not a string; a namespace name is a string", owner, prefix)
		}
		if prefix != "" && !xmlname.IsNCName(prefix) {
			return nil, w.declinef(loc, "the \"$xmlns\" member of %s binds %q, which is not an NCName; no xmlns:prefix attribute declares it", owner, prefix)
		}
		if prefix == "xml" || prefix == "xmlns" {
			return nil, w.declinef(loc, "the \"$xmlns\" member of %s binds the reserved prefix %q", owner, prefix)
		}
		if prefix != "" && uri == "" {
			return nil, w.declinef(vloc, "the \"$xmlns\" member of %s binds %q to the empty string; a prefix cannot be undeclared", owner, prefix)
		}
		if slices.ContainsFunc(bindings, func(b binding) bool { return b.prefix == prefix }) {
			return nil, w.declinef(loc, "the \"$xmlns\" member of %s binds %q twice", owner, prefix)
		}
		bindings = append(bindings, binding{prefix: prefix, uri: uri})
	}
}

// classifyingType is the type definition e's members are classified under,
// tokLoc locating its object: the declaration's {type definition}, which must
// resolve, or, where e carries an xsi:type typ whose lexical names a type
// (instanceType), that ·instance-specified type definition·. The second is
// taken whether or not it ·overrides· the declared one: where it does not, the
// engine charges cvc-elt clause 4 and assesses the members under the declared
// type, and they are still the members of the type "$type" names.
func (w *walker) classifyingType(e *element, typ *attribute, tokLoc xsderr.Loc) (xsd.TypeDefinition, error) {
	declared, ok := w.schema.ResolvedType(e.decl.TypeDefinition())
	if !ok {
		return nil, w.declinef(tokLoc, "the {type definition} of %s does not resolve; its members cannot be classified", e.decl.Name())
	}
	if typ == nil {
		return declared, nil
	}
	if instance, ok := w.instanceType(e, typ.value); ok {
		return instance, nil
	}
	return declared, nil
}

// instanceType is the top-level type definition an xsi:type of e whose value
// is lexical names, and false where it names none. The lexical is resolved
// against e's bindings (e.LookupPrefix, so an unprefixed name takes the
// declaration's target namespace) by [xmlqname.Resolve], the one encoding
// validate's walk resolves an xsi:type through too (cvc-resolve-instance,
// §3.17.6.3, there resolveInstanceQName; STYLE T4), and the name is looked up
// among the schema's top-level type definitions. Where it names none, the
// engine charges cvc-attribute (clause 3 or 5) and assesses e under its
// declared type.
func (w *walker) instanceType(e *element, lexical string) (xsd.TypeDefinition, bool) {
	space, local, ok := xmlqname.Resolve(lexical, e.LookupPrefix)
	if !ok {
		return nil, false
	}
	return w.schema.ResolvedType(xsd.TypeDefinitionRef{Name: xsd.QName{Space: space, Local: local}})
}

// scalarLexical is the lexical form of a scalar token — a string's decoded
// characters; a number's source text, never a float64 round trip; true or
// false — and false for a token that is no scalar: null or a delimiter.
func scalarLexical(tok json.Token) (string, bool) {
	switch v := tok.(type) {
	case string:
		return v, true
	case json.Number:
		return string(v), true
	case bool:
		if v {
			return "true", true
		}
		return "false", true
	}
	return "", false
}

// scalar builds a scalar-valued element in the scope sc: lexical is its simple
// content. An empty lexical is no text at all, as <size/> carries none.
func (w *walker) scalar(decl xsd.ElementDeclaration, loc xsderr.Loc, lexical string, tokLoc xsderr.Loc, sc *scope) *element {
	kids := &scalarChildren{}
	if lexical != "" {
		kids.text = &text{data: lexical, loc: tokLoc}
	}
	return &element{w: w, decl: decl, loc: loc, scope: sc, kids: kids}
}

// nilled builds a null-valued element: one attribute, xsi:nil with the
// lexical form true, located at the null token tokLoc, and no [[children]]
// (package doc, "The mapping"). It reads no property of decl: whether the
// element is ·nilled· (key-nilled) and whether cvc-elt clause 3.1 or 3.2.3.2
// charges it are the engine's, as for the XML twin <e xsi:nil="true"/>. Where
// it is ·nilled·, the xsi:nil keeps it from cvc-elt clause 5.1's {value
// constraint}, which an empty element without one would take.
func (w *walker) nilled(decl xsd.ElementDeclaration, loc, tokLoc xsderr.Loc, sc *scope) *element {
	attrs := []validate.Attribute{&attribute{name: xsiNil, value: "true", loc: tokLoc}}
	return &element{w: w, decl: decl, loc: loc, attrs: attrs, scope: sc, kids: &scalarChildren{}}
}

// object is what an object-valued element of decl classifies its members
// against, from td, the type classifyingType chose: a complex type's
// contained declarations and {attribute uses}; a simple type has neither, so
// every member of its object declines and an empty object is an element with
// no children.
func (w *walker) object(decl xsd.ElementDeclaration, td xsd.TypeDefinition) *object {
	switch t := td.(type) {
	case xsd.ComplexType:
		_, simple := t.ContentType().(xsd.SimpleContent)
		return &object{owner: decl.Name(), contained: w.schema.ContainedElementDeclarations(t), uses: t.AttributeUses(), simpleContent: simple}
	case *xsd.SimpleType:
		return &object{owner: decl.Name()}
	}
	panic(fmt.Sprintf("jsonsrc: TypeDefinition arm %T", td))
}

// attributes reads the members of o's object that name attribute uses
// (classify), from tok at loc, the first after its reserved members, each an
// attribute of the element; and returns them with the token that ends the run
// — the first key naming a child, or the object's close — for the element's
// members cursor to start from.
//
// The engine reads an element's [[attributes]] at its entry, before its
// [[children]], and again after them, so they must be whole and unchanged
// from the first read. Reading the leading run as the element is built makes
// them so without holding any of the document (STYLE P4); an attribute member
// after a child member therefore declines, under members.key's GAP(jsonsrc)
// marker.
//
// Each attribute is named by its use's {attribute declaration}'s expanded name
// and located at its key. Its [[normalized value]] is its scalar's lexical
// form (scalarLexical) unchanged, and the empty string is an attribute whose
// value is empty. XML 1.0 §3.3.3 maps a literal #x9, #xA or #xD to #x20 and
// keeps one a character reference names; a JSON string holds none literally
// (RFC 8259 §7 requires a control character to be escaped), and its escape is
// read as the character reference would be. It declines a null, object or
// array value, and a key repeated in the run: an element carries one attribute
// of each expanded name (XML 1.0 §3.1, Unique Att Spec).
func (w *walker) attributes(o *object, tok json.Token, loc xsderr.Loc) ([]validate.Attribute, peeked, error) {
	var attrs []validate.Attribute
	for {
		key, ok := tok.(string)
		if !ok {
			return attrs, peeked{tok: tok, loc: loc}, nil
		}
		m, err := w.classify(o, key, loc)
		if err != nil {
			return nil, peeked{}, err
		}
		if m.use == nil {
			return attrs, peeked{tok: tok, loc: loc, member: m}, nil
		}
		name := m.use.DeclarationName()
		if slices.ContainsFunc(attrs, func(a validate.Attribute) bool { return a.Name() == name }) {
			return nil, peeked{}, w.declinef(loc, "attribute member %q repeats in the object of %s; an element carries one attribute of each expanded name", key, o.owner)
		}
		vtok, vloc, err := w.token()
		if err != nil {
			return nil, peeked{}, unexpectedEOF(err, vloc)
		}
		lexical, ok := scalarLexical(vtok)
		if !ok {
			return nil, peeked{}, w.declinef(vloc, "attribute member %q of %s is not a string, number or boolean; an attribute's value is a scalar", key, o.owner)
		}
		attrs = append(attrs, &attribute{name: name, value: lexical, loc: loc})
		if tok, loc, err = w.token(); err != nil {
			return nil, peeked{}, unexpectedEOF(err, loc)
		}
	}
}

// end reads what follows the root's value once the assessment returns: the
// rest of the root's subtree the assessment did not read, classified as the
// walk would have, then the top-level object's close and the end of the input.
func (w *walker) end(root *element) error {
	root.finish()
	if w.err != nil {
		return w.err
	}
	tok, loc, err := w.token()
	if err != nil {
		return unexpectedEOF(err, loc)
	}
	if key, ok := tok.(string); ok {
		return w.declinef(loc, "the top-level object has a second member %q; its one member names the validation root", key)
	}
	_, loc, err = w.token()
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	return w.declinef(loc, "content follows the top-level object")
}

// unexpectedEOF turns the io.EOF token reports at the end of the input into
// the malformed-JSON error it is anywhere a value or a close is still due,
// and passes any other error through.
func unexpectedEOF(err error, loc xsderr.Loc) error {
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("jsonsrc: %s: malformed JSON: %w", loc, io.ErrUnexpectedEOF)
	}
	return err
}
