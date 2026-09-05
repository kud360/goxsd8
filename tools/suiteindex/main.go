// Command suiteindex takes a census of the W3C xsdtests corpus: which
// fixtures carry a given XML construct, across every file in the tree and in
// every encoding the corpus ships.
//
// A grounding round predicting ratchet movement needs the set of fixtures
// exercising a construct. Deriving that set by grep failed three landings
// running, each by a different mechanism, and the fix installed for each
// reached only that one: a UTF-16LE fixture read as interleaved NULs by a
// UTF-8 grep, a fixture outside the single directory the previous finding
// had named, and a fixture found by matching one clause's SHAPE instead of
// the construct it carries (#1239). PRINCIPLES 27 turns that shape of
// repeated, deterministic work into a tool.
//
// # What it matches
//
// A query names a construct, never a spelling of one. Element names match on
// the resolved (namespace URI, local name) pair, so `<xsd:element>`,
// `<xs:element>` and a bare `<element>` under
// `xmlns="http://www.w3.org/2001/XMLSchema"` are one construct and all three
// answer one query. Matching prefix text is the defect this tool retires:
// the corpus writes all three spellings, sometimes within one directory.
//
// Attribute names match by local name in no namespace, which is what an
// unprefixed attribute resolves to — XSD's own vocabulary attributes
// (targetNamespace, form, ref) are always written unprefixed. An element
// must carry EVERY attribute a query names to count as an occurrence, and
// the report prints their values.
//
// # Encoding
//
// Encoding is this tool's problem, not the caller's. Every fixture is read
// through [xmltree.Reader], which detects a byte-order mark and transcodes
// UTF-16 to UTF-8 before any token reaches the matcher (XML 1.0 §4.3.3).
// The suite's UTF-16 fixtures all carry a mark, so xmltree's markless-UTF-16
// gap (#361) does not reach this corpus.
//
// # What it reports
//
// Matches, sorted by path with document order preserved within each file,
// then every file the census could not read all the way through as XML: the
// ones that broke off partway, and the ones that held no element at all.
//
// The corpus ships deliberately malformed fixtures, so a parse fault is
// content rather than a failure: the matches found ahead of it are kept and
// the file is listed, because a construct BEHIND the fault is invisible here
// and the reader is the one who gets to judge that. Both groups are listed in
// full and nothing is elided — most of the second group is the suite's
// images, prose and stylesheets, but a census that counts what it could not
// read instead of naming it under-reports exactly as the greps did.
//
// # Usage
//
//	go tool suiteindex element@targetNamespace
//	go tool suiteindex attribute@targetNamespace,form
//	go tool suiteindex '{http://www.w3.org/1999/XSL/Transform}stylesheet'
//	go tool suiteindex element@targetNamespace testdata/xsdtests/ibmData
//
// The query is `local[@attr[,attr…]]`. Any name may be written in Clark
// notation (`{uri}local`) to name its namespace outright; a braceless
// element name is in the XML Schema namespace and a braceless attribute name
// is in no namespace. The second argument is the tree to walk, defaulting to
// the suite at [defaultRoot]; narrowing it is for reading one directory's
// output, never for taking the census the whole corpus answers.
//
// An absent or fixture-free root is a supported mode, not a failure: the
// submodule is absent in a fresh container (#659), so the tool says the
// corpus is not there, names the command that initializes it, and exits 0.
//
// It exits 0 for a completed census whatever it finds, and 2 for an
// operational error: an unparseable query, or a file or directory it cannot
// read.
package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsd"
)

// defaultRoot is the corpus this tool exists to census — the W3C suite
// submodule, at its fixed path in the tree.
const defaultRoot = "testdata/xsdtests"

// usage is printed for any argument the tool cannot act on.
const usage = "usage: suiteindex <local[@attr,...]> [dir]"

func main() {
	if err := run(os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "suiteindex: %v\n", err)
		os.Exit(2)
	}
}

// run drives the census end to end: parse the query, walk the corpus, render
// the report. It takes its destination and arguments so a test can drive the
// whole command without a subprocess.
func run(stdout io.Writer, args []string) error {
	q, root, err := parseArgs(args)
	if err != nil {
		return err
	}
	rep, err := census(root, q)
	if err != nil {
		return err
	}
	return printReport(stdout, rep)
}

// parseArgs splits the command line into the query and the tree to walk.
func parseArgs(args []string) (query, string, error) {
	if len(args) == 0 || len(args) > 2 {
		return query{}, "", errors.New(usage)
	}
	q, err := parseQuery(args[0])
	if err != nil {
		return query{}, "", err
	}
	if len(args) == 2 {
		return q, args[1], nil
	}
	return q, defaultRoot, nil
}

// query is one construct census: the element to look for, and the attribute
// names an occurrence must all carry.
type query struct {
	Element xsd.QName
	Attrs   []xsd.QName
}

// String renders the query in the canonical form the report echoes: every
// name in Clark notation, so the reader sees the namespace that was matched
// rather than the prefix some fixture happened to spell it with.
func (q query) String() string {
	var b strings.Builder
	b.WriteString(q.Element.String())
	for i, a := range q.Attrs {
		sep := ","
		if i == 0 {
			sep = "@"
		}
		b.WriteString(sep)
		b.WriteString(a.String())
	}
	return b.String()
}

// parseQuery parses `local[@attr[,attr…]]`, where any name may carry a Clark
// `{uri}` wrapper. A braceless element name is in the XML Schema namespace —
// the vocabulary every schema fixture in this corpus is written in — and a
// braceless attribute name is in no namespace, which is what an unprefixed
// attribute resolves to.
func parseQuery(s string) (query, error) {
	elem, rest, err := splitName(s, xsd.XMLSchemaNS)
	if err != nil {
		return query{}, fmt.Errorf("query %q: %w", s, err)
	}
	q := query{Element: elem}
	if rest == "" {
		return q, nil
	}
	if !strings.HasPrefix(rest, "@") {
		return query{}, fmt.Errorf("query %q: expected \"@\" before the attribute list, found %q", s, rest)
	}
	rest = rest[1:]
	for {
		attr, more, err := splitName(rest, "")
		if err != nil {
			return query{}, fmt.Errorf("query %q: %w", s, err)
		}
		q.Attrs = append(q.Attrs, attr)
		if more == "" {
			return q, nil
		}
		if !strings.HasPrefix(more, ",") {
			return query{}, fmt.Errorf("query %q: expected \",\" between attribute names, found %q", s, more)
		}
		rest = more[1:]
	}
}

// splitName consumes one name from the head of s — an optional Clark
// `{uri}` wrapper, then a local part — and returns it with whatever follows.
// The URI is taken as everything up to the closing brace, so a namespace
// containing "@" or "," survives the separators around it.
func splitName(s, defaultSpace string) (n xsd.QName, rest string, err error) {
	space := defaultSpace
	if strings.HasPrefix(s, "{") {
		end := strings.Index(s, "}")
		if end < 0 {
			return xsd.QName{}, "", fmt.Errorf("unterminated \"{\" in %q", s)
		}
		space = s[1:end]
		s = s[end+1:]
	}
	local := s
	if i := strings.IndexAny(s, "@,"); i >= 0 {
		local, rest = s[:i], s[i:]
	}
	if local == "" {
		return xsd.QName{}, "", errors.New("empty local name")
	}
	if strings.ContainsAny(local, "{}") {
		return xsd.QName{}, "", fmt.Errorf("local name %q contains a brace: write a namespace as a leading {uri}", local)
	}
	return xsd.QName{Space: space, Local: local}, rest, nil
}

// hit is one occurrence of the queried construct.
type hit struct {
	// File is the fixture's slash-separated path relative to the census root.
	File string
	Line int
	Col  int
	// Values holds the queried attributes' values, in query order.
	Values []string
}

// fileNote is one file the census could not read all the way through as XML,
// with the fault that stopped it — nil for a file that ended cleanly without
// ever holding an element.
type fileNote struct {
	File string
	Err  error
}

// report is one completed census. Only facts that cannot be recomputed are
// stored: the counts of fully read files and of matched fixtures are derived
// where they are printed (STYLE D3).
//
// Every walked file lands in exactly one of three places — read to the end,
// [report.Partial], or [report.NoElement] — and the two latter groups are
// listed rather than counted. A census that silently drops the files it could
// not read is the defect this tool exists to retire, one layer down.
type report struct {
	Query query
	Root  string
	// RootPresent is whether the root directory exists at all. It is not
	// derivable from Walked, which is 0 for an empty directory too, and the
	// two cases want different advice (#659).
	RootPresent bool
	// Walked is how many regular files the census opened.
	Walked int
	Hits   []hit
	// Partial holds files whose read broke off after at least one start tag,
	// so a construct behind the fault is invisible to the census.
	Partial []fileNote
	// NoElement holds files that yielded no start tag at all: the corpus's
	// images, stylesheets and prose, plus any fixture that faulted ahead of
	// its root element.
	NoElement []fileNote
}

// census walks root and matches q against every file under it. Files are
// scanned in path-sorted order and each file's hits stay in document order,
// so the whole report is deterministic (STYLE D1).
func census(root string, q query) (report, error) {
	paths, present, err := walkFiles(root)
	if err != nil {
		return report{}, err
	}
	rep := report{Query: q, Root: root, RootPresent: present, Walked: len(paths)}
	for _, rel := range paths {
		scan, err := scanFile(filepath.Join(root, filepath.FromSlash(rel)), rel, q)
		if err != nil {
			return report{}, err
		}
		rep.Hits = append(rep.Hits, scan.Hits...)
		if scan.Elems == 0 {
			rep.NoElement = append(rep.NoElement, fileNote{File: rel, Err: scan.Err})
			continue
		}
		if scan.Err != nil {
			rep.Partial = append(rep.Partial, fileNote{File: rel, Err: scan.Err})
		}
	}
	return rep, nil
}

// walkFiles lists every regular file under root as a slash path relative to
// it, sorted. present is false when root does not exist — the fresh-container
// case, which is a mode rather than an error.
func walkFiles(root string) (paths []string, present bool, err error) {
	info, err := os.Stat(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading census root %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, false, fmt.Errorf("census root %s is not a directory", root)
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Dot entries are the submodule's git plumbing, never fixtures.
		if path != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relating %s to %s: %w", path, root, err)
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, true, fmt.Errorf("walking %s: %w", root, err)
	}
	sort.Strings(paths)
	return paths, true, nil
}

// fixtureScan is what one fixture yielded: its matches, how many start tags
// it held at all, and the fault that ended the read early, if any. Err is a
// property of the fixture's own bytes — a malformed document is content in
// this corpus, not an operational failure — which is why it is a field here
// and not a returned error.
type fixtureScan struct {
	Hits  []hit
	Elems int
	Err   error
}

// scanFile opens path and scans it as uri. It is the only filesystem-touching
// step of the match pipeline; [scanFixture] itself reads a stream. Its error
// is operational — a file that could not be opened or closed — and aborts the
// census rather than being counted as an unreadable fixture.
func scanFile(path, uri string, q query) (scan fixtureScan, err error) {
	f, err := os.Open(path)
	if err != nil {
		return fixtureScan{}, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", path, cerr)
		}
	}()
	return scanFixture(uri, f, q), nil
}

// scanFixture reports every start tag in r matching q, in document order.
// [fixtureScan.Elems] counts the start tags seen at all, which is how the
// caller tells a fixture that broke off partway from a file that was never
// XML.
//
// A read fault comes back WITH the hits found ahead of it rather than in
// place of them: the suite ships deliberately malformed fixtures, and the
// constructs before the fault are evidence the census must keep.
func scanFixture(uri string, r io.Reader, q query) fixtureScan {
	var scan fixtureScan
	rd := xmltree.NewReader(uri, r)
	for {
		node, err := rd.Token()
		if errors.Is(err, io.EOF) {
			return scan
		}
		if err != nil {
			// Kept, not wrapped: an xmltree error already names the document
			// and the position of the fault (STYLE E3), and the report prints
			// the path beside it.
			scan.Err = err
			return scan
		}
		start, ok := node.(*xmltree.StartElement)
		if !ok {
			continue
		}
		scan.Elems++
		values, ok := match(start, q)
		if !ok {
			continue
		}
		loc := start.Loc()
		scan.Hits = append(scan.Hits, hit{File: uri, Line: loc.Line, Col: loc.Col, Values: values})
	}
}

// match reports whether start is an occurrence of q's construct — its
// resolved name equals q's and it carries every attribute q names — and
// those attributes' values in query order.
func match(start *xmltree.StartElement, q query) ([]string, bool) {
	if !sameName(q.Element, start.Name()) {
		return nil, false
	}
	values := make([]string, len(q.Attrs))
	for i, want := range q.Attrs {
		v, ok := attrValue(start, want)
		if !ok {
			return nil, false
		}
		values[i] = v
	}
	return values, true
}

// attrValue returns the value of start's want attribute. The attribute list
// is a document-ordered slice, so the first match is the only one a
// well-formed document can have.
func attrValue(start *xmltree.StartElement, want xsd.QName) (string, bool) {
	for _, a := range start.Attributes() {
		if sameName(want, a.Name()) {
			return a.Value(), true
		}
	}
	return "", false
}

// sameName compares a queried name with a name the reader resolved. The
// comparison is on namespace URI and local part, never on the prefix the
// document spelled — that equivalence is the tool's whole point.
func sameName(want xsd.QName, got xmltree.Name) bool {
	return want.Space == got.Space() && want.Local == got.Local()
}

// printReport renders rep in the fixed layout the package doc describes.
// Formatting lives here, apart from [census], so the matching logic stays
// testable without string-matching rendered text.
func printReport(dst io.Writer, rep report) error {
	w := &latchWriter{w: dst}
	printReportTo(w, rep)
	if w.err != nil {
		return fmt.Errorf("writing report: %w", w.err)
	}
	return nil
}

// printReportTo does the rendering itself. Its writes go to a [latchWriter],
// which is why they are not individually checked.
func printReportTo(w io.Writer, rep report) {
	if !rep.RootPresent {
		printAbsent(w, rep.Root, "does not exist")
		return
	}
	if rep.Walked == 0 {
		printAbsent(w, rep.Root, "holds no files")
		return
	}

	_, _ = fmt.Fprintf(w, "suiteindex: %d occurrence(s) of %s in %d fixture(s) under %s\n",
		len(rep.Hits), rep.Query, countFiles(rep.Hits), rep.Root)
	_, _ = fmt.Fprintf(w, "  walked %d file(s): %d read to the end, %d read only partly, %d with no XML element\n",
		rep.Walked, rep.Walked-len(rep.NoElement)-len(rep.Partial), len(rep.Partial), len(rep.NoElement))

	_, _ = fmt.Fprintln(w, "\n=== Matches (path order, document order within a file) ===")
	if len(rep.Hits) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, h := range rep.Hits {
		_, _ = fmt.Fprintf(w, "  %s:%d:%d%s\n", h.File, h.Line, h.Col, renderValues(rep.Query.Attrs, h.Values))
	}

	printNotes(w, "Read only partly: a match behind the fault is invisible to this census", rep.Partial)
	printNotes(w, "No XML element: the corpus's images, prose and stylesheets, and any"+
		" fixture that faulted ahead of its root", rep.NoElement)
}

// printAbsent renders the corpus-absent mode: why there was nothing to
// census, and the command that fixes it. The mode exits 0 — a fresh container
// has no suite submodule (#659), which is a state to report, not a failure.
func printAbsent(w io.Writer, root, reason string) {
	_, _ = fmt.Fprintf(w, "suiteindex: %s %s — corpus-absent mode, nothing to census.\n", root, reason)
	_, _ = fmt.Fprintf(w, "Initialize the suite with `git submodule update --init %s` (#659).\n", defaultRoot)
}

// printNotes renders one group of unread files under its heading. Both groups
// are listed in full: what a census could not read is the reader's to judge,
// and counting it instead is how a corpus survey under-reports (#1239).
func printNotes(w io.Writer, heading string, notes []fileNote) {
	_, _ = fmt.Fprintf(w, "\n=== %s ===\n", heading)
	if len(notes) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, n := range notes {
		if n.Err == nil {
			_, _ = fmt.Fprintf(w, "  %s — read to the end, no element in it\n", n.File)
			continue
		}
		_, _ = fmt.Fprintf(w, "  %s — %v\n", n.File, n.Err)
	}
}

// countFiles reports how many distinct fixtures the hits fall in. Hits
// arrive path-sorted, so distinctness is a neighbour comparison and needs no
// set.
func countFiles(hits []hit) int {
	n := 0
	prev := ""
	for i, h := range hits {
		if i == 0 || h.File != prev {
			n++
		}
		prev = h.File
	}
	return n
}

// renderValues formats a hit's attribute values against the names that
// selected them, in query order.
func renderValues(attrs []xsd.QName, values []string) string {
	var b strings.Builder
	for i, a := range attrs {
		if i >= len(values) {
			break
		}
		fmt.Fprintf(&b, " %s=%q", a.Local, values[i])
	}
	return b.String()
}

// latchWriter is an [io.Writer] that remembers its first failure and drops
// every write after it. A report is a long run of small writes to one
// destination, so checking each in place would bury the layout in error
// handling; the caller checks err once, at the end.
type latchWriter struct {
	w   io.Writer
	err error
}

func (l *latchWriter) Write(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	n, err := l.w.Write(p)
	l.err = err
	return n, err
}
