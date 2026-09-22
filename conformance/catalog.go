package conformance

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// This file is the catalog READER (issue #1642): it answers which case IDs
// name a given suite document, for a caller that holds fixture paths and needs
// the IDs conformance/testdata/expectations/<lane>.txt is keyed by. It runs no
// case and decides nothing — parseSuite and everything under it are untouched
// by it, and a reader's answer never reaches a lane's score.
//
// It is a SECOND walk over the same catalog, not a second reading of it. Every
// fact it reports comes from the runner's own primitives — caseID, caseDocs,
// groupSchemaDocs, versionApplicable and resolveExpected — so the ID it hands
// out is the ID makeCase stamps, by construction and not by agreement (STYLE
// D3). What differs is the traversal: casesFromSet stops descending at the
// level that withheld a case and produces nothing for it, because a case the
// suite scoped away is not this processor's to run; a reader of the catalog
// must report that entry anyway, since its ID is a real catalog entry that
// merely carries no line in any lane. The two also differ on a malformed
// entry: casesFromSet ends the run, and the reader reports the entry with no
// document rather than refusing to describe the catalog it was asked about.

// suiteIndexIn names the suite index inside a suite checkout — the file whose
// absence means the submodule is not initialized. It is the ONE construction
// of that name (STYLE D3): suitePath takes it over the package-relative
// suiteRoot, Catalog over whatever directory its caller names.
func suiteIndexIn(dir string) string { return filepath.Join(dir, "suite.xml") }

// CatalogEntry is one entry of the W3C suite catalog — one schemaTest or
// instanceTest — as a reader holding a fixture path needs it: the ID the
// expectation files are keyed by, the documents the entry names, and what the
// catalog declares about it.
//
// Its consumer is tools/casejoin, which turns a tools/suiteindex census of
// fixture paths into the case IDs CLAUDE.md's ratchet-prediction join runs
// against (STYLE T5). It is a plain record because it is a REPORT: every field
// is a fact read off the catalog, and nothing downstream may construct one and
// have it mean anything.
type CatalogEntry struct {
	// ID is the entry's case ID, `<testSet>/<testGroup>/<kind>/<test-name>`
	// (runner.go "Case IDs"), built by the same caseID makeCase stamps a
	// produced case with. It is what an expectation-file line is keyed by, so
	// an ID that carries a line matches it by exact string.
	ID string
	// Docs are the documents UNDER TEST, in catalog order: a schemaTest's
	// <schemaDocument> list, or an instanceTest's one <instanceDocument>.
	// Each is a slash path relative to the suite root, which is how a corpus
	// census names a fixture (tools/suiteindex). It is empty for an entry
	// naming no document at all — the malformed schemaTest caseDocs refuses.
	Docs []string
	// SchemaDocs are the schema documents an INSTANCE entry is assessed
	// against: its test group's sibling schemaTest's <schemaDocument> list,
	// resolved exactly as Docs are (groupSchemaDocs). It is empty for a
	// schemaTest, whose own Docs are its schema documents already (STYLE D3),
	// and for an instanceTest whose group declares no single schemaTest. The
	// two lists are kept apart because they answer different questions of a
	// matched fixture: whether the case tests that document, or is merely
	// assessed against it.
	SchemaDocs []string
	// Withheld reports that the suite's own OR-connected `version` metadata
	// scopes this entry away from this processor at its testSet, testGroup or
	// test level (versionApplicable, issue #446), so discovery produces no
	// case for it. A withheld entry carries no line in any lane and can never
	// flip a lane's score (issue #1412); it is still a catalog entry, which is
	// why a reader reports it rather than omitting it.
	Withheld bool
	// DeclaredValid reports that the suite declares this entry's document
	// VALID for this processor's configuration (resolveExpected, the same
	// reading the runner scores against). It is false for an entry declared
	// invalid, for one declared indeterminate, and for one declaring no
	// expected outcome at all.
	DeclaredValid bool
}

// Catalog reads the suite catalog rooted at suiteDir — the directory holding
// suite.xml, `testdata/xsdtests` from the module root — and returns every
// entry in it, sorted by ID (STYLE D1). Withheld entries are included and
// marked, because the set of IDs the catalog NAMES is a larger set than the
// cases a run produces, and a reader holding a fixture path is asking about
// the former.
//
// It errors on an absent suite, a malformed index or an unreadable test set.
// Uniqueness of IDs is NOT asserted here — parseSuite asserts it over the
// cases a run produces, which is where a collision would decide a score — so a
// caller counting entries counts distinct IDs rather than trusting this slice
// to hold each once.
func Catalog(suiteDir string) ([]CatalogEntry, error) {
	index := suiteIndexIn(suiteDir)
	if err := checkSuitePresent(index); err != nil {
		return nil, err
	}
	seenSets := map[string]struct{}{}
	var entries []CatalogEntry
	for _, path := range suiteIndexPaths(index) {
		found, err := catalogFromIndex(path, suiteDir, seenSets)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}
	slices.SortFunc(entries, func(a, b CatalogEntry) int { return strings.Compare(a.ID, b.ID) })
	return entries, nil
}

// catalogFromIndex reads every entry reachable from one suite index, sharing
// seenSets with any sibling index so a test set both of them reference —
// common/introspection.testSet — is read once, exactly as casesFromIndex
// shares it.
func catalogFromIndex(indexPath, suiteDir string, seenSets map[string]struct{}) ([]CatalogEntry, error) {
	idx, err := decodeSuiteIndex(indexPath)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(indexPath)
	var entries []CatalogEntry
	for _, ref := range idx.Refs {
		if ref.Href == "" {
			continue
		}
		setPath := filepath.Join(baseDir, filepath.FromSlash(ref.Href))
		if _, done := seenSets[setPath]; done {
			continue
		}
		seenSets[setPath] = struct{}{}
		set, err := decodeTestSet(setPath)
		if err != nil {
			return nil, fmt.Errorf("test set %s: %w", ref.Href, err)
		}
		found, err := catalogFromSet(set, filepath.Dir(setPath), suiteDir)
		if err != nil {
			return nil, fmt.Errorf("test set %s: %w", ref.Href, err)
		}
		entries = append(entries, found...)
	}
	return entries, nil
}

// catalogFromSet flattens one testSet into entries, in catalog order.
//
// Unlike casesFromSet it descends THROUGH a withheld level rather than
// stopping at it: applicability is carried down as a flag and stamped on each
// entry, so a set or group the suite scopes away still yields one entry per
// test it declares. Withholding at the coarsest level that decided it — the
// property that keeps the runner's produced and withheld sets disjoint — is
// preserved by construction here, since an entry is withheld when ANY level
// above it is.
func catalogFromSet(set testSet, setDir, suiteDir string) ([]CatalogEntry, error) {
	setOK := versionApplicable(set.Version)
	var entries []CatalogEntry
	for _, g := range set.Groups {
		groupOK := setOK && versionApplicable(g.Version)
		for _, st := range g.SchemaTests {
			e, err := catalogEntry(set.Name, kindSchema, st, g, setDir, suiteDir, groupOK)
			if err != nil {
				return nil, err
			}
			entries = append(entries, e)
		}
		for _, it := range g.InstanceTests {
			e, err := catalogEntry(set.Name, kindInstance, it, g, setDir, suiteDir, groupOK)
			if err != nil {
				return nil, err
			}
			entries = append(entries, e)
		}
	}
	return entries, nil
}

// catalogEntry describes one schemaTest or instanceTest. levelsOK is whether
// every level ABOVE this test is applicable to this processor; the test's own
// `version` is read here, so the three levels versionApplicable governs meet
// in one flag.
func catalogEntry(setName, kind string, t validityTest, g testGroup, setDir, suiteDir string, levelsOK bool) (CatalogEntry, error) {
	docs, err := catalogDocs(kind, t, setDir, suiteDir)
	if err != nil {
		return CatalogEntry{}, err
	}
	schemaDocs, err := groupCatalogSchemaDocs(kind, g, setDir, suiteDir)
	if err != nil {
		return CatalogEntry{}, err
	}
	want, declared := resolveExpected(t.Expected)
	return CatalogEntry{
		ID:            caseID(setName, g.Name, kind, t.Name),
		Docs:          docs,
		SchemaDocs:    schemaDocs,
		Withheld:      !levelsOK || !versionApplicable(t.Version),
		DeclaredValid: declared && want.wantsValid(),
	}, nil
}

// catalogDocs is one entry's documents under test, suite-relative. It reads
// the declarations caseDocs reads, through resolveDoc and resolveDocs — the
// ONE reading of each list (STYLE T4) — and parts from caseDocs on the entry
// caseDocs REFUSES: a schemaTest declaring no <schemaDocument> at all yields
// an entry naming NO DOCUMENT rather than an error ending the walk. makeCase
// is right to end a run over a catalog entry it cannot test; a reader asked
// which entries name a path must still describe the entry that names none,
// and such an entry matches no path, so nothing is admitted by it.
func catalogDocs(kind string, t validityTest, setDir, suiteDir string) ([]string, error) {
	if kind == kindInstance {
		return suiteRelative(suiteDir, []string{resolveDoc(setDir, t.InstanceDoc.Href)})
	}
	doc, extra, ok := resolveDocs(t.SchemaDocs, setDir)
	if !ok {
		return nil, nil
	}
	return suiteRelative(suiteDir, append([]string{doc}, extra...))
}

// groupCatalogSchemaDocs is the schema documents an instance entry is assessed
// against, suite-relative. groupSchemaDocs yields none for a schemaTest and
// none for a group that declares no single schemaTest, and this reader carries
// that refusal through unchanged.
func groupCatalogSchemaDocs(kind string, g testGroup, setDir, suiteDir string) ([]string, error) {
	doc, extra := groupSchemaDocs(kind, g, setDir)
	if doc == "" {
		return nil, nil
	}
	return suiteRelative(suiteDir, append([]string{doc}, extra...))
}

// suiteRelative restates resolved document paths as slash paths relative to
// the suite root. That is the spelling a corpus census names a fixture with
// (tools/suiteindex reports every path relative to its census root), and a
// join of the two sets compares strings, so the two spellings have to be one.
func suiteRelative(suiteDir string, paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, err := filepath.Rel(suiteDir, p)
		if err != nil {
			return nil, fmt.Errorf("relating document %s to suite root %s: %w", p, suiteDir, err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	return out, nil
}
