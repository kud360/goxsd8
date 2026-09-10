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
// (targetNamespace, form, ref) are always written unprefixed. A namespace
// declaration (xmlns, xmlns:p) is not an attribute and never matches
// ([xmltree.StartElement.Attributes]).
//
// A local name may be written `*`, which stands for every local name in its
// namespace: `*` is every element in the XML Schema namespace and `@*` is
// every attribute in no namespace. The wildcard is what makes a census
// UNANCHORED — an axis over every element, or over every attribute name,
// rather than a lookup of one construct — and it needs no escape, since `*`
// is not an NCName and no document can spell a name that collides with it.
//
// Attribute names are joined by `,` (an element must carry EVERY one to count
// as an occurrence) or by `|` (ANY one of them is enough). One query uses one
// join, never both, and `@*` stands alone: it already names the whole
// attribute axis. The report prints the matched attributes' values, in query
// order — in document order for `@*`, which fixes no order of its own.
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
// A query that names the attributes it looks for reports its MATCHES, sorted
// by path with document order preserved within each file. A query that
// wildcards them (`@*`) reports the attribute-NAME AXIS instead: the
// (element, attribute) pairs, then the fixtures carrying each pair. Both end
// with every file the census could not read all the way through as XML: the
// ones that broke off partway, and the ones that held no element at all.
//
// The axis is grouped because ungrouped it is not a report anyone reads:
// `*@*` over the suite is 172,565 attribute occurrences, which one line each
// renders unreadable, and 185 pairs, which a table renders in a screenful
// (#1391). Its first section is that table — pair, occurrence count, fixture
// count, in name order — and its second lists each pair's fixtures in path
// order, deduplicated, so a pair's own fixtures are read off the section
// rather than by re-querying it.
//
// A match line carries its position, then the matched element's name where
// the query left it open with a wildcard and never where the query already
// fixed it, then `parent=` naming the element it is a direct child of —
// `(none)` for a document element — then `children=[…]` listing the elements
// directly under it, then the matched attributes' values. The parent is what
// separates a local occurrence from a top-level one, which no query over an
// element's OWN attributes can express (#1282). It is the parent the
// document actually spells, never one corrected against the grammar: an
// `xs:element` under `xs:redefine` is reported there, wrong though that
// document is, and its children are reported the same way.
//
// The child list is in document order and keeps repeats, because a
// content-model census turns on multiplicity: "no child outside
// `annotation | simpleType | complexType`" survives deduplication and "nor a
// second `simpleType` beside a first" does not (#1304). `children=[]` is an
// occurrence with no element child at all. A list whose last entry is
// `(unclosed)` is one whose read broke off before the match's own end tag, so
// the names ahead of the marker are a prefix rather than the whole list — a
// fact about that MATCH and not about its file, since a fixture can fault at
// its tail with every match in it already closed and every child list
// complete.
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
//	go tool suiteindex '*@*'
//	go tool suiteindex '*@mixed|abstract|nillable'
//	go tool suiteindex element@targetNamespace testdata/xsdtests/ibmData
//
// The query is `local[@attr[,attr…]]` with `|` in place of `,` for the ANY
// join, and any local part may be `*`. Any name may be written in Clark
// notation (`{uri}local`) to name its namespace outright; a braceless element
// name is in the XML Schema namespace and a braceless attribute name is in no
// namespace. The second argument is the tree to walk, defaulting to the suite
// at [defaultRoot]; narrowing it is for reading one directory's output, never
// for taking the census the whole corpus answers.
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
const usage = `usage: suiteindex <local[@attr[,attr...]]> [dir]; "," joins the names as all, "|" as any, and any local name may be "*"`

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

// wildcard is the local part that stands for every local name in its
// namespace. It needs no escape and can never be ambiguous: "*" is not an
// NCName (Namespaces in XML §4), so no name a document spells collides with
// it.
const wildcard = "*"

// namePat is one name a query matches: a namespace URI, and either a local
// name or [wildcard]. The wildcard is the local part itself rather than a
// second field beside it — one fact, one encoding (STYLE D3), and an illegal
// "wildcard named foo" is unrepresentable.
type namePat struct {
	Space string
	Local string
}

// isAny reports whether p stands for every local name in its namespace, which
// is what makes a census an axis rather than a lookup.
func (p namePat) isAny() bool {
	return p.Local == wildcard
}

// matches reports whether n is one of the names p stands for. The comparison
// is on namespace URI and local part, never on the prefix a document spelled —
// that equivalence is the tool's whole point.
func (p namePat) matches(n xsd.QName) bool {
	return p.Space == n.Space && (p.isAny() || p.Local == n.Local)
}

// String renders p in the Clark notation the report echoes, wildcard and all
// (`{uri}*`), so the reader sees the namespace that was matched.
func (p namePat) String() string {
	return xsd.QName{Space: p.Space, Local: p.Local}.String()
}

// attrJoin is what a query's attribute list means. Each constant IS its own
// separator, so the query language, the parser and [query.String] hold one
// spelling of it between them rather than a mapping table.
type attrJoin string

const (
	// joinAll admits only an element carrying EVERY name in the list.
	joinAll attrJoin = ","
	// joinAny admits one carrying at least one of them.
	joinAny attrJoin = "|"
)

// query is one census: the elements to look for, the attribute names an
// occurrence carries, and how that list is read.
type query struct {
	Element namePat
	Attrs   []namePat
	Join    attrJoin
}

// anyAttr reports whether the query censuses the attribute-NAME axis — every
// attribute in a namespace rather than named ones. It is derived from the
// attribute list, which parsing has already restricted to a lone wildcard
// (STYLE D3): `@*` names the whole axis and cannot be joined with anything.
func (q query) anyAttr() bool {
	return len(q.Attrs) == 1 && q.Attrs[0].isAny()
}

// String renders the query in the canonical form the report echoes: every
// name in Clark notation, so the reader sees the namespace that was matched
// rather than the prefix some fixture happened to spell it with.
func (q query) String() string {
	var b strings.Builder
	b.WriteString(q.Element.String())
	for i, a := range q.Attrs {
		sep := string(q.Join)
		if i == 0 {
			sep = "@"
		}
		b.WriteString(sep)
		b.WriteString(a.String())
	}
	return b.String()
}

// parseQuery parses `local[@attr[,attr…]]` — `|` in place of `,` for the ANY
// join — where any name may carry a Clark `{uri}` wrapper and any local part
// may be [wildcard]. A braceless element name is in the XML Schema namespace
// — the vocabulary every schema fixture in this corpus is written in — and a
// braceless attribute name is in no namespace, which is what an unprefixed
// attribute resolves to.
func parseQuery(s string) (query, error) {
	elem, rest, err := splitName(s, xsd.XMLSchemaNS)
	if err != nil {
		return query{}, fmt.Errorf("query %q: %w", s, err)
	}
	q := query{Element: elem, Join: joinAll}
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
			return closeAttrs(s, q)
		}
		join := attrJoin(more[:1])
		if join != joinAll && join != joinAny {
			return query{}, fmt.Errorf("query %q: expected %q or %q between attribute names, found %q", s, joinAll, joinAny, more)
		}
		if len(q.Attrs) > 1 && join != q.Join {
			return query{}, fmt.Errorf("query %q: attribute names are joined by %q (every one) or by %q (any one), never both", s, joinAll, joinAny)
		}
		q.Join = join
		rest = more[1:]
	}
}

// closeAttrs finishes a parsed attribute list, rejecting the one combination
// the grammar admits and the matcher cannot mean: a wildcard beside another
// name. `@*` already stands for every attribute, so joining it to a second
// name says nothing under either join.
func closeAttrs(s string, q query) (query, error) {
	if len(q.Attrs) == 1 {
		return q, nil
	}
	for _, a := range q.Attrs {
		if a.isAny() {
			return query{}, fmt.Errorf("query %q: %q names the whole attribute axis and stands alone", s, "@"+wildcard)
		}
	}
	return q, nil
}

// splitName consumes one name from the head of s — an optional Clark
// `{uri}` wrapper, then a local part — and returns it with whatever follows.
// The URI is taken as everything up to the closing brace, so a namespace
// containing a separator survives the separators around it.
func splitName(s, defaultSpace string) (p namePat, rest string, err error) {
	space := defaultSpace
	if strings.HasPrefix(s, "{") {
		end := strings.Index(s, "}")
		if end < 0 {
			return namePat{}, "", fmt.Errorf("unterminated \"{\" in %q", s)
		}
		space = s[1:end]
		s = s[end+1:]
	}
	local := s
	if i := strings.IndexAny(s, "@"+string(joinAll)+string(joinAny)); i >= 0 {
		local, rest = s[:i], s[i:]
	}
	if local == "" {
		return namePat{}, "", errors.New("empty local name")
	}
	if strings.ContainsAny(local, "{}") {
		return namePat{}, "", fmt.Errorf("local name %q contains a brace: write a namespace as a leading {uri}", local)
	}
	return namePat{Space: space, Local: local}, rest, nil
}

// attrHit is one attribute an occurrence carried and the query matched: the
// name as the document resolved it, and its value. The name is the hit's own
// and not the query's, because a query can leave it open — `@*` names no
// attribute at all and the `|` join names more than the element carries.
type attrHit struct {
	Name  xsd.QName
	Value string
}

// hit is one occurrence of the queried construct.
type hit struct {
	// File is the fixture's slash-separated path relative to the census root.
	File string
	Line int
	Col  int
	// Element is the resolved name of the matched element itself. It repeats
	// the query for a query that names one element, and is the only record of
	// which element was hit for a wildcard one — the axis report groups on it.
	Element xsd.QName
	// Parent is the resolved name of the element this match is a direct child
	// of, taken from the document as written and never reconciled with the
	// grammar. The zero QName means the match is the document element and has
	// no parent at all — a distinct value rather than a second field, because
	// no name has an empty local part ([xsd.QName]).
	Parent xsd.QName
	// Children are the resolved names of the elements directly under this
	// match, in document order and with repeats kept. The multiplicity is the
	// point: a content-model census asks whether a second xs:simpleType stands
	// beside a first, which a deduplicated list or a distinct-child tally
	// cannot answer (#1304).
	Children []xsd.QName
	// ChildrenUnclosed reports that the read ended before this match's own end
	// tag, so Children is a prefix of the real list rather than the whole of
	// it. It is a fact about the MATCH, which [report.Partial] — a fact about
	// the file — cannot carry: a fixture that faults at its tail can hold
	// matches that all closed cleanly, and a file-level flag would report
	// their complete child lists as unread.
	ChildrenUnclosed bool
	// Attrs holds the attributes the query matched, in query order — in
	// document order for `@*`, which fixes no order of its own.
	Attrs []attrHit
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

// pairName is the attribute-name axis's unit: one element name and one
// attribute name found on it.
type pairName struct {
	Element xsd.QName
	Attr    xsd.QName
}

// pairGroup is what the census found for one pair — how many occurrences, and
// the fixtures they fall in, each named once and in path order.
type pairGroup struct {
	pairName
	Occurrences int
	Fixtures    []string
}

// groupPairs collects the axis from the hits, in name order. Hits arrive in
// path order, so a pair's fixture list is deduplicated by comparing with the
// last path appended to it and needs no set, exactly as [countFiles] does.
//
// The map is an index into the slice being built and is never ranged over;
// the report's order is the sort below (STYLE D2).
func groupPairs(hits []hit) []pairGroup {
	at := map[pairName]int{}
	var groups []pairGroup
	for _, h := range hits {
		for _, a := range h.Attrs {
			name := pairName{Element: h.Element, Attr: a.Name}
			i, ok := at[name]
			if !ok {
				groups = append(groups, pairGroup{pairName: name})
				i = len(groups) - 1
				at[name] = i
			}
			g := &groups[i]
			g.Occurrences++
			if len(g.Fixtures) == 0 || g.Fixtures[len(g.Fixtures)-1] != h.File {
				g.Fixtures = append(g.Fixtures, h.File)
			}
		}
	}
	sort.Slice(groups, func(i, j int) bool { return lessPair(groups[i].pairName, groups[j].pairName) })
	return groups
}

// lessPair orders the axis by element name and then attribute name, each
// namespace before local part, which is the order the pairs are printed in.
func lessPair(a, b pairName) bool {
	if a.Element != b.Element {
		return a.Element.Space < b.Element.Space ||
			(a.Element.Space == b.Element.Space && a.Element.Local < b.Element.Local)
	}
	return a.Attr.Space < b.Attr.Space ||
		(a.Attr.Space == b.Attr.Space && a.Attr.Local < b.Attr.Local)
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
//
// open is the chain of elements enclosing the token in hand, pushed and
// popped over this one stream rather than recovered by a second parse. Each
// entry also remembers where its own hit lives, which is the whole mechanism
// on the child side: the element a start tag is directly under is the top of
// the stack, so appending the name there needs no depth arithmetic and a
// match nested inside another match is not a special case. A hit is appended
// at its START tag, which is what keeps the report in document order, and its
// child list fills in as the children arrive.
//
// The pop cannot underflow: xmltree emits an end tag only for an element it
// holds open under a matching name, and any other end tag is a fault that
// returns above.
func scanFixture(uri string, r io.Reader, q query) fixtureScan {
	var scan fixtureScan
	var open []openElem
	rd := xmltree.NewReader(uri, r)
	for {
		node, err := rd.Token()
		if err != nil {
			// A match still open when the stream ends never saw its end tag,
			// so its child list stops where the read did.
			markUnclosed(scan.Hits, open)
			if !errors.Is(err, io.EOF) {
				// Kept, not wrapped: an xmltree error already names the
				// document and the position of the fault (STYLE E3), and the
				// report prints the path beside it.
				scan.Err = err
			}
			return scan
		}
		if _, ok := node.(*xmltree.EndElement); ok {
			open = open[:len(open)-1]
			continue
		}
		start, ok := node.(*xmltree.StartElement)
		if !ok {
			continue
		}
		// Both reads happen before the push, so the top of the stack is the
		// enclosing element rather than this one. Each entry is the name the
		// reader resolved in that element's OWN scope, so a parent binding its
		// prefix differently from its child still resolves correctly.
		name := qnameOf(start.Name())
		parent := innermost(open)
		recordChild(scan.Hits, open, name)
		open = append(open, openElem{Name: name, Hit: noHit})
		scan.Elems++
		attrs, ok := match(start, q)
		if !ok {
			continue
		}
		loc := start.Loc()
		scan.Hits = append(scan.Hits, hit{File: uri, Line: loc.Line, Col: loc.Col, Element: name, Parent: parent, Attrs: attrs})
		open[len(open)-1].Hit = len(scan.Hits) - 1
	}
}

// openElem is one element the walk is inside: the name a child of it reports
// as its parent, and where its own hit lives in [fixtureScan.Hits] so that
// child can be recorded against it.
type openElem struct {
	Name xsd.QName
	Hit  int
}

// noHit is [openElem.Hit] for an element that is not itself a match, so
// nothing collects what stands under it.
const noHit = -1

// innermost is the name of the enclosing element the open chain ends with, or
// the zero QName at the document level, where there is none.
func innermost(open []openElem) xsd.QName {
	if len(open) == 0 {
		return xsd.QName{}
	}
	return open[len(open)-1].Name
}

// recordChild writes name into the child list of the match the open chain
// ends with, when that element is one. It writes through hits, which the
// caller holds as [fixtureScan.Hits].
func recordChild(hits []hit, open []openElem, name xsd.QName) {
	if len(open) == 0 {
		return
	}
	at := open[len(open)-1].Hit
	if at == noHit {
		return
	}
	hits[at].Children = append(hits[at].Children, name)
}

// markUnclosed records on every match still open that its child list is a
// prefix of the real one. Every enclosing match is marked and not just the
// innermost, because the read reached none of their end tags.
func markUnclosed(hits []hit, open []openElem) {
	for _, e := range open {
		if e.Hit == noHit {
			continue
		}
		hits[e.Hit].ChildrenUnclosed = true
	}
}

// match reports whether start is an occurrence of q's construct — its
// resolved name is one q's element pattern stands for, and its attributes
// satisfy q's list under q's join — and the attributes that matched.
//
// An element carrying none of the names a `|` query lists is not an
// occurrence, and neither is one carrying no attribute at all under `@*`: an
// attribute census has nothing to say about an element with no attribute.
func match(start *xmltree.StartElement, q query) ([]attrHit, bool) {
	if !q.Element.matches(qnameOf(start.Name())) {
		return nil, false
	}
	if q.anyAttr() {
		return axisAttrs(start, q.Attrs[0])
	}
	var got []attrHit
	for _, want := range q.Attrs {
		v, ok := attrValue(start, want)
		if !ok && q.Join == joinAll {
			return nil, false
		}
		if !ok {
			continue
		}
		got = append(got, attrHit{Name: xsd.QName{Space: want.Space, Local: want.Local}, Value: v})
	}
	if q.Join == joinAny && len(got) == 0 {
		return nil, false
	}
	return got, true
}

// axisAttrs is the attribute-name axis at one start tag: every attribute in
// the wildcard's namespace, in the order the tag spells them, which is the
// only order a query naming no attribute leaves.
func axisAttrs(start *xmltree.StartElement, want namePat) ([]attrHit, bool) {
	var got []attrHit
	for _, a := range start.Attributes() {
		name := qnameOf(a.Name())
		if !want.matches(name) {
			continue
		}
		got = append(got, attrHit{Name: name, Value: a.Value()})
	}
	return got, len(got) > 0
}

// attrValue returns the value of start's want attribute. The attribute list
// is a document-ordered slice, so the first match is the only one a
// well-formed document can have.
func attrValue(start *xmltree.StartElement, want namePat) (string, bool) {
	for _, a := range start.Attributes() {
		if want.matches(qnameOf(a.Name())) {
			return a.Value(), true
		}
	}
	return "", false
}

// qnameOf restates a name the reader resolved in the form the query language
// and the report both speak.
func qnameOf(n xmltree.Name) xsd.QName {
	return xsd.QName{Space: n.Space(), Local: n.Local()}
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

	printBody(w, rep)

	printNotes(w, "Read only partly: a match behind the fault is invisible to this census", rep.Partial)
	printNotes(w, "No XML element: the corpus's images, prose and stylesheets, and any"+
		" fixture that faulted ahead of its root", rep.NoElement)
}

// printBody renders the half of the report the query's shape chooses: the
// attribute-name axis for a query that wildcards its attributes, the matches
// themselves for one that names them.
func printBody(w io.Writer, rep report) {
	if rep.Query.anyAttr() {
		printAxis(w, groupPairs(rep.Hits))
		return
	}
	printMatches(w, rep)
}

// printMatches renders one line per occurrence, which is the whole report for
// every query that names the attributes it looks for.
func printMatches(w io.Writer, rep report) {
	_, _ = fmt.Fprintln(w, "\n=== Matches (path order, document order within a file) ===")
	if len(rep.Hits) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, h := range rep.Hits {
		_, _ = fmt.Fprintf(w, "  %s:%d:%d %sparent=%s children=%s%s\n",
			h.File, h.Line, h.Col, renderMatched(rep.Query.Element, h), renderName(h.Parent),
			renderChildren(h), renderAttrs(h.Attrs))
	}
}

// printAxis renders the attribute-name axis in its two sections: the pairs
// with their counts, then each pair's fixtures. The counts come first because
// that section is the one that fits on a screen; the listing under it runs to
// a line per (pair, fixture), 101,897 of them over the suite (#1391).
func printAxis(w io.Writer, groups []pairGroup) {
	occurrences := 0
	for _, g := range groups {
		occurrences += g.Occurrences
	}
	_, _ = fmt.Fprintf(w, "\n=== Attribute-name axis: %d (element, attribute) pair(s), %d attribute occurrence(s) ===\n",
		len(groups), occurrences)
	if len(groups) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, g := range groups {
		_, _ = fmt.Fprintf(w, "  %s — %d occurrence(s) in %d fixture(s)\n",
			renderPair(g), g.Occurrences, len(g.Fixtures))
	}

	_, _ = fmt.Fprintln(w, "\n=== Fixtures per pair (path order) ===")
	if len(groups) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, g := range groups {
		_, _ = fmt.Fprintf(w, "  %s\n", renderPair(g))
		for _, f := range g.Fixtures {
			_, _ = fmt.Fprintf(w, "      %s\n", f)
		}
	}
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

// renderName names one element around a hit — its parent, or one of its
// children — in the same Clark notation [query.String] echoes. Both sides go
// through this one renderer, so however #1297 settles the spelling of a name
// in no namespace, one fix reaches both.
//
// The zero QName is the parent of a document element, which has none; a child
// is never zero. "(none)" can never collide with a name: parentheses are not
// NCName characters.
func renderName(n xsd.QName) string {
	if n.Local == "" {
		return "(none)"
	}
	return n.String()
}

// renderChildren lists the elements directly under a hit, in document order
// and with repeats kept, as "[a, b]" — "[]" for a match with no element
// child. A list ending in "(unclosed)" is one whose read stopped before the
// match's end tag, so the names ahead of the marker are a prefix; the marker
// cannot be read as a name for the same reason "(none)" cannot.
func renderChildren(h hit) string {
	names := make([]string, 0, len(h.Children)+1)
	for _, c := range h.Children {
		names = append(names, renderName(c))
	}
	if h.ChildrenUnclosed {
		names = append(names, "(unclosed)")
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// renderAttrs formats the attributes a hit matched, each named as the hit
// records it. The local part alone is what a reader recognizes, and no
// ambiguity follows it: a query naming two attributes of one local name in
// different namespaces is not one anybody writes.
func renderAttrs(attrs []attrHit) string {
	var b strings.Builder
	for _, a := range attrs {
		fmt.Fprintf(&b, " %s=%q", a.Name.Local, a.Value)
	}
	return b.String()
}

// renderMatched names the element a hit matched, for a query that left the
// element open with a wildcard — and nothing at all for one that fixed it,
// since the report's own header already echoes the query. The trailing space
// belongs to the field, so the line closes up when there is none.
func renderMatched(want namePat, h hit) string {
	if !want.isAny() {
		return ""
	}
	return "element=" + renderName(h.Element) + " "
}

// renderPair names one axis pair in the form the two axis sections share, so
// a pair's heading in the second is greppable from its line in the first.
func renderPair(g pairGroup) string {
	return renderName(g.Element) + " @" + renderName(g.Attr)
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
