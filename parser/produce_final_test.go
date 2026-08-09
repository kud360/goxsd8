package parser_test

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// finalSchema wraps body in a <schema> carrying the given <schema>-level default
// attributes (a leading space and the attribute text, or "").
func finalSchema(defaults, body string) string {
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + defaults + `>` + body + `</xs:schema>`
}

// producedSimpleType produces doc and returns the named top-level simple type.
func producedSimpleType(t *testing.T, doc, local string) *xsd.SimpleType {
	t.Helper()
	s, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	def, ok := s.Type(xsd.QName{Local: local})
	if !ok {
		t.Fatalf("simple type %s not found", local)
	}
	st, ok := def.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("type %s is %T, want *xsd.SimpleType", local, def)
	}
	return st
}

// producedComplexType produces doc and returns the named top-level complex type.
func producedComplexType(t *testing.T, doc, local string) xsd.ComplexType {
	t.Helper()
	s, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	def, ok := s.Type(xsd.QName{Local: local})
	if !ok {
		t.Fatalf("complex type %s not found", local)
	}
	ct, ok := def.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is %T, want xsd.ComplexType", local, def)
	}
	return ct
}

// TestProduceSimpleTypeFinalMapped pins §3.16.2.1 map.std.common's {final} row:
// ·FS· is final= if present, else the <schema>'s finalDefault, else the empty
// string; the empty string is the empty set, "#all" is the FOUR-keyword set
// {restriction, extension, list, union}, and any other value names the keywords
// its list contains. The membership assertions are direct, so a copy of the
// element-side three-keyword expansion would fail here rather than pass by
// accident.
func TestProduceSimpleTypeFinalMapped(t *testing.T) {
	all := []xsd.DerivationMethod{xsd.DerivationRestriction, xsd.DerivationExtension, xsd.DerivationList, xsd.DerivationUnion}
	for _, tc := range []struct {
		name         string
		finalDefault string
		final        string
		want         []xsd.DerivationMethod
	}{
		{name: "absent is the empty set"},
		{name: "empty final is the empty set", final: ` final=""`},
		{name: "#all is all four keywords", final: ` final="#all"`, want: all},
		{name: "one keyword", final: ` final="union"`,
			want: []xsd.DerivationMethod{xsd.DerivationUnion}},
		{name: "canonical order not lexical", final: ` final="union list"`,
			want: []xsd.DerivationMethod{xsd.DerivationList, xsd.DerivationUnion}},
		// substitution is a blockDefault keyword, never a finalDefault one, so it
		// names no member of this set and is dropped like any other stray item.
		{name: "unrecognized items ignored", final: ` final="substitution list"`,
			want: []xsd.DerivationMethod{xsd.DerivationList}},
		{name: "finalDefault fallback", finalDefault: ` finalDefault="union"`,
			want: []xsd.DerivationMethod{xsd.DerivationUnion}},
		{name: "finalDefault #all is all four keywords", finalDefault: ` finalDefault="#all"`, want: all},
		{name: "final overrides finalDefault", finalDefault: ` finalDefault="#all"`, final: ` final="list"`,
			want: []xsd.DerivationMethod{xsd.DerivationList}},
		// An EMPTY final= is PRESENT, so it is the ·FS· and takes case 1; the walk
		// to finalDefault never happens. This is the case a ""-means-absent reading
		// would lose.
		{name: "empty final overrides finalDefault", finalDefault: ` finalDefault="#all"`, final: ` final=""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := finalSchema(tc.finalDefault,
				`<xs:simpleType name="st"`+tc.final+`><xs:restriction base="xs:string"/></xs:simpleType>`)
			if got := producedSimpleType(t, doc, "st").Final(); !slices.Equal(got, tc.want) {
				t.Fatalf("simple type {final} = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProduceComplexTypeFinalMapped pins §3.4.2.1 dcl.ctd.common's {final} row —
// "[a]s for {prohibited substitutions} above, but using the final and
// finalDefault attributes" — whose keyword set is the TWO-member {extension,
// restriction}, not the simple type's four and not the element's three. The
// "#all" and finalDefault cases assert the exact set, which is what distinguishes
// this mapping from a copy of either sibling.
func TestProduceComplexTypeFinalMapped(t *testing.T) {
	both := []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction}
	for _, tc := range []struct {
		name         string
		finalDefault string
		final        string
		want         []xsd.DerivationMethod
	}{
		{name: "absent is the empty set"},
		{name: "empty final is the empty set", final: ` final=""`},
		{name: "#all is exactly extension and restriction", final: ` final="#all"`, want: both},
		{name: "one keyword", final: ` final="extension"`,
			want: []xsd.DerivationMethod{xsd.DerivationExtension}},
		{name: "canonical order not lexical", final: ` final="restriction extension"`, want: both},
		// list and union are finalDefault keywords a complex type's {final} does not
		// consume; they are ignored, never rejected.
		{name: "list and union ignored", final: ` final="list union restriction"`,
			want: []xsd.DerivationMethod{xsd.DerivationRestriction}},
		{name: "finalDefault fallback", finalDefault: ` finalDefault="extension"`,
			want: []xsd.DerivationMethod{xsd.DerivationExtension}},
		// finalDefault's own vocabulary is the WIDER four-keyword one, so its "#all"
		// must still narrow to this property's two members here.
		{name: "finalDefault #all narrows to two", finalDefault: ` finalDefault="#all"`, want: both},
		{name: "finalDefault list and union ignored", finalDefault: ` finalDefault="list union"`},
		{name: "final overrides finalDefault", finalDefault: ` finalDefault="#all"`, final: ` final="extension"`,
			want: []xsd.DerivationMethod{xsd.DerivationExtension}},
		{name: "empty final overrides finalDefault", finalDefault: ` finalDefault="#all"`, final: ` final=""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := finalSchema(tc.finalDefault,
				`<xs:complexType name="ct"`+tc.final+`><xs:sequence/></xs:complexType>`)
			if got := producedComplexType(t, doc, "ct").Final(); !slices.Equal(got, tc.want) {
				t.Fatalf("complex type {final} = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProduceComplexTypeBlockMapped pins §3.4.2.1 dcl.ctd.common's {prohibited
// substitutions} row. Its ·EBV· is a <complexType>'s block=, else the <schema>'s
// blockDefault, and its keyword set is {extension, restriction} — the same two
// the complex type's {final} draws from, and NOT the three an <element>'s
// {disallowed substitutions} draws from. blockDefault's grammar admits
// substitution, and the row's own Note says values outside this property's set
// "are ignored in the determination of {prohibited substitutions} for complex
// type definitions (they are used elsewhere)".
func TestProduceComplexTypeBlockMapped(t *testing.T) {
	both := []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction}
	for _, tc := range []struct {
		name         string
		blockDefault string
		block        string
		want         []xsd.DerivationMethod
	}{
		{name: "absent is the empty set"},
		{name: "empty block is the empty set", block: ` block=""`},
		{name: "#all is exactly extension and restriction", block: ` block="#all"`, want: both},
		{name: "one keyword", block: ` block="restriction"`,
			want: []xsd.DerivationMethod{xsd.DerivationRestriction}},
		{name: "canonical order not lexical", block: ` block="restriction extension"`, want: both},
		{name: "substitution ignored", block: ` block="substitution extension"`,
			want: []xsd.DerivationMethod{xsd.DerivationExtension}},
		{name: "blockDefault fallback", blockDefault: ` blockDefault="restriction"`,
			want: []xsd.DerivationMethod{xsd.DerivationRestriction}},
		// The element-side expansion of blockDefault="#all" has three members; this
		// property's has two, and the difference is exactly what this case pins.
		{name: "blockDefault #all narrows to two", blockDefault: ` blockDefault="#all"`, want: both},
		{name: "blockDefault substitution ignored", blockDefault: ` blockDefault="substitution"`},
		{name: "block overrides blockDefault", blockDefault: ` blockDefault="#all"`, block: ` block="restriction"`,
			want: []xsd.DerivationMethod{xsd.DerivationRestriction}},
		{name: "empty block overrides blockDefault", blockDefault: ` blockDefault="#all"`, block: ` block=""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := finalSchema(tc.blockDefault,
				`<xs:complexType name="ct"`+tc.block+`><xs:sequence/></xs:complexType>`)
			if got := producedComplexType(t, doc, "ct").ProhibitedSubstitutions(); !slices.Equal(got, tc.want) {
				t.Fatalf("complex type {prohibited substitutions} = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProduceFinalOrderingIsDeterministic is STYLE D2 stated as a test: these
// properties are SETS, so the two spellings of one set must build the identical
// component. A mapping that echoed the attribute's lexical order would pass every
// membership assertion above and still fail here.
func TestProduceFinalOrderingIsDeterministic(t *testing.T) {
	simple := func(spelling string) []xsd.DerivationMethod {
		doc := finalSchema("", `<xs:simpleType name="st" final="`+spelling+`"><xs:restriction base="xs:string"/></xs:simpleType>`)
		return producedSimpleType(t, doc, "st").Final()
	}
	if a, b := simple("list union"), simple("union list"); !slices.Equal(a, b) {
		t.Fatalf(`simple type {final}: final="list union" = %v, final="union list" = %v`, a, b)
	}
	complexFinal := func(spelling string) []xsd.DerivationMethod {
		doc := finalSchema("", `<xs:complexType name="ct" final="`+spelling+`"><xs:sequence/></xs:complexType>`)
		return producedComplexType(t, doc, "ct").Final()
	}
	if a, b := complexFinal("extension restriction"), complexFinal("restriction extension"); !slices.Equal(a, b) {
		t.Fatalf(`complex type {final}: extension-first = %v, restriction-first = %v`, a, b)
	}
	complexBlock := func(spelling string) []xsd.DerivationMethod {
		doc := finalSchema("", `<xs:complexType name="ct" block="`+spelling+`"><xs:sequence/></xs:complexType>`)
		return producedComplexType(t, doc, "ct").ProhibitedSubstitutions()
	}
	if a, b := complexBlock("extension restriction"), complexBlock("restriction extension"); !slices.Equal(a, b) {
		t.Fatalf(`complex type {prohibited substitutions}: extension-first = %v, restriction-first = %v`, a, b)
	}
}

// TestProduceSimpleTypeFinalBlocksRestriction is st-props-correct clause 3
// (§3.16.6.1) reached END TO END from a parsed schema for the first time: the
// check has always been written and tested in package xsd, but with every
// producer passing a nil {final} it could not fire on any document. The control
// is the same schema without final=, which must still be accepted.
func TestProduceSimpleTypeFinalBlocksRestriction(t *testing.T) {
	schema := func(final string) string {
		return finalSchema("",
			`<xs:simpleType name="B"`+final+`><xs:restriction base="xs:string"/></xs:simpleType>`+
				`<xs:simpleType name="D"><xs:restriction base="B"><xs:maxLength value="4"/></xs:restriction></xs:simpleType>`)
	}
	_, err := produce(t, schema(` final="restriction"`))
	assertRule(t, err, "st-props-correct")
	if _, err := produce(t, schema("")); err != nil {
		t.Fatalf("the same derivation with no final= on the base must be accepted: %v", err)
	}
}

// TestProduceSimpleTypeFinalDefaultBlocksRestriction is the same clause reached
// through the FALLBACK half of the ·FS· rule: no local final= anywhere, only the
// <schema>'s finalDefault, which every simple type in the document inherits.
func TestProduceSimpleTypeFinalDefaultBlocksRestriction(t *testing.T) {
	body := `<xs:simpleType name="B"><xs:restriction base="xs:string"/></xs:simpleType>` +
		`<xs:simpleType name="D"><xs:restriction base="B"><xs:maxLength value="4"/></xs:restriction></xs:simpleType>`
	_, err := produce(t, finalSchema(` finalDefault="restriction"`, body))
	assertRule(t, err, "st-props-correct")
	if _, err := produce(t, finalSchema("", body)); err != nil {
		t.Fatalf("the same document without finalDefault must be accepted: %v", err)
	}
}

// TestProduceComplexTypeFinalBlocksExtension is cos-ct-extends clause 1.1
// (§3.4.6.2) — "B.{final} does not contain extension" for a COMPLEX base —
// reached end to end. Until {final} was mapped this clause passed vacuously on
// every parsed schema; conformance/schema.go recorded it as data-dead.
func TestProduceComplexTypeFinalBlocksExtension(t *testing.T) {
	schema := func(final string) string {
		return finalSchema("",
			`<xs:complexType name="B"`+final+`><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="D"><xs:complexContent><xs:extension base="B">`+
				`<xs:sequence><xs:element name="y" type="xs:string"/></xs:sequence>`+
				`</xs:extension></xs:complexContent></xs:complexType>`)
	}
	_, err := produce(t, schema(` final="extension"`))
	assertRule(t, err, "cos-ct-extends")
	if _, err := produce(t, schema("")); err != nil {
		t.Fatalf("the same extension with no final= on the base must be accepted: %v", err)
	}
}

// TestProduceSimpleBaseFinalBlocksSimpleContentExtension is cos-ct-extends
// clause 2.2, the SIMPLE-base twin of clause 1.1: a <simpleContent> <extension>
// whose base is a simple type carrying final="extension". The base's {final}
// comes from the simple-type mapping and the reader is the case-2 arm, so this
// is a different site from the test above, not a re-run of it.
func TestProduceSimpleBaseFinalBlocksSimpleContentExtension(t *testing.T) {
	schema := func(final string) string {
		return finalSchema("",
			`<xs:simpleType name="B"`+final+`><xs:restriction base="xs:string"/></xs:simpleType>`+
				`<xs:complexType name="D"><xs:simpleContent><xs:extension base="B">`+
				`<xs:attribute name="a" type="xs:string"/>`+
				`</xs:extension></xs:simpleContent></xs:complexType>`)
	}
	_, err := produce(t, schema(` final="extension"`))
	assertRule(t, err, "cos-ct-extends")
	if _, err := produce(t, schema("")); err != nil {
		t.Fatalf("the same extension with no final= on the simple base must be accepted: %v", err)
	}
}

// TestProduceComplexTypeFinalBlocksRestriction is derivation-ok-restriction
// clause 1 (§3.4.6.3) — "the {final} of B does not contain restriction" —
// reached end to end.
func TestProduceComplexTypeFinalBlocksRestriction(t *testing.T) {
	schema := func(final string) string {
		return finalSchema("",
			`<xs:complexType name="B"`+final+`><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="D"><xs:complexContent><xs:restriction base="B">`+
				`<xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>`+
				`</xs:restriction></xs:complexContent></xs:complexType>`)
	}
	_, err := produce(t, schema(` final="restriction"`))
	assertRule(t, err, "derivation-ok-restriction")
	if _, err := produce(t, schema("")); err != nil {
		t.Fatalf("the same restriction with no final= on the base must be accepted: %v", err)
	}
}

// TestProduceComplexTypeBlockNarrowsSubstitutionGroup is cos-equiv-derived-ok-rec
// clause 2.2 (§3.3.6.3): the blocking union a head contributes to is its element
// declaration's {disallowed substitutions} UNIONED with its TYPE's {prohibited
// substitutions}, and only the first half of that union had a producer before
// this mapping. The head element carries no block= at all here — the whole effect
// comes from the type.
//
// It is the type-side shape of TestProduceElementBlockSubstitutionNarrowsGroup:
// with extension blocked on the head's type, the member is in no ·substitution
// group· of the head, so the two names do not ·overlap· and cos-nonambig has
// nothing to charge; drop the block and they do.
func TestProduceComplexTypeBlockNarrowsSubstitutionGroup(t *testing.T) {
	schema := func(block string) string {
		return finalSchema("",
			`<xs:complexType name="H"`+block+`><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="M"><xs:complexContent><xs:extension base="H">`+
				`<xs:sequence><xs:element name="y" type="xs:string"/></xs:sequence>`+
				`</xs:extension></xs:complexContent></xs:complexType>`+
				`<xs:element name="head" type="H"/>`+
				`<xs:element name="member" type="M" substitutionGroup="head"/>`+
				// minOccurs="0" on the first makes both particles live at the start
				// state, so the two ·compete· exactly when they ·overlap·.
				`<xs:complexType name="CT"><xs:sequence>`+
				`<xs:element ref="member" minOccurs="0"/><xs:element ref="head"/>`+
				`</xs:sequence></xs:complexType>`)
	}
	if _, err := produce(t, schema(` block="extension"`)); err != nil {
		t.Fatalf("the head's type blocks extension, so member is in no group of head and the two do not ·overlap·: %v", err)
	}
	_, err := produce(t, schema(""))
	assertRule(t, err, "cos-nonambig")
}

// TestProduceComplexTypeBlockBlocksLocalTypeSubstitution is cos-ct-derived-ok
// (§3.4.6.5) reached through ·validly substitutable· (§3.4.6.4 key-val-sub-type
// case 1), the one arm that unions the SUPER type's {prohibited substitutions}
// into the blocking set. Restricting a base whose element e is typed P by an
// element e typed Q asks exactly that question of the pair, and with restriction
// blocked on P the answer flips.
//
// This is the third distinct reader of {prohibited substitutions} and the only
// one that reaches it through the type-substitutability apparatus rather than
// through a substitution group.
func TestProduceComplexTypeBlockBlocksLocalTypeSubstitution(t *testing.T) {
	schema := func(block string) string {
		return finalSchema("",
			`<xs:complexType name="P"`+block+`><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="Q"><xs:complexContent><xs:restriction base="P">`+
				`<xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>`+
				`</xs:restriction></xs:complexContent></xs:complexType>`+
				`<xs:complexType name="B"><xs:sequence><xs:element name="e" type="P"/></xs:sequence></xs:complexType>`+
				`<xs:complexType name="D"><xs:complexContent><xs:restriction base="B">`+
				`<xs:sequence><xs:element name="e" type="Q"/></xs:sequence>`+
				`</xs:restriction></xs:complexContent></xs:complexType>`)
	}
	_, err := produce(t, schema(` block="restriction"`))
	assertRule(t, err, "derivation-ok-restriction")
	if _, err := produce(t, schema("")); err != nil {
		t.Fatalf("the same restriction with no block= on the locally declared type must be accepted: %v", err)
	}
}
