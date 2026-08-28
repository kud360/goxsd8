package conformance

import (
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
)

// maxCensusViolationsLogged caps the per-run report of soundness violations. The
// count is always exact; only the naming of individual documents stops, because
// a predicate that has gone unsound is unsound on thousands of suite documents
// at once and the first few name the shape as well as all of them would.
const maxCensusViolationsLogged = 10

// TestUnmappedCensusSoundAgainstShapeGate holds parser's coverage census
// (parser.AssembledDocument.Unmapped, #1029) to the ONE direction this session
// establishes, over every schema document of every suite case:
//
//	len(d.Unmapped) > 0  ⟹  !schemaShapeDecidable(d.Doc)
//
// That is the false-accept direction. The census is the signal a later session
// replaces this file's hand-kept allowlist with, and a replacement is only safe
// while the census names NOTHING the allowlist still admits: a construct flagged
// unmapped that the gate would go on to decide is a case decided on a document
// the producer did not read.
//
// The converse — every document the gate declines holds a flagged construct —
// is NOT asserted, and does not hold. Those documents are counted as the
// RESIDUAL and logged, so a widening session can read how far it has to go and
// drive that number down without ever being able to make this test pass by
// widening the census wrongly. It will not reach zero by censusing more, because
// the residual has THREE causes and only one of them is a region left unwalked
// (parser/census.go "Scope"):
//
//   - a region the census does not walk yet;
//   - a shape the producer REJECTS, which the gate declines anyway out of
//     conservatism — a <simpleType> naming none of §3.16.2.1's three
//     alternatives (#786), a nested <group>/<attributeGroup> with no ref, a
//     model group's non-particle child, a named definition with no usable name.
//     No census can report these: they are verdicts, not silences, so each is a
//     gate widening with its own ratchet measurement;
//   - a shape the producer MAPS while the assembled schema is short of a
//     property or of a verdict on it — anonymousComplexTypeDecidable's
//     non-implicit shapes (#414/#438). These are the bulk of the residual, they
//     are finalize-side rather than dispatch-side, and UnmappedNoDispatch is by
//     construction not their reason.
//
// So the residual is the measure of what still separates the two, not of how
// much census is left to write.
//
// One class of document is exempt, and counted rather than asserted about:
// schemaShapeDecidable's FIRST answer is not an allowlist verdict at all but the
// unconditional-rejection short-circuit (holdsMisplacedNotation), which admits a
// document precisely because the producer rejects it whatever it holds. Three
// suite fixtures are both — msData/notations/notatF005, notatF029 and notatF039
// hang their misplaced <notation> off a top-level <xs:any>, <xs:key> and
// <xs:keyref> the census duly flags — and no unmapped construct can make a
// rejected document accept anything, so flagging one there is no false-accept
// hazard.
func TestUnmappedCensusSoundAgainstShapeGate(t *testing.T) {
	skipWithoutSuite(t)
	found, err := parseSuite(suitePath())
	if err != nil {
		t.Fatalf("parsing suite: %v", err)
	}
	backend := strict.New()
	// Assembled roots are deduplicated because a whole testGroup's instanceTests
	// share one schema document, and assembling it once per case would multiply
	// the run for one repeated verdict. The map is a membership set only; every
	// reported number and message comes from the case-ID-sorted walk (STYLE D2).
	assembled := map[string]struct{}{}
	var discoveries, flagged, violations, exempt, residual int
	for _, c := range found.cases {
		root := censusRoot(c)
		if root == "" {
			continue
		}
		if _, dup := assembled[root]; dup {
			continue
		}
		assembled[root] = struct{}{}
		// The report, never the error: a rejected assembly still reports the
		// documents it read, and their censuses with them.
		_, report, _ := parser.ParseReport(filepath.Base(root),
			parser.WithResolver(loader.Dir(filepath.Dir(root))), parser.WithBackend(backend))
		for _, d := range report.Documents() {
			discoveries++
			decidable := schemaShapeDecidable(d.Doc)
			if len(d.Unmapped) == 0 {
				if !decidable {
					residual++
				}
				continue
			}
			flagged++
			if !decidable {
				continue
			}
			if holdsMisplacedNotation(d.Doc.Root()) {
				// Admitted by the short-circuit, not by the allowlist: the producer
				// rejects this document unconditionally, so nothing it holds decides
				// anything vacuously.
				exempt++
				continue
			}
			violations++
			if violations <= maxCensusViolationsLogged {
				t.Errorf("%s: %s: census reports <%s> unmapped at %s, but schemaShapeDecidable admits the document",
					c.id, d.Location, d.Unmapped[0].Name, d.Unmapped[0].At)
			}
		}
	}
	if violations > maxCensusViolationsLogged {
		t.Errorf("census unsound on %d document discoveries in all; %d named above",
			violations, maxCensusViolationsLogged)
	}
	t.Logf("coverage census: %d document discoveries over %d assembled roots, %d flagged unmapped (%d of them in unconditionally-rejected documents), %d residual (gate declines, census silent)",
		discoveries, len(assembled), flagged, exempt, residual)
}

// censusRoot returns the schema document c roots an assembly at — c.doc for a
// schemaTest, the group's schema for an instanceTest — or "" when the case names
// none (an instanceTest whose group declares no schema document, which
// execInstanceCase declines for the same reason).
func censusRoot(c caseSpec) string {
	if c.kind == kindSchema {
		return c.doc
	}
	return c.schemaDoc
}
