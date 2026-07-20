package xsd_test

import "github.com/kud360/goxsd8/xsd"

// Compile-time assertions that the sealed Term sum has exactly the three
// implementers §3.9 names — Element Declaration, Wildcard, Model Group — and
// that TermOrRef has exactly its three variants. If a retrofit marker method is
// dropped, or a variant stops satisfying its sum, this file fails to compile.
var (
	_ xsd.Term = xsd.ElementDeclaration{}
	_ xsd.Term = xsd.Wildcard{}
	_ xsd.Term = xsd.ModelGroup{}

	_ xsd.TermOrRef = xsd.ResolvedTerm{}
	_ xsd.TermOrRef = xsd.ElementDeclarationRef{}
	_ xsd.TermOrRef = xsd.ModelGroupRef{}
)
