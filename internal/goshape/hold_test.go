package goshape

import (
	"io"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// holdSchema is the schema TestParticleHold and TestChoiceRoot read: model
// group definitions for ParticleHold's groups and complex types for
// ChoiceRoot's, each named for its row.
const holdSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="seq"><xs:sequence><xs:element name="a"/><xs:element name="b"/></xs:sequence></xs:group>
  <xs:group name="seqEmptiable"><xs:sequence><xs:element name="a" minOccurs="0"/></xs:sequence></xs:group>
  <xs:group name="seqEdge"><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:group>
  <xs:group name="all"><xs:all><xs:element name="a"/></xs:all></xs:group>
  <xs:group name="choice"><xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice></xs:group>
  <xs:group name="choiceEmpty"><xs:choice/></xs:group>
  <xs:group name="choiceGroupEmptiable"><xs:choice><xs:sequence><xs:element name="a" minOccurs="0"/></xs:sequence><xs:element name="b"/></xs:choice></xs:group>
  <xs:group name="choiceEdge"><xs:choice><xs:element name="a" maxOccurs="unbounded"/><xs:element name="b"/></xs:choice></xs:group>
  <xs:complexType name="root"><xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice></xs:complexType>
  <xs:complexType name="rootEmpty"><xs:choice/></xs:complexType>
  <xs:complexType name="rootRef"><xs:group ref="choice"/></xs:complexType>
  <xs:complexType name="sequence"><xs:sequence><xs:element name="a"/><xs:element name="b"/></xs:sequence></xs:complexType>
  <xs:complexType name="optional"><xs:choice minOccurs="0"><xs:element name="a"/><xs:element name="b"/></xs:choice></xs:complexType>
  <xs:complexType name="mixed" mixed="true"><xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice></xs:complexType>
  <xs:complexType name="attribute"><xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice><xs:attribute name="x"/></xs:complexType>
  <xs:complexType name="anyAttribute"><xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice><xs:anyAttribute/></xs:complexType>
  <xs:complexType name="open"><xs:openContent><xs:any/></xs:openContent><xs:choice><xs:element name="a"/><xs:element name="b"/></xs:choice></xs:complexType>
  <xs:complexType name="emptiable"><xs:choice><xs:element name="a" minOccurs="0"/><xs:element name="b"/></xs:choice></xs:complexType>
  <xs:complexType name="simple"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
  <xs:complexType name="empty"/>
</xs:schema>`

// parseHoldSchema parses holdSchema.
func parseHoldSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	resolve := loader.ResolverFunc(func(_, location string) (io.ReadCloser, string, error) {
		return io.NopCloser(strings.NewReader(holdSchema)), location, nil
	})
	schema, err := parser.Parse("hold.xsd", parser.WithResolver(resolve))
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

// TestParticleHold pins ParticleHold's answer per arm, over the model group
// definitions of holdSchema at the occurrences each row names.
func TestParticleHold(t *testing.T) {
	schema := parseHoldSchema(t)
	bounded := func(min, max int) xsd.Occurs {
		o, err := xsd.NewOccurs(xsderr.Loc{}, min, max)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	unbounded, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		group  string
		occurs xsd.Occurs
		want   Hold
	}{
		{"seq", bounded(1, 1), Spliced},
		{"all", bounded(1, 1), Spliced},
		{"seqEmptiable", bounded(0, 1), Spliced},
		{"seq", bounded(0, 1), Occurrence},
		{"seqEmptiable", bounded(1, 2), Flattened},
		{"seqEdge", bounded(1, 2), Flattened},
		{"seq", unbounded, Occurrence},
		{"choiceGroupEmptiable", bounded(1, 1), Flattened},
		{"choice", bounded(1, 1), OnceChoice},
		{"choiceEmpty", bounded(1, 1), OnceChoice},
		{"choice", bounded(0, 1), Occurrence},
		{"choiceEdge", unbounded, Flattened},
		{"choice", unbounded, Occurrence},
		{"seq", bounded(0, 0), Occurrence},
	}
	for _, row := range rows {
		def, ok := schema.ModelGroup(xsd.QName{Local: row.group})
		if !ok {
			t.Fatalf("no model group definition %s", row.group)
		}
		got, err := ParticleHold(schema, def.ModelGroup(), row.occurs)
		if err != nil {
			t.Fatal(err)
		}
		if got != row.want {
			t.Errorf("ParticleHold(%s, %+v) = %d, want %d", row.group, row.occurs, got, row.want)
		}
	}
}

// TestChoiceRoot pins ChoiceRoot per conjunct over the complex types of
// holdSchema, and that each type it answers true for has a content
// {particle} ParticleHold answers OnceChoice for.
func TestChoiceRoot(t *testing.T) {
	schema := parseHoldSchema(t)
	for _, row := range []struct {
		typ  string
		want bool
	}{
		{"root", true},
		{"rootEmpty", true},
		{"rootRef", true},
		{"sequence", false},
		{"optional", false},
		{"mixed", false},
		{"attribute", false},
		{"anyAttribute", false},
		{"open", false},
		{"emptiable", false},
		{"simple", false},
		{"empty", false},
	} {
		td, ok := schema.Type(xsd.QName{Local: row.typ})
		if !ok {
			t.Fatalf("no type definition %s", row.typ)
		}
		ct := td.(xsd.ComplexType)
		got, err := ChoiceRoot(schema, ct)
		if err != nil {
			t.Fatal(err)
		}
		if got != row.want {
			t.Errorf("ChoiceRoot(%s) = %t, want %t", row.typ, got, row.want)
		}
		if !got {
			continue
		}
		content := ct.ContentType().(xsd.ElementContent)
		group, _, err := ModelGroupTerm(schema, content.Particle.Term())
		if err != nil {
			t.Fatal(err)
		}
		if hold, err := ParticleHold(schema, group, content.Particle.Occurs()); err != nil || hold != OnceChoice {
			t.Errorf("ChoiceRoot(%s) is true but ParticleHold answers %d, %v; want OnceChoice", row.typ, hold, err)
		}
	}
}
