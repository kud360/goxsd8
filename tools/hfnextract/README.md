# hfnextract

Extracts tables from the local spec Markdown (`docs/specs/md/`) as JSON —
the first half of the builtin-type bootstrap pipeline.

```sh
go tool hfnextract -file docs/specs/md/xmlschema11-2.md [-header <text>] [-index <n>]
```

Matches GFM pipe tables by header text (partial match) or position and
emits them as indented JSON on stdout (`{"header": [...], "rows": [...]}`),
logging to stderr via slog.

## Builtin datatype extraction

```sh
go tool hfnextract -builtins
```

The `-builtins` mode is the spec-parsing half of the M1 pipeline. It reads
`docs/specs/md/xmlschema11-2.md` and `docs/specs/md/xsd-precisionDecimal.md`
and emits the 49 builtin datatypes as JSON (name, base, variety, item type,
fundamental facets, applicable constraining facets + defaults). The parsing
lives in the importable `builtins` subpackage (`ParseBuiltins` via
`builtins.Parse`); the fundamental facets are cross-checked against
Appendix F.1, and any inconsistency fails loudly. `tools/typespecgen`
consumes `builtins.Parse` directly to emit `builtin/gen_typespec.go` — the
JSON mode is for inspection.

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
