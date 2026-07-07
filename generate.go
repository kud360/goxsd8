package goxsd8

// Regenerate the greppable spec Markdown from the committed pristine HTML.
// Deterministic: running twice produces byte-identical output.
//
// The rule catalog (xsderr) and builtin TypeSpec table (builtin) gain
// their own go:generate directives when those packages land (M1/M2);
// this file only carries the module-wide spec conversion.

//go:generate go tool spec2md -in docs/specs/html -out docs/specs/md
