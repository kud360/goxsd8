package parser

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The Schema Representation Constraints (§3.x.3) and other rules this package
// charges. Each string is a live entry in xsderr's generated catalog.
const (
	ruleSrcInclude            xsderr.Rule = "src-include"
	ruleSrcImport             xsderr.Rule = "src-import"
	ruleSrcImportNoSelfImport xsderr.Rule = "src-import-noselfimport"
	ruleSrcOverride           xsderr.Rule = "src-override"
	ruleSrcRedefine           xsderr.Rule = "src-redefine"
	ruleSrcExpRedefine        xsderr.Rule = "src-expredef"
	ruleSrcElement            xsderr.Rule = "src-element"
	ruleSrcAttribute          xsderr.Rule = "src-attribute"
	ruleSrcSimpleType         xsderr.Rule = "src-simple-type"
	ruleSrcResolve            xsderr.Rule = "src-resolve"
	ruleSTPropsCorr           xsderr.Rule = "st-props-correct"
	ruleCTPropsCorr           xsderr.Rule = "ct-props-correct"
	ruleSrcCT                 xsderr.Rule = "src-ct"
	ruleCosAllLimited         xsderr.Rule = "cos-all-limited"
	ruleSrcWildcard           xsderr.Rule = "src-wildcard"
	ruleParticleCorr          xsderr.Rule = "p-props-correct"
	ruleWildcardCorr          xsderr.Rule = "w-props-correct"
	ruleSrcIdentityConstraint xsderr.Rule = "src-identity-constraint"
	// ruleEPropsCorrect is the Element Declaration Properties Correct Schema
	// Component Constraint (§3.3.6.1). The producer charges only clause 3 ("If
	// E.{substitution group affiliations} is non-empty, then E.{scope}.{variety}
	// = global") and only in its SYNTACTIC form — a local <element> carrying a
	// substitutionGroup attribute, which the schema for schema documents declares
	// use="prohibited" on xs:localElement (§3.3.2). xsd.NewElementDeclaration
	// charges the same clause on the component, for the programmatic path that
	// bypasses this producer; the two are the same rule seen from either side of
	// the mapping.
	ruleEPropsCorrect xsderr.Rule = "e-props-correct"
	// ruleDatatypeValid is the generic "this attribute's value is not valid
	// against the simple type the schema for schema documents declares for it"
	// rule (Datatypes §4.1.4, cvc-datatype-valid). It is charged where a schema
	// document attribute fails its own declared type and no Structures Schema
	// Representation Constraint covers the case — notably an unrecognized ##
	// token in notQName, whose value space is fixed by xs:qnameList/xs:qnameListA
	// (§3.10.2) rather than by any src-wildcard clause.
	ruleDatatypeValid xsderr.Rule = "cvc-datatype-valid"
)

// Produce maps the TOP-LEVEL <simpleType>, <element>, <attribute>,
// <complexType>, <attributeGroup>, <group>, and <notation> declarations of a
// single already-parsed schema document into xsd components, in document order,
// and returns the finalized [xsd.Schema]. The identity constraints of an
// <element> (global or local) are produced with it; each name= form is registered
// as a schema-level {identity-constraint definitions} member (§3.17.1), while a
// ref= form contributes the definition it names and registers nothing (§3.11.2).
//
// Produce is the SINGLE-DOCUMENT entry point: it never dereferences an
// <include>/<import>/<redefine>/<override>, so schema(D) here is immed(D) alone
// (§4.2.1). Multi-document assembly through <include> — and with it chameleon
// coercion (§4.2.3 clause 3.2, §F.1) — is [Parse]'s job; a caller holding only a
// document and a backend, with no resolver, keeps this entry point. Its
// <import> children are still READ, though never followed: src-resolve clause 4
// licenses this document's QName references from them (see parser/doc.go's
// Composition section), so a type= into a foreign namespace needs an <import>
// element naming it here as much as it does under [Parse].
//
// backend is passed explicitly rather than defaulted to a builtin/strict policy
// here: that default belongs to [Parse], keeping this leaf free of a
// builtin/strict edge. Produce seeds the builtin datatypes from backend
// ([builtin.Seed]) so a type="xs:…" reference resolves at finalize; the SAME
// *[xsd.SimpleType] pointer identity is both AddType'd into the builder and used
// as a simple-type base, as [xsd.SimpleType] requires. backend also supplies the
// finalized schema's value space ([value.NewValueSpace] through
// [xsd.SchemaBuilder.FinalizeWith]), so the finalize-time constraints that reach
// into a value space decide there instead of failing open on the {lexical form}s:
// the two that COMPARE two {value}s — au-props-correct (§3.5.6) clause 3,
// loc-testSubP (§3.4.6.4) clauses 4.2 and 5.2.2 — and the two that VALIDATE one
// against its type — a-props-correct (§3.2.6.1) clause 2 and au-props-correct
// clause 2, both charging Simple Default Valid (§3.2.6.2).
//
// DEVIATION from parser/doc.go's "the parser collects them in document order
// rather than stopping at the first": Produce returns only the FIRST error. That
// promise is for the eventual full Parse; this first slice does not yet
// implement multi-error collection.
//
// A document whose root is not <schema> is a caller precondition fault, not a
// schema-validity verdict — §3.17.2 even allows <schema> not to be the document
// element — so it is reported as a plain Go error, deliberately NOT routed
// through xsderr (mirroring [builtin.MissingPrimitivesError]'s rationale): no
// src-*/cos-* rule governs "the document handed to a producer must be a schema
// document".
func Produce(doc *Document, backend value.Backend) (*xsd.Schema, error) {
	if !doc.IsSchema() {
		return nil, fmt.Errorf("parser: Produce requires a <schema> document root, got %s", doc.Root().Name().Local())
	}
	builder := xsd.NewSchemaBuilder()
	sym, err := newSymbols(builder, backend)
	if err != nil {
		return nil, err
	}
	// A lone document is its own assembly, so its effective target namespace is
	// its own targetNamespace and it is never a chameleon (§4.2.3 clause 2.3
	// needs an including document to borrow a namespace from).
	p := newProducer(doc, attrOr(doc.Root(), "targetNamespace"), nil, nil, nil, builder, sym)
	p.prescan()
	if err := p.run(); err != nil {
		return nil, err
	}
	return builder.FinalizeWith(value.NewValueSpace(backend))
}

// symbols is the ASSEMBLY-WIDE symbol table: one set of indexes shared by every
// document's producer across an entire <include> closure, so a base= or
// <attributeGroup ref> in one document reaches a definition contributed by
// another (§4.2.3 clause 3.1.2, c-incl-incl). All four are pure lookup indexes,
// never ranged to produce user-visible order (STYLE D2).
type symbols struct {
	// simpleTypes maps each top-level named <simpleType>'s expanded name to its
	// source (raw element plus the producer of the document that declares it),
	// filled by the pre-scan so forward base= references between simple types
	// resolve (Structures §3.1.3).
	//
	// The owning producer is carried for the same reason complexTypes carries one:
	// §3.16.2.1 identifies {base type definition} as what "the actual value of the
	// base attribute" resolves to, and that attribute belongs to the DECLARING
	// document — src-resolve (§3.17.6.2) clause 4.1.1 scopes its absent-namespace
	// default to "the schema document containing the QName", and §F.1 task (b)
	// coerces a chameleon-included document's own unqualified references to that
	// document's effective target namespace. Built under a referring producer
	// instead, unqualifiedRefNS/declares would answer for the wrong document and
	// falsely reject a valid base=.
	simpleTypes map[xsd.QName]typeSource

	// complexTypes maps each top-level named <complexType>'s expanded name to
	// its source (raw element plus the producer of the document that declares
	// it), filled by the pre-scan so a base= on a <complexContent>/<simpleContent>
	// derivation reaches its {base type definition} regardless of document order
	// (§3.4.2's preamble: "the mapping rules … depend upon the {base type
	// definition} having been identified before they apply").
	//
	// The owning producer is carried, not just the element, because producers are
	// per-DOCUMENT (parse.go's compile) while this index is assembly-wide: a
	// complex type built on demand from a REFERRING document must still be built
	// under its OWN document's producer, or every local element declaration
	// inside it takes the wrong {target namespace} (localTargetNS reads
	// schemaElem's elementFormDefault, and unqualifiedRefNS/declares answer for
	// the declaring document).
	complexTypes map[xsd.QName]typeSource

	// attributeGroups maps each top-level named <attributeGroup>'s expanded name
	// to its source (raw element plus the producer of the document that declares
	// it), filled by the pre-scan so an <attributeGroup ref> (from a
	// <complexType>/<restriction> or another <attributeGroup>) resolves and is
	// inlined at mapping time regardless of document order (§3.6.2.1).
	//
	// The owning producer is carried for BOTH reasons the two type indexes carry
	// one, because an <attributeGroup> body holds local declarations AND unqualified
	// references: §3.2.2.2 takes a local <attribute>'s {target namespace} from "the
	// ancestor <schema> element information item", which is the DECLARING document's,
	// and src-resolve (§3.17.6.2) clause 4.1.1 scopes the absent-namespace default of
	// its type=/ref= to "the schema document containing the QName", which §F.1 task
	// (b) coerces when that document is a chameleon. Folded under a referring
	// producer instead, localTargetNS would mint the local names in the wrong
	// namespace and unqualifiedRefNS/declares would answer for the wrong document.
	attributeGroups map[xsd.QName]typeSource

	// modelGroups maps each top-level named <group>'s expanded name to its source
	// (raw element plus the producer of the document that declares it), filled by
	// the pre-scan so a <group ref> reaches its definition regardless of document
	// order (§3.1.3: "forward reference to named definitions and declarations is
	// allowed"). Only one mapping rule consults it — §3.4.2.3.3 clause 4.2.3's
	// sub-case test, which must know the {compositor} of the Model Group a
	// <group ref> ·base particle· resolves to (allGroupOf); every OTHER <group ref>
	// stays an unresolved ModelGroupRef until finalize (produceGroupRefParticle).
	//
	// The owning producer is carried for both reasons attributeGroups carries one:
	// a <group> body holds local <element> declarations, whose {target namespace}
	// §3.3.2.3 takes from "the ancestor <schema> element information item" of the
	// DECLARING document, and unqualified type=/ref= references inside it take that
	// document's §F.1 chameleon coercion.
	modelGroups map[xsd.QName]typeSource

	// identityConstraints maps each NAMED <unique>/<key>/<keyref>'s expanded name
	// to its source, filled by the pre-scan so a <key ref="…"> reaches its
	// definition regardless of document order (§3.1.3: "forward reference to named
	// definitions and declarations is allowed, both within and between schema
	// documents"). Unlike the three indexes above it is filled from the WHOLE
	// document tree, not just the top level: §3.17.2 sources a schema's
	// {identity-constraint definitions} from the constraints "anywhere within the
	// [[children]]", so a reference may name one declared on an arbitrarily deep
	// local <element>.
	identityConstraints map[xsd.QName]identityConstraintSource

	// built is the memo + cycle guard for simple-type construction, mirroring
	// xsd/resolve.go's color-map idiom collapsed into one map: an ABSENT key is
	// unstarted, a PRESENT-nil value is on the build stack (being built), and a
	// PRESENT-non-nil value is done. The pre-seeded builtins start out done.
	built map[xsd.QName]*xsd.SimpleType

	// builtComplex is the same memo + cycle guard for COMPLEX-type construction,
	// with the identical tri-state (absent unstarted, present-nil on the build
	// stack, present-non-nil done) so "started but unrecorded" stays
	// unrepresentable (STYLE T1/D3). The pre-seeded xs:anyType starts out done, so
	// <extension base="xs:anyType"> resolves without a special case. The on-stack
	// state is what terminates demand-driven base construction on a circular
	// chain, charged ct-props-correct clause 3 (§3.4.6.1) — the SAME rule
	// xsd/resolve.go's checkComplexBaseAcyclic charges for the programmatic
	// SchemaBuilder path; see buildComplexType.
	builtComplex map[xsd.QName]*xsd.ComplexType

	// builtGroups is the memo + on-stack guard for MODEL GROUP DEFINITION
	// construction, with the same tri-state as built/builtComplex (absent
	// unstarted, present-nil on the build stack, present-non-nil done).
	//
	// The memo half is a correctness requirement, not an optimization: mapping a
	// definition whose body holds a local <element> with a NAMED
	// <unique>/<key>/<keyref> REGISTERS that identity-constraint definition with
	// the builder (produceIdentityConstraint), so mapping the same <group> twice —
	// once on demand for a clause 4.2.3 sub-case test, once at its own
	// document-order position — would register it twice and fabricate a
	// sch-props-correct (§3.17.6.1) clause 2 duplicate-name collision against the
	// very definition it duplicates.
	//
	// The on-stack half is a TERMINATION guard only, never a verdict: its sole
	// reader is resolveModelGroup, which reports an in-progress definition as
	// "does not resolve here" and lets the caller fall through. Rejecting a
	// circular <group ref> graph stays mg-props-correct clause 2's job at finalize
	// (xsd/resolve.go's checkModelGroupsAcyclic), which is where the whole graph is
	// visible; a second rejection path here would be a second encoding of one fact.
	//
	// That state IS reachable, as of #340: a definition body that re-enters
	// complex-type construction now exists — a local <element> with an inline
	// <complexType> — while every other child (a typed local <element>, a
	// wildcard, a nested compositor, a <group ref>) either retains its reference
	// or recurses only within the body. The sentinel is this guard's live
	// termination path for that shape (a <group ref> whose definition body
	// contains a local element whose inline <complexType> re-enters the same
	// <group ref>): it reports "does not resolve here" and lets the caller fall
	// through instead of recursing until the stack dies. It must be WRITTEN
	// regardless, since the memo cannot be filled before the definition exists,
	// so reading it here costs one branch always and stops a real infinite
	// recursion now.
	builtGroups map[xsd.QName]*xsd.ModelGroupDefinition

	// builtIC is the build-once memo for identity-constraint construction. It has
	// NO on-stack sentinel, unlike built/builtComplex: mapping a definition reads
	// only its own <selector>/<field> and retains its refer= as an unresolved
	// QName, so construction never recurses into another definition and there is
	// no circularity to guard (PRINCIPLES 5).
	//
	// The memo is what makes §3.11.2's ref= mapping literal — "the corresponding
	// schema component IS the identity-constraint definition resolved to by the
	// actual value of the ref attribute" — rather than approximate: every
	// reference to a name yields the very component its definition contributed,
	// never a rebuilt twin.
	builtIC map[xsd.QName]xsd.IdentityConstraint

	// backend is the assembly's [value.Backend], retained past the one-time
	// [builtin.Seed] call so constructSimpleType can charge the value-space
	// facet constraints of cos-st-restricts ([builtin.CheckSimpleTypeRestriction])
	// on every simple type it builds. It lives here rather than on the
	// per-document producer for the same reason the indexes do: it is
	// assembly-wide and identical for every document.
	backend value.Backend
}

// typeSource is one entry of symbols.simpleTypes, symbols.complexTypes or
// symbols.attributeGroups: a top-level <simpleType>/<complexType>/
// <attributeGroup> element together with the producer of the document that
// DECLARES it. On-demand construction from a reference runs through owner, never
// through the producer that happens to be asking, so the definition's local
// element and attribute declarations take their own document's target namespace
// and form defaults (§3.3.2.3 dcl.elt.local, §3.2.2.2 dcl.att.local) and its own
// unqualified QName references take their own document's §F.1 coercion — all
// properties of the declaring document, which assembly-wide visibility (§4.2.3
// c-incl-incl) does not transfer to the asker.
type typeSource struct {
	elem  *Element
	owner *producer
}

// identityConstraintSource is one entry of symbols.identityConstraints: a NAMED
// <unique>/<key>/<keyref> element, the {identity-constraint category} its local
// name fixes (§3.11.2), and the producer of the document that DECLARES it. The
// category is carried rather than re-derived at each use because the pre-scan
// already computed it to decide whether to index the element at all (STYLE D3
// cuts the other way here: re-deriving would mint a second, fallible answer to a
// question already settled).
//
// A reference is built through owner, never through the producer that happens to
// be asking, for the reasons typeSource's doc gives: the <selector>/<field>
// XPath Expression records take their {namespace bindings} and {default
// namespace} from the DECLARING document (§3.13.1, and the xpathDefaultNamespace
// chain rooted at that document's <schema>), which assembly-wide visibility does
// not transfer to the asker.
type identityConstraintSource struct {
	elem     *Element
	category xsd.IdentityConstraintCategory
	owner    *producer
}

// newSymbols returns the empty assembly-wide symbol table, having seeded the
// builtin datatypes and xs:anyType into builder — EXACTLY ONCE for the whole
// assembly, which is why seeding lives here and not in the per-document
// producer: seeding per document would add xs:string (and every other builtin)
// once per <include>d document and trip sch-props-correct (§3.17.6.1) clause 2
// on any schema assembled from two or more documents.
//
// xs:anyType is the ur-type Complex Type Definition (§3.4.7). [builtin.Seed]
// yields only simple types (its doc defers anyType to M4 as "a parser-level
// structural concern"), so without it a bare type=-less <element>/<attribute> —
// which defaults to xs:anyType (§3.3.2.1 case 4) — would fail src-resolve at
// finalize. It is added to {type definitions} exactly like a produced complex
// type, so a type= reference to it resolves.
func newSymbols(builder *xsd.SchemaBuilder, backend value.Backend) (*symbols, error) {
	builtins, err := builtin.Seed(backend)
	if err != nil {
		return nil, err
	}
	built := make(map[xsd.QName]*xsd.SimpleType, len(builtins))
	for _, b := range builtins {
		builder.AddType(b)
		built[b.Name()] = b
	}
	anyType, err := seedAnyType()
	if err != nil {
		return nil, err
	}
	builder.AddType(anyType)
	return &symbols{
		simpleTypes:         make(map[xsd.QName]typeSource),
		complexTypes:        make(map[xsd.QName]typeSource),
		attributeGroups:     make(map[xsd.QName]typeSource),
		modelGroups:         make(map[xsd.QName]typeSource),
		identityConstraints: make(map[xsd.QName]identityConstraintSource),
		built:               built,
		builtGroups:         make(map[xsd.QName]*xsd.ModelGroupDefinition),
		builtIC:             make(map[xsd.QName]xsd.IdentityConstraint),
		// xs:anyType is seeded DONE so a derivation naming it resolves to the very
		// component AddType registered, rather than to a rebuilt twin.
		builtComplex: map[xsd.QName]*xsd.ComplexType{anyTypeName: &anyType},
		backend:      backend,
	}, nil
}

// producer is the build context for ONE schema document within an assembly.
// Document order comes solely from walking schemaElem.Children(); the shared
// symbol table holds no order (STYLE D2).
type producer struct {
	schemaElem *Element
	// target is the EFFECTIVE target namespace this document's components are
	// minted in, which is per-DOCUMENT and not necessarily the document's own:
	// under chameleon inclusion (§4.2.3 clause 2.3) a document with no
	// targetNamespace of its own contributes its components to the INCLUDING
	// namespace (§F.1 task a). An <import>ed document is the opposite case — it is
	// always minted in its own namespace, since §4.2.6.2 applies no coercion. The
	// document's own targetNamespace attribute stays readable on schemaElem, so
	// chameleon is derived, never stored twice (STYLE D3).
	target string
	// ov is the ·override pre-processing· in force over this document (§4.2.5,
	// §F.2), nil when it was reached plainly or is the root. It substitutes for
	// this document's OWN top-level source declarations; the substituted
	// declarations are nonetheless produced by THIS producer, so they take this
	// document's target namespace and schema-level defaults, which is what §4.2.5's
	// document-level-defaults note requires (PRINCIPLES 16). The one thing they do
	// NOT take is this document's §F.1 chameleon coercion of unqualified QName
	// references, which §4.2.5 clause 3.2.1 orders BEFORE the substitution — see
	// unqualifiedRefNS.
	ov *overrideSet
	// rd is the ·redefinition· in force over this document (§4.2.4,
	// parser/redefine.go), nil unless it was reached through an <xs:redefine>.
	// Like ov it is a property of the PATH the document was reached by, not of the
	// document (STYLE D3). It EXCEPTS this document's own top-level declarations of
	// the redefined names from contributing components (§4.2.4 clause 4.1.2), and
	// it is where each of them is recorded as the original a self-reference in the
	// redefining document resolves to (src-expredef).
	rd *redefineSet
	// redefines are the sets read from THIS document's own <redefine> children, in
	// document order — the other side of the same values: rd is what some other
	// document's <redefine> did to this one, redefines is what this document's
	// <redefine>s did to others. It is empty on the [Produce] path, which follows
	// no ·inter-schema-document reference·, and that emptiness is what makes a
	// <redefine> skipped there rather than half-applied.
	redefines []*redefineSet
	builder   *xsd.SchemaBuilder
	symbols   *symbols
}

// newProducer returns the build context for one document of an assembly. target
// is THIS document's effective target namespace (see [producer].target), ov the
// override in force over it (nil for none), rd the redefinition in force over it
// (nil for none), redefines the sets read from its own <redefine> children (empty
// for none) and sym the assembly's shared symbol table; all are set at
// construction and never mutated into validity afterward (STYLE T1).
func newProducer(doc *Document, target string, ov *overrideSet, rd *redefineSet, redefines []*redefineSet, builder *xsd.SchemaBuilder, sym *symbols) *producer {
	return &producer{schemaElem: doc.Root(), target: target, ov: ov, rd: rd, redefines: redefines, builder: builder, symbols: sym}
}

// chameleon reports whether this document is produced under chameleon coercion
// (§4.2.3 clause 2.3 / §F.1): it declares no targetNamespace of its own, yet the
// document that <include>d it has one. An <import>ed no-namespace document is
// therefore NOT a chameleon — the assembler discovers it under its own (absent)
// namespace, leaving target empty. It is a derived predicate computed on each
// call, never a stored field (STYLE D3), mirroring [Document.IsSchema].
func (p *producer) chameleon() bool {
	if p.target == "" {
		return false
	}
	_, own := p.schemaElem.Attr("targetNamespace")
	return !own
}

// prescan registers this document's top-level named <simpleType>s and
// <complexType>s (forward base= references, §3.1.3/§3.4.2), named
// <attributeGroup>s (forward <attributeGroup ref> inlining, §3.6.2.1) and named
// <group>s (forward <group ref> resolution for §3.4.2.3.3 clause 4.2.3's
// sub-case test, resolveModelGroup) in the assembly-wide symbol table, building
// nothing yet.
// EVERY document's prescan runs before ANY document's run, so a reference in one
// document reaches a definition in another (§4.2.3 c-incl-incl). Names are
// minted in the effective target namespace, so a chameleon document's
// definitions are registered under the including namespace (§F.1 task a).
//
// A name an <override> in force over this document substitutes for is registered
// with the OVERRIDING declaration (§F.2 clause 1), so a base= or
// <attributeGroup ref> naming it reaches the replacement rather than the
// replaced definition — "overriding components are constructed as if the
// overridden components had never existed" (§4.2.5).
//
// A <redefine> child of this document contributes its OWN children as this
// document's definitions (§4.2.4 clause 4.1.1), so they are registered too — see
// prescanRedefine. In the other direction, a name a <redefine> in force over
// this document replaces is NOT registered here: §4.2.4 clause 4.1.2 excepts it
// from the components this document contributes, and it is recorded instead as
// the original the redefining declaration's self-reference resolves to
// (src-expredef clauses 1.1 and 2).
//
// It also registers this document's named <unique>/<key>/<keyref>s for forward
// <key ref="…"> resolution, but from the WHOLE subtree of each top-level
// declaration rather than from its top level: see prescanIdentityConstraints.
func (p *producer) prescan() {
	for _, child := range p.schemaElem.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if isXSD(el, "redefine") {
			p.prescanRedefine(el)
			continue
		}
		decl := p.ov.replacement(el)
		if !compositionDirective(el) {
			p.prescanIdentityConstraints(decl)
		}
		name, ok := el.Attr("name")
		if !ok {
			continue
		}
		if p.rd.excepts(el) {
			// §4.2.4 clause 4.1.2: this declaration is explicitly redefined, so it
			// contributes no component under its own name. It survives as the hidden
			// original src-expredef pairs the redefining declaration with, recorded
			// under THIS producer so its body is still mapped in its own document's
			// context.
			p.rd.recordOriginal(componentKey{kind: el.Name().Local(), name: name}, typeSource{elem: decl, owner: p})
			continue
		}
		switch {
		case isXSD(el, "simpleType"):
			p.symbols.simpleTypes[xsd.QName{Space: p.target, Local: name}] = typeSource{elem: decl, owner: p}
		case isXSD(el, "complexType"):
			p.symbols.complexTypes[xsd.QName{Space: p.target, Local: name}] = typeSource{elem: decl, owner: p}
		case isXSD(el, "attributeGroup"):
			p.symbols.attributeGroups[xsd.QName{Space: p.target, Local: name}] = typeSource{elem: decl, owner: p}
		case isXSD(el, "group"):
			p.symbols.modelGroups[xsd.QName{Space: p.target, Local: name}] = typeSource{elem: decl, owner: p}
		}
	}
}

// prescanIdentityConstraints registers every NAMED <unique>/<key>/<keyref> in
// el's subtree, so a <key ref="…"> resolves whatever the document order and
// whichever document of the assembly declares the target (§3.1.3). It descends
// the whole subtree because §3.17.2 sources a schema's {identity-constraint
// definitions} from "all the <key>, <keyref>, and <unique> element information
// items ANYWHERE within the [[children]]" — a constraint on a local <element>
// nested arbitrarily deep in a content model is as referenceable as a top-level
// declaration's.
//
// The ref= form is deliberately NOT indexed: it declares nothing (§3.11.2), so
// a reference chain is unrepresentable and there is no cycle to guard
// (PRINCIPLES 5). A name is minted in the effective target namespace, exactly as
// produceIdentityConstraint mints the definition's own {name}, so the index key
// and the component name agree under chameleon coercion (§F.1 task a).
//
// The walk is confined to what THIS producer actually maps, by two exclusions.
// prescan withholds the composition directives' subtrees (compositionDirective),
// whose contents belong to another document's producer. And the walk below
// withholds every <annotation> subtree, entering neither it nor anything under
// it. <appinfo> and <documentation> hold mixed, processContents="lax" content
// (§A), and §3 is explicit that "neither the correspondences described nor the
// XML Representation Constraints apply to elements in the Schema namespace which
// occur as descendants of <appinfo> or <documentation>": a <key name="…"> there
// is prose — an illustration, possibly truncated — and is mapped to no component
// by anyone. Indexing it would make the index a strict SUPERSET of
// {identity-constraint definitions}, the very property src-resolve clause 1.7
// looks a ref= up in, letting prose shadow a real same-named definition or
// satisfy a ref= that resolves to nothing. The guard covers foreign-namespace
// elements too: in a schema document they occur only inside annotations.
func (p *producer) prescanIdentityConstraints(el *Element) {
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok {
			continue
		}
		if isXSD(c, "annotation") {
			continue
		}
		p.prescanIdentityConstraints(c)
		if c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		category, ok := identityConstraintCategoryOf(c.Name().Local())
		if !ok {
			continue
		}
		name, ok := c.Attr("name")
		if !ok {
			continue
		}
		p.symbols.identityConstraints[xsd.QName{Space: p.target, Local: name}] =
			identityConstraintSource{elem: c, category: category, owner: p}
	}
}

// compositionDirective reports whether el is one of the four <schema> children
// whose subtree the identity-constraint pre-scan must NOT walk from here.
//
// Three of them — <include>, <import>, <override> — contribute no component of
// their own at all (§4.2.3, §4.2.6.2, §4.2.5), only the components of the
// document they name, and run skips all three: an <override>'s children are
// top-level declarations of the OVERRIDDEN document (§F.2 clause 1) and are
// produced, with their own target namespace and their own <schema> defaults, by
// that document's producer, which pre-scans them itself through
// overrideSet.replacement. The fourth, <redefine>, is the exception in both
// respects: its children ARE definitions of this document (§4.2.4 clause 4.1.1),
// so run produces them and prescanRedefine walks them — which is why prescan
// handles <redefine> before consulting this predicate at all, and why the
// predicate is not the same question as "does run skip it".
func compositionDirective(el *Element) bool {
	return isXSD(el, "include") || isXSD(el, "import") || isXSD(el, "override") || isXSD(el, "redefine")
}

// run walks the <schema> children in strict document order, producing each
// in-scope top-level declaration into the shared builder. prescan must already
// have run — for every document of the assembly, not just this one.
//
// Each child is first passed through the override in force over this document
// (§F.2 clauses 1 and 2): a top-level source declaration an <override> names is
// produced from the OVERRIDING declaration in the overridden one's place, and
// every other child is produced as written. The substitution never changes the
// element TYPE — it is half of the matching key — so the switch below is the same
// whichever declaration is produced.
//
// A declaration a <redefine> in force over this document replaces is skipped
// outright (§4.2.4 clause 4.1.2): the redefining declaration is produced by the
// REDEFINING document's producer, in its own document-order position there, and
// this one survives only as the hidden original behind it (src-expredef).
//
// The five named kinds whose {name} the schema for schema documents makes
// use="required" — <complexType>, <group>, <attributeGroup>, <element>,
// <attribute> — take it from topLevelName, which rejects an unusable one before
// any of them is built.
func (p *producer) run() error {
	for _, child := range p.schemaElem.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if p.rd.excepts(el) {
			continue
		}
		decl := p.ov.replacement(el)
		switch decl.Name().Local() {
		case "simpleType":
			name, _ := decl.Attr("name")
			st, err := p.buildSimpleType(xsd.QName{Space: p.target, Local: name}, decl)
			if err != nil {
				return err
			}
			p.builder.AddType(st)
		case "element":
			name, err := p.topLevelName(decl)
			if err != nil {
				return err
			}
			ed, err := p.produceElement(name, decl)
			if err != nil {
				return err
			}
			p.builder.AddElement(ed)
		case "attribute":
			name, err := p.topLevelName(decl)
			if err != nil {
				return err
			}
			ad, err := p.produceAttribute(name, decl)
			if err != nil {
				return err
			}
			p.builder.AddAttribute(ad)
		case "complexType":
			name, err := p.topLevelName(decl)
			if err != nil {
				return err
			}
			// AddType happens HERE, at this type's own document-order position, and
			// never as a side effect of an on-demand base build from some other
			// type: buildComplexType populates the memo only, so a type built early
			// to serve a derivation still enters {type definitions} in document
			// order (STYLE D2).
			ct, err := p.buildComplexType(name, decl)
			if err != nil {
				return err
			}
			p.builder.AddType(ct)
		case "attributeGroup":
			name, err := p.topLevelName(decl)
			if err != nil {
				return err
			}
			ag, err := p.buildAttributeGroup(name, decl)
			if err != nil {
				return err
			}
			p.builder.AddAttributeGroup(ag)
		case "group":
			name, err := p.topLevelName(decl)
			if err != nil {
				return err
			}
			// AddModelGroup happens HERE, at this definition's own document-order
			// position, and never inside the on-demand build a clause 4.2.3 sub-case
			// test triggers (buildModelGroupDefinition populates the memo only), so
			// {model group definitions} stays in document order (STYLE D2).
			mgd, err := p.buildModelGroupDefinition(name, decl)
			if err != nil {
				return err
			}
			p.builder.AddModelGroup(mgd)
		case "include", "import", "override":
			// Consumed by the assembler (parse.go) during discovery, BEFORE any
			// document is produced: none contributes a component of its own
			// (§4.2.3, §4.2.5, §4.2.6.2), only the components of the document it
			// names, which reach the builder through that document's own producer.
			// [Produce], which follows no inter-document reference, skips them.
		case "redefine":
			// The one composition directive that DOES contribute components of its
			// own: its children are definitions of this document (§4.2.4 clause
			// 4.1.1, src-expredef). The document it names is still the assembler's
			// to discover, and reaches the builder through its own producer minus
			// the declarations this <redefine> excepts (clause 4.1.2).
			if err := p.produceRedefine(decl); err != nil {
				return err
			}
		case "notation":
			n, err := p.produceNotation(decl)
			if err != nil {
				return err
			}
			p.builder.AddNotation(n)
		default:
			// annotation, defaultOpenContent, … — not this slice's scope
			// (§3.1.2), skipped, not invalid.
		}
	}
	return nil
}

// topLevelName expands the name attribute of a top-level <complexType>,
// <group>, <attributeGroup>, <element> or <attribute> into this document's
// target namespace (§3.17.2: a top-level declaration's {target namespace} is
// the <schema>'s), rejecting one that cannot serve as a {name} at all.
//
// The rejection is a plain grammar fault, not an xsderr rule verdict, and it is
// the same fault for all five kinds. The schema for schema documents makes name
// use="required" with type xs:NCName on xs:topLevelComplexType, xs:namedGroup,
// xs:namedAttributeGroup, xs:topLevelElement and xs:topLevelAttribute, so an
// absent attribute and an empty one are equally unusable — which is why the
// presence flag is deliberately discarded rather than branched on. No numbered
// Schema Representation Constraint states a clause of its own for it: §3.4.3
// src-ct incorporates the schema for schema documents by reference over its own
// five clauses (none about name), and §3.6.3 src-attribute_group is literally
// "None as such". Charging src-ct, e-props-correct or a-props-correct here
// would be a fabricated verdict (STYLE E2); this is the footing <include> with
// no schemaLocation already stands on (parse.go).
//
// For the two DECLARATION kinds the parser-level rejection is deliberate
// defense in depth rather than the only enforcement: xsd.NewElementDeclaration
// and xsd.NewAttributeDeclaration independently reject an empty {name} citing
// e-props-correct clause 1 (§3.3.6.1) and a-props-correct clause 1 (§3.2.6.1),
// each through its component's property tableau, where {name} is a Required
// xs:NCName. That verdict stays theirs and is still what the LOCAL paths
// produce; this one supersedes it only for a top-level declaration, where the
// fault belongs to the schema document's grammar and must be reported without
// having built anything.
//
// Rejecting here, in run's dispatch, is what makes the verdict
// CONTENT-INDEPENDENT: every one of the five kinds is judged before a single
// child of it is walked, so a nameless declaration cannot be judged by whether
// its content happens to hold a local element (whose own construction would
// otherwise charge an unrelated rule, and only sometimes) — the defect #206
// found for two of the five and this closes for all five.
//
// The other two top-level named kinds do not come through here. <notation> is
// covered where it is built: xsd.NewNotation rejects an empty {name} citing
// n-props-correct (§3.14.6), which a nameless <notation> document already
// produces end to end.
//
// GAP(parser): a top-level <simpleType> is the one kind still outside this
// helper — run expands its name inline and xsd.NewSimpleType has no {name}
// guard of its own, so a <simpleType> with an absent or empty name is currently
// registered under QName{target, ""} and the document produces without error.
// #305 scoped itself to the five kinds named above; what a schema so registered
// goes on to do downstream is not established here, so no error direction is
// claimed for it.
func (p *producer) topLevelName(decl *Element) (xsd.QName, error) {
	name, _ := decl.Attr("name")
	if name == "" {
		return xsd.QName{}, fmt.Errorf("parser: top-level <%s> at %s has no usable name: its name attribute is absent or empty, and the schema for schema documents requires an xs:NCName", decl.Name().Local(), decl.Loc())
	}
	return xsd.QName{Space: p.target, Local: name}, nil
}

// buildSimpleType returns the compiled simple type named name, building it (and
// its base chain) on demand with memoization and a cycle guard. name is the zero
// QName only via constructSimpleType for anonymous inline types, which never
// enter this memoized path.
func (p *producer) buildSimpleType(name xsd.QName, elem *Element) (*xsd.SimpleType, error) {
	if st, started := p.symbols.built[name]; started {
		if st != nil {
			return st, nil
		}
		// PRESENT-nil: name is on the current build stack — a circular base chain.
		return nil, xsderr.New(ruleSTPropsCorr, elem.Loc(),
			"circular simple type definition: %s derives ultimately from itself, but st-props-correct clause 2 requires every simple type derive from xs:anySimpleType", name)
	}
	p.symbols.built[name] = nil // mark on-stack

	st, err := p.constructSimpleType(name, elem)
	if err != nil {
		return nil, err
	}
	p.symbols.built[name] = st // replace the on-stack sentinel with the finished node
	return st, nil
}

// buildComplexType returns the compiled complex type named name, building it
// (and, through resolveBaseType, its base chain) on demand with memoization and
// a cycle guard — the complex-type twin of buildSimpleType, for the same reason:
// §3.4.2's mapping rules "depend upon the {base type definition} having been
// identified before they apply", and xsd.NewComplexType demands a complete
// {content type} at construction, so the base COMPONENT must exist before the
// derived type can be built at all.
//
// It is the single entry point for a NAMED complex type: run's top-level
// dispatch and resolveBaseType's on-demand construction both go through it, so a
// named type is mapped exactly once. It populates the memo only — registering
// the component with the builder is run's job, at the type's own document-order
// position.
//
// An inline ANONYMOUS <complexType> deliberately does NOT come through here: it
// calls produceComplexType directly (produceElement, produceLocalElement).
// Neither of this function's two services applies to it — the memo is keyed by
// name and an anonymous type has none, and it can be no cycle member for exactly
// the same reason, since a {base type definition} chain is followed BY NAME and
// nothing can name it (STYLE 5 / PRINCIPLES 5: no cycle check where the
// construction order makes one impossible).
//
// A name already on the build stack (the PRESENT-nil memo state) is a circular
// {base type definition} chain, charged ct-props-correct clause 3 (§3.4.6.1).
// That is the SAME rule, with the same verdict, that xsd/resolve.go's
// checkComplexBaseAcyclic charges: two entry points on one rule for the two
// construction paths — this one for the producer, whose demand-driven recursion
// would otherwise not terminate, and that one for the programmatic
// SchemaBuilder, which has no producer and must stay self-defending. Neither
// substitutes for the other (PRINCIPLES 5's "detect once at construction" applies
// per construction path).
func (p *producer) buildComplexType(name xsd.QName, elem *Element) (xsd.ComplexType, error) {
	if ct, started := p.symbols.builtComplex[name]; started {
		if ct != nil {
			return *ct, nil
		}
		return xsd.ComplexType{}, xsderr.New(ruleCTPropsCorr, elem.Loc(),
			"circular complex type definition: %s derives ultimately from itself, but ct-props-correct clause 3 forbids a circular {base type definition} chain (only xs:anyType may be its own base)", name)
	}
	p.symbols.builtComplex[name] = nil // mark on-stack

	ct, err := p.produceComplexType(namedComplexType(name), elem)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	p.symbols.builtComplex[name] = &ct // replace the on-stack sentinel with the finished node
	return ct, nil
}

// resolveBaseType identifies the {base type definition} COMPONENT a base=
// attribute names (§3.4.2 preamble), building it on demand when it is a
// not-yet-mapped complex or simple type of this assembly. at is the
// <restriction>/<extension> carrying the base=, charged for a failure.
//
// A base of either variety is built through its OWN document's producer
// (typeSource's owner), never through p: see symbols.simpleTypes and
// symbols.complexTypes. An already-built simple base returns the very
// *xsd.SimpleType symbols.built holds — never a rebuilt twin, whose component
// identity would silently diverge (xsd/typedefinition.go); routing through the
// owner does not change that, since the memo is assembly-wide and every producer
// of the assembly shares it.
//
// A name that resolves to no type at all is charged src-resolve clause 1.1
// (§3.17.6.2) — the same rule, at the same clause, that resolveBase charges for
// a simple type's base and that finalize charges for an unresolvable
// {base type definition} reference (xsd/resolve.go's resolveTypeName). One rule,
// three entry points, identical verdict.
func (p *producer) resolveBaseType(at *Element, name xsd.QName) (xsd.TypeDefinition, error) {
	if ct, done := p.symbols.builtComplex[name]; done && ct != nil {
		return *ct, nil
	}
	if src, ok := p.symbols.complexTypes[name]; ok {
		// Unbuilt or on-stack: buildComplexType handles the memo hit and the
		// ct-props-correct clause 3 cycle rejection alike.
		ct, err := src.owner.buildComplexType(name, src.elem)
		if err != nil {
			return nil, err
		}
		return ct, nil
	}
	if st, ok := p.symbols.built[name]; ok && st != nil {
		return st, nil
	}
	if src, ok := p.symbols.simpleTypes[name]; ok {
		return src.owner.buildSimpleType(name, src.elem)
	}
	return nil, xsderr.New(ruleSrcResolve, at.Loc(),
		"base type %s does not resolve to any type definition in scope (src-resolve clause 1.1)", name)
}

// buildModelGroupDefinition returns the Model Group Definition named name,
// building it on demand with memoization — the model-group twin of
// buildComplexType, and like it the SINGLE entry point for mapping a top-level
// named <group>: run's document-order dispatch and resolveModelGroup's
// demand-driven resolution both go through it, so one <group> is mapped exactly
// ONCE. That is a correctness requirement here, not a saving — see
// symbols.builtGroups for the duplicate identity-constraint registration a second
// mapping would fabricate. It populates the memo only; registering the component
// with the builder is run's job.
//
// Unlike buildComplexType it charges no cycle rejection of its own. It WRITES the
// on-stack sentinel, but the reader that makes a circular <group ref> graph
// terminate is resolveModelGroup — the only caller re-enterable for a name still
// being built — and the REJECTION stays mg-props-correct clause 2's at finalize,
// where the whole graph is visible. A name therefore arrives here either unstarted
// or done, never on-stack: run is not re-entrant, and resolveModelGroup answers
// "does not resolve" for an on-stack name rather than calling in.
func (p *producer) buildModelGroupDefinition(name xsd.QName, el *Element) (xsd.ModelGroupDefinition, error) {
	if mgd := p.symbols.builtGroups[name]; mgd != nil {
		return *mgd, nil
	}
	p.symbols.builtGroups[name] = nil // mark on-stack

	mgd, err := p.produceModelGroupDefinition(name, el)
	if err != nil {
		return xsd.ModelGroupDefinition{}, err
	}
	p.symbols.builtGroups[name] = &mgd // replace the on-stack sentinel with the finished node
	return mgd, nil
}

// resolveModelGroup identifies the Model Group a <group ref> resolves its
// particle's {term} to (§3.7.2, xr.mgd3: "the {model group} of the model group
// definition ·resolved· to by the ·actual value· of the ref attribute"), building
// that definition on demand through its OWN document's producer for the reasons
// typeSource's doc gives.
//
// It is deliberately NOT the general <group ref> resolution — that stays
// finalize's (produceGroupRefParticle keeps the reference deferred). Its one
// caller is allGroupOf, whose §3.4.2.3.3 clause 4.2.3 sub-case test must know the
// {compositor} behind a ·base particle· or ·effective content· BEFORE the
// {content type} is synthesized, which is at produce time and nowhere else.
//
// ok is false, with no error, for the two states in which no Model Group is
// (yet) knowable here, both of which leave the caller to fall through to a
// sub-case that assumes nothing:
//
//   - name matches no top-level <group> of the assembly. A dangling <group ref>
//     is charged src-resolve clause 1.5 at finalize, against the retained
//     ModelGroupRef; anticipating that verdict here would be a second encoding.
//   - name is already on the build stack, i.e. the reference closes a
//     <group ref> cycle. mg-props-correct clause 2 rejects that at finalize; this
//     function must only terminate, never reject (see buildModelGroupDefinition).
//     No schema reaches that branch today — see symbols.builtGroups for why, and
//     for why the branch is kept rather than left to the stack.
func (p *producer) resolveModelGroup(name xsd.QName) (xsd.ModelGroup, bool, error) {
	if mgd, started := p.symbols.builtGroups[name]; started {
		if mgd == nil {
			return xsd.ModelGroup{}, false, nil // PRESENT-nil: on the build stack
		}
		return mgd.ModelGroup(), true, nil
	}
	src, ok := p.symbols.modelGroups[name]
	if !ok {
		return xsd.ModelGroup{}, false, nil
	}
	mgd, err := src.owner.buildModelGroupDefinition(name, src.elem)
	if err != nil {
		return xsd.ModelGroup{}, false, err
	}
	return mgd.ModelGroup(), true, nil
}

// constructSimpleType maps one <simpleType> element (named or anonymous) into a
// component: it reads the single <restriction> child, resolves the base, maps
// the own facets, and constructs. It does NOT memoize — the memo/cycle bookkeeping
// lives in buildSimpleType; an anonymous inline type has no name to key on and is
// unreferenceable, so it is built here directly, once.
//
// The finished component is then charged with the facet-VALUE sub-clauses of
// cos-st-restricts (§3.16.6.2) through [builtin.CheckSimpleTypeRestriction] —
// facet applicability against the primitive, and the bound/enumeration
// constraints in the base type's value space. That check needs both the builtin
// applicability table and a [value.Backend], neither of which package xsd may
// depend on, so it cannot live inside xsd.NewSimpleType and runs here instead,
// at this package's SOLE NewSimpleType call site. A rejection is returned as-is:
// it is already an *xsderr.Error carrying the specific per-facet rule.
func (p *producer) constructSimpleType(name xsd.QName, elem *Element) (*xsd.SimpleType, error) {
	restriction, err := restrictionOf(elem)
	if err != nil {
		return nil, err
	}
	base, err := p.resolveBase(restriction)
	if err != nil {
		return nil, err
	}
	facets, err := p.restrictionFacets(restriction)
	if err != nil {
		return nil, err
	}
	// {variety} of a restriction is the {variety} of its base (§3.16.2.1). Reusing
	// base.Variety() propagates the base's own {primitive type definition} pointer
	// for an atomic base (warden finding #4), and the item/member pointers for a
	// list/union base.
	st, err := xsd.NewSimpleType(elem.Loc(), name, base.Variety(), base, facets, nil)
	if err != nil {
		return nil, err
	}
	if err := builtin.CheckSimpleTypeRestriction(p.symbols.backend, st); err != nil {
		return nil, err
	}
	return st, nil
}

// restrictionOf returns the single <restriction> child of a <simpleType>. A
// <simpleType> using <list> or <union> instead has no <restriction> child; that
// is rejected explicitly (never silently skipped), since this slice only
// implements the restriction case (§3.16.3 src-simple-type governs the required
// <restriction>|<list>|<union> shape).
func restrictionOf(elem *Element) (*Element, error) {
	if r := childElement(elem, xsd.XMLSchemaNS, "restriction"); r != nil {
		return r, nil
	}
	return nil, xsderr.New(ruleSrcSimpleType, elem.Loc(),
		"simpleType has no <restriction> child; this producer does not yet support <list> or <union> simple types")
}

// resolveBase resolves a <restriction>'s {base type definition} to a live
// *SimpleType. It enforces src-simple-type clause 2 (§3.16.3): exactly one of a
// base= attribute or an inline <simpleType> child, never both, never neither. A
// base= is discharged EARLY here — unlike element/attribute type=, which defers
// to finalize — because NewSimpleType demands a live base pointer at construction.
func (p *producer) resolveBase(restriction *Element) (*xsd.SimpleType, error) {
	baseLex, hasBase := restriction.Attr("base")
	inline := childElement(restriction, xsd.XMLSchemaNS, "simpleType")

	if hasBase && inline != nil {
		return nil, xsderr.New(ruleSrcSimpleType, restriction.Loc(),
			"restriction has both a base attribute and an inline <simpleType> child, but src-simple-type clause 2 allows only one")
	}
	if !hasBase && inline == nil {
		return nil, xsderr.New(ruleSrcSimpleType, restriction.Loc(),
			"restriction has neither a base attribute nor an inline <simpleType> child, but src-simple-type clause 2 requires exactly one")
	}

	if inline != nil {
		// Anonymous base: built inline, once, with an absent {name} (zero QName).
		return p.constructSimpleType(xsd.QName{}, inline)
	}

	qn, err := p.resolveQName(restriction, baseLex)
	if err != nil {
		return nil, err
	}
	if src, redefining := p.redefinedTypeBase(restriction, qn); redefining {
		// src-expredef clause 1.1/1.2: this <restriction> is a redefining
		// <simpleType>'s own, and its base names that type itself, so it resolves to
		// the ORIGINAL — "one component which corresponds to the top-level definition
		// item with the same name in the <redefine>d schema document … except that its
		// {name} is ·absent·". Built here, once, with the zero QName and under the
		// REDEFINED document's producer, so it enters no symbol table, is registered
		// with no builder, and takes its own document's namespace and defaults. That
		// is what keeps the redefinition an ordinary restriction rather than a
		// self-derivation (st-props-correct clause 2).
		return src.owner.constructSimpleType(xsd.QName{}, src.elem)
	}
	// Pre-seeded builtins and already-finished locals resolve directly.
	if st, ok := p.symbols.built[qn]; ok && st != nil {
		return st, nil
	}
	// An assembly-visible one (unbuilt or on-stack) recurses through its OWN
	// document's producer, never p — see symbols.simpleTypes; buildSimpleType
	// handles memo hit and cycle rejection.
	if src, ok := p.symbols.simpleTypes[qn]; ok {
		return src.owner.buildSimpleType(qn, src.elem)
	}
	return nil, xsderr.New(ruleSrcResolve, restriction.Loc(),
		"base type %s does not resolve to any simple type in scope (src-resolve clause 1.1)", qn)
}

// restrictionFacets maps the constraining-facet children of a <restriction> in
// document order. The plain-lexical facets map one-to-one, with two folding
// exceptions, each landing at the position of its kind's FIRST child element so
// the returned slice stays in document order (STYLE D2):
//
//   - every <assertion> child (Datatypes §4.3.13.2) folds into the SINGLE
//     assertions facet the §4.3.13 {value} rule describes — "a sequence of
//     Assertion components";
//   - every <pattern> child folds into the SINGLE pattern facet xr-pattern
//     (§4.3.4.2) describes, one {value} member per sibling, in document order.
//
// enumeration needs a richer sub-shape and is not yet produced: rather than
// silently dropping a constraint (a false-accept), an actual <enumeration> child
// is rejected. The non-facet children <annotation> and the inline base
// <simpleType> are skipped.
func (p *producer) restrictionFacets(restriction *Element) ([]xsd.Facet, error) {
	var facets []xsd.Facet
	var assertions []xsd.Assertion
	assertionsAt := 0
	var patterns []string
	patternAt := 0
	for _, child := range restriction.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		local := el.Name().Local()
		if local == "annotation" || local == "simpleType" {
			continue
		}
		if local == "enumeration" {
			return nil, xsderr.New(ruleSrcSimpleType, el.Loc(),
				"restriction has an <enumeration> facet, which this producer does not yet support; refusing to silently drop it")
		}
		if local == "assertion" {
			if len(assertions) == 0 {
				assertionsAt = len(facets)
			}
			assertions = append(assertions, xsd.NewAssertion(p.buildXPathExpression(el, "test"), nil))
			continue
		}
		kind, ok := facetKindOf(local)
		if !ok {
			continue
		}
		val, _ := el.Attr("value")
		if kind == xsd.FacetPattern {
			// xr-pattern (§4.3.4.2): all the <pattern> children of ONE <restriction>
			// contribute BRANCHES of a single regular expression — patterns on the
			// same derivation step are ORed, only patterns on DIFFERENT steps are
			// ANDed (cvc-pattern-valid §4.3.4.4, one check per surviving facet). One
			// facet per sibling would carry cross-step meaning: two same-kind
			// ownFacets, which st-props-correct clause 4 rejects outright, so a
			// two-<pattern> restriction was unconstructible. The pattern facet has no
			// {fixed} property, and <pattern> has no fixed attribute (dc-pattern
			// §4.3.4.1, element-pattern §4.3.4.2).
			patterns = append(patterns, val)
			folded := xsd.NewFacet(kind, patterns, false)
			if len(patterns) == 1 {
				patternAt = len(facets)
				facets = append(facets, folded)
				continue
			}
			facets[patternAt] = folded
			continue
		}
		fixed, err := facetFixed(el)
		if err != nil {
			return nil, err
		}
		facets = append(facets, xsd.NewFacet(kind, []string{val}, fixed))
	}
	if len(assertions) == 0 {
		return facets, nil
	}
	return slices.Insert(facets, assertionsAt, xsd.NewAssertionsFacet(assertions)), nil
}

// produceElement maps a top-level <element> into a global Element Declaration
// (§3.3.2.2 dcl.elt.global), including its {identity-constraint definitions}
// (§3.3.2.1). Registering the produced identity constraints with the schema
// builder is the caller's job (run), keeping this one focused on building the
// declaration.
//
// qname is the expanded {name}/{target namespace}, taken from topLevelName by
// the single caller (run) rather than read from the element here, so an
// unusable name is a grammar fault charged before any of this mapping runs.
//
// Its {type definition} is §3.3.2.1 dcl.elt.common's tier chain, which is a
// COMMON mapping rule — §3.3.2.2 supplements only {scope} and {target
// namespace}, never {type definition} — so tier 1's inline <complexType> child
// is mapped here exactly as produceLocalElement maps it, through one minted
// xsd.ComponentID that serves as both the anonymous type's {context} (§3.4.2.1
// dcl.ctd.common) and the {scope}.{parent} of its own nested local elements
// (#340). The two paths differ only in the {scope} the declaration itself gets.
//
// Tier 1's inline <simpleType> child is the one form still declined on this
// path: #229 widened §3.2.2.2/§3.3.2.3's LOCAL mapping only, and the global
// simple-type widening is a separate, adjacent change. It is declined with a
// plain "not yet produced" error, never a fabricated src-element verdict — the
// schema is legal and it is this producer that is incomplete (STYLE E2) — and
// conformance/schema.go's elementDecidable declines the shape so the limitation
// never reaches a validity verdict.
func (p *producer) produceElement(qname xsd.QName, elem *Element) (xsd.ElementDeclaration, error) {
	typeLex, hasType := elem.Attr("type")
	inlineSimple := childElement(elem, xsd.XMLSchemaNS, "simpleType")
	inlineComplex := childElement(elem, xsd.XMLSchemaNS, "complexType")

	if hasType && (inlineSimple != nil || inlineComplex != nil) {
		return xsd.ElementDeclaration{}, xsderr.New(ruleSrcElement, elem.Loc(),
			"element has both a type attribute and an inline <simpleType>/<complexType> child, but src-element clause 3 forbids both")
	}
	if err := rejectBothInlineTypes(elem, inlineSimple, inlineComplex); err != nil {
		return xsd.ElementDeclaration{}, err
	}
	// The unproduced form is declined BEFORE anything else is mapped, so the
	// limitation never depends on what the rest of the declaration happens to
	// hold (the same ordering discipline topLevelName's name check keeps).
	if inlineSimple != nil {
		return xsd.ElementDeclaration{}, fmt.Errorf("parser: a top-level <element> at %s with an inline <simpleType> is not yet produced (§3.3.2.1 dcl.elt.common clause 1; the local form is)", elem.Loc())
	}

	vc, err := valueConstraintOf(elem, ruleSrcElement)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}

	typeName := xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"} // §3.3.2.1 case 4
	if hasType {
		typeName, err = p.resolveQName(elem, typeLex)
		if err != nil {
			return xsd.ElementDeclaration{}, err
		}
	}
	constraints, err := p.identityConstraintsOf(elem)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	affiliations, err := p.substitutionGroupAffiliations(elem)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	// §3.3.2.2 dcl.elt.global: {scope} is {variety} global, {parent} ·absent·.
	if inlineComplex != nil {
		edID := xsd.NewComponentID()
		ct, err := p.produceComplexType(anonymousComplexType(edID), inlineComplex)
		if err != nil {
			return xsd.ElementDeclaration{}, err
		}
		return xsd.NewElementDeclarationOwningType(elem.Loc(), edID, qname, ct, nil, xsd.NewGlobalScope(), vc,
			false, constraints, affiliations, nil, false, p.disallowedSubstitutions(elem), nil)
	}
	return xsd.NewElementDeclaration(elem.Loc(), qname, xsd.TypeDefinitionRef{Name: typeName}, nil, xsd.NewGlobalScope(), vc,
		false, constraints, affiliations, nil, false, p.disallowedSubstitutions(elem), nil)
}

// substitutionGroupAffiliations maps the substitutionGroup attribute of a
// top-level <element> into {substitution group affiliations} (§3.3.2.1
// dcl.elt.common: "A set of the element declarations ·resolved· to by the items
// in the ·actual value· of the substitutionGroup attribute, if present, otherwise
// the empty set").
//
// The attribute is typed `List of QName` (§3.3.2), so XSD 1.1 permits SEVERAL
// heads, and EVERY item contributes — unlike {type definition} clause 3, which
// reads the first item alone (see localDeclaredType's GAP marker). Items are
// resolved and returned in LEXICAL order and the property is carried as a slice,
// not a set (STYLE D2): it reaches users through the cos-nonambig and
// cos-element-consistent diagnostics, and no map is ranged anywhere on the path.
//
// Only the prefix→namespace-name half of ·resolved· happens here, through the one
// in-scope-bindings resolver (STYLE T4), which also applies §F.1 task (b)'s
// chameleon coercion to an unqualified item. Existence is not checked here at
// all, and — unlike every other by-name reference this producer emits — it is not
// checked at finalize either: a head naming no declaration is an ·absent· member
// under §5.3 (Missing Sub-components), which makes the schema valid and only its
// USE invalid. See xsd/resolve.go's resolveElementDecl for that decision; the
// consequence here is that this function never fails on an unknown head, only on
// a lexically unresolvable prefix.
//
// It is called only from produceElement. The local path never builds
// affiliations: the INLINE local form is rejected for carrying the attribute at
// all (e-props-correct clause 3, produceLocalElement), and the <element ref>
// form returns from produceElementParticle before that check, ignoring the
// attribute instead of rejecting it (see the GAP marker on that branch).
//
// {substitution group exclusions} — the final=/finalDefault twin — is deliberately
// NOT mapped alongside it. The property is read by exactly one rule,
// e-props-correct clause 4 (c-vs-sg), which this package does not implement and
// which xsd/substitutiongroup.go records as unimplemented; mapping a property no
// constraint consults would add a fact with no reader.
func (p *producer) substitutionGroupAffiliations(elem *Element) ([]xsd.QName, error) {
	lexical, ok := elem.Attr("substitutionGroup")
	if !ok {
		return nil, nil
	}
	var heads []xsd.QName
	for _, item := range strings.Fields(lexical) {
		head, err := p.resolveQName(elem, item)
		if err != nil {
			return nil, err
		}
		heads = append(heads, head)
	}
	return heads, nil
}

// disallowedSubstitutions maps an <element>'s block attribute into {disallowed
// substitutions} (§3.3.2.1 dcl.elt.common). The ·effective block value· is the
// block attribute if present, otherwise the ancestor <schema>'s blockDefault if
// present, otherwise the empty string; the empty string maps to the empty set,
// "#all" to all three keywords, and any other value to the keywords its
// whitespace-separated list names. Values outside {extension, restriction,
// substitution} are IGNORED rather than rejected — the mapping table's own Note
// says blockDefault "may include values other than extension, restriction or
// substitution" and that "those values are ignored in the determination of
// {disallowed substitutions} for element declarations".
//
// It is mapped in THIS slice, beside {substitution group affiliations}, because
// the two are one fact operationally: {disallowed substitutions} is read by
// cos-equiv-derived-ok-rec clause 2.1 (§3.3.6.3), the clause that decides whether
// a head admits substitution at all, so populating the affiliation edges while
// leaving the property universally empty would publish an OVER-BROAD
// ·substitution group· — every declared head would admit every member, which
// false-rejects valid schemas through cos-nonambig's ·overlap· relation. W3C
// MS-Element elemZ028a is exactly that shape: an affiliation chain a→b→c→d in
// which c and d each carry block="substitution", with b, c and d then named side
// by side in one <xs:all>. Half the pair is not a smaller change, it is a wrong
// one.
//
// The result is returned in the spec's own canonical order — extension,
// restriction, substitution, the order the table's case 2 writes the set in —
// rather than in the attribute's lexical order. The property IS a set, its
// members are drawn from that fixed three-element set, and a canonical order is
// deterministic for every input spelling (STYLE D2), which a lexical one is not:
// block="restriction extension" and block="extension restriction" denote the same
// set and must not produce two different components.
//
// {substitution group exclusions} has no such mapping here — see
// substitutionGroupAffiliations for why final= is left unmapped.
func (p *producer) disallowedSubstitutions(elem *Element) []xsd.DerivationMethod {
	ebv, ok := elem.Attr("block")
	if !ok {
		ebv, _ = p.schemaElem.Attr("blockDefault")
	}
	if ebv == "" {
		return nil // case 1
	}
	if ebv == "#all" {
		return []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction, xsd.DerivationSubstitution} // case 2
	}
	items := strings.Fields(ebv)
	var blocked []xsd.DerivationMethod // case 3
	for _, m := range []struct {
		token  string
		method xsd.DerivationMethod
	}{
		{"extension", xsd.DerivationExtension},
		{"restriction", xsd.DerivationRestriction},
		{"substitution", xsd.DerivationSubstitution},
	} {
		if slices.Contains(items, m.token) {
			blocked = append(blocked, m.method)
		}
	}
	return blocked
}

// produceNotation maps a top-level <notation> into a Notation Declaration
// (§3.14.2): {name} bundled with the schema's target namespace, {system
// identifier} from system= and {public identifier} from public=, each absent
// when its attribute is. Both absent is rejected inside [xsd.NewNotation]
// (n-props-correct, §3.14.6) — §3.14.3 defines no Schema Representation
// Constraint of its own. <notation> occurs only as a <schema> child (§3.17.2),
// so there is no nested form to map.
func (p *producer) produceNotation(elem *Element) (xsd.Notation, error) {
	name, _ := elem.Attr("name")
	qname := xsd.QName{Space: p.target, Local: name}
	var systemID, publicID *string
	if v, ok := elem.Attr("system"); ok {
		systemID = &v
	}
	if v, ok := elem.Attr("public"); ok {
		publicID = &v
	}
	return xsd.NewNotation(elem.Loc(), qname, systemID, publicID, nil)
}

// produceAttribute maps a top-level <attribute> into a global Attribute
// Declaration (§3.2.2.1 dcl.att.global). type= form only. qname reaches it from
// topLevelName through run, for the reason produceElement's doc gives.
func (p *producer) produceAttribute(qname xsd.QName, elem *Element) (xsd.AttributeDeclaration, error) {
	typeLex, hasType := elem.Attr("type")
	inline := childElement(elem, xsd.XMLSchemaNS, "simpleType") != nil

	if hasType && inline {
		return xsd.AttributeDeclaration{}, xsderr.New(ruleSrcAttribute, elem.Loc(),
			"attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both")
	}
	if inline {
		return xsd.AttributeDeclaration{}, xsderr.New(ruleSrcAttribute, elem.Loc(),
			"attribute has an inline <simpleType> child, which this producer does not yet support (only the type attribute form); src-attribute clause 4")
	}

	vc, err := valueConstraintOf(elem, ruleSrcAttribute)
	if err != nil {
		return xsd.AttributeDeclaration{}, err
	}

	typeName := xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"} // §3.2.2.1
	if hasType {
		typeName, err = p.resolveQName(elem, typeLex)
		if err != nil {
			return xsd.AttributeDeclaration{}, err
		}
	}
	return xsd.NewAttributeDeclaration(elem.Loc(), qname, xsd.TypeDefinitionRef{Name: typeName}, xsd.ScopeGlobal, vc, false, nil)
}

// valueConstraintOf maps the default/fixed attributes of an <element>/<attribute>
// to a *ValueConstraint, rejecting the both-present case (src-element clause 1 /
// src-attribute clause 1). rule selects which of the two constraints is charged.
// It serves both a declaration's own {value constraint} (vc_e, vc_a) and an
// Attribute Use's (vc_au, §3.5.1) — the mapping from the two XML attributes to
// the {variety}/{lexical form} record is identical; which component the result is
// attached to is the caller's decision (§3.2.2.2 / §3.2.2.3).
func valueConstraintOf(elem *Element, rule xsderr.Rule) (*xsd.ValueConstraint, error) {
	defLex, hasDef := elem.Attr("default")
	fixLex, hasFix := elem.Attr("fixed")
	if hasDef && hasFix {
		return nil, xsderr.New(rule, elem.Loc(),
			"declaration has both default and fixed, but %s clause 1 forbids both", rule)
	}
	if hasDef {
		vc := xsd.NewValueConstraint(xsd.ValueDefault, defLex)
		return &vc, nil
	}
	if hasFix {
		vc := xsd.NewValueConstraint(xsd.ValueFixed, fixLex)
		return &vc, nil
	}
	return nil, nil
}

// resolveQName resolves a QName-valued lexical that must name a schema component
// (a type=/base=/ref= value) to its expanded name, under BOTH halves of
// §3.17.6.2 src-resolve clause 4 (cl.qnr.nsdeclared): bindQName maps the lexical
// against the namespace bindings in scope at elem, and licensedNamespace then
// requires the namespace name it landed on to be one the containing schema
// document may reach into at all.
//
// A QName-valued attribute whose value does NOT resolve to a component —
// notQName, whose items §3.10.2's {disallowed names} mapping takes as plain QName
// values — takes bindQName directly: src-resolve governs ·resolution·, so
// charging clause 4 there would reject a name the spec never asks to resolve.
func (p *producer) resolveQName(elem *Element, lexical string) (xsd.QName, error) {
	qn, err := p.bindQName(elem, lexical)
	if err != nil {
		return xsd.QName{}, err
	}
	if err := p.licensedNamespace(elem, lexical, qn.Space); err != nil {
		return xsd.QName{}, err
	}
	return qn, nil
}

// bindQName maps a QName-valued lexical to its expanded name against the
// namespace bindings in scope at elem (Datatypes §3.3.18). A prefixed name whose
// prefix is unbound is rejected. An unprefixed name binds to the in-scope default
// namespace, or — when none is declared (or the default is declared empty) — to
// unqualifiedRefNS: the no-namespace name (src-resolve clause 4.1.1) normally,
// the assembly's namespace under chameleon coercion, deliberately never the
// schema's own targetNamespace otherwise.
func (p *producer) bindQName(elem *Element, lexical string) (xsd.QName, error) {
	before, after, found := strings.Cut(lexical, ":")
	prefix, local := "", before
	if found {
		prefix, local = before, after
	}

	if prefix == "" {
		// An absent default binding and an explicitly empty one (xmlns="") both
		// leave the reference's namespace name ·absent·, so both take the same path.
		uri, _ := elem.lookupPrefix("")
		if uri == "" {
			return xsd.QName{Space: p.unqualifiedRefNS(elem), Local: local}, nil
		}
		return xsd.QName{Space: uri, Local: local}, nil
	}

	uri, ok := elem.lookupPrefix(prefix)
	if !ok {
		return xsd.QName{}, xsderr.New(ruleSrcResolve, elem.Loc(),
			"the QName prefix %q of %q does not resolve to an in-scope namespace (src-resolve)", prefix, lexical)
	}
	return xsd.QName{Space: uri, Local: local}, nil
}

// licensedNamespace charges src-resolve clause 4 (cl.qnr.nsdeclared, §3.17.6.2)
// on a reference written at elem whose namespace name resolved to ns: the schema
// document CONTAINING the reference must itself license that namespace, which is
// what §4.2.6.1 calls "licensing references to components across namespaces". The
// license is per-document and never assembly-wide — a namespace some sibling
// document of the assembly <import>ed licenses nothing here — so a reference that
// clauses 1-3 would happily resolve is still rejected when its own document never
// asked for the namespace (#279).
//
// §4.2.6.1 is explicit that this is NOT a §5.3 missing sub-component: such
// references "are not handled as if they referred to missing components", so the
// verdict is charged here, at the reference, rather than deferred to finalize.
//
// The two namespace facts it tests are the containing document's, read as the
// spec words them. Clause 4.1.1 asks whether that document's <schema> carries a
// targetNamespace ATTRIBUTE, which for an <override>'s children is the OVERRIDING
// document's. Clause 4.2.1 tests the EFFECTIVE target namespace p.target instead:
// §F.1 task (a) has already made the including namespace a chameleon document's
// own by the time clause 4 is evaluated, and that coercion is constant across an
// <include>/<override> cascade, so p.target is the containing document's coerced
// targetNamespace either way.
func (p *producer) licensedNamespace(elem *Element, lexical, ns string) error {
	schemaElem := containingSchema(elem)
	if ns == "" {
		// Clause 4.1: an ·absent· namespace name is licensed by a document that
		// declares no targetNamespace (4.1.1) or that <import>s the unqualified
		// namespace (4.1.2, §4.2.6.1's "if that attribute is absent, then the import
		// allows unqualified reference to components with no target namespace").
		own := attrOr(schemaElem, "targetNamespace")
		if own == "" || importsNamespace(schemaElem, "") {
			return nil
		}
		return xsderr.New(ruleSrcResolve, elem.Loc(),
			"the unqualified reference %q resolves into the ·absent· namespace, which this schema document does not license: src-resolve clause 4.1 needs it to declare no targetNamespace (it declares %q) or to carry an <import> with no namespace attribute",
			lexical, own)
	}
	// Clause 4.2.1 (the document's own target namespace) and clauses 4.2.3 / 4.2.4
	// (the XSD and XSI namespaces, licensed with no <import> at all).
	if ns == p.target || ns == xsd.XMLSchemaNS || ns == xsd.XMLSchemaInstanceNS {
		return nil
	}
	if importsNamespace(schemaElem, ns) { // clause 4.2.2
		return nil
	}
	return xsderr.New(ruleSrcResolve, elem.Loc(),
		"the reference %q resolves into namespace %q, which this schema document does not license: src-resolve clause 4.2 needs it to be the document's own target namespace %q, the namespace of one of the document's own <import> elements, or the XSD or XSI namespace",
		lexical, ns, p.target)
}

// importsNamespace reports whether the <schema> element information item schema
// carries an <import> licensing references into ns — clause 4.2.2 for a namespace
// name, clause 4.1.2 for the ·absent· one (ns == ""). Only that document's OWN
// <import> children answer, since 4.2.2 scopes them to "some <import> element
// information item contained in the <schema> element information item of THAT
// schema document".
//
// The ·absent· namespace is the empty string throughout this package, so an
// <import namespace=""> reads as a bare one and licenses an absent-namespace
// reference; keeping the attribute's PRESENCE apart here would be a second
// encoding of namespace-absence (STYLE D3), the same fold checkNoSelfImport
// documents.
func importsNamespace(schema *Element, ns string) bool {
	for _, child := range schema.Children() {
		el, ok := child.(*Element)
		if !ok || !isXSD(el, "import") {
			continue
		}
		if attrOr(el, "namespace") == ns {
			return true
		}
	}
	return false
}

// containingSchema returns the <schema> element information item of the schema
// document elem textually lives in — src-resolve clause 4's "the schema document
// containing the ·QName·". That is usually the producer's own document, but a
// declaration substituted into its top level by an <override> in a DIFFERENT
// document (§F.2 clause 1) is licensed by the OVERRIDING document's <import>s, so
// the answer is the topmost ancestor of elem rather than p.schemaElem. Every
// document reaching a producer has been checked to be a <schema>
// (Document.IsSchema), so the walk cannot land on anything else.
func containingSchema(elem *Element) *Element {
	root := elem
	for e := elem; e != nil; e = e.Parent() {
		root = e
	}
	return root
}

// unqualifiedRefNS is the namespace name the unqualified QName REFERENCE carried
// by elem resolves to. Normally that is the ·absent· namespace (src-resolve
// clause 4.1.1). Under chameleon coercion it is the assembly's target namespace:
// §F.1 task (b) "updates all unqualified QName references so that their
// namespace names become the actual value of the targetNamespace attribute" —
// every reference in the document, including one naming a sibling component of
// the same document — and §4.2.3's closing paragraph extends the conversion to
// still-unresolved reference QNames retained for finalize.
//
// elem is needed because a chameleon document's top level can hold a declaration
// that is NOT its own: §4.2.5 clause 3.2.1 orders the two transformations "first
// [§F.1] and then [§F.2]", so an <override>'s children are substituted into Dold
// AFTER §F.1 has run over it and are therefore untouched by task (b). They keep
// the ·absent· namespace for their unqualified references, even though the
// components they define are minted in the coerced target namespace.
func (p *producer) unqualifiedRefNS(elem *Element) string {
	if p.chameleon() && p.declares(elem) {
		return p.target
	}
	return ""
}

// declares reports whether elem is part of THIS document's tree, as opposed to a
// declaration substituted into its top level by an <override> in a different
// document (§F.2 clause 1). It asks containingSchema rather than storing
// provenance on the element, since provenance is a property of the producer's
// question, not of the raw node (STYLE D3).
func (p *producer) declares(elem *Element) bool {
	return containingSchema(elem) == p.schemaElem
}

// facetKindOf maps a plain-lexical constraining-facet element's local name to its
// [xsd.FacetKind]. enumeration and assertion are deliberately absent — their
// {value} is not a lexical string, so restrictionFacets handles them separately
// (assertion through xsd.NewAssertionsFacet, enumeration by rejection).
func facetKindOf(local string) (xsd.FacetKind, bool) {
	switch local {
	case "length":
		return xsd.FacetLength, true
	case "minLength":
		return xsd.FacetMinLength, true
	case "maxLength":
		return xsd.FacetMaxLength, true
	case "pattern":
		return xsd.FacetPattern, true
	case "whiteSpace":
		return xsd.FacetWhiteSpace, true
	case "maxInclusive":
		return xsd.FacetMaxInclusive, true
	case "maxExclusive":
		return xsd.FacetMaxExclusive, true
	case "minInclusive":
		return xsd.FacetMinInclusive, true
	case "minExclusive":
		return xsd.FacetMinExclusive, true
	case "totalDigits":
		return xsd.FacetTotalDigits, true
	case "fractionDigits":
		return xsd.FacetFractionDigits, true
	case "explicitTimezone":
		return xsd.FacetExplicitTimezone, true
	case "maxScale":
		// xr-maxScale/xr-minScale (xsd-precisionDecimal.md §4.2.2/§4.3.2): the two
		// precisionDecimal extension facets have exactly the totalDigits shape —
		// {value} is the value attribute's xs:integer lexical form, {fixed} the
		// fixed attribute's boolean (false when absent) — so they take the generic
		// plain-lexical path, not the enumeration rejection.
		return xsd.FacetMaxScale, true
	case "minScale":
		return xsd.FacetMinScale, true
	}
	return 0, false
}

// isXSD reports whether el's expanded name is {XMLSchemaNS}local.
func isXSD(el *Element, local string) bool {
	return el.Name().Space() == xsd.XMLSchemaNS && el.Name().Local() == local
}

// childElement returns el's first child element with the expanded name
// {space}local, or nil.
func childElement(el *Element, space, local string) *Element {
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok {
			continue
		}
		if name := c.Name(); name.Space() == space && name.Local() == local {
			return c
		}
	}
	return nil
}

// facetFixed maps a facet element's fixed attribute to that facet's {fixed}
// property: "The actual value of the fixed [attribute], if present, otherwise
// false" (xsd-precisionDecimal.md §4.2.2 xr-maxScale and §4.3.2 xr-minScale, the
// same wording Datatypes §4.3.x carries for each of the other eleven
// {fixed}-bearing facets, e.g. §4.3.6.1 f-w-fixed). "Actual value" is the
// xs:boolean VALUE the attribute's declared type maps its literal to, not the
// literal itself, so the two mapping stages run in the order §4.1.4 fixes:
//
//   - pre-lexical. xs:boolean fixes whiteSpace to collapse (§3.3.2.3, §4.3.6) and
//     the whiteSpace facet is applied BEFORE lexical-space membership is tested
//     (§4.1.4), so " true " is the value true. Trimming exactly #x9/#xA/#xD/#x20 —
//     the only characters §4.3.6's replace and collapse steps ever touch, as
//     value/whitespace.go spells out — decides that membership exactly as a full
//     collapse would. Let T be the trimmed literal and R its collapse: if T holds
//     interior whitespace then R holds a #x20 and no booleanRep literal contains
//     one, so both reject; otherwise R == T. The trim set is load-bearing and
//     cannot be strings.TrimSpace, whose unicode.IsSpace class also cuts U+0085,
//     U+00A0, U+2028 and the rest — characters §4.3.6 is NOT whitespace for and
//     collapse preserves, so trimming them would accept literals the spec rejects.
//     A four-character trim is not a third private copy of the collapse algorithm
//     (STYLE T4).
//   - lexical. booleanRep ::= 'true' | 'false' | '1' | '0' (§3.3.2.2), case
//     sensitive: "TRUE" is not in the lexical space. Anything outside those four
//     is charged cvc-datatype-valid (§4.1.4) — a literal outside a datatype's
//     lexical space is invalid, and no clause lets it default to false.
//
// An absent attribute is the {fixed} = false default and is NOT an error; that is
// a branch distinct from present-but-invalid.
func facetFixed(el *Element) (bool, error) {
	lexical, ok := el.Attr("fixed")
	if !ok {
		return false, nil
	}
	switch strings.Trim(lexical, "\x09\x0A\x0D\x20") {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	}
	return false, xsderr.New(ruleDatatypeValid, el.Loc(),
		"<%s> fixed value %q is not in the lexical space of xs:boolean (true, false, 1, 0) that the schema for schema documents declares for it",
		el.Name().Local(), lexical)
}
