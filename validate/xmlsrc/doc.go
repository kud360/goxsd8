// Package xmlsrc adapts XML instance documents onto the validate
// infoset — the first and reference source adapter.
//
// # Contract (implemented in M5)
//
//	func Validate(v *validate.Validator, r io.Reader, opts ...Option) (*validate.Result, error)
//
// Backed by parser/xmltree: streaming, namespace-scoped, every node
// carrying Loc and byte offset, so each violation cites the exact
// instance position. xsi:type, xsi:nil, and default/fixed value
// synthesis follow the engine's rules; xsi:schemaLocation hints are
// surfaced to the caller (parser.SchemaLocationHints) rather than
// silently loaded — schema loading policy belongs to the caller and its
// Resolver.
package xmlsrc
