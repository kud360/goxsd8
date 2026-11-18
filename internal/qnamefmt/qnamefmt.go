// Package qnamefmt spells an expanded name for output, once for every library
// package that renders one: xsd's QName.String and QName.Display, and
// parser/xmltree's well-formedness messages, which cannot import xsd.
//
// # Contract
//
//	const XMLSchemaNS = "http://www.w3.org/2001/XMLSchema"
//	const XMLSchemaInstanceNS = "http://www.w3.org/2001/XMLSchema-instance"
//	func Clark(space, local string) string
//	func Display(space, local string) string
//
// Clark and Display spell a name exactly as xsd.QName.String and
// xsd.QName.Display do, which delegate here; those two doc comments own the
// spellings. The constants are the values xsd.XMLSchemaNS and
// xsd.XMLSchemaInstanceNS are defined from, and those doc comments own what
// each names.
//
// It depends on nothing but the standard library.
package qnamefmt

// XMLSchemaNS is the value of xsd.XMLSchemaNS.
const XMLSchemaNS = "http://www.w3.org/2001/XMLSchema"

// XMLSchemaInstanceNS is the value of xsd.XMLSchemaInstanceNS.
const XMLSchemaInstanceNS = "http://www.w3.org/2001/XMLSchema-instance"

// Clark is the spelling xsd.QName.String documents.
func Clark(space, local string) string {
	if space == "" {
		return local
	}
	return "{" + space + "}" + local
}

// Display is the spelling xsd.QName.Display documents.
func Display(space, local string) string {
	if space == XMLSchemaNS {
		return "xs:" + local
	}
	if space == XMLSchemaInstanceNS {
		return "xsi:" + local
	}
	return Clark(space, local)
}
