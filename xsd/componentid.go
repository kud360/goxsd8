package xsd

// ComponentID is an opaque identity for a schema component: a token minted once
// by NewComponentID and threaded into every component that must agree on "the
// same component" in a slot where no NAME can serve that role.
//
// Structures §3.4.1's {context} is the first such slot. §3.4.2.1
// dcl.ctd.common makes an anonymous complex type's {context} "the Element
// Declaration corresponding to the nearest <element> information item among the
// ancestor element information items", and that element declaration is very
// often LOCAL — and local element declarations are not QName-unique (two
// sibling complex types may each contain <element name="a">). A bare (kind,
// QName) reference — the pattern ElementScopeParent uses for a {scope}.{parent}
// whose container HAS a name — therefore cannot identify the target; this token
// can. The same reasoning gave ElementScopeParent an identity arm of its own,
// AnonymousComplexTypeScopeParent, for the reciprocal direction.
//
// The producer mints the identity BEFORE either endpoint exists and passes it
// to both, so nothing is mutated after construction and no second phase is
// needed:
//
//	edID := xsd.NewComponentID()
//	ct, err := xsd.NewAnonymousComplexType(ctLoc, xsd.ElementDeclarationContext{Component: edID}, …)
//	ed, err := xsd.NewElementDeclarationOwningType(edLoc, edID, name, ct, …)
//
// The same edID is what every local element declaration nested in ct's own
// content model reports as its {scope}.{parent}, through
// xsd.AnonymousComplexTypeScopeParent{Owner: edID}: one mint per inline
// construct identifies the construct from both directions.
//
// The zero ComponentID is the UNMINTED identity: it identifies no component and
// is the absent value. Callers test presence by comparing against it
// (id == ComponentID{}); this package exports no IsPresent helper, because the
// comparison IS the contract and a second spelling of it would be a second
// story (STYLE T5). NewComponentID is the only way to obtain a present
// identity — cell is unexported, so no package outside xsd can forge or alias
// one (STYLE T1).
//
// # Identity is ==, and reflect.DeepEqual cannot see it
//
// A ComponentID is a struct with one pointer field, so it is ==-comparable and
// survives by-value copies: copying a ComplexType copies the interface value
// holding the context arm, which copies this struct, which copies the POINTER,
// so both copies still compare == on their {context} identity.
//
// reflect.DeepEqual is IDENTITY-BLIND here and must never be used to compare
// two ComponentIDs. DeepEqual follows pointers and compares pointees, and every
// identityCell is the same zero byte, so two DISTINCT minted identities compare
// DeepEqual-equal — measured, not conjectured; componentid_test.go pins the
// behaviour so a future stdlib change is caught rather than silently assumed.
// A component-level DeepEqual assertion will therefore accept a wrong
// {context}; assert on the ID with == instead.
//
// # No String, no ordering
//
// ComponentID has no String method and is never rendered: its underlying value
// is an ADDRESS, so any textual form (and any address ordering) is
// nondeterministic run to run, which STYLE D1 forbids outright. Rejections
// describe the offending arm with %T instead.
//
// The same fact binds any future ID→component resolver — no issue owns one yet,
// so none is built (STYLE T5): #438, the nearest landing to touch the anonymous
// complex types reached only through an InlineTypeDefinition, walks them by
// TRAVERSAL from the owning Element Declaration rather than by identity, so it
// does not need one either. Such a resolver must never range a
// map[ComponentID]… to produce output or ordering, because that iteration order
// is address order (STYLE D2). Order such output by the components' own document
// order, never by their identities.
type ComponentID struct{ cell *identityCell }

// identityCell is the allocation whose ADDRESS is a ComponentID's identity.
//
// Do NOT remove the blank byte field and do NOT replace identityCell with
// struct{}. The field is load-bearing, not a placeholder: Go does not guarantee
// distinct addresses for distinct zero-size allocations, and an ESCAPING
// new(struct{}) is returned from runtime.zerobase, so every zero-size cell
// would share one address and two independently minted identities would compare
// equal. Measured on go1.26.4: escaping new(struct{}) pairs compare equal;
// escaping new(struct{ _ byte }) pairs do not. The failure is escape-dependent,
// so a struct{} cell can pass a stack-allocated unit test and then collide the
// moment the identities escape into components — which is exactly what this
// package does with them. A future "unused field" cleanup that deletes the byte
// silently breaks component identity.
type identityCell struct{ _ byte }

// NewComponentID mints a fresh component identity, distinct from every other
// minted identity and from the zero (unminted) ComponentID. It cannot fail.
//
// Mint one per component that needs to be pointed AT by identity, before
// constructing either endpoint; see ComponentID for the threading order.
func NewComponentID() ComponentID {
	return ComponentID{cell: new(identityCell)}
}
