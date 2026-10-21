package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// Each {test} CompileAssertionTest declines carries the reason its *Decline
// names, verbatim: what the compile declined first — a function, an
// expression kind, an operand type, a static error by its code and fact, or,
// where no production named a construct, the token the parse stopped at.
// Every row fails with the reason dropped, since no want is empty.
func TestCompileAssertionTestDeclineReason(t *testing.T) {
	uses := asUses(t, map[string]string{"x": "string", "d": "date", "b": "boolean"})
	for _, tc := range []struct{ expr, want string }{
		// An unsupported function, named by its fn:-prefixed name.
		{"unparsed-text('x') eq ''", "function fn:unparsed-text is not supported"},
		// The function is judged before its arguments are read, so the
		// unsupported fn:matches is named, not the ',' after its first
		// argument, and not the quantified expression after it.
		{"matches(string(@x), '^a', 'x') or (some $i in (1,2) satisfies $i eq 3)", "function fn:matches is not supported"},
		// An unsupported expression kind: a quantified expression over a
		// binding sequence no form this engine evaluates admits.
		{"some $i in (1,2) satisfies $i eq 3", "this form of quantified expression is not supported"},
		// An unsupported operand type: B.2 types `xs:date + xs:dayTimeDuration`,
		// which this engine computes nothing in.
		{"xs:date(@d) + xs:dayTimeDuration('P1D') gt xs:date('2000-01-01')", "arithmetic over an operand of type xs:date and an operand of type xs:dayTimeDuration is not supported"},
		{"xs:date(@b) eq xs:date('2000-01-01')", "a cast from an operand of type xs:boolean to xs:date is not supported"},
		// A listed function in a form it is not evaluated in.
		{"contains(@x, 'a', 'b')", "this call to fn:contains is not supported in this form"},
		{"ancestor::a", "the ancestor axis is not supported in this position"},
		// A static error is named by its code and fact, never as a limitation.
		{"@p:x eq 'a'", `it carries the XPath static error err:XPST0081: no statically-known namespace binding for prefix "p", which err:XPST0081 forbids`},
		{"1 eq", `it carries the XPath static error err:XPST0003: "1 eq" is not an XPath 2.0 expression: the grammar needs more than its end, which err:XPST0003 forbids`},
		// No production names a construct: the token the parse stopped at.
		{"(1, 2)", `the expression is outside the XPath subset this engine evaluates, which stops reading it at ","`},
	} {
		test, declined := CompileAssertionTest(asRecord(tc.expr), seededTypes, xsd.EmptyContent{}, uses, asNoElems)
		if declined == nil {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined because %s", tc.expr, tc.want)
			continue
		}
		if got := declined.String(); got != tc.want {
			t.Errorf("CompileAssertionTest(%q) declined because %q, want %q", tc.expr, got, tc.want)
		}
		if test.Tally() != nil || test.ReadsContextItem() {
			t.Errorf("CompileAssertionTest(%q): declined with a non-zero AssertionTest", tc.expr)
		}
	}
}

// A {test} CompileAssertionTest compiles returns a nil *Decline.
func TestCompileAssertionTestCompiledHasNoDecline(t *testing.T) {
	if _, declined := CompileAssertionTest(asRecord("@x eq 'a'"), seededTypes, xsd.EmptyContent{}, asUses(t, map[string]string{"x": "string"}), asNoElems); declined != nil {
		t.Errorf(`CompileAssertionTest("@x eq 'a'") declined because %s, want compiled`, declined)
	}
}
