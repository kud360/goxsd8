package xsd

// NamespaceBinding is a Namespace Binding property record (Structures
// §3.13.1, id="nb"): {prefix} (an xs:NCName, Required) paired with
// {namespace} (an xs:anyURI, Required). One entry of an XPathExpression's
// {namespace bindings} set, corresponding to an in-scope-namespaces entry
// of the host element (§3.13.2).
//
// Construct only through NewNamespaceBinding. NamespaceBinding is immutable
// after construction.
type NamespaceBinding struct {
	prefix    string
	namespace string
}

// NewNamespaceBinding builds a NamespaceBinding pairing prefix with namespace.
func NewNamespaceBinding(prefix, namespace string) NamespaceBinding {
	return NamespaceBinding{prefix: prefix, namespace: namespace}
}

// Prefix returns the {prefix} property (an xs:NCName).
func (b NamespaceBinding) Prefix() string {
	return b.prefix
}

// Namespace returns the {namespace} property (an xs:anyURI).
func (b NamespaceBinding) Namespace() string {
	return b.namespace
}

// XPathExpression is the XPath Expression property record (Structures
// §3.13.1, id="x"): {namespace bindings} (a set of NamespaceBinding,
// modeled as a document-order slice per STYLE D2 — order carries no spec
// weight but determinism demands it), {default namespace} (an optional
// anyURI), {base URI} (an optional anyURI), and {expression} (the raw
// XPath 2.0 expression text, Required).
//
// {expression} is held VERBATIM — not parsed, not compiled, not evaluated.
// Like Annotation, this package preserves the XPath text but never
// interprets it; no dependency on package xpath. Compilation/evaluation is
// deferred to the M6/M7 XPath engine.
//
// {default namespace} is the ALREADY-RESOLVED value, not a raw local
// attribute: §3.13.2's mapping defines it as the result of a two-level
// chain (the host element's own xpathDefaultNamespace attribute, else the
// <schema> ancestor's, which itself defaults to ##local, resolving to
// absent) — that XML-document-tree walk is the parser/adapter's job, not
// this pure-leaf package's; XPathExpression simply carries the finished
// value (or its absence, which is a legitimate spec-sanctioned outcome, not
// an error).
//
// This type is shared machinery: it backs Assertion.Test() (this file's
// consumer) and is intended for reuse by TypeAlternative.Test() (a later
// issue, where {test} itself is Optional — unlike Assertion's Required) and
// identity-constraint {selector}/{fields} (a later issue) — §3.13.2 defines
// this property record once and all three XML mappings reuse it verbatim.
//
// Construct only through NewXPathExpression. XPathExpression is immutable
// after construction.
type XPathExpression struct {
	expression          string
	namespaceBindings   []NamespaceBinding
	defaultNamespace    string
	hasDefaultNamespace bool
	baseURI             string
	hasBaseURI          bool
}

// NewXPathExpression builds an XPathExpression. namespaceBindings is copied;
// the caller's backing array is not aliased. A nil defaultNamespace or
// baseURI means the corresponding property is absent; a non-nil pointer
// (including to "") means it is present, because "" is a legal anyURI and
// cannot double as an absence sentinel (mirrors AppInfo's {source}
// discipline).
//
// There is no rejectable state at this structural layer: {expression}'s
// "Required" is a presence requirement satisfied by the parameter existing,
// not a non-empty-string check — an empty XPath's legality is a
// static-analysis verdict the M6+ engine makes, not something this
// pure-leaf package can or should reject.
func NewXPathExpression(expression string, namespaceBindings []NamespaceBinding, defaultNamespace, baseURI *string) XPathExpression {
	x := XPathExpression{expression: expression}
	if len(namespaceBindings) > 0 {
		x.namespaceBindings = append([]NamespaceBinding(nil), namespaceBindings...)
	}
	if defaultNamespace != nil {
		x.defaultNamespace, x.hasDefaultNamespace = *defaultNamespace, true
	}
	if baseURI != nil {
		x.baseURI, x.hasBaseURI = *baseURI, true
	}
	return x
}

// Expression returns the {expression} property: the raw, verbatim XPath 2.0
// expression text. No parsed AST — this package only carries the text.
func (x XPathExpression) Expression() string {
	return x.expression
}

// NamespaceBindings returns the {namespace bindings} property in document
// order. It returns a copy: mutating the result does not affect x. An empty
// {namespace bindings} yields nil.
func (x XPathExpression) NamespaceBindings() []NamespaceBinding {
	if len(x.namespaceBindings) == 0 {
		return nil
	}
	return append([]NamespaceBinding(nil), x.namespaceBindings...)
}

// DefaultNamespace returns the {default namespace} property (an anyURI),
// already resolved through the §3.13.2 xpathDefaultNamespace chain; the
// second result is false when it is absent (a legitimate, spec-sanctioned
// outcome — see the type doc), in which case the first result is not
// meaningful.
func (x XPathExpression) DefaultNamespace() (string, bool) {
	return x.defaultNamespace, x.hasDefaultNamespace
}

// BaseURI returns the {base URI} property (an anyURI); the second result is
// false when it is absent, in which case the first result is not
// meaningful.
func (x XPathExpression) BaseURI() (string, bool) {
	return x.baseURI, x.hasBaseURI
}
