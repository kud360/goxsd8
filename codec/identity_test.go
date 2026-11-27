package codec

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsderr"
)

// identitySchema declares, in urn:s, r, whose k elements hold a key kk and
// whose f elements a keyref kr to it; u, whose k elements hold a unique uu;
// n, whose repeated g particle declares a unique gu over its own k elements,
// so that a k value repeats across two g elements and never within one
// (cvc-identity-constraint clause 2: the target node set is g's); o, whose k
// elements hold a key ok on their optional attribute id; and m, whose k
// elements hold a unique mu on their x children, of which a k holds up to
// two.
const identitySchema = `
  <xs:element name="r">
    <xs:complexType><xs:sequence>
      <xs:element name="k" type="xs:string" maxOccurs="unbounded"/>
      <xs:element name="f" type="xs:string" maxOccurs="unbounded"/>
    </xs:sequence></xs:complexType>
    <xs:key name="kk"><xs:selector xpath="s:k"/><xs:field xpath="."/></xs:key>
    <xs:keyref name="kr" refer="s:kk"><xs:selector xpath="s:f"/><xs:field xpath="."/></xs:keyref>
  </xs:element>
  <xs:element name="u">
    <xs:complexType><xs:sequence>
      <xs:element name="k" type="xs:string" maxOccurs="unbounded"/>
    </xs:sequence></xs:complexType>
    <xs:unique name="uu"><xs:selector xpath="s:k"/><xs:field xpath="."/></xs:unique>
  </xs:element>
  <xs:element name="n">
    <xs:complexType><xs:sequence>
      <xs:element name="g" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:element name="k" type="xs:string" maxOccurs="unbounded"/>
        </xs:sequence></xs:complexType>
        <xs:unique name="gu"><xs:selector xpath="s:k"/><xs:field xpath="."/></xs:unique>
      </xs:element>
    </xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="o">
    <xs:complexType><xs:sequence>
      <xs:element name="k" maxOccurs="unbounded">
        <xs:complexType><xs:attribute name="id" type="xs:string"/></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:key name="ok"><xs:selector xpath="s:k"/><xs:field xpath="@id"/></xs:key>
  </xs:element>
  <xs:element name="m">
    <xs:complexType><xs:sequence>
      <xs:element name="k" maxOccurs="unbounded">
        <xs:complexType><xs:sequence>
          <xs:element name="x" type="xs:string" maxOccurs="2"/>
        </xs:sequence></xs:complexType>
      </xs:element>
    </xs:sequence></xs:complexType>
    <xs:unique name="mu"><xs:selector xpath="s:k"/><xs:field xpath="s:x"/></xs:unique>
  </xs:element>`

// Codegen's types for identitySchema's r, u, n, o and m.
type (
	identityR  struct{ K, F []string }
	identityU  struct{ K []string }
	identityN  struct{ G []identityG }
	identityG  struct{ K []string }
	identityO  struct{ K []identityOK }
	identityOK struct{ Id *string }
	identityM  struct{ K []identityMK }
	identityMK struct{ X []string }
)

// TestAppendIdentityConstraint pins Append's encoding of a declaration with
// {identity-constraint definitions} whose paths icpath compiles: a value
// satisfying r's key and the keyref to it, and n's unique per g, is written,
// accepted by validate and decoded back equal, after a dst prefix that is no
// XML, which Append's assessment never reads; and CheckAppend returns nil.
func TestAppendIdentityConstraint(t *testing.T) {
	schema := schemaOf(t, identitySchema)
	r := identityR{K: []string{"a", "b"}, F: []string{"b", "a", "b"}}
	roundTrip(t, schema, sname("r"), r, `<r xmlns="urn:s"><k>a</k><k>b</k><f>b</f><f>a</f><f>b</f></r>`, func(a, b identityR) bool {
		return slices.Equal(a.K, b.K) && slices.Equal(a.F, b.F)
	})
	n := identityN{G: []identityG{{K: []string{"a", "b"}}, {K: []string{"a"}}}}
	roundTrip(t, schema, sname("n"), n, `<n xmlns="urn:s"><g><k>a</k><k>b</k></g><g><k>a</k></g></n>`, func(a, b identityN) bool {
		return slices.EqualFunc(a.G, b.G, func(x, y identityG) bool { return slices.Equal(x.K, y.K) })
	})
	got, err := appendChecked(t, []byte("pre<"), schema, strict.New(), sname("r"), r)
	if err != nil || !strings.HasPrefix(string(got), "pre<<r ") {
		t.Fatalf("Append after a prefix = %q, %v", got, err)
	}
	if err := CheckAppend[identityR](schema, sname("r")); err != nil {
		t.Errorf("CheckAppend = %v", err)
	}
}

// TestAppendIdentityViolation pins item 8 of Append's doc: a value whose
// written document validate charges, each row by the cvc-identity-constraint
// clause its message names — a duplicate key (4.2.2), a duplicate unique
// (4.1), a dangling keyref (4.3), a duplicate unique within one particle's
// element, a key's target node with no id (4.2.1) and a field selecting two
// nodes (3) — returns dst unextended and an error that errors.Is reports as
// ErrInvalidValue, no *DeclineError and not errors.ErrUnsupported, opening
// with codec's statement of the charge, its first violation reached by
// errors.As and xsderr.RuleOf, every one by ViolationsOf, each charged to the
// written document under no URI and naming the constraint; CheckAppend
// returns nil for it.
func TestAppendIdentityViolation(t *testing.T) {
	schema := schemaOf(t, identitySchema)
	rows := []struct {
		name, clause, constraint string
		append                   func() ([]byte, error)
	}{
		{"duplicate key", "4.2.2", "{urn:s}kk", func() ([]byte, error) {
			return appendChecked(t, []byte("pre"), schema, strict.New(), sname("r"), identityR{K: []string{"a", "a"}, F: []string{"a"}})
		}},
		{"duplicate unique", "4.1", "{urn:s}uu", func() ([]byte, error) {
			return appendChecked(t, []byte("pre"), schema, strict.New(), sname("u"), identityU{K: []string{"a", "b", "a"}})
		}},
		{"dangling keyref", "4.3", "{urn:s}kr", func() ([]byte, error) {
			return appendChecked(t, []byte("pre"), schema, strict.New(), sname("r"), identityR{K: []string{"a"}, F: []string{"a", "z"}})
		}},
		{"particle duplicate unique", "4.1", "{urn:s}gu", func() ([]byte, error) {
			return appendChecked(t, []byte("pre"), schema, strict.New(), sname("n"), identityN{G: []identityG{{K: []string{"a"}}, {K: []string{"b", "b"}}}})
		}},
		{"absent key field", "4.2.1", "{urn:s}ok", func() ([]byte, error) {
			id := "a"
			return appendChecked(t, []byte("pre"), schema, strict.New(), sname("o"), identityO{K: []identityOK{{Id: &id}, {}}})
		}},
		{"field of two nodes", "3", "{urn:s}mu", func() ([]byte, error) {
			return appendChecked(t, []byte("pre"), schema, strict.New(), sname("m"), identityM{K: []identityMK{{X: []string{"a"}}, {X: []string{"b", "c"}}}})
		}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			got, err := row.append()
			assessedCharge(t, got, err, "cvc-identity-constraint", row.clause, row.constraint)
		})
	}
}

// assessedCharge fails t unless got and err are item 8 of Append's doc for
// an Append after the dst "pre": got is dst unextended, and err an error
// errors.Is reports as ErrInvalidValue, no *DeclineError and not
// errors.ErrUnsupported, opening with codec's statement of the charge, whose
// first violation, reached by errors.As and xsderr.RuleOf, is of rule and
// leads ViolationsOf, every one charged to the written document under no
// URI, and names subject and, where clause is other than "", "clause " and
// clause.
func assessedCharge(t *testing.T, got []byte, err error, rule xsderr.Rule, clause, subject string) {
	t.Helper()
	if string(got) != "pre" {
		t.Errorf("Append returned %q, want dst unextended", got)
	}
	var d *DeclineError
	if !errors.Is(err, ErrInvalidValue) || errors.As(err, &d) || errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Append returned %v, want an error errors.Is reports as ErrInvalidValue and no decline", err)
	}
	const opening = "codec: the instance is invalid against the schema: ?:"
	if !strings.HasPrefix(err.Error(), opening) {
		t.Errorf("error %q does not open with %q", err, opening)
	}
	var xe *xsderr.Error
	if r, _ := xsderr.RuleOf(err); !errors.As(err, &xe) || r != rule {
		t.Fatalf("error %v reaches no %s *xsderr.Error", err, rule)
	}
	vs := ViolationsOf(err)
	if len(vs) == 0 || vs[0] != xe {
		t.Fatalf("ViolationsOf = %v, want the first violation %v first", vs, xe)
	}
	for _, v := range vs {
		if v.Loc.URI != "" || v.Loc.Line != 1 {
			t.Errorf("violation %v is not charged to the written document under no URI", v)
		}
	}
	if !strings.Contains(xe.Error(), subject) || clause != "" && !strings.Contains(xe.Error(), "clause "+clause) {
		t.Errorf("violation %q names neither %s nor clause %s", xe, subject, clause)
	}
}

// specialFieldSchema declares, in urn:s, r, whose k elements, of
// xs:anySimpleType, hold a unique su.
const specialFieldSchema = `
  <xs:element name="r">
    <xs:complexType><xs:sequence>
      <xs:element name="k" type="xs:anySimpleType" maxOccurs="unbounded"/>
    </xs:sequence></xs:complexType>
    <xs:unique name="su"><xs:selector xpath="s:k"/><xs:field xpath="."/></xs:unique>
  </xs:element>`

// specialFieldR is codegen's type for specialFieldSchema's r.
type specialFieldR struct{ K []string }

// TestAppendIdentityUndecided pins item 9 of Append's doc: a value whose
// written document validate leaves Undecided, two k elements of
// xs:anySimpleType whose lexicals differ, a ·key-sequence· member pair validate
// compares only where the lexicals are byte-identical (the validate package
// doc), returns dst unextended and an error errors.As reaches as an
// *UndecidedError for cvc-identity-constraint, which errors.Is reports as
// errors.ErrUnsupported and not as ErrInvalidValue; CheckAppend returns nil
// for it.
func TestAppendIdentityUndecided(t *testing.T) {
	schema := schemaOf(t, specialFieldSchema)
	got, err := appendChecked(t, []byte("pre"), schema, strict.New(), sname("r"), specialFieldR{K: []string{"a", "b"}})
	if string(got) != "pre" {
		t.Errorf("Append returned %q, want dst unextended", got)
	}
	var ue *UndecidedError
	if !errors.As(err, &ue) || !errors.Is(err, errors.ErrUnsupported) || errors.Is(err, ErrInvalidValue) {
		t.Fatalf("Append returned %v, want an *UndecidedError that errors.Is reports as errors.ErrUnsupported and not ErrInvalidValue", err)
	}
	if rule := ue.Unevaluated().Rule(); rule != "cvc-identity-constraint" {
		t.Errorf("Unevaluated().Rule() = %q, want cvc-identity-constraint", rule)
	}
	if err := CheckAppend[specialFieldR](schema, sname("r")); err != nil {
		t.Errorf("CheckAppend = %v", err)
	}
}
