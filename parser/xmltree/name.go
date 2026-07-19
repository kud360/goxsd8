package xmltree

// XMLNamespaceURI is the namespace name permanently bound to the reserved
// prefix "xml". This binding is fixed by Namespaces in XML and needs no
// xmlns:xml declaration to resolve; it is generic-XML knowledge, not an
// XSD-layer fact, so it lives in this leaf (unlike the XMLSchema /
// XMLSchema-instance URIs, which belong to the xsd/parser packages). It is
// exported for parser, which recognizes xml:base attributes by matching
// Name{Space: XMLNamespaceURI, Local: "base"} rather than duplicating the URI.
const XMLNamespaceURI = "http://www.w3.org/XML/1998/namespace"

// xmlnsPrefix and xmlPrefix are the two reserved prefixes. "xmlns" may only
// introduce namespace declarations, never qualify an element or attribute
// name; "xml" is implicitly bound to XMLNamespaceURI.
const (
	xmlnsPrefix = "xmlns"
	xmlPrefix   = "xml"
)

// Name is a namespace-resolved element or attribute name: the (namespace
// name, local name) pair that Datatypes §3.3.18 QName defines as the value
// of a qualified name. Space is the namespace URI the prefix resolved to in
// the scope where the name occurred, or "" for a name in no namespace.
//
// Name is a comparable value; the zero Name is a name in no namespace with
// an empty local part.
type Name struct {
	space string
	local string
}

// Space returns the namespace URI of the name, or "" when the name is in no
// namespace (an unprefixed element under no default namespace, or any
// unprefixed attribute).
func (n Name) Space() string { return n.space }

// Local returns the local part of the name — the part after the prefix.
func (n Name) Local() string { return n.local }
