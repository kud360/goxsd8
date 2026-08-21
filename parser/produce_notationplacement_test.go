package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// misplacedNotation is the <notation> every row below writes, on a line of its
// own so the rejection can be asserted to name it at line 3 column 1 while its
// enclosing element opens on line 2.
const misplacedNotation = `<xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>`

// notationInside wraps prefix + the notation + suffix in a <schema>, one line
// each.
func notationInside(prefix, suffix string) string {
	return wrap("", "\n"+prefix+"\n"+misplacedNotation+"\n"+suffix)
}

// TestProduceMisplacedNotationRejected pins where <notation> may stand: the
// schema for schema documents declares it in the xs:schemaTop group alone
// (xmlschema11-1.md:4462), referenced only by <schema> (:4562) and <override>
// (:5577), so every row here is a grammar fault. <redefine> is among them
// because its own model reaches xs:redefinable (:5558), which omits <notation>.
//
// Before this check each of these documents was silently ACCEPTED: produceNotation
// was reached only from run's dispatch over <schema>'s children, and a producer
// whose grammar does not mention <notation> simply never looked at the element.
// The rows are the suite's own witnesses, one per illegal parent, so a walk that
// fails to reach some position — a facet's child, an identity constraint's
// <selector>, an unfollowed <include> — is caught here rather than in the lane
// score.
//
// The fault carries no rule ID (§5.1's first bullet, :4296): §3.14.3 and §3.14.4
// both read "None as such." (:3409, :3413), so a verdict here would be
// fabricated (STYLE E2).
func TestProduceMisplacedNotationRejected(t *testing.T) {
	for _, tc := range []struct {
		name   string
		parent string
		prefix string
		suffix string
	}{
		{"all (notatF001)", "all", `<xs:complexType name="foo"><xs:all>`, `</xs:all></xs:complexType>`},
		{"annotation (notatF003)", "annotation", `<xs:annotation>`, `</xs:annotation>`},
		{"any (notatF005)", "any", `<xs:any>`, `</xs:any>`},
		{"anyAttribute (notatF007)", "anyAttribute", `<xs:complexType name="foo"><xs:anyAttribute>`, `</xs:anyAttribute></xs:complexType>`},
		{"attribute (notatF011)", "attribute", `<xs:complexType name="c"><xs:attribute name="foo">`, `</xs:attribute></xs:complexType>`},
		{"attributeGroup (notatF013)", "attributeGroup", `<xs:attributeGroup name="bar">`, `</xs:attributeGroup>`},
		{"choice (notatF015)", "choice", `<xs:complexType name="foo"><xs:choice>`, `</xs:choice></xs:complexType>`},
		{"complexType (notatF019)", "complexType", `<xs:complexType name="foo">`, `</xs:complexType>`},
		{"element (notatF023)", "element", `<xs:element name="foo">`, `</xs:element>`},
		{"enumeration (notatF025)", "enumeration", `<xs:simpleType name="foo"><xs:restriction base="xs:string"><xs:enumeration value="1 2">`, `</xs:enumeration></xs:restriction></xs:simpleType>`},
		{"extension (notatF027)", "extension", `<xs:complexType name="bar"><xs:complexContent><xs:extension>`, `</xs:extension></xs:complexContent></xs:complexType>`},
		{"field (notatF029)", "field", `<xs:key name="foo"><xs:field>`, `</xs:field></xs:key>`},
		{"all in a named group (notatF031)", "all", `<xs:group name="foo"><xs:all>`, `</xs:all></xs:group>`},
		{"include (notatF035)", "include", `<xs:include>`, `</xs:include>`},
		{"key (notatF037)", "key", `<xs:element name="foo"><xs:key name="bar">`, `</xs:key></xs:element>`},
		{"keyref (notatF039)", "keyref", `<xs:keyref name="personRef" refer="fullName">`, `</xs:keyref>`},
		{"length (notatF041)", "length", `<xs:simpleType name="foo"><xs:restriction base="xs:string"><xs:length value="8">`, `</xs:length></xs:restriction></xs:simpleType>`},
		{"maxInclusive (notatF045)", "maxInclusive", `<xs:simpleType name="foo"><xs:restriction base="xs:integer"><xs:maxInclusive value="0">`, `</xs:maxInclusive></xs:restriction></xs:simpleType>`},
		{"minInclusive (notatF049)", "minInclusive", `<xs:simpleType name="foo"><xs:restriction base="xs:integer"><xs:minInclusive value="0">`, `</xs:minInclusive></xs:restriction></xs:simpleType>`},
		{"pattern (notatF053)", "pattern", `<xs:simpleType name="foo"><xs:restriction base="xs:integer"><xs:pattern value="0">`, `</xs:pattern></xs:restriction></xs:simpleType>`},
		{"redefine (notatF055)", "redefine", `<xs:redefine schemaLocation="foo">`, `</xs:redefine>`},
		{"restriction (notatF057)", "restriction", `<xs:simpleType name="foo"><xs:restriction base="xs:string">`, `</xs:restriction></xs:simpleType>`},
		{"selector (notatF061)", "selector", `<xs:element name="bar"><xs:key name="foo"><xs:selector xpath="foo">`, `</xs:selector></xs:key></xs:element>`},
		{"sequence (notatF063)", "sequence", `<xs:complexType name="foo"><xs:sequence>`, `</xs:sequence></xs:complexType>`},
		{"simpleType (notatF067)", "simpleType", `<xs:simpleType name="foo">`, `</xs:simpleType>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, notationInside(tc.prefix, tc.suffix))
			if err == nil {
				t.Fatalf("Produce accepted a <notation> inside <%s>", tc.parent)
			}
			if _, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, want a plain grammar fault rather than a rule verdict", err)
			}
			if !strings.Contains(err.Error(), "<notation> at "+produceURI+":3:1") {
				t.Errorf("error = %v, want it to name the <notation> at line 3", err)
			}
			if !strings.Contains(err.Error(), "<"+tc.parent+"> at "+produceURI+":2:") {
				t.Errorf("error = %v, want it to name the enclosing <%s> on line 2", err, tc.parent)
			}
		})
	}
}

// TestProduceNotationInLaxContentAccepted pins the exclusion the placement guard
// must carry: <appinfo> and <documentation> hold <xs:any processContents="lax">
// content (xmlschema11-1.md:5727, :5740), so an element that happens to be named
// {XMLSchemaNS}notation there is content this grammar does not govern. notatF009
// is the suite's witness, and it is valid.
func TestProduceNotationInLaxContentAccepted(t *testing.T) {
	for _, tc := range []struct {
		name string
		host string
	}{
		{"appinfo (notatF009)", "appinfo"},
		{"documentation", "documentation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := wrap("", `<xs:annotation><xs:`+tc.host+`>`+misplacedNotation+`</xs:`+tc.host+`></xs:annotation>`)
			schema, err := produce(t, doc)
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			if got := len(schema.Notations()); got != 0 {
				t.Errorf("Notations() = %d declarations, want 0: lax wildcard content declares none", got)
			}
		})
	}
}

// TestParseOverrideNotationAccepted pins the second legal parent: <override>'s
// content model references xs:schemaTop (xmlschema11-1.md:5577) exactly as
// <schema>'s does, so a <notation> child of an <override> is admitted and
// substitutes for the identically-named declaration of the overridden document
// (§F.2 clause 1). Only <schema> and <override> are on that list; if the guard
// were keyed on <schema> alone this document would be rejected.
func TestParseOverrideNotationAccepted(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>`+
			`</xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:notation name="jpeg" public="stale" system="stale.exe"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	notations := s.Notations()
	if len(notations) != 1 {
		t.Fatalf("Notations() = %d declarations, want 1", len(notations))
	}
	if got, want := notations[0].Name(), (xsd.QName{Space: "urn:a", Local: "jpeg"}); got != want {
		t.Fatalf("Name() = %s, want %s", got, want)
	}
	public, ok := notations[0].PublicIdentifier()
	if !ok || public != "image/jpeg" {
		t.Errorf("PublicIdentifier() = %q, %t, want the overriding declaration's %q", public, ok, "image/jpeg")
	}
}
