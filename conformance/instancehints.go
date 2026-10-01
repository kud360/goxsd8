package conformance

import (
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/internal/schemaloc"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file gives the instance lane a schema for an instanceTest whose test
// group declares no schemaTest (groupSchemaDocs yields none): the schema its
// own xsi:schemaLocation and xsi:noNamespaceSchemaLocation hints locate
// (§2.7.3, §4.3.2), assembled through parser.ParseSet and parser.HintAt
// (#2013). §4.3.2 clause 3 lets a processor dereference hints and obliges it to
// dereference none, so every shape below that the lane cannot read completely
// DECLINES rather than deciding against a partial schema.
//
// The reader is a second copy of cmd/goxsd8's instanceHints/hintsOf, narrowed
// to what the lane can decide; #755 owns the single home both should share
// (STYLE T4).

// caseSchema assembles the schema c is assessed against, with assembleCase's
// four results: the group's schema documents where discovery attached them,
// and otherwise the schema the instance's own hints locate (#2013, §4.3.2
// clauses 3-5). A case with neither — no group schema AND no hint — has no
// stated schema at all and is not decidable: ParseSet over no root answers a
// plain error, which read as a verdict would be a schema rejection nobody
// stated.
func caseSchema(backend value.Backend, c caseSpec) (*xsd.Schema, *parser.AssemblyReport, bool, error) {
	if c.schemaDoc != "" {
		return assembleCase(backend, c.schemaDoc, c.schemaExtraDocs)
	}
	root, hints, ok := instanceHints(c.doc)
	if !ok || len(hints) == 0 {
		return nil, nil, false, nil
	}
	return assembleHints(backend, c.doc, root, hints)
}

// instanceHints reads the schema location hints off the document element of
// the instance at doc, each resolved against the instance's base URI through
// internal/schemaloc (§4.3.2 clause 4), in the order its attributes carry them,
// together with that element's ·expanded name·. ok is false — the caller
// DECLINES — when the hints the root carries may not be the whole of what
// §4.3.2 clause 5 makes global to the assessment:
//
//   - the document will not resolve or decode, or carries a DOCTYPE, whose DTD
//     could default a hint onto any element (rootStart);
//   - the root carries xml:base, which moves the base URI clause 4 resolves
//     against;
//   - an xsi:schemaLocation value has an odd member count, so one location
//     pairs with no namespace;
//   - any element below the root carries a hint, or any element is an inline
//     xs:schema: clause 5 makes either one's effect global, and a root-only
//     read would assess against a schema short of it (the addB156-addB169
//     ordering family, several decided VALID on suite-invalid fixtures
//     otherwise).
//
// A root carrying no hint is ok with no hints; caseSchema declines that shape
// too.
func instanceHints(doc string) (root xsd.QName, hints []parser.Root, ok bool) {
	rc, _, err := loader.Dir(filepath.Dir(doc)).Resolve("", filepath.Base(doc))
	if err != nil {
		return xsd.QName{}, nil, false
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	dec := xml.NewDecoder(rc)
	start, ok := rootStart(dec)
	if !ok || isInlineSchema(start.Name) {
		return xsd.QName{}, nil, false
	}
	hints, ok = hintsOf(start, filepath.Base(doc))
	if !ok || !belowRootHintFree(dec) {
		return xsd.QName{}, nil, false
	}
	return expandedName(start.Name), hints, true
}

// hintsOf reads root's hints, resolving each location against base, the
// instance's base URI as the resolver serving it spells it: xsi:schemaLocation
// pairs a namespace with a location, and xsi:noNamespaceSchemaLocation names a
// location whose document has no target namespace, parser.HintAt's "". ok is
// false for an xml:base on root and for an odd xsi:schemaLocation member count.
func hintsOf(root xml.StartElement, base string) (hints []parser.Root, ok bool) {
	for _, a := range root.Attr {
		if a.Name.Space == xmlPrefixNS && a.Name.Local == "base" {
			return nil, false
		}
		if !isLocationHint(a) {
			continue
		}
		fields := strings.Fields(a.Value)
		if a.Name.Local == "noNamespaceSchemaLocation" {
			for _, location := range fields {
				hints = append(hints, parser.HintAt("", schemaloc.Resolve(base, location)))
			}
			continue
		}
		if len(fields)%2 != 0 {
			return nil, false
		}
		for i := 0; i < len(fields); i += 2 {
			hints = append(hints, parser.HintAt(fields[i], schemaloc.Resolve(base, fields[i+1])))
		}
	}
	return hints, true
}

// belowRootHintFree reads dec, positioned just past the document element's
// start tag, to the end of the document, and reports whether it decodes and no
// element there carries a hint or is an inline xs:schema.
func belowRootHintFree(dec *xml.Decoder) bool {
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return true
		}
		if err != nil {
			return false
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if isInlineSchema(start.Name) || slices.ContainsFunc(start.Attr, isLocationHint) {
			return false
		}
	}
}

// isInlineSchema reports whether an element named n is an xs:schema.
func isInlineSchema(n xml.Name) bool {
	return n.Space == xsd.XMLSchemaNS && n.Local == "schema"
}

// assembleHints assembles the schema the hints locate, the instance lane's
// counterpart of assembleCase for a case with no group schema, and reports
// whether the outcome may be read on assembleCase's terms. decidable is false —
// the caller DECLINES — when:
//
//   - a hint resolved to no document (parser.AssemblyReport.UnfollowedRoots,
//     legal to skip under §4.2.6.2), so the schema is short of a document the
//     instance named;
//   - the closure leaves the decidable subset (closureDecidable), or the
//     rejection is one an unfollowed directive could have fabricated
//     (fabricatedRejection);
//   - a document of the closure carries an empty prefixed namespace
//     declaration (closurePrefixUndeclared);
//   - the assembly succeeded and declares no top-level element for root. A
//     root the hinted schema does not declare is charged cvc-assess-elt, and
//     §4.3.2 clause 3's "less than complete ·assessment· outcomes" is exactly
//     that charge when a hinted document was read short: addB063's hinted
//     test72702.imp carries character data after its document element, which
//     the parser accepts, and the suite declares the case invalid for that
//     document rather than for the undeclared root.
//
// The resolver is assembleCase's: pinnedResolver over a loader.Dir rooted at
// the instance's own directory, which every hint location is relative to
// (instanceHints), so a location climbing above it is refused as unresolved.
func assembleHints(backend value.Backend, doc string, root xsd.QName, hints []parser.Root) (*xsd.Schema, *parser.AssemblyReport, bool, error) {
	resolver := pinnedResolver{dir: loader.Dir(filepath.Dir(doc))}
	schema, report, perr := parser.ParseSet(hints, parser.WithResolver(resolver), parser.WithBackend(backend))
	if len(report.UnfollowedRoots()) > 0 || !closureDecidable(report) || fabricatedRejection(report, perr) {
		return nil, nil, false, nil
	}
	if closurePrefixUndeclared(report) {
		return nil, nil, false, nil
	}
	if perr != nil {
		return schema, report, true, perr
	}
	if _, ok := schema.Element(root); !ok {
		return nil, nil, false, nil
	}
	return schema, report, true, nil
}

// closurePrefixUndeclared reports whether any schema document the assembly read
// carries a prefixed namespace declaration with an empty value (xmlns:p=""), or
// cannot be read to say. Namespaces in XML 1.0's "No Prefix Undeclaring"
// constraint makes such a document not namespace-well-formed, the parser
// accepts it, and the suite declares addB138 and addB139 invalid for exactly
// that: assessed against what the parser built, addB139 comes out valid and
// addB138 invalid on an unrelated wildcard charge. Each document is re-read
// from its parser.AssembledDocument.Location, an on-disk path for
// pinnedResolver, as closureVersioned does.
func closurePrefixUndeclared(report *parser.AssemblyReport) bool {
	return slices.ContainsFunc(report.Documents(), func(d parser.AssembledDocument) bool {
		return documentCarries(d.Location, isPrefixUndeclaring)
	})
}

// isPrefixUndeclaring reports whether a is a prefixed namespace declaration
// with an empty value.
func isPrefixUndeclaring(a xml.Attr) bool {
	return a.Name.Space == "xmlns" && a.Value == ""
}
