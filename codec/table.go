package codec

import (
	"fmt"
	"reflect"

	"github.com/kud360/goxsd8/internal/goname"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// tableRoot is the sealed interface codegen emits over the governing types of
// an element declaration with a {type table} that has zero or two or more of
// them (codegen/doc.go "# Element declarations"), as Decode reads it: table,
// the declaration's {type table}; types, its governing types in slot order;
// and sealed, one branch per type in that order, each holding the one field
// typedField gives an element of that type occurring once (branchField).
type tableRoot struct {
	table  xsd.TypeTable
	types  []goshape.GoverningType
	sealed *sealed[field]
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
	table, _ := decl.TypeTable()
	t := &tableRoot{table: table, types: types, sealed: &sealed[field]{p: run.p}}
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

// fill sets out, a value of the interface, as a filler does: root's xsi:type
// declines first, item 7 of Decode's doc, and then out holds the branch of
// the governing type the validator selects for root (tableRoot.selected), its
// one field read from root as an element of that type occurring once.
func (b *boundTable) fill(r *xmltree.Reader, root *xmltree.StartElement, out reflect.Value) error {
	if err := checkXsiType(root); err != nil {
		return err
	}
	i, err := b.root.selected(b.bound.p, root)
	if err != nil {
		return err
	}
	return b.bound.set(r, root, i, 0, out, &fillState{})
}

// selected returns the index in t.types of the type the {type table} t.table
// ·conditionally selects· for the element start (§3.3.4.1 key-cta-select):
// validate.Validator.ConditionallySelectedIndex's answer, the selection Assess
// made, by position, for start's [attributes] as validate/xmlsrc presents them
// and p.inherited, start's [inherited attributes], mapped to its slot
// (goshape.TableSlot), resolved and matched to its branch (tableRoot.matched).
// Decode validated the instance first, and Assess records a withheld selection
// as undecided (item 4 of Decode's doc), so a withheld one, a position no slot
// holds and a slot that does not resolve are each a plain error naming start's
// instance Loc and key-cta-ta-select, as tableRoot.matched's answer of no
// branch is, a defect of Decode's: no fallback to the declared or default type
// is ever taken.
func (t *tableRoot) selected(p *pipeline, start *xmltree.StartElement) (int, error) {
	name := expanded(start.Name())
	index, decided := p.validator.ConditionallySelectedIndex(t.table, infosetAttributes(start), p.inherited)
	if !decided {
		return 0, fmt.Errorf("codec: %s: element %s: its {type table} withholds the selection (key-cta-ta-select) of an instance validate did not leave undecided", start.Loc(), name)
	}
	slot, ref, mapped := goshape.TableSlot(t.table, index)
	td, resolved := p.schema.ResolvedType(ref)
	if !mapped || !resolved {
		return 0, fmt.Errorf("codec: %s: element %s: the type its {type table} selects (key-cta-ta-select) does not resolve", start.Loc(), name)
	}
	return t.matched(start, slot, td)
}

// matched returns the index in t.types of td, the type the {type table}
// t.table ·conditionally selects· for the element start, held at slot, as
// codegen's identity holds types one (goshape.GoverningTypes) and as Decode's
// doc matches them: for a named td, the first type named as td is, which holds
// a type a later slot holds too, and for an anonymous td, the type whose Slot
// is slot (validate.Validator.ConditionallySelectedIndex's doc says why). No
// type is compared to td by value. A td that is none of t.types,
// xs:error's, which GoverningTypes drops, is a plain error naming start's
// instance Loc and pointing at key-cta-ta-select.
func (t *tableRoot) matched(start *xmltree.StartElement, slot goshape.Slot, td xsd.TypeDefinition) (int, error) {
	name := td.Name()
	anonymous := name == (xsd.QName{})
	for i, g := range t.types {
		if anonymous && g.Slot() == slot || !anonymous && g.Type().Name() == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("codec: %s: element %s: the type its {type table} selects (key-cta-ta-select) at its %s is none of its governing types", start.Loc(), expanded(start.Name()), slot.Name())
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
