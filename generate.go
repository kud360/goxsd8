package goxsd8

// Regenerate the greppable spec Markdown from the committed pristine HTML.
// Deterministic: running twice produces byte-identical output.
//
// The builtin TypeSpec table carries its own go:generate directive in
// builtin/gen.go (M1, tools/typespecgen); the rule catalog (xsderr) gains
// one when it lands (M2). This file only carries the module-wide spec
// conversion.

//go:generate go tool spec2md -in docs/specs/html -out docs/specs/md
