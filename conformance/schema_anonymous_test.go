package conformance

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/kud360/goxsd8/parser"
)

// This file pins anonymousComplexTypeDecidable's own narrowing — the decline of
// an inline <complexType> carrying <simpleContent> or <complexContent> — which
// #443 found unpinned. The safety argument in one sentence: only the
// implicit-content form is admitted, because on that shape alone the two
// attribute folds an anonymous type never receives are provably the identity;
// the full argument, with its spec citations and the list of what stays
// unenforced, is anonymousComplexTypeDecidable's doc comment in schema.go and is
// deliberately not restated here (STYLE T4).
//
// The narrowing needs pinning of its own precisely because contentDecidable's
// default arm happens to decline the same two element names today, which MASKS
// the narrowing: no input can make the guarded and unguarded functions return
// different values, so deleting the guard is a behaviourally equivalent mutation
// and no assertion over return values can catch it. The two tests below split
// that problem in half rather than pretending one test solves it:
//
//   - TestAnonymousComplexTypeDecidableNarrowsExplicitContent asserts the
//     VERDICTS, both polarities, calling anonymousComplexTypeDecidable directly
//     on directly-navigated inline <complexType> elements. It pairs each explicit
//     form with an implicit-content sibling holding identical inner content, so
//     the wrapper is the only difference, and it asserts complexTypeDecidable
//     admits the very same element — which is what makes the decline attributable
//     to the anonymity narrowing rather than to undecidable content, and keeps
//     the two functions' verdicts separable in the test's own construction.
//   - TestAnonymousComplexTypeNarrowingPrecedesDelegation asserts the STRUCTURE:
//     the guard exists and runs before the contentDecidable call. This is the
//     half that fails when the two lines are deleted, and it is a source scan
//     because, per the masking above, nothing else can be. It follows the
//     precedent of xsderr/rulecatalog_enforcement_test.go: a claim the type
//     system cannot carry, carried by a scan.

// inlineComplexType returns the inline <complexType> element of a single global
// <element> whose body is ct — the exact *parser.Element
// elementDecidable hands to anonymousComplexTypeDecidable, reached by
// navigation rather than by any decidability predicate.
func inlineComplexType(t *testing.T, ct string) *parser.Element {
	t.Helper()
	doc := schemaDoc(t, `<xs:element name="e"><xs:complexType>`+ct+`</xs:complexType></xs:element>`)
	el := childXSD(doc.Root(), "element")
	if el == nil {
		t.Fatalf("no <element> child in the built document for %q", ct)
	}
	inline := childXSD(el, "complexType")
	if inline == nil {
		t.Fatalf("no inline <complexType> child in the built document for %q", ct)
	}
	return inline
}

// TestAnonymousComplexTypeDecidableNarrowsExplicitContent proves the verdicts of
// the narrowing on both polarities: an inline <complexType> with a
// <simpleContent> or a <complexContent> child is DECLINED even though the very
// same element is ADMITTED by complexTypeDecidable (so the decline is the
// anonymity narrowing, not undecidable content), while the implicit-content
// sibling holding the identical inner content is ADMITTED (so a guard that
// declined everything would fail here too).
func TestAnonymousComplexTypeDecidableNarrowsExplicitContent(t *testing.T) {
	cases := []struct {
		name string
		// wrapper is a %s format wrapping inner in the explicit-content child.
		wrapper string
		// inner is the content shared by the explicit form and its
		// implicit-content sibling, so the wrapper is the only difference.
		inner string
	}{
		{
			"complexContent restriction",
			`<xs:complexContent><xs:restriction base="xs:anyType">%s</xs:restriction></xs:complexContent>`,
			`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="x" type="xs:string"/>`,
		},
		{
			// xs:simpleExtensionType admits no particle, so this pair's shared
			// inner content is the attribute alone.
			"simpleContent extension",
			`<xs:simpleContent><xs:extension base="xs:string">%s</xs:extension></xs:simpleContent>`,
			`<xs:attribute name="x" type="xs:string"/>`,
		},
	}
	for _, tc := range cases {
		explicit := inlineComplexType(t, fmt.Sprintf(tc.wrapper, tc.inner))
		if anonymousComplexTypeDecidable(explicit) {
			t.Errorf("%s: anonymousComplexTypeDecidable = true, want false — an anonymous type is in no {type definitions} set, so the folds this form needs never run", tc.name)
		}
		if !complexTypeDecidable(explicit) {
			t.Errorf("%s: complexTypeDecidable = false on the same element, want true — the decline above must be the anonymity narrowing, not undecidable content, or this case no longer tests what it claims", tc.name)
		}
		if implicit := inlineComplexType(t, tc.inner); !anonymousComplexTypeDecidable(implicit) {
			t.Errorf("%s: anonymousComplexTypeDecidable = false on the implicit-content sibling holding the identical inner content, want true", tc.name)
		}
	}
}

// TestAnonymousComplexTypeNarrowingPrecedesDelegation proves the narrowing is
// still IN anonymousComplexTypeDecidable and still runs BEFORE the
// contentDecidable call: an early `return false` guarded by both element names,
// ahead of the delegation. Deleting the two guard lines changes no return value
// anywhere (contentDecidable's default arm declines the same names), so this
// scan — not an assertion over verdicts — is what makes the deletion fail a
// test.
func TestAnonymousComplexTypeNarrowingPrecedesDelegation(t *testing.T) {
	const (
		fn   = "anonymousComplexTypeDecidable"
		file = "schema.go"
	)
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, file, nil, goparser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}
	body := funcBody(f, fn)
	if body == nil {
		t.Fatalf("%s declares no func %s: the scan has nothing to check", file, fn)
	}

	delegation, earlyFalse := token.NoPos, token.NoPos
	names := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "contentDecidable" && delegation == token.NoPos {
				delegation = x.Pos()
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				names[x.Value] = true
			}
		case *ast.ReturnStmt:
			if earlyFalse == token.NoPos && len(x.Results) == 1 {
				if id, ok := x.Results[0].(*ast.Ident); ok && id.Name == "false" {
					earlyFalse = x.Pos()
				}
			}
		}
		return true
	})

	if delegation == token.NoPos {
		t.Fatalf("%s no longer calls contentDecidable; this test asserts the narrowing runs before that call and must be rewritten for the new shape", fn)
	}
	if earlyFalse == token.NoPos || earlyFalse > delegation {
		t.Errorf("%s delegates to contentDecidable with no `return false` ahead of it: the explicit-content narrowing is gone. It is deliberately redundant with contentDecidable's default arm and carries its own rationale (see the doc comment); restore it, or, if you moved it into a helper, update this scan", fn)
	}
	for _, want := range []string{`"simpleContent"`, `"complexContent"`} {
		if !names[want] {
			t.Errorf("%s no longer mentions %s: the narrowing must decline BOTH explicit-content children, not one", fn, want)
		}
	}
}

// funcBody returns the body of the top-level func named name, or nil.
func funcBody(f *ast.File, name string) *ast.BlockStmt {
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && fd.Recv == nil && fd.Name.Name == name {
			return fd.Body
		}
	}
	return nil
}
