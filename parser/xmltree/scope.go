package xmltree

import "slices"

// binding is one prefix→namespace declaration. The empty prefix is the
// default namespace (xmlns="…"); a non-empty prefix comes from xmlns:p="…".
type binding struct {
	prefix string
	uri    string
}

// scope is an immutable, persistent chain of in-scope namespace bindings:
// one link per element that declares namespaces, pointing at the enclosing
// scope. Because a scope is never mutated after construction, a node may
// retain its scope indefinitely and still resolve prefixes against the
// bindings that were live where it occurred (Datatypes §3.3.18: QName
// resolution depends on the bindings in scope where the literal occurs).
//
// Elements that declare no namespaces reuse their parent scope, so the chain
// depth tracks declaration sites, not nesting depth.
type scope struct {
	parent   *scope
	bindings []binding
}

// child returns the scope produced by layering bindings on top of s, or s
// itself when bindings is empty (no new link needed).
func (s *scope) child(bindings []binding) *scope {
	if len(bindings) == 0 {
		return s
	}
	return &scope{parent: s, bindings: bindings}
}

// lookup resolves a prefix to its namespace URI in scope s (which may be
// nil, the empty base scope).
//
// The reserved prefix "xml" always resolves to XMLNamespaceURI without a
// declaration. The empty prefix resolves to the default namespace, or to ""
// (no namespace) when no default is in scope. A non-empty prefix explicitly
// undeclared with xmlns:p="" (Namespaces in XML 1.1) or never declared is
// unbound: ok is false and the caller reports an error with location.
func (s *scope) lookup(prefix string) (uri string, ok bool) {
	if prefix == xmlPrefix {
		return XMLNamespaceURI, true
	}
	for cur := s; cur != nil; cur = cur.parent {
		for i := len(cur.bindings) - 1; i >= 0; i-- {
			b := cur.bindings[i]
			if b.prefix != prefix {
				continue
			}
			if b.uri == "" && prefix != "" {
				return "", false
			}
			return b.uri, true
		}
	}
	if prefix == "" {
		return "", true
	}
	return "", false
}

// prefixes returns the distinct CANDIDATE prefixes of scope s: every non-empty,
// non-"xmlns" prefix declared anywhere in the chain, plus the reserved "xml",
// which is bound whether or not it is declared. Candidacy is not bindingness —
// a candidate may be shadowed or explicitly undeclared (xmlns:p="") further
// down — so a caller must resolve each one through lookup and keep only those
// that resolve. Order is unspecified; the exported enumerator sorts.
//
// The empty prefix is excluded: the default namespace is a separate fact,
// answered by lookup(""), and never one of these candidates. The reserved
// "xmlns" is excluded because an (ill-formed) xmlns:xmlns="…" declaration is
// recorded as a binding by bindingsOf and must never surface as a usable
// prefix.
func (s *scope) prefixes() []string {
	candidates := []string{xmlPrefix}
	seen := map[string]struct{}{xmlPrefix: {}}
	for cur := s; cur != nil; cur = cur.parent {
		for _, b := range cur.bindings {
			if b.prefix == "" || b.prefix == xmlnsPrefix {
				continue
			}
			if _, dup := seen[b.prefix]; dup {
				continue
			}
			seen[b.prefix] = struct{}{}
			candidates = append(candidates, b.prefix)
		}
	}
	return candidates
}

// Namespace is one prefix→namespace-name binding in scope at a StartElement:
// an entry of the element's in-scope namespaces with a non-empty prefix.
// Like Attribute and Name it is built only inside this package (unexported
// fields, accessor methods) and is immutable.
type Namespace struct {
	prefix string
	uri    string
}

// Prefix returns the bound prefix. It is never empty: the default namespace is
// not enumerated (see StartElement.InScopePrefixes).
func (n Namespace) Prefix() string { return n.prefix }

// URI returns the namespace name the prefix is bound to.
func (n Namespace) URI() string { return n.uri }

// InScopePrefixes returns every PREFIXED namespace binding in scope at this
// element, sorted by prefix (STYLE D2: a deterministic order, no map
// iteration). The reserved "xml" prefix is always present, declared or not.
//
// The default namespace is deliberately NOT included: that one fact is served
// by LookupPrefix(""), and duplicating it here would both encode one fact twice
// and offer callers a binding with an empty prefix, which several consumers
// (Structures §3.13.1's Namespace Binding {prefix} is a Required xs:NCName)
// cannot represent.
//
// Every candidate prefix is resolved through the same scope lookup LookupPrefix
// uses, so the two can never disagree: a shadowed binding reports its innermost
// namespace name, and a prefix explicitly undeclared with xmlns:p="" is absent
// from the result entirely.
func (e *StartElement) InScopePrefixes() []Namespace {
	candidates := e.scope.prefixes()
	slices.Sort(candidates)
	var out []Namespace
	for _, prefix := range candidates {
		uri, ok := e.scope.lookup(prefix)
		if !ok {
			continue
		}
		out = append(out, Namespace{prefix: prefix, uri: uri})
	}
	return out
}
