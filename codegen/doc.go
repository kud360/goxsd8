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
// Supported, each construct emitted from or accepted as the rules below state:
//
//   - element declarations, top-level and local
//   - <element ref> particles and substitution groups
//   - the special types xs:anySimpleType and xs:anyAtomicType
//   - atomic simple types
//   - list simple types
//   - union simple types
//   - xs:anyType
//   - complex types, named or anonymous, derived by restriction or extension
//   - empty, simple and element-only content
//   - sequences, choices and all groups, inline or through <group ref>
//   - optional and repeated particles
//   - wildcard particles
//   - attribute uses and attribute wildcards
//   - facets, {assertions}, value constraints and identity constraints
//
// Declined today, each construct named as [Construct.String] prints it for a
// [DeclineError]'s [DeclineError.Construct], for display only: the names may
// change in any release, so code compares [Construct] values, never a name.
//
//   - element {nillable}
//   - element {abstract}
//   - {type table}
//   - absent {variety}
//   - no {primitive type definition}
//   - uncovered primitive
//   - choice with attributes
//   - choice with no {particles}
//   - emptiable choice
//   - complex type {abstract}
//   - mixed content
//   - {open content}
//   - particle with {max occurs} 0
//   - struct-value recursion
//
// A list or union declines with the Construct its item type or member declines
// with.
//
// GAP(codegen): Generate declines each construct the list above names; #2900 is
// the GitHub issue tracking these lifts.
//
// Generate emits, in document order, one Go type per top-level element
// declaration, one per anonymous complex type nested in its type definition or
// in a model group definition a <group ref> particle names, one per named
// complex type a local element declaration reaches, one per place a struct's or
// a branch struct's particles hold a choice or a sequence occurring other than
// exactly once — a sequence occurring exactly once in a sequence or as a
// choice's alternative, and an all group occurring exactly once in an all
// group, giving the places of its own particles, and a complex type's own
// content particle, a sequence, a choice or an all group, counting only when it
// occurs other than exactly once — so a particle that a model group definition
// or an extension's base holds gets one in each struct that holds it, each
// shared as an anonymous complex type nested beside it is, and one branch
// struct per alternative of each choice it emits as a sealed interface — an
// <element ref> particle gets none of its own, its field holding the type
// emitted for the top-level element declaration it names, or, when that
// declaration's substitution group holds another declaration, one sealed
// interface per such head with one branch struct per declaration of its group —
// then, after every other declaration and in the order the walk first needs
// each, one datatype struct per date, time or duration datatype, and one for
// QName and NOTATION together, a declaration's type or a value field is
// declared over, the four raw-content structs once an element's type is
// xs:anyType, a model group holds a wildcard particle, or a type has an
// {attribute wildcard}, and one sealed interface per union simple type a
// declaration's type, a value field, a list's item or a union's member is
// declared over, each followed by one variant type per basic member, and
// declines everything outside this slice:
//
//   - An element whose {type definition} is a simple type gets a named type
//     over string for a special type (Datatypes §2.4.2 dt-special),
//     xs:anySimpleType or xs:anyAtomicType, known by identity whatever its
//     {variety}, and otherwise over the atomic Go type an atomic simple type
//     gets, over a slice of one, or over the union's sealed interface. A
//     special type's string holds the literal as a string: neither a value of
//     the type nor an xsi:type naming another type is represented.
//   - For an atomic type, the nearest of the eight bounded integer types on the
//     type's {base type definition} chain (§3.16.1), the type itself first,
//     gives the Go integer whose range is exactly its value space (Datatypes
//     §3.4.16–§3.4.19, §3.4.21–§3.4.24): long gives int64, int int32, short
//     int16, byte int8, unsignedLong uint64, unsignedInt uint32, unsignedShort
//     uint16, unsignedByte uint8. Failing one, a chain that reaches xs:integer
//     gives big.Int, which holds every integer (Datatypes §3.4.13): xs:integer,
//     the four unbounded integer types and every type derived from them other
//     than through the eight bounded ones; and a chain that reaches
//     dateTimeStamp, dayTimeDuration or yearMonthDuration gives that builtin's
//     datatype struct.
//   - Failing that, the type's {primitive type definition} (Datatypes §4.1.1)
//     chooses: string and anyURI give string, boolean gives bool, float gives
//     float32 and double float64, whose IEEE 754 binary32 and binary64 value
//     spaces (Datatypes §3.3.4.1, §3.3.5.1) those Go types hold exactly,
//     decimal gives big.Rat, which holds every decimal value exactly (Datatypes
//     §3.3.3), hexBinary and base64Binary give []byte, which holds every finite
//     sequence of octets exactly (Datatypes §3.3.15.1, §3.3.16.1), nil and the
//     empty slice being one value, no octets, and dateTime, time, date,
//     gYearMonth, gYear, gMonthDay, gDay, gMonth and duration each give their
//     datatype struct, and QName and NOTATION give QName's (Datatypes §3.3.18,
//     §3.3.19). That Go integer, string, bool, float32, float64, big.Int,
//     big.Rat, []byte or datatype struct is the type's atomic Go type.
//   - A date, time or duration datatype struct has one field per property its
//     datatype's values carry (Datatypes §3.3.6.1, §3.3.7.1–§3.3.14.1,
//     §3.4.26–§3.4.28), in the order of the seven-property model (§D.2.1) or of
//     duration's two properties: Year a big.Int; Month, Day, Hour and Minute a
//     uint8; Second a big.Rat; TimezoneOffset, in minutes, an int16 for
//     dateTimeStamp, whose values always carry one, and a *int16, nil when
//     absent, for the others; Months a big.Int and Seconds a big.Rat, duration
//     having both, dayTimeDuration Seconds alone and yearMonthDuration Months
//     alone.
//   - A date, time or duration datatype struct holds every value of its
//     datatype exactly, in the local property values the model stores rather
//     than converted to UTC (§D.2.1). It also holds struct values no value maps
//     to, such as a Month of 13, a Day beyond its month's last, or Months and
//     Seconds of opposite signs, and it represents neither its datatype's
//     equality nor its order.
//   - QName's datatype struct, which NOTATION shares, holds the expanded name a
//     literal resolves to (Datatypes §3.3.18, §3.3.19): Space is the namespace
//     name, "" for none, and Local the local part; no prefix is represented. It
//     also holds struct values no value maps to, such as a Local that is not an
//     NCName or, for NOTATION, a name no <notation> declares. The generated
//     file imports math/big exactly when a declaration names it: a big.Int or
//     big.Rat, or a datatype struct, all but GMonthDay, GDay, GMonth and QName
//     holding one.
//   - A list type (Datatypes §2.4.1.2) gives a slice of the Go type its {item
//     type definition} (§3.16.1) gets, an atomic Go type or a union's sealed
//     interface, a restriction of a list taking its base's item (§3.16.2.3);
//     the list declines exactly when its item does, by the item's construct.
//     Wherever a field holds a list, the pointer and slice rules below apply to
//     that slice as to any atomic Go type; in a field that is not a pointer,
//     nil and the empty slice are one value, the empty list. Facets are not
//     represented and do not change the Go type a simple type gets, and a named
//     user simple type other than a union gets no Go type of its own.
//   - A union type (Datatypes §2.4.1.3) gives a sealed interface whose one
//     method is unexported, "is" plus the interface's name, and one variant
//     type per basic member (dt-basicmember), emitted after the interface in
//     basic-member order: its {member type definitions} (§3.16.1), a
//     restriction of a union taking its base's (§3.16.2.1), in order, each
//     member that is itself a union replaced by its own basic members, and each
//     member kept at its first position only, a list member being basic. Each
//     variant is a named type over the Go type its member gets, an atomic Go
//     type or a list's slice, and implements the method; consumers type-switch
//     over the variants.
//   - Each union, named or anonymous, gets one interface however many
//     declarations reach it, and a restriction of a union gets its own. The
//     union declines exactly when one of its basic members does, by that
//     member's construct, at the first declining member in basic-member order.
//   - A union's interface does not enforce the ·active basic member·, the first
//     member in order that accepts a literal: with xs:int first, a string
//     variant can hold "5", which xs:int accepts. Facets on the union itself
//     are not represented. A union with no basic member, which §3.16.1 admits
//     though no schema document yields one
//     (src-union-memberTypes-or-simpleTypes), gets its interface alone, which
//     no variant implements.
//   - An element whose {type definition} is xs:anyType (§3.4.7
//     any-type-itself), named by its type attribute or the default for an
//     element declared with no type and no substitutionGroup (§3.3.2), holds
//     its raw content. A top-level element gets a named type over the struct
//     AnyType. A local element's field, and a choice alternative's branch,
//     holds AnyType, by the occurrence rules below. This is xs:anyType's alone:
//     a user complex type of the same shape, or derived from xs:anyType, is a
//     complex type as the complex-type bullets below state.
//   - AnyType holds the element's [attributes], xsi: ones included, in document
//     order as AnyAttribute values of Space, Local and Value, at most one per
//     expanded name, which the struct does not enforce; then its mixed content
//     as Text, the characters before the first child element, "" for none, and
//     Children, one AnyChild per child element in order, holding that child as
//     Element and as Tail the characters after it, before the next child or the
//     end, so that each gap between children is one string.
//   - An AnyElement holds a child's expanded name, Space and Local, and its own
//     content as an AnyType: a wildcard's match is held raw, whether or not a
//     declaration or an xsi:type would govern it. Namespace declarations are
//     not represented, so a prefix inside a value, such as an xsi:type's, does
//     not resolve from the struct. AnyType reaches itself only through the
//     slice Children, so no element of type xs:anyType closes a struct-value
//     recursion, and none of the four names math/big.
//   - An element whose {type definition} is a complex type other than
//     xs:anyType, named or anonymous, gets a type of its own whatever its
//     derivation, restriction or extension, of complex or simple content
//     (§3.4.2.2 dcl.ctd.ctsc, §3.4.2.3 dcl.ctd.ctcc). The type is built from
//     its own {content type} (§3.4.2.3.3 dcl.ctd.ctcc.common, §3.4.2.2
//     dcl.ctd.ctsc) and {attribute uses}, which hold the uses it inherits, all
//     of an extension's base's and those of a restriction's base it neither
//     overrides nor prohibits (§3.4.2.4 dcl.ctd.attuses), and has no Go-level
//     relation to its base's: no embedding, no conversion.
//   - A complex type's {assertions}, inherited ones included (§3.4.2.1
//     dcl.ctd.common), are accepted and not represented. An assertion is one
//     more condition an instance must satisfy and adds no content or attribute
//     (§3.4.4.2 cvc-complex-type clause 6). So the type emits exactly as it
//     would without them, and enforcing them is the validator's.
//   - The complex type has {abstract} false and one of three contents: empty
//     content ({variety} empty, §3.4.1); simple content ({variety} simple),
//     which gives a struct of the value field stated below and the attribute
//     fields; or element-only content (§3.4.2.3.3 dcl.ctd.ctcc.common) that has
//     no {open content} and whose {particle}, which carries the <sequence>'s,
//     <choice>'s or <all>'s own {min occurs} and {max occurs} (§3.9.1), or a
//     <group ref>'s, whose {term} is the referenced definition's {model group}
//     (§3.7.2), holds a sequence, a choice or an all group of element, wildcard
//     (§3.10.2.1 dcl.wc.p) and model group particles, each model group
//     particle, inline or a <group ref>, a sequence, choice or all group that
//     holds the same: element particles are local element declarations, and
//     <element ref> particles, whose {term} is the top-level declaration the
//     ref names (§3.3.2.4).
//   - A sequence or an all group gives a struct, and so does empty content,
//     with no particle field; a struct with no field is spelled struct{}.
//   - Simple content gives a struct whose first field, the value field, is
//     named from the local name value and holds a special type's string, the
//     atomic Go type, the slice, or the union's sealed interface its {simple
//     type definition} gives as an element's {type definition} does, never a
//     pointer to it; the value field declines exactly when that type does, by
//     the type's construct. That type is the base itself for an extension of a
//     simple type, the base's own for an extension of a complex type, and for a
//     restriction an anonymous restriction of the base's own or of an inline
//     <simpleType>, whose facets are not represented (§3.4.2.2 dcl.ctd.ctsc).
//   - A choice gives a sealed interface, the closed-sum exception: an interface
//     whose one method is unexported, "is" plus the interface's name, and one
//     branch struct per alternative in particle order, emitted after the
//     interface, each holding the fields the alternative's particle gives as a
//     particle of a sequence would — an element or a wildcard its one field, a
//     sequence occurring exactly once the fields of its own particles, any
//     other model group its one field — and implementing the method; consumers
//     type-switch over the branches, and "N pointer fields, exactly one
//     non-nil" never appears.
//   - A choice is accepted only when one occurrence of it is not emptiable: it
//     has {particles}, and none is emptiable ([xsd.Schema.ParticleEmptiable]),
//     so that a particle of it with {min occurs} 1 is not emptiable (§3.8.6.6
//     cos-choice-range, §3.9.6.3 cos-group-emptiable).
//   - An <element ref> particle to a head whose substitution group
//     ([xsd.Schema.SubstitutionGroup], §3.3.6.4 key-eq) holds a declaration
//     other than the head gives a field holding a sealed interface over that
//     group, the closed-sum exception as a choice's is: one method, unexported,
//     "is" plus the interface's name, and one branch struct per declaration of
//     the group, the head included, in the order [xsd.Schema.Elements] returns
//     them, emitted after the interface, each holding one field, named from the
//     declaration's local name, of the type emitted for that top-level
//     declaration, and implementing the method. The field takes the occurrence
//     rules below, the bare interface for {min occurs} 0.
//   - A substitution group's declarations' own types are emitted unchanged,
//     each at its own position, and one interface is emitted per head however
//     many particles reach it.
//   - A head whose {disallowed substitutions} contains substitution
//     (cos-equiv-derived-ok-rec clause 2.1), or whose group is otherwise itself
//     alone, gives the field of an <element ref> particle to it the head's own
//     type. {abstract} plays no part in the group: where the interface is first
//     needed, each declaration of the group is checked in group order as at an
//     <element ref> particle to it, its properties and the shape of its type,
//     so an abstract head or member declines it there.
//   - A {particle} that occurs other than exactly once gives a struct whose
//     first field holds what one occurrence of its model group gives — a struct
//     of the sequence's or all group's particle fields alone, or the choice's
//     sealed interface — by the occurrence rules below, followed by the
//     attribute fields; the field and that type are named as if from the local
//     name sequence, choice or all ("Naming"). A model group particle in a
//     sequence or as a choice's alternative, a choice or a sequence occurring
//     other than exactly once, gives one such field in the same way, a choice
//     occurring exactly once giving the bare interface. The occurrence's fields
//     lie behind that slice, pointer or interface, so no struct-value recursion
//     passes through it.
//   - Each particle of a sequence gives one field, in particle order, but a
//     sequence particle occurring exactly once, inline or a <group ref> to a
//     sequence group, which gives in its place the fields its own particles
//     give, so an extension (§3.4.2.3.3 clause 4.2.3.3) gives its base
//     particle's fields and then its own: a local element with a simple type
//     gets a special type's string, the atomic Go type, the slice, or the
//     interface an element's type is declared over, one with an anonymous
//     complex type of the same shape gets a struct or interface of its own,
//     emitted after its parent.
//   - A local element with a named complex type of the same shape gets the
//     struct or interface emitted for that type, once per expanded name, after
//     the parent of the first local element of that type, and shared by every
//     local element of that type in any parent, the type's own included; a
//     top-level element of that type keeps the type named from the element.
//   - A type nested in a named complex type is emitted once per path of
//     expanded names from that type, after the first element whose walk reaches
//     it, and every element of that type or of a type extending it, top-level
//     or local, shares it; a type nested in a model group definition is emitted
//     once per path of expanded names from that group, after the first element
//     whose walk reaches it, and every <group ref> particle to the group shares
//     it, each element the group declares being one component (§3.7.2); two
//     sibling local elements whose expanded names differ only in {target
//     namespace} (§3.3.2.3), a qualified x beside an unqualified x, get a type
//     each.
//   - An <element ref> particle gets the type emitted for the top-level element
//     declaration it names, named from that element whatever its type
//     definition, or the sealed interface over its substitution group as stated
//     above, and its field is named from that declaration's local name.
//   - A wildcard particle gives one field named Any by "Naming", holding
//     AnyElement by the occurrence rules below. Neither its {namespace
//     constraint} nor its {process contents} (§3.10.1) is represented: the
//     field holds elements the wildcard would not admit (cvc-wildcard), and
//     holds every match raw whether a strict or lax wildcard would assess it;
//     enforcing either is the validator's.
//   - A named complex type, a top-level element through <element ref>
//     particles, or a local element of an anonymous type through <group ref>
//     particles to the group declaring it, that reaches itself through elements
//     that each occur exactly once in a sequence or all group that is a
//     {particle} occurring exactly once or in a sequence or all group spliced
//     into one, and whose types are not choices, declines: its Go type would
//     hold itself as a struct value. A field holding a substitution group's
//     interface holds no struct value, so no recursion passes it.
//   - {max occurs} above 1 or unbounded gives a slice whatever {min occurs} is,
//     of a list's slice too ([][]T); {min occurs} 0 with {max occurs} 1 gives a
//     pointer to a struct, atomic Go type, or list's slice (*[]T), and the bare
//     interface, a choice's or a union's, never a pointer to one, nil meaning
//     absent.
//   - An all group gives the fields a sequence of the same particles gives, in
//     particle order, by the same occurrence rules, an all group particle
//     occurring exactly once in it, a <group ref> to an all group (§3.8.6.2
//     cos-all-limited clause 1.3), giving in its place the fields its own
//     particles give. The order in which its children occur in the instance is
//     not represented: each field holds, in order, the children its particle
//     accepts, and how one particle's children interleave with another's is
//     lost. Every such interleaving is valid (§3.8.4.1.3 all-mg: an all group's
//     language is the interleave of its particles').
//   - A wildcard particle's field in an all group holds the children attributed
//     to it: a child that an element particle of the group could take at that
//     point in the sequence is attributed to the element particle instead
//     (§3.8.4.2 key-vpath).
//   - Each attribute use of a type emitted as a struct gives one field after
//     the particle fields or the value field, if any, in the order
//     [xsd.ComplexType.AttributeUses] returns its {attribute uses} (§3.4.2.4).
//     The field is named from the attribute declaration's local name, so a
//     qualified and an unqualified attribute of one local name, or an attribute
//     named like an element of the struct or like the value field, take the
//     suffix "Naming" states, the later field taking it: an attribute value
//     beside the value field gives Value2.
//   - An attribute use's {type definition}, xs:anySimpleType for a declaration
//     with no type attribute and no <simpleType> child (§3.2.2.1
//     dcl.att.global, §3.2.2.2 dcl.att.local), gives a special type's string,
//     an atomic Go type, a slice, or a union's sealed interface, as an
//     element's does; {required} true gives that type, false a pointer to it
//     (*[]T for a list), but the bare interface for a union, nil meaning
//     absent.
//   - The use's effective value constraint (§3.5.4 key-evc), default or fixed,
//     whether the use or a global declaration it references carries it, is
//     accepted and not represented. The field holds the attribute as the
//     element's [attributes] carry it, so a nil field still means absent from
//     the instance, although for an optional use whose declaration is not a
//     built-in xsi: one the post-schema-validation infoset supplies the
//     constraint's value when the attribute is absent (§3.4.5.1 key-dflt-att).
//     Supplying that value and enforcing a fixed one are the validator's.
//   - A choice {particle} that occurs exactly once is accepted only when its
//     type has no {attribute uses} and no {attribute wildcard}.
//   - An {attribute wildcard} of a type emitted as a struct, its own, one an
//     extension inherits, or one an attribute group reference or the schema's
//     defaultAttributes brings (§3.4.2.5 dcl.ctd.anyatt), gives one field after
//     every attribute field, the struct's last, named as if from the local name
//     anyAttribute ("Naming"), so that an attribute anyAttribute gives the
//     wildcard's field AnyAttribute2.
//   - An {attribute wildcard}'s field is a slice of AnyAttribute, never a
//     pointer, nil and the empty slice being one value, and holds the element's
//     [attributes] that no attribute use's declaration names, in document
//     order, but xsi:type, xsi:nil, xsi:schemaLocation and
//     xsi:noNamespaceSchemaLocation (cvc-complex-type clause 2, a use taking
//     precedence by clause 2.1), at most one per expanded name and none named
//     like a declared use, which the struct does not enforce; AnyType's
//     Attributes, by contrast, holds the xsi: ones.
//   - Neither an {attribute wildcard}'s {namespace constraint} nor its {process
//     contents} (§3.10.1) is represented: the field holds attributes the
//     wildcard would not admit (cvc-wildcard, cvc-complex-type clause 2.2.2),
//     and holds every match raw whether a strict or lax wildcard would assess
//     it; enforcing either is the validator's.
//   - Every element declaration emitted from has {nillable} and {abstract}
//     false and no {type table}. A member of a substitution group is emitted as
//     every top-level declaration is, and enters the interface of each head
//     whose group holds it.
//   - An element declaration's {value constraint}, default or fixed, and its
//     {identity-constraint definitions} are accepted and not represented. A
//     value constraint supplies a value only for an empty element and changes
//     neither its content nor the type its value is drawn from (§3.3.1). An
//     identity constraint restricts the values an instance holds and adds no
//     content or attribute (§3.11.1). So the element emits exactly as it would
//     without either, and supplying a default and enforcing a fixed value or an
//     identity constraint are the validator's.
//
// ExampleGenerate prints the whole file Generate writes for a schema inside
// this slice.
//
// # Determinism
//
// The output is a function of the schema and of each document's location
// relative to the target's Dir, so two trees laid out alike under different
// absolute roots produce byte-identical output; "Naming" states where a
// header's location is not relative. No map iteration anywhere near the
// emitter; emission order is the walk's, which document order alone fixes, as
// "Coverage" states (datatype structs, raw-content structs, and union
// interfaces with their variants last, in the order the walk first needs
// each). Golden-file tests pin the output.
//
// # Naming
//
// Generated identifiers are not stable before v1.0: a release may rename any
// type or field this section names, and the shortening of an anonymous type's
// path that "Planned" states will rename the types named by such a path.
//
// A single namer component owns every XSD-name → Go-identifier decision.
// Each XSD local name becomes an exported identifier: '_' and every rune Go
// does not admit in an identifier break words, every word starts upper-case,
// and a result with no upper-case first letter is prefixed with
// X. That upper-case start is also what keeps every identifier clear of the Go keywords. A
// top-level element's type is named from the element, its name claimed when the walk first
// reaches the element: at its own position, at the first <element ref> particle to it met
// before that, or at the branch of the first substitution-group interface holding it. A named
// complex type a local element reaches is named from the type (complex type ct → Ct), whichever
// local elements have type ct. An anonymous type nested in either is named by the path of names
// from the named complex type that declares its element, when one does (element x in complex
// type ct → CtX, whichever elements have type ct or a type extending ct, which reaches it
// through ct's own particle), and otherwise from its nearest named ancestor: the top-level
// element when the element's own type is anonymous (element shipTo under element purchaseOrder
// → PurchaseOrderShipTo). The original of a complex type a <redefine> redefines has no name
// (§4.2.4 src-expredef clause 1.1), so the types of the local elements it declares are named
// from the nearest named ancestor: its element x → TX for elements of type t, UX for a type u
// extending t. An anonymous type of a local element a model group definition declares, and
// every type nested in it, is named by the path of names from that group (element e in group g
// → GE), whichever <group ref> particles reach it. The type for one occurrence of a model group
// particle is named by the path of names of the complex type whose content model holds it — Ct
// for a named complex type ct, the path an anonymous one is named by above, GE for the
// anonymous type of element e in group g — never from a name its interface or a branch claimed,
// followed by the local name sequence, choice or all of each occurrence type enclosing it and
// then its own (CtSequence; a choice in that sequence → CtSequenceChoice; a repeated sequence
// in a branch of named choice type alt, reached by element fifth → AltSequence, never
// FifthSequenceSequence), and the types nested in its particles as if it were absent; it is
// named and keyed under that complex type whichever <group ref> particle brought its group, so
// two complex types referencing one group get an occurrence type each. A choice's branch struct
// is named from its interface's name plus the alternative's local name, its compositor for a
// model group alternative, or any for a wildcard alternative (alternative a of interface First
// → FirstA, a sequence alternative → FirstSequence, a wildcard alternative → FirstAny), so two
// top-level elements of one named choice type get distinct interfaces and branches. A
// substitution group's interface is named from its head's local name plus SubstitutionGroup
// (head h → HSubstitutionGroup), and each branch from the interface's name plus its
// declaration's local name (member m1 → HSubstitutionGroupM1); the interface's name is claimed
// at the first <element ref> particle to the head, then each branch's in group order, each
// followed, when the walk has not reached its declaration yet, by the names that declaration's
// type claims as at an <element ref> particle to it. Each choice branch's name is claimed when
// the walk reaches its alternative, after the types every earlier alternative first needs and
// ahead of its own, so the raw-content structs a wildcard alternative first needs follow its
// branch and every earlier one and precede every later one: alternatives type, a wildcard and
// element of interface Any name the branches AnyType, AnyAny and AnyElement2, and the
// raw-content structs AnyType2, AnyAttribute, AnyChild and AnyElement. A residual collision —
// two names folding to one identifier, among a package's types or one struct's fields — is
// disambiguated deterministically by document order: the later name takes the smallest free
// numeric suffix from 2, so a top-level element ct and a complex type ct reached locally take
// Ct in the order they are first reached. A datatype struct is named from its datatype's local
// name (dateTime → DateTime, gYearMonth → GYearMonth, dayTimeDuration → DayTimeDuration, QName
// → QName, which an xs:NOTATION type shares), claimed when the walk first needs it: for a
// top-level element of a simple type, ahead of that element's own type, so that a top-level
// element date of type xs:date names its type Date2; for a field of a struct, the value field
// and attribute fields included, after that struct's own name, so that a top-level element date
// whose complex type extends xs:date names its type Date and the datatype struct Date2. The
// raw-content structs are named AnyType, AnyAttribute, AnyChild and AnyElement, claimed
// together in that order when the walk first reaches a declaration of type xs:anyType or a
// wildcard particle, ahead of that declaration's own type, so that a top-level element anyType
// with no type names its type AnyType2; or when it first reaches the {attribute wildcard} of a
// type emitted as a struct, after that struct's own name and the types its other fields first
// need, so that a top-level element anyType whose anonymous complex type has an <anyAttribute>
// names its type AnyType and the raw-content structs AnyType2, AnyAttribute, AnyChild and
// AnyElement. A wildcard particle's field is named as if from local name any, and an {attribute
// wildcard}'s as if from local name anyAttribute. A union's interface is named from the type
// when it is named (simple type size → Size), claimed when the walk first needs it as a
// datatype struct is: for a top-level element of a union type, ahead of that element's own
// type, so that a top-level element size of type size names its type Size2; for a field of a
// struct, after that struct's own name. An anonymous union is named by the path of names an
// anonymous complex type would take in its place, with the local name union last: a top-level
// element's (element pin → PinUnion), a local element's (element code in complex type box →
// BoxCodeUnion), an attribute's from the global attribute declaration, the named complex type
// or attribute group definition declaring it, or the path of the struct holding it for an
// anonymous complex type (attribute mode in attribute group shared → SharedModeUnion, attribute
// kind of element tag → TagKindUnion), and a value field's from its struct's path with value
// before union (complex type narrow → NarrowValueUnion). A list's item is named from the list
// whose <list> declares it: from that list's local name when it is named (simple type codes →
// CodesUnion, for codes and every restriction of it), and from the path the list takes
// otherwise. A variant is named from its interface's name plus its member's local name: an
// anonymous list member's is list, and any other anonymous member's is that of the nearest
// named type on its {base type definition} chain, the member itself first (xs:int in Size →
// SizeInt, an anonymous restriction of xs:string → SizeString). The variants are claimed after
// their interface in basic-member order, each ahead of its member's own Go type, which takes
// the variant's name as its path (an anonymous union item of the list variant MixedList →
// MixedListUnion). Every other generated type's header comment but a union interface's or
// variant's records the QName and the schema Loc of the declaration it is named from: the
// element declaration, or the complex type definition for a type named from one. A union
// interface's header records the simple type it is generated from, by its QName or as
// (anonymous), and that type's schema Loc, and a variant's header its member the same way, a
// builtin member, having no schema document, by its QName alone. A datatype struct's header
// records its datatype's QName alone, a builtin having no schema document, then a second line
// stating what its fields' names do not, when there is any: TimezoneOffset's unit and whether
// it may be absent, that duration's Months and Seconds share one sign, or that QName's Space is
// "" for no namespace. A raw-content struct's header records xs:anyType's QName alone, then,
// for AnyType and AnyChild, a second line stating what Text or Tail holds. A Loc whose URI
// filepath.IsAbs accepts renders relative to the target's Dir made absolute against the working
// directory, slash-separated and climbing with ".." out of Dir. A document outside the tree
// laid out alike, such as one an absolute schemaLocation names, renders a climbing path that
// differs with that tree's depth below the filesystem root. Any other URI renders as is, a URL
// or a relative loader key alike, as does one with no path relative to Dir, such as one on
// another Windows volume. Symbolic links are not resolved.
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
// # The Emitter seam (value.Emitter; M9 — not yet implemented)
//
// Backends contribute specialized decode/encode source for their value
// representations — parsing straight from the reader's byte window into
// the target field, facet checks inlined, no intermediate string, no
// boxed value. A backend without an Emitter falls back to emitting calls
// into codec's runtime path. Emitted fast paths carry the same pipeline
// stage and rule-ID metadata as the runtime path; codec's differential
// tests hold the two to identical behavior.
package codegen
