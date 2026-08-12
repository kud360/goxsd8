package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// cos-all-limited (§3.8.6.2) is decided over the resolved {term}/{particles}
// graph, so these tests are driven from schema documents and read the verdict
// off Produce, whose finalize step charges it (xsd/allgrouplimited.go, #469).
// The <group ref> cases are the point: §3.7.2 xr.mgd3 makes a <group ref>
// particle's {term} the referenced definition's {model group}, so nothing in the
// XML at the usage site names a compositor at all.
//
// MUTATION CHECK. Removing the checkAllGroupsLimited call from Schema.resolve
// turns every want-a-rejection case here red and leaves every legal case green;
// no other check reaches these shapes (the producer's own charge, which covered
// the literal nesting alone, was deleted with this change).

// allLimitedDoc names one schema-document body and the verdict it must draw.
type allLimitedDoc struct {
	name string
	body string
	want xsderr.Rule // empty: the document must be accepted
}

// allGroupDef is a top-level all-bodied model group definition — clause 1.1's
// legal home, and the component every <group ref> below resolves its {term} to.
const allGroupDef = `<xs:group name="AllG"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>`

// seqGroupDef is a top-level sequence-bodied model group definition, the clause
// 2 counterexample when it is referenced from inside an <all>.
const seqGroupDef = `<xs:group name="SeqG"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`

func TestProduceAllGroupLimitedThroughGroupRef(t *testing.T) {
	for _, tc := range []allLimitedDoc{{
		// Clause 1: the <all> reached through the reference is the {term} of a
		// particle among a <sequence>'s {particles}, which no sub-clause names.
		name: "clause 1 all-bodied group ref inside a sequence",
		body: allGroupDef + `<xs:complexType name="CT"><xs:sequence><xs:group ref="AllG"/></xs:sequence></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		name: "clause 1 all-bodied group ref inside a choice",
		body: allGroupDef + `<xs:complexType name="CT"><xs:choice><xs:group ref="AllG"/></xs:choice></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		// Clause 1 again, one level down, through a group DEFINITION that no
		// <group ref> points at: §3.8.6 binds every Model Group component.
		name: "clause 1 all-bodied group ref inside an unreferenced definition",
		body: allGroupDef + `<xs:group name="Outer"><xs:sequence><xs:group ref="AllG"/></xs:sequence></xs:group>`,
		want: "cos-all-limited",
	}, {
		// Clause 2: the member's {term} is a sequence group, and every model
		// group among an all group's {particles} must have {compositor} all.
		name: "clause 2 sequence-bodied group ref inside an all",
		body: seqGroupDef + `<xs:complexType name="CT"><xs:all><xs:group ref="SeqG"/></xs:all></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		name: "clause 2 choice-bodied group ref inside an all",
		body: `<xs:group name="ChG"><xs:choice><xs:element name="b" type="xs:string"/></xs:choice></xs:group>` +
			`<xs:complexType name="CT"><xs:all><xs:group ref="ChG"/></xs:all></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		// Clause 1.2 admits an all group at a content particle only with {max
		// occurs} = 1. The schema for schema documents restricts <all>'s own
		// maxOccurs to {0,1}, but says nothing about a <group ref>'s.
		name: "clause 1.2 all-bodied group ref as a repeatable content particle",
		body: allGroupDef + `<xs:complexType name="CT"><xs:group ref="AllG" maxOccurs="unbounded"/></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		// Clause 1.3 admits a nested all group only at {min occurs} = {max
		// occurs} = 1.
		name: "clause 1.3 optional all-bodied group ref inside an all",
		body: allGroupDef + `<xs:complexType name="CT"><xs:all><xs:group ref="AllG" minOccurs="0"/></xs:all></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		// Clause 1.1: a definition is a legal home for an all group in itself,
		// and stays so with nothing referring to it.
		name: "legal 1.1 an all-bodied definition on its own",
		body: allGroupDef,
	}, {
		// Clause 1.2: the reference is the complex type's content particle, and
		// maxOccurs defaults to 1.
		name: "legal 1.2 all-bodied group ref as the content particle",
		body: allGroupDef + `<xs:complexType name="CT"><xs:group ref="AllG"/></xs:complexType>`,
	}, {
		name: "legal 1.2 optional all-bodied group ref as the content particle",
		body: allGroupDef + `<xs:complexType name="CT"><xs:group ref="AllG" minOccurs="0"/></xs:complexType>`,
	}, {
		// Clause 1.3: nested inside another all group at min = max = 1, which is
		// what Appendix A's xs:allModel fixes a <group ref> child of <all> to.
		name: "legal 1.3 all-bodied group ref inside an all",
		body: allGroupDef + `<xs:complexType name="CT"><xs:all><xs:group ref="AllG"/></xs:all></xs:complexType>`,
	}, {
		// Clause 1.3 is recursive: two levels of all-in-all is still legal, and a
		// checker that only looked one level out would reject the inner one.
		name: "legal 1.3 recursively, three all groups deep",
		body: allGroupDef +
			`<xs:group name="MidG"><xs:all><xs:group ref="AllG"/></xs:all></xs:group>` +
			`<xs:complexType name="CT"><xs:all><xs:group ref="MidG"/></xs:all></xs:complexType>`,
	}, {
		// A sequence-bodied group referenced from a sequence is untouched: the
		// constraint has nothing to say about a model group that is not all.
		name: "legal a sequence-bodied group ref inside a sequence",
		body: seqGroupDef + `<xs:complexType name="CT"><xs:sequence><xs:group ref="SeqG"/></xs:sequence></xs:complexType>`,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected the document to be accepted, got %v", err)
				}
				return
			}
			assertRule(t, err, tc.want)
		})
	}
}

// TestProduceAllGroupLimitedLiteralNesting pins the literal spellings, which the
// component-level check subsumes: a literal <all> under a <sequence>/<choice>
// builds exactly the component shape a <group ref> to an all-bodied definition
// does, so the producer's own charge was deleted rather than kept beside it.
func TestProduceAllGroupLimitedLiteralNesting(t *testing.T) {
	for _, tc := range []allLimitedDoc{{
		name: "literal all inside a sequence",
		body: `<xs:complexType name="CT"><xs:sequence><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:sequence></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		name: "literal all inside a choice",
		body: `<xs:complexType name="CT"><xs:choice><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:choice></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		name: "literal all inside a sequence inside a group definition",
		body: `<xs:group name="G"><xs:sequence><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:sequence></xs:group>`,
		want: "cos-all-limited",
	}, {
		name: "legal literal all as the content particle",
		body: `<xs:complexType name="CT"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected the document to be accepted, got %v", err)
				}
				return
			}
			assertRule(t, err, tc.want)
		})
	}
}

// TestProduceAllGroupLimitedInAnonymousType pins that an inline anonymous
// complex type is reached: it is in no index, so a walk driven only from the
// schema's {type definitions} would accept the shape there.
func TestProduceAllGroupLimitedInAnonymousType(t *testing.T) {
	body := allGroupDef +
		`<xs:element name="root"><xs:complexType><xs:sequence><xs:group ref="AllG"/></xs:sequence></xs:complexType></xs:element>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "cos-all-limited")
}

// TestProduceAllGroupLimitedThroughExtensionSynthesis pins the shape no author
// writes. §3.4.2.3.3 clause 4.2.3.3 synthesizes {term} = a sequence over (·base
// particle·, ·effective content·), so extending an all-content base with non-all
// content — or any non-empty base with an <all> — puts an all group inside a
// sequence with no <group ref> and no literal nesting anywhere.
//
// cos-ct-extends (§3.4.6.2) does NOT keep the shape out: its clause 1.4.3.2.2.2
// defers to cos-particle-extend (§3.9.6.2), whose clause 2 positively PERMITS a
// 1..1 sequence whose first member is the base particle. Nothing else carves it
// out either, so cos-all-limited is what rejects it — which the W3C suite's
// msData/complexType ctH013-ctH024 expect.
func TestProduceAllGroupLimitedThroughExtensionSynthesis(t *testing.T) {
	for _, tc := range []allLimitedDoc{{
		// 4.2.3.3 through a non-all base and an <all> extension.
		name: "sequence base extended with an all",
		body: `<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>` +
			`<xs:complexType name="D"><xs:complexContent><xs:extension base="B">` +
			`<xs:all><xs:element name="b" type="xs:string"/></xs:all></xs:extension></xs:complexContent></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		// 4.2.3.3 the other way round: an all base and a <sequence> extension.
		name: "all base extended with a sequence",
		body: `<xs:complexType name="B"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>` +
			`<xs:complexType name="D"><xs:complexContent><xs:extension base="B">` +
			`<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>`,
		want: "cos-all-limited",
	}, {
		// 4.2.3.2 instead: both terms are all groups, so the merge flattens them
		// into ONE all group and no nesting is synthesized.
		name: "legal an all base extended with an all merges under 4.2.3.2",
		body: `<xs:complexType name="B"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>` +
			`<xs:complexType name="D"><xs:complexContent><xs:extension base="B">` +
			`<xs:all><xs:element name="b" type="xs:string"/></xs:all></xs:extension></xs:complexContent></xs:complexType>`,
	}, {
		// 4.2.1: an EMPTY base routes to clause 4.1, whose {particle} is the
		// effective content itself — still the content particle, clause 1.2.
		name: "legal an empty base extended with an all",
		body: `<xs:complexType name="B"/>` +
			`<xs:complexType name="D"><xs:complexContent><xs:extension base="B">` +
			`<xs:all><xs:element name="b" type="xs:string"/></xs:all></xs:extension></xs:complexContent></xs:complexType>`,
	}, {
		// 4.2.3.1: an all base with an empty ·explicit content· yields the base
		// particle unchanged, so adding only attributes stays legal.
		name: "legal an all base extended with attributes only",
		body: `<xs:complexType name="B"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>` +
			`<xs:complexType name="D"><xs:complexContent><xs:extension base="B">` +
			`<xs:attribute name="x" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType>`,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected the document to be accepted, got %v", err)
				}
				return
			}
			assertRule(t, err, tc.want)
		})
	}
}

// TestProduceAllGroupLimitedCarriesPosition pins that the finalize-time charge
// arrives with a real file:line:column and not the "?:" of a zero Loc (#310,
// #359). A Particle and a Model Group retain no position, so the charge names
// the nearest enclosing component that does — the complex type or model group
// definition a reader must open.
func TestProduceAllGroupLimitedCarriesPosition(t *testing.T) {
	for _, tc := range []struct {
		name     string
		body     string
		wantLine int
	}{{
		name:     "charged to the complex type",
		body:     "\n" + allGroupDef + "\n" + `<xs:complexType name="CT"><xs:sequence><xs:group ref="AllG"/></xs:sequence></xs:complexType>`,
		wantLine: 3,
	}, {
		name:     "charged to the model group definition",
		body:     "\n" + allGroupDef + "\n" + `<xs:group name="Outer"><xs:sequence><xs:group ref="AllG"/></xs:sequence></xs:group>`,
		wantLine: 3,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			assertRule(t, err, "cos-all-limited")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:%d:col (E3)", err, produceURI, tc.wantLine)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the enclosing declaration at %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}

// TestProduceCompositorChildOfAllRejected pins which mechanism refuses each
// compositor child of an <all>, because the three do not share one.
//
// An <all> child is a bare grammar fault against Appendix A's xs:allModel, which
// admits only <element>, <any> and <group> inside an <all>: it carries no rule
// ID — §3.8.3 states none for <all> — and in particular is NOT cos-all-limited,
// whose clause 1.3 permits the very component that spelling maps to, leaving the
// document ill-formed rather than the component ill-shaped.
//
// A <sequence> or <choice> child is the opposite case: its component is exactly
// what clause 2 forbids, so finalize charges cos-all-limited with a real Loc and
// the parser must not pre-empt that with an unnamed fault.
func TestProduceCompositorChildOfAllRejected(t *testing.T) {
	for _, tc := range []struct {
		child string
		want  xsderr.Rule // empty: a grammar fault naming xs:allModel, no rule ID
	}{
		{child: "all"},
		{child: "sequence", want: "cos-all-limited"},
		{child: "choice", want: "cos-all-limited"},
	} {
		t.Run(tc.child, func(t *testing.T) {
			body := `<xs:complexType name="CT"><xs:all><xs:` + tc.child + `><xs:element name="a" type="xs:string"/></xs:` + tc.child + `></xs:all></xs:complexType>`
			_, err := produce(t, wrap("", body))
			if err == nil {
				t.Fatalf("<%s> inside an <all> was accepted", tc.child)
			}
			if tc.want == "" {
				if _, ok := xsderr.RuleOf(err); ok {
					t.Fatalf("error %v carries a rule ID, want a plain grammar fault", err)
				}
				if !strings.Contains(err.Error(), "xs:allModel") {
					t.Fatalf("error %v does not name xs:allModel", err)
				}
				return
			}
			assertRule(t, err, tc.want)
			if !strings.Contains(err.Error(), "clause 2") {
				t.Fatalf("error %v does not name clause 2, which is the sub-clause a %s member violates", err, tc.child)
			}
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want the enclosing complex type in %s (E3)", err, produceURI)
			}
			if loc.URI != produceURI || loc.Line == 0 || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want a real line and column in %s", loc.URI, loc.Line, loc.Col, produceURI)
			}
		})
	}
}
