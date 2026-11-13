package codec

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsd"
)

// nillableSum is codegen's Nillable[T, A] over one nillable element
// declaration (goshape.Nillable), as Decode reads it (codegen/doc.go
// "# Element declarations"): attrs, the binding whose attribute fields and
// {attribute wildcard} field A holds, nil where A is struct{}, a T of simple
// type or a complex type codegen emits as a bare sealed interface; and xsiNil,
// the scalar xsi:nil is read as, of xs:boolean, the {type definition} of the
// built-in xsi:nil declaration (§3.2.7.2). attrs is read at check time only,
// so a binding whose attribute fields are bound after this sum, through a type
// that reaches itself, is read whole.
type nillableSum struct {
	p      *pipeline
	attrs  *binding
	xsiNil scalar
}

// booleanName is the expanded name of xs:boolean.
var booleanName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "boolean"}

// newNillableSum returns the nillableSum of a declaration whose one
// governing type's unit, if complex, is attrs where that is a *binding, read
// under p. xs:boolean that is no simple type of p's schema is a plain error:
// every parsed schema holds the built-ins.
func newNillableSum(p *pipeline, attrs *binding) (nillableSum, error) {
	td, ok := p.schema.Type(booleanName)
	st, simple := td.(*xsd.SimpleType)
	if !ok || !simple {
		return nillableSum{}, fmt.Errorf("codec: %s, the {type definition} of the built-in xsi:nil (§3.2.7.2), is no simple type of the schema", booleanName)
	}
	g, err := simpleDecline(p.schema, st, "attribute xsi:nil")
	if err != nil {
		return nillableSum{}, err
	}
	return nillableSum{p: p, attrs: attrs, xsiNil: newScalar(g, st, "attribute xsi:nil", xsd.ValueConstraint{}, false)}, nil
}

// attributesOf returns the binding whose attribute fields the A of
// Nillable over a's V holds: a structArm's unit, and nil for every other arm,
// whose A is struct{}. fieldArm is a closed sum, so the assertion is STYLE
// T2's sealed-sum exception.
func attributesOf(a fieldArm) *binding {
	s, isStruct := a.(structArm)
	if !isStruct {
		return nil
	}
	return s.unit
}

// check holds it, the Go type at the site site names, to codegen's
// Nillable[T, A], item 6 of Decode's doc: an interface whose one method is
// unexported, for which cx's sets hold exactly two struct arms, told apart by
// field name alone, never by position or type, since T and A may both be
// structs of no field: one of exactly the field Content, the other of exactly
// the field Attributes, whose type is A, a struct of exactly n's attribute
// fields and then its {attribute wildcard}'s, each as T's own is held
// (binding.check), or of no field for a nil n.attrs. It returns the arms bound;
// Content's type is its caller's to hold to T.
func (n nillableSum) check(it reflect.Type, site string, cx *checkContext) (boundNillable, error) {
	if !sealedShaped(it) {
		return boundNillable{}, fmt.Errorf("%w: %s is %s, which is not an interface of one unexported method, codegen's Nillable", ErrTypeMismatch, site, it)
	}
	arms, supplied := cx.sets.of(it)
	if !supplied {
		return boundNillable{}, fmt.Errorf("%w: no WithBranches supplies the arms of %s, codegen's Nillable at %s, where codegen emits 2, Nilled and Content", ErrTypeMismatch, it, site)
	}
	if len(arms) != 2 {
		return boundNillable{}, fmt.Errorf("%w: WithBranches supplies %d arms of %s where codegen emits Nillable's 2, Nilled and Content", ErrTypeMismatch, len(arms), it)
	}
	b := boundNillable{sum: n, t: it}
	for i, at := range arms {
		if at == nil {
			return boundNillable{}, fmt.Errorf("%w: WithBranches supplies a nil arm of %s", ErrTypeMismatch, it)
		}
		if slices.Contains(arms[:i], at) {
			return boundNillable{}, fmt.Errorf("%w: WithBranches supplies %s twice as an arm of %s", ErrTypeMismatch, at, it)
		}
		switch {
		case fieldsNamed(at, "Content") && b.content == nil:
			b.content = at
		case fieldsNamed(at, "Attributes") && b.nilled == nil:
			b.nilled = at
		default:
			return boundNillable{}, fmt.Errorf("%w: arm %s of %s is neither a struct of exactly the field Content nor one of exactly the field Attributes that an earlier arm is not, codegen's Content and Nilled", ErrTypeMismatch, at, it)
		}
	}
	if err := n.checkAttributes(b.nilled.Field(0).Type, cx); err != nil {
		return boundNillable{}, err
	}
	return b, nil
}

// checkAttributes holds a, the type of a Nilled arm's Attributes, to A.
func (n nillableSum) checkAttributes(a reflect.Type, cx *checkContext) error {
	want := 0
	if n.attrs != nil {
		want = len(n.attrs.attrs) + n.attrs.anyAttribute.width()
	}
	if a.Kind() != reflect.Struct || a.NumField() != want {
		return fmt.Errorf("%w: %s, the Attributes of a nilled element, is no struct of %d fields, the attribute fields codegen emits for it", ErrTypeMismatch, a, want)
	}
	if n.attrs == nil {
		return nil
	}
	for i, f := range n.attrs.attrs {
		if _, err := f.field().check(a, i, cx); err != nil {
			return err
		}
	}
	if n.attrs.anyAttribute != nil {
		return n.attrs.anyAttribute.check(a, len(n.attrs.attrs))
	}
	return nil
}

// fillAttributes sets a, a value of A, from start's attributes as
// binding.fillAttributes sets a struct's, or, for a nil n.attrs, returns
// noAttributes' error for an attribute outside the xsi namespace.
func (n nillableSum) fillAttributes(start *xmltree.StartElement, a reflect.Value) error {
	if n.attrs == nil {
		return noAttributes(start)
	}
	return n.attrs.fillAttributes(start, a, 0)
}

// boundNillable is a nillableSum whose check held: t, the interface, and its
// two arms, content, a struct of exactly Content, and nilled, a struct of
// exactly Attributes.
type boundNillable struct {
	sum             nillableSum
	t               reflect.Type
	content, nilled reflect.Type
}

// contentAt is the site of Content in b's content arm, to which its caller
// holds T as a field of codegen's struct is held.
func (b boundNillable) contentAt(arm fieldArm) fieldAt {
	return fieldAt{owner: b.content, index: 0, want: field{name: "Content", shape: shapeOne, arm: arm}}
}

// isNilled reports whether the element start is ·nilled· (§3.3.1 key-nilled):
// it carries xsi:nil whose ·actual value· is true, read as an xs:boolean
// value is read, under the backend's governing mapping (item 8 of Decode's
// doc where it fails), never compared to a literal. Its declaration is
// nillable, which its caller's binding holds. An element of a declaration
// that is not nillable carries no xsi:nil (cvc-elt clause 3.1), which
// validation charged first.
func (b boundNillable) isNilled(start *xmltree.StartElement) (bool, error) {
	for _, a := range start.Attributes() {
		if a.Name().Space() != xsd.XMLSchemaInstanceNS || a.Name().Local() != "nil" {
			continue
		}
		v, err := b.sum.xsiNil.decode(b.sum.p, a.Value(), attributeSource{a: a, owner: start})
		if err != nil {
			return false, err
		}
		return v.Bool(), nil
	}
	return false, nil
}

// nilledValue returns a new value of b.t holding the Nilled arm of the nilled
// element start, its Attributes filled as T's attribute fields are
// (binding.fillAttributes), defaults included, since cvc-complex-type clauses
// 2 to 4 hold for a nilled element and only clause 1 is guarded by "If E is
// not nilled", and no value constraint supplied to it (cvc-elt clause 5.1);
// then it reads r up to start's end tag. A character or element child there
// is a plain error, since validation charged it first (cvc-elt clause
// 3.2.3.1).
func (b boundNillable) nilledValue(r *xmltree.Reader, start *xmltree.StartElement) (reflect.Value, error) {
	arm := reflect.New(b.nilled).Elem()
	if err := b.sum.fillAttributes(start, arm.Field(0)); err != nil {
		return reflect.Value{}, err
	}
	for {
		tok, err := r.Token()
		if err != nil {
			return reflect.Value{}, err
		}
		switch tok.(type) {
		case *xmltree.EndElement:
			v := reflect.New(b.t).Elem()
			v.Set(arm)
			return v, nil
		case *xmltree.StartElement, *xmltree.CharData:
			return reflect.Value{}, fmt.Errorf("codec: %s: the nilled element %s has a character or element child (cvc-elt clause 3.2.3.1)", start.Loc(), expanded(start.Name()))
		}
	}
}

// contentValue returns a new value of b.t holding the Content arm, its
// Content v, stored as a field's value is (shape.store).
func (b boundNillable) contentValue(v reflect.Value) reflect.Value {
	arm := reflect.New(b.content).Elem()
	shapeOne.store(b.sum.p, arm.Field(0), v)
	out := reflect.New(b.t).Elem()
	out.Set(arm)
	return out
}

// nillableArm is a field's V where the field's element declaration is
// nillable and codegen wraps its field (goshape.Nillable; codegen/doc.go
// "# Element declarations"): I, codegen's Nillable[T, A], held to sum, its
// Content holding the V content, the arm the field would hold unwrapped,
// occurring once, never a pointer.
type nillableArm struct {
	content fieldArm
	sum     nillableSum
}

// bind holds v to a's Nillable and its Content to a.content, and returns the
// nillableReader that reads the field's element.
func (a nillableArm) bind(v reflect.Type, at fieldAt, cx *checkContext) (elementReader, error) {
	if !a.admits(v) {
		return nil, at.mismatch()
	}
	b, err := a.sum.check(v, at.site(), cx)
	if err != nil {
		return nil, err
	}
	content, err := a.content.bind(b.content.Field(0).Type, b.contentAt(a.content), cx)
	if err != nil {
		return nil, err
	}
	return nillableReader{bound: b, content: content}, nil
}

// admits reports whether v is an interface, the kind of every Nillable.
func (nillableArm) admits(v reflect.Type) bool { return v.Kind() == reflect.Interface }

// describe spells V.
func (a nillableArm) describe() string {
	return "I for codegen's Nillable whose Content is " + a.content.describe()
}

// nillableReader reads a nillable element into a new value of its
// interface: the Nilled arm for a nilled one (boundNillable.nilledValue), and
// otherwise the Content arm holding what content reads from it.
type nillableReader struct {
	bound   boundNillable
	content elementReader
}

// read returns the element start's value, st passed to content.
func (n nillableReader) read(p *pipeline, r *xmltree.Reader, start *xmltree.StartElement, st *fillState) (reflect.Value, error) {
	nilled, err := n.bound.isNilled(start)
	if err != nil {
		return reflect.Value{}, err
	}
	if nilled {
		return n.bound.nilledValue(r, start)
	}
	v, err := n.content.read(p, r, start, st)
	if err != nil {
		return reflect.Value{}, err
	}
	return n.bound.contentValue(v), nil
}

// nillableRoot is the T of a document element whose declaration codegen
// wraps (goshape.Nillable): Nillable over R, the type codegen emits for the
// declaration, which content binds as T itself where the declaration is not
// nillable, held to sum.
type nillableRoot struct {
	content binder
	sum     nillableSum
}

// check holds t to the Nillable and its Content's type to content, and returns
// the filler that fills t.
func (n *nillableRoot) check(t reflect.Type, cx *checkContext) (filler, error) {
	b, err := n.sum.check(t, "T", cx)
	if err != nil {
		return nil, err
	}
	f, err := n.content.check(b.content.Field(0).Type, cx)
	if err != nil {
		return nil, err
	}
	return boundNillableRoot{bound: b, content: f}, nil
}

// boundNillableRoot is a nillableRoot whose check held, content R's filler.
type boundNillableRoot struct {
	bound   boundNillable
	content filler
}

// fill sets out as a filler does: root's xsi:type declines first, item 7 of
// Decode's doc, then its xsi:nil is read, and out holds the Nilled arm of a
// nilled root or the Content arm holding R as content fills it.
func (b boundNillableRoot) fill(r *xmltree.Reader, root *xmltree.StartElement, out reflect.Value) error {
	if err := checkXsiType(root); err != nil {
		return err
	}
	nilled, err := b.bound.isNilled(root)
	if err != nil {
		return err
	}
	if nilled {
		v, err := b.bound.nilledValue(r, root)
		if err != nil {
			return err
		}
		out.Set(v)
		return nil
	}
	v := reflect.New(b.bound.content.Field(0).Type).Elem()
	if err := b.content.fill(r, root, v); err != nil {
		return err
	}
	out.Set(b.bound.contentValue(v))
	return nil
}
