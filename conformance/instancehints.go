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

// This file gives the instance lane the schema location hints its instance
// carries, xsi:schemaLocation and xsi:noNamespaceSchemaLocation (§2.7.3,
// §4.3.2), assembled through parser.ParseSet and parser.HintAt. §4.3.2 clause 3
// lets a processor dereference hints and obliges it to dereference none. For an
// instanceTest whose test group declares no schemaTest (groupSchemaDocs yields
// none) the hints of every element, the first per namespace, locate the whole
// schema (#2013, #2171), and every shape the lane cannot read completely
// DECLINES rather than deciding against a partial schema. A document carrying
// no hint is assessed against the built-in components alone where its root's
// xsi:type names a built-in type definition (builtinsSchema, #2151). For a
// group that declares one, the root's hints alone only add namespaces the
// group's documents leave out, and never decline (groupSchema, #771).
//
// The reader is a second copy of cmd/goxsd8's instanceHints/hintsOf, narrowed
// to what the lane can decide, and on the no-schemaTest path it reads the hints
// below the document element that the CLI's copy ignores; #755 owns the single
// home both should share (STYLE T4).

// caseSchema assembles the schema c is assessed against, with assembleCase's
// four results, its decidable false reported as the refusal that declined the
// case: the group's schema documents where discovery attached them, widened by
// the hints groupSchema adds, otherwise the schema the instance's own hints
// locate (#2013, §4.3.2 clauses 3-5), and for a case with neither — no group
// schema AND no hint — the built-in components where builtinsSchema finds the
// case decidable against them.
func caseSchema(backend value.Backend, c caseSpec) (*xsd.Schema, *parser.AssemblyReport, refusal, error) {
	if c.schemaDoc != "" {
		return groupSchema(backend, c)
	}
	root, hints, why := instanceHints(c.doc)
	if why != "" {
		return nil, nil, why, nil
	}
	if len(hints) == 0 {
		return builtinsSchema(backend, root)
	}
	return assembleHints(backend, c.doc, expandedName(root.Name), hints)
}

// groupSchema is what caseSchema answers for a case whose group declares a
// schemaTest: the schema assembleCase assembles from the group's documents
// (refuseGroupAssembly where it declines), widened by the hints of c's instance
// root for namespaces that assembly leaves uncovered (#771, §4.3.2 clauses 3-5).
//
// A hint is added only where its namespace — "" for
// xsi:noNamespaceSchemaLocation — is the targetNamespace of no document the
// group's closure consumed (closureNamespaces). One for a covered namespace is
// left out: a stale hint naming a document that redeclares the group's
// components would make the union a schema error (sch-props-correct clause 2),
// and §4.2.6.2's Note lets a processor keep the components it already has for
// a namespace. A chameleon-included document takes its includer's namespace,
// which the includer's own targetNamespace already covers.
//
// A hint that cannot be followed, or a widened assembly that declines or errs,
// never declines the case and never decides it invalid; a followed hint's
// components are assessed like any other. §4.3.2 clause 3 makes a hint that
// cannot be followed no error, so every failure falls back to the group's own
// schema — where the group's assembly errs (its perr is what the caller reads,
// as without hints), the instance has a shape instanceHints refuses, no root
// hint names an uncovered namespace (a hint below the root is never read on
// this path), or the widened assembly declines or errs (src-import clause 3.1,
// a hinted document whose targetNamespace is not the hint's). A hint resolving
// to no document is skipped by parser.ParseSet itself
// (parser.AssemblyReport.UnfollowedRoots), and the widened assembly is used
// without it.
func groupSchema(backend value.Backend, c caseSpec) (*xsd.Schema, *parser.AssemblyReport, refusal, error) {
	schema, report, decidable, perr := assembleCase(backend, c.schemaDoc, c.schemaExtraDocs)
	if !decidable {
		return nil, nil, refuseGroupAssembly, nil
	}
	if perr != nil {
		return schema, report, "", perr
	}
	hints := uncoveredHints(c, report)
	if len(hints) == 0 {
		return schema, report, "", nil
	}
	hinted, hintedReport, decidable, perr := assembleCaseWith(backend, c.schemaDoc, c.schemaExtraDocs, hints)
	if decidable && perr == nil {
		return hinted, hintedReport, "", nil
	}
	return schema, report, "", nil
}

// uncoveredHints returns, as parser.HintAt roots, the hints of c's instance root
// whose namespace no document of report covers (closureNamespaces), and none
// where instanceHints refuses the instance. A hint below the root is not read
// here (#771's GAP(conformance) in instance.go). Each location is resolved
// against the instance's path relative to c.schemaDoc's directory, the one
// assembleCaseWith's resolver serves, so a location climbing above it resolves
// to no document.
func uncoveredHints(c caseSpec, report *parser.AssemblyReport) []parser.Root {
	root, _, why := instanceHints(c.doc)
	if why != "" {
		return nil
	}
	rel, err := filepath.Rel(filepath.Dir(c.schemaDoc), c.doc)
	if err != nil {
		return nil
	}
	// hintsOf's one refusal, an odd member count, depends on root alone, and
	// instanceHints has just read root's hints without it, so this read names
	// none.
	hints, _ := hintsOf(root, filepath.ToSlash(rel), closureNamespaces(report))
	return hints
}

// closureNamespaces is the set of targetNamespace values of the documents
// report lists, an absent attribute read as "": the namespaces the assembly
// already holds components for. A chameleon-included document has no
// targetNamespace and so adds "" too, which can only withhold an
// xsi:noNamespaceSchemaLocation hint and leave the group's own schema.
func closureNamespaces(report *parser.AssemblyReport) map[string]bool {
	covered := map[string]bool{}
	for _, d := range report.Documents() {
		tns, _ := d.Doc.Root().Attr("targetNamespace")
		covered[tns] = true
	}
	return covered
}

// builtinsSchemaDoc is the package-relative path of a schema document with no
// children and no targetNamespace: the schema assembled from it holds the
// built-in components alone, which every schema contains — the built-in type
// definitions (§3.4.7, §3.16.7), all in the XSD namespace, and the four xsi:
// attribute declarations (§3.2.7) — and no element declaration.
const builtinsSchemaDoc = "testdata/builtins/builtins.xsd"

// builtinsSchema is what caseSchema answers for a case with no group schema
// whose document element, root, carries no hint: the schema assembled from
// builtinsSchemaDoc through assembleCase, handed out only where the spec
// decides the case against the built-in components, and otherwise the refusal
// naming the arm that leaves it undecided — the caller DECLINES (#2151). With
// no hint to follow, and §4.3.2 clause 3 obliging a processor to follow none,
// the built-in components every schema holds are the whole schema. The arms:
//
//   - root's xsi:type, resolved against the namespace declarations root
//     carries (resolveQName) — its whole [in-scope namespaces] but xml:, root
//     being the document element — names a type definition of that schema,
//     so a built-in one in the XSD namespace. With no ·governing element
//     declaration·, it is root's ·governing type definition·
//     (key-governing-type-elem clause 8, key-itd), root is ·strictly assessed·
//     against it (cvc-assess-elt clause 1), and cvc-type decides root. The
//     schema is handed out.
//   - root's xsi:type resolves to no QName, or to one naming no built-in type
//     definition (refuseNoHintXsiTypeUnresolved). key-itd clause 3 fails, so
//     root has no ·governing type definition·, is ·laxly assessed·
//     (cvc-assess-elt clause 3), and its [validity] is notKnown (e-validity
//     clause 2). cvc-attribute clause 5 charges the xsi:type attribute, not
//     root.
//   - root carries no xsi:type, and an attribute in the xsi namespace whose
//     local name is none of §3.2.7's four (unknownXsi,
//     refuseNoHintUnknownXsi). Root is ·laxly assessed· against xs:anyType,
//     whose lax attribute wildcard admits the attribute, and no declaration
//     exists to assess it (cvc-assess-elt clause 2.2): no rule charges it, so
//     a suite expectation of invalid is a suite expectation (PRINCIPLES 25),
//     not a verdict.
//   - root carries neither (refuseNoHint): a ·laxly assessed· root, its
//     [validity] notKnown (#2013).
//
// Handed the schema, the walk would decide either undecided arm on its
// cvc-assess-elt charge for a root with no top-level declaration and no
// resolving xsi:type, a ·strict assessment· the spec does not make there.
func builtinsSchema(backend value.Backend, root xml.StartElement) (*xsd.Schema, *parser.AssemblyReport, refusal, error) {
	i := slices.IndexFunc(root.Attr, func(a xml.Attr) bool { return a.Name == xsiType })
	if i < 0 {
		if slices.ContainsFunc(root.Attr, unknownXsi) {
			return nil, nil, refuseNoHintUnknownXsi, nil
		}
		return nil, nil, refuseNoHint, nil
	}
	name, ok := resolveQName(root.Attr, root.Attr[i].Value)
	if !ok {
		return nil, nil, refuseNoHintXsiTypeUnresolved, nil
	}
	schema, report, decidable, perr := assembleCase(backend, builtinsSchemaDoc, nil)
	if !decidable {
		return nil, nil, refuseBuiltinsAssembly, nil
	}
	if perr != nil {
		return schema, report, "", perr
	}
	if _, ok := schema.Type(name); !ok {
		return nil, nil, refuseNoHintXsiTypeUnresolved, nil
	}
	return schema, report, "", nil
}

// unknownXsi reports whether a is an attribute in the xsi namespace whose
// local name is none of the four §3.2.7 declares, compared case-sensitively,
// so xsi:Type is one.
func unknownXsi(a xml.Attr) bool {
	return a.Name.Space == xsd.XMLSchemaInstanceNS && notExcepted(a)
}

// instanceHints reads the schema location hints of the instance at doc, in
// document order, the document element's first, together with that element's
// start tag. Each location is resolved against the instance's base URI through
// internal/schemaloc (§4.3.2 clause 4). §4.3.2 clause 5 makes a hint on any
// element global to the assessment, so every element's hints are read before
// assessment begins, a whole-document reading §4.1 lets stand for the lazy one
// (#2171). The first element to hint a namespace — "" for
// xsi:noNamespaceSchemaLocation — supplies it, and a later element's hint for
// that namespace is left out (hintsOf), so it never reaches parser.ParseSet:
// clause 3 makes following any hint optional, and sch-props-correct clause 2
// would make a second document redeclaring the first one's components a schema
// error. The namespace is the hint's, never that of the element carrying it.
//
// It names a refusal — the caller DECLINES — when the hints read may not be the
// whole of what clause 5 makes global, or may not resolve where clause 4 says:
//
//   - the document will not resolve or decode, or carries a DOCTYPE whose DTD
//     could default a hint onto any element: one with an external subset, an
//     <!ATTLIST or a parameter-entity reference (rootStart);
//   - the root carries xml:base, or an element carrying a hint has xml:base
//     on itself or an ancestor: it moves the base URI clause 4 resolves that
//     element's locations against;
//   - an xsi:schemaLocation value has an odd member count, so one location
//     pairs with no namespace;
//   - any element is an inline xs:schema, whose effect clause 5 makes global
//     too and which the lane does not read.
//
// A document carrying no hint names no refusal and returns no hints;
// caseSchema hands that shape to builtinsSchema.
//
// The reading decides four suite-invalid cases of the MS-Additional ordering
// family (test group addB156-addB166) VALID, each an expected divergence
// (PRINCIPLES 25): addB159 (test93490_5.xml), addB161 (test93490_7.xml),
// addB162 (test93490_8.xml) and addB164 (test93490_10.xml). Each hints, below
// the document element, a namespace an earlier hint already supplied, naming a
// different document, and the family's description charges that ordering
// ("schema location or inline schema seen after item from schema target
// namespace seen"). No rule of the Recommendation does: §4.3.2 clause 5 gives
// a hint on any element global effect and names no error for a late one, and
// clause 3 makes following it optional, so the later document is left out and
// the instance is valid against the earlier one's components. addB156 and
// addB158 have the same shape, and the suite expects both valid under 1.1.
func instanceHints(doc string) (root xml.StartElement, hints []parser.Root, why refusal) {
	rc, _, err := loader.Dir(filepath.Dir(doc)).Resolve("", filepath.Base(doc))
	if err != nil {
		return xml.StartElement{}, nil, refuseHintsUnresolved
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	dec := rawDecoder(rc)
	start, why := rootStart(dec)
	if why != "" {
		return xml.StartElement{}, nil, why
	}
	if isInlineSchema(start.Name) {
		return xml.StartElement{}, nil, refuseInlineSchema
	}
	if slices.ContainsFunc(start.Attr, isXMLBase) {
		return xml.StartElement{}, nil, refuseXMLBase
	}
	base := filepath.Base(doc)
	supplied := map[string]bool{}
	hints, why = hintsOf(start, base, supplied)
	if why != "" {
		return xml.StartElement{}, nil, why
	}
	later, why := belowRootHints(dec, base, supplied)
	if why != "" {
		return xml.StartElement{}, nil, why
	}
	return start, append(hints, later...), ""
}

// hintsOf reads el's hints, resolving each location against base, the
// instance's base URI as the resolver serving it spells it: xsi:schemaLocation
// pairs a namespace with a location, and xsi:noNamespaceSchemaLocation names a
// location whose document has no target namespace, parser.HintAt's "". It
// leaves out every hint whose namespace covered holds, then adds the namespace
// of every hint it returns to covered, so a later element's hint for that
// namespace is left out too (first-wins); el's own hints for one namespace are
// all returned. It refuses an odd xsi:schemaLocation member count
// (refuseOddLocation).
func hintsOf(el xml.StartElement, base string, covered map[string]bool) (hints []parser.Root, why refusal) {
	var namespaces []string
	for _, a := range el.Attr {
		if !isLocationHint(a) {
			continue
		}
		fields := strings.Fields(a.Value)
		if a.Name.Local == "noNamespaceSchemaLocation" {
			if covered[""] {
				continue
			}
			for _, location := range fields {
				hints = append(hints, parser.HintAt("", schemaloc.Resolve(base, location)))
				namespaces = append(namespaces, "")
			}
			continue
		}
		if len(fields)%2 != 0 {
			return nil, refuseOddLocation
		}
		for i := 0; i < len(fields); i += 2 {
			if covered[fields[i]] {
				continue
			}
			hints = append(hints, parser.HintAt(fields[i], schemaloc.Resolve(base, fields[i+1])))
			namespaces = append(namespaces, fields[i])
		}
	}
	for _, ns := range namespaces {
		covered[ns] = true
	}
	return hints, ""
}

// belowRootHints reads dec, positioned just past the document element's start
// tag, to the end of the document, and returns the hints hintsOf reads off
// every element there, in document order, against base and supplied. It names
// the refusal where dec does not decode (refuseDecode), an element is an inline
// xs:schema (refuseInlineSchema), or an element carrying a hint has xml:base on
// itself or an ancestor (refuseXMLBase), and passes on hintsOf's.
func belowRootHints(dec *xml.Decoder, base string, supplied map[string]bool) (hints []parser.Root, why refusal) {
	// based holds, per open element, whether an xml:base is in scope there. The
	// document element's entry is false: instanceHints refuses one there.
	based := []bool{false}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return hints, ""
		}
		if err != nil {
			return nil, refuseDecode
		}
		if _, ok := tok.(xml.EndElement); ok {
			based = based[:len(based)-1]
			continue
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if isInlineSchema(start.Name) {
			return nil, refuseInlineSchema
		}
		inBase := based[len(based)-1] || slices.ContainsFunc(start.Attr, isXMLBase)
		based = append(based, inBase)
		if !slices.ContainsFunc(start.Attr, isLocationHint) {
			continue
		}
		if inBase {
			return nil, refuseXMLBase
		}
		got, why := hintsOf(start, base, supplied)
		if why != "" {
			return nil, why
		}
		hints = append(hints, got...)
	}
}

// isXMLBase reports whether a is xml:base.
func isXMLBase(a xml.Attr) bool {
	return a.Name.Space == xmlPrefixNS && a.Name.Local == "base"
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
