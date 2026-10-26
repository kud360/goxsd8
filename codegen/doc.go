// Package codegen emits Go type declarations from a compiled schema set. It
// emits no decode or encode code: that arrives with package codec (M10), whose
// package doc states that contract.
//
// # Entry point
//
//	type Target struct { Schema *xsd.Schema; Dir, Package, Name string }
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
// one per named complex type a local element declaration reaches, and one
// branch struct per alternative of each choice it emits as a sealed
// interface, then, after every other declaration and in the order the walk
// first needs each, one datatype struct per date, time or duration datatype
// a declaration's type is declared over, and declines everything outside this
// slice:
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
//   - An element whose {type definition} is a complex type, named or
//     anonymous, gets a type of its own when the type is a restriction of
//     xs:anyType, implicit (§3.4.2.3.2 dcl.ctd.ctcc.implicit) or explicit
//     (<complexContent><restriction base="xs:anyType">, §3.4.2.3.1
//     dcl.ctd.ctcc.explicit), with no {attribute wildcard}, {assertions} or
//     {abstract}, and with one of two contents: empty content ({variety}
//     empty, §3.4.1); or element-only content (§3.4.2.3.3 dcl.ctd.ctcc.common)
//     that has no {open content} and whose {particle} occurs exactly once and
//     holds a sequence or a choice of local element declarations. A sequence
//     gives a struct, and so does empty content, with no particle field; a
//     struct with no field is spelled struct{}.
//   - A choice gives a sealed interface, the closed-sum exception: an
//     interface whose one method is unexported, "is" plus the interface's
//     name, and one branch struct per alternative in particle order, emitted
//     after the interface, each holding the one field the alternative's
//     particle gives and implementing the method; consumers type-switch over
//     the branches, and "N pointer fields, exactly one non-nil" never
//     appears. A choice is accepted only when its particle is not emptiable
//     (§3.9.6.3 cos-group-emptiable): it has {particles}, and each has
//     {min occurs} at least 1 (§3.8.6.6 cos-choice-range).
//   - Each particle of a sequence gives one field, in particle order: a local element with
//     an atomic or list simple type gets the atomic Go type, or the slice, an
//     element's type is declared over, one with an anonymous complex type of the same shape
//     gets a struct or interface of its own, emitted after its parent. One with a named
//     complex type of the same shape gets the struct or interface emitted for that type,
//     once per expanded name, after the parent of the first local element of that type, and
//     shared by every local element of that type in any parent, the type's own included; a
//     top-level element of that type keeps the type named from the element. A type nested in
//     a named complex type is emitted once per path of expanded names from that type, after
//     the first element whose walk reaches it, and every element of that type, top-level or
//     local, shares it; two sibling local elements whose expanded names differ only in
//     {target namespace} (§3.3.2.3), a qualified x beside an unqualified x, get a type each.
//     A named complex type that reaches itself through elements that each occur exactly once
//     in a sequence, and whose types are not choices, declines: its Go type would hold
//     itself as a struct value. {max occurs} above 1 or unbounded gives a slice whatever
//     {min occurs} is, of a list's slice too ([][]T); {min occurs} 0 with {max occurs} 1
//     gives a pointer to a struct, atomic Go type, or list's slice (*[]T), and
//     the bare interface, never a pointer to one, nil meaning absent.
//   - Each attribute use of a type with sequence or empty content gives one
//     field after the particle fields, if any, in the order [xsd.ComplexType.AttributeUses]
//     returns its {attribute uses} (§3.4.2.4). The field is named from the attribute
//     declaration's local name, so a qualified and an unqualified attribute of one local
//     name, or an attribute named like an element of the struct, take the suffix "Naming"
//     states, the later field taking it. Its {type definition} must be an atomic or list
//     simple type and gives an atomic Go type, or a slice, as an element's
//     does; {required} true gives that type, false a pointer to it (*[]T for a list), nil
//     meaning absent. The use's effective value constraint (§3.5.4 key-evc) must be absent,
//     whether the use or a global declaration it references carries the default or fixed
//     value. A choice is accepted only when its type has no {attribute uses}.
//   - Every element declaration emitted from has {nillable} and {abstract}
//     false and no {value constraint}, {type table} or {substitution group
//     affiliations}. Its {identity-constraint definitions} are accepted and
//     not represented: an identity constraint restricts the values an
//     instance holds and adds no content or attribute (§3.11.1), so the
//     element emits exactly as it would without one, and enforcing it is
//     the validator's.
//
// GAP(codegen): every other construct declines: the all compositor, an
// emptiable choice, a choice with no {particles}, nested model groups,
// <group ref> and <element ref> particles, wildcards, a particle with {max
// occurs} 0, attribute wildcards, attribute uses on a type whose content is a
// choice, attribute uses with an effective value constraint, mixed and
// simple content, extension, restriction of any base but xs:anyType, a
// named complex type reaching itself through struct values, xs:anyType, the
// special simple types, union types, a list whose {item type definition}
// declines, and every primitive the first bullet does not map, for an element
// or an attribute alike. #2900 owns lifting them.
//
// # Determinism
//
// The output is a function of the schema and of each document's location
// relative to the target's Dir, so two trees laid out alike under different
// absolute roots produce byte-identical output; "Naming" states where a
// header's location is not relative. No map iteration anywhere near the
// emitter; emission order is the walk's, which document order alone fixes, as
// the first paragraph states (datatype structs last, in the order the walk
// first needs each). Golden-file tests pin the output.
//
// # Naming
//
// A single namer component owns every XSD-name → Go-identifier decision.
// Each XSD local name becomes an exported identifier: '_' and every rune Go
// does not admit in an identifier break words, every word starts upper-case,
// and a result with no upper-case first letter is prefixed with
// X. That upper-case start is also what keeps every identifier clear of the
// Go keywords. A top-level element's type is named from the element. A named
// complex type a local element reaches is named from the type (complex type
// ct → Ct), whichever local elements have type ct. An anonymous type nested
// in either is named by the path of names from its nearest named ancestor:
// the top-level element when the element's own type is anonymous (element
// shipTo under element purchaseOrder → PurchaseOrderShipTo), the complex type
// when that type is named (element x in complex type ct → CtX, whichever
// elements have type ct). A choice's branch struct is named from its
// interface's name plus the alternative's local name (alternative a of
// interface First → FirstA), so two top-level elements of one named choice
// type get distinct interfaces and branches. A residual collision — two
// names folding to one identifier, among a package's types or one struct's
// fields — is disambiguated deterministically by document order: the later
// name takes the smallest free numeric suffix from 2, so a top-level element
// ct and a complex type ct reached locally take Ct in the order they are
// first reached. A datatype struct is named from its datatype's local name
// (dateTime → DateTime, gYearMonth → GYearMonth, dayTimeDuration →
// DayTimeDuration), claimed when the walk first reaches a declaration whose
// type needs it, ahead of that declaration's own type, so that a top-level
// element date of type xs:date names its type Date2. Every other generated
// type's header comment records the QName and the schema Loc of the
// declaration it is named from: the element declaration, or the complex type
// definition for a type named from one. A datatype struct's header records
// its datatype's QName alone, a builtin having no schema document, then a
// second line stating what its fields' names do not, when there is any:
// TimezoneOffset's unit and whether it may be absent, or that duration's
// Months and Seconds share one sign. A Loc whose URI filepath.IsAbs accepts
// renders relative to the target's Dir made absolute against the working
// directory, slash-separated and climbing with ".." out of Dir. A document
// outside the tree laid out alike, such as one an absolute schemaLocation
// names, renders a climbing path that differs with that tree's depth below
// the filesystem root. Any other URI renders as is, a URL or a relative
// loader key alike, as does one with no path relative to Dir, such as one on
// another Windows volume. Symbolic links are not resolved.
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
