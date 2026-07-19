package xmltree

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
