// Package codegen emits Go type declarations from a compiled schema set. It
// emits no decode or encode code: that arrives with package codec (M10), whose
// package doc states that contract.
//
// # Entry point
//
//	type Target struct { Schema *xsd.Schema; Dir, Package, Name string }
//	func Generate(targets []Target) error
//	func CheckTargets(targets []Target) []error
//
// Each target carries one finalized [xsd.Schema] — what parser.Parse
// assembles from a root location and its composition closure; the
// §3.17.1 Schema component is already multi-namespace-capable, so there
// is no separate schema-set type. Each target emits one package into its
// own directory. [Generate] states the phases, the decline and destination
// rules, and the file it writes; [CheckTargets] states which targets it
// refuses whatever their Schemas hold.
//
// # Coverage
//
// Generate emits, in document order, one Go type per top-level element
// declaration, one per anonymous complex type nested in its type definition,
// one per named complex type a local element declaration reaches, one per
// content model group whose particle occurs other than exactly once, and one
// branch struct per alternative of each choice it emits as a sealed
// interface — an <element ref> particle gets none of its own, its field
// holding the type emitted for the top-level element declaration it names —
// then, after every other declaration and in the order the walk
// first needs each, one datatype struct per date, time or duration datatype
// a declaration's type or a value field is declared over and the four
// raw-content structs once an element's type is xs:anyType or a sequence
// holds a wildcard particle, and declines everything outside this slice:
//
//   - An element whose {type definition} is an atomic or list simple type
//     gets a named type over the atomic Go type an atomic simple type gets,
//     or over a slice of one. For an atomic type, the nearest of the eight
//     bounded integer types on the type's {base type definition} chain
//     (§3.16.1), the type itself first, gives the Go integer whose range is
//     exactly its value space (Datatypes §3.4.16–§3.4.19, §3.4.21–§3.4.24):
//     long gives int64, int int32, short int16, byte int8, unsignedLong
//     uint64, unsignedInt uint32, unsignedShort uint16, unsignedByte uint8.
//     Failing one, a chain that reaches xs:integer gives big.Int, which holds
//     every integer (Datatypes §3.4.13): xs:integer, the four unbounded
//     integer types and every type derived from them other than through the
//     eight bounded ones; and a chain that reaches dateTimeStamp,
//     dayTimeDuration or yearMonthDuration gives that builtin's datatype
//     struct. Failing that, the type's {primitive type definition} (Datatypes
//     §4.1.1) chooses: string and anyURI give string, boolean gives bool,
//     float gives float32 and double float64, whose IEEE 754 binary32 and
//     binary64 value spaces (Datatypes §3.3.4.1, §3.3.5.1) those Go types
//     hold exactly, decimal gives big.Rat, which holds every decimal value
//     exactly (Datatypes §3.3.3), and dateTime, time, date, gYearMonth, gYear,
//     gMonthDay, gDay, gMonth and duration each give their datatype struct.
//     That Go integer, string, bool, float32, float64, big.Int, big.Rat or
//     datatype struct is the type's atomic Go type.
//   - A datatype struct has one field per property its datatype's values
//     carry (Datatypes §3.3.6.1, §3.3.7.1–§3.3.14.1, §3.4.26–§3.4.28), in
//     the order of the seven-property model (§D.2.1) or of duration's two
//     properties: Year a big.Int; Month, Day, Hour and Minute a uint8; Second
//     a big.Rat; TimezoneOffset, in minutes, an int16 for dateTimeStamp,
//     whose values always carry one, and a *int16, nil when absent, for the
//     others; Months a big.Int and Seconds a big.Rat, duration having both,
//     dayTimeDuration Seconds alone and yearMonthDuration Months alone. It
//     holds every value of its datatype exactly, in the local
//     property values the model stores rather than converted to UTC
//     (§D.2.1). It also holds struct values no value maps to, such as a Month
//     of 13, a Day beyond its month's last, or Months and Seconds of
//     opposite signs, and it represents neither its datatype's equality nor
//     its order. The generated file imports math/big exactly when a
//     declaration names it: a big.Int or big.Rat, or a datatype struct, all
//     but GMonthDay, GDay and GMonth holding one.
//   - A list type (Datatypes §2.4.1.2) gives a slice of the atomic Go type
//     its {item type definition} (§3.16.1) gives, a restriction of a list
//     taking its base's item (§3.16.2.3); the list declines exactly when its
//     item does, by the item's construct. Wherever a field holds a list, the
//     pointer and slice rules below apply to that slice as to any atomic Go
//     type; in a field that is not a pointer, nil and the empty slice are one
//     value, the empty list. Facets are not represented and do not change an
//     atomic Go type, and a named user simple type gets no Go type of its
//     own.
//   - An element whose {type definition} is xs:anyType (§3.4.7
//     any-type-itself), named by its type attribute or the default for an
//     element declared with no type and no substitutionGroup (§3.3.2), holds
//     its raw content. A top-level element gets a named type over the struct
//     AnyType. A local element's field, and a choice alternative's branch,
//     holds AnyType, by the occurrence rules below. This is xs:anyType's
//     alone: a user complex type of the same shape, or derived from
//     xs:anyType, is a complex type as the next bullet states. AnyType holds
//     the element's [attributes], xsi: ones included, in document order as
//     AnyAttribute values of Space, Local and Value, at most one per expanded
//     name, which the struct does not enforce; then its mixed content as Text,
//     the characters before the first child element, "" for none, and
//     Children, one AnyChild per child element in order, holding that child as
//     Element and as Tail the characters after it, before the next child or
//     the end, so that each gap between children is one string. An AnyElement
//     holds a child's expanded name, Space and Local, and its own content as
//     an AnyType: a wildcard's match is held raw, whether or not a
//     declaration or an xsi:type would govern it. Namespace declarations are
//     not represented, so a prefix inside a value, such as an xsi:type's, does
//     not resolve from the struct. AnyType reaches itself only through the
//     slice Children, so no element of type xs:anyType closes a struct-value
//     recursion, and none of the four names math/big.
//   - An element whose {type definition} is a complex type other than
//     xs:anyType, named or anonymous, gets a type of its own when the type is
//     a restriction of xs:anyType, implicit (§3.4.2.3.2 dcl.ctd.ctcc.implicit)
//     or explicit (<complexContent><restriction base="xs:anyType">, §3.4.2.3.1
//     dcl.ctd.ctcc.explicit), a <complexContent> restriction of any other
//     complex type, or a type with simple content (§3.4.1), by <simpleContent>
//     extension or restriction of any base (§3.4.2.2 dcl.ctd.ctsc). The type
//     is built from its own {content type} (§3.4.2.3.3 dcl.ctd.ctcc.common,
//     §3.4.2.2 dcl.ctd.ctsc) and {attribute uses}, which hold the uses it
//     inherits, all of an extension's base's and those of a restriction's
//     base it neither overrides nor prohibits (§3.4.2.4 dcl.ctd.attuses), and
//     has no Go-level relation to its base's: no embedding, no conversion. The
//     type has no {attribute wildcard}, its own or one an extension inherits
//     (§3.4.2.5 dcl.ctd.anyatt), no {assertions}, inherited ones included
//     (§3.4.2.1 dcl.ctd.common), and {abstract} false, and has one of three
//     contents: empty content ({variety} empty, §3.4.1); simple content
//     ({variety} simple), which gives a struct of the value field the next
//     bullet states and the attribute fields; or element-only
//     content (§3.4.2.3.3 dcl.ctd.ctcc.common) that has no {open content} and
//     whose {particle}, which carries the <sequence>'s or <choice>'s own
//     {min occurs} and {max occurs} (§3.9.1), holds a sequence of element
//     and wildcard particles (§3.10.2.1 dcl.wc.p) and of sequence particles
//     occurring exactly once that hold the same, or a choice of element
//     particles: element particles are local element declarations, and
//     <element ref> particles, whose {term} is the top-level declaration the
//     ref names (§3.3.2.4). A sequence gives a struct, and so does empty
//     content, with no particle field; a struct with no field is spelled
//     struct{}.
//   - Simple content gives a struct whose first field, the value field, is
//     named from the local name value and holds the atomic Go type, or the
//     slice, its {simple type definition} gives as an element's {type
//     definition} does, never a pointer to it; the value field declines
//     exactly when that type does, by the type's construct. That type is the
//     base itself for an extension of a simple type, the base's own for an
//     extension of a complex type, and for a restriction an anonymous
//     restriction of the base's own or of an inline <simpleType>, whose
//     facets are not represented (§3.4.2.2 dcl.ctd.ctsc).
//   - A choice gives a sealed interface, the closed-sum exception: an
//     interface whose one method is unexported, "is" plus the interface's
//     name, and one branch struct per alternative in particle order, emitted
//     after the interface, each holding the one field the alternative's
//     particle gives and implementing the method; consumers type-switch over
//     the branches, and "N pointer fields, exactly one non-nil" never
//     appears. A choice is accepted only when one occurrence of it is not
//     emptiable: it has {particles}, and each has {min occurs} at least 1,
//     so that a particle of it with {min occurs} 1 is not emptiable
//     (§3.8.6.6 cos-choice-range, §3.9.6.3 cos-group-emptiable).
//   - A {particle} that occurs other than exactly once gives a struct whose
//     first field holds what one occurrence of its model group gives — a
//     struct of the sequence's particle fields alone, or the choice's sealed
//     interface — by the occurrence rules below, followed by the attribute
//     fields; the field and that type are named as if from the local name
//     sequence or choice ("Naming"). The occurrence's fields lie behind that
//     slice, pointer or interface, so no struct-value recursion passes
//     through it.
//   - Each particle of a sequence gives one field, in particle order, but a sequence particle
//     occurring exactly once, which gives in its place the fields its own particles give: a local
//     element with an atomic or list simple type gets the atomic Go type, or the slice, an
//     element's type is declared over, one with an anonymous complex type of the same shape gets a
//     struct or interface of its own, emitted after its parent. One with a named complex type of
//     the same shape gets the struct or interface emitted for that type, once per expanded name,
//     after the parent of the first local element of that type, and shared by every local element
//     of that type in any parent, the type's own included; a top-level element of that type keeps
//     the type named from the element. A type nested in a named complex type is emitted once per
//     path of expanded names from that type, after the first element whose walk reaches it, and
//     every element of that type, top-level or local, shares it; two sibling local elements whose
//     expanded names differ only in {target namespace} (§3.3.2.3), a qualified x beside an
//     unqualified x, get a type each. An <element ref> particle gets the type emitted for the
//     top-level element declaration it names, named from that element whatever its type definition,
//     and its field is named from that declaration's local name. A wildcard particle gives one
//     field named Any by "Naming", holding AnyElement by the occurrence rules below. Neither its
//     {namespace constraint} nor its {process contents} (§3.10.1) is represented: the field holds
//     elements the wildcard would not admit (cvc-wildcard), and holds every match raw whether a
//     strict or lax wildcard would assess it; enforcing either is the validator's. A named complex
//     type, or a top-level element through <element ref> particles, that reaches itself through
//     elements that each occur exactly once in a sequence that is a {particle} occurring exactly
//     once or in a sequence spliced into one, and whose types are not choices, declines: its Go
//     type would hold itself as a struct value. {max occurs} above 1 or unbounded gives a slice
//     whatever {min occurs} is, of a list's slice too ([][]T); {min occurs} 0 with {max occurs} 1
//     gives a pointer to a struct, atomic Go type, or list's slice (*[]T), and the bare interface,
//     never a pointer to one, nil meaning absent.
//   - Each attribute use of a type emitted as a struct gives one field after the particle
//     fields or the value field, if any, in the order [xsd.ComplexType.AttributeUses] returns
//     its {attribute uses} (§3.4.2.4). The field is named from the attribute declaration's
//     local name, so a qualified and an unqualified attribute of one local name, or an
//     attribute named like an element of the struct or like the value field, take the
//     suffix "Naming" states, the later field taking it: an attribute value beside the
//     value field gives Value2. Its {type definition} must be an atomic or list simple type and gives an
//     atomic Go type, or a slice, as an element's does; {required} true gives that type,
//     false a pointer to it (*[]T for a list), nil meaning absent. The use's effective
//     value constraint (§3.5.4 key-evc) must be absent, whether the use or a global declaration it
//     references carries the default or fixed value. A choice {particle} that occurs exactly once is
//     accepted only when its type has no {attribute uses}.
//   - Every element declaration emitted from has {nillable} and {abstract}
//     false and no {value constraint}, {type table} or {substitution group
//     affiliations}. Its {identity-constraint definitions} are accepted and
//     not represented: an identity constraint restricts the values an
//     instance holds and adds no content or attribute (§3.11.1), so the
//     element emits exactly as it would without one, and enforcing it is
//     the validator's.
//
// ExampleGenerate prints the whole file Generate writes for a schema inside
// this slice.
//
// GAP(codegen): every other construct declines: the all compositor, an
// emptiable choice, a choice with no {particles}, nested model groups but a
// sequence occurring exactly once in a sequence, <group ref> particles, a wildcard
// alternative of a choice, a particle with {max occurs} 0, attribute wildcards,
// attribute uses on a type whose content is a choice occurring exactly once, attribute
// uses with an effective value constraint, mixed content, complex-content extension, a
// named complex type, or a top-level element through <element ref> particles, reaching
// itself through struct values, the special simple types, union types, a list whose
// {item type definition} declines, and every primitive the first bullet does not map,
// for an element or an attribute alike. #2900 owns lifting them.
//
// # Determinism
//
// The output is a function of the schema and of each document's location
// relative to the target's Dir, so two trees laid out alike under different
// absolute roots produce byte-identical output; "Naming" states where a
// header's location is not relative. No map iteration anywhere near the
// emitter; emission order is the walk's, which document order alone fixes, as
// the first paragraph states (datatype structs and raw-content structs last,
// in the order the walk first needs each). Golden-file tests pin the output.
//
// # Naming
//
// A single namer component owns every XSD-name → Go-identifier decision.
// Each XSD local name becomes an exported identifier: '_' and every rune Go
// does not admit in an identifier break words, every word starts upper-case,
// and a result with no upper-case first letter is prefixed with
// X. That upper-case start is also what keeps every identifier clear of the
// Go keywords. A top-level element's type is named from the element, its name
// claimed when the walk first reaches the element: at its own position, or at
// the first <element ref> particle to it met before that. A named
// complex type a local element reaches is named from the type (complex type
// ct → Ct), whichever local elements have type ct. An anonymous type nested
// in either is named by the path of names from its nearest named ancestor:
// the top-level element when the element's own type is anonymous (element
// shipTo under element purchaseOrder → PurchaseOrderShipTo), the complex type
// when that type is named (element x in complex type ct → CtX, whichever
// elements have type ct). The type for one occurrence of a content model
// group is named by the path with the local name sequence or choice last
// (CtSequence), and the types nested in its particles as if it were absent. A
// choice's branch struct is named from its interface's name plus the
// alternative's local name (alternative a of interface First → FirstA), so
// two top-level elements of one named choice type get distinct interfaces and
// branches. A residual collision — two names folding to one identifier, among
// a package's types or one struct's fields — is disambiguated
// deterministically by document order: the later name takes the smallest free
// numeric suffix from 2, so a top-level element ct and a complex type ct
// reached locally take Ct in the order they are first reached. A datatype
// struct is named from its datatype's local name (dateTime → DateTime,
// gYearMonth → GYearMonth, dayTimeDuration → DayTimeDuration), claimed when
// the walk first needs it: for a top-level element of a simple type, ahead
// of that element's own type, so that a top-level element date of type
// xs:date names its type Date2; for a field of a struct, the value field and
// attribute fields included, after that struct's own name, so that a
// top-level element date whose complex type extends xs:date names its type
// Date and the datatype struct Date2. The raw-content structs are named AnyType,
// AnyAttribute, AnyChild and AnyElement, claimed together in that order when
// the walk first reaches a declaration of type xs:anyType or a wildcard
// particle, ahead of that declaration's own type, so that a top-level element
// anyType with no type names its type AnyType2. A wildcard particle's field
// is named as if from local name any. Every other generated type's header
// comment records the QName and the schema Loc of the declaration it is named
// from: the element declaration, or the complex type definition for a type
// named from one. A datatype struct's header records its datatype's QName
// alone, a builtin having no schema document, then a second line stating what
// its fields' names do not, when there is any: TimezoneOffset's unit and
// whether it may be absent, or that duration's Months and Seconds share one
// sign. A raw-content struct's header records xs:anyType's QName alone, then,
// for AnyType and AnyChild, a second line stating what Text or Tail holds. A
// Loc whose URI filepath.IsAbs accepts renders relative to the target's Dir
// made absolute against the working directory, slash-separated and climbing
// with ".." out of Dir. A document outside the tree laid out alike, such as
// one an absolute schemaLocation names, renders a climbing path that differs
// with that tree's depth below the filesystem root. Any other URI renders as
// is, a URL or a relative loader key alike, as does one with no path relative
// to Dir, such as one on another Windows volume. Symbolic links are not
// resolved.
//
// # Planned (M9 — not yet implemented)
//
// Shortening an anonymous type's path to only as much of it as uniqueness
// requires. Cross-target type references import across the generated
// packages, in place of each target declaring the types of a document its
// Schema shares with another.
//
// Type narrowing in interfaces, the generated-code idiom:
//
//   - Generated readers/views expose the narrowest interface a consumer
//     needs; optionality and nillability are modeled in the types, not
//     in comments.
//   - Simple-content fields of a type "# Coverage" maps to no Go type are
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
