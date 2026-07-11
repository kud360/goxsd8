package conformance

import (
	"bufio"
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file activates the datatypes lane (issue #15, extended by issue #57) by
// giving the datatypes entry of defaultLanes a real selector and executor. It
// touches nothing else in the runner (the #6 seam). It is package-internal
// conformance support: it exports nothing and no library code imports it.
//
// # The lexical cohort (issue #15)
//
// The lane claims the Microsoft datatype LEXICAL cases under
// msData/datatypes/{boolean,decimal,string}NNN.xml. Each such schema declares
// an element of an UNRESTRICTED builtin primitive (xsd:boolean / xsd:decimal /
// xsd:string — comp_foo directly, simpleTest via a facet-free restriction), so
// an instance is valid iff its content lies in that primitive's lexical space.
// That is exactly what value.Mapping.Parse decides, so the executor is a
// genuine, complete check: both polarities are decided for the right reason,
// and Parse really discriminates (boolean rejects "True"/"+1"/""; decimal
// rejects "1E2"/"INF"/"NaN"/"13.1513.561"/"ABCDEF").
//
// # The facet cohort (issue #57)
//
// The lane additionally claims the Microsoft *Facets* instance cases under
// msData/datatypes/Facets/{string,decimal}/<prim>_<facet>NNN.xml. Each such
// schema restricts a strict-mapped primitive (xsd:string or xsd:decimal) by one
// or more constraining facets (length/minLength/maxLength/pattern/enumeration on
// string; minInclusive/maxInclusive/minExclusive/maxExclusive/totalDigits/
// pattern/enumeration on decimal). Validity there depends on FACET checking, not
// just primitive lexical-space membership: an instance can be lexically valid
// yet facet-invalid (e.g. a 5-character string under length=4). The executor
// synthesizes the corresponding xsd.SimpleType (the seeded primitive as base,
// the schema's facet children as ownFacets) and decides validity through the
// now-complete facet pipeline (strict.ValidateLexical, issue #45) — pattern
// (cvc-pattern-valid §4.3.4.4), lexical mapping (cvc-datatype-valid §4.1.4),
// then the value facets cvc-enumeration-valid (§4.3.5.4),
// cvc-min/maxInclusive/Exclusive-valid (§4.3.7–4.3.10), cvc-totalDigits-valid
// (§4.3.11.3) and cvc-length/minLength/maxLength-valid (§4.3.1.3–4.3.3.3). This
// is the facet-invalid-but-lexically-valid class the original #15 landing could
// not discriminate with Parse alone.
//
// The executor OWNS facet applicability (cos-applicable-facets §4.1.5): it
// attaches a facet to the synthesized leaf only when builtin's applicable-facet
// metadata says it applies to the base primitive, so an instance-level facet
// violation always returns an *xsderr.Error through the normal path and the
// panic precondition ValidateLexical documents is never reached. A case pairing
// an inapplicable facet with a primitive (a schema-construction error, not an
// instance validity case) is declined rather than fed through and crashed.
//
// # Still deferred
//
// Facets over primitives strict.New() does not map (the int/integer/long/token/
// normalizedString/… dirs, whose narrower lexical spaces and own facets a
// decimal/string mapping would mis-decide), xsd:boolean facets (no Facets dir
// exists for it), the NIST corpus, and list/union varieties remain out of
// scope until their backends land. boolean018 (a list-of-boolean + enumeration
// on a user-defined "myList") resolves to a non-seeded type and is honestly
// recorded as a gap (Fail); it flips only when list variety is reachable here.

const xsdNS = "http://www.w3.org/2001/XMLSchema"

// synthNS namespaces the anonymous leaf types the facet cohort synthesizes. It
// is deliberately outside xsdNS so a synthesized leaf is never mistaken for a
// backend-mapped builtin (the widest-space facet checks resolve to its primitive
// base's mapping, never the leaf's own).
const synthNS = "urn:goxsd8:conformance:facets"

// datatypesCase matches an instance case in the lexical cohort.
var datatypesCase = regexp.MustCompile(`msData/datatypes/(boolean|decimal|string)[0-9]+\.xml$`)

// facetsCase matches an instance case in the facet cohort: an MS Facets instance
// restricting a strict-mapped primitive (string or decimal).
var facetsCase = regexp.MustCompile(`msData/datatypes/Facets/(string|decimal)/(string|decimal)_[A-Za-z]+[0-9]+\.xml$`)

// selectsDatatypes claims the instance cases of both cohorts. It is a cheap path
// predicate; the executor does the real document reading.
func selectsDatatypes(c caseSpec) bool {
	if c.kind != kindInstance {
		return false
	}
	doc := filepath.ToSlash(c.doc)
	return datatypesCase.MatchString(doc) || facetsCase.MatchString(doc)
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
	// claims decimal/boolean/string (lexical cohort) and string/decimal (facet
	// cohort) cases.
	strictBackend := strict.New()
	backend := value.Override(fallbackPrimitives{}, strictBackend)

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
		if facetsCase.MatchString(filepath.ToSlash(c.doc)) {
			return execFacetsCase(backend, strictBackend, sym, c)
		}
		return execLexicalCase(backend, sym, c)
	}
}

// execLexicalCase decides a lexical-cohort case: an instance is valid iff every
// tested leaf value lies in the tested primitive's lexical space (value.Parse).
func execLexicalCase(backend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
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

// execFacetsCase decides a facet-cohort case: it synthesizes the schema's
// faceted leaf type and runs the tested value through the real facet pipeline
// (strict.ValidateLexical). A case whose base is not strict-mapped, whose schema
// cannot be read, or that pairs an inapplicable facet with its primitive is
// declined (Fail, a recorded gap) rather than mis-decided or crashed.
func execFacetsCase(backend, strictBackend value.Backend, sym map[xsd.QName]*xsd.SimpleType, c caseSpec) Status {
	raw, base, children, ok := readFacetsCase(c.doc)
	if !ok {
		return Fail()
	}
	qn := xsd.QName{Space: xsdNS, Local: base}
	// Authoritative cohort guard: ask strict itself whether it maps base, so the
	// no-op fallback (which "maps" every primitive) can never route a non-strict
	// primitive through ValidateLexical and mis-decide it.
	if _, mapped := strictBackend.Mapping(qn); !mapped {
		return Fail()
	}
	prim, seeded := sym[qn]
	if !seeded {
		return Fail()
	}
	ownFacets, ok := buildOwnFacets(base, children)
	if !ok {
		return Fail()
	}
	leaf, err := xsd.NewSimpleType(xsderr.Loc{},
		xsd.QName{Space: synthNS, Local: base + "-facets"},
		xsd.Atomic{Primitive: prim}, prim, ownFacets, nil)
	if err != nil {
		return Fail()
	}
	_, verr := strict.ValidateLexical(backend, leaf, raw, nil)
	observedValid := verr == nil
	if observedValid == c.expectValid {
		return Pass()
	}
	return Fail()
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
// (their fixed whiteSpace facet), preserve for string. This is the lexical
// cohort's path only; the facet cohort normalizes inside strict.ValidateLexical.
func parseOK(m value.Mapping, prim, raw string) bool {
	_, err := m.Parse(normalizeWhiteSpace(prim, raw), nil)
	return err == nil
}

// normalizeWhiteSpace applies prim's whiteSpace facet (read from the generated
// builtin table) to raw. Used only by the lexical cohort (parseOK); the facet
// cohort's normalization lives in strict.ValidateLexical's whiteSpace stage, so
// there is exactly one normalization per path and no double-normalizing.
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

// readLexicalCase reads one lexical-cohort instance: it decodes the instance's
// leaf values (comp_foo and simpleTest) and the schema-under-test's tested
// primitive (from the instance's noNamespaceSchemaLocation). ok is false when
// either document cannot be read for this shape.
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

// lexicalInstance mirrors the lexical cohort's instance shape: a root carrying
// the same value in complexTest/comp_foo (the primitive directly) and simpleTest
// (a facet-free restriction of it).
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

// lexicalSchema mirrors the lexical cohort's schema shape: its simplefooType
// restricts the tested builtin primitive with no facets.
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
			return localName(base), nil
		}
	}
	return "", nil
}

// facetChild is one constraining-facet element read from a Facets-cohort schema:
// its element local name (e.g. "length") and its value attribute.
type facetChild struct {
	name  string
	value string
}

// facetKinds is the set of facet kinds the facet cohort recognizes: the value-
// and pattern-facet kinds strict.ValidateLexical decides for string/decimal.
// whiteSpace (normalization, no cvc-* rule), assertions and explicitTimezone are
// deliberately excluded, so a schema carrying one is declined rather than
// silently ignored.
var facetKinds = []xsd.FacetKind{
	xsd.FacetLength, xsd.FacetMinLength, xsd.FacetMaxLength,
	xsd.FacetPattern, xsd.FacetEnumeration,
	xsd.FacetMaxInclusive, xsd.FacetMaxExclusive,
	xsd.FacetMinExclusive, xsd.FacetMinInclusive,
	xsd.FacetTotalDigits, xsd.FacetFractionDigits,
}

// facetKindOf maps a facet element's local name to its xsd.FacetKind by matching
// the kind's spec token (never a hand-typed name table; the token is
// FacetKind.String's own output). ok is false for an unrecognized name.
func facetKindOf(name string) (xsd.FacetKind, bool) {
	for _, k := range facetKinds {
		if k.String() == name {
			return k, true
		}
	}
	return 0, false
}

// typeSpecOf returns the builtin TypeSpec for the primitive named name, carrying
// its applicable-facet metadata (cos-applicable-facets). ok is false if unknown.
func typeSpecOf(name string) (builtin.TypeSpec, bool) {
	for _, t := range builtin.Types {
		if t.Name == name {
			return t, true
		}
	}
	return builtin.TypeSpec{}, false
}

// buildOwnFacets translates the schema's facet children into the leaf's
// ownFacets, grouping same-kind children (pattern/enumeration carry a set of
// {value}s) into one facet in first-seen order (D2: the map is a lookup, output
// order comes from the order slice). It returns ok=false — declining the case —
// when a child names an unrecognized facet or a facet inapplicable to base
// (cos-applicable-facets §4.1.5), so the synthesized leaf never carries a facet
// that would trip ValidateLexical's panic precondition.
func buildOwnFacets(base string, children []facetChild) ([]xsd.Facet, bool) {
	spec, ok := typeSpecOf(base)
	if !ok {
		return nil, false
	}
	var order []xsd.FacetKind
	values := map[xsd.FacetKind][]string{}
	for _, ch := range children {
		kind, ok := facetKindOf(ch.name)
		if !ok {
			return nil, false
		}
		if !spec.Applies(builtin.FacetName(kind.String())) {
			return nil, false
		}
		if _, seen := values[kind]; !seen {
			order = append(order, kind)
		}
		values[kind] = append(values[kind], ch.value)
	}
	facets := make([]xsd.Facet, 0, len(order))
	for _, kind := range order {
		facets = append(facets, xsd.NewFacet(kind, values[kind], false))
	}
	return facets, true
}

// readFacetsCase reads one facet-cohort instance: the tested value (the <foo>
// leaf text, un-normalized — ValidateLexical's whiteSpace stage normalizes it)
// and, from the schema at the instance's noNamespaceSchemaLocation, the
// restriction's base primitive and facet children. ok is false when either
// document cannot be read for this shape.
func readFacetsCase(instancePath string) (raw, base string, children []facetChild, ok bool) {
	inst, err := decodeFacetsInstance(instancePath)
	if err != nil || inst.SchemaLoc == "" {
		return "", "", nil, false
	}
	schemaPath := filepath.Join(filepath.Dir(instancePath), filepath.FromSlash(inst.SchemaLoc))
	base, children, ok = decodeRestriction(schemaPath)
	if !ok || base == "" || len(children) == 0 {
		return "", "", nil, false
	}
	return inst.Foo, base, children, true
}

// facetsInstance mirrors the Facets cohort's instance shape: a <test> root whose
// single <foo> child holds the tested value.
type facetsInstance struct {
	SchemaLoc string `xml:"http://www.w3.org/2001/XMLSchema-instance noNamespaceSchemaLocation,attr"`
	Foo       string `xml:"foo"`
}

func decodeFacetsInstance(path string) (facetsInstance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return facetsInstance{}, err
	}
	var inst facetsInstance
	if err := xml.Unmarshal(data, &inst); err != nil {
		return facetsInstance{}, err
	}
	return inst, nil
}

// decodeRestriction streams the schema and returns the base primitive (prefix
// stripped) and the constraining-facet children of its first xsd:restriction.
// Facet children are the restriction's direct element children in the XML Schema
// namespace, in document order (P4: token stream, no whole-document buffer). ok
// is false when no restriction is found.
func decodeRestriction(path string) (base string, children []facetChild, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, false
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	dec := xml.NewDecoder(bufio.NewReader(f))
	inRestriction := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if end, isEnd := tok.(xml.EndElement); isEnd {
			if inRestriction && end.Name.Local == "restriction" && end.Name.Space == xsdNS {
				return base, children, true
			}
			continue
		}
		se, isStart := tok.(xml.StartElement)
		if !isStart {
			continue
		}
		if !inRestriction {
			if se.Name.Local == "restriction" && se.Name.Space == xsdNS {
				inRestriction = true
				base = localName(attrValue(se, "base"))
			}
			continue
		}
		if se.Name.Space == xsdNS {
			children = append(children, facetChild{name: se.Name.Local, value: attrValue(se, "value")})
		}
	}
	if inRestriction {
		return base, children, true
	}
	return "", nil, false
}

// attrValue returns the value of se's unqualified attribute local, or "".
func attrValue(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// localName strips a QName's prefix, returning its local part.
func localName(qn string) string {
	if i := strings.LastIndexByte(qn, ':'); i >= 0 {
		return qn[i+1:]
	}
	return qn
}
