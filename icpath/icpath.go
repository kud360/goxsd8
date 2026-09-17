package icpath

import (
	"github.com/kud360/goxsd8/xsd"
)

// Expr is one compiled {selector} or one compiled member of a {fields}: the
// union of its Paths, in the order they were written, so what a level reports
// never varies between runs (STYLE D2).
//
// The zero value selects nothing and advances to nothing. It is what a declined
// compile returns, and it is never a legal compiled expression: production [1]
// admits no empty union.
type Expr struct{ paths []path }

// CompileSelector compiles the {selector} of an identity-constraint definition
// (§3.11.1, an [xsd.XPathExpression] property record) into a matchable [Expr],
// reporting ok false for an {expression} outside the ·selector subset· —
// productions [1] through [4], with no attribute step anywhere.
//
// ok false is the WITHHOLD direction (PRINCIPLES 20) and is not a verdict about
// the schema: a legal XPath 2.0 path this subset does not admit looks exactly
// like an illegal one to a restricted-subset parser. Its consequence — the
// constraint charging nothing at all — is argued at the caller
// (validate/cvcidentityconstraint.go's icFrame.declined), which carries the GAP
// marker for it.
//
// It covers the four shapes [SelectorViolation] CHARGES as well, under the same
// one encoding: a tree built over a name that did not resolve is not matchable
// whatever the schema's fate, and a component assembled directly through
// [xsd.NewIdentityConstraint] reaches no assembler and so no charge.
func CompileSelector(x xsd.XPathExpression) (Expr, bool) {
	expr, d := compile(x, false)
	if d.kind != noDefect {
		return Expr{}, false
	}
	return expr, true
}

// CompileField compiles one member of an identity-constraint definition's
// {fields} (§3.11.1, an [xsd.XPathExpression] property record) into a matchable
// [Expr], reporting ok false for an {expression} outside the ·field subset· —
// the same productions with [7] replacing [2], so the FINAL step alone may name
// an attribute.
//
// ok false carries what [CompileSelector]'s does, on the same terms.
func CompileField(x xsd.XPathExpression) (Expr, bool) {
	expr, d := compile(x, true)
	if d.kind != noDefect {
		return Expr{}, false
	}
	return expr, true
}

// Live is one [Expr]'s live match state at one level of a descent: the step
// indices each path is waiting to match against the NEXT element down,
// ascending and deduplicated. A path selecting its context node holds no live
// index — it has no step to advance — and [Expr.Self] reports it instead.
//
// It carries the [Expr] it was opened over, so advancing one expression's live
// state against another is unrepresentable rather than merely documented (STYLE
// T1). The zero Live advances to nothing, which is what the zero [Expr] selects.
type Live struct {
	x    Expr
	step [][]int
}

// Selection is what one level of a descent selects: the element the descent
// just entered, some of its [[attributes]], or neither.
//
// The zero value selects nothing, which is what a level matching no path
// reports.
type Selection struct {
	element bool
	attrs   []nameTest
}

// SelectsElement reports whether the level selects the element itself — a path
// whose last Step matched it.
func (s Selection) SelectsElement() bool { return s.element }

// SelectsAttributes reports whether the level selects any of the element's
// [[attributes]], which is the gate a caller reads its [[attributes]] behind:
// production [7]'s final `'@' NameTest` is the only way an attribute is ever
// selected, so a Selection reporting false here admits none of them.
func (s Selection) SelectsAttributes() bool { return len(s.attrs) > 0 }

// SelectsAttribute reports whether the level selects the attribute whose
// ·expanded name· is n. The NameTests behind it never cross this boundary: their
// prefixes are resolved at compile time and their axis asymmetry is settled
// there too (see nameTest), so the answer here is a comparison of ·expanded
// names· and the caller needs no axis of its own.
func (s Selection) SelectsAttribute(n xsd.QName) bool {
	for _, t := range s.attrs {
		if t.matches(n) {
			return true
		}
	}
	return false
}

// Start opens x at its context node, with every path waiting on its first step.
//
// A `.//` path is seeded EMPTY, because candidates adds step 0 back at every
// level including this one. Seeding it here as well would put 0 in the candidate
// set twice, and every index derived from it twice again one level down — a live
// set doubling per level rather than one holding each step index once.
func (x Expr) Start() Live {
	l := Live{x: x, step: make([][]int, len(x.paths))}
	for i := range x.paths {
		if len(x.paths[i].steps) > 0 && !x.paths[i].anyDepth {
			l.step[i] = []int{0}
		}
	}
	return l
}

// Self reports what x selects at its own CONTEXT node: the node itself (`.`),
// and the attributes of it that a path selects (`@id`). Both are the zero-step
// case, which [Live.Advance] never reaches — it only ever looks one level down.
func (x Expr) Self() Selection {
	var s Selection
	for _, p := range x.paths {
		if len(p.steps) > 0 {
			continue
		}
		if p.hasAttr {
			s.attrs = append(s.attrs, p.attr)
			continue
		}
		s.element = true
	}
	return s
}

// Advance carries l one level down, into a child element whose ·expanded name·
// is n, and reports the live state at that child together with what the child is
// selected as: the selected node itself for a path whose last step matched, or
// the child's attributes for a field path whose element steps all matched.
//
// A path whose anyDepth is set re-seeds step 0 at EVERY level, which is the
// whole of `('.' '//')?`: descendant-or-self::node()/child::Step matches the
// first step at any depth at or below the context node's children, and every
// later step relative to wherever that match landed.
func (l Live) Advance(n xsd.QName) (Live, Selection) {
	next := Live{x: l.x, step: make([][]int, len(l.x.paths))}
	var s Selection
	for i, p := range l.x.paths {
		if len(p.steps) == 0 {
			continue
		}
		for _, j := range candidates(l.step[i], p.anyDepth) {
			if !p.steps[j].matches(n) {
				continue
			}
			if j+1 == len(p.steps) {
				if p.hasAttr {
					s.attrs = append(s.attrs, p.attr)
					continue
				}
				s.element = true
				continue
			}
			next.step[i] = append(next.step[i], j+1)
		}
	}
	return next, s
}

// candidates is the step indices one path tries against the next element down:
// the ones carried from the level above, plus step 0 again for a `.//` path. The
// seed is prepended rather than appended so the result stays ascending, which is
// what keeps the next live set ascending and duplicate-free without a sort — a
// carried index is always at least 1, so a live set never holds 0.
func candidates(live []int, anyDepth bool) []int {
	if !anyDepth {
		return live
	}
	return append([]int{0}, live...)
}

// nameTest is one NameTest of production [4], with its prefix ALREADY resolved:
// the {namespace bindings} of the XPath Expression property record that carried
// the expression resolve a prefixed name, and an unprefixed one resolves per the
// AXIS it was reached by — the {default namespace} for an element step, no
// namespace at all for an attribute step (PRINCIPLES 15, XPath 2.0 §3.2.1.2:
// "otherwise, it has no namespace URI"). That asymmetry is settled at compile
// time, so matches is a comparison of ·expanded names· and carries no axis of
// its own.
type nameTest struct {
	space    string
	local    string
	anyLocal bool // 'NCName:*'
	anySpace bool // bare '*', which is any namespace AND any local name
	// unresolved marks a NameTest whose prefix had no binding in the record's
	// {namespace bindings}. It matches NOTHING and is never evaluated — compile
	// discards the tree it sits in — and it is a field rather than a reserved
	// space/local pair because no namespace URI and no local name is uninhabited,
	// so neither can stand in for a name that resolved to none.
	unresolved bool
}

// matches reports whether the test admits the ·expanded name· n.
func (t nameTest) matches(n xsd.QName) bool {
	if t.unresolved {
		return false
	}
	if t.anySpace {
		return true
	}
	if t.space != n.Space {
		return false
	}
	return t.anyLocal || t.local == n.Local
}

// path is one Path of production [2] or [7], with the `.` self steps already
// removed: a self step selects the node the path is already at, so it changes
// nothing about which nodes the path selects, and keeping it would leave every
// step index off by an amount the matcher would have to re-derive.
//
// A path with no steps left and no attr selects its own CONTEXT node (`.`); one
// with no steps and an attr selects an attribute of the context node (`@id`).
// anyDepth with no steps is declined at compile time rather than modeled:
// `.//.` is descendant-or-self over every node kind, which production [3] does
// not otherwise reach and which this package would have to guess at.
type path struct {
	anyDepth bool
	steps    []nameTest
	attr     nameTest
	hasAttr  bool
}
