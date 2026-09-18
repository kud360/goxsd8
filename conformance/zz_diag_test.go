package conformance

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// TestDiagContentMatcherDeclines is a THROWAWAY diagnostic for issue #1588: it
// censuses, over every W3C suite case whose assembly succeeds, how many complex
// types with element content ContentMatcher declines, and splits the declines
// into the flatten arm (a ref resolving to nothing) and the partitionsBounded
// arm (the GAP the ruling is about). Run with DIAG=1; delete before handoff.
func TestDiagContentMatcherDeclines(t *testing.T) {
	if os.Getenv("DIAG") != "1" {
		t.Skip("DIAG=1 not set")
	}
	skipWithoutSuite(t)
	found, err := parseSuite(suitePath())
	if err != nil {
		t.Fatalf("parsing suite: %v", err)
	}
	backend := strict.New()
	assembled := map[string]struct{}{}
	var roots, built, elementContent, declined, flattenArm, boundedArm int
	var boundedSites []string
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
			ec, ok := ct.ContentType().(xsd.ElementContent)
			if !ok {
				continue
			}
			elementContent++
			if _, ok := s.ContentMatcher(ct); ok {
				continue
			}
			declined++
			if !diagFlattens(s, ec.Particle, 0) {
				flattenArm++
				continue
			}
			boundedArm++
			boundedSites = append(boundedSites, root+" "+ct.Name().String()+" "+ct.Loc().String())
		}
	}
	sort.Strings(boundedSites)
	t.Logf("roots=%d built=%d elementContent=%d declined=%d flattenArm=%d partitionsBoundedArm=%d",
		roots, built, elementContent, declined, flattenArm, boundedArm)
	for _, site := range boundedSites {
		t.Logf("partitionsBounded decline: %s", site)
	}
}

// diagComplexTypes collects every ComplexType reachable from the schema's own
// type list, its global element declarations, and the anonymous types hanging
// off local element declarations inside content models.
func diagComplexTypes(s *xsd.Schema) []xsd.ComplexType {
	var out []xsd.ComplexType
	seen := map[string]struct{}{}
	var addRef func(ref xsd.TypeDefinitionOrRef, depth int)
	var addComplex func(ct xsd.ComplexType, depth int)
	var addParticle func(p xsd.Particle, depth int)
	addRef = func(ref xsd.TypeDefinitionOrRef, depth int) {
		if depth > 40 {
			return
		}
		td, ok := s.ResolvedType(ref)
		if !ok {
			return
		}
		ct, ok := td.(xsd.ComplexType)
		if !ok {
			return
		}
		addComplex(ct, depth)
	}
	addComplex = func(ct xsd.ComplexType, depth int) {
		if depth > 40 {
			return
		}
		key := ct.Loc().String() + "|" + ct.Name().String()
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		out = append(out, ct)
		ec, ok := ct.ContentType().(xsd.ElementContent)
		if !ok {
			return
		}
		addParticle(ec.Particle, depth+1)
	}
	addParticle = func(p xsd.Particle, depth int) {
		if depth > 40 {
			return
		}
		rt, ok := p.Term().(xsd.ResolvedTerm)
		if !ok {
			return
		}
		switch term := rt.Term.(type) {
		case xsd.ElementDeclaration:
			addRef(term.TypeDefinition(), depth+1)
		case xsd.ModelGroup:
			for _, c := range term.Particles() {
				addParticle(c, depth+1)
			}
		}
	}
	for _, td := range s.Types() {
		if ct, isComplex := td.(xsd.ComplexType); isComplex {
			addComplex(ct, 0)
		}
	}
	for _, e := range s.Elements() {
		addRef(e.TypeDefinition(), 0)
	}
	return out
}

// diagFlattens replicates Matcher.flatten's resolvability check: it reports
// whether every term of the particle tree resolves to a live component.
func diagFlattens(s *xsd.Schema, p xsd.Particle, depth int) bool {
	if depth > 200 {
		return false
	}
	var term xsd.Term
	switch t := p.Term().(type) {
	case xsd.ResolvedTerm:
		if t.Term == nil {
			return false
		}
		term = t.Term
	case xsd.ElementDeclarationRef:
		d, ok := s.Element(t.Name)
		if !ok {
			return false
		}
		term = d
	case xsd.ModelGroupRef:
		mgd, ok := s.ModelGroup(t.Name)
		if !ok {
			return false
		}
		term = mgd.ModelGroup()
	default:
		return false
	}
	g, isGroup := term.(xsd.ModelGroup)
	if !isGroup {
		return true
	}
	for _, c := range g.Particles() {
		if !diagFlattens(s, c, depth+1) {
			return false
		}
	}
	return true
}
