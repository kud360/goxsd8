// Package jsonsrc maps JSON documents onto the validate infoset so JSON
// instances can be assessed against a compiled schema set.
//
// The engine never imports a JSON package; this adapter builds infoset
// values (using the stdlib streaming Decoder token walk — member order
// preserved, positions tagged) and hands them over.
//
// # Mapping (contract; implemented in M8)
//
//   - Object members match by LOCAL NAME against the enclosing complex
//     type, schema-aware: a member resolves to a named attribute use OR
//     a child element declaration (content model walked via xsd's walk
//     API). A name declared as both → element wins + a non-fatal
//     warning on the Result.
//   - A scalar-valued member IS the element's simple content
//     ({"size":10} ≡ <size>10</size>); an object-valued member carries
//     attributes/children.
//   - An array value ⇒ repeated children under that key; JSON null ⇒
//     xsi:nil="true" (nillability itself stays the engine's cvc-elt
//     concern).
//   - Reserved members: "$type" ⇒ xsi:type; "$xmlns" ⇒ {prefix: uri}
//     bindings for QName-valued content, inherited by descendants
//     through an immutable parent-linked scope chain; an element's
//     default (empty-prefix) namespace is its matched declaration's
//     target namespace and deliberately shadows any "$xmlns" override
//     of "".
//
// Verdicts come from the same engine with the same cvc-* rule IDs as
// XML; the json conformance lane pins the mapping with curated cases.
package jsonsrc
