package conformance

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file activates the datatypes lane (issue #15, the M3 deliverable) by
// giving the datatypes entry of defaultLanes a real selector and executor. It
// touches nothing else in the runner (the #6 seam). It is package-internal
// conformance support: it exports nothing and no library code imports it.
//
// # The cohort
//
// The lane claims exactly the Microsoft datatype LEXICAL cases under
// msData/datatypes/{boolean,decimal,string}NNN.xml. Each such schema declares
// an element of an UNRESTRICTED builtin primitive (xsd:boolean / xsd:decimal /
// xsd:string — comp_foo directly, simpleTest via a facet-free restriction), so
// an instance is valid iff its content lies in that primitive's lexical space.
// That is exactly what value.Mapping.Parse decides, so the executor is a
// genuine, complete check: both polarities are decided for the right reason,
// and Parse really discriminates (boolean rejects "True"/"+1"/""; decimal
// rejects "1E2"/"INF"/"NaN"/"13.1513.561"/"ABCDEF").
//
// Other datatype corpora (NIST, the MS Facets sets) were investigated and
// rejected for this first lane: they restrict a primitive by a FACET, and every
// facet-invalid instance still carries a lexically-valid value, so a Parse-only
// executor could not discriminate them — a facet engine is a later milestone.
// Per issue #15: a small, genuine, moving lane beats broad-but-fake coverage.
//
// One numbered instance, boolean018, is a list-of-boolean with enumeration
// facets (its own boolean018.xsd), not the unrestricted-primitive shape. The
// executor makes no special case for it: its tested type resolves to the
// user-defined "myList", which is not a seeded builtin, so the executor
// honestly records it as a gap (Fail) rather than pretend to validate a list.
// That real fail, alongside the genuine lexical rejections, is what proves the
// executor is not a fake-always-pass check.

const xsdNS = "http://www.w3.org/2001/XMLSchema"

// datatypesCase matches an instance case in the claimed lexical cohort.
var datatypesCase = regexp.MustCompile(`msData/datatypes/(boolean|decimal|string)[0-9]+\.xml$`)

// selectsDatatypes claims the instance cases of the lexical cohort. It is a
// cheap path predicate; the executor does the real document reading.
func selectsDatatypes(c caseSpec) bool {
	return c.kind == kindInstance && datatypesCase.MatchString(filepath.ToSlash(c.doc))
}

// newDatatypesExec builds the lane's executor: it composes builtin/strict with
// a trivial fallback so builtin.Seed's all-primitives precondition is met,
// Seeds the builtins once (the M3 composition step), and captures the composed
// backend plus the seeded symbol table in the returned closure.
func newDatatypesExec() executor {
	// strict.New() maps only decimal/boolean/string; Seed requires all 20
	// primitives, so the fallback covers the other 17 with a no-op mapping.
	// strict wins where it maps (Override yields partial first), so those
	// fallback mappings are never actually exercised — the lane's selector only
	// claims decimal/boolean/string cases.
	backend := value.Override(fallbackPrimitives{}, strict.New())

	// Seed proves the composed backend satisfies the precondition and yields
	// the builtin components; the executor confirms a claimed case's type is a
	// seeded builtin before validating it. The composed backend is complete by
	// construction (every primitive covered by the fallback, guarded by
	// TestDatatypesBackendSeeds), so a Seed error here is a programming error,
	// not a runtime condition — panic rather than drop it.
	types, err := builtin.Seed(backend)
	if err != nil {
		panic("conformance: datatypes lane backend must Seed by construction: " + err.Error())
	}
	sym := make(map[xsd.QName]*xsd.SimpleType, len(types))
	for _, t := range types {
		sym[t.Name()] = t
	}

	return func(c caseSpec) Status {
		prim, values, ok := readLexicalCase(c.doc)
		if !ok {
			return Fail()
		}
		qn := xsd.QName{Space: xsdNS, Local: prim}
		if _, seeded := sym[qn]; !seeded {
			return Fail()
		}
		m, mapped := backend.Mapping(qn)
		if !mapped {
			return Fail()
		}
		observedValid := true
		for _, v := range values {
			if !parseOK(m, prim, v) {
				observedValid = false
				break
			}
		}
		if observedValid == c.expectValid {
			return Pass()
		}
		return Fail()
	}
}

// fallbackPrimitives maps every builtin primitive with a no-op identity mapping.
// It exists ONLY to satisfy builtin.Seed's all-primitives precondition for the
// 17 primitives strict.New() does not cover; the datatypes selector never
// claims a case that would exercise these mappings.
type fallbackPrimitives struct{}

func (fallbackPrimitives) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ.Space != xsdNS {
		return value.Mapping{}, false
	}
	for _, t := range builtin.Types {
		if t.IsPrimitive() && t.Name == typ.Local {
			return value.Mapping{
				Parse: func(lexical string, _ value.Context) (value.Value, error) { return lexical, nil },
			}, true
		}
	}
	return value.Mapping{}, false
}

// parseOK reports whether raw is in prim's lexical space, after applying prim's
// whiteSpace normalization (Datatypes §4.3.6) — collapse for boolean/decimal
// (their fixed whiteSpace facet), preserve for string.
func parseOK(m value.Mapping, prim, raw string) bool {
	_, err := m.Parse(normalizeWhiteSpace(prim, raw), nil)
	return err == nil
}

// normalizeWhiteSpace applies prim's whiteSpace facet (read from the generated
// builtin table) to raw.
func normalizeWhiteSpace(prim, raw string) string {
	switch whiteSpaceOf(prim) {
	case "collapse":
		return strings.Join(strings.Fields(raw), " ")
	case "replace":
		return strings.Map(func(r rune) rune {
			if r == '\t' || r == '\n' || r == '\r' {
				return ' '
			}
			return r
		}, raw)
	default: // preserve
		return raw
	}
}

// whiteSpaceOf returns the spec whiteSpace value for a primitive, from the
// generated builtin table (never hand-typed); "" if the primitive is unknown.
func whiteSpaceOf(prim string) string {
	for _, t := range builtin.Types {
		if t.Name != prim {
			continue
		}
		for _, f := range t.Facets {
			if f.Name == "whiteSpace" {
				return f.Default
			}
		}
	}
	return ""
}

// readLexicalCase reads one cohort instance: it decodes the instance's leaf
// values (comp_foo and simpleTest) and the schema-under-test's tested primitive
// (from the instance's noNamespaceSchemaLocation). ok is false when either
// document cannot be read for this shape.
func readLexicalCase(instancePath string) (prim string, values []string, ok bool) {
	inst, err := decodeLexicalInstance(instancePath)
	if err != nil {
		return "", nil, false
	}
	if inst.SchemaLoc == "" {
		return "", nil, false
	}
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(inst.SchemaLoc))
	prim, err = decodeTestedPrimitive(schemaPath)
	if err != nil || prim == "" {
		return "", nil, false
	}
	return prim, []string{inst.ComplexTest.CompFoo, inst.SimpleTest}, true
}

// lexicalInstance mirrors the cohort's instance shape: a root carrying the same
// value in complexTest/comp_foo (the primitive directly) and simpleTest (a
// facet-free restriction of it).
type lexicalInstance struct {
	SchemaLoc   string `xml:"http://www.w3.org/2001/XMLSchema-instance noNamespaceSchemaLocation,attr"`
	ComplexTest struct {
		CompFoo string `xml:"comp_foo"`
	} `xml:"complexTest"`
	SimpleTest string `xml:"simpleTest"`
}

func decodeLexicalInstance(path string) (lexicalInstance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return lexicalInstance{}, err
	}
	var inst lexicalInstance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return lexicalInstance{}, err
	}
	return inst, nil
}

// lexicalSchema mirrors the cohort's schema shape: its simplefooType restricts
// the tested builtin primitive with no facets.
type lexicalSchema struct {
	SimpleTypes []struct {
		Restriction struct {
			Base string `xml:"base,attr"`
		} `xml:"restriction"`
	} `xml:"simpleType"`
}

// decodeTestedPrimitive returns the local name of the primitive the schema
// tests (the restriction base of its first simpleType, prefix stripped).
func decodeTestedPrimitive(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var s lexicalSchema
	if err := xml.Unmarshal(data, &s); err != nil {
		return "", err
	}
	for _, st := range s.SimpleTypes {
		if base := st.Restriction.Base; base != "" {
			if i := strings.LastIndexByte(base, ':'); i >= 0 {
				return base[i+1:], nil
			}
			return base, nil
		}
	}
	return "", nil
}
