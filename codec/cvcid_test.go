package codec

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// idSchema declares, in urn:s, the document elements whose simple types are
// xs:ID, xs:IDREF or built from one, each reach of encodeWalk.atomicEncodes
// at least once: doc, whose item elements carry a required xs:ID attribute
// id and whose ref child is an xs:IDREF; p, whose a children are of xs:ID,
// so that each binds p (Structures §3.17.5.2 sic-id); u, whose c children
// carry an optional xs:ID attribute id with the default dd; and, each with
// no other ID or IDREF in it, e of xs:ID, sc of simple content extending
// xs:ID, at of a required xs:IDREF attribute ref, refs of xs:IDREFS, a list
// of xs:IDREF, and un of a union of xs:int and xs:IDREF.
const idSchema = `
  <xs:element name="doc"><xs:complexType><xs:sequence>
    <xs:element name="item" maxOccurs="unbounded"><xs:complexType><xs:attribute name="id" type="xs:ID" use="required"/></xs:complexType></xs:element>
    <xs:element name="ref" type="xs:IDREF"/>
  </xs:sequence></xs:complexType></xs:element>
  <xs:element name="p"><xs:complexType><xs:sequence><xs:element name="a" type="xs:ID" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="u"><xs:complexType><xs:sequence>
    <xs:element name="c" maxOccurs="unbounded"><xs:complexType><xs:attribute name="id" type="xs:ID" default="dd"/></xs:complexType></xs:element>
  </xs:sequence></xs:complexType></xs:element>
  <xs:element name="e" type="xs:ID"/>
  <xs:element name="sc"><xs:complexType><xs:simpleContent><xs:extension base="xs:ID"/></xs:simpleContent></xs:complexType></xs:element>
  <xs:element name="at"><xs:complexType><xs:attribute name="ref" type="xs:IDREF" use="required"/></xs:complexType></xs:element>
  <xs:element name="refs" type="xs:IDREFS"/>
  <xs:element name="un"><xs:simpleType><xs:union memberTypes="xs:int xs:IDREF"/></xs:simpleType></xs:element>`

// Codegen's types for idSchema's doc, its item elements and u's c elements,
// whose id field holds the default for an absent attribute (codegen's
// package doc, "# Attributes"); p; u; sc; at; and un's sealed interface
// and its variants.
type (
	idDoc struct {
		Item []idItem
		Ref  string
	}
	idItem       struct{ Id string }
	idP          struct{ A []string }
	idU          struct{ C []idItem }
	idSC         struct{ Value string }
	idAt         struct{ Ref string }
	idUnion      interface{ isIDUnion() }
	idUnionInt   int32
	idUnionIDREF string
)

func (idUnionInt) isIDUnion()   {}
func (idUnionIDREF) isIDUnion() {}

// idUnionBranches is the WithBranches set of idUnion.
var idUnionBranches = WithBranches[idUnion](idUnionInt(0), idUnionIDREF(""))

// TestAppendID pins Append's encoding of xs:ID and xs:IDREF values, which it
// assesses (Structures §3.3.4.5 cvc-id): two distinct ids on distinct
// elements and an IDREF naming one are written, accepted by validate and
// decoded back equal, as are two equal xs:ID children of one parent, which
// bind that one parent (§3.17.5.2 sic-id), a binding of one member; and
// CheckAppend returns nil for each.
func TestAppendID(t *testing.T) {
	schema := schemaOf(t, idSchema)
	doc := idDoc{Item: []idItem{{Id: "a"}, {Id: "b"}}, Ref: "b"}
	roundTrip(t, schema, sname("doc"), doc, `<doc xmlns="urn:s"><item id="a"/><item id="b"/><ref>b</ref></doc>`, func(x, y idDoc) bool {
		return slices.Equal(x.Item, y.Item) && x.Ref == y.Ref
	})
	p := idP{A: []string{"x", "x"}}
	roundTrip(t, schema, sname("p"), p, `<p xmlns="urn:s"><a>x</a><a>x</a></p>`, func(x, y idP) bool { return slices.Equal(x.A, y.A) })
	if err := CheckAppend[idDoc](schema, sname("doc")); err != nil {
		t.Errorf("CheckAppend[idDoc] = %v", err)
	}
	if err := CheckAppend[idP](schema, sname("p")); err != nil {
		t.Errorf("CheckAppend[idP] = %v", err)
	}
}

// TestAppendIDViolation pins item 8 of Append's doc for cvc-id, each row
// refused by assessedCharge's terms under the clause its message names: two
// equal ids on distinct elements (clause 2); an IDREF naming no id (clause
// 1); two c elements whose id field holds the default dd, which Append writes
// (clause 2); and, at each reach of encodeWalk.atomicEncodes, an xs:ID value
// that binds no element, the document element's or its simple content's
// (clause 1), and an IDREF naming no id, an attribute's, a list item's and a
// union member's (clause 1). Every row fails with the walk's trigger
// removed. With one reach's call into the walk given a walk of its own, whose
// flag no one reads, the rows that fail are: for the element reach, element,
// list item and union member, whose types each document element's
// declaration reaches first; for simple content, simple content; for the
// attribute reach, attribute and two defaults; for the list item, list item;
// and for the union member, union member.
func TestAppendIDViolation(t *testing.T) {
	schema := schemaOf(t, idSchema)
	pre := []byte("pre")
	rows := []struct {
		name, clause, subject string
		append                func() ([]byte, error)
	}{
		{"equal ids on distinct elements", "2", `the id "a"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("doc"), idDoc{Item: []idItem{{Id: "a"}, {Id: "a"}}, Ref: "a"})
		}},
		{"an IDREF naming no id", "1", `the id "z"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("doc"), idDoc{Item: []idItem{{Id: "a"}}, Ref: "z"})
		}},
		{"two defaults", "2", `the id "dd"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("u"), idU{C: []idItem{{Id: "dd"}, {Id: "dd"}}})
		}},
		{"element", "1", `the id "x"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("e"), "x")
		}},
		{"simple content", "1", `the id "x"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("sc"), idSC{Value: "x"})
		}},
		{"attribute", "1", `the id "z"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("at"), idAt{Ref: "z"})
		}},
		{"list item", "1", `the id "z"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("refs"), []string{"z"})
		}},
		{"union member", "1", `the id "z"`, func() ([]byte, error) {
			return appendChecked(t, pre, schema, strict.New(), sname("un"), idUnion(idUnionIDREF("z")), idUnionBranches)
		}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			got, err := row.append()
			assessedCharge(t, got, err, "cvc-id", row.clause, row.subject)
		})
	}
}
