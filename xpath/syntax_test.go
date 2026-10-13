package xpath

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// TestXPathSyntaxAccepts pins texts A.1's grammar derives, the hard lexical
// corners among them: each must recognize, because rejecting one charges
// ta-props-correct against a schema whose {test} is legal XPath 2.0.
func TestXPathSyntaxAccepts(t *testing.T) {
	for _, tc := range []struct{ expr, why string }{
		{"()", "[46] ParenthesizedExpr with no Expr: the empty sequence (s3_12si04 line 58)"},
		{"() (: :)", "a Comment after the last terminal is a separator"},
		{"@a (: c :) = 1", "a Comment between terminals"},
		{"(: a (: nested :) one :) 1", "Comments nest when balanced"},
		{"5 instance (: odd :) of xs:integer", "a Comment between the words of a two-word operator"},
		{"for (: loop :) $i in $x return $i", "a Comment between 'for' and '$' (gn: parens)"},
		{"address (: may be empty :)", "a name before a Comment is no FunctionCall"},
		{"div div div", "'div' as NameTest, operator, NameTest"},
		{"and and and", "'and' as NameTest, operator, NameTest"},
		{"AND", "keywords are case sensitive, so AND is a name"},
		{"/", "a leading lone slash (xgc: leading-lone-slash)"},
		{"(/) * 5", "a lone slash parenthesized as an operand"},
		{"5 * /", "a lone slash as the right operand"},
		{"/ union", "'union' after a slash is a NameTest (xgc: leading-lone-slash)"},
		{"$x instance of item", "item without '()' is an AtomicType QName"},
		{"1.", "DecimalLiteral with no fraction digits"},
		{".5", "DecimalLiteral with no integer digits"},
		{"1e3", "DoubleLiteral"},
		{"1.5E-2", "DoubleLiteral with a signed exponent"},
		{"foo-foo", "one QName: '-' is a NameChar"},
		{"foo -foo", "a QName, the minus operator, a QName"},
		{"foo(: c :)- foo", "a Comment separates a name from '-'"},
		{"p:if(x)", "a prefixed FunctionCall may use a reserved local name"},
		{"if (@a) then 1 else 2", "[7] IfExpr"},
		{"some $x in @a satisfies $x = 1", "[6] QuantifiedExpr"},
		{"every $x in (1, 2), $y in 3 satisfies $x lt $y", "QuantifiedExpr with two bindings"},
		{"for $x in 1 to 3 return $x * 2", "[4] ForExpr and [11] RangeExpr"},
		{"@a cast as xs:double", "[49] SingleType"},
		{"@a cast as xs:double? = 1", "SingleType with its '?'"},
		{"@kind cast as messageTypeString='string'", "a cast to a user type is legal syntax (s3_12ii06, #894)"},
		{"string cast as string", "a NameTest cast to an unprefixed QName"},
		{"double('3' cast as float > 2)", "a FunctionCall over a comparison"},
		{"@a castable as xs:int", "[18] CastableExpr"},
		{"$x treat as item()*", "[17] TreatExpr with an OccurrenceIndicator"},
		{"$x instance of empty-sequence()", "SequenceType empty-sequence()"},
		{"$x instance of element(*, xs:anyType?)+", "an ElementTest with a nillable TypeName"},
		{"$x instance of attribute(a, xs:string)", "an AttributeTest"},
		{"$x instance of document-node(schema-element(p:e))", "a DocumentTest over a SchemaElementTest"},
		{"$x instance of processing-instruction('pi')", "a PITest over a StringLiteral"},
		{"a << b", "[24] NodeComp '<<'"},
		{"a >> b", "[24] NodeComp '>>'"},
		{"a is b", "[24] NodeComp 'is'"},
		{"@a != 'x' and @b <= 2 or @c >= 3", "GeneralComp operators under and/or"},
		{"*", "the bare Wildcard"},
		{"p:*", "Wildcard NCName ':' '*'"},
		{"*:n", "Wildcard '*' ':' NCName"},
		{"child::a/descendant-or-self::node()/@b", "axis steps and a KindTest"},
		{"ancestor-or-self::*[1]", "a reverse axis with a predicate"},
		{"../@a", "the abbreviated parent step"},
		{".//a[@b = 'c'][2]", "a context item and two predicates"},
		{"a | b union c intersect d except e", "union and intersect/except chains"},
		{"- - +1", "[20] UnaryExpr with several signs"},
		{"10 idiv 3 mod 2", "idiv and mod"},
		{`"a""b" = 'c''d'`, "StringLiteral escapes"},
		{"\t@a\r\n=\t1 ", "XML S around and between terminals"},
		{"namespace::*", "the namespace axis is in the grammar"},
		{"text()", "a KindTest as a step"},
		{"@attribute", "a keyword as an attribute NameTest"},
		{"for", "'for' with no '$' after it is a NameTest"},
		{"if", "'if' with no '(' after it is a NameTest"},
	} {
		if at, bad := xpathSyntaxError(tc.expr); bad {
			t.Errorf("xpathSyntaxError(%q) rejects at offset %d, want accepted (%s)", tc.expr, at, tc.why)
		}
	}
}

// TestXPathSyntaxRejects pins texts no derivation of A.1's grammar covers,
// each with the byte offset the grammar stops at.
func TestXPathSyntaxRejects(t *testing.T) {
	for _, tc := range []struct {
		expr string
		at   int
		why  string
	}{
		{")(", 0, "no ExprSingle opens with ')'"},
		{">", 0, "no ExprSingle opens with a comparator"},
		{"((7>=6)", 7, "one ')' short"},
		{"@6='hi'", 1, "a NodeTest is no number"},
		{"@numberOfChildren < 5 AND @numberOfChildren > 0", 22, "AND is a name, and no name follows a complete operand"},
		{"(@type cast as xs1::double)='double'", 18, "a SingleType is one QName; '::' cannot follow it"},
		{"xs1::double", 0, "xs1 is no axis"},
		{"", 0, "Expr needs an ExprSingle"},
		{"(: :)", 5, "a Comment alone is no ExprSingle"},
		{"@p:kind = 'x' extra", 14, "trailing tokens"},
		{"@p:kind = ", 10, "a comparator with no right operand"},
		{"(: open", 0, "an unclosed Comment"},
		{"'open", 0, "an unclosed StringLiteral"},
		{"1 = 2 = 3", 6, "ComparisonExpr takes one comparator"},
		{"1 to 2 to 3", 7, "RangeExpr takes one 'to'"},
		{"/ * 5", 4, "xgc: leading-lone-slash reads '/ *' first"},
		{"10div 3", 2, "A.2.2: a number and a name need a separator"},
		{"10 div3", 3, "div3 is one name, which cannot follow an operand"},
		{"1.2.3", 3, "A.2.2: '.' beside a numeric literal needs a separator"},
		{"foo- foo", 5, "foo- is one QName, and a name cannot follow it"},
		{"if(@a)", 6, "'if(' opens an IfExpr, which needs 'then'"},
		{"a/if(x)", 2, "if is a reserved function name"},
		{"item()", 0, "item is a reserved function name"},
		{"empty-sequence()", 0, "empty-sequence is a reserved function name"},
		{"typeswitch(1)", 0, "typeswitch is a reserved function name"},
		{"@a cast as xs:double+", 21, "SingleType takes no '+', and '+' then needs an operand"},
		{"4 treat as item() + 5", 20, "xgc: occurrence-indicators takes the '+'"},
		{"@ a : b", 4, "no white space inside a QName"},
		{"p : *", 2, "no white space inside a Wildcard"},
		{"*:*", 1, "no Wildcard '*' ':' '*'"},
		{"a ! b", 2, "'!' only opens '!='"},
		{"@a = 1", 2, "white space is XML S only, never U+00A0"},
		{"element(*:a)", 8, "an ElementTest takes '*' or a QName, never '*:n'"},
		{"/ union /", 9, "'union' after a slash is a NameTest, so the second '/' needs a step"},
		{"(1, )", 4, "a comma needs an ExprSingle after it"},
		{"a[1", 3, "an unclosed predicate"},
	} {
		at, bad := xpathSyntaxError(tc.expr)
		if !bad {
			t.Errorf("xpathSyntaxError(%q) accepts, want rejected at offset %d (%s)", tc.expr, tc.at, tc.why)
			continue
		}
		if at != tc.at {
			t.Errorf("xpathSyntaxError(%q) rejects at offset %d, want %d (%s)", tc.expr, at, tc.at, tc.why)
		}
	}
}

// TestXPathSyntaxAcceptsSuiteValidTests is the over-charge guard: it feeds the
// recognizer every alternative/@test the W3C suite carries in a schema
// document of a schemaTest it declares valid, under any version, and fails
// on any value rejected — each such rejection becomes a ta-props-correct
// charge against a schema the suite says is valid. It reads testdata/xsdtests
// at run time, so it holds for whatever suite revision is checked out, and an
// absent submodule skips it.
//
// Only the schemaTest's own schemaDocuments are read, never a document they
// include or import.
func TestXPathSyntaxAcceptsSuiteValidTests(t *testing.T) {
	const suite = "../testdata/xsdtests"
	if _, err := os.Stat(filepath.Join(suite, "suite.xml")); err != nil {
		t.Skip("suite submodule absent: git submodule update --init testdata/xsdtests")
	}
	var docs []string
	err := filepath.WalkDir(suite, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".testSet") {
			return nil
		}
		valid, err := suiteValidSchemaDocuments(path)
		if err != nil {
			return err
		}
		docs = append(docs, valid...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	tests := 0
	for _, doc := range docs {
		if seen[doc] {
			continue
		}
		seen[doc] = true
		values, err := alternativeTests(doc)
		if err != nil {
			t.Fatalf("%s: %v", doc, err)
		}
		for _, v := range values {
			tests++
			if at, bad := xpathSyntaxError(v); bad {
				t.Errorf("%s: xpathSyntaxError(%q) rejects at offset %d, want accepted: the suite declares this schema valid", doc, v, at)
			}
		}
	}
	// A floor, so a reader that silently finds nothing cannot pass: suite
	// 7bc3365 yields well over this many.
	if tests < 100 {
		t.Fatalf("read %d alternative/@test values from suite-valid schemas, want at least 100", tests)
	}
	t.Logf("%d alternative/@test values from %d suite-valid schema documents", tests, len(seen))
}

// suiteTSNamespace is the W3C test-suite metadata namespace.
const suiteTSNamespace = "http://www.w3.org/XML/2004/xml-schema-test-suite/"

// suiteValidSchemaDocuments reads one testSet and returns the paths of the
// schemaDocuments of every schemaTest with an <expected validity="valid"/>,
// in document order.
func suiteValidSchemaDocuments(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := xml.NewDecoder(f)
	var out, pending []string
	valid, inSchemaTest := false, false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if el.Name.Space != suiteTSNamespace {
				continue
			}
			switch el.Name.Local {
			case "schemaTest":
				inSchemaTest, valid, pending = true, false, nil
			case "schemaDocument":
				if inSchemaTest {
					pending = append(pending, filepath.Join(filepath.Dir(path), suiteAttr(el, "href")))
				}
			case "expected":
				if inSchemaTest && suiteAttr(el, "validity") == "valid" {
					valid = true
				}
			}
		case xml.EndElement:
			if el.Name.Space == suiteTSNamespace && el.Name.Local == "schemaTest" {
				if valid {
					out = append(out, pending...)
				}
				inSchemaTest = false
			}
		}
	}
}

// suiteAttr is the value of el's attribute with the given local name.
func suiteAttr(el xml.StartElement, local string) string {
	for _, a := range el.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// suiteDecode returns raw as UTF-8 text, transcoding a document that opens
// with a UTF-16 byte-order mark (XML 1.0 App. F.1) and passing any other
// through for encoding/xml to read under its declaration.
func suiteDecode(raw []byte) string {
	if len(raw) < 2 || !(raw[0] == 0xFF && raw[1] == 0xFE || raw[0] == 0xFE && raw[1] == 0xFF) {
		return string(raw)
	}
	units := make([]uint16, 0, len(raw)/2)
	for i := 2; i+1 < len(raw); i += 2 {
		if raw[0] == 0xFF {
			units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
			continue
		}
		units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
	}
	return string(utf16.Decode(units))
}

// suiteCharsetReader serves the encoding declarations the suite's schema
// documents carry: UTF-16, already transcoded by suiteDecode, and ISO-8859-1,
// whose bytes are the code points they name.
func suiteCharsetReader(label string, in io.Reader) (io.Reader, error) {
	switch strings.ToLower(label) {
	case "utf-16":
		return in, nil
	case "iso-8859-1", "latin1":
		raw, err := io.ReadAll(in)
		if err != nil {
			return nil, err
		}
		runes := make([]rune, len(raw))
		for i, b := range raw {
			runes[i] = rune(b)
		}
		return strings.NewReader(string(runes)), nil
	}
	return nil, fmt.Errorf("no reader for encoding %q", label)
}

// alternativeTests returns the test attribute of every XSD alternative
// element in the schema document at path, in document order.
func alternativeTests(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := suiteDecode(raw)
	if !strings.Contains(text, "alternative") {
		return nil, nil // no element can be named alternative
	}
	// encoding/xml reads no XML 1.1 declaration; the label changes nothing
	// an attribute value of an alternative element is read under.
	if decl, rest, ok := strings.Cut(text, "?>"); ok && strings.HasPrefix(decl, "<?xml") {
		text = strings.Replace(decl, "1.1", "1.0", 1) + "?>" + rest
	}
	dec := xml.NewDecoder(strings.NewReader(text))
	dec.CharsetReader = suiteCharsetReader
	var out []string
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		el, ok := tok.(xml.StartElement)
		if !ok || el.Name.Space != "http://www.w3.org/2001/XMLSchema" || el.Name.Local != "alternative" {
			continue
		}
		for _, a := range el.Attr {
			if a.Name.Space == "" && a.Name.Local == "test" {
				out = append(out, a.Value)
			}
		}
	}
}
