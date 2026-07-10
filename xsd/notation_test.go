package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func strptr(s string) *string { return &s }

func TestNewNotationValid(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "png"}
	cases := []struct {
		name       string
		system     *string
		public     *string
		wantSystem string
		wantSysOK  bool
		wantPublic string
		wantPubOK  bool
	}{
		{"both", strptr("http://sys"), strptr("-//pub"), "http://sys", true, "-//pub", true},
		{"system-only", strptr("http://sys"), nil, "http://sys", true, "", false},
		{"public-only", nil, strptr("-//pub"), "", false, "-//pub", true},
		{"empty-system-is-present", strptr(""), nil, "", true, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := xsd.NewNotation(xsderr.Loc{}, name, c.system, c.public, nil)
			if err != nil {
				t.Fatalf("NewNotation unexpected error: %v", err)
			}
			if n.Name() != name {
				t.Errorf("Name() = %v, want %v", n.Name(), name)
			}
			gotSys, sysOK := n.SystemIdentifier()
			if sysOK != c.wantSysOK || gotSys != c.wantSystem {
				t.Errorf("SystemIdentifier() = (%q, %v), want (%q, %v)", gotSys, sysOK, c.wantSystem, c.wantSysOK)
			}
			gotPub, pubOK := n.PublicIdentifier()
			if pubOK != c.wantPubOK || gotPub != c.wantPublic {
				t.Errorf("PublicIdentifier() = (%q, %v), want (%q, %v)", gotPub, pubOK, c.wantPublic, c.wantPubOK)
			}
		})
	}
}

func TestNewNotationRejectsBothAbsent(t *testing.T) {
	_, err := xsd.NewNotation(xsderr.Loc{}, xsd.QName{Local: "n"}, nil, nil, nil)
	if err == nil {
		t.Fatal("NewNotation(nil, nil) succeeded, want n-props-correct error")
	}
	assertRule(t, err, "n-props-correct")
}

func TestNotationAnnotationsRoundTrip(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "second")}, nil),
	}
	n, err := xsd.NewNotation(xsderr.Loc{}, xsd.QName{Local: "n"}, strptr("sys"), nil, anns)
	if err != nil {
		t.Fatalf("NewNotation: %v", err)
	}
	got := n.Annotations()
	if len(got) != 2 {
		t.Fatalf("Annotations() len = %d, want 2", len(got))
	}
	if docs := got[0].Documentation(); len(docs) != 1 || docs[0].Content() != "first" {
		t.Errorf("Annotations()[0] documentation = %+v, want content %q", docs, "first")
	}
}

func TestNotationAnnotationsNilWhenEmpty(t *testing.T) {
	n, err := xsd.NewNotation(xsderr.Loc{}, xsd.QName{Local: "n"}, strptr("sys"), nil, nil)
	if err != nil {
		t.Fatalf("NewNotation: %v", err)
	}
	if got := n.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty {annotations}", got)
	}
}

func TestNotationDoesNotAliasConstructorSlice(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil),
	}
	n, err := xsd.NewNotation(xsderr.Loc{}, xsd.QName{Local: "n"}, strptr("sys"), nil, anns)
	if err != nil {
		t.Fatalf("NewNotation: %v", err)
	}
	// Mutate the ORIGINAL slice after construction.
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	got := n.Annotations()
	docs := got[0].Documentation()
	if len(docs) != 1 || docs[0].Content() != "keep" {
		t.Errorf("Notation aliased the constructor slice: got content %q, want %q", docs[0].Content(), "keep")
	}
}

func TestNotationAnnotationsAccessorDoesNotAlias(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil),
	}
	n, err := xsd.NewNotation(xsderr.Loc{}, xsd.QName{Local: "n"}, strptr("sys"), nil, anns)
	if err != nil {
		t.Fatalf("NewNotation: %v", err)
	}
	// Mutate the RETURNED slice.
	first := n.Annotations()
	first[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	second := n.Annotations()
	docs := second[0].Documentation()
	if len(docs) != 1 || docs[0].Content() != "keep" {
		t.Errorf("Annotations() returned an aliased slice: got content %q, want %q", docs[0].Content(), "keep")
	}
}
