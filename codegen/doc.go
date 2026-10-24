// Package codegen emits Go type declarations from a compiled schema set. It
// emits no decode or encode code: that arrives with package codec (M10), whose
// package doc states that contract.
//
// # Entry point
//
//	type Target struct { Schema *xsd.Schema; Dir, Package string }
//	func Generate(targets []Target) error
//
// Each target carries one finalized [xsd.Schema] — what parser.Parse
// assembles from a root location and its composition closure; the
// §3.17.1 Schema component is already multi-namespace-capable, so there
// is no separate schema-set type. Each target emits one package into its
// own directory. [Generate] states the phases, the decline and destination
// rules, and the file it writes.
//
// # Coverage
//
// Generate emits, in document order, one Go type per top-level element
// declaration, one per anonymous complex type nested in its type definition,
// and one branch struct per alternative of each choice it emits as a sealed
// interface, and declines everything outside this slice:
//
//   - An element whose {type definition} is an atomic simple type gets a
//     named type over a Go builtin. The nearest of the eight bounded integer
//     types on the type's {base type definition} chain (§3.16.1), the type
//     itself first, gives the Go integer whose range is exactly its value
//     space (Datatypes §3.4.16–§3.4.19, §3.4.21–§3.4.24): long gives int64,
//     int int32, short int16, byte int8, unsignedLong uint64, unsignedInt
//     uint32, unsignedShort uint16, unsignedByte uint8. Failing one, the
//     type's {primitive type definition} (Datatypes §4.1.1) chooses: string
//     and anyURI give string, boolean gives bool. Facets are not represented,
//     and a named user simple type gets no Go type of its own.
//   - An element whose {type definition} is a complex type, named or
//     anonymous, gets a type of its own when the type is a restriction of
//     xs:anyType, implicit (§3.4.2.3.2 dcl.ctd.ctcc.implicit) or explicit
//     (<complexContent><restriction base="xs:anyType">, §3.4.2.3.1
//     dcl.ctd.ctcc.explicit), with element-only content (§3.4.2.3.3
//     dcl.ctd.ctcc.common) and no {open content}, {attribute wildcard},
//     {assertions} or {abstract}, whose {particle} occurs exactly once and
//     holds a sequence or a choice of local element declarations. A sequence
//     gives a struct.
//   - A choice gives a sealed interface, the closed-sum exception: an
//     interface whose one method is unexported, "is" plus the interface's
//     name, and one branch struct per alternative in particle order, emitted
//     after the interface, each holding the one field the alternative's
//     particle gives and implementing the method; consumers type-switch over
//     the branches, and "N pointer fields, exactly one non-nil" never
//     appears. A choice is accepted only when its particle is not emptiable
//     (§3.9.6.3 cos-group-emptiable): it has {particles}, and each has
//     {min occurs} at least 1 (§3.8.6.6 cos-choice-range).
//   - Each particle of a sequence gives one field, in particle order: a
//     local element with an atomic simple type gets its builtin, one with an
//     anonymous complex type of the same shape gets a struct or interface of
//     its own, emitted after its parent. A type nested in a named complex
//     type is emitted once per path of expanded names from that type, after
//     the first top-level element of that type, and every element of that
//     type shares it; two sibling local elements whose expanded names differ
//     only in {target namespace} (§3.3.2.3), a qualified x beside an
//     unqualified x, get a type each. {max occurs} above 1 or unbounded
//     gives a slice whatever {min occurs} is; {min occurs} 0 with {max
//     occurs} 1 gives a pointer to a struct or builtin, and the bare
//     interface, never a pointer to one, nil meaning absent.
//   - Each attribute use of a sequence's type gives one field after the
//     particle fields, in the order [xsd.ComplexType.AttributeUses] returns
//     its {attribute uses} (§3.4.2.4). The field is named from the
//     attribute declaration's local name, so a qualified and an unqualified
//     attribute of one local name, or an attribute named like an element of
//     the struct, take the suffix "Naming" states, the later field taking it.
//     Its {type definition} must be an atomic simple type and gives a builtin as
//     an element's does; {required} true gives the builtin, false a pointer to
//     it, nil meaning absent. The use's effective value constraint (§3.5.4
//     key-evc) must be absent, whether the use or a global declaration it
//     references carries the default or fixed value. A choice is accepted only
//     when its type has no {attribute uses}.
//   - Every element declaration emitted from has {nillable} and {abstract}
//     false and no {value constraint}, {type table}, {substitution group
//     affiliations} or {identity-constraint definitions}.
//
// GAP(codegen): every other construct declines: the all compositor, an
// emptiable choice, a choice with no {particles}, nested model groups,
// <group ref> and <element ref> particles, wildcards, a particle with {max
// occurs} 0, attribute wildcards, attribute uses on a type whose content is a
// choice, attribute uses with an effective value constraint, mixed, simple
// and empty content, extension, restriction of any base but xs:anyType, a
// local element with a named complex type, xs:anyType, the special simple
// types, list and union types, and every primitive but the three above —
// decimal among them, so xs:integer, the four unbounded integer types and
// every type derived from them other than through the eight bounded ones
// decline, as do float and double — for an element or an attribute alike.
// #2900 owns lifting them.
//
// # Determinism
//
// Identical schema input produces byte-identical output. No map
// iteration anywhere near the emitter; emission order is document order.
// Golden-file tests pin the output.
//
// # Naming
//
// A single namer component owns every XSD-name → Go-identifier decision.
// Each XSD local name becomes an exported identifier: '_' and every rune Go
// does not admit in an identifier break words, every word starts upper-case,
// and a result with no upper-case first letter is prefixed with
// X. That upper-case start is also what keeps every identifier clear of the
// Go keywords. A top-level element's type is named from the element. An
// anonymous type nested in it is named by the path of names from its nearest
// named ancestor: the top-level element when the element's own type is
// anonymous (element shipTo under element purchaseOrder → PurchaseOrderShipTo),
// the complex type when that type is named (element x in complex type ct → CtX,
// whichever elements have type ct). A choice's branch struct is named from its
// interface's name plus the alternative's local name (alternative a of
// interface First → FirstA), so two top-level elements of one named choice
// type get distinct interfaces and branches. A residual collision — two
// names folding to one identifier, among a package's types or one struct's
// fields — is disambiguated deterministically by document order: the later
// name takes the smallest free numeric suffix from 2. Every generated
// type's header comment records the element declaration's QName and its
// schema Loc.
//
// # Planned (M9 — not yet implemented)
//
// Shortening an anonymous type's path to only as much of it as uniqueness
// requires. Cross-target type references import across the generated
// packages.
//
// Type narrowing in interfaces, the generated-code idiom:
//
//   - Generated readers/views expose the narrowest interface a consumer
//     needs; optionality and nillability are modeled in the types, not
//     in comments.
//   - Simple-content fields of a type "# Coverage" maps to no Go builtin are
//     typed by the chosen backend's representation (strict or native, per
//     generation option). A field of a type "# Coverage" maps keeps that
//     mapping whichever backend is chosen.
//
// # The Emitter seam (value.Emitter; API frozen in M9; not yet declared here)
//
// Backends contribute specialized decode/encode source for their value
// representations — parsing straight from the reader's byte window into
// the target field, facet checks inlined, no intermediate string, no
// boxed value. A backend without an Emitter falls back to emitting calls
// into codec's runtime path. Emitted fast paths carry the same pipeline
// stage and rule-ID metadata as the runtime path; codec's differential
// tests hold the two to identical behavior.
package codegen
