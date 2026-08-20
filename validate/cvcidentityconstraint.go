package validate

import (
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCvcIdentityConstraint is Identity-constraint Satisfied (Structures
// §3.11.4, cvc-identity-constraint). The clause charged goes in the message on
// ruleCvcElt's terms: the catalog carries the bare name, so
// "cvc-identity-constraint.4.2.1" is not a valid [xsderr.Rule].
const ruleCvcIdentityConstraint xsderr.Rule = "cvc-identity-constraint"

// This file decides cvc-identity-constraint (§3.11.4) for every element the
// walk assesses, and builds the [identity-constraint table] of §3.11.5 for
// every element it visits. The path evaluation the two rest on is icpath.go's.
//
// The whole design follows from one sentence of §3.11.5: an element's node
// tables are "assembled strictly recursively from the node tables of
// descendants". Node tables therefore propagate UPWARD and only upward
// (PRINCIPLES 15) — which is what makes clause 4.3's Note true, that "only
// element information items within the sub-tree rooted at the element
// information item being ·validated· can be referenced successfully", and what
// makes a keyref on E blind to a key sourced in a SIBLING's subtree.
//
// The walk is the recursion §3.11.5 asks for, so the state rides it rather than
// being rebuilt over a tree the engine never holds ([Children] is a pull
// cursor):
//
//   - ENTERING an element, [walk.identityCheck] advances every live selector
//     and field path one level ([icExpr.advance]), opens a frame for each
//     identity constraint its own ·governing element declaration· declares, and
//     registers the element as a ·target node· or as a field node wherever a
//     path completed.
//   - LEAVING it, [walk.identityExit] fills the field slots this element's own
//     ·initial value· supplies, settles clauses 3 and 4 for the frames rooted
//     here, merges its children's node tables with its own qualified entries
//     under §3.11.5's conflict resolution, settles clause 4.3 against the
//     result, and hands the merged table to its parent.
//
// Clause 2 — "each node in the ·target node set· is either the context node or
// an element node among its descendants" — is satisfied by construction and
// never charged: a selector cursor is seeded at the element the constraint is
// declared on and only ever advances DOWNWARD into that element's [[children]],
// so no path this file evaluates can reach a node outside the subtree.
//
// ·Skipped· nodes need no filtering either (clause 1's "after omitting all
// element nodes corresponding to element information items that are
// ·skipped·"): [walk.child] returns before [walk.element] for a child
// ·attributed to· a skip wildcard, so no icCheck is ever built for it or for
// anything beneath it, and its subtree contributes no target, no field value
// and no node-table entry.

// icCheck is the identity-constraint state of ONE element information item,
// built by [walk.identityCheck] as the walk enters it and settled by
// [walk.identityExit] as the walk leaves. It is the analogue of contentCheck
// for §3.11.4, and like it holds nothing that outlives its element — except
// table, which is the one thing §3.11.5 has travel upward.
//
// sels and flds are the cursors live AT this element, ready to advance into its
// [[children]]; frames are the constraints declared on this element, whose
// targets and node table close here. pending records the field slots this
// element's OWN ·initial value· fills, which is why gather exists: an element's
// character content is complete only once its [[children]] are exhausted, so a
// field node's value is recorded on the way out and not on the way in.
type icCheck struct {
	e      Element
	g      governance
	node   int
	parent *icCheck

	frames []*icFrame
	sels   []icSelCursor
	flds   []icFieldCursor

	pending []icPending
	gather  bool
	initial strings.Builder

	table icTable
}

// icPending is one field slot waiting on the ·initial value· of the element
// whose icCheck holds it.
type icPending struct {
	target *icTarget
	index  int
}

// icSelCursor is one constraint's {selector} evaluation, live at one element.
type icSelCursor struct {
	frame *icFrame
	live  icLive
}

// icFieldCursor is one ·target node·'s evaluation of one {fields} member, live
// at one element. index is that member's position in {fields}, which is also
// its position in the ·key-sequence· (§3.11.4 clause 3, "in the order of the
// {fields} property").
type icFieldCursor struct {
	target *icTarget
	index  int
	live   icLive
}

// icFrame is one identity constraint being evaluated against the element it is
// declared on — E, in §3.11.4's terms. Its targets accumulate as the descent
// finds them and are settled together when E's [[children]] are exhausted,
// because a ·target node·'s ·key-sequence· is complete only once its own
// subtree has been walked.
//
// declined marks a constraint whose {selector} or {fields} icpath.go could not
// compile. It charges nothing, and — through icTable's own declined flag — no
// keyref referring to it charges either, so a path this processor cannot read
// costs a rejection in neither direction.
type icFrame struct {
	ic       xsd.IdentityConstraint
	sel      icExpr
	fields   []icExpr
	targets  []*icTarget
	declined bool
}

// icTarget is one member of a constraint's ·target node set· (§3.11.4 clause
// 1), with one slot per {fields} member. node is the ordinal the walk assigns
// each element it visits, which is this package's node IDENTITY: §3.11.5's
// conflict resolution turns on "the same key-sequence but distinct nodes", and
// an [Element] is an interface an adapter implements, so == on one compares
// whatever the adapter's dynamic type compares.
type icTarget struct {
	frame *icFrame
	e     Element
	node  int
	slots []icSlot
}

// icSlot is what one {fields} member selected for one ·target node·.
//
// Clause 3 bounds the sequence a field evaluates to at "at most one node with a
// non-absent [schema actual value]", so the three states here are the whole of
// it: filled (exactly one), neither filled nor extra (none, which shortens the
// ·key-sequence· and drops the node out of the ·qualified node set·), and extra
// (more than one, which violates clause 3 outright).
//
// declined is not a fourth state of the rule but this processor's own: a field
// node whose ·governing type definition· could not be determined has no
// [schema actual value] to read (§3.3.5.4, which defines the property only "if
// and only if a governing type definition is known"), and a lexical comparison
// in its place is licensed nowhere.
type icSlot struct {
	filled   bool
	member   icKeyMember
	extra    bool
	declined bool
	loc      xsderr.Loc
}

// icKeyMember is one member of a ·key-sequence·: an ·actual value· in the value
// space its own type governs, never a lexical (Datatypes §2.2, and see
// [walk.sameKeyMember]).
//
// element and nillable travel with it for clause 4.2.3 alone, which asks
// whether an ELEMENT member "was assessed as ·valid· by reference to an element
// declaration whose {nillable} is true". The spec's own Note licenses reading
// the declaration's property directly where the PSVI contribution recording it
// is not provided, which this package does not provide.
type icKeyMember struct {
	st       *xsd.SimpleType
	v        value.Value
	element  bool
	nillable bool
}

// icKeySequence is the ·key-sequence· of one ·target node· (§3.11.4 clause 3),
// in {fields} order.
type icKeySequence []icKeyMember

// identityCheck opens the identity-constraint state of one element: the cursors
// its parent's live paths reach it with, and the frames its own ·governing
// element declaration· declares.
//
// The node ordinal is assigned here, once per element the walk enters, so it
// increases in document order and no two elements share one.
func (w *walk) identityCheck(e Element, g governance, parent *icCheck) *icCheck {
	w.nodes++
	c := &icCheck{e: e, g: g, node: w.nodes, parent: parent}
	if parent != nil {
		c.inherit(w, parent)
	}
	c.open(w, g)
	if candidate, decided := w.idCandidate(g.valueType()); !decided || candidate {
		// The ·initial value· of an element clause 3 of the ·eligible item set·
		// admits is read on the way out (cvcid.go's idElement), so its character
		// content has to be gathered on the way through. The gate is the
		// CANDIDACY one and never the classification, which needs that value to
		// run at all, and it gathers for an undecided answer too so that it is
		// never narrower than idRecord's own.
		c.gather = true
	}
	return c
}

// inherit advances every cursor live at the parent one level down, into c's
// element, and records what completed there: a {selector} path completing makes
// c's element a ·target node· of that frame, a {fields} path completing makes it
// (or one of its [[attributes]]) that target's field node.
func (c *icCheck) inherit(w *walk, parent *icCheck) {
	for _, cur := range parent.sels {
		live, selected, _ := cur.frame.sel.advance(cur.live, c.e.Name())
		c.sels = append(c.sels, icSelCursor{frame: cur.frame, live: live})
		if selected {
			c.addTarget(w, cur.frame)
		}
	}
	for _, cur := range parent.flds {
		x := cur.target.field(cur.index)
		live, selected, attrs := x.advance(cur.live, c.e.Name())
		c.flds = append(c.flds, icFieldCursor{target: cur.target, index: cur.index, live: live})
		if selected {
			c.pendElement(cur.target, cur.index)
		}
		if len(attrs) > 0 {
			c.fieldAttributes(w, cur.target, cur.index, attrs)
		}
	}
}

// open builds one frame per identity constraint of c's ·governing element
// declaration· — §3.11.4 quantifies over "an identity-constraint", and
// cvc-elt clause 6 applies it to each member of the declaration's
// {identity-constraint definitions}.
//
// An element with no ·governing element declaration· declares no constraint:
// its own subtree is still walked, and still propagates its descendants' node
// tables upward, because §3.11.5's clause 1 quantifies over the children's
// tables and not over anything the element itself declares.
func (c *icCheck) open(w *walk, g governance) {
	if !g.hasDecl {
		return
	}
	for _, ic := range g.decl.IdentityConstraints() {
		f := &icFrame{ic: ic}
		c.frames = append(c.frames, f)
		sel, ok := icCompile(ic.Selector(), false)
		if !ok {
			f.declined = true
			continue
		}
		for _, x := range ic.Fields() {
			fx, ok := icCompile(x, true)
			if !ok {
				f.declined = true
				break
			}
			f.fields = append(f.fields, fx)
		}
		if f.declined {
			continue
		}
		f.sel = sel
		c.sels = append(c.sels, icSelCursor{frame: f, live: sel.start()})
		if self, _ := sel.self(); self {
			c.addTarget(w, f)
		}
	}
}

// addTarget records c's element as a member of f's ·target node set· and opens
// one field cursor per {fields} member with that element as the context node,
// settling on the spot the two shapes advance never sees: a field selecting the
// target itself (`.`) and one selecting an attribute of it (`@id`).
func (c *icCheck) addTarget(w *walk, f *icFrame) {
	t := &icTarget{e: c.e, node: c.node, slots: make([]icSlot, len(f.fields)), frame: f}
	f.targets = append(f.targets, t)
	for i := range f.fields {
		c.flds = append(c.flds, icFieldCursor{target: t, index: i, live: f.fields[i].start()})
		self, attrs := f.fields[i].self()
		if self {
			c.pendElement(t, i)
		}
		if len(attrs) > 0 {
			c.fieldAttributes(w, t, i, attrs)
		}
	}
}

// pendElement records that c's own element is the field node for one slot. The
// value is read on the way out (icCheck.fill), so gather is set here to start
// collecting the character information item [[children]] the ·initial value· is
// the concatenation of.
func (c *icCheck) pendElement(t *icTarget, i int) {
	c.pending = append(c.pending, icPending{target: t, index: i})
	c.gather = true
}

// fieldAttributes settles one slot from c's element's [[attributes]], for the
// field paths ending in `@NameTest` that completed here. Every matching
// attribute is offered to the slot, not just the first: clause 3 bounds the
// sequence at one valued node, and a NameTest of `@*` or `@p:*` can select
// several, which the slot records as the clause 3 violation it is.
//
// The scan is over the ATTRIBUTES and not over the tests, so an attribute two
// branches of a union both select is offered once. An XPath union is a sequence
// of distinct NODES, so offering it twice would charge clause 3 for a field
// like `@id|@*` that selects exactly one.
//
// An attribute whose ·governing type definition· [walk.attributeType] could not
// name declines the slot rather than being skipped: the [schema actual value]
// clause 3 reads is that type's, and comparing the member under the wrong
// simple type decides clause 4.1 by the wrong ·value space·.
func (c *icCheck) fieldAttributes(w *walk, t *icTarget, i int, tests []icNameTest) {
	for _, a := range c.e.Attributes() {
		if !icMatchesAny(tests, a.Name()) {
			continue
		}
		st, typed := w.attributeType(c.e, c.g, a)
		if !typed {
			t.slots[i].declined = true
			continue
		}
		m, present, decided := w.keyMember(st, a.Value(), c.e, false, false)
		t.slots[i].record(m, present, decided, a.Loc())
	}
}

// icMatchesAny reports whether any of the NameTests admits n.
func icMatchesAny(tests []icNameTest, n xsd.QName) bool {
	for _, t := range tests {
		if t.matches(n) {
			return true
		}
	}
	return false
}

// text gathers one run of character information items into the ·initial value·
// a field node's [schema actual value] is read off, for an element some field
// path selected and for no other (see pendElement). An element no field selects
// holds nothing here.
func (c *icCheck) text(t Text) {
	if !c.gather {
		return
	}
	c.initial.WriteString(t.Data())
}

// identityExit settles everything about one element that only its exhausted
// [[children]] can settle, in the order §3.11.5 and §3.11.4 depend on:
//
//  1. the field slots this element's own ·initial value· fills;
//  2. clauses 3 and 4.1/4.2 for the frames rooted here, which is also what
//     produces this element's clause 2 entries (§3.11.5's c-kc);
//  3. the merge of those with the children's entries, under §3.11.5's conflict
//     resolution;
//  4. clause 4.3 for the keyref frames rooted here, which reads the table step
//     3 just finished — §3.11.4 calls §3.11.5 "logically prior to this clause";
//  5. the handover of the merged table to the parent, which is the whole of the
//     upward propagation.
func (w *walk) identityExit(c *icCheck) {
	c.fill(w)
	c.evaluate(w)
	c.table.resolveConflicts(w)
	c.keyrefs(w)
	if c.parent != nil {
		c.parent.table.absorb(c.table)
	}
}

// fill records c's element as the field node it was selected as, from its own
// ·governing type definition· and its own ·initial value·.
//
// Three shapes decline rather than contribute a value, each because the
// [schema actual value] §3.11.4 clause 3 reads does not exist for them:
//
//   - a ·governing type definition· that is not a simple type definition and
//     not a complex type with {content type}.{variety} simple, which is
//     governance.valueType's nil — including the case where no type was
//     determinable at all (a {type table} carrying a {test} the §3.12.6
//     evaluator declines, an unresolvable slot, an xsi:type whose ·override·
//     could not be decided).
//   - GAP(validate): an element that is ·nilled· (§3.3.4.3, key-nilled).
//     §3.3.5.4 gives it an absent [schema normalized value] and so an absent
//     [schema actual value], and §3.11.4's own Note names a nilled node as
//     leaving the ·key-sequence· short — which for a key is a clause 4.2.1
//     charge. Recording that absence rather than declining would widen clause
//     4.2.1 on the strength of this reading, so the slot declines instead.
//   - GAP(validate): an element with NO character information item [[child]]
//     whose declaration carries a {value constraint}. cvc-elt clause 5.1
//     replaces the item assessed with one whose ·normalized value· is that
//     constraint's {lexical form}, and §3.11.4's Note is explicit that "default
//     or fixed value constraints may play a part in ·key-sequences·"; reading
//     the empty ·initial value· instead would compare a value the document does
//     not have.
func (c *icCheck) fill(w *walk) {
	if len(c.pending) == 0 {
		return
	}
	m, present, decided := w.elementKeyMember(c)
	for _, p := range c.pending {
		p.target.slots[p.index].record(m, present, decided, c.e.Loc())
	}
}

// elementKeyMember is the ·key-sequence· member c's element contributes as a
// field node, on fill's terms.
func (w *walk) elementKeyMember(c *icCheck) (icKeyMember, bool, bool) {
	st := c.g.valueType()
	if st == nil || nilled(c.e, c.g) {
		return icKeyMember{}, false, false
	}
	if c.initial.Len() == 0 && c.g.hasDecl {
		if _, constrained := c.g.decl.ValueConstraint(); constrained {
			return icKeyMember{}, false, false
		}
	}
	nillable := c.g.hasDecl && c.g.decl.Nillable()
	return w.keyMember(st, c.initial.String(), c.e, true, nillable)
}

// keyMember maps one field node's lexical to the ·actual value· that is its
// [schema actual value], through the same String Valid (§3.16.4) pipeline the
// attribute charges run (value.ValidateLexical, under the namespace bindings in
// scope at the node that owns the lexical — elementContext).
//
// The three answers are the three the rule distinguishes. present=true is a
// non-absent [schema actual value]. present=false with decided=true is an
// ABSENT one — a lexical outside the type's lexical space has no ·actual value·
// (§3.3.5.4 clause 1.3), so the node contributes nothing to the ·key-sequence·
// and its own invalidity is cvc-attribute's or cvc-complex-type's to charge,
// not this rule's. decided=false is this processor declining.
//
// GAP(validate): that decline covers a value.ValidateLexical error that is a
// fault of the TYPE or of the backend rather than a verdict about the lexical
// (value.IsDatatypeVerdict) — an ungoverned type above all, which reports under
// cvc-datatype-valid exactly as a genuine rejection does. Reading one as
// "absent" would silently shorten a ·key-sequence·, and a short one is what
// clause 4.2.1 charges a key for (#774).
func (w *walk) keyMember(st *xsd.SimpleType, lexical string, owner Element, element, nillable bool) (icKeyMember, bool, bool) {
	if st == nil {
		return icKeyMember{}, false, false
	}
	v, err := value.ValidateLexical(w.backend, w.schema, st, lexical, elementContext{owner: owner})
	if err != nil {
		return icKeyMember{}, false, value.IsDatatypeVerdict(err)
	}
	return icKeyMember{st: st, v: v, element: element, nillable: nillable}, true, true
}

// record takes one candidate field node's answer into the slot, per clause 3's
// bound of at most one valued node. A declined node poisons the slot outright:
// the value it did not yield could have been the one that qualified the target,
// and guessing either way is what the decline exists to avoid.
func (s *icSlot) record(m icKeyMember, present, decided bool, loc xsderr.Loc) {
	if !decided {
		s.declined = true
		return
	}
	if !present {
		return
	}
	if s.filled {
		s.extra, s.loc = true, loc
		return
	}
	s.filled, s.member, s.loc = true, m, loc
}

// field is the compiled {fields} member at index i of the frame this target
// belongs to.
func (t *icTarget) field(i int) icExpr { return t.frame.fields[i] }

// sequence is the target's ·key-sequence·: the values of its filled slots, in
// {fields} order. It is only meaningful for a member of the ·qualified node
// set·, where every slot is filled by construction.
func (t *icTarget) sequence() icKeySequence {
	seq := make(icKeySequence, 0, len(t.slots))
	for i := range t.slots {
		if !t.slots[i].filled {
			continue
		}
		seq = append(seq, t.slots[i].member)
	}
	return seq
}

// evaluate settles clause 4 for the key and unique frames rooted at c's
// element, and records their ·qualified node set· as this element's own clause
// 2 entries in §3.11.5's c-kc.
//
// The keyref frames are NOT settled here but in keyrefs, after the table is
// merged and conflict-resolved: clause 4.3 reads "a node table associated with
// the {referenced key} in the [identity-constraint table] of E", and a key
// declared on E itself contributes to that table through this very pass.
//
// §3.11.5's ·eligible identity-constraint· gate is deliberately not applied to
// the entries: a binding is declared for every key and unique frame here,
// eligible or not. The property that gate decides — whether an
// [identity-constraint table] carries a binding at all — is unobservable to
// this package, and the only observable consequence of declaring one anyway is
// that clause 4.3 finds MORE entries to match, which costs a rejection and
// manufactures none. Its whole consumer set is icCheck.keyrefs, which charges
// on a key-sequence NOT found.
func (c *icCheck) evaluate(w *walk) {
	for _, f := range c.frames {
		if f.ic.Category() == xsd.IdentityConstraintKeyref {
			continue
		}
		b := c.table.declare(f.ic.Name())
		if f.declined {
			b.declined = true
			continue
		}
		q, ok := f.qualify(w, c.e)
		if !ok {
			b.declined = true
			continue
		}
		f.duplicates(w, c.e, q)
		if f.ic.Category() == xsd.IdentityConstraintKey {
			f.keyOnly(w, c.e, q)
		}
		for _, t := range q {
			b.entries = append(b.entries, icEntry{seq: t.sequence(), node: t.node})
		}
	}
}

// qualify is clause 4's ·qualified node set·: the members of the ·target node
// set· whose ·key-sequence· is as long as {fields}. It charges clause 3 for a
// field that selected more than one valued node on the way.
//
// It reports ok=false where the frame must not settle any further clause: a
// slot this processor declined (icSlot.declined), and a clause 3 charge, after
// which the ·key-sequences· of the offending target are not the rule's. Both
// leave the frame's node-table binding declined, so a keyref referring to it
// declines too rather than charging for entries that were never assembled.
func (f *icFrame) qualify(w *walk, host Element) ([]*icTarget, bool) {
	var q []*icTarget
	ok := true
	for _, t := range f.targets {
		short := false
		for i := range t.slots {
			s := &t.slots[i]
			if s.declined {
				return nil, false
			}
			if s.extra {
				w.res.violations = append(w.res.violations, xsderr.New(ruleCvcIdentityConstraint, s.loc,
					"the field %q of the identity constraint %s declared on %s selects more than one node with a non-absent [schema actual value] for the ·target node· %s, but cvc-identity-constraint clause 3 admits at most one",
					f.ic.Fields()[i].Expression(), f.ic.Name(), host.Name(), t.e.Name()))
				ok = false
				continue
			}
			if !s.filled {
				short = true
			}
		}
		if !short {
			q = append(q, t)
		}
	}
	if !ok {
		return nil, false
	}
	return q, true
}

// duplicates is clause 4.1 for a unique and clause 4.2.2 for a key, which are
// one test over two categories: no two members of the ·qualified node set· have
// ·key-sequences· whose members are pairwise equal or identical.
//
// The scan is over pairs in document order and charges the LATER member of each
// duplicated pair, which is the one a reader has to delete; an earlier member
// already charged is not charged again for a third occurrence, so n equal
// key-sequences charge n-1 times and not n(n-1)/2.
func (f *icFrame) duplicates(w *walk, host Element, q []*icTarget) {
	charged := make([]bool, len(q))
	seqs := make([]icKeySequence, len(q))
	for i, t := range q {
		seqs[i] = t.sequence()
	}
	for i := range q {
		for j := i + 1; j < len(q); j++ {
			if charged[j] {
				continue
			}
			same, decided := w.sameKeySequence(seqs[i], seqs[j])
			if !decided || !same {
				continue
			}
			charged[j] = true
			w.res.violations = append(w.res.violations, xsderr.New(ruleCvcIdentityConstraint, q[j].e.Loc(),
				"the ·target node· %s has a ·key-sequence· equal or identical to the one at %s, but the identity constraint %s declared on %s is a %s, whose %s forbids two members of the ·qualified node set· to share one (Datatypes §2.2 Equality and Identity)",
				q[j].e.Name(), q[i].e.Loc(), f.ic.Name(), host.Name(), f.ic.Category(), f.duplicateClause()))
		}
	}
}

// duplicateClause names the clause the shared test is charged under for this
// frame's category: 4.1 for a unique, 4.2.2 for a key.
func (f *icFrame) duplicateClause() string {
	if f.ic.Category() == xsd.IdentityConstraintKey {
		return "clause 4.2.2"
	}
	return "clause 4.1"
}

// keyOnly charges the two clauses a key has and a unique does not.
//
// Clause 4.2.1 — the ·target node set· and the ·qualified node set· are equal —
// is what makes a MISSING field an error for a key where it is none for a
// unique, whose own Note says so outright: "a selected unique node may have
// fields that do not have corresponding [schema actual value]s". The charge
// carries the ·target node·'s own location, which is where the missing field
// was to have been.
//
// Clause 4.2.3 forbids an ELEMENT member of a ·key-sequence· that was assessed
// against a declaration whose {nillable} is true, whatever the element's
// content turned out to be. It is read off the declaration's own property, per
// the clause's Note.
func (f *icFrame) keyOnly(w *walk, host Element, q []*icTarget) {
	qualified := make(map[int]bool, len(q))
	for _, t := range q {
		qualified[t.node] = true
	}
	for _, t := range f.targets {
		if qualified[t.node] {
			continue
		}
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcIdentityConstraint, t.e.Loc(),
			"the ·target node· %s has a ·key-sequence· shorter than the {fields} of the identity constraint %s declared on %s, so it is not a member of the ·qualified node set·, and cvc-identity-constraint clause 4.2.1 requires a key's two node sets to be equal",
			t.e.Name(), f.ic.Name(), host.Name()))
	}
	for _, t := range q {
		for i := range t.slots {
			m := t.slots[i].member
			if !m.element || !m.nillable {
				continue
			}
			w.res.violations = append(w.res.violations, xsderr.New(ruleCvcIdentityConstraint, t.slots[i].loc,
				"the field %q of the identity constraint %s declared on %s contributes an element member to the ·key-sequence· of %s whose element declaration has {nillable} true, which cvc-identity-constraint clause 4.2.3 forbids for a key",
				f.ic.Fields()[i].Expression(), f.ic.Name(), host.Name(), t.e.Name()))
		}
	}
}

// keyrefs charges clause 4.3 for the keyref frames rooted at c's element:
// each member of the ·qualified node set· must find a ·key-sequence· equal or
// identical to its own in the node table associated with the {referenced key}
// in E's own [identity-constraint table].
//
// E's OWN table is the whole of the rule's reach, and the reason a keyref
// resolves only inside its own subtree: §3.11.5 assembles that table from its
// children's tables and from what E itself qualified, never from a sibling's or
// an ancestor's.
//
// Three shapes decline instead of charging: a frame whose paths did not
// compile, a ·qualified node set· qualify could not settle, and a binding some
// descendant declined for one of those same reasons. A referenced key with no
// binding AT ALL is not one of them — that is a key whose own element never
// occurred in the subtree, which is exactly the "there is a node table" half of
// clause 4.3 failing, and it is charged.
func (c *icCheck) keyrefs(w *walk) {
	for _, f := range c.frames {
		if f.ic.Category() != xsd.IdentityConstraintKeyref || f.declined {
			continue
		}
		q, ok := f.qualify(w, c.e)
		if !ok {
			continue
		}
		ref, _ := f.ic.ReferencedKeyName()
		b, found := c.table.binding(ref)
		if found && b.declined {
			continue
		}
		for _, t := range q {
			if found {
				same, decided := b.lookup(w, t.sequence())
				if !decided || same {
					continue
				}
			}
			w.res.violations = append(w.res.violations, xsderr.New(ruleCvcIdentityConstraint, t.e.Loc(),
				"the ·key-sequence· of %s matches no entry in the node table of %s, the {referenced key} of the keyref %s declared on %s, and cvc-identity-constraint clause 4.3 resolves a keyref only against the node tables assembled from the sub-tree rooted at that element (§3.11.5)",
				t.e.Name(), ref, f.ic.Name(), c.e.Name()))
		}
	}
}

// sameKeySequence is clause 4.1/4.2.2/4.3's "equal or identical, member for
// member". decided is false wherever any member pair could not be compared, so
// clause 4.3 — the one that charges on a NON-match — never charges off a
// comparison this processor could not make.
func (w *walk) sameKeySequence(a, b icKeySequence) (same, decided bool) {
	if len(a) != len(b) {
		return false, true
	}
	for i := range a {
		same, decided := w.sameKeyMember(a[i], b[i])
		if !decided {
			return false, false
		}
		if !same {
			return false, true
		}
	}
	return true, true
}

// sameKeyMember compares two ·key-sequence· members in the VALUE space, which
// is the whole of Datatypes §2.2's contribution to this rule: "1" and "01" are
// one xs:integer key, and a lexical comparison in its place would report them
// as two.
//
// The comparison is refused unless both members were validated against the SAME
// [xsd.SimpleType] node, which a compiled schema shares one of per type
// (xsd/simpletype.go). §2.2.1 makes values of different ·primitive· datatypes
// "artificially distinct" even where they look alike, and a backend's mapping
// for a derived type may in addition represent values in a space of its own, so
// two values parsed under two mappings are not comparable by asking one of them
// — the answer would be a decided NOT-same, which is the one outcome clause 4.3
// turns into a rejection. Declining costs recall in the other three clauses,
// which charge on a decided SAME, and nothing at all in correctness.
//
// The equal-or-identical union itself is Datatypes §2.2.2's — "all comparisons
// for 'sameness' prescribed by this specification test for either equality or
// identity, not for identity alone" — over the two capability interfaces
// package value publishes for exactly this question.
func (w *walk) sameKeyMember(a, b icKeyMember) (same, decided bool) {
	if a.st != b.st {
		return false, false
	}
	id, hasIdentical := a.v.(value.Identical)
	if hasIdentical && id.Identical(b.v) {
		return true, true
	}
	eq, hasEq := a.v.(value.Eq)
	if hasEq && eq.Eq(b.v) {
		return true, true
	}
	return false, hasIdentical || hasEq
}

// icTable is one element's [identity-constraint table] (§3.11.5): one binding
// per identity-constraint definition, in the order the definitions were first
// met, so several charges off one table arrive in a fixed order (STYLE D2). The
// definitions are keyed by their {name}, which is a schema-wide identity —
// xsd's own idcIndex is keyed by it, and it is what a keyref's {referenced key}
// names.
type icTable struct{ bindings []*icBinding }

// icBinding is one Identity-constraint Binding of §3.11.5: a {definition} and
// its ·node table·. declined marks a binding this processor could not assemble
// faithfully, so clause 4.3 declines against it rather than charging for
// entries that were never built.
type icBinding struct {
	def      xsd.QName
	entries  []icEntry
	declined bool
}

// icEntry is one (·key-sequence·, node) pair of a ·node table·. fromChild
// records which of §3.11.5's two clauses the entry owes its inclusion to, which
// is the only thing the conflict resolution reads: "potential conflicts are
// resolved by not including any conflicting entries which would have owed their
// inclusion to clause 1".
type icEntry struct {
	seq       icKeySequence
	node      int
	fromChild bool
}

// declare returns the binding for def, adding it in first-seen order where the
// table has none. A key or unique frame declares its binding whether or not it
// found anything, so an ABSENT binding means the constraint's own element never
// occurred in the subtree — which is what clause 4.3 charges on.
func (t *icTable) declare(def xsd.QName) *icBinding {
	if b, found := t.binding(def); found {
		return b
	}
	b := &icBinding{def: def}
	t.bindings = append(t.bindings, b)
	return b
}

// binding reports the binding for def, or false where the table has none.
func (t *icTable) binding(def xsd.QName) (*icBinding, bool) {
	for _, b := range t.bindings {
		if b.def == def {
			return b, true
		}
	}
	return nil, false
}

// absorb merges one child's [identity-constraint table] into this one, which is
// §3.11.5's clause 1 and the whole of the upward propagation: every entry of
// every binding in a child's table is an entry of the parent's, marked as owing
// its inclusion to that clause. A declined binding stays declined all the way
// up, so a keyref anywhere above the decline declines with it.
func (t *icTable) absorb(child icTable) {
	for _, cb := range child.bindings {
		b := t.declare(cb.def)
		if cb.declined {
			b.declined = true
		}
		for _, e := range cb.entries {
			e.fromChild = true
			b.entries = append(b.entries, e)
		}
	}
}

// resolveConflicts applies §3.11.5's proviso to every binding: the table holds
// its entries "provided no two entries have the same key-sequence but distinct
// nodes".
func (t *icTable) resolveConflicts(w *walk) {
	for _, b := range t.bindings {
		b.entries = resolveEntryConflicts(w, b.entries)
	}
}

// resolveEntryConflicts drops the conflicting entries §3.11.5 names: those that
// "would have owed their inclusion to clause 1", i.e. the ones that arrived
// from a child. Where BOTH sides of a conflict arrived that way both go, which
// the spec spells out — "if all the conflicting entries arose under clause 1
// above, this means no entry at all will appear for the offending
// key-sequence".
//
// Two conflicting entries that both arose LOCALLY are a shape the proviso does
// not name, because clause 4.1/4.2.2 has already charged for it. The earlier one
// is kept, which is deterministic (the scan is in document order) and keeps the
// table larger.
//
// An undecided comparison is not a conflict either, and both departures keep
// MORE entries than the proviso would. The whole consumer set of a node table
// is icCheck.keyrefs, whose charge condition is a key-sequence NOT found among
// them, so an extra entry costs a rejection and can manufacture none.
func resolveEntryConflicts(w *walk, entries []icEntry) []icEntry {
	drop := make([]bool, len(entries))
	for i := range entries {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].node == entries[j].node {
				continue
			}
			same, decided := w.sameKeySequence(entries[i].seq, entries[j].seq)
			if !decided || !same {
				continue
			}
			if entries[i].fromChild {
				drop[i] = true
			}
			if entries[j].fromChild || !entries[i].fromChild {
				drop[j] = true
			}
		}
	}
	kept := make([]icEntry, 0, len(entries))
	for i, e := range entries {
		if drop[i] {
			continue
		}
		kept = append(kept, e)
	}
	return kept
}

// lookup is clause 4.3's search: is there an entry in this node table whose
// ·key-sequence· is equal or identical to seq, member for member? decided is
// false where no entry matched but some comparison could not be made, so the
// caller declines instead of charging.
func (b *icBinding) lookup(w *walk, seq icKeySequence) (same, decided bool) {
	decided = true
	for _, e := range b.entries {
		match, ok := w.sameKeySequence(e.seq, seq)
		if match {
			return true, true
		}
		if !ok {
			decided = false
		}
	}
	return false, decided
}
