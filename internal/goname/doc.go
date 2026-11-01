// Package goname turns XSD local names into exported Go identifiers, once
// for every library package that must name a generated declaration alike:
// today codegen alone, which emits the declarations.
//
// # Contract
//
//	type Namer struct{ ... }
//	func NewNamer() *Namer
//	func (n *Namer) Claim(path ...string) string
//	func (n *Namer) ClaimCandidate(candidate string) string
//	func Render(path []string) string
//	func Identifier(local string) string
//
// The rules are codegen's package doc's "# Naming": that section owns them,
// and this package is their one implementation. Identifier maps one local
// name, Render joins a path of them, and a Namer scopes one set of
// identifiers that must not collide, settling each collision by claim order.
// Two importers that claim the same candidates in the same order on fresh
// Namers receive the same identifiers.
//
// It depends on nothing but the standard library.
package goname
