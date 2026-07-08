package value_test

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// mapBackend is a test-only value.Backend over an explicit table, so tests can
// declare exactly which QNames a backend covers.
type mapBackend map[xsd.QName]value.Mapping

func (b mapBackend) Mapping(typ xsd.QName) (value.Mapping, bool) {
	m, ok := b[typ]
	return m, ok
}

func constMapping(tag string) value.Mapping {
	return value.Mapping{
		Parse: func(_ string, _ value.Context) (value.Value, error) {
			return tag, nil
		},
	}
}

func TestOverride(t *testing.T) {
	const xsdNS = "http://www.w3.org/2001/XMLSchema"
	decimal := xsd.QName{Space: xsdNS, Local: "decimal"}
	str := xsd.QName{Space: xsdNS, Local: "string"}
	boolean := xsd.QName{Space: xsdNS, Local: "boolean"}
	missing := xsd.QName{Space: xsdNS, Local: "date"}

	// base covers the rest; partial covers only decimal.
	base := mapBackend{
		decimal: constMapping("base-decimal"),
		str:     constMapping("base-string"),
		boolean: constMapping("base-boolean"),
	}
	partial := mapBackend{
		decimal: constMapping("partial-decimal"),
	}

	backend := value.Override(base, partial)

	tag := func(t *testing.T, typ xsd.QName) string {
		t.Helper()
		m, ok := backend.Mapping(typ)
		if !ok {
			t.Fatalf("Mapping(%s): ok = false, want true", typ)
		}
		v, err := m.Parse("", nil)
		if err != nil {
			t.Fatalf("Parse for %s: %v", typ, err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("Parse for %s: value %T, want string", typ, v)
		}
		return s
	}

	if got := tag(t, decimal); got != "partial-decimal" {
		t.Errorf("decimal: got %q, want partial's mapping", got)
	}
	if got := tag(t, str); got != "base-string" {
		t.Errorf("string: got %q, want base's mapping", got)
	}
	if got := tag(t, boolean); got != "base-boolean" {
		t.Errorf("boolean: got %q, want base's mapping", got)
	}
	if _, ok := backend.Mapping(missing); ok {
		t.Errorf("Mapping(%s): ok = true, want false (neither backend covers it)", missing)
	}
}
