package jsonsrc

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The engine rules whose charges a note extends (note.hint).
const (
	ruleCvcType           xsderr.Rule = "cvc-type"
	ruleCvcComplexType    xsderr.Rule = "cvc-complex-type"
	ruleCvcComplexContent xsderr.Rule = "cvc-complex-content"
)

// note is what the mapping knows about one member, in the instance's own
// terms, kept at the location the engine charges that member at: hint is the
// clause [Validate] appends to the Msg of a charge of rule there, and "" for
// a rule the note says nothing about. Its arms are characters and unnamed.
type note interface {
	hint(rule xsderr.Rule) string
}

// characters is a scalar member whose element's declared {type definition}
// admits no character content, kept at the scalar's location: key is the
// member's key as spelled, an array item's being its array's, and lexical the
// scalar's lexical form (walker.noteCharacters).
type characters struct {
	key, lexical string
}

// hint answers a cvc-complex-type charge. The engine charges that rule at a
// character information item's location under clause 1.1, where the
// {content type} is empty, and clause 1.3, where it is element-only and the
// text is not white space; a JSON object spells either element's content, and
// a scalar spells none.
func (c characters) hint(rule xsderr.Rule) string {
	if rule != ruleCvcComplexType {
		return ""
	}
	return fmt.Sprintf("; JSON member %q has the scalar value %q, where the type of its element admits no character content: object expected, {%q:{...}}",
		c.key, c.lexical, c.key)
}

// unnamed is a child member no child element declaration, no attribute use
// and no wildcard of its parent's type names (object.namesNone), kept at the
// member's location: key is its key as spelled, owner the parent's, and
// object what the parent's members are classified against
// (walker.noteUnnamed).
type unnamed struct {
	key, owner string
	object     *object
}

// hint answers a cvc-type charge where the parent's type is a simple type,
// whose element cvc-type clause 3.1.2 charges at its first child, and a
// cvc-complex-content charge where it is a complex type, which charges a child
// the content model ·attributes· to no particle at the child's location. It
// names the keys the type does carry, its attribute uses' and then its child
// declarations' local names, and, for a key that is one use's local name
// behind one "@", "_" or "-", the key that names it.
func (u unnamed) hint(rule xsderr.Rule) string {
	if u.object.simpleType && rule == ruleCvcType {
		return fmt.Sprintf("; JSON member %q is a member of the object of %q, whose type is a simple type: scalar expected, {%q:<string, number or boolean>}",
			u.key, u.owner, u.owner)
	}
	if u.object.simpleType || rule != ruleCvcComplexContent {
		return ""
	}
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
// wildcard, a declaredElement it contains being one it names.
func (o *object) namesNone(m elementMember) bool {
	if len(o.elementWildcards) > 0 || o.attributeWildcard != nil {
		return false
	}
	if d, ok := m.(declaredElement); ok {
		return d.named
	}
	return true
}

// noteUnnamed keeps an unnamed note at loc, for the child member key that m
// classifies under in's object, where that object's type names none of it
// (object.namesNone). The notes kept grow with the members no type names, not
// with the document (STYLE P4).
func (w *walker) noteUnnamed(m elementMember, loc xsderr.Loc, key string, in *members) {
	if !in.object.namesNone(m) {
		return
	}
	w.notes[loc] = unnamed{key: key, owner: in.owner.key, object: in.object}
}

// noteCharacters keeps a characters note at loc, the location of the scalar
// whose lexical form is lexical, for the member key that m classifies, where
// lexical is not empty and m's declaration's {type definition} is a complex
// type whose {content type} is empty or element-only. The {type definition}
// bounds the notes kept to the scalars the engine charges, the document never
// being held (STYLE P4); a type a {type table} selects instead is not read,
// and a charge under one carries no hint.
func (w *walker) noteCharacters(m elementMember, key, lexical string, loc xsderr.Loc) {
	d, ok := m.(declaredElement)
	if !ok || lexical == "" {
		return
	}
	t, ok := w.schema.ResolvedType(d.decl.TypeDefinition())
	if !ok {
		return
	}
	ct, ok := t.(xsd.ComplexType)
	if !ok {
		return
	}
	if v := ct.ContentType().Variety(); v != xsd.ContentEmpty && v != xsd.ContentElementOnly {
		return
	}
	w.notes[loc] = characters{key: key, lexical: lexical}
}

// annotate appends to the Msg of each violation of res that a note is kept at
// the location of the clause that note's hint gives its rule. The violations
// are res's own *xsderr.Error values, so the result Validate returns carries
// the extended messages; the rule ID and location stay the engine's.
func (w *walker) annotate(res *validate.Result) {
	for _, e := range res.Violations() {
		n, ok := w.notes[e.Loc]
		if !ok {
			continue
		}
		e.Msg += n.hint(e.Rule)
	}
}
