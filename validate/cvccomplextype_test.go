package validate

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive the ATTRIBUTE-EXISTENCE half of cvc-complex-type
// (§3.4.4.2) — clauses 2 and 3, which need no value space — over
// the validation root, which is the only element whose ·governing type
// definition· this package determines. Every schema is built through the
// exported constructors: the producer is not involved, so a shape it cannot
// yet emit (an attribute wildcard) is still reachable here. No declaration
// here names a {type definition}, which is what keeps the value charges out —
// they live in cvcattribute_test.go, over schemas that seed the builtins.

// aUse builds an attribute use over a sibling local declaration named local
// in no namespace, with the given {required} and the use's own {value
// constraint} — the one cvc-au (§3.5.4) reads, not the declaration's. The
// declaration carries NO {type definition}, so every value charge declines on
// it; typedUse (cvcattribute_test.go) is the one that names a type.
func aUse(t *testing.T, local string, required bool, vc *xsd.ValueConstraint) xsd.AttributeUse {
	t.Helper()
	decl, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: local}, nil,
		xsd.NewAttributeGlobalScope(), nil, false, nil)
	if err != nil {
		t.Fatalf("building the %s attribute declaration: %v", local, err)
	}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, required,
		xsd.LocalAttributeDeclaration{Declaration: decl}, vc, false, nil)
	if err != nil {
		t.Fatalf("building the %s attribute use: %v", local, err)
	}
	return u
}

// anyWildcard is the ·complete wildcard·: {variety} any, so clause 2.2.1 is
// satisfied for every name and only cvc-wildcard's own clauses could reject.
func anyWildcard(t *testing.T) *xsd.Wildcard {
	t.Helper()
	c, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the namespace constraint: %v", err)
	}
	w, err := xsd.NewWildcard(xsderr.Loc{}, c, xsd.ProcessStrict, nil)
	if err != nil {
		t.Fatalf("building the wildcard: %v", err)
	}
	return &w
}

// governedSchema declares "root" governed by a complex type carrying uses and
// wildcard. The type's {base type definition} is left ·absent·: the fold that
// would inherit a base's uses is finalize's, and a type with no base carries
// exactly the uses named here.
func governedSchema(t *testing.T, uses []xsd.AttributeUse, wildcard *xsd.Wildcard) *xsd.Schema {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "RootType"}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, uses, nil, wildcard, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the governed schema: %v", err)
	}
	return schema
}

// attributedRoot is a childless <root> carrying the named attributes in
// source order, each at a distinct Loc.
func attributedRoot(names ...xsd.QName) *testElement {
	e := &testElement{name: xsd.QName{Local: "root"}, loc: loc(1, 1)}
	for i, n := range names {
		e.attrs = append(e.attrs, &testAttribute{name: n, value: "v", loc: loc(1, 10+i)})
	}
	return e
}

func local(name string) xsd.QName { return xsd.QName{Local: name} }

// assessRoot assesses root against a schema declaring "root" over uses and
// wildcard, and returns the violations charged.
func assessRoot(t *testing.T, root Element, uses []xsd.AttributeUse, wildcard *xsd.Wildcard) []*xsderr.Error {
	t.Helper()
	v, err := New(governedSchema(t, uses, wildcard), testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res.Violations()
}

// wantSilence fails unless the assessment charged nothing. It is the
// assertion every DECLINE shares with every PASS — the two are
// indistinguishable in a Result, which is exactly why conformance/instance.go
// reads an empty Result as a decline and never as "valid".
func wantSilence(t *testing.T, got []*xsderr.Error, why string) {
	t.Helper()
	if len(got) != 0 {
		t.Errorf("Violations() = %v, want none: %s", got, why)
	}
}

// assessOutcomes assesses root and returns the clause/outcome pair the debug
// log recorded for each attribute, in visit order. It is how a DECLINE is
// told apart from a PASS at all: both charge nothing, so the log is the only
// place the difference is observable (STYLE L1) and the only place a test can
// pin which branch ran.
func assessOutcomes(t *testing.T, root Element, uses []xsd.AttributeUse, wildcard *xsd.Wildcard) []string {
	t.Helper()
	log, visits := recordingLogger()
	v, err := New(governedSchema(t, uses, wildcard), testBackend(), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	v.Assess(root)
	outcomes := []string{}
	for _, line := range *visits {
		if !strings.HasPrefix(line, "assessing attribute ") {
			continue
		}
		clause, outcome := field(line, "validate.clause="), field(line, "validate.outcome=")
		outcomes = append(outcomes, clause+"/"+outcome)
	}
	return outcomes
}

// field pulls one space-delimited key=value out of a recorded log line, or
// "-" where the line does not carry that key.
func field(line, key string) string {
	for _, f := range strings.Fields(line) {
		if after, ok := strings.CutPrefix(f, key); ok {
			return after
		}
	}
	return "-"
}

// wantCharge fails unless the assessment charged exactly one
// cvc-complex-type violation naming clause and citing at.
func wantCharge(t *testing.T, got []*xsderr.Error, clause string, at xsderr.Loc, names ...string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one", got)
	}
	if got[0].Rule != "cvc-complex-type" {
		t.Errorf("Rule = %q, want cvc-complex-type", got[0].Rule)
	}
	if !strings.Contains(got[0].Msg, clause) {
		t.Errorf("Msg = %q, want it to name %q — the catalog cannot carry a dotted rule ID", got[0].Msg, clause)
	}
	if got[0].Loc != at {
		t.Errorf("Loc = %s, want %s", got[0].Loc, at)
	}
	for _, n := range names {
		if !strings.Contains(got[0].Msg, n) {
			t.Errorf("Msg = %q, want it to name %s", got[0].Msg, n)
		}
	}
}

// Clause 3 is an existence test over ·expanded names·: a required use with a
// matching attribute information item present is satisfied, and clause 2.1
// then settles that attribute through a cvc-au that constrains nothing.
func TestRequiredAttributePresentIsNotCharged(t *testing.T) {
	got := assessRoot(t, attributedRoot(local("id")), []xsd.AttributeUse{aUse(t, "id", true, nil)}, nil)
	wantSilence(t, got, "a required attribute use with its attribute present satisfies clauses 2.1 and 3")
}

// Clause 3: a {required} use with no attribute information item of that
// ·expanded name· is charged, against the ELEMENT's position — there is no
// missing attribute to cite one for.
func TestRequiredAttributeAbsentChargesClauseThree(t *testing.T) {
	got := assessRoot(t, attributedRoot(), []xsd.AttributeUse{aUse(t, "id", true, nil)}, nil)
	wantCharge(t, got, "clause 3", loc(1, 1), "id", "root")
}

// An OPTIONAL use with no matching attribute is silent: clause 3 quantifies
// over {required} uses alone, and the ·defaulted attribute· an absent
// optional use produces is a §3.4.5.1 infoset contribution, not a violation.
func TestOptionalAttributeAbsentIsNotCharged(t *testing.T) {
	got := assessRoot(t, attributedRoot(), []xsd.AttributeUse{aUse(t, "id", false, nil)}, nil)
	wantSilence(t, got, "clause 3 says nothing about an optional use")
}

// Clause 2 has no third arm: an attribute matching no attribute use (clause
// 2.1) whose type has no {attribute wildcard} (clause 2.2.1) satisfies
// neither case, so clause 2 is violated outright — decidable with no
// datatype backend at all. The charge cites the ATTRIBUTE's position.
func TestUndeclaredAttributeWithNoWildcardChargesClauseTwo(t *testing.T) {
	got := assessRoot(t, attributedRoot(local("stray")), []xsd.AttributeUse{aUse(t, "id", false, nil)}, nil)
	wantCharge(t, got, "clause 2", loc(1, 10), "stray")
}

// The same attribute against a type carrying an {attribute wildcard} is
// DECLINED, not charged: clause 2.2 is a conjunction whose second half sends
// the attribute to cvc-wildcard (§3.10.4.1), which this package does not
// evaluate (#717). The control row is what keeps the decline honest — the
// identical document IS charged when the wildcard is gone, so silence here
// comes from the wildcard and not from an accidental match.
func TestUndeclaredAttributeWithWildcardIsDeclined(t *testing.T) {
	uses := []xsd.AttributeUse{aUse(t, "id", false, nil)}
	got := assessRoot(t, attributedRoot(local("stray")), uses, anyWildcard(t))
	wantSilence(t, got, "clause 2.2.2 needs cvc-wildcard, so clause 2 is undecided")
	if outcomes := assessOutcomes(t, attributedRoot(local("stray")), uses, anyWildcard(t)); !slices.Equal(outcomes, []string{"2.2/declined"}) {
		t.Errorf("assessed %v, want the attribute DECLINED under clause 2.2 — not matched to a use", outcomes)
	}

	control := assessRoot(t, attributedRoot(local("stray")), uses, nil)
	if len(control) != 1 {
		t.Fatalf("the control charged %v, want exactly one: the decline above proves nothing otherwise", control)
	}
}

// A wildcard does not silence clause 3: the missing required use is charged
// whatever the type admits beyond its {attribute uses}, since clause 3 reads
// only the uses.
func TestWildcardDoesNotSilenceClauseThree(t *testing.T) {
	got := assessRoot(t, attributedRoot(), []xsd.AttributeUse{aUse(t, "id", true, nil)}, anyWildcard(t))
	wantCharge(t, got, "clause 3", loc(1, 1), "id")
}

// aUse's declaration carries no {type definition} at all, so every attribute
// matched to one declines at cvc-attribute clause 3 — before cvc-au is
// reached, whatever {value constraint} the use carries. The fixed and default
// rows are the same silence for that one reason, which is what makes this the
// control for the typed charges in cvcattribute_test.go: those fire only
// because their declaration names a type this backend governs.
func TestMatchedAttributeWithoutATypeIsDeclined(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)
	dflt := xsd.NewValueConstraint(xsd.ValueDefault, "1", nil, nil)
	for _, vc := range []*xsd.ValueConstraint{nil, &fixed, &dflt} {
		uses := []xsd.AttributeUse{aUse(t, "id", true, vc)}
		wantSilence(t, assessRoot(t, attributedRoot(local("id")), uses, nil),
			"a declaration with no {type definition} yields no ·actual value· to judge")
		if outcomes := assessOutcomes(t, attributedRoot(local("id")), uses, nil); !slices.Equal(outcomes, []string{"3/declined"}) {
			t.Errorf("assessed %v, want cvc-attribute clause 3 DECLINED", outcomes)
		}
	}
}

// The four xsi: attributes of §3.2.7 are excepted from clause 2's quantifier
// by its own text, so one present but declared by no attribute use is not
// charged — the arm that would charge it never runs.
func TestInstanceAttributesAreExemptFromClauseTwo(t *testing.T) {
	xsi := func(name string) xsd.QName {
		return xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: name}
	}
	// All FOUR are excepted, xsi:type and xsi:nil included: each is ·attributed
	// to· nothing at all, whatever it goes on to decide elsewhere.
	for _, n := range []string{"type", "nil", "schemaLocation", "noNamespaceSchemaLocation"} {
		uses := []xsd.AttributeUse{aUse(t, "id", false, nil)}
		if outcomes := assessOutcomes(t, attributedRoot(xsi(n)), uses, nil); !slices.Equal(outcomes, []string{"2/exempt"}) {
			t.Errorf("assessed xsi:%s as %v, want it EXEMPT from clause 2 rather than matched", n, outcomes)
		}
	}
	// Three of them charge nothing anywhere. xsi:nil is the exception, and not
	// through clause 2: this root's declaration is not {nillable}, so cvc-elt
	// clause 3.1 charges its mere presence (cvcelt_test.go).
	for _, n := range []string{"type", "schemaLocation", "noNamespaceSchemaLocation"} {
		uses := []xsd.AttributeUse{aUse(t, "id", false, nil)}
		wantSilence(t, assessRoot(t, attributedRoot(xsi(n)), uses, nil), "xsi:"+n+" is excepted from clause 2")
	}
	// An xsi:-namespaced attribute that is NOT one of the four is inside the
	// quantifier and charged like any other, so the exemption is by name and
	// not by namespace.
	got := assessRoot(t, attributedRoot(xsi("invented")), []xsd.AttributeUse{aUse(t, "id", false, nil)}, nil)
	wantCharge(t, got, "clause 2", loc(1, 10), "invented")
}

// An xsi:type naming a type nothing declares ·resolves· to no
// ·instance-specified type definition· at all, so the ·selected type definition·
// stays the ·governing· one (key-governing-type-elem clause 4) and the attribute
// half runs against it unchanged — the fallback the Note under cvc-elt states.
func TestUnresolvableXSITypeStillAssessesAgainstTheDeclaredType(t *testing.T) {
	root := attributedRoot(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}, local("stray"))
	got := assessRoot(t, root, []xsd.AttributeUse{aUse(t, "id", false, nil)}, nil)
	wantCharge(t, got, "clause 2", loc(1, 11), "stray")
}

// A declaration carrying a {type table} withholds the assessment too: the
// ·selected type definition· is then whichever <alternative> ·conditionally
// selects· one for this instance (§3.3.4.2), which means evaluating XPath
// (#56), and D.{type definition} is only the table's fallback.
func TestTypeTabledRootDeclinesTheAttributeHalf(t *testing.T) {
	test := xsd.NewXPathExpression("@id", nil, nil, nil)
	alt := xsd.NewTypeAlternative(&test, xsd.QName{Local: "RootType"}, nil)
	table, err := xsd.NewTypeTable(xsderr.Loc{}, []xsd.TypeAlternative{alt},
		xsd.NewTypeAlternative(nil, xsd.QName{Local: "RootType"}, nil))
	if err != nil {
		t.Fatalf("building the type table: %v", err)
	}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "RootType"}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, []xsd.AttributeUse{aUse(t, "id", true, nil)}, nil, nil,
		xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, &table, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the tabled schema: %v", err)
	}
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// The same root against the same type WITHOUT the table is charged clause
	// 3 for the missing @id, so the silence here is the table's doing.
	wantSilence(t, v.Assess(attributedRoot()).Violations(),
		"a {type table} leaves the ·selected type definition· undetermined")
	wantCharge(t, assessRoot(t, attributedRoot(), []xsd.AttributeUse{aUse(t, "id", true, nil)}, nil),
		"clause 3", loc(1, 1), "id")
}

// A SIMPLE-typed root's attributes stay entirely undecided. cvc-type clause
// 3.2 dispatches to cvc-complex-type only for a complex T; a simple T is
// clause 3.1.1's much narrower question, which this package does not decide.
func TestSimpleTypedRootIsUndecided(t *testing.T) {
	st, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Local: "RootType"}, nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(st)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the simple-typed schema: %v", err)
	}
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(attributedRoot(local("stray")))
	wantSilence(t, res.Violations(), "a simple ·governing type definition· reaches cvc-complex-type through no clause")
}

// A descendant's governing type comes from the particle it is ·attributed to·
// in its parent's {content type} (§3.3.4.6 clause 3.1), and a child its parent
// attributed to NOTHING is assessed against nothing: the same stray attribute
// that is charged on the root is silent below one.
//
// The child here is that shape because it is charged: governedSchema's type has
// an empty {content type}, which admits no element information item
// [[children]] at all (clause 1.1). That charge is the root's, at the child's
// position, and it names no attribute.
func TestDescendantAttributesOfAnUnattributedChildAreNotAssessed(t *testing.T) {
	child := &testElement{
		name:  xsd.QName{Local: "child"},
		attrs: []Attribute{&testAttribute{name: local("stray"), value: "v", loc: loc(2, 3)}},
		loc:   loc(2, 1),
	}
	root := attributedRoot()
	root.kids = []Child{ElementChild(child)}

	got := assessRoot(t, root, []xsd.AttributeUse{aUse(t, "id", false, nil)}, nil)

	if len(got) != 1 || got[0].Loc != loc(2, 1) {
		t.Fatalf("Violations() = %v, want the one clause 1.1 charge at the child's own location", got)
	}
	if strings.Contains(got[0].Msg, "stray") {
		t.Errorf("Msg = %q, want no charge about a descendant's attribute", got[0].Msg)
	}
}

// The two clauses charge independently and in the order they are found:
// clause 2 over the attributes present, in source order, then clause 3 over
// the uses required. A Result therefore carries several honest charges for
// one root, which is why conformance's decidedNotValid does not pin the
// count at one.
func TestClauseTwoAndThreeChargeInDiscoveryOrder(t *testing.T) {
	root := attributedRoot(local("strayA"), local("strayB"))
	got := assessRoot(t, root, []xsd.AttributeUse{aUse(t, "req", true, nil)}, nil)
	if len(got) != 3 {
		t.Fatalf("Violations() = %v, want three", got)
	}
	wantLocs := []xsderr.Loc{loc(1, 10), loc(1, 11), loc(1, 1)}
	gotLocs := make([]xsderr.Loc, 0, len(got))
	for _, e := range got {
		gotLocs = append(gotLocs, e.Loc)
		if e.Rule != "cvc-complex-type" {
			t.Errorf("Rule = %q, want cvc-complex-type", e.Rule)
		}
	}
	if !slices.Equal(gotLocs, wantLocs) {
		t.Errorf("charged at %v, want %v: the attributes present in source order, then the uses required", gotLocs, wantLocs)
	}
	if !strings.Contains(got[2].Msg, "clause 3") {
		t.Errorf("the last charge is %q, want clause 3", got[2].Msg)
	}
}

// The debug log names the rule and clause that settled each attribute (STYLE
// L1), and says nothing about a rule where nothing was settled.
func TestAttributeLogNamesTheRuleAndClause(t *testing.T) {
	log, visits := recordingLogger()
	v, err := New(governedSchema(t, []xsd.AttributeUse{aUse(t, "id", false, nil)}, nil), testBackend(), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	v.Assess(attributedRoot(local("id"), local("stray")))

	want := []string{
		"assessing element validate.name=root validate.loc=instance.xml:1:1",
		"assessing attribute validate.name=id validate.loc=instance.xml:1:10 validate.rule=cvc-attribute validate.clause=3 validate.outcome=declined",
		"assessing attribute validate.name=stray validate.loc=instance.xml:1:11 validate.rule=cvc-complex-type validate.clause=2 validate.outcome=charged",
	}
	if !slices.Equal(*visits, want) {
		t.Errorf("walk logged\n\t%s\nwant\n\t%s",
			strings.Join(*visits, "\n\t"), strings.Join(want, "\n\t"))
	}
}

// anonymousRootSchema declares "root" governed by the ANONYMOUS complex type
// it owns (§3.3.2.1 dcl.elt.common clause 1), derived from base by derivation
// and carrying uses. Base is a named type carrying one attribute use and an
// ##any wildcard, so a fold that HAD run would give the anonymous type both.
// It is the component shape parser.Parse builds for an inline <complexType>
// child of an <element>, reached here through the constructors because this
// package imports no parser.
func anonymousRootSchema(t *testing.T, base xsd.QName, derivation xsd.DerivationMethod, uses []xsd.AttributeUse) *xsd.Schema {
	t.Helper()
	id := xsd.NewComponentID()
	ct, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, xsd.ElementDeclarationContext{Component: id},
		base, nil, derivation, false, uses, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the anonymous root type: %v", err)
	}
	e, err := xsd.NewElementDeclarationOwningType(xsderr.Loc{}, id, xsd.QName{Local: "root"}, ct, nil,
		xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	baseType, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "Base"}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, []xsd.AttributeUse{aUse(t, "fromBase", false, nil)}, nil,
		anyWildcard(t), xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building Base: %v", err)
	}
	// xs:anyType is the one complex type permitted to be its own base (§3.4.7)
	// and carries the ##any wildcard that section gives it — which a
	// RESTRICTION of it does not inherit (§3.4.2.5 clause 2.1), so its presence
	// here cannot be what makes the assessed case below silent.
	anyType, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"},
		xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}, nil, xsd.DerivationRestriction, false,
		nil, nil, anyWildcard(t), xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building xs:anyType: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddElement(e)
	b.AddType(baseType)
	b.AddType(anyType)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the anonymous-root schema: %v", err)
	}
	return schema
}

// assessAgainst assesses root against schema and returns what was charged.
func assessAgainst(t *testing.T, schema *xsd.Schema, root Element) []*xsderr.Error {
	t.Helper()
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res.Violations()
}

// An ANONYMOUS governing type derived from a named base withholds the whole
// attribute assessment: §3.4.2.4 clause 3's {attribute uses} and §3.4.2.5
// clause 2's {attribute wildcard} are folded over the schema's NAMED type
// definitions alone (#414), so Base's attribute use and its <anyAttribute>
// are both missing from the component the walk would read. Under-report the
// uses and @fromBase looks unmatched; under-report the wildcard and clause
// 2.2 does not decline it — together they charge a document Base admits.
func TestAnonymousDerivedRootDeclinesTheAttributeHalf(t *testing.T) {
	for _, derivation := range []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction} {
		schema := anonymousRootSchema(t, xsd.QName{Local: "Base"}, derivation,
			[]xsd.AttributeUse{aUse(t, "own", true, nil)})
		wantSilence(t, assessAgainst(t, schema, attributedRoot(local("fromBase"), local("own"))),
			"an unfolded anonymous type reports neither Base's uses nor its wildcard ("+derivation.String()+")")
		// The control: clause 3 is withheld on the same type too. The
		// unfolded type's own {required} use is present in it, so a walk that
		// assessed this type at all would charge the missing @own here.
		wantSilence(t, assessAgainst(t, schema, attributedRoot()),
			"the decline covers clause 3, not clause 2 alone ("+derivation.String()+")")
	}
}

// The one anonymous shape that IS assessed: a restriction of xs:anyType, the
// §3.4.2.3.2 implicit-content form, where both folds are provably the
// identity — §3.4.7 makes xs:anyType's {attribute uses} empty so clause 3
// inherits nothing, and §3.4.2.5 clause 2 unions the base's wildcard for an
// EXTENSION only. Declining it as well would withdraw every inline
// <complexType> with no <complexContent>/<simpleContent> child from the
// assessment, which is the shape the conformance suite's decidable subset is
// made of.
func TestAnonymousRestrictionOfAnyTypeIsAssessed(t *testing.T) {
	schema := anonymousRootSchema(t, xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"},
		xsd.DerivationRestriction, []xsd.AttributeUse{aUse(t, "id", true, nil)})
	wantCharge(t, assessAgainst(t, schema, attributedRoot(local("id"), local("stray"))),
		"clause 2", loc(1, 11), "stray")
	wantCharge(t, assessAgainst(t, schema, attributedRoot()), "clause 3", loc(1, 1), "id")
}
