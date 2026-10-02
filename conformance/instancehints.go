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
// four results, its decidable false reported as the refusal that declined the
// case: the group's schema documents where discovery attached them
// (refuseGroupAssembly), and otherwise the schema the instance's own hints
// locate (#2013, §4.3.2 clauses 3-5). A case with neither — no group schema
// AND no hint — has no stated schema at all and is not decidable
// (refuseNoHint): ParseSet over no root answers a plain error, which read as a
// verdict would be a schema rejection nobody stated.
func caseSchema(backend value.Backend, c caseSpec) (*xsd.Schema, *parser.AssemblyReport, refusal, error) {
	if c.schemaDoc != "" {
		schema, report, decidable, perr := assembleCase(backend, c.schemaDoc, c.schemaExtraDocs)
		if !decidable {
			return nil, nil, refuseGroupAssembly, nil
		}
		return schema, report, "", perr
	}
	root, hints, why := instanceHints(c.doc)
	if why != "" {
		return nil, nil, why, nil
	}
	if len(hints) == 0 {
		return nil, nil, refuseNoHint, nil
	}
	return assembleHints(backend, c.doc, root, hints)
}

// instanceHints reads the schema location hints off the document element of
// the instance at doc, each resolved against the instance's base URI through
// internal/schemaloc (§4.3.2 clause 4), in the order its attributes carry them,
// together with that element's ·expanded name·. It names a refusal — the
// caller DECLINES — when the hints the root carries may not be the whole of
// what §4.3.2 clause 5 makes global to the assessment:
//
//   - the document will not resolve or decode, or carries a DOCTYPE whose DTD
//     could default a hint onto any element: one with an external subset, an
//     <!ATTLIST or a parameter-entity reference (rootStart);
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
// A root carrying no hint names no refusal and returns no hints; caseSchema
// declines that shape too.
func instanceHints(doc string) (root xsd.QName, hints []parser.Root, why refusal) {
	rc, _, err := loader.Dir(filepath.Dir(doc)).Resolve("", filepath.Base(doc))
	if err != nil {
		return xsd.QName{}, nil, refuseHintsUnresolved
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	dec := rawDecoder(rc)
	start, why := rootStart(dec)
	if why != "" {
		return xsd.QName{}, nil, why
	}
	if isInlineSchema(start.Name) {
		return xsd.QName{}, nil, refuseInlineSchema
	}
	hints, why = hintsOf(start, filepath.Base(doc))
	if why != "" {
		return xsd.QName{}, nil, why
	}
	if why := belowRootHintFree(dec); why != "" {
		return xsd.QName{}, nil, why
	}
	return expandedName(start.Name), hints, ""
}

// hintsOf reads root's hints, resolving each location against base, the
// instance's base URI as the resolver serving it spells it: xsi:schemaLocation
// pairs a namespace with a location, and xsi:noNamespaceSchemaLocation names a
// location whose document has no target namespace, parser.HintAt's "". It
// refuses an xml:base on root (refuseXMLBase) and an odd xsi:schemaLocation
// member count (refuseOddLocation).
func hintsOf(root xml.StartElement, base string) (hints []parser.Root, why refusal) {
	for _, a := range root.Attr {
		if a.Name.Space == xmlPrefixNS && a.Name.Local == "base" {
			return nil, refuseXMLBase
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
			return nil, refuseOddLocation
		}
		for i := 0; i < len(fields); i += 2 {
			hints = append(hints, parser.HintAt(fields[i], schemaloc.Resolve(base, fields[i+1])))
		}
	}
	return hints, ""
}

// belowRootHintFree reads dec, positioned just past the document element's
// start tag, to the end of the document, and names the refusal where it does
// not decode (refuseDecode) or an element there is an inline xs:schema
// (refuseInlineSchema) or carries a hint (refuseHintBelowRoot).
func belowRootHintFree(dec *xml.Decoder) refusal {
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return ""
		}
		if err != nil {
			return refuseDecode
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if isInlineSchema(start.Name) {
			return refuseInlineSchema
		}
		if slices.ContainsFunc(start.Attr, isLocationHint) {
			return refuseHintBelowRoot
		}
	}
}

// isInlineSchema reports whether an element named n is an xs:schema.
func isInlineSchema(n xml.Name) bool {
	return n.Space == xsd.XMLSchemaNS && n.Local == "schema"
}

// assembleHints assembles the schema the hints locate, the instance lane's
// counterpart of assembleCase for a case with no group schema, and names the
// refusal where the outcome may not be read on assembleCase's terms — the
// caller DECLINES — which is when:
//
//   - a hint resolved to no document (parser.AssemblyReport.UnfollowedRoots,
//     legal to skip under §4.2.6.2), so the schema is short of a document the
//     instance named (refuseHintUnfollowed);
//   - assemblyDeclined, assembleCase's own gate, refuses the outcome: the
//     closure leaves the decidable subset (closureDecidable) and the assembly
//     did not fail with a grammarRejection, or the rejection is one an
//     unfollowed directive could have fabricated (fabricatedRejection)
//     (refuseHintAssembly);
//   - the assembly succeeded and declares no top-level element for root
//     (refuseHintUndeclared). A root no declaration governs is laxly
//     assessed (cvc-assess-elt clause 3), and §3.3.5.1 gives a root not
//     strictly assessed [validity] notKnown, not invalid; no ruling
//     licenses deciding such a case "not valid".
//
// The resolver is assembleCase's: pinnedResolver over a loader.Dir rooted at
// the instance's own directory, which every hint location is relative to
// (instanceHints), so a location climbing above it is refused as unresolved.
func assembleHints(backend value.Backend, doc string, root xsd.QName, hints []parser.Root) (*xsd.Schema, *parser.AssemblyReport, refusal, error) {
	resolver := pinnedResolver{dir: loader.Dir(filepath.Dir(doc))}
	schema, report, perr := parser.ParseSet(hints, parser.WithResolver(resolver), parser.WithBackend(backend))
	if len(report.UnfollowedRoots()) > 0 {
		return nil, nil, refuseHintUnfollowed, nil
	}
	if assemblyDeclined(report, perr) {
		return nil, nil, refuseHintAssembly, nil
	}
	if perr != nil {
		return schema, report, "", perr
	}
	if _, ok := schema.Element(root); !ok {
		return nil, nil, refuseHintUndeclared, nil
	}
	return schema, report, "", nil
}
