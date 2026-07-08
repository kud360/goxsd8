package value

import "github.com/kud360/goxsd8/xsd"

// Context supplies the dynamic information a lexical mapping needs beyond the
// literal itself. QName and NOTATION mapping is context-dependent: it resolves
// a prefix against the in-scope namespace bindings at the point of the literal
// (Datatypes §3.3.18/§3.3.19, xmlschema11-2.md — prose-only, with no Appendix E
// hfn), which is why [Mapping.Parse] receives one (PRINCIPLES 19).
type Context interface {
	// LookupNamespace resolves prefix to its bound namespace name in scope;
	// ok is false when the prefix has no binding. The empty prefix is the
	// default namespace.
	LookupNamespace(prefix string) (namespace string, ok bool)
}

// Mapping is a type's lexical↔value mapping.
//
// Parse turns a normalized lexical form into a [Value], using ctx for the
// context-dependent types. A lexical that the wider-space facet checks accept
// but a narrow derived representation cannot hold is a mapping error on the
// derived type — Parse returns it as an *xsderr.Error (the concrete rule ID is
// chosen by the concrete backend that produces the value; no such backend
// exists yet, so the rule is TBD until one lands), never a false validity
// verdict.
//
// Canonical renders a value back to its canonical lexical form. It MAY be nil:
// some types have no canonical mapping (QName's is context-dependent and the
// spec defines none per dt-canonical-representation). Callers must treat a nil
// Canonical as "this type has no canonical form", not as an error.
type Mapping struct {
	// Parse maps a normalized lexical form to a value, or returns an
	// *xsderr.Error describing why the lexical is not in the type's space.
	Parse func(lexical string, ctx Context) (Value, error)
	// Canonical maps a value back to its canonical lexical form. It is nil for
	// types that have no canonical mapping.
	Canonical func(v Value) (string, error)
}

// Backend supplies the lexical↔value mappings for builtin types. A backend
// MUST cover the primitives (directly or via composition); it MAY also map
// derived builtins to give them their own, typically narrower, representation.
//
// A derived type's own mapping governs only the VALUE the application receives,
// never the space in which inherited facet checks run: enumeration and bound
// facets are always compared in the value space of the type that DECLARES the
// facet, parsed by that type's governing mapping (its own, or its nearest
// mapped ancestor's — ultimately the primitive's, which is the widest). Using a
// narrow derived representation for base-chain semantics would corrupt them
// (overflow, collapsed precision, different ordering). Comparison and facet
// capabilities are not backend methods; they live on the values a Mapping
// produces (see [Ordered], [DigitCounted], …).
type Backend interface {
	// Mapping returns the mapping for typ; ok is false when this backend does
	// not map typ.
	Mapping(typ xsd.QName) (Mapping, bool)
}

// Override composes two backends per type: it returns a [Backend] that yields
// partial's mapping for every type partial defines, and base's mapping
// otherwise. This is how a program backs only xs:decimal with a money type and
// keeps the rest of base unchanged.
//
// The widest-space rule (see [Backend]) still governs: an override changes only
// the value an application receives for the overridden type, not the space in
// which any type's inherited facet checks run.
func Override(base, partial Backend) Backend {
	return overrideBackend{base: base, partial: partial}
}

// overrideBackend layers partial over base for [Override].
type overrideBackend struct {
	base    Backend
	partial Backend
}

func (o overrideBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if m, ok := o.partial.Mapping(typ); ok {
		return m, true
	}
	return o.base.Mapping(typ)
}

// LexicalFacet is a facet checked on the normalized lexical form, before the
// lexical mapping runs (the whiteSpace and pattern stages of the pipeline;
// ARCHITECTURE.md). A failure is an *xsderr.Error carrying the facet's rule ID.
type LexicalFacet interface {
	// CheckLexical validates the normalized lexical form, returning an
	// *xsderr.Error on rejection.
	CheckLexical(normalized string) error
}

// ValueFacet is a facet checked on the parsed value, after the lexical mapping
// runs (the value-facet stage of the pipeline; ARCHITECTURE.md). A failure is
// an *xsderr.Error carrying the facet's rule ID. Its name pairs with
// [LexicalFacet] to name the two pipeline stages; the "Facet" suffix is
// load-bearing, so the pairing is kept despite the value.ValueFacet stutter.
//
//nolint:revive // ValueFacet/LexicalFacet name the pipeline stages; suffix is load-bearing.
type ValueFacet interface {
	// CheckValue validates the parsed value, returning an *xsderr.Error on
	// rejection.
	CheckValue(v Value) error
}
