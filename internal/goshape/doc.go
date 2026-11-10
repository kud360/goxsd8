// Package goshape holds the one code answering the decisions that map an XSD
// component to the shape of the Go declaration codegen emits for it, for
// every library package that must reach the same answer: codegen, which
// renders its declarations from it, and codec, which binds and checks a
// caller's T against it.
//
// # Contract
//
//	func SubstitutionSum(schema *xsd.Schema, decl xsd.ElementDeclaration) (members []xsd.ElementDeclaration, sum bool)
//	func BasicMembers(r xsd.TypeResolver, st *xsd.SimpleType) ([]*xsd.SimpleType, error)
//	func Splices(schema *xsd.Schema, group xsd.ModelGroup, occurs xsd.Occurs) (bool, error)
//	func Flattens(schema *xsd.Schema, group xsd.ModelGroup, occurs xsd.Occurs) (bool, error)
//	func Reach(schema *xsd.Schema, group xsd.ModelGroup, visit func(p xsd.Particle, position []int) error) error
//	func TermEmptiable(schema *xsd.Schema, term xsd.TermOrRef) (bool, error)
//	func ExactlyOnce() xsd.Occurs
//	func ModelGroupTerm(schema *xsd.Schema, term xsd.TermOrRef) (xsd.ModelGroup, bool, error)
//	func FieldAttributeUses(t xsd.ComplexType) []xsd.AttributeUse
//	func HasAttributeFields(t xsd.ComplexType) bool
//	type Shape uint8
//	const Bare, Pointer, Slice Shape
//	func ParticleShape(occurs xsd.Occurs, sealed bool) (s Shape, ok bool)
//	func AttributeShape(schema *xsd.Schema, u xsd.AttributeUse, sealed bool) Shape
//	const ValueLocal = "value"
//	const ValueShape = Bare
//
// The rules are those codegen's package doc states, each in its own section
// ("# Simple types", "# Complex types", "# Substitution groups",
// "# Particles", "# Attributes"): those sections own them, and this package is
// their one implementation. TermEmptiable, ExactlyOnce and ModelGroupTerm are
// the reads those rules are made from, exported for the packages that make
// the same reads around them: all three for codegen, and ExactlyOnce and
// ModelGroupTerm for codec.
//
// It imports xsd, xsderr and the standard library.
package goshape
