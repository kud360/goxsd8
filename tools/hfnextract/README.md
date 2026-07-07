# hfnextract

Extracts tables from the local spec Markdown (`docs/specs/md/`) as JSON —
the first half of the builtin-type bootstrap pipeline.

```sh
go tool hfnextract -file docs/specs/md/xmlschema11-2.md [-header <text>] [-index <n>]
```

Matches GFM pipe tables by header text (partial match) or position and
emits them as indented JSON on stdout (`{"header": [...], "rows": [...]}`),
logging to stderr via slog.

## The pipeline this feeds (milestone M1)

Builtin type definitions are **generated, never hand-typed**
(PRINCIPLES 26). The Datatypes spec defines each builtin normatively via
Appendix E function definitions ("hfn", anchors `f-*`,
`<type>-lexical-mapping` / `-canonical-mapping`) and per-type property
tables (anchors `<typename>`, facets at `rf-*`); `xsd-precisionDecimal.md`
adds precisionDecimal.

The M1 generator consumes these (via this tool's table extraction) and
emits `builtin/gen_typespec.go` — the backend-neutral data table (name,
base, variety, fundamental facets, applicable facets + defaults) for all
49 builtins **including precisionDecimal** — wired to `go generate`.

Acceptance bar: byte-identical regeneration, zero hand-typed rows, and
the JSON→Go step kept separate from spec parsing (this tool stays a
generic table extractor).
