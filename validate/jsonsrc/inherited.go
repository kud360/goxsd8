package jsonsrc

import (
	"slices"

	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is §3.3.5.6's [inherited attributes] (e-inherited_attributes) as
// the mapping determines them before the engine sees the infoset: its one
// reader is the selection of a {type table} below the root
// (walker.selector, walker.attributes), whose {test}s read them beside an
// element's own [attributes] (key-cta-ta-select clause 1.1.3).

// handDown is the [inherited attributes] of every element child of the
// element at loc whose [attributes] are attrs, whose members are classified
// against o, and whose own [inherited attributes] are inherited, where known
// says the mapping determines those; and whether the mapping determines the
// result.
//
// It determines them where every one of o.governing, the types that may
// govern the element, hands down the same ones under
// validate.Validator.HandedDown (walker.handed), so that the one the engine
// determines does too; and not where o.governing is empty or inherited is not
// known. An element carrying a "$type" hands down under the types
// walker.instanceGoverning finds may govern it, or walker.overrode and
// walker.governOverriding settle once its attributes are read, and not under
// the one its members are classified under before them
// (walker.instanceGoverned), which is the type the "$type" names, standing
// for several, where its declaration's {type table} can select types that
// settle differently.
//
// key-p-inherited clause 2, the same [validation context], holds for every
// pair this is asked of: one mapping builds one infoset under one ·validation
// root·.
func (w *walker) handDown(attrs []validate.Attribute, loc xsderr.Loc, o *object, inherited []validate.Attribute, known bool) ([]validate.Attribute, bool) {
	if !known || len(o.governing) == 0 {
		return nil, false
	}
	handed := w.handed(o.governing[0].typ, loc, attrs, inherited)
	for _, gt := range o.governing[1:] {
		if !slices.EqualFunc(handed, w.handed(gt.typ, loc, attrs, inherited), sameAttribute) {
			return nil, false
		}
	}
	return handed, true
}

// sameAttribute reports whether a and b are of one ·expanded name· and one
// value, all a {test} reads of an inherited attribute.
func sameAttribute(a, b validate.Attribute) bool {
	return a.Name() == b.Name() && a.Value() == b.Value()
}
