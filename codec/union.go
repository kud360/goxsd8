package codec

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// unionGo is the Go type codegen gives a union simple type Decode decodes, as
// simpleDecline reads it (codegen/doc.go "# Simple types"): a sealed interface
// of one variant per basic member, members holding each basic member and the
// Go type its variant is over, in basic-member order.
type unionGo struct {
	members []unionMember
}

// unionMember is one basic member st of a union and g, the Go type codegen
// gives it, an atomic Go type or a list's slice.
type unionMember struct {
	st *xsd.SimpleType
	g  simpleGo
}

// forScalar returns the unionScalar of u, a union labelled union: each basic
// member's scalar, labelled by memberLabel after union.
func (u *unionGo) forScalar(union string) scalarElem {
	s := &unionScalar{members: make([]scalar, len(u.members))}
	for i, m := range u.members {
		s.members[i] = scalarOf(m.g, m.st, memberLabel(m.st, union))
	}
	return s
}

// unionDecline returns the unionGo of st, a union simple type of the item
// subject names, or else the *DeclineError Decode's doc states for st, item 5,
// or a plain error, goshape.BasicMembers' naming subject: each basic member in
// basic-member order (goshape.BasicMembers), examined as simpleDecline
// examines an atomic or list type, the first declining member's construct,
// its detail naming the member after subject.
func unionDecline(r xsd.TypeResolver, st *xsd.SimpleType, subject string) (*unionGo, error) {
	basic, err := goshape.BasicMembers(r, st)
	if err != nil {
		return nil, fmt.Errorf("codec: the basic members of the union simple type of %s: %w", subject, err)
	}
	u := &unionGo{members: make([]unionMember, 0, len(basic))}
	for _, m := range basic {
		g, err := simpleDecline(r, m, memberSubject(subject, m))
		if err != nil {
			return nil, err
		}
		u.members = append(u.members, unionMember{st: m, g: g})
	}
	return u, nil
}

// memberSubject names, after subject, m, the basic member of the union whose
// member declines, and m's schema Loc.
func memberSubject(subject string, m *xsd.SimpleType) string {
	if m.Name() == (xsd.QName{}) {
		return fmt.Sprintf("%s, through the union's anonymous basic member, a type defined at %s", subject, m.Loc())
	}
	return fmt.Sprintf("%s, through the union's basic member %s, a type defined at %s", subject, m.Name(), m.Loc())
}

// memberLabel names m, a basic member of the union union names, for a stage
// error and a trace: its QName and then "a basic member of " and union, or
// for an anonymous m that phrase alone.
func memberLabel(m *xsd.SimpleType, union string) string {
	if m.Name() != (xsd.QName{}) {
		return m.Name().String() + ", a basic member of " + union
	}
	return "a basic member of " + union
}

// unionScalar is the basic members of a union as Decode decodes them, one
// scalar per member in basic-member order, each scalar's st that member.
type unionScalar struct {
	members []scalar
}

// unionValue is codec's value of a union: u, the union's basic members, i, the
// position in u.members of its ·active basic member·, and v, that member's
// value as its scalar's decode returns it. It converts to no caller type:
// pipeline.union builds the caller's variant from it.
type unionValue struct {
	u *unionScalar
	i int
	v reflect.Value
}

// unionValueType is unionValue's reflect.Type, the bare Go type of a union
// scalar.
var unionValueType = reflect.TypeFor[unionValue]()

// bare returns unionValueType: codec holds none of the union's interface.
func (*unionScalar) bare() reflect.Type { return unionValueType }

// spell spells a union's interface for an error message.
func (*unionScalar) spell() string { return "I for a union's sealed interface I" }

// datatypeStruct reports false: a union's interface is no datatype struct.
func (*unionScalar) datatypeStruct() bool { return false }

// sealedShape reports whether t is an interface of one unexported method, the
// shape of every sealed interface codegen emits.
func sealedShape(t reflect.Type) bool {
	return t.Kind() == reflect.Interface && t.NumMethod() == 1 && !t.Method(0).IsExported()
}

// check holds it, a sealed interface at the site site names, to u's variants,
// item 6 of Decode's doc: cx's sets hold, for it, one variant per basic
// member, as bind matches each, and each is held to the sets its member's own
// Go type reads (scalar.holdSets). Neither the interface's nor a variant's
// name is compared.
func (u *unionScalar) check(it reflect.Type, site string, cx *checkContext) error {
	variants, supplied := cx.sets.of(it)
	if !supplied && len(u.members) > 0 {
		return fmt.Errorf("%w: no WithBranches supplies the variants of %s, the union's interface of %s, where codegen emits %d", ErrTypeMismatch, it, site, len(u.members))
	}
	if len(variants) != len(u.members) {
		return fmt.Errorf("%w: WithBranches supplies %d variants of %s where codegen emits %d", ErrTypeMismatch, len(variants), it, len(u.members))
	}
	bound, err := u.bind(it, variants)
	if err != nil {
		return err
	}
	for j, vt := range bound {
		if err := u.members[j].holdSets(vt, fmt.Sprintf("variant %s of %s", vt, site), cx); err != nil {
			return err
		}
	}
	return nil
}

// bind returns the variant of it, a union's sealed interface, that variants,
// the set WithBranches supplies for it, holds for each of u's basic members,
// in basic-member order: each variant, in the order supplied, matches the
// first member whose Go type it is over (scalar.underlies) and that no
// earlier variant matched, as Decode's doc, item 6, states. A nil variant,
// one supplied twice, and one matching no member are the ErrTypeMismatch
// errors item 6 states; a member no variant matches is left nil, which check's
// count of variants rules out. No name is compared.
func (u *unionScalar) bind(it reflect.Type, variants []reflect.Type) ([]reflect.Type, error) {
	bound := make([]reflect.Type, len(u.members))
	for i, vt := range variants {
		if vt == nil {
			return nil, fmt.Errorf("%w: WithBranches supplies a nil variant of %s", ErrTypeMismatch, it)
		}
		if slices.Contains(variants[:i], vt) {
			return nil, fmt.Errorf("%w: WithBranches supplies %s twice as a variant of %s", ErrTypeMismatch, vt, it)
		}
		j := u.firstUnbound(bound, vt)
		if j < 0 {
			return nil, fmt.Errorf("%w: variant %s of %s is over the Go type of no basic member that an earlier variant did not match", ErrTypeMismatch, vt, it)
		}
		bound[j] = vt
	}
	return bound, nil
}

// firstUnbound returns the position of the first of u's members whose Go type
// vt is over (scalar.underlies) and that bound, bind's variants so far, holds
// none for, or -1 when there is none.
func (u *unionScalar) firstUnbound(bound []reflect.Type, vt reflect.Type) int {
	for j, m := range u.members {
		if bound[j] == nil && m.underlies(vt) {
			return j
		}
	}
	return -1
}

// identify returns the position in u.members of the ·active basic member· of
// st, the union u's members are of, for lexical (Datatypes dt-active-basic-member,
// cvc-datatype-valid clause 2.3), as value.ValidatingType identifies it on
// p's backend under src's bindings and p.assertions, the evaluator
// validate.Result.FacetAssertions returns, whose doc states why the member is
// the one validate selected, or the *stageError of item 8, label naming st, at
// position token as stageError states it: an identification that fails never
// selects a later member.
func (u *unionScalar) identify(p *pipeline, st *xsd.SimpleType, label, lexical string, src source, token int) (int, error) {
	m, _, err := value.ValidatingType(p.backend, p.schema, st, lexical, src.bindings(), p.assertions)
	if err != nil {
		return 0, &stageError{stage: stageIdentification, label: label, fragment: lexical, token: token, src: src, cause: err}
	}
	i := slices.IndexFunc(u.members, func(s scalar) bool { return s.st == m })
	if i < 0 {
		return 0, &stageError{stage: stageIdentification, label: label, fragment: lexical, token: token, src: src,
			cause: errors.New("value.ValidatingType identifies a type that is no basic member of the union")}
	}
	return i, nil
}

// item returns the value of tok, the list item at position token of a list
// whose {item type definition}, item, is the union u's members are of, from
// src: the unionValue of its ·active basic member· (identify), tok read by
// that member's lexical stages, as a list item of an atomic item type is.
// cos-st-restricts clause 2.1 (checkListGraph, which Finalize runs) holds
// every basic member of such a union to {variety} atomic; a member of any
// other is an error. tok holds no whitespace character once the list's
// collapse has split it out, so the member's whiteSpace would leave it
// unchanged, and runs no stage here, as an atomic item's does not.
func (u *unionScalar) item(p *pipeline, item *listItem, tok string, src source, token int) (reflect.Value, error) {
	i, err := u.identify(p, item.st, item.label, tok, src, token)
	if err != nil {
		return reflect.Value{}, err
	}
	member := u.members[i]
	// scalarElem is a closed sum, so the assertion is STYLE T2's sealed-sum
	// exception.
	atom, isAtomic := member.elem.(atomic)
	if !isAtomic || member.item != nil {
		return reflect.Value{}, fmt.Errorf("codec: %s: %s is no atomic type, which a basic member of a list's {item type definition} is (cos-st-restricts clause 2.1)", src.loc(), member.label)
	}
	ls, err := newLexicalStages(p, atom, member.st, member.label, src, tok, token)
	if err != nil {
		return reflect.Value{}, err
	}
	v, err := ls.run(p, tok, token)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(unionValue{u: u, i: i, v: v}), nil
}

// union returns the value of t, a sealed interface of a union, that v, a
// unionValue, holds: t's variant that unionScalar.bind matches to v's member,
// of the set p.branches supplies for t, holding v's member value converted to
// it as convertTo converts. Item 6 held the variants first
// (unionScalar.check), so an interface with none, or a set bind refuses or
// matches no variant to the member, is an internal bug.
func (p *pipeline) union(v reflect.Value, t reflect.Type) reflect.Value {
	uv := v.Interface().(unionValue)
	variants, ok := p.branches.of(t)
	if !ok {
		panic(fmt.Sprintf("codec: no variants of %s reached the fill", t))
	}
	bound, err := uv.u.bind(t, variants)
	if err != nil || bound[uv.i] == nil {
		panic(fmt.Sprintf("codec: no variant of %s is bound to %s: %v", t, uv.u.members[uv.i].label, err))
	}
	out := reflect.New(t).Elem()
	out.Set(convertTo(p, uv.v, bound[uv.i]))
	return out
}
