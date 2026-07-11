package xsd

// XMLSchemaNS is the XML Schema namespace name, the literal
// "http://www.w3.org/2001/XMLSchema" (Structures §1.3.1.1 "The Schema
// Namespace (xs)", anchor xsd-namespace; independently stated for the built-in
// datatypes in Datatypes §3.1 "Namespace considerations").
//
// This is the exact string every builtin type carries as its [QName.Space]:
// builtin.Seed uses this constant as every builtin type's QName.Space, so a
// consumer keys a value.Backend.Mapping lookup on it rather than hand-rolling
// the literal, e.g.
//
//	m, ok := backend.Mapping(xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"})
//
// It is a plain untyped string, not a named type: [QName.Space] is a plain
// string and namespace names are an open universe (unlike the closed-set tags
// in closedsets.go), so a named type would add conversion friction without any
// T1 safety.
//
// Scope: XMLSchemaNS and [XMLSchemaInstanceNS] are only the two well-known
// XSD URIs — the schema vocabulary and the instance-markup vocabulary. This is
// not a general namespace registry; other well-known URIs (the XML namespace,
// the versioning namespace, the XMLSchema-datatypes namespace) are out of scope
// and do not live here.
const XMLSchemaNS = "http://www.w3.org/2001/XMLSchema"

// XMLSchemaInstanceNS is the XML Schema Instance namespace name, the literal
// "http://www.w3.org/2001/XMLSchema-instance" (Structures §1.3.1.2 "The Schema
// Instance Namespace (xsi)", anchor xsi-namespace). It is the namespace of the
// instance-document attributes xsi:type, xsi:nil, xsi:schemaLocation and
// xsi:noNamespaceSchemaLocation (Structures §2.7 "Schema-Related Markup in
// Documents Being Validated").
//
// Scope: [XMLSchemaNS] and XMLSchemaInstanceNS are only the two well-known XSD
// URIs — the schema vocabulary and the instance-markup vocabulary. This is not
// a general namespace registry; other well-known URIs are out of scope and do
// not live here.
const XMLSchemaInstanceNS = "http://www.w3.org/2001/XMLSchema-instance"
