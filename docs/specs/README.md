# Local specs — the ground truth

Agents grep these files instead of trusting memory (STYLE P1). Never
answer a spec question from recollection when the clause is one grep away.

| File | Spec |
|---|---|
| `md/xmlschema11-1.md` | XSD 1.1 Part 1: Structures |
| `md/xmlschema11-2.md` | XSD 1.1 Part 2: Datatypes (Appendix E hfn definitions are the source of truth for builtin types) |
| `md/xpath20.md` | XPath 2.0 |
| `md/xpath-functions.md` | XQuery 1.0 and XPath 2.0 Functions and Operators (F&O) — the function library and regex flavor XPath 2.0 binds to |
| `md/xpath-datamodel.md` | XQuery 1.0 and XPath 2.0 Data Model (XDM), 2nd Edition — typed value, string value, and node accessors the F&O functions are defined on |
| `md/xsd-precisionDecimal.md` | The precisionDecimal datatype |
| `md/xml-names.md` | Namespaces in XML 1.0, 3rd Edition — `QName`/`NCName`/`Prefix` productions and prefix-binding rules XSD §1.4 binds to |
| `md/xml.md` | Extensible Markup Language (XML) 1.0, 5th Edition — `Char`/`Name`/`S` (whitespace) productions and well-formedness constraints the reader and Name-family datatypes rest on |
| `md/xml-infoset.md` | XML Information Set, 2nd Edition — the information items and properties XSD Structures and the PSVI are defined in terms of |

`html/` holds the pristine committed downloads (`go tool fetchspecs`
refreshes them); `md/` is generated from them by `go tool spec2md` via
`go generate ./...` — same input, byte-identical output. Edit neither by
hand. The F&O and XDM downloads are pinned to their dated 1.0/2nd-Edition
URIs (the undated shortnames have moved to 3.x); the other seven track the
editions XSD 1.1 §1.4 cites via their still-current undated shortnames.

XML 1.1 and Namespaces in XML 1.1 are deliberately omitted: XSD §1.4 states
their `NCName`/`Name`/whitespace definitions are identical to the 1.0
editions here, so the 1.0 copies are the ground truth for grepping.

## Grep conventions (anchors survive conversion)

- **Validation rule IDs** grep directly: `cvc-complex-type.2.1`,
  `cos-st-restricts`, `src-resolve`, `derivation-ok-restriction`.
- **hfn function definitions**: `id="f-<name>"` (e.g. `id="f-decimalLexmap"`),
  plus `<type>-lexical-mapping` / `<type>-canonical-mapping` anchors.
- **Facets**: `id="rf-<facet>"` (e.g. `id="rf-maxInclusive"`).
- **Builtin types**: `id="<typename>"` (e.g. `id="decimal"`, `id="dateTime"`).
- **F&O functions**: `fn:<name>` greps directly (e.g. `fn:matches`).
- **EBNF productions** (xml.md, xml-names.md, xpath-datamodel.md):
  `id="NT-<name>"` (e.g. `id="NT-QName"`, `id="NT-NCName"`, `id="NT-Char"`,
  `id="NT-S"`).
- **Infoset items/properties**: `infoitem.<kind>` / `id="infoitem…"`
  (e.g. `infoitem.element`); property names grep as bracketed prose,
  e.g. `[children]`, `[namespace name]`.
- **XDM accessors**: `dm:<name>` and `id="dm-<name>"` (e.g.
  `dm:string-value`, `id="dm-typed-value"`).
