package xsd

// QName is an expanded name: Structures §2.2 defines it as "a pair
// consisting of a namespace name, which may be ·absent·, and a local name."
// §3.17.2.1 restates this for the QName value type ("expanded names with a
// local name and a namespace name"). Space is the namespace name; Local is
// the local name (an NCName).
//
// The zero value denotes an absent/anonymous name. This collapse is sound
// because no legal QName has an empty Local: local names are NCNames, whose
// grammar has no empty-string production, so "" can never be a real local
// name. Treating absence as the Go zero value is an implementation choice
// consistent with §2.2 key-null, which notes that "absent" need not be
// represented by a distinguished value.
//
// Note that a zero Space (no-namespace, a present name) is distinct from an
// absent ·target namespace· of a schema component (§2.2 key-targetNS); QName
// carries only the former distinction.
//
// QName is comparable: it is usable with == and as a map key, which the
// consumers (value.Backend, the component-model indexes) rely on.
type QName struct {
	Space, Local string
}

// String renders Clark notation: "{space}local" for a namespaced name,
// "local" for a no-namespace name, and "" for the zero value (absent).
func (q QName) String() string {
	if q.Space == "" {
		return q.Local
	}
	return "{" + q.Space + "}" + q.Local
}

// Display renders q for message text only: "xs:local" when q is in the XML
// Schema namespace ([XMLSchemaNS]), "xsi:local" when it is in the XML Schema
// instance namespace ([XMLSchemaInstanceNS]), "local" for a no-namespace name,
// "{space}local" for any other namespace, and "" for the zero value (absent).
// The "xs:" and "xsi:" are fixed display spellings, not namespace bindings and
// not lexical QNames: never parse Display's output back or key on it.
// [QName.String] stays Clark notation for every other consumer.
func (q QName) Display() string {
	if q.Space == XMLSchemaNS {
		return "xs:" + q.Local
	}
	if q.Space == XMLSchemaInstanceNS {
		return "xsi:" + q.Local
	}
	return q.String()
}
