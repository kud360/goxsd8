// Package xmlsrc adapts XML instance documents onto the validate
// infoset — the first and reference source adapter.
//
// # Contract (implemented in M5)
//
//	func Validate(v *validate.Validator, r io.Reader, opts ...Option) (*validate.Result, error)
//
// Backed by parser/xmltree: streaming, namespace-scoped, every node
// carrying Loc and byte offset, so each violation cites the exact
// instance position. WithURI names the document those positions belong
// to, since an io.Reader carries no name.
//
// The adapter decides no cvc- rule and names no attribute specially.
// xsi:type, xsi:nil, xsi:schemaLocation and xsi:noNamespaceSchemaLocation
// reach the engine as ordinary Attributes entries in the
// XMLSchema-instance namespace, per §2.7's note that they are attribute
// information items identified by namespace name and local name;
// cvc-complex-type clause 2's "excepting" carve-out is the engine's to
// apply, and a hint is a caller's to follow or ignore with its own
// loader.Resolver. This package reaches neither loader nor parser, which
// an imports test pins.
//
// One shared xmltree.Reader threads through the whole walk, and each
// element's Children cursor is a depth-tracked view over it: a subtree
// the engine is handed but does not descend into is discarded, never
// reported to its parent as further children. Character content arrives
// as one Text run per source run — text, CDATA and entity-expanded text
// are never coalesced, because the engine assembles the ·initial value·
// from the runs and a coalesced run would carry only the first one's Loc.
//
//   - GAP(xml): content OUTSIDE the document element is not inspected:
//     character data before it is dropped, and anything after its end tag
//     is never read, so trailing character content and a second document
//     element alike go unreported. Tracked by #753.
package xmlsrc
