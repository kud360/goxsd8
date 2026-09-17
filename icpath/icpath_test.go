package icpath

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
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
		{"a//b", true, "'//' is admitted only as the leading './/'"},
		{"a[1]", true, "predicates are outside the subset"},
		{"child::a", true, "an explicit axis is outside the subset"},
		{"/a", true, "an absolute path is outside the subset"},
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
