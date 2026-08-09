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
	// <group ref> ·base particle· resolves to (xsd.ExtensionContentType, through
	// resolveModelGroup); every OTHER <group ref> stays an unresolved
	// ModelGroupRef until finalize (produceGroupRefParticle).
	//
	// The owning producer is carried for both reasons attributeGroups carries one:
	// a <group> body holds local <element> declarations, whose {target namespace}
	// §3.3.2.3 takes from "the ancestor <schema> element information item" of the
	// DECLARING document, and unqualified type=/ref= references inside it take that
	// document's §F.1 chameleon coercion.
	modelGroups map[xsd.QName]typeSource

	// elements maps each top-level <element>'s expanded name to its source (raw
	// element plus the producer of the document that declares it), filled by the
	// pre-scan so §3.3.2.1's {type definition} clause 3 — "the declared {type
	// definition} of the Element Declaration ·resolved· to by the first QName in
	// the ·actual value· of the substitutionGroup attribute" — reaches a HEAD
	// declared later in the document or in another document of the assembly
	// (§3.1.3: forward reference to named declarations is allowed).
	//
	// It holds SOURCES, not built components, and clause 3 reads only ATTRIBUTES
	// and CHILD SHAPES off them (substitutionGroupHeadType: an inline type child,
	// then type=, then substitutionGroup= to walk on) rather than building the
	// head declaration on demand: an Element Declaration is not memoized the way
	// a type is, so building one here would produce a second component for the
	// same name and register its identity constraints twice. Where the head's
	// type is an inline anonymous <complexType>, the walk therefore yields a
	// REFERENCE to the head (xsd.SubstitutionGroupHeadTypeRef) and finalize reads
	// the head's own already-built {type definition} through it — the one place
	// the built component is needed, reached after every declaration exists
	// rather than during the pre-scan (#342).
	//
	// The owning producer is carried for the reason every other index carries one:
	// the head's type= is a QName in the HEAD's document, so it resolves under that
	// document's in-scope prefix bindings and §F.1 chameleon coercion (src-resolve
	// clause 4.1.1). Resolved under a referring producer instead, an unqualified
	// or differently-prefixed head type would name the wrong type.
	elements map[xsd.QName]typeSource

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
	// no circularity to guard (PRINCIPLES 9).
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
		elements:            make(map[xsd.QName]typeSource),
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
// <attributeGroup>s (forward <attributeGroup ref> inlining, §3.6.2.1), named
// <group>s (forward <group ref> resolution for §3.4.2.3.3 clause 4.2.3's
// sub-case test, resolveModelGroup) and top-level <element>s (forward
// substitutionGroup= heads for §3.3.2.1's {type definition} clause 3,
// substitutionGroupHeadType) in the assembly-wide symbol table, building
// nothing yet. EVERY document's prescan runs before ANY document's run, so a
// reference in one document reaches a definition in another (§4.2.3
// c-incl-incl). Names are minted in the effective target namespace, so a
// chameleon document's definitions are registered under the including namespace
// (§F.1 task a).
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
		case isXSD(el, "element"):
			p.symbols.elements[xsd.QName{Space: p.target, Local: name}] = typeSource{elem: decl, owner: p}
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
// (PRINCIPLES 9). A name is minted in the effective target namespace, exactly as
// produceIdentityConstraint mints the definition's own {name}, so the index key
// and the component name agree under chameleon coercion (§F.1 task a).
//
// The walk is confined to what THIS producer actually maps, by three exclusions.
// prescan withholds the composition directives' subtrees (compositionDirective),
// whose contents belong to another document's producer. The two below are the
// two ways an element can stand in a schema document while corresponding to no
// component at all, and each is tested on el ITSELF, before el is indexed and
// before its children are reached — so the exclusion covers the whole subtree,
// including el, at both the roots the pre-scan is entered from (prescan,
// prescanRedefine) and every node beneath them.
//
// An element outside the Schema namespace is excluded because §A gives the
// schema for schema documents no element wildcard outside <appinfo> and
// <documentation>: such an element corresponds to no component wherever it
// stands, and so does everything written beneath it.
//
// An <annotation> is excluded because <appinfo> and <documentation> hold mixed,
// processContents="lax" content (§A), and §3 is explicit that "neither the
// correspondences described nor the XML Representation Constraints apply to
// elements in the Schema namespace which occur as descendants of <appinfo> or
// <documentation>": a <key name="…"> there is prose — an illustration, possibly
// truncated — and is mapped to no component by anyone.
//
// Indexing either would make the index a strict SUPERSET of
// {identity-constraint definitions}, the very property src-resolve clause 1.7
// looks a ref= up in, letting an unmapped element shadow a real same-named
// definition or satisfy a ref= that resolves to nothing.
func (p *producer) prescanIdentityConstraints(el *Element) {
	if el.Name().Space() != xsd.XMLSchemaNS {
		return
	}
	if isXSD(el, "annotation") {
		return
	}
	p.indexIdentityConstraint(el)
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok {
			continue
		}
		p.prescanIdentityConstraints(c)
	}
}

// indexIdentityConstraint registers el in the assembly-wide index under its
// name=, when el is a NAMED <unique>/<key>/<keyref>; every other element is left
// unindexed, including the ref= form, which declares nothing (§3.11.2).
//
// The caller owns the question of whether el is mapped at all
// (prescanIdentityConstraints); this function owns only the question of whether
// a mapped element declares an identity constraint.
func (p *producer) indexIdentityConstraint(el *Element) {
	category, ok := identityConstraintCategoryOf(el.Name().Local())
	if !ok {
		return
	}
	name, ok := el.Attr("name")
	if !ok {
		return
	}
	p.symbols.identityConstraints[xsd.QName{Space: p.target, Local: name}] =
		identityConstraintSource{elem: el, category: category, owner: p}
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
// claimed for it. Tracked as #523.
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
// dispatch, produceRedefinition's redefining <complexType>, and
// resolveBaseType's on-demand construction all go through it, so a named type is
// mapped exactly once. That is what makes a reference to a redefined name
// resolve to the REDEFINITION from both documents, as src-expredef's note
// requires: the redefining declaration is what prescanRedefine registered under
// that name, so every route ends at this one memo entry. It populates the memo
// only — registering the component with the builder is run's or
// produceRedefinition's job, at the type's own document-order position.
//
// An ANONYMOUS <complexType> deliberately does NOT come through here: it calls
// produceComplexType directly (produceElement and produceLocalElement for an
// inline child, redefinedComplexBase for src-expredef clause 1.1's original).
// The memo is keyed by name and an anonymous type has none, so it would have
// nothing to key on.
//
// Nor can it MEMBER a cycle this function's guard would catch — but the reason
// is narrower than it once was, and the difference is load-bearing. Nothing can
// NAME an anonymous type, so it can be no cycle's entry point and this
// name-keyed sentinel would never see it. It can nonetheless sit ON a chain that
// closes: src-expredef clause 1.1's original is an anonymous type whose own base=
// names a top-level type again, so a cycle can run THROUGH it. PRINCIPLES 9's
// "construction order makes one impossible" therefore does NOT discharge the
// anonymous hop, and the blanket claim that it did was false the moment #505
// landed. The rejection for such a chain is the on-stack sentinel below, reached
// at the named type the chain comes back to, and its finalize-side twin
// xsd/resolve.go's checkComplexBaseAcyclic, which descends the anonymous hop for
// exactly this reason.
//
// A name already on the build stack (the PRESENT-nil memo state) is a circular
// {base type definition} chain, charged ct-props-correct clause 3 (§3.4.6.1).
// That is the SAME rule, with the same verdict, that xsd/resolve.go's
// checkComplexBaseAcyclic charges: two entry points on one rule for the two
// construction paths — this one for the producer, whose demand-driven recursion
// would otherwise not terminate, and that one for the programmatic
// SchemaBuilder, which has no producer and must stay self-defending. Neither
// substitutes for the other (PRINCIPLES 9's "detect once at construction" applies
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

	ct, err := p.produceComplexType(p.namedComplexTypeIdentity(name, elem), elem)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	p.symbols.builtComplex[name] = &ct // replace the on-stack sentinel with the finished node
	return ct, nil
}

// namedComplexTypeIdentity chooses which NAMED arm of complexTypeIdentity elem
// is built under: the redefining one when elem is a <complexType> child of a
// <redefine> this document followed, and the plain one otherwise.
//
// The question is asked HERE, of the element, rather than at produceRedefinition
// alone, because that is not the only route to a redefining declaration:
// prescanRedefine registers it under its own expanded name, so a reference to
// the redefined name from either document arrives through resolveBaseType and
// buildComplexType instead. Deciding by element makes every route agree, which
// is what keeps the memo holding ONE component for that name (see
// buildComplexType).
//
// The redefining arm mints the identity src-expredef clause 1.1 needs for the
// original's {context}. One mint per redefinition, and only one is possible:
// buildComplexType's memo means the redefining declaration is produced exactly
// once per assembly.
func (p *producer) namedComplexTypeIdentity(name xsd.QName, elem *Element) complexTypeIdentity {
	if _, _, redefining := p.redefinitionOf(elem); redefining {
		return redefiningComplexType{name: name, owner: xsd.NewComponentID()}
	}
	return namedComplexType{name: name}
}

// resolveBaseType identifies the {base type definition} a base= attribute names
// (§3.4.2 preamble), in BOTH the forms a §3.4.2 mapping needs: the resolved
// COMPONENT, which the content-type tableaux and §3.4.2.1 clause 1's
// {assertions} fold read, and the xsd.TypeDefinitionOrRef SLOT the built
// component stores. The two are returned together because one decision fixes
// both, and splitting them would let a caller pair a component with a slot that
// does not name it (STYLE D3). at is the <restriction>/<extension> carrying the
// base=, charged for a failure; id is the identity of the type being built,
// which is what makes the redefine branch below reachable.
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
func (p *producer) resolveBaseType(id complexTypeIdentity, at *Element, name xsd.QName) (xsd.TypeDefinition, xsd.TypeDefinitionOrRef, error) {
	if orig, owned, err := p.redefinedComplexBase(id, at, name); owned || err != nil {
		if err != nil {
			return nil, nil, err
		}
		return orig, xsd.InlineTypeDefinition{Definition: orig}, nil
	}
	if ct, done := p.symbols.builtComplex[name]; done && ct != nil {
		return *ct, xsd.TypeDefinitionRef{Name: name}, nil
	}
	if src, ok := p.symbols.complexTypes[name]; ok {
		// Unbuilt or on-stack: buildComplexType handles the memo hit and the
		// ct-props-correct clause 3 cycle rejection alike.
		ct, err := src.owner.buildComplexType(name, src.elem)
		if err != nil {
			return nil, nil, err
		}
		return ct, xsd.TypeDefinitionRef{Name: name}, nil
	}
	if st, ok := p.symbols.built[name]; ok && st != nil {
		return st, xsd.TypeDefinitionRef{Name: name}, nil
	}
	if src, ok := p.symbols.simpleTypes[name]; ok {
		st, err := src.owner.buildSimpleType(name, src.elem)
		if err != nil {
			return nil, nil, err
		}
		return st, xsd.TypeDefinitionRef{Name: name}, nil
	}
	return nil, nil, xsderr.New(ruleSrcResolve, at.Loc(),
		"base type %s does not resolve to any type definition in scope (src-resolve clause 1.1)", name)
}

// redefinedComplexBase builds src-expredef clause 1.1's ORIGINAL when at's base=
// is a redefining <complexType>'s self-reference: "one component which
// corresponds to the top-level definition item with the same name in the
// <redefine>d schema document, as defined in Schema Component Details (§3),
// except that its {name} is ·absent· and its {context} is the redefining
// component". owned is false for every other base=, which then resolves
// ordinarily — including a base= inside a redefining type that names some OTHER
// type, and a reference to the redefined name from anywhere else, both of which
// src-expredef's own note requires to reach the REDEFINITION.
//
// It is the complex-type twin of resolveBase's redefine branch (see there), and
// makes the same three moves for the same reasons:
//
//   - the source is the REDEFINED document's declaration (redefinedTypeBase),
//     never the redefining one, so the original's own base= resolves to whatever
//     that document said and is never re-pointed at the redefinition — the false
//     circularity the pairing exists to prevent;
//   - it is built under that document's OWN producer (src.owner), so it enters
//     no symbol table, is registered with no builder, and takes its own
//     document's target namespace and schema-level defaults;
//   - it goes to produceComplexType DIRECTLY rather than through
//     buildComplexType, because that memo is keyed by name and this component
//     has none (see buildComplexType).
//
// The identity it is built with carries the REDEFINING type's minted
// xsd.ComponentID, which is what makes the original's {context} point back at
// its owner; xsd.NewComplexTypeOwningBase checks the two agree.
func (p *producer) redefinedComplexBase(id complexTypeIdentity, at *Element, name xsd.QName) (xsd.ComplexType, bool, error) {
	r, redefining := id.(redefiningComplexType)
	if !redefining {
		return xsd.ComplexType{}, false, nil
	}
	src, self := p.redefinedTypeBase(at, name)
	if !self {
		return xsd.ComplexType{}, false, nil
	}
	orig, err := src.owner.produceComplexType(redefineOriginalComplexType{owner: r.owner}, src.elem)
	if err != nil {
		return xsd.ComplexType{}, true, err
	}
	return orig, true, nil
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
// caller is xsd.ExtensionContentType, to which it is passed as the group-lookup
// callback, and whose §3.4.2.3.3 clause 4.2.3 sub-case test must know the
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
// the own facets and {final} (§3.16.2.1, simpleTypeFinal), and constructs. It
// does NOT memoize — the memo/cycle bookkeeping lives in buildSimpleType; an
// anonymous inline type has no name to key on and is unreferenceable, so it is
// built here directly, once.
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
	// The declared derivation is ·restriction· — this producer's only
	// <simpleType> alternative today (restrictionOf rejects <list>/<union>). It
	// carries no property of its own: §3.16.2.1 gives a restriction the
	// {variety}, {primitive type definition}, {item type definition} and {member
	// type definitions} of its {base type definition}, and xsd.SimpleType derives
	// all four from the base chain, so the producer no longer re-derives any of
	// them here (STYLE D3).
	st, err := xsd.NewSimpleType(elem.Loc(), name, xsd.RestrictionDerivation{}, base, facets, p.simpleTypeFinal(elem))
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
// This is also the ONLY path that can reach the chain's clause 3, the head's
// declared type: substitutionGroup= is legal on a top-level <element> alone
// (§3.3.2), so produceLocalElement rejects it outright and localDeclaredType has
// no tier for it. Clause 3 is decided by substitutionGroupHeadType, from the
// FIRST resolved affiliation, which is why the affiliations are mapped before
// the type here.
//
// Tier 1's inline <simpleType> child is the one form still declined on this
// path: #229 widened §3.2.2.2/§3.3.2.3's LOCAL mapping only, and the global
// simple-type widening is a separate, adjacent change. It is declined with a
// plain "not yet produced" error, never a fabricated src-element verdict — the
// schema is legal and it is this producer that is incomplete (STYLE E2) — and
// conformance/schema.go's elementDecidable declines the shape so the limitation
// never reaches a validity verdict. That decline is also why clause 3 declines a
// head whose type is an inline <simpleType> rather than referencing it: a
// reference to a declaration this path can never build would name nothing.
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

	constraints, err := p.identityConstraintsOf(elem)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	affiliations, err := p.substitutionGroupAffiliations(elem)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	var typeDef xsd.TypeDefinitionOrRef = xsd.TypeDefinitionRef{Name: anyTypeName} // §3.3.2.1 case 4
	switch {
	case inlineComplex != nil:
		// Clause 1 wins outright and typeDef is never read on this path (the
		// inline branch below builds the type itself), so the lower clauses must
		// not run at all: clause 3's lookup can decline the HEAD's inline
		// <simpleType> (#442), a limitation of a type this element never reaches,
		// and letting it run would fail an element clause 1 fully decides.
	case hasType:
		name, qerr := p.resolveQName(elem, typeLex) // case 2
		if qerr != nil {
			return xsd.ElementDeclaration{}, qerr
		}
		typeDef = xsd.TypeDefinitionRef{Name: name}
	case len(affiliations) > 0:
		inherited, herr := p.substitutionGroupHeadType(elem, affiliations[0]) // case 3
		if herr != nil {
			return xsd.ElementDeclaration{}, herr
		}
		typeDef = inherited
	}
	// §3.3.2.2 dcl.elt.global: {scope} is {variety} global, {parent} ·absent·.
	if inlineComplex != nil {
		edID := xsd.NewComponentID()
		ct, err := p.produceComplexType(elementOwnedComplexType{owner: edID}, inlineComplex)
		if err != nil {
			return xsd.ElementDeclaration{}, err
		}
		return xsd.NewElementDeclarationOwningType(elem.Loc(), edID, qname, ct, nil, xsd.NewGlobalScope(), vc,
			false, constraints, affiliations, p.substitutionGroupExclusions(elem), false, p.disallowedSubstitutions(elem), nil)
	}
	return xsd.NewElementDeclaration(elem.Loc(), qname, typeDef, nil, xsd.NewGlobalScope(), vc,
		false, constraints, affiliations, p.substitutionGroupExclusions(elem), false, p.disallowedSubstitutions(elem), nil)
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
// {substitution group exclusions} — the final=/finalDefault twin — is mapped
// alongside it by substitutionGroupExclusions, for the same operational reason:
// e-props-correct clause 4 (c-vs-sg) reads it on the HEAD of an affiliation
// edge, and xsd enforces that clause (xsd/substitutiongrouptypes.go, #395), so
// leaving it universally empty would silently un-block every head that spells
// final= on itself.
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

// substitutionGroupHeadType is §3.3.2.1 dcl.elt.common's {type definition} case
// 3: "The declared {type definition} of the Element Declaration ·resolved· to by
// the first QName in the ·actual value· of the substitutionGroup attribute, if
// present." head is that first QName — the caller passes affiliations[0], so the
// "first item alone" reading is applied to the RESOLVED list rather than
// re-derived from the lexical form (STYLE T4); every item feeds {substitution
// group affiliations}, but only the first feeds this property.
//
// It reads the head's SOURCE rather than its built component (symbols.elements),
// which is what makes a forward or cross-document head resolve: the head may be
// declared after this member or in another document, and every document's
// pre-scan runs before any document's run. Building the head declaration on
// demand is not an option — element declarations carry no build memo, so a
// second construction would register the head's identity constraints a second
// time and fabricate a sch-props-correct clause 2 collision.
//
// The result is a SLOT (xsd.TypeDefinitionOrRef), not a component, because that
// is what a produced declaration's {type definition} holds; finalize resolves it.
// Which arm depends on how the terminal head spells its own type:
//
//   - a NAMED type — the head's type= — is an xsd.TypeDefinitionRef, resolved at
//     finalize exactly as this element's own type= would be;
//   - an inline anonymous <complexType> is an xsd.SubstitutionGroupHeadTypeRef
//     naming that head. Case 3 makes the member's {type definition} the head's
//     own component, which no by-name reference can reach and no second
//     declaration can OWN (§3.4.2.1 dcl.ctd.common ties an anonymous type's
//     {context} to one declaration), so the slot references the OWNER instead —
//     see that type, and §3.4.6.5's no-identity Note for why identity rather
//     than a copy is what the spec asks for.
//
// Three cases yield case 4's xs:anyType instead, each of them the honest answer
// rather than a fallback:
//
//   - the head names no top-level <element> in the assembly. The affiliation is
//     an ·absent· member under §5.3 (Missing Sub-components) — a valid schema,
//     W3C saxonData/Missing missing002 — so there is no declaration to take a
//     declared type from, and e-props-correct clause 4 skips the same edge
//     (xsd/substitutiongrouptypes.go);
//   - the head carries neither a type child, type=, nor substitutionGroup=, so
//     ITS own {type definition} is case 4's xs:anyType and the member inherits
//     exactly that;
//   - the walk returns to a head it has already visited. That is a circular
//     substitution group, which e-props-correct clause 5 rejects at finalize
//     (xsd/resolve.go's checkSubstitutionGroupsAcyclic) over the whole graph;
//     seen is a walk-scoped TERMINATION guard and never a verdict, the same split
//     symbols.builtGroups' on-stack half records. It is read only by key, so no
//     map iteration reaches any output (STYLE D2).
//
// CLAUSE ORDER. The walk tests the inline type CHILD before type=, matching the
// order §3.3.2.1's own table states ("the first of the following that applies",
// clause 1 being the child). It used to test type= first, which was unobservable
// — produceElement charges src-element clause 3 when a head carries both, so no
// head that reaches this walk with both survives its own production — but with
// clause 1 now yielding a real answer rather than a decline, the inverted order
// would be a genuine mis-mapping rather than a harmless one (#342, the #395
// post-land addendum). The both-present head is still rejected, by src-element
// clause 3 at the head's own turn; this walk reads the head's SOURCE out of the
// pre-scan index and never runs produceElement on it, so it deliberately charges
// no verdict of its own here.
//
// ONE clause-1 shape stays declined, with a plain "not yet produced" error and
// never a fabricated rule verdict (STYLE E2): a head whose type is an inline
// <simpleType>. That is not a limitation of the sharing mechanism — the arm
// would express it — but of the head itself: produceElement declines a top-level
// <element> with an inline <simpleType> outright (#442), so a member must not be
// handed a reference to a declaration this producer can never build.
func (p *producer) substitutionGroupHeadType(at *Element, head xsd.QName) (xsd.TypeDefinitionOrRef, error) {
	seen := map[xsd.QName]bool{}
	for !seen[head] {
		seen[head] = true
		src, ok := p.symbols.elements[head]
		if !ok {
			return xsd.TypeDefinitionRef{Name: anyTypeName}, nil // an ·absent· head (§5.3)
		}
		if childElement(src.elem, xsd.XMLSchemaNS, "complexType") != nil { // clause 1
			return xsd.SubstitutionGroupHeadTypeRef{Head: head}, nil
		}
		if childElement(src.elem, xsd.XMLSchemaNS, "simpleType") != nil {
			return nil, fmt.Errorf("parser: <element> at %s inherits its {type definition} from the substitution group head %s (§3.3.2.1 dcl.elt.common clause 3), whose type is an inline <simpleType> that a top-level <element> is not yet produced with", at.Loc(), head)
		}
		if lex, has := src.elem.Attr("type"); has { // clause 2
			name, err := src.owner.resolveQName(src.elem, lex)
			if err != nil {
				return nil, err
			}
			return xsd.TypeDefinitionRef{Name: name}, nil
		}
		lexHeads, has := src.elem.Attr("substitutionGroup")
		if !has {
			return xsd.TypeDefinitionRef{Name: anyTypeName}, nil // the head's own {type definition} is case 4
		}
		items := strings.Fields(lexHeads)
		if len(items) == 0 {
			return xsd.TypeDefinitionRef{Name: anyTypeName}, nil
		}
		next, err := src.owner.resolveQName(src.elem, items[0])
		if err != nil {
			return nil, err
		}
		head = next
	}
	// A circular affiliation chain; clause 5 charges it at finalize.
	return xsd.TypeDefinitionRef{Name: anyTypeName}, nil
}

// elementBlockKeywords is the ·relevant set· the {disallowed substitutions} row
// draws from — the §3.3.1 subset of {substitution, extension, restriction} — in
// the canonical order the row's own case 2 writes it in. It is the only one of
// the four sets here that contains substitution.
var elementBlockKeywords = []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction, xsd.DerivationSubstitution}

// elementFinalKeywords is the ·relevant set· the {substitution group exclusions}
// row draws from: {extension, restriction} (§3.3.1, §3.3.2.1), TWO keywords and
// not the block row's three. substitution is not one of them, and
// xsd.NewElementDeclaration rejects a set containing it as an e-props-correct
// clause 1 tableau violation.
//
// It has the same MEMBERS as complexTypeKeywords below and is deliberately a
// separate var, not an alias: §3.3.2.1's row states this relevant set in its own
// right ("with the relevant set being {extension, restriction}") while §3.4.2.1
// states the complex-type one, so these are two spec facts that happen to
// coincide, not one fact written twice — STYLE D3 forbids the latter and says
// nothing about the former. Sharing one var would silently couple an element
// property to a complex-type property.
var elementFinalKeywords = []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction}

// complexTypeKeywords is the ·relevant set· of BOTH complex-type properties in
// this family — {prohibited substitutions} and {final} (§3.4.2.1
// dcl.ctd.common). It is one set for the two because the table defines the
// second entirely by reference to the first: "[a]s for {prohibited
// substitutions} above, but using the final and finalDefault attributes in place
// of the block and blockDefault attributes". TWO members: substitution is not
// among them though blockDefault's grammar admits it, and neither list nor union
// is though finalDefault's does. Both §3.4.1 property definitions say "a subset
// of {extension, restriction}".
var complexTypeKeywords = []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction}

// simpleTypeFinalKeywords is the ·relevant set· of a simple type's {final}
// (§3.16.2.1 map.std.common): FOUR members, in the order that rule's case 2
// writes them. It is the widest of the four sets here and is exactly
// finalDefault's own vocabulary, so nothing is ever dropped on this path.
var simpleTypeFinalKeywords = []xsd.DerivationMethod{xsd.DerivationRestriction, xsd.DerivationExtension, xsd.DerivationList, xsd.DerivationUnion}

// effectiveDerivationSet is the three-case ·effective block value· analysis that
// every member of the block=/final= mapping family runs: {disallowed
// substitutions} and {substitution group exclusions} on an element (§3.3.2.1),
// {prohibited substitutions} and {final} on a complex type (§3.4.2.1), and
// {final} on a simple type (§3.16.2.1). The five mappings differ in exactly two
// things — the attribute-name pair and the ·relevant set· — so they are ONE
// function taking both rather than five copies of the case analysis (STYLE T4).
//
// The EBV is local on elem if PRESENT, otherwise fallback on the ancestor
// <schema> if present, otherwise the empty string. "Present" is a test on the
// attribute, not on its value: a local block=""/final="" is present, so it wins
// over the schema-level default and takes case 1 below — the distinction a "" ==
// absent reading would lose. Then the empty string maps to the empty set (case
// 1), "#all" to the whole relevant set (case 2), and anything else to the
// relevant-set members its whitespace-separated list names (case 3).
//
// Items outside relevant are IGNORED, never rejected, on both paths. Each table
// says so of its own Default attribute — blockDefault "may include values other
// than restriction or extension", and "those values are ignored in the
// determination of {prohibited substitutions} for complex type definitions (they
// are used elsewhere)" — and the local attributes are treated the same way from
// the other side: their vocabulary is fixed by the schema for schema documents,
// which this producer runs no validation pass against, so an out-of-vocabulary
// local token is a grammar fault nothing here is positioned to charge. It is
// also what makes final="substitution" contribute nothing to {substitution group
// exclusions} and a shared finalDefault="list" nothing to a complex type's
// {final}.
//
// The result is in relevant's own fixed order, not the attribute's lexical
// order. These properties ARE sets drawn from a fixed set, so one set must have
// one encoding (STYLE D2): final="restriction extension" and final="extension
// restriction" must build the identical component. Each token is matched against
// xsd.DerivationMethod's own String(), which returns the verbatim spec token, so
// no relevant set writes the token spellings down a second time beside the
// methods that already carry them (STYLE D3). Case 2 returns a fresh slice,
// never relevant itself, so no caller can reach a package-level set.
func (p *producer) effectiveDerivationSet(elem *Element, local, fallback string, relevant []xsd.DerivationMethod) []xsd.DerivationMethod {
	ebv, ok := elem.Attr(local)
	if !ok {
		ebv, _ = p.schemaElem.Attr(fallback)
	}
	if ebv == "" {
		return nil // case 1
	}
	if ebv == "#all" {
		return slices.Clone(relevant) // case 2
	}
	items := strings.Fields(ebv)
	var set []xsd.DerivationMethod // case 3
	for _, m := range relevant {
		if slices.Contains(items, m.String()) {
			set = append(set, m)
		}
	}
	return set
}

// disallowedSubstitutions maps an <element>'s block attribute into {disallowed
// substitutions} (§3.3.2.1 dcl.elt.common), over the three-keyword element set.
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
// The result is in the spec's own canonical order — extension, restriction,
// substitution, the order the table's case 2 writes the set in — rather than in
// the attribute's lexical order; effectiveDerivationSet's doc carries the STYLE
// D2 reasoning, which is the same for every member of the family.
//
// The ·relevant set· it draws from is elementBlockKeywords; the case analysis
// itself lives in effectiveDerivationSet, which the other four members of the
// family reuse unchanged.
func (p *producer) disallowedSubstitutions(elem *Element) []xsd.DerivationMethod {
	return p.effectiveDerivationSet(elem, "block", "blockDefault", elementBlockKeywords)
}

// substitutionGroupExclusions maps an <element>'s final attribute into
// {substitution group exclusions} (§3.3.2.1 dcl.elt.common, whose row for the
// property reads in full: "As for {disallowed substitutions} above, but using
// the final and finalDefault [attributes] in place of the block and blockDefault
// [attributes] and with the relevant set being {extension, restriction}"). So it
// is disallowedSubstitutions' case analysis over a different attribute pair and
// a different ·relevant set·, which is exactly what effectiveDerivationSet takes
// as parameters (STYLE T4) — including "#all", whose expansion here is the TWO
// keywords of elementFinalKeywords and not the three of the block row.
//
// It is mapped for the TOP-LEVEL form only. A local <element> carries no final
// attribute at all (§3.3.2 gives it to the top-level form alone), so its EBV
// could come only from the ancestor <schema>'s finalDefault — and the property
// would be unreadable there whatever it held: e-props-correct clause 4 reads
// {substitution group exclusions} on the HEAD of an affiliation edge, an
// affiliation ·resolves· by expanded name and so names a top-level declaration,
// and clause 3 keeps a local declaration out of the property in the first place.
// Mapping it on the local path would add a fact with no reader (STYLE D4).
func (p *producer) substitutionGroupExclusions(elem *Element) []xsd.DerivationMethod {
	return p.effectiveDerivationSet(elem, "final", "finalDefault", elementFinalKeywords)
}

// complexTypeProhibitedSubstitutions maps a <complexType>'s block attribute into
// {prohibited substitutions} (§3.4.2.1 dcl.ctd.common), over the two-keyword
// complex-type set — NOT the three-keyword element set disallowedSubstitutions
// uses, since substitution is not a member of this property.
//
// It is read by cos-equiv-derived-ok-rec clause 2.2/2.3's blocking union
// (§3.3.6.3, substitutiongroup.go) and by ·validly substitutable· (§3.4.6.4,
// complexderivation.go), both of which saw a universally empty set until this
// mapping existed.
func (p *producer) complexTypeProhibitedSubstitutions(ctElem *Element) []xsd.DerivationMethod {
	return p.effectiveDerivationSet(ctElem, "block", "blockDefault", complexTypeKeywords)
}

// complexTypeFinal maps a <complexType>'s final attribute into {final} (§3.4.2.1
// dcl.ctd.common, "[a]s for {prohibited substitutions} above, but using the
// final and finalDefault attributes"), over the same two-keyword set — NOT the
// four-keyword simple-type set, so a list or union token inherited from a shared
// finalDefault is dropped here.
//
// It is read by cos-ct-extends clause 1.1 (§3.4.6.2) and derivation-ok-
// restriction clause 1 (§3.4.6.3).
func (p *producer) complexTypeFinal(ctElem *Element) []xsd.DerivationMethod {
	return p.effectiveDerivationSet(ctElem, "final", "finalDefault", complexTypeKeywords)
}

// simpleTypeFinal maps a <simpleType>'s final attribute into {final} (§3.16.2.1
// map.std.common's ·FS·), over the four-keyword simple-type set.
//
// It applies to an ANONYMOUS inline <simpleType> as well as a named one: the
// rule is stated for every Simple Type Definition, and although the schema for
// schema documents prohibits final= on the local form, an absent local attribute
// still falls through to the ancestor <schema>'s finalDefault.
//
// It is read by st-props-correct clause 3 (§3.16.6.1) and cos-st-restricts
// clauses 2.2.1.1 and 3.2.1.1 (§3.16.6.2).
func (p *producer) simpleTypeFinal(stElem *Element) []xsd.DerivationMethod {
	return p.effectiveDerivationSet(stElem, "final", "finalDefault", simpleTypeFinalKeywords)
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
	return xsd.NewAttributeDeclaration(elem.Loc(), qname, xsd.TypeDefinitionRef{Name: typeName}, xsd.NewAttributeGlobalScope(), vc, false, nil)
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

// facetKindOf maps a PLAIN-LEXICAL constraining-facet element's local name to its
// [xsd.FacetKind], delegating the name↔kind bijection to
// [builtin.FacetKindByName] rather than retyping it: that one scan covers §4.3's
// whole constraining-facet set including the precisionDecimal extension facets
// maxScale/minScale (xsd-precisionDecimal.md §4.2/§4.3), which have exactly the
// totalDigits shape — {value} is the value attribute's xs:integer lexical form,
// {fixed} the fixed attribute's boolean — and so need no arm of their own here
// (#323).
//
// The two non-lexical kinds are excluded: enumeration's and assertions' {value}
// is not a lexical string (§4.3.5/§4.3.13), so xsd.NewFacet panics on both, and
// restrictionFacets handles them separately (assertion through
// xsd.NewAssertionsFacet, enumeration by rejection) in checks that run strictly
// above this lookup. Excluding them HERE too is belt-and-suspenders, not
// redundant: the bridge table spells the assertions facet in the plural, so the
// singular <assertion> element those upstream checks intercept would not shield
// a schema's literal <assertions> child from reaching NewFacet.
func facetKindOf(local string) (xsd.FacetKind, bool) {
	kind, ok := builtin.FacetKindByName(builtin.FacetName(local))
	if !ok || kind == xsd.FacetEnumeration || kind == xsd.FacetAssertions {
		return 0, false
	}
	return kind, true
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
