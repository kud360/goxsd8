package icpath

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// Clause 2.2's admitted family — a child-axis head before a NameTest, and for a
// field's final step an attribute-axis head before one — compiles to the SAME
// tree its abbreviated spelling does (xpath20 §3.2.4 rules 1 and 2), with white
// space on either side of the '::'. The tree is compared whole, because
// parsePath reaches it by taking the abbreviated twin's own arm and the equality
// is what that construction promises; the walk after it is the matcher's half.
func TestUnabbreviatedSpellingsCompileAsTheirAbbreviatedTwins(t *testing.T) {
	def := "urn:d"
	p := []xsd.NamespaceBinding{
		xsd.NewNamespaceBinding("imp", "urn:imp"),
		xsd.NewNamespaceBinding("myNS", "urn:my"),
	}
	for _, tc := range []struct {
		long, short string
		field       bool
	}{
		{"child::imp:iid", "imp:iid", false},
		{"child::myNS:*", "myNS:*", false},
		{"child::*", "*", false},
		{"child::a", "a", false},
		{"child ::imp:iid", "imp:iid", false},
		{"child:: imp:iid", "imp:iid", false},
		{"child :: a", "a", false},
		{"./child::a/.", "./a/.", false},
		{".//child::a/child::b", ".//a/b", false},
		{"child::a|child::b", "a|b", false},
		{"child::a", "a", true},
		{"attribute::imp:sid", "@imp:sid", true},
		{"attribute::*", "@*", true},
		{"attribute::myNS:*", "@myNS:*", true},
		{"attribute::a", "@a", true},
		{"attribute ::imp:sid", "@imp:sid", true},
		{"attribute:: imp:sid", "@imp:sid", true},
		{"child::a/attribute::b", "a/@b", true},
		{"child::a/@b|attribute::c", "a/attribute::b|@c", true},
	} {
		long := exprOf(t, tc.long, tc.field, &def, p...)
		short := exprOf(t, tc.short, tc.field, &def, p...)
		if !reflect.DeepEqual(long, short) {
			t.Errorf("compiling %q (field=%v) = %#v, want %#v, the tree of %q", tc.long, tc.field, long, short, tc.short)
		}
	}

	// The attribute head resolves as '@' does: an unprefixed name takes no
	// namespace, although the element step before it takes the default one.
	sel := walk(exprOf(t, "child::a/attribute::id", true, &def), xsd.QName{Space: def, Local: "a"})
	if !sel.SelectsAttribute(xsd.QName{Local: "id"}) || sel.SelectsAttribute(xsd.QName{Space: def, Local: "id"}) {
		t.Errorf("child::a/attribute::id selected %v at {urn:d}a, want the no-namespace id alone", sel)
	}
}

// axisKeywords is xpath20 production [30] ForwardAxis and production [33]
// ReverseAxis, the whole of XPath 2.0's axis vocabulary, spelled out here so the
// lexer's switch is pinned against the productions rather than against itself.
var axisKeywords = []string{
	"child", "descendant", "attribute", "self", "descendant-or-self", "following-sibling", "following", "namespace",
	"parent", "ancestor", "preceding-sibling", "preceding", "ancestor-or-self",
}

// Every axis spelling clause 2.2 does not admit is charged, and none is left a
// decline. With no NodeTest after its '::' ANY of the thirteen axes is no XPath
// 2.0 expression (productions [29] and [32]), a clause-1 fault; with one, every
// axis but child — and for a field, attribute — fails both arms of clause 2,
// since production [5] spells no axis and clause 2.2 names none but those. The
// message is pinned by its opening, which names the subject, the {expression}
// and the clause, so a charge reached by the wrong branch fails the row.
func TestUnabbreviatedResidualIsCharged(t *testing.T) {
	type row struct {
		expr  string
		field bool
		msg   string
	}
	var rows []row
	for _, field := range []bool{false, true} {
		scc, subj := "c-selector-xpath", "{selector}"
		if field {
			scc, subj = "c-fields-xpaths", "{fields} member"
		}
		for _, ax := range axisKeywords {
			for _, expr := range []string{ax + "::", "a/" + ax + "::", ax + "::/a", ax + "::.", ax + "::@a", ax + "::child::a"} {
				rows = append(rows, row{expr, field, fmt.Sprintf(`the %s %q has an axis step %q with no NodeTest after it, but %s clause 1`, subj, expr, ax+"::", scc)})
			}
			if ax == "child" || (ax == "attribute" && field) {
				continue
			}
			if ax == "attribute" {
				for _, expr := range []string{"attribute::a", "attribute::*", "a/attribute::b", "attribute::node()"} {
					rows = append(rows, row{expr, field, fmt.Sprintf(`the {selector} %q names an attribute, but c-selector-xpath clause 2`, expr)})
				}
				continue
			}
			for _, expr := range []string{ax + "::a", ax + "::*", ax + "::node()", "a/" + ax + "::p:*", ax + " :: node ( )"} {
				rows = append(rows, row{expr, field, fmt.Sprintf(`the %s %q steps along the %s axis, but %s clause 2 admits it under neither arm`, subj, expr, ax, scc)})
			}
		}
	}
	rows = append(rows,
		row{"attribute::a/b", true, `the {fields} member "attribute::a/b" names an attribute before its final step`},
		row{"self::node()/child::a", false, `the {selector} "self::node()/child::a" steps along the self axis`},
		row{"child::a|self::*", false, `the {selector} "child::a|self::*" steps along the self axis`},
		row{"child::q:a", false, `the {selector} "child::q:a" has an XPath static error`},
	)
	p := []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:p")}
	for _, tc := range rows {
		x := xsd.NewXPathExpression(tc.expr, p, nil, nil)
		err := violationAt(xsderr.Loc{}, x, tc.field)
		var e *xsderr.Error
		if !errors.As(err, &e) {
			t.Errorf("charging %q (field=%v) = %v, want an *xsderr.Error", tc.expr, tc.field, err)
			continue
		}
		if !strings.HasPrefix(e.Msg, tc.msg) {
			t.Errorf("charging %q (field=%v): message = %q, want it to open %q", tc.expr, tc.field, e.Msg, tc.msg)
		}
		if _, ok := compileOf(tc.expr, tc.field, nil, p...); ok {
			t.Errorf("compiling %q (field=%v) succeeded; a charged {expression} must decline too", tc.expr, tc.field)
		}
	}
}

// The spelled-out unbound prefix is the SAME clause-1 charge as the abbreviated
// one, with err:XPST0081 as its wrapped cause: once the child head reads, the
// stream is a complete production and its prefix resolves like any other.
func TestUnabbreviatedUnboundPrefixWrapsTheXPathCode(t *testing.T) {
	err := violationOf("child::q:a", false)
	if err == nil {
		t.Fatal(`charging "child::q:a" = nil, want the c-selector-xpath clause-1 charge`)
	}
	got, ok := xsderr.RuleOf(errors.Unwrap(err))
	if !ok || got != "err:XPST0081" {
		t.Errorf("cause rule = %q (ok=%v), want err:XPST0081", got, ok)
	}
}
