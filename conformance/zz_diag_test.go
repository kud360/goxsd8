package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// TestDiagPartitionDeclines re-measures the corpus population behind
// xsd/contentmatcher.go's partition-width GAP at the ceiling this tree
// carries. Throwaway diagnostic (#1601), DIAG=1 only, deleted before handoff.
func TestDiagPartitionDeclines(t *testing.T) {
	if os.Getenv("DIAG") != "1" {
		t.Skip("DIAG=1 only")
	}
	skipWithoutSuite(t)
	found, err := parseSuite(suitePath())
	if err != nil {
		t.Fatalf("parsing suite: %v", err)
	}
	backend := strict.New()
	assembled := map[string]struct{}{}
	var roots, built, ec, declined, flattenArm, boundedArm int
	var lines []string
	for _, c := range found.cases {
		root := censusRoot(c)
		if root == "" {
			continue
		}
		if _, dup := assembled[root]; dup {
			continue
		}
		assembled[root] = struct{}{}
		roots++
		s, perr := parser.Parse(filepath.Base(root),
			parser.WithResolver(loader.Dir(filepath.Dir(root))), parser.WithBackend(backend))
		if perr != nil || s == nil {
			continue
		}
		built++
		for _, ct := range diagComplexTypes(s) {
			content, ok := ct.ContentType().(xsd.ElementContent)
			if !ok {
				continue
			}
			ec++
			if _, ok := s.ContentMatcher(ct); ok {
				continue
			}
			declined++
			arm := "partitionsBounded"
			if !diagResolves(s, content.Particle) {
				arm = "flatten"
				flattenArm++
			} else {
				boundedArm++
			}
			lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s",
				filepath.Base(root), ct.Name(), ct.Loc(), arm))
		}
	}
	sort.Strings(lines)
	for _, l := range lines {
		t.Log(l)
	}
	t.Logf("roots=%d built=%d elementContent=%d declined=%d flattenArm=%d partitionsBoundedArm=%d",
		roots, built, ec, declined, flattenArm, boundedArm)
}

// diagComplexTypes collects every complex type reachable from the top-level
// type definitions, the top-level element declarations, and the anonymous
// types nested in element content — the same three routes #1588's probe walked.
func diagComplexTypes(s *xsd.Schema) []xsd.ComplexType {
	var out []xsd.ComplexType
	seen := map[string]bool{}
	var addType func(td xsd.TypeDefinition)
	var addParticle func(p xsd.Particle)
	addType = func(td xsd.TypeDefinition) {
		ct, ok := td.(xsd.ComplexType)
		if !ok {
			return
		}
		key := fmt.Sprintf("%s|%s", ct.Loc(), ct.Name())
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, ct)
		if content, ok := ct.ContentType().(xsd.ElementContent); ok {
			addParticle(content.Particle)
		}
	}
	addParticle = func(p xsd.Particle) {
		rt, ok := p.Term().(xsd.ResolvedTerm)
		if !ok {
			return
		}
		switch term := rt.Term.(type) {
		case xsd.ElementDeclaration:
			if inline, ok := term.TypeDefinition().(xsd.InlineTypeDefinition); ok {
				addType(inline.Definition)
			}
		case xsd.ModelGroup:
			for _, sub := range term.Particles() {
				addParticle(sub)
			}
		}
	}
	for _, td := range s.Types() {
		addType(td)
	}
	for _, e := range s.Elements() {
		if inline, ok := e.TypeDefinition().(xsd.InlineTypeDefinition); ok {
			addType(inline.Definition)
		}
	}
	return out
}

// diagResolves reports whether every term in p's tree is already resolved or
// resolvable through s — the flatten arm's own condition, replicated so a
// decline can be attributed to one arm or the other.
func diagResolves(s *xsd.Schema, p xsd.Particle) bool {
	switch term := p.Term().(type) {
	case xsd.ResolvedTerm:
		g, ok := term.Term.(xsd.ModelGroup)
		if !ok {
			return true
		}
		for _, sub := range g.Particles() {
			if !diagResolves(s, sub) {
				return false
			}
		}
		return true
	case xsd.ElementDeclarationRef:
		_, ok := s.Element(term.Name)
		return ok
	case xsd.ModelGroupRef:
		mgd, ok := s.ModelGroup(term.Name)
		if !ok {
			return false
		}
		for _, sub := range mgd.ModelGroup().Particles() {
			if !diagResolves(s, sub) {
				return false
			}
		}
		return true
	}
	return false
}
