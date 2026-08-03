// Package schemaloc resolves a schemaLocation URI reference against the base URI
// in scope at the element carrying it (§4.3.2 clause 4).
//
// # Contract
//
//	func Resolve(base, location string) string
//
// It depends on nothing but the standard library, and sits below every package
// in the module, because two independent walks over the same schema documents
// must agree byte for byte on what a location hint names: the parser's assembly,
// whose resolution decides which documents Parse actually reads, and the
// conformance harness's schema-closure walk, which gates the shape of every
// document Parse will read. A walk resolving a hint even slightly differently
// would discover a different document set than the parser does, and a document it
// under-discovered would be one whose shape was never gated — the false accept
// that walk exists to close.
//
// Resolve only resolves. Whether a location that fails to resolve is an error is
// the caller's question, answered by src-include, src-override and src-import
// separately and cited where each caller branches on it.
package schemaloc
