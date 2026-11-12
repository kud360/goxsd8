package validate

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is §3.3.5.6's [inherited attributes] (e-inherited_attributes) and
// nothing else. Its one reader is the XDM instance a Type Alternative's {test}
// evaluates against, §3.12.4 key-cta-ta-select clause 1.1.3 (ctaAttributes,
// cta.go).

// suppliedAttribute is a member of an element's [inherited attributes] that no
// source presents: a ·defaulted attribute· (§3.4.5.1) of an ancestor, its
// ·expanded name· and [[normalized value]], which is all clause 1.1.3's copy
// reads, located at its [[owner element]], which carries no attribute item for
// it. The owner itself is not kept: e-inherited_attributes clause 2 is the only
// rule that reads it, and [walk.handDown] settles that clause by the order it
// composes in, while clause 1.1.3 gives the copy E as its owner.
type suppliedAttribute struct {
	name  xsd.QName
	value string
	loc   xsderr.Loc
}

func (a suppliedAttribute) Name() xsd.QName { return a.name }
func (a suppliedAttribute) Value() string   { return a.value }
func (a suppliedAttribute) Loc() xsderr.Loc { return a.loc }

// HandedDown is the [inherited attributes] (§3.3.5.6, e-inherited_attributes)
// of every element child of an element E whose ·governing type definition· is
// governing, whose [attributes] are attributes, in source order, and whose own
// [inherited attributes] are inherited, nil where E is a validation root: the
// inherited argument [Validator.ConditionallySelected] takes for each child.
// It is the composition [Validator.Assess] makes for E, in this order:
//
//  1. Each of attributes that is ·potentially inherited· (key-p-inherited):
//     the {inheritable} of the attribute use of governing whose ·expanded
//     name· it has (clause 3.1, as [xsd.Schema.ResolvedInheritable] answers
//     it); not inheritable where it matches no use and a skip {attribute
//     wildcard} of governing ·skips· it (key-governing-ad clause 3); otherwise
//     the {inheritable} of the top-level attribute declaration its name
//     ·resolves· to (clause 3.2), not inheritable where it resolves to none.
//  2. Each ·defaulted attribute· of E (key-dflt-att) whose use's {inheritable}
//     is true, in governing's {attribute uses} order, valued by the {lexical
//     form} of the use's ·effective value constraint· and located at at.
//  3. Each member of inherited whose ·expanded name· no member of 1 or 2 has.
//
// Computed top-down from the root, e-inherited_attributes clause 2 — of two
// same-named candidates, the one whose [[owner element]] is the other's
// descendant — is this one-level composition: E's own candidates are the
// nearest any child has, and every candidate from further up had its
// shadowing settled when inherited was composed for E. An attribute of E that
// is not ·potentially inherited· shadows nothing above it. key-p-inherited
// clause 2, the same [validation context], is the caller's to keep: every
// inherited handed here descends from one validation root's nil.
//
// governing nil is xs:anyType's answer, which has no {attribute uses} and a
// lax {attribute wildcard} (§3.4.7): every attribute is read by its top-level
// declaration, as for a ·laxly assessed· E. The members of 1 and 3 are the
// caller's own values, and at, E's own location, is the Loc of each member of
// 2. HandedDown neither modifies nor retains its arguments, and returns
// inherited itself where E hands down nothing of its own.
//
// The property exists only for a child that is not attributed to a skip
// Wildcard, of a parent that is not ·skipped·: a caller asks this for no
// other.
func (v *Validator) HandedDown(governing xsd.TypeDefinition, at xsderr.Loc, attributes, inherited []Attribute) []Attribute {
	w := walk{schema: v.schema}
	return w.handDown(governing, at, attributes, inherited)
}

// handedDown is [Validator.HandedDown] for e, walked under g, whose own
// [inherited attributes] are inherited; it reads only g's type.
//
// An e whose ·governing type definition· this package could not determine
// hands down a set read off the top-level declarations alone, which a use of
// its real type might contradict. No reader sees it: every element [[child]]
// of such an e, ·nilled· or not, is [walk.child]'s undecided shape
// ([contentCheck.element]), which consults no {type table} and hands its own
// [[children]] the same undecided shape, so no element below e is typed (#2211).
// A ·laxly assessed· e hands down the same top-level reading, and there it is
// the spec's: xs:anyType's (HandedDown's nil governing), and the {type table}
// of a child that ·resolves· reads it ([walk.child]). Nothing here tests for a
// child attributed to a skip Wildcard or a ·skipped· parent, because neither
// can arise: [walk.child] stops before [walk.element] for such a child, and a
// ·skipped· parent is never walked at all.
func (w *walk) handedDown(e Element, g governance, inherited []Attribute) []Attribute {
	return w.handDown(g.typ, e.Loc(), e.Attributes(), inherited)
}

// handDown is the one body of [Validator.HandedDown] and [walk.handedDown],
// for an element governed by typ. It reads nothing off w but w.schema, which
// is what lets HandedDown run it on a walk holding the schema alone.
func (w *walk) handDown(typ xsd.TypeDefinition, at xsderr.Loc, attrs, inherited []Attribute) []Attribute {
	g := governance{typ: typ}
	var own []Attribute
	for _, a := range attrs {
		if w.inheritable(g, a) {
			own = append(own, a)
		}
	}
	// key-p-inherited counts an attribute "defaulted as described in Attribute
	// Default Value (§3.4.5.1)" alongside a specified one, and a ·defaulted
	// attribute· is attributed to the use that supplies it, so clause 3.1 reads
	// that use's {inheritable}.
	if ct := g.complexType(); ct != nil {
		for _, u := range ct.AttributeUses() {
			if !w.schema.ResolvedInheritable(u) {
				continue
			}
			vc, defaulted := w.defaultedConstraint(u, attrs)
			if !defaulted {
				continue
			}
			own = append(own, suppliedAttribute{name: u.DeclarationName(), value: vc.LexicalForm(), loc: at})
		}
	}
	if len(own) == 0 {
		return inherited
	}
	nearest := len(own)
	for _, a := range inherited {
		if hasAttributeNamed(own[:nearest], a.Name()) {
			continue
		}
		own = append(own, a)
	}
	return own
}

// inheritable is key-p-inherited clause 3 for one attribute information item a
// of an element whose governance is g: the {inheritable} of the Attribute Use a
// is ·attributed to· (clause 3.1), or where it is attributed to none, the
// {inheritable} of its ·governing attribute declaration· (clause 3.2).
//
// The attribution is [walk.attributeType]'s (cvcid.go), arm for arm but one, and
// reads the same three encodings: attributeUseNamed for clause 2.1's match,
// skippedAttribute for the item key-governing-ad (§3.2.4.2) clause 3 leaves
// with no declaration, and the top-level resolution for every other. The two
// differ in what they read off the result — attributeType the {type
// definition} of the use's resolved declaration, this the use's {inheritable}
// ([xsd.Schema.ResolvedInheritable]), which clause 3.1 names and which may
// differ from its declaration's: a use's own inheritable attribute wins over
// the declaration's (§3.2.2.3 ref.att.local).
//
// The one arm this does not share is attributeType's decline for an element
// whose ·governing type definition· this package could not determine
// ([governance.typeUndetermined]): this reads such an element's attributes by
// ·expanded name·, although key-governing-ad clause 2's ·context-determined
// declaration·, from a use of the undetermined type, may come first. It needs
// no decline in the walk, whose result for such an element no {type table}
// reads: every child of it, a ·nilled· one's included, is walked undecided
// ([contentCheck.element], #2211). HandedDown is handed a type, never an
// undetermined one.
func (w *walk) inheritable(g governance, a Attribute) bool {
	if ct := g.complexType(); ct != nil {
		if u, matched := attributeUseNamed(ct.AttributeUses(), a.Name()); matched {
			return w.schema.ResolvedInheritable(u)
		}
	}
	if w.skippedAttribute(g, a) {
		return false
	}
	d, resolved := w.schema.Attribute(a.Name())
	return resolved && d.Inheritable()
}
