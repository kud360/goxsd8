package jsonsrc

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The engine rules whose charges a note answers, and that validate's
// contentCheck.element charges a parent's content under (contentCharge).
const (
	ruleCvcType           xsderr.Rule = "cvc-type"
	ruleCvcComplexType    xsderr.Rule = "cvc-complex-type"
	ruleCvcComplexContent xsderr.Rule = "cvc-complex-content"
)

// noteAt is the charge a note answers: its rule and its location.
type noteAt struct {
	loc  xsderr.Loc
	rule xsderr.Rule
}

// note is what the mapping knows about one member, in the instance's own
// terms, kept at the charge it answers (noteAt): hint is the clause
// [Validate] appends to that charge's Msg, and own reports whether the charge
// is the element's own, which follows any charge its parent makes at the same
// rule and location, rather than the parent's, which precedes every charge of
// the element's (walker.annotate). Its arms are characters, scalarExpected and
// unnamed.
type note interface {
	hint() string
	own() bool
}

// characters is a scalar member whose element has no type that admits
// character content, kept at the cvc-complex-type charge at the scalar's
// location: key is the member's key as spelled, an array item's being its
// array's, and lexical the scalar's lexical form (walker.noteCharacters).
type characters struct {
	key, lexical string
}

// hint answers the element's own cvc-complex-type charge, which the engine
// makes at a character information item's location under clause 1.1, where
// the {content type} is empty, and clause 1.3, where it is element-only and
// the text is not white space; a JSON object spells either element's content,
// and a scalar spells none.
func (c characters) hint() string {
	return fmt.Sprintf("; JSON member %q has the scalar value %q, where the type of its element admits no character content: object expected, {%q:{...}}",
		c.key, c.lexical, c.key)
}

// own is true: at an array item, whose location is its scalar's, the parent's
// cvc-complex-type charge where its {content type} is empty or simple
// precedes it.
func (characters) own() bool { return true }

// scalarExpected is the first child member of the object of an element whose
// type is a simple type, kept at the cvc-type charge clause 3.1.2 makes there
// (walker.noteUnnamed): key is the member's key as spelled, and owner the
// parent's.
type scalarExpected struct {
	key, owner string
}

// hint answers the parent's cvc-type clause 3.1.2 charge.
func (s scalarExpected) hint() string {
	return fmt.Sprintf("; JSON member %q is a member of the object of %q, whose type is a simple type: scalar expected, {%q:<string, number or boolean>}",
		s.key, s.owner, s.owner)
}

func (scalarExpected) own() bool { return false }

// unnamed is a child member no child element declaration and no attribute use
// of its parent's type names, where that type has no element wildcard and no
// {attribute wildcard} (object.namesNone), kept at the cvc-complex-content
// charge the parent's content model makes at the member's location
// (walker.noteUnnamed): key is its key as spelled, owner the parent's, and
// object what the parent's members are classified against.
type unnamed struct {
	key, owner string
	object     *object
}

// hint answers the parent's cvc-complex-content charge. It names the keys the
// type does carry, its attribute uses' and then its child declarations' local
// names, and, for a key that is one use's local name behind one "@", "_" or
// "-", the key that names it.
func (u unnamed) hint() string {
	var keys, uses []string
	for _, a := range u.object.uses {
		uses = append(uses, a.DeclarationName().Local)
	}
	for _, local := range uses {
		keys = appendKey(keys, local)
	}
	for _, d := range u.object.contained {
		keys = appendKey(keys, d.Name().Local)
	}
	hint := fmt.Sprintf("; JSON member %q names no child element declaration and no attribute use of the type of %q", u.key, u.owner)
	if len(keys) > 0 {
		hint += ", whose members are keyed " + strings.Join(keys, ", ")
	}
	if len(u.key) > 1 && strings.ContainsRune("@_-", rune(u.key[0])) && slices.Contains(uses, u.key[1:]) {
		hint += fmt.Sprintf("; an attribute member is keyed by the attribute's own name, %q, not %q", u.key[1:], u.key)
	}
	return hint
}

func (unnamed) own() bool { return false }

// appendKey appends local, quoted as a key spells it, to keys unless it is
// there already.
func appendKey(keys []string, local string) []string {
	q := fmt.Sprintf("%q", local)
	if slices.Contains(keys, q) {
		return keys
	}
	return append(keys, q)
}

// namesNone reports whether m, a member classified against o, is a child no
// declaration o contains, no use and no wildcard of its type names: the child
// walker.wildcardMember names through namedChild where the type has no
// element wildcard and no {attribute wildcard}, a declaredElement it contains
// being one it names.
func (o *object) namesNone(m elementMember) bool {
	if len(o.elementWildcards) > 0 || o.attributeWildcard != nil {
		return false
	}
	if d, ok := m.(declaredElement); ok {
		return d.named
	}
	return true
}

// noteUnnamed keeps a note at loc, for the child member key that m classifies
// under in's object, standing at at, where that object's type names none of
// it (object.namesNone) and the parent's content is charged there
// (placement.parentCharge): a scalarExpected at cvc-type clause 3.1.2's
// charge, and an unnamed at cvc-complex-content's. The notes kept grow with
// the members no type names, not with the document (STYLE P4).
//
// GAP(jsonsrc): no note is kept where the types that may govern the parent
// differ on the charge at the member, a {type table} below the root
// selecting among them by a {test} the mapping does not evaluate (selection),
// though the engine charges the one it selects; the charge carries no clause.
// Owned by #2699, which computes the [inherited attributes] such a {test}
// reads.
//
// GAP(jsonsrc): no note is kept where the mapping attributes none of the
// parent's children (object.unattributed), xsd.Schema.ContentMatcher
// declining the content model of a type that may govern it; the charge
// carries no clause. Owned by #2670, which retires with #1601's ruling.
func (w *walker) noteUnnamed(m elementMember, loc xsderr.Loc, key string, in *members, at placement) {
	rule, settled := at.parentCharge()
	if !settled || !in.object.namesNone(m) {
		return
	}
	if rule == ruleCvcType {
		w.notes[noteAt{loc: loc, rule: rule}] = scalarExpected{key: key, owner: in.owner.key}
	}
	if rule == ruleCvcComplexContent {
		w.notes[noteAt{loc: loc, rule: rule}] = unnamed{key: key, owner: in.owner.key, object: in.object}
	}
}

// noteCharacters keeps a characters note at loc, the location of the scalar
// whose lexical form is lexical, for the member key that m classifies,
// standing at at, where lexical is not empty and each type that may govern
// its element (walker.scalarTypes) is a complex type whose {content type} is
// empty or element-only. The types bound the notes kept to the scalars the
// engine charges, the document never being held (STYLE P4).
//
// GAP(jsonsrc): no note is kept where those types differ on whether their
// {content type} admits the scalar, a {type table} below the root selecting
// among them by a {test} the mapping does not evaluate (selection), though
// the engine charges the one it selects; the charge carries no clause. Owned
// by #2699, which computes the [inherited attributes] such a {test} reads.
//
// GAP(jsonsrc): no note is kept where the mapping attributes none of the
// element's siblings (object.unattributed), so that no type is determined to
// govern it; the charge carries no clause. Owned by #2670, which retires with
// #1601's ruling.
func (w *walker) noteCharacters(m elementMember, key, lexical string, loc xsderr.Loc, at placement) {
	d, ok := m.(declaredElement)
	if !ok || lexical == "" {
		return
	}
	ts := w.scalarTypes(d, at)
	if len(ts) == 0 {
		return
	}
	for _, t := range ts {
		ct, ok := t.(xsd.ComplexType)
		if !ok {
			return
		}
		if v := ct.ContentType().Variety(); v != xsd.ContentEmpty && v != xsd.ContentElementOnly {
			return
		}
	}
	w.notes[noteAt{loc: loc, rule: ruleCvcComplexType}] = characters{key: key, lexical: lexical}
}

// scalarTypes is each type that may govern the scalar-valued element d
// declares, standing at at: below the root, walker.governing's; at the root,
// the type d's {type table} selects where it carries one, which the root's
// attributes, none for a scalar, decide alone (key-cta-ta-select clause
// 1.1.2, key-p-inherited clause 1), and d's {type definition} otherwise. It is
// empty where none is determined.
func (w *walker) scalarTypes(d declaredElement, at placement) []xsd.TypeDefinition {
	table, tabled := d.decl.TypeTable()
	if at.parent == nil && tabled {
		slot, decided := w.selected(table, nil, nil)
		if !decided {
			return nil
		}
		t, ok := w.schema.ResolvedType(slot)
		if !ok {
			return nil
		}
		return []xsd.TypeDefinition{t}
	}
	declared, ok := w.schema.ResolvedType(d.decl.TypeDefinition())
	if !ok {
		return nil
	}
	return w.governing(d.decl, declared, at)
}

// annotate appends to the Msg of each violation of res that a note answers
// the clause that note's hint gives: a note kept at a charge's rule and
// location answers the first violation there where the charge is the
// parent's, and the last where it is the element's own (note.own), so that a
// note answers neither the parent's charge nor the element's at a rule and
// location they share. Writing Msg through res.Violations() writes res's own
// violations, as validate.Result.Violations states; the rule ID and location
// stay the engine's.
func (w *walker) annotate(res *validate.Result) {
	var answers []*xsderr.Error
	answered := map[noteAt]int{}
	for _, e := range res.Violations() {
		at := noteAt{loc: e.Loc, rule: e.Rule}
		n, ok := w.notes[at]
		if !ok {
			continue
		}
		i, seen := answered[at]
		if !seen {
			answered[at] = len(answers)
			answers = append(answers, e)
			continue
		}
		if n.own() {
			answers[i] = e
		}
	}
	for _, e := range answers {
		e.Msg += w.notes[noteAt{loc: e.Loc, rule: e.Rule}].hint()
	}
}
