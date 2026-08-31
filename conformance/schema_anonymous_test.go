package conformance

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/kud360/goxsd8/parser"
)

// This file pins what an ANONYMOUS complex type — the inline <complexType> child
// of an <element>, §3.3.2.1 dcl.elt.common clause 1 — is judged by. Until #1126
// it was judged by a separate, narrower predicate that admitted the
// implicit-content form alone; the subject now is that no such narrowing exists,
// which is a claim about a MISSING distinction and so needs pinning at least as
// much as the distinction did. The spec argument in one sentence: §3.3.2.1
// clause 1 is a common mapping rule and §3.4.2.1 makes anonymity fix {name},
// {target namespace} and {context} and nothing else, so a narrower gate here
// would state a distinction the spec does not make; the full argument, with its
// citations and the one-directional residual it leaves, is complexTypeDecidable's
// doc comment in schema.go and is deliberately not restated here (STYLE T4).
//
// The two tests below split the claim in half, as #443's pair split the
// narrowing, and for the same reason — a verdict assertion alone cannot carry it:
//
//   - TestAnonymousComplexTypeDecidesAsTheNamedTypeDoes asserts the VERDICTS, on
//     both polarities and at both nesting depths, by running each body through
//     the anonymous path and through the named one and demanding they AGREE. It
//     is an equivalence rather than a fixed expectation, so a later widening of
//     complexTypeDecidable moves both sides at once and this test keeps testing
//     what it claims instead of pinning yesterday's subset.
//   - TestAnonymousComplexTypeRoutesThroughTheNamedPredicate asserts the
//     STRUCTURE: both element gates hand their inline <complexType> to
//     complexTypeDecidable, and no other function in schema.go reads the two
//     content-alternative element names — the only way a narrowing can be
//     spelled. This is the half a verdict test cannot reach — an equivalence
//     over a table samples shapes, so a narrowing reintroduced for a shape the
//     table does not hold would pass it — and it is a source scan because, like
//     its predecessor, nothing else can be. It follows the same precedent,
//     xsderr/rulecatalog_enforcement_test.go: a claim the type system cannot
//     carry, carried by a scan.

// globalElement returns the single global <element> owning an inline
// <complexType> whose body is ct — the exact *parser.Element elementDecidable is
// handed, reached by navigation rather than by any decidability predicate.
func globalElement(t *testing.T, ct string) *parser.Element {
	t.Helper()
	doc := schemaDoc(t, `<xs:element name="e"><xs:complexType>`+ct+`</xs:complexType></xs:element>`)
	el := childXSD(doc.Root(), "element")
	if el == nil {
		t.Fatalf("no <element> child in the built document for %q", ct)
	}
	return el
}

// inlineComplexType returns that <element>'s inline <complexType> child, the
// element elementDecidable passes on to complexTypeDecidable.
func inlineComplexType(t *testing.T, ct string) *parser.Element {
	t.Helper()
	inline := childXSD(globalElement(t, ct), "complexType")
	if inline == nil {
		t.Fatalf("no inline <complexType> child in the built document for %q", ct)
	}
	return inline
}

// namedComplexType returns the top-level <complexType name="T"> element whose
// body is ct, the named counterpart of inlineComplexType's anonymous one.
func namedComplexType(t *testing.T, ct string) *parser.Element {
	t.Helper()
	doc := schemaDoc(t, `<xs:complexType name="T">`+ct+`</xs:complexType>`)
	named := childXSD(doc.Root(), "complexType")
	if named == nil {
		t.Fatalf("no top-level <complexType> child in the built document for %q", ct)
	}
	return named
}

// nestedElement returns the LOCAL <element> that owns an inline <complexType>
// whose body is ct, the particle child localElementDecidable is handed.
func nestedElement(t *testing.T, ct string) *parser.Element {
	t.Helper()
	doc := schemaDoc(t, `<xs:complexType name="Outer"><xs:sequence><xs:element name="a"><xs:complexType>`+
		ct+`</xs:complexType></xs:element></xs:sequence></xs:complexType>`)
	outer := childXSD(doc.Root(), "complexType")
	if outer == nil {
		t.Fatalf("no top-level <complexType> child in the built document for %q", ct)
	}
	seq := childXSD(outer, "sequence")
	if seq == nil {
		t.Fatalf("no <sequence> child in the built document for %q", ct)
	}
	local := childXSD(seq, "element")
	if local == nil {
		t.Fatalf("no local <element> child in the built document for %q", ct)
	}
	return local
}

// TestAnonymousComplexTypeDecidesAsTheNamedTypeDoes proves the verdicts agree:
// for every body, the anonymous inline form reaches the same verdict as the
// named top-level form, on the global element path (elementDecidable) and on the
// local one (localElementDecidable) alike. want records what the shared verdict
// IS, so a table on which both sides moved together — every case admitted, say —
// still fails rather than passing vacuously.
func TestAnonymousComplexTypeDecidesAsTheNamedTypeDoes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		// The explicit-content forms #1126 admitted — all four §3.4.2.2/§3.4.2.3
		// alternants. Before it, each was declined on the anonymous path and
		// admitted on the named one.
		{
			"complexContent restriction",
			`<xs:complexContent><xs:restriction base="xs:anyType"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="x" type="xs:string"/></xs:restriction></xs:complexContent>`,
			true,
		},
		{
			"complexContent extension",
			`<xs:complexContent><xs:extension base="xs:anyType"><xs:sequence/><xs:anyAttribute namespace="##other"/></xs:extension></xs:complexContent>`,
			true,
		},
		{
			"simpleContent extension",
			`<xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="x" type="xs:string"/></xs:extension></xs:simpleContent>`,
			true,
		},
		{
			"simpleContent restriction with facets",
			`<xs:simpleContent><xs:restriction base="B"><xs:maxLength value="4"/><xs:attribute name="x" type="xs:string"/></xs:restriction></xs:simpleContent>`,
			true,
		},
		{
			// §3.4.2.3.2's implicit content, the one form the narrowing admitted.
			"implicit content",
			`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="x" type="xs:string"/>`,
			true,
		},
		// The other polarity, so an equivalence that admitted everything would
		// fail here. Each of these is declined on BOTH paths, and for a reason
		// that is about the producer rather than about anonymity: a particle under
		// xs:simpleExtensionType is dropped in silence, a bare <group> with no ref
		// is malformed, an inline <simpleType> naming none of §3.16.2.1's three
		// alternatives is outside the produced subset.
		{
			"simpleContent extension carrying a particle",
			`<xs:simpleContent><xs:extension base="xs:string"><xs:sequence/></xs:extension></xs:simpleContent>`,
			false,
		},
		{
			"simpleContent restriction carrying a particle",
			`<xs:simpleContent><xs:restriction base="B"><xs:sequence/></xs:restriction></xs:simpleContent>`,
			false,
		},
		{
			"complexContent restriction over a bare group",
			`<xs:complexContent><xs:restriction base="xs:anyType"><xs:group name="inner"><xs:sequence/></xs:group></xs:restriction></xs:complexContent>`,
			false,
		},
		{
			"complexContent extension nesting an undecidable inline simpleType",
			`<xs:complexContent><xs:extension base="xs:anyType"><xs:sequence><xs:element name="a"><xs:simpleType/></xs:element></xs:sequence></xs:extension></xs:complexContent>`,
			false,
		},
	}
	for _, tc := range cases {
		named := complexTypeDecidable(namedComplexType(t, tc.body))
		if named != tc.want {
			t.Errorf("%s: complexTypeDecidable on the NAMED form = %t, want %t — the case no longer tests what it claims", tc.name, named, tc.want)
		}
		if got := complexTypeDecidable(inlineComplexType(t, tc.body)); got != named {
			t.Errorf("%s: complexTypeDecidable on the anonymous form = %t, want %t to match the named form: §3.3.2.1 clause 1 maps the two through the same §3.4.2 rules and §3.4.2.1 makes anonymity fix {name}/{target namespace}/{context} alone", tc.name, got, named)
		}
		if got := elementDecidable(globalElement(t, tc.body)); got != named {
			t.Errorf("%s: elementDecidable on the owning global <element> = %t, want %t to match the named form", tc.name, got, named)
		}
		if got := localElementDecidable(nestedElement(t, tc.body)); got != named {
			t.Errorf("%s: localElementDecidable on the owning local <element> = %t, want %t to match the named form: the recursion through contentDecidable → modelGroupDecidable reaches the same predicate at every depth", tc.name, got, named)
		}
	}
}

// TestAnonymousComplexTypeRoutesThroughTheNamedPredicate proves the STRUCTURE
// the equivalence above can only sample: both element gates hand their inline
// <complexType> to complexTypeDecidable, and no other function in schema.go
// reads the two content-alternative element names — the only way a narrowing can
// be spelled. A narrowing reintroduced for a shape the table does not hold would
// pass the equivalence and fail here, which is why this scan exists rather than
// more table rows.
func TestAnonymousComplexTypeRoutesThroughTheNamedPredicate(t *testing.T) {
	const file = "schema.go"
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, file, nil, goparser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}

	for _, fn := range []string{"elementDecidable", "localElementDecidable"} {
		body := funcBody(f, fn)
		if body == nil {
			t.Fatalf("%s declares no func %s: the scan has nothing to check", file, fn)
		}
		if !calls(body, "complexTypeDecidable") {
			t.Errorf("%s does not call complexTypeDecidable: §3.3.2.1 clause 1's inline <complexType> must be judged by the predicate a NAMED <complexType> takes, so that anonymity narrows nothing (see complexTypeDecidable's doc comment)", fn)
		}
	}

	// Any narrowing of the content alternatives must READ their element names,
	// whatever it is called and wherever it sits — so the file-wide census, not a
	// check on two function bodies, is what a differently-named helper cannot
	// slip past. Exactly one function is licensed to read them: the shared
	// predicate, whose dispatch on them is the mapping rule itself (§3.4.2.2,
	// §3.4.2.3).
	const licensed = "complexTypeDecidable"
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv != nil || fd.Body == nil || fd.Name.Name == licensed {
			continue
		}
		names := stringLiterals(fd.Body)
		if names[`"simpleContent"`] || names[`"complexContent"`] {
			t.Errorf("%s names a content alternative and is not %s: a second site reading those names is where an anonymity narrowing would live, and #1126 removed the one that did. If it is back deliberately, it owes the reason %s's doc comment now denies, and this scan is what must be rewritten", fd.Name.Name, licensed, licensed)
		}
	}
	if body := funcBody(f, licensed); body != nil {
		names := stringLiterals(body)
		if !names[`"simpleContent"`] || !names[`"complexContent"`] {
			t.Errorf("%s no longer reads both content alternatives, so the census above licenses a name nothing writes and can never fail; rewrite this scan for the new dispatch", licensed)
		}
	}
}

// calls reports whether body holds a call to the plain function named name.
func calls(body *ast.BlockStmt, name string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if id, isIdent := call.Fun.(*ast.Ident); isIdent && id.Name == name {
			found = true
		}
		return true
	})
	return found
}

// stringLiterals returns the set of string literals written in body, in the
// source spelling the scan compares against (quotes included).
func stringLiterals(body *ast.BlockStmt) map[string]bool {
	names := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			names[lit.Value] = true
		}
		return true
	})
	return names
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
