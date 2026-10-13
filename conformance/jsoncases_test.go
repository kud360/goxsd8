package conformance

import (
	"encoding/xml"
	"os"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/jsonsrc"
	"github.com/kud360/goxsd8/xsderr"
)

// curatedForTest is the committed curated corpus, or the test ends.
func curatedForTest(t *testing.T) []caseSpec {
	t.Helper()
	cases, err := curatedJSONCases(jsonCorpusIndex)
	if err != nil {
		t.Fatalf("curatedJSONCases: %v", err)
	}
	return cases
}

// TestCuratedJSONCorpus pins the corpus's shape: kindJSON cases only, with IDs
// an expectation line and the arbiter's superseded list can carry (no ',', ':'
// or white space, parseSupersededAssertions), a valid and an invalid case for
// each arm's group, and every case agreeing with its declared validity under
// the json executor — while the opposite declaration fails, so the agreement
// is the executor's observation and not a constant.
func TestCuratedJSONCorpus(t *testing.T) {
	cases := curatedForTest(t)
	if len(cases) < 6 {
		t.Fatalf("the corpus holds %d cases, want at least 6", len(cases))
	}
	exec := newJSONExec()
	polarity := map[string][2]int{}
	for _, c := range cases {
		if c.kind != kindJSON {
			t.Errorf("%s: kind %v, want kindJSON", c.id, c.kind)
		}
		if strings.ContainsFunc(c.id, func(r rune) bool { return r == ',' || r == ':' || unicode.IsSpace(r) }) {
			t.Errorf("%s: the ID carries ',', ':' or white space", c.id)
		}
		group := strings.Split(c.id, "/")[1]
		p := polarity[group]
		if c.expect.wantsValid() {
			p[0]++
		} else {
			p[1]++
		}
		polarity[group] = p
		if got, why := exec(c); !got.IsPass() {
			t.Errorf("%s: %v (refusal %q), want pass", c.id, got, why)
		}
		flipped := c
		flipped.expect = expectValidity(!c.expect.wantsValid())
		if got, _ := exec(flipped); got.IsPass() {
			t.Errorf("%s: passes declared the other way too", c.id)
		}
	}
	for _, group := range []string{"scalar", "child", "array"} {
		if p := polarity[group]; p[0] == 0 || p[1] == 0 {
			t.Errorf("group %s holds %d valid and %d invalid cases, want at least one of each", group, p[0], p[1])
		}
	}
}

// TestNoOtherLaneClaimsACuratedCase pins that the curated corpus reaches the
// json lane alone, which is what keeps every other lane's total unmoved by it:
// the read-only gate cannot see such a leak, a leaked case being New.
func TestNoOtherLaneClaimsACuratedCase(t *testing.T) {
	cases := curatedForTest(t)
	for _, l := range defaultLanes() {
		for _, c := range cases {
			if got, want := l.selects(c), l.name == "json"; got != want {
				t.Errorf("lane %s selects %s = %v, want %v", l.name, c.id, got, want)
			}
		}
	}
}

// corpusAnnotations mirrors the curated testSet's instanceTests with the one
// field the suite's structs do not read: the documentation that names an
// invalid case's expected rule.
type corpusAnnotations struct {
	Groups []struct {
		Name  string `xml:"name,attr"`
		Tests []struct {
			Name string `xml:"name,attr"`
			Doc  string `xml:"annotation>documentation"`
		} `xml:"instanceTest"`
	} `xml:"testGroup"`
}

// TestJSONCorpusChargesItsNamedRules holds each invalid case to the rule its
// documentation names ("expects <rule>"): that rule is among the
// Violations() the assessment charges it.
func TestJSONCorpusChargesItsNamedRules(t *testing.T) {
	raw, err := os.ReadFile(jsonCorpusIndex)
	if err != nil {
		t.Fatalf("reading %s: %v", jsonCorpusIndex, err)
	}
	var ann corpusAnnotations
	if err := xml.Unmarshal(raw, &ann); err != nil {
		t.Fatalf("decoding %s: %v", jsonCorpusIndex, err)
	}
	named := map[string]string{}
	for _, g := range ann.Groups {
		for _, it := range g.Tests {
			rule, ok := strings.CutPrefix(strings.TrimSpace(it.Doc), "expects ")
			if !ok {
				continue
			}
			rule, _, _ = strings.Cut(rule, ":")
			named[g.Name+"/"+it.Name] = rule
		}
	}
	backend := strict.New()
	for _, c := range curatedForTest(t) {
		if c.expect.wantsValid() {
			continue
		}
		parts := strings.Split(c.id, "/")
		rule, ok := named[parts[1]+"/"+parts[3]]
		if !ok {
			t.Errorf("%s: invalid, and its documentation names no expected rule", c.id)
			continue
		}
		schema, _, _, perr := assembleCase(backend, c.schemaDoc, c.schemaExtraDocs)
		if perr != nil {
			t.Fatalf("%s: assembling %s: %v", c.id, c.schemaDoc, perr)
		}
		v, err := validate.New(schema, backend)
		if err != nil {
			t.Fatalf("validate.New: %v", err)
		}
		f, err := os.Open(c.doc)
		if err != nil {
			t.Fatalf("%s: %v", c.id, err)
		}
		res, err := jsonsrc.Validate(v, f, jsonsrc.WithURI(c.doc))
		_ = f.Close()
		if err != nil {
			t.Fatalf("%s: jsonsrc.Validate: %v", c.id, err)
		}
		var charged []xsderr.Rule
		for _, e := range res.Violations() {
			charged = append(charged, e.Rule)
		}
		if !slices.Contains(charged, xsderr.Rule(rule)) {
			t.Errorf("%s: expects %s, charged %v", c.id, rule, charged)
		}
	}
}
