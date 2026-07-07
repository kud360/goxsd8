// Command spec2md converts W3C specification HTML into Markdown suitable
// for grepping and agent consumption.
//
// Design goals:
//   - Preserve section structure (headings), code blocks, tables, and
//     definition lists.
//   - Preserve every anchor (element id= or <a name=>) as an inline
//     <a id="..."></a>, so spec cross-references and rule IDs
//     (cvc-*, cos-*, src-*, hfn function names) stay greppable and
//     linkable in the Markdown.
//   - Deterministic output: same input, byte-identical output.
//
// Usage:
//
//	spec2md -in docs/specs/html -out docs/specs/md
//	spec2md -in one-file.html -out outdir
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func main() {
	in := flag.String("in", "", "input HTML file or directory")
	out := flag.String("out", "", "output directory for .md files")
	flag.Parse()
	if *in == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "spec2md:", err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	info, err := os.Stat(in)
	if err != nil {
		return fmt.Errorf("stat input %s: %w", in, err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", out, err)
	}
	if !info.IsDir() {
		return convertFile(in, out)
	}
	entries, err := os.ReadDir(in)
	if err != nil {
		return fmt.Errorf("read input dir %s: %w", in, err)
	}
	var errs []error
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		if err := convertFile(filepath.Join(in, e.Name()), out); err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrs(errs)
}

func joinErrs(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return fmt.Errorf("%d file(s) failed:\n%s", len(errs), strings.Join(msgs, "\n"))
}

func convertFile(path, outDir string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	// Read-only close; its error carries no information (STYLE S3).
	defer func() { _ = f.Close() }()

	doc, err := html.Parse(f)
	if err != nil {
		return fmt.Errorf("parse HTML %s: %w", path, err)
	}

	r := &renderer{}
	r.renderChildren(doc)
	md := tidy(r.b.String())

	base := strings.TrimSuffix(filepath.Base(path), ".html") + ".md"
	outPath := filepath.Join(outDir, base)
	if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	fmt.Printf("%s -> %s (%d bytes)\n", path, outPath, len(md))
	return nil
}

// renderer walks the HTML tree in document order emitting Markdown.
type renderer struct {
	b         strings.Builder
	listDepth int
	ordinals  []int // per-depth <ol> counters; 0 means unordered
	inTable   bool
}

func (r *renderer) renderChildren(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.render(c)
	}
}

func (r *renderer) render(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		r.text(n.Data)
		return
	case html.ElementNode:
		// fall through to element handling below
	default:
		return
	}

	r.emitAnchor(n)

	switch n.DataAtom {
	case atom.Head, atom.Script, atom.Style, atom.Title:
		return
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		level := int(n.Data[1] - '0')
		r.block()
		r.b.WriteString(strings.Repeat("#", level) + " ")
		r.b.WriteString(inlineText(n))
		r.b.WriteString("\n\n")
	case atom.P:
		r.block()
		r.renderChildren(n)
		r.b.WriteString("\n\n")
	case atom.Pre:
		r.block()
		r.b.WriteString("```\n")
		r.b.WriteString(strings.Trim(rawText(n), "\n"))
		r.b.WriteString("\n```\n\n")
	case atom.Ul, atom.Ol:
		start := 0
		if n.DataAtom == atom.Ol {
			start = 1
		}
		r.listDepth++
		r.ordinals = append(r.ordinals, start)
		r.renderChildren(n)
		r.ordinals = r.ordinals[:len(r.ordinals)-1]
		r.listDepth--
		if r.listDepth == 0 {
			r.block()
		}
	case atom.Li:
		r.block()
		indent := strings.Repeat("  ", max(r.listDepth-1, 0))
		marker := "-"
		if len(r.ordinals) > 0 && r.ordinals[len(r.ordinals)-1] > 0 {
			marker = fmt.Sprintf("%d.", r.ordinals[len(r.ordinals)-1])
			r.ordinals[len(r.ordinals)-1]++
		}
		r.b.WriteString(indent + marker + " ")
		r.renderChildren(n)
		r.b.WriteString("\n")
	case atom.Dl:
		r.block()
		r.renderChildren(n)
		r.b.WriteString("\n")
	case atom.Dt:
		r.block()
		r.b.WriteString("**" + inlineText(n) + "**\n")
	case atom.Dd:
		r.b.WriteString(": ")
		r.renderChildren(n)
		r.b.WriteString("\n\n")
	case atom.Table:
		r.renderTable(n)
	case atom.Blockquote:
		r.block()
		inner := &renderer{}
		inner.renderChildren(n)
		for line := range strings.SplitSeq(strings.TrimSpace(tidy(inner.b.String())), "\n") {
			r.b.WriteString("> " + line + "\n")
		}
		r.b.WriteString("\n")
	case atom.Br:
		r.b.WriteString("\n")
	case atom.Hr:
		r.block()
		r.b.WriteString("---\n\n")
	case atom.A:
		r.renderLink(n)
	case atom.Em, atom.I, atom.Var:
		r.b.WriteString("*")
		r.renderChildren(n)
		r.b.WriteString("*")
	case atom.Strong, atom.B:
		r.b.WriteString("**")
		r.renderChildren(n)
		r.b.WriteString("**")
	case atom.Code, atom.Tt, atom.Kbd, atom.Samp:
		txt := rawText(n)
		if strings.Contains(txt, "`") {
			r.text(txt)
			return
		}
		r.b.WriteString("`" + collapse(txt) + "`")
	default:
		r.renderChildren(n)
	}
}

// emitAnchor preserves id= / <a name=> so spec cross-references stay
// greppable in the Markdown.
func (r *renderer) emitAnchor(n *html.Node) {
	id := attr(n, "id")
	if id == "" && n.DataAtom == atom.A {
		id = attr(n, "name")
	}
	if id == "" {
		return
	}
	r.b.WriteString(`<a id="` + id + `"></a>`)
}

func (r *renderer) renderLink(n *html.Node) {
	href := attr(n, "href")
	label := collapse(inlineText(n))
	if href == "" || label == "" {
		r.renderChildren(n)
		return
	}
	if r.inTable {
		// Keep table cells simple; brackets survive, pipes are escaped later.
		r.text(label)
		return
	}
	r.b.WriteString("[" + label + "](" + href + ")")
}

func (r *renderer) renderTable(n *html.Node) {
	r.block()
	var rows [][]string
	var walk func(*html.Node)
	walk = func(m *html.Node) {
		if m.Type == html.ElementNode && m.DataAtom == atom.Tr {
			var cells []string
			for c := m.FirstChild; c != nil; c = c.NextSibling {
				if c.Type != html.ElementNode {
					continue
				}
				if c.DataAtom != atom.Td && c.DataAtom != atom.Th {
					continue
				}
				cell := &renderer{inTable: true}
				cell.renderChildren(c)
				txt := collapse(strings.ReplaceAll(cell.b.String(), "\n", " "))
				cells = append(cells, strings.ReplaceAll(txt, "|", `\|`))
			}
			rows = append(rows, cells)
			return
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	if len(rows) == 0 {
		return
	}
	width := 0
	for _, row := range rows {
		width = max(width, len(row))
	}
	if width == 0 {
		return
	}
	for i, row := range rows {
		for len(row) < width {
			row = append(row, "")
		}
		r.b.WriteString("| " + strings.Join(row, " | ") + " |\n")
		if i == 0 {
			r.b.WriteString("|" + strings.Repeat(" --- |", width) + "\n")
		}
	}
	r.b.WriteString("\n")
}

// text emits inline text with whitespace collapsed.
func (r *renderer) text(s string) {
	c := collapse(s)
	if c == "" {
		return
	}
	// Preserve a single boundary space when the source had one.
	if strings.TrimLeft(s, " \t\n\r") != s && !endsOpen(r.b.String()) {
		r.b.WriteString(" ")
	}
	r.b.WriteString(strings.TrimLeft(c, " "))
	if strings.TrimRight(s, " \t\n\r") != s {
		r.b.WriteString(" ")
	}
}

// endsOpen reports whether the buffer already ends in whitespace or an
// opening construct, so no boundary space is needed.
func endsOpen(s string) bool {
	if s == "" {
		return true
	}
	switch s[len(s)-1] {
	case ' ', '\n', '(', '[', '*', '`', '#':
		return true
	}
	return false
}

// block ensures we are at the start of a fresh line before block output.
func (r *renderer) block() {
	s := r.b.String()
	if s == "" || strings.HasSuffix(s, "\n") {
		return
	}
	r.b.WriteString("\n")
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// rawText returns the concatenated text content with original whitespace.
func rawText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(m *html.Node) {
		if m.Type == html.TextNode {
			b.WriteString(m.Data)
			return
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// inlineText is rawText with anchors preserved and whitespace collapsed.
func inlineText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(m *html.Node) {
		if m.Type == html.TextNode {
			b.WriteString(m.Data)
			return
		}
		if m.Type == html.ElementNode {
			id := attr(m, "id")
			if id == "" && m.DataAtom == atom.A {
				id = attr(m, "name")
			}
			if id != "" {
				b.WriteString(`<a id="` + id + `"></a>`)
			}
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return collapse(b.String())
}

var spaceRun = regexp.MustCompile(`[ \t\r\n]+`)

func collapse(s string) string {
	return strings.TrimSpace(spaceRun.ReplaceAllString(s, " "))
}

var blankRun = regexp.MustCompile(`\n{3,}`)

// tidy normalizes runs of blank lines and trailing space.
func tidy(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	out := blankRun.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimLeft(out, "\n")
}
