package icpath

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These fixtures drive the compiler and the matcher directly, so a production of
// §3.11.6.2/§3.11.6.3 can be pinned without a schema and an instance around it.
// The evaluation the walk actually performs is pinned by
// validate/cvcidentityconstraint_test.go.

// exprOf compiles one expression under the given prefix bindings and default
// namespace, failing the test when the subset declines it.
func exprOf(t *testing.T, expr string, field bool, def *string, bindings ...xsd.NamespaceBinding) Expr {
	t.Helper()
	x, ok := compileOf(expr, field, def, bindings...)
	if !ok {
		t.Fatalf("compiling %q (field=%v) declined; want a compiled expression", expr, field)
	}
	return x
}

// compileOf routes one expression through the façade its kind names, so the
// fixtures below exercise the exported entry points and never the shared core
// behind them.
func compileOf(expr string, field bool, def *string, bindings ...xsd.NamespaceBinding) (Expr, bool) {
	x := xsd.NewXPathExpression(expr, bindings, def, nil)
	if field {
		return CompileField(x)
	}
	return CompileSelector(x)
}

// walk runs one compiled expression down a path of element names from its
// context node, reporting what was selected at the LAST name.
func walk(x Expr, names ...xsd.QName) Selection {
	if len(names) == 0 {
		return x.Self()
	}
	live := x.Start()
	var sel Selection
	for _, n := range names {
		live, sel = live.Advance(n)
	}
	return sel
}

func TestSelectorSubsetSelectsByDepth(t *testing.T) {
	a, b := xsd.QName{Local: "a"}, xsd.QName{Local: "b"}

	// Production [2] with no './/' prefix fixes the depth exactly.
	x := exprOf(t, "a/b", false, nil)
	if walk(x, a).SelectsElement() {
		t.Error(`"a/b" selected at depth 1; want a selection at depth 2 only`)
	}
	if !walk(x, a, b).SelectsElement() {
		t.Error(`"a/b" selected nothing at a/b; want the b element`)
	}
	if walk(x, b, b).SelectsElement() {
		t.Error(`"a/b" selected at b/b; want no selection`)
	}

	// './/' re-seeds the first step at every level, so the same path matches at
	// any depth at or below the context node's children.
	deep := exprOf(t, ".//a/b", false, nil)
	if !walk(deep, a, b).SelectsElement() {
		t.Error(`".//a/b" selected nothing at a/b; want the b element`)
	}
	if !walk(deep, b, a, b).SelectsElement() {
		t.Error(`".//a/b" selected nothing at b/a/b; want the b element`)
	}

	// '.' is the context node itself, and a '.' step anywhere is a no-op.
	if !walk(exprOf(t, ".", false, nil)).SelectsElement() {
		t.Error(`"." selected nothing at its context node; want the context node`)
	}
	if !walk(exprOf(t, "./a/.", false, nil), a).SelectsElement() {
		t.Error(`"./a/." selected nothing at a; want the a element`)
	}
}

// A live set holds each step index ONCE per path, at every depth. It is the
// './/' re-seed that could break it — the seed and the carried set both naming
// step 0 — and a set that doubles per level makes a deep document cost
// exponentially rather than linearly in its depth.
func TestPathLiveSetHoldsEachStepIndexOnce(t *testing.T) {
	a := xsd.QName{Local: "a"}
	x := exprOf(t, ".//a/a/a", false, nil)

	live := x.Start()
	for depth := 1; depth <= 12; depth++ {
		live, _ = live.Advance(a)
		seen := map[int]bool{}
		for _, j := range live.step[0] {
			if seen[j] {
				t.Fatalf("depth %d: live set %v holds step %d twice", depth, live.step[0], j)
			}
			seen[j] = true
		}
	}
	// Every prefix of a/a/a matched at every depth, so the deepest step is live.
	if !walk(x, a, a, a).SelectsElement() {
		t.Error(`".//a/a/a" selected nothing at a/a/a; want the third a`)
	}
	if walk(x, a, a).SelectsElement() {
		t.Error(`".//a/a/a" selected at a/a; want a selection at depth 3 or more`)
	}
}

func TestSelectorSubsetUnionAndWildcards(t *testing.T) {
	a := xsd.QName{Local: "a"}
	pa := xsd.QName{Space: "urn:p", Local: "a"}

	if !walk(exprOf(t, "b|a", false, nil), a).SelectsElement() {
		t.Error(`"b|a" selected nothing at a; want the a element`)
	}
	if !walk(exprOf(t, "*", false, nil), pa).SelectsElement() {
		t.Error(`"*" selected nothing at {urn:p}a; want any element`)
	}
	star := exprOf(t, "p:*", false, nil, xsd.NewNamespaceBinding("p", "urn:p"))
	if !walk(star, pa).SelectsElement() {
		t.Error(`"p:*" selected nothing at {urn:p}a; want any element of urn:p`)
	}
	if walk(star, a).SelectsElement() {
		t.Error(`"p:*" selected the no-namespace a; want urn:p only`)
	}
}

// The default namespace applies to an ELEMENT step and never to an attribute
// step (PRINCIPLES 15, XPath 2.0 §3.2.1.2). This is the compiler's half of that
// asymmetry; validate/cvcidentityconstraint_test.go pins the evaluated half.
func TestFieldSubsetDefaultNamespaceStopsAtTheAttributeAxis(t *testing.T) {
	def := "urn:p"
	x := exprOf(t, "a/@id", true, &def)

	sel := walk(x, xsd.QName{Space: "urn:p", Local: "a"})
	if !sel.SelectsAttributes() {
		t.Fatal("a/@id selected no attribute at {urn:p}a, want the id attribute")
	}
	if !sel.SelectsAttribute(xsd.QName{Local: "id"}) {
		t.Error("@id did not match the no-namespace id; the default namespace must not reach the attribute axis")
	}
	if sel.SelectsAttribute(xsd.QName{Space: "urn:p", Local: "id"}) {
		t.Error("@id matched {urn:p}id; the default namespace must not reach the attribute axis")
	}
	if walk(x, xsd.QName{Local: "a"}).SelectsAttributes() {
		t.Error("a/@id selected at the no-namespace a; the element step must take the default namespace")
	}
}

func TestFieldSubsetSelectsAnAttributeOfTheContextNode(t *testing.T) {
	sel := walk(exprOf(t, "@id", true, nil))
	if !sel.SelectsAttribute(xsd.QName{Local: "id"}) || sel.SelectsElement() {
		t.Fatalf("@id selected %v at its context node, want the no-namespace id alone", sel)
	}
}

// Everything outside the two productions declines, and declining is what keeps
// the constraint carrying it from charging (validate's icFrame.declined GAP).
func TestPathSubsetDeclinesWhatItDoesNotAdmit(t *testing.T) {
	cases := []struct {
		expr  string
		field bool
		why   string
	}{
		{"@id", false, "a selector may not name an attribute (production [2] has no '@')"},
		{"@id/a", true, "an attribute step is only ever the FINAL step (production [7])"},
		{"a//b", true, "'//' is admitted only as the leading './/', and a non-initial one is charged"},
		{"a[1]", true, "predicates are outside the subset"},
		{"child::node()", true, "a KindTest step is charged after the child axis"},
		{"attribute::node()", true, "and after the attribute axis"},
		{"@node()", true, "and after an '@'"},
		{"node()", false, "and as an abbreviated step"},
		{"foo::a", true, "'foo' is none of the thirteen axis keywords"},
		{"::a", true, "a '::' with no axis keyword before it"},
		{"child::a/text()", false, "and so is every other argument-free KindTest"},
		{"a/element(b)", false, "a KindTest with an argument does not lex at all"},
		{"/a", true, "a root-relative path is charged"},
		{"a/", true, "a trailing '/' has no Step after it"},
		{"", true, "an empty expression is no Path at all"},
		{".//.", true, "'.//' with no element step left is not modeled"},
		{"q:a", true, "an unbound prefix cannot be resolved"},
	}
	for _, c := range cases {
		if _, ok := compileOf(c.expr, c.field, nil); ok {
			t.Errorf("compiling %q (field=%v) succeeded; want a decline: %s", c.expr, c.field, c.why)
		}
	}
}

// '.' is a legal NCName character after the first, so longest-token tokenizing
// makes "a.b" one NameTest rather than a name, a self step and a second name.
func TestPathSubsetTokenizesADottedNameAsOneNameTest(t *testing.T) {
	if !walk(exprOf(t, "a.b", false, nil), xsd.QName{Local: "a.b"}).SelectsElement() {
		t.Error(`"a.b" selected nothing at a.b; want one NameTest`)
	}
	if walk(exprOf(t, "a.b", false, nil), xsd.QName{Local: "a"}).SelectsElement() {
		t.Error(`"a.b" selected at a; want one NameTest and not a self step`)
	}
}

// TestScanNCNameIsTheXMLNameClass pins the NCName scan against XML's
// NameStartChar and NameChar — the classes Datatypes §G.4.2.5 defines \i and \c
// to be — and not against Unicode's letter and digit categories. The four rows
// written as code-point escapes are the boundaries the two classes draw
// differently; each of them is where one NameTest ends and the next token of
// production [4] begins, so the prefix a binding is looked up under depends on
// drawing it exactly.
func TestScanNCNameIsTheXMLNameClass(t *testing.T) {
	for _, tc := range []struct {
		s    string
		want int
		why  string
	}{
		{"a", 1, "an ASCII name is the whole of it"},
		{"a.b-c_d", 7, "'.', '-' and '_' continue a name"},
		{"_x", 2, "'_' opens one"},
		{"élan", 5, "U+00E9 is a NameStartChar and a letter both"},
		{"日本", 6, "and so is U+65E5"},
		{"9x", 0, "a digit opens no name"},
		{"/x", 0, "nor does a step separator"},
		{"", 0, "nor does the empty string"},
		{"p:*", 1, "':' is subtracted from both classes: [\\i-[:]][\\c-[:]]*"},
		{"µx", 0, "U+00B5 MICRO SIGN is a Unicode letter and NO NameStartChar"},
		{"aª", 1, "U+00AA is a Unicode letter and NO NameChar"},
		{"a·b", 4, "U+00B7 MIDDLE DOT is a NameChar and neither letter nor digit"},
		{"áb", 4, "and so is U+0301 COMBINING ACUTE ACCENT"},
	} {
		if got := scanNCName(tc.s, 0); got != tc.want {
			t.Errorf("scanNCName(%q, 0) = %d, want %d (%s)", tc.s, got, tc.want, tc.why)
		}
	}
}

// TestNCNamePatternPinned fails when ncNamePattern stops being xs:NCName's
// pattern facet as the generated builtin row carries it (Datatypes §3.4.7.1),
// behind the '^' the FO prefix scan adds.
func TestNCNamePatternPinned(t *testing.T) {
	if want := "^" + builtin.NCNamePattern(); ncNamePattern != want {
		t.Errorf("ncNamePattern = %q, want %q (\"^\" + builtin.NCNamePattern())", ncNamePattern, want)
	}
}

// TestPathSubsetFollowsTheXMLNameClass is the same boundary reaching the
// tokenizer: a name character of production [4] is read into the NameTest it
// belongs to, and a character that is none opens no NameTest at all — which is a
// decline of the whole {expression}, not a shorter name.
func TestPathSubsetFollowsTheXMLNameClass(t *testing.T) {
	if !walk(exprOf(t, "a·b", false, nil), xsd.QName{Local: "a·b"}).SelectsElement() {
		t.Error(`"a·b" selected nothing at a·b; want one NameTest — U+00B7 continues the name`)
	}
	if _, ok := compileOf("µ", false, nil); ok {
		t.Error(`compiling "µ" succeeded; want a decline — U+00B5 opens no NCName`)
	}
}

// A charged {expression} is also a DECLINE. The two answers are independent:
// parser charges over a schema DOCUMENT, and a component assembled directly
// through xsd.NewIdentityConstraint reaches no assembler at all, so the matcher
// must still refuse what the SCC rejects rather than match a tree built over a
// name that did not resolve.
func TestChargedExpressionsAlsoDecline(t *testing.T) {
	for _, tc := range []struct {
		expr  string
		field bool
	}{
		{"q:a", false}, {"q:a", true}, {"a[b]", true}, {"@x/b", true}, {"@x", false},
	} {
		if _, ok := compileOf(tc.expr, tc.field, nil); ok {
			t.Errorf("compiling %q (field=%v) succeeded; a charged {expression} must decline too", tc.expr, tc.field)
		}
	}
}

// The '@' scan is PER UNION MEMBER, because production [1] is a union of Paths
// and a member's final step is its own. `a/@b|c/@d` is two legal field Paths and
// reads as one illegal one if the '|' is ignored — which is the shape the W3C
// suite's own field expressions are written in.
func TestFieldSubsetJudgesEachUnionMemberSFinalStep(t *testing.T) {
	legal := xsd.NewXPathExpression("a/@b|c/@d", nil, nil, nil)
	if err := FieldViolation(xsderr.Loc{}, legal); err != nil {
		t.Errorf("FieldViolation(%q) = %v, want nil — each member ends in its own attribute step", legal.Expression(), err)
	}
	illegal := xsd.NewXPathExpression("a/@b/e|c", nil, nil, nil)
	if err := FieldViolation(xsderr.Loc{}, illegal); err == nil {
		t.Errorf("FieldViolation(%q) = nil, want a charge — the '@' is not that member's final step", illegal.Expression())
	}
}

// UNSUPPORTED DOMINATES: an {expression} that does not reach the end of a
// complete production is declined however it continues, so neither a bracket nor
// an unbound prefix inside one is charged. Under-charging is a rejection
// validate can still make; over-charging rejects a conforming schema before any
// instance exists.
func TestPathSubsetUnsupportedDominates(t *testing.T) {
	for _, tc := range []struct {
		expr  string
		field bool
		why   string
	}{
		{"a[1]", true, "'1' opens no token, so the '[' is not read as a predicate"},
		{"a[b!='c']", true, "nor does '!=', of the six GeneralComp operators only '=' is read"},
		{"a='c'", false, "'=' and a StringLiteral are read inside a predicate alone, so a comparison outside one does not lex"},
		{"//='c'", false, "which keeps a leading '//' before a '=' from being charged as root-relative"},
		{"@='c'", false, "and a selector's '@' before one from being charged as an attribute"},
		{"//=a", false, "'=' alone outside a predicate opens no token either"},
		{"@=a", false, "nor before a selector's '@'"},
		{"@'c'", false, "and a StringLiteral alone outside one opens none"},
		{"a[b='c]", true, "an unterminated StringLiteral does not lex"},
		{"q:a/processing-instruction('x')", false, "a KindTest with an argument does not lex, so the unbound q is not read in isolation"},
		{"q:a/element(b)", true, "nor does element(b), whose argument is a QName this lexer does not read"},
		{"a/schema-element()", false, "schema-element requires an argument, so its empty parentheses do not lex either"},
		{"foo::a", false, "'foo::' spells no axis of XPath 2.0, so the stream does not lex whole"},
		{"q:a|.//.", false, "the stream lexes whole but parses to nothing, so the recorded unbound prefix is discarded"},
		{".//.", false, "production [3]'s bare '.' Step derives it — assembly-legal, only unmatchable"},
		{"child::a", false, "clause 2.2 admits the unabbreviated form of an abbreviated path"},
		{"attribute::a", true, "and, for a field's final step, the attribute axis as well"},
		{"tid :*|a[1]", false, "a split name is charged only over a stream that lexes whole"},
	} {
		if err := violationOf(tc.expr, tc.field); err != nil {
			t.Errorf("charging %q (field=%v) = %v, want nil — %s", tc.expr, tc.field, err, tc.why)
		}
	}
}

// violationOf routes one expression through the charging façade its kind names.
func violationOf(expr string, field bool) error {
	x := xsd.NewXPathExpression(expr, nil, nil, nil)
	if field {
		return FieldViolation(xsderr.Loc{}, x)
	}
	return SelectorViolation(xsderr.Loc{}, x)
}

// An '@' with no NameTest after it is charged on both arms, under different
// clauses. A SELECTOR is charged for naming the attribute axis at all, which
// clause 2.2 withholds from it whatever follows. A FIELD is charged under clause
// 1: production [31] AbbrevForwardStep requires a NodeTest after the '@', so the
// member is no XPath 2.0 expression. `@*` and `@p:*` carry a NameTest and stay
// legal; `@b/` has one and is still the position charge. The message is pinned
// by its opening, because a lone '@' fails the position check too and only the
// message tells which branch charged it.
func TestBareAttributeAxisIsCharged(t *testing.T) {
	for _, tc := range []struct {
		expr  string
		field bool
		msg   string
	}{
		{"@", false, `the {selector} "@" names an attribute, but c-selector-xpath clause 2`},
		{"@", true, `the {fields} member "@" has an '@' with no NodeTest after it, but c-fields-xpaths clause 1`},
		{"a/@", true, `the {fields} member "a/@" has an '@' with no NodeTest after it`},
		{"@/a", true, `the {fields} member "@/a" has an '@' with no NodeTest after it`},
		{"@.", true, `the {fields} member "@." has an '@' with no NodeTest after it`},
		{"a/@b|@", true, `the {fields} member "a/@b|@" has an '@' with no NodeTest after it`},
		{"@b/", true, `the {fields} member "@b/" names an attribute before its final step`},
	} {
		err := violationOf(tc.expr, tc.field)
		var e *xsderr.Error
		if !errors.As(err, &e) {
			t.Errorf("charging %q (field=%v) = %v, want an *xsderr.Error", tc.expr, tc.field, err)
			continue
		}
		if !strings.HasPrefix(e.Msg, tc.msg) {
			t.Errorf("charging %q (field=%v): message = %q, want it to open %q", tc.expr, tc.field, e.Msg, tc.msg)
		}
		if _, ok := compileOf(tc.expr, tc.field, nil); ok {
			t.Errorf("compiling %q (field=%v) succeeded; want a decline", tc.expr, tc.field)
		}
	}
	p := []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:p")}
	for _, expr := range []string{"@*", "@p:*", "a/@*"} {
		if err := FieldViolation(xsderr.Loc{}, xsd.NewXPathExpression(expr, p, nil, nil)); err != nil {
			t.Errorf("FieldViolation(%q) = %v, want nil — '*' and 'NCName:*' are NameTests (production [4])", expr, err)
		}
	}
}

// A root-relative path, a non-initial '//' and a KindTest step each fail both
// arms of clause 2 whatever names fill them, and are charged. The message is
// pinned whole up to its clause and reason, so a charge reached by another arm
// — the parse's decline, the axis charge, the attribute charge — fails the row.
// Where a member holds two faults the leftmost is charged: `self::node()` keeps
// its axis charge, and a selector's `/@a` is charged for its '/'.
func TestRootRelativeNonInitialDescendantAndKindTestAreCharged(t *testing.T) {
	const (
		selRoot  = `, but c-selector-xpath clause 2 admits it under neither arm — xpath20 production [25] expands a leading %q to a step from the root, and production [2]'s Path is context-relative, its only leading '//' the './/' pair`
		fldRoot  = `, but c-fields-xpaths clause 2 admits it under neither arm — xpath20 production [25] expands a leading %q to a step from the root, and production [7]'s Path is context-relative, its only leading '//' the './/' pair`
		selDesc  = `has a non-initial '//', but c-selector-xpath clause 2 admits it under neither arm — xpath20 §3.2.4 rule 3 expands it to /descendant-or-self::node()/, and production [2] admits '//' only in its leading './/' pair`
		fldDesc  = `has a non-initial '//', but c-fields-xpaths clause 2 admits it under neither arm — xpath20 §3.2.4 rule 3 expands it to /descendant-or-self::node()/, and production [7] admits '//' only in its leading './/' pair`
		selKind  = `, but c-selector-xpath clause 2 admits it under neither arm — production [3] Step is '.' or a NameTest, and xpath20 production [35] makes a KindTest no NameTest`
		fldKind  = `, but c-fields-xpaths clause 2 admits it under neither arm — production [3] Step is '.' or a NameTest and production [7]'s final step is '@' NameTest, and xpath20 production [35] makes a KindTest no NameTest`
		selector = `the {selector} %q `
		fieldM   = `the {fields} member %q `
	)
	root := func(expr, sp string, field bool) string {
		if field {
			return fmt.Sprintf(fieldM+"is root-relative (it opens with %q)"+fldRoot, expr, sp, sp)
		}
		return fmt.Sprintf(selector+"is root-relative (it opens with %q)"+selRoot, expr, sp, sp)
	}
	desc := func(expr string, field bool) string {
		if field {
			return fmt.Sprintf(fieldM+fldDesc, expr)
		}
		return fmt.Sprintf(selector+selDesc, expr)
	}
	kind := func(expr, kt string, field bool) string {
		if field {
			return fmt.Sprintf(fieldM+"has a KindTest step %q"+fldKind, expr, kt)
		}
		return fmt.Sprintf(selector+"has a KindTest step %q"+selKind, expr, kt)
	}
	type row struct {
		expr  string
		field bool
		msg   string
	}
	var rows []row
	for _, field := range []bool{false, true} {
		rows = append(rows,
			row{"/a", field, root("/a", "/", field)},
			row{"/", field, root("/", "/", field)},
			row{"//a", field, root("//a", "//", field)},
			row{"/./a", field, root("/./a", "/", field)},
			row{"b|/a", field, root("b|/a", "/", field)},
			row{"a//b", field, desc("a//b", field)},
			row{"a/.//b", field, desc("a/.//b", field)},
			row{".//a//b", field, desc(".//a//b", field)},
			row{"a //.", field, desc("a //.", field)},
			row{"a/text()", field, kind("a/text()", "text()", field)},
			row{"text()", field, kind("text()", "text()", field)},
			row{"a/comment ( )", field, kind("a/comment ( )", "comment()", field)},
			row{"processing-instruction()", field, kind("processing-instruction()", "processing-instruction()", field)},
			row{"document-node()", field, kind("document-node()", "document-node()", field)},
			row{"a/element()", field, kind("a/element()", "element()", field)},
			row{"child::node()", field, kind("child::node()", "node()", field)},
			row{"node()", field, kind("node()", "node()", field)},
			row{"q:a/text()", field, kind("q:a/text()", "text()", field)},
		)
	}
	rows = append(rows,
		row{"attribute::node()", true, kind("attribute::node()", "node()", true)},
		row{"@node()", true, kind("@node()", "node()", true)},
		row{"a/attribute()", true, kind("a/attribute()", "attribute()", true)},
		row{"@text()", true, kind("@text()", "text()", true)},
		row{"attribute::node()", false, `the {selector} "attribute::node()" names an attribute, but c-selector-xpath clause 2`},
		row{"self::node()", false, `the {selector} "self::node()" steps along the self axis`},
		row{"self::text()", true, `the {fields} member "self::text()" steps along the self axis`},
		row{"/@a", false, root("/@a", "/", false)},
		row{"a//@b", true, desc("a//@b", true)},
	)
	for _, tc := range rows {
		err := violationOf(tc.expr, tc.field)
		var e *xsderr.Error
		if !errors.As(err, &e) {
			t.Errorf("charging %q (field=%v) = %v, want an *xsderr.Error", tc.expr, tc.field, err)
			continue
		}
		if !strings.HasPrefix(e.Msg, tc.msg) {
			t.Errorf("charging %q (field=%v): message = %q, want it to open %q", tc.expr, tc.field, e.Msg, tc.msg)
		}
		if _, ok := compileOf(tc.expr, tc.field, nil); ok {
			t.Errorf("compiling %q (field=%v) succeeded; a charged {expression} must decline too", tc.expr, tc.field)
		}
	}
}

// An empty union member, a '/' or '//' with no Step after it and a name split
// by white space are no XPath 2.0 expression at all, and are charged under
// clause 1. The message is pinned by its opening up to the clause, so a charge
// reached by another arm — root-relative, non-initial '//', unbound prefix —
// fails the row. `tid` and `imp` are unbound throughout: the grammar charge
// wraps no cause, so a split name read as a prefix and charged err:XPST0081
// fails the row too. Where a member holds several faults the leftmost is
// charged: `//a//` for its leading '//', `/ /a` for its first '/'.
func TestNonExpressionsAreChargedUnderClause1(t *testing.T) {
	msg := func(expr, fault string, field bool) string {
		if field {
			return fmt.Sprintf("the {fields} member %q %s, but c-fields-xpaths clause 1 requires it to satisfy xpath-valid", expr, fault)
		}
		return fmt.Sprintf("the {selector} %q %s, but c-selector-xpath clause 1 requires it to satisfy xpath-valid", expr, fault)
	}
	const empty = "has an empty operand where a Path is required"
	sep := func(sp string) string { return fmt.Sprintf("has a %q with no Step after it", sp) }
	split := func(name string) string { return fmt.Sprintf("has a name %q split by white space", name) }
	type row struct {
		expr  string
		field bool
		fault string
	}
	var rows []row
	for _, field := range []bool{false, true} {
		rows = append(rows,
			row{"", field, empty},
			row{"  ", field, empty},
			row{"|", field, empty},
			row{"| imp:iid", field, empty},
			row{"a||b", field, empty},
			row{"a|", field, empty},
			row{"//", field, sep("//")},
			row{"a//", field, sep("//")},
			row{".//", field, sep("//")},
			row{"a///b", field, sep("//")},
			row{"a////b", field, sep("//")},
			row{"a/ //b", field, sep("/")},
			row{"////a", field, sep("//")},
			row{"/ /a", field, sep("/")},
			row{"//|a", field, sep("//")},
			row{"./ /.", field, sep("/")},
			row{"a/", field, sep("/")},
			row{"tid :*", field, split("tid :*")},
			row{"tid: *", field, split("tid: *")},
			row{"tid : *", field, split("tid : *")},
			row{"tid : x", field, split("tid : x")},
			row{"child: :imp:iid", field, split("child: :")},
			row{"attribute: :imp:sid", field, split("attribute: :")},
			row{"a/tid :*", field, split("tid :*")},
			row{"tid :*|//", field, split("tid :*")},
			row{"a//|tid :*", field, sep("//")},
		)
	}
	rows = append(rows, row{".///@*", true, sep("//")})
	for _, tc := range rows {
		err := violationOf(tc.expr, tc.field)
		var e *xsderr.Error
		if !errors.As(err, &e) {
			t.Errorf("charging %q (field=%v) = %v, want an *xsderr.Error", tc.expr, tc.field, err)
			continue
		}
		if want := msg(tc.expr, tc.fault, tc.field); !strings.HasPrefix(e.Msg, want) {
			t.Errorf("charging %q (field=%v): message = %q, want it to open %q", tc.expr, tc.field, e.Msg, want)
		}
		if cause := errors.Unwrap(err); cause != nil {
			t.Errorf("charging %q (field=%v) wraps %v; a grammar charge wraps no cause", tc.expr, tc.field, cause)
		}
		if _, ok := compileOf(tc.expr, tc.field, nil); ok {
			t.Errorf("compiling %q (field=%v) succeeded; a charged {expression} must decline too", tc.expr, tc.field)
		}
	}
	for _, field := range []bool{false, true} {
		err := violationOf("//a//", field)
		if err == nil || !strings.Contains(err.Error(), `is root-relative (it opens with "//")`) {
			t.Errorf("charging %q (field=%v) = %v, want the leftmost fault, its leading '//', charged as root-relative", "//a//", field, err)
		}
	}
}

// What the clause-1 charges above do NOT reach, each stays uncharged. The
// leading './/' pair before a Step is admitted, and a '//' with a Step after it
// is no separator fault. Two Steps with no separator are left undecided,
// because the lexer reads the legal `..` as the same two '.' tokens as `. .`
// (#1829). A legal Wildcard `*:a` and colon residue a split name does not spell
// open a token this lexer does not read, so the stream is declined whatever else
// it holds. `child :: a` is an axis head, which is read before any split name.
func TestTheLeadingPairStepAdjacencyAndUnreadableRunsAreNotCharged(t *testing.T) {
	p := []xsd.NamespaceBinding{
		xsd.NewNamespaceBinding("xpns", "urn:x"),
		xsd.NewNamespaceBinding("xpns1", "urn:x1"),
	}
	for _, expr := range []string{
		". //.", "xpns1:* | .//xpns:*/.", ".//a",
		"..", ". .", "a b",
		"*:a", "(: tid :* :)", "a:b :c", "p:", ":a",
		"child :: a",
	} {
		for _, field := range []bool{false, true} {
			x := xsd.NewXPathExpression(expr, p, nil, nil)
			if err := violationAt(xsderr.Loc{}, x, field); err != nil {
				t.Errorf("charging %q (field=%v) = %v, want nil", expr, field, err)
			}
		}
	}
}

// The split-name reader takes exactly its two runs, whatever follows them: an
// axis head with white space around an unsplit '::' stays an axis head, an
// unsplit QName or Wildcard stays a NameTest, and colon residue it does not
// spell opens no token.
func TestTokenizeReadsASplitNameAsOneToken(t *testing.T) {
	for _, tc := range []struct {
		s    string
		want []token
	}{
		{"tid :*", []token{{kind: 'W', text: "tid :*"}}},
		{"tid : x/a", []token{{kind: 'W', text: "tid : x"}, {kind: '/'}, {kind: 'n', text: "a"}}},
		{"child: :imp:iid", []token{{kind: 'W', text: "child: :"}, {kind: 'n', text: "imp:iid"}}},
		{"child :: a", []token{{kind: 'C', text: "child"}, {kind: 'n', text: "a"}}},
		{"tid:*", []token{{kind: 'n', text: "tid:*"}}},
		{"a:b :c", []token{{kind: 'n', text: "a:b"}, {kind: '?'}, {kind: 'n', text: "c"}}},
	} {
		got := tokenize(tc.s)
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// A NameTest whose prefix did not resolve matches NOTHING. No compiled tree
// holds one today — compile discards the tree an unbound prefix sits in and
// reports a violation over it instead — so this is the unit-level proof that the
// defence behind that stands on its own.
func TestUnresolvedNameTestMatchesNothing(t *testing.T) {
	var r names
	got := r.test("q:a", false)
	if !got.unresolved {
		t.Fatalf(`test("q:a") = %#v, want an unresolved NameTest`, got)
	}
	if !r.hasUnbound || r.unbound != "q" {
		t.Errorf("recorded unbound prefix %q (present=%v), want q", r.unbound, r.hasUnbound)
	}
	if got.matches(xsd.QName{Local: "a"}) {
		t.Error("an unresolved NameTest matched the no-namespace a; it must match nothing")
	}
	if got.matches(xsd.QName{Space: "urn:q", Local: "a"}) {
		t.Error("an unresolved NameTest matched {urn:q}a; it must match nothing")
	}
}

// A FunctionCall is no Step of production [3], and is charged under clause 2 of
// either SCC wherever it sits in a member. The message is pinned whole from its
// opening, so the predicate charge, the root-relative charge or a decline fails
// the row. A FunctionCall whose NodeTest-less head precedes it is charged
// leftmost, under clause 1: `child::f('x')` and a field's `@f('x')` are no XPath
// 2.0 expression at all.
func TestFunctionCallIsCharged(t *testing.T) {
	const (
		sel = `the {selector} %q has a FunctionCall %q, but c-selector-xpath clause 2 admits it under neither arm — production [3] Step is '.' or a NameTest, and xpath20 production [27] makes a FunctionCall a FilterExpr and no AxisStep`
		fld = `the {fields} member %q has a FunctionCall %q, but c-fields-xpaths clause 2 admits it under neither arm — production [3] Step is '.' or a NameTest and production [7]'s final step is '@' NameTest, and xpath20 production [27] makes a FunctionCall a FilterExpr and no AxisStep`
	)
	call := func(expr, fc string, field bool) string {
		if field {
			return fmt.Sprintf(fld, expr, fc)
		}
		return fmt.Sprintf(sel, expr, fc)
	}
	type row struct {
		expr  string
		field bool
		msg   string
	}
	var rows []row
	for _, field := range []bool{false, true} {
		rows = append(rows,
			row{"document('')", field, call("document('')", "document('')", field)},
			row{`document("")`, field, call(`document("")`, `document("")`, field)},
			row{"document ( '' )", field, call("document ( '' )", "document ( '' )", field)},
			row{"f()", field, call("f()", "f()", field)},
			row{`concat('a', "b" ,'c''d')`, field, call(`concat('a', "b" ,'c''d')`, `concat('a', "b" ,'c''d')`, field)},
			row{"a/document('')", field, call("a/document('')", "document('')", field)},
			row{"self('x')", field, call("self('x')", "self('x')", field)},
			row{"q:a/f('[')", field, call("q:a/f('[')", "f('[')", field)},
			row{"child::f('x')", field, fmt.Sprintf(`the %s "child::f('x')" has an axis step "child::" with no NodeTest after it, but %s clause 1`, subject(field), sccOf(field))},
		)
	}
	rows = append(rows,
		row{"@f('x')", true, `the {fields} member "@f('x')" has an '@' with no NodeTest after it, but c-fields-xpaths clause 1`},
		row{"a/@f('x')", true, `the {fields} member "a/@f('x')" has an '@' with no NodeTest after it, but c-fields-xpaths clause 1`},
	)
	for _, tc := range rows {
		err := violationOf(tc.expr, tc.field)
		var e *xsderr.Error
		if !errors.As(err, &e) {
			t.Errorf("charging %q (field=%v) = %v, want an *xsderr.Error", tc.expr, tc.field, err)
			continue
		}
		if !strings.HasPrefix(e.Msg, tc.msg) {
			t.Errorf("charging %q (field=%v): message = %q, want it to open %q", tc.expr, tc.field, e.Msg, tc.msg)
		}
		if cause := errors.Unwrap(err); cause != nil {
			t.Errorf("charging %q (field=%v) wraps %v; a grammar charge wraps no cause", tc.expr, tc.field, cause)
		}
		if _, ok := compileOf(tc.expr, tc.field, nil); ok {
			t.Errorf("compiling %q (field=%v) succeeded; a charged {expression} must decline too", tc.expr, tc.field)
		}
	}
}

// A predicate whose '=' comparison has a StringLiteral operand now lexes whole,
// and is charged as the predicate it is — the W3C suite's idI151 and idJ209
// shapes, with `imp` bound and unbound alike: the predicate is decided from the
// tokens before any prefix resolves, so the charge wraps no err:XPST0081. A
// quoted bracket is a literal's content and not a second predicate bracket.
func TestPredicateWithAStringComparisonIsCharged(t *testing.T) {
	const pred = `carries a predicate, but %s clause 2 admits none — production [3] Step is '.' or a NameTest, and abbreviating an unabbreviated XPath does not drop a predicate`
	imp := []xsd.NamespaceBinding{xsd.NewNamespaceBinding("imp", "urn:imp")}
	for _, bindings := range [][]xsd.NamespaceBinding{nil, imp} {
		for _, tc := range []struct {
			expr  string
			field bool
		}{
			{`imp:iid[type="predicate"]`, false},
			{`imp:sid[type="predicate"]`, true},
			{`a[b = 'x']`, false},
			{`a[@b='x']`, true},
			{`a[b="]"]`, false},
			{`a['x'=b]/c`, true},
			{`a[b='it''s']`, true},
			{`a[f('x')]`, false},
		} {
			x := xsd.NewXPathExpression(tc.expr, bindings, nil, nil)
			err := violationAt(xsderr.Loc{}, x, tc.field)
			var e *xsderr.Error
			if !errors.As(err, &e) {
				t.Errorf("charging %q (field=%v, bindings=%d) = %v, want an *xsderr.Error", tc.expr, tc.field, len(bindings), err)
				continue
			}
			want := fmt.Sprintf("the %s %q "+pred, subject(tc.field), tc.expr, sccOf(tc.field))
			if e.Msg != want {
				t.Errorf("charging %q (field=%v, bindings=%d): message = %q, want %q", tc.expr, tc.field, len(bindings), e.Msg, want)
			}
			if cause := errors.Unwrap(err); cause != nil {
				t.Errorf("charging %q (field=%v, bindings=%d) wraps %v; the predicate charge wraps no cause", tc.expr, tc.field, len(bindings), cause)
			}
		}
	}
}

// What the FunctionCall reader does not read stays declined: a prefixed name,
// whose prefix nothing here resolves; a reserved function name, which before a
// '(' is a KindTest or a keyword (xpath20 A.3); an argument other than a
// StringLiteral; a malformed argument list; and a comment after the name, which
// is no argument list at all (A.1.3, gn: parens).
func TestFunctionCallReaderDeclinesWhatItDoesNotRead(t *testing.T) {
	for _, expr := range []string{
		"p:f('x')", "if('x')", "item()", "element('a')", "processing-instruction('x')",
		"f(a)", "f(1)", "f('x'", "f('x',)", "f(,)", "f('x' 'y')", "f (: c :)", "f('x)",
	} {
		for _, field := range []bool{false, true} {
			if err := violationOf(expr, field); err != nil {
				t.Errorf("charging %q (field=%v) = %v, want nil — the call is not read, so the stream is declined", expr, field, err)
			}
		}
	}
}

// The lexer reads a FunctionCall as ONE token carrying its spelling, and '=' and
// a StringLiteral only between a predicate's brackets: outside one each opens no
// token. A doubled delimiter is production [75]/[76]'s escape and stays inside
// the literal.
func TestTokenizeReadsFunctionCallsAndPredicateLiterals(t *testing.T) {
	for _, tc := range []struct {
		s    string
		want []token
	}{
		{"document('')", []token{{kind: 'F', text: "document('')"}}},
		{"a/f( 'x' , \"y\" )", []token{{kind: 'n', text: "a"}, {kind: '/'}, {kind: 'F', text: "f( 'x' , \"y\" )"}}},
		{`a[b="x"]`, []token{{kind: 'n', text: "a"}, {kind: '['}, {kind: 'n', text: "b"}, {kind: '='}, {kind: 's', text: `"x"`}, {kind: ']'}}},
		{"a[b='it''s']", []token{{kind: 'n', text: "a"}, {kind: '['}, {kind: 'n', text: "b"}, {kind: '='}, {kind: 's', text: "'it''s'"}, {kind: ']'}}},
		{"a='x'", []token{{kind: 'n', text: "a"}, {kind: '?'}, {kind: '?'}, {kind: 'n', text: "x"}, {kind: '?'}}},
		{"a[b]='x'", []token{{kind: 'n', text: "a"}, {kind: '['}, {kind: 'n', text: "b"}, {kind: ']'}, {kind: '?'}, {kind: '?'}, {kind: 'n', text: "x"}, {kind: '?'}}},
		{"if('x')", []token{{kind: 'n', text: "if"}, {kind: '?'}, {kind: '?'}, {kind: 'n', text: "x"}, {kind: '?'}, {kind: '?'}}},
	} {
		got := tokenize(tc.s)
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}
