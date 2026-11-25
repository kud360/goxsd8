// Package codec is the dataset codec: schema-directed decode of instance data
// into Go values, and encode of Go values back out in canonical form for the
// slice "# What encodes" states; the allocation-frugal decode paths
// "# Planned decode paths" names are not yet built. [Decode] assesses the
// instance itself, so no separate validate call precedes it; its doc states
// how. [Append]'s doc states when it assesses the document it writes and how
// a caller gets a verdict on it otherwise. Given a value from Decode that it
// encodes without error, Append writes that value, not the input's bytes:
// [Append]'s doc and "# What encodes" state the written form.
//
// Call [CheckDecode] once at start-up for each top-level element declaration
// and T a program decodes a document element into, and [CheckAppend] for each
// it encodes one from. Pass CheckDecode the options [Decode] is passed and
// CheckAppend those [Append] is passed, of which each doc names the ones it
// reads; then call Decode for each instance and Append for each value. Each
// Check's doc states what it reports.
//
// # Decode
//
//	func Decode[T any](schema *xsd.Schema, backend value.Backend, data []byte, opts ...Option) (*T, error)
//	func CheckDecode[T any](schema *xsd.Schema, name xsd.QName, opts ...Option) error
//	type Option func(*config)
//	func WithURI(uri string) Option
//	func WithBranches[I any](branches ...I) Option
//	func WithPrecisionDecimalArms[I any](numerical, positiveInfinity, negativeInfinity, notANumber I) Option
//	func WithLogger(l *slog.Logger) Option
//	type DeclineError struct{ ... }
//	func (e *DeclineError) Loc() (xsderr.Loc, bool)
//	func (e *DeclineError) Error() string
//	func (e *DeclineError) Unwrap() error
//	func (e *DeclineError) Construct() Construct
//	type Construct struct{ ... }
//	func (c Construct) String() string
//	var ErrTypeMismatch error
//	func ViolationsOf(err error) []*xsderr.Error
//	type UndecidedError struct{ ... }
//	func (e *UndecidedError) Unevaluated() validate.Unevaluated
//	func (e *UndecidedError) Error() string
//	func (e *UndecidedError) Unwrap() error
//
// [Decode] decodes a valid XML instance into the type package codegen emits
// for its document element's declaration, and [Decode]'s doc states what it
// returns in every other case; [CheckDecode]'s states what it returns for a
// declaration before an instance arrives. It reads the schema, not T:
// codegen's structs carry no tags, so Decode derives each field's name, order
// and Go type from the declaration by codegen's own rules, those whose answer
// both packages reach, the datatype structs' fields among them, held once in
// internal/goshape and internal/goname, and refuses a T that differs, and
// reads a sealed interface's branch or variant types from [WithBranches], and
// a PrecisionDecimal's arms from [WithPrecisionDecimalArms]. Package codec
// does not import codegen.
//
// # What decodes
//
// Decoded, each construct under the conditions and into the Go type the
// paragraphs after the list state:
//
//   - An atomic simple type of one of the twenty primitives the next
//     paragraph names, xs:precisionDecimal only with
//     [WithPrecisionDecimalArms].
//   - A list simple type.
//   - A union simple type, into a sealed interface.
//   - A top-level element declaration, as the document element's.
//   - An element declaration with a {type table}, into codegen's governing
//     types' interface where it has other than one governing type.
//   - A complex type with empty, simple, element-only or mixed content.
//   - A choice, inline or through <group ref>, occurring exactly once, as a
//     complex type's content {particle}.
//   - A content sequence or all group, inline or through <group ref>, that
//     codegen splices (codegen's package doc, "# Particles"), into the fields
//     its own particles give.
//   - A nested sequence, inline or through <group ref>, and an all group's
//     nested all group particle, through <group ref>, that codegen splices
//     (codegen's package doc, "# Particles"), into the fields its own
//     particles give.
//   - A nested choice, inline or through <group ref>, occurring exactly once,
//     that codegen does not flatten (codegen's package doc, "# Particles"),
//     into its field of a sealed interface.
//   - A sequence or choice, inline or through <group ref>, that codegen
//     flattens (codegen's package doc, "# Particles"), as a complex type's
//     content {particle}, a choice occurring exactly once included, or as a
//     sequence's nested particle outside a content particle held in an
//     occurrence field, into its slice of a sealed interface.
//   - A sequence or choice, inline or through <group ref>, occurring other
//     than exactly once, that codegen neither splices nor flattens (codegen's
//     package doc, "# Particles"), as a complex type's content {particle}, a
//     sequence's nested particle or a choice's model-group alternative,
//     outside a content particle held in an occurrence field, and a content
//     all group, inline or through <group ref>, that codegen does not splice,
//     into its field of the type for one occurrence.
//   - An element particle, local or <element ref>.
//   - A substitution group, into a sealed interface.
//   - A choice, into a sealed interface.
//   - A wildcard particle, any number of them in one content model, a nested
//     choice's alternatives among its particles.
//   - An element declaration of type xs:anyType, the document element's or a
//     particle's.
//   - A nillable element declaration, the document element's or a particle's,
//     into codegen's Nillable (codegen's package doc, "# Element
//     declarations").
//   - An attribute use.
//   - An {open content}, into its field after the particle fields.
//   - An {attribute wildcard}, into its field after the attribute fields.
//   - A value constraint, default or fixed.
//
// A simple type decodes when it is atomic with {primitive type definition}
// xs:string, xs:anyURI, xs:boolean, xs:float, xs:double, xs:decimal,
// xs:dateTime, xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay,
// xs:gDay, xs:gMonth, xs:duration, xs:hexBinary, xs:base64Binary, xs:QName,
// xs:NOTATION or xs:precisionDecimal. So xs:normalizedString, xs:token,
// xs:language, xs:Name, xs:NCName, xs:NMTOKEN, xs:ID, xs:IDREF, xs:ENTITY,
// xs:integer and every builtin derived from it, xs:dateTimeStamp,
// xs:dayTimeDuration, xs:yearMonthDuration, and a user restriction of any of
// them decode. Each decodes into the atomic Go type codegen gives it, by the
// rules of codegen's package doc, "# Simple types", a date, time or duration
// type into its datatype struct, an xs:QName or xs:NOTATION type into
// QName's, the expanded name its lexical resolves to as [Decode]'s doc
// states, and an xs:precisionDecimal type into PrecisionDecimal's, its Value
// the arm [WithPrecisionDecimalArms] supplies for its lexical's value, with
// that lexical's ·scale· and ·sign·, as [Decode]'s doc states. A special
// type, xs:anySimpleType or xs:anyAtomicType, decodes into the string
// codegen gives it (codegen's package doc, "# Simple types"), its value as
// [Decode]'s doc states. A list type decodes when its {item type definition}
// is such an atomic type or a union that decodes, a restriction of a list
// taking its base's (Structures §3.16.2.3), xs:NMTOKENS among them, into
// []A, A its item's atomic Go type, or []I, I its item's sealed interface,
// one element per list item; the empty list is a slice of length 0.
//
// A union type (Datatypes §2.4.1.3) decodes when each of its basic members,
// in codegen's basic-member order (codegen's package doc, "# Simple types"),
// is such an atomic or list type, item 5 of [Decode]'s doc stating each. It
// decodes into codegen's sealed interface, the variant of the value's
// ·active basic member·, as [Decode]'s doc states.
//
// A document element decodes when its declaration, a top-level one, has governing types that
// decode (codegen's package doc, "An element declaration's governing types"; item 5 of
// [Decode]'s doc states when): of one, a declaration with no {type table} having its {type
// definition} alone, or, where that is abstract, its one member, that type is either xs:anyType
// itself, or a simple type that decodes, its value T's own, or a complex type that decodes, under the
// {nillable} item 5 admits, T being Nillable over that type for a nillable declaration codegen wraps
// (codegen's package doc, "# Element declarations"); of two or more, of a declaration that is not
// nillable, each is such a type. A complex type decodes when it has either empty content, simple
// content whose {simple type definition} is a simple type that decodes (Structures §3.4.2.2
// dcl.ctd.ctsc), its value field's, or element-only or mixed content whose {particle} is a sequence
// or choice, inline or the {model group} of the definition a <group ref> names, or is an all group,
// inline or through <group ref>. Each particle of a sequence or choice not flattened or of an all
// group, a sequence's nested sequence particle and an all group's nested all group particle, through
// <group ref>, that codegen splices replaced by its own particles at any depth, a sequence's nested
// choice particle occurring exactly once that codegen does not flatten by its alternatives, a nested
// particle held in an occurrence field by its own particles, and, of a flattened one, the content
// {particle} or a sequence's nested particle, in place of its particles, each basic particle its
// model group reaches through its model group particles at any depth, is an element declaration whose
// governing types decode as the document element's do, by these same rules at any depth: a local one,
// or the top-level one an <element ref> names where codegen gives the particle's field that
// declaration's own type; or the particle is an <element ref> where codegen gives its field a sealed
// interface over the referenced declaration's substitution group (codegen's package doc, "#
// Substitution groups"), and each declaration of the group with {abstract} false is such a
// declaration, an abstract head's own properties never read; or the particle is a wildcard, whatever
// its {namespace constraint} and {process contents}. Two element particles taking one expanded name
// decode, as item 5 of [Decode]'s doc states. Each attribute use but xsi:nil's, to which codegen
// gives no field (codegen's package doc, "# Attributes"), is typed the same way. No element in the
// instance carries xsi:type, but where item 7 of [Decode]'s doc admits it.
//
// Item 6 of [Decode]'s doc states the Go type of each field, naming it V, S
// or I as the next paragraph does. The field of a particle of complex type
// other than xs:anyType is filled from the child element as T is from the
// document element, so a type that reaches itself through an optional or
// repeated particle decodes to the instance's depth.
//
// A choice decodes into codegen's sealed interface, the branch of the one
// alternative the instance matches (§3.8.4.3 cvc-model-group) holding its
// fields. An element or wildcard alternative's branch holds its one field: V,
// or a []V for an alternative whose {max occurs} is above 1 or unbounded, its
// {min occurs} 0 read as 1, or, for an <element ref> alternative, the same
// over the referenced declaration's named type, or, for an alternative of
// complex type, the same over its S or I, or its AnyType for one of type
// xs:anyType. A model-group alternative's, inline or through <group ref>,
// holds the fields codegen gives it (codegen's package doc, "# Complex
// types"), each filled as a sequence's particle's field is: for a sequence
// alternative codegen splices, the fields of its own particles, and for any
// other the one field of a nested choice, of a flattened particle or of an
// occurrence field. A content choice occurring exactly once that codegen does
// not flatten has its interface T itself, or, where the type has an attribute
// field (codegen's package doc, "# Attributes": an attribute use other than
// xsi:nil's, or an {attribute wildcard}), an {open content} (codegen's package
// doc, "# Open content"), mixed content (§3.4.1) or one occurrence of the
// choice is emptiable (§3.9.6.3 cos-group-emptiable), the first field of T
// but for mixed content's Text, a struct whose further fields are the
// open-content and attribute fields; item 6 of [Decode]'s doc places a nested
// choice's field and the occurrence field's interface. An empty match is the
// nil interface. A particle's complex type gets its interface or struct by the
// same rule.
//
// A flattened particle, the content {particle} or nested, decodes into the
// slice field codegen emits for it (codegen's package doc, "# Particles"): one
// item per child element its basic particles take, in document order, the
// branch of the basic particle (Structures §2.2.3.2 key-basic-particle) the
// child's validation-path attributes it to (§3.8.4.2 key-vpath), as [Decode]'s
// doc states; an empty match is the nil slice.
//
// A content {particle} or a nested particle, occurring other than exactly
// once, that codegen neither splices nor flattens decodes into its occurrence
// field, one item per sub-sequence of a ·partition· of the children (§3.9.4.3
// cvc-accept clause 3), in document order; [Decode]'s doc states the field, its
// items and the rule that partitions the children.
//
// A substitution group's interface holds, for each child, the branch of the
// group's declaration the child's expanded name resolves to (§3.9.4.3
// cvc-accept clause 2.3.2), whose field and alternatives item 6 of
// [Decode]'s doc states.
//
// Mixed content (Structures §3.4.1) decodes into the struct codegen emits for
// it (codegen's package doc, "# Complex types"); [Decode]'s doc states how its
// fields are matched and filled.
//
// Each wildcard particle decodes into its own AnyElement field codegen emits,
// AnyChild in mixed content, or a choice's branch holding one, each child
// attributed to that particle held raw; [Decode]'s doc states the field,
// which children are attributed to it, and the raw content.
//
// A complex type's {open content} decodes into the field codegen emits for it
// (codegen's package doc, "# Open content"), each child validation attributes
// to it (§3.4.4.4) held raw, whatever its {mode} and its wildcard's
// constraints; [Decode]'s doc states the field and which children it holds.
//
// A complex type's {attribute wildcard} (Structures §3.4.2.5
// dcl.ctd.anyatt), its own, one an extension inherits, or one an attribute
// group or the schema's defaultAttributes brings, read from the finalized
// type, decodes into the slice of AnyAttribute codegen emits as the struct's
// last field, each attribute no attribute use's declaration names held raw,
// whatever the wildcard's {process contents}; [Decode]'s doc states the field
// and which attributes it holds.
//
// An element declaration whose one governing type is xs:anyType itself
// (Structures §3.4.7) decodes, under the {nillable} item 5 of [Decode]'s doc
// admits for it: the document element's into the named type
// codegen emits over AnyType, and a particle's, local or referenced, a
// choice's alternative and a substitution group's member included, into
// codegen's AnyType field, or the named type codegen emits over it, its
// content held raw as a wildcard's match's is; [Decode]'s doc states T and
// the field. A type derived from xs:anyType decodes or declines as its own
// complex type.
//
// Every other construct is a [*DeclineError], never a zero value:
//
//   - A content model xsd.Schema.ContentMatcher does not track, flattened,
//     holding a nested flattened particle, held in an occurrence field, or
//     holding a nested occurrence field.
//   - Two alternatives of a choice, two basic particles a flattened particle
//     reaches, or two governing types of an element declaration, whose
//     branch structs Decode cannot tell apart; item 5 of [Decode]'s doc
//     states when.
//   - Inside a content particle held in an occurrence field, a nested
//     sequence or choice, inline or through <group ref>, a sequence's
//     particle or a choice's model-group alternative, an alternative's {min
//     occurs} 0 read as 1, that codegen flattens, or that it neither splices
//     nor flattens and that is no choice occurring exactly once, whatever its
//     occurrence.
//   - A nillable element (codegen's package doc, "# Element declarations")
//     of two or more governing types, or of a {type table} slot holding
//     xs:error, whose interface codegen wraps branch by branch, and one of a
//     type other than xs:anyType itself examined as a branch of a substitution
//     group's interface; item 5 of [Decode]'s doc states when.
//   - A primitive other than those twenty: an ·implementation-defined· one
//     (Datatypes dt-primitive), which xsd.NewPrimitiveType mints and no
//     schema document yields.
//   - A list whose {item type definition} declines, by its item's construct.
//   - A union one of whose basic members declines, by that member's
//     construct.
//   - An xsi:type.
//
// One of those is a ruling rather than coverage still to come: an element
// carrying xsi:type declines, whatever type it names, but where item 7 of
// [Decode]'s doc admits it. Where the ·selected type definition· is not
// abstract, codegen emits no branch for a type derived from it (Structures
// §3.3.4.3 cvc-elt clause 4), and decoding the instance into the selected
// type's field would lose what the derived type governs: under
// xsi:type="xs:token" an xs:string element's value collapses. That refusal is
// permanent. An element of a declaration with a {type table} declines by its
// xsi:type too where the selected type is abstract.
//
// An element whose declaration has two or more governing types decodes into
// the branch of the type the validator selected
// (validate.Validator.ConditionallySelectedIndex) where the declaration has a
// {type table}, and of the member its xsi:type names where it has none, as
// [Decode]'s doc states, and codec never evaluates a {test} itself.
//
// A value constraint of an element declaration or attribute use, default or
// fixed, decodes; [Decode]'s doc states the value it supplies, and that it
// supplies none to an element of type xs:anyType.
//
// Every error about the instance carries its instance Loc, under the URI
// [WithURI] names: a violation's, an undecided check's, and a decline's,
// which also names its construct and, for a construct of the schema, the
// declining component, as [DeclineError.Error] states; and
// [DeclineError.Construct] returns its construct.
//
// # Encode
//
//	func Append[T any](dst []byte, schema *xsd.Schema, backend value.Backend, name xsd.QName, v T, opts ...Option) ([]byte, error)
//	func CheckAppend[T any](schema *xsd.Schema, name xsd.QName, opts ...Option) error
//	var ErrInvalidValue error
//
// [Append]'s doc states what it writes and what it returns;
// [CheckAppend]'s states what it returns for a declaration and T. Append
// reads the schema, not T, as [Decode] does, and holds T to the declaration
// by Decode's comparisons, under the [WithBranches] sets and the
// [WithPrecisionDecimalArms] arms it is passed.
//
// # What encodes
//
// Append encodes each construct the list names under the conditions its
// bullet states, into the Go type the bullet names where it names one:
//
//   - A top-level element declaration, as the document element's, neither
//     abstract nor nillable, with no {type table} or value constraint, each of
//     whose {substitution group affiliations} names a top-level element
//     declaration of the schema, each of whose {identity-constraint
//     definitions} has a {selector} and {fields} icpath compiles, and for a
//     keyref a {referenced key} whose own it compiles, whose {type definition}
//     is a simple type that encodes, a complex type that encodes, or
//     xs:anyType itself.
//   - A complex type with {abstract} false, no {assertions} and no {open
//     content}, of empty content, of simple content whose {simple type
//     definition} (Structures §3.4.2.2 dcl.ctd.ctsc) is a simple type that
//     encodes, its value field (codegen's package doc, "# Complex types")
//     written as the element's character content after its attributes, or of
//     element-only content whose {particle} is a sequence that codegen
//     splices, a choice occurring exactly once that codegen does not flatten,
//     or a sequence or choice codegen holds in an occurrence field, inline or
//     through <group ref>, each of whose particles is an element particle, a
//     wildcard particle with {process contents} skip or lax, or a nested
//     sequence or choice particle, inline or through <group ref>, held as
//     codegen's package doc "# Particles" holds it: one codegen splices, as
//     its own particles in its place; a choice occurring exactly once that
//     codegen does not flatten, into its sealed interface field, as its
//     alternatives; one codegen flattens, into its slice field, each basic
//     particle of its reach such an element or wildcard particle; and any
//     other, into its occurrence field, as its own particles. A choice's
//     alternative, the content choice's or a nested one's, is such a particle
//     too, its model group held at the occurrence its branch reads, {min
//     occurs} 0 read as 1, into its branch's fields. Inside a content particle
//     codegen holds in an occurrence field, a nested particle codegen flattens
//     or holds in an occurrence field of its own is a [*DeclineError],
//     whatever its occurrence, as it is for Decode ("# What decodes"). A
//     content model holding a wildcard particle, a nested flattened particle
//     or a nested occurrence field, at any depth, is one
//     xsd.Schema.ContentMatcher tracks. The content {particle} may also be a
//     sequence or choice codegen flattens, each basic particle of whose reach
//     (codegen's package doc, "# Particles") is such an element or wildcard
//     particle, and whose content model xsd.Schema.ContentMatcher tracks, with
//     or without a wildcard particle. The choice occurring exactly once
//     encodes into its sealed interface, T itself or the first field of T's
//     struct, the sequence or choice held in an occurrence field into that
//     field, and the flattened one into its slice field, as "# What decodes"
//     states.
//   - An element particle of that sequence or choice or of a model group
//     nested in it, at any depth, or one a flattened sequence or choice
//     reaches, with {max occurs} at least 1, its {term} a local element
//     declaration or, for an <element ref>, the top-level declaration it
//     names or, where codegen's package doc "# Substitution groups" gives its
//     field a sealed interface, each declaration that interface has a branch
//     for, each such declaration having the same restrictions as the document
//     element's and a simple type or a complex type that encodes, or
//     xs:anyType itself, into the field item 6 of [Decode]'s doc holds to it,
//     the branch a value of that interface holds written as an element of its
//     own declaration's expanded name; or, for a choice's alternative, into
//     its branch's field, and for a basic particle a flattened sequence or
//     choice reaches, into the one field of its branch, each item of the
//     slice one element, as "# What decodes" states.
//   - A wildcard particle of that sequence or choice or of a model group
//     nested in it, at any depth, or one a flattened sequence or choice
//     reaches, with {process contents} skip or lax and {max
//     occurs} at least 1, into the field "# What decodes" states for it, each
//     element it holds written raw, as below.
//   - xs:anyType itself (Structures §3.4.7), where the bullets above admit
//     it, into the Go type "# What decodes" states for its declaration, its
//     content written raw, as below.
//   - An attribute use other than xsi:nil's, unqualified, with no value
//     constraint and of a simple type that encodes, into V or *V.
//   - An {attribute wildcard} (Structures §3.4.2.5 dcl.ctd.anyatt), the
//     type's effective one, of any {process contents}, into the field "# What
//     decodes" states for it, each attribute it holds written raw, as below.
//   - xs:anySimpleType or xs:anyAtomicType itself (Datatypes §2.4.2
//     dt-special), into string.
//   - An atomic simple type whose {primitive type definition} is one the list
//     of primitives below names, into the Go type that list gives it. It has
//     no assertions facet, its own or inherited. It is derived from none of
//     xs:ID, xs:IDREF and xs:ENTITY, though xs:string is the primitive of
//     each.
//   - A list simple type whose {item type definition} encodes and which has
//     no assertions facet, its own or inherited, into []V, V its item's Go
//     type.
//   - A union simple type (Datatypes §2.4.1.3) each of whose basic members, in
//     codegen's basic-member order (codegen's package doc, "# Simple types"),
//     is an atomic or list simple type that encodes, and neither it
//     nor any union among its {member type definitions}, at any depth, has an
//     assertions facet, its own or inherited, into codegen's sealed
//     interface, whose variants [WithBranches] supplies, as "# What decodes"
//     states.
//
// The primitives an atomic simple type that encodes has are these, each into
// the Go type its item gives, a primitive's datatype struct being the one
// codegen's package doc "# Simple types" gives it:
//
//   - xs:string, into string.
//   - xs:boolean, into bool.
//   - xs:decimal, a type derived from xs:integer included, into the Go type
//     codegen's "# Simple types" gives it.
//   - xs:float, into float32.
//   - xs:double, into float64.
//   - xs:duration, into its datatype struct.
//   - xs:dateTime, into its datatype struct.
//   - xs:time, into its datatype struct.
//   - xs:date, into its datatype struct.
//   - xs:gYearMonth, into its datatype struct.
//   - xs:gYear, into its datatype struct.
//   - xs:gMonthDay, into its datatype struct.
//   - xs:gDay, into its datatype struct.
//   - xs:gMonth, into its datatype struct.
//   - xs:hexBinary, into []byte.
//   - xs:base64Binary, into []byte.
//   - xs:anyURI, into string.
//   - xs:QName, into QName's datatype struct.
//   - xs:NOTATION, into QName's datatype struct.
//   - xs:precisionDecimal, into PrecisionDecimal's datatype struct, whose arms
//     [WithPrecisionDecimalArms] supplies.
//
// Every other construct is a [*DeclineError], item 4 of Append's doc, a
// list's item declining by the item's construct, a union's basic member
// by that member's construct and a complex type's simple content by its
// {simple type definition}'s construct, and so is what item 7 of Append's doc
// states. The one exception is the declaration name names, as the document
// element's, with {abstract} true: it is item 3, a plain error. A declaration
// with {abstract} true that a particle reaches is no exception.
//
// The document Append writes has no XML declaration, so it is XML 1.0 in
// UTF-8, and no white space between tags. It binds element names by default
// declarations alone, never prefixing one but an element held raw, in a
// wildcard's field or an xs:anyType element's content, at any depth, in the
// XML namespace, written xml:Local, which is never declared, and writes no
// attribute in the XSI namespace: the document element carries xmlns="ns" for
// its non-absent namespace ns, and nothing where it is absent, and every other
// element carries xmlns="ns", or xmlns="" for an absent namespace, exactly
// where its namespace differs from its parent's. An xs:QName or xs:NOTATION
// value, written in an attribute or as content of an element, is spelled Local
// unprefixed where its namespace name is that element's namespace, the absent
// one included; xml:Local for the XML namespace, which is never declared; and
// otherwise pN:Local, where pN is the prefix an open start tag, that element's
// or an ancestor's, declares for its namespace name, or else p1, p2 and so on,
// the first not in scope, declared as xmlns:pN="ns" on that element's start tag
// ahead of the attribute or content that writes it. An attribute use's
// attribute is written unprefixed, in the order of codegen's fields (codegen's
// package doc, "# Attributes"), each value delimited by '"', and after them
// each attribute an {attribute wildcard}'s field holds, in held order, as a raw
// element's attributes are written below, nothing in it resolved, but for the
// ones item 7 of Append's doc declines. An element held raw in a skip
// wildcard's field, at any depth, is written raw, nothing in it assessed or
// resolved, and so is one held in a lax wildcard's field and the content of an
// element of type xs:anyType itself, its own attributes included, nothing in
// either resolved, but for the items item 7 of Append's doc declines in it: its
// attributes in held order, one of absent namespace unprefixed, one in the XML
// namespace xml:Local, and every other one pN:Local, pN chosen and declared as
// for an xs:QName value, though its namespace be that element's own; then its
// Text, then each of its Children, written so, followed by its Tail. A nil
// pointer field writes nothing, and so does a nil interface field where nil
// means absent (codegen's package doc, "# Particles", "# Attributes"), a
// particle with {max occurs} above 1 writes one element per item of its slice,
// a flattened sequence or choice one per item of its slice, in order, its
// branch's basic particle's, whatever the particle's occurrence, an occurrence
// field writes each item it holds, in order, as its model group's particles in
// order, a choice's branch writes its alternative's elements, the empty match,
// the nil interface of a present element, writes no child element, and an
// element with no content is written as an empty-element tag, <x/>. A string
// value, of xs:string or xs:anyURI primitive, is written as itself, its
// canonical representation, since xs:string's canonical mapping is the identity
// (Datatypes §E.4 f-stringCanmap) and so is xs:anyURI's lexical mapping, so
// that an xs:anyURI value is neither normalized nor percent-escaped
// (§3.3.17.2), and item 6 of Append's doc holds the string to its own
// whiteSpace-normalized form. A value of xs:anySimpleType or xs:anyAtomicType
// itself is written as the string it holds, the ·literal· Decode returns, whose
// lexical mapping is no function (Datatypes §3.2.1.2, §3.2.2.2), and item 6 of
// Append's doc holds it to the Char production alone. A list value is written
// as its items are written, in order, separated by single U+0020 characters
// (Datatypes §2.4.1.2), and the empty list, nil or empty, as no content: <x/>
// for an element, a="" for an attribute. A union value is written as its
// variant's member value is, under that member, item 7 of Append's doc stating
// when the text it writes would decode into another variant. It escapes '<',
// '&' and '>' as &lt;, &amp; and &gt; everywhere, so that "]]>" is never
// written, and CR as &#xD; everywhere, which XML 1.0 end-of-line handling
// (§2.11) would otherwise rewrite to LF or remove; in an attribute value it
// escapes '"' as &quot;, TAB as &#x9; and LF as &#xA;, which attribute-value
// normalization (§3.3.3) would otherwise turn into spaces; and every other
// character, TAB and LF in content and the apostrophe everywhere, is written as
// itself. So the Order codegen emits for README's order.xsd, of Item {"a"} and
// Kind "k", is written <order xmlns="urn:o" kind="k"><item>a</item></order>.
//
// # Debuggability
//
// Decode runs three stages on a value: whiteSpace, then, for an atomic Go
// type other than string, the lexical mapping and the conversion to the Go
// type. On a list value whiteSpace runs once, under the list type, and the
// other two on each list item, under its {item type definition}. On a union
// value, or a list item of a union, Decode first identifies the ·active
// basic member· (value.ValidatingType), and runs the stages under that
// member. Pattern, facet and assertion stages are validate's, decided on the
// same backend first. A whiteSpace failure, a fault of the simple type, never
// reaches Decode's return: validate runs the same whiteSpace stage first and
// leaves the instance undecided, item 4 of [Decode]'s doc. An identification,
// lexical mapping or conversion that fails is the error [Decode]'s doc states
// in item 8, naming the stage, the type, the value or the list item and its
// position, the instance Loc and, for an element's character content, its
// byte offset. Decode traces each stage it runs on the logger [WithLogger]
// sets, whose doc states each record. No package reads GOXSD_DEBUG.
//
// # Planned decode paths
//
// Decode today is the slice above, validate's facet pipeline followed by the
// stages the Debuggability section names; no value.Emitter exists. Append
// follows the appender convention named below. Two paths are planned, neither
// implemented: a runtime path, always available, of the facet pipeline and
// value.Mapping driven by the compiled schema, general, reflective and
// allocation-tolerant, whose hot-path APIs follow the appender convention
// (AppendCanonical(dst []byte, v) []byte; ParseBytes([]byte)) so that it too
// is allocation-frugal; and a generated fast path, code emitted at codegen
// time via value.Emitter, with no intermediate strings, no boxed values,
// facet checks inlined and zero-allocation scalar decode on the native
// backend. Both implement the SAME pipeline stages with the SAME spec rule
// IDs: differential tests feed identical input to both and require identical
// values and identical error rule IDs, testing.AllocsPerRun benchmarks pin
// the fast path's budget, and a fast path that disagrees with the runtime
// path is wrong by definition. Generated fast paths preserve what the
// Debuggability section states and map back to their emitting backend and
// schema construct via header comments.
package codec
