package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// resolverFor builds a producer over doc (whose targetNamespace is its own, the
// lone-document case Produce uses) and returns its resolveQName bound to the
// first top-level <element>, the element every case below writes its reference
// on.
func resolverFor(t *testing.T, doc string) func(string) (xsd.QName, error) {
	t.Helper()
	d, err := ReadDocument("mem://licensed.xsd", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	builder := xsd.NewSchemaBuilder()
	sym, err := newSymbols(builder, strict.New())
	if err != nil {
		t.Fatalf("newSymbols: %v", err)
	}
	p := newProducer(d, attrOr(d.Root(), "targetNamespace"), nil, nil, nil, builder, sym)
	elem := childElement(d.Root(), xsd.XMLSchemaNS, "element")
	if elem == nil {
		t.Fatal("the test document has no top-level <element>")
	}
	return func(lexical string) (xsd.QName, error) { return p.resolveQName(elem, lexical) }
}

// TestResolveQNameLicensedNamespaces is package-internal because it pins
// src-resolve clause 4 (cl.qnr.nsdeclared, §3.17.6.2) on its own, before clauses
// 1-3 get a say: from outside the package every case would also have to make the
// named component EXIST, which would confuse an unlicensed namespace (rejected
// here, at the reference) with a namespace nothing supplied (rejected at
// finalize). Both verdicts are src-resolve, so only the reference-time call
// distinguishes them.
//
// The document licenses urn:b by <import> and nothing else; the XSD and XSI
// namespaces are licensed by clauses 4.2.3/4.2.4 with no <import> at all, which
// is the false-reject this table exists to catch.
func TestResolveQNameLicensedNamespaces(t *testing.T) {
	resolve := resolverFor(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
		` xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`+
		` xmlns:tns="urn:a" xmlns:imp="urn:b" xmlns:un="urn:c"`+
		` targetNamespace="urn:a">`+
		`<xs:import namespace="urn:b"/>`+
		`<xs:element name="e" type="xs:string"/>`+
		`</xs:schema>`)

	for _, tc := range []struct {
		name     string
		lexical  string
		want     xsd.QName
		licensed bool
	}{
		{"own target namespace (4.2.1)", "tns:c", xsd.QName{Space: "urn:a", Local: "c"}, true},
		{"imported namespace (4.2.2)", "imp:c", xsd.QName{Space: "urn:b", Local: "c"}, true},
		{"XSD namespace, unimported (4.2.3)", "xs:string", xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}, true},
		{"XSI namespace, unimported (4.2.4)", "xsi:type", xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}, true},
		{"unimported namespace", "un:c", xsd.QName{}, false},
		{"absent namespace, no bare import (4.1)", "c", xsd.QName{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolve(tc.lexical)
			if !tc.licensed {
				assertSrcResolve(t, err, tc.lexical)
				return
			}
			if err != nil {
				t.Fatalf("resolveQName(%q) = %v, want the reference licensed by src-resolve clause 4", tc.lexical, err)
			}
			if got != tc.want {
				t.Fatalf("resolveQName(%q) = %s, want %s", tc.lexical, got, tc.want)
			}
		})
	}
}

// TestResolveQNameBareImportLicensesAbsentNamespace is the other half of clause
// 4.1: the SAME unqualified reference the table above rejects resolves once the
// document carries an <import> with no namespace attribute (clause 4.1.2,
// §4.2.6.1's "if that attribute is absent, then the import allows unqualified
// reference to components with no target namespace").
func TestResolveQNameBareImportLicensesAbsentNamespace(t *testing.T) {
	resolve := resolverFor(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
		` targetNamespace="urn:a">`+
		`<xs:import schemaLocation="lib.xsd"/>`+
		`<xs:element name="e" type="xs:string"/>`+
		`</xs:schema>`)
	got, err := resolve("c")
	if err != nil {
		t.Fatalf("resolveQName: %v, want the bare <import> to license the ·absent· namespace (clause 4.1.2)", err)
	}
	if want := (xsd.QName{Local: "c"}); got != want {
		t.Fatalf("resolveQName = %s, want %s", got, want)
	}
}

// TestResolveQNameNotQNameIsNotLicensed pins the boundary the split between
// resolveQName and bindQName draws: §3.10.2's {disallowed names} mapping takes
// notQName's items as QName VALUES and never ·resolves· them to components, so
// clause 4 does not reach them — an <any notQName="un:c"> blocks a name in a
// namespace this document never imported, and that is not a src-resolve failure.
func TestResolveQNameNotQNameIsNotLicensed(t *testing.T) {
	const doc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:un="urn:c" targetNamespace="urn:a">
		<xs:complexType name="ct"><xs:sequence>
			<xs:any notQName="un:c" processContents="lax"/>
		</xs:sequence></xs:complexType>
	</xs:schema>`
	d, err := ReadDocument("mem://notqname.xsd", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	if _, err := Produce(d, strict.New()); err != nil {
		t.Fatalf("Produce: %v, want notQName to be exempt from src-resolve clause 4", err)
	}
}

// assertSrcResolve fails unless err is an *xsderr.Error charging src-resolve and
// naming the offending lexical, so a case cannot pass on some other rejection.
func assertSrcResolve(t *testing.T, err error, lexical string) {
	t.Helper()
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("resolveQName(%q) error = %v, want an *xsderr.Error charging src-resolve", lexical, err)
	}
	if xe.Rule != ruleSrcResolve {
		t.Fatalf("resolveQName(%q) rule = %q, want %s", lexical, xe.Rule, ruleSrcResolve)
	}
	if !strings.Contains(xe.Error(), lexical) {
		t.Fatalf("resolveQName(%q) message %q does not name the reference", lexical, xe.Error())
	}
}
