package validate

import (
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCvcID is Validation Root Valid (ID/IDREF) (Structures §3.3.4.5, cvc-id).
// The clause charged goes in the message on ruleCvcElt's terms: the catalog
// carries the bare name, so "cvc-id.2" is not a valid [xsderr.Rule] — and the
// spec's own Note asks only that the two cases be told apart, "to ensure that
// distinct error codes are associated with these two cases".
const ruleCvcID xsderr.Rule = "cvc-id"

// This file assembles the [ID/IDREF table] of §3.17.5.2 as the walk goes and
// charges cvc-id (§3.3.4.5) against it once, at the ·validation root· and
// nowhere else. That restriction is the rule's own: cvc-elt clause 7 invokes it
// "if E is the validation root", so an interior element with a duplicate id
// below the root is the ROOT's violation and never its own.
//
// §3.17.5.2 builds one ID/IDREF binding per distinct string that is an ·ID
// value· or an ·IDREF value· in the ·actual value· of a member of the ·eligible
// item set·, whose [binding] is the set of elements that purport to HAVE that
// id. cvc-id then reads two shapes off the result: an empty [binding] is a
// reference to an id nothing declares (clause 1), and a [binding] with more
// than one member is an id two elements both claim (clause 2, c-uba).
//
// The [binding] is a set of ELEMENTS and not of items, which decides two things
// this file would otherwise get wrong. An ID-typed ATTRIBUTE binds its owner;
// an ID-typed ELEMENT binds its PARENT — §3.17.5.2 asks for an element "which
// has an attribute in its [[attributes]] or an element in its [[children]]"
// carrying the value — so a root element whose own content is an id has nothing
// to bind to and yields an entry with an empty [binding], which is the case the
// spec's own Note calls out as not useful. And two ID-typed items on ONE
// element contribute one member between them, not two, so clause 2 does not
// fire for them.

// idRole is what the ·governing type definition· of an item makes its ·actual
// value·: an id its element declares, a reference to one, or neither.
type idRole int

const (
	idRoleNone idRole = iota
	idRoleDeclare
	idRoleReference
)

// idTable is the ·validation root·'s [ID/IDREF table]: one entry per distinct
// id string, in the order the strings were first met, so several charges arrive
// in a fixed order (STYLE D2). The index is internal and read by string alone,
// never iterated into output.
//
// declined records that some item of the subtree could have been in the
// ·eligible item set· and could not be read — see idTable.charge for which of
// the two clauses that suppresses, and why it suppresses only one.
type idTable struct {
	entries  []*idEntry
	index    map[string]*idEntry
	declined bool
}

// idEntry is one ID/IDREF binding: the [id] string, the [binding] as the
// ordinals of the elements that purport to have it, and the two locations the
// charges cite — where the string was first seen at all, for clause 1's
// unresolved reference, and where the SECOND element claimed it, for clause 2's
// duplicate.
type idEntry struct {
	id      string
	binding []int
	first   xsderr.Loc
	dup     xsderr.Loc
}

// declare records that the element with ordinal node purports to have id. A
// node of -1 is an ID-typed element with no parent, which contributes the
// string to the table with nothing bound to it (see the file comment).
func (t *idTable) declare(id string, node int, loc xsderr.Loc) {
	e := t.entry(id, loc)
	for _, n := range e.binding {
		if n == node {
			return // one element, one member: [binding] is a SET of elements
		}
	}
	if node < 0 {
		return
	}
	e.binding = append(e.binding, node)
	if len(e.binding) == 2 {
		e.dup = loc
	}
}

// reference records that some item's ·actual value· names id, which puts the
// string in the table whether or not anything declares it.
func (t *idTable) reference(id string, loc xsderr.Loc) {
	t.entry(id, loc)
}

// entry returns the table's entry for id, adding it in first-seen order.
func (t *idTable) entry(id string, loc xsderr.Loc) *idEntry {
	if e, found := t.index[id]; found {
		return e
	}
	if t.index == nil {
		t.index = make(map[string]*idEntry)
	}
	e := &idEntry{id: id, first: loc}
	t.index[id] = e
	t.entries = append(t.entries, e)
	return e
}

// charge settles cvc-id against the assembled table, at the ·validation root·
// alone. Entries are read in first-seen order, so the charges are ordered by
// where each id first appeared in the document.
//
// Clause 2 is charged whether or not something declined, and clause 1 is not.
// The asymmetry is the direction each clause fails in under a missing item: an
// item this package could not read can only ADD entries and members, so it can
// turn a one-member [binding] into a two-member one — clause 2 stays true once
// observed — while an unobserved DECLARATION is exactly what would turn a
// clause 1 charge into no violation at all. Charging clause 1 off an incomplete
// table would reject a document for a gap in the processor.
func (t *idTable) charge(w *walk, root Element) {
	if w.res.err != nil {
		// A walk that stopped on a source fault assembled a table over part of
		// a document, and the part it never reached is where the missing
		// declaration would be (walk.element).
		return
	}
	for _, e := range t.entries {
		if len(e.binding) > 1 {
			w.res.violations = append(w.res.violations, xsderr.New(ruleCvcID, e.dup,
				"the id %q is claimed by more than one element information item of the sub-tree rooted at the ·validation root· %s, so its ID/IDREF binding has more than one member, which cvc-id clause 2 forbids",
				e.id, root.Name()))
			continue
		}
		if len(e.binding) == 0 && !t.declined {
			w.res.violations = append(w.res.violations, xsderr.New(ruleCvcID, e.first,
				"the id %q is referred to but no element information item of the sub-tree rooted at the ·validation root· %s declares it, so its ID/IDREF binding is the empty set, which cvc-id clause 1 forbids",
				e.id, root.Name()))
		}
	}
}

// idAttributes reads the ·eligible item set·'s attribute half off one element:
// every attribute information item whose ·governing type definition· this
// package can name, which is walk.attributeType's job.
//
// An attribute that matches no attribute use and resolves to no top-level
// declaration is not a decline where the element's ·governing type definition·
// was determinable AND folded: it is then ·laxly assessed· against nothing, so
// it has no ·governing type definition· and §3.17.5.2 clause 3 excludes it from
// the ·eligible item set· outright.
//
// GAP(xsd): where the governing type is complex and its {attribute uses} were
// NOT folded, the same attribute may match a use this package cannot see
// (assess.go's attributePropertiesFolded, #414), so the type read off a
// top-level declaration — or the absence of one — is not the governing type,
// and it declines instead.
func (w *walk) idAttributes(c *icCheck) {
	ct := c.g.complexType()
	folded := ct != nil && attributePropertiesFolded(*ct)
	attrs := c.e.Attributes()
	for _, a := range attrs {
		st, typed := w.attributeType(c.e, c.g, a)
		if typed {
			w.idRecord(st, a.Value(), c.e, c.node, a.Loc())
			continue
		}
		if ct != nil && !folded {
			w.ids.declined = true
		}
	}
	if folded {
		w.idDefaultedAttributes(c, attrs, *ct)
	}
}

// idDefaultedAttributes declines for a ·defaulted attribute· whose declaration
// is ID-governed.
//
// GAP(validate): §3.17.5.2's own Note puts one in the ·eligible item set· —
// "the use of [schema actual value] ... means that default or fixed value
// constraints may play a part" — because cvc-complex-type clause 4 supplies the
// item and its ·actual value· is the constraint's. This package does not
// synthesize the item, so the id it would declare is one cvc-id never saw, and
// clause 1 would charge an empty binding for it (#774).
func (w *walk) idDefaultedAttributes(c *icCheck, attrs []Attribute, ct xsd.ComplexType) {
	for _, u := range ct.AttributeUses() {
		if u.Required() || hasAttributeNamed(attrs, u.DeclarationName()) {
			continue
		}
		if _, constrained := w.schema.EffectiveValueConstraint(u); !constrained {
			continue
		}
		d, resolved := w.schema.ResolvedAttributeDeclaration(u)
		if !resolved {
			w.ids.declined = true
			continue
		}
		st, simple := w.schema.ResolvedSimpleType(d.TypeDefinition())
		if !simple {
			w.ids.declined = true
			continue
		}
		role, _, decided := w.idRole(st)
		if !decided || role != idRoleNone {
			w.ids.declined = true
		}
	}
}

// idElement reads the ·eligible item set·'s element half off one element, once
// its [[children]] are exhausted and its ·initial value· is complete. The
// declaring node is the element's PARENT, per §3.17.5.2's [binding], and -1
// where it has none.
//
// GAP(validate): three shapes decline instead — a declaration whose type was
// not determinable (a {type table}, an unresolvable {type definition}, an
// xsi:type whose ·override· could not be decided), an element that is ·nilled·,
// and an element with no character information item [[child]] whose declaration
// carries a {value constraint}, whose ·initial value· cvc-elt clause 5.1 takes
// from that constraint and not from the empty content.
//
// The middle one is a value the spec says IS ·absent· — §3.3.5.4 gives a
// ·nilled· element an absent [schema normalized value] and so an absent [schema
// actual value] — and it declines rather than recording that absence because
// recording it would let cvc-id clause 1 charge an empty binding on the
// strength of it, which is a widening of that clause and not a reading of this
// one. An element with NO ·governing element declaration· is not among the
// three: it is ·laxly assessed· against xs:anyType, whose complex {content
// type} is not derived from ID and so contributes nothing under clause 3.
func (w *walk) idElement(c *icCheck) {
	if c.g.hasDecl && c.g.typ == nil {
		w.ids.declined = true
		return
	}
	st := c.g.simpleType()
	if st == nil {
		return
	}
	if nilled(c.e, c.g) {
		w.ids.declined = true
		return
	}
	if c.initial.Len() == 0 && c.g.hasDecl {
		if _, constrained := c.g.decl.ValueConstraint(); constrained {
			w.ids.declined = true
			return
		}
	}
	node := -1
	if c.parent != nil {
		node = c.parent.node
	}
	w.idRecord(st, c.initial.String(), c.e, node, c.e.Loc())
}

// idRecord adds whatever one item contributes to the table. node is the element
// the item's ·ID values· bind to (its owner for an attribute, its parent for an
// element); it is unread for a reference.
//
// The value is gated on String Valid succeeding, which is what makes the
// [schema actual value] non-absent as clause 2 of the ·eligible item set·
// requires: a lexical outside the type's lexical space has no ·actual value· to
// be an ·ID value·, and its own invalidity is cvc-attribute's or
// cvc-complex-type's to charge.
//
// GAP(validate): a value.ValidateLexical error that is not a VERDICT
// (value.IsDatatypeVerdict) declines, on cvcattribute.go's terms — an ungoverned
// type reports under cvc-datatype-valid exactly as a genuine rejection does, and
// reading one as "no id here" would hide a declaration clause 1 charges for the
// absence of (#774).
func (w *walk) idRecord(st *xsd.SimpleType, lexical string, owner Element, node int, loc xsderr.Loc) {
	role, list, decided := w.idRole(st)
	if !decided {
		w.ids.declined = true
		return
	}
	if role == idRoleNone {
		return
	}
	if _, err := value.ValidateLexical(w.backend, w.schema, st, lexical, elementContext{owner: owner}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			w.ids.declined = true
		}
		return
	}
	for _, v := range idValues(lexical, list) {
		if role == idRoleDeclare {
			w.ids.declare(v, node, loc)
			continue
		}
		w.ids.reference(v, loc)
	}
}

// idValues is the ·ID values· or ·IDREF values· in one item's ·actual value·:
// every item of a list, or the one value of an atomic type.
//
// The values are read off the COLLAPSED lexical rather than out of the parsed
// value, which is exact for these three types and for every type derived from
// them: ID, IDREF and the item type of IDREFS all derive from xs:NCName, whose
// ·value space· is its ·lexical space· (Datatypes §3.4.7) under the collapse
// whiteSpace its ancestor xs:token fixes. idRecord has already run String Valid
// over the lexical, so a value reaching here is one that mapping accepted.
func idValues(lexical string, list bool) []string {
	fields := strings.Fields(lexical)
	if list || len(fields) < 2 {
		return fields
	}
	return fields[:1]
}

// idRole classifies one ·governing type definition· under clause 3 of the
// ·eligible item set·: the built-in ID, IDREF or IDREFS, "or a type definition
// ·derived· or ·constructed· directly or indirectly from any of these". The
// derivation walk is up {base type definition}s; the construction one is into a
// list's {item type definition}, which is how a user-defined list of IDREF —
// and IDREFS itself, were it not named — reaches the set.
//
// list distinguishes the two readings of the ·actual value·: a list's holds one
// ·IDREF value· per item, an atomic type's exactly one.
//
// GAP(validate): a UNION anywhere in the closure is undecided rather than
// classified none. Which member governs the value is decided by String Valid's
// own member scan, which this classification does not run, and reading a union
// with an IDREF member as "no id here" would drop an id the document really
// declares. The undecided answer's consumers are idRecord's two callers,
// idAttributes and idElement, both of which route it to idTable.declined, whose
// one reader is idTable.charge's clause 1 arm — so it suppresses a rejection and
// manufactures none. No issue owns its retirement yet.
func (w *walk) idRole(st *xsd.SimpleType) (role idRole, list, decided bool) {
	if st == nil {
		return idRoleNone, false, true
	}
	for t := st; t != nil; {
		switch t.Name() {
		case idName:
			return idRoleDeclare, false, true
		case idrefName:
			return idRoleReference, false, true
		case idrefsName:
			return idRoleReference, true, true
		}
		if t.IsAnySimpleType() {
			break
		}
		base, err := t.Base(w.schema)
		if err != nil {
			return idRoleNone, false, false
		}
		if base == t {
			break
		}
		t = base
	}
	variety, err := st.Variety(w.schema)
	if err != nil {
		return idRoleNone, false, false
	}
	switch variety.(type) {
	case xsd.List:
		item, err := st.Item(w.schema)
		if err != nil || item == nil {
			return idRoleNone, false, false
		}
		role, _, decided = w.idRole(item)
		return role, true, decided
	case xsd.Union:
		return idRoleNone, false, false
	}
	return idRoleNone, false, true
}

// The three built-in types §3.17.5.2 clause 3 names by hand.
var (
	idName     = xsd.QName{Space: xsd.XMLSchemaNS, Local: "ID"}
	idrefName  = xsd.QName{Space: xsd.XMLSchemaNS, Local: "IDREF"}
	idrefsName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "IDREFS"}
)

// attributeType is the ·governing type definition· of one attribute
// information item, narrowed to the simple type its ·actual value· is read
// under. It is the one reading both cvc-id and a `@NameTest` identity-constraint
// field need, and neither re-derives it (STYLE T4).
//
// The two arms are cvc-complex-type clause 2's own: an attribute matching an
// attribute use of the element's ·governing type definition· is governed by
// that use's {attribute declaration}, and one matching none is admitted (if at
// all) by an {attribute wildcard}, whose strict and lax {process contents} both
// ·resolve· the attribute's ·expanded name· among the top-level attribute
// declarations (§3.10.4.1). A name that resolves to nothing under either arm
// leaves the attribute with no governing type, which is reported false.
//
// The wildcard arm is taken WITHOUT checking that a wildcard is present, and
// that is deliberate rather than an omission: an attribute matching neither a
// use nor a wildcard already violates cvc-complex-type clause 2, which
// [walk.unmatchedAttribute] charges in its own right, and the type this reads
// off a top-level declaration is the type that attribute would have been
// assessed against wherever it is admitted at all.
//
// GAP(xsd): a governing type whose {attribute uses} are NOT folded (assess.go's
// attributePropertiesFolded, #414) reports false for EVERY attribute, whether
// or not a top-level declaration of the name exists. The wildcard arm is no
// fallback for a use arm that cannot be decided: that declaration is a
// DIFFERENT component from the unseen use's, so a lookup succeeding there
// governs the attribute by whichever type the schema happens to ALSO declare at
// the top level. Both callers turn the false into their own decline
// ([walk.idAttributes], [icCheck.fieldAttributes]).
func (w *walk) attributeType(e Element, g governance, a Attribute) (*xsd.SimpleType, bool) {
	if ct := g.complexType(); ct != nil {
		if !attributePropertiesFolded(*ct) {
			return nil, false
		}
		if u, matched := attributeUseNamed(ct.AttributeUses(), a.Name()); matched {
			d, resolved := w.schema.ResolvedAttributeDeclaration(u)
			if !resolved {
				return nil, false
			}
			return w.schema.ResolvedSimpleType(d.TypeDefinition())
		}
	}
	d, found := w.schema.Attribute(a.Name())
	if !found {
		return nil, false
	}
	return w.schema.ResolvedSimpleType(d.TypeDefinition())
}
