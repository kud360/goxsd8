package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestNewAppInfo(t *testing.T) {
	t.Run("with-source", func(t *testing.T) {
		src := "http://src"
		a := xsd.NewAppInfo(&src, "<x/>")
		got, ok := a.Source()
		if !ok || got != src {
			t.Errorf("Source() = (%q, %v), want (%q, true)", got, ok, src)
		}
		if a.Content() != "<x/>" {
			t.Errorf("Content() = %q, want %q", a.Content(), "<x/>")
		}
	})
	t.Run("absent-source", func(t *testing.T) {
		a := xsd.NewAppInfo(nil, "raw")
		if got, ok := a.Source(); ok {
			t.Errorf("Source() = (%q, true), want (_, false)", got)
		}
		if a.Content() != "raw" {
			t.Errorf("Content() = %q, want %q", a.Content(), "raw")
		}
	})
	t.Run("empty-source-is-present", func(t *testing.T) {
		empty := ""
		a := xsd.NewAppInfo(&empty, "")
		got, ok := a.Source()
		if !ok || got != "" {
			t.Errorf("Source() = (%q, %v), want (\"\", true)", got, ok)
		}
	})
}

func TestNewDocumentation(t *testing.T) {
	t.Run("source-and-lang", func(t *testing.T) {
		src, lang := "http://src", "en"
		d := xsd.NewDocumentation(&src, &lang, "hello")
		if got, ok := d.Source(); !ok || got != src {
			t.Errorf("Source() = (%q, %v), want (%q, true)", got, ok, src)
		}
		if got, ok := d.Lang(); !ok || got != lang {
			t.Errorf("Lang() = (%q, %v), want (%q, true)", got, ok, lang)
		}
		if d.Content() != "hello" {
			t.Errorf("Content() = %q, want %q", d.Content(), "hello")
		}
	})
	t.Run("both-absent", func(t *testing.T) {
		d := xsd.NewDocumentation(nil, nil, "hi")
		if got, ok := d.Source(); ok {
			t.Errorf("Source() = (%q, true), want (_, false)", got)
		}
		if got, ok := d.Lang(); ok {
			t.Errorf("Lang() = (%q, true), want (_, false)", got)
		}
	})
}

func TestNewAttr(t *testing.T) {
	name := xsd.QName{Space: "urn:x", Local: "a"}
	at := xsd.NewAttr(name, "v")
	if at.Name() != name {
		t.Errorf("Name() = %v, want %v", at.Name(), name)
	}
	if at.Value() != "v" {
		t.Errorf("Value() = %q, want %q", at.Value(), "v")
	}
}

func TestNewAnnotationRoundTrip(t *testing.T) {
	appinfo := []xsd.AppInfo{xsd.NewAppInfo(nil, "ai0"), xsd.NewAppInfo(nil, "ai1")}
	docs := []xsd.Documentation{xsd.NewDocumentation(nil, strptr("en"), "d0")}
	attrs := []xsd.Attr{xsd.NewAttr(xsd.QName{Local: "k"}, "v")}

	a := xsd.NewAnnotation(appinfo, docs, attrs)

	gotAI := a.AppInfo()
	if len(gotAI) != 2 || gotAI[0].Content() != "ai0" || gotAI[1].Content() != "ai1" {
		t.Errorf("AppInfo() = %+v, want document order ai0, ai1", gotAI)
	}
	gotDocs := a.Documentation()
	if len(gotDocs) != 1 || gotDocs[0].Content() != "d0" {
		t.Errorf("Documentation() = %+v, want [d0]", gotDocs)
	}
	if lang, ok := gotDocs[0].Lang(); !ok || lang != "en" {
		t.Errorf("Documentation()[0].Lang() = (%q, %v), want (en, true)", lang, ok)
	}
	gotAttrs := a.Attributes()
	if len(gotAttrs) != 1 || gotAttrs[0].Value() != "v" {
		t.Errorf("Attributes() = %+v, want [k=v]", gotAttrs)
	}
}

func TestNewAnnotationEmptyInputs(t *testing.T) {
	a := xsd.NewAnnotation(nil, nil, nil)
	if got := a.AppInfo(); got != nil {
		t.Errorf("AppInfo() = %v, want nil", got)
	}
	if got := a.Documentation(); got != nil {
		t.Errorf("Documentation() = %v, want nil", got)
	}
	if got := a.Attributes(); got != nil {
		t.Errorf("Attributes() = %v, want nil", got)
	}
	// Also confirm empty (non-nil) slices in yield nil out.
	b := xsd.NewAnnotation([]xsd.AppInfo{}, []xsd.Documentation{}, []xsd.Attr{})
	if b.AppInfo() != nil || b.Documentation() != nil || b.Attributes() != nil {
		t.Errorf("empty-slice inputs did not yield nil accessors")
	}
}

func TestAnnotationDoesNotAliasConstructorSlices(t *testing.T) {
	appinfo := []xsd.AppInfo{xsd.NewAppInfo(nil, "keep")}
	docs := []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}
	attrs := []xsd.Attr{xsd.NewAttr(xsd.QName{Local: "k"}, "keep")}

	a := xsd.NewAnnotation(appinfo, docs, attrs)

	// Mutate the ORIGINAL backing arrays.
	appinfo[0] = xsd.NewAppInfo(nil, "tampered")
	docs[0] = xsd.NewDocumentation(nil, nil, "tampered")
	attrs[0] = xsd.NewAttr(xsd.QName{Local: "k"}, "tampered")

	if a.AppInfo()[0].Content() != "keep" {
		t.Errorf("AppInfo aliased constructor slice: got %q", a.AppInfo()[0].Content())
	}
	if a.Documentation()[0].Content() != "keep" {
		t.Errorf("Documentation aliased constructor slice: got %q", a.Documentation()[0].Content())
	}
	if a.Attributes()[0].Value() != "keep" {
		t.Errorf("Attributes aliased constructor slice: got %q", a.Attributes()[0].Value())
	}
}

func TestAnnotationAccessorsDoNotAlias(t *testing.T) {
	a := xsd.NewAnnotation(
		[]xsd.AppInfo{xsd.NewAppInfo(nil, "keep")},
		[]xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")},
		[]xsd.Attr{xsd.NewAttr(xsd.QName{Local: "k"}, "keep")},
	)

	// Mutate the RETURNED slices.
	a.AppInfo()[0] = xsd.NewAppInfo(nil, "tampered")
	a.Documentation()[0] = xsd.NewDocumentation(nil, nil, "tampered")
	a.Attributes()[0] = xsd.NewAttr(xsd.QName{Local: "k"}, "tampered")

	if a.AppInfo()[0].Content() != "keep" {
		t.Errorf("AppInfo() returned an aliased slice: got %q", a.AppInfo()[0].Content())
	}
	if a.Documentation()[0].Content() != "keep" {
		t.Errorf("Documentation() returned an aliased slice: got %q", a.Documentation()[0].Content())
	}
	if a.Attributes()[0].Value() != "keep" {
		t.Errorf("Attributes() returned an aliased slice: got %q", a.Attributes()[0].Value())
	}
}
