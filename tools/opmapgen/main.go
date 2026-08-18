// Command opmapgen emits xpath/gen_opmap.go: the comparison rows of
// xpath20.md Appendix B.2's operator mapping table, which is what decides
// whether an operator may be applied to a given pair of operand types at all
// — "If the types of the operands, after evaluation, are not a valid
// combination for the given operator, according to the rules in B.2 Operator
// Mapping, a type error is raised [err:XPTY0004]" (§3.5.1).
//
// It reads the table from the local spec Markdown rather than carrying a
// transcription of it (PRINCIPLES 26), and expands the two prose macros B.2's
// type cells use — numeric and Gregorian — from their own Definition spans in
// the same section. Output is gofmt'd and deterministic, so running it twice
// is byte-identical (STYLE D1).
//
// Rows for the operators outside the six comparisons — the arithmetic, node
// and sequence operators — are read and skipped: they have differently shaped
// rows, their own XPTY0004 clause (§3.4), and no production in §3.12.6's
// required subset. Anything else the table holds is an error rather than a
// skip, so a spec re-render that changes the table's shape stops the build
// instead of silently emitting a short table.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"strings"
)

func main() {
	xpathPath := flag.String("xpath", "docs/specs/md/xpath20.md", "XPath 2.0 spec Markdown")
	outPath := flag.String("out", "xpath/gen_opmap.go", "output Go file")
	flag.Parse()

	if err := run(*xpathPath, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "opmapgen: %v\n", err)
		os.Exit(1)
	}
}

func run(xpathPath, outPath string) error {
	text, err := os.ReadFile(xpathPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", xpathPath, err)
	}
	rows, err := parse(strings.Split(string(text), "\n"))
	if err != nil {
		return fmt.Errorf("%s: %w", xpathPath, err)
	}
	src, err := emit(rows)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, src, 0o644)
}

// row is one emitted table row: an operator and the local names of the two
// operand types it admits, with every macro already expanded.
type row struct {
	op   string
	a, b string
}

// comparisonOps maps B.2's operator tokens for the six value comparisons to
// the ctaComparator constants the xpath package spells the same operators
// with. §3.12.6's required subset writes them as the GENERAL comparisons
// ('=', '!=', '<', '<=', '>', '>='), and B.2's table as the value comparisons
// (eq, ne, lt, le, gt, ge), but one table governs both: §3.5.3's own type-error
// clause for the general comparisons points at the same B.2 rules §3.5.1 does.
var comparisonOps = map[string]string{
	"eq": "ctaEqual",
	"ne": "ctaNotEqual",
	"lt": "ctaLess",
	"le": "ctaLessEqual",
	"gt": "ctaGreater",
	"ge": "ctaGreaterEqual",
}

// tableHeader is the header row of B.2's operator mapping table, which is
// what tells it apart from the illustrative four-column table printed above it
// in the same section.
const tableHeader = "| Operator | Type(A) | Type(B) | Function | Result type |"

// mappingHeading is the B.2 section heading, from which the table and both
// macro definitions are read. The anchor is matched rather than the section
// number so a renumbered appendix does not silently select a different table.
const mappingHeading = `<a id="mapping"></a>`

// macro is one of B.2's prose type macros: the types it names, and whether
// B.2 restricts a binary operator over it to operands of the SAME type.
type macro struct {
	types        []string
	sameTypeOnly bool
}

// macroAnchors are the two Definition spans B.2's type cells refer to, by the
// anchor each is defined under. A type cell naming anything else is an error
// (parse), so a third macro added to the table stops the build here rather
// than silently dropping its rows.
var macroAnchors = []string{`<a id="dt-numeric"></a>`, `<a id="dt-gregorian"></a>`}

// sameTypeSentence is the restriction B.2 states in prose beside the Gregorian
// definition: "For binary operators that accept two Gregorian-type operands,
// both operands must have the same type". Where it holds, the macro expands to
// its diagonal instead of to every pair.
const sameTypeSentence = "both operands must have the same type"

var (
	// definitionSpan captures one bracketed Definition: its bolded term and
	// the text through the closing "]".
	definitionSpan = regexp.MustCompile(`\[<a id="dt-[a-z]+"></a>Definition: [^*]*\*\*([A-Za-z]+)\*\*(.*?)\.\]`)
	// backtickedType captures one `xs:name` mention inside a Definition span.
	backtickedType = regexp.MustCompile("`xs:([A-Za-z0-9]+)`")
	// operatorCell captures the operator token of a row's first cell, whose
	// every row in this table is shaped "A <token> B".
	operatorCell = regexp.MustCompile(`^A (\S+) B$`)
)

// parse reads B.2's comparison rows from the spec Markdown lines, in table
// order and with both macros expanded.
func parse(lines []string) ([]row, error) {
	start := indexOf(lines, mappingHeading)
	if start < 0 {
		return nil, fmt.Errorf("no B.2 section heading (%s)", mappingHeading)
	}
	macros, err := readMacros(lines[start:])
	if err != nil {
		return nil, err
	}
	header := indexOf(lines[start:], tableHeader)
	if header < 0 {
		return nil, fmt.Errorf("no operator mapping table under B.2 (%s)", tableHeader)
	}
	// The line after the header is Markdown's alignment separator, and the
	// table runs from there to the first line that is not a row.
	body := lines[start+header+2:]
	var rows []row
	for _, line := range body {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			break
		}
		expanded, err := expand(line, macros)
		if err != nil {
			return nil, err
		}
		rows = append(rows, expanded...)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("operator mapping table holds no comparison row")
	}
	return rows, nil
}

// readMacros reads the two Definition spans B.2's type cells name, keyed by
// the bolded term each defines — which is the spelling the cells use.
func readMacros(section []string) (map[string]macro, error) {
	found := make(map[string]macro)
	for _, anchor := range macroAnchors {
		at := indexOf(section, anchor)
		if at < 0 {
			return nil, fmt.Errorf("no macro definition for anchor %s", anchor)
		}
		name, m, err := readMacro(section[at])
		if err != nil {
			return nil, fmt.Errorf("macro %s: %w", anchor, err)
		}
		found[name] = m
	}
	return found, nil
}

// readMacro reads one macro from the paragraph defining it: the types named
// inside its Definition span, and the same-type restriction from the prose
// around it.
func readMacro(paragraph string) (string, macro, error) {
	span := definitionSpan.FindStringSubmatchIndex(paragraph)
	if span == nil {
		return "", macro{}, fmt.Errorf("no Definition span")
	}
	name := paragraph[span[2]:span[3]]
	var types []string
	for _, m := range backtickedType.FindAllStringSubmatch(paragraph[span[0]:span[1]], -1) {
		types = append(types, m[1])
	}
	if len(types) == 0 {
		return "", macro{}, fmt.Errorf("definition span names no type")
	}
	return name, macro{types: types, sameTypeOnly: strings.Contains(paragraph, sameTypeSentence)}, nil
}

// expand turns one table line into the rows it contributes, which is none for
// an operator outside the six comparisons and one per admitted type pair
// otherwise.
func expand(line string, macros map[string]macro) ([]row, error) {
	cells := splitRow(line)
	if len(cells) != 5 {
		return nil, fmt.Errorf("row %q: got %d cells, want 5", line, len(cells))
	}
	token := operatorCell.FindStringSubmatch(cells[0])
	if token == nil {
		return nil, fmt.Errorf("row %q: operator cell %q is not shaped \"A <operator> B\"", line, cells[0])
	}
	op, compares := comparisonOps[token[1]]
	if !compares {
		return nil, nil
	}
	left, leftMacro, err := operandTypes(cells[1], macros)
	if err != nil {
		return nil, fmt.Errorf("row %q: Type(A): %w", line, err)
	}
	right, _, err := operandTypes(cells[2], macros)
	if err != nil {
		return nil, fmt.Errorf("row %q: Type(B): %w", line, err)
	}
	if leftMacro.sameTypeOnly && cells[1] == cells[2] {
		return diagonal(op, left), nil
	}
	return pairs(op, left, right), nil
}

// operandTypes resolves one Type(A)/Type(B) cell to the types it admits: the
// one type an `xs:name` cell names, or the several a macro cell expands to.
func operandTypes(cell string, macros map[string]macro) ([]string, macro, error) {
	if m, isMacro := macros[cell]; isMacro {
		return m.types, m, nil
	}
	local, isXSD := strings.CutPrefix(cell, "xs:")
	if !isXSD {
		return nil, macro{}, fmt.Errorf("%q names neither a macro nor a type in the XSD namespace", cell)
	}
	return []string{local}, macro{}, nil
}

// pairs is the cross product of two operand-type sets, which is what B.2's
// promotion rule makes of a row naming a macro on both sides: "that operator
// can be applied to an operand of type AT if type AT can be converted to type
// ET by a combination of type promotion and subtype substitution".
func pairs(op string, left, right []string) []row {
	var out []row
	for _, a := range left {
		for _, b := range right {
			out = append(out, row{op: op, a: a, b: b})
		}
	}
	return out
}

// diagonal is the same-type restriction's expansion: one row per type, both
// operands of it.
func diagonal(op string, types []string) []row {
	var out []row
	for _, t := range types {
		out = append(out, row{op: op, a: t, b: t})
	}
	return out
}

// splitRow splits one Markdown table line into its trimmed cells, honoring the
// backslash escape a cell uses to carry a literal pipe (B.2's `A \| B` row).
func splitRow(line string) []string {
	var cells []string
	var cell strings.Builder
	escaped := false
	for _, r := range strings.TrimSpace(line) {
		if escaped {
			cell.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '|' {
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
			continue
		}
		cell.WriteRune(r)
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	// A well-formed row opens and closes with a pipe, so the split yields an
	// empty cell at each end.
	if len(cells) < 2 {
		return nil
	}
	return cells[1 : len(cells)-1]
}

// indexOf reports the first line holding needle, or -1.
func indexOf(lines []string, needle string) int {
	for i, line := range lines {
		if strings.Contains(line, needle) {
			return i
		}
	}
	return -1
}

func emit(rows []row) ([]byte, error) {
	var b strings.Builder
	b.WriteString("// Code generated by tools/opmapgen; DO NOT EDIT.\n\n")
	b.WriteString("package xpath\n\n")
	b.WriteString("// ctaB2Comparisons is xpath20.md Appendix B.2's operator mapping table,\n")
	b.WriteString("// restricted to the six comparison operators and with its two prose macros\n")
	b.WriteString("// expanded: numeric to the four numeric types in every combination its\n")
	b.WriteString("// promotion rule admits, Gregorian to the five Gregorian types paired only\n")
	b.WriteString("// with themselves. Rows are in B.2's own order.\n")
	b.WriteString("//\n")
	b.WriteString("// A pair of operand types is a valid combination for an operator exactly\n")
	b.WriteString("// where a row holds it (§3.5.1, err:XPTY0004); ctaB2Admits is the lookup.\n")
	b.WriteString("var ctaB2Comparisons = []ctaB2Row{\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "\t{op: %s, a: %q, b: %q},\n", r.op, r.a, r.b)
	}
	b.WriteString("}\n")

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w", err)
	}
	return src, nil
}
