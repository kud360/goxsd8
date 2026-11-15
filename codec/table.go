package codec

import (
	"fmt"
	"reflect"

	"github.com/kud360/goxsd8/internal/goname"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/internal/xmlqname"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// tableRoot is the sealed interface codegen emits over the governing types of
// an element declaration that has zero or two or more of them (codegen/doc.go
// "An element declaration's governing types"), as Decode reads it: selector,
// how an instance's branch is chosen; types, its governing types in slot
// order; and sealed, one branch per type in that order, each holding the one
// field typedField gives an element of that type occurring once
// (branchField).
type tableRoot struct {
	selector selector
	types    []goshape.GoverningType
	sealed   *sealed[field]
}

// A selector returns the index in t.types of the governing type of the
// element start, an instance of a declaration t is the tableRoot of, or a
// plain error naming start's instance Loc, a defect of Decode's, which
// validated the instance first. tableSelector is a {type table}'s and
// instanceSelector that of a declaration with none, whose governing types
// stand for its abstract {type definition}; bindTable picks by
// xsd.ElementDeclaration.TypeTable.
type selector interface {
	selected(t *tableRoot, p *pipeline, start *xmltree.StartElement) (int, error)
}

// table returns the tableRoot of el, an element declaration whose governing
// types, types, are zero or two or more, keyed as codegen emits its interface:
// by el's expanded name for a top-level el, whose interface every <element
// ref> particle to it shares, by its groupPath for a local el declared in a
// model group definition, and under no key for any other local one, bound in
// place inside the unit that declares it (bindRun).
func (run *bindRun) table(el xsd.ElementDeclaration, types []goshape.GoverningType) (complexUnit, error) {
	if el.ScopeVariety() == xsd.ScopeGlobal {
		return bindTable(run, el, types, run.elements, el.Name())
	}
	if group, inGroup := groupScope(el); inGroup {
		return bindTable(run, el, types, run.groups, groupPath{group: group, element: el.Name()})
	}
	return bindTable(run, el, types, unitIndex[xsd.QName](nil), xsd.QName{})
}

// bindTable returns the tableRoot of decl, whose governing types are types,
// the one index already holds for key where it holds one, or the first decline
// item 5 of Decode's doc states for it: each governing type's branch field's
// in slot order (branchField), then a branch field that may admit a Go type an
// earlier one's does (field.sharesBranch). The unit is stored before any
// branch is bound, so a type that reaches decl again links to it.
func bindTable[K comparable](run *bindRun, decl xsd.ElementDeclaration, types []goshape.GoverningType, index unitIndex[K], key K) (complexUnit, error) {
	if u, ok := index[key]; ok {
		return u, nil
	}
	var sel selector = instanceSelector{}
	if table, hasTable := decl.TypeTable(); hasTable {
		sel = tableSelector{table: table}
	}
	t := &tableRoot{selector: sel, types: types, sealed: &sealed[field]{p: run.p}}
	index.store(key, t)
	for _, g := range types {
		f, err := branchField(run, decl, g)
		if err != nil {
			return nil, err
		}
		t.sealed.alts = append(t.sealed.alts, f)
	}
	for i, f := range t.sealed.alts {
		for _, earlier := range t.sealed.alts[:i] {
			if earlier.sharesBranch(f) {
				return nil, decline(constructSharedBranch, declSubject(decl))
			}
		}
	}
	return t, nil
}

// branchField returns the one field of the branch struct of g, a governing
// type of decl: named from g's local name (goshape.GoverningType.Local),
// holding what typedField gives an element of g's type occurring once, the Go
// type codegen gives a simple type itself and, of a complex type, a named
// one's unit keyed by its QName and an anonymous one's bound in place under no
// key.
func branchField(run *bindRun, decl xsd.ElementDeclaration, g goshape.GoverningType) (field, error) {
	once := func(sealed bool) (shape, error) {
		return occursShape(goshape.ExactlyOnce(), sealed, g.Slot().Name()+" of "+declSubject(decl))
	}
	f, err := typedField(run, decl, g.Type(), valueBare, once, func(ct xsd.ComplexType) (complexUnit, error) {
		if ct.Name() != (xsd.QName{}) {
			return bindComplex(run, decl, ct, run.types, ct.Name())
		}
		return bindComplex(run, decl, ct, unitIndex[xsd.QName](nil), xsd.QName{})
	})
	if err != nil {
		return field{}, err
	}
	f.name = goname.NewNamer().Claim(g.Local())
	return f, nil
}

// field returns the field of shape s whose V is t's interface.
func (t *tableRoot) field(s shape) field {
	return field{shape: s, arm: ifaceArm{unit: t}}
}

// sealedInterface is true: t's type is a sealed interface.
func (t *tableRoot) sealedInterface() bool { return true }

// check holds rt, T itself, to t's interface, and returns the boundTable that
// fills it.
func (t *tableRoot) check(rt reflect.Type, cx *checkContext) (filler, error) {
	return t.checkAt(rt, "T", cx)
}

// checkAt holds rt, the Go type at the site site names, to t's interface, one
// branch struct per governing type (sealed.checkSite), and returns the
// boundTable that fills it, the one cx already bound for t and rt where it
// holds one.
func (t *tableRoot) checkAt(rt reflect.Type, site string, cx *checkContext) (filler, error) {
	key := boundKey{unit: t, t: rt}
	if f, ok := cx.bound[key]; ok {
		return f, nil
	}
	bt := &boundTable{root: t}
	cx.bound[key] = bt
	bc, err := t.sealed.checkSite(rt, site, cx)
	if err != nil {
		return nil, err
	}
	bt.bound = bc
	return bt, nil
}

// boundTable is root whose check held, bound its branches.
type boundTable struct {
	root  *tableRoot
	bound *boundChoice
}

// fill sets out, a value of the interface, as a filler does: out holds the
// branch of the governing type root.selector chooses for root, its one field
// read from root as an element of that type occurring once.
func (b *boundTable) fill(r *xmltree.Reader, root *xmltree.StartElement, out reflect.Value) error {
	i, err := b.root.selector.selected(b.root, b.bound.p, root)
	if err != nil {
		return err
	}
	return b.bound.set(r, root, i, 0, out, &fillState{})
}

// tableSelector selects by the {type table} table.
type tableSelector struct{ table xsd.TypeTable }

// selected returns the index in t.types of the type the {type table} s.table
// ·conditionally selects· for the element start (§3.3.4.1 key-cta-select):
// validate.Validator.ConditionallySelectedIndex's answer, the selection Assess
// made, by position, for start's [attributes] as validate/xmlsrc presents them
// and p.inherited, start's [inherited attributes], mapped to its slot
// (goshape.TableSlot), resolved and matched to its branch (tableRoot.matched).
// Decode validated the instance first, and Assess records a withheld selection
// as undecided (item 4 of Decode's doc), so a withheld one, a position no slot
// holds, a slot that does not resolve and a type matched to no branch (match)
// are each a plain error naming start's instance Loc and key-cta-ta-select, a
// defect of Decode's: no fallback to the declared or default type is ever
// taken.
func (s tableSelector) selected(t *tableRoot, p *pipeline, start *xmltree.StartElement) (int, error) {
	name := expanded(start.Name())
	index, decided := p.validator.ConditionallySelectedIndex(s.table, infosetAttributes(start), p.inherited)
	if !decided {
		return 0, fmt.Errorf("codec: %s: element %s: its {type table} withholds the selection (key-cta-ta-select) of an instance validate did not leave undecided", start.Loc(), name)
	}
	slot, ref, mapped := goshape.TableSlot(s.table, index)
	td, resolved := p.schema.ResolvedType(ref)
	if !mapped || !resolved {
		return 0, fmt.Errorf("codec: %s: element %s: the type its {type table} selects (key-cta-ta-select) does not resolve", start.Loc(), name)
	}
	return s.match(t, start, slot, td)
}

// match returns tableRoot.matched's index of td, the type s.table selects for
// the element start at slot, and in place of its false a plain error naming
// start's instance Loc and pointing at key-cta-ta-select.
func (tableSelector) match(t *tableRoot, start *xmltree.StartElement, slot goshape.Slot, td xsd.TypeDefinition) (int, error) {
	i, matched := t.matched(slot, td)
	if !matched {
		return 0, fmt.Errorf("codec: %s: element %s: the type its {type table} selects (key-cta-ta-select) at its %s is none of its governing types", start.Loc(), expanded(start.Name()), slot.Name())
	}
	return i, nil
}

// instanceSelector selects by the element's xsi:type.
type instanceSelector struct{}

// selected returns the index in t.types of the type the xsi:type of the
// element start names, which ·overrides· its declaration's abstract {type
// definition} (§3.3.4.3 cvc-elt clause 4): its value resolved against start's
// bindings (xmltree.StartElement.LookupPrefix) by xmlqname.Resolve, looked up
// among the schema's top-level type definitions, as validate's walk resolves
// it (cvc-resolve-instance), and matched to its branch by name
// (tableRoot.matched). Decode validated the instance first, and no instance
// of that {type definition} is valid without an xsi:type that overrides it
// (cvc-type clause 2), each such type being a member (codegen/doc.go "An
// element declaration's governing types"), so no xsi:type, one that does not
// resolve, and one naming a type no branch holds are each a plain error
// naming start's instance Loc and key-overrides, a defect of Decode's.
func (instanceSelector) selected(t *tableRoot, p *pipeline, start *xmltree.StartElement) (int, error) {
	name := expanded(start.Name())
	for _, a := range start.Attributes() {
		if a.Name().Space() != xsd.XMLSchemaInstanceNS || a.Name().Local() != "type" {
			continue
		}
		space, local, ok := xmlqname.Resolve(a.Value(), start.LookupPrefix)
		td, resolved := p.schema.ResolvedType(xsd.TypeDefinitionRef{Name: xsd.QName{Space: space, Local: local}})
		if !ok || !resolved {
			return 0, fmt.Errorf("codec: %s: element %s: its xsi:type %q names no type definition, but one overriding its {type definition} (key-overrides) is required", start.Loc(), name, a.Value())
		}
		i, matched := t.matched(goshape.DeclaredSlot{}, td)
		if !matched {
			return 0, fmt.Errorf("codec: %s: element %s: its xsi:type %q names %s, none of its governing types, the types that override its {type definition} (key-overrides)", start.Loc(), name, a.Value(), td.Name())
		}
		return i, nil
	}
	return 0, fmt.Errorf("codec: %s: element %s carries no xsi:type, but one overriding its {type definition} (key-overrides) is required", start.Loc(), name)
}

// matched returns the index in t.types of td, the type selected for an
// element at slot, as codegen's identity holds types one
// (goshape.GoverningTypes) and as Decode's doc matches them: for a named td,
// the first type named as td is, which holds a type a later slot holds too,
// and for an anonymous td, the type whose Slot is slot
// (validate.Validator.ConditionallySelectedIndex's doc says why), and false
// where td is none of t.types, xs:error's included, which GoverningTypes
// drops. No type is compared to td by value.
func (t *tableRoot) matched(slot goshape.Slot, td xsd.TypeDefinition) (int, bool) {
	name := td.Name()
	anonymous := name == (xsd.QName{})
	for i, g := range t.types {
		if anonymous && g.Slot() == slot || !anonymous && g.Type().Name() == name {
			return i, true
		}
	}
	return 0, false
}

// handDown sets p.inherited to the [inherited attributes] of the children of
// the element start, governed by typ, which validate.Validator.HandedDown
// composes from start's [attributes] and start's own, p.inherited, and
// returns start's own, which its caller restores once start's content is
// read.
func (p *pipeline) handDown(typ xsd.TypeDefinition, start *xmltree.StartElement) []validate.Attribute {
	own := p.inherited
	p.inherited = p.validator.HandedDown(typ, start.Loc(), infosetAttributes(start), own)
	return own
}

// infosetAttribute is an attribute of an instance element as validate/xmlsrc
// presents it to Assess: its expanded name, its value as parser/xmltree
// normalized it (XML 1.0 §3.3.3), and its Loc.
type infosetAttribute struct{ a xmltree.Attribute }

func (a infosetAttribute) Name() xsd.QName { return expanded(a.a.Name()) }
func (a infosetAttribute) Value() string   { return a.a.Value() }
func (a infosetAttribute) Loc() xsderr.Loc { return a.a.Loc() }

// infosetAttributes returns start's [attributes] in start-tag order, namespace
// declarations excluded, as validate/xmlsrc presents them to Assess
// (xmltree.StartElement.Attributes).
func infosetAttributes(start *xmltree.StartElement) []validate.Attribute {
	attrs := start.Attributes()
	out := make([]validate.Attribute, 0, len(attrs))
	for _, a := range attrs {
		out = append(out, infosetAttribute{a: a})
	}
	return out
}
