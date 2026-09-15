package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceXmlnsAttributeDeclarationRejected pins no-xmlns (§3.2.6.3): an
// attribute declaration whose {name} is the bare string "xmlns" is rejected,
// local or top-level, and charged at its own position.
//
// One row per CODE PATH, because the two productions are separate call sites and
// a top-level-only implementation leaves the local rows accepted outright: the
// declaration reaches no by-name index, so nothing downstream charges it
// anything. The two local containers §3.2.2 admits share one Go call site
// (produceLocalAttribute), and both are here because a container-specific
// mistake in how the caller reaches it is not otherwise visible.
//
// The charged LINE is what pins the subject. The {name} in the message is the
// constant the rule names and so discriminates nothing: a charge raised against
// some other declaration in the document carries the identical text, and only
// the position tells the two apart. No W3C suite fixture puts two xmlns-named
// declarations in one document, so that separation is pinned here or nowhere
// (#1465).
func TestProduceXmlnsAttributeDeclarationRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts on
	// line 2 and puts the offending <attribute> on wantLine, so the charged
	// position is pinned to the declaration's own element.
	cases := []struct {
		name     string
		doc      string
		wantLine int
	}{
		{
			// §3.2.2.1 dcl.att.global. This is msData/attribute/attKa012, minus the
			// dangling ref= that charged it src-resolve at finalize instead.
			name:     "top-level",
			doc:      wrap("", "\n"+`<xs:attribute name="xmlns" type="xs:string"/>`),
			wantLine: 2,
		},
		{
			// §3.2.2.2 dcl.att.local, reached through produceAttributeGroup.
			name: "local in <attributeGroup>",
			doc: wrap("", "\n"+
				`<xs:attributeGroup name="attG">`+"\n"+
				`<xs:attribute name="xmlns"/>`+"\n"+
				`</xs:attributeGroup>`),
			wantLine: 3,
		},
		{
			// §3.2.2.2 again, reached through the <complexType> attribute walk.
			name: "local in <complexType>",
			doc: wrap("", "\n"+
				`<xs:complexType name="CT">`+"\n"+
				`<xs:attribute name="xmlns"/>`+"\n"+
				`</xs:complexType>`),
			wantLine: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			assertRule(t, err, "no-xmlns")
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("error %v carries no *xsderr.Error", err)
			}
			if xe.Loc.Line != tc.wantLine {
				t.Fatalf("charged at line %d, want the <attribute> at line %d (%v)", xe.Loc.Line, tc.wantLine, err)
			}
			const want = `attribute declaration has "xmlns" as its {name}`
			if !strings.Contains(xe.Msg, want) {
				t.Fatalf("message = %q, want it to name the offending property: %q", xe.Msg, want)
			}
		})
	}
}

// TestProduceXmlnsPrefixedAttributeNameAccepted is the guard against a PREFIX
// comparison in place of the exact one. "The {name} … must not match xmlns" is a
// whole-string match, so xmlnsx and xmlns-1 are ordinary NCNames and ordinary
// attribute declarations.
//
// The colonized forms are NOT that guard, though §3.2.6.3's Note is about them:
// declarationName rejects a name outside NCName's ·lexical space· before
// rejectXmlnsName is reached, so xmlns: and xmlns:a keep their
// cvc-datatype-valid charge under a prefix comparison too and
// TestProduceXmlnsAttributeNameChargedNCName stays green through that mutation
// (verified). What a prefix comparison actually breaks is the name that is a
// valid NCName and reaches the check, which is what this pins — and no W3C suite
// fixture declares an attribute whose name merely starts with "xmlns", so it is
// pinned here or nowhere (#1465).
func TestProduceXmlnsPrefixedAttributeNameAccepted(t *testing.T) {
	for _, name := range []string{"xmlnsx", "xmlns-1", "xmlns_"} {
		t.Run(name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:attribute name="`+name+`" type="xs:string"/>`))
			if err != nil {
				t.Fatalf("Produce: %v, want %q accepted: no-xmlns (§3.2.6.3) forbids the {name} that MATCHES xmlns, not one beginning with it", err, name)
			}
			if _, ok := s.Attribute(xsd.QName{Local: name}); !ok {
				t.Fatalf("{attribute declarations} = %v, want the declaration named {,%s}", s.Attributes(), name)
			}
		})
	}
}

// TestProduceXmlnsElementDeclarationAccepted is the guard against the wrong
// implementation of the test above: siting the check in declarationName, the one
// helper every declaration form shares. §3.2.6.3 constrains attribute
// declarations only and §3.3.6.1 e-props-correct carries no equivalent clause,
// so an element named "xmlns" is valid — msData/element/elemA016, which that
// mistake would regress.
func TestProduceXmlnsElementDeclarationAccepted(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:element name="xmlns"/>`))
	if err != nil {
		t.Fatalf("Produce: %v, want an element declaration named \"xmlns\" accepted: no-xmlns (§3.2.6.3) is stated over attribute declarations and §3.3.6.1 has no equivalent clause", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "xmlns"}); !ok {
		t.Fatalf("{element declarations} = %v, want the declaration named {,xmlns}", s.Elements())
	}
}
