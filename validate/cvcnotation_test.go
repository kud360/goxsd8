package validate

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These fixtures drive the declared-notation half of String Valid clause 2,
// which value.ValidateLexical decides against the schema: NOTATION's ·value
// space· is "the set of QNames of notations declared in the current schema"
// (Datatypes §3.3.19), so a value the backend's QName mapping accepts is
// still not Datatype Valid where its QName names no notation declaration. The
// schemas are parsed from documents, because a notation declaration and an
// <override> are what the check reads.

// notationSchema declares the notations foo and bar in no namespace, an
// enumeration subtype of NOTATION listing both, and a root carrying an
// attribute of each: @enum of that subtype and @direct of xs:NOTATION itself.
//
// Only @direct reaches the undeclared-name charge. The parser rejects an
// enumeration member naming no declared notation under
// enumeration-valid-restriction (Datatypes §4.3.5.5), so every value a parsed
// enumeration subtype admits names a declared notation, and an undeclared one
// is an enumeration miss first.
const notationSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="foo" public="pubfoo"/>
  <xs:notation name="bar" public="pubbar"/>
  <xs:simpleType name="FooBar">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="foo"/>
      <xs:enumeration value="bar"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="enum" type="FooBar"/>
      <xs:attribute name="direct" type="xs:NOTATION"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// ruleCvcDatatypeValid is Datatype Valid (Datatypes §4.1.4,
// cvc-datatype-valid), the rule value.ValidateLexical charges an undeclared
// NOTATION value under.
const ruleCvcDatatypeValid xsderr.Rule = "cvc-datatype-valid"

// parsedSchema assembles the schema rooted at main.xsd among docs.
func parsedSchema(t *testing.T, docs map[string]string) *xsd.Schema {
	t.Helper()
	schema, err := parser.Parse("main.xsd", parser.WithResolver(loader.Map(docs)), parser.WithBackend(testBackend()))
	if err != nil {
		t.Fatalf("parser.Parse: %v", err)
	}
	return schema
}

// notationRoot is a childless element named name carrying one attribute @attr
// of the given lexical, at the Loc every assertion below cites, with bindings
// as its in-scope namespaces.
func notationRoot(name xsd.QName, attr, lexical string, bindings map[string]string) *testElement {
	return &testElement{
		name:     name,
		attrs:    []Attribute{&testAttribute{name: local(attr), value: lexical, loc: loc(3, 5)}},
		bindings: bindings,
		loc:      loc(3, 1),
	}
}

// wantUndeclaredNotation fails unless got is exactly one cvc-attribute charge
// at the attribute's 3:5 whose wrapped cause is the cvc-datatype-valid verdict
// value.ValidateLexical words for value resolving to name. The verdict carries
// no Loc of its own, the charge wrapping it being what locates it.
func wantUndeclaredNotation(t *testing.T, got []*xsderr.Error, value string, name xsd.QName) {
	t.Helper()
	viol := onlyCharge(t, got, ruleCvcAttribute)
	if viol.Loc != loc(3, 5) {
		t.Errorf("Loc = %s, want the attribute's %s", viol.Loc, loc(3, 5))
	}
	var cause *xsderr.Error
	if !errors.As(errors.Unwrap(viol), &cause) {
		t.Fatalf("Unwrap(%v) holds no *xsderr.Error, want the Datatype Valid verdict", viol)
	}
	if cause.Rule != ruleCvcDatatypeValid {
		t.Fatalf("wrapped Rule = %q, want %q", cause.Rule, ruleCvcDatatypeValid)
	}
	if cause.Loc != (xsderr.Loc{}) {
		t.Errorf("wrapped Loc = %s, want none", cause.Loc)
	}
	prefix := `the NOTATION value "` + value + `" resolves to the QName ` + name.String() + `, which names no notation declaration`
	suffix := `"the set of QNames of notations declared in the current schema", which cvc-datatype-valid clause 2.1 requires it to be in`
	if !strings.HasPrefix(cause.Msg, prefix) || !strings.HasSuffix(cause.Msg, suffix) {
		t.Fatalf("wrapped Msg = %q, want it to open %q and close %q", cause.Msg, prefix, suffix)
	}
}

// causeRule is the rule of the verdict viol wraps, "" where it wraps none.
func causeRule(viol *xsderr.Error) xsderr.Rule {
	rule, _ := xsderr.RuleOf(errors.Unwrap(viol))
	return rule
}

// A value of an enumeration subtype of NOTATION, and a value of xs:NOTATION
// itself, is ·valid· where it names a declared notation. A value of xs:NOTATION naming
// none is charged cvc-attribute clause 3 at the attribute, with a cvc-datatype-valid
// cause — bez passes the backend's QName mapping and still names no declaration, so the
// charge is the declared-notation check's.
func TestANotationValueMustNameADeclaredNotation(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": notationSchema})
	root := xsd.QName{Local: "root"}
	for _, attr := range []string{"enum", "direct"} {
		t.Run(attr, func(t *testing.T) {
			for _, declared := range []string{"foo", "bar", " foo "} {
				got, _ := assessRecorded(t, schema, notationRoot(root, attr, declared, nil))
				wantSilence(t, got, declared+" names a declared notation")
			}
		})
	}
	got, _ := assessRecorded(t, schema, notationRoot(root, "direct", "bez", nil))
	wantUndeclaredNotation(t, got, "bez", xsd.QName{Local: "bez"})
}

// An enumeration miss stays the backend's charge: baz is outside FooBar's
// enumeration, so the cause is cvc-enumeration-valid and not the
// declared-notation check's cvc-datatype-valid.
func TestAnEnumerationMissIsNotANotationCharge(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": notationSchema})
	got, _ := assessRecorded(t, schema, notationRoot(xsd.QName{Local: "root"}, "enum", "baz", nil))
	viol := onlyCharge(t, got, ruleCvcAttribute)
	if rule := causeRule(viol); rule != "cvc-enumeration-valid" {
		t.Fatalf("wrapped rule = %q, want the backend's cvc-enumeration-valid: %v", rule, viol)
	}
}

// A NOTATION value's QName resolves against the owner's in-scope namespaces,
// an unprefixed one taking the DEFAULT namespace (Datatypes §3.3.18) and not
// the no-namespace reading an unprefixed attribute name gets: under a target
// namespace, foo is declared only as {urn:n}foo, and p:bar, which resolves,
// names nothing.
func TestANotationValueResolvesAgainstTheInScopeNamespaces(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:n" xmlns:n="urn:n">
  <xs:notation name="foo" public="pubfoo"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="direct" type="xs:NOTATION"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`})
	root := xsd.QName{Space: "urn:n", Local: "root"}

	got, _ := assessRecorded(t, schema, notationRoot(root, "direct", "p:foo", map[string]string{"p": "urn:n"}))
	wantSilence(t, got, "p:foo resolves to {urn:n}foo")
	got, _ = assessRecorded(t, schema, notationRoot(root, "direct", "foo", map[string]string{"": "urn:n"}))
	wantSilence(t, got, "foo takes the default namespace urn:n")

	got, _ = assessRecorded(t, schema, notationRoot(root, "direct", "foo", nil))
	wantUndeclaredNotation(t, got, "foo", xsd.QName{Local: "foo"})
	got, _ = assessRecorded(t, schema, notationRoot(root, "direct", "p:bar", map[string]string{"p": "urn:n"}))
	wantUndeclaredNotation(t, got, "p:bar", xsd.QName{Space: "urn:n", Local: "bar"})
}

// The declared set is the assembled schema's {notation declarations}, an
// ·override· host's own top-level declaration among them: over015.xsd
// overrides over015a.xsd and declares bez outside the <override>, so bez is
// declared and ·valid· (Override/over015.v01.xml), while qux, which neither
// document declares, is charged against @direct, of xs:NOTATION itself: the
// override's Nota cannot enumerate qux, since the parser rejects an
// undeclared member under enumeration-valid-restriction (Datatypes §4.3.5.5).
func TestAnOverrideHostsNotationIsDeclared(t *testing.T) {
	schema := parsedSchema(t, map[string]string{
		"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:override schemaLocation="base.xsd">
    <xs:notation name="foo" public="pubfoo2"/>
    <xs:simpleType name="Nota">
      <xs:restriction base="xs:NOTATION">
        <xs:enumeration value="foo"/>
        <xs:enumeration value="bez"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:override>
  <xs:notation name="bez" public="pubbez"/>
</xs:schema>`,
		"base.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="enum" type="Nota"/>
      <xs:attribute name="direct" type="xs:NOTATION"/>
    </xs:complexType>
  </xs:element>
  <xs:simpleType name="Nota">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="foo"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:notation name="foo" public="pubfoo"/>
</xs:schema>`,
	})
	var names []string
	for _, n := range schema.Notations() {
		names = append(names, n.Name().String())
	}
	if !strings.Contains(" "+strings.Join(names, " ")+" ", " bez ") {
		t.Fatalf("Notations() = %v, want the host's top-level bez among them", names)
	}
	root := xsd.QName{Local: "root"}
	for _, attr := range []string{"enum", "direct"} {
		for _, declared := range []string{"foo", "bez"} {
			got, _ := assessRecorded(t, schema, notationRoot(root, attr, declared, nil))
			wantSilence(t, got, declared+" is declared in the override host")
		}
	}
	got, _ := assessRecorded(t, schema, notationRoot(root, "direct", "qux", nil))
	wantUndeclaredNotation(t, got, "qux", xsd.QName{Local: "qux"})
}

// An element's ·initial value· is held to the same check under cvc-type clause
// 3.1.3, at the element, and a union reads only what its NOTATION member
// validated: union(integer, NOTATION) takes 42 as an xs:integer.
func TestANotationInitialValueMustNameADeclaredNotation(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="foo" public="pubfoo"/>
  <xs:simpleType name="IntOrNota"><xs:union memberTypes="xs:integer xs:NOTATION"/></xs:simpleType>
  <xs:element name="root" type="IntOrNota"/>
</xs:schema>`})
	valued := func(lexical string) *testElement {
		return &testElement{
			name: xsd.QName{Local: "root"},
			kids: []Child{TextChild(&testText{data: lexical, loc: loc(3, 7)})},
			loc:  loc(3, 1),
		}
	}
	for _, valid := range []string{"foo", "42"} {
		got, _ := assessRecorded(t, schema, valued(valid))
		wantSilence(t, got, valid+" is valid against IntOrNota")
	}
	got, _ := assessRecorded(t, schema, valued("bez"))
	viol := onlyCharge(t, got, ruleCvcType)
	if viol.Loc != loc(3, 1) {
		t.Errorf("Loc = %s, want the element's %s", viol.Loc, loc(3, 1))
	}
	if rule := causeRule(viol); rule != ruleCvcDatatypeValid {
		t.Errorf("wrapped rule = %q, want %q: %v", rule, ruleCvcDatatypeValid, viol)
	}
}
