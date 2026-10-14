package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/jsonsrc"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file is the json lane: the curated JSON corpus's discovery and the
// executor that runs it through validate/jsonsrc (conformance/doc.go, "The
// curated JSON corpus").
//
// The corpus is a SECOND case root beside the W3C suite, and it reaches no
// lane but json. parseSuite never reads it, so the suite's discovery, the
// Catalog reader and the agreement test between them see the suite alone;
// TestConformance appends the curated cases to the suite's, and every curated
// case is kindJSON, which only the json lane's selector claims
// (TestNoOtherLaneClaimsACuratedCase).

// jsonCorpusIndex is the curated corpus's one testSet document, as `go test`
// sees it from the package directory.
const jsonCorpusIndex = "testdata/json/json.testSet"

// curatedJSONCases discovers the curated corpus at index: one kindJSON case per
// instanceTest, sorted by ID, each assessed against its group's schemaTest
// documents as an instance case is (groupSchemaDocs). A schemaTest is not a
// case: the schema lane scores the suite's schema documents, not this one.
//
// The testSet is read through the suite's own structs and makeCase, so a case
// carries an ID and an expectation built as a suite case's is. Unlike the
// suite's, a curated testSet scopes nothing away by `version`: every case it
// declares is the lane's to run.
func curatedJSONCases(index string) ([]caseSpec, error) {
	set, err := decodeTestSet(index)
	if err != nil {
		return nil, err
	}
	setDir := filepath.Dir(index)
	seen := map[string]struct{}{}
	var cases []caseSpec
	for _, g := range set.Groups {
		for _, it := range g.InstanceTests {
			c, err := makeCase(set.Name, kindJSON, it, g, setDir, seen)
			if err != nil {
				return nil, err
			}
			cases = append(cases, c)
		}
	}
	slices.SortFunc(cases, func(a, b caseSpec) int { return strings.Compare(a.id, b.id) })
	return cases, nil
}

// The json lane's refusals, beside the instance lane's it shares.
const (
	refuseJSONUnresolved refusal = "json-unresolved" // execJSONCase: the instance will not open
	refuseJSONDeclined   refusal = "json-declined"   // execJSONCase: jsonsrc.Validate returned an error
	// execJSONCase: attributeGovernanceDiverges, on a "valid" reading
	refuseJSONAttributeGovernance refusal = "json-attribute-governance"
	// execJSONCase: nillableErrorType, on a "valid" reading
	refuseJSONNilledErrorType refusal = "json-nilled-error-type"
)

// newJSONExec builds the json lane's executor over the strict backend, built
// once, as newInstanceExec builds the instance lane's.
func newJSONExec() laneExecutor {
	backend := strict.New()
	return func(c caseSpec) (Status, refusal) {
		return execJSONCase(backend, c)
	}
}

// execJSONCase decides one curated case, or declines it naming the refusal: it
// assembles the group's schema (assembleCase), assesses the JSON instance
// through jsonsrc.Validate, and reads the Result as the instance lane reads
// one. A set of decidable charges (decidedNotValid) is "not valid"; an empty
// Result with nothing in Result.Unevaluated is "valid".
//
// That second reading is wider than the instance lane's, which also holds the
// document to assessedSubtreeRoot's shape. It is sound for the infoset jsonsrc
// builds alone, because each refusal of that gate is enforced here, by the
// walk, or unrepresentable in that infoset:
//
//   - versioned: refused here, through closureVersioned as the gate does;
//   - the raw re-read's (gate-unresolved, decode, doctype, prolog-text,
//     epilog-directive): unrepresentable, jsonsrc building no DOCTYPE, prolog
//     or epilog, and refusing an instance that will not open
//     (refuseJSONUnresolved) or will not map (refuseJSONDeclined);
//   - nil-lexical: unrepresentable, the one xsi:nil jsonsrc presents being a
//     null's, with the lexical form true, which has an ·actual value·
//     (jsonsrc's walker.nilled): a "$type" member presents xsi:type alone
//     (walker.lead), and a key naming an attribute use in the xsi namespace
//     declines (walker.useNamed);
//   - the three xsi-type ones, at the xsi:type a "$type" member presents:
//     xsi-type-unresolved charged by the walk, cvc-attribute clause 5 for a
//     name no type definition carries (validate's walk.instanceTypeResolves,
//     the json case type/type-unresolved.i) and clause 3 for a lexical with
//     no ·actual value· (walk.instanceTypeLexical, jsonsrc's TestTypeMember
//     row "unbound prefix"), and recorded in Result.Unevaluated where the
//     walk declines either clause; xsi-type-not-overriding charged by the
//     walk, cvc-elt clause 4 (walk.instanceOverride, the json case
//     type/type-unrelated.i); and xsi-type-undecided recorded in
//     Result.Unevaluated by walk.instanceOverride and walk.localGovernance,
//     which no finalized Schema reaches (#2174). Where the xsi:type governs,
//     jsonsrc has classified the element's members under the type it names,
//     resolved through the walk's own resolver (internal/xmlqname, from
//     jsonsrc's walker.instanceType);
//   - error-type: refused here on a "valid" reading, through
//     nillableErrorType, which says why no other route reaches it; the walk
//     charges cvc-type clause 3.1.3 for an element ·xs:error· governs that is
//     not ·nilled·;
//   - type-table-whitespace: unrepresentable while jsonsrc declines a matched
//     declaration carrying a {type table}. A wildcard's child that a top-level
//     declaration governs in its place has none either: cos-element-
//     consistent's wildcard condition gives the two equivalent {type table}s;
//   - strict-attribute-unresolved and attribute-ldt: refused here on a "valid"
//     reading, through attributeGovernanceDiverges, which says why no other
//     element reaches them;
//   - undeclared-root: unrepresentable, jsonsrc naming the root only by a
//     top-level declaration;
//   - recorded in Result.Unevaluated by the walk: type-table-undecided
//     (key-cta-ta-select), type-unresolved (cvc-elt clause 1, charged
//     instead for an ·absent· substitution group head), attribute-type
//     (cvc-complex-type clause 4 for a ·defaulted attribute·, the one use
//     whose absent attribute's type the walk reads, and cvc-attribute clause
//     3 for a present attribute, validate's walk.declaredAttribute, which no
//     schema document reaches: §3.2.2.2 types a typeless declaration
//     xs:anySimpleType, and src-resolve refuses a type= naming a complex
//     type), and content-matcher (cvc-complex-content, at each unmatched element
//     [[child]] or, with none, at the element: validate's
//     contentCheck.sequenceEnd);
//   - charged by the walk: element-child (cvc-type clause 3.1.2,
//     cvc-complex-type clauses 1.1 and 1.2), simple-attribute (cvc-type clause
//     3.1.1), attribute-unadmitted (cvc-complex-type clause 2.2.1 or 2.2.2,
//     validate's walk.unmatchedAttribute), content-incomplete and
//     content-rejected (cvc-complex-content);
//   - unreachable on a finalized Schema: attribute-use-unresolved,
//     member-unresolved and any-type; and type-kind, content-type and
//     attribution, each the default of a sealed sum.
//
// jsonsrc's error channel is a decline here, never a verdict: Validate returns
// (nil, err) for malformed JSON and for every input its arms do not map, and
// neither says anything of the document's validity.
func execJSONCase(backend value.Backend, c caseSpec) (Status, refusal) {
	schema, report, decidable, perr := assembleCase(backend, c.schemaDoc, c.schemaExtraDocs)
	if !decidable {
		return Fail(), refuseGroupAssembly
	}
	if perr != nil {
		return Fail(), refuseSchemaError
	}
	if closureVersioned(report, c.doc) {
		return Fail(), refuseVersioned
	}
	v, err := validate.New(schema, backend)
	if err != nil {
		return Fail(), refuseValidator
	}
	result, why := assessJSON(v, c.doc)
	if why != "" {
		return Fail(), why
	}
	if violations := result.Violations(); len(violations) > 0 {
		if !decidedNotValid(violations) {
			return Fail(), refuseUndecidedRule
		}
		return decideAgreement(false, c.expect.wantsValid()), ""
	}
	if unevaluated := result.Unevaluated(); len(unevaluated) > 0 {
		return Fail(), unevaluatedRefusal(unevaluated)
	}
	if attributeGovernanceDiverges(schema) {
		return Fail(), refuseJSONAttributeGovernance
	}
	if nillableErrorType(schema) {
		return Fail(), refuseJSONNilledErrorType
	}
	return decideAgreement(true, c.expect.wantsValid()), ""
}

// nillableErrorType reports whether schema has an element declaration D — a
// top-level one, one a named model group owns, or one the content model of a
// Complex Type Definition reached below contains — whose {nillable} is
// true and whose {type definition} is ·xs:error·. It is the condition, over
// the schema alone, without which refuseErrorType cannot arise in an infoset
// jsonsrc builds: §3.16.7.3 says an item ·xs:error· governs "will be invalid",
// and the walk charges that as cvc-type clause 3.1.3 only for an element that
// is not ·nilled·, while a JSON null presents xsi:nil="true" (key-nilled
// needs D.{nillable} too). The other routes to ·xs:error· stay closed: jsonsrc
// declines a declaration carrying a {type table}; a wildcard's child governed
// by a top-level declaration is governed by one of those visited here; and an
// xsi:type naming ·xs:error· is presented only by a "$type" member, which
// rides an object value and never a null, so its element carries no xsi:nil,
// is never ·nilled·, and is charged cvc-type clause 3.1.3 wherever ·xs:error·
// governs it (jsonsrc's TestTypeMember row "xs:error is not nilled"); where it
// does not govern — it does not ·override· the type a declaration or a
// ·locally declared type· supplies — another type does, and no ·xs:error· item
// arises from it. It refuses whether or not the document reaches D, as
// attributeGovernanceDiverges does.
//
// The declarations visited are ownership's.
func nillableErrorType(schema *xsd.Schema) bool {
	return ownership{
		schema: schema,
		decl:   func(d xsd.ElementDeclaration) bool { return nilledError(schema, d) },
		typ:    func(xsd.ComplexType) bool { return false },
	}.any()
}

// nilledError reports whether d's {nillable} is true and its {type
// definition} is ·xs:error·.
func nilledError(schema *xsd.Schema, d xsd.ElementDeclaration) bool {
	if !d.Nillable() {
		return false
	}
	td, ok := schema.ResolvedType(d.TypeDefinition())
	return ok && td.Name() == errorTypeName
}

// ownership is a walk over schema's element declarations and Complex Type
// Definitions, for the json lane's schema-only refusal predicates: any reports
// whether decl answers true for a declaration it visits or typ for a type.
//
// The types visited are every named one (Schema.Types) and every anonymous
// one an ownership edge reaches from them, from a top-level declaration, or
// from a declaration a named model group owns (Schema.ModelGroups): the inline
// {type definition} of a declaration whose {scope}.{parent} is the type being
// visited, and an inline {base type definition} (src-expredef clause 1.1).
// Each anonymous type has one owner, so those edges form a tree rooted at the
// top-level components and the walk needs no visited set (STYLE D4). The
// declarations visited are the top-level ones, those a named model group owns,
// and those each visited type's content model contains
// (Schema.ContainedElementDeclarations), a declaration reached on two paths
// being visited twice.
//
// A declaration whose {scope}.{parent} is an arm the walk does not know
// answers true, refusing rather than leaving its declarations unvisited.
type ownership struct {
	schema *xsd.Schema
	decl   func(xsd.ElementDeclaration) bool
	typ    func(xsd.ComplexType) bool
}

// any runs the walk from the top-level components.
func (o ownership) any() bool {
	for _, d := range o.schema.Elements() {
		if o.declares(d) {
			return true
		}
	}
	for _, td := range o.schema.Types() {
		if t, ok := td.(xsd.ComplexType); ok && o.contains(t) {
			return true
		}
	}
	for _, g := range o.schema.ModelGroups() {
		if o.groupOwns(g.ModelGroup()) {
			return true
		}
	}
	return false
}

// declares visits d, and the walk below d's inline Complex Type Definition.
func (o ownership) declares(d xsd.ElementDeclaration) bool {
	if o.decl(d) {
		return true
	}
	t, ok := ownedComplexType(d.TypeDefinition())
	return ok && o.contains(t)
}

// groupOwns is the walk over the declarations a named model group's {model
// group} g owns: its inline element particles, through its inline model
// groups. A reference by name is no ownership edge: an <element ref> names a
// top-level declaration and a <group ref> a model group definition, each
// visited from its own loop in any.
func (o ownership) groupOwns(g xsd.ModelGroup) bool {
	for _, p := range g.Particles() {
		resolved, ok := p.Term().(xsd.ResolvedTerm)
		if !ok {
			continue // an ElementDeclarationRef or a ModelGroupRef
		}
		switch term := resolved.Term.(type) {
		case xsd.ElementDeclaration:
			if o.declares(term) {
				return true
			}
		case xsd.ModelGroup:
			if o.groupOwns(term) {
				return true
			}
		case xsd.Wildcard:
			// declares nothing
		}
	}
	return false
}

// contains visits t, the declarations t's content model contains, and the
// walk down each ownership edge from t.
func (o ownership) contains(t xsd.ComplexType) bool {
	if o.typ(t) {
		return true
	}
	if base, ok := ownedComplexType(t.Base()); ok && o.contains(base) {
		return true
	}
	for _, d := range o.schema.ContainedElementDeclarations(t) {
		if o.decl(d) {
			return true
		}
		inner, ok := ownedComplexType(d.TypeDefinition())
		if !ok {
			continue
		}
		parent, _ := d.Scope().Parent()
		switch p := parent.(type) {
		case nil:
			// A top-level declaration: any visits its type.
		case xsd.ModelGroupScopeParent:
			// A named model group's: visited from schema.ModelGroups().
		case xsd.ComplexTypeScopeParent:
			if p.Name == t.Name() && o.contains(inner) {
				return true
			}
		case xsd.AnonymousComplexTypeScopeParent:
			ctx, ok := t.Context()
			if ok && ctx.ID() == p.Owner && o.contains(inner) {
				return true
			}
		default:
			// An ElementScopeParent arm this walk does not know: refuse rather
			// than leave its declarations unvisited.
			return true
		}
	}
	return false
}

// ownedComplexType is the anonymous Complex Type Definition ref owns (its
// InlineTypeDefinition arm), and false for a reference by name, an ·absent·
// slot or a simple type.
func ownedComplexType(ref xsd.TypeDefinitionOrRef) (xsd.ComplexType, bool) {
	inline, ok := ref.(xsd.InlineTypeDefinition)
	if !ok {
		return xsd.ComplexType{}, false
	}
	t, ok := inline.Definition.(xsd.ComplexType)
	return t, ok
}

// attributeGovernanceDiverges reports whether an attribute jsonsrc presents
// under one type can be assessed under another that admits it through its
// {attribute wildcard} alone. It is the condition, over the schema alone,
// without which neither refuseStrictAttribute nor refuseAttributeLDT can arise
// in an infoset jsonsrc builds, and it holds where either of these does:
//
//  1. a top-level element declaration G has a {type definition} T that is a
//     Complex Type Definition with a strict or lax {attribute wildcard} and
//     lacks an attribute use some type on its {base type definition} chain
//     carries (subtreeGate.onChain, which answers true for a base that does
//     not resolve, as this does for a G whose type does not);
//  2. a Complex Type Definition P (ownership's walk) has an {open content}
//     whose {wildcard} is strict or lax, and a ·locally declared type· L
//     within P (key-ldt-elem: the {type definition} of a declaration the
//     content model of a type on P's {base type definition} chain contains)
//     has a strict or lax {attribute wildcard} and lacks an attribute use some
//     named Complex Type Definition X carries that does not ·override· L
//     (localGovernanceDiverges).
//
// Why those two and no other:
//
//   - jsonsrc presents an attribute only under a use of the type its element's
//     members are classified under — the one its "$type" member names where
//     that resolves, and otherwise the {type definition} of the declaration D
//     its element's key matched (jsonsrc's walker.classifyingType) — so the
//     attribute matches no use (cvc-complex-type clause 2.2, where both
//     refusals sit) only where some other type governs the element;
//   - where a "$type" names a type X, X governs wherever it ·overrides· the
//     type a ·governing element declaration· or a ·locally declared type·
//     supplies, and wherever there is neither (key-governing-type-elem clause
//     8). Where X does not ·override· a declaration's type, the walk charges
//     cvc-elt clause 4 (walk.instanceOverride). Where it does not ·override· a
//     non-·absent· ·locally declared type· L — a child ·attributed to· an
//     {open content}, which then has no ·governing element declaration·
//     (key-governing-ed clause 4.3) — L governs (key-governing-type-elem
//     clause 7) and nothing is charged (walk.localGovernance), which is
//     condition 2. Where the "$type" names none, the walk charges
//     cvc-attribute or records it unevaluated (execJSONCase's xsi-type
//     bullet);
//   - with no "$type", D's type governs at the root and under an element
//     particle D governs (cos-element-consistent gives same-named contained
//     declarations one type), and under an {open content} the ·locally
//     declared type· does, which is D's type;
//   - which leaves a child ·attributed to· a strict or lax Wildcard, or to an
//     {open content} with an ·absent· ·locally declared type·, and governed by
//     the top-level G its name ·resolves· to, its members classified under
//     D's type: condition 1. The walk charges cvc-complex-type clause 5 unless
//     T is ·validly substitutable· for D's type, which carries the use T
//     lacks, and a skip {attribute wildcard} or none admits nothing either
//     refusal reads.
//
// The walk charges and records nothing for either shape (subtreeGate's
// wildcardAttribute), so a "valid" reading is refused wherever the condition
// holds, whether or not the document reaches it.
func attributeGovernanceDiverges(schema *xsd.Schema) bool {
	g := subtreeGate{schema: schema}
	for _, d := range schema.Elements() {
		td, ok := schema.ResolvedType(d.TypeDefinition())
		if !ok {
			return true
		}
		t, ok := td.(xsd.ComplexType)
		if !ok || !admitsByWildcard(t) {
			continue
		}
		if g.onChain(t, func(c xsd.ComplexType) bool { return lacksUseOf(t, c) }) {
			return true
		}
	}
	return ownership{
		schema: schema,
		decl:   func(xsd.ElementDeclaration) bool { return false },
		typ:    func(p xsd.ComplexType) bool { return localGovernanceDiverges(g, p) },
	}.any()
}

// localGovernanceDiverges is attributeGovernanceDiverges's condition 2 for the
// Complex Type Definition p. A declaration on p's chain whose {type
// definition} does not resolve answers true, as onChain does for a base.
func localGovernanceDiverges(g subtreeGate, p xsd.ComplexType) bool {
	content, ok := p.ContentType().(xsd.ElementContent)
	if !ok || content.OpenContent == nil || content.OpenContent.Wildcard().ProcessContents() == xsd.ProcessSkip {
		return false
	}
	return g.onChain(p, func(c xsd.ComplexType) bool {
		return slices.ContainsFunc(g.schema.ContainedElementDeclarations(c), func(d xsd.ElementDeclaration) bool {
			ldt, ok := g.schema.ResolvedType(d.TypeDefinition())
			if !ok {
				return true
			}
			l, ok := ldt.(xsd.ComplexType)
			return ok && admitsByWildcard(l) && strayUse(g.schema, l)
		})
	})
}

// strayUse reports whether a named Complex Type Definition x carries an
// attribute use l lacks and does not ·override· l: ·validly substitutable·
// for it ·without limitation· (key-overrides clause 2, the empty set of
// blocking keywords, as walk.localGovernance asks). An error from
// ValidlySubstitutable, unreachable on a finalized Schema, answers true.
func strayUse(schema *xsd.Schema, l xsd.ComplexType) bool {
	for _, td := range schema.Types() {
		x, ok := td.(xsd.ComplexType)
		if !ok || !lacksUseOf(l, x) {
			continue
		}
		overrides, err := schema.ValidlySubstitutable(x, l, nil)
		if err != nil || !overrides {
			return true
		}
	}
	return false
}

// admitsByWildcard reports whether t has a strict or lax {attribute
// wildcard}: a skip one or none admits nothing refuseStrictAttribute or
// refuseAttributeLDT reads.
func admitsByWildcard(t xsd.ComplexType) bool {
	wild, ok := t.AttributeWildcard()
	return ok && wild.ProcessContents() != xsd.ProcessSkip
}

// lacksUseOf reports whether c carries an attribute use whose {attribute
// declaration}'s expanded name none of t's {attribute uses} carries.
func lacksUseOf(t, c xsd.ComplexType) bool {
	own := t.AttributeUses()
	return slices.ContainsFunc(c.AttributeUses(), func(u xsd.AttributeUse) bool {
		return !slices.ContainsFunc(own, func(o xsd.AttributeUse) bool { return o.DeclarationName() == u.DeclarationName() })
	})
}

// assessJSON opens the JSON instance at doc and assesses it against v, or
// declines, naming the refusal.
func assessJSON(v *validate.Validator, doc string) (*validate.Result, refusal) {
	f, err := os.Open(doc)
	if err != nil {
		return nil, refuseJSONUnresolved
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the verdict
	result, err := jsonsrc.Validate(v, f, jsonsrc.WithURI(doc))
	if err != nil {
		return nil, refuseJSONDeclined
	}
	return result, ""
}
