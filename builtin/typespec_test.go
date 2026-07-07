package builtin

import "testing"

// byName indexes Types for lookups; the table is document-ordered, so a map
// here (test-only, never rendered to output) is fine.
func byName(t *testing.T) map[string]TypeSpec {
	t.Helper()
	m := make(map[string]TypeSpec, len(Types))
	for _, ts := range Types {
		if _, dup := m[ts.Name]; dup {
			t.Fatalf("duplicate type %q in Types", ts.Name)
		}
		m[ts.Name] = ts
	}
	return m
}

func TestRowCount(t *testing.T) {
	if len(Types) != 49 {
		t.Fatalf("len(Types) = %d, want 49 (19 primitives + 28 ordinary + anyAtomicType + precisionDecimal)", len(Types))
	}
}

// classicPrimitives are the 19 §3.3 primitive datatypes, in spec order. This
// list is the independent cross-check; the generated table must not hardcode
// it.
var classicPrimitives = []string{
	"string", "boolean", "decimal", "float", "double", "duration", "dateTime",
	"time", "date", "gYearMonth", "gYear", "gMonthDay", "gDay", "gMonth",
	"hexBinary", "base64Binary", "anyURI", "QName", "NOTATION",
}

func TestPrimitiveSet(t *testing.T) {
	m := byName(t)

	// The 19 §3.3 primitives each derive from anyAtomicType (§4.1.6 dummy-def)
	// and report IsPrimitive. anyAtomicType is their base, so the base is a
	// shared ur-type, never another primitive.
	for _, name := range classicPrimitives {
		ts, ok := m[name]
		if !ok {
			t.Errorf("primitive %q missing from Types", name)
			continue
		}
		if ts.Base != "anyAtomicType" {
			t.Errorf("%s.Base = %q, want anyAtomicType", name, ts.Base)
		}
		if !ts.IsPrimitive() {
			t.Errorf("%s.IsPrimitive() = false, want true", name)
		}
	}

	// precisionDecimal is the 20th primitive (an implementation-defined
	// primitive, base anyAtomicType per §4.1.6); every other row is
	// non-primitive.
	var primitives []string
	for _, ts := range Types {
		if ts.IsPrimitive() {
			primitives = append(primitives, ts.Name)
		}
	}
	if len(primitives) != 20 {
		t.Fatalf("IsPrimitive() count = %d %v, want 20 (19 §3.3 primitives + precisionDecimal)", len(primitives), primitives)
	}
	if pd := m["precisionDecimal"]; !pd.IsPrimitive() || pd.Base != "anyAtomicType" {
		t.Errorf("precisionDecimal: IsPrimitive=%v Base=%q, want true / anyAtomicType", pd.IsPrimitive(), pd.Base)
	}
}

func TestAnyAtomicType(t *testing.T) {
	ts, ok := byName(t)["anyAtomicType"]
	if !ok {
		t.Fatal("anyAtomicType missing from Types")
	}
	// §4.1.6 anyAtomicType-def: base anySimpleType, empty {facets} and empty
	// {fundamental facets}.
	if ts.Base != "anySimpleType" {
		t.Errorf("anyAtomicType.Base = %q, want anySimpleType", ts.Base)
	}
	if ts.IsPrimitive() {
		t.Error("anyAtomicType.IsPrimitive() = true, want false")
	}
	if len(ts.Facets) != 0 {
		t.Errorf("anyAtomicType has %d facets, want 0", len(ts.Facets))
	}
	// The empty {fundamental facets} is a nil *Fundamental, not a partial mix.
	if ts.Fundamental != nil {
		t.Errorf("anyAtomicType.Fundamental = %+v, want nil (empty fundamental facets)", ts.Fundamental)
	}
}

func TestListVariety(t *testing.T) {
	wantList := map[string]string{"NMTOKENS": "NMTOKEN", "IDREFS": "IDREF", "ENTITIES": "ENTITY"}
	for _, ts := range Types {
		l, isList := ts.Variety.(List)
		wantItem, wantIsList := wantList[ts.Name]
		if isList != wantIsList {
			t.Errorf("%s is List is %v, want %v", ts.Name, isList, wantIsList)
		}
		if !wantIsList {
			continue
		}
		if l.Item != wantItem {
			t.Errorf("%s List.Item = %q, want %q", ts.Name, l.Item, wantItem)
		}
	}
}

func TestPrecisionDecimalFacets(t *testing.T) {
	pd, ok := byName(t)["precisionDecimal"]
	if !ok {
		t.Fatal("precisionDecimal missing from Types")
	}
	// precisionDecimal.md §3.3 / §4: precision is controlled by totalDigits,
	// maxScale, minScale — NOT fractionDigits, and the length family does not
	// apply (cos-applicable-facets generalized per §3.3).
	for _, f := range []FacetName{"totalDigits", "maxScale", "minScale"} {
		if !pd.Applies(f) {
			t.Errorf("precisionDecimal should apply facet %q", f)
		}
	}
	for _, f := range []FacetName{"fractionDigits", "length", "minLength", "maxLength"} {
		if pd.Applies(f) {
			t.Errorf("precisionDecimal must not apply facet %q", f)
		}
	}
	if _, ok := pd.Variety.(Atomic); !ok {
		t.Errorf("precisionDecimal.Variety = %T, want Atomic", pd.Variety)
	}
}
