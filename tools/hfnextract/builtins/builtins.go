// Package builtins parses the built-in datatype definitions out of the
// local Datatypes spec (docs/specs/md/xmlschema11-2.md) and the
// precisionDecimal Note (docs/specs/md/xsd-precisionDecimal.md) into a
// backend-neutral, JSON-serializable form.
//
// It is the spec-parsing half of the M1 builtin pipeline (PRINCIPLES 26/27):
// it reads Markdown and emits structured data, and knows nothing about Go
// code generation. tools/typespecgen consumes [Parse]'s output and emits
// builtin/gen_typespec.go. Keeping the two apart means a spec-parsing bug
// and an emission bug never hide in the same file.
//
// Every value is read from the normative prose; nothing is hand-transcribed.
// The per-type "Facets" subsections (§3.3.N.·/§3.4.N.· with anchor
// <type>-facets) are the source for applicable facets, their spec defaults,
// and the four fundamental facets; each type's opening "[Definition:]"
// sentence is the source for base type, variety, and (for lists) item type.
// The fundamental facets are independently cross-checked against the
// Appendix F.1 table (id="app-fundamental-facets"); a disagreement is a
// hard error, so a parsing slip in either source fails the generate step
// rather than emitting a wrong row (STYLE S3).
package builtins

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kud360/goxsd8/tools/hfnextract/internal"
)

// Builtin is one parsed builtin datatype row. String-typed and
// JSON-tagged so the parser stays a pure spec-to-data step; the Go enum
// mapping lives in the emitter.
type Builtin struct {
	Name        string  `json:"name"`
	Base        string  `json:"base"`
	Variety     string  `json:"variety"`               // "atomic" or "list"
	Item        string  `json:"item,omitempty"`        // list item type; empty for atomic
	Ordered     string  `json:"ordered,omitempty"`     // "" (absent) for anyAtomicType
	Bounded     string  `json:"bounded,omitempty"`     // ""
	Cardinality string  `json:"cardinality,omitempty"` // ""
	Numeric     string  `json:"numeric,omitempty"`     // ""
	Facets      []Facet `json:"facets,omitempty"`      // applicable constraining facets, spec order
}

// Facet is one applicable constraining facet with its spec-given default.
type Facet struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"` // spec default/fixed value; "" if no default
	Fixed bool   `json:"fixed,omitempty"` // value must not be changed by restriction
}

const (
	primitiveBase = "anyAtomicType" // §4.1.6 dummy-def: every primitive derives from anyAtomicType
	listBase      = "anySimpleType" // §3.4/§4.1.6: a list restricts an anonymous list whose base is anySimpleType
)

// fundamentalNames is the closed set of fundamental-facet names (§4.2); a
// facet bullet naming one of these is routed to the Builtin's own
// fundamental fields rather than to Facets.
var fundamentalNames = map[string]bool{"ordered": true, "bounded": true, "cardinality": true, "numeric": true}

var (
	reTypeHeader = regexp.MustCompile(`^#### <a id="([A-Za-z0-9]+)"></a>(3\.[234]\.[0-9]+) `)
	reSubHeader  = regexp.MustCompile(`^#{4,6} `)
	reFacetName  = regexp.MustCompile(`#rf-([A-Za-z]+)\)`)
)

// Parse reads the two spec files and returns all 49 builtin datatypes in
// spec order: the 19 primitives (§3.3), the 28 ordinary datatypes (§3.4),
// anyAtomicType (§3.2.2), then precisionDecimal (xsd-precisionDecimal.md
// §3). anySimpleType is deliberately excluded: it is the ur-type with an
// empty {facets} set and cannot be a facet-based restriction base
// (§3.2.1.3), so it has no meaningful row.
func Parse(structuresPath, precisionPath string) ([]Builtin, error) {
	content, err := os.ReadFile(structuresPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", structuresPath, err)
	}
	lines := strings.Split(string(content), "\n")

	primitives, ordinary, aat, err := parseStructures(lines)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", structuresPath, err)
	}
	if len(primitives) != 19 {
		return nil, fmt.Errorf("parsing %s: found %d primitive datatypes, want 19", structuresPath, len(primitives))
	}
	if len(ordinary) != 28 {
		return nil, fmt.Errorf("parsing %s: found %d ordinary datatypes, want 28", structuresPath, len(ordinary))
	}

	if err := crossCheckFundamental(structuresPath, append(append([]Builtin{}, primitives...), ordinary...)); err != nil {
		return nil, err
	}

	pd, err := parsePrecisionDecimal(precisionPath)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", precisionPath, err)
	}

	out := make([]Builtin, 0, 49)
	out = append(out, primitives...)
	out = append(out, ordinary...)
	out = append(out, aat)
	out = append(out, pd)
	return out, nil
}

// parseStructures walks the Datatypes spec once, slicing each builtin's
// section between consecutive type headers and parsing it.
func parseStructures(lines []string) (primitives, ordinary []Builtin, aat Builtin, err error) {
	type section struct {
		name, sec string
		start     int
	}
	var secs []section
	for i, line := range lines {
		m := reTypeHeader.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if m[2] == "3.2.1" { // anySimpleType — excluded from the 49
			continue
		}
		secs = append(secs, section{name: m[1], sec: m[2], start: i})
	}

	haveAAT := false
	for i, s := range secs {
		end := len(lines)
		if i+1 < len(secs) {
			end = secs[i+1].start
		}
		b, perr := parseSection(s.name, s.sec, lines[s.start:end])
		if perr != nil {
			return nil, nil, Builtin{}, perr
		}
		switch {
		case s.sec == "3.2.2":
			aat = b
			haveAAT = true
		case strings.HasPrefix(s.sec, "3.3."):
			primitives = append(primitives, b)
		case strings.HasPrefix(s.sec, "3.4."):
			ordinary = append(ordinary, b)
		}
	}
	if !haveAAT {
		return nil, nil, Builtin{}, fmt.Errorf("anyAtomicType (§3.2.2) section not found")
	}
	return primitives, ordinary, aat, nil
}

// parseSection parses one builtin from its section lines. section is the
// dotted section number (e.g. "3.4.13"); it selects how base/variety are
// determined.
func parseSection(name, section string, lines []string) (Builtin, error) {
	// Definition text runs from the type header to the first subheader.
	defEnd := len(lines)
	facetStart := -1
	for i := 1; i < len(lines); i++ {
		if !reSubHeader.MatchString(lines[i]) {
			continue
		}
		if defEnd == len(lines) {
			defEnd = i
		}
		if facetStart == -1 && strings.HasSuffix(strings.TrimSpace(lines[i]), "Facets") {
			facetStart = i
		}
	}
	def := strings.Join(lines[1:defEnd], "\n")

	b := Builtin{Name: name}
	switch {
	case section == "3.2.2": // anyAtomicType
		b.Base = linkAfter(def, `restriction·\]\(#dt-restriction\) of \[`)
		b.Variety = "atomic"
	case strings.HasPrefix(section, "3.3."): // primitive
		b.Base = primitiveBase
		b.Variety = "atomic"
	default: // ordinary (§3.4)
		if strings.Contains(def, "an anonymous list type is defined") {
			b.Variety = "list"
			b.Base = listBase
			b.Item = linkAfter(def, `#dt-itemType\) of .*?\bis \[`)
			if b.Item == "" {
				return Builtin{}, fmt.Errorf("%s: list item type not found", name)
			}
			break
		}
		b.Variety = "atomic"
		// Most ordinary types state the base in a "·base type· of X is Y"
		// sentence (X either bold, of **integer**is [decimal], or a link,
		// of [language]… is [token]). The three 1.1 additions
		// (yearMonthDuration, dayTimeDuration, dateTimeStamp) instead only
		// say "·derived· from [Y]"; fall back to that.
		b.Base = linkAfter(def, `#dt-basetype\) of .*?\bis \[`)
		if b.Base == "" {
			b.Base = linkAfter(def, `#dt-derived\) from \[`)
		}
	}
	if b.Base == "" {
		return Builtin{}, fmt.Errorf("%s: base type not found", name)
	}

	if facetStart != -1 {
		facetEnd := len(lines)
		for i := facetStart + 1; i < len(lines); i++ {
			if reSubHeader.MatchString(lines[i]) {
				facetEnd = i
				break
			}
		}
		if err := parseFacets(&b, lines[facetStart+1:facetEnd]); err != nil {
			return Builtin{}, fmt.Errorf("%s: %w", name, err)
		}
	}

	// anyAtomicType alone has no fundamental facets (§4.1.6 anyAtomicType-def:
	// {fundamental facets} is empty). Every other type must state all four,
	// or the Facets subsection was misparsed (fail loud, STYLE S3).
	if section != "3.2.2" {
		if b.Ordered == "" || b.Bounded == "" || b.Cardinality == "" || b.Numeric == "" {
			return Builtin{}, fmt.Errorf("%s: incomplete fundamental facets (ordered=%q bounded=%q cardinality=%q numeric=%q)", name, b.Ordered, b.Bounded, b.Cardinality, b.Numeric)
		}
	}
	return b, nil
}

// parseFacets reads the "- [facet](#rf-facet) = ***value***(fixed)" bullets
// of a Facets subsection, routing the four fundamental facets to b's own
// fields and the rest to b.Facets in document order.
func parseFacets(b *Builtin, lines []string) error {
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "- ") || !strings.Contains(t, "#rf-") {
			continue
		}
		m := reFacetName.FindStringSubmatch(t)
		if m == nil {
			return fmt.Errorf("facet bullet without #rf- anchor: %q", t)
		}
		name := m[1]
		value, fixed := parseFacetValue(t)
		if fundamentalNames[name] {
			if value == "" {
				return fmt.Errorf("fundamental facet %q has no value: %q", name, t)
			}
			switch name {
			case "ordered":
				b.Ordered = value
			case "bounded":
				b.Bounded = value
			case "cardinality":
				b.Cardinality = value
			case "numeric":
				b.Numeric = value
			}
			continue
		}
		b.Facets = append(b.Facets, Facet{Name: name, Value: value, Fixed: fixed})
	}
	return nil
}

// parseFacetValue extracts the bold value of a facet bullet. It strips
// exactly one "***" delimiter from each end so that values containing their
// own asterisks (e.g. the Name pattern `\i\c*`) survive intact.
func parseFacetValue(bullet string) (value string, fixed bool) {
	i := strings.Index(bullet, "= ***")
	if i == -1 {
		return "", false
	}
	rem := bullet[i+len("= ***"):]
	rem = strings.TrimRight(rem, " ")
	if strings.HasSuffix(rem, "(fixed)") {
		fixed = true
		rem = strings.TrimRight(strings.TrimSuffix(rem, "(fixed)"), " ")
	}
	value = strings.TrimSuffix(rem, "***")
	return value, fixed
}

// linkAfter returns the target of the first "[target](#..." markdown link
// occurring immediately after prefixRE, or "" if prefixRE is not found.
func linkAfter(text, prefixRE string) string {
	re := regexp.MustCompile(prefixRE + `([A-Za-z0-9]+)\]`)
	m := re.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return m[1]
}

// crossCheckFundamental verifies each parsed type's four fundamental facets
// against Appendix F.1 (id="app-fundamental-facets"). F.1 covers exactly the
// 19 primitives and 28 ordinary types; anyAtomicType and precisionDecimal
// are absent from it and are checked elsewhere.
func crossCheckFundamental(structuresPath string, types []Builtin) error {
	f1, err := fundamentalTable(structuresPath)
	if err != nil {
		return err
	}
	if len(f1) != len(types) {
		return fmt.Errorf("spec Appendix F.1 lists %d datatypes, parsed %d", len(f1), len(types))
	}
	for _, b := range types {
		got := [4]string{b.Ordered, b.Bounded, b.Cardinality, b.Numeric}
		want, ok := f1[b.Name]
		if !ok {
			return fmt.Errorf("%s: absent from Appendix F.1", b.Name)
		}
		if got != want {
			return fmt.Errorf("%s: fundamental facets %v disagree with Appendix F.1 %v", b.Name, got, want)
		}
	}
	return nil
}

// fundamentalTable extracts Appendix F.1 as a name→(ordered,bounded,
// cardinality,numeric) map. F.1 rows carry a group-label first column
// ("primitive"/"non-primitive") only on the first row of each group; on
// other rows the datatype shifts into that column and a trailing empty cell
// appears. Dropping the label cells and trailing blanks normalizes every
// data row to exactly [name, ordered, bounded, cardinality, numeric].
func fundamentalTable(structuresPath string) (map[string][4]string, error) {
	tables, err := internal.ExtractTables(structuresPath, "", -1)
	if err != nil {
		return nil, err
	}
	var f1 *internal.Table
	for i := range tables {
		if headerHas(tables[i].Header, "ordered", "bounded", "cardinality", "numeric") {
			f1 = &tables[i]
			break
		}
	}
	if f1 == nil {
		return nil, fmt.Errorf("spec Appendix F.1 fundamental-facets table not found")
	}

	out := make(map[string][4]string)
	for _, row := range f1.Rows {
		var cells []string
		for _, c := range row {
			if c == "primitive" || c == "non-primitive" {
				continue
			}
			cells = append(cells, c)
		}
		for len(cells) > 0 && cells[len(cells)-1] == "" {
			cells = cells[:len(cells)-1]
		}
		if len(cells) == 0 {
			continue // spacer row
		}
		if len(cells) != 5 {
			return nil, fmt.Errorf("spec Appendix F.1 row has %d cells, want 5: %v", len(cells), row)
		}
		out[cells[0]] = [4]string{cells[1], cells[2], cells[3], cells[4]}
	}
	return out, nil
}

func headerHas(header []string, want ...string) bool {
	joined := strings.Join(header, "|")
	for _, w := range want {
		if !strings.Contains(joined, w) {
			return false
		}
	}
	return true
}

// parsePrecisionDecimal reads the single precisionDecimal datatype from the
// precisionDecimal Note. Its base type is not stated in the Note; as an
// implementation-defined primitive it takes anyAtomicType per xmlschema11-2
// §4.1.6 dummy-def (the only spec-consistent reading; the Note is silent).
func parsePrecisionDecimal(precisionPath string) (Builtin, error) {
	content, err := os.ReadFile(precisionPath)
	if err != nil {
		return Builtin{}, err
	}
	lines := strings.Split(string(content), "\n")

	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "### ") && strings.HasSuffix(strings.TrimSpace(line), "Facets") && strings.Contains(line, "sec-f-pD") {
			start = i
			break
		}
	}
	if start == -1 {
		return Builtin{}, fmt.Errorf("precisionDecimal Facets subsection (id=sec-f-pD) not found")
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") || strings.HasPrefix(lines[i], "### ") {
			end = i
			break
		}
	}

	b := Builtin{Name: "precisionDecimal", Base: primitiveBase, Variety: "atomic"}
	if err := parseFacets(&b, lines[start+1:end]); err != nil {
		return Builtin{}, fmt.Errorf("precisionDecimal: %w", err)
	}
	if b.Ordered == "" || b.Bounded == "" || b.Cardinality == "" || b.Numeric == "" {
		return Builtin{}, fmt.Errorf("precisionDecimal: incomplete fundamental facets")
	}
	return b, nil
}
