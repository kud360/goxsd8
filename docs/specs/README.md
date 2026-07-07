# Local specs — the ground truth

Agents grep these files instead of trusting memory (STYLE P1). Never
answer a spec question from recollection when the clause is one grep away.

| File | Spec |
|---|---|
| `md/xmlschema11-1.md` | XSD 1.1 Part 1: Structures |
| `md/xmlschema11-2.md` | XSD 1.1 Part 2: Datatypes (Appendix E hfn definitions are the source of truth for builtin types) |
| `md/xpath20.md` | XPath 2.0 |
| `md/xpath-functions.md` | XQuery 1.0 and XPath 2.0 Functions and Operators (F&O) — the function library and regex flavor XPath 2.0 binds to |
| `md/xsd-precisionDecimal.md` | The precisionDecimal datatype |

`html/` holds the pristine committed downloads (`go tool fetchspecs`
refreshes them); `md/` is generated from them by `go tool spec2md` via
`go generate ./...` — same input, byte-identical output. Edit neither by
hand.

## Grep conventions (anchors survive conversion)

- **Validation rule IDs** grep directly: `cvc-complex-type.2.1`,
  `cos-st-restricts`, `src-resolve`, `derivation-ok-restriction`.
- **hfn function definitions**: `id="f-<name>"` (e.g. `id="f-decimalLexmap"`),
  plus `<type>-lexical-mapping` / `<type>-canonical-mapping` anchors.
- **Facets**: `id="rf-<facet>"` (e.g. `id="rf-maxInclusive"`).
- **Builtin types**: `id="<typename>"` (e.g. `id="decimal"`, `id="dateTime"`).
- **F&O functions**: `fn:<name>` greps directly (e.g. `fn:matches`).
